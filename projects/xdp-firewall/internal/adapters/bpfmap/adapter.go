//go:build linux

package bpfmap

import (
	"fmt"
	"net"
	"unsafe"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/tour-of-go/xdp-firewall/internal/core"
)

// BPFAdapter implements core.BPFMapPort and core.CounterReader.
// Wraps the bpf2go-generated objects from xdp_firewall.c.
type BPFAdapter struct {
	prog     *ebpf.Program
	blacklist *ebpf.Map
	counters  *ebpf.Map
	xdpLink  link.Link
}

// Load loads the compiled BPF ELF and attaches the XDP program to iface.
// bpfObjPath is the path to the compiled xdp_firewall.o ELF file.
func Load(iface, bpfObjPath string) (*BPFAdapter, error) {
	spec, err := ebpf.LoadCollectionSpec(bpfObjPath)
	if err != nil {
		return nil, fmt.Errorf("loading BPF spec: %w", err)
	}
	var objs struct {
		XdpFirewallProg *ebpf.Program `ebpf:"xdp_firewall_prog"`
		Blacklist       *ebpf.Map     `ebpf:"blacklist"`
		Counters        *ebpf.Map     `ebpf:"counters"`
	}
	if err := spec.LoadAndAssign(&objs, nil); err != nil {
		return nil, fmt.Errorf("loading BPF objects: %w", err)
	}
	netIface, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("interface %q not found: %w", iface, err)
	}
	l, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.XdpFirewallProg,
		Interface: netIface.Index,
	})
	if err != nil {
		objs.XdpFirewallProg.Close()
		objs.Blacklist.Close()
		objs.Counters.Close()
		return nil, fmt.Errorf("attaching XDP to %s: %w", iface, err)
	}
	return &BPFAdapter{
		prog:     objs.XdpFirewallProg,
		blacklist: objs.Blacklist,
		counters:  objs.Counters,
		xdpLink:  l,
	}, nil
}

// Insert adds a CIDR to the LPM trie blacklist.
func (a *BPFAdapter) Insert(cidr string) error {
	key, err := cidrToLPMKey(cidr)
	if err != nil {
		return err
	}
	val := uint8(1)
	return a.blacklist.Put(unsafe.Pointer(&key), unsafe.Pointer(&val))
}

// Delete removes a CIDR from the LPM trie blacklist.
func (a *BPFAdapter) Delete(cidr string) error {
	key, err := cidrToLPMKey(cidr)
	if err != nil {
		return err
	}
	return a.blacklist.Delete(unsafe.Pointer(&key))
}

// List returns all CIDRs currently in the LPM trie.
func (a *BPFAdapter) List() ([]string, error) {
	var cidrs []string
	var key lpmKey
	var val uint8
	iter := a.blacklist.Iterate()
	for iter.Next(&key, &val) {
		cidrs = append(cidrs, lpmKeyToCIDR(key))
	}
	return cidrs, iter.Err()
}

// ReadCounters reads per-CPU counters and sums across all CPUs.
func (a *BPFAdapter) ReadCounters() (core.Counters, error) {
	var c core.Counters
	numCPU, err := ebpf.PossibleCPU()
	if err != nil {
		return c, err
	}
	packets := make([]uint64, numCPU)
	drops := make([]uint64, numCPU)
	key0, key1 := uint32(0), uint32(1)
	if err := a.counters.Lookup(&key0, &packets); err != nil {
		return c, fmt.Errorf("reading packets counter: %w", err)
	}
	if err := a.counters.Lookup(&key1, &drops); err != nil {
		return c, fmt.Errorf("reading drops counter: %w", err)
	}
	for i := 0; i < numCPU; i++ {
		c.PacketsTotal += packets[i]
		c.DropsTotal += drops[i]
	}
	return c, nil
}

// Close detaches the XDP program and closes all BPF objects.
func (a *BPFAdapter) Close() {
	if a.xdpLink != nil {
		a.xdpLink.Close()
	}
	a.prog.Close()
	a.blacklist.Close()
	a.counters.Close()
}
