package convert

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gkmz/sharp/internal/tools/jsonutil"
	"github.com/gkmz/sharp/pkg/tool"
	"gopkg.in/yaml.v3"
)

// Register adds format conversion tools to r.
func Register(r *tool.Registry) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          "json.to_yaml",
		NameValue:        "JSON to YAML",
		CategoryValue:    tool.CategoryConvert,
		DescriptionValue: "Convert JSON to YAML.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			if err := jsonutil.RejectDuplicateObjectKeys(input.Text); err != nil {
				return tool.Output{}, err
			}
			var v any
			if err := json.Unmarshal([]byte(input.Text), &v); err != nil {
				return tool.Output{}, err
			}
			b, err := yaml.Marshal(v)
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b), MimeType: "application/yaml"}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "yaml.to_json",
		NameValue:        "YAML to JSON",
		CategoryValue:    tool.CategoryConvert,
		DescriptionValue: "Convert YAML to JSON.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			if err := ctx.Err(); err != nil {
				return tool.Output{}, err
			}
			doc, err := decodeSingleYAMLDocument(input.Text)
			if err != nil {
				return tool.Output{}, err
			}
			v, err := (&yamlConversion{ctx: ctx, visiting: map[*yaml.Node]bool{}, remaining: 100000, scalarBytes: 16 << 20}).value(&doc)
			if err != nil {
				return tool.Output{}, err
			}
			b, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b), MimeType: "application/json"}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "csv.to_json",
		NameValue:        "CSV to JSON",
		CategoryValue:    tool.CategoryConvert,
		DescriptionValue: "Convert CSV with header row to JSON array.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			rows, err := csv.NewReader(strings.NewReader(input.Text)).ReadAll()
			if err != nil {
				return tool.Output{}, err
			}
			if len(rows) == 0 {
				return tool.Output{Text: "[]", MimeType: "application/json"}, nil
			}
			header, err := normalizeCSVHeader(rows[0])
			if err != nil {
				return tool.Output{}, err
			}
			var out []map[string]string
			for _, row := range rows[1:] {
				obj := map[string]string{}
				for i, h := range header {
					if i < len(row) {
						obj[h] = row[i]
					}
				}
				out = append(out, obj)
			}
			b, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b), MimeType: "application/json"}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "headers.to_json",
		NameValue:        "HTTP Headers to JSON",
		CategoryValue:    tool.CategoryConvert,
		DescriptionValue: "Convert HTTP headers to JSON.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			out := map[string]any{}
			canonicalNames := map[string]string{}
			for _, line := range strings.Split(input.Text, "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				parts := strings.SplitN(line, ":", 2)
				if len(parts) != 2 {
					return tool.Output{}, fmt.Errorf("invalid header line: %s", line)
				}
				name := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if name == "" {
					return tool.Output{}, fmt.Errorf("header name cannot be empty")
				}
				name = canonicalHeaderName(name, canonicalNames)
				appendHeaderValue(out, name, value)
			}
			b, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b), MimeType: "application/json"}, nil
		},
	})
}

func decodeSingleYAMLDocument(input string) (yaml.Node, error) {
	decoder := yaml.NewDecoder(strings.NewReader(input))
	var doc yaml.Node
	if err := decoder.Decode(&doc); err != nil {
		if err == io.EOF {
			return yaml.Node{}, nil
		}
		return yaml.Node{}, err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != nil {
		if err == io.EOF {
			return doc, nil
		}
		return yaml.Node{}, err
	}
	// JSON 只有一个根值；多文档 YAML 若只取第一段会静默丢数据，因此直接拒绝。
	return yaml.Node{}, fmt.Errorf("YAML stream contains multiple documents")
}

type yamlConversion struct {
	ctx         context.Context
	visiting    map[*yaml.Node]bool
	remaining   int
	scalarBytes int
}

// Budget every expanded visit, including aliases, to bound CPU and output size.
func (c *yamlConversion) enter(n *yaml.Node) error {
	if err := c.ctx.Err(); err != nil {
		return err
	}
	c.remaining--
	if c.remaining < 0 {
		return fmt.Errorf("YAML expansion limit exceeded")
	}
	if len(c.visiting) >= 256 {
		return fmt.Errorf("YAML nesting limit exceeded")
	}
	if n.Kind == yaml.ScalarNode {
		c.scalarBytes -= len(n.Value)
		if c.scalarBytes < 0 {
			return fmt.Errorf("YAML expanded scalar byte limit exceeded")
		}
	}
	if c.visiting[n] {
		return fmt.Errorf("YAML alias cycle is not supported")
	}
	c.visiting[n] = true
	return nil
}

func (c *yamlConversion) value(n *yaml.Node) (any, error) {
	if n == nil {
		return nil, nil
	}
	if err := c.enter(n); err != nil {
		return nil, err
	}
	defer delete(c.visiting, n)

	switch n.Kind {
	case 0:
		return nil, nil
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, nil
		}
		return c.value(n.Content[0])
	case yaml.MappingNode:
		if len(n.Content) > c.remaining {
			return nil, fmt.Errorf("YAML expansion limit exceeded")
		}
		out := make(map[string]any, len(n.Content)/2)
		explicit := make(map[string]bool, len(n.Content)/2)
		var merge *yaml.Node
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i]
			if key.Kind == yaml.ScalarNode && key.Tag == "!!merge" {
				if merge != nil {
					return nil, fmt.Errorf("duplicate YAML merge key")
				}
				merge = n.Content[i+1]
				continue
			}
			// YAML 允许数字、布尔等标量作为 map key；转换为 JSON 时必须先规范化为字符串 key。
			normalizedKey, err := c.key(key)
			if err != nil {
				return nil, err
			}
			if explicit[normalizedKey] {
				return nil, fmt.Errorf("YAML map key %q is duplicated after JSON normalization", normalizedKey)
			}
			explicit[normalizedKey] = true
			normalizedValue, err := c.value(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			out[normalizedKey] = normalizedValue
		}
		if merge != nil {
			merged, err := c.value(merge)
			if err != nil {
				return nil, err
			}
			maps := []any{merged}
			if sequence, ok := merged.([]any); ok {
				maps = sequence
			}
			for _, item := range maps {
				mapping, ok := item.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("YAML merge requires a mapping or sequence of mappings")
				}
				for key, value := range mapping {
					if _, exists := out[key]; !exists {
						out[key] = value
					}
				}
			}
		}
		return out, nil
	case yaml.SequenceNode:
		if len(n.Content) > c.remaining {
			return nil, fmt.Errorf("YAML expansion limit exceeded")
		}
		out := make([]any, len(n.Content))
		for i, item := range n.Content {
			normalizedItem, err := c.value(item)
			if err != nil {
				return nil, err
			}
			out[i] = normalizedItem
		}
		return out, nil
	case yaml.ScalarNode:
		var v any
		if err := n.Decode(&v); err != nil {
			return nil, err
		}
		return v, nil
	case yaml.AliasNode:
		return c.value(n.Alias)
	default:
		return nil, fmt.Errorf("unsupported YAML node kind: %d", n.Kind)
	}
}

func (c *yamlConversion) key(n *yaml.Node) (string, error) {
	if n != nil {
		if err := c.enter(n); err != nil {
			return "", err
		}
		defer delete(c.visiting, n)
		if n.Kind == yaml.AliasNode {
			return c.key(n.Alias)
		}
	}
	return yamlScalarKeyToString(n)
}

func yamlScalarKeyToString(n *yaml.Node) (string, error) {
	if n == nil || n.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("unsupported YAML map key type: %s", yamlNodeTypeName(n))
	}
	var key any
	if err := n.Decode(&key); err != nil {
		return "", err
	}
	switch x := key.(type) {
	case string:
		return x, nil
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(x), nil
	default:
		return "", fmt.Errorf("unsupported YAML map key type: %T", key)
	}
}

func yamlNodeTypeName(n *yaml.Node) string {
	if n == nil {
		return "<nil>"
	}
	switch n.Kind {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.MappingNode:
		return "mapping"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	default:
		return fmt.Sprintf("kind %d", n.Kind)
	}
}

func normalizeCSVHeader(header []string) ([]string, error) {
	seen := map[string]bool{}
	normalized := make([]string, len(header))
	for i, name := range header {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("CSV header %d is empty", i+1)
		}
		if seen[name] {
			return nil, fmt.Errorf("CSV header %q is duplicated", name)
		}
		seen[name] = true
		normalized[i] = name
	}
	return normalized, nil
}

func canonicalHeaderName(name string, seen map[string]string) string {
	key := strings.ToLower(name)
	if existing, ok := seen[key]; ok {
		return existing
	}
	// HTTP header 名大小写不敏感；输出使用标准 Canonical 形式，避免同名 header 被拆成多个 JSON 字段。
	canonical := http.CanonicalHeaderKey(name)
	seen[key] = canonical
	return canonical
}

func appendHeaderValue(out map[string]any, name string, value string) {
	existing, ok := out[name]
	if !ok {
		out[name] = value
		return
	}
	switch x := existing.(type) {
	case string:
		out[name] = []string{x, value}
	case []string:
		out[name] = append(x, value)
	}
}
