package concurrency

import (
	"fmt"
	"time"
)

func selectExample() {
	fmt.Println("Select statement:")

	ch1 := make(chan string, 1)
	ch2 := make(chan string, 1)

	go func() {
		time.Sleep(1 * time.Millisecond)
		ch1 <- "from ch1"
	}()
	go func() {
		time.Sleep(2 * time.Millisecond)
		ch2 <- "from ch2"
	}()

	// Receive from whichever channel is ready first
	for i := 0; i < 2; i++ {
		select {
		case msg := <-ch1:
			fmt.Println("  received:", msg)
		case msg := <-ch2:
			fmt.Println("  received:", msg)
		}
	}

	// Default case — non-blocking check
	fmt.Println("\n  Non-blocking select with default:")
	empty := make(chan int)
	select {
	case v := <-empty:
		fmt.Println("  got:", v)
	default:
		fmt.Println("  channel empty, took default branch")
	}

	// Timeout pattern
	fmt.Println("\n  Timeout pattern:")
	slow := make(chan string)
	select {
	case msg := <-slow:
		fmt.Println("  got:", msg)
	case <-time.After(5 * time.Millisecond):
		fmt.Println("  timed out waiting for slow channel")
	}
}
