package main

import (
	"fmt"
	"os"
	"tour_of_go/concurrency"
	ctx_examples "tour_of_go/context"
	"tour_of_go/error_handling"
	"tour_of_go/flow_control_statements"
	"tour_of_go/generics"
	"tour_of_go/interfaces"
	"tour_of_go/methods"
	"tour_of_go/more_types"
	"tour_of_go/packages"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run . <topic> [example]")
		fmt.Println()
		fmt.Println("Learning Path (recommended order):")
		fmt.Println("  1. packages              - Variables, functions, types, constants")
		fmt.Println("  2. flow_control_statements - For, if, switch, defer")
		fmt.Println("  3. more_types            - Pointers, structs, slices, maps, closures")
		fmt.Println("  4. methods               - Value/pointer receivers, Stringer")
		fmt.Println("  5. interfaces            - Implicit satisfaction, type assertions, embedding")
		fmt.Println("  6. error_handling        - Custom errors, wrapping, panic/recover")
		fmt.Println("  7. generics              - Type parameters, constraints, generic types")
		fmt.Println("  8. concurrency           - Goroutines, channels, select, mutex, worker pool")
		fmt.Println("  9. context               - Cancellation, timeouts, values")
		fmt.Println()
		fmt.Println("Advanced (see more-internals/):")
		fmt.Println("  Runnable snippets: go run ./more-internals/runnable/<topic>/")
		fmt.Println("  Projects:          see projects/ directory")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run . packages                  # Run all examples in a topic")
		fmt.Println("  go run . packages basic            # Run a specific example")
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
	case "more_types":
		if example != "" {
			more_types.RunExample(example)
		} else {
			more_types.Run()
		}
	case "methods":
		if example != "" {
			methods.RunExample(example)
		} else {
			methods.Run()
		}
	case "interfaces":
		if example != "" {
			interfaces.RunExample(example)
		} else {
			interfaces.Run()
		}
	case "error_handling":
		if example != "" {
			error_handling.RunExample(example)
		} else {
			error_handling.Run()
		}
	case "generics":
		if example != "" {
			generics.RunExample(example)
		} else {
			generics.Run()
		}
	case "concurrency":
		if example != "" {
			concurrency.RunExample(example)
		} else {
			concurrency.Run()
		}
	case "context":
		if example != "" {
			ctx_examples.RunExample(example)
		} else {
			ctx_examples.Run()
		}
	default:
		fmt.Printf("Unknown topic: %s\n", topic)
		fmt.Println("Run 'go run .' to see available topics.")
		os.Exit(1)
	}
}
