package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type editField int

const (
	editFieldInput editField = iota
	editFieldOptions
)

// 每次修改与显式操作都会推进版本，迟到的预览不能覆盖新结果。
type workspacePreview struct {
	revision uint64
	stale    bool
}

type workspacePreviewMsg struct {
	page     toolPage
	revision uint64
}

func (p *workspacePreview) changed()               { p.revision++; p.stale = true }
func (p *workspacePreview) settled()               { p.revision++; p.stale = false }
func (p *workspacePreview) OutputReady() bool      { return !p.stale }
func (p *workspacePreview) previewVersion() uint64 { return p.revision }
func (p *workspacePreview) schedule(page toolPage) tea.Cmd {
	revision := p.revision
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return workspacePreviewMsg{page: page, revision: revision}
	})
}
func (p *workspacePreview) title(action string) string {
	title := workspaceOutputTitle(action)
	if p.stale {
		title += " · stale"
	}
	return title
}

// 两种页面使用同一尺寸预算，参数只占一行内容，其余空间分配给文档。
func workspaceSizes(width, height int, hasInput, hasOptions bool) (pageWidth, inputH, optionsH, outputH int) {
	pageWidth = max(8, width)
	if hasOptions {
		optionsH = 3
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

func workspaceInputView(input textarea.Model, active bool) string {
	if !active {
		input.Blur()
	}
	return input.View()
}

func workspaceOutputTitle(action string) string {
	if action == "" {
		return "Output"
	}
	return "Output · " + action
}

func workspaceFooter(autoPreview, hasInput, hasOptions, editing bool, field editField, optionsTitle string) string {
	if editing {
		if field == editFieldOptions {
			return statusLine("EDIT "+optionsTitle, statusHint("Enter", "run"), statusHint("Esc", "done"))
		}
		return statusLine("EDIT Input", statusHint("Esc", "done"))
	}
	hints := []string{}
	if !autoPreview {
		hints = append(hints, statusHint("r/Enter", "run"))
	}
	if hasInput {
		hints = append(hints, statusHint("v", "paste"), statusHint("i", "edit"), statusHint("x", "clear"))
	}
	if hasOptions {
		hints = append(hints, statusHint("o", "options"))
	}
	hints = append(hints, statusHint("y", "copy"))
	if hasInput {
		hints = append(hints, statusHint("p", "apply"))
	}
	hints = append(hints, statusHint("?", "help"))
	return statusLine("NORMAL", hints...)
}
