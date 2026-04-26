package bpfmap

import (
	"testing"
)

func TestCIDRToLPMKey(t *testing.T) {
	tests := []struct {
		cidr      string
		prefixlen uint32
		wantErr   bool
	}{
		{"10.0.0.0/8", 8, false},
		{"192.168.1.0/24", 24, false},
		{"0.0.0.0/0", 0, false},
		{"10.0.0.0/32", 32, false},
		{"not-a-cidr", 0, true},
		{"999.0.0.0/8", 0, true},
	}
	for _, tc := range tests {
		key, err := cidrToLPMKey(tc.cidr)
		if tc.wantErr {
			if err == nil {
				t.Errorf("cidrToLPMKey(%q): expected error", tc.cidr)
			}
			continue
		}
		if err != nil {
			t.Errorf("cidrToLPMKey(%q): unexpected error: %v", tc.cidr, err)
			continue
		}
		if key.Prefixlen != tc.prefixlen {
			t.Errorf("cidrToLPMKey(%q): prefixlen = %d, want %d", tc.cidr, key.Prefixlen, tc.prefixlen)
		}
	}
}

func TestLPMKeyRoundTrip(t *testing.T) {
	cidrs := []string{"10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"}
	for _, cidr := range cidrs {
		key, err := cidrToLPMKey(cidr)
		if err != nil {
			t.Fatal(err)
		}
		got := lpmKeyToCIDR(key)
		if got != cidr {
			t.Errorf("round-trip %q → key → %q", cidr, got)
		}
	}
}
