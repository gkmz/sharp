package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingConfigReturnsWritableEmptyConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SelectedTheme() != "" || len(cfg.ThemeDefinitions()) != 0 {
		t.Fatalf("loaded config = theme %q, definitions %v", cfg.SelectedTheme(), cfg.ThemeDefinitions())
	}
	if err := cfg.SaveTheme("sharp-color"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "theme: sharp-color\n" {
		t.Fatalf("saved config = %q", data)
	}
}

func TestLoadThemeDefinitions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte("theme: mine\nthemes:\n  mine:\n    extends: sharp-color\n    dark:\n      focus: '#112233'\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	definitions := cfg.ThemeDefinitions()
	if cfg.SelectedTheme() != "mine" || definitions["mine"].Extends != "sharp-color" || definitions["mine"].Dark["focus"] != "#112233" {
		t.Fatalf("decoded config = theme %q, definitions %#v", cfg.SelectedTheme(), definitions)
	}
	definitions["extra"] = definitions["mine"]
	if len(cfg.ThemeDefinitions()) != 1 {
		t.Fatal("ThemeDefinitions exposed mutable configuration state")
	}
}

func TestSaveThemePreservesUnknownKeysAndComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	original := "# user config\ntheme: sharp\nfuture:\n  enabled: true\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SaveTheme("lazygit"); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{[]byte("# user config"), []byte("future:"), []byte("enabled: true"), []byte("theme: lazygit")} {
		if !bytes.Contains(saved, want) {
			t.Fatalf("saved config %q does not contain %q", saved, want)
		}
	}
}

func TestMalformedConfigIsNotOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	original := []byte("theme: [")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load() error = nil, want malformed YAML error")
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(saved, original) {
		t.Fatalf("malformed file changed to %q", saved)
	}
}

func TestSaveThemeReportsRenameFailureWithoutReplacingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	original := []byte("theme: sharp\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.rename = func(string, string) error { return errors.New("rename denied") }
	if err := cfg.SaveTheme("lazygit"); err == nil || !bytes.Contains([]byte(err.Error()), []byte("rename denied")) {
		t.Fatalf("SaveTheme() error = %v", err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(saved, original) {
		t.Fatalf("file changed after rename failure: %q", saved)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".config-*.tmp"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files after failure = %v, error %v", matches, err)
	}
}

func TestDefaultPathUsesInjectedUserConfigDirectory(t *testing.T) {
	old := userConfigDir
	userConfigDir = func() (string, error) { return "/config-home", nil }
	t.Cleanup(func() { userConfigDir = old })

	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/config-home", "sharp", "config.yaml")
	if got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}
