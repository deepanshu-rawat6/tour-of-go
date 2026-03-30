package tui

import (
	"fmt"
	"sort"
	"strings"
)

// View renders the entire UI as a string.
// This is the "V" in Elm architecture — pure function from Model to string.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("⚙  Scheduler TUI") + "\n")

	// Tab bar
	tabs := make([]string, tabCount)
	for i, name := range tabNames {
		if Tab(i) == m.activeTab {
			tabs[i] = activeTab.Render(name)
		} else {
			tabs[i] = inactiveTab.Render(name)
		}
	}
	b.WriteString(strings.Join(tabs, " ") + "\n\n")

	// Content
	switch m.activeTab {
	case TabConcurrency:
		b.WriteString(m.viewConcurrency())
	case TabSubmit:
		b.WriteString(m.viewSubmit())
	}

	// Help bar
	b.WriteString(helpStyle.Render("\n  tab: switch view  •  r: refresh  •  q: quit"))

	return b.String()
}

func (m Model) viewConcurrency() string {
	var b strings.Builder

	if m.loading && len(m.concurrencyKeys) == 0 {
		b.WriteString("  " + m.spinner.View() + " Connecting to scheduler...\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  ✗ %v\n", m.err)))
		b.WriteString("  Make sure distributed-scheduler is running on :8080\n")
		return b.String()
	}

	if len(m.concurrencyKeys) == 0 {
		b.WriteString("  No active concurrency keys.\n")
		b.WriteString("  Submit jobs to the scheduler to see data here.\n")
		return b.String()
	}

	// Sort keys for stable display
	keys := make([]string, 0, len(m.concurrencyKeys))
	for k := range m.concurrencyKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxBarWidth := 40
	b.WriteString(fmt.Sprintf("  %-30s  %s\n", "Key", "Usage"))
	b.WriteString("  " + strings.Repeat("─", 60) + "\n")

	for _, key := range keys {
		val := m.concurrencyKeys[key]
		barLen := int(val)
		if barLen > maxBarWidth {
			barLen = maxBarWidth
		}
		bar := barFill.Render(strings.Repeat("█", barLen)) +
			barEmpty.Render(strings.Repeat("░", maxBarWidth-barLen))

		label := key
		if len(label) > 28 {
			label = label[:25] + "..."
		}
		b.WriteString(fmt.Sprintf("  %-30s  %s %d\n", label, bar, val))
	}

	if !m.lastRefresh.IsZero() {
		b.WriteString(helpStyle.Render(fmt.Sprintf("\n  Last refresh: %s  (auto-refreshes every 2s)",
			m.lastRefresh.Format("15:04:05"))))
	}

	return b.String()
}

func (m Model) viewSubmit() string {
	var b strings.Builder

	b.WriteString("  Submit a new job to the distributed-scheduler\n\n")

	fields := []struct {
		label string
		input interface{ View() string }
	}{
		{"Job Name", m.jobNameInput},
		{"Tenant ID", m.tenantInput},
		{"Priority (1=high, 10=low)", m.priorityInput},
	}

	for i, f := range fields {
		prefix := "  "
		if m.focusedField == i {
			prefix = "> "
		}
		b.WriteString(fmt.Sprintf("%s%s\n", prefix, f.label))
		b.WriteString(fmt.Sprintf("  %s\n\n", f.input.View()))
	}

	b.WriteString("  Press Enter to submit\n")

	if m.submitResult != "" {
		if strings.HasPrefix(m.submitResult, "✓") {
			b.WriteString("\n  " + lipglossGreen(m.submitResult) + "\n")
		} else {
			b.WriteString("\n  " + errorStyle.Render(m.submitResult) + "\n")
		}
	}

	return b.String()
}

func lipglossGreen(s string) string {
	return "\033[32m" + s + "\033[0m"
}
