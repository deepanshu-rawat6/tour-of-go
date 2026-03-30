# ADR-001: html/template with Tailwind over Templ

**Status:** Accepted (pragmatic choice)

## Context

The plan called for `a-h/templ` (type-safe Go templates). However, templ requires a code generation step (`templ generate`) and the `templ` CLI tool to be installed. This adds friction for a learning project.

## Decision

Use `html/template` (stdlib) with Tailwind CSS via CDN.

## Rationale

- Zero tooling: `go run ./cmd/console/` works immediately
- `html/template` is the foundation — understanding it is prerequisite to templ
- Tailwind CDN provides the same visual result without npm

## When to Use Templ Instead

Use `a-h/templ` when:
- You want compile-time type checking on template data (catch missing fields at build time)
- You're building reusable component libraries
- Your team is comfortable with a code generation step

The templ approach would look like:
```go
// greeting_list.templ
templ GreetingList(greetings []k8s.Greeting) {
    for _, g := range greetings {
        <tr>
            <td>{ g.Name }</td>  // compile error if Name doesn't exist
        </tr>
    }
}
```
