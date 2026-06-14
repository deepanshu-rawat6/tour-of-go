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

	variablesWithInitializersExample()
	fmt.Println()

	varsTypes()
	fmt.Println()

	typeConversions()
	fmt.Println()

	typeInference()
	fmt.Println()

	constants()
	fmt.Println()

	numericConstants()
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
	case "variables-with-initializers":
		variablesWithInitializersExample()
	case "vars-types":
		varsTypes()
	case "type-conversions":
		typeConversions()
	case "type-inference":
		typeInference()
	case "constants":
		constants()
	case "numeric-constants":
		numericConstants()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  basic")
		fmt.Println("  exported-names")
		fmt.Println("  functions")
		fmt.Println("  multiple-results")
		fmt.Println("  named-results")
		fmt.Println("  variables")
		fmt.Println("  variables-with-initializers")
		fmt.Println("  vars-types")
		fmt.Println("  type-conversions")
		fmt.Println("  type-inference")
		fmt.Println("  constants")
		fmt.Println("  numeric-constants")
		os.Exit(1)
	}
}
