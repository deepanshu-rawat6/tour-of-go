// xdp_firewall.c — XDP packet filter with LPM trie CIDR blacklist.
// Compiled by bpf2go: clang -target bpf -O2 -c xdp_firewall.c
//go:build ignore

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// LPM trie key: must match Go struct XdpFirewallLpmKey exactly.
struct lpm_key {
    __u32 prefixlen; // prefix length in bits (e.g. 24 for /24)
    __u32 addr;      // IPv4 address in network byte order
};

// BPF_MAP_TYPE_LPM_TRIE: longest-prefix-match for CIDR blacklist.
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __uint(max_entries, 65536);
    __type(key, struct lpm_key);
    __type(value, __u8);
    __uint(map_flags, BPF_F_NO_PREALLOC);
} blacklist SEC(".maps");

// Counter indices.
#define COUNTER_PACKETS 0
#define COUNTER_DROPS   1
#define COUNTER_MAX     2

// BPF_MAP_TYPE_PERCPU_ARRAY: lock-free per-CPU counters.
// Each CPU has its own copy — no contention at line rate.
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, COUNTER_MAX);
    __type(key, __u32);
    __type(value, __u64);
} counters SEC(".maps");

static __always_inline void inc_counter(__u32 idx) {
    __u64 *val = bpf_map_lookup_elem(&counters, &idx);
    if (val)
        __sync_fetch_and_add(val, 1);
}

SEC("xdp")
int xdp_firewall_prog(struct xdp_md *ctx) {
    void *data     = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    // Parse Ethernet header.
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        goto pass;

    // Only process IPv4.
    if (bpf_ntohs(eth->h_proto) != ETH_P_IP)
        goto pass;

    // Parse IP header.
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        goto pass;

    // LPM trie lookup on source IP.
    struct lpm_key key = {
        .prefixlen = 32,       // match exact IP; trie handles prefix matching
        .addr      = ip->saddr,
    };

    if (bpf_map_lookup_elem(&blacklist, &key)) {
        inc_counter(COUNTER_DROPS);
        return XDP_DROP;
    }

pass:
    inc_counter(COUNTER_PACKETS);
    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
