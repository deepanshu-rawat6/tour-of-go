package main

import (
	"fmt"
	"sync"
)

// ── Pipeline Pattern ──────────────────────────────────────────────────────────
// Each stage receives from an input channel and sends to an output channel.

func generate(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		for _, n := range nums {
			out <- n
		}
		close(out)
	}()
	return out
}

func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		for n := range in {
			out <- n * n
		}
		close(out)
	}()
	return out
}

func double(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		for n := range in {
			out <- n * 2
		}
		close(out)
	}()
	return out
}

// ── Fan-out / Fan-in ──────────────────────────────────────────────────────────
// Fan-out: distribute work across multiple goroutines.
// Fan-in: merge multiple result channels into one.

func fanOut(in <-chan int, workers int) []<-chan int {
	channels := make([]<-chan int, workers)
	for i := 0; i < workers; i++ {
		channels[i] = square(in) // each worker reads from the same input
	}
	return channels
}

func merge(channels ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	out := make(chan int, 10)

	drain := func(c <-chan int) {
		defer wg.Done()
		for v := range c {
			out <- v
		}
	}

	wg.Add(len(channels))
	for _, c := range channels {
		go drain(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func main() {
	fmt.Println("=== Pipeline Pattern ===")
	// generate → square → double
	src := generate(1, 2, 3, 4, 5)
	squared := square(src)
	doubled := double(squared)

	fmt.Print("generate(1..5) → square → double: ")
	for v := range doubled {
		fmt.Print(v, " ")
	}
	fmt.Println()

	fmt.Println("\n=== Fan-out / Fan-in ===")
	// Two workers both squaring from the same source
	src2 := generate(10, 20, 30, 40)
	w1 := square(src2)
	// Note: for true fan-out you'd split the source; here we show merge
	extra := generate(100, 200)
	w2 := square(extra)

	fmt.Print("merged results: ")
	results := []int{}
	for v := range merge(w1, w2) {
		results = append(results, v)
	}
	// sort for deterministic output
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i] > results[j] {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	fmt.Println(results)
}
