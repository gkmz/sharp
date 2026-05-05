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
	"github.com/gkmz/sharp/internal/clipboard"
	"github.com/gkmz/sharp/pkg/tool"
)

type focus int

const (
	focusCategories focus = iota
	focusTabs
	focusWorkspace
	focusSearch
)

type categoryGroup struct {
	Category    tool.Category
	Subcategory string
	Tabs        []tabItem
	Header      bool
}

type tabItem struct {
	Tool tool.Tool
}

type toolSubgroup struct {
	Name string
	Tabs []tabItem
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
	OutputHalfPageDown()
	OutputHalfPageUp()
	InputHalfPageDown()
	InputHalfPageUp()
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
	helpVP viewport.Model
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
	statusKeyStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("31"))
	statusTextStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
)

// Run starts the interactive Bubble Tea application for registry.
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
		helpVP:       viewport.New(80, 20),
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
				m.helpVP.HalfPageUp()
				return m, nil
			case "ctrl+d":
				m.helpVP.HalfPageDown()
				return m, nil
			case "ctrl+x":
				m.help.SetValue("")
				m.refreshHelp()
				return m, nil
			}
			var cmd tea.Cmd
			m.help, cmd = m.help.Update(msg)
			m.refreshHelp()
			return m, cmd
		}

		if m.focus == focusSearch {
			if msg.String() == "esc" || msg.String() == "enter" {
				m.focus = m.searchFrom
				m.search.Blur()
				m.applySearch()
				if msg.String() == "enter" && m.focusSingleSearchResult() {
					m.focus = focusWorkspace
				}
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

		if msg.String() == "ctrl+d" || msg.String() == "ctrl+u" || msg.String() == "alt+d" || msg.String() == "alt+u" {
			if p := m.currentPage(); p != nil && m.focus == focusWorkspace {
				switch msg.String() {
				case "ctrl+d":
					p.OutputHalfPageDown()
				case "ctrl+u":
					p.OutputHalfPageUp()
				case "alt+d":
					p.InputHalfPageDown()
				case "alt+u":
					p.InputHalfPageUp()
				}
				m.status = p.Status()
				return m, nil
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
	helpBoxWidth := clamp(m.width*70/100, 46, max(46, m.width-4))
	helpBoxHeight := clamp(m.height*70/100, 10, max(10, m.height-2))
	m.helpVP.Width = max(20, helpBoxWidth-2)
	m.helpVP.Height = max(4, helpBoxHeight-2)
	if m.showHelp {
		m.refreshHelp()
	}
}

func (m model) searchView(width, height int) string {
	if m.focus == focusSearch {
		return m.search.View()
	}
	query := strings.TrimSpace(m.search.Value())
	if query == "" {
		query = "/"
	} else {
		query = fmt.Sprintf("/ %s", query)
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
		line := m.categoryLine(c)
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

func (m model) categoryLine(c categoryGroup) string {
	if c.Header {
		return fmt.Sprintf("%s", c.Category)
	}
	if c.Subcategory != "" {
		return fmt.Sprintf("  %-10s %d", c.Subcategory, len(c.Tabs))
	}
	return fmt.Sprintf("%-12s %d", c.Category, len(c.Tabs))
}

func (m model) tabsView(width, height int) string {
	cat := m.currentCategory()
	if cat == nil {
		return dimStyle.Render("No category selected")
	}
	if cat.Category == tool.CategoryJSON {
		return flowLabels(width, height, []string{"Workspace"}, 0)
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
		return labelRows(width, []string{"Workspace"})
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
		return statusLine("HELP",
			statusHint("type", "filter"),
			statusHint("ctrl+u/d", "scroll"),
			statusHint("ctrl+x", "clear"),
			statusHint("Esc/?", "close"),
		)
	}
	switch m.focus {
	case focusSearch:
		return statusLine("Search", statusHint("Esc", "close"), statusHint("Enter", "accept"))
	case focusCategories:
		return statusLine("Categories",
			statusHint("1", "search"),
			statusHint("2", "categories"),
			statusHint("3", "pages"),
			statusHint("j/k", "move"),
			statusHint("Enter", "pages"),
		)
	case focusTabs:
		if p := m.currentPage(); p != nil {
			if p.IsEditing() {
				return statusLine("Insert", statusHint("Esc", "normal"), statusHint("Tab", "next field"), statusHint("/", "text"))
			}
			return statusLine("Pages",
				statusHint("1", "search"),
				statusHint("2", "categories"),
				statusHint("3", "pages"),
				statusHint("4", "workspace"),
				statusHint("h/l", "select"),
				statusHint("r", "run"),
			)
		}
	case focusWorkspace:
		if p := m.currentPage(); p != nil {
			if p.IsEditing() {
				if hints := p.FooterHints(); hints != "" {
					return hints
				}
				return statusLine("Insert", statusHint("Esc", "normal"), statusHint("Tab", "field"), statusHint("/", "text"))
			}
			if hints := p.FooterHints(); hints != "" {
				return hints
			}
			return statusLine("Workspace",
				statusHint("1", "search"),
				statusHint("2", "categories"),
				statusHint("3", "pages"),
				statusHint("4", "workspace"),
				statusHint("i", "edit"),
				statusHint("v", "paste"),
				statusHint("x", "clear in"),
				statusHint("X", "clear out"),
				statusHint("r", "run"),
				statusHint("y", "copy"),
			)
		}
	}
	return m.status
}

func statusLine(scope string, hints ...string) string {
	parts := make([]string, 0, len(hints)+1)
	if scope != "" {
		parts = append(parts, appTitleStyle.Render(scope))
	}
	parts = append(parts, hints...)
	return strings.Join(parts, "  ")
}

func statusHint(key string, label string) string {
	if label == "" {
		return statusKeyStyle.Render("[" + key + "]")
	}
	return statusKeyStyle.Render("["+key+"]") + " " + statusTextStyle.Render(label)
}

func (m model) helpView() string {
	return m.helpVP.View()
}

func (m model) helpContent() string {
	sections := m.helpSections()
	query := strings.ToLower(strings.TrimSpace(m.help.Value()))

	var b strings.Builder
	b.WriteString(sectionLabel("Filter", true, true) + " ")
	b.WriteString(m.help.View() + "\n")
	b.WriteString(dimStyle.Render("Esc/? close | Ctrl+u/d scroll help | Ctrl+x clear filter") + "\n")
	b.WriteString(sectionRuleStyle.Render(strings.Repeat("-", 64)) + "\n")
	matched := 0
	for _, section := range sections {
		filtered := filterHelpCommands(section.Commands, query)
		if len(filtered) == 0 {
			continue
		}
		if matched > 0 {
			b.WriteString("\n")
		}
		b.WriteString(panelTitleStyle.Render(section.Title) + "\n")
		for _, cmd := range filtered {
			b.WriteString(renderHelpCommand(cmd, 64))
		}
		matched += len(filtered)
	}
	if matched == 0 {
		b.WriteString(dimStyle.Render("No matching commands") + "\n")
	}
	return b.String()
}

func (m model) helpSections() []helpSection {
	global := []helpCommand{
		{Key: "1", Command: "Search Area", Description: "Focus global category/tool search."},
		{Key: "2", Command: "Categories", Description: "Focus category list."},
		{Key: "3", Command: "Pages/Tools", Description: "Focus page or tool strip."},
		{Key: "4", Command: "Workspace", Description: "Focus current workspace."},
		{Key: "?", Command: "Help", Description: "Open or close this command help."},
		{Key: "q", Command: "Quit", Description: "Quit sharp from normal mode."},
		{Key: "Esc", Command: "Back", Description: "Leave edit mode, close overlays, or move focus back."},
	}
	context := make([]helpCommand, 0)
	switch m.focus {
	case focusCategories:
		context = append(context,
			helpCommand{Key: "j/down", Command: "Next Category", Description: "Move category selection down."},
			helpCommand{Key: "k/up", Command: "Previous Category", Description: "Move category selection up."},
			helpCommand{Key: "Enter", Command: "Open Pages", Description: "Move focus to area 3."},
		)
	case focusTabs:
		context = append(context,
			helpCommand{Key: "h/left", Command: "Previous", Description: "Select previous tool for non-workbench categories."},
			helpCommand{Key: "l/right", Command: "Next", Description: "Select next tool for non-workbench categories."},
			helpCommand{Key: "Enter/i", Command: "Edit Input", Description: "Focus workspace input editor."},
			helpCommand{Key: "o", Command: "Edit Options", Description: "Focus options or path editor when available."},
		)
	case focusWorkspace:
		context = append(context,
			helpCommand{Key: "i/Enter", Command: "Edit Input", Description: "Enter insert mode for input."},
			helpCommand{Key: "o", Command: "Edit Options", Description: "Edit options or JSON path."},
			helpCommand{Key: "Tab", Command: "Next Field", Description: "Cycle input, options/path, and output."},
			helpCommand{Key: "ctrl+u/d", Command: "Scroll Output", Description: "Scroll output up or down by half a page."},
			helpCommand{Key: "alt+u/d", Command: "Scroll Input", Description: "Scroll input up or down by half a page without changing focus."},
			helpCommand{Key: "Shift+Tab", Command: "Previous Field", Description: "Cycle fields backward."},
			helpCommand{Key: "v", Command: "Paste", Description: "Paste clipboard into input."},
			helpCommand{Key: "x", Command: "Clear Input", Description: "Clear the current input."},
			helpCommand{Key: "X", Command: "Clear Output", Description: "Clear the current output."},
			helpCommand{Key: "y", Command: "Copy Output", Description: "Copy trimmed output text."},
			helpCommand{Key: "r", Command: "Run", Description: "Run default action for this page."},
		)
	}
	sections := []helpSection{
		{Title: "Global", Commands: global},
	}
	if len(context) > 0 {
		sections = append(sections, helpSection{Title: "Current Area", Commands: context})
	}
	if p := m.currentPage(); p != nil {
		if commands := p.HelpCommands(); len(commands) > 0 {
			sections = append(sections, helpSection{Title: "Current Tool", Commands: commands})
		}
	}
	return sections
}

func filterHelpCommands(commands []helpCommand, query string) []helpCommand {
	filtered := make([]helpCommand, 0, len(commands))
	for _, cmd := range commands {
		haystack := strings.ToLower(cmd.Key + " " + cmd.Command + " " + cmd.Description)
		if query == "" || strings.Contains(haystack, query) {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

func renderHelpCommand(cmd helpCommand, width int) string {
	keyWidth := 12
	commandWidth := 18
	key := activeLabelStyle.Render(fitWidth(" "+cmd.Key+" ", keyWidth))
	command := fitWidth(cmd.Command, commandWidth)
	prefixWidth := keyWidth + 1 + commandWidth + 1
	descWidth := max(10, width-prefixWidth)
	desc := wrapDisplayText(cmd.Description, descWidth)
	descLines := strings.Split(desc, "\n")
	var b strings.Builder
	b.WriteString(key + " " + command + " " + descLines[0] + "\n")
	for _, line := range descLines[1:] {
		b.WriteString(strings.Repeat(" ", prefixWidth) + line + "\n")
	}
	return b.String()
}

func (m *model) openHelp() {
	m.showHelp = true
	m.help.Focus()
	m.refreshHelp()
}

func (m *model) refreshHelp() {
	m.helpVP.SetContent(m.helpContent())
	m.helpVP.GotoTop()
}

func panelBox(num int, title string, active bool, content string, width, height int) string {
	return panelBoxWithTitle(fmt.Sprintf("[%d]-%s", num, title), active, content, width, height)
}

func panelBoxTitle(title string, active bool, content string, width, height int) string {
	return panelBoxWithTitle(title, active, content, width, height)
}

func panelBoxWithTitle(titleText string, active bool, content string, width, height int) string {
	borderColor := lipgloss.Color("250")
	titleStyle := panelTitleStyle
	if active {
		borderColor = lipgloss.Color("39")
		titleStyle = appTitleStyle
	}
	// 手写 legend 风格边框：标题嵌入上边框，内容永远被裁剪/补齐到固定宽高。
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
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
	next := m.nextSelectableCategory(m.selectedCat, delta)
	if next == m.selectedCat {
		return
	}
	m.selectedCat = next
	m.restoreTabForSelectedCategory()
	m.rebuildPage()
	m.status = m.categoryStatus()
}

func (m *model) nextSelectableCategory(from int, delta int) int {
	if delta == 0 || len(m.categories) == 0 {
		return from
	}
	next := from
	for {
		next = clamp(next+delta, 0, len(m.categories)-1)
		if !m.categories[next].Header {
			return next
		}
		if next == 0 || next == len(m.categories)-1 {
			return from
		}
	}
}

func (m *model) moveTab(delta int) {
	cat := m.currentCategory()
	if cat == nil || len(cat.Tabs) == 0 {
		return
	}
	maxTab := len(cat.Tabs) - 1
	if cat.Category == tool.CategoryJSON {
		maxTab = 0
	}
	next := clamp(m.selectedTab+delta, 0, maxTab)
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
	m.selectedCat = m.nearestSelectableCategory(m.selectedCat)
	m.restoreTabForSelectedCategory()
	m.rebuildPage()
}

func (m *model) nearestSelectableCategory(idx int) int {
	if len(m.categories) == 0 {
		return 0
	}
	idx = clamp(idx, 0, len(m.categories)-1)
	if !m.categories[idx].Header {
		return idx
	}
	for i := idx + 1; i < len(m.categories); i++ {
		if !m.categories[i].Header {
			return i
		}
	}
	for i := idx - 1; i >= 0; i-- {
		if !m.categories[i].Header {
			return i
		}
	}
	return 0
}

func (m *model) focusSingleSearchResult() bool {
	query := strings.TrimSpace(m.search.Value())
	if query == "" {
		return false
	}
	matches := 0
	matchCat := 0
	matchTab := 0
	for ci, cat := range m.categories {
		if cat.Header {
			continue
		}
		for ti := range cat.Tabs {
			matches++
			matchCat = ci
			matchTab = ti
			if matches > 1 {
				return false
			}
		}
	}
	if matches != 1 {
		return false
	}
	m.selectedCat = matchCat
	m.setSelectedTab(matchTab)
	m.rebuildPage()
	return true
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
	m.selectedCat = m.nearestSelectableCategory(m.selectedCat)
	m.restoreTabForSelectedCategory()
}

func (m *model) restoreTabForSelectedCategory() {
	cat := m.currentCategory()
	if cat == nil || cat.Header || len(cat.Tabs) == 0 {
		m.selectedTab = 0
		return
	}
	maxTab := len(cat.Tabs) - 1
	if cat.Category == tool.CategoryJSON {
		maxTab = 0
	}
	if idx, ok := m.lastTabByCat[cat.Category]; ok {
		m.selectedTab = clamp(idx, 0, maxTab)
		return
	}
	if m.selectedTab > maxTab {
		m.selectedTab = maxTab
	}
}

func (m *model) setSelectedTab(idx int) {
	cat := m.currentCategory()
	if cat == nil || cat.Header {
		return
	}
	maxTab := len(cat.Tabs) - 1
	if cat.Category == tool.CategoryJSON {
		maxTab = 0
	}
	if idx < 0 || idx > maxTab {
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
	if cat == nil || cat.Header || len(cat.Tabs) == 0 {
		return nil
	}
	if cat.Category == tool.CategoryJSON {
		return &cat.Tabs[0]
	}
	if m.selectedTab < 0 || m.selectedTab >= len(cat.Tabs) {
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
	before := p.Status()
	p.SetInput(text)
	if p.Status() != before {
		m.status = p.Status()
	} else {
		m.status = successStyle.Render("pasted clipboard to input")
	}
}

func (m *model) categoryStatus() string {
	cat := m.currentCategory()
	if cat == nil {
		return "No category selected"
	}
	if cat.Subcategory != "" {
		return fmt.Sprintf("category %s/%s | %d tools", cat.Category, cat.Subcategory, len(cat.Tabs))
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
	flat := make([]categoryGroup, 0)
	if len(tools) == 0 {
		return flat
	}

	var current *categoryGroup
	for _, t := range tools {
		if current == nil || current.Category != t.Category() {
			flat = append(flat, categoryGroup{Category: t.Category()})
			current = &flat[len(flat)-1]
		}
		current.Tabs = append(current.Tabs, tabItem{Tool: t})
	}
	return expandCategoryRows(flat)
}

func filterCategories(all []categoryGroup, q string) []categoryGroup {
	if q == "" {
		return all
	}
	byCategory := make([]categoryGroup, 0, len(all))
	for _, c := range all {
		if c.Header {
			continue
		}
		filteredTabs := make([]tabItem, 0, len(c.Tabs))
		for _, t := range c.Tabs {
			haystack := strings.ToLower(t.Tool.ID() + " " + t.Tool.Name() + " " + string(t.Tool.Category()) + " " + toolSubcategory(t.Tool) + " " + t.Tool.Description())
			if strings.Contains(haystack, q) {
				filteredTabs = append(filteredTabs, t)
			}
		}
		if len(filteredTabs) == 0 {
			continue
		}
		byCategory = append(byCategory, categoryGroup{
			Category: c.Category,
			Tabs:     filteredTabs,
		})
	}
	return expandCategoryRows(byCategory)
}

func expandCategoryRows(categories []categoryGroup) []categoryGroup {
	out := make([]categoryGroup, 0, len(categories))
	for _, c := range categories {
		groups := groupedTabs(c.Tabs)
		if len(groups) <= 1 {
			subcategory := ""
			if len(groups) == 1 {
				subcategory = groups[0].Name
			}
			out = append(out, categoryGroup{Category: c.Category, Subcategory: subcategory, Tabs: c.Tabs})
			continue
		}
		out = append(out, categoryGroup{Category: c.Category, Header: true})
		for _, group := range groups {
			out = append(out, categoryGroup{Category: c.Category, Subcategory: group.Name, Tabs: group.Tabs})
		}
	}
	return out
}

func groupedTabs(tabs []tabItem) []toolSubgroup {
	groups := make([]toolSubgroup, 0)
	indexByName := make(map[string]int)
	for _, tab := range tabs {
		name := toolSubcategory(tab.Tool)
		idx, ok := indexByName[name]
		if !ok {
			groups = append(groups, toolSubgroup{Name: name})
			idx = len(groups) - 1
			indexByName[name] = idx
		}
		groups[idx].Tabs = append(groups[idx].Tabs, tab)
	}
	return groups
}

func toolSubcategory(t tool.Tool) string {
	if t == nil {
		return "tools"
	}
	id := t.ID()
	if prefix, _, ok := strings.Cut(id, "."); ok && prefix != "" {
		return prefix
	}
	words := strings.Fields(t.Name())
	if len(words) > 1 {
		return strings.ToLower(words[0])
	}
	return "tools"
}

type genericToolPage struct {
	tool        tool.Tool
	input       textarea.Model
	optionBox   textinput.Model
	output      viewport.Model
	outputText  string
	outputErr   bool
	options     tool.Options
	inputUndo   []string
	inputRedo   []string
	width       int
	inputBoxH   int
	actionsBoxH int
	optionsBoxH int
	outputBoxH  int
	focus       pageFocus
	editing     bool
	status      string
	hasOptions  bool
	hasInput    bool
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
		hasInput:   toolUsesInput(t),
	}
	p.loadSelectedOptions()
	return p
}

// SetSize recalculates the generic page layout and child component sizes.
func (p *genericToolPage) SetSize(width, height int) {
	// 内部各块都用固定边框，组件尺寸必须按边框内侧计算。
	p.width = max(18, width)
	p.actionsBoxH = 3
	p.optionsBoxH = 0
	if p.hasOptions {
		p.optionsBoxH = clamp(3+len(p.tool.Options()), 4, max(4, height/4))
	}
	p.inputBoxH = 0
	if p.hasInput {
		remaining := max(6, height-p.optionsBoxH-p.actionsBoxH)
		p.inputBoxH = max(5, remaining/2)
		p.outputBoxH = max(5, height-p.optionsBoxH-p.actionsBoxH-p.inputBoxH)
		if p.outputBoxH < 5 {
			p.outputBoxH = 5
			p.inputBoxH = max(5, height-p.optionsBoxH-p.actionsBoxH-p.outputBoxH)
		}
	} else {
		p.outputBoxH = max(5, height-p.optionsBoxH-p.actionsBoxH)
	}
	innerWidth := max(16, p.width-2)
	p.input.SetWidth(innerWidth)
	if p.hasInput {
		p.input.SetHeight(max(3, panelContentHeight(p.inputBoxH)))
	}
	p.optionBox.Width = max(16, innerWidth)
	p.output.Width = innerWidth
	p.output.Height = max(3, panelContentHeight(p.outputBoxH))
	p.renderOutput()
}

// Update forwards key events to the focused editor while preserving normal mode.
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
		switch msg.String() {
		case "ctrl+z":
			p.undoInput()
			return p, nil
		case "ctrl+y":
			p.redoInput()
			return p, nil
		}
		switch p.focus {
		case pageFocusInput:
			before := p.input.Value()
			var cmd tea.Cmd
			p.input, cmd = p.input.Update(msg)
			p.trackInputChange(before)
			return p, cmd
		case pageFocusOptions:
			var cmd tea.Cmd
			p.optionBox, cmd = p.optionBox.Update(msg)
			return p, cmd
		}
	}
	return p, nil
}

// View renders the generic Input/Options/Actions/Output workspace.
func (p *genericToolPage) View() string {
	blocks := make([]string, 0, 4)
	if p.hasInput {
		blocks = append(blocks, panelBoxTitle("Input", p.focus == pageFocusInput, p.input.View(), p.width, p.inputBoxH))
	}

	if p.hasOptions {
		var b strings.Builder
		if p.focus == pageFocusOptions || p.editing {
			b.WriteString(p.optionBox.View() + "\n")
		} else {
			b.WriteString(dimStyle.Render(p.optionBox.Value()) + "\n")
		}
		for _, opt := range p.tool.Options() {
			req := ""
			if opt.Required {
				req = " required"
			}
			line := fmt.Sprintf("--%s%s %s", opt.Name, req, opt.Description)
			b.WriteString(dimStyle.Render(line) + "\n")
		}
		blocks = append(blocks, panelBoxTitle("Options", p.focus == pageFocusOptions, b.String(), p.width, p.optionsBoxH))
	} else {
		// 无参数工具不再占一整块区域，把高度留给输入/输出。
	}

	blocks = append(blocks, panelBoxTitle("Actions", false, p.actionsView(), p.width, p.actionsBoxH))
	blocks = append(blocks, panelBoxTitle("Output", p.focus == pageFocusOutput, p.output.View(), p.width, p.outputBoxH))
	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

func (p *genericToolPage) actionsView() string {
	hints := []string{statusHint("r", "run"), statusHint("y", "copy"), statusHint("s", "save")}
	if p.hasInput {
		hints = append(hints, statusHint("v", "paste"), statusHint("x", "clear in"), statusHint("p", "pipe"))
	}
	hints = append(hints, statusHint("X", "clear out"))
	return strings.Join(hints, "  ")
}

// FocusInput enters insert mode for the input editor when this tool accepts input.
func (p *genericToolPage) FocusInput() {
	if !p.hasInput {
		if p.hasOptions {
			p.FocusOptions()
			return
		}
		p.FocusOutput()
		return
	}
	p.focus = pageFocusInput
	p.editing = true
	p.optionBox.Blur()
	p.input.Focus()
}

// FocusOptions enters insert mode for option editing when options are available.
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

// FocusOutput leaves insert mode and targets output scrolling.
func (p *genericToolPage) FocusOutput() {
	p.focus = pageFocusOutput
	p.stopEditing()
}

// FocusNext cycles forward through visible editable panes.
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
		if p.hasInput {
			p.FocusInput()
		} else if p.hasOptions {
			p.FocusOptions()
		}
	}
}

// FocusPrev cycles backward through visible editable panes.
func (p *genericToolPage) FocusPrev() {
	switch p.focus {
	case pageFocusInput:
		p.FocusOutput()
	case pageFocusOptions:
		if p.hasInput {
			p.FocusInput()
		} else {
			p.FocusOutput()
		}
	case pageFocusOutput:
		if p.hasOptions {
			p.FocusOptions()
		} else if p.hasInput {
			p.FocusInput()
		}
	}
}

// OutputHalfPageDown scrolls output down by half a page.
func (p *genericToolPage) OutputHalfPageDown() {
	p.output.HalfPageDown()
}

// OutputHalfPageUp scrolls output up by half a page.
func (p *genericToolPage) OutputHalfPageUp() {
	p.output.HalfPageUp()
}

// InputHalfPageDown scrolls input down by half a page when input is visible.
func (p *genericToolPage) InputHalfPageDown() {
	if !p.hasInput {
		return
	}
	textareaHalfPageDown(&p.input)
}

// InputHalfPageUp scrolls input up by half a page when input is visible.
func (p *genericToolPage) InputHalfPageUp() {
	if !p.hasInput {
		return
	}
	textareaHalfPageUp(&p.input)
}

// Run executes the selected tool with current input and parsed options.
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

// SetInput replaces input text and records undo history.
func (p *genericToolPage) SetInput(value string) {
	if !p.hasInput {
		p.status = dimStyle.Render("this tool has no input")
		return
	}
	p.pushInputUndo()
	p.input.SetValue(value)
	p.inputRedo = nil
}

// ClearInput clears input text and records undo history.
func (p *genericToolPage) ClearInput() {
	if !p.hasInput {
		p.status = dimStyle.Render("this tool has no input")
		return
	}
	p.pushInputUndo()
	p.input.SetValue("")
	p.inputRedo = nil
}

// ClearOutput clears raw and rendered output.
func (p *genericToolPage) ClearOutput() {
	p.outputText = ""
	p.outputErr = false
	p.output.SetContent("")
}

// OutputText returns trimmed raw output for copy, save, and pipe operations.
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

// Status returns the latest user-facing page status.
func (p *genericToolPage) Status() string {
	return p.status
}

// HelpText returns a compact help summary for legacy callers.
func (p *genericToolPage) HelpText() string {
	return fmt.Sprintf("%s | Esc back/exit edit | Tab field | r run | y copy | s save | p pipe", p.tool.Name())
}

// HelpCommands returns structured commands for the popup help overlay.
func (p *genericToolPage) HelpCommands() []helpCommand {
	commands := []helpCommand{
		{Key: "r", Command: "Run Tool", Description: "Run " + p.tool.Name() + " with current input and options."},
		{Key: "s", Command: "Save Output", Description: "Save trimmed output to sharp-output.txt."},
		{Key: "y", Command: "Copy Output", Description: "Copy trimmed output text."},
		{Key: "X", Command: "Clear Output", Description: "Clear the current output."},
	}
	if p.hasInput {
		commands = append(commands,
			helpCommand{Key: "i/Enter", Command: "Edit Input", Description: "Enter insert mode for the input editor."},
			helpCommand{Key: "v", Command: "Paste Input", Description: "Paste clipboard data into the input editor."},
			helpCommand{Key: "p", Command: "Pipe Output", Description: "Move output into input for another run."},
		)
	}
	if p.hasOptions {
		commands = append(commands, helpCommand{Key: "o", Command: "Edit Options", Description: "Edit option=value pairs for this tool."})
	}
	return commands
}

// FooterHints returns status bar shortcuts customized to the current tool shape.
func (p *genericToolPage) FooterHints() string {
	if !p.hasInput {
		return statusLine(p.tool.Name(),
			statusHint("r", "run"),
			statusHint("o", "options"),
			statusHint("y", "copy"),
			statusHint("s", "save"),
			statusHint("X", "clear out"),
			statusHint("?", "help"),
		)
	}
	return ""
}

// IsEditing reports whether key events should be routed to a text editor.
func (p *genericToolPage) IsEditing() bool {
	return p.editing
}

func (p *genericToolPage) trackInputChange(before string) {
	if !p.hasInput {
		return
	}
	after := p.input.Value()
	if after == before {
		return
	}
	p.pushInputUndoValue(before)
	p.inputRedo = nil
}

func (p *genericToolPage) pushInputUndo() {
	p.pushInputUndoValue(p.input.Value())
}

func (p *genericToolPage) pushInputUndoValue(value string) {
	if len(p.inputUndo) > 0 && p.inputUndo[len(p.inputUndo)-1] == value {
		return
	}
	p.inputUndo = append(p.inputUndo, value)
	if len(p.inputUndo) > 100 {
		p.inputUndo = p.inputUndo[len(p.inputUndo)-100:]
	}
}

func (p *genericToolPage) undoInput() {
	if len(p.inputUndo) == 0 {
		p.status = dimStyle.Render("nothing to undo")
		return
	}
	current := p.input.Value()
	last := p.inputUndo[len(p.inputUndo)-1]
	p.inputUndo = p.inputUndo[:len(p.inputUndo)-1]
	p.inputRedo = append(p.inputRedo, current)
	p.input.SetValue(last)
	p.status = successStyle.Render("undo input")
}

func (p *genericToolPage) redoInput() {
	if len(p.inputRedo) == 0 {
		p.status = dimStyle.Render("nothing to redo")
		return
	}
	current := p.input.Value()
	next := p.inputRedo[len(p.inputRedo)-1]
	p.inputRedo = p.inputRedo[:len(p.inputRedo)-1]
	p.pushInputUndoValue(current)
	p.input.SetValue(next)
	p.status = successStyle.Render("redo input")
}

// HandleKey handles page-specific normal-mode shortcuts.
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

func toolUsesInput(t tool.Tool) bool {
	switch t.ID() {
	case "time.now", "uuid.v4", "password", "token":
		return false
	default:
		return true
	}
}

type jsonToolPage struct {
	input      textarea.Model
	pathBox    textinput.Model
	output     viewport.Model
	tools      map[string]tool.Tool
	width      int
	inputBoxH  int
	pathBoxH   int
	outputBoxH int
	focus      pageFocus
	editing    bool
	inputUndo  []string
	inputRedo  []string
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

// SetSize recalculates the JSON workbench layout and child component sizes.
func (p *jsonToolPage) SetSize(width, height int) {
	// JSON 工作台固定保留一行 Path，其余高度在 Input/Output 之间分配。
	p.width = max(18, width)
	p.pathBoxH = 3
	remaining := max(6, height-p.pathBoxH)
	p.inputBoxH = max(5, remaining/2)
	p.outputBoxH = max(5, height-p.pathBoxH-p.inputBoxH)
	if p.outputBoxH < 5 {
		p.outputBoxH = 5
		p.inputBoxH = max(5, height-p.pathBoxH-p.outputBoxH)
	}
	innerWidth := max(16, p.width-2)
	p.input.SetWidth(innerWidth)
	p.input.SetHeight(max(3, panelContentHeight(p.inputBoxH)))
	p.pathBox.Width = innerWidth
	p.output.Width = innerWidth
	p.output.Height = max(3, panelContentHeight(p.outputBoxH))
	p.renderOutput()
}

// Update forwards key events to the active JSON workbench editor.
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
		switch msg.String() {
		case "ctrl+z":
			p.undoInput()
			return p, nil
		case "ctrl+y":
			p.redoInput()
			return p, nil
		}
		switch p.focus {
		case pageFocusInput:
			before := p.input.Value()
			var cmd tea.Cmd
			p.input, cmd = p.input.Update(msg)
			p.trackInputChange(before)
			return p, cmd
		case pageFocusOptions:
			if msg.String() == "enter" {
				p.runPathQuery()
				p.stopEditing()
				return p, nil
			}
			var cmd tea.Cmd
			p.pathBox, cmd = p.pathBox.Update(msg)
			return p, cmd
		}
	}
	return p, nil
}

// View renders the JSON workbench with input, path, and output panes.
func (p *jsonToolPage) View() string {
	inputBlock := panelBoxTitle("Input", p.focus == pageFocusInput, p.input.View(), p.width, p.inputBoxH)
	var path strings.Builder
	if p.focus == pageFocusOptions || p.editing {
		path.WriteString(p.pathBox.View())
	} else {
		value := p.pathBox.Value()
		if value == "" {
			value = "gjson path"
		}
		path.WriteString(dimStyle.Render(value))
	}
	pathBlock := panelBoxTitle("Path", p.focus == pageFocusOptions, path.String(), p.width, p.pathBoxH)
	var output strings.Builder
	if p.lastAction != "" {
		output.WriteString(dimStyle.Render("Last: "+p.lastAction+" | actions read Input; [a] promotes Output to Input") + "\n")
	}
	output.WriteString(p.output.View())
	outputBlock := panelBoxTitle("Output", p.focus == pageFocusOutput, output.String(), p.width, p.outputBoxH)
	return lipgloss.JoinVertical(lipgloss.Left, inputBlock, pathBlock, outputBlock)
}

// FocusInput enters insert mode for the JSON input editor.
func (p *jsonToolPage) FocusInput() {
	p.focus = pageFocusInput
	p.editing = true
	p.pathBox.Blur()
	p.input.Focus()
}

// FocusOptions enters insert mode for the JSON path editor.
func (p *jsonToolPage) FocusOptions() {
	p.focus = pageFocusOptions
	p.editing = true
	p.input.Blur()
	p.pathBox.Focus()
}

// FocusOutput leaves insert mode and targets output scrolling.
func (p *jsonToolPage) FocusOutput() {
	p.focus = pageFocusOutput
	p.stopEditing()
}

// FocusNext cycles forward through JSON input, path, and output panes.
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

// FocusPrev cycles backward through JSON input, path, and output panes.
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

// OutputHalfPageDown scrolls output down by half a page.
func (p *jsonToolPage) OutputHalfPageDown() {
	p.output.HalfPageDown()
}

// OutputHalfPageUp scrolls output up by half a page.
func (p *jsonToolPage) OutputHalfPageUp() {
	p.output.HalfPageUp()
}

// InputHalfPageDown scrolls JSON input down by half a page.
func (p *jsonToolPage) InputHalfPageDown() {
	textareaHalfPageDown(&p.input)
}

// InputHalfPageUp scrolls JSON input up by half a page.
func (p *jsonToolPage) InputHalfPageUp() {
	textareaHalfPageUp(&p.input)
}

// Run executes the default JSON action, currently pretty printing.
func (p *jsonToolPage) Run() {
	p.runAction("json.pretty", nil)
}

// ClearInput clears JSON input and records undo history.
func (p *jsonToolPage) ClearInput() {
	p.pushInputUndo()
	p.input.SetValue("")
	p.inputRedo = nil
}

// ClearOutput clears raw and rendered output.
func (p *jsonToolPage) ClearOutput() {
	p.outputText = ""
	p.outputErr = false
	p.output.SetContent("")
}

// SetInput replaces JSON input and records undo history.
func (p *jsonToolPage) SetInput(value string) {
	p.pushInputUndo()
	p.input.SetValue(value)
	p.inputRedo = nil
}

// OutputText returns trimmed raw output for copy, save, and apply operations.
func (p *jsonToolPage) OutputText() string {
	return strings.TrimSpace(p.outputText)
}

// Status returns the latest user-facing JSON workbench status.
func (p *jsonToolPage) Status() string {
	return p.status
}

// HelpText returns a compact help summary for legacy callers.
func (p *jsonToolPage) HelpText() string {
	return "JSON Workspace | i edit | v paste | o path | p pretty | m minify | c check | s sort | e escape | u unescape | g get | a apply | z undo"
}

// FooterHints returns JSON-specific status bar shortcuts.
func (p *jsonToolPage) FooterHints() string {
	if p.IsEditing() {
		if p.focus == pageFocusOptions {
			return statusLine("INSERT Path", statusHint("Enter", "query"), statusHint("Esc", "normal"), statusHint("Tab", "next field"))
		}
		return statusLine("INSERT Input", statusHint("Esc", "normal"), statusHint("Tab", "next field"))
	}
	return statusLine("JSON",
		statusHint("i", "edit"),
		statusHint("o", "path"),
		statusHint("v", "paste"),
		statusHint("p", "pretty"),
		statusHint("m", "minify"),
		statusHint("c", "check"),
		statusHint("e", "escape"),
		statusHint("g", "get"),
		statusHint("a", "apply"),
		statusHint("ctrl+u/d", "output"),
		statusHint("alt+u/d", "input"),
		statusHint("?", "help"),
	)
}

// HelpCommands returns structured JSON workbench commands for the help overlay.
func (p *jsonToolPage) HelpCommands() []helpCommand {
	return []helpCommand{
		{Key: "i/Enter", Command: "Edit JSON Input", Description: "Enter insert mode for the JSON input editor."},
		{Key: "o", Command: "Edit Path", Description: "Edit gjson path; press Enter in the Path field to query immediately."},
		{Key: "v", Command: "Paste Input", Description: "Paste clipboard data into the input editor."},
		{Key: "ctrl+z", Command: "Undo Input", Description: "Undo the last input edit while editing the input field."},
		{Key: "ctrl+y", Command: "Redo Input", Description: "Redo the last undone input edit while editing the input field."},
		{Key: "p", Command: "Pretty JSON", Description: "Format Input with two-space indentation into Output."},
		{Key: "m", Command: "Minify JSON", Description: "Compact Input into one-line JSON in Output."},
		{Key: "c", Command: "Check JSON", Description: "Validate Input JSON syntax."},
		{Key: "s", Command: "Sort Keys", Description: "Recursively sort JSON object keys into Output."},
		{Key: "e", Command: "Escape String", Description: "Encode Input as a JSON string literal."},
		{Key: "u", Command: "Unescape String", Description: "Decode a JSON string literal from Input."},
		{Key: "g", Command: "Get Path", Description: "Query Input with the current Path value. If Path is empty, focus the Path field first."},
		{Key: "a", Command: "Apply Output", Description: "Promote Output into Input for the next action."},
		{Key: "z", Command: "Undo Input", Description: "Restore previous Input after apply, paste, or clear."},
	}
}

// IsEditing reports whether key events should be routed to a text editor.
func (p *jsonToolPage) IsEditing() bool {
	return p.editing
}

// HandleKey handles JSON-specific normal-mode action shortcuts.
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
		p.runPathQuery()
	case "a":
		p.applyOutput()
	case "z":
		p.undoInput()
	default:
		return false
	}
	return true
}

func (p *jsonToolPage) runPathQuery() {
	path := strings.TrimSpace(p.pathBox.Value())
	if path == "" {
		p.FocusOptions()
		p.status = dimStyle.Render("enter path, then press Enter")
		return
	}
	p.runAction("json.get", tool.Options{"path": path})
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
	p.pushInputUndo()
	p.input.SetValue(text)
	p.inputRedo = nil
	p.ClearOutput()
	p.status = successStyle.Render("applied output to input")
}

func (p *jsonToolPage) undoInput() {
	if len(p.inputUndo) == 0 {
		p.status = dimStyle.Render("nothing to undo")
		return
	}
	current := p.input.Value()
	last := p.inputUndo[len(p.inputUndo)-1]
	p.inputUndo = p.inputUndo[:len(p.inputUndo)-1]
	p.inputRedo = append(p.inputRedo, current)
	p.input.SetValue(last)
	p.status = successStyle.Render("undo input")
}

func (p *jsonToolPage) redoInput() {
	if len(p.inputRedo) == 0 {
		p.status = dimStyle.Render("nothing to redo")
		return
	}
	current := p.input.Value()
	next := p.inputRedo[len(p.inputRedo)-1]
	p.inputRedo = p.inputRedo[:len(p.inputRedo)-1]
	p.pushInputUndoValue(current)
	p.input.SetValue(next)
	p.status = successStyle.Render("redo input")
}

func (p *jsonToolPage) trackInputChange(before string) {
	after := p.input.Value()
	if after == before {
		return
	}
	p.pushInputUndoValue(before)
	p.inputRedo = nil
}

func (p *jsonToolPage) pushInputUndo() {
	p.pushInputUndoValue(p.input.Value())
}

func (p *jsonToolPage) pushInputUndoValue(value string) {
	if len(p.inputUndo) > 0 && p.inputUndo[len(p.inputUndo)-1] == value {
		return
	}
	p.inputUndo = append(p.inputUndo, value)
	if len(p.inputUndo) > 100 {
		p.inputUndo = p.inputUndo[len(p.inputUndo)-100:]
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

func textareaHalfPageDown(t *textarea.Model) {
	steps := max(1, t.Height()/2)
	for i := 0; i < steps; i++ {
		t.CursorDown()
	}
	// 让 textarea 根据新光标位置刷新内部 viewport。
	updated, _ := t.Update(tea.KeyMsg{})
	*t = updated
}

func textareaHalfPageUp(t *textarea.Model) {
	steps := max(1, t.Height()/2)
	for i := 0; i < steps; i++ {
		t.CursorUp()
	}
	updated, _ := t.Update(tea.KeyMsg{})
	*t = updated
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
