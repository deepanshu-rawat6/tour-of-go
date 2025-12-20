# Tour of Go

![go-mascot](./.img/go.png)

## Running topics:

### From the root directory

```shell
go run . packages
```

Show help

```shell
go run .
```

Adding new topics (e.g., generics):

1. Create a new directory:
   mkdir generics

2. Create generics/generics.go:
   package generics

```go
import "fmt"

// Run executes the generics example
func Run() {
fmt.Println("=== Generics Example ===")
// Your generics code here
}
```

3. Update main.go - add the import at line 6:

```go
   import (
   "fmt"
   "os"
   "tour_of_go/packages"
   "tour_of_go/generics"  // Add this
   )
```

4. Add the new case in the switch statement (around line 20):

```go
   case "generics":
   generics.Run()
```

5. Update the help text to include the new topic.


## Building go

### Build the executable:

```shell
go build .
```

This creates a `tour_of_go` executable in the root directory.

### Run the built executable:

```shell
./tour_of_go packages
./tour_of_go generics
./tour_of_go          # Shows help
```

Or specify a custom output name:

```shell
go build -o myapp .
./myapp packages
```
