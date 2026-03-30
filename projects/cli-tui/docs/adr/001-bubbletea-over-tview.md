# ADR-001: Bubble Tea over tview for the TUI

**Status:** Accepted

## Decision

Use `charmbracelet/bubbletea` instead of `rivo/tview`.

## Rationale

| Concern | tview | Bubble Tea |
|---|---|---|
| Architecture | Imperative (widget tree, callbacks) | Elm (Model-Update-View, pure functions) |
| Testability | Hard (UI state in widgets) | Easy (Model is a plain struct, Update is pure) |
| Composability | Widgets are tightly coupled | Components are just functions |
| Learning value | Widget API | Elm architecture (transferable to Elm, Redux, SwiftUI) |
| Ecosystem | Standalone | charmbracelet suite (lipgloss, bubbles, glamour) |

## Elm Architecture in Go

```
Model (state struct)
  ↓ Init() → initial Cmd
  ↓ Update(Msg) → (Model, Cmd)   ← pure function
  ↓ View(Model) → string          ← pure function
```

Commands (`tea.Cmd`) are functions that run in goroutines and return messages. This is how async operations (HTTP calls, timers) integrate without callbacks or shared mutable state.
