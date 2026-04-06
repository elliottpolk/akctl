package ui

import "github.com/charmbracelet/lipgloss"

var (
	HeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	SuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	WarnStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	ErrorStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	PathStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)
