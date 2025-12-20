package main

import (
	"fmt"
	"os"
	"tour_of_go/generics"
	"tour_of_go/packages"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run . <topic>")
		fmt.Println("Available topics:")
		fmt.Println("  packages - Learn about packages and imports")
		fmt.Println("\nExample: go run . packages")
		os.Exit(1)
	}

	topic := os.Args[1]

	switch topic {
	case "packages":
		packages.Run()
	case "generics":
		generics.Run()
	default:
		fmt.Printf("Unknown topic: %s\n", topic)
		fmt.Println("Available topics: packages")
		os.Exit(1)
	}
}
