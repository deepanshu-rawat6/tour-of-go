//go:build linux

package cgroup

import (
	"testing"
)

func TestIsV2(t *testing.T) {
	// Just verify the function runs without panic — result depends on host kernel.
	_ = isV2()
}

func TestLimitsZeroMeansUnlimited(t *testing.T) {
	l := Limits{}
	if l.MemoryBytes != 0 || l.CPUQuota != 0 || l.PidsMax != 0 {
		t.Error("zero Limits should mean unlimited")
	}
}
