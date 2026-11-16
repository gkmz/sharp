package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	sharptheme "github.com/gkmz/sharp/internal/theme"
	"github.com/gkmz/sharp/pkg/tool"
)

// Options supplies resolved theme state to the interactive application.
type Options struct {
	Themes        *sharptheme.Registry
	ActiveTheme   sharptheme.Theme
	SaveTheme     func(string) error
	StartupAlerts []string
}

type themePicker struct {
	open       bool
	originalID string
	selected   int
	entries    []sharptheme.Theme
}

func newModelWithOptions(registry *tool.Registry, options Options) model {
	if options.Themes == nil {
		options.Themes, _ = sharptheme.NewRegistry(nil)
	}
	if options.ActiveTheme.ID == "" {
		options.ActiveTheme, _ = options.Themes.Lookup("sharp-color")
	}
	m := newModelWithStyles(registry, NewStyleSet(options.ActiveTheme))
	m.themes = options.Themes
	m.activeThemeID = options.ActiveTheme.ID
	m.saveTheme = options.SaveTheme
	if len(options.StartupAlerts) > 0 {
		m.setStatus(warningStatus(strings.Join(options.StartupAlerts, "; ")))
	}
	return m
}

func (m *model) openThemePicker() {
	if m.themes == nil {
		return
	}
	entries := m.themes.All()
	if len(entries) == 0 {
		return
	}
	m.themePicker = themePicker{
		open:       true,
		originalID: m.activeThemeID,
		entries:    entries,
	}
	for i, current := range entries {
		if current.ID == m.activeThemeID {
			m.themePicker.selected = i
			break
		}
	}
}

func (m *model) updateThemePicker(key string) {
	if !m.themePicker.open || len(m.themePicker.entries) == 0 {
		return
	}
	switch key {
	case "esc":
		m.applyTheme(m.themePicker.originalID)
		m.themePicker.open = false
	case "j", "down":
		m.themePicker.selected = (m.themePicker.selected + 1) % len(m.themePicker.entries)
		m.applyTheme(m.themePicker.entries[m.themePicker.selected].ID)
	case "k", "up":
		m.themePicker.selected = (m.themePicker.selected - 1 + len(m.themePicker.entries)) % len(m.themePicker.entries)
		m.applyTheme(m.themePicker.entries[m.themePicker.selected].ID)
	case "enter":
		id := m.themePicker.entries[m.themePicker.selected].ID
		m.applyTheme(id)
		m.themePicker.open = false
		if m.saveTheme == nil {
			m.setStatus(errorStatus("theme changed for this session; configuration persistence is unavailable"))
			return
		}
		if err := m.saveTheme(id); err != nil {
			m.setStatus(errorStatus("theme changed for this session; save failed: " + err.Error()))
			return
		}
		m.setStatus(successStatus("Theme saved: " + id))
	}
}

func (m *model) applyTheme(id string) bool {
	if m.themes == nil {
		return false
	}
	current, ok := m.themes.Lookup(id)
	if !ok {
		return false
	}
	m.activeThemeID = id
	m.applyStyles(NewStyleSet(current))
	return true
}

func (m model) themePickerView() string {
	if !m.themePicker.open {
		return ""
	}
	width := min(58, max(28, m.width-4))
	innerWidth := max(1, width-2)
	rows := []string{m.styles.Dim.Render(" Preview with j/k or arrows · Enter save · Esc cancel"), ""}
	for i, current := range m.themePicker.entries {
		prefix := "  "
		style := m.styles.Text
		if i == m.themePicker.selected {
			prefix = "› "
			style = m.styles.SelectedRow
		}
		label := current.Name
		if current.Name != current.ID {
			label += " · " + current.ID
		}
		rows = append(rows, style.Render(fitWidth(prefix+truncateDisplay(label, max(1, innerWidth-2)), innerWidth)))
	}
	content := strings.Join(rows, "\n")
	height := min(lipgloss.Height(content)+2, max(6, m.height-2))
	return panelBoxWithTitleStyles(m.styles, "Themes", true, content, width, height)
}

func unavailableThemeSaver(path string) func(string) error {
	return func(string) error {
		if path == "" {
			return errors.New("configuration path is unavailable")
		}
		return fmt.Errorf("configuration %s is unavailable", path)
	}
}
