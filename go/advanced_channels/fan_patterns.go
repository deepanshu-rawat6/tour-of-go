// fan_patterns.go — fan-out and fan-in (merge) patterns.
//
// FAN-OUT:
//   One input channel → N goroutines all reading from it.
//   Workers COMPETE for items (each item goes to exactly one worker).
//   Use when: CPU-bound work that can be parallelised; order doesn't matter.
//
// FAN-IN (MERGE):
//   N input channels → one output channel.
//   Each input goroutine forwards its values to the shared output.
//   Use when: combining results from multiple independent sources.
//
// PIPELINE PATTERN:
//   gen() <-chan int → stage1 → stage2 → ... → sink
//   Each stage is a function: func(in <-chan int) <-chan int
//   Closing the input channel propagates the "done" signal downstream.
package advanced_channels

import (
	"fmt"
	"sync"
)

// fanOutExample demonstrates distributing work across N workers.
func fanOutExample() {
	fmt.Println("1. Fan-out: N workers competing for jobs")
	basicFanOut()

	fmt.Println("\n2. Pipeline: gen → sq → print")
	pipelineExample()
}

// fanInExample demonstrates merging N channels into one.
func fanInExample() {
	fmt.Println("3. Fan-in: merging N channels into one")
	basicFanIn()

	fmt.Println("\n4. Full pipeline: gen → fanOut(sq) → fanIn → sink")
	fullPipeline()
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper: gen creates a source channel from a list of values.
// ─────────────────────────────────────────────────────────────────────────────

// gen sends all values on a new channel and closes it when done.
// Closing the channel is the signal to downstream stages that no more
// values are coming. Pipeline stages use "for v := range in" to consume
// until close.
func gen(values ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out) // always close: lets downstream range loops terminate
		for _, v := range values {
			out <- v
		}
	}()
	return out
}

// sq squares each value from in and sends results downstream.
// This is the classic pipeline stage signature.
func sq(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in { // terminates when in is closed
			out <- v * v
		}
	}()
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. Basic fan-out: workers compete for items
// ─────────────────────────────────────────────────────────────────────────────

// fanOutWorker reads from jobs until it's closed, sending results to results.
// Multiple workers all reading from the same jobs channel: they COMPETE.
// There is no coordination needed — channel sends/receives are atomic.
func fanOutWorker(id int, jobs <-chan int, results chan<- string) {
	for j := range jobs {
		// Simulate work by just formatting the result
		results <- fmt.Sprintf("worker%d: job%d → %d²=%d", id, j, j, j*j)
	}
}

func basicFanOut() {
	const numWorkers = 3
	const numJobs = 7

	jobs := make(chan int, numJobs)
	results := make(chan string, numJobs)

	// Start N workers — they all block on the same jobs channel.
	// Each job goes to exactly ONE worker (whichever is free first).
	for w := 1; w <= numWorkers; w++ {
		go fanOutWorker(w, jobs, results)
	}

	// Send all jobs, then CLOSE the channel to signal workers to stop.
	// Closing is the graceful shutdown mechanism: workers' range loops exit.
	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs) // signals all workers to stop after draining remaining jobs

	// Collect all results
	for i := 0; i < numJobs; i++ {
		fmt.Printf("  %s\n", <-results)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. Simple pipeline: gen → sq → print
// ─────────────────────────────────────────────────────────────────────────────

func pipelineExample() {
	// Connect stages by passing channels between them.
	// Each stage runs in its own goroutine.
	// The pipeline is lazy: values flow only when downstream reads.
	c := gen(2, 3, 4, 5)     // source
	out := sq(sq(c))          // two squaring stages chained
	for v := range out {
		fmt.Printf("  pipeline result: %d\n", v)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. Fan-in: merge N channels into one output
// ─────────────────────────────────────────────────────────────────────────────

// fanIn merges any number of input channels into a single output channel.
// It starts one goroutine per input; all goroutines write to the shared output.
// A WaitGroup tracks when all inputs are exhausted; then output is closed.
func fanIn(inputs ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	out := make(chan int)

	// forward copies all values from one input channel to out.
	forward := func(ch <-chan int) {
		defer wg.Done()
		for v := range ch { // exits when ch is closed
			out <- v
		}
	}

	// Start one forwarding goroutine per input channel.
	wg.Add(len(inputs))
	for _, ch := range inputs {
		go forward(ch)
	}

	// Close out once all forwarding goroutines have finished.
	// This runs in a separate goroutine so fanIn returns immediately.
	go func() {
		wg.Wait()
		close(out) // signals downstream that all inputs are done
	}()

	return out
}

func basicFanIn() {
	// Three independent producers
	c1 := gen(1, 4, 7)
	c2 := gen(2, 5, 8)
	c3 := gen(3, 6, 9)

	// Merge all three into a single stream.
	// Order is non-deterministic — which goroutine runs first is up to the scheduler.
	merged := fanIn(c1, c2, c3)

	var results []int
	for v := range merged {
		results = append(results, v)
	}

	// Sort for deterministic output (fan-in is inherently unordered)
	fmt.Printf("  fan-in received %d values (order may vary): %v\n", len(results), results)
}

// ─────────────────────────────────────────────────────────────────────────────
// 4. Full pipeline: gen → fan-out(sq) → fan-in → sink
// ─────────────────────────────────────────────────────────────────────────────

// fanOutSq fans out the squaring work to N parallel workers and merges results.
// This is the classic "scatter-gather" pattern:
//   - scatter: distribute work across N goroutines (fan-out)
//   - gather:  collect results into one channel (fan-in)
func fanOutSq(in <-chan int, workers int) <-chan int {
	// Start N independent sq pipelines — all reading from the same input.
	// The input channel is shared; workers compete for values.
	// Because sq starts its own goroutine, each worker truly runs in parallel.
	stages := make([](<-chan int), workers)
	for i := 0; i < workers; i++ {
		stages[i] = sq(in) // each sq goroutine races to read from in
	}
	return fanIn(stages...) // merge N result streams back into one
}

func fullPipeline() {
	// Source: 1..8
	source := gen(1, 2, 3, 4, 5, 6, 7, 8)

	// Fan-out to 3 parallel squaring workers, then fan-in the results.
	results := fanOutSq(source, 3)

	var collected []int
	for v := range results {
		collected = append(collected, v)
	}

	// Results are unordered because workers run concurrently.
	// In production you'd either sort or use a different pattern if order matters.
	fmt.Printf("  full pipeline: %d results (unordered): %v\n", len(collected), collected)
}
