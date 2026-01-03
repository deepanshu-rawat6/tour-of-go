package main

import (
	"fmt"
	"os"
	"tour_of_go/flow_control_statements"
	"tour_of_go/generics"
	more_types "tour_of_go/more-types"
	"tour_of_go/packages"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run . <topic> [example]")
		fmt.Println("Available topics:")
		fmt.Println("  packages - Learn about packages and imports")
		fmt.Println("  flow control statements - Learn about loops, if statements, etc")
		fmt.Println("  more-types - pointers, struct, etc")
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
	case "flow_control_statements":
		if example != "" {
			flow_control_statements.RunExample(example)
		} else {
			flow_control_statements.Run()
		}
	case "more-types":
		if example != "" {
			more_types.RunExample(example)
		} else {
			more_types.Run()
		}
	case "generics":
		generics.Run()
	default:
		fmt.Printf("Unknown topic: %s\n", topic)
		fmt.Println("Available topics: packages, flow_control_statements, more-types, generics")
		os.Exit(1)
	}
}
