package core

import (
	"fmt"
	"net/netip"
	"sync"
)

// ThreatEngine is the core domain — manages the CIDR blacklist.
// It coordinates two outbound ports: BPFMapPort (kernel) and RuleStorePort (disk).
type ThreatEngine struct {
	mu    sync.RWMutex
	rules map[string]struct{} // in-memory set for fast lookups
	bpf   BPFMapPort
	store RuleStorePort
}

func NewThreatEngine(bpf BPFMapPort, store RuleStorePort) *ThreatEngine {
	return &ThreatEngine{
		rules: make(map[string]struct{}),
		bpf:   bpf,
		store: store,
	}
}

// BlockCIDR validates and adds a CIDR to the blacklist.
func (e *ThreatEngine) BlockCIDR(cidr string) error {
	if err := validateCIDR(cidr); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.store.Save(e.rulesSlice(cidr)); err != nil {
		return fmt.Errorf("store save: %w", err)
	}
	if err := e.bpf.Insert(cidr); err != nil {
		return fmt.Errorf("bpf insert: %w", err)
	}
	e.rules[cidr] = struct{}{}
	return nil
}

// UnblockCIDR removes a CIDR from the blacklist.
func (e *ThreatEngine) UnblockCIDR(cidr string) error {
	if err := validateCIDR(cidr); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, cidr)
	if err := e.store.Save(e.rulesSliceAll()); err != nil {
		return fmt.Errorf("store save: %w", err)
	}
	if err := e.bpf.Delete(cidr); err != nil {
		return fmt.Errorf("bpf delete: %w", err)
	}
	return nil
}

// BatchBlock adds multiple CIDRs atomically.
func (e *ThreatEngine) BatchBlock(cidrs []string) error {
	for _, cidr := range cidrs {
		if err := validateCIDR(cidr); err != nil {
			return fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, cidr := range cidrs {
		e.rules[cidr] = struct{}{}
	}
	if err := e.store.Save(e.rulesSliceAll()); err != nil {
		return fmt.Errorf("store save: %w", err)
	}
	for _, cidr := range cidrs {
		if err := e.bpf.Insert(cidr); err != nil {
			return fmt.Errorf("bpf insert %s: %w", cidr, err)
		}
	}
	return nil
}

// ImportList is an alias for BatchBlock — used for threat feed imports.
func (e *ThreatEngine) ImportList(cidrs []string) error {
	return e.BatchBlock(cidrs)
}

// ListRules returns all currently blocked CIDRs.
func (e *ThreatEngine) ListRules() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.rulesSliceAll()
}

// Reconcile syncs file state ↔ kernel map state on startup.
// Inserts rules present in file but missing from kernel.
// Removes rules present in kernel but missing from file.
func (e *ThreatEngine) Reconcile() error {
	fileRules, err := e.store.Load()
	if err != nil {
		fileRules = []string{} // treat missing file as empty
	}
	kernelRules, err := e.bpf.List()
	if err != nil {
		return fmt.Errorf("bpf list: %w", err)
	}

	fileSet := toSet(fileRules)
	kernelSet := toSet(kernelRules)

	// Insert rules in file but not in kernel
	for cidr := range fileSet {
		if _, ok := kernelSet[cidr]; !ok {
			if err := e.bpf.Insert(cidr); err != nil {
				return fmt.Errorf("reconcile insert %s: %w", cidr, err)
			}
		}
	}

	// Remove rules in kernel but not in file
	for cidr := range kernelSet {
		if _, ok := fileSet[cidr]; !ok {
			if err := e.bpf.Delete(cidr); err != nil {
				return fmt.Errorf("reconcile delete %s: %w", cidr, err)
			}
		}
	}

	// Sync in-memory state from file (file is source of truth)
	e.mu.Lock()
	e.rules = fileSet
	e.mu.Unlock()
	return nil
}

// --- helpers ---

func validateCIDR(cidr string) error {
	_, err := netip.ParsePrefix(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}
	return nil
}

// rulesSlice returns all current rules plus an optional new one (no lock — caller holds it).
func (e *ThreatEngine) rulesSlice(extra string) []string {
	s := e.rulesSliceAll()
	if extra != "" {
		s = append(s, extra)
	}
	return s
}

func (e *ThreatEngine) rulesSliceAll() []string {
	out := make([]string, 0, len(e.rules))
	for cidr := range e.rules {
		out = append(out, cidr)
	}
	return out
}

func toSet(s []string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}
