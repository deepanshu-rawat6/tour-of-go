//go:build linux

package main

import "testing"

func TestParseMemory(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		err   bool
	}{
		{"128m", 128 * 1024 * 1024, false},
		{"1g", 1024 * 1024 * 1024, false},
		{"512k", 512 * 1024, false},
		{"1073741824", 1073741824, false},
		{"", 0, false},
		{"abc", 0, true},
	}
	for _, tc := range tests {
		got, err := parseMemory(tc.input)
		if tc.err && err == nil {
			t.Errorf("parseMemory(%q): expected error", tc.input)
		}
		if !tc.err && err != nil {
			t.Errorf("parseMemory(%q): unexpected error: %v", tc.input, err)
		}
		if !tc.err && got != tc.want {
			t.Errorf("parseMemory(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
