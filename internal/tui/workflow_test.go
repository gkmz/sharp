package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/pkg/tool"
)

func workflowModel(category tool.Category) model {
	m := newModel(tools.NewRegistry())
	m.width, m.height = 120, 40
	for i, c := range m.categories {
		if c.Category == category && !c.Header {
			m.selectedCat = i
			break
		}
	}
	m.resize()
	return m
}

func workflowKey(m model, key string) model {
	var msg tea.KeyMsg
	switch key {
	case "enter":
		msg.Type = tea.KeyEnter
	case "esc":
		msg.Type = tea.KeyEsc
	case "down":
		msg.Type = tea.KeyDown
	case "up":
		msg.Type = tea.KeyUp
	case "tab":
		msg.Type = tea.KeyTab
	case "right":
		msg.Type = tea.KeyRight
	case "backspace":
		msg.Type = tea.KeyBackspace
	default:
		msg.Type = tea.KeyRunes
		msg.Runes = []rune(key)
	}
	updated, _ := m.Update(msg)
	return updated.(model)
}

func selectWorkflowAction(t *testing.T, m *model, id string) {
	t.Helper()
	for i, tab := range m.currentCategory().Tabs {
		if tab.Tool.ID() == id {
			m.setSelectedTab(i)
			m.rebuildPage()
			return
		}
	}
	t.Fatalf("missing action %s", id)
}

func TestWorkflowCategoryTypeToSearch(t *testing.T) {
	m := workflowModel(tool.CategoryConvert)
	for _, key := range []string{"j", "s", "o", "n"} {
		m = workflowKey(m, key)
	}
	if m.focus != focusSearch || m.search.Value() != "json" {
		t.Fatalf("typing json: focus=%v query=%q", m.focus, m.search.Value())
	}
	if strings.Contains(stripANSI(m.View()), "[1]-Search") {
		t.Fatal("standalone search panel remains")
	}
	if !strings.Contains(stripANSI(m.View()), "/ json") {
		t.Fatal("inline query is not visible")
	}
	before := m.selectedCat
	m = workflowKey(m, "down")
	if m.selectedCat == before || m.search.Value() != "json" {
		t.Fatal("arrow should navigate results without editing the query")
	}
	m = workflowKey(m, "esc")
	if m.search.Value() != "" || m.focus != focusCategories || len(m.categories) != len(m.allCategories) {
		t.Fatal("Esc should clear query and restore categories")
	}
}

func TestWorkflowSearchFirstLetterDoesNotNavigateOrQuit(t *testing.T) {
	for _, letter := range []string{"j", "k", "q"} {
		t.Run(letter, func(t *testing.T) {
			m := workflowModel(tool.CategoryText)
			m = workflowKey(m, letter)
			if m.search.Value() != letter || m.focus != focusSearch {
				t.Fatalf("%s did not start search", letter)
			}
		})
	}
}

func TestWorkflowJSONActionSelectionPreservesInput(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	p := m.currentPage().(*jsonToolPage)
	p.SetInput(`{"b":2,"a":1}`)
	selectWorkflowAction(t, &m, "json.minify")
	m.focus = focusTabs
	m = workflowKey(m, "r")
	if got := p.OutputText(); got != `{"b":2,"a":1}` {
		t.Fatalf("minify ran wrong action: %q", got)
	}
	selectWorkflowAction(t, &m, "json.sort")
	if m.currentPage() != p || p.input.Value() != `{"b":2,"a":1}` {
		t.Fatal("action switch lost the document")
	}
	m = workflowKey(m, "r")
	if !strings.Contains(p.OutputText(), `"a": 1`) {
		t.Fatalf("sort result=%q", p.OutputText())
	}
	if strings.Contains(stripANSI(m.tabsView(100, 5)), "Workspace") {
		t.Fatal("JSON actions still hidden behind Workspace")
	}
}

func TestWorkflowSearchRunsMatchingJSONAction(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	p := m.currentPage().(*jsonToolPage)
	p.SetInput("{\n \"x\": 1\n}")
	m.focus = focusCategories
	m = workflowKey(m, "minify")
	m = workflowKey(m, "enter")
	m = workflowKey(m, "r")
	if p.OutputText() != `{"x":1}` {
		t.Fatalf("search minify ran %q", p.OutputText())
	}
}

func TestWorkflowJSONPathOnlyShownForQuery(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	p := m.currentPage().(*jsonToolPage)
	p.SetInput(`{"data":{"name":"sharp"}}`)
	selectWorkflowAction(t, &m, "json.pretty")
	if strings.Contains(stripANSI(p.View()), "─Path") {
		t.Fatal("Path consumes space outside Query")
	}
	selectWorkflowAction(t, &m, "json.get")
	if !strings.Contains(stripANSI(p.View()), "─Path") {
		t.Fatal("Query needs Path field")
	}
	m.focus = focusWorkspace
	p.FocusOptions()
	p.pathBox.SetValue("data.name")
	m = workflowKey(m, "enter")
	if p.OutputText() != "sharp" {
		t.Fatalf("Path Enter result=%q", p.OutputText())
	}
}

func TestWorkflowEditorsDoNotStartSearch(t *testing.T) {
	for _, category := range []tool.Category{tool.CategoryJSON, tool.CategoryText} {
		m := workflowModel(category)
		m.focus = focusWorkspace
		m = workflowKey(m, "i")
		if !m.currentPage().IsEditing() {
			t.Fatal("i must edit")
		}
		m = workflowKey(m, "json")
		if m.search.Value() != "" || m.focus != focusWorkspace {
			t.Fatal("editor text leaked into navigation search")
		}
		m = workflowKey(m, "esc")
		if m.currentPage().IsEditing() {
			t.Fatal("Esc must leave editor")
		}
		m = workflowKey(m, "enter")
		if m.currentPage().IsEditing() {
			t.Fatal("Enter should execute the action without starting editing")
		}
	}
}

func TestWorkflowJSONApplyRejectsErrorsAndCheckIsStatus(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	p := m.currentPage().(*jsonToolPage)
	p.SetInput(`{"a":`)
	selectWorkflowAction(t, &m, "json.pretty")
	p.Run()
	p.HandleKey("a")
	if p.input.Value() != `{"a":` {
		t.Fatal("apply replaced input with an error")
	}
	p.SetInput(`{"ok":true}`)
	p.Run()
	result := p.OutputText()
	selectWorkflowAction(t, &m, "json.validate")
	p.Run()
	if p.OutputText() != result {
		t.Fatalf("Check replaced data result with %q", p.OutputText())
	}
	if !strings.Contains(stripANSI(p.Status()), "valid JSON") {
		t.Fatal("Check needs visible validation status")
	}
}

func TestWorkflowOutputReceivesArrowKeys(t *testing.T) {
	for _, category := range []tool.Category{tool.CategoryJSON, tool.CategoryText} {
		m := workflowModel(category)
		m.focus = focusWorkspace
		switch p := m.currentPage().(type) {
		case *jsonToolPage:
			p.outputText = strings.Repeat("line\n", 60)
			p.renderOutput()
			m = workflowKey(m, "down")
			if p.output.YOffset == 0 {
				t.Fatal("JSON output did not receive Down")
			}
		case *genericToolPage:
			p.outputText = strings.Repeat("line\n", 60)
			p.renderOutput()
			m = workflowKey(m, "down")
			if p.output.YOffset == 0 {
				t.Fatal("generic output did not receive Down")
			}
		}
	}
}

func TestWorkflowNotificationAndTerminalBounds(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {160, 50}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m := workflowModel(tool.CategoryJSON)
			m.width, m.height = size[0], size[1]
			m.focus = focusWorkspace
			m.resize()
			m.status = "copied output"
			view := m.View()
			if !strings.Contains(stripANSI(view), "copied output") {
				t.Fatal("notification hidden by shortcut hints")
			}
			if lipgloss.Width(view) > m.width || lipgloss.Height(view) > m.height {
				t.Fatalf("rendered %dx%d exceeds %dx%d", lipgloss.Width(view), lipgloss.Height(view), m.width, m.height)
			}
		})
	}
}

func TestWorkflowJSONOutputLastLineVisible(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	p := m.currentPage().(*jsonToolPage)
	p.SetInput(`{"ok":true}`)
	p.Run()
	rows := make([]string, p.output.Height)
	for i := range rows {
		rows[i] = fmt.Sprintf("ROW%02d", i)
	}
	p.outputText = strings.Join(rows, "\n")
	p.renderOutput()
	if !strings.Contains(p.View(), rows[len(rows)-1]) {
		t.Fatal("output metadata hides last viewport line")
	}
}

func TestWorkflowJSONActionPreviewAndHiddenPathFocus(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	p := m.currentPage().(*jsonToolPage)
	p.SetInput(`{"b":2,"a":1}`)
	m.focus = focusTabs
	m = workflowKey(m, "right")
	if p.OutputText() != `{"b":2,"a":1}` {
		t.Fatalf("selecting Minify did not preview: %q", p.OutputText())
	}
	selectWorkflowAction(t, &m, "json.get")
	p.FocusOptions()
	selectWorkflowAction(t, &m, "json.pretty")
	if p.field == editFieldOptions || p.editing {
		t.Fatal("action switch retained focus on hidden Path")
	}
}

func TestWorkflowSearchBackspaceAndNoResults(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m = workflowKey(m, "not-a-real-tool-987")
	if len(m.categories) != 0 {
		t.Fatal("expected empty results")
	}
	m = workflowKey(m, "down")
	m = workflowKey(m, "enter")
	if m.focus != focusCategories {
		t.Fatal("empty result Enter should return to Categories")
	}
	m = workflowKey(m, "esc")
	if len(m.categories) == 0 {
		t.Fatal("clearing an empty search did not restore tools")
	}
	m = workflowKey(m, "j")
	m = workflowKey(m, "backspace")
	if m.focus != focusCategories || m.search.Value() != "" {
		t.Fatal("deleting last search character should exit search")
	}
}

func TestWorkflowShortcutsFocusThreeVisibleRegions(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	for _, item := range []struct {
		key  string
		want focus
	}{{"2", focusTabs}, {"3", focusWorkspace}, {"1", focusCategories}} {
		m = workflowKey(m, item.key)
		if m.focus != item.want {
			t.Fatalf("%s focused %v, want %v", item.key, m.focus, item.want)
		}
	}
}

func TestWorkflowGenericOptionsFitWorkspace(t *testing.T) {
	for _, id := range []string{"regex.replace", "password", "csv.to_json"} {
		m := workflowModel(tool.CategoryText)
		m.width, m.height = 80, 24
		for ci, c := range m.categories {
			if c.Header {
				continue
			}
			for ti, tab := range c.Tabs {
				if tab.Tool.ID() == id {
					m.selectedCat = ci
					m.selectedTab = ti
				}
			}
		}
		m.resize()
		p := m.currentPage().(*genericToolPage)
		// 预算由外框、工具栏、页面标题与底部状态共同决定。
		bodyHeight := m.height - 2
		toolHeight := clamp(m.toolRows(56)+2, 3, max(3, bodyHeight-8))
		budget := panelContentHeight(bodyHeight-toolHeight) - 4
		if got := lipgloss.Height(p.View()); got > budget {
			t.Fatalf("%s uses %d rows, budget %d", id, got, budget)
		}
	}
}

func TestWorkflowSearchRetainsCategoryHierarchy(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m = workflowKey(m, "json")
	headers := 0
	for _, category := range m.categories {
		if category.Header && category.Category == tool.CategoryConvert {
			headers++
		}
	}
	if headers != 1 {
		t.Fatalf("filtered Convert has %d headers, want one grouped parent", headers)
	}
	for _, category := range m.categories {
		if category.Category == tool.CategoryJSON && !category.Header {
			line := m.categoryLine(category)
			if !strings.HasPrefix(line, "JSON") {
				t.Fatalf("standalone JSON appears nested under another category: %q", line)
			}
		}
	}
}

func TestWorkflowSearchAndHelpReceiveAsyncMessages(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m = workflowKey(m, "j")
	selected := m.selectedCat
	updated, cmd := m.Update(textinput.Blink())
	m = updated.(model)
	if cmd == nil {
		t.Fatal("inline search dropped cursor/async messages")
	}
	if m.search.Value() != "j" || m.selectedCat != selected {
		t.Fatal("cursor event changed search results")
	}
	m.openHelp()
	m.helpVP.YOffset = 1
	_, cmd = m.Update(textinput.Blink())
	if cmd == nil {
		t.Fatal("help dropped cursor/async messages")
	}
}

func TestWorkflowGeneratorRootHintsMatchAvailableFields(t *testing.T) {
	m := workflowModel(tool.CategoryGenerator)
	m.focus = focusWorkspace
	footer := stripANSI(m.footerLine())
	if strings.Contains(footer, "[i]") || strings.Contains(footer, "[v]") {
		t.Fatalf("no-input generator shows unavailable input actions: %q", footer)
	}
	if !strings.Contains(footer, "[o] options") {
		t.Fatal("configurable generator needs options shortcut")
	}
}

func TestWorkflowSearchSelectionSurvivesCategoryRoundTrip(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m = workflowKey(m, "minify")
	m = workflowKey(m, "enter")
	m.clearCategorySearch()
	if m.currentTab().Tool.ID() != "json.minify" {
		t.Fatal("clearing search lost selected Minify")
	}
	m.moveCategory(1)
	m.moveCategory(-1)
	if m.currentTab().Tool.ID() != "json.minify" {
		t.Fatalf("category round trip changed selected action to %s", m.currentTab().Tool.ID())
	}
}
