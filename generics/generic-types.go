package generics

import "fmt"

// Stack is a generic LIFO data structure.
// T can be any type — the compiler generates a concrete version per usage.
type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, true
}

func (s *Stack[T]) Len() int { return len(s.items) }

func genericTypesExample() {
	fmt.Println("Generic Stack[int]:")
	var intStack Stack[int]
	intStack.Push(10)
	intStack.Push(20)
	intStack.Push(30)
	for intStack.Len() > 0 {
		v, _ := intStack.Pop()
		fmt.Println("  popped:", v)
	}

	fmt.Println("\nGeneric Stack[string]:")
	var strStack Stack[string]
	strStack.Push("go")
	strStack.Push("generics")
	v, _ := strStack.Pop()
	fmt.Println("  popped:", v)
}
