# fmt.Println()

### fmt.Println() signature:

```go
func Println(a ...interface{}) (n int, err error)
```

It returns two values:
- `n int` - the number of bytes written
- `err error` - any error (or `nil` if successful)

Your code breakdown:

```go
fmt.Println(fmt.Println("Mutliple results:"))
```

1. Inner call: `fmt.Println("Mutliple results:")` prints "Mutliple results:" and returns `(18, nil)`
   - 18 = number of bytes written ("Mutliple results:" + newline)
   - nil = no error
2. Outer call: `fmt.Println(18, nil)` prints those two return values

The fix:

```go
fmt.Println("Mutliple results:")  // Just one call, no nesting
```

## Real-World Context: Why do we care about `n` and `err`?

While we often ignore the return values of `fmt.Println` in simple scripts, they are critical in professional systems.

### 1. Reliable Logging
In a high-throughput system, if the disk fills up, `fmt.Println` will return an error. If you ignore it, you might lose critical audit logs without knowing.
```go
n, err := fmt.Fprintln(logFile, "Transaction completed")
if err != nil {
    // Alert the SRE team! Disk might be full or permissions failed.
}
```

### 2. Network CLI Tools
If you are writing to a network socket (which is also an `io.Writer`), checking `n` ensures all your data was actually sent.
```go
n, err := fmt.Fprintf(conn, "GET / HTTP/1.1\r\n\r\n")
if n < expectedBytes {
    // Handle partial write!
}
```