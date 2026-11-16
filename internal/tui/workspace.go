package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/pkg/tool"
)

type editField int

const (
	editFieldInput editField = iota
	editFieldOptions
)

// 每次修改与显式操作都会推进版本，迟到的预览不能覆盖新结果。
type workspacePreview struct {
	revision       uint64
	stale          bool
	usedTool       string
	previewPending bool
}

func (p *workspacePreview) markToolUsed(id string) { p.usedTool = id }
func (p *workspacePreview) takeToolUse() string {
	id := p.usedTool
	p.usedTool = ""
	return id
}

type workspacePreviewMsg struct {
	page     toolPage
	revision uint64
}

func (p *workspacePreview) changed()               { p.revision++; p.stale = true; p.previewPending = true }
func (p *workspacePreview) settled()               { p.revision++; p.stale = false; p.previewPending = false }
func (p *workspacePreview) OutputReady() bool      { return !p.stale }
func (p *workspacePreview) previewVersion() uint64 { return p.revision }
func (p *workspacePreview) schedule(page toolPage) tea.Cmd {
	revision := p.revision
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return workspacePreviewMsg{page: page, revision: revision}
	})
}
func (p *workspacePreview) title(action, selectedAction string) string {
	title := "Output"
	// Retained output can belong to a different action, including JSON Check
	// which intentionally leaves the preceding data result untouched.
	if action != "" && action != selectedAction {
		title += " · " + action
	}
	if p.stale {
		title += " · stale"
	}
	return title
}

// Reserve one row per option; the remaining space belongs to the document.
func workspaceSizes(width, height int, hasInput bool, optionRows int) (pageWidth, inputH, optionsH, outputH int) {
	pageWidth = max(8, width)
	if optionRows > 0 {
		optionsH = 2 + optionRows
	}
	remaining := max(3, height-optionsH)
	if hasInput {
		inputH = max(3, remaining/2)
		outputH = max(3, remaining-inputH)
	} else {
		outputH = remaining
	}
	return
}

func newWorkspaceInput() textarea.Model {
	return newWorkspaceInputWithStyles(defaultStyleSet())
}

func newWorkspaceInputWithStyles(styles StyleSet) textarea.Model {
	input := textarea.New()
	input.Prompt = ""
	input.ShowLineNumbers = false
	input.SetHeight(8)
	applyWorkspaceInputStyles(&input, styles)
	input.Blur()
	return input
}

func applyWorkspaceInputStyles(input *textarea.Model, styles StyleSet) {
	style := textarea.Style{
		Base: styles.Text, Text: styles.Text, CursorLine: styles.Text,
		Placeholder: styles.Dim, EndOfBuffer: styles.Dim,
	}
	input.FocusedStyle, input.BlurredStyle = style, style
	input.Cursor.Style = styles.Focus
}

func workspaceInputView(input textarea.Model, active bool, hint string, styles StyleSet) string {
	if !active {
		input.Blur()
	}
	if input.Value() == "" && !input.Focused() {
		return styles.Text.Render(wrapDisplayText(hint, max(1, input.Width())))
	}
	return input.View()
}

// Open sections use a heading and whitespace instead of a second set of boxes.
// Keep the same two-row budget as the editor: one heading, one trailing gap.
func workspaceSection(title string, editing bool, content string, width, height int, styles StyleSet) string {
	if height <= 0 {
		return ""
	}
	innerWidth := max(1, width-2)
	label := " " + truncateDisplay(title, max(1, innerWidth-2)) + " "
	style := styles.SectionTitle
	if editing {
		style = styles.Focus
	}
	heading := style.Render(label) + styles.SectionRule.Render(strings.Repeat("─", max(0, width-lipgloss.Width(label))))
	lines := []string{fitWidth(heading, width)}
	for _, line := range fixedLines(content, innerWidth, max(0, height-2)) {
		lines = append(lines, " "+line+" ")
	}
	if height > 1 {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

func workspaceInputHint(t tool.Tool) string {
	kind := "text"
	prefix, _, _ := strings.Cut(t.ID(), ".")
	switch prefix {
	case "json":
		kind = "JSON"
	case "yaml":
		kind = "YAML"
	case "csv":
		kind = "CSV"
	}
	return "Paste " + kind + " with v, or i to edit"
}

func workspaceOutputHint(autoPreview, hasInput bool) string {
	if !hasInput {
		return "r / Enter to generate"
	}
	if !autoPreview {
		return "r / Enter to run"
	}
	return "Waiting for input"
}

func workspaceOutputView(output viewport.Model, text, action, hint string, styles StyleSet) string {
	if text == "" && action == "" {
		return styles.Dim.Render(wrapDisplayText(hint, max(1, output.Width)))
	}
	return output.View()
}

func workspaceRunStatus(t tool.Tool, preview bool, _ StyleSet) statusMessage {
	verb := "Completed"
	if preview {
		verb = "Updated"
	} else if t.Category() == tool.CategoryGenerator {
		verb = "Generated"
	}
	return successStatus(verb + " " + t.Name())
}

func workspaceFooter(autoPreview, hasInput, hasOptions, editing bool, field editField, optionsTitle string, styles StyleSet) string {
	if editing {
		if field == editFieldOptions {
			return modeStatusLine(styles, "INSERT", optionsTitle, "type text", statusHintWithStyles(styles, "Enter", "run"), statusHintWithStyles(styles, "Esc", "normal"))
		}
		return modeStatusLine(styles, "INSERT", "Input", "type text", statusHintWithStyles(styles, "Enter", "newline"), statusHintWithStyles(styles, "Esc", "normal"))
	}
	hints := []string{}
	if !autoPreview {
		hints = append(hints, statusHintWithStyles(styles, "r/Enter", "run"))
	}
	if hasInput {
		hints = append(hints, statusHintWithStyles(styles, "v", "paste"), statusHintWithStyles(styles, "i", "edit"), statusHintWithStyles(styles, "x", "clear"))
	}
	if hasOptions {
		hints = append(hints, statusHintWithStyles(styles, "o", "options"))
	}
	hints = append(hints, statusHintWithStyles(styles, "y", "copy"))
	if hasInput {
		hints = append(hints, statusHintWithStyles(styles, "p", "apply"))
	}
	hints = append(hints, statusHintWithStyles(styles, "?", "help"))
	return modeStatusLine(styles, "NORMAL", "", hints...)
}

// Mode is derived from the focused context, never stored separately from it.
func modeStatusLine(styles StyleSet, mode, target string, hints ...string) string {
	background := lipgloss.TerminalColor(styles.RetainedColor)
	foreground := lipgloss.TerminalColor(styles.AccentColor)
	switch mode {
	case "INSERT":
		foreground = styles.WarningColor
	case "SEARCH":
		foreground = styles.Focus.GetForeground()
	case "HELP":
		foreground = styles.AccentColor
	}
	badge := lipgloss.NewStyle().Foreground(foreground).Background(background).Render(mode)
	line := statusLineWithStyles(styles, target, hints...)
	if line == "" {
		return badge
	}
	return badge + " " + line
}
