package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/pkg/tool"
)

type genericToolPage struct {
	workspacePreview
	height       int
	optionDrafts map[string]optionDraft
	lastAction   string
	active       bool
	tool         tool.Tool
	input        textarea.Model
	optionForm   optionForm
	output       viewport.Model
	outputText   string
	outputErr    bool
	options      tool.Options
	inputUndo    []string
	inputRedo    []string
	width        int
	inputBoxH    int
	optionsBoxH  int
	outputBoxH   int
	field        editField
	editing      bool
	status       string
	hasOptions   bool
	hasInput     bool
}

func newToolPage(t tool.Tool) toolPage {
	if factory, ok := specialPageFactories[t.ID()]; ok {
		return factory(t)
	}
	return newGenericToolPage(t)
}

var specialPageFactories = map[string]func(tool.Tool) toolPage{}

func newGenericToolPage(t tool.Tool) *genericToolPage {
	input := newWorkspaceInput()

	output := viewport.New(80, 12)

	p := &genericToolPage{
		active:       true,
		optionDrafts: map[string]optionDraft{},
		tool:         t,
		input:        input,
		optionForm:   newOptionForm(t.Options(), nil),
		output:       output,
		options:      tool.Options{},
		status:       "i edit | o options | r/Enter run | y copy | s save | p apply",
		hasOptions:   len(t.Options()) > 0,
		hasInput:     toolUsesInput(t),
	}
	p.FocusInput()
	return p
}

// SetSize recalculates the generic page layout and child component sizes.
func (p *genericToolPage) SetSize(width, height int) {
	p.height = height
	p.width, p.inputBoxH, p.optionsBoxH, p.outputBoxH = workspaceSizes(width, height, p.hasInput, len(p.tool.Options()))
	innerWidth := max(1, p.width-2)
	p.input.SetWidth(innerWidth)
	if p.hasInput {
		p.input.SetHeight(max(1, p.inputBoxH-2))
	}
	p.optionForm.SetWidth(innerWidth)
	p.output.Width = innerWidth
	p.output.Height = max(1, p.outputBoxH-2)
	p.renderOutput()
}

func (p *genericToolPage) SelectTool(t tool.Tool) {
	if p.tool.ID() == t.ID() {
		return
	}
	p.optionDrafts[p.tool.ID()] = optionDraft{values: p.optionForm.Values(), selected: p.optionForm.selected}
	p.stopEditing()
	p.revision++
	p.tool = t
	p.hasInput = toolUsesInput(t)
	p.hasOptions = len(t.Options()) > 0
	draft := p.optionDrafts[t.ID()]
	p.optionForm = newOptionForm(t.Options(), draft.values)
	p.optionForm.selected = draft.selected
	p.options = tool.Options{}
	_ = p.parseOptions()
	if (p.field == editFieldOptions && !p.hasOptions) || (p.field == editFieldInput && !p.hasInput) {
		p.FocusInput()
	}
	if p.width > 0 {
		p.SetSize(p.width, p.height)
	}
	p.status = dimStyle.Render("Selected " + t.Name() + "; r/Enter runs on Input")
}

func (p *genericToolPage) Preview() {
	p.revision++
	if !canPreview(p.tool) {
		return
	}
	if strings.TrimSpace(p.input.Value()) == "" {
		p.ClearOutput()
		p.status = dimStyle.Render("Input empty; v paste or i edit")
		return
	}
	if err := p.parseOptions(); err != nil {
		p.stale = true
		p.status = dimStyle.Render("Options invalid or required; press o to edit")
		return
	}
	p.run(true)
}

func (p *genericToolPage) SetActive(active bool) { p.active = active }

// Apply 的语义与 JSON 一致，错误和无输入动作都不能覆盖文档。
func (p *genericToolPage) ApplyOutput() {
	if !p.OutputReady() {
		p.status = errorStyle.Render("Output is stale; update Input/Options or run again")
		return
	}
	if !p.hasInput {
		p.status = dimStyle.Render("This action has no input")
		return
	}
	if p.outputErr {
		p.status = errorStyle.Render("Cannot apply an error; fix Input or Options and run again")
		return
	}
	text := p.OutputText()
	if text == "" {
		p.status = dimStyle.Render("nothing to apply")
		return
	}
	p.SetInput(text)
	p.ClearOutput()
	p.status = successStyle.Render("applied output to input")
}

// Update keeps command keys out of editors and debounces local previews.
func (p *genericToolPage) Update(msg tea.Msg) (toolPage, tea.Cmd) {
	switch event := msg.(type) {
	case workspacePreviewMsg:
		if event.revision == p.revision {
			p.Preview()
		}
		return p, nil
	case tea.WindowSizeMsg:
		p.SetSize(event.Width, event.Height)
		return p, nil
	case tea.KeyMsg:
		if event.String() == "esc" {
			p.stopEditing()
			p.Preview()
			return p, nil
		}
		if !p.editing {
			var cmd tea.Cmd
			p.output, cmd = p.output.Update(msg)
			return p, cmd
		}
		if p.field == editFieldOptions && event.String() == "enter" {
			p.Run()
			p.stopEditing()
			return p, nil
		}
		// Tab stays inside Options; Input still reserves it for Normal navigation.
		if (event.String() == "tab" || event.String() == "shift+tab") && p.field == editFieldInput {
			return p, nil
		}
	}
	if !p.editing {
		return p, nil
	}
	var cmd tea.Cmd
	if p.field == editFieldInput {
		before := p.input.Value()
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "ctrl+z" {
			p.undoInput()
		} else if ok && key.String() == "ctrl+y" {
			p.redoInput()
		} else {
			p.input, cmd = p.input.Update(msg)
			p.trackInputChange(before)
		}
		if p.input.Value() != before && canPreview(p.tool) {
			return p, tea.Batch(cmd, p.schedule(p))
		}
	} else {
		var changed bool
		changed, cmd = p.optionForm.Update(msg)
		if changed {
			p.changed()
			if canPreview(p.tool) {
				return p, tea.Batch(cmd, p.schedule(p))
			}
		}
	}
	return p, cmd
}

// View renders the generic Input/Options/Output workspace.
func (p *genericToolPage) View() string {
	blocks := make([]string, 0, 4)
	if p.hasInput {
		blocks = append(blocks, workspaceSection("Input", p.active && p.editing && p.field == editFieldInput, workspaceInputView(p.input, p.active, workspaceInputHint(p.tool)), p.width, p.inputBoxH))
	}

	if p.hasOptions {
		editing := p.active && p.editing && p.field == editFieldOptions
		blocks = append(blocks, workspaceSection("Options", editing, p.optionForm.View(editing, max(1, p.optionsBoxH-2)), p.width, p.optionsBoxH))
	}

	output := workspaceOutputView(p.output, p.outputText, p.lastAction, workspaceOutputHint(canPreview(p.tool), p.hasInput))
	blocks = append(blocks, workspaceSection(p.workspacePreview.title(p.lastAction, p.tool.Name()), false, output, p.width, p.outputBoxH))
	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

// FocusInput selects the input pane; i explicitly starts editing.
func (p *genericToolPage) FocusInput() {
	if !p.hasInput {
		if p.hasOptions {
			p.FocusOptions()
			return
		}
		p.stopEditing()
		return
	}
	p.field = editFieldInput
	p.editing = false
	p.optionForm.Blur()
	p.input.Blur()
}

// FocusOptions selects options without starting editing.
func (p *genericToolPage) FocusOptions() {
	if !p.hasOptions {
		p.FocusInput()
		return
	}
	p.field = editFieldOptions
	p.editing = false
	p.input.Blur()
	p.optionForm.Blur()
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
func (p *genericToolPage) Run() { p.run(false) }

func (p *genericToolPage) run(preview bool) {
	p.revision++
	if err := p.parseOptions(); err != nil {
		p.settled()
		p.lastAction = p.tool.Name()
		p.outputText = err.Error()
		p.outputErr = true
		p.renderOutput()
		p.status = errorStyle.Render("invalid options: " + p.tool.ID())
		return
	}
	input := ""
	if p.hasInput {
		input = p.input.Value()
	}
	out, err := p.tool.Run(context.Background(), tool.Input{Text: input}, p.options)
	if err != nil {
		if preview {
			p.stale = true
			p.status = dimStyle.Render("Input or options invalid; preview paused")
			return
		}
		p.settled()
		p.lastAction = p.tool.Name()
		p.outputText = err.Error()
		p.outputErr = true
		p.renderOutput()
		p.status = errorStyle.Render("run failed: " + p.tool.ID())
		return
	}
	p.settled()
	p.lastAction = p.tool.Name()
	p.outputText = out.Text
	p.outputErr = false
	p.renderOutput()
	p.status = workspaceRunStatus(p.tool, preview)
	p.markToolUsed(p.tool.ID())
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
	p.changed()
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
	p.changed()
}

// ClearOutput clears raw and rendered output.
func (p *genericToolPage) ClearOutput() {
	p.outputText = ""
	p.outputErr = false
	p.lastAction = ""
	p.output.SetContent("")
	p.settled()
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
	return fmt.Sprintf("%s | i/o Insert | Esc Normal/back | Tab area in Normal | r/Enter run | y copy | s save | p apply", p.tool.Name())
}

// HelpCommands returns structured commands for the popup help overlay.
func (p *genericToolPage) HelpCommands() []helpCommand {
	commands := []helpCommand{
		{Key: "r/Enter", Command: "Run Action", Description: "Run " + p.tool.Name() + " with current input and options."},
		{Key: "s", Command: "Save Output", Description: "Save trimmed output to sharp-output.txt."},
		{Key: "y", Command: "Copy Output", Description: "Copy trimmed output text."},
		{Key: "X", Command: "Clear Output", Description: "Clear the current output."},
	}
	if p.hasInput {
		commands = append(commands,
			helpCommand{Key: "i", Command: "Edit Input", Description: "Edit Input directly; Esc returns to Normal."},
			helpCommand{Key: "v", Command: "Paste Input", Description: "Paste clipboard data into the input editor."},
			helpCommand{Key: "x", Command: "Clear Input", Description: "Clear the current input."},
			helpCommand{Key: "p/a", Command: "Apply Output", Description: "Apply successful output to Input and clear Output; errors cannot be applied."},
		)
	}
	if p.hasOptions {
		commands = append(commands, helpCommand{Key: "o", Command: "Edit Options", Description: "Edit values beside fixed labels; Tab/Shift+Tab moves between fields, Left/Right selects choices, Enter runs, Esc returns to Normal."})
		for _, opt := range p.tool.Options() {
			commands = append(commands, helpCommand{Key: "--" + opt.Name, Command: "Option", Description: opt.Description})
		}
	}
	return commands
}

// FooterHints returns status bar shortcuts customized to the current tool shape.
func (p *genericToolPage) FooterHints() string {
	if p.editing && p.field == editFieldOptions {
		hints := []string{}
		if p.optionForm.HasChoices() {
			hints = append(hints, statusHint("←/→", "choose"))
		}
		if len(p.tool.Options()) > 1 {
			hints = append(hints, statusHint("Tab/S-Tab", "field"))
		}
		hints = append(hints, statusHint("Enter", "run"), statusHint("Esc", "normal"))
		return modeStatusLine("INSERT", "Options", hints...)
	}
	return workspaceFooter(canPreview(p.tool), p.hasInput, p.hasOptions, p.editing, p.field, "Options")
}

// IsEditing reports whether key events should be routed to a text editor.
func (p *genericToolPage) IsEditing() bool {
	return p.editing
}

func (p *genericToolPage) EditTarget() string {
	if !p.editing {
		return ""
	}
	if p.field == editFieldOptions {
		return "Options"
	}
	return "Input"
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
	p.changed()
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
	p.changed()
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
	p.changed()
	p.status = successStyle.Render("redo input")
}

// HandleKey handles page-specific normal-mode shortcuts.
func (p *genericToolPage) HandleKey(key string) bool {
	switch key {
	case "i":
		if !p.hasInput {
			p.status = dimStyle.Render("This action has no input; o edits options")
			return true
		}
		p.field = editFieldInput
		p.editing = true
		p.optionForm.Blur()
		p.input.Focus()
	case "o":
		if !p.hasOptions {
			p.status = dimStyle.Render("This action has no options")
			return true
		}
		p.FocusOptions()
		p.editing = true
		p.optionForm.Focus()
	case "a":
		p.ApplyOutput()
	case "z":
		p.undoInput()
		p.Preview()
	default:
		return false
	}
	return true
}

func (p *genericToolPage) stopEditing() {
	p.editing = false
	p.input.Blur()
	p.optionForm.Blur()
}

func (p *genericToolPage) parseOptions() error {
	options := p.optionForm.Values()
	for _, opt := range p.tool.Options() {
		if opt.Required && strings.TrimSpace(options[opt.Name]) == "" {
			return fmt.Errorf("missing required option: %s", opt.Name)
		}
	}
	p.options = options
	return nil
}
