package concurrency

import "fmt"

func channelsExample() {
	fmt.Println("Channels:")

	// Unbuffered channel — sender blocks until receiver reads
	ch := make(chan int)
	go func() { ch <- 42 }()
	fmt.Println("  unbuffered receive:", <-ch)

	// Buffered channel — sender only blocks when buffer is full
	buf := make(chan string, 3)
	buf <- "a"
	buf <- "b"
	buf <- "c"
	fmt.Println("  buffered len:", len(buf))
	fmt.Println("  buffered receive:", <-buf)

	// Range over channel until closed
	nums := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		nums <- i
	}
	close(nums)

	fmt.Print("  range over closed channel:")
	for n := range nums {
		fmt.Print(" ", n)
	}
	fmt.Println()
}
