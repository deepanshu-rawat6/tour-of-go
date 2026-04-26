package core

import (
	"errors"
	"testing"
)

// --- mocks ---

type mockBPF struct {
	rules map[string]struct{}
	err   error
}

func newMockBPF() *mockBPF { return &mockBPF{rules: make(map[string]struct{})} }

func (m *mockBPF) Insert(cidr string) error {
	if m.err != nil {
		return m.err
	}
	m.rules[cidr] = struct{}{}
	return nil
}
func (m *mockBPF) Delete(cidr string) error {
	delete(m.rules, cidr)
	return m.err
}
func (m *mockBPF) List() ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make([]string, 0, len(m.rules))
	for k := range m.rules {
		out = append(out, k)
	}
	return out, nil
}

type mockStore struct {
	rules []string
}

func (m *mockStore) Save(rules []string) error { m.rules = rules; return nil }
func (m *mockStore) Load() ([]string, error)   { return m.rules, nil }

// --- tests ---

func TestBlockCIDR_Valid(t *testing.T) {
	bpf := newMockBPF()
	store := &mockStore{}
	e := NewThreatEngine(bpf, store)

	if err := e.BlockCIDR("10.0.0.0/8"); err != nil {
		t.Fatal(err)
	}
	if _, ok := bpf.rules["10.0.0.0/8"]; !ok {
		t.Error("expected CIDR in BPF map")
	}
	if len(store.rules) == 0 {
		t.Error("expected rules saved to store")
	}
}

func TestBlockCIDR_InvalidCIDR(t *testing.T) {
	e := NewThreatEngine(newMockBPF(), &mockStore{})
	if err := e.BlockCIDR("not-a-cidr"); err == nil {
		t.Fatal("expected error for invalid CIDR")
	}
	if err := e.BlockCIDR("999.0.0.0/8"); err == nil {
		t.Fatal("expected error for invalid IP")
	}
}

func TestUnblockCIDR(t *testing.T) {
	bpf := newMockBPF()
	store := &mockStore{}
	e := NewThreatEngine(bpf, store)

	e.BlockCIDR("192.168.0.0/16")
	if err := e.UnblockCIDR("192.168.0.0/16"); err != nil {
		t.Fatal(err)
	}
	if _, ok := bpf.rules["192.168.0.0/16"]; ok {
		t.Error("expected CIDR removed from BPF map")
	}
}

func TestBatchBlock(t *testing.T) {
	bpf := newMockBPF()
	e := NewThreatEngine(bpf, &mockStore{})
	cidrs := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	if err := e.BatchBlock(cidrs); err != nil {
		t.Fatal(err)
	}
	if len(bpf.rules) != 3 {
		t.Errorf("expected 3 rules in BPF map, got %d", len(bpf.rules))
	}
}

func TestBatchBlock_InvalidCIDR(t *testing.T) {
	e := NewThreatEngine(newMockBPF(), &mockStore{})
	if err := e.BatchBlock([]string{"10.0.0.0/8", "bad"}); err == nil {
		t.Fatal("expected error for invalid CIDR in batch")
	}
}

func TestReconcile_InsertsFileMissingFromKernel(t *testing.T) {
	bpf := newMockBPF()
	store := &mockStore{rules: []string{"10.0.0.0/8", "192.168.0.0/16"}}
	e := NewThreatEngine(bpf, store)

	if err := e.Reconcile(); err != nil {
		t.Fatal(err)
	}
	if len(bpf.rules) != 2 {
		t.Errorf("expected 2 rules inserted into kernel, got %d", len(bpf.rules))
	}
}

func TestReconcile_RemovesKernelRulesMissingFromFile(t *testing.T) {
	bpf := newMockBPF()
	bpf.rules["10.0.0.0/8"] = struct{}{}   // stale kernel rule
	bpf.rules["172.16.0.0/12"] = struct{}{} // stale kernel rule
	store := &mockStore{rules: []string{}}   // file is empty
	e := NewThreatEngine(bpf, store)

	if err := e.Reconcile(); err != nil {
		t.Fatal(err)
	}
	if len(bpf.rules) != 0 {
		t.Errorf("expected stale rules removed from kernel, got %d", len(bpf.rules))
	}
}

func TestReconcile_BPFError(t *testing.T) {
	bpf := newMockBPF()
	bpf.err = errors.New("bpf error")
	e := NewThreatEngine(bpf, &mockStore{})
	if err := e.Reconcile(); err == nil {
		t.Fatal("expected error when BPF list fails")
	}
}
