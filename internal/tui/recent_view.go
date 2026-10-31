package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) openRecent() {
	m.showRecent = true
	m.recentSelected = 0
	m.recentItems = nil
	for _, id := range m.recent.ids {
		if _, ok := m.registry.Get(id); ok {
			m.recentItems = append(m.recentItems, id)
		}
	}
}

func (m *model) updateRecent(key string) {
	switch key {
	case "ctrl+r":
		m.showRecent = false
	case "up":
		m.recentSelected = max(0, m.recentSelected-1)
	case "down":
		m.recentSelected = min(max(0, len(m.recentItems)-1), m.recentSelected+1)
	case "enter":
		if len(m.recentItems) == 0 {
			return
		}
		id := m.recentItems[m.recentSelected]
		// Find the registered action in the complete hierarchy. Opening history
		// changes only navigation: no Run/Preview, including on manual tools.
		for ci, cat := range m.allCategories {
			if cat.Header {
				continue
			}
			for ti, tab := range cat.Tabs {
				if tab.Tool.ID() != id {
					continue
				}
				m.categories = m.allCategories
				m.search.SetValue("")
				m.search.Blur()
				m.selectedCat = ci
				m.setSelectedTab(ti)
				m.rebuildPage()
				m.focus = focusWorkspace
				m.showRecent = false
				m.status = dimStyle.Render("Selected " + tab.Tool.Name())
				return
			}
		}
	}
}

func (m model) recentView() string {
	width := min(64, max(10, m.width-4))
	innerWidth := max(1, width-2)
	rows := []string{dimStyle.Render(" Open a tool without running it"), ""}
	if len(m.recentItems) == 0 {
		rows = append(rows, textStyle.Render(" No recent tools yet"), dimStyle.Render(" Run a tool or preview valid input to add it"))
	} else {
		for i, id := range m.recentItems {
			t, ok := m.registry.Get(id)
			if !ok {
				continue
			}
			label := workspaceGroupTitle(t) + " · " + actionLabel(t)
			style, prefix := textStyle, "  "
			if i == m.recentSelected {
				style, prefix = selectedRowStyle, "› "
			}
			rows = append(rows, style.Render(fitWidth(prefix+truncateDisplay(label, max(1, innerWidth-2)), innerWidth)))
		}
	}
	if m.recent.warning != "" {
		rows = append(rows, "", errorStyle.Render(wrapDisplayText(" "+m.recent.warning, innerWidth)))
	}
	content := strings.Join(rows, "\n")
	height := min(lipgloss.Height(content)+2, max(4, m.height-2))
	return panelBoxWithTitle("Recent tools", true, content, width, height)
}
