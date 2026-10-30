package tui

import "github.com/charmbracelet/lipgloss"

// Keep the terminal background; assign color by purpose rather than component.
var (
	textColor      = lipgloss.AdaptiveColor{Light: "#283445", Dark: "#d5dbe7"}
	mutedColor     = lipgloss.AdaptiveColor{Light: "#667386", Dark: "#8b96ad"}
	borderColor    = lipgloss.AdaptiveColor{Light: "#bac3d0", Dark: "#454b62"}
	focusColor     = lipgloss.AdaptiveColor{Light: "#416e28", Dark: "#a6ce72"}
	accentColor    = lipgloss.AdaptiveColor{Light: "#325d99", Dark: "#b4caff"}
	selectionColor = lipgloss.AdaptiveColor{Light: "#d4e1f5", Dark: "#3d5485"}
	retainedColor  = lipgloss.AdaptiveColor{Light: "#e8edf4", Dark: "#272e42"}
	warningColor   = lipgloss.AdaptiveColor{Light: "#916413", Dark: "#e5be78"}
	failureColor   = lipgloss.AdaptiveColor{Light: "#b43b46", Dark: "#e68b92"}

	textStyle          = lipgloss.NewStyle().Foreground(textColor)
	focusStyle         = lipgloss.NewStyle().Foreground(focusColor)
	appTitleStyle      = lipgloss.NewStyle().Bold(true).Foreground(textColor)
	panelTitleStyle    = lipgloss.NewStyle().Foreground(mutedColor)
	sectionTitleStyle  = lipgloss.NewStyle().Foreground(textColor)
	activeLabelStyle   = lipgloss.NewStyle().Bold(true).Foreground(textColor).Background(selectionColor)
	retainedLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(accentColor).Underline(true)
	dimStyle           = lipgloss.NewStyle().Foreground(mutedColor)
	errorStyle         = lipgloss.NewStyle().Foreground(failureColor)
	successStyle       = lipgloss.NewStyle().Foreground(focusColor)
	selectedRowStyle   = lipgloss.NewStyle().Bold(true).Foreground(textColor).Background(selectionColor)
	retainedRowStyle   = lipgloss.NewStyle().Foreground(textColor).Background(retainedColor)
	sectionRuleStyle   = lipgloss.NewStyle().Foreground(borderColor)
	footerStyle        = lipgloss.NewStyle().Foreground(mutedColor)
	statusKeyStyle     = lipgloss.NewStyle().Foreground(accentColor)
	statusTextStyle    = lipgloss.NewStyle().Foreground(mutedColor)
)
