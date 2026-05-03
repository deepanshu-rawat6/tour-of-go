package resp_test

import (
	"bufio"
	"strings"
	"testing"

	"tour_of_go/projects/from-scratch/07-distributed-cache/internal/resp"
)

func parse(t *testing.T, s string) resp.Value {
	t.Helper()
	v, err := resp.Parse(bufio.NewReader(strings.NewReader(s)))
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return v
}

func TestParse_SimpleString(t *testing.T) {
	v := parse(t, "+OK\r\n")
	if v.Type != '+' || v.Str != "OK" {
		t.Fatalf("got %+v", v)
	}
}

func TestParse_BulkString(t *testing.T) {
	v := parse(t, "$5\r\nhello\r\n")
	if v.Type != '$' || v.Str != "hello" {
		t.Fatalf("got %+v", v)
	}
}

func TestParse_Integer(t *testing.T) {
	v := parse(t, ":42\r\n")
	if v.Type != ':' || v.Integer != 42 {
		t.Fatalf("got %+v", v)
	}
}

func TestParse_Array(t *testing.T) {
	v := parse(t, "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n")
	if v.Type != '*' || len(v.Array) != 3 {
		t.Fatalf("got %+v", v)
	}
	cmd, args, err := resp.Command(v)
	if err != nil || cmd != "SET" || args[0] != "foo" || args[1] != "bar" {
		t.Fatalf("cmd=%s args=%v err=%v", cmd, args, err)
	}
}

func TestParse_NullBulk(t *testing.T) {
	v := parse(t, "$-1\r\n")
	if v.Type != '$' || v.Str != "" {
		t.Fatalf("got %+v", v)
	}
}
