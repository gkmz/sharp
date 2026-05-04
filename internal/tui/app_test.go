package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hank/sharp/internal/tools"
	"github.com/hank/sharp/pkg/tool"
)

func TestGenericToolPageOutputTextUsesTrimmedRawOutput(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "test.echo",
		NameValue:        "Echo",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "test echo",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: "\n  alpha\nbeta  \n\n"}, nil
		},
	})
	page.SetSize(20, 8)
	page.Run()

	if got, want := page.OutputText(), "alpha\nbeta"; got != want {
		t.Fatalf("OutputText() = %q, want %q", got, want)
	}
	if got := page.output.View(); got == page.OutputText() {
		t.Fatalf("viewport view unexpectedly matched raw output; test should guard against copying rendered output")
	}
}

func TestGenericToolPageClearOutputClearsRawOutput(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "test.echo",
		NameValue:        "Echo",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "test echo",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: "result"}, nil
		},
	})
	page.Run()
	page.ClearOutput()

	if got := page.OutputText(); got != "" {
		t.Fatalf("OutputText() after ClearOutput() = %q, want empty", got)
	}
}

func TestGenericToolPageWrapsRenderedOutput(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "test.echo",
		NameValue:        "Echo",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "test echo",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJK"}, nil
		},
	})
	page.SetSize(10, 8)
	page.Run()

	if got := page.output.TotalLineCount(); got < 3 {
		t.Fatalf("TotalLineCount() = %d, want wrapped output across at least 3 lines", got)
	}
	if got, want := page.OutputText(), "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJK"; got != want {
		t.Fatalf("OutputText() = %q, want raw output %q", got, want)
	}
}

func TestWrapDisplayTextUsesTerminalWidth(t *testing.T) {
	if got, want := wrapDisplayText("中文abcd", 4), "中文\nabcd"; got != want {
		t.Fatalf("wrapDisplayText() = %q, want %q", got, want)
	}
}

func TestWrapActionLineKeepsActionsVisible(t *testing.T) {
	got := wrapActionLine("Actions:", []string{
		"[p] Pretty",
		"[m] Minify",
		"[c] Check",
		"[s] Sort",
		"[e] Escape",
		"[u] Unescape",
		"[g] Get Path",
	}, 34)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("wrapActionLine() = %q, want multiple lines", got)
	}
	for _, line := range lines {
		if width := lipgloss.Width(line); width > 34 {
			t.Fatalf("line %q width = %d, want <= 34", line, width)
		}
	}
	if !strings.Contains(got, "[g] Get Path") {
		t.Fatalf("wrapActionLine() = %q, want later actions still visible", got)
	}
}

func TestJSONToolPageCanChainActionsWithApplyAndUndo(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	for i, c := range model.categories {
		if c.Category == tool.CategoryJSON {
			model.selectedCat = i
			break
		}
	}

	page, ok := model.currentPage().(*jsonToolPage)
	if !ok {
		t.Fatalf("currentPage() = %T, want *jsonToolPage", model.currentPage())
	}
	page.SetInput(`{"b":2,"a":1}`)

	if !page.HandleKey("p") {
		t.Fatal("pretty action was not handled")
	}
	if got, want := page.OutputText(), "{\n  \"b\": 2,\n  \"a\": 1\n}"; got != want {
		t.Fatalf("pretty output = %q, want %q", got, want)
	}

	page.HandleKey("m")
	page.HandleKey("a")
	if got, want := page.input.Value(), `{"b":2,"a":1}`; got != want {
		t.Fatalf("input after apply = %q, want %q", got, want)
	}

	page.HandleKey("e")
	if got, want := page.OutputText(), `"{\"b\":2,\"a\":1}"`; got != want {
		t.Fatalf("escape output = %q, want %q", got, want)
	}

	page.HandleKey("z")
	if got, want := page.input.Value(), `{"b":2,"a":1}`; got != want {
		t.Fatalf("input after undo = %q, want previous input %q", got, want)
	}
}

func TestJSONToolPageDoesNotClaimPasteShortcut(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	for i, c := range model.categories {
		if c.Category == tool.CategoryJSON {
			model.selectedCat = i
			break
		}
	}

	page := model.currentPage()
	if page.HandleKey("v") {
		t.Fatal("JSON page should not claim v; v is reserved for paste")
	}
	if !page.HandleKey("c") {
		t.Fatal("JSON page should claim c for validate/check")
	}
}

func TestJSONTabsRenderPagesNotAtomicTools(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	for i, c := range model.categories {
		if c.Category == tool.CategoryJSON {
			model.selectedCat = i
			break
		}
	}

	view := model.tabsView(80, 3)
	if !strings.Contains(view, "Workspace") || !strings.Contains(view, "Actions") {
		t.Fatalf("JSON tabsView() = %q, want workspace pages", view)
	}
	if strings.Contains(view, "Pretty JSON") {
		t.Fatalf("JSON tabsView() = %q, should not show atomic JSON tools", view)
	}
}

func TestHelpOverlayShowsAndFiltersJSONCommands(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	model.width = 100
	model.height = 30
	for i, c := range model.categories {
		if c.Category == tool.CategoryJSON {
			model.selectedCat = i
			break
		}
	}
	model.focus = focusWorkspace
	model.resize()
	model.openHelp()

	help := model.helpView()
	if !strings.Contains(help, "Edit Path") || !strings.Contains(help, "Apply Output") {
		t.Fatalf("helpView() = %q, want JSON command details", help)
	}

	model.help.SetValue("path")
	help = model.helpView()
	if !strings.Contains(help, "Edit Path") || strings.Contains(help, "Pretty JSON") {
		t.Fatalf("filtered helpView() = %q, want only path-related commands", help)
	}
}

func TestJSONActionsViewWrapsWithinPageWidth(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	for i, c := range model.categories {
		if c.Category == tool.CategoryJSON {
			model.selectedCat = i
			break
		}
	}
	page := model.currentPage().(*jsonToolPage)
	page.SetSize(42, 18)

	actions := page.actionsView()
	if !strings.Contains(actions, "[z] Undo") {
		t.Fatalf("actionsView() = %q, want trailing actions visible", actions)
	}
	for _, line := range strings.Split(actions, "\n") {
		if width := lipgloss.Width(line); width > 42 {
			t.Fatalf("actions line %q width = %d, want <= 42", line, width)
		}
	}
}

func TestQuestionMarkOpensHelpOverlay(t *testing.T) {
	registry := tools.NewRegistry()
	m := newModel(registry)
	m.width = 100
	m.height = 30
	m.focus = focusWorkspace

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	got := updated.(model)
	if !got.showHelp {
		t.Fatal("? should open help overlay")
	}
}
