package more_types

import (
	"fmt"
	"strings"

	"golang.org/x/tour/wc"
)

/*
A map maps keys to values.

The zero value of a map is nil. A nil map has no keys, nor can keys be added.

The make function returns a map of the given type, initialized and ready for use.
*/

type Vert struct {
	Lat, Long float64
}

var m map[string]Vert

func mapsIntroduction() {
	m = make(map[string]Vert)
	m["Dehradun"] = Vert{
		30.3165, 78.0322,
	}
	fmt.Printf("Co-ordinates of Dehradun: %f\n", m["Dehradun"])
}

// Map literals are like struct literals, but the keys are required.
var cod = map[string]Vert{
	"Dehradun": Vert{
		30.3165, 78.0322,
	},
	"Google": Vert{
		37.42202, -122.08408,
	},
}

func mapLiterals() {
	fmt.Println(cod)
}

// If the top-level type is just a type name, you can omit it from the elements of the literal.

var cod2 = map[string]Vert{
	"Bell Labs": {40.68433, -74.39967},
	"Google":    {37.42202, -122.08408},
}

func mapLiteralsCond() {
	fmt.Println(cod2)
}

/*
Insert or update an element in map m:

m[key] = elem
Retrieve an element:

elem = m[key]
Delete an element:

delete(m, key)
Test that a key is present with a two-value assignment:

elem, ok = m[key]
If key is in m, ok is true. If not, ok is false.

If key is not in the map, then elem is the zero value for the map's element type.

Note: If elem or ok have not yet been declared you could use a short declaration form:

elem, ok := m[key]
*/

func mutatingMaps() {
	println("Updating/Inserting element in the map")
	cod["Google"] = Vert{30.3165, 78.0322}
	fmt.Println("Updated co-ordinates of Google:", cod["Google"])

	println("Deleting Google co-ordinates from map cod")
	delete(cod, "Google")
	fmt.Println("The value: ", m["Google"])

	fmt.Println("Checking if key Dehradun exists or not?")
	v, ok := cod["Dehradun"]
	fmt.Println("The value: ", v, "Present ?", ok)
}

/*
Implement WordCount. It should return a map of the counts of each “word” in the string s.
The wc.Test function runs a test suite against the provided function and prints success or failure.
*/

func WordCount(s string) map[string]int {
	m := make(map[string]int)
	for _, word := range strings.Fields(s) {
		m[word]++
	}
	return m
}

/*
  While a 26-element array like [26]int is more memory-efficient if you only care about lowercase 'a'-'z',
  a map is generally preferred in Go to properly support UTF-8 characters.
*/

func CharCount(s string) map[rune]int {
	m := make(map[rune]int)
	for _, r := range s {
		m[r]++
	}
	return m
}

func exerciseMaps() {
	fmt.Println("Testing WordCount:")
	wc.Test(WordCount)

	/*
	   Note that since Go represents rune as its integer Unicode code point, fmt displays them as numbers
	   (e.g., 104 for 'h'). If you prefer seeing the characters themselves, we can use string(rune) when
	   printing, but the current output accurately shows the character-to-count mapping.
	*/

	fmt.Println("\nTesting CharCount:")
	samples := []string{"hello", "Go! 👋"}
	for _, s := range samples {
		fmt.Printf("CharCount(%q):\n", s)
		m := CharCount(s)
		for r, n := range m {
			fmt.Printf("  %q (%d): %d\n", r, r, n)
		}
	}
}

func mapsExample() {
	fmt.Println("Maps in Go:")
	mapsIntroduction()

	fmt.Println("Map Literals:")
	mapLiterals()

	fmt.Println("Map Literals Cond.")
	mapLiteralsCond()

	fmt.Println("Mutating Maps:")
	mutatingMaps()

	fmt.Println("Exercise Maps:")
	exerciseMaps()

}
