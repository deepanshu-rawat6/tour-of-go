// Package resp implements the Redis Serialization Protocol (RESP).
// Supports: Simple Strings (+), Errors (-), Integers (:), Bulk Strings ($), Arrays (*)
package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Value represents a RESP value.
type Value struct {
	Type    byte   // '+' '-' ':' '$' '*'
	Str     string // for +, -, $
	Integer int64  // for :
	Array   []Value
}

// Parse reads one RESP value from r.
func Parse(r *bufio.Reader) (Value, error) {
	b, err := r.ReadByte()
	if err != nil {
		return Value{}, err
	}
	switch b {
	case '+': // Simple String
		line, err := readLine(r)
		return Value{Type: '+', Str: line}, err
	case '-': // Error
		line, err := readLine(r)
		return Value{Type: '-', Str: line}, err
	case ':': // Integer
		line, err := readLine(r)
		if err != nil {
			return Value{}, err
		}
		n, err := strconv.ParseInt(line, 10, 64)
		return Value{Type: ':', Integer: n}, err
	case '$': // Bulk String
		line, err := readLine(r)
		if err != nil {
			return Value{}, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return Value{}, err
		}
		if n == -1 {
			return Value{Type: '$', Str: ""}, nil // null bulk string
		}
		buf := make([]byte, n+2) // +2 for \r\n
		if _, err := io.ReadFull(r, buf); err != nil {
			return Value{}, err
		}
		return Value{Type: '$', Str: string(buf[:n])}, nil
	case '*': // Array
		line, err := readLine(r)
		if err != nil {
			return Value{}, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return Value{}, err
		}
		arr := make([]Value, n)
		for i := range arr {
			arr[i], err = Parse(r)
			if err != nil {
				return Value{}, err
			}
		}
		return Value{Type: '*', Array: arr}, nil
	default:
		return Value{}, fmt.Errorf("unknown RESP type: %c", b)
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// Command extracts the command name and string arguments from an Array value.
func Command(v Value) (string, []string, error) {
	if v.Type != '*' || len(v.Array) == 0 {
		return "", nil, errors.New("expected array")
	}
	cmd := strings.ToUpper(v.Array[0].Str)
	args := make([]string, len(v.Array)-1)
	for i, a := range v.Array[1:] {
		args[i] = a.Str
	}
	return cmd, args, nil
}
