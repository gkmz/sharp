package tui

import (
	"context"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/pkg/tool"
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

func TestGenericToolPageHidesInputForGenerators(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "uuid.v4",
		NameValue:        "UUID v4",
		CategoryValue:    tool.CategoryGenerator,
		DescriptionValue: "generate uuid",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			if input.Text != "" {
				t.Fatalf("generator received input %q, want empty", input.Text)
			}
			return tool.Output{Text: "id"}, nil
		},
	})
	page.SetSize(40, 12)
	view := stripANSI(page.View())
	if strings.Contains(view, "Input") {
		t.Fatalf("View() = %q, should hide Input for uuid generator", view)
	}
	if !strings.Contains(view, "Output") || strings.Contains(view, "Actions") {
		t.Fatalf("View() = %q, want Output without a duplicate Actions block", view)
	}

	page.SetInput("ignored")
	if page.input.Value() != "" {
		t.Fatalf("input after SetInput on no-input tool = %q, want empty", page.input.Value())
	}
	page.Run()
	if got, want := page.OutputText(), "id"; got != want {
		t.Fatalf("OutputText() = %q, want %q", got, want)
	}
}

func TestGenericToolPageShowsOptionsWithoutInput(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "password",
		NameValue:        "Random Password",
		CategoryValue:    tool.CategoryGenerator,
		DescriptionValue: "generate password",
		OptionsValue:     []tool.Option{{Name: "length", Description: "Password length", Default: "24"}},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			if input.Text != "" {
				t.Fatalf("password received input %q, want empty", input.Text)
			}
			return tool.Output{Text: options["length"]}, nil
		},
	})
	page.SetSize(40, 14)
	view := stripANSI(page.View())
	if strings.Contains(view, "Input") {
		t.Fatalf("View() = %q, should hide Input for password generator", view)
	}
	if !strings.Contains(view, "Options") || !strings.Contains(view, "length=24") {
		t.Fatalf("View() = %q, want options block", view)
	}
}

func TestGenericToolPageValidatesRequiredOptions(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "hmac.test",
		NameValue:        "HMAC Test",
		CategoryValue:    tool.CategoryCrypto,
		DescriptionValue: "test required option",
		OptionsValue:     []tool.Option{{Name: "key", Description: "secret", Required: true}},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			t.Fatal("RunFunc should not be called when required option is empty")
			return tool.Output{}, nil
		},
	})
	page.optionBox.SetValue("key=")
	page.Run()

	if got := page.OutputText(); !strings.Contains(got, "missing required option: key") {
		t.Fatalf("OutputText() = %q, want missing required option error", got)
	}
	if !page.outputErr {
		t.Fatal("outputErr = false, want true")
	}
}

func TestGenericToolPageParsesQuotedOptionValues(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "regex.replace",
		NameValue:        "Regex Replace",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "replace",
		OptionsValue: []tool.Option{
			{Name: "pattern", Description: "Regular expression", Required: true},
			{Name: "replacement", Description: "Replacement text"},
		},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: options["replacement"]}, nil
		},
	})
	page.optionBox.SetValue(`pattern=\d+ replacement="hello world"`)
	page.Run()

	if got, want := page.OutputText(), "hello world"; got != want {
		t.Fatalf("OutputText() = %q, want %q", got, want)
	}
}

func TestGenericToolPageRejectsUnknownOptions(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "password",
		NameValue:        "Random Password",
		CategoryValue:    tool.CategoryGenerator,
		DescriptionValue: "generate password",
		OptionsValue:     []tool.Option{{Name: "length", Description: "Password length", Default: "24"}},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			t.Fatal("RunFunc should not be called when an unknown option is present")
			return tool.Output{}, nil
		},
	})
	page.optionBox.SetValue("length=12 typo=1")
	page.Run()

	if got := page.OutputText(); !strings.Contains(got, "unknown option: typo") {
		t.Fatalf("OutputText() = %q, want unknown option error", got)
	}
}

func TestGenericToolPageMasksSecretOptionsInNormalView(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "hmac.test",
		NameValue:        "HMAC Test",
		CategoryValue:    tool.CategoryCrypto,
		DescriptionValue: "test secret option",
		OptionsValue:     []tool.Option{{Name: "key", Description: "secret", Required: true, Secret: true}},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: options["key"]}, nil
		},
	})
	page.optionBox.SetValue("key=secret")
	page.SetSize(60, 14)
	page.Run()

	view := stripANSI(page.View())
	if strings.Contains(view, "key=secret") {
		t.Fatalf("View() = %q, should not expose secret option value", view)
	}
	if !strings.Contains(view, "key=<secret>") {
		t.Fatalf("View() = %q, want masked secret option", view)
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

	page.selectedAction = "json.pretty"
	page.Run()
	if got, want := page.OutputText(), "{\n  \"b\": 2,\n  \"a\": 1\n}"; got != want {
		t.Fatalf("pretty output = %q, want %q", got, want)
	}

	page.selectedAction = "json.minify"
	page.Run()
	page.HandleKey("a")
	if got, want := page.input.Value(), `{"b":2,"a":1}`; got != want {
		t.Fatalf("input after apply = %q, want %q", got, want)
	}

	page.selectedAction = "json.escape"
	page.Run()
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
	for _, key := range []string{"p", "s", "c"} {
		if page.HandleKey(key) {
			t.Fatalf("JSON should not override shared/action selection key %s", key)
		}
	}
}

func TestJSONTabsRenderVisibleActions(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	view := stripANSI(m.tabsView(100, 3))
	for _, want := range []string{"Pretty", "Minify", "Sort", "Query", "Escape", "Unescape", "Check"} {
		if !strings.Contains(view, want) {
			t.Fatalf("JSON actions %q missing %q", view, want)
		}
	}
	if strings.Contains(view, "Workspace") {
		t.Fatal("placeholder tab remains")
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
		t.Fatalf("selectedTab after l = %d, want next action", got.selectedTab)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	got = updated.(model)
	if got.selectedTab != 0 {
		t.Fatalf("selectedTab after h = %d, want 0", got.selectedTab)
	}
}

func TestCategoryViewShowsSelectableSubcategories(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	for i, c := range model.categories {
		if c.Category == tool.CategoryEncode && c.Header {
			model.selectedCat = i
			break
		}
	}

	view := stripANSI(model.categoryView(30, 30))
	for _, want := range []string{"Encode", "  Base64", "  URL"} {
		if !strings.Contains(view, want) {
			t.Fatalf("categoryView() = %q, want category marker %q", view, want)
		}
	}
}

func TestTabsViewStaysSimpleForSelectedSubcategory(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	for i, c := range model.categories {
		if c.Category == tool.CategoryEncode && c.Subcategory == "url" {
			model.selectedCat = i
			break
		}
	}

	view := stripANSI(model.tabsView(100, 8))
	if !strings.Contains(view, "Encode") || !strings.Contains(view, "Decode") {
		t.Fatalf("tabsView() = %q, want selected subcategory tools", view)
	}
	if strings.Contains(view, "[url]") || strings.Contains(view, "Base64 Encode") {
		t.Fatalf("tabsView() = %q, want simple tools only for current subcategory", view)
	}
}

func TestCategoryNavigationSkipsParentHeaders(t *testing.T) {
	registry := tools.NewRegistry()
	m := newModel(registry)
	for i, c := range m.categories {
		if c.Category == tool.CategoryEncode && c.Header {
			m.selectedCat = i
			break
		}
	}

	m.moveCategory(1)
	if got := m.currentCategory(); got == nil || got.Header || got.Subcategory == "" {
		t.Fatalf("currentCategory after moving from header = %#v, want selectable subcategory", got)
	}
}

func TestSearchFiltersConcreteTools(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	model.search.SetValue("hmac")
	model.applySearch()

	if len(model.categories) != 1 || model.categories[0].Category != tool.CategoryCrypto || model.categories[0].Subcategory != "hmac" {
		t.Fatalf("categories after hmac search = %#v, want only Crypto", model.categories)
	}
	view := stripANSI(model.tabsView(100, 5))
	if !strings.Contains(view, "SHA256") {
		t.Fatalf("tabsView() after hmac search = %q, want hmac tools", view)
	}
	if strings.Contains(view, "MD5") {
		t.Fatalf("tabsView() after hmac search = %q, should not include unmatched hash tools", view)
	}
}

func TestSearchUsesFuzzyMatch(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	model.search.SetValue("jv")
	model.applySearch()

	if len(model.categories) != 1 || model.categories[0].Category != tool.CategoryJSON {
		t.Fatalf("categories after fuzzy search = %#v, want JSON", model.categories)
	}
	view := stripANSI(model.tabsView(100, 5))
	if !strings.Contains(view, "Check") {
		t.Fatalf("tabsView() after fuzzy search = %q, want matching Check action", view)
	}
}

func TestSearchEnterFocusesSingleToolResult(t *testing.T) {
	registry := tools.NewRegistry()
	m := newModel(registry)
	m.focus = focusSearch
	m.searchFrom = focusCategories
	m.search.Focus()
	m.search.SetValue("jwt.decode")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(model)
	if got.focus != focusWorkspace {
		t.Fatalf("focus after single search result enter = %v, want workspace", got.focus)
	}
	tab := got.currentTab()
	if tab == nil || tab.Tool.ID() != "jwt.decode" {
		t.Fatalf("currentTab after search enter = %#v, want jwt.decode", tab)
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
	if !strings.Contains(help, "Edit Path") || !strings.Contains(help, "Apply Output") || !strings.Contains(help, "Run Action") {
		t.Fatalf("helpView() = %q, want JSON command details", help)
	}
	if !strings.Contains(help, "Path field") || !strings.Contains(help, "selected JSON action") {
		t.Fatalf("helpView() = %q, want path enter instructions", help)
	}

	model.help.SetValue("path")
	help = model.helpContent()
	if !strings.Contains(help, "Edit Path") || strings.Contains(help, "Save Output") {
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
	for _, want := range []string{"NORMAL", "[i] edit", "[v] paste", "[x] clear", "[p] apply", "[y] copy", "[?] help"} {
		if !strings.Contains(footer, want) {
			t.Fatalf("footerLine() = %q, want %q", footer, want)
		}
	}
}

func TestGenericFooterHidesUnavailableOptionsShortcut(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "uuid.v4",
		NameValue:        "UUID v4",
		CategoryValue:    tool.CategoryGenerator,
		DescriptionValue: "generate UUID",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: "id"}, nil
		},
	})

	footer := stripANSI(page.FooterHints())
	if strings.Contains(footer, "[o] options") {
		t.Fatalf("FooterHints() = %q, should not show options shortcut for optionless tool", footer)
	}
	if !strings.Contains(footer, "[r/Enter] run") || !strings.Contains(footer, "[?] help") {
		t.Fatalf("FooterHints() = %q, want common no-input shortcuts", footer)
	}
}

func TestGenericFooterShowsOptionsShortcutWhenAvailable(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "password",
		NameValue:        "Random Password",
		CategoryValue:    tool.CategoryGenerator,
		DescriptionValue: "generate password",
		OptionsValue:     []tool.Option{{Name: "length", Description: "Password length", Default: "24"}},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: "secret"}, nil
		},
	})

	footer := stripANSI(page.FooterHints())
	if !strings.Contains(footer, "[o] options") {
		t.Fatalf("FooterHints() = %q, want options shortcut for configurable tool", footer)
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
	page.selectedAction = "json.get"
	page.FocusOptions()
	page.pathBox.SetValue("data.name")
	page.startEditingFocusedField()

	_, _ = page.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got, want := page.OutputText(), "sharp"; got != want {
		t.Fatalf("OutputText() = %q, want %q", got, want)
	}
	if page.IsEditing() {
		t.Fatal("path enter should leave edit mode after query")
	}
}

func TestJSONInputAutoValidatesWhenSet(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	for i, c := range model.categories {
		if c.Category == tool.CategoryJSON {
			model.selectedCat = i
			break
		}
	}
	page := model.currentPage().(*jsonToolPage)
	page.SetSize(80, 20)

	page.SetInput(`{"ok":true}`)
	if got := stripANSI(page.Status()); got != "valid JSON" {
		t.Fatalf("Status() after valid input = %q, want valid JSON", got)
	}
	if view := stripANSI(page.View()); !strings.Contains(view, "Input - valid JSON") {
		t.Fatalf("View() = %q, want input validity label", view)
	}

	page.SetInput(`{"ok":`)
	if got := stripANSI(page.Status()); !strings.Contains(got, "invalid JSON") {
		t.Fatalf("Status() after invalid input = %q, want invalid JSON", got)
	}
	if view := stripANSI(page.View()); !strings.Contains(view, "Input - invalid JSON") {
		t.Fatalf("View() = %q, want invalid label", view)
	}
}

func TestJSONFocusSelectsBeforeInsertMode(t *testing.T) {
	registry := tools.NewRegistry()
	model := newModel(registry)
	for i, c := range model.categories {
		if c.Category == tool.CategoryJSON {
			model.selectedCat = i
			break
		}
	}
	page := model.currentPage().(*jsonToolPage)

	page.FocusInput()
	if page.IsEditing() {
		t.Fatal("FocusInput should select JSON input without entering insert mode")
	}

	if !page.HandleKey("i") {
		t.Fatal("JSON page should handle i for insert mode")
	}
	if !page.IsEditing() {
		t.Fatal("i should enter insert mode for selected JSON input")
	}

	_, _ = page.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if page.IsEditing() {
		t.Fatal("Esc should leave JSON insert mode")
	}

	page.selectedAction = "json.get"
	page.FocusOptions()
	if page.IsEditing() {
		t.Fatal("o should select path without entering insert mode")
	}
	page.HandleKey("o")
	if !page.IsEditing() || page.field != editFieldOptions {
		t.Fatalf("o should directly edit path, editing=%v focus=%v", page.IsEditing(), page.field)
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
	before := page.output.YOffset
	page.OutputHalfPageDown()
	if page.output.YOffset <= before {
		t.Fatalf("output YOffset after HalfPageDown = %d, want > %d", page.output.YOffset, before)
	}
}

func TestWorkspaceCtrlDScrollsOutputEvenWhenInputFocused(t *testing.T) {
	registry := tool.NewRegistry()
	registry.MustRegister(tool.SimpleTool{
		IDValue:          "test.scroll",
		NameValue:        "Scroll",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "scroll",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{}, nil
		},
	})
	m := newModel(registry)
	m.width = 100
	m.height = 30
	m.focus = focusWorkspace
	page := m.currentPage()
	page.SetSize(60, 14)
	page.ClearOutput()
	page.(*genericToolPage).outputText = strings.Join([]string{
		"line 1", "line 2", "line 3", "line 4", "line 5", "line 6", "line 7", "line 8", "line 9", "line 10",
	}, "\n")
	page.(*genericToolPage).renderOutput()
	page.(*genericToolPage).stopEditing()
	before := page.(*genericToolPage).output.YOffset

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	got := updated.(model)
	gotPage := got.currentPage().(*genericToolPage)
	if gotPage.output.YOffset <= before {
		t.Fatalf("output YOffset after ctrl+d = %d, want > %d", gotPage.output.YOffset, before)
	}
}

func TestAltUScrollsInput(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "test.input",
		NameValue:        "Input",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "input scroll",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{}, nil
		},
	})
	page.SetSize(30, 8)
	page.SetInput(strings.Join([]string{
		"line 1", "line 2", "line 3", "line 4", "line 5", "line 6", "line 7", "line 8",
	}, "\n"))
	before := page.input.Line()
	page.InputHalfPageUp()
	if page.input.Line() >= before {
		t.Fatalf("input line after InputHalfPageUp = %d, want < %d", page.input.Line(), before)
	}
}

func TestGenericInputUndoRedo(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "test.undo",
		NameValue:        "Undo",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "undo",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{}, nil
		},
	})
	page.FocusInput()
	page.HandleKey("i")
	page.input.SetValue("one")
	page.trackInputChange("")

	_, _ = page.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	if got, want := page.input.Value(), ""; got != want {
		t.Fatalf("input after undo = %q, want %q", got, want)
	}
	_, _ = page.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	if got, want := page.input.Value(), "one"; got != want {
		t.Fatalf("input after redo = %q, want %q", got, want)
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
