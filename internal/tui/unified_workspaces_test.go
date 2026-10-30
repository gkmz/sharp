package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/pkg/tool"
)

func selectWorkspaceTool(t *testing.T, m *model, id string) {
	t.Helper()
	for ci, c := range m.categories {
		if c.Header {
			continue
		}
		for ti, tab := range c.Tabs {
			if tab.Tool.ID() == id {
				m.selectedCat = ci
				m.setSelectedTab(ti)
				m.rebuildPage()
				return
			}
		}
	}
	t.Fatalf("missing tool %s", id)
}

func TestUnifiedWorkspaceSharesInputWithinGroup(t *testing.T) {
	for _, pair := range [][2]string{{"text.upper", "text.lower"}, {"b64.encode", "b64.decode"}, {"hash.sha256", "hash.sha512"}, {"regex.test", "regex.replace"}} {
		t.Run(pair[0], func(t *testing.T) {
			m := workflowModel(tool.CategoryText)
			selectWorkspaceTool(t, &m, pair[0])
			first := m.currentPage().(*genericToolPage)
			first.SetInput("shared input")
			selectWorkspaceTool(t, &m, pair[1])
			second := m.currentPage().(*genericToolPage)
			if second != first || second.input.Value() != "shared input" {
				t.Fatal("switching related actions lost the shared document")
			}
			second.HandleKey("z")
			if second.input.Value() != "" {
				t.Fatal("undo history did not follow the shared document")
			}
		})
	}
}

func TestUnifiedWorkspaceActionsPreviewOriginalInput(t *testing.T) {
	m := workflowModel(tool.CategoryText)
	selectWorkspaceTool(t, &m, "text.upper")
	p := m.currentPage().(*genericToolPage)
	p.SetInput("Hello Sharp")
	p.Run()
	cat := m.currentCategory()
	target := 0
	for i, tab := range cat.Tabs {
		if tab.Tool.ID() == "text.lower" {
			target = i
		}
	}
	m.focus = focusTabs
	m.moveTab(target - m.selectedTab)
	p = m.currentPage().(*genericToolPage)
	if p.input.Value() != "Hello Sharp" || p.OutputText() != "hello sharp" {
		t.Fatalf("preview changed source or failed: input=%q output=%q", p.input.Value(), p.OutputText())
	}
}

func TestUnifiedWorkspaceBase64ApplyAndDecode(t *testing.T) {
	m := workflowModel(tool.CategoryEncode)
	selectWorkspaceTool(t, &m, "b64.encode")
	m.focus = focusWorkspace
	p := m.currentPage().(*genericToolPage)
	p.SetInput("hello")
	m = workflowKey(m, "r")
	m = workflowKey(m, "p")
	if p.input.Value() != "aGVsbG8=" || p.OutputText() != "" {
		t.Fatal("Apply must replace Input and clear Output, matching JSON")
	}
	selectWorkspaceTool(t, &m, "b64.decode")
	m = workflowKey(m, "r")
	if m.currentPage().OutputText() != "hello" {
		t.Fatal("decode did not receive applied input")
	}
	m = workflowKey(m, "z")
	if p.input.Value() != "hello" {
		t.Fatal("undo did not restore original input across actions")
	}
}

func TestUnifiedWorkspaceGroupsStayIsolated(t *testing.T) {
	m := workflowModel(tool.CategoryEncode)
	selectWorkspaceTool(t, &m, "url.encode")
	encoded := m.currentPage().(*genericToolPage)
	encoded.SetInput("encoded document")
	selectWorkspaceTool(t, &m, "url.parse")
	parsed := m.currentPage().(*genericToolPage)
	if parsed == encoded || parsed.input.Value() != "" {
		t.Fatal("same prefix in different categories leaked document")
	}
	selectWorkspaceTool(t, &m, "url.encode")
	if m.currentPage() != encoded || encoded.input.Value() != "encoded document" {
		t.Fatal("returning to group lost document")
	}
}

func TestUnifiedWorkspaceOptionDraftsFollowAction(t *testing.T) {
	m := workflowModel(tool.CategoryText)
	selectWorkspaceTool(t, &m, "regex.test")
	p := m.currentPage().(*genericToolPage)
	p.SetInput("hello world")
	p.optionBox.SetValue(`pattern="hello world"`)
	selectWorkspaceTool(t, &m, "regex.replace")
	replace := m.currentPage().(*genericToolPage)
	replace.optionBox.SetValue(`pattern=world replacement=sharp`)
	selectWorkspaceTool(t, &m, "regex.test")
	p = m.currentPage().(*genericToolPage)
	if p.optionBox.Value() != `pattern="hello world"` {
		t.Fatalf("Match draft=%q", p.optionBox.Value())
	}
	selectWorkspaceTool(t, &m, "regex.replace")
	p = m.currentPage().(*genericToolPage)
	if p.optionBox.Value() != `pattern=world replacement=sharp` {
		t.Fatalf("Replace draft=%q", p.optionBox.Value())
	}
	m.focus = focusWorkspace
	m = workflowKey(m, "o")
	m = workflowKey(m, "enter")
	if p.IsEditing() || p.OutputText() != "hello sharp" {
		t.Fatalf("parameter Enter did not run: editing=%t output=%q", p.IsEditing(), p.OutputText())
	}
}

func TestUnifiedWorkspaceErrorsCannotBeApplied(t *testing.T) {
	for _, id := range []string{"b64.decode", "regex.test", "yaml.to_json"} {
		t.Run(id, func(t *testing.T) {
			m := workflowModel(tool.CategoryEncode)
			selectWorkspaceTool(t, &m, id)
			m.focus = focusWorkspace
			p := m.currentPage().(*genericToolPage)
			p.SetInput("[invalid input")
			m = workflowKey(m, "r")
			if !p.outputErr {
				t.Fatal("test needs failed action")
			}
			m = workflowKey(m, "p")
			if p.input.Value() != "[invalid input" || !p.outputErr {
				t.Fatal("Apply consumed an error result")
			}
		})
	}
}

func TestUnifiedWorkspaceInputlessActionPreservesHiddenInput(t *testing.T) {
	registry := tool.NewRegistry()
	registry.MustRegister(tool.SimpleTool{IDValue: "time.from", NameValue: "From", CategoryValue: tool.CategoryTime, RunFunc: func(_ context.Context, in tool.Input, _ tool.Options) (tool.Output, error) {
		return tool.Output{Text: in.Text}, nil
	}})
	registry.MustRegister(tool.SimpleTool{IDValue: "time.now", NameValue: "Now", CategoryValue: tool.CategoryTime, RunFunc: func(_ context.Context, in tool.Input, _ tool.Options) (tool.Output, error) {
		if in.Text != "" {
			t.Fatalf("no-input action received %q", in.Text)
		}
		return tool.Output{Text: "now"}, nil
	}})
	m := newModel(registry)
	m.width, m.height = 100, 30
	selectWorkspaceTool(t, &m, "time.from")
	p := m.currentPage().(*genericToolPage)
	p.SetInput("1714723200")
	selectWorkspaceTool(t, &m, "time.now")
	p = m.currentPage().(*genericToolPage)
	p.Run()
	if strings.HasPrefix(stripANSI(p.View()), " Input ") {
		t.Fatal("no-input action shows Input")
	}
	selectWorkspaceTool(t, &m, "time.from")
	if m.currentPage() != p || p.input.Value() != "1714723200" {
		t.Fatal("hidden source document was lost")
	}
}

func TestUnifiedWorkspaceGeneratorsNeedExplicitRun(t *testing.T) {
	m := workflowModel(tool.CategoryGenerator)
	selectWorkspaceTool(t, &m, "password")
	p := m.currentPage().(*genericToolPage)
	p.optionBox.SetValue("length=8")
	p.Run()
	original := p.OutputText()
	m.focus = focusTabs
	m.moveTab(1)
	if m.currentPage() != p || p.OutputText() != original {
		t.Fatal("generator switch should preserve output without generating")
	}
	p.optionBox.SetValue("bytes=12")
	selectWorkspaceTool(t, &m, "password")
	if p.optionBox.Value() != "length=8" {
		t.Fatal("password settings lost on action switch")
	}
	m.focus = focusWorkspace
	m = workflowKey(m, "o")
	m = workflowKey(m, "enter")
	if p.IsEditing() || len(p.OutputText()) != 8 {
		t.Fatal("generator parameter Enter did not execute with restored settings")
	}
}

func TestUnifiedWorkspaceSecretDraftsStayMasked(t *testing.T) {
	m := workflowModel(tool.CategoryCrypto)
	selectWorkspaceTool(t, &m, "hmac.sha256")
	p := m.currentPage().(*genericToolPage)
	p.SetInput("message")
	p.optionBox.SetValue("key=do-not-show-this")
	selectWorkspaceTool(t, &m, "hmac.sha512")
	selectWorkspaceTool(t, &m, "hmac.sha256")
	p = m.currentPage().(*genericToolPage)
	if p.optionBox.Value() != "key=do-not-show-this" {
		t.Fatal("secret draft was not restored in memory")
	}
	m.focus = focusWorkspace
	m = workflowKey(m, "o")
	if strings.Contains(stripANSI(m.View()), "do-not-show-this") {
		t.Fatal("restored secret exposed in edit view")
	}
	m = workflowKey(m, "esc")
	if strings.Contains(stripANSI(m.View()), "do-not-show-this") {
		t.Fatal("restored secret exposed in normal view")
	}
}

func TestUnifiedWorkspaceMalformedSecretOptionsStayMasked(t *testing.T) {
	for _, draft := range []string{"key=secret unquoted-secret-tail", "key=secret unquoted-secret-tail=value"} {
		m := workflowModel(tool.CategoryCrypto)
		selectWorkspaceTool(t, &m, "hmac.sha256")
		p := m.currentPage().(*genericToolPage)
		p.SetInput("message")
		p.optionBox.SetValue(draft)
		m.focus = focusWorkspace
		m = workflowKey(m, "o")
		m = workflowKey(m, "enter")
		if !p.outputErr || p.IsEditing() {
			t.Fatal("malformed options must fail and leave editing")
		}
		if strings.Contains(stripANSI(m.View())+p.OutputText(), "unquoted-secret-tail") {
			t.Fatal("option parser error exposed secret fragments")
		}
	}
}

func TestUnifiedWorkspaceParameterUndoDoesNotChangeInput(t *testing.T) {
	for _, id := range []string{"regex.replace", "json.get"} {
		for _, key := range []tea.KeyType{tea.KeyCtrlZ, tea.KeyCtrlY} {
			t.Run(fmt.Sprintf("%s/%d", id, key), func(t *testing.T) {
				m := workflowModel(tool.CategoryText)
				selectWorkspaceTool(t, &m, id)
				p := m.currentPage()
				p.SetInput("original input")
				p.SetInput("precious input")
				if key == tea.KeyCtrlY {
					p.HandleKey("z")
				}
				input := func() string {
					switch page := p.(type) {
					case *genericToolPage:
						return page.input.Value()
					case *jsonToolPage:
						return page.input.Value()
					}
					panic("unexpected page")
				}
				before := input()
				m.focus = focusWorkspace
				m = workflowKey(m, "o")
				m.Update(tea.KeyMsg{Type: key})
				if input() != before {
					t.Fatal("undo/redo while editing parameters changed Input")
				}
			})
		}
	}
}

func TestUnifiedWorkspaceOutputNamesProducingAction(t *testing.T) {
	m := workflowModel(tool.CategoryText)
	selectWorkspaceTool(t, &m, "text.upper")
	p := m.currentPage().(*genericToolPage)
	p.SetInput("hello")
	p.Run()
	selectWorkspaceTool(t, &m, "text.lower")
	p = m.currentPage().(*genericToolPage)
	if !strings.Contains(stripANSI(p.View()), "Output · Uppercase") {
		t.Fatal("output must identify producing action even before next execution")
	}
}

func TestUnifiedWorkspaceOptionsSummaryShowsCurrentDraft(t *testing.T) {
	m := workflowModel(tool.CategoryText)
	selectWorkspaceTool(t, &m, "regex.replace")
	p := m.currentPage().(*genericToolPage)
	m.focus = focusWorkspace
	m = workflowKey(m, "o")
	p.optionBox.SetValue(`pattern=hello replacement=sharp`)
	m = workflowKey(m, "esc")
	if !strings.Contains(stripANSI(p.View()), "replacement=sharp") {
		t.Fatal("normal Options view shows stale values after editing")
	}
}

func TestUnifiedWorkspaceAllRegisteredToolsFit(t *testing.T) {
	for _, current := range tools.NewRegistry().All() {
		for _, size := range [][2]int{{80, 24}, {120, 40}} {
			t.Run(fmt.Sprintf("%s/%dx%d", current.ID(), size[0], size[1]), func(t *testing.T) {
				m := workflowModel(current.Category())
				m.width, m.height = size[0], size[1]
				selectWorkspaceTool(t, &m, current.ID())
				m.focus = focusWorkspace
				page := m.currentPage()
				view := m.View()
				if lipgloss.Width(view) > m.width || lipgloss.Height(view) > m.height {
					t.Fatal("workspace exceeds terminal")
				}
				toolsHeight := clamp(m.toolRows(m.width-clamp(m.width*20/100, 22, max(22, m.width-60))-2)+2, 3, max(3, m.height-2-8))
				budget := panelContentHeight(m.height-2-toolsHeight) - 1
				if lipgloss.Height(page.View()) > budget {
					t.Fatalf("page uses %d rows, budget %d", lipgloss.Height(page.View()), budget)
				}
				if !strings.Contains(stripANSI(m.footerLine()), "[?] help") {
					t.Fatal("workspace lacks common help hint")
				}
			})
		}
	}
}

func TestUnifiedWorkspaceRootParamEnterMatchesJSON(t *testing.T) {
	m := workflowModel(tool.CategoryText)
	selectWorkspaceTool(t, &m, "regex.replace")
	m.focus = focusWorkspace
	m = workflowKey(m, "i")
	m = workflowKey(m, "hello")
	m = workflowKey(m, "esc")
	m = workflowKey(m, "o")
	p := m.currentPage().(*genericToolPage)
	p.optionBox.SetValue("pattern=hello replacement=sharp")
	m = workflowKey(m, "enter")
	if m.currentPage().IsEditing() || m.currentPage().OutputText() != "sharp" {
		t.Fatal("shared run/edit contract differs from JSON")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if m.currentPage().IsEditing() {
		t.Fatal("field navigation should not enter editing")
	}
}

func TestUnifiedWorkspaceEveryFooterFitsVisibleTerminal(t *testing.T) {
	for _, current := range tools.NewRegistry().All() {
		m := workflowModel(current.Category())
		m.width, m.height = 80, 24
		selectWorkspaceTool(t, &m, current.ID())
		m.focus = focusWorkspace
		if lipgloss.Width(m.footerLine()) > m.width {
			t.Errorf("%s footer width=%d exceeds 80", current.ID(), lipgloss.Width(m.footerLine()))
		}
		lines := strings.Split(stripANSI(m.View()), "\n")
		if !strings.Contains(lines[len(lines)-1], "[?] help") {
			t.Errorf("%s hides Help offscreen", current.ID())
		}
	}
}

func TestUnifiedWorkspaceOptionsShortcutDoesNotEditInput(t *testing.T) {
	for _, id := range []string{"json.pretty", "text.upper", "uuid.v4"} {
		m := workflowModel(tool.CategoryText)
		selectWorkspaceTool(t, &m, id)
		m.focus = focusWorkspace
		m = workflowKey(m, "o")
		if m.currentPage().IsEditing() {
			t.Errorf("%s: o entered Input instead of reporting no options", id)
		}
	}
}

func TestUnifiedWorkspaceSearchReturnsToSharedTextInput(t *testing.T) {
	m := workflowModel(tool.CategoryText)
	selectWorkspaceTool(t, &m, "text.upper")
	p := m.currentPage().(*genericToolPage)
	p.SetInput("Hello")
	m.focus = focusCategories
	m = workflowKey(m, "text.lower")
	m = workflowKey(m, "enter")
	m = workflowKey(m, "r")
	if m.currentPage() != p || p.OutputText() != "hello" {
		t.Fatal("search opened a separate document for a related action")
	}
	m.clearCategorySearch()
	selectWorkspaceTool(t, &m, "json.pretty")
	selectWorkspaceTool(t, &m, "text.lower")
	if m.currentPage() != p || p.input.Value() != "Hello" {
		t.Fatal("category round trip lost shared text source")
	}
}
