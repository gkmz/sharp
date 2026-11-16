package tui

import (
	"context"
	"strings"
	"testing"

	sharptheme "github.com/gkmz/sharp/internal/theme"
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
