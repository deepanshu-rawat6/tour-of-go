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

This is actually a perfect example for your multiple results' lesson! The swap function you have
returns multiple values, just like fmt.Println does.