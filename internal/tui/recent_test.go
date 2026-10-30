package tui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/pkg/tool"
)

func TestRecentOrderDeduplicatesAndLimitsActions(t *testing.T) {
	var recent recentTools
	for _, id := range []string{"json.pretty", "json.get", "uuid.v4", "text.upper", "b64.decode", "hash.sha256", "json.get"} {
		recent.record(id)
	}
	want := []string{"json.get", "hash.sha256", "b64.decode", "text.upper", "uuid.v4"}
	if !reflect.DeepEqual(recent.ids, want) {
		t.Fatalf("recent = %v, want %v", recent.ids, want)
	}
	if recent.record("json.get") {
		t.Fatal("repeating the first item unnecessarily changed history")
	}
}

func recentKey(m model) model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	return updated.(model)
}

func TestRecentRecordsSuccessfulUseNotBrowsingOrFailure(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m = workflowKey(m, "down")
	m = workflowKey(m, "json")
	m = workflowKey(m, "esc")
	if len(m.recent.ids) != 0 {
		t.Fatal("navigation populated history")
	}
	selectWorkspaceTool(t, &m, "json.pretty")
	m.focus = focusWorkspace
	m.currentPage().SetInput("invalid JSON")
	m = workflowKey(m, "r")
	if len(m.recent.ids) != 0 {
		t.Fatal("failed execution populated history")
	}
	m.currentPage().SetInput(`{"name":"sharp"}`)
	m = workflowKey(m, "r")
	if !reflect.DeepEqual(m.recent.ids, []string{"json.pretty"}) {
		t.Fatalf("execution history = %v", m.recent.ids)
	}
	selectWorkspaceTool(t, &m, "json.validate")
	m = workflowKey(m, "r")
	if m.recent.ids[0] != "json.validate" {
		t.Fatal("status-only validation not recorded")
	}
	selectWorkspaceTool(t, &m, "b64.encode")
	m.readClipboard = func() (string, error) { return "hello", nil }
	m = workflowKey(m, "v")
	if m.recent.ids[0] != "b64.encode" {
		t.Fatal("paste preview not recorded")
	}
	selectWorkspaceTool(t, &m, "text.upper")
	p := m.currentPage()
	p.SetInput("hello")
	updated, _ := m.Update(workspacePreviewMsg{page: p, revision: p.previewVersion()})
	m = updated.(model)
	if m.recent.ids[0] != "text.upper" {
		t.Fatal("delayed preview not recorded")
	}
	p.SetInput("")
	updated, _ = m.Update(workspacePreviewMsg{page: p, revision: p.previewVersion()})
	m = updated.(model)
	if len(m.recent.ids) != 4 {
		t.Fatalf("empty preview changed history: %v", m.recent.ids)
	}
}

func TestRecentPickerRestoresExactActionWithoutExecution(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m.recent.ids = []string{"uuid.v4", "json.get", "b64.decode", "dns.lookup"}
	for _, index := range []int{0, 1, 2, 3} {
		m.focus = focusCategories
		m = recentKey(m)
		if !m.showRecent {
			t.Fatal("Ctrl+R did not open picker")
		}
		for i := 0; i < index; i++ {
			m = workflowKey(m, "down")
		}
		m = workflowKey(m, "enter")
		if m.showRecent || m.focus != focusWorkspace || m.currentTab().Tool.ID() != m.recent.ids[index] {
			t.Fatalf("wrong destination: show=%v focus=%v tool=%s", m.showRecent, m.focus, m.currentTab().Tool.ID())
		}
		if m.currentPage().OutputText() != "" {
			t.Fatal("opening recent item ran a tool")
		}
		if len(m.categories) != len(m.allCategories) {
			t.Fatal("full groups not restored")
		}
		if m.currentTab().Tool.ID() == "json.get" {
			if !strings.Contains(stripANSI(m.currentPage().View()), " Path ") || len(m.currentCategory().Tabs) < 2 {
				t.Fatal("Query group/path not restored")
			}
		}
	}
}

func TestRecentPickerNeverRunsManualToolsWhileNavigating(t *testing.T) {
	for _, id := range []string{"uuid.v4", "dns.lookup"} {
		registry := tool.NewRegistry()
		calls := 0
		registry.MustRegister(tool.SimpleTool{
			IDValue: id, NameValue: id, CategoryValue: tool.CategoryNetwork,
			RunFunc: func(context.Context, tool.Input, tool.Options) (tool.Output, error) {
				calls++
				return tool.Output{Text: "executed"}, nil
			},
		})
		m := newModel(registry)
		m.width, m.height = 80, 24
		m.resize()
		m.recent.ids = []string{id}
		m = recentKey(m)
		m.View()
		m = workflowKey(m, "enter")
		m.View()
		if calls != 0 {
			t.Fatalf("opening %s made %d calls", id, calls)
		}
		m = workflowKey(m, "r")
		if calls != 1 {
			t.Fatal("manual execution no longer works")
		}
	}
}

func TestRecentPickerContextAndFrozenOrder(t *testing.T) {
	for _, area := range []focus{focusCategories, focusTabs, focusWorkspace} {
		m := workflowModel(tool.CategoryJSON)
		m.focus = area
		m.recent.ids = []string{"uuid.v4", "json.pretty"}
		before := m.currentTab().Tool.ID()
		m = recentKey(m)
		if !m.showRecent {
			t.Fatalf("picker not open from area %v", area)
		}
		p := m.currentPage()
		p.SetInput(`{"a":1}`)
		updated, _ := m.Update(workspacePreviewMsg{page: p, revision: p.previewVersion()})
		m = updated.(model)
		if !reflect.DeepEqual(m.recent.ids, []string{"json.pretty", "uuid.v4"}) {
			t.Fatal("background preview did not update actual history")
		}
		if !reflect.DeepEqual(m.recentItems, []string{"uuid.v4", "json.pretty"}) {
			t.Fatal("open picker reordered")
		}
		m = workflowKey(m, "]")
		if m.currentTab().Tool.ID() != before {
			t.Fatal("picker leaked navigation into background")
		}
		m = workflowKey(m, "esc")
		if m.showRecent || m.focus != area || m.currentTab().Tool.ID() != before {
			t.Fatal("Esc did not preserve original context")
		}
	}
	for _, context := range []string{"input", "options", "search", "help"} {
		m := workflowModel(tool.CategoryJSON)
		selectWorkspaceTool(t, &m, "json.get")
		m.focus = focusWorkspace
		switch context {
		case "input":
			m = workflowKey(m, "i")
		case "options":
			m = workflowKey(m, "o")
		case "search":
			m = workflowKey(m, "/")
		case "help":
			m = workflowKey(m, "?")
		}
		m = recentKey(m)
		if m.showRecent {
			t.Fatalf("Ctrl+R intercepted %s", context)
		}
	}
}

func TestRecentTracksQueryOptionsAndEditingPreview(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m.focus = focusWorkspace
	selectWorkspaceTool(t, &m, "json.get")
	p := m.currentPage().(*jsonToolPage)
	p.SetInput(`{"name":"sharp"}`)
	m = workflowKey(m, "o")
	m = workflowKey(m, "name")
	m = workflowKey(m, "enter")
	if !reflect.DeepEqual(m.recent.ids, []string{"json.get"}) || p.OutputText() != "sharp" {
		t.Fatalf("Query submit not recorded: %v, output=%q", m.recent.ids, p.OutputText())
	}
	selectWorkspaceTool(t, &m, "text.upper")
	m = workflowKey(m, "i")
	m = workflowKey(m, "hello")
	page := m.currentPage()
	updated, _ := m.Update(workspacePreviewMsg{page: page, revision: page.previewVersion()})
	m = updated.(model)
	if m.recent.ids[0] != "text.upper" || page.OutputText() != "HELLO" {
		t.Fatal("editing preview was not recorded")
	}
	// Replaying the timer must not cause another operation after it settled.
	oldOrder := append([]string(nil), m.recent.ids...)
	updated, _ = m.Update(workspacePreviewMsg{page: page, revision: 0})
	m = updated.(model)
	if !reflect.DeepEqual(m.recent.ids, oldOrder) {
		t.Fatal("obsolete timer changed history")
	}
}

func TestRecentBrowsingActionsDoesNotDisplaceUsedTools(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m.focus = focusWorkspace
	m.currentPage().SetInput(`{"a":1}`)
	m = workflowKey(m, "r")
	want := append([]string(nil), m.recent.ids...)
	m = workflowKey(m, "]")
	if m.currentPage().OutputText() == "" {
		t.Fatal("navigation preview disappeared")
	}
	if !reflect.DeepEqual(m.recent.ids, want) {
		t.Fatalf("browsing displaced used tools: %v", m.recent.ids)
	}
	m = workflowKey(m, "/")
	m = workflowKey(m, "json sort")
	m = workflowKey(m, "enter")
	if !reflect.DeepEqual(m.recent.ids, want) {
		t.Fatal("search/open preview recorded use")
	}
}

func TestRecentPickerFrameAndEmptyState(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		m := workflowModel(tool.CategoryJSON)
		m.width, m.height = size[0], size[1]
		m.resize()
		m = recentKey(m)
		if !strings.Contains(stripANSI(m.View()), "No recent tools") {
			t.Fatal("missing empty state")
		}
		m = recentKey(m)
		m.recent.ids = []string{"json.pretty", "b64.decode", "uuid.v4", "json.get", "hash.sha256"}
		m = recentKey(m)
		m = workflowKey(m, "down")
		m = workflowKey(m, "down")
		view := m.View()
		if lipgloss.Width(view) > size[0] || lipgloss.Height(view) > size[1] {
			t.Fatalf("picker exceeds frame %v", size)
		}
		for _, want := range []string{"Recent tools", "Pretty", "Decode", "UUID v4", "Query", "SHA256"} {
			if !strings.Contains(stripANSI(view), want) {
				t.Errorf("missing %q", want)
			}
		}
	}
}

func TestRecentRootPersistsAndSurvivesStorageFailure(t *testing.T) {
	m := workflowModel(tool.CategoryGenerator)
	m.recent.path = filepath.Join(t.TempDir(), "sharp", "recent.json")
	selectWorkspaceTool(t, &m, "uuid.v4")
	m.focus = focusWorkspace
	m = workflowKey(m, "r")
	loaded, err := loadRecentTools(m.recent.path, m.registry)
	if err != nil || !reflect.DeepEqual(loaded.ids, []string{"uuid.v4"}) {
		t.Fatalf("root did not persist: %v %v", loaded.ids, err)
	}
	parent := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(parent, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	m.recent.path = filepath.Join(parent, "recent.json")
	selectWorkspaceTool(t, &m, "json.pretty")
	m.currentPage().SetInput(`{"a":1}`)
	m = workflowKey(m, "r")
	if m.currentPage().OutputText() == "" || m.recent.ids[0] != "json.pretty" || m.recent.warning == "" {
		t.Fatal("storage failure broke execution or lost in-memory history")
	}
}

func TestRecentStorageRoundTripContainsOnlyIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sharp", "recent-tools.json")
	recent := recentTools{path: path}
	recent.record("json.pretty")
	recent.record("uuid.v4")
	if err := recent.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadRecentTools(path, tools.NewRegistry())
	if err != nil || !reflect.DeepEqual(loaded.ids, recent.ids) {
		t.Fatalf("restart history = %v, err=%v", loaded.ids, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || len(fields) != 1 || fields["tools"] == nil {
		t.Fatalf("unexpected history fields: %s, err=%v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("history permissions: %v, err=%v", info, err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files left after save: %v, err=%v", entries, err)
	}
	recent.record("json.get")
	if err := recent.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err = loadRecentTools(path, tools.NewRegistry())
	if err != nil || loaded.ids[0] != "json.get" {
		t.Fatalf("replacement failed: %v, err=%v", loaded.ids, err)
	}
}

func TestRecentStorageFiltersInvalidEntries(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		want       []string
		wantErr    bool
	}{
		{"missing", "", nil, false},
		{"corrupt", `{"tools":`, nil, true},
		{"unknown and duplicate", `{"tools":["deleted.tool","json.pretty","json.pretty","uuid.v4","json.get","text.upper","b64.decode","hash.sha256"]}`, []string{"json.pretty", "uuid.v4", "json.get", "text.upper", "b64.decode"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "recent.json")
			if tc.data != "" {
				if err := os.WriteFile(path, []byte(tc.data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			recent, err := loadRecentTools(path, tools.NewRegistry())
			if (err != nil) != tc.wantErr || !reflect.DeepEqual(recent.ids, tc.want) {
				t.Fatalf("history=%v, err=%v; want %v, error=%v", recent.ids, err, tc.want, tc.wantErr)
			}
		})
	}
}
