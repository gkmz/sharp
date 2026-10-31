package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/pkg/tool"
)

func TestInteractionModeFollowsContext(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	for _, tc := range []struct {
		region focus
		want   string
	}{
		{focusCategories, "NORMAL Categories"},
		{focusTabs, "NORMAL Actions"},
		{focusWorkspace, "NORMAL"},
	} {
		m.focus = tc.region
		if footer := stripANSI(m.footerLine()); !strings.HasPrefix(strings.TrimSpace(footer), tc.want) {
			t.Errorf("region %v footer = %q, want %q", tc.region, footer, tc.want)
		}
	}
	if !strings.Contains(stripANSI(m.View()), "[3]-Workspace · NORMAL") {
		t.Error("active Workspace border does not identify Normal mode")
	}
	m = workflowKey(m, "/")
	if !strings.Contains(stripANSI(m.footerLine()), "SEARCH Categories") {
		t.Error("search mode is not identified")
	}
	m = workflowKey(m, "esc")
	m = workflowKey(m, "?")
	if !strings.Contains(stripANSI(m.footerLine()), "HELP Commands") {
		t.Error("help mode is not identified")
	}
	m = workflowKey(m, "esc")
	if !strings.Contains(stripANSI(m.footerLine()), "NORMAL Categories") {
		t.Error("closing Help did not restore the underlying mode")
	}
}

func TestInteractionInsertIdentifiesFieldAndKeepsCommandsAsText(t *testing.T) {
	for _, tc := range []struct {
		id, key, target string
	}{
		{"json.pretty", "i", "Input"},
		{"text.upper", "i", "Input"},
		{"json.get", "o", "Path"},
		{"regex.replace", "o", "Options"},
	} {
		t.Run(tc.id, func(t *testing.T) {
			m := workflowModel(tool.CategoryJSON)
			selectWorkspaceTool(t, &m, tc.id)
			m.focus = focusTabs
			m = workflowKey(m, tc.key)
			footer := stripANSI(m.footerLine())
			for _, want := range []string{"INSERT " + tc.target, "[Esc] normal"} {
				if !strings.Contains(footer, want) {
					t.Errorf("footer = %q, want %q", footer, want)
				}
			}
			enterHint := "[Enter] run"
			if tc.target == "Input" {
				enterHint = "[Enter] newline"
			}
			if !strings.Contains(footer, enterHint) {
				t.Errorf("footer = %q, want %q", footer, enterHint)
			}
			if !strings.Contains(stripANSI(m.View()), "[3]-Workspace · INSERT "+tc.target) {
				t.Error("workspace border does not identify the edited field")
			}

			p := m.currentPage()
			var value func() string
			switch page := p.(type) {
			case *jsonToolPage:
				value = func() string { return page.input.Value() }
				if tc.target == "Path" {
					value = func() string { return page.pathBox.Value() }
				}
			case *genericToolPage:
				value = func() string { return page.input.Value() }
				if tc.target == "Options" {
					setOptionValues(page, tool.Options{"pattern": ""})
					value = func() string { return page.optionForm.Values()["pattern"] }
				}
			}
			for _, key := range []string{"x", "y", "p", "r", "[", "]", "1", "2", "3", "?"} {
				m = workflowKey(m, key)
			}
			if got := value(); got != "xypr[]123?" || m.currentTab().Tool.ID() != tc.id || m.search.Value() != "" || m.showHelp {
				t.Fatalf("Insert routed text as commands: value=%q action=%s", got, m.currentTab().Tool.ID())
			}
			m = workflowKey(m, "esc")
			if p.IsEditing() || m.focus != focusWorkspace || value() != "xypr[]123?" {
				t.Fatal("Esc did not leave editing in place and preserve the draft")
			}
			if strings.Contains(stripANSI(m.View()), "Workspace · INSERT") || !strings.HasPrefix(strings.TrimSpace(stripANSI(m.footerLine())), "NORMAL") {
				t.Fatal("Esc left an Insert mode indicator behind")
			}
		})
	}
}

func TestInteractionParameterSubmitReturnsToNormal(t *testing.T) {
	for _, id := range []string{"json.get", "regex.replace"} {
		t.Run(id, func(t *testing.T) {
			m := workflowModel(tool.CategoryJSON)
			selectWorkspaceTool(t, &m, id)
			p := m.currentPage()
			switch page := p.(type) {
			case *jsonToolPage:
				page.SetInput(`{"name":"sharp"}`)
				page.pathBox.SetValue("name")
			case *genericToolPage:
				page.SetInput("hello")
				setOptionValues(page, tool.Options{"pattern": "hello", "replacement": "sharp"})
			}
			m.focus = focusWorkspace
			m = workflowKey(m, "o")
			m = workflowKey(m, "enter")
			if p.IsEditing() || m.focus != focusWorkspace || p.OutputText() != "sharp" {
				t.Fatalf("submit did not execute and exit editor: output=%q", p.OutputText())
			}
			if !strings.Contains(stripANSI(m.View()), "Workspace · NORMAL") || strings.Contains(stripANSI(m.footerLine()), "INSERT") {
				t.Error("submit left stale mode indicators")
			}
		})
	}
}

func TestInteractionEscClosesOnlyCurrentLayer(t *testing.T) {
	for _, region := range []focus{focusCategories, focusTabs, focusWorkspace, focusSearch} {
		t.Run(fmt.Sprint(region), func(t *testing.T) {
			m := workflowModel(tool.CategoryJSON)
			p := m.currentPage()
			p.SetInput(`{"name":"sharp"}`)
			p.Run()
			m.focus = region
			if region == focusSearch {
				m.search.SetValue("json")
				m.search.Focus()
			}
			m.openHelp()
			before, revision := p.OutputText(), p.previewVersion()
			m = workflowKey(m, "esc")
			if m.showHelp || m.focus != region || p.OutputText() != before || p.previewVersion() != revision {
				t.Fatal("Esc closed more than Help or recomputed the document")
			}
			if region == focusSearch && (m.search.Value() != "json" || !m.search.Focused()) {
				t.Fatal("closing Help lost the underlying search")
			}
		})
	}

	m := workflowModel(tool.CategoryJSON)
	m.focus = focusWorkspace
	m = workflowKey(m, "i")
	m, _ = normalText(m, `{"name":"sharp"}`)
	for _, want := range []focus{focusWorkspace, focusTabs, focusCategories} {
		m = workflowKey(m, "esc")
		if m.focus != want || m.currentPage().IsEditing() {
			t.Fatalf("Esc reached %v instead of %v", m.focus, want)
		}
	}
	if p := m.currentPage().(*jsonToolPage); p.input.Value() != `{"name":"sharp"}` || !strings.Contains(p.OutputText(), `"name": "sharp"`) {
		t.Fatal("back navigation lost the document or its preview")
	}
	beforeStatus, beforeRevision := m.status, m.currentPage().previewVersion()
	m = workflowKey(m, "esc")
	if m.focus != focusCategories || m.status != beforeStatus || m.currentPage().previewVersion() != beforeRevision {
		t.Error("Esc at idle Categories must do nothing")
	}
}

func TestInteractionModeViewsFitTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		for _, id := range []string{"json.get", "regex.replace", "password"} {
			m := workflowModel(tool.CategoryJSON)
			selectWorkspaceTool(t, &m, id)
			m.width, m.height = size[0], size[1]
			m.resize()
			check := func() {
				t.Helper()
				view := m.View()
				if lipgloss.Width(m.footerLine()) > m.width || lipgloss.Width(view) > m.width || lipgloss.Height(view) > m.height {
					t.Fatalf("%s mode view exceeds %dx%d: %q", id, m.width, m.height, stripANSI(m.footerLine()))
				}
				lines := strings.Split(stripANSI(view), "\n")
				if !strings.Contains(lines[len(lines)-1], strings.TrimSpace(stripANSI(m.footerLine()))) {
					t.Fatal("mode footer was truncated")
				}
			}
			for _, region := range []focus{focusCategories, focusTabs, focusWorkspace} {
				m.focus = region
				check()
			}
			for _, key := range []string{"i", "o"} {
				m = workflowKey(m, key)
				check()
				m = workflowKey(m, "esc")
			}
			m = workflowKey(m, "/")
			check()
			m = workflowKey(m, "esc")
			m = workflowKey(m, "?")
			check()
		}
	}
}
