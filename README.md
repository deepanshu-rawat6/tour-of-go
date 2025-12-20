# Tour of Go

![go-mascot](./.img/go.png)

## Running Topics

### Run all examples in a topic

```shell
go run . packages
go run . generics
```

### Run a specific example

```shell
go run . packages basic
go run . packages exported-names
go run . packages functions
go run . packages multiple-results
```

### Show help

```shell
go run .
```

## Building

### Build the executable

```shell
go build .
```

This creates a `tour_of_go` executable in the root directory.

### Run the built executable

```shell
./tour_of_go packages
./tour_of_go packages basic
./tour_of_go          # Shows help
```

Or specify a custom output name:

```shell
go build -o myapp .
./myapp packages
```

## Adding New Topics

### 1. Create a new directory

```shell
mkdir generics
```

### 2. Create the main file with Run() function

Create `generics/generics.go`:

```go
package generics

import "fmt"

// Run executes all examples in the generics topic
func Run() {
    fmt.Println("=== Generics ===")
    fmt.Println()

    basicGenericsExample()
    fmt.Println()
}

// RunExample runs a specific example by name
func RunExample(name string) {
    fmt.Printf("=== Generics: %s ===\n\n", name)

    switch name {
    case "basic":
        basicGenericsExample()
    default:
        fmt.Printf("Unknown example: %s\n", name)
        os.Exit(1)
    }
}
```

### 3. Add example files

Create `generics/basic.go`:

```go
package generics

import "fmt"

func basicGenericsExample() {
    fmt.Println("Basic Generics:")
    // Your code here
}
```

### 4. Update main.go

Add the import:

```go
import (
    "fmt"
    "os"
    "tour_of_go/packages"
    "tour_of_go/generics"  // Add this
)
```

Add the case in the switch statement:

```go
case "generics":
    if example != "" {
        generics.RunExample(example)
    } else {
        generics.Run()
    }
```

### 5. Update the help text in main.go

Add "generics" to the available topics list.
