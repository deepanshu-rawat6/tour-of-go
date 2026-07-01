package io_bufio

import (
	"fmt"
	"os"
)

// Run executes all io/bufio examples
func Run() {
	fmt.Println("=== io / bufio ===")
	fmt.Println()

	basicReadersExample()
	fmt.Println()

	basicWritersExample()
	fmt.Println()

	bufioReaderExample()
	fmt.Println()

	bufioWriterExample()
	fmt.Println()

	streamingExample()
	fmt.Println()

	pipesExample()
	fmt.Println()
}

// RunExample runs a specific io/bufio example by name
func RunExample(name string) {
	fmt.Printf("=== io/bufio: %s ===\n\n", name)

	switch name {
	case "readers":
		basicReadersExample()
	case "writers":
		basicWritersExample()
	case "bufio-reader":
		bufioReaderExample()
	case "bufio-writer":
		bufioWriterExample()
	case "streaming":
		streamingExample()
	case "pipes":
		pipesExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  readers      - io.Reader interface and implementations")
		fmt.Println("  writers      - io.Writer interface and implementations")
		fmt.Println("  bufio-reader - buffered reading, ReadString, Scanner")
		fmt.Println("  bufio-writer - buffered writing and Flush")
		fmt.Println("  streaming    - composing readers/writers (TeeReader, MultiWriter)")
		fmt.Println("  pipes        - io.Pipe for connecting producer/consumer goroutines")
		os.Exit(1)
	}
}
