package tool

import (
	"context"
	"fmt"
	"sort"
)

type Category string

const (
	CategoryJSON      Category = "JSON"
	CategoryEncode    Category = "Encode"
	CategoryCrypto    Category = "Crypto"
	CategoryTime      Category = "Time"
	CategoryText      Category = "Text"
	CategoryNetwork   Category = "Network"
	CategoryGenerator Category = "Generator"
	CategoryConvert   Category = "Convert"
	CategoryInspector Category = "Inspector"
)

type Input struct {
	Text string
}

type Output struct {
	Text     string
	MimeType string
}

type Option struct {
	Name        string
	Description string
	Default     string
	Required    bool
	Secret      bool
}

type Options map[string]string

type Tool interface {
	ID() string
	Name() string
	Category() Category
	Description() string
	Options() []Option
	Run(ctx context.Context, input Input, options Options) (Output, error)
}

type SimpleTool struct {
	IDValue          string
	NameValue        string
	CategoryValue    Category
	DescriptionValue string
	OptionsValue     []Option
	RunFunc          func(ctx context.Context, input Input, options Options) (Output, error)
}

func (t SimpleTool) ID() string {
	return t.IDValue
}

func (t SimpleTool) Name() string {
	return t.NameValue
}

func (t SimpleTool) Category() Category {
	return t.CategoryValue
}

func (t SimpleTool) Description() string {
	return t.DescriptionValue
}

func (t SimpleTool) Options() []Option {
	return t.OptionsValue
}

func (t SimpleTool) Run(ctx context.Context, input Input, options Options) (Output, error) {
	return t.RunFunc(ctx, input, options)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

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

func (r *Registry) MustRegister(t Tool) {
	if err := r.Register(t); err != nil {
		panic(err)
	}
}

func (r *Registry) Get(id string) (Tool, bool) {
	t, ok := r.tools[id]
	return t, ok
}

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

func (r *Registry) ByCategory(category Category) []Tool {
	var out []Tool
	for _, t := range r.All() {
		if t.Category() == category {
			out = append(out, t)
		}
	}
	return out
}
