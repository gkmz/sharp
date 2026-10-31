package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/pkg/tool"
)

func TestCategoryNormalJKMatchesArrowsAndSkipsHeaders(t *testing.T) {
	m := workflowModel(tool.CategoryConvert)
	m.width, m.height = 80, 24
	m.resize()
	for _, key := range []string{"j", "k"} {
		arrow := "down"
		if key == "k" {
			arrow = "up"
		}
		for range len(m.categories) + 2 {
			want := workflowKey(m, arrow)
			m = workflowKey(m, key)
			if m.selectedCat != want.selectedCat || m.currentCategory().Header || m.focus != focusCategories || m.search.Value() != "" {
				t.Fatalf("%s should navigate exactly like %s, skipping headers", key, arrow)
			}
			start, end := m.categoryWindow(m.height - 4)
			if m.selectedCat < start || m.selectedCat >= end {
				t.Fatal("selection scrolled offscreen")
			}
		}
	}
}

func TestCategorySearchRequiresExplicitEntry(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	selected := m.selectedCat
	m = workflowKey(m, "json")
	if m.focus != focusCategories || m.search.Value() != "" || m.selectedCat != selected {
		t.Fatal("typing started search without s or /")
	}
	m = workflowKey(m, "s")
	if m.focus != focusSearch || m.search.Value() != "" || !m.search.Focused() {
		t.Fatal("s should open an empty focused search without inserting itself")
	}
	m = workflowKey(m, "jkqs")
	if m.focus != focusSearch || m.search.Value() != "jkqs" {
		t.Fatal("search letters were treated as navigation or commands")
	}
	m = workflowKey(m, "esc")
	if m.focus != focusCategories || m.search.Value() != "" || m.search.Focused() {
		t.Fatal("Esc did not leave search")
	}
	m = workflowKey(m, "s")
	m = workflowKey(m, "minify")
	m = workflowKey(m, "enter")
	if m.currentTab().Tool.ID() != "json.minify" || m.focus != focusWorkspace || m.search.Value() != "" || len(m.currentCategory().Tabs) != 7 {
		t.Fatal("search did not open the exact tool and restore its actions")
	}
}

func TestCategorySearchStaysActiveWhenValueBecomesEmpty(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	m = workflowKey(m, "s")
	m = workflowKey(m, "j")
	m = workflowKey(m, "backspace")
	if m.focus != focusSearch || !m.search.Focused() || m.search.Value() != "" {
		t.Fatal("deleting the query unexpectedly left search mode")
	}
	m = workflowKey(m, "k")
	if m.search.Value() != "k" {
		t.Fatal("search typing became navigation after clearing")
	}
}

func TestCategorySearchKeyDoesNotInterceptOtherContexts(t *testing.T) {
	for _, id := range []string{"json.pretty", "regex.test"} {
		m := workflowModel(tool.CategoryJSON)
		selectWorkspaceTool(t, &m, id)
		m.focus = focusWorkspace
		key := "i"
		if id == "regex.test" {
			key = "o"
		}
		m = workflowKey(m, key)
		m = workflowKey(m, "sjk")
		if m.focus != focusWorkspace || !m.currentPage().IsEditing() || m.search.Value() != "" {
			t.Fatal("search shortcut intercepted editor text")
		}
		if id == "json.pretty" && m.currentPage().(*jsonToolPage).input.Value() != "sjk" {
			t.Fatal("input lost shortcut letters")
		}
		if id == "regex.test" && m.currentPage().(*genericToolPage).optionForm.Values()["pattern"] != "sjk" {
			t.Fatal("option field lost shortcut letters")
		}
	}
	for _, focus := range []focus{focusCategories, focusTabs, focusWorkspace} {
		m := workflowModel(tool.CategoryJSON)
		m.focus = focus
		m = workflowKey(m, "/")
		if m.focus != focusSearch || m.search.Value() != "" {
			t.Fatal("/ must still start search in Normal mode")
		}
	}
}

func TestCategoryNavigationHintsFitTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {160, 50}} {
		m := workflowModel(tool.CategoryJSON)
		m.width, m.height = size[0], size[1]
		m.resize()
		footer := stripANSI(m.footerLine())
		if !strings.Contains(footer, "[s]") || !strings.Contains(footer, "j/k") || strings.Contains(footer, "[type]") {
			t.Fatalf("stale hints: %q", footer)
		}
		for _, key := range []string{"s", "esc"} {
			m = workflowKey(m, key)
			view := m.View()
			if lipgloss.Width(view) > m.width || lipgloss.Height(view) > m.height {
				t.Fatal("navigation/search view exceeds terminal")
			}
			lines := strings.Split(stripANSI(view), "\n")
			if !strings.Contains(lines[len(lines)-1], "[Esc] clear") && !strings.Contains(lines[len(lines)-1], "[?] help") {
				t.Fatal("navigation/search footer is truncated")
			}
		}
	}
}

func TestCategorySearchShortcutKeepsSaveInOtherRegions(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, region := range []focus{focusTabs, focusWorkspace} {
		m := workflowModel(tool.CategoryJSON)
		p := m.currentPage()
		p.SetInput(`{"name":"sharp"}`)
		p.Run()
		m.focus = region
		m = workflowKey(m, "s")
		data, err := os.ReadFile("sharp-output.txt")
		if err != nil || strings.TrimSpace(string(data)) != p.OutputText() || m.focus != region {
			t.Fatalf("s did not save output in region %v: %v", region, err)
		}
		if err := os.Remove("sharp-output.txt"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCategoryNormalQQuitAndSearchQText(t *testing.T) {
	m := workflowModel(tool.CategoryJSON)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should quit in Normal Categories")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q did not produce Quit")
	}
	m = workflowKey(m, "s")
	m = workflowKey(m, "q")
	if m.search.Value() != "q" {
		t.Fatal("q should be text in Search")
	}
}
