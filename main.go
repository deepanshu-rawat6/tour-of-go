package main

import (
	"fmt"
	"os"
	"tour_of_go/go/advanced_channels"
	"tour_of_go/go/concurrency"
	ctx_examples "tour_of_go/go/context"
	"tour_of_go/go/encoding"
	"tour_of_go/go/error_handling"
	"tour_of_go/go/flow_control_statements"
	"tour_of_go/go/generics"
	"tour_of_go/go/interfaces"
	"tour_of_go/go/io_bufio"
	"tour_of_go/go/iterators"
	"tour_of_go/go/methods"
	"tour_of_go/go/more_types"
	net_http "tour_of_go/go/net_http"
	"tour_of_go/go/packages"
	"tour_of_go/go/stdlib_collections"
	"tour_of_go/go/sync_deep_dive"
	"tour_of_go/go/unsafe_pkg"
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
		fmt.Println(" 10. encoding              - JSON marshal/unmarshal, streaming, custom types")
		fmt.Println(" 11. net_http              - Handlers, middleware, transport, client")
		fmt.Println(" 12. sync_deep_dive        - Once, Cond, RWMutex, Pool, errgroup pattern")
		fmt.Println(" 13. unsafe_pkg            - Pointer arithmetic, struct layout, reflect+unsafe")
		fmt.Println(" 14. stdlib_collections    - slices, maps, cmp packages (Go 1.21+)")
		fmt.Println(" 15. advanced_channels     - Done channels, nil channels, fan-out/fan-in, directional")
		fmt.Println(" 16. iterators             - range-over-func, iter.Seq, pull iterators (Go 1.23+)")
		fmt.Println(" 17. io_bufio              - io.Reader/Writer, bufio, streaming, pipes")
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
	case "encoding":
		if example != "" {
			encoding.RunExample(example)
		} else {
			encoding.Run()
		}
	case "net_http":
		if example != "" {
			net_http.RunExample(example)
		} else {
			net_http.Run()
		}
	case "advanced_channels":
		if example != "" {
			advanced_channels.RunExample(example)
		} else {
			advanced_channels.Run()
		}
	case "iterators":
		if example != "" {
			iterators.RunExample(example)
		} else {
			iterators.Run()
		}
	case "sync_deep_dive":
		if example != "" {
			sync_deep_dive.RunExample(example)
		} else {
			sync_deep_dive.Run()
		}
	case "unsafe_pkg":
		if example != "" {
			unsafe_pkg.RunExample(example)
		} else {
			unsafe_pkg.Run()
		}
	case "stdlib_collections":
		if example != "" {
			stdlib_collections.RunExample(example)
		} else {
			stdlib_collections.Run()
		}
	case "io_bufio":
		if example != "" {
			io_bufio.RunExample(example)
		} else {
			io_bufio.Run()
		}
	default:
		fmt.Printf("Unknown topic: %s\n", topic)
		fmt.Println("Run 'go run .' to see available topics.")
		os.Exit(1)
	}
}
