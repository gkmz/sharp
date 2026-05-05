package json

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/gkmz/sharp/pkg/tool"
	"github.com/tidwall/gjson"
)

// Register adds JSON formatting, querying, and string helper tools to r.
func Register(r *tool.Registry) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          "json.pretty",
		NameValue:        "Pretty JSON",
		CategoryValue:    tool.CategoryJSON,
		DescriptionValue: "Format JSON with indentation.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			var buf bytes.Buffer
			if err := json.Indent(&buf, []byte(input.Text), "", "  "); err != nil {
				return tool.Output{}, fmt.Errorf("invalid JSON: %w", err)
			}
			return tool.Output{Text: buf.String(), MimeType: "application/json"}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "json.minify",
		NameValue:        "Minify JSON",
		CategoryValue:    tool.CategoryJSON,
		DescriptionValue: "Compact JSON into one line.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			var buf bytes.Buffer
			if err := json.Compact(&buf, []byte(input.Text)); err != nil {
				return tool.Output{}, fmt.Errorf("invalid JSON: %w", err)
			}
			return tool.Output{Text: buf.String(), MimeType: "application/json"}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "json.validate",
		NameValue:        "Validate JSON",
		CategoryValue:    tool.CategoryJSON,
		DescriptionValue: "Validate JSON syntax.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			if !json.Valid([]byte(input.Text)) {
				return tool.Output{}, fmt.Errorf("invalid JSON")
			}
			return tool.Output{Text: "valid JSON"}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "json.get",
		NameValue:        "Query JSON",
		CategoryValue:    tool.CategoryJSON,
		DescriptionValue: "Query JSON with gjson path syntax.",
		OptionsValue: []tool.Option{{
			Name:        "path",
			Description: "gjson path, for example data.user.name",
			Required:    true,
		}},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			path := options["path"]
			if path == "" {
				return tool.Output{}, fmt.Errorf("missing required option: path")
			}
			if !json.Valid([]byte(input.Text)) {
				return tool.Output{}, fmt.Errorf("invalid JSON")
			}
			result := gjson.Get(input.Text, path)
			if !result.Exists() {
				return tool.Output{}, fmt.Errorf("path not found: %s", path)
			}
			if result.IsArray() || result.IsObject() {
				var buf bytes.Buffer
				if err := json.Indent(&buf, []byte(result.Raw), "", "  "); err != nil {
					return tool.Output{Text: result.Raw, MimeType: "application/json"}, nil
				}
				return tool.Output{Text: buf.String(), MimeType: "application/json"}, nil
			}
			return tool.Output{Text: result.String()}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "json.sort",
		NameValue:        "Sort JSON Keys",
		CategoryValue:    tool.CategoryJSON,
		DescriptionValue: "Sort JSON object keys recursively.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			var v any
			if err := json.Unmarshal([]byte(input.Text), &v); err != nil {
				return tool.Output{}, fmt.Errorf("invalid JSON: %w", err)
			}
			v = sortValue(v)
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b), MimeType: "application/json"}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "json.escape",
		NameValue:        "Escape JSON String",
		CategoryValue:    tool.CategoryJSON,
		DescriptionValue: "Escape text as a JSON string.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			b, err := json.Marshal(input.Text)
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b)}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "json.unescape",
		NameValue:        "Unescape JSON String",
		CategoryValue:    tool.CategoryJSON,
		DescriptionValue: "Unescape a JSON string literal.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			var s string
			if err := json.Unmarshal([]byte(input.Text), &s); err != nil {
				return tool.Output{}, fmt.Errorf("invalid JSON string: %w", err)
			}
			return tool.Output{Text: s}, nil
		},
	})
}

func sortValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(x))
		for _, k := range keys {
			out[k] = sortValue(x[k])
		}
		return out
	case []any:
		for i := range x {
			x[i] = sortValue(x[i])
		}
	}
	return v
}
