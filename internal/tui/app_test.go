package tui

import (
	"context"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestJSONTabsMoveWithHL(t *testing.T) {
	registry := tools.NewRegistry()
	m := newModel(registry)
	for i, c := range m.categories {
		if c.Category == tool.CategoryJSON {
			m.selectedCat = i
			break
		}
	}
	m.focus = focusTabs

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	got := updated.(model)
	if got.selectedTab != 1 {
		t.Fatalf("selectedTab after l = %d, want 1", got.selectedTab)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	got = updated.(model)
	if got.selectedTab != 0 {
		t.Fatalf("selectedTab after h = %d, want 0", got.selectedTab)
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

	help := model.helpContent()
	if !strings.Contains(help, "Global") || !strings.Contains(help, "Current Area") || !strings.Contains(help, "Current Tool") {
		t.Fatalf("helpView() = %q, want grouped sections", help)
	}
	if !strings.Contains(help, "Edit Path") || !strings.Contains(help, "Apply Output") || !strings.Contains(help, "Get Path") {
		t.Fatalf("helpView() = %q, want JSON command details", help)
	}
	if !strings.Contains(help, "Path field") || !strings.Contains(help, "query immediat") {
		t.Fatalf("helpView() = %q, want path enter instructions", help)
	}

	model.help.SetValue("path")
	help = model.helpContent()
	if !strings.Contains(help, "Edit Path") || strings.Contains(help, "Pretty JSON") {
		t.Fatalf("filtered helpView() = %q, want only path-related commands", help)
	}
}

func TestJSONFooterShowsCommonActions(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	for i, c := range model.categories {
		if c.Category == tool.CategoryJSON {
			model.selectedCat = i
			break
		}
	}
	model.focus = focusWorkspace

	footer := stripANSI(model.footerLine())
	for _, want := range []string{"[p] pretty", "[m] minify", "[c] check", "[e] escape", "[g] get", "[?] help"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("footerLine() = %q, want %q", footer, want)
		}
	}
}

func TestStatusHintHighlightsShortcutKey(t *testing.T) {
	hint := statusHint("p", "pretty")
	if got := stripANSI(hint); got != "[p] pretty" {
		t.Fatalf("plain statusHint() = %q, want key and label", got)
	}
}

func stripANSI(s string) string {
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansi.ReplaceAllString(s, "")
}

func TestJSONPathEnterRunsQuery(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	for i, c := range model.categories {
		if c.Category == tool.CategoryJSON {
			model.selectedCat = i
			break
		}
	}
	page := model.currentPage().(*jsonToolPage)
	page.SetInput(`{"data":{"name":"sharp"}}`)
	page.FocusOptions()
	page.pathBox.SetValue("data.name")

	_, _ = page.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got, want := page.OutputText(), "sharp"; got != want {
		t.Fatalf("OutputText() = %q, want %q", got, want)
	}
	if page.IsEditing() {
		t.Fatal("path enter should leave edit mode after query")
	}
}

func TestOutputHalfPageScrollUsesFocusedOutput(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "test.long",
		NameValue:        "Long",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "long output",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: strings.Join([]string{
				"line 1", "line 2", "line 3", "line 4", "line 5", "line 6", "line 7", "line 8",
			}, "\n")}, nil
		},
	})
	page.SetSize(30, 10)
	page.Run()
	page.FocusOutput()
	before := page.output.YOffset
	page.HalfPageDown()
	if page.output.YOffset <= before {
		t.Fatalf("output YOffset after HalfPageDown = %d, want > %d", page.output.YOffset, before)
	}
}

func TestHelpHalfPageScroll(t *testing.T) {
	registry := tools.NewRegistry()
	m := newModel(registry)
	m.width = 100
	m.height = 14
	m.focus = focusWorkspace
	m.resize()
	m.openHelp()

	before := m.helpVP.YOffset
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	got := updated.(model)
	if got.helpVP.YOffset <= before {
		t.Fatalf("help YOffset after ctrl+d = %d, want > %d", got.helpVP.YOffset, before)
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
