package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gkmz/sharp/internal/config"
	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/internal/tui"
	"github.com/gkmz/sharp/pkg/tool"
)

func TestThemeStartupPrecedence(t *testing.T) {
	cases := []struct {
		flag, env, saved, want string
	}{
		{flag: "lazygit", env: "sharp", saved: "catppuccin", want: "lazygit"},
		{env: "sharp", saved: "catppuccin", want: "sharp"},
		{saved: "catppuccin", want: "catppuccin"},
		{want: "sharp-color"},
	}
	for _, tc := range cases {
		if got := selectThemeID(tc.flag, tc.env, tc.saved); got != tc.want {
			t.Fatalf("selectThemeID(%q, %q, %q) = %q, want %q", tc.flag, tc.env, tc.saved, got, tc.want)
		}
	}
}

func TestToolSubcommandDoesNotLoadThemeConfig(t *testing.T) {
	loads := 0
	cmd := newRootCommand(tools.NewRegistry(), rootDependencies{
		getenv:     func(string) string { return "lazygit" },
		configPath: func() (string, error) { return "config.yaml", nil },
		loadConfig: func(string) (*config.Config, error) {
			loads++
			return nil, errors.New("must not run")
		},
		runTUI: func(*tool.Registry, tui.Options) error {
			t.Fatal("tool subcommand launched the TUI")
			return nil
		},
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"json", "pretty", `{"a":1}`})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if loads != 0 {
		t.Fatalf("theme config loads = %d, want 0", loads)
	}
	if out.String() != "{\n  \"a\": 1\n}\n" {
		t.Fatalf("tool output = %q", out.String())
	}
}

func TestRootThemeFlagWinsAndSaveCallbackPersists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("theme: catppuccin\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var captured tui.Options
	cmd := newRootCommand(tools.NewRegistry(), rootDependencies{
		getenv:     func(string) string { return "sharp" },
		configPath: func() (string, error) { return path, nil },
		loadConfig: config.Load,
		runTUI: func(_ *tool.Registry, options tui.Options) error {
			captured = options
			return options.SaveTheme("high-contrast")
		},
	})
	cmd.SetArgs([]string{"--theme", "lazygit"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if captured.ActiveTheme.ID != "lazygit" {
		t.Fatalf("active theme = %q", captured.ActiveTheme.ID)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(saved, []byte("theme: high-contrast")) {
		t.Fatalf("saved config = %q", saved)
	}
}

func TestUnknownStartupThemeFallsBackWithAlert(t *testing.T) {
	var captured tui.Options
	cmd := newRootCommand(tools.NewRegistry(), rootDependencies{
		getenv:     func(string) string { return "missing" },
		configPath: func() (string, error) { return filepath.Join(t.TempDir(), "config.yaml"), nil },
		loadConfig: config.Load,
		runTUI: func(_ *tool.Registry, options tui.Options) error {
			captured = options
			return nil
		},
	})
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if captured.ActiveTheme.ID != "sharp-color" || len(captured.StartupAlerts) == 0 {
		t.Fatalf("captured options = active %q alerts %v", captured.ActiveTheme.ID, captured.StartupAlerts)
	}
}
