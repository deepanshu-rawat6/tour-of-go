package filestore

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "rules.json"))
	rules := []string{"10.0.0.0/8", "192.168.0.0/16"}
	if err := s.Save(rules); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 rules, got %d", len(got))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "nonexistent.json"))
	rules, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Errorf("expected empty rules for missing file, got %d", len(rules))
	}
}

func TestSave_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, "rules.json"))
	s.Save([]string{"10.0.0.0/8"})
	entries, _ := filepath.Glob(dir + "/*.tmp")
	if len(entries) != 0 {
		t.Errorf("expected no .tmp files after save, found %d", len(entries))
	}
}
