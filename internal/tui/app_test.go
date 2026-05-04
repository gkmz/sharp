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

func TestGenericToolPageWrapsRenderedOutput(t *testing.T) {
	page := newGenericToolPage(tool.SimpleTool{
		IDValue:          "test.echo",
		NameValue:        "Echo",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "test echo",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJK"}, nil
		},
	})
	page.SetSize(10, 8)
	page.Run()

	if got := page.output.TotalLineCount(); got < 3 {
		t.Fatalf("TotalLineCount() = %d, want wrapped output across at least 3 lines", got)
	}
	if got, want := page.OutputText(), "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJK"; got != want {
		t.Fatalf("OutputText() = %q, want raw output %q", got, want)
	}
}

func TestWrapDisplayTextUsesTerminalWidth(t *testing.T) {
	if got, want := wrapDisplayText("中文abcd", 4), "中文\nabcd"; got != want {
		t.Fatalf("wrapDisplayText() = %q, want %q", got, want)
	}
}
