package tui

import (
	"github.com/charmbracelet/lipgloss"
	sharptheme "github.com/gkmz/sharp/internal/theme"
)

// StyleSet is the complete model-local rendering surface for one theme.
type StyleSet struct {
	Text          lipgloss.Style
	Focus         lipgloss.Style
	AppTitle      lipgloss.Style
	PanelTitle    lipgloss.Style
	SectionTitle  lipgloss.Style
	Category      lipgloss.Style
	Group         lipgloss.Style
	Action        lipgloss.Style
	Shortcut      lipgloss.Style
	ActiveLabel   lipgloss.Style
	RetainedLabel lipgloss.Style
	Dim           lipgloss.Style
	Error         lipgloss.Style
	Warning       lipgloss.Style
	Success       lipgloss.Style
	SelectedRow   lipgloss.Style
	RetainedRow   lipgloss.Style
	SectionRule   lipgloss.Style
	Footer        lipgloss.Style
	StatusKey     lipgloss.Style
	StatusText    lipgloss.Style

	BorderColor   lipgloss.AdaptiveColor
	AccentColor   lipgloss.AdaptiveColor
	RetainedColor lipgloss.AdaptiveColor
	WarningColor  lipgloss.AdaptiveColor
}

// NewStyleSet converts a domain theme to adaptive Lip Gloss styles.
func NewStyleSet(current sharptheme.Theme) StyleSet {
	color := func(light, dark string) lipgloss.AdaptiveColor {
		return lipgloss.AdaptiveColor{Light: light, Dark: dark}
	}
	text := color(current.Light.Text, current.Dark.Text)
	muted := color(current.Light.Muted, current.Dark.Muted)
	border := color(current.Light.Border, current.Dark.Border)
	focus := color(current.Light.Focus, current.Dark.Focus)
	accent := color(current.Light.Accent, current.Dark.Accent)
	category := color(current.Light.Category, current.Dark.Category)
	group := color(current.Light.Group, current.Dark.Group)
	action := color(current.Light.Action, current.Dark.Action)
	shortcut := color(current.Light.Shortcut, current.Dark.Shortcut)
	selectionForeground := color(current.Light.SelectionForeground, current.Dark.SelectionForeground)
	selectionBackground := color(current.Light.SelectionBackground, current.Dark.SelectionBackground)
	retainedForeground := color(current.Light.RetainedForeground, current.Dark.RetainedForeground)
	retainedBackground := color(current.Light.RetainedBackground, current.Dark.RetainedBackground)
	success := color(current.Light.Success, current.Dark.Success)
	warning := color(current.Light.Warning, current.Dark.Warning)
	failure := color(current.Light.Error, current.Dark.Error)

	return StyleSet{
		Text:          lipgloss.NewStyle().Foreground(text),
		Focus:         lipgloss.NewStyle().Foreground(focus),
		AppTitle:      lipgloss.NewStyle().Bold(true).Foreground(text),
		PanelTitle:    lipgloss.NewStyle().Foreground(muted),
		SectionTitle:  lipgloss.NewStyle().Foreground(text),
		Category:      lipgloss.NewStyle().Bold(true).Foreground(category),
		Group:         lipgloss.NewStyle().Foreground(group),
		Action:        lipgloss.NewStyle().Foreground(action),
		Shortcut:      lipgloss.NewStyle().Foreground(shortcut),
		ActiveLabel:   lipgloss.NewStyle().Bold(true).Foreground(selectionForeground).Background(selectionBackground),
		RetainedLabel: lipgloss.NewStyle().Bold(true).Foreground(accent).Underline(true),
		Dim:           lipgloss.NewStyle().Foreground(muted),
		Error:         lipgloss.NewStyle().Foreground(failure),
		Warning:       lipgloss.NewStyle().Foreground(warning),
		Success:       lipgloss.NewStyle().Foreground(success),
		SelectedRow:   lipgloss.NewStyle().Bold(true).Foreground(selectionForeground).Background(selectionBackground),
		RetainedRow:   lipgloss.NewStyle().Foreground(retainedForeground).Background(retainedBackground),
		SectionRule:   lipgloss.NewStyle().Foreground(border),
		Footer:        lipgloss.NewStyle().Foreground(muted),
		StatusKey:     lipgloss.NewStyle().Foreground(shortcut),
		StatusText:    lipgloss.NewStyle().Foreground(muted),
		BorderColor:   border,
		AccentColor:   accent,
		RetainedColor: retainedBackground,
		WarningColor:  warning,
	}
}

func defaultStyleSet() StyleSet {
	registry, _ := sharptheme.NewRegistry(nil)
	current, _ := registry.Lookup("sharp-color")
	return NewStyleSet(current)
}

type statusKind uint8

const (
	statusMuted statusKind = iota
	statusSuccess
	statusWarning
	statusError
)

type statusMessage struct {
	text string
	kind statusKind
}

func mutedStatus(text string) statusMessage   { return statusMessage{text: text, kind: statusMuted} }
func successStatus(text string) statusMessage { return statusMessage{text: text, kind: statusSuccess} }
func warningStatus(text string) statusMessage { return statusMessage{text: text, kind: statusWarning} }
func errorStatus(text string) statusMessage   { return statusMessage{text: text, kind: statusError} }

func (s statusMessage) Render(styles StyleSet) string {
	switch s.kind {
	case statusSuccess:
		return styles.Success.Render(s.text)
	case statusWarning:
		return styles.Warning.Render(s.text)
	case statusError:
		return styles.Error.Render(s.text)
	default:
		return styles.Dim.Render(s.text)
	}
}
