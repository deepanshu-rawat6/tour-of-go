package main

import (
	"fmt"
	"sort"
	"sync"
)

// LSMTree implements a simplified LSM-tree with memtable and SSTables.
type LSMTree struct {
	memtable    map[string]string // current writable memtable
	memSize     int
	maxMemSize  int
	sstables    [][]sstEntry // flushed sorted runs (newest first)
	mu          sync.RWMutex
}

type sstEntry struct {
	Key   string
	Value string
	Tomb  bool // tombstone for deletes
}

func NewLSMTree(maxMemSize int) *LSMTree {
	return &LSMTree{
		memtable:   make(map[string]string),
		maxMemSize: maxMemSize,
	}
}

func (l *LSMTree) Put(key, value string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.memtable[key] = value
	l.memSize++
	if l.memSize >= l.maxMemSize {
		l.flush()
	}
}

func (l *LSMTree) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.memtable[key] = "" // tombstone marker
	l.memSize++
	if l.memSize >= l.maxMemSize {
		l.flush()
	}
}

func (l *LSMTree) Get(key string) (string, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check memtable first (newest data)
	if v, ok := l.memtable[key]; ok {
		if v == "" {
			return "", false // tombstone
		}
		return v, true
	}

	// Check SSTables from newest to oldest
	for _, sst := range l.sstables {
		idx := sort.Search(len(sst), func(i int) bool { return sst[i].Key >= key })
		if idx < len(sst) && sst[idx].Key == key {
			if sst[idx].Tomb {
				return "", false
			}
			return sst[idx].Value, true
		}
	}
	return "", false
}

// flush converts memtable to a sorted SSTable.
func (l *LSMTree) flush() {
	entries := make([]sstEntry, 0, len(l.memtable))
	for k, v := range l.memtable {
		entries = append(entries, sstEntry{Key: k, Value: v, Tomb: v == ""})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })

	// Prepend (newest first)
	l.sstables = append([][]sstEntry{entries}, l.sstables...)
	l.memtable = make(map[string]string)
	l.memSize = 0
}

// Compact merges all SSTables into one (simplified single-level compaction).
func (l *LSMTree) Compact() {
	l.mu.Lock()
	defer l.mu.Unlock()

	merged := make(map[string]sstEntry)
	// Oldest first so newer overwrites
	for i := len(l.sstables) - 1; i >= 0; i-- {
		for _, e := range l.sstables[i] {
			merged[e.Key] = e
		}
	}

	entries := make([]sstEntry, 0, len(merged))
	for _, e := range merged {
		if !e.Tomb {
			entries = append(entries, e)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	l.sstables = [][]sstEntry{entries}
}

func (l *LSMTree) Stats() (memKeys, sstCount int) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.memtable), len(l.sstables)
}

func demoLSM() {
	fmt.Println("\n=== LSM-Tree Demo ===")
	lsm := NewLSMTree(3) // flush every 3 writes

	lsm.Put("user:1", "alice")
	lsm.Put("user:2", "bob")
	lsm.Put("user:3", "charlie") // triggers flush

	lsm.Put("user:4", "dave")
	lsm.Put("user:2", "bob-updated") // overwrite
	lsm.Put("user:5", "eve")         // triggers flush

	mem, sst := lsm.Stats()
	fmt.Printf("Memtable keys: %d, SSTable count: %d\n", mem, sst)

	for _, k := range []string{"user:1", "user:2", "user:5", "user:99"} {
		v, ok := lsm.Get(k)
		fmt.Printf("  Get(%s) = %q, found=%v\n", k, v, ok)
	}

	lsm.Delete("user:1")
	v, ok := lsm.Get("user:1")
	fmt.Printf("  After delete: Get(user:1) = %q, found=%v\n", v, ok)

	lsm.Compact()
	_, sst = lsm.Stats()
	fmt.Printf("  After compaction: SSTable count: %d\n", sst)
}
