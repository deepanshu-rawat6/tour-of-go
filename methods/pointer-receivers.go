package methods

import "fmt"

type Counter struct {
	value int
}

// Inc uses a pointer receiver — it can mutate the original struct.
// Use pointer receivers when: (1) you need to mutate, or (2) the struct is large.
func (c *Counter) Inc() {
	c.value++
}

func (c *Counter) Reset() {
	c.value = 0
}

// Value returns the current count. Could be value receiver, but we use pointer
// for consistency — mixing receiver types on the same type is bad practice.
func (c *Counter) Value() int {
	return c.value
}

func pointerReceiversExample() {
	fmt.Println("Pointer Receivers:")

	c := &Counter{}
	c.Inc()
	c.Inc()
	c.Inc()
	fmt.Println("  After 3 Inc():", c.Value())

	c.Reset()
	fmt.Println("  After Reset():", c.Value())

	// Go auto-dereferences: you can call pointer methods on addressable values
	c2 := Counter{}
	c2.Inc() // Go rewrites this as (&c2).Inc()
	fmt.Println("\n  Auto-dereference: c2.Inc() works even on non-pointer:", c2.Value())
}
