package encode

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"strconv"

	"github.com/gkmz/sharp/pkg/tool"
)

func Register(r *tool.Registry) {
	base64Tool(r, "b64.encode", "Base64 Encode", base64.StdEncoding.EncodeToString)
	base64DecodeTool(r, "b64.decode", "Base64 Decode", base64.StdEncoding.DecodeString)
	base64Tool(r, "b64url.encode", "Base64 URL Encode", base64.URLEncoding.EncodeToString)
	base64DecodeTool(r, "b64url.decode", "Base64 URL Decode", base64.URLEncoding.DecodeString)
	base64Tool(r, "b64rawurl.encode", "Raw Base64 URL Encode", base64.RawURLEncoding.EncodeToString)
	base64DecodeTool(r, "b64rawurl.decode", "Raw Base64 URL Decode", base64.RawURLEncoding.DecodeString)

	r.MustRegister(tool.SimpleTool{
		IDValue:          "url.encode",
		NameValue:        "URL Encode",
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: "Percent-encode text for URLs.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: url.QueryEscape(input.Text)}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "url.decode",
		NameValue:        "URL Decode",
		CategoryValue:    tool.CategoryEncode,
		DescriptionValue: "Decode percent-encoded URL text.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			s, err := url.QueryUnescape(input.Text)
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
			b, err := hex.DecodeString(input.Text)
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
			b, err := fn(input.Text)
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b)}, nil
		},
	})
}
