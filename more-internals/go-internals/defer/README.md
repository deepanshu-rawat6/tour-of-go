# defer

The behavior of defer statements is straightforward and predictable. There are three simple rules:

### A deferred function’s arguments are evaluated when the defer statement is evaluated.

In this example, the expression “i” is evaluated when the Println call is deferred. The deferred call will print “0” after the function returns.

```go
func a() {
    i := 0
    defer fmt.Println(i)
    i++
    return
}
```

### Deferred function calls are executed in Last In First Out order after the surrounding function returns.

This function prints “3210”:

```go
func b() {
    for i := 0; i < 4; i++ {
    defer fmt.Print(i)
    }
}
```


### Deferred functions may read and assign to the returning function’s named return values.

In this example, a deferred function increments the return value i after the surrounding function returns. Thus, this function returns 2:

```go
func c() (i int) {
    defer func() { i++ }()
    return 1
}

```

## Stack Visualization for `stackingDefer()` 

src: (flow_control_statements/defer.go:25-33)

### Code

```go
func switchStatement() {
	fmt.Println("Switch Statements:")

	standardSwitch()

	fmt.Println("\nOrder of Switch Statements:")

	orderOfSwitchStatement()

	fmt.Println("\nSwitch Statements with no Conditions:")

	switchWithNoConditions()
}
```

When the function executes:

1. Print "counting"          → Output: counting
2. Loop i=0: defer Println(0) → Push to stack: [0]
3. Loop i=1: defer Println(1) → Push to stack: [0, 1]
4. Loop i=2: defer Println(2) → Push to stack: [0, 1, 2]
   ... continues ...
5. Loop i=9: defer Println(9) → Push to stack: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
6. Print "done"              → Output: done
7. Function returns          → Execute deferred calls in LIFO order

### The Defer Stack (LIFO - Last In, First Out):

TOP     → defer fmt.Println(9)  ← Executes 1st
defer fmt.Println(8)  ← Executes 2nd
defer fmt.Println(7)
defer fmt.Println(6)
defer fmt.Println(5)
defer fmt.Println(4)
defer fmt.Println(3)
defer fmt.Println(2)
defer fmt.Println(1)
BOTTOM  → defer fmt.Println(0)  ← Executes 10th (last)

### Key Points:

- Arguments are evaluated immediately: When defer fmt.Println(i) is called, the value of i is
  captured at that moment
- Execution is delayed: The actual function call waits until the surrounding function returns
- LIFO order: Like a stack of plates - last one added is the first one removed (9, 8, 7... down to 0)

This is why you see the numbers printed in reverse order (9 → 0) after "done" is printed!

## Real-World Use Case: Resource Cleanup

In production, `defer` is the #1 tool for preventing resource leaks.

### 1. Database Transactions
If you open a transaction, you **must** either commit it or roll it back.
```go
func processOrder(db *sql.DB) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback() // Ensures rollback if any error occurs later

    // ... perform logic ...

    return tx.Commit() // Rollback is skipped if Commit succeeds
}
```

### 2. Mutex Unlocking
Prevent deadlocks by ensuring the lock is released even if the function panics or returns early.
```go
func (c *Counter) Inc() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.val++
}
```