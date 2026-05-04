package tui

import (
	"context"
	"testing"

	"github.com/hank/sharp/pkg/tool"
)

func TestGenericToolPageOutputTextUsesTrimmedRawOutput(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "test.echo",
		NameValue:        "Echo",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "test echo",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: "\n  alpha\nbeta  \n\n"}, nil
		},
	})
	page.SetSize(20, 8)
	page.Run()

	if got, want := page.OutputText(), "alpha\nbeta"; got != want {
		t.Fatalf("OutputText() = %q, want %q", got, want)
	}
	if got := page.output.View(); got == page.OutputText() {
		t.Fatalf("viewport view unexpectedly matched raw output; test should guard against copying rendered output")
	}
}

func TestGenericToolPageClearOutputClearsRawOutput(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "test.echo",
		NameValue:        "Echo",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "test echo",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: "result"}, nil
		},
	})
	page.Run()
	page.ClearOutput()

	if got := page.OutputText(); got != "" {
		t.Fatalf("OutputText() after ClearOutput() = %q, want empty", got)
	}
}
