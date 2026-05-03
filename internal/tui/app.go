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
	focusTools focus = iota
	focusInput
	focusOptions
	focusOutput
	focusSearch
)

type model struct {
	registry *tool.Registry
	tools    []tool.Tool
	filtered []tool.Tool
	selected int
	focus    focus

	input     textarea.Model
	output    viewport.Model
	search    textinput.Model
	optionBox textinput.Model
	options   tool.Options

	width  int
	height int
	status string
}

var (
	borderStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	activeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("31"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

func Run(registry *tool.Registry) error {
	p := tea.NewProgram(newModel(registry), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newModel(registry *tool.Registry) model {
	input := textarea.New()
	input.Placeholder = "Paste or type input here"
	input.ShowLineNumbers = false
	input.SetHeight(8)
	input.Blur()

	output := viewport.New(80, 12)

	search := textinput.New()
	search.Placeholder = "Search tools"
	search.Blur()

	optionBox := textinput.New()
	optionBox.Placeholder = "option=value option2=value2"
	optionBox.Blur()

	all := registry.All()
	m := model{
		registry:  registry,
		tools:     all,
		filtered:  all,
		input:     input,
		output:    output,
		search:    search,
		optionBox: optionBox,
		options:   tool.Options{},
		status:    "Enter run | / search | Tab focus | y copy | s save | p pipe | q quit",
	}
	m.loadSelectedOptions()
	return m
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
	case tea.KeyMsg:
		switch m.focus {
		case focusSearch:
			if msg.String() == "esc" || msg.String() == "enter" {
				m.focus = focusTools
				m.search.Blur()
				m.applySearch()
				return m, nil
			}
		case focusInput:
			if msg.String() == "esc" {
				m.focus = focusTools
				m.input.Blur()
				return m, nil
			}
		case focusOptions:
			if msg.String() == "esc" || msg.String() == "enter" {
				m.focus = focusTools
				m.optionBox.Blur()
				m.parseOptions()
				return m, nil
			}
		}

		if m.focus == focusInput {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}
		if m.focus == focusSearch {
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			m.applySearch()
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}
		if m.focus == focusOptions {
			var cmd tea.Cmd
			m.optionBox, cmd = m.optionBox.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.nextFocus()
		case "shift+tab":
			m.prevFocus()
		case "/":
			m.focus = focusSearch
			m.search.Focus()
		case "i":
			m.focus = focusInput
			m.input.Focus()
		case "o":
			m.focus = focusOptions
			m.optionBox.Focus()
		case "enter", "r":
			m.runSelected()
		case "j", "down":
			if m.selected < len(m.filtered)-1 {
				m.selected++
				m.loadSelectedOptions()
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
				m.loadSelectedOptions()
			}
		case "y":
			if err := clipboard.Write(m.output.View()); err != nil {
				m.status = errorStyle.Render("copy failed: " + err.Error())
			} else {
				m.status = successStyle.Render("copied output")
			}
		case "s":
			if err := os.WriteFile("sharp-output.txt", []byte(m.output.View()), 0600); err != nil {
				m.status = errorStyle.Render("save failed: " + err.Error())
			} else {
				m.status = successStyle.Render("saved to sharp-output.txt")
			}
		case "p":
			m.input.SetValue(m.output.View())
			m.focus = focusTools
			m.status = successStyle.Render("output moved to input; choose next tool")
		case "?":
			m.showHelp()
		}
	case tea.MouseMsg:
	}

	if m.focus == focusOutput {
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.width == 0 {
		return "sharp"
	}
	leftWidth := max(24, m.width/4)
	rightWidth := max(40, m.width-leftWidth-4)
	bodyHeight := max(18, m.height-4)

	left := borderStyle.Width(leftWidth).Height(bodyHeight).Render(m.toolsView(leftWidth, bodyHeight))
	right := borderStyle.Width(rightWidth).Height(bodyHeight).Render(m.detailView(rightWidth, bodyHeight))
	footer := mutedStyle.Render(m.status)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right) + "\n" + footer
}

func (m *model) resize() {
	leftWidth := max(24, m.width/4)
	rightWidth := max(40, m.width-leftWidth-8)
	m.input.SetWidth(rightWidth - 4)
	m.optionBox.Width = rightWidth - 4
	m.search.Width = leftWidth - 4
	m.output.Width = rightWidth - 4
	m.output.Height = max(8, m.height-21)
}

func (m model) toolsView(width int, height int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("sharp") + "\n")
	if m.focus == focusSearch {
		b.WriteString(m.search.View() + "\n\n")
	} else {
		query := m.search.Value()
		if query == "" {
			query = "/ search"
		}
		b.WriteString(mutedStyle.Render(query) + "\n\n")
	}
	for i, t := range m.filtered {
		line := fmt.Sprintf("%-10s %s", t.Category(), t.Name())
		if i == m.selected {
			line = activeStyle.Width(width - 4).Render(line)
		}
		b.WriteString(line + "\n")
		if i > height-7 {
			break
		}
	}
	return b.String()
}

func (m model) detailView(width int, height int) string {
	t := m.currentTool()
	if t == nil {
		return "No tools"
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(t.Name()) + " " + mutedStyle.Render(t.ID()) + "\n")
	b.WriteString(mutedStyle.Render(t.Description()) + "\n\n")
	b.WriteString(sectionTitle("Input", m.focus == focusInput) + "\n")
	b.WriteString(m.input.View() + "\n")
	b.WriteString(sectionTitle("Options", m.focus == focusOptions) + "\n")
	b.WriteString(m.optionBox.View() + "\n")
	if len(t.Options()) > 0 {
		for _, opt := range t.Options() {
			req := ""
			if opt.Required {
				req = " required"
			}
			b.WriteString(mutedStyle.Render(fmt.Sprintf("--%s%s %s", opt.Name, req, opt.Description)) + "\n")
		}
	}
	b.WriteString("\n" + sectionTitle("Output", m.focus == focusOutput) + "\n")
	b.WriteString(m.output.View())
	return b.String()
}

func sectionTitle(s string, active bool) string {
	if active {
		return activeStyle.Render(" " + s + " ")
	}
	return titleStyle.Render(s)
}

func (m *model) nextFocus() {
	switch m.focus {
	case focusTools:
		m.focus = focusInput
		m.input.Focus()
	case focusInput:
		m.input.Blur()
		m.focus = focusOptions
		m.optionBox.Focus()
	case focusOptions:
		m.optionBox.Blur()
		m.focus = focusOutput
	case focusOutput:
		m.focus = focusTools
	}
}

func (m *model) prevFocus() {
	switch m.focus {
	case focusTools:
		m.focus = focusOutput
	case focusInput:
		m.input.Blur()
		m.focus = focusTools
	case focusOptions:
		m.optionBox.Blur()
		m.focus = focusInput
		m.input.Focus()
	case focusOutput:
		m.focus = focusOptions
		m.optionBox.Focus()
	}
}

func (m *model) applySearch() {
	q := strings.ToLower(strings.TrimSpace(m.search.Value()))
	if q == "" {
		m.filtered = m.tools
	} else {
		m.filtered = nil
		for _, t := range m.tools {
			haystack := strings.ToLower(t.ID() + " " + t.Name() + " " + string(t.Category()) + " " + t.Description())
			if strings.Contains(haystack, q) {
				m.filtered = append(m.filtered, t)
			}
		}
	}
	if m.selected >= len(m.filtered) {
		m.selected = max(0, len(m.filtered)-1)
	}
	m.loadSelectedOptions()
}

func (m *model) currentTool() tool.Tool {
	if len(m.filtered) == 0 || m.selected < 0 || m.selected >= len(m.filtered) {
		return nil
	}
	return m.filtered[m.selected]
}

func (m *model) loadSelectedOptions() {
	t := m.currentTool()
	if t == nil {
		return
	}
	parts := make([]string, 0, len(t.Options()))
	for _, opt := range t.Options() {
		value := opt.Default
		if value == "" && opt.Required {
			value = ""
		}
		parts = append(parts, fmt.Sprintf("%s=%s", opt.Name, value))
	}
	m.optionBox.SetValue(strings.Join(parts, " "))
	m.parseOptions()
}

func (m *model) parseOptions() {
	m.options = tool.Options{}
	for _, part := range strings.Fields(m.optionBox.Value()) {
		k, v, ok := strings.Cut(part, "=")
		if ok {
			m.options[k] = v
		}
	}
}

func (m *model) runSelected() {
	t := m.currentTool()
	if t == nil {
		return
	}
	m.parseOptions()
	out, err := t.Run(context.Background(), tool.Input{Text: m.input.Value()}, m.options)
	if err != nil {
		m.output.SetContent(errorStyle.Render(err.Error()))
		m.status = errorStyle.Render("error")
		return
	}
	m.output.SetContent(out.Text)
	m.status = successStyle.Render("ran " + t.ID())
}

func (m *model) showHelp() {
	t := m.currentTool()
	if t == nil {
		return
	}
	var b strings.Builder
	b.WriteString(t.Name() + "\n\n")
	b.WriteString(t.Description() + "\n\n")
	b.WriteString("Shortcuts:\n")
	b.WriteString("/ search\nTab focus\nEnter run\ny copy\ns save\np pipe output to input\ni edit input\no edit options\nq quit\n")
	m.output.SetContent(b.String())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
