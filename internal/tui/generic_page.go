package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/pkg/tool"
)

type genericToolPage struct {
	workspacePreview
	height       int
	optionDrafts map[string]string
	lastAction   string
	active       bool
	tool         tool.Tool
	input        textarea.Model
	optionBox    textinput.Model
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
	input := textarea.New()
	input.Placeholder = ""
	input.ShowLineNumbers = false
	input.SetHeight(8)
	input.Blur()

	optionBox := textinput.New()
	optionBox.Placeholder = "option=value option2=value2"
	if hasSecretOptions(t.Options()) {
		// Secret 参数所在的整行使用密码模式，避免 HMAC key 等值在编辑时直接暴露。
		optionBox.EchoMode = textinput.EchoPassword
		optionBox.EchoCharacter = '*'
	}
	optionBox.Blur()

	output := viewport.New(80, 12)

	p := &genericToolPage{
		active:       true,
		optionDrafts: map[string]string{},
		tool:         t,
		input:        input,
		optionBox:    optionBox,
		output:       output,
		options:      tool.Options{},
		status:       "i edit | o options | r/Enter run | y copy | s save | p apply",
		hasOptions:   len(t.Options()) > 0,
		hasInput:     toolUsesInput(t),
	}
	p.loadSelectedOptions()
	p.FocusInput()
	return p
}

// SetSize recalculates the generic page layout and child component sizes.
func (p *genericToolPage) SetSize(width, height int) {
	p.height = height
	p.width, p.inputBoxH, p.optionsBoxH, p.outputBoxH = workspaceSizes(width, height, p.hasInput, p.hasOptions)
	innerWidth := max(1, p.width-2)
	p.input.SetWidth(innerWidth)
	if p.hasInput {
		p.input.SetHeight(max(1, p.inputBoxH-2))
	}
	p.optionBox.Width = innerWidth
	p.output.Width = innerWidth
	p.output.Height = max(1, p.outputBoxH-2)
	p.renderOutput()
}

func (p *genericToolPage) SelectTool(t tool.Tool) {
	if p.tool.ID() == t.ID() {
		return
	}
	p.optionDrafts[p.tool.ID()] = p.optionBox.Value()
	p.stopEditing()
	p.revision++
	p.tool = t
	p.hasInput = toolUsesInput(t)
	p.hasOptions = len(t.Options()) > 0
	p.optionBox.EchoMode = textinput.EchoNormal
	if hasSecretOptions(t.Options()) {
		p.optionBox.EchoMode = textinput.EchoPassword
		p.optionBox.EchoCharacter = '*'
	}
	p.options = tool.Options{}
	if draft, ok := p.optionDrafts[t.ID()]; ok {
		p.optionBox.SetValue(draft)
	} else {
		p.loadSelectedOptions()
	}
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
		// Tab 属于区域导航；编辑中先 Esc，不能隐式切到另一个字段。
		if event.String() == "tab" || event.String() == "shift+tab" {
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
		before := p.optionBox.Value()
		p.optionBox, cmd = p.optionBox.Update(msg)
		if p.optionBox.Value() != before {
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
		blocks = append(blocks, panelBoxTitle("Input", p.active && p.editing && p.field == editFieldInput, workspaceInputView(p.input, p.active), p.width, p.inputBoxH))
	}

	if p.hasOptions {
		var b strings.Builder
		if p.active && p.editing && p.field == editFieldOptions {
			b.WriteString(p.optionBox.View() + "\n")
		} else {
			b.WriteString(dimStyle.Render(p.optionSummary()) + "\n")
		}
		blocks = append(blocks, panelBoxTitle("Options", p.active && p.editing && p.field == editFieldOptions, b.String(), p.width, p.optionsBoxH))
	} else {
		// 无参数工具不再占一整块区域，把高度留给输入/输出。
	}

	blocks = append(blocks, panelBoxTitle(p.workspacePreview.title(p.lastAction), false, p.output.View(), p.width, p.outputBoxH))
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
	p.optionBox.Blur()
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
	p.optionBox.Blur()
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
		if hasSecretOptions(p.tool.Options()) {
			// 解析错误可能引用未加引号的密钥片段，不能直接回显。
			p.outputText = "Invalid options; use key=value and quote values containing spaces. Press o to edit."
		}
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
	return fmt.Sprintf("%s | i edit | Esc back/exit edit | Tab field | r/Enter run | y copy | s save | p apply", p.tool.Name())
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
		commands = append(commands, helpCommand{Key: "o", Command: "Edit Options", Description: "Edit option=value pairs; Enter runs, Esc returns to Normal."})
		for _, opt := range p.tool.Options() {
			commands = append(commands, helpCommand{Key: "--" + opt.Name, Command: "Option", Description: opt.Description})
		}
	}
	return commands
}

// FooterHints returns status bar shortcuts customized to the current tool shape.
func (p *genericToolPage) FooterHints() string {
	return workspaceFooter(canPreview(p.tool), p.hasInput, p.hasOptions, p.editing, p.field, "Options")
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
		p.optionBox.Blur()
		p.input.Focus()
	case "o":
		if !p.hasOptions {
			p.status = dimStyle.Render("This action has no options")
			return true
		}
		p.FocusOptions()
		p.editing = true
		p.optionBox.Focus()
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
	_ = p.parseOptions()
}

func (p *genericToolPage) optionSummary() string {
	// 编辑结束即展示当前草稿，非法参数不静默显示上次成功的值。
	options, err := parseOptionText(p.optionBox.Value(), p.tool.Options())
	if err != nil {
		if hasSecretOptions(p.tool.Options()) {
			return "Invalid options; press o to edit"
		}
		return p.optionBox.Value() + " (invalid; press o to edit)"
	}
	parts := make([]string, 0, len(p.tool.Options()))
	for _, opt := range p.tool.Options() {
		value := options[opt.Name]
		if opt.Secret {
			if strings.TrimSpace(value) == "" {
				value = ""
			} else {
				value = "<secret>"
			}
		}
		parts = append(parts, fmt.Sprintf("%s=%s", opt.Name, value))
	}
	return strings.Join(parts, " ")
}

func (p *genericToolPage) parseOptions() error {
	options, err := parseOptionText(p.optionBox.Value(), p.tool.Options())
	if err != nil {
		return err
	}
	p.options = options
	return nil
}
