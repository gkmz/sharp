package theme

import (
	"fmt"
	"regexp"
	"sort"
)

var (
	themeIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	colorPattern   = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// Palette assigns a terminal color to each semantic interface role.
type Palette struct {
	Text                string
	Muted               string
	Border              string
	Focus               string
	Accent              string
	Category            string
	Group               string
	Action              string
	Shortcut            string
	SelectionForeground string
	SelectionBackground string
	RetainedForeground  string
	RetainedBackground  string
	Success             string
	Warning             string
	Error               string
}

// Validate reports the first missing or malformed semantic color.
func (p Palette) Validate() error {
	for role, color := range p.values() {
		if !colorPattern.MatchString(color) {
			return fmt.Errorf("%s must use #RRGGBB, got %q", role, color)
		}
	}
	return nil
}

func (p Palette) values() map[string]string {
	return map[string]string{
		"text": p.Text, "muted": p.Muted, "border": p.Border,
		"focus": p.Focus, "accent": p.Accent, "category": p.Category,
		"group": p.Group, "action": p.Action, "shortcut": p.Shortcut,
		"selection_foreground": p.SelectionForeground,
		"selection_background": p.SelectionBackground,
		"retained_foreground":  p.RetainedForeground,
		"retained_background":  p.RetainedBackground,
		"success":              p.Success, "warning": p.Warning, "error": p.Error,
	}
}

func (p *Palette) apply(overrides map[string]string) error {
	roles := make([]string, 0, len(overrides))
	for role := range overrides {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		color := overrides[role]
		if !colorPattern.MatchString(color) {
			return fmt.Errorf("%s must use #RRGGBB, got %q", role, color)
		}
		switch role {
		case "text":
			p.Text = color
		case "muted":
			p.Muted = color
		case "border":
			p.Border = color
		case "focus":
			p.Focus = color
		case "accent":
			p.Accent = color
		case "category":
			p.Category = color
		case "group":
			p.Group = color
		case "action":
			p.Action = color
		case "shortcut":
			p.Shortcut = color
		case "selection_foreground":
			p.SelectionForeground = color
		case "selection_background":
			p.SelectionBackground = color
		case "retained_foreground":
			p.RetainedForeground = color
		case "retained_background":
			p.RetainedBackground = color
		case "success":
			p.Success = color
		case "warning":
			p.Warning = color
		case "error":
			p.Error = color
		default:
			return fmt.Errorf("unknown color role %q", role)
		}
	}
	return nil
}

// Theme is a named pair of complete light and dark palettes.
type Theme struct {
	ID    string
	Name  string
	Light Palette
	Dark  Palette
}

// Definition describes a custom theme as partial overrides of one built-in.
type Definition struct {
	Extends string            `yaml:"extends"`
	Light   map[string]string `yaml:"light,omitempty"`
	Dark    map[string]string `yaml:"dark,omitempty"`
}

// Diagnostic excludes one invalid custom theme without rejecting its peers.
type Diagnostic struct {
	ThemeID string
	Message string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("theme %q: %s", d.ThemeID, d.Message)
}

// Registry contains immutable resolved themes in display order.
type Registry struct {
	themes map[string]Theme
	order  []string
}

// NewRegistry returns all built-ins plus independently validated custom themes.
func NewRegistry(definitions map[string]Definition) (*Registry, []Diagnostic) {
	builtins := builtInThemes()
	r := &Registry{themes: make(map[string]Theme, len(builtins)+len(definitions))}
	for _, current := range builtins {
		r.themes[current.ID] = current
		r.order = append(r.order, current.ID)
	}

	ids := make([]string, 0, len(definitions))
	for id := range definitions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	diagnostics := make([]Diagnostic, 0)
	for _, id := range ids {
		definition := definitions[id]
		if !themeIDPattern.MatchString(id) {
			diagnostics = append(diagnostics, Diagnostic{id, "ID must contain lowercase letters, numbers, and hyphens"})
			continue
		}
		if _, exists := r.themes[id]; exists {
			diagnostics = append(diagnostics, Diagnostic{id, "cannot replace a built-in theme"})
			continue
		}
		base, exists := r.themes[definition.Extends]
		if !exists || !isBuiltInID(definition.Extends) {
			diagnostics = append(diagnostics, Diagnostic{id, fmt.Sprintf("extends unknown built-in %q", definition.Extends)})
			continue
		}
		resolved := Theme{ID: id, Name: id, Light: base.Light, Dark: base.Dark}
		if err := resolved.Light.apply(definition.Light); err != nil {
			diagnostics = append(diagnostics, Diagnostic{id, "light palette: " + err.Error()})
			continue
		}
		if err := resolved.Dark.apply(definition.Dark); err != nil {
			diagnostics = append(diagnostics, Diagnostic{id, "dark palette: " + err.Error()})
			continue
		}
		r.themes[id] = resolved
		r.order = append(r.order, id)
	}
	return r, diagnostics
}

// Lookup resolves a stable theme ID.
func (r *Registry) Lookup(id string) (Theme, bool) {
	if r == nil {
		return Theme{}, false
	}
	current, ok := r.themes[id]
	return current, ok
}

// All returns themes in stable picker order.
func (r *Registry) All() []Theme {
	if r == nil {
		return nil
	}
	all := make([]Theme, 0, len(r.order))
	for _, id := range r.order {
		all = append(all, r.themes[id])
	}
	return all
}

func isBuiltInID(id string) bool {
	for _, current := range builtInThemes() {
		if current.ID == id {
			return true
		}
	}
	return false
}
