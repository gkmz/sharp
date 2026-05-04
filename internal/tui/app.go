package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hank/sharp/internal/clipboard"
	"github.com/hank/sharp/pkg/tool"
)

type focus int

const (
	focusCategories focus = iota
	focusTabs
	focusWorkspace
	focusSearch
)

type categoryGroup struct {
	Category tool.Category
	Tabs     []tabItem
}

type tabItem struct {
	Tool tool.Tool
}

type helpCommand struct {
	Key         string
	Command     string
	Description string
}

type helpSection struct {
	Title    string
	Commands []helpCommand
}

type toolPage interface {
	SetSize(width, height int)
	Update(msg tea.Msg) (toolPage, tea.Cmd)
	View() string
	FocusInput()
	FocusOptions()
	FocusOutput()
	FocusNext()
	FocusPrev()
	Run()
	ClearInput()
	ClearOutput()
	SetInput(string)
	OutputText() string
	Status() string
	HelpText() string
	HelpCommands() []helpCommand
	FooterHints() string
	IsEditing() bool
	HandleKey(string) bool
}

type model struct {
	registry *tool.Registry

	allCategories []categoryGroup
	categories    []categoryGroup
	selectedCat   int
	selectedTab   int
	lastTabByCat  map[tool.Category]int

	focus      focus
	searchFrom focus

	search textinput.Model
	help   textinput.Model
	pages  map[string]toolPage

	showHelp bool

	width  int
	height int
	status string
}

var (
	appTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	panelTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	activeLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("31"))
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	successStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	selectedRowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("24"))
	sectionRuleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	footerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func Run(registry *tool.Registry) error {
	p := tea.NewProgram(newModel(registry), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newModel(registry *tool.Registry) model {
	search := textinput.New()
	search.Placeholder = "Search categories or tools"
	search.Blur()
	help := textinput.New()
	help.Placeholder = "Filter commands"
	help.Blur()

	m := model{
		registry:     registry,
		lastTabByCat: make(map[tool.Category]int),
		pages:        make(map[string]toolPage),
		search:       search,
		help:         help,
		status:       "1 search | 2 categories | 3 tools | 4 workspace | q quit",
	}
	m.allCategories = buildCategories(registry.All())
	m.categories = m.allCategories
	m.syncSelection()
	return m
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.showHelp {
			switch msg.String() {
			case "esc", "?":
				m.showHelp = false
				m.help.Blur()
				return m, nil
			case "ctrl+u":
				m.help.SetValue("")
				return m, nil
			}
			var cmd tea.Cmd
			m.help, cmd = m.help.Update(msg)
			return m, cmd
		}

		if m.focus == focusSearch {
			if msg.String() == "esc" || msg.String() == "enter" {
				m.focus = m.searchFrom
				m.search.Blur()
				m.applySearch()
				return m, nil
			}
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			m.applySearch()
			return m, cmd
		}

		if p := m.currentPage(); m.focus == focusWorkspace && p != nil && p.IsEditing() {
			switch msg.String() {
			case "tab":
				p.FocusNext()
				return m, nil
			case "shift+tab":
				p.FocusPrev()
				return m, nil
			default:
				var cmd tea.Cmd
				updated, cmd := p.Update(msg)
				m.setCurrentPage(updated)
				m.status = updated.Status()
				return m, cmd
			}
		}

		if msg.String() == "q" {
			return m, tea.Quit
		}

		switch msg.String() {
		case "1":
			m.searchFrom = m.focus
			m.focus = focusSearch
			m.search.Focus()
			return m, nil
		case "2":
			m.focus = focusCategories
			return m, nil
		case "3":
			m.focus = focusTabs
			return m, nil
		case "4":
			m.focus = focusWorkspace
			return m, nil
		case "/":
			m.searchFrom = m.focus
			m.focus = focusSearch
			m.search.Focus()
			return m, nil
		}

		switch m.focus {
		case focusCategories:
			switch msg.String() {
			case "j", "down":
				m.moveCategory(1)
			case "k", "up":
				m.moveCategory(-1)
			case "tab", "enter":
				m.focus = focusTabs
			}
		case focusTabs:
			switch msg.String() {
			case "l", "right", "]":
				if !m.currentCategoryIsJSON() {
					m.moveTab(1)
				}
			case "h", "left", "[":
				if !m.currentCategoryIsJSON() {
					m.moveTab(-1)
				}
			case "tab":
				m.focus = focusWorkspace
			case "enter":
				if p := m.currentPage(); p != nil {
					m.focus = focusWorkspace
					p.FocusInput()
				}
			case "i":
				if p := m.currentPage(); p != nil {
					m.focus = focusWorkspace
					p.FocusInput()
				}
			case "o":
				if p := m.currentPage(); p != nil {
					m.focus = focusWorkspace
					p.FocusOptions()
				}
			case "r":
				m.runSelected()
			case "y":
				m.copyOutput()
			case "s":
				if p := m.currentPage(); p != nil && p.HandleKey(msg.String()) {
					m.status = p.Status()
				} else {
					m.saveOutput()
				}
			case "p":
				if p := m.currentPage(); p != nil && p.HandleKey(msg.String()) {
					m.status = p.Status()
				} else {
					m.pipeOutput()
				}
			case "esc":
				m.focus = focusCategories
			case "?":
				m.openHelp()
			default:
				if p := m.currentPage(); p != nil && p.HandleKey(msg.String()) {
					m.status = p.Status()
				}
			}
		case focusWorkspace:
			p := m.currentPage()
			if p == nil {
				return m, nil
			}
			switch msg.String() {
			case "esc":
				m.focus = focusTabs
			case "tab":
				p.FocusNext()
			case "shift+tab":
				p.FocusPrev()
			case "enter", "i":
				p.FocusInput()
			case "o":
				p.FocusOptions()
			case "r":
				p.Run()
				m.status = p.Status()
			case "v":
				if p.HandleKey(msg.String()) {
					m.status = p.Status()
				} else {
					m.pasteInput()
				}
			case "x":
				p.ClearInput()
				m.status = successStyle.Render("cleared input")
			case "X":
				p.ClearOutput()
				m.status = successStyle.Render("cleared output")
			case "y":
				m.copyOutput()
			case "s":
				if p.HandleKey(msg.String()) {
					m.status = p.Status()
				} else {
					m.saveOutput()
				}
			case "p":
				if p.HandleKey(msg.String()) {
					m.status = p.Status()
				} else {
					m.pipeOutput()
				}
			case "?":
				m.openHelp()
			default:
				if p.HandleKey(msg.String()) {
					m.status = p.Status()
				}
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "sharp"
	}

	// 左侧保持 20% 宽度，右侧吃掉剩余空间；上下边框手动固定，避免内容反向撑开布局。
	leftWidth := clamp(m.width*20/100, 22, max(22, m.width-60))
	rightWidth := max(40, m.width-leftWidth)
	bodyHeight := max(10, m.height-1)

	// legend 标题已经在边框上，不再额外占内容行；Search 只需要一行输入。
	searchHeight := 3
	categoryHeight := max(7, bodyHeight-searchHeight)
	toolHeight := clamp(m.toolRows(rightWidth-2)+2, 3, max(3, bodyHeight-8))
	workspaceHeight := max(6, bodyHeight-toolHeight)

	search := panelBox(1, "Search", m.focus == focusSearch, m.searchView(leftWidth-2, panelContentHeight(searchHeight)), leftWidth, searchHeight)
	categories := panelBox(2, "Categories", m.focus == focusCategories, m.categoryView(leftWidth-2, panelContentHeight(categoryHeight)), leftWidth, categoryHeight)
	left := lipgloss.JoinVertical(lipgloss.Left, search, categories)
	tabs := panelBox(3, m.tabsTitle(), m.focus == focusTabs, m.tabsView(rightWidth-2, panelContentHeight(toolHeight)), rightWidth, toolHeight)
	page := panelBox(4, "Workspace", m.focus == focusWorkspace, m.pageView(rightWidth-2, panelContentHeight(workspaceHeight)), rightWidth, workspaceHeight)
	right := lipgloss.JoinVertical(lipgloss.Left, tabs, page)
	footer := footerStyle.Render(m.footerLine())
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	view := body + "\n" + footer
	if m.showHelp {
		view = overlayPanel(view, m.helpView(), m.width, m.height)
	}
	return fitHeight(view, m.height)
}

func (m *model) resize() {
	if p := m.currentPage(); p != nil {
		leftWidth := clamp(m.width*20/100, 22, max(22, m.width-60))
		rightWidth := max(40, m.width-leftWidth)
		bodyHeight := max(10, m.height-1)
		toolHeight := clamp(m.toolRows(rightWidth-2)+2, 3, max(3, bodyHeight-8))
		workspaceHeight := max(6, bodyHeight-toolHeight)
		p.SetSize(rightWidth-2, max(4, panelContentHeight(workspaceHeight)-4))
	}
	m.search.Width = max(10, clamp(m.width*20/100, 22, max(22, m.width-60))-4)
	m.help.Width = max(18, min(42, m.width-10))
}

func (m model) searchView(width, height int) string {
	if m.focus == focusSearch {
		return m.search.View()
	}
	query := strings.TrimSpace(m.search.Value())
	if query == "" {
		query = "/"
	}
	return dimStyle.Render(query)
}

func (m model) categoryView(width, height int) string {
	var b strings.Builder
	visible := max(1, height)
	start := 0
	if m.selectedCat >= visible {
		start = m.selectedCat - visible + 1
	}

	for i := start; i < len(m.categories); i++ {
		c := m.categories[i]
		line := fmt.Sprintf("%-12s %d", c.Category, len(c.Tabs))
		if len(line) > width {
			line = trimToWidth(line, width)
		}
		if i == m.selectedCat {
			line = selectedToken(line)
		} else {
			line = dimStyle.Render(line)
		}
		b.WriteString(line + "\n")
		if i-start+1 >= visible {
			break
		}
	}

	if len(m.categories) == 0 {
		b.WriteString(dimStyle.Render("No categories") + "\n")
	}
	return b.String()
}

func (m model) tabsView(width, height int) string {
	cat := m.currentCategory()
	if cat == nil {
		return dimStyle.Render("No category selected")
	}
	if cat.Category == tool.CategoryJSON {
		return flowLabels(width, height, []string{"Workspace", "Actions", "Raw Tools"}, 0)
	}

	labels := make([]string, 0, len(cat.Tabs))
	for _, tab := range cat.Tabs {
		labels = append(labels, tab.Tool.Name())
	}
	return flowLabels(width, height, labels, m.selectedTab)
}

func flowLabels(width, height int, labels []string, selected int) string {
	var b strings.Builder
	lineWidth := 0
	lines := 0
	// 工具区按标签宽度流式排布：一行放不下才换行，类似 lazygit 顶部 tab 标题。
	for i, labelText := range labels {
		label := trimToWidth(labelText, max(8, min(24, width)))
		cellWidth := len(label) + 2
		separator := 1
		if lineWidth == 0 {
			separator = 0
		}
		if lineWidth > 0 && lineWidth+separator+cellWidth > width {
			b.WriteString("\n")
			lineWidth = 0
			lines++
			if lines >= height {
				break
			}
		}
		if lineWidth > 0 {
			b.WriteString(" ")
			lineWidth++
		}
		cell := " " + label + " "
		if i == selected {
			b.WriteString(activeLabelStyle.Render(cell))
		} else {
			b.WriteString(dimStyle.Render(cell))
		}
		lineWidth += cellWidth
		if lines >= height && i < len(labels)-1 {
			break
		}
	}
	return b.String()
}

func (m model) toolRows(width int) int {
	cat := m.currentCategory()
	if cat == nil || width <= 0 {
		return 1
	}
	if cat.Category == tool.CategoryJSON {
		return labelRows(width, []string{"Workspace", "Actions", "Raw Tools"})
	}
	labels := make([]string, 0, len(cat.Tabs))
	for _, tab := range cat.Tabs {
		labels = append(labels, tab.Tool.Name())
	}
	return labelRows(width, labels)
}

func labelRows(width int, labels []string) int {
	if len(labels) == 0 || width <= 0 {
		return 1
	}
	rows := 1
	lineWidth := 0
	// 先用同一套显示宽度规则预估工具标签需要几行，再把剩余高度全部交给 Workspace。
	for _, labelText := range labels {
		label := trimToWidth(labelText, max(8, min(24, width)))
		cellWidth := len(label) + 2
		separator := 0
		if lineWidth > 0 {
			separator = 1
		}
		if lineWidth > 0 && lineWidth+separator+cellWidth > width {
			rows++
			lineWidth = cellWidth
			continue
		}
		lineWidth += separator + cellWidth
	}
	return rows
}

func (m model) pageView(width, height int) string {
	p := m.currentPage()
	if p == nil {
		return dimStyle.Render("No tool selected")
	}

	var b strings.Builder
	if tab := m.currentTab(); tab != nil {
		b.WriteString(appTitleStyle.Render(tab.Tool.Name()) + " " + dimStyle.Render(tab.Tool.ID()) + "\n")
		b.WriteString(dimStyle.Render(tab.Tool.Description()) + "\n")
		b.WriteString(sectionRuleStyle.Render(strings.Repeat("-", max(8, width))) + "\n")
	}
	b.WriteString(p.View())
	return b.String()
}

func (m model) tabsTitle() string {
	if cat := m.currentCategory(); cat != nil {
		if cat.Category == tool.CategoryJSON {
			return "JSON Pages"
		}
		return "Tools - " + string(cat.Category)
	}
	return "Tools"
}

func (m model) footerLine() string {
	if m.showHelp {
		return "HELP | type to filter | Ctrl+u clear | Esc/? close"
	}
	switch m.focus {
	case focusSearch:
		return "1 search | Esc close | Enter accept"
	case focusCategories:
		return "1 search | 2 categories | 3 tools | j/k move | Enter tools"
	case focusTabs:
		if p := m.currentPage(); p != nil {
			if p.IsEditing() {
				return "Esc exit edit | Tab next field | / and numbers are text"
			}
			return "1 search | 2 categories | 3 tools | 4 workspace | h/l select | r run"
		}
	case focusWorkspace:
		if p := m.currentPage(); p != nil {
			if p.IsEditing() {
				return "Esc exit edit | Tab input/output | / and numbers are text"
			}
			if hints := p.FooterHints(); hints != "" {
				return hints
			}
			return "1 search | 2 categories | 3 tools | 4 workspace | i edit | v paste | x clear input | X clear output | r run | y copy"
		}
	}
	return m.status
}

func (m model) helpView() string {
	commands := m.helpCommands()
	query := strings.ToLower(strings.TrimSpace(m.help.Value()))
	filtered := make([]helpCommand, 0, len(commands))
	for _, cmd := range commands {
		haystack := strings.ToLower(cmd.Key + " " + cmd.Command + " " + cmd.Description)
		if query == "" || strings.Contains(haystack, query) {
			filtered = append(filtered, cmd)
		}
	}

	var b strings.Builder
	b.WriteString(sectionLabel("Filter", true, true) + " ")
	b.WriteString(m.help.View() + "\n")
	b.WriteString(dimStyle.Render("Esc/? close | Ctrl+u clear filter") + "\n")
	b.WriteString(sectionRuleStyle.Render(strings.Repeat("-", 64)) + "\n")
	if len(filtered) == 0 {
		b.WriteString(dimStyle.Render("No matching commands") + "\n")
		return b.String()
	}
	for _, cmd := range filtered {
		key := activeLabelStyle.Render(" " + cmd.Key + " ")
		line := fmt.Sprintf("%-18s %s", cmd.Command, cmd.Description)
		b.WriteString(key + " " + line + "\n")
	}
	return b.String()
}

func (m model) helpCommands() []helpCommand {
	commands := []helpCommand{
		{Key: "1", Command: "Search Area", Description: "Focus global category/tool search."},
		{Key: "2", Command: "Categories", Description: "Focus category list."},
		{Key: "3", Command: "Pages/Tools", Description: "Focus page or tool strip."},
		{Key: "4", Command: "Workspace", Description: "Focus current workspace."},
		{Key: "?", Command: "Help", Description: "Open or close this command help."},
		{Key: "q", Command: "Quit", Description: "Quit sharp from normal mode."},
		{Key: "Esc", Command: "Back", Description: "Leave edit mode, close overlays, or move focus back."},
	}
	switch m.focus {
	case focusCategories:
		commands = append(commands,
			helpCommand{Key: "j/down", Command: "Next Category", Description: "Move category selection down."},
			helpCommand{Key: "k/up", Command: "Previous Category", Description: "Move category selection up."},
			helpCommand{Key: "Enter", Command: "Open Pages", Description: "Move focus to area 3."},
		)
	case focusTabs:
		commands = append(commands,
			helpCommand{Key: "h/left", Command: "Previous", Description: "Select previous tool for non-workbench categories."},
			helpCommand{Key: "l/right", Command: "Next", Description: "Select next tool for non-workbench categories."},
			helpCommand{Key: "Enter/i", Command: "Edit Input", Description: "Focus workspace input editor."},
			helpCommand{Key: "o", Command: "Edit Options", Description: "Focus options or path editor when available."},
		)
	case focusWorkspace:
		commands = append(commands,
			helpCommand{Key: "i/Enter", Command: "Edit Input", Description: "Enter insert mode for input."},
			helpCommand{Key: "o", Command: "Edit Options", Description: "Edit options or JSON path."},
			helpCommand{Key: "Tab", Command: "Next Field", Description: "Cycle input, options/path, and output."},
			helpCommand{Key: "Shift+Tab", Command: "Previous Field", Description: "Cycle fields backward."},
			helpCommand{Key: "v", Command: "Paste", Description: "Paste clipboard into input."},
			helpCommand{Key: "x", Command: "Clear Input", Description: "Clear the current input."},
			helpCommand{Key: "X", Command: "Clear Output", Description: "Clear the current output."},
			helpCommand{Key: "y", Command: "Copy Output", Description: "Copy trimmed output text."},
			helpCommand{Key: "r", Command: "Run", Description: "Run default action for this page."},
		)
	}
	if p := m.currentPage(); p != nil {
		commands = append(commands, p.HelpCommands()...)
	}
	return commands
}

func (m *model) openHelp() {
	m.showHelp = true
	m.help.Focus()
}

func panelBox(num int, title string, active bool, content string, width, height int) string {
	borderColor := lipgloss.Color("250")
	titleStyle := panelTitleStyle
	if active {
		borderColor = lipgloss.Color("39")
		titleStyle = appTitleStyle
	}
	// 手写 legend 风格边框：标题嵌入上边框，内容永远被裁剪/补齐到固定宽高。
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleText := fmt.Sprintf("[%d]-%s", num, title)
	titleText = truncateDisplay(titleText, max(1, width-4))
	renderedTitle := titleStyle.Render(titleText)

	innerWidth := max(0, width-2)
	bodyHeight := max(0, height-2)
	titleWidth := lipgloss.Width(titleText)
	rightRuleWidth := max(0, innerWidth-titleWidth-1)
	top := borderStyle.Render("┌─") + renderedTitle + borderStyle.Render(strings.Repeat("─", rightRuleWidth)+"┐")
	bottom := borderStyle.Render("└" + strings.Repeat("─", innerWidth) + "┘")

	contentLines := fixedLines(content, innerWidth, bodyHeight)
	lines := make([]string, 0, height)
	lines = append(lines, top)
	for _, line := range contentLines {
		lines = append(lines, borderStyle.Render("│")+line+borderStyle.Render("│"))
	}
	lines = append(lines, bottom)
	return strings.Join(lines, "\n")
}

func overlayPanel(base string, overlay string, width, height int) string {
	if width <= 0 || height <= 0 {
		return base
	}
	boxWidth := clamp(width*70/100, 46, max(46, width-4))
	boxHeight := clamp(height*70/100, 10, max(10, height-2))
	box := panelBox(0, "Help", true, overlay, boxWidth, boxHeight)
	baseLines := fixedLines(base, width, height)
	boxLines := strings.Split(box, "\n")
	top := max(0, (height-boxHeight)/2)
	left := max(0, (width-boxWidth)/2)
	for i, line := range boxLines {
		row := top + i
		if row >= len(baseLines) {
			break
		}
		prefix := truncateDisplay(baseLines[row], left)
		if lipgloss.Width(prefix) < left {
			prefix += strings.Repeat(" ", left-lipgloss.Width(prefix))
		}
		suffixStart := min(width, left+boxWidth)
		suffix := ""
		if suffixStart < width {
			suffix = cutDisplayLeft(baseLines[row], suffixStart)
			suffix = truncateDisplay(suffix, width-suffixStart)
		}
		baseLines[row] = fitWidth(prefix+line+suffix, width)
	}
	return strings.Join(baseLines, "\n")
}

func cutDisplayLeft(s string, width int) string {
	if width <= 0 {
		return s
	}
	remaining := width
	for i, r := range s {
		rw := lipgloss.Width(string(r))
		if rw == 0 {
			continue
		}
		if remaining-rw < 0 {
			return s[i:]
		}
		remaining -= rw
		if remaining == 0 {
			return s[i+len(string(r)):]
		}
	}
	return ""
}

func panelContentHeight(outerHeight int) int {
	// 外框上下两行是边框，剩余行才是真正可渲染的内容。
	return max(1, outerHeight-2)
}

func fitHeight(s string, height int) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= height {
		return s
	}
	return strings.Join(lines[:height], "\n")
}

func fixedLines(s string, width int, height int) []string {
	out := make([]string, 0, height)
	lines := strings.Split(s, "\n")
	// 去掉尾部空行，避免组件自然输出的最后一个换行变成多余空白。
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i := 0; i < height; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		out = append(out, fitWidth(line, width))
	}
	return out
}

func fitWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	// 所有内容行都按显示宽度处理，兼容中文、全角字符和 ANSI 样式。
	s = truncateDisplay(s, width)
	w := lipgloss.Width(s)
	if w < width {
		s += strings.Repeat(" ", width-w)
	}
	return s
}

func truncateDisplay(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	// lipgloss 的 MaxWidth 会按终端显示宽度截断，比 len/rune 数更适合 TUI。
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}

func (m *model) moveCategory(delta int) {
	if len(m.categories) == 0 {
		return
	}
	next := clamp(m.selectedCat+delta, 0, len(m.categories)-1)
	if next == m.selectedCat {
		return
	}
	m.selectedCat = next
	m.restoreTabForSelectedCategory()
	m.rebuildPage()
	m.status = m.categoryStatus()
}

func (m *model) moveTab(delta int) {
	cat := m.currentCategory()
	if cat == nil || len(cat.Tabs) == 0 {
		return
	}
	next := clamp(m.selectedTab+delta, 0, len(cat.Tabs)-1)
	if next == m.selectedTab {
		return
	}
	m.setSelectedTab(next)
	m.rebuildPage()
	m.status = m.tabStatus()
}

func (m *model) applySearch() {
	q := strings.ToLower(strings.TrimSpace(m.search.Value()))
	m.categories = filterCategories(m.allCategories, q)
	if len(m.categories) == 0 {
		m.selectedCat = 0
		m.selectedTab = 0
		m.rebuildPage()
		return
	}
	if m.selectedCat >= len(m.categories) {
		m.selectedCat = len(m.categories) - 1
	}
	m.restoreTabForSelectedCategory()
	m.rebuildPage()
}

func (m *model) syncSelection() {
	if len(m.categories) == 0 {
		m.selectedCat = 0
		m.selectedTab = 0
		return
	}
	if m.selectedCat >= len(m.categories) {
		m.selectedCat = len(m.categories) - 1
	}
	m.restoreTabForSelectedCategory()
}

func (m *model) restoreTabForSelectedCategory() {
	cat := m.currentCategory()
	if cat == nil || len(cat.Tabs) == 0 {
		m.selectedTab = 0
		return
	}
	if idx, ok := m.lastTabByCat[cat.Category]; ok {
		m.selectedTab = clamp(idx, 0, len(cat.Tabs)-1)
		return
	}
	if m.selectedTab >= len(cat.Tabs) {
		m.selectedTab = len(cat.Tabs) - 1
	}
}

func (m *model) setSelectedTab(idx int) {
	cat := m.currentCategory()
	if cat == nil || idx < 0 || idx >= len(cat.Tabs) {
		return
	}
	m.selectedTab = idx
	m.lastTabByCat[cat.Category] = idx
}

func (m *model) currentCategory() *categoryGroup {
	if len(m.categories) == 0 || m.selectedCat < 0 || m.selectedCat >= len(m.categories) {
		return nil
	}
	return &m.categories[m.selectedCat]
}

func (m *model) currentCategoryIsJSON() bool {
	cat := m.currentCategory()
	return cat != nil && cat.Category == tool.CategoryJSON
}

func (m *model) currentTab() *tabItem {
	cat := m.currentCategory()
	if cat == nil || len(cat.Tabs) == 0 || m.selectedTab < 0 || m.selectedTab >= len(cat.Tabs) {
		return nil
	}
	return &cat.Tabs[m.selectedTab]
}

func (m *model) currentPage() toolPage {
	tab := m.currentTab()
	if tab == nil {
		return nil
	}
	id := tab.Tool.ID()
	if tab.Tool.Category() == tool.CategoryJSON {
		id = "json.workspace"
	}
	if p, ok := m.pages[id]; ok {
		return p
	}
	p := m.newToolPage(tab.Tool)
	m.pages[id] = p
	return p
}

func (m *model) setCurrentPage(p toolPage) {
	tab := m.currentTab()
	if tab == nil || p == nil {
		return
	}
	id := tab.Tool.ID()
	if tab.Tool.Category() == tool.CategoryJSON {
		id = "json.workspace"
	}
	m.pages[id] = p
}

func (m *model) newToolPage(t tool.Tool) toolPage {
	if t.Category() == tool.CategoryJSON {
		tools := make(map[string]tool.Tool)
		for _, c := range m.allCategories {
			if c.Category != tool.CategoryJSON {
				continue
			}
			for _, tab := range c.Tabs {
				tools[tab.Tool.ID()] = tab.Tool
			}
			break
		}
		return newJSONToolPage(t, tools)
	}
	return newToolPage(t)
}

func (m *model) rebuildPage() {
	if m.currentTab() == nil {
		return
	}
	m.resize()
}

func (m *model) runSelected() {
	if p := m.currentPage(); p != nil {
		p.Run()
		m.status = p.Status()
	}
}

func (m *model) copyOutput() {
	p := m.currentPage()
	if p == nil {
		return
	}
	text := p.OutputText()
	if text == "" {
		m.status = dimStyle.Render("nothing to copy")
		return
	}
	if err := clipboard.Write(text); err != nil {
		m.status = errorStyle.Render("copy failed: " + err.Error())
		return
	}
	m.status = successStyle.Render("copied output")
}

func (m *model) saveOutput() {
	p := m.currentPage()
	if p == nil {
		return
	}
	text := p.OutputText()
	if text == "" {
		m.status = dimStyle.Render("nothing to save")
		return
	}
	if err := os.WriteFile("sharp-output.txt", []byte(text), 0600); err != nil {
		m.status = errorStyle.Render("save failed: " + err.Error())
		return
	}
	m.status = successStyle.Render("saved to sharp-output.txt")
}

func (m *model) pipeOutput() {
	p := m.currentPage()
	if p == nil {
		return
	}
	text := p.OutputText()
	if text == "" {
		m.status = dimStyle.Render("nothing to pipe")
		return
	}
	p.SetInput(text)
	m.focus = focusWorkspace
	p.FocusInput()
	m.status = successStyle.Render("output moved to input")
}

func (m *model) pasteInput() {
	p := m.currentPage()
	if p == nil {
		return
	}
	text, err := clipboard.Read()
	if err != nil {
		m.status = errorStyle.Render("paste failed: " + err.Error())
		return
	}
	p.SetInput(text)
	m.status = successStyle.Render("pasted clipboard to input")
}

func (m *model) categoryStatus() string {
	cat := m.currentCategory()
	if cat == nil {
		return "No category selected"
	}
	return fmt.Sprintf("category %s | %d tools", cat.Category, len(cat.Tabs))
}

func (m *model) tabStatus() string {
	tab := m.currentTab()
	if tab == nil {
		return "No tab selected"
	}
	return fmt.Sprintf("tab %s", tab.Tool.Name())
}

func buildCategories(tools []tool.Tool) []categoryGroup {
	out := make([]categoryGroup, 0)
	if len(tools) == 0 {
		return out
	}

	var current *categoryGroup
	for _, t := range tools {
		if current == nil || current.Category != t.Category() {
			out = append(out, categoryGroup{Category: t.Category()})
			current = &out[len(out)-1]
		}
		current.Tabs = append(current.Tabs, tabItem{Tool: t})
	}
	return out
}

func filterCategories(all []categoryGroup, q string) []categoryGroup {
	if q == "" {
		return all
	}
	out := make([]categoryGroup, 0, len(all))
	for _, c := range all {
		filteredTabs := make([]tabItem, 0, len(c.Tabs))
		for _, t := range c.Tabs {
			haystack := strings.ToLower(t.Tool.ID() + " " + t.Tool.Name() + " " + string(t.Tool.Category()) + " " + t.Tool.Description())
			if strings.Contains(haystack, q) {
				filteredTabs = append(filteredTabs, t)
			}
		}
		if len(filteredTabs) == 0 {
			continue
		}
		out = append(out, categoryGroup{
			Category: c.Category,
			Tabs:     filteredTabs,
		})
	}
	return out
}

type genericToolPage struct {
	tool       tool.Tool
	input      textarea.Model
	optionBox  textinput.Model
	output     viewport.Model
	outputText string
	outputErr  bool
	options    tool.Options
	focus      pageFocus
	editing    bool
	status     string
	hasOptions bool
}

type pageFocus int

const (
	pageFocusInput pageFocus = iota
	pageFocusOptions
	pageFocusOutput
)

func newToolPage(t tool.Tool) toolPage {
	if factory, ok := specialPageFactories[t.ID()]; ok {
		return factory(t)
	}
	return newGenericToolPage(t)
}

var specialPageFactories = map[string]func(tool.Tool) toolPage{}

func newGenericToolPage(t tool.Tool) *genericToolPage {
	input := textarea.New()
	input.Placeholder = ""
	input.ShowLineNumbers = false
	input.SetHeight(8)
	input.Blur()

	optionBox := textinput.New()
	optionBox.Placeholder = "option=value option2=value2"
	optionBox.Blur()

	output := viewport.New(80, 12)

	p := &genericToolPage{
		tool:       t,
		input:      input,
		optionBox:  optionBox,
		output:     output,
		options:    tool.Options{},
		status:     "Enter/i edit | o options | r run | y copy | s save | p pipe",
		hasOptions: len(t.Options()) > 0,
	}
	p.loadSelectedOptions()
	return p
}

func (p *genericToolPage) SetSize(width, height int) {
	// 通用工具页先做“一个输入、一个输出”的布局：输入和输出按可用高度近似各占一半。
	inputHeight := max(3, height/2-1)
	outputHeight := max(3, height-inputHeight-3)
	p.input.SetWidth(max(18, width))
	p.input.SetHeight(inputHeight)
	p.optionBox.Width = max(18, width)
	p.output.Width = max(18, width)
	p.output.Height = outputHeight
	p.renderOutput()
}

func (p *genericToolPage) Update(msg tea.Msg) (toolPage, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.SetSize(msg.Width, msg.Height)
		return p, nil
	case tea.KeyMsg:
		if msg.String() == "esc" {
			p.stopEditing()
			return p, nil
		}
		if !p.editing {
			if p.focus == pageFocusOutput {
				var cmd tea.Cmd
				p.output, cmd = p.output.Update(msg)
				return p, cmd
			}
			return p, nil
		}
		switch p.focus {
		case pageFocusInput:
			var cmd tea.Cmd
			p.input, cmd = p.input.Update(msg)
			return p, cmd
		case pageFocusOptions:
			var cmd tea.Cmd
			p.optionBox, cmd = p.optionBox.Update(msg)
			return p, cmd
		}
	}
	return p, nil
}

func (p *genericToolPage) View() string {
	var b strings.Builder
	b.WriteString(sectionLabel("Input", p.focus == pageFocusInput, p.editing) + "\n")
	b.WriteString(p.input.View() + "\n")

	if p.hasOptions {
		if p.focus == pageFocusOptions || p.editing {
			b.WriteString(sectionLabel("Options", p.focus == pageFocusOptions, p.editing) + " ")
			b.WriteString(p.optionBox.View() + "\n")
		} else {
			b.WriteString(dimStyle.Render("Options: "+p.optionBox.Value()) + "\n")
		}
		for _, opt := range p.tool.Options() {
			req := ""
			if opt.Required {
				req = " required"
			}
			line := fmt.Sprintf("--%s%s %s", opt.Name, req, opt.Description)
			b.WriteString(dimStyle.Render(line) + "\n")
		}
	} else {
		b.WriteString(dimStyle.Render("Options: none") + "\n")
	}

	b.WriteString(sectionLabel("Output", p.focus == pageFocusOutput, false) + "\n")
	b.WriteString(p.output.View())
	return b.String()
}

func (p *genericToolPage) FocusInput() {
	p.focus = pageFocusInput
	p.editing = true
	p.optionBox.Blur()
	p.input.Focus()
}

func (p *genericToolPage) FocusOptions() {
	if !p.hasOptions {
		p.FocusInput()
		return
	}
	p.focus = pageFocusOptions
	p.editing = true
	p.input.Blur()
	p.optionBox.Focus()
}

func (p *genericToolPage) FocusOutput() {
	p.focus = pageFocusOutput
	p.stopEditing()
}

func (p *genericToolPage) FocusNext() {
	switch p.focus {
	case pageFocusInput:
		if p.hasOptions {
			p.FocusOptions()
		} else {
			p.FocusOutput()
		}
	case pageFocusOptions:
		p.FocusOutput()
	case pageFocusOutput:
		p.FocusInput()
	}
}

func (p *genericToolPage) FocusPrev() {
	switch p.focus {
	case pageFocusInput:
		p.FocusOutput()
	case pageFocusOptions:
		p.FocusInput()
	case pageFocusOutput:
		if p.hasOptions {
			p.FocusOptions()
		} else {
			p.FocusInput()
		}
	}
}

func (p *genericToolPage) Run() {
	p.parseOptions()
	out, err := p.tool.Run(context.Background(), tool.Input{Text: p.input.Value()}, p.options)
	if err != nil {
		p.outputText = err.Error()
		p.outputErr = true
		p.renderOutput()
		p.status = errorStyle.Render("run failed: " + p.tool.ID())
		return
	}
	p.outputText = out.Text
	p.outputErr = false
	p.renderOutput()
	p.status = successStyle.Render("ran " + p.tool.ID())
}

func (p *genericToolPage) SetInput(value string) {
	p.input.SetValue(value)
}

func (p *genericToolPage) ClearInput() {
	p.input.SetValue("")
}

func (p *genericToolPage) ClearOutput() {
	p.outputText = ""
	p.outputErr = false
	p.output.SetContent("")
}

func (p *genericToolPage) OutputText() string {
	// 复制、保存、pipe 都应该使用工具真实输出；viewport.View() 是屏幕渲染结果，会包含补齐空格或被滚动裁剪。
	return strings.TrimSpace(p.outputText)
}

func (p *genericToolPage) renderOutput() {
	// viewport 不会自动软换行；这里按当前显示宽度预处理，避免长行横向溢出导致内容不可见。
	wrapped := wrapDisplayText(p.outputText, max(1, p.output.Width))
	if p.outputErr {
		wrapped = errorStyle.Render(wrapped)
	}
	p.output.SetContent(wrapped)
}

func (p *genericToolPage) Status() string {
	return p.status
}

func (p *genericToolPage) HelpText() string {
	return fmt.Sprintf("%s | Esc back/exit edit | Tab field | r run | y copy | s save | p pipe", p.tool.Name())
}

func (p *genericToolPage) HelpCommands() []helpCommand {
	commands := []helpCommand{
		{Key: "r", Command: "Run Tool", Description: "Run " + p.tool.Name() + " with current input and options."},
		{Key: "p", Command: "Pipe Output", Description: "Move output into input for another run."},
		{Key: "s", Command: "Save Output", Description: "Save trimmed output to sharp-output.txt."},
	}
	if p.hasOptions {
		commands = append(commands, helpCommand{Key: "o", Command: "Edit Options", Description: "Edit option=value pairs for this tool."})
	}
	return commands
}

func (p *genericToolPage) IsEditing() bool {
	return p.editing
}

func (p *genericToolPage) HandleKey(key string) bool {
	return false
}

func (p *genericToolPage) optionHeight() int {
	if !p.hasOptions {
		return 1
	}
	return 2 + len(p.tool.Options())
}

func (p *genericToolPage) stopEditing() {
	p.editing = false
	p.input.Blur()
	p.optionBox.Blur()
}

func (p *genericToolPage) loadSelectedOptions() {
	parts := make([]string, 0, len(p.tool.Options()))
	for _, opt := range p.tool.Options() {
		value := opt.Default
		if value == "" && opt.Required {
			value = ""
		}
		parts = append(parts, fmt.Sprintf("%s=%s", opt.Name, value))
	}
	p.optionBox.SetValue(strings.Join(parts, " "))
	p.parseOptions()
}

func (p *genericToolPage) parseOptions() {
	p.options = tool.Options{}
	for _, part := range strings.Fields(p.optionBox.Value()) {
		k, v, ok := strings.Cut(part, "=")
		if ok {
			p.options[k] = v
		}
	}
}

type jsonToolPage struct {
	input      textarea.Model
	pathBox    textinput.Model
	output     viewport.Model
	tools      map[string]tool.Tool
	width      int
	focus      pageFocus
	editing    bool
	inputUndo  []string
	outputText string
	outputErr  bool
	status     string
	lastAction string
}

func newJSONToolPage(t tool.Tool, tools map[string]tool.Tool) *jsonToolPage {
	input := textarea.New()
	input.Placeholder = ""
	input.ShowLineNumbers = false
	input.SetHeight(8)
	input.Blur()

	pathBox := textinput.New()
	pathBox.Placeholder = "gjson path, e.g. data.user.name"
	pathBox.Blur()

	p := &jsonToolPage{
		input:   input,
		pathBox: pathBox,
		output:  viewport.New(80, 12),
		tools:   tools,
		status:  "JSON Workspace | i edit | v paste | p pretty | m minify | c check | e escape | a apply",
	}
	if _, ok := p.tools[t.ID()]; !ok {
		p.tools[t.ID()] = t
	}
	return p
}

func (p *jsonToolPage) SetSize(width, height int) {
	// JSON 工作台同样保持输入/输出上下分布，Path 行和 Actions 行固定占用少量高度。
	p.width = max(18, width)
	actionHeight := lipgloss.Height(p.actionsView())
	inputHeight := max(3, height/2-2)
	outputHeight := max(3, height-inputHeight-actionHeight-5)
	p.input.SetWidth(max(18, width))
	p.input.SetHeight(inputHeight)
	p.pathBox.Width = max(18, width-10)
	p.output.Width = max(18, width)
	p.output.Height = outputHeight
	p.renderOutput()
}

func (p *jsonToolPage) Update(msg tea.Msg) (toolPage, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.SetSize(msg.Width, msg.Height)
		return p, nil
	case tea.KeyMsg:
		if msg.String() == "esc" {
			p.stopEditing()
			return p, nil
		}
		if !p.editing {
			if p.focus == pageFocusOutput {
				var cmd tea.Cmd
				p.output, cmd = p.output.Update(msg)
				return p, cmd
			}
			return p, nil
		}
		switch p.focus {
		case pageFocusInput:
			var cmd tea.Cmd
			p.input, cmd = p.input.Update(msg)
			return p, cmd
		case pageFocusOptions:
			var cmd tea.Cmd
			p.pathBox, cmd = p.pathBox.Update(msg)
			return p, cmd
		}
	}
	return p, nil
}

func (p *jsonToolPage) View() string {
	var b strings.Builder
	b.WriteString(sectionLabel("Input", p.focus == pageFocusInput, p.editing) + "\n")
	b.WriteString(p.input.View() + "\n")
	b.WriteString(sectionLabel("Path", p.focus == pageFocusOptions, p.editing) + " ")
	if p.focus == pageFocusOptions || p.editing {
		b.WriteString(p.pathBox.View() + "\n")
	} else {
		value := p.pathBox.Value()
		if value == "" {
			value = "gjson path"
		}
		b.WriteString(dimStyle.Render(value) + "\n")
	}
	b.WriteString(p.actionsView() + "\n")
	if p.lastAction != "" {
		b.WriteString(dimStyle.Render("Last: "+p.lastAction+" | actions read Input; [a] promotes Output to Input") + "\n")
	}
	b.WriteString(sectionLabel("Output", p.focus == pageFocusOutput, false) + "\n")
	b.WriteString(p.output.View())
	return b.String()
}

func (p *jsonToolPage) FocusInput() {
	p.focus = pageFocusInput
	p.editing = true
	p.pathBox.Blur()
	p.input.Focus()
}

func (p *jsonToolPage) FocusOptions() {
	p.focus = pageFocusOptions
	p.editing = true
	p.input.Blur()
	p.pathBox.Focus()
}

func (p *jsonToolPage) FocusOutput() {
	p.focus = pageFocusOutput
	p.stopEditing()
}

func (p *jsonToolPage) FocusNext() {
	switch p.focus {
	case pageFocusInput:
		p.FocusOptions()
	case pageFocusOptions:
		p.FocusOutput()
	case pageFocusOutput:
		p.FocusInput()
	}
}

func (p *jsonToolPage) FocusPrev() {
	switch p.focus {
	case pageFocusInput:
		p.FocusOutput()
	case pageFocusOptions:
		p.FocusInput()
	case pageFocusOutput:
		p.FocusOptions()
	}
}

func (p *jsonToolPage) Run() {
	p.runAction("json.pretty", nil)
}

func (p *jsonToolPage) ClearInput() {
	p.pushUndo()
	p.input.SetValue("")
}

func (p *jsonToolPage) ClearOutput() {
	p.outputText = ""
	p.outputErr = false
	p.output.SetContent("")
}

func (p *jsonToolPage) SetInput(value string) {
	p.pushUndo()
	p.input.SetValue(value)
}

func (p *jsonToolPage) OutputText() string {
	return strings.TrimSpace(p.outputText)
}

func (p *jsonToolPage) Status() string {
	return p.status
}

func (p *jsonToolPage) HelpText() string {
	return "JSON Workspace | i edit | v paste | o path | p pretty | m minify | c check | s sort | e escape | u unescape | g get | a apply | z undo"
}

func (p *jsonToolPage) actionsView() string {
	actions := []string{
		"[p] Pretty",
		"[m] Minify",
		"[c] Check",
		"[s] Sort",
		"[e] Escape",
		"[u] Unescape",
		"[g] Get Path",
		"[a] Apply",
		"[z] Undo",
	}
	return dimStyle.Render(wrapActionLine("Actions:", actions, max(18, p.width)))
}

func (p *jsonToolPage) HelpCommands() []helpCommand {
	return []helpCommand{
		{Key: "i/Enter", Command: "Edit JSON Input", Description: "Enter insert mode for the JSON input editor."},
		{Key: "o", Command: "Edit Path", Description: "Edit gjson path used by Get Path."},
		{Key: "v", Command: "Paste Input", Description: "Paste clipboard data into the input editor."},
		{Key: "p", Command: "Pretty JSON", Description: "Format Input with two-space indentation into Output."},
		{Key: "m", Command: "Minify JSON", Description: "Compact Input into one-line JSON in Output."},
		{Key: "c", Command: "Check JSON", Description: "Validate Input JSON syntax."},
		{Key: "s", Command: "Sort Keys", Description: "Recursively sort JSON object keys into Output."},
		{Key: "e", Command: "Escape String", Description: "Encode Input as a JSON string literal."},
		{Key: "u", Command: "Unescape String", Description: "Decode a JSON string literal from Input."},
		{Key: "g", Command: "Get Path", Description: "Query Input with the current gjson path."},
		{Key: "a", Command: "Apply Output", Description: "Promote Output into Input for the next action."},
		{Key: "z", Command: "Undo Input", Description: "Restore previous Input after apply, paste, or clear."},
	}
}

func (p *jsonToolPage) IsEditing() bool {
	return p.editing
}

func (p *jsonToolPage) HandleKey(key string) bool {
	switch key {
	case "p":
		p.runAction("json.pretty", nil)
	case "m":
		p.runAction("json.minify", nil)
	case "c":
		p.runAction("json.validate", nil)
	case "s":
		p.runAction("json.sort", nil)
	case "e":
		p.runAction("json.escape", nil)
	case "u":
		p.runAction("json.unescape", nil)
	case "g", "enter":
		path := strings.TrimSpace(p.pathBox.Value())
		if path == "" {
			p.FocusOptions()
			p.status = dimStyle.Render("enter path, then press Esc and g")
			return true
		}
		p.runAction("json.get", tool.Options{"path": path})
	case "a":
		p.applyOutput()
	case "z":
		p.undoInput()
	default:
		return false
	}
	return true
}

func (p *jsonToolPage) runAction(id string, options tool.Options) {
	t, ok := p.tools[id]
	if !ok {
		p.outputText = "missing JSON action: " + id
		p.outputErr = true
		p.renderOutput()
		p.status = errorStyle.Render(p.outputText)
		return
	}
	out, err := t.Run(context.Background(), tool.Input{Text: p.input.Value()}, options)
	p.lastAction = t.Name()
	if err != nil {
		p.outputText = err.Error()
		p.outputErr = true
		p.renderOutput()
		p.status = errorStyle.Render("failed: " + t.Name())
		return
	}
	p.outputText = out.Text
	p.outputErr = false
	p.renderOutput()
	p.status = successStyle.Render("ran " + t.Name())
}

func (p *jsonToolPage) applyOutput() {
	text := p.OutputText()
	if text == "" {
		p.status = dimStyle.Render("nothing to apply")
		return
	}
	p.pushUndo()
	p.input.SetValue(text)
	p.ClearOutput()
	p.status = successStyle.Render("applied output to input")
}

func (p *jsonToolPage) undoInput() {
	if len(p.inputUndo) == 0 {
		p.status = dimStyle.Render("nothing to undo")
		return
	}
	last := p.inputUndo[len(p.inputUndo)-1]
	p.inputUndo = p.inputUndo[:len(p.inputUndo)-1]
	p.input.SetValue(last)
	p.status = successStyle.Render("restored previous input")
}

func (p *jsonToolPage) pushUndo() {
	current := p.input.Value()
	if len(p.inputUndo) > 0 && p.inputUndo[len(p.inputUndo)-1] == current {
		return
	}
	p.inputUndo = append(p.inputUndo, current)
	if len(p.inputUndo) > 20 {
		p.inputUndo = p.inputUndo[len(p.inputUndo)-20:]
	}
}

func (p *jsonToolPage) renderOutput() {
	wrapped := wrapDisplayText(p.outputText, max(1, p.output.Width))
	if p.outputErr {
		wrapped = errorStyle.Render(wrapped)
	}
	p.output.SetContent(wrapped)
}

func (p *jsonToolPage) stopEditing() {
	p.editing = false
	p.input.Blur()
	p.pathBox.Blur()
}

func sectionLabel(title string, active bool, editing bool) string {
	if active && editing {
		return activeLabelStyle.Render(" " + title + " ")
	}
	if active {
		return panelTitleStyle.Render("> " + title)
	}
	return panelTitleStyle.Render(title)
}

func trimToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func wrapDisplayText(s string, width int) string {
	if width <= 0 || s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapDisplayLine(line, width)...)
	}
	return strings.Join(wrapped, "\n")
}

func wrapActionLine(prefix string, actions []string, width int) string {
	if width <= 0 {
		return ""
	}
	var lines []string
	line := prefix
	for _, action := range actions {
		next := line + " " + action
		if lipgloss.Width(next) > width && line != prefix {
			lines = append(lines, line)
			line = strings.Repeat(" ", lipgloss.Width(prefix)) + " " + action
			continue
		}
		line = next
	}
	lines = append(lines, line)
	return strings.Join(lines, "\n")
}

func wrapDisplayLine(line string, width int) []string {
	if line == "" {
		return []string{""}
	}
	var lines []string
	var b strings.Builder
	currentWidth := 0
	for _, r := range line {
		ch := string(r)
		chWidth := lipgloss.Width(ch)
		if chWidth == 0 {
			b.WriteRune(r)
			continue
		}
		// 按终端显示宽度折行，兼容中文、全角标点和普通英文长串。
		if currentWidth > 0 && currentWidth+chWidth > width {
			lines = append(lines, b.String())
			b.Reset()
			currentWidth = 0
		}
		b.WriteRune(r)
		currentWidth += chWidth
	}
	lines = append(lines, b.String())
	return lines
}

func selectedToken(s string) string {
	if s == "" {
		return ""
	}
	return selectedRowStyle.Render(s)
}

func clamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
