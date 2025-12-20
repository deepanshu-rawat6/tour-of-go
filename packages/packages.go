package packages

import "fmt"

// Run executes all examples in the packages topic
func Run() {
	fmt.Println("=== Packages & Imports ===")
	fmt.Println()

	basicExample()
	fmt.Println()

	exportedNamesExample()
	fmt.Println()

	functions()
	fmt.Println()

	mutlipleResults()
	fmt.Println()

}
