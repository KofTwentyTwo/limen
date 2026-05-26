// Package theme defines the fixed v1 terminal styling for limen.
package theme

import "github.com/charmbracelet/lipgloss"

var (
	Accent      = lipgloss.Color("#5fd7ff")
	Online      = lipgloss.Color("#5fdf87")
	Unreachable = lipgloss.Color("#6c6c6c")
	Probing     = lipgloss.Color("#87afdf")
	Dim         = lipgloss.Color("#8a8a8a")

	Title = lipgloss.NewStyle().
		Foreground(Accent).
		Bold(true)

	Pane = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Accent).
		Padding(0, 1)

	Selected = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0f1720")).
			Background(Accent)

	Dimmed = lipgloss.NewStyle().
		Foreground(Dim)

	OnlineStyle = lipgloss.NewStyle().
			Foreground(Online)

	UnreachableStyle = lipgloss.NewStyle().
				Foreground(Unreachable)

	ProbingStyle = lipgloss.NewStyle().
			Foreground(Probing)
)
