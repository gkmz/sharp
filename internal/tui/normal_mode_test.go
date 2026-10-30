package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkmz/sharp/pkg/tool"
)

func normalKey(m model, kind tea.KeyType) (model, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: kind})
	return updated.(model), cmd
}

func normalText(m model, text string) (model, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
	return updated.(model), cmd
}

// The editor command can include a cursor blink alongside the preview timer.
func previewEvent(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("input change did not schedule a preview")
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		return previewEvent(t, batch[len(batch)-1])
	}
	return msg
}

func TestNormalRegionNavigation(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	for _, want := range []focus{focusTabs, focusWorkspace, focusCategories} {
		m, _ = normalKey(m, tea.KeyTab)
		if m.focus != want {
			t.Fatalf("Tab focus=%v want %v", m.focus, want)
		}
	}
	for _, want := range []focus{focusWorkspace, focusTabs, focusCategories} {
		m, _ = normalKey(m, tea.KeyShiftTab)
		if m.focus != want {
			t.Fatalf("Shift+Tab focus=%v want %v", m.focus, want)
		}
	}
	m.focus = focusWorkspace
	m = workflowKey(m, "i")
	for _, want := range []focus{focusWorkspace, focusTabs, focusCategories} {
		m = workflowKey(m, "esc")
		if m.focus != want || m.currentPage().IsEditing() {
			t.Fatalf("Esc focus=%v want %v", m.focus, want)
		}
	}
}

func TestNormalBracketActionsPreserveRegion(t *testing.T) {
	for _, region := range []focus{focusCategories, focusTabs, focusWorkspace} {
		m := workflowModel(tool.CategoryJSON)
		p := m.currentPage().(*jsonToolPage)
		p.SetInput(`{"x":1}`)
		m.focus = region
		m = workflowKey(m, "]")
		if m.currentTab() == nil || m.currentTab().Tool.ID() != "json.minify" || m.focus != region || m.search.Value() != "" || p.OutputText() != `{"x":1}` {
			t.Fatalf("bracket did not switch action in region %v", region)
		}
		m = workflowKey(m, "[")
		if m.currentTab().Tool.ID() != "json.pretty" || m.focus != region {
			t.Fatal("previous action moved focus")
		}
	}
}

func TestNormalDirectEditorsAndTabIsolation(t *testing.T) {
	for _, id := range []string{"json.get", "regex.replace"} {
		m := workflowModel(tool.CategoryJSON)
		selectWorkspaceTool(t, &m, id)
		m.focus = focusWorkspace
		m = workflowKey(m, "o")
		m = workflowKey(m, "esc")
		m = workflowKey(m, "i")
		p := m.currentPage()
		m, _ = normalText(m, "hello")
		switch page := p.(type) {
		case *jsonToolPage:
			if page.input.Value() != "hello" || page.pathBox.Value() != "" {
				t.Fatal("i edited Path instead of Input")
			}
		case *genericToolPage:
			if page.input.Value() != "hello" {
				t.Fatal("i edited Options instead of Input")
			}
		}
		m, _ = normalKey(m, tea.KeyTab)
		if !p.IsEditing() || m.focus != focusWorkspace {
			t.Fatal("Tab left editor or selected another panel")
		}
		m = workflowKey(m, "esc")
		m, _ = normalKey(m, tea.KeyTab)
		if m.focus != focusCategories {
			t.Fatal("normal Tab must switch regions")
		}
	}
}

func TestNormalTypingPreviewsWithoutLeavingEditor(t *testing.T) {
	for _, id := range []string{"json.pretty", "text.upper"} {
		m := workflowModel(tool.CategoryJSON)
		selectWorkspaceTool(t, &m, id)
		m.focus = focusWorkspace
		m = workflowKey(m, "i")
		input, want := `{"x":1}`, `"x": 1`
		if id == "text.upper" {
			input, want = "Hello Sharp", "HELLO SHARP"
		}
		var cmd tea.Cmd
		m, cmd = normalText(m, input)
		updated, _ := m.Update(previewEvent(t, cmd))
		m = updated.(model)
		if !m.currentPage().IsEditing() || !strings.Contains(m.currentPage().OutputText(), want) {
			t.Fatalf("preview missing while editing %s: %q", id, m.currentPage().OutputText())
		}
		m = workflowKey(m, "esc")
		if m.currentPage().IsEditing() || !strings.Contains(m.currentPage().OutputText(), want) {
			t.Fatal("Esc failed to settle preview in Normal")
		}
	}
}

func TestNormalInputEnterAndCommandLettersStayText(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m.focus = focusWorkspace
	m = workflowKey(m, "i")
	m, _ = normalText(m, "pyr[]123")
	m, _ = normalKey(m, tea.KeyEnter)
	p := m.currentPage().(*jsonToolPage)
	if p.input.Value() != "pyr[]123\n" || !p.IsEditing() || m.selectedTab != 0 {
		t.Fatalf("editor consumed commands: %q", p.input.Value())
	}
}

func TestNormalEscFlushesPreviewAndInvalidatesTimer(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m.focus = focusWorkspace
	m = workflowKey(m, "i")
	var cmd tea.Cmd
	m, cmd = normalText(m, `{"x":1}`)
	msg := previewEvent(t, cmd)
	m = workflowKey(m, "esc")
	p := m.currentPage().(*jsonToolPage)
	if !strings.Contains(p.OutputText(), `"x": 1`) {
		t.Fatal("Esc did not flush preview")
	}
	m = workflowKey(m, "X")
	updated, _ := m.Update(msg)
	m = updated.(model)
	if m.currentPage().OutputText() != "" {
		t.Fatal("obsolete timer restored cleared Output")
	}
}

func TestNormalInvalidInputMarksOldResultStale(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m.focus = focusWorkspace
	p := m.currentPage().(*jsonToolPage)
	p.SetInput(`{"ok":true}`)
	p.Run()
	old := p.OutputText()
	m = workflowKey(m, "i")
	m, _ = normalKey(m, tea.KeyBackspace)
	m = workflowKey(m, "esc")
	if p.OutputText() != old || !strings.Contains(stripANSI(p.View()), "stale") {
		t.Fatal("incomplete input did not retain and mark old preview stale")
	}
	before := p.input.Value()
	m = workflowKey(m, "p")
	if p.input.Value() != before {
		t.Fatal("Apply consumed stale preview")
	}
	for _, key := range []string{"y", "s"} {
		m = workflowKey(m, key)
		if !strings.Contains(stripANSI(m.status), "stale") {
			t.Fatalf("%s did not reject stale output", key)
		}
	}
}

func TestNormalOutputScrollNeedsNoPaneSelection(t *testing.T) {
	for _, id := range []string{"json.pretty", "text.upper"} {
		m := workflowModel(tool.CategoryJSON)
		selectWorkspaceTool(t, &m, id)
		m.focus = focusWorkspace
		switch p := m.currentPage().(type) {
		case *jsonToolPage:
			p.outputText = strings.Repeat("line\n", 80)
			p.renderOutput()
			m = workflowKey(m, "down")
			if p.output.YOffset == 0 {
				t.Fatal("JSON Down needs output selection")
			}
		case *genericToolPage:
			p.outputText = strings.Repeat("line\n", 80)
			p.renderOutput()
			m = workflowKey(m, "down")
			if p.output.YOffset == 0 {
				t.Fatal("generic Down needs output selection")
			}
		}
	}
}

func TestNormalPastePreviewsAndKeepsCommandsAvailable(t *testing.T) {
	for _, id := range []string{"json.pretty", "text.upper"} {
		m := workflowModel(tool.CategoryJSON)
		selectWorkspaceTool(t, &m, id)
		m.focus = focusTabs
		text, want := `{"x":1}`, `"x": 1`
		if id == "text.upper" {
			text, want = "Hello Sharp", "HELLO SHARP"
		}
		m.readClipboard = func() (string, error) { return text, nil }
		m = workflowKey(m, "v")
		p := m.currentPage()
		if m.focus != focusWorkspace || p.IsEditing() || !p.OutputReady() || !strings.Contains(p.OutputText(), want) {
			t.Fatal("v did not paste and preview in Normal mode")
		}
		m = workflowKey(m, "p")
		if p.IsEditing() || p.OutputText() != "" {
			t.Fatal("Apply not available immediately after paste")
		}
	}
}

func TestNormalClearInputShortcut(t *testing.T) {
	for _, id := range []string{"json.pretty", "text.upper"} {
		t.Run(id, func(t *testing.T) {
			m := workflowModel(tool.CategoryJSON)
			selectWorkspaceTool(t, &m, id)
			p := m.currentPage()
			p.SetInput("content")
			p.Run()
			m.focus = focusWorkspace
			m = workflowKey(m, "x")
			if got := p.OutputText(); got != "" {
				t.Fatalf("clear input should invalidate output, got %q", got)
			}
			switch page := p.(type) {
			case *jsonToolPage:
				if page.input.Value() != "" {
					t.Fatalf("JSON input = %q, want empty", page.input.Value())
				}
			case *genericToolPage:
				if page.input.Value() != "" {
					t.Fatalf("generic input = %q, want empty", page.input.Value())
				}
			}
		})
	}
}

func TestNormalLatestPreviewWinsAcrossOverlaysAndActions(t *testing.T) {
	m := workflowModel(tool.CategoryText)
	selectWorkspaceTool(t, &m, "text.upper")
	m.focus = focusWorkspace
	m = workflowKey(m, "i")
	m, first := normalText(m, "one")
	old := previewEvent(t, first)
	m, second := normalText(m, "two")
	latest := previewEvent(t, second)
	updated, _ := m.Update(old)
	m = updated.(model)
	if m.currentPage().OutputText() != "" {
		t.Fatal("outdated timer ran current input too early")
	}
	// Cursor events and help/search overlays must not swallow the preview timer.
	m.showHelp = true
	updated, _ = m.Update(latest)
	m = updated.(model)
	if m.currentPage().OutputText() != "ONETWO" || !m.showHelp {
		t.Fatal("help swallowed preview or changed mode")
	}
	m.showHelp = false
	m = workflowKey(m, "esc")
	p := m.currentPage().(*genericToolPage)
	msg := workspacePreviewMsg{page: p, revision: p.revision}
	m = workflowKey(m, "]")
	result := p.OutputText()
	updated, _ = m.Update(msg)
	m = updated.(model)
	if p.OutputText() != result {
		t.Fatal("old action timer replaced current action result")
	}
}

func TestNormalManualToolsNeverAutoRun(t *testing.T) {
	calls := 0
	registry := tool.NewRegistry()
	registry.MustRegister(tool.SimpleTool{IDValue: "dns.lookup", NameValue: "Lookup", CategoryValue: tool.CategoryNetwork, RunFunc: func(_ context.Context, in tool.Input, _ tool.Options) (tool.Output, error) {
		calls++
		return tool.Output{Text: in.Text}, nil
	}})
	m := newModel(registry)
	m.width, m.height = 100, 30
	m.resize()
	m.focus = focusWorkspace
	m.readClipboard = func() (string, error) { return "example.com", nil }
	m = workflowKey(m, "v")
	m = workflowKey(m, "i")
	m, _ = normalText(m, "x")
	m = workflowKey(m, "esc")
	m = workflowKey(m, "]")
	if calls != 0 {
		t.Fatal("manual tool auto-ran on paste, edit, Esc or selection")
	}
	m = workflowKey(m, "r")
	if calls != 1 {
		t.Fatal("explicit run failed")
	}
}

func TestNormalEditorsAreOnlyHighlightedWhileEditing(t *testing.T) {
	for _, id := range []string{"json.get", "regex.replace"} {
		m := workflowModel(tool.CategoryJSON)
		selectWorkspaceTool(t, &m, id)
		m.focus = focusWorkspace
		before := m.currentPage().View()
		m = workflowKey(m, "o")
		m = workflowKey(m, "esc")
		// Both normal views have structural panel borders; options never retain a selection.
		p := m.currentPage()
		if p.IsEditing() {
			t.Fatal("editor retained focus")
		}
		m = workflowKey(m, "i")
		if !p.IsEditing() {
			t.Fatal("i failed to edit input")
		}
		if strings.Contains(stripANSI(m.footerLine()), "Tab") || before == "" {
			t.Fatal("editor advertises field cycling")
		}
	}
}

func TestNormalSearchOpenRestoresActionsForBracketSwitching(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m = workflowKey(m, "s")
	m = workflowKey(m, "pretty")
	m = workflowKey(m, "enter")
	if m.currentTab() == nil || m.currentTab().Tool.ID() != "json.pretty" || m.focus != focusWorkspace {
		t.Fatal("search did not open Pretty")
	}
	m = workflowKey(m, "]")
	if m.currentTab().Tool.ID() != "json.minify" || m.focus != focusWorkspace {
		t.Fatal("search filter prevents switching related actions")
	}
}

func TestNormalDiscardedTimerKeepsCommandNotification(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m.focus = focusWorkspace
	m = workflowKey(m, "i")
	m, cmd := normalText(m, `{"x":1}`)
	msg := previewEvent(t, cmd)
	m = workflowKey(m, "esc")
	m = workflowKey(m, "X")
	status := m.status
	updated, _ := m.Update(msg)
	m = updated.(model)
	if m.status != status {
		t.Fatal("discarded preview timer replaced operation feedback")
	}
}
