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
