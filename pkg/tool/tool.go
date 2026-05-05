// Package tool defines the runtime contract shared by CLI, TUI, and concrete
// developer tools.
package tool

import (
	"context"
	"fmt"
	"sort"
)

// Category groups related tools for discovery and navigation.
type Category string

const (
	// CategoryJSON contains JSON formatting, querying, and string helpers.
	CategoryJSON Category = "JSON"
	// CategoryEncode contains text encoding and decoding tools.
	CategoryEncode Category = "Encode"
	// CategoryCrypto contains hashing and message authentication tools.
	CategoryCrypto Category = "Crypto"
	// CategoryTime contains timestamp and datetime tools.
	CategoryTime Category = "Time"
	// CategoryText contains text transformation and regex tools.
	CategoryText Category = "Text"
	// CategoryNetwork contains URL, query string, DNS, and CIDR tools.
	CategoryNetwork Category = "Network"
	// CategoryGenerator contains random or generated value tools.
	CategoryGenerator Category = "Generator"
	// CategoryConvert contains cross-format conversion tools.
	CategoryConvert Category = "Convert"
	// CategoryInspector contains parsers and read-only inspection tools.
	CategoryInspector Category = "Inspector"
)

// Input is the text payload passed to a Tool.
type Input struct {
	Text string
}

// Output is the text payload returned by a Tool.
type Output struct {
	Text string
	// MimeType optionally identifies structured output, such as JSON or YAML.
	MimeType string
}

// Option describes one named runtime parameter accepted by a Tool.
type Option struct {
	Name        string
	Description string
	Default     string
	Required    bool
	Secret      bool
}

// Options stores runtime option values keyed by option name.
type Options map[string]string

// Tool is the public contract implemented by every executable tool.
type Tool interface {
	// ID returns the stable command identifier, such as "json.pretty".
	ID() string
	// Name returns the human-readable display name.
	Name() string
	// Category returns the top-level navigation group.
	Category() Category
	// Description returns a short user-facing summary.
	Description() string
	// Options returns supported runtime parameters.
	Options() []Option
	// Run executes the tool against input and option values.
	Run(ctx context.Context, input Input, options Options) (Output, error)
}

// SimpleTool is a lightweight Tool implementation backed by function fields.
type SimpleTool struct {
	IDValue          string
	NameValue        string
	CategoryValue    Category
	DescriptionValue string
	OptionsValue     []Option
	RunFunc          func(ctx context.Context, input Input, options Options) (Output, error)
}

// ID returns the stable command identifier.
func (t SimpleTool) ID() string {
	return t.IDValue
}

// Name returns the human-readable display name.
func (t SimpleTool) Name() string {
	return t.NameValue
}

// Category returns the top-level navigation group.
func (t SimpleTool) Category() Category {
	return t.CategoryValue
}

// Description returns a short user-facing summary.
func (t SimpleTool) Description() string {
	return t.DescriptionValue
}

// Options returns supported runtime parameters.
func (t SimpleTool) Options() []Option {
	return t.OptionsValue
}

// Run executes the tool function.
func (t SimpleTool) Run(ctx context.Context, input Input, options Options) (Output, error) {
	return t.RunFunc(ctx, input, options)
}

// Registry stores tools by ID and provides deterministic listing helpers.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry and rejects nil, empty, or duplicate IDs.
func (r *Registry) Register(t Tool) error {
	if t == nil {
		return fmt.Errorf("cannot register nil tool")
	}
	if t.ID() == "" {
		return fmt.Errorf("cannot register tool with empty id")
	}
	if _, exists := r.tools[t.ID()]; exists {
		return fmt.Errorf("tool %q already registered", t.ID())
	}
	r.tools[t.ID()] = t
	return nil
}

// MustRegister adds a tool to the registry or panics.
func (r *Registry) MustRegister(t Tool) {
	if err := r.Register(t); err != nil {
		panic(err)
	}
}

// Get returns the tool for id when registered.
func (r *Registry) Get(id string) (Tool, bool) {
	t, ok := r.tools[id]
	return t, ok
}

// All returns every registered tool sorted by category and display name.
func (r *Registry) All() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category() == out[j].Category() {
			return out[i].Name() < out[j].Name()
		}
		return out[i].Category() < out[j].Category()
	})
	return out
}

// ByCategory returns tools in a single category using the same ordering as All.
func (r *Registry) ByCategory(category Category) []Tool {
	var out []Tool
	for _, t := range r.All() {
		if t.Category() == category {
			out = append(out, t)
		}
	}
	return out
}
