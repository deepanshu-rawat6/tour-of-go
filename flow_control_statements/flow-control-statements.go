package flow_control_statements

import (
	"fmt"
	"os"
)

// Run executes all examples in the packages topic
func Run() {
	fmt.Println("=== Flow Control Statements ===")
	fmt.Println()

	forLoop()
	fmt.Println()

	ifStatement()
	fmt.Println()

	exerciseLoopsAndFunctions()
	fmt.Println()

	switchStatement()
	fmt.Println()

	deferStatement()
	fmt.Println()
}

// RunExample runs a specific example by name
func RunExample(name string) {
	fmt.Printf("=== Packages & Imports: %s ===\n\n", name)

	switch name {
	case "for-loop":
		forLoop()
	case "if-statement":
		ifStatement()
	case "exercise-loops-and-functions":
		exerciseLoopsAndFunctions()
	case "switch-statements":
		switchStatement()
	case "defer-statement":
		deferStatement()
	default:
		fmt.Printf("Unknown example: %s\n", name)
		fmt.Println("\nAvailable examples:")
		fmt.Println("  for")
		fmt.Println("  if")
		fmt.Println("  exercise-loops-and-functions")
		fmt.Println("  switch")
		fmt.Println("  defer")
		os.Exit(1)
	}
}
