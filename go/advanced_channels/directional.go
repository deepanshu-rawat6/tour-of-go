// directional.go — directional (typed) channels.
//
// Go channels can be restricted to send-only or receive-only at the type level:
//
//	chan<- T   send-only channel (you can only send into it)
//	<-chan T   receive-only channel (you can only receive from it)
//
// WHY BOTHER?
// 1. Self-documenting APIs: the signature tells you who produces and who consumes.
// 2. Compile-time enforcement: the compiler prevents misuse.
// 3. Ownership clarity: the creator holds the full chan T; workers get only the
//    restricted direction they need.
//
// A bidirectional chan T is implicitly convertible to either direction.
// The reverse (direction → bidirectional) is NOT allowed.
package advanced_channels

import "fmt"

// directionalExample runs all sub-demos.
func directionalExample() {
	fmt.Println("1. Basic send-only and receive-only usage")
	basicDirectional()

	fmt.Println("\n2. Pipeline stage with directional channels")
	pipelineStageExample()

	fmt.Println("\n3. Ownership: creator owns bidirectional, passes directional")
	ownershipExample()
}

// ─────────────────────────────────────────────────────────────────────────────
// 1. Basic directional channel usage
// ─────────────────────────────────────────────────────────────────────────────

// sendOnly accepts a send-only channel.
// You can ONLY send to it; attempting to receive is a compile error:
//
//	v := <-out  // ERROR: invalid operation: cannot receive from send-only channel
func sendOnly(out chan<- int, values []int) {
	for _, v := range values {
		out <- v
	}
	close(out) // closing a send-only channel is allowed
}

// receiveOnly accepts a receive-only channel.
// You can ONLY receive from it; attempting to send is a compile error:
//
//	out <- 42  // ERROR: invalid operation: cannot send to receive-only channel
func receiveOnly(in <-chan int) []int {
	var result []int
	for v := range in {
		result = append(result, v)
	}
	return result
}

func basicDirectional() {
	// The creator holds the bidirectional channel and controls its lifetime.
	ch := make(chan int, 5)

	// Pass restricted views to the functions that need them.
	// Go converts chan T → chan<- T and chan T → <-chan T automatically.
	sendOnly(ch, []int{10, 20, 30})
	result := receiveOnly(ch)

	fmt.Printf("  received: %v\n", result)
}

// ─────────────────────────────────────────────────────────────────────────────
// 2. Pipeline stages with directional channels
// ─────────────────────────────────────────────────────────────────────────────

// generator produces integers 1..n on a receive-only channel.
// Callers can only read from it — they cannot accidentally send values back.
//
// The pattern "function returns <-chan T" means:
//   - the function starts a goroutine internally
//   - the caller is the consumer, not the producer
func generator(nums ...int) <-chan int {
	out := make(chan int) // full bidirectional inside
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n
		}
	}()
	return out // returned as <-chan int (receive-only to callers)
}

// square is a pipeline stage: receives from in, sends to returned channel.
// Signature makes the data flow explicit:
//   - in  <-chan int  → this stage is a CONSUMER of int
//   - <-chan int (return) → this stage is a PRODUCER of int
func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in { // range on a channel reads until close
			out <- v * v
		}
	}()
	return out
}

// filter keeps only values satisfying the predicate.
// The pred function type uses a receive-only input — pure data transformation.
func filter(in <-chan int, pred func(int) bool) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			if pred(v) {
				out <- v
			}
		}
	}()
	return out
}

func pipelineStageExample() {
	// Pipeline: generator → square → filter (keep > 10) → print
	nums := generator(1, 2, 3, 4, 5)
	squared := square(nums)
	bigSquares := filter(squared, func(v int) bool { return v > 10 })

	fmt.Print("  squares > 10: ")
	for v := range bigSquares {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	// Type signatures in the pipeline make the data flow self-documenting:
	//   generator: () → <-chan int          (source)
	//   square:   <-chan int → <-chan int   (transformer)
	//   filter:   <-chan int → <-chan int   (transformer)
	//   for range:             <-chan int   (sink)
}

// ─────────────────────────────────────────────────────────────────────────────
// 3. Ownership: creator holds bidirectional, passes directional
// ─────────────────────────────────────────────────────────────────────────────

// worker represents a goroutine that consumes jobs and produces results.
// It only receives jobs and only sends results — it cannot close the job
// channel because it only has the receive-only view.
func worker(id int, jobs <-chan int, results chan<- string) {
	for j := range jobs {
		// Worker can receive from jobs (ok) and send to results (ok).
		// Worker CANNOT: close(jobs) — compile error: cannot close receive-only channel
		// Worker CANNOT: jobs <- 99  — compile error: cannot send to receive-only channel
		// Worker CANNOT: <-results   — compile error: cannot receive from send-only channel
		results <- fmt.Sprintf("worker%d processed job%d", id, j)
	}
}

func ownershipExample() {
	// The orchestrator creates and OWNS both channels (bidirectional).
	jobs := make(chan int, 3)
	results := make(chan string, 9)

	// Start 3 workers, each receiving restricted views.
	// The compiler enforces that workers cannot close the jobs channel or
	// accidentally read from the results channel.
	for i := 1; i <= 3; i++ {
		go worker(i, jobs, results) // chan int → <-chan int, chan string → chan<- string
	}

	// The owner sends jobs and closes the channel.
	// Only the owner can close because only the owner has the full chan.
	for j := 1; j <= 6; j++ {
		jobs <- j
	}
	close(jobs) // safe: orchestrator owns jobs

	// Collect results
	for i := 0; i < 6; i++ {
		fmt.Printf("  %s\n", <-results)
	}
}
