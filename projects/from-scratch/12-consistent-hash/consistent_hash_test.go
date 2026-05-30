package main

import (
	"strconv"
	"testing"
)

func TestGetNodeEmpty(t *testing.T) {
	ring := NewHashRing(100)
	if got := ring.GetNode("key"); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestDistribution(t *testing.T) {
	ring := NewHashRing(150)
	ring.AddNode("A")
	ring.AddNode("B")
	ring.AddNode("C")

	dist := make(map[string]int)
	for i := 0; i < 10000; i++ {
		node := ring.GetNode("key:" + strconv.Itoa(i))
		dist[node]++
	}

	for node, count := range dist {
		pct := float64(count) / 100
		if pct < 20 || pct > 45 {
			t.Errorf("node %s has %d keys (%.1f%%) — too skewed", node, count, pct)
		}
	}
}

func TestMinimalRedistribution(t *testing.T) {
	ring := NewHashRing(150)
	ring.AddNode("A")
	ring.AddNode("B")
	ring.AddNode("C")

	before := make(map[string]string)
	for i := 0; i < 1000; i++ {
		key := "k" + strconv.Itoa(i)
		before[key] = ring.GetNode(key)
	}

	ring.RemoveNode("B")

	moved := 0
	for i := 0; i < 1000; i++ {
		key := "k" + strconv.Itoa(i)
		if ring.GetNode(key) != before[key] {
			moved++
		}
	}

	// Ideally ~1/3 keys move when removing 1 of 3 nodes
	pct := float64(moved) / 10
	if pct > 50 {
		t.Errorf("too many keys moved: %d (%.1f%%)", moved, pct)
	}
}
