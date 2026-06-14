package interfaces

import "fmt"

// Reader and Writer are small, focused interfaces (Go idiom: keep interfaces small)
type Reader interface {
	Read() string
}

type Writer interface {
	Write(s string)
}

// ReadWriter composes Reader and Writer — interface embedding
type ReadWriter interface {
	Reader
	Writer
}

type Buffer struct {
	data string
}

func (b *Buffer) Read() string      { return b.data }
func (b *Buffer) Write(s string)    { b.data += s }

func interfaceEmbeddingExample() {
	fmt.Println("Interface Embedding:")

	var rw ReadWriter = &Buffer{}
	rw.Write("hello ")
	rw.Write("world")
	fmt.Println("  ReadWriter buffer:", rw.Read())

	// Buffer satisfies Reader, Writer, AND ReadWriter — all implicitly
	var r Reader = &Buffer{data: "read-only"}
	fmt.Println("  As Reader:", r.Read())

	fmt.Println("\n  Key insight: compose small interfaces rather than one big interface")
	fmt.Println("  io.ReadWriter in stdlib is exactly this pattern")
}
