package tui

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gkmz/sharp/pkg/tool"
)

// optionForm keeps labels outside the editors. Values are passed directly to
// tools, so spaces, quotes and equals signs never require shell-like escaping.
type optionForm struct {
	specs    []tool.Option
	inputs   []textinput.Model
	selected int
	width    int
	labelW   int
}

type optionDraft struct {
	values   tool.Options
	selected int
}

func newOptionForm(specs []tool.Option, draft tool.Options) optionForm {
	f := optionForm{specs: specs}
	for _, spec := range specs {
		input := textinput.New()
		input.Prompt = ""
		input.TextStyle, input.PlaceholderStyle = textStyle, dimStyle
		input.Cursor.Style = focusStyle
		input.Placeholder = "optional"
		if spec.Required {
			input.Placeholder = "required"
		}
		if spec.Secret {
			input.EchoMode = textinput.EchoPassword
			input.EchoCharacter = '*'
		}
		value := spec.Default
		if saved, ok := draft[spec.Name]; ok {
			value = saved
		}
		input.SetValue(value)
		input.Blur()
		f.inputs = append(f.inputs, input)
		f.labelW = max(f.labelW, lipgloss.Width(spec.Name))
	}
	f.SetWidth(60)
	return f
}

func (f *optionForm) SetWidth(width int) {
	f.width = max(1, width)
	for i := range f.inputs {
		f.inputs[i].Width = max(1, f.width-f.labelW-5)
	}
}

func (f *optionForm) Values() tool.Options {
	values := tool.Options{}
	for i, spec := range f.specs {
		values[spec.Name] = f.inputs[i].Value()
	}
	return values
}

func (f *optionForm) Blur() {
	for i := range f.inputs {
		f.inputs[i].Blur()
	}
}

func (f *optionForm) Focus() {
	f.Blur()
	if len(f.inputs) == 0 {
		return
	}
	f.selected = clamp(f.selected, 0, len(f.inputs)-1)
	if !f.HasChoices() {
		f.inputs[f.selected].Focus()
	}
}

func (f *optionForm) HasChoices() bool {
	return len(f.specs) > 0 && len(f.specs[f.selected].Choices) > 0
}

func (f *optionForm) Update(msg tea.Msg) (bool, tea.Cmd) {
	if len(f.inputs) == 0 {
		return false, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "shift+tab":
			delta := 1
			if key.String() == "shift+tab" {
				delta = -1
			}
			f.selected = (f.selected + delta + len(f.inputs)) % len(f.inputs)
			f.Focus()
			return false, nil
		}
		if f.HasChoices() {
			choices := f.specs[f.selected].Choices
			delta := 0
			switch key.String() {
			case "left":
				delta = -1
			case "right":
				delta = 1
			}
			if delta == 0 {
				return false, nil
			}
			index := slices.Index(choices, f.inputs[f.selected].Value())
			value := choices[(index+delta+len(choices))%len(choices)]
			changed := value != f.inputs[f.selected].Value()
			f.inputs[f.selected].SetValue(value)
			return changed, nil
		}
	}
	if f.HasChoices() {
		return false, nil
	}
	before := f.inputs[f.selected].Value()
	input, cmd := f.inputs[f.selected].Update(msg)
	f.inputs[f.selected] = input
	return input.Value() != before, cmd
}

func (f *optionForm) View(editing bool, rows int) string {
	start := 0
	if editing && f.selected >= rows {
		start = f.selected - rows + 1
	}
	lines := []string{}
	for i := start; i < min(len(f.specs), start+rows); i++ {
		spec := f.specs[i]
		active := editing && i == f.selected
		label := spec.Name + strings.Repeat(" ", max(0, f.labelW-lipgloss.Width(spec.Name))) + " = "
		marker := "  "
		labelStyle := dimStyle
		if active {
			marker = "> "
			labelStyle = focusStyle
		}
		value := f.inputs[i].Value()
		if spec.Secret && value != "" {
			value = "<secret>"
		}
		valueWidth := max(1, f.width-f.labelW-5)
		var rendered string
		if len(spec.Choices) > 0 {
			value = "‹ " + value + " ›"
			rendered = textStyle.Render(truncateDisplay(value, valueWidth))
			if active {
				rendered = focusStyle.Render(truncateDisplay(value, valueWidth))
			}
		} else if active {
			rendered = f.inputs[i].View()
		} else {
			if value == "" {
				value = f.inputs[i].Placeholder
			}
			rendered = textStyle.Render(truncateDisplay(value, valueWidth))
		}
		lines = append(lines, labelStyle.Render(marker+label)+rendered)
	}
	return strings.Join(lines, "\n")
}
