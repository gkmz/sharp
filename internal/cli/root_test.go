package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gkmz/sharp/internal/tools"
)

func TestHelpIncludesLogo(t *testing.T) {
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"local dev toolkit", "Usage:", "Available Commands:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output = %q, want %q", got, want)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	oldVersion := Version
	Version = "1.2.3"
	defer func() {
		Version = oldVersion
	}()

	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"local dev toolkit", "version: 1.2.3"} {
		if !strings.Contains(got, want) {
			t.Fatalf("version output = %q, want %q", got, want)
		}
	}
	for _, unwanted := range []string{"commit:", "date:"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("version output = %q, should not contain %q", got, unwanted)
		}
	}
}

func TestRuntimeErrorsDoNotPrintUsage(t *testing.T) {
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"password", "--length", "abc"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() returned nil error, want invalid length error")
	}
	if got := err.Error(); got != "length must be an integer" {
		t.Fatalf("error = %q, want invalid length message", got)
	}
	if got := out.String(); strings.Contains(got, "Usage:") || strings.Contains(got, "Error:") {
		t.Fatalf("output = %q, should not include cobra usage or duplicate error", got)
	}
}
