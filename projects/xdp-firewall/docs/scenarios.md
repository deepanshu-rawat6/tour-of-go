# Use Cases & Scenarios

## 1. DDoS Mitigation — Drop Attack Traffic Before It Costs You

**Scenario:** Your server is under a volumetric DDoS attack from a botnet. The attack traffic is consuming CPU and memory just to be rejected by your application.

**How xdp-firewall helps:** Block the attacking CIDR ranges at the NIC driver level. Packets are dropped before the kernel allocates a socket buffer, before the TCP stack processes them, before your application sees them.

```bash
# Block the attacking subnet immediately
curl -X POST localhost:8080/api/blacklist -d '{"cidr": "198.51.100.0/24"}'

# Check drop rate
curl localhost:8080/stats
# {"drops_total": 4200000, "packets_total": 4200150, ...}
```

**Why XDP beats iptables here:** iptables processes packets after the kernel networking stack has already allocated memory. XDP drops them at the driver level — 10x lower CPU overhead at high packet rates.

---

## 2. Threat Intelligence Feed Import — Block Thousands of IPs at Once

**Scenario:** Your SIEM outputs a daily list of 50,000 malicious IPs from threat intelligence feeds (Spamhaus, AbuseIPDB, internal honeypot data).

**How xdp-firewall helps:** The `/api/blacklist/import` endpoint accepts a plain text file (one CIDR per line) and batch-inserts all rules into the LPM trie in one API call.

```bash
# Download and import a threat feed
curl -s https://www.spamhaus.org/drop/drop.txt | grep -v '^;' > drop.txt
curl -X POST localhost:8080/api/blacklist/import -F "file=@drop.txt"
# {"imported": 1247}

# Or import from JSON
curl -X POST localhost:8080/api/blacklist/import \
  -d '{"cidrs": ["192.0.2.0/24", "203.0.113.0/24", ...]}'
```

---

## 3. Incident Response — Block a Malicious Actor in Seconds

**Scenario:** Your SOC identifies an active credential stuffing attack from a specific IP range. You need to block it immediately without restarting any services or modifying firewall rules in the cloud console.

```bash
# Block immediately — takes effect in microseconds
curl -X POST localhost:8080/api/blacklist -d '{"cidr": "203.0.113.42/32"}'

# Verify it's blocked
curl localhost:8080/api/blacklist | jq '.rules'

# Unblock when the incident is resolved
curl -X DELETE localhost:8080/api/blacklist -d '{"cidr": "203.0.113.42/32"}'
```

The rule is written to both the kernel LPM trie (immediate effect) and `rules.json` (survives restarts).

---

## 4. Compliance Audit — Verify Blocked Ranges

**Scenario:** Your compliance team needs to verify that certain IP ranges (e.g., Tor exit nodes, known malicious ASNs) are blocked on all production servers.

```bash
# List all currently blocked CIDRs
curl localhost:8080/api/blacklist | jq '.rules | sort'

# Check Prometheus metrics for audit trail
curl localhost:8080/metrics | grep xdp_
# xdp_blacklist_size 1247
# xdp_drops_total 8.4e+06
# xdp_packets_total 1.2e+09
```

---

## 5. Daemon Crash Recovery — Reconciliation in Action

**Scenario:** The Go daemon crashes (OOM, bug, deployment). The XDP program keeps running in the kernel — packets are still being dropped. When the daemon restarts, it must re-sync with the kernel state.

**What happens on restart:**
1. Daemon loads `rules.json` (source of truth)
2. Daemon reads current kernel LPM trie entries
3. Diff: rules in file but not in kernel → insert
4. Diff: rules in kernel but not in file → remove
5. In-memory state synced from file

```bash
# Simulate: daemon restarts, rules.json has 3 CIDRs
sudo systemctl restart xdp-firewall
# Logs: "reconcile: inserted 3 rules, removed 0 stale rules"
```

---

## 6. Development & Testing — No Root Needed

**Scenario:** You're developing on macOS or a non-root Linux environment and want to test the API and domain logic without loading actual eBPF programs.

The daemon uses a `noopBPF` adapter when the real BPF adapter isn't available. All API endpoints, CIDR validation, batch operations, and file persistence work identically — only the kernel interaction is stubbed.

```bash
# Build and run without root (uses noopBPF)
make build
./xdp-fw start --config config.example.yaml

# Full API works
curl -X POST localhost:8080/api/blacklist -d '{"cidr": "10.0.0.0/8"}'
curl localhost:8080/api/blacklist
curl localhost:8080/stats
```

All 22 unit tests run without root, without Linux, without clang.
