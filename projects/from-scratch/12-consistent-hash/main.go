package main

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

// HashRing implements consistent hashing with virtual nodes.
type HashRing struct {
	nodes    map[uint32]string
	sorted   []uint32
	replicas int
	mu       sync.RWMutex
}

func NewHashRing(replicas int) *HashRing {
	return &HashRing{
		nodes:    make(map[uint32]string),
		replicas: replicas,
	}
}

func (r *HashRing) hash(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

func (r *HashRing) AddNode(node string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := 0; i < r.replicas; i++ {
		virtualKey := node + "-vn" + strconv.Itoa(i)
		h := r.hash(virtualKey)
		r.nodes[h] = node
		r.sorted = append(r.sorted, h)
	}
	sort.Slice(r.sorted, func(i, j int) bool { return r.sorted[i] < r.sorted[j] })
}

func (r *HashRing) RemoveNode(node string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := 0; i < r.replicas; i++ {
		virtualKey := node + "-vn" + strconv.Itoa(i)
		h := r.hash(virtualKey)
		delete(r.nodes, h)
	}
	newSorted := r.sorted[:0]
	for _, h := range r.sorted {
		if _, ok := r.nodes[h]; ok {
			newSorted = append(newSorted, h)
		}
	}
	r.sorted = newSorted
}

func (r *HashRing) GetNode(key string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.sorted) == 0 {
		return ""
	}
	h := r.hash(key)
	idx := sort.Search(len(r.sorted), func(i int) bool { return r.sorted[i] >= h })
	if idx >= len(r.sorted) {
		idx = 0
	}
	return r.nodes[r.sorted[idx]]
}

func main() {
	ring := NewHashRing(150) // 150 virtual nodes per physical node

	// Add nodes
	nodes := []string{"cache-1", "cache-2", "cache-3"}
	for _, n := range nodes {
		ring.AddNode(n)
	}

	// Distribute keys
	keys := []string{"user:1", "user:2", "session:abc", "order:42", "product:99", "cart:7"}
	fmt.Println("=== Initial Distribution ===")
	mapping := make(map[string]string)
	for _, k := range keys {
		node := ring.GetNode(k)
		mapping[k] = node
		fmt.Printf("  %s → %s\n", k, node)
	}

	// Remove a node — show minimal redistribution
	fmt.Println("\n=== After Removing cache-2 ===")
	ring.RemoveNode("cache-2")
	moved := 0
	for _, k := range keys {
		node := ring.GetNode(k)
		marker := " "
		if mapping[k] != node {
			marker = "*"
			moved++
		}
		fmt.Printf("%s %s → %s\n", marker, k, node)
	}
	fmt.Printf("\nKeys moved: %d/%d (%.0f%%)\n", moved, len(keys), float64(moved)/float64(len(keys))*100)

	// Distribution uniformity test
	fmt.Println("\n=== Distribution Uniformity (1000 keys, 3 nodes) ===")
	ring2 := NewHashRing(150)
	for _, n := range nodes {
		ring2.AddNode(n)
	}
	dist := make(map[string]int)
	for i := 0; i < 1000; i++ {
		node := ring2.GetNode("key:" + strconv.Itoa(i))
		dist[node]++
	}
	for n, count := range dist {
		fmt.Printf("  %s: %d keys (%.1f%%)\n", n, count, float64(count)/10)
	}
}
