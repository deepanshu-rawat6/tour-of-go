package more_types

import (
	"reflect"
	"testing"
)

func TestCharCount(t *testing.T) {
	tests := []struct {
		input    string
		expected map[rune]int
	}{
		{"hello", map[rune]int{'h': 1, 'e': 1, 'l': 2, 'o': 1}},
		{"aaabbb", map[rune]int{'a': 3, 'b': 3}},
		{"", map[rune]int{}},
		{"👋🌍👋", map[rune]int{'👋': 2, '🌍': 1}},
	}

	for _, tc := range tests {
		result := CharCount(tc.input)
		if !reflect.DeepEqual(result, tc.expected) {
			t.Errorf("CharCount(%q) = %v; want %v", tc.input, result, tc.expected)
		}
	}
}
