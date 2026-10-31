package tui

import (
	"encoding/base64"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/pkg/tool"
)

func editURLTokenOptions(m model, values tool.Options) model {
	m = workflowKey(m, "o")
	p := m.currentPage().(*genericToolPage)
	for p.optionForm.selected != 0 {
		m = formKey(m, tea.KeyTab)
	}
	for i, spec := range p.tool.Options() {
		if i > 0 {
			m = formKey(m, tea.KeyTab)
		}
		value, ok := values[spec.Name]
		if !ok {
			continue
		}
		if len(spec.Choices) > 0 {
			for attempts := 0; p.optionForm.Values()[spec.Name] != value && attempts < len(spec.Choices); attempts++ {
				m = formKey(m, tea.KeyRight)
			}
		} else {
			m = formKey(m, tea.KeyCtrlE)
			m = formKey(m, tea.KeyCtrlU)
			m = workflowKey(m, value)
		}
	}
	return m
}

func TestURLWorkspaceModePreviewAndDrafts(t *testing.T) {
	m := workflowModel(tool.CategoryEncode)
	selectWorkspaceTool(t, &m, "url.encode")
	m.focus = focusWorkspace
	p := m.currentPage().(*genericToolPage)
	if p.optionForm.Values()["mode"] != "query" {
		t.Fatal("URL default mode missing from Options")
	}
	p.SetInput("^@#$ a+b")
	m = editURLTokenOptions(m, tool.Options{"mode": "component"})
	updated, _ := m.Update(workspacePreviewMsg{page: p, revision: p.previewVersion()})
	m = updated.(model)
	if p.OutputText() != "%5E%40%23%24%20a%2Bb" {
		t.Fatalf("component preview = %q", p.OutputText())
	}
	m = workflowKey(m, "esc")
	selectWorkspaceTool(t, &m, "url.decode")
	if p.optionForm.Values()["mode"] != "query" {
		t.Fatal("Decode draft should start with query mode")
	}
	p.SetInput("a+b%2Bc")
	m = editURLTokenOptions(m, tool.Options{"mode": "component"})
	m = workflowKey(m, "enter")
	if p.OutputText() != "a+b+c" || p.IsEditing() {
		t.Fatal("Decode mode submit did not preserve plus or leave Insert")
	}
	selectWorkspaceTool(t, &m, "url.encode")
	if p.optionForm.Values()["mode"] != "component" {
		t.Fatal("Encode mode draft lost")
	}
	if !reflect.DeepEqual(m.recent.ids, []string{"url.decode", "url.encode"}) {
		t.Fatalf("URL usage missing from recent tools: %v", m.recent.ids)
	}
}

func TestTokenWorkspaceFormatsRemainManual(t *testing.T) {
	m := workflowModel(tool.CategoryGenerator)
	selectWorkspaceTool(t, &m, "token")
	m.focus = focusWorkspace
	p := m.currentPage().(*genericToolPage)
	if !reflect.DeepEqual(p.optionForm.Values(), tool.Options{"bytes": "32", "format": "base64url"}) {
		t.Fatal("default token options changed")
	}
	m = editURLTokenOptions(m, tool.Options{"bytes": "32", "format": "hex"})
	if p.OutputText() != "" {
		t.Fatal("parameter editing generated a token")
	}
	m = workflowKey(m, "esc")
	if p.OutputText() != "" {
		t.Fatal("Esc generated a token")
	}
	m = workflowKey(m, "r")
	value := p.OutputText()
	if raw, err := hex.DecodeString(value); err != nil || len(raw) != 32 {
		t.Fatalf("explicit Hex token: bytes=%d error=%v options=%q parsed=%v status=%q", len(raw), err, p.optionForm.Values(), p.options, stripANSI(p.Status()))
	}
	m = editURLTokenOptions(m, tool.Options{"bytes": "64", "format": "base64"})
	if p.OutputText() != value {
		t.Fatal("parameter edits replaced the previous token")
	}
	m = workflowKey(m, "enter")
	value = p.OutputText()
	if raw, err := base64.StdEncoding.Strict().DecodeString(value); err != nil || len(raw) != 64 {
		t.Fatal("parameter submit did not produce 64-byte Base64")
	}
	selectWorkspaceTool(t, &m, "password")
	selectWorkspaceTool(t, &m, "token")
	if !reflect.DeepEqual(p.optionForm.Values(), tool.Options{"bytes": "64", "format": "base64"}) || p.OutputText() != value {
		t.Fatal("action navigation changed token draft/result")
	}
	m = recentKey(m)
	m = workflowKey(m, "enter")
	if m.currentTab().Tool.ID() != "token" || p.OutputText() != value {
		t.Fatal("opening history regenerated token")
	}
	if canPreview(p.tool) || p.hasInput {
		t.Fatal("Token became an auto-preview/input tool")
	}
	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		m.width, m.height = size[0], size[1]
		m.resize()
		view := m.View()
		if lipgloss.Width(view) > size[0] || lipgloss.Height(view) > size[1] || !strings.Contains(stripANSI(view), "Options") {
			t.Fatal("token options exceed frame")
		}
	}
}
