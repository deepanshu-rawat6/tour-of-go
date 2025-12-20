package main

import (
	"fmt"
	"os"
	"tour_of_go/generics"
	"tour_of_go/packages"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run . <topic> [example]")
		fmt.Println("Available topics:")
		fmt.Println("  packages - Learn about packages and imports")
		fmt.Println("  generics - Learn about generics")
		fmt.Println("\nExamples:")
		fmt.Println("  go run . packages           # Run all packages examples")
		fmt.Println("  go run . packages basic     # Run specific example")
		os.Exit(1)
	}

	topic := os.Args[1]
	var example string
	if len(os.Args) >= 3 {
		example = os.Args[2]
	}

	switch topic {
	case "packages":
		if example != "" {
			packages.RunExample(example)
		} else {
			packages.Run()
		}
	case "generics":
		generics.Run()
	default:
		fmt.Printf("Unknown topic: %s\n", topic)
		fmt.Println("Available topics: packages, generics")
		os.Exit(1)
	}
}
