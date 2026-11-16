package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	sharptheme "github.com/gkmz/sharp/internal/theme"
	sharptools "github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/pkg/tool"
)

func TestStyleSetsAreThemeSpecific(t *testing.T) {
	registry, diagnostics := sharptheme.NewRegistry(nil)
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	sharp, _ := registry.Lookup("sharp")
	lazygit, _ := registry.Lookup("lazygit")
	first := NewStyleSet(sharp)
	second := NewStyleSet(lazygit)
	if first.Category.GetForeground() == second.Category.GetForeground() {
		t.Fatal("different themes produced the same category style")
	}
	if first.SelectedRow.GetBackground() == second.SelectedRow.GetBackground() {
		t.Fatal("different themes produced the same selection background")
	}
}

func TestThemePickerPreviewsCancelsAndPersists(t *testing.T) {
	registry, _ := sharptheme.NewRegistry(nil)
	active, _ := registry.Lookup("sharp-color")
	var saved []string
	m := newModelWithOptions(sharptools.NewRegistry(), Options{
		Themes:      registry,
		ActiveTheme: active,
		SaveTheme: func(id string) error {
			saved = append(saved, id)
			return nil
		},
	})

	m = workflowKey(m, "t")
	if !m.themePicker.open || m.themePicker.originalID != "sharp-color" {
		t.Fatalf("picker after open = %#v", m.themePicker)
	}
	m = workflowKey(m, "down")
	preview := m.activeThemeID
	if preview == "sharp-color" || len(saved) != 0 {
		t.Fatalf("preview = %q, saved = %v", preview, saved)
	}
	m = workflowKey(m, "esc")
	if m.themePicker.open || m.activeThemeID != "sharp-color" || len(saved) != 0 {
		t.Fatalf("cancel left picker=%v active=%q saved=%v", m.themePicker.open, m.activeThemeID, saved)
	}

	m = workflowKey(m, "t")
	m = workflowKey(m, "down")
	want := m.activeThemeID
	m = workflowKey(m, "enter")
	if m.themePicker.open || len(saved) != 1 || saved[0] != want {
		t.Fatalf("confirm left picker=%v active=%q saved=%v", m.themePicker.open, m.activeThemeID, saved)
	}
}

func TestThemePickerSaveFailureKeepsPreview(t *testing.T) {
	registry, _ := sharptheme.NewRegistry(nil)
	active, _ := registry.Lookup("sharp-color")
	m := newModelWithOptions(sharptools.NewRegistry(), Options{
		Themes:      registry,
		ActiveTheme: active,
		SaveTheme: func(string) error {
			return errors.New("disk full")
		},
	})
	m = workflowKey(m, "t")
	m = workflowKey(m, "down")
	preview := m.activeThemeID
	m = workflowKey(m, "enter")
	if m.themePicker.open || m.activeThemeID != preview {
		t.Fatalf("save failure reverted active theme: picker=%v active=%q", m.themePicker.open, m.activeThemeID)
	}
	if !strings.Contains(m.status, "disk full") || m.statusKind != statusError {
		t.Fatalf("save failure status = %q kind=%v", m.status, m.statusKind)
	}
}

func TestThemePickerDoesNotOpenWhileEditing(t *testing.T) {
	registry, _ := sharptheme.NewRegistry(nil)
	active, _ := registry.Lookup("sharp-color")
	m := newModelWithOptions(sharptools.NewRegistry(), Options{Themes: registry, ActiveTheme: active})
	m.focus = focusWorkspace
	m = workflowKey(m, "i")
	m = workflowKey(m, "t")
	if m.themePicker.open {
		t.Fatal("theme picker opened while a workspace editor was active")
	}
	if page := m.currentPage(); page == nil || !page.IsEditing() {
		t.Fatal("theme shortcut interrupted workspace editing")
	}
}

func TestPageApplyStylesPreservesWorkspaceState(t *testing.T) {
	registry, _ := sharptheme.NewRegistry(nil)
	sharp, _ := registry.Lookup("sharp")
	lazygit, _ := registry.Lookup("lazygit")
	page := newGenericToolPageWithStyles(tool.SimpleTool{
		IDValue:   "test.echo",
		NameValue: "Echo",
		RunFunc: func(_ context.Context, _ tool.Input, _ tool.Options) (tool.Output, error) {
			return tool.Output{Text: "result"}, nil
		},
	}, NewStyleSet(sharp))
	page.SetInput("keep me")
	page.Run()
	beforeOutput := page.OutputText()
	beforeRevision := page.previewVersion()
	if strings.Contains(page.status.text, "\x1b[") {
		t.Fatal("page status cached ANSI styling")
	}

	page.ApplyStyles(NewStyleSet(lazygit))

	if page.input.Value() != "keep me" {
		t.Fatalf("input after theme change = %q", page.input.Value())
	}
	if page.OutputText() != beforeOutput || page.previewVersion() != beforeRevision {
		t.Fatal("theme change altered output or workspace revision")
	}
	if page.styles.Category.GetForeground() != NewStyleSet(lazygit).Category.GetForeground() {
		t.Fatal("page did not accept the new styles")
	}
}
