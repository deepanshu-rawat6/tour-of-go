package concurrency

import (
	"fmt"
	"sync"
)

func workerPoolExample() {
	fmt.Println("Worker Pool:")

	const numWorkers = 3
	const numJobs = 9

	jobs := make(chan int, numJobs)
	results := make(chan int, numJobs)

	// Start workers
	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range jobs {
				result := j * j // simulate work: square the job number
				fmt.Printf("  worker %d processed job %d → %d\n", id, j, result)
				results <- result
			}
		}(w)
	}

	// Send jobs
	for j := 1; j <= numJobs; j++ {
		jobs <- j
	}
	close(jobs) // signal workers: no more jobs

	// Wait for all workers then close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var total int
	for r := range results {
		total += r
	}
	fmt.Println("  sum of all results:", total)
}
