package main

import (
	"fmt"
	"hash/fnv"
	"math"
)

// --- Bloom Filter ---

type BloomFilter struct {
	bits    []bool
	size    uint
	hashNum uint
}

func NewBloomFilter(expectedItems uint, fpRate float64) *BloomFilter {
	m := optimalSize(expectedItems, fpRate)
	k := optimalHashCount(m, expectedItems)
	return &BloomFilter{bits: make([]bool, m), size: m, hashNum: k}
}

func optimalSize(n uint, p float64) uint {
	return uint(math.Ceil(-float64(n) * math.Log(p) / (math.Log(2) * math.Log(2))))
}

func optimalHashCount(m, n uint) uint {
	return uint(math.Ceil(float64(m) / float64(n) * math.Log(2)))
}

func (bf *BloomFilter) Add(item string) {
	for _, pos := range bf.hashes(item) {
		bf.bits[pos] = true
	}
}

func (bf *BloomFilter) Contains(item string) bool {
	for _, pos := range bf.hashes(item) {
		if !bf.bits[pos] {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) hashes(item string) []uint {
	h := fnv.New64a()
	h.Write([]byte(item))
	sum := h.Sum64()
	h1 := uint(sum)
	h2 := uint(sum >> 32)

	positions := make([]uint, bf.hashNum)
	for i := uint(0); i < bf.hashNum; i++ {
		positions[i] = (h1 + i*h2) % bf.size
	}
	return positions
}

// --- HyperLogLog ---

type HyperLogLog struct {
	registers []uint8
	p         uint8 // precision (log2 of register count)
	m         uint32
}

func NewHyperLogLog(precision uint8) *HyperLogLog {
	m := uint32(1) << precision
	return &HyperLogLog{registers: make([]uint8, m), p: precision, m: m}
}

func (hll *HyperLogLog) Add(item string) {
	h := fnv.New64a()
	h.Write([]byte(item))
	x := h.Sum64()

	// Use lower p bits for register index
	idx := x & uint64(hll.m-1)
	// Use upper bits to count leading zeros
	w := x >> hll.p
	// rho: position of leftmost 1-bit (1-indexed)
	rho := uint8(1)
	for bit := uint8(0); bit < (64 - hll.p); bit++ {
		if w&(1<<(63-hll.p-bit)) != 0 {
			break
		}
		rho++
	}

	if rho > hll.registers[idx] {
		hll.registers[idx] = rho
	}
}

func (hll *HyperLogLog) Count() uint64 {
	// Alpha constant
	var alpha float64
	switch {
	case hll.m == 16:
		alpha = 0.673
	case hll.m == 32:
		alpha = 0.697
	case hll.m == 64:
		alpha = 0.709
	default:
		alpha = 0.7213 / (1.0 + 1.079/float64(hll.m))
	}

	sum := 0.0
	zeros := 0
	for _, val := range hll.registers {
		sum += 1.0 / float64(uint64(1)<<val)
		if val == 0 {
			zeros++
		}
	}

	estimate := alpha * float64(hll.m) * float64(hll.m) / sum

	// Small range correction (linear counting)
	if estimate <= 2.5*float64(hll.m) && zeros > 0 {
		estimate = float64(hll.m) * math.Log(float64(hll.m)/float64(zeros))
	}

	return uint64(math.Round(estimate))
}

// --- Demo ---

func main() {
	fmt.Println("=== Bloom Filter Demo ===")
	bf := NewBloomFilter(1000, 0.01) // 1% false positive rate
	fmt.Printf("Bit array size: %d, Hash functions: %d\n", bf.size, bf.hashNum)

	items := []string{"user:1", "user:2", "user:3", "session:abc", "order:42"}
	for _, item := range items {
		bf.Add(item)
	}

	fmt.Println("\nMembership tests:")
	for _, test := range []string{"user:1", "user:99", "order:42", "nonexistent"} {
		fmt.Printf("  Contains(%q) = %v\n", test, bf.Contains(test))
	}

	// Measure actual false positive rate
	fp := 0
	tests := 10000
	for i := 0; i < tests; i++ {
		key := fmt.Sprintf("random:%d", i)
		if bf.Contains(key) {
			fp++
		}
	}
	fmt.Printf("\nFalse positive rate: %d/%d = %.2f%%\n", fp, tests, float64(fp)/float64(tests)*100)

	// HyperLogLog demo
	fmt.Println("\n=== HyperLogLog Demo ===")
	hll := NewHyperLogLog(14) // 16384 registers, ~0.81% error

	// Add 100,000 unique items
	n := 100000
	for i := 0; i < n; i++ {
		hll.Add(fmt.Sprintf("item:%d", i))
	}

	estimate := hll.Count()
	errorPct := math.Abs(float64(estimate)-float64(n)) / float64(n) * 100
	fmt.Printf("Actual unique items: %d\n", n)
	fmt.Printf("HLL estimate:        %d\n", estimate)
	fmt.Printf("Error:               %.2f%%\n", errorPct)
	fmt.Printf("Memory used:         %d bytes (vs %d bytes for exact set)\n", hll.m, n*8)
}
