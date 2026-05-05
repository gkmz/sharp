package convert

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gkmz/sharp/pkg/tool"
	"gopkg.in/yaml.v3"
)

func Register(r *tool.Registry) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          "json.to_yaml",
		NameValue:        "JSON to YAML",
		CategoryValue:    tool.CategoryConvert,
		DescriptionValue: "Convert JSON to YAML.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
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
			var v any
			if err := yaml.Unmarshal([]byte(input.Text), &v); err != nil {
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
			header := rows[0]
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
			out := map[string]string{}
			for _, line := range strings.Split(input.Text, "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				parts := strings.SplitN(line, ":", 2)
				if len(parts) != 2 {
					return tool.Output{}, fmt.Errorf("invalid header line: %s", line)
				}
				out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
			b, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b), MimeType: "application/json"}, nil
		},
	})
}
