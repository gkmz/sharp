package generator

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/gkmz/sharp/pkg/tool"
	"github.com/google/uuid"
)

const passwordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}:,.?"

// Register adds random value generator tools to r.
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
			length, err := intOption(options, "length", 24)
			if err != nil {
				return tool.Output{}, err
			}
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
		DescriptionValue: "Generate a secure random token as hex, base64, or base64url.",
		OptionsValue: []tool.Option{
			{Name: "bytes", Description: "Number of random bytes before encoding (1-4096), not output characters", Default: "32"},
			{Name: "format", Description: "hex: lowercase; base64: standard with padding; base64url: URL-safe without padding", Default: "base64url", Choices: []string{"hex", "base64", "base64url"}},
		},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			n, err := intOption(options, "bytes", 32)
			if err != nil {
				return tool.Output{}, err
			}
			if n < 1 || n > 4096 {
				return tool.Output{}, fmt.Errorf("bytes must be between 1 and 4096")
			}
			encodeToken := base64.RawURLEncoding.EncodeToString
			switch options["format"] {
			case "", "base64url":
			case "hex":
				encodeToken = hex.EncodeToString
			case "base64":
				encodeToken = base64.StdEncoding.EncodeToString
			default:
				return tool.Output{}, fmt.Errorf("format must be hex, base64, or base64url")
			}
			b := make([]byte, n)
			if _, err := rand.Read(b); err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: encodeToken(b)}, nil
		},
	})
}

func intOption(options tool.Options, key string, fallback int) (int, error) {
	value := strings.TrimSpace(options[key])
	if value == "" {
		return fallback, nil
	}
	// 生成类工具会进入脚本流水线；非法参数必须失败，不能静默回退默认值。
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return n, nil
}

func randomString(length int, chars string) (string, error) {
	out := make([]byte, length)
	max := big.NewInt(int64(len(chars)))
	for i := range out {
		// 密码/token 相关随机性只使用 crypto/rand，避免误用 math/rand 带来的可预测输出。
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = chars[n.Int64()]
	}
	return string(out), nil
}
