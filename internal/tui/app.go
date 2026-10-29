package tui

import (
	"fmt"
	"os"
	"regexp"
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
	OutputHalfPageDown()
	OutputHalfPageUp()
	InputHalfPageDown()
	InputHalfPageUp()
	Run()
	SelectTool(tool.Tool)
	Preview()
	SetActive(bool)
	ApplyOutput()
	ClearInput()
	ClearOutput()
	SetInput(string)
	OutputText() string
	previewVersion() uint64
	OutputReady() bool
	Status() string
	HelpText() string
	HelpCommands() []helpCommand
	FooterHints() string
	IsEditing() bool
	HandleKey(string) bool
}

type model struct {
	registry      *tool.Registry
	readClipboard func() (string, error)

	allCategories []categoryGroup
	categories    []categoryGroup
	selectedCat   int
	selectedTab   int
	lastTabByCat  map[string]string

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
	search.Placeholder = "Search tools"
	search.Prompt = ""
	search.Blur()
	help := textinput.New()
	help.Placeholder = "Filter commands"
	help.Blur()

	m := model{
		registry:      registry,
		readClipboard: clipboard.Read,
		lastTabByCat:  make(map[string]string),
		pages:         make(map[string]toolPage),
		search:        search,
		help:          help,
		helpVP:        viewport.New(80, 20),
		status:        "Type to search · ↑/↓ select · Enter open · Ctrl+C quit",
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
	case workspacePreviewMsg:
		if msg.revision != msg.page.previewVersion() {
			return m, nil
		}
		// 定时消息始终交回原工作台，不受搜索、帮助或当前区域影响。
		msg.page.Update(msg)
		if p := m.currentPage(); p == msg.page && !m.showHelp && m.focus != focusSearch {
			m.status = p.Status()
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" || key == "ctrl+q" {
			return m, tea.Quit
		}
		if m.showHelp {
			switch key {
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
			return m.updateCategorySearch(msg)
		}
		if p := m.currentPage(); m.focus == focusWorkspace && p != nil && p.IsEditing() {
			updated, cmd := p.Update(msg)
			m.setCurrentPage(updated)
			m.status = updated.Status()
			return m, cmd
		}
		// 分类区域的字母优先用于搜索，j/k/q 也不能先被全局命令消费。
		if m.focus == focusCategories && msg.Type == tea.KeyRunes && !msg.Alt && key != "/" && key != "?" && key != "1" && key != "2" && key != "3" && key != "[" && key != "]" {
			return m.startCategorySearch(msg)
		}
		switch key {
		case "tab":
			m.focus = (m.focus + 1) % 3
			return m, nil
		case "shift+tab":
			m.focus = (m.focus + 2) % 3
			return m, nil
		case "[", "]":
			delta := 1
			if key == "[" {
				delta = -1
			}
			m.moveTab(delta)
			return m, nil
		case "q":
			return m, tea.Quit
		case "1":
			m.focus = focusCategories
			return m, nil
		case "2":
			m.focus = focusTabs
			return m, nil
		case "3":
			m.focus = focusWorkspace
			return m, nil
		case "/":
			return m.startCategorySearch(tea.KeyMsg{})
		case "?":
			m.openHelp()
			return m, nil
		}
		switch m.focus {
		case focusCategories:
			switch key {
			case "down":
				m.moveCategory(1)
			case "up":
				m.moveCategory(-1)
			case "esc":
				m.clearCategorySearch()
			case "enter", "right":
				m.openSelection()
			}
		case focusTabs:
			switch key {
			case "l", "right":
				m.moveTab(1)
			case "h", "left":
				m.moveTab(-1)
			case "esc":
				m.focus = focusCategories
			case "enter":
				m.focus = focusWorkspace
				m.runSelected()
			default:
				return m.updateWorkspaceKey(msg)
			}
		case focusWorkspace:
			switch key {
			case "esc":
				m.focus = focusTabs
			default:
				return m.updateWorkspaceKey(msg)
			}
		}
	default:
		if m.showHelp {
			before := m.help.Value()
			var cmd tea.Cmd
			m.help, cmd = m.help.Update(msg)
			if m.help.Value() != before {
				m.refreshHelp()
			}
			return m, cmd
		}
		if m.focus == focusSearch {
			before := m.search.Value()
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			if m.search.Value() != before {
				m.applySearch()
			}
			return m, cmd
		}
		if p := m.currentPage(); m.focus == focusWorkspace && p != nil && p.IsEditing() {
			updated, cmd := p.Update(msg)
			m.setCurrentPage(updated)
			return m, cmd
		}
	}
	return m, nil
}

// updateWorkspaceKey 是公共动作的唯一入口，页面只处理自己的编辑与领域动作。
func (m model) updateWorkspaceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.currentPage()
	if p == nil {
		return m, nil
	}
	switch msg.String() {
	case "i":
		m.focus = focusWorkspace
		p.HandleKey("i")
		m.status = p.Status()
		return m, textarea.Blink
	case "o":
		m.focus = focusWorkspace
		p.HandleKey("o")
		m.status = p.Status()
		return m, textarea.Blink
	case "enter", "r":
		p.Run()
	case "v":
		m.pasteInput()
		return m, nil
	case "x":
		p.ClearInput()
		p.Preview()
	case "X":
		p.ClearOutput()
		m.status = successStyle.Render("cleared output")
		return m, nil
	case "y":
		m.copyOutput()
		return m, nil
	case "s":
		m.saveOutput()
		return m, nil
	case "p", "a":
		p.ApplyOutput()
		m.focus = focusWorkspace
	case "ctrl+d":
		p.OutputHalfPageDown()
	case "ctrl+u":
		p.OutputHalfPageUp()
	case "alt+d":
		p.InputHalfPageDown()
	case "alt+u":
		p.InputHalfPageUp()
	default:
		if !p.HandleKey(msg.String()) {
			updated, cmd := p.Update(msg)
			m.setCurrentPage(updated)
			m.status = updated.Status()
			return m, cmd
		}
	}
	m.status = p.Status()
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "sharp"
	}

	// 左侧保持 20% 宽度，右侧吃掉剩余空间；上下边框手动固定，避免内容反向撑开布局。
	leftWidth := clamp(m.width*20/100, 22, max(22, m.width-60))
	rightWidth := max(40, m.width-leftWidth)
	bodyHeight := max(10, m.height-2)

	categoryHeight := bodyHeight
	toolHeight := clamp(m.toolRows(rightWidth-2)+2, 3, max(3, bodyHeight-8))
	workspaceHeight := max(6, bodyHeight-toolHeight)

	categories := panelBox(1, "Categories", m.focus == focusCategories || m.focus == focusSearch, m.categoryView(leftWidth-2, panelContentHeight(categoryHeight)), leftWidth, categoryHeight)
	left := categories
	tabs := panelBox(2, m.tabsTitle(), m.focus == focusTabs, m.tabsView(rightWidth-2, panelContentHeight(toolHeight)), rightWidth, toolHeight)
	page := panelBox(3, "Workspace", m.focus == focusWorkspace, m.pageView(rightWidth-2, panelContentHeight(workspaceHeight)), rightWidth, workspaceHeight)
	right := lipgloss.JoinVertical(lipgloss.Left, tabs, page)
	footer := fitWidth(footerStyle.Render(m.footerLine()), m.width)
	notification := fitWidth(m.status, m.width)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	view := body + "\n" + notification + "\n" + footer
	if m.showHelp {
		view = overlayPanel(view, m.helpView(), m.width, m.height)
	}
	lines := strings.Split(fitHeight(view, m.height), "\n")
	for i, line := range lines {
		lines[i] = fitWidth(line, m.width)
	}
	return strings.Join(lines, "\n")
}

func (m *model) resize() {
	if p := m.currentPage(); p != nil {
		leftWidth := clamp(m.width*20/100, 22, max(22, m.width-60))
		rightWidth := max(40, m.width-leftWidth)
		bodyHeight := max(10, m.height-2)
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
		return "/ " + m.search.View()
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
	if m.focus == focusSearch || m.search.Value() != "" {
		b.WriteString(m.searchView(width, 1) + "\n")
		visible = max(1, visible-1)
	}
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
	if c.Subcategory != "" && m.categoryHasHeader(c.Category) {
		return fmt.Sprintf("  %-10s %d", c.Subcategory, len(c.Tabs))
	}
	return fmt.Sprintf("%-12s %d", c.Category, len(c.Tabs))
}

func (m model) tabsView(width, height int) string {
	cat := m.currentCategory()
	if cat == nil {
		return dimStyle.Render("No category selected")
	}

	labels := make([]string, 0, len(cat.Tabs))
	for _, tab := range cat.Tabs {
		labels = append(labels, actionLabel(tab.Tool))
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
	labels := make([]string, 0, len(cat.Tabs))
	for _, tab := range cat.Tabs {
		labels = append(labels, actionLabel(tab.Tool))
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
		name := workspaceGroupTitle(tab.Tool) + " · " + actionLabel(tab.Tool)
		b.WriteString(appTitleStyle.Render(name) + "\n")
		b.WriteString(dimStyle.Render(tab.Tool.Description()) + "\n")
		b.WriteString(sectionRuleStyle.Render(strings.Repeat("-", max(8, width))) + "\n")
	}
	p.SetActive(m.focus == focusWorkspace)
	b.WriteString(p.View())
	return b.String()
}

func (m model) tabsTitle() string {
	if tab := m.currentTab(); tab != nil {
		return workspaceGroupTitle(tab.Tool) + " Actions"
	}
	return "Actions"
}

func (m model) footerLine() string {
	if m.showHelp {
		return statusLine("Help", statusHint("type", "filter"), statusHint("ctrl+u/d", "scroll"), statusHint("Esc", "close"))
	}
	switch m.focus {
	case focusSearch:
		return statusLine("Filter", statusHint("↑/↓", "select"), statusHint("Enter", "open"), statusHint("Esc", "clear"))
	case focusCategories:
		return statusLine("Categories", statusHint("type", "search"), statusHint("↑/↓", "select"), statusHint("Tab", "area"), statusHint("?", "help"))
	case focusTabs:
		return statusLine("Actions", statusHint("h/l or ←/→", "select"), statusHint("Tab", "area"), statusHint("Enter", "run"), statusHint("?", "help"))
	case focusWorkspace:
		if p := m.currentPage(); p != nil {
			if hints := p.FooterHints(); hints != "" {
				return hints
			}
			return statusLine("Workspace", statusHint("r/Enter", "run"), statusHint("i", "edit"), statusHint("v", "paste"), statusHint("y", "copy"), statusHint("?", "help"))
		}
	}
	return ""
}

func statusLine(scope string, hints ...string) string {
	parts := make([]string, 0, len(hints)+1)
	if scope != "" {
		parts = append(parts, appTitleStyle.Render(scope))
	}
	parts = append(parts, hints...)
	// Keep contextual shortcuts on one line even on an 80-column terminal.
	return strings.Join(parts, " ")
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
		{Key: "Tab/Shift+Tab", Command: "Change Area", Description: "Cycle Categories, Actions and Workspace in Normal mode; Esc leaves editing first."},
		{Key: "[ / ]", Command: "Change Action", Description: "Select the previous/next action in the current group without changing area."},
		{Key: "/", Command: "Search Tools", Description: "Filter tools inside Categories; typing there also starts search."},
		{Key: "1", Command: "Categories", Description: "Focus category list; type to search, arrows to navigate."},
		{Key: "2", Command: "Tools/Actions", Description: "Focus the tool or JSON action strip."},
		{Key: "3", Command: "Workspace", Description: "Focus current workspace."},
		{Key: "?", Command: "Help", Description: "Open or close this command help."},
		{Key: "ctrl+c/ctrl+q", Command: "Quit", Description: "Quit from any area. q also quits outside Categories and editors."},
		{Key: "Esc", Command: "Back", Description: "Leave edit mode, close overlays, or move focus back."},
	}
	context := make([]helpCommand, 0)
	switch m.focus {
	case focusCategories:
		context = append(context,
			helpCommand{Key: "down", Command: "Next Category", Description: "Move category selection down."},
			helpCommand{Key: "up", Command: "Previous Category", Description: "Move category selection up."},
			helpCommand{Key: "Enter", Command: "Open Pages", Description: "Open tools/actions; a single tool opens directly."},
		)
	case focusTabs:
		context = append(context,
			helpCommand{Key: "h/left", Command: "Previous", Description: "Select previous tool or JSON action."},
			helpCommand{Key: "l/right", Command: "Next", Description: "Select next tool or JSON action."},
			helpCommand{Key: "Enter", Command: "Run Action", Description: "Run the selected action and enter the workspace."},
			helpCommand{Key: "o", Command: "Edit Options", Description: "Focus options or path editor when available."},
		)
	case focusWorkspace:
		context = append(context,
			helpCommand{Key: "i", Command: "Edit Input", Description: "Edit Input directly; Esc returns to Normal without changing area."},
			helpCommand{Key: "o", Command: "Edit Options", Description: "Edit options or JSON path."},
			helpCommand{Key: "ctrl+u/d", Command: "Scroll Output", Description: "Scroll output up or down by half a page."},
			helpCommand{Key: "alt+u/d", Command: "Scroll Input", Description: "Scroll input up or down by half a page without changing focus."},
			helpCommand{Key: "v", Command: "Paste", Description: "Paste clipboard into Input and preview local transformations in Normal mode."},
			helpCommand{Key: "x", Command: "Clear Input", Description: "Clear the current input."},
			helpCommand{Key: "X", Command: "Clear Output", Description: "Clear the current output."},
			helpCommand{Key: "y", Command: "Copy Output", Description: "Copy trimmed output text."},
			helpCommand{Key: "r/Enter", Command: "Run", Description: "Run the selected tool or JSON action."},
			helpCommand{Key: "p/a", Command: "Apply Output", Description: "Apply data output to Input."},
			helpCommand{Key: "s", Command: "Save Output", Description: "Save output to sharp-output.txt."},
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
	next := clamp(m.selectedTab+delta, 0, maxTab)
	if next == m.selectedTab {
		return
	}
	m.setSelectedTab(next)
	m.rebuildPage()
	if p := m.currentPage(); p != nil {
		p.Preview()
		m.status = p.Status()
	}
}

func (m *model) applySearch() {
	id := ""
	if tab := m.currentTab(); tab != nil {
		id = tab.Tool.ID()
	}
	q := strings.ToLower(strings.TrimSpace(m.search.Value()))
	m.categories = filterCategories(m.allCategories, q)
	m.selectedCat = 0
	m.selectedTab = 0
	if len(m.categories) == 0 {
		return
	}
	m.selectedCat = m.nearestSelectableCategory(0)
	for ci, cat := range m.categories {
		if cat.Header {
			continue
		}
		for ti, tab := range cat.Tabs {
			if tab.Tool.ID() == id {
				m.selectedCat = ci
				m.selectedTab = ti
				m.rebuildPage()
				return
			}
		}
	}
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
	m.selectedTab = 0
	if cat == nil || cat.Header {
		return
	}
	// 筛选会改变下标，使用工具 ID 记忆每个子类目的选择。
	id := m.lastTabByCat[string(cat.Category)+":"+cat.Subcategory]
	for i, tab := range cat.Tabs {
		if tab.Tool.ID() == id {
			m.selectedTab = i
			return
		}
	}
}

func (m *model) setSelectedTab(idx int) {
	cat := m.currentCategory()
	if cat == nil || cat.Header {
		return
	}
	maxTab := len(cat.Tabs) - 1
	if idx < 0 || idx > maxTab {
		return
	}
	m.selectedTab = idx
	m.lastTabByCat[string(cat.Category)+":"+cat.Subcategory] = cat.Tabs[idx].Tool.ID()
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
		if m.selectedTab < 0 || m.selectedTab >= len(cat.Tabs) {
			m.selectedTab = 0
		}
		return &cat.Tabs[m.selectedTab]
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
	id := workspaceKey(tab.Tool)
	if p, ok := m.pages[id]; ok {
		p.SelectTool(tab.Tool)
		return p
	}
	p := m.newToolPage(tab.Tool)
	m.pages[id] = p
	return p
}

func (m *model) setCurrentPage(p toolPage) {
	tab := m.currentTab()
	if tab != nil && p != nil {
		m.pages[workspaceKey(tab.Tool)] = p
	}
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
	if !p.OutputReady() {
		m.status = errorStyle.Render("Output is stale; update Input/Options or run again")
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
	if !p.OutputReady() {
		m.status = errorStyle.Render("Output is stale; update Input/Options or run again")
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

func (m *model) pasteInput() {
	p := m.currentPage()
	if p == nil {
		return
	}
	text, err := m.readClipboard()
	if err != nil {
		m.status = errorStyle.Render("paste failed: " + err.Error())
		return
	}
	p.SetInput(text)
	p.Preview()
	m.focus = focusWorkspace
	m.status = p.Status()
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
	for i := range flat {
		orderJSONActions(flat[i].Tabs)
	}
	return expandCategoryRows(flat)
}

func filterCategories(all []categoryGroup, q string) []categoryGroup {
	if q == "" {
		return all
	}
	byCategory := make([]categoryGroup, 0, len(all))
	indexByCategory := map[tool.Category]int{}
	for _, c := range all {
		if c.Header {
			continue
		}
		filteredTabs := make([]tabItem, 0, len(c.Tabs))
		for _, t := range c.Tabs {
			haystack := strings.ToLower(t.Tool.ID() + " " + t.Tool.Name() + " " + string(t.Tool.Category()) + " " + toolSubcategory(t.Tool) + " " + t.Tool.Description() + " " + actionLabel(t.Tool) + " " + workspaceGroupTitle(t.Tool))
			if fuzzyMatch(haystack, q) {
				filteredTabs = append(filteredTabs, t)
			}
		}
		if len(filteredTabs) == 0 {
			continue
		}
		idx, exists := indexByCategory[c.Category]
		if !exists {
			idx = len(byCategory)
			indexByCategory[c.Category] = idx
			byCategory = append(byCategory, categoryGroup{Category: c.Category})
		}
		byCategory[idx].Tabs = append(byCategory[idx].Tabs, filteredTabs...)
	}
	for i := range byCategory {
		orderJSONActions(byCategory[i].Tabs)
	}
	return expandCategoryRows(byCategory)
}

func fuzzyMatch(haystack string, query string) bool {
	haystack = strings.ToLower(haystack)
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	if strings.Contains(haystack, query) {
		return true
	}
	words := strings.FieldsFunc(haystack, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	compactQuery := strings.ReplaceAll(query, " ", "")
	if acronymMatch(words, compactQuery) {
		return true
	}
	for _, part := range strings.Fields(query) {
		if !wordPrefixMatch(words, part) {
			return false
		}
	}
	return len(strings.Fields(query)) > 0
}

func acronymMatch(words []string, query string) bool {
	if query == "" {
		return false
	}
	var acronym strings.Builder
	for _, word := range words {
		if word == "" {
			continue
		}
		acronym.WriteByte(word[0])
	}
	return strings.Contains(acronym.String(), query)
}

func wordPrefixMatch(words []string, query string) bool {
	if query == "" {
		return true
	}
	for _, word := range words {
		if strings.HasPrefix(word, query) {
			return true
		}
	}
	return false
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

func parseOptionText(text string, specs []tool.Option) (tool.Options, error) {
	options := tool.Options{}
	allowed := make(map[string]bool, len(specs))
	for _, opt := range specs {
		allowed[opt.Name] = true
		if opt.Default != "" {
			options[opt.Name] = opt.Default
		}
	}

	// TUI 使用轻量 key=value 语法；带空格的值必须加引号，例如 replacement="hello world"。
	for _, part := range optionPattern.FindAllString(text, -1) {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid option %q, expected key=value", part)
		}
		k = strings.TrimSpace(k)
		if k == "" {
			return nil, fmt.Errorf("option name cannot be empty")
		}
		if !allowed[k] {
			return nil, fmt.Errorf("unknown option: %s", k)
		}
		options[k] = unquoteOptionValue(strings.TrimSpace(v))
	}

	// 和 CLI 保持一致：必填项缺失时阻止运行工具，把错误显示在输出区域。
	for _, opt := range specs {
		if opt.Required && strings.TrimSpace(options[opt.Name]) == "" {
			return nil, fmt.Errorf("missing required option: %s", opt.Name)
		}
	}
	return options, nil
}

var optionPattern = regexp.MustCompile(`\S+="[^"]*"|\S+='[^']*'|\S+`)

func unquoteOptionValue(value string) string {
	if len(value) < 2 {
		return value
	}
	if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

func hasSecretOptions(options []tool.Option) bool {
	for _, opt := range options {
		if opt.Secret {
			return true
		}
	}
	return false
}

func toolUsesInput(t tool.Tool) bool {
	switch t.ID() {
	case "time.now", "uuid.v4", "password", "token":
		return false
	default:
		return true
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
