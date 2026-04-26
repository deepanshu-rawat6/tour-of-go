//go:build linux

package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Limits holds the resource constraints for a container.
type Limits struct {
	MemoryBytes int64   // 0 = unlimited
	CPUQuota    float64 // 0 = unlimited; 0.5 = 50% of one core
	PidsMax     int64   // 0 = unlimited
}

// Manager creates and manages a cgroup for one container.
type Manager struct {
	id   string
	v2   bool
	root string // cgroup root path for this container
}

// New creates a new cgroup for the given container ID and applies limits.
func New(id string, limits Limits) (*Manager, error) {
	v2 := isV2()
	m := &Manager{id: id, v2: v2}

	if v2 {
		m.root = filepath.Join("/sys/fs/cgroup/gocker", id)
	} else {
		m.root = filepath.Join("/sys/fs/cgroup/memory/gocker", id)
	}

	if err := os.MkdirAll(m.root, 0755); err != nil {
		return nil, fmt.Errorf("create cgroup: %w", err)
	}

	if v2 {
		if err := m.applyV2(limits); err != nil {
			return nil, err
		}
	} else {
		if err := m.applyV1(limits); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// AddPid adds a process to this cgroup.
func (m *Manager) AddPid(pid int) error {
	procsFile := filepath.Join(m.root, "cgroup.procs")
	if !m.v2 {
		procsFile = filepath.Join("/sys/fs/cgroup/memory/gocker", m.id, "tasks")
	}
	return os.WriteFile(procsFile, []byte(strconv.Itoa(pid)), 0700)
}

// Remove deletes the cgroup.
func (m *Manager) Remove() error {
	if m.v2 {
		return os.RemoveAll(m.root)
	}
	// v1: remove each controller directory
	for _, ctrl := range []string{"memory", "cpu", "pids"} {
		os.RemoveAll(filepath.Join("/sys/fs/cgroup", ctrl, "gocker", m.id))
	}
	return nil
}

func (m *Manager) applyV2(l Limits) error {
	if l.MemoryBytes > 0 {
		if err := write(m.root, "memory.max", strconv.FormatInt(l.MemoryBytes, 10)); err != nil {
			return err
		}
	}
	if l.CPUQuota > 0 {
		// cpu.max format: "quota period" — quota/period = fraction of one CPU
		quota := int64(l.CPUQuota * 100000)
		if err := write(m.root, "cpu.max", fmt.Sprintf("%d 100000", quota)); err != nil {
			return err
		}
	}
	if l.PidsMax > 0 {
		if err := write(m.root, "pids.max", strconv.FormatInt(l.PidsMax, 10)); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) applyV1(l Limits) error {
	if l.MemoryBytes > 0 {
		dir := filepath.Join("/sys/fs/cgroup/memory/gocker", m.id)
		os.MkdirAll(dir, 0755)
		if err := write(dir, "memory.limit_in_bytes", strconv.FormatInt(l.MemoryBytes, 10)); err != nil {
			return err
		}
	}
	if l.CPUQuota > 0 {
		dir := filepath.Join("/sys/fs/cgroup/cpu/gocker", m.id)
		os.MkdirAll(dir, 0755)
		quota := int64(l.CPUQuota * 100000)
		write(dir, "cpu.cfs_period_us", "100000")
		if err := write(dir, "cpu.cfs_quota_us", strconv.FormatInt(quota, 10)); err != nil {
			return err
		}
	}
	if l.PidsMax > 0 {
		dir := filepath.Join("/sys/fs/cgroup/pids/gocker", m.id)
		os.MkdirAll(dir, 0755)
		if err := write(dir, "pids.max", strconv.FormatInt(l.PidsMax, 10)); err != nil {
			return err
		}
	}
	return nil
}

func isV2() bool {
	_, err := os.Stat("/sys/fs/cgroup/cgroup.controllers")
	return err == nil
}

func write(dir, file, value string) error {
	return os.WriteFile(filepath.Join(dir, file), []byte(value), 0700)
}
