package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"

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
	focusSearch
	focusPage
)

type pageBackMsg struct{}

type categoryGroup struct {
	Category tool.Category
	Hotkey   rune
	Tabs     []tabItem
}

type tabItem struct {
	Tool   tool.Tool
	Hotkey rune
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
	SetInput(string)
	OutputText() string
	Status() string
	HelpText() string
}

type model struct {
	registry *tool.Registry

	allCategories []categoryGroup
	categories    []categoryGroup
	selectedCat   int
	selectedTab   int
	lastTabByCat  map[tool.Category]int

	focus focus

	search textinput.Model
	page   toolPage
	pages  map[string]toolPage

	width  int
	height int
	status string
}

var (
	appBorderStyle   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
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
		status:       "j/k categories | [ ] tabs | letters jump | Enter run | / search | y copy | s save | p pipe | q quit",
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
	case pageBackMsg:
		m.focus = focusTabs
		m.status = dimStyle.Render("back to tabs")
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		switch m.focus {
		case focusSearch:
			if msg.String() == "esc" || msg.String() == "enter" {
				m.focus = focusCategories
				m.search.Blur()
				m.applySearch()
				return m, nil
			}
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			m.applySearch()
			return m, cmd

		case focusCategories:
			switch msg.String() {
			case "/":
				m.focus = focusSearch
				m.search.Focus()
			case "tab":
				m.focus = focusTabs
			case "shift+tab":
				m.focus = focusPage
				m.currentPage().FocusOutput()
			case "j", "down":
				m.moveCategory(1)
			case "k", "up":
				m.moveCategory(-1)
			case "enter":
				m.focus = focusTabs
			case "i":
				m.focus = focusPage
				m.currentPage().FocusInput()
			case "o":
				m.focus = focusPage
				m.currentPage().FocusOptions()
			case "y":
				m.copyOutput()
			case "s":
				m.saveOutput()
			case "p":
				m.pipeOutput()
			case "?":
				if p := m.currentPage(); p != nil {
					m.status = p.HelpText()
				}
			default:
				if key, ok := hotkeyFromMsg(msg); ok && m.jumpCategoryByKey(key) {
					m.status = m.categoryStatus()
				}
			}

		case focusTabs:
			switch msg.String() {
			case "/":
				m.focus = focusSearch
				m.search.Focus()
			case "tab":
				m.focus = focusPage
				m.currentPage().FocusInput()
			case "shift+tab":
				m.focus = focusCategories
			case "j", "down":
				m.moveTab(1)
			case "k", "up":
				m.moveTab(-1)
			case "[":
				m.moveTab(-1)
			case "]":
				m.moveTab(1)
			case "enter", "r":
				m.runSelected()
			case "i":
				m.focus = focusPage
				m.currentPage().FocusInput()
			case "o":
				m.focus = focusPage
				m.currentPage().FocusOptions()
			case "y":
				m.copyOutput()
			case "s":
				m.saveOutput()
			case "p":
				m.pipeOutput()
			case "?":
				if p := m.currentPage(); p != nil {
					m.status = p.HelpText()
				}
			default:
				if key, ok := hotkeyFromMsg(msg); ok && m.jumpTabByKey(key) {
					m.status = m.tabStatus()
				}
			}

		case focusPage:
			p := m.currentPage()
			if p == nil {
				m.focus = focusTabs
				break
			}
			switch msg.String() {
			case "tab":
				p.FocusNext()
			case "shift+tab":
				p.FocusPrev()
			case "i":
				p.FocusInput()
			case "o":
				p.FocusOptions()
			case "enter", "r":
				p.Run()
				m.status = p.Status()
			case "y":
				m.copyOutput()
			case "s":
				m.saveOutput()
			case "p":
				m.pipeOutput()
			case "?":
				m.status = p.HelpText()
			default:
				var cmd tea.Cmd
				m.page, cmd = p.Update(msg)
				m.status = m.page.Status()
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "sharp"
	}

	leftWidth := max(28, m.width/4)
	rightWidth := max(48, m.width-leftWidth-3)
	bodyHeight := max(20, m.height-2)

	left := appBorderStyle.Width(leftWidth).Height(bodyHeight).Render(m.categoryView(leftWidth, bodyHeight))
	right := appBorderStyle.Width(rightWidth).Height(bodyHeight).Render(m.tabView(rightWidth, bodyHeight))
	footer := footerStyle.Render(m.footerLine())

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right) + "\n" + footer
}

func (m *model) resize() {
	if p := m.currentPage(); p != nil {
		leftWidth := max(28, m.width/4)
		rightWidth := max(48, m.width-leftWidth-3)
		p.SetSize(rightWidth-4, max(16, m.height-6))
	}
	m.search.Width = max(16, max(28, m.width/4)-4)
}

func (m model) categoryView(width, height int) string {
	var b strings.Builder
	b.WriteString(appTitleStyle.Render("sharp") + "\n")

	if m.focus == focusSearch {
		b.WriteString(m.search.View() + "\n")
	} else {
		q := strings.TrimSpace(m.search.Value())
		if q == "" {
			q = "/ search"
		}
		b.WriteString(dimStyle.Render(q) + "\n")
	}

	b.WriteString(sectionRuleStyle.Render(strings.Repeat("-", max(8, width-4))) + "\n")
	b.WriteString(panelTitleStyle.Render("Categories") + "\n")

	visible := max(6, height-8)
	start := 0
	if m.selectedCat >= visible {
		start = m.selectedCat - visible + 1
	}

	for i := start; i < len(m.categories); i++ {
		c := m.categories[i]
		line := fmt.Sprintf("[%c] %-10s (%d)", c.Hotkey, c.Category, len(c.Tabs))
		if len(line) > width-4 {
			line = trimToWidth(line, width-4)
		}
		if i == m.selectedCat {
			line = selectedRowStyle.Width(width - 4).Render(" " + line)
		} else {
			line = dimStyle.Render(" " + line)
		}
		b.WriteString(line + "\n")
		if i-start >= visible {
			break
		}
	}

	return b.String()
}

func (m model) tabView(width, height int) string {
	cat := m.currentCategory()
	if cat == nil {
		return "No tools"
	}

	var b strings.Builder
	b.WriteString(appTitleStyle.Render(string(cat.Category)) + " " + dimStyle.Render("category") + "\n")
	b.WriteString(dimStyle.Render("letters switch tabs | [ ] prev/next | enter run") + "\n")
	b.WriteString(sectionRuleStyle.Render(strings.Repeat("-", max(8, width-4))) + "\n")

	b.WriteString(m.tabBar(width))
	b.WriteString("\n")
	b.WriteString(m.currentPageHeader())
	b.WriteString(sectionRuleStyle.Render(strings.Repeat("-", max(8, width-4))) + "\n")
	if p := m.currentPage(); p != nil {
		b.WriteString(p.View())
	}
	return b.String()
}

func (m model) tabBar(width int) string {
	cat := m.currentCategory()
	if cat == nil {
		return ""
	}

	var parts []string
	for i, tab := range cat.Tabs {
		label := fmt.Sprintf("[%c] %s", tab.Hotkey, tab.Tool.Name())
		if len(label) > 18 {
			label = trimToWidth(label, 18)
		}
		if i == m.selectedTab {
			parts = append(parts, activeLabelStyle.Render(" "+label+" "))
		} else {
			parts = append(parts, dimStyle.Render(" "+label+" "))
		}
	}
	line := strings.Join(parts, " ")
	return trimToWidth(line, max(0, width-4))
}

func (m model) currentPageHeader() string {
	cat := m.currentCategory()
	tab := m.currentTab()
	if cat == nil || tab == nil {
		return dimStyle.Render("No tool selected") + "\n"
	}
	return fmt.Sprintf("%s / %s\n%s\n", appTitleStyle.Render(tab.Tool.Name()), dimStyle.Render(tab.Tool.ID()), dimStyle.Render(tab.Tool.Description()))
}

func (m model) footerLine() string {
	switch m.focus {
	case focusSearch:
		return "Esc/Enter close search | type to filter"
	case focusCategories:
		return "j/k categories | letters jump | Tab tabs | Enter tabs | / search"
	case focusTabs:
		return "[ / ] tab prev/next | letters jump | Tab page | Enter run | i/o page"
	case focusPage:
		if p := m.currentPage(); p != nil {
			return p.HelpText()
		}
		return "No tool selected"
	default:
		return m.status
	}
}

func (m *model) moveCategory(delta int) {
	if len(m.categories) == 0 {
		return
	}
	next := m.selectedCat + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.categories) {
		next = len(m.categories) - 1
	}
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
	next := m.selectedTab + delta
	if next < 0 {
		next = 0
	}
	if next >= len(cat.Tabs) {
		next = len(cat.Tabs) - 1
	}
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
		if idx >= len(cat.Tabs) {
			idx = len(cat.Tabs) - 1
		}
		m.selectedTab = idx
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

func (m *model) rebuildPage() {
	tab := m.currentTab()
	if tab == nil {
		m.page = nil
		return
	}
	m.page = m.currentPage()
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
	m.focus = focusPage
	p.FocusInput()
	m.status = successStyle.Render("output moved to input; choose next tool")
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

func (m *model) jumpCategoryByKey(key rune) bool {
	key = unicode.ToLower(key)
	for i, c := range m.categories {
		if c.Hotkey == key {
			m.selectedCat = i
			m.restoreTabForSelectedCategory()
			m.rebuildPage()
			return true
		}
	}
	return false
}

func (m *model) jumpTabByKey(key rune) bool {
	key = unicode.ToLower(key)
	cat := m.currentCategory()
	if cat == nil {
		return false
	}
	for i, t := range cat.Tabs {
		if t.Hotkey == key {
			m.setSelectedTab(i)
			m.rebuildPage()
			return true
		}
	}
	return false
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

	categoryLabels := make([]string, 0, len(out))
	for _, c := range out {
		categoryLabels = append(categoryLabels, string(c.Category))
	}
	categoryHotkeys := assignHotkeys(categoryLabels)
	for i := range out {
		out[i].Hotkey = categoryHotkeys[i]
		tabLabels := make([]string, 0, len(out[i].Tabs))
		for _, t := range out[i].Tabs {
			tabLabels = append(tabLabels, t.Tool.Name())
		}
		tabHotkeys := assignHotkeys(tabLabels)
		for j := range out[i].Tabs {
			out[i].Tabs[j].Hotkey = tabHotkeys[j]
		}
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
			Hotkey:   c.Hotkey,
			Tabs:     filteredTabs,
		})
	}
	return out
}

func assignHotkeys(labels []string) []rune {
	used := make(map[rune]bool)
	out := make([]rune, len(labels))

	for i, label := range labels {
		label = strings.ToLower(label)
		for _, r := range label {
			if !isHotkeyChar(r) {
				continue
			}
			if !used[r] {
				out[i] = r
				used[r] = true
				break
			}
		}
		if out[i] != 0 {
			continue
		}
		for r := rune('a'); r <= 'z'; r++ {
			if !used[r] {
				out[i] = r
				used[r] = true
				break
			}
		}
		if out[i] == 0 {
			out[i] = rune('a' + (i % 26))
		}
	}
	return out
}

func isHotkeyChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func hotkeyFromMsg(msg tea.KeyMsg) (rune, bool) {
	rs := []rune(msg.String())
	if len(rs) != 1 || !isHotkeyChar(rs[0]) {
		return 0, false
	}
	return unicode.ToLower(rs[0]), true
}

type genericToolPage struct {
	tool       tool.Tool
	input      textarea.Model
	optionBox  textinput.Model
	output     viewport.Model
	options    tool.Options
	focus      pageFocus
	width      int
	height     int
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
	input.Placeholder = "Paste or type input here"
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
		status:     "Enter run | Tab focus | Esc back | y copy | s save | p pipe",
		hasOptions: len(t.Options()) > 0,
	}
	p.loadSelectedOptions()
	return p
}

func (p *genericToolPage) SetSize(width, height int) {
	p.width = width
	p.height = height

	inputHeight := 6
	if p.hasOptions {
		inputHeight = 5
	}
	p.input.SetWidth(max(18, width-4))
	p.input.SetHeight(max(5, inputHeight))
	p.optionBox.Width = max(18, width-4)
	p.output.Width = max(18, width-4)
	p.output.Height = max(8, height-p.input.Height()-p.optionHeight()-10)
}

func (p *genericToolPage) Update(msg tea.Msg) (toolPage, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.SetSize(msg.Width, msg.Height)
		return p, nil
	case tea.KeyMsg:
		switch p.focus {
		case pageFocusInput:
			if msg.String() == "esc" {
				p.input.Blur()
				return p, func() tea.Msg { return pageBackMsg{} }
			}
			var cmd tea.Cmd
			p.input, cmd = p.input.Update(msg)
			return p, cmd
		case pageFocusOptions:
			if msg.String() == "esc" {
				p.optionBox.Blur()
				p.focus = pageFocusInput
				p.input.Focus()
				return p, nil
			}
			var cmd tea.Cmd
			p.optionBox, cmd = p.optionBox.Update(msg)
			return p, cmd
		case pageFocusOutput:
			if msg.String() == "esc" {
				return p, func() tea.Msg { return pageBackMsg{} }
			}
			var cmd tea.Cmd
			p.output, cmd = p.output.Update(msg)
			return p, cmd
		}
	}
	return p, nil
}

func (p *genericToolPage) View() string {
	var b strings.Builder
	b.WriteString(sectionLabel("Input", p.focus == pageFocusInput) + "\n")
	b.WriteString(p.input.View() + "\n")

	if p.hasOptions {
		b.WriteString(sectionLabel("Options", p.focus == pageFocusOptions) + "\n")
		b.WriteString(p.optionBox.View() + "\n")
		for _, opt := range p.tool.Options() {
			req := ""
			if opt.Required {
				req = " required"
			}
			line := fmt.Sprintf("--%s%s %s", opt.Name, req, opt.Description)
			b.WriteString(dimStyle.Render(line) + "\n")
		}
	} else {
		b.WriteString(dimStyle.Render("No options") + "\n")
	}

	b.WriteString("\n")
	b.WriteString(sectionLabel("Output", p.focus == pageFocusOutput) + "\n")
	b.WriteString(p.output.View())
	return b.String()
}

func (p *genericToolPage) FocusInput() {
	p.focus = pageFocusInput
	p.optionBox.Blur()
	p.input.Focus()
}

func (p *genericToolPage) FocusOptions() {
	if !p.hasOptions {
		p.FocusInput()
		return
	}
	p.focus = pageFocusOptions
	p.input.Blur()
	p.optionBox.Focus()
}

func (p *genericToolPage) FocusOutput() {
	p.focus = pageFocusOutput
	p.input.Blur()
	p.optionBox.Blur()
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

func (p *genericToolPage) OutputText() string {
	return p.output.View()
}

func (p *genericToolPage) Status() string {
	return p.status
}

func (p *genericToolPage) HelpText() string {
	return fmt.Sprintf("%s | Tab focus | Esc back | Enter run | y copy | s save | p pipe", p.tool.Name())
}

func (p *genericToolPage) optionHeight() int {
	if !p.hasOptions {
		return 1
	}
	return 2 + len(p.tool.Options())
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

func sectionLabel(title string, active bool) string {
	if active {
		return activeLabelStyle.Render(" " + title + " ")
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
	if width <= 1 {
		return s[:width]
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
