package bpfmap

import (
	"encoding/binary"
	"fmt"
	"net/netip"
)

// lpmKey mirrors the C struct lpm_key exactly.
// Must match: struct { __u32 prefixlen; __u32 addr; }
type lpmKey struct {
	Prefixlen uint32
	Addr      uint32 // IPv4 in network byte order
}

// cidrToLPMKey converts a CIDR string to the LPM trie key format.
func cidrToLPMKey(cidr string) (lpmKey, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return lpmKey{}, fmt.Errorf("invalid CIDR: %w", err)
	}
	prefix = prefix.Masked()
	addr := prefix.Addr().As4()
	return lpmKey{
		Prefixlen: uint32(prefix.Bits()),
		Addr:      binary.BigEndian.Uint32(addr[:]),
	}, nil
}

// lpmKeyToCIDR converts an LPM trie key back to a CIDR string.
func lpmKeyToCIDR(k lpmKey) string {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], k.Addr)
	addr := netip.AddrFrom4(b)
	return netip.PrefixFrom(addr, int(k.Prefixlen)).String()
}
