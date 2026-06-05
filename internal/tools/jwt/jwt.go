package jwt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gkmz/sharp/pkg/tool"
)

type decodedToken struct {
	Header            json.RawMessage `json:"header"`
	Payload           json.RawMessage `json:"payload"`
	SignatureVerified bool            `json:"signatureVerified"`
}

// Register adds JWT inspection tools to r.
func Register(r *tool.Registry) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          "jwt.decode",
		NameValue:        "JWT Decode",
		CategoryValue:    tool.CategoryInspector,
		DescriptionValue: "Decode JWT header and payload without verifying signature.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			parts := strings.Split(strings.TrimSpace(input.Text), ".")
			if len(parts) != 3 {
				return tool.Output{}, fmt.Errorf("JWT must contain exactly 3 dot-separated parts")
			}
			header, err := decodePart(parts[0])
			if err != nil {
				return tool.Output{}, fmt.Errorf("invalid header: %w", err)
			}
			payload, err := decodePart(parts[1])
			if err != nil {
				return tool.Output{}, fmt.Errorf("invalid payload: %w", err)
			}
			// JWT 检查结果必须保持为合法 JSON，才能继续 pipe 到 JSON pretty/query 等工具。
			b, err := json.MarshalIndent(decodedToken{Header: header, Payload: payload, SignatureVerified: false}, "", "  ")
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b), MimeType: "application/json"}, nil
		},
	})
}

func decodePart(s string) (json.RawMessage, error) {
	if strings.TrimSpace(s) == "" {
		return nil, fmt.Errorf("part is empty")
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if !json.Valid(b) {
		return nil, fmt.Errorf("part must be JSON")
	}
	return json.RawMessage(b), nil
}
