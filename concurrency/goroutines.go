package concurrency

import (
	"fmt"
	"sync"
)

func goroutinesExample() {
	fmt.Println("Goroutines and WaitGroup:")

	var wg sync.WaitGroup

	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Printf("  goroutine %d running\n", id)
		}(i) // pass i as argument to avoid closure capture issue
	}

	wg.Wait()
	fmt.Println("  all goroutines done")
}
