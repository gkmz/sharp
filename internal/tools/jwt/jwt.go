package jwt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gkmz/sharp/pkg/tool"
)

func Register(r *tool.Registry) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          "jwt.decode",
		NameValue:        "JWT Decode",
		CategoryValue:    tool.CategoryInspector,
		DescriptionValue: "Decode JWT header and payload without verifying signature.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			parts := strings.Split(strings.TrimSpace(input.Text), ".")
			if len(parts) < 2 {
				return tool.Output{}, fmt.Errorf("JWT must contain at least header and payload")
			}
			header, err := decodePart(parts[0])
			if err != nil {
				return tool.Output{}, fmt.Errorf("invalid header: %w", err)
			}
			payload, err := decodePart(parts[1])
			if err != nil {
				return tool.Output{}, fmt.Errorf("invalid payload: %w", err)
			}
			return tool.Output{Text: fmt.Sprintf("Header:\n%s\n\nPayload:\n%s\n\nSignature verified: false", header, payload), MimeType: "application/json"}, nil
		},
	})
}

func decodePart(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, b, "", "  "); err != nil {
		return string(b), nil
	}
	return buf.String(), nil
}
