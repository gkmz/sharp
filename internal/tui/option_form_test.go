package tui

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/pkg/tool"
)

func formKey(m model, key tea.KeyType) model {
	updated, _ := m.Update(tea.KeyMsg{Type: key})
	return updated.(model)
}

func TestOptionFormEditsOnlyValuesAndNavigatesFields(t *testing.T) {
	m := workflowModel(tool.CategoryGenerator)
	selectWorkspaceTool(t, &m, "token")
	m.focus = focusWorkspace
	p := m.currentPage().(*genericToolPage)
	m = workflowKey(m, "o")
	m = formKey(m, tea.KeyCtrlE)
	m = formKey(m, tea.KeyCtrlU)
	m = workflowKey(m, "64")
	m = formKey(m, tea.KeyTab)
	m = formKey(m, tea.KeyRight) // base64url -> hex
	if !p.IsEditing() || m.focus != focusWorkspace || p.OutputText() != "" {
		t.Fatal("field navigation left the form or generated a token")
	}
	m = workflowKey(m, "enter")
	if p.options["bytes"] != "64" || p.options["format"] != "hex" {
		t.Fatalf("form values = %v; labels must survive value replacement", p.options)
	}
	if raw, err := hex.DecodeString(p.OutputText()); err != nil || len(raw) != 64 || p.IsEditing() {
		t.Fatal("form submit did not generate the selected token and return to Normal")
	}
}

func TestOptionFormAcceptsLiteralSpacesQuotesAndEquals(t *testing.T) {
	m := workflowModel(tool.CategoryText)
	selectWorkspaceTool(t, &m, "regex.replace")
	m.focus = focusWorkspace
	p := m.currentPage().(*genericToolPage)
	p.SetInput("hello")
	m = workflowKey(m, "o")
	m = workflowKey(m, "hello")
	m = formKey(m, tea.KeyTab)
	want := `say "hello world" key=a\b`
	m = workflowKey(m, want)
	m = workflowKey(m, "enter")
	if p.OutputText() != want {
		t.Fatalf("literal replacement = %q, want %q", p.OutputText(), want)
	}
	lines := strings.Split(stripANSI(p.View()), "\n")
	var patternRow, replacementRow int
	for i, line := range lines {
		if strings.Contains(line, "pattern") {
			patternRow = i
		}
		if strings.Contains(line, "replacement") {
			replacementRow = i
		}
	}
	if patternRow == 0 || replacementRow != patternRow+1 {
		t.Fatal("each parameter must occupy its own form row")
	}
}

func TestOptionFormKeepsLabelsAndChoiceValuesProtected(t *testing.T) {
	m := workflowModel(tool.CategoryGenerator)
	selectWorkspaceTool(t, &m, "token")
	m.focus = focusWorkspace
	p := m.currentPage().(*genericToolPage)
	m = workflowKey(m, "o")
	for range 10 {
		m = formKey(m, tea.KeyBackspace)
	}
	if values := p.optionForm.Values(); values["bytes"] != "" || len(values) != 2 {
		t.Fatalf("Backspace should clear only the value: %v", values)
	}
	m = formKey(m, tea.KeyShiftTab) // first -> last
	m = workflowKey(m, "not-a-format")
	m = formKey(m, tea.KeyBackspace)
	if p.optionForm.Values()["format"] != "base64url" {
		t.Fatal("typing changed a choice to an unsupported value")
	}
	m = formKey(m, tea.KeyLeft)
	if p.optionForm.Values()["format"] != "base64" {
		t.Fatal("Left did not select the previous format")
	}
	m = workflowKey(m, "esc")
	selectWorkspaceTool(t, &m, "password")
	selectWorkspaceTool(t, &m, "token")
	m = workflowKey(m, "o")
	if p.optionForm.selected != 1 || p.optionForm.Values()["bytes"] != "" {
		t.Fatal("action round trip lost the selected field or empty draft")
	}
	m = formKey(m, tea.KeyTab) // last -> first
	m = workflowKey(m, "64")
	m = workflowKey(m, "enter")
	if p.options["bytes"] != "64" || p.options["format"] != "base64" || p.outputErr {
		t.Fatal("restored fields did not submit together")
	}
}

func TestOptionFormMasksOnlySecretField(t *testing.T) {
	f := newOptionForm([]tool.Option{
		{Name: "key", Secret: true, Required: true},
		{Name: "label", Default: "public"},
	}, tool.Options{"key": `a secret "key"=value`})
	f.Focus()
	for _, editing := range []bool{true, false} {
		view := stripANSI(f.View(editing, 2))
		if strings.Contains(view, "a secret") || !strings.Contains(view, "public") || !strings.Contains(view, "key") {
			t.Fatalf("field masking hides labels or exposes a secret: %q", view)
		}
	}
}

func TestOptionFormRegisteredToolsFitWhileEditing(t *testing.T) {
	for _, current := range tools.NewRegistry().All() {
		if len(current.Options()) == 0 {
			continue
		}
		for _, size := range [][2]int{{80, 24}, {120, 40}, {160, 50}} {
			t.Run(fmt.Sprintf("%s/%dx%d", current.ID(), size[0], size[1]), func(t *testing.T) {
				m := workflowModel(current.Category())
				selectWorkspaceTool(t, &m, current.ID())
				m.width, m.height = size[0], size[1]
				m.resize()
				m.focus = focusWorkspace
				m = workflowKey(m, "o")
				for range len(current.Options()) {
					view := m.View()
					if lipgloss.Width(view) > m.width || lipgloss.Height(view) > m.height {
						t.Fatal("editing form exceeds terminal")
					}
					lines := strings.Split(stripANSI(view), "\n")
					if !strings.Contains(lines[len(lines)-1], "[Esc] normal") {
						t.Fatal("form footer is truncated")
					}
					if current.ID() != "json.get" {
						for _, spec := range current.Options() {
							if !strings.Contains(stripANSI(m.currentPage().View()), spec.Name) {
								t.Fatalf("parameter %s is hidden", spec.Name)
							}
						}
					}
					m = formKey(m, tea.KeyTab)
				}
			})
		}
	}
}

// Set individual values for page tests that exercise running or draft storage.
// Keyboard behavior is covered through root-model events above.
func setOptionValues(p *genericToolPage, values tool.Options) {
	for i, spec := range p.tool.Options() {
		if value, ok := values[spec.Name]; ok {
			p.optionForm.inputs[i].SetValue(value)
		}
	}
}
