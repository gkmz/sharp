package theme

import (
	"slices"
	"testing"
)

func TestBuiltinsAreCompleteAndStable(t *testing.T) {
	registry, diagnostics := NewRegistry(nil)
	if len(diagnostics) != 0 {
		t.Fatalf("NewRegistry() diagnostics = %v", diagnostics)
	}

	want := []string{"sharp", "sharp-color", "lazygit", "catppuccin", "high-contrast"}
	got := themeIDs(registry.All())
	if !slices.Equal(got, want) {
		t.Fatalf("theme IDs = %v, want %v", got, want)
	}
	for _, current := range registry.All() {
		if err := current.Light.Validate(); err != nil {
			t.Fatalf("%s light palette: %v", current.ID, err)
		}
		if err := current.Dark.Validate(); err != nil {
			t.Fatalf("%s dark palette: %v", current.ID, err)
		}
	}
}

func TestCustomThemeInheritsAndInvalidThemesAreIsolated(t *testing.T) {
	registry, diagnostics := NewRegistry(map[string]Definition{
		"mine": {
			Extends: "sharp-color",
			Dark:    map[string]string{"focus": "#112233"},
		},
		"bad-color": {
			Extends: "sharp-color",
			Dark:    map[string]string{"focus": "red"},
		},
		"bad-role": {
			Extends: "sharp-color",
			Light:   map[string]string{"unknown": "#112233"},
		},
	})

	mine, ok := registry.Lookup("mine")
	if !ok {
		t.Fatal("custom theme was not registered")
	}
	base, _ := registry.Lookup("sharp-color")
	if mine.Dark.Focus != "#112233" {
		t.Fatalf("mine dark focus = %q", mine.Dark.Focus)
	}
	if mine.Dark.Text != base.Dark.Text || mine.Light != base.Light {
		t.Fatal("custom theme did not inherit the untouched base palette")
	}
	if _, ok := registry.Lookup("bad-color"); ok {
		t.Fatal("invalid color theme was registered")
	}
	if _, ok := registry.Lookup("bad-role"); ok {
		t.Fatal("unknown role theme was registered")
	}
	if len(diagnostics) != 2 {
		t.Fatalf("diagnostics = %v, want 2", diagnostics)
	}
}

func TestCustomThemeValidationDoesNotMutateBuiltins(t *testing.T) {
	first, _ := NewRegistry(map[string]Definition{
		"mine": {Extends: "sharp", Dark: map[string]string{"text": "#010203"}},
	})
	second, _ := NewRegistry(nil)

	firstSharp, _ := first.Lookup("sharp")
	secondSharp, _ := second.Lookup("sharp")
	if firstSharp != secondSharp {
		t.Fatal("custom theme resolution mutated a built-in theme")
	}
}

func TestCustomThemesAreSortedAfterBuiltins(t *testing.T) {
	registry, diagnostics := NewRegistry(map[string]Definition{
		"zebra": {Extends: "sharp"},
		"amber": {Extends: "sharp"},
	})
	if len(diagnostics) != 0 {
		t.Fatal(diagnostics)
	}
	want := []string{"sharp", "sharp-color", "lazygit", "catppuccin", "high-contrast", "amber", "zebra"}
	if got := themeIDs(registry.All()); !slices.Equal(got, want) {
		t.Fatalf("theme IDs = %v, want %v", got, want)
	}
}

func TestCustomThemeRejectsInvalidIDBaseAndBuiltinReplacement(t *testing.T) {
	registry, diagnostics := NewRegistry(map[string]Definition{
		"Bad ID":       {Extends: "sharp"},
		"missing-base": {Extends: "not-installed"},
		"sharp":        {Extends: "sharp-color"},
	})
	if got := len(registry.All()); got != 5 {
		t.Fatalf("registered themes = %d, want only 5 built-ins", got)
	}
	if len(diagnostics) != 3 {
		t.Fatalf("diagnostics = %v, want 3", diagnostics)
	}
}

func themeIDs(themes []Theme) []string {
	ids := make([]string, 0, len(themes))
	for _, current := range themes {
		ids = append(ids, current.ID)
	}
	return ids
}
