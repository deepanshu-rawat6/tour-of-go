# cli-tui

A terminal dashboard for the distributed scheduler built with Bubble Tea — demonstrating the Elm architecture (Model/Update/View) in a Go TUI.

---

## Architecture

```mermaid
graph TD
    MAIN[main.go\nprogram.Start] --> BT[Bubble Tea Runtime]
    BT -->|Init| CMD[tea.Cmd\nfetchJobs HTTP]
    CMD -->|response| MSG[JobsLoadedMsg]
    MSG -->|Update| MODEL[Model\njobs list + selected + tab]
    MODEL -->|View| RENDER[lipgloss\nstyled terminal output]
    BT -->|keypress| UPDATE[Update\nj/k navigate\ntab switch\nr refresh]
    UPDATE --> MODEL
```

## Elm Architecture in Go

```mermaid
sequenceDiagram
    participant U as User Input
    participant BT as Bubble Tea
    participant M as Model
    participant V as View

    BT->>M: Init() → Cmd (fetch jobs)
    M-->>BT: tea.Cmd
    BT->>M: Update(JobsLoadedMsg)
    M-->>BT: updated Model, nil Cmd
    BT->>V: View(Model)
    V-->>BT: string (terminal frame)
    U->>BT: keypress 'j'
    BT->>M: Update(KeyMsg{j})
    M-->>BT: updated Model (cursor moved)
    BT->>V: View(Model)
```

## Key Concepts

- **Model** — immutable snapshot of all UI state: job list, selected index, active tab, loading flag.
- **Update** — pure function `(Model, Msg) → (Model, Cmd)`. No side effects — all I/O happens in `Cmd`.
- **View** — pure function `Model → string`. lipgloss applies colors, borders, and layout.
- **Cmd** — a function that runs asynchronously and returns a `Msg`. Used for HTTP calls, tickers, etc.

## Key Bindings

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `tab` | Switch tab |
| `r` | Refresh jobs |
| `q` / `ctrl+c` | Quit |

## Quick Start

```bash
# Requires the distributed-scheduler to be running on :8080
make run
```
