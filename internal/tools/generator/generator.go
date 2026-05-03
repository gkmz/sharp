package generator

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/hank/sharp/pkg/tool"
)

const passwordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}:,.?"

func Register(r *tool.Registry) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          "uuid.v4",
		NameValue:        "UUID v4",
		CategoryValue:    tool.CategoryGenerator,
		DescriptionValue: "Generate a UUID v4.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			return tool.Output{Text: uuid.NewString()}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "password",
		NameValue:        "Random Password",
		CategoryValue:    tool.CategoryGenerator,
		DescriptionValue: "Generate a cryptographically secure password.",
		OptionsValue: []tool.Option{{
			Name:        "length",
			Description: "Password length",
			Default:     "24",
		}},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			length := intOption(options, "length", 24)
			if length < 1 || length > 4096 {
				return tool.Output{}, fmt.Errorf("length must be between 1 and 4096")
			}
			s, err := randomString(length, passwordChars)
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: s}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "token",
		NameValue:        "Random Token",
		CategoryValue:    tool.CategoryGenerator,
		DescriptionValue: "Generate a cryptographically secure base64url token.",
		OptionsValue: []tool.Option{{
			Name:        "bytes",
			Description: "Number of random bytes",
			Default:     "32",
		}},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			n := intOption(options, "bytes", 32)
			if n < 1 || n > 4096 {
				return tool.Output{}, fmt.Errorf("bytes must be between 1 and 4096")
			}
			b := make([]byte, n)
			if _, err := rand.Read(b); err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: base64.RawURLEncoding.EncodeToString(b)}, nil
		},
	})
}

func intOption(options tool.Options, key string, fallback int) int {
	var n int
	if _, err := fmt.Sscanf(options[key], "%d", &n); err != nil || n == 0 {
		return fallback
	}
	return n
}

func randomString(length int, chars string) (string, error) {
	out := make([]byte, length)
	max := big.NewInt(int64(len(chars)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = chars[n.Int64()]
	}
	return string(out), nil
}
