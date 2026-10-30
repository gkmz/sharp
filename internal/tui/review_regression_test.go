package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/pkg/tool"
)

func TestReviewJSONAcrossSubgroups(t *testing.T) {
	r := tool.NewRegistry()
	for _, id := range []string{"json.pretty", "custom.inspect", "custom.other"} {
		id := id
		r.MustRegister(tool.SimpleTool{IDValue: id, NameValue: id, CategoryValue: tool.CategoryJSON, RunFunc: func(context.Context, tool.Input, tool.Options) (tool.Output, error) {
			return tool.Output{Text: id}, nil
		}})
	}
	m := newModel(r)
	for _, id := range []string{"json.pretty", "custom.inspect", "custom.other"} {
		selectWorkspaceTool(t, &m, id)
		p := m.currentPage()
		p.SetInput(`{}`)
		p.Run()
		if p.OutputText() != id {
			t.Fatalf("%s: %q", id, p.OutputText())
		}
	}
}

func TestReviewJSONValidationUsesRegisteredTool(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	selectWorkspaceTool(t, &m, "json.pretty")
	p := m.currentPage().(*jsonToolPage)
	p.SetInput(`{"x":1}`)
	p.Run()
	retained := p.OutputText()
	called := false
	p.tools["json.validate"] = tool.SimpleTool{IDValue: "json.validate", NameValue: "Check", CategoryValue: tool.CategoryJSON, RunFunc: func(context.Context, tool.Input, tool.Options) (tool.Output, error) {
		called = true
		return tool.Output{}, fmt.Errorf("custom rejection")
	}}
	p.SelectTool(p.tools["json.validate"])
	p.takeToolUse()
	p.Run()
	if !called || !strings.Contains(stripANSI(p.Status()), "custom rejection") || p.OutputText() != retained || p.takeToolUse() != "" {
		t.Fatalf("validation bypassed tool: called=%v status=%s output=%s", called, p.Status(), p.OutputText())
	}
}

func TestReviewEscapeUnchangedEditorDoesNotRecordUse(t *testing.T) {
	for _, id := range []string{"json.pretty", "json.validate", "text.upper"} {
		m := workflowModel(tool.CategoryJSON)
		selectWorkspaceTool(t, &m, id)
		m.focus = focusWorkspace
		m.currentPage().SetInput(`{"x":1}`)
		m = workflowKey(m, "r")
		m.recent.ids = []string{"hash.sha256", id}
		m = workflowKey(m, "i")
		m = workflowKey(m, "esc")
		if m.recent.ids[0] != "hash.sha256" {
			t.Fatalf("%s reordered history on unchanged Escape: %v", id, m.recent.ids)
		}
	}
}

func TestReviewPreviewPreservesBuiltInEligibility(t *testing.T) {
	for _, current := range tools.NewRegistry().All() {
		want := toolUsesInput(current) && current.ID() != "dns.lookup"
		if canPreview(current) != want {
			t.Fatalf("changed built-in preview eligibility: %s", current.ID())
		}
	}
}

func TestReviewShortTerminalRetainsStatusAndFooter(t *testing.T) {
	for _, height := range []int{2, 4, 8, 10, 11, 12} {
		m := workflowModel(tool.CategoryJSON)
		m.width, m.height = 80, height
		m.status = "unique notification"
		m.resize()
		lines := strings.Split(stripANSI(m.View()), "\n")
		if len(lines) != height || !strings.Contains(lines[height-2], "unique notification") || !strings.Contains(lines[height-1], "[?] help") {
			t.Fatalf("height=%d lost status/footer: %q", height, lines)
		}
	}
}

func TestReviewUnknownToolsNeverAutoPreview(t *testing.T) {
	for _, category := range []tool.Category{tool.CategoryNetwork, tool.CategoryJSON} {
		calls := 0
		current := tool.SimpleTool{IDValue: "url.fetch", NameValue: "Fetch", CategoryValue: category, RunFunc: func(context.Context, tool.Input, tool.Options) (tool.Output, error) {
			calls++
			return tool.Output{Text: "network result"}, nil
		}}
		r := tool.NewRegistry()
		r.MustRegister(current)
		m := newModel(r)
		p := m.currentPage()
		p.SetInput("https://example.com")
		p.Preview()
		p.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if calls != 0 {
			t.Fatalf("category=%s automatically ran unknown tool %d times", category, calls)
		}
		p.Run()
		if calls != 1 {
			t.Fatalf("explicit Run calls=%d", calls)
		}
	}
}
