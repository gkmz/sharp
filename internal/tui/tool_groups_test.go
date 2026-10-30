package tui

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/pkg/tool"
)

var consolidatedCases = []struct {
	category     tool.Category
	entry, title string
	ids, labels  []string
}{
	{tool.CategoryConvert, "Convert", "Convert",
		[]string{"json.to_yaml", "yaml.to_json", "csv.to_json", "headers.to_json"},
		[]string{"JSON→YAML", "YAML→JSON", "CSV→JSON", "Headers→JSON"}},
	{tool.CategoryEncode, "Base64", "Encode · Base64",
		[]string{"b64.encode", "b64.decode", "b64url.encode", "b64url.decode", "b64rawurl.encode", "b64rawurl.decode"},
		[]string{"Standard Encode", "Standard Decode", "URL Encode", "URL Decode", "Raw URL Encode", "Raw URL Decode"}},
	{tool.CategoryGenerator, "Generator", "Generator",
		[]string{"password", "token", "uuid.v4"},
		[]string{"Password", "Token", "UUID v4"}},
}

func TestConsolidatedCategoryEntriesAndActions(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	entries := 0
	seen := map[string]bool{}
	for _, c := range m.categories {
		if !c.Header {
			entries++
			for _, tab := range c.Tabs {
				if seen[tab.Tool.ID()] {
					t.Fatalf("duplicate action %s", tab.Tool.ID())
				}
				seen[tab.Tool.ID()] = true
			}
		}
	}
	if entries != 18 || len(seen) != len(m.registry.All()) {
		t.Errorf("got %d entries and %d tools, want 18 entries and all %d tools", entries, len(seen), len(m.registry.All()))
	}
	for _, tc := range consolidatedCases {
		t.Run(tc.title, func(t *testing.T) {
			selectWorkspaceTool(t, &m, tc.ids[0])
			cat := m.currentCategory()
			var ids, labels []string
			for _, tab := range cat.Tabs {
				ids = append(ids, tab.Tool.ID())
				labels = append(labels, actionLabel(tab.Tool))
			}
			if !reflect.DeepEqual(ids, tc.ids) || !reflect.DeepEqual(labels, tc.labels) {
				t.Fatalf("actions=%v labels=%v; want %v %v", ids, labels, tc.ids, tc.labels)
			}
			if !strings.HasPrefix(strings.TrimSpace(m.categoryLine(*cat, 30)), tc.entry) || m.tabsTitle() != tc.title {
				t.Fatalf("wrong entry/title: %q / %q", m.categoryLine(*cat, 30), m.tabsTitle())
			}
			if tc.category != tool.CategoryEncode {
				rows := 0
				for _, c := range m.categories {
					if c.Category == tc.category {
						rows++
					}
				}
				if rows != 1 {
					t.Fatalf("category has %d rows, want one selectable entry", rows)
				}
			}
		})
	}
	selectWorkspaceTool(t, &m, "json.pretty")
	if len(m.currentCategory().Tabs) != 7 {
		t.Fatal("conversion actions leaked into the JSON workbench")
	}
}

func TestConsolidatedConvertApplyReverseAndUndo(t *testing.T) {
	m := workflowModel(tool.CategoryConvert)
	selectWorkspaceTool(t, &m, "json.to_yaml")
	m.focus = focusWorkspace
	p := m.currentPage().(*genericToolPage)
	source := `{"name":"sharp"}`
	p.SetInput(source)
	m = workflowKey(m, "r")
	if !strings.Contains(p.OutputText(), "name: sharp") {
		t.Fatal("JSON→YAML did not run")
	}
	m = workflowKey(m, "p")
	if !strings.Contains(p.input.Value(), "name: sharp") || p.OutputText() != "" {
		t.Fatal("Apply did not move YAML into Input")
	}
	m = workflowKey(m, "]")
	if m.currentPage() != p || m.currentTab().Tool.ID() != "yaml.to_json" || !p.OutputReady() {
		t.Fatal("reverse conversion did not use the shared input")
	}
	var object map[string]string
	if err := json.Unmarshal([]byte(p.OutputText()), &object); err != nil || object["name"] != "sharp" {
		t.Fatal("reverse preview did not restore JSON")
	}
	previous := p.OutputText()
	m = workflowKey(m, "[") // YAML input is not valid for JSON→YAML.
	if p.OutputText() != previous || p.OutputReady() || !strings.Contains(stripANSI(p.View()), "YAML to JSON · stale") {
		t.Fatal("mismatched conversion must retain a stale result with its source")
	}
	before := p.input.Value()
	m = workflowKey(m, "p")
	if p.input.Value() != before {
		t.Fatal("stale result overwrote input")
	}
	m = workflowKey(m, "z")
	if p.input.Value() != source || !strings.Contains(p.OutputText(), "name: sharp") {
		t.Fatal("shared undo lost the original JSON")
	}
	selectWorkspaceTool(t, &m, "json.pretty")
	if m.currentPage() == p {
		t.Fatal("Convert and JSON should keep separate documents")
	}
}

func TestConsolidatedBase64VariantsKeepSourceAndRoundTrip(t *testing.T) {
	m := workflowModel(tool.CategoryEncode)
	selectWorkspaceTool(t, &m, "b64.encode")
	m.focus = focusWorkspace
	p := m.currentPage().(*genericToolPage)
	p.SetInput("??>?")
	m = workflowKey(m, "r")
	for _, tc := range []struct{ id, output string }{
		{"b64.encode", "Pz8+Pw=="}, {"b64url.encode", "Pz8-Pw=="}, {"b64rawurl.encode", "Pz8-Pw"},
	} {
		selectWorkflowAction(t, &m, tc.id)
		p.Preview()
		if m.currentPage() != p || p.input.Value() != "??>?" || p.OutputText() != tc.output {
			t.Fatalf("%s lost source or produced wrong encoding", tc.id)
		}
	}
	m = workflowKey(m, "p")
	m = workflowKey(m, "]")
	if m.currentPage() != p || m.currentTab().Tool.ID() != "b64rawurl.decode" || p.OutputText() != "??>?" {
		t.Fatal("raw URL round trip did not use shared input")
	}
}

func TestConsolidatedGeneratorDraftsAndResultsSurviveUUID(t *testing.T) {
	m := workflowModel(tool.CategoryGenerator)
	selectWorkspaceTool(t, &m, "token")
	m.focus = focusWorkspace
	p := m.currentPage().(*genericToolPage)
	m = workflowKey(m, "o")
	m = formKey(m, tea.KeyCtrlE)
	m = formKey(m, tea.KeyCtrlU)
	m = workflowKey(m, "64")
	m = workflowKey(m, "tab")
	m = workflowKey(m, "right")
	m = workflowKey(m, "enter")
	original := p.OutputText()
	if len(original) != 128 {
		t.Fatal("need a 64-byte hex token")
	}
	m = workflowKey(m, "]")
	if m.currentPage() != p || m.currentTab().Tool.ID() != "uuid.v4" || p.OutputText() != original || p.hasOptions || p.hasInput {
		t.Fatal("UUID navigation replaced the result or retained unavailable fields")
	}
	if !strings.Contains(stripANSI(p.View()), "Random Token") {
		t.Fatal("retained token lost its source label")
	}
	m = workflowKey(m, "o")
	if p.IsEditing() {
		t.Fatal("UUID opened an empty Options form")
	}
	m = workflowKey(m, "[")
	m = workflowKey(m, "o")
	if p.optionForm.selected != 1 || p.optionForm.Values()["bytes"] != "64" || p.optionForm.Values()["format"] != "hex" || p.OutputText() != original {
		t.Fatal("token draft, selected field or output was lost")
	}
	m = workflowKey(m, "esc")
	m = workflowKey(m, "[")
	if p.optionForm.Values()["length"] != "24" || p.OutputText() != original {
		t.Fatal("Password should have its own options without regenerating")
	}
}

func TestConsolidatedSearchAndRecentRestoreCompleteGroups(t *testing.T) {
	for _, tc := range consolidatedCases {
		for _, id := range tc.ids {
			t.Run(id, func(t *testing.T) {
				m := workflowModel(tool.CategoryJSON)
				m = workflowKey(m, id)
				m = workflowKey(m, "enter")
				if m.currentTab().Tool.ID() != id || m.focus != focusWorkspace || len(m.currentCategory().Tabs) != len(tc.ids) {
					t.Fatal("search did not restore exact action and complete group")
				}
				if len(m.recent.ids) != 0 {
					t.Fatal("search opening was recorded as use")
				}
				selectWorkspaceTool(t, &m, "json.pretty")
				m.recent.ids = []string{id}
				m = recentKey(m)
				m = workflowKey(m, "enter")
				if m.currentTab().Tool.ID() != id || len(m.currentCategory().Tabs) != len(tc.ids) || m.currentPage().OutputText() != "" {
					t.Fatal("history did not restore the complete group without execution")
				}
			})
		}
	}
	for _, query := range []string{"Base64 URL", "Raw Base64 URL", "random", "CSV to JSON", "HTTP Headers"} {
		m := workflowModel(tool.CategoryJSON)
		m = workflowKey(m, query)
		if m.currentTab() == nil {
			t.Errorf("old search alias %q no longer finds a tool", query)
		}
	}
}

func TestConsolidatedActionLabelsAndFramesFit(t *testing.T) {
	for _, tc := range consolidatedCases {
		for _, size := range [][2]int{{80, 24}, {120, 40}, {160, 50}} {
			t.Run(fmt.Sprintf("%s/%dx%d", tc.title, size[0], size[1]), func(t *testing.T) {
				m := workflowModel(tc.category)
				selectWorkspaceTool(t, &m, tc.ids[0])
				m.width, m.height = size[0], size[1]
				m.resize()
				m.focus = focusTabs
				for range tc.ids {
					view := m.View()
					if lipgloss.Width(view) > m.width || lipgloss.Height(view) > m.height {
						t.Fatal("merged frame exceeds terminal")
					}
					for _, label := range tc.labels {
						if !strings.Contains(stripANSI(view), label) {
							t.Fatalf("action %q is hidden", label)
						}
					}
					lines := strings.Split(stripANSI(view), "\n")
					if !strings.Contains(lines[len(lines)-1], "[?] help") {
						t.Fatal("footer is truncated")
					}
					m = workflowKey(m, "right")
				}
			})
		}
	}
}
