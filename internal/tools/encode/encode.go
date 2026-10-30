package encode

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/gkmz/sharp/pkg/tool"
)

// Register adds encoding and decoding tools to r.
func Register(r *tool.Registry) {
	base64Tool(r, "b64.encode", "Base64 Encode", base64.StdEncoding.EncodeToString)
	base64DecodeTool(r, "b64.decode", "Base64 Decode", base64.StdEncoding.DecodeString)
	base64Tool(r, "b64url.encode", "Base64 URL Encode", base64.URLEncoding.EncodeToString)
	base64DecodeTool(r, "b64url.decode", "Base64 URL Decode", base64.URLEncoding.DecodeString)
	base64Tool(r, "b64rawurl.encode", "Raw Base64 URL Encode", base64.RawURLEncoding.EncodeToString)
	base64DecodeTool(r, "b64rawurl.decode", "Raw Base64 URL Decode", base64.RawURLEncoding.DecodeString)

	urlModeOption := tool.Option{
		Name: "mode", Default: "query", Choices: []string{"query", "component"},
		Description: "query: form/query values (space as +); component: URI values such as database passwords (space as %20, literal + preserved on decode)",
	}
	r.MustRegister(tool.SimpleTool{
		IDValue:          "url.encode",
		NameValue:        "URL Encode",
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: "Percent-encode a URL value in query or component mode.",
		OptionsValue:     []tool.Option{urlModeOption},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			mode, err := urlMode(options)
			if err != nil {
				return tool.Output{}, err
			}
			encoded := url.QueryEscape(input.Text)
			if mode == "component" {
				// QueryEscape already escapes everything except RFC 3986
				// unreserved bytes and spaces. Literal plus is escaped as %2B.
				encoded = strings.ReplaceAll(encoded, "+", "%20")
			}
			return tool.Output{Text: encoded}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "url.decode",
		NameValue:        "URL Decode",
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: "Decode a URL value in query or component mode.",
		OptionsValue:     []tool.Option{urlModeOption},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			mode, err := urlMode(options)
			if err != nil {
				return tool.Output{}, err
			}
			decode := url.QueryUnescape
			if mode == "component" {
				decode = url.PathUnescape // Percent-decode without changing literal +.
			}
			s, err := decode(input.Text)
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: s}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "hex.encode",
		NameValue:        "Hex Encode",
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: "Encode text as hex.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: hex.EncodeToString([]byte(input.Text))}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "hex.decode",
		NameValue:        "Hex Decode",
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: "Decode hex text.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			b, err := hex.DecodeString(removeEncodingWhitespace(input.Text))
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b)}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "html.escape",
		NameValue:        "HTML Escape",
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: "Escape HTML entities.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: html.EscapeString(input.Text)}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "html.unescape",
		NameValue:        "HTML Unescape",
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: "Unescape HTML entities.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: html.UnescapeString(input.Text)}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "unicode.escape",
		NameValue:        "Unicode Escape",
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: "Escape text with Go-style quoted Unicode sequences.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: strconv.QuoteToASCII(input.Text)}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "unicode.unescape",
		NameValue:        "Unicode Unescape",
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: "Unescape Go-style quoted Unicode sequences.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			s, err := strconv.Unquote(input.Text)
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: s}, nil
		},
	})
}

func urlMode(options tool.Options) (string, error) {
	switch mode := options["mode"]; mode {
	case "", "query":
		return "query", nil
	case "component":
		return mode, nil
	default:
		return "", fmt.Errorf("mode must be query or component")
	}
}

func base64Tool(r *tool.Registry, id string, name string, fn func([]byte) string) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          id,
		NameValue:        name,
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: fmt.Sprintf("%s text.", name),
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: fn([]byte(input.Text))}, nil
		},
	})
}

func base64DecodeTool(r *tool.Registry, id string, name string, fn func(string) ([]byte, error)) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          id,
		NameValue:        name,
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: fmt.Sprintf("%s text.", name),
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			b, err := fn(removeEncodingWhitespace(input.Text))
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b)}, nil
		},
	})
}

func removeEncodingWhitespace(s string) string {
	// 编码文本经常来自复制粘贴或换行展示，解码前去掉空白能避免无意义的格式错误。
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}
