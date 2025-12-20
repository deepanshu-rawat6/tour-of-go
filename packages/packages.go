package packages

import (
	"fmt"
	"os"
)

// Run executes all examples in the packages topic
func Run() {
	fmt.Println("=== Packages & Imports ===")
	fmt.Println()

	basicExample()
	fmt.Println()

	exportedNamesExample()
	fmt.Println()

	functionsExample()
	fmt.Println()

	mutlipleResultsExample()
	fmt.Println()

	namedResultsExample()
	fmt.Println()

	variablesExample()
	fmt.Println()

}

// RunExample runs a specific example by name
func RunExample(name string) {
	fmt.Printf("=== Packages & Imports: %s ===\n\n", name)

	switch name {
	case "basic":
		basicExample()
	case "exported-names":
		exportedNamesExample()
	case "functions":
		functionsExample()
	case "multiple-results":
		mutlipleResultsExample()
	case "named-results":
		namedResultsExample()
	case "variables":
		variablesExample()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  basic")
		fmt.Println("  exported-names")
		fmt.Println("  functions")
		fmt.Println("  multiple-results")
		fmt.Println("  named-results")
		fmt.Println("  variables")
		os.Exit(1)
	}
}
