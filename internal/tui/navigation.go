package tui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gkmz/sharp/pkg/tool"
)

// 搜索临时占用 Categories 的一行，不引入额外导航区域。
func (m model) startCategorySearch() (tea.Model, tea.Cmd) {
	m.focus = focusSearch
	m.searchFrom = focusCategories
	cmd := m.search.Focus()
	m.applySearch()
	return m, cmd
}

func (m model) updateCategorySearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.applySearch()
		m.search.Blur()
		single := m.focusSingleSearchResult()
		if m.currentTab() == nil {
			m.focus = focusCategories
			return m, nil
		}
		// 搜索负责定位；打开后恢复整组动作，避免筛选把 [ / ] 导航锁住。
		m.setSelectedTab(m.selectedTab)
		m.search.SetValue("")
		m.applySearch()
		if single {
			m.focus = focusWorkspace
		} else {
			m.openSelection()
		}
		if p := m.currentPage(); p != nil {
			p.Preview()
			p.takeToolUse() // Opening a search result is navigation only.
			m.status = p.Status()
		}
		return m, nil
	case "down":
		m.moveCategory(1)
		return m, nil
	case "up":
		m.moveCategory(-1)
		return m, nil
	case "ctrl+x":
		m.clearCategorySearch()
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.applySearch()
	return m, cmd
}

// Esc consumes exactly one layer; page editors own their preview cleanup.
func (m model) leaveContext() model {
	switch {
	case m.showRecent:
		m.showRecent = false
	case m.showHelp:
		m.showHelp = false
		m.help.Blur()
	case m.focus == focusSearch || (m.focus == focusCategories && m.search.Value() != ""):
		m.clearCategorySearch()
	case m.focus == focusWorkspace:
		if p := m.currentPage(); p != nil && p.IsEditing() {
			updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
			m.setCurrentPage(updated)
			m.status = updated.Status()
		} else {
			m.focus = focusTabs
		}
	case m.focus == focusTabs:
		m.focus = focusCategories
	}
	return m
}

func (m *model) clearCategorySearch() {
	m.search.SetValue("")
	m.search.Blur()
	m.focus = focusCategories
	m.applySearch()
	m.status = "Filter cleared"
}

func (m *model) openSelection() {
	cat := m.currentCategory()
	if cat == nil {
		m.focus = focusCategories
		return
	}
	if len(cat.Tabs) == 1 {
		m.focus = focusWorkspace
	} else {
		m.focus = focusTabs
	}
}

var jsonActionOrder = []string{"json.pretty", "json.minify", "json.sort", "json.get", "json.escape", "json.unescape", "json.validate"}

func orderToolActions(tabs []tabItem) {
	rank := func(t tool.Tool) int {
		if group, index := consolidatedGroup(t); group != nil {
			return index
		}
		if t.Category() == tool.CategoryJSON {
			for i, id := range jsonActionOrder {
				if t.ID() == id {
					return i
				}
			}
		}
		return len(jsonActionOrder)
	}
	sort.SliceStable(tabs, func(i, j int) bool { return rank(tabs[i].Tool) < rank(tabs[j].Tool) })
}

func actionLabel(t tool.Tool) string {
	if group, index := consolidatedGroup(t); group != nil {
		return group.actions[index].label
	}
	if label, ok := commonActionLabels[t.ID()]; ok {
		return label
	}
	if t.Category() == tool.CategoryCrypto {
		_, name, _ := strings.Cut(t.ID(), ".")
		return strings.ToUpper(name)
	}
	if t.Category() != tool.CategoryJSON {
		return t.Name()
	}
	switch t.ID() {
	case "json.pretty":
		return "Pretty"
	case "json.minify":
		return "Minify"
	case "json.sort":
		return "Sort"
	case "json.get":
		return "Query"
	case "json.escape":
		return "Escape"
	case "json.unescape":
		return "Unescape"
	case "json.validate":
		return "Check"
	}
	return strings.TrimPrefix(t.Name(), "JSON ")
}

func (m model) categoryHasHeader(category tool.Category) bool {
	for _, item := range m.categories {
		if item.Category == category && item.Header {
			return true
		}
	}
	return false
}

// 分类是工作台身份的一部分，避免 Encode/url 与 Network/url 共用输入。
func workspaceKey(t tool.Tool) string {
	if t.Category() == tool.CategoryJSON {
		return "json.workspace"
	}
	if _, special := specialPageFactories[t.ID()]; special {
		return t.ID()
	}
	return string(t.Category()) + ":" + toolSubcategory(t)
}

// 预览仅允许已知本地转换；查询网络、读取时钟与随机生成需要显式运行。
func canPreview(t tool.Tool) bool {
	if !toolUsesInput(t) {
		return false
	}
	prefix, _, _ := strings.Cut(t.ID(), ".")
	switch prefix {
	case "json", "yaml", "csv", "headers", "b64", "b64url", "b64rawurl", "url", "hex", "html", "unicode", "hash", "hmac", "text", "regex", "query", "cidr", "jwt":
		return true
	case "time":
		return t.ID() == "time.from" || t.ID() == "time.parse"
	}
	return false
}

var commonActionLabels = map[string]string{
	"url.encode": "Encode", "url.decode": "Decode", "url.parse": "Parse",
	"hex.encode": "Encode", "hex.decode": "Decode",
	"html.escape": "Escape", "html.unescape": "Unescape",
	"unicode.escape": "Escape", "unicode.unescape": "Unescape",
	"text.sort": "Sort", "text.uniq": "Dedupe", "text.count": "Count",
	"regex.test": "Match", "regex.replace": "Replace",
	"time.now": "Now", "time.from": "From timestamp", "time.parse": "To timestamp",
	"query.parse": "Parse", "dns.lookup": "Lookup", "cidr.parse": "Parse",
	"jwt.decode": "Decode",
}

func workspaceGroupTitle(t tool.Tool) string {
	category := string(t.Category())
	subgroup := toolSubcategory(t)
	if t.Category() == tool.CategoryJSON || strings.EqualFold(category, subgroup) {
		return category
	}
	return category + " · " + subgroupDisplayName(subgroup)
}

// Display names are shared by navigation and the Actions title. Internal IDs
// remain unchanged for tool lookup, grouping, and search aliases.
func subgroupDisplayName(subgroup string) string {
	names := map[string]string{"b64": "Base64", "b64url": "Base64 URL", "b64rawurl": "Raw Base64 URL", "hex": "Hex", "html": "HTML", "unicode": "Unicode", "url": "URL", "hash": "Hash", "hmac": "HMAC", "regex": "Regex", "query": "Query", "dns": "DNS", "cidr": "CIDR", "csv": "CSV", "yaml": "YAML", "json": "JSON", "headers": "HTTP Headers", "jwt": "JWT", "uuid": "UUID", "random": "Random", "text": "Text", "time": "Time", "password": "Password", "token": "Token"}
	if name, ok := names[subgroup]; ok {
		return name
	}
	return subgroup
}
