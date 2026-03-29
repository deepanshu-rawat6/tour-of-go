# Error Handling Mastery: Beyond the Boilerplate

In production Go code, `if err != nil` is just the beginning. You need to know **where** the error happened, **what** caused it, and **if** it should be retried.

## 1. Error Wrapping (Go 1.13+)
Use the `%w` verb with `fmt.Errorf` to "wrap" an error. This allows you to add context while still allowing the original error to be detected later.

### Go Snippet (Wrapping & unwrapping)
```go
// Wrapping
if err := db.Connect(); err != nil {
    return fmt.Errorf("failed to process order: %w", err)
}

// Checking (errors.Is)
if errors.Is(err, sql.ErrNoRows) {
    // We specifically want to handle the 'Not Found' case
}

// Extracting (errors.As)
var mysqlErr *mysql.MySQLError
if errors.As(err, &mysqlErr) {
    // We now have access to DB-specific fields like ErrorCode
    fmt.Println(mysqlErr.Number) 
}
```

## 2. Platform Pattern: Retryable vs. Fatal Errors
In a job scheduler or worker, you need to know if an error is a "Temporary Glitch" or a "Final Failure."

### Example: Custom Error Types
```go
type PlatformError struct {
    Code      string
    Message   string
    Retryable bool
}

func (e *PlatformError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Real-World Use Case:
func processMessage() error {
    if networkTimeout {
        return &PlatformError{Code: "NET_FAIL", Message: "Timeout", Retryable: true}
    }
    if invalidJSON {
        return &PlatformError{Code: "INVALID_DATA", Message: "Bad JSON", Retryable: false}
    }
    return nil
}
```

## 3. Real-World Example: HashiCorp Vault
Tools like **Vault** or **Terraform** use rich error types to communicate status back to the CLI. For example, Vault returns a `403 Forbidden` wrapped in a Go error that the CLI then uses to trigger a re-authentication flow.

## 4. Engineering Tip: Never Log and Return
A common mistake is:
```go
if err != nil {
    log.Println(err) // WRONG
    return err
}
```
**Why?** This results in duplicate logs. Log the error **only once** at the top-level handler or entry point.
