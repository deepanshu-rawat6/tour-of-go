// Package tui implements the Bubble Tea terminal UI.
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Tab bar styles
	activeTab   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Underline(true).Padding(0, 2)
	inactiveTab = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 2)

	// Status badge colors
	statusColors = map[string]lipgloss.Color{
		"WAITING":      "220", // yellow
		"RETRY":        "214", // orange
		"PUBLISHED":    "39",  // blue
		"PROCESSING":   "51",  // cyan
		"SUCCESSFUL":   "82",  // green
		"FAILED":       "196", // red
		"FAILED_BY_JFC": "197", // bright red
		"CANCELLED":    "240", // gray
	}

	// Layout
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).MarginBottom(1)
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginTop(1)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	barFill     = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	barEmpty    = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

// StatusBadge returns a colored status string.
func StatusBadge(status string) string {
	color, ok := statusColors[status]
	if !ok {
		color = "240"
	}
	return lipgloss.NewStyle().Foreground(color).Render("● " + status)
}
