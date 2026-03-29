# Go "Twisters" & Quirks

Subtle behaviors in Go that often surprise new and experienced developers.

## 1. Nil Interfaces are Not (Always) Nil
A `nil` interface is only `nil` if both its **type** and **value** are `nil`. If you assign a `nil` pointer to an interface, the interface is no longer `nil`.

### Example
```go
var p *int = nil
var i interface{} = p

if i == nil {
    // This will NOT be reached
}
```

### Real-World Danger: Custom Error Handling
This is a frequent bug when returning custom error pointers.
```go
func returnError() error {
    var p *MyError = nil
    return p // Returns an interface with type *MyError and nil value
}

err := returnError()
if err != nil {
    // This is TRUE! Even though we returned nil, err itself is NOT nil.
}
```

## 2. Closure Variable Shadowing (Pre-Go 1.22)
*Note: This was fixed in Go 1.22, but still relevant for understanding older code.*
Inside a loop, the loop variable is often reused. If you capture it in a goroutine, all goroutines might see the final value.

### Example
```go
for i := 0; i < 3; i++ {
    go func() {
        fmt.Println(i) // Might print 3, 3, 3 instead of 0, 1, 2
    }()
}
```
**Fix:** Pass the variable as an argument: `go func(val int) { ... }(i)`

## 3. Slice Capacity vs Length
Appending to a slice might or might not modify the original underlying array, depending on if it has enough capacity.

### Example
```go
s1 := []int{1, 2, 3}
s2 := s1[:2]
s2 = append(s2, 10) 

fmt.Println(s1) // prints [1, 2, 10]! 
// s2 had capacity for 3, so append reused the underlying array of s1.
```

## 4. Map Iteration Order is Random
Go maps do not maintain order. In fact, Go intentionally randomizes the iteration order to prevent developers from relying on it.

### Real-World Danger: Deterministic Tests
If you write a test that expects a JSON output of a map in a specific order, it will pass on one machine and fail on another.
**Fix:** If you need order, sort the keys first.


## 5. iota Bitmasking
`iota` is more than a simple counter. It's often used for flags and bitmasks.

### Example
```go
const (
    Read = 1 << iota // 1 (0001)
    Write            // 2 (0010)
    Exec             // 4 (0100)
)
// Read | Write results in 3 (0011)
```

