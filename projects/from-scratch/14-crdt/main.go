package main

import (
	"fmt"
	"time"
)

// --- G-Counter ---

type GCounter struct {
	id      string
	counts  map[string]uint64
}

func NewGCounter(id string) *GCounter {
	return &GCounter{id: id, counts: map[string]uint64{id: 0}}
}

func (g *GCounter) Increment() {
	g.counts[g.id]++
}

func (g *GCounter) Value() uint64 {
	var sum uint64
	for _, v := range g.counts {
		sum += v
	}
	return sum
}

func (g *GCounter) Merge(other *GCounter) {
	for id, val := range other.counts {
		if val > g.counts[id] {
			g.counts[id] = val
		}
	}
}

// --- PN-Counter ---

type PNCounter struct {
	pos *GCounter
	neg *GCounter
}

func NewPNCounter(id string) *PNCounter {
	return &PNCounter{pos: NewGCounter(id), neg: NewGCounter(id)}
}

func (pn *PNCounter) Increment() { pn.pos.Increment() }
func (pn *PNCounter) Decrement() { pn.neg.Increment() }

func (pn *PNCounter) Value() int64 {
	return int64(pn.pos.Value()) - int64(pn.neg.Value())
}

func (pn *PNCounter) Merge(other *PNCounter) {
	pn.pos.Merge(other.pos)
	pn.neg.Merge(other.neg)
}

// --- LWW-Register ---

type LWWRegister struct {
	value     string
	timestamp int64
}

func NewLWWRegister() *LWWRegister {
	return &LWWRegister{}
}

func (r *LWWRegister) Set(value string, ts int64) {
	if ts > r.timestamp {
		r.value = value
		r.timestamp = ts
	}
}

func (r *LWWRegister) Get() string { return r.value }

func (r *LWWRegister) Merge(other *LWWRegister) {
	if other.timestamp > r.timestamp {
		r.value = other.value
		r.timestamp = other.timestamp
	}
}

// --- Demo ---

func main() {
	fmt.Println("=== G-Counter (Grow-Only) ===")
	nodeA := NewGCounter("A")
	nodeB := NewGCounter("B")
	nodeC := NewGCounter("C")

	nodeA.Increment()
	nodeA.Increment()
	nodeA.Increment()
	nodeB.Increment()
	nodeB.Increment()
	nodeC.Increment()

	fmt.Printf("Node A local: %d\n", nodeA.Value())
	fmt.Printf("Node B local: %d\n", nodeB.Value())
	fmt.Printf("Node C local: %d\n", nodeC.Value())

	// Merge all into A
	nodeA.Merge(nodeB)
	nodeA.Merge(nodeC)
	fmt.Printf("After merge at A: %d (expected 6)\n", nodeA.Value())

	fmt.Println("\n=== PN-Counter (Increment + Decrement) ===")
	pnA := NewPNCounter("A")
	pnB := NewPNCounter("B")

	pnA.Increment()
	pnA.Increment()
	pnA.Increment()
	pnB.Increment()
	pnB.Decrement()

	fmt.Printf("Node A: %d (3 inc)\n", pnA.Value())
	fmt.Printf("Node B: %d (1 inc, 1 dec)\n", pnB.Value())

	pnA.Merge(pnB)
	fmt.Printf("After merge: %d (expected 3: 4 inc - 1 dec)\n", pnA.Value())

	fmt.Println("\n=== LWW-Register (Last-Writer-Wins) ===")
	regA := NewLWWRegister()
	regB := NewLWWRegister()

	t1 := time.Now().UnixNano()
	t2 := t1 + 1000

	regA.Set("alice", t1)
	regB.Set("bob", t2) // later timestamp

	fmt.Printf("Node A: %q (t=%d)\n", regA.Get(), t1)
	fmt.Printf("Node B: %q (t=%d)\n", regB.Get(), t2)

	regA.Merge(regB)
	regB.Merge(regA)
	fmt.Printf("After merge A: %q\n", regA.Get())
	fmt.Printf("After merge B: %q\n", regB.Get())
	fmt.Println("Both converge to the same value (bob wins — later timestamp)")
}
