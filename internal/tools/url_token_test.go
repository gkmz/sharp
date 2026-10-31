package tools_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/pkg/tool"
)

func TestURLEncodingModes(t *testing.T) {
	for _, tc := range []struct{ name, mode, input, want string }{
		{"query default", "", "a b+c", "a+b%2Bc"},
		{"query explicit", "query", "a b+c", "a+b%2Bc"},
		{"component password", "component", "^@#$", "%5E%40%23%24"},
		{"component space and plus", "component", "a b+c", "a%20b%2Bc"},
		{"component unreserved", "component", "AZaz09-._~", "AZaz09-._~"},
		{"component reserved", "component", ":/?#[]@!$&'()*+,;=%", "%3A%2F%3F%23%5B%5D%40%21%24%26%27%28%29%2A%2B%2C%3B%3D%25"},
		{"component UTF8", "component", "密码", "%E5%AF%86%E7%A0%81"},
		{"component empty", "component", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := tool.Options{"mode": tc.mode}
			out := runTool(t, "url.encode", tc.input, options)
			if out.Text != tc.want {
				t.Fatalf("encoded = %q, want %q", out.Text, tc.want)
			}
			decoded := runTool(t, "url.decode", out.Text, options)
			if decoded.Text != tc.input {
				t.Fatalf("round trip = %q, want %q", decoded.Text, tc.input)
			}
		})
	}
	for _, tc := range []struct{ mode, want string }{{"", "a b+c"}, {"query", "a b+c"}, {"component", "a+b+c"}} {
		out := runTool(t, "url.decode", "a+b%2Bc", tool.Options{"mode": tc.mode})
		if out.Text != tc.want {
			t.Errorf("decode mode %q = %q, want %q", tc.mode, out.Text, tc.want)
		}
	}
}

func TestURLComponentSupportsDatabasePasswordAndEveryByte(t *testing.T) {
	password := "p^@#$ :/+?=密码"
	encoded := runTool(t, "url.encode", password, tool.Options{"mode": "component"}).Text
	u, err := url.Parse("postgres://user:" + encoded + "@localhost:5432/app")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := u.User.Password(); !ok || got != password {
		t.Fatalf("password changed: %q, present=%v", got, ok)
	}
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}
	encoded = runTool(t, "url.encode", string(data), tool.Options{"mode": "component"}).Text
	decoded := runTool(t, "url.decode", encoded, tool.Options{"mode": "component"}).Text
	if decoded != string(data) {
		t.Fatal("component codec changed byte data")
	}
}

func TestURLModesRejectInvalidOptionsAndEscapes(t *testing.T) {
	registry := tools.NewRegistry()
	for _, id := range []string{"url.encode", "url.decode"} {
		current, _ := registry.Get(id)
		if _, err := current.Run(context.Background(), tool.Input{Text: "private input"}, tool.Options{"mode": "invalid"}); err == nil || !strings.Contains(err.Error(), "mode") || strings.Contains(err.Error(), "private input") {
			t.Errorf("%s invalid mode error = %v", id, err)
		}
	}
	decode, _ := registry.Get("url.decode")
	for _, mode := range []string{"query", "component"} {
		for _, value := range []string{"%", "%0", "%GG"} {
			if _, err := decode.Run(context.Background(), tool.Input{Text: value}, tool.Options{"mode": mode}); err == nil {
				t.Errorf("mode %s accepted malformed %q", mode, value)
			}
		}
	}
}

func TestTokenFormatsRepresentRequestedRandomBytes(t *testing.T) {
	for _, n := range []int{1, 2, 3, 32, 64, 4096} {
		for _, format := range []string{"hex", "base64", "base64url", ""} {
			t.Run(fmt.Sprintf("%s/%d", format, n), func(t *testing.T) {
				out := runTool(t, "token", "", tool.Options{"bytes": fmt.Sprint(n), "format": format}).Text
				var data []byte
				var err error
				length := 0
				switch format {
				case "hex":
					data, err = hex.DecodeString(out)
					length = 2 * n
					if out != strings.ToLower(out) {
						t.Error("hex is not lowercase")
					}
				case "base64":
					data, err = base64.StdEncoding.Strict().DecodeString(out)
					length = base64.StdEncoding.EncodedLen(n)
				default:
					data, err = base64.RawURLEncoding.Strict().DecodeString(out)
					length = base64.RawURLEncoding.EncodedLen(n)
				}
				if err != nil || len(data) != n || len(out) != length || strings.ContainsAny(out, "\r\n") {
					t.Fatalf("%s: decoded bytes=%d, chars=%d (want %d), error=%v", format, len(data), len(out), length, err)
				}
			})
		}
	}
	defaultToken := runTool(t, "token", "", nil).Text
	if data, err := base64.RawURLEncoding.DecodeString(defaultToken); err != nil || len(data) != 32 {
		t.Fatal("default token changed from 32-byte raw Base64URL")
	}
}

func TestTokenRejectsInvalidFormatsAndSizes(t *testing.T) {
	token, _ := tools.NewRegistry().Get("token")
	for _, options := range []tool.Options{{"format": "rot13"}, {"bytes": "0"}, {"bytes": "-1"}, {"bytes": "4097"}, {"bytes": "1.5"}, {"bytes": "abc"}} {
		if _, err := token.Run(context.Background(), tool.Input{}, options); err == nil {
			t.Errorf("accepted invalid options %v", options)
		}
	}
}
