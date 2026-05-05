package text

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gkmz/sharp/pkg/tool"
)

// Register adds text transformation, counting, sorting, and regex tools to r.
func Register(r *tool.Registry) {
	textTool(r, "text.upper", "Uppercase", "Convert text to uppercase.", strings.ToUpper)
	textTool(r, "text.lower", "Lowercase", "Convert text to lowercase.", strings.ToLower)
	textTool(r, "text.trim", "Trim", "Trim leading and trailing whitespace.", strings.TrimSpace)
	textTool(r, "text.snake", "snake_case", "Convert text to snake_case.", func(s string) string { return strings.Join(words(s), "_") })
	textTool(r, "text.kebab", "kebab-case", "Convert text to kebab-case.", func(s string) string { return strings.Join(words(s), "-") })
	textTool(r, "text.camel", "camelCase", "Convert text to camelCase.", camel)
	textTool(r, "text.pascal", "PascalCase", "Convert text to PascalCase.", pascal)

	r.MustRegister(tool.SimpleTool{
		IDValue:          "text.sort",
		NameValue:        "Sort Lines",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "Sort lines alphabetically.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			lines := strings.Split(strings.ReplaceAll(input.Text, "\r\n", "\n"), "\n")
			sort.Strings(lines)
			return tool.Output{Text: strings.Join(lines, "\n")}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "text.uniq",
		NameValue:        "Unique Lines",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "Remove duplicate lines while preserving first occurrence.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			seen := map[string]bool{}
			var out []string
			for _, line := range strings.Split(strings.ReplaceAll(input.Text, "\r\n", "\n"), "\n") {
				if !seen[line] {
					seen[line] = true
					out = append(out, line)
				}
			}
			return tool.Output{Text: strings.Join(out, "\n")}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "text.count",
		NameValue:        "Count Text",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "Count bytes, runes, words, and lines.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			lineCount := 0
			if input.Text != "" {
				lineCount = strings.Count(input.Text, "\n") + 1
			}
			return tool.Output{Text: fmt.Sprintf("Bytes: %d\nRunes: %d\nWords: %d\nLines: %d", len(input.Text), utf8.RuneCountInString(input.Text), len(strings.Fields(input.Text)), lineCount)}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "regex.test",
		NameValue:        "Regex Test",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "Find regex matches in input text.",
		OptionsValue:     []tool.Option{{Name: "pattern", Description: "Regular expression", Required: true}},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			re, err := regexp.Compile(options["pattern"])
			if err != nil {
				return tool.Output{}, err
			}
			matches := re.FindAllString(input.Text, -1)
			if len(matches) == 0 {
				return tool.Output{Text: "no matches"}, nil
			}
			return tool.Output{Text: strings.Join(matches, "\n")}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "regex.replace",
		NameValue:        "Regex Replace",
		CategoryValue:    tool.CategoryText,
		DescriptionValue: "Replace regex matches.",
		OptionsValue: []tool.Option{
			{Name: "pattern", Description: "Regular expression", Required: true},
			{Name: "replacement", Description: "Replacement text"},
		},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			re, err := regexp.Compile(options["pattern"])
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: re.ReplaceAllString(input.Text, options["replacement"])}, nil
		},
	})
}

func textTool(r *tool.Registry, id string, name string, description string, fn func(string) string) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          id,
		NameValue:        name,
		CategoryValue:    tool.CategoryText,
		DescriptionValue: description,
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: fn(input.Text)}, nil
		},
	})
}

func words(s string) []string {
	re := regexp.MustCompile(`[A-Za-z0-9]+`)
	parts := re.FindAllString(s, -1)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return parts
}

func camel(s string) string {
	parts := words(s)
	if len(parts) == 0 {
		return ""
	}
	for i := 1; i < len(parts); i++ {
		parts[i] = capitalize(parts[i])
	}
	return strings.Join(parts, "")
}

func pascal(s string) string {
	parts := words(s)
	for i := range parts {
		parts[i] = capitalize(parts[i])
	}
	return strings.Join(parts, "")
}

func capitalize(s string) string {
	rs := []rune(s)
	if len(rs) == 0 {
		return s
	}
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}
