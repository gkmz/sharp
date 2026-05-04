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
	IsEditing() bool
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
	pages  map[string]toolPage

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

	m := model{
		registry:     registry,
		lastTabByCat: make(map[tool.Category]int),
		pages:        make(map[string]toolPage),
		search:       search,
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
				m.moveTab(1)
			case "h", "left", "[":
				m.moveTab(-1)
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
				m.saveOutput()
			case "p":
				m.pipeOutput()
			case "esc":
				m.focus = focusCategories
			case "?":
				if p := m.currentPage(); p != nil {
					m.status = p.HelpText()
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
				m.pasteInput()
			case "x":
				p.ClearInput()
				m.status = successStyle.Render("cleared input")
			case "X":
				p.ClearOutput()
				m.status = successStyle.Render("cleared output")
			case "y":
				m.copyOutput()
			case "s":
				m.saveOutput()
			case "p":
				m.pipeOutput()
			case "?":
				m.status = p.HelpText()
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
	return fitHeight(body+"\n"+footer, m.height)
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

	var b strings.Builder
	lineWidth := 0
	lines := 0
	// 工具区按标签宽度流式排布：一行放不下才换行，类似 lazygit 顶部 tab 标题。
	for i := 0; i < len(cat.Tabs); i++ {
		tab := cat.Tabs[i]
		label := trimToWidth(tab.Tool.Name(), max(8, min(24, width)))
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
		if i == m.selectedTab {
			b.WriteString(activeLabelStyle.Render(cell))
		} else {
			b.WriteString(dimStyle.Render(cell))
		}
		lineWidth += cellWidth
		if lines >= height && i < len(cat.Tabs)-1 {
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
	rows := 1
	lineWidth := 0
	// 先用同一套显示宽度规则预估工具标签需要几行，再把剩余高度全部交给 Workspace。
	for _, tab := range cat.Tabs {
		label := trimToWidth(tab.Tool.Name(), max(8, min(24, width)))
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
		return "Tools - " + string(cat.Category)
	}
	return "Tools"
}

func (m model) footerLine() string {
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
			return "1 search | 2 categories | 3 tools | 4 workspace | i edit | v paste | x clear input | X clear output | r run | y copy"
		}
	}
	return m.status
}

func panelBox(num int, title string, active bool, content string, width, height int) string {
	borderColor := lipgloss.Color("238")
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
	if p, ok := m.pages[id]; ok {
		return p
	}
	p := newToolPage(tab.Tool)
	m.pages[id] = p
	return p
}

func (m *model) setCurrentPage(p toolPage) {
	tab := m.currentTab()
	if tab == nil || p == nil {
		return
	}
	m.pages[tab.Tool.ID()] = p
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
	if err := clipboard.Write(p.OutputText()); err != nil {
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
	if err := os.WriteFile("sharp-output.txt", []byte(p.OutputText()), 0600); err != nil {
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
	p.SetInput(p.OutputText())
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
		p.output.SetContent(errorStyle.Render(err.Error()))
		p.status = errorStyle.Render("run failed: " + p.tool.ID())
		return
	}
	p.output.SetContent(out.Text)
	p.status = successStyle.Render("ran " + p.tool.ID())
}

func (p *genericToolPage) SetInput(value string) {
	p.input.SetValue(value)
}

func (p *genericToolPage) ClearInput() {
	p.input.SetValue("")
}

func (p *genericToolPage) ClearOutput() {
	p.output.SetContent("")
}

func (p *genericToolPage) OutputText() string {
	return p.output.View()
}

func (p *genericToolPage) Status() string {
	return p.status
}

func (p *genericToolPage) HelpText() string {
	return fmt.Sprintf("%s | Esc back/exit edit | Tab field | r run | y copy | s save | p pipe", p.tool.Name())
}

func (p *genericToolPage) IsEditing() bool {
	return p.editing
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
