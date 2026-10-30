package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/internal/tools/jsonutil"
	"github.com/gkmz/sharp/pkg/tool"
)

type jsonToolPage struct {
	workspacePreview
	active         bool
	selectedAction string
	height         int
	input          textarea.Model
	pathBox        textinput.Model
	output         viewport.Model
	tools          map[string]tool.Tool
	width          int
	inputBoxH      int
	pathBoxH       int
	outputBoxH     int
	field          editField
	editing        bool
	inputUndo      []string
	inputRedo      []string
	outputText     string
	outputErr      bool
	inputState     string
	inputErr       bool
	status         string
	lastAction     string
}

func newJSONToolPage(t tool.Tool, tools map[string]tool.Tool) *jsonToolPage {
	input := newWorkspaceInput()

	pathBox := textinput.New()
	pathBox.Placeholder = "gjson path, e.g. data.user.name"
	pathBox.Prompt = ""
	pathBox.TextStyle, pathBox.PlaceholderStyle = textStyle, dimStyle
	pathBox.Cursor.Style = focusStyle
	pathBox.Blur()

	p := &jsonToolPage{
		active:         true,
		selectedAction: t.ID(),
		input:          input,
		pathBox:        pathBox,
		output:         viewport.New(80, 12),
		tools:          tools,
		status:         "JSON | choose an action | i edit | r/Enter run | p apply",
	}
	if _, ok := p.tools[t.ID()]; !ok {
		p.tools[t.ID()] = t
	}
	p.refreshInputState()
	return p
}

// SetSize recalculates the JSON workbench layout and child component sizes.
func (p *jsonToolPage) SetSize(width, height int) {
	p.height = height
	optionRows := 0
	if p.selectedAction == "json.get" {
		optionRows = 1
	}
	p.width, p.inputBoxH, p.pathBoxH, p.outputBoxH = workspaceSizes(width, height, true, optionRows)
	innerWidth := max(1, p.width-2)
	p.input.SetWidth(innerWidth)
	p.input.SetHeight(max(1, p.inputBoxH-2))
	p.pathBox.Width = innerWidth
	p.output.Width = innerWidth
	p.output.Height = max(1, p.outputBoxH-2)
	p.renderOutput()
}

func (p *jsonToolPage) SelectTool(t tool.Tool) {
	if p.selectedAction == t.ID() {
		return
	}
	p.stopEditing()
	p.revision++
	p.selectedAction = t.ID()
	if p.field == editFieldOptions && t.ID() != "json.get" {
		p.FocusInput()
	}
	if p.width > 0 {
		p.SetSize(p.width, p.height)
	}
}

func (p *jsonToolPage) Preview() {
	p.revision++
	if strings.TrimSpace(p.input.Value()) == "" {
		p.ClearOutput()
		p.refreshInputState()
		return
	}
	if p.selectedAction == "json.get" && strings.TrimSpace(p.pathBox.Value()) == "" {
		p.stale = true
		p.status = dimStyle.Render("Path required; press o to edit")
		return
	}
	options := tool.Options(nil)
	if p.selectedAction == "json.get" {
		options = tool.Options{"path": strings.TrimSpace(p.pathBox.Value())}
	}
	p.runAction(p.selectedAction, options, true)
}

func (p *jsonToolPage) SetActive(active bool) { p.active = active }
func (p *jsonToolPage) ApplyOutput()          { p.applyOutput() }

// Update keeps command keys out of editors and debounces local previews.
func (p *jsonToolPage) Update(msg tea.Msg) (toolPage, tea.Cmd) {
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
			p.runPathQuery()
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
		if p.input.Value() != before {
			return p, tea.Batch(cmd, p.schedule(p))
		}
	} else {
		before := p.pathBox.Value()
		p.pathBox, cmd = p.pathBox.Update(msg)
		if p.pathBox.Value() != before {
			p.changed()
			return p, tea.Batch(cmd, p.schedule(p))
		}
	}
	return p, cmd
}

// View renders the JSON workbench with input, path, and output panes.
func (p *jsonToolPage) View() string {
	inputTitle := "Input"
	if p.inputState != "" {
		inputTitle += " - " + p.inputState
	}
	inputBlock := workspaceSection(inputTitle, p.active && p.editing && p.field == editFieldInput, workspaceInputView(p.input, p.active, "Paste JSON with v, or i to edit"), p.width, p.inputBoxH)
	var path strings.Builder
	if p.field == editFieldOptions && p.editing {
		path.WriteString(p.pathBox.View())
	} else {
		value := p.pathBox.Value()
		if value == "" {
			value = "gjson path"
		}
		path.WriteString(dimStyle.Render(value))
	}
	pathBlock := workspaceSection("Path", p.active && p.editing && p.field == editFieldOptions, path.String(), p.width, p.pathBoxH)
	output := workspaceOutputView(p.output, p.outputText, p.lastAction, workspaceOutputHint(true, true))
	selectedName := ""
	if t := p.tools[p.selectedAction]; t != nil {
		selectedName = t.Name()
	}
	outputBlock := workspaceSection(p.workspacePreview.title(p.lastAction, selectedName), false, output, p.width, p.outputBoxH)
	if p.pathBoxH > 0 {
		return lipgloss.JoinVertical(lipgloss.Left, inputBlock, pathBlock, outputBlock)
	}
	return lipgloss.JoinVertical(lipgloss.Left, inputBlock, outputBlock)
}

// FocusInput selects the JSON input pane without entering insert mode.
func (p *jsonToolPage) FocusInput() {
	p.field = editFieldInput
	p.editing = false
	p.pathBox.Blur()
	p.input.Blur()
}

// FocusOptions selects the JSON path pane without entering insert mode.
func (p *jsonToolPage) FocusOptions() {
	if p.selectedAction != "json.get" {
		p.FocusInput()
		return
	}
	p.field = editFieldOptions
	p.editing = false
	p.input.Blur()
	p.pathBox.Blur()
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

// Run executes the selected JSON action against the shared input.
func (p *jsonToolPage) Run() {
	if p.selectedAction == "json.get" {
		p.runPathQuery()
		return
	}
	id := p.selectedAction
	if id == "" {
		id = "json.pretty"
	}
	p.runAction(id, nil, false)
}

// ClearInput clears JSON input and records undo history.
func (p *jsonToolPage) ClearInput() {
	p.pushInputUndo()
	p.input.SetValue("")
	p.inputRedo = nil
	p.changed()
	p.refreshInputState()
}

// ClearOutput clears raw and rendered output.
func (p *jsonToolPage) ClearOutput() {
	p.outputText = ""
	p.outputErr = false
	p.lastAction = ""
	p.output.SetContent("")
	p.settled()
}

// SetInput replaces JSON input and records undo history.
func (p *jsonToolPage) SetInput(value string) {
	p.pushInputUndo()
	p.input.SetValue(value)
	p.inputRedo = nil
	p.changed()
	p.refreshInputState()
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
	return "JSON | r/Enter run selected action | i edit | v paste | y copy | s save | p apply | z undo"
}

// FooterHints returns JSON-specific status bar shortcuts.
func (p *jsonToolPage) FooterHints() string {
	return workspaceFooter(true, true, p.selectedAction == "json.get", p.editing, p.field, "Path")
}

// HelpCommands returns structured JSON workbench commands for the help overlay.
func (p *jsonToolPage) HelpCommands() []helpCommand {
	return []helpCommand{
		{Key: "r/Enter", Command: "Run Action", Description: "Run the selected JSON action on Input; actions share the same document."},
		{Key: "i", Command: "Edit Input", Description: "Edit Input directly; results preview while typing, Esc returns to Normal."},
		{Key: "o", Command: "Edit Path", Description: "Select Query in the action strip, then edit its Path field."},
		{Key: "x", Command: "Clear Input", Description: "Clear the current input and invalidate the previous result."},
		{Key: "p/a", Command: "Apply Output", Description: "Apply a successful data result to Input. Validation and errors are not data."},
		{Key: "s", Command: "Save Output", Description: "Save output to sharp-output.txt."},
		{Key: "z/ctrl+z", Command: "Undo Input", Description: "Restore Input after editing, applying, pasting, or clearing."},
		{Key: "ctrl+y", Command: "Redo Input", Description: "Redo the last undone input edit."},
		{Key: "ctrl+u/d", Command: "Scroll Output", Description: "Scroll output by half a page; arrows scroll Output in Normal mode."},
		{Key: "alt+u/d", Command: "Scroll Input", Description: "Scroll input by half a page."},
	}
}

// IsEditing reports whether key events should be routed to a text editor.
func (p *jsonToolPage) IsEditing() bool {
	return p.editing
}

func (p *jsonToolPage) EditTarget() string {
	if !p.editing {
		return ""
	}
	if p.field == editFieldOptions {
		return "Path"
	}
	return "Input"
}

// HandleKey handles JSON-specific normal-mode action shortcuts.
func (p *jsonToolPage) HandleKey(key string) bool {
	switch key {
	case "i":
		p.FocusInput()
		p.startEditingFocusedField()
	case "o":
		if p.selectedAction != "json.get" {
			p.status = dimStyle.Render("This action has no options")
			return true
		}
		p.FocusOptions()
		p.startEditingFocusedField()
	case "a":
		p.applyOutput()
	case "z":
		p.undoInput()
		p.Preview()
	default:
		return false
	}
	return true
}

func (p *jsonToolPage) runPathQuery() {
	path := strings.TrimSpace(p.pathBox.Value())
	if path == "" {
		p.FocusOptions()
		p.status = dimStyle.Render("Path empty; press o to edit, then Enter to query")
		return
	}
	p.runAction("json.get", tool.Options{"path": path}, false)
}

func (p *jsonToolPage) runAction(id string, options tool.Options, preview bool) {
	p.revision++
	t, ok := p.tools[id]
	if !ok {
		p.outputText = "missing JSON action: " + id
		p.outputErr = true
		p.renderOutput()
		p.status = errorStyle.Render(p.outputText)
		return
	}
	// 校验只更新状态，不覆盖可继续处理的数据结果。
	if id == "json.validate" {
		if err := jsonutil.RejectDuplicateObjectKeys(p.input.Value()); err != nil {
			p.status = errorStyle.Render(err.Error())
		} else {
			p.status = successStyle.Render("valid JSON")
			p.markToolUsed(id)
		}
		return
	}
	out, err := t.Run(context.Background(), tool.Input{Text: p.input.Value()}, options)
	if err != nil {
		if preview {
			p.stale = true
			p.status = dimStyle.Render("JSON incomplete or invalid; preview paused")
			return
		}
		p.settled()
		p.lastAction = t.Name()
		p.outputText = err.Error()
		p.outputErr = true
		p.renderOutput()
		p.status = errorStyle.Render("failed: " + t.Name())
		return
	}
	p.settled()
	p.lastAction = t.Name()
	p.outputText = out.Text
	p.outputErr = false
	p.renderOutput()
	p.status = workspaceRunStatus(t, preview)
	p.markToolUsed(id)
}

func (p *jsonToolPage) applyOutput() {
	if !p.OutputReady() {
		p.status = errorStyle.Render("Output is stale; update Input/Options or run again")
		return
	}
	if p.outputErr {
		p.status = errorStyle.Render("Cannot apply an error; fix Input and run again")
		return
	}
	text := p.OutputText()
	if text == "" {
		p.status = dimStyle.Render("nothing to apply")
		return
	}
	p.pushInputUndo()
	p.input.SetValue(text)
	p.inputRedo = nil
	p.changed()
	p.refreshInputState()
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
	p.changed()
	p.refreshInputState()
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
	p.changed()
	p.refreshInputState()
	p.status = successStyle.Render("redo input")
}

func (p *jsonToolPage) trackInputChange(before string) {
	after := p.input.Value()
	if after == before {
		return
	}
	p.pushInputUndoValue(before)
	p.inputRedo = nil
	p.changed()
	p.refreshInputState()
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

func (p *jsonToolPage) startEditingFocusedField() {
	p.editing = true
	switch p.field {
	case editFieldOptions:
		p.input.Blur()
		p.pathBox.Focus()
	default:
		p.field = editFieldInput
		p.pathBox.Blur()
		p.input.Focus()
	}
}

func (p *jsonToolPage) refreshInputState() {
	text := strings.TrimSpace(p.input.Value())
	if text == "" {
		p.inputState = "empty"
		p.inputErr = false
		p.status = dimStyle.Render("JSON input empty")
		return
	}
	if err := jsonutil.RejectDuplicateObjectKeys(text); err != nil {
		p.inputState = "invalid JSON"
		p.inputErr = true
		p.status = errorStyle.Render(err.Error())
		return
	}
	p.inputState = "valid JSON"
	p.inputErr = false
	p.status = successStyle.Render("valid JSON")
}
