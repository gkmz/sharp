package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/gkmz/sharp/pkg/tool"
	"github.com/muesli/termenv"
)

func TestVisualCategorySelectionSpansWidth(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	const width = 27
	lines := strings.Split(stripANSI(m.categoryView(width, 50)), "\n")
	if selected := lines[m.selectedCat]; lipgloss.Width(selected) != width {
		t.Fatalf("selected row width = %d, want %d: %q", lipgloss.Width(selected), width, selected)
	}
}

func TestVisualCategoryNamesAndCountsFitAvailableWidth(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	for _, width := range []int{3, 20, 30, 48} {
		for _, c := range m.categories {
			line := m.categoryLine(c, width)
			if c.Header {
				continue
			}
			if lipgloss.Width(line) != width || !strings.HasSuffix(line, fmt.Sprint(len(c.Tabs))) {
				t.Errorf("%s/%s at width %d: count missing or misaligned: %q", c.Category, c.Subcategory, width, line)
			}
			if width >= 20 && c.Subcategory == "b64" && !strings.HasPrefix(line, "  Base64") {
				t.Errorf("category still displays the internal ID: %q", line)
			}
		}
	}
}

func TestVisualCategoryScrollIndicatorsPreserveRows(t *testing.T) {
	for _, searching := range []bool{false, true} {
		m := workflowModel(tool.CategoryJSON)
		if searching {
			m.focus = focusSearch
			m.search.SetValue("json")
		}
		const width, height = 30, 10
		visible := height - 2
		if searching {
			visible--
		}
		for _, selected := range []int{0, len(m.categories) / 2, len(m.categories) - 1} {
			m.selectedCat = selected
			// Inspect the complete frame: indicators must not push the selected
			// row or the last available list row outside the panel.
			m.width, m.height = 120, height+2
			m.resize()
			frame := stripANSI(m.View())
			lines := strings.Split(frame, "\n")
			leftWidth := 24
			top := truncateDisplay(lines[0], leftWidth)
			bottom := truncateDisplay(lines[height-1], leftWidth)
			wantUp := selected >= visible
			wantDown := selected < len(m.categories)-1
			if strings.Contains(top, "↑") != wantUp || strings.Contains(bottom, "↓") != wantDown {
				t.Errorf("search=%v selected=%d: wrong scroll markers: top=%q bottom=%q", searching, selected, top, bottom)
			}
			content := stripANSI(m.categoryView(width-2, height-2))
			rows := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
			if len(rows) != height-2 {
				t.Errorf("scroll hints consumed rows: got %d, want %d", len(rows), height-2)
			}
			selectedLine := m.categoryLine(m.categories[selected], width-2)
			if !strings.Contains(content, selectedLine) {
				t.Errorf("selected row disappeared: %q", selectedLine)
			}
		}
	}
}

func TestVisualCategoryScrollIndicatorsDisappearWhenListFits(t *testing.T) {
	for _, query := range []string{"", "uuid", "no-such-tool"} {
		m := workflowModel(tool.CategoryJSON)
		if query != "" {
			m = workflowKey(m, query)
		}
		m.width, m.height = 120, 60
		m.resize()
		view := stripANSI(m.View())
		lines := strings.Split(view, "\n")
		if strings.ContainsAny(truncateDisplay(lines[0], 24)+truncateDisplay(lines[57], 24), "↑↓") {
			t.Fatalf("scroll indicators shown when list fits, query=%q", query)
		}
	}
}

func TestVisualOutputNamesOnlyRetainedProducingAction(t *testing.T) {
	for _, tc := range []struct{ first, next, input string }{
		{"text.upper", "text.lower", "hello"},
		{"json.pretty", "json.get", `{"name":"sharp"}`},
		{"uuid.v4", "", ""},
	} {
		t.Run(tc.first, func(t *testing.T) {
			m := workflowModel(tool.CategoryJSON)
			selectWorkspaceTool(t, &m, tc.first)
			p := m.currentPage()
			if tc.input != "" {
				p.SetInput(tc.input)
			}
			p.Run()
			result := p.OutputText()
			if result == "" {
				t.Fatal("expected a result")
			}
			name := m.currentTab().Tool.Name()
			if strings.Contains(stripANSI(p.View()), "Output · "+name) {
				t.Fatal("current action repeated in Output heading")
			}
			if tc.next == "" {
				return
			}
			selectWorkspaceTool(t, &m, tc.next)
			p = m.currentPage()
			if p.OutputText() != result || !strings.Contains(stripANSI(p.View()), "Output · "+name) {
				t.Fatal("retained result lost its source action")
			}
		})
	}
}

func TestVisualSelectionFollowsFocusAndShortcutsStayUnfilled(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	m := workflowModel(tool.CategoryJSON)
	activeCategory := m.categoryView(24, 50)
	inactiveActions := m.tabsView(94, 2)
	m.focus = focusTabs
	if activeCategory == m.categoryView(24, 50) {
		t.Error("category selection did not become subdued after losing focus")
	}
	if inactiveActions == m.tabsView(94, 2) {
		t.Error("action selection did not become active after gaining focus")
	}
	activeActions := m.tabsView(94, 2)
	m.openHelp()
	if activeActions == m.tabsView(94, 2) {
		t.Error("action selection remains active under Help")
	}
	if strings.Contains(statusHint("x", "clear"), "48;") {
		t.Error("shortcut keys still have a solid background")
	}
}

func TestVisualWorkspaceHasOneDescriptionAndOpenSections(t *testing.T) {
	for _, id := range []string{"json.pretty", "json.get", "regex.replace", "password"} {
		t.Run(id, func(t *testing.T) {
			m := workflowModel(tool.CategoryJSON)
			selectWorkspaceTool(t, &m, id)
			m.focus = focusWorkspace
			m.resize()
			view := stripANSI(m.pageView(94, 32))
			if strings.ContainsAny(view, "┌└│") {
				t.Error("Workspace still nests framed sections")
			}
			if strings.Contains(view, workspaceGroupTitle(m.currentTab().Tool)+" · "+actionLabel(m.currentTab().Tool)) {
				t.Error("Workspace repeats the Actions tool header")
			}
			if first := strings.Split(view, "\n")[0]; !strings.Contains(first, m.currentTab().Tool.Description()) {
				t.Errorf("first line should be the tool description, got %q", first)
			}
		})
	}
}

func TestVisualEmptyStatesNeverBecomeDocumentData(t *testing.T) {
	for _, tc := range []struct {
		id, want string
	}{
		{"json.pretty", "Paste JSON with v"},
		{"text.upper", "Paste text with v"},
		{"csv.to_json", "Paste CSV with v"},
		{"dns.lookup", "r / Enter to run"},
		{"password", "r / Enter to generate"},
	} {
		t.Run(tc.id, func(t *testing.T) {
			m := workflowModel(tool.CategoryJSON)
			selectWorkspaceTool(t, &m, tc.id)
			p := m.currentPage()
			view := stripANSI(p.View())
			if !strings.Contains(view, tc.want) {
				t.Errorf("empty state is missing %q", tc.want)
			}
			if p.OutputText() != "" {
				t.Fatal("view-only hint became output data")
			}
			m.focus = focusWorkspace
			m = workflowKey(m, "i")
			if p.IsEditing() && strings.Contains(stripANSI(p.View()), "Paste ") {
				t.Error("Input empty-state hint covers the active editor")
			}
		})
	}
}

func TestVisualNormalInputDoesNotShowEditorGutter(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	p := m.currentPage().(*jsonToolPage)
	p.SetInput(`{"name":"sharp"}`)
	view := stripANSI(p.View())
	if strings.ContainsAny(view, "┃│") {
		t.Error("Normal document still displays editor gutter or nested borders")
	}
	if p.input.Value() != `{"name":"sharp"}` {
		t.Fatal("rendering changed the source document")
	}
}

func TestVisualFullFrameKeepsLastOutputLine(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {160, 50}} {
		for _, id := range []string{"json.get", "text.upper", "regex.replace", "password"} {
			m := workflowModel(tool.CategoryJSON)
			m.width, m.height = size[0], size[1]
			selectWorkspaceTool(t, &m, id)
			m.focus = focusWorkspace
			p := m.currentPage()
			height := 0
			switch page := p.(type) {
			case *jsonToolPage:
				height = page.output.Height
			case *genericToolPage:
				height = page.output.Height
			}
			rows := make([]string, height)
			for i := range rows {
				rows[i] = fmt.Sprintf("OUTPUT-%02d", i)
			}
			switch page := p.(type) {
			case *jsonToolPage:
				page.outputText = strings.Join(rows, "\n")
				page.renderOutput()
			case *genericToolPage:
				page.outputText = strings.Join(rows, "\n")
				page.renderOutput()
			}
			if view := stripANSI(m.View()); !strings.Contains(view, rows[len(rows)-1]) {
				t.Errorf("%s at %dx%d hides last output row", id, m.width, m.height)
			}
		}
	}
}

func TestVisualHelpKeepsCommandsAndBackgroundIntact(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	base := lipgloss.NewStyle().Foreground(lipgloss.Color("#a0b8e6")).Render("abcdefghij")
	if got := ansi.Strip(cutDisplayLeft(base, 4)); got != "efghij" {
		t.Errorf("cutting a styled background left ANSI fragments: %q", got)
	}
	for _, width := range []int{52, 80} {
		command := helpCommand{Key: "Tab/Shift+Tab", Command: "Change Area", Description: "Cycle Categories, Actions and Workspace in Normal mode."}
		view := stripANSI(renderHelpCommand(command, width))
		if !strings.Contains(view, command.Key) || !strings.Contains(view, command.Command) || lipgloss.Width(view) > width {
			t.Errorf("help clips keys or labels at width %d: %q", width, view)
		}
	}
}
