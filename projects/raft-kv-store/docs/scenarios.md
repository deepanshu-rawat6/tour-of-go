# Use Cases & Scenarios

## 1. Understanding Raft Leader Election

**Scenario:** You want to see how a Raft cluster recovers from a leader failure.

```bash
# Start 3-node cluster
make run-cluster

# Write some data
curl -X PUT localhost:8001/kv/counter -d '0'

# Kill the leader (node1)
kill $(pgrep -f "config-node1")

# Within 300ms, node2 or node3 wins election
curl localhost:8002/status
# {"role":"Leader","term":2,"leader_id":"node2",...}

# Cluster still serves reads and writes
curl -X PUT localhost:8002/kv/counter -d '1'
```

**What you observe:** The cluster elects a new leader within one election timeout (150–300ms). The new leader has a higher term number.

---

## 2. Log Replication and Quorum

**Scenario:** You want to verify that a write is only committed after a majority of nodes acknowledge it.

```bash
# Write to leader
curl -X PUT localhost:8001/kv/config -d 'production'

# Read from all nodes — all should return the same value after commit
curl localhost:8001/kv/config  # → production
curl localhost:8002/kv/config  # → production
curl localhost:8003/kv/config  # → production
```

**What you observe:** The value is consistent across all nodes because the leader waited for 2/3 nodes to acknowledge before committing.

---

## 3. Follower Redirect

**Scenario:** A client doesn't know which node is the leader and sends a write to a follower.

```bash
# Send write to follower (node2)
curl -v -X PUT localhost:8002/kv/foo -d 'bar'
# < HTTP/1.1 307 Temporary Redirect
# < Location: http://localhost:8001/kv/foo

# Follow the redirect automatically
curl -L -X PUT localhost:8002/kv/foo -d 'bar'
# → bar (success, transparently redirected to leader)
```

**What you observe:** The follower returns a 307 redirect with the leader's HTTP address. The `-L` flag makes curl follow it automatically.

---

## 4. WAL Recovery After Restart

**Scenario:** A node crashes and restarts. It should recover its log from the WAL and rejoin the cluster.

```bash
# Write some data
curl -X PUT localhost:8001/kv/persistent -d 'value'

# Kill node3
kill $(pgrep -f "config-node3")

# Write more data while node3 is down
curl -X PUT localhost:8001/kv/new -d 'data'

# Restart node3
./raft-kv start --config config-node3.yaml &

# node3 recovers WAL, receives missing entries from leader
curl localhost:8003/kv/new  # → data (eventually consistent)
```

**What you observe:** The restarted node recovers its committed log from the WAL and receives any missing entries from the leader via AppendEntries.

---

## 5. Split Vote and Re-election

**Scenario:** Two candidates start an election simultaneously (split vote). The cluster must resolve it.

This happens naturally when two followers time out at nearly the same time. Because timeouts are randomized (150–300ms), one candidate will start a new election before the other and win.

```bash
# Observe term numbers increasing during split votes
watch -n 0.5 'curl -s localhost:8001/status | jq .term'
```

**What you observe:** The term number increments each time an election starts. Eventually one candidate wins with a majority.

---

## 6. Cluster Status Monitoring

**Scenario:** You want to monitor the health of the cluster.

```bash
# Check all nodes
for port in 8001 8002 8003; do
  echo "Node on :$port:"
  curl -s localhost:$port/status | jq '{role, term, leader_id, commit_index}'
done
```

**Example output:**
```json
{"role": "Leader",   "term": 3, "leader_id": "node1", "commit_index": 42}
{"role": "Follower", "term": 3, "leader_id": "node1", "commit_index": 42}
{"role": "Follower", "term": 3, "leader_id": "node1", "commit_index": 41}
```

Node3 is one entry behind — it will catch up on the next heartbeat.
