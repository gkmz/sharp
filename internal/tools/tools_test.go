package tools_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/pkg/tool"
)

func TestJSONPrettyAndGet(t *testing.T) {
	registry := tools.NewRegistry()
	pretty, ok := registry.Get("json.pretty")
	if !ok {
		t.Fatal("json.pretty not registered")
	}
	out, err := pretty.Run(context.Background(), tool.Input{Text: `{"data":{"id":1}}`}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Text, `"id": 1`) {
		t.Fatalf("unexpected pretty output: %s", out.Text)
	}

	get, ok := registry.Get("json.get")
	if !ok {
		t.Fatal("json.get not registered")
	}
	out, err = get.Run(context.Background(), tool.Input{Text: `{"data":{"id":1}}`}, tool.Options{"path": "data.id"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "1" {
		t.Fatalf("expected 1, got %q", out.Text)
	}
}

func TestBase64(t *testing.T) {
	registry := tools.NewRegistry()
	enc, _ := registry.Get("b64.encode")
	dec, _ := registry.Get("b64.decode")
	out, err := enc.Run(context.Background(), tool.Input{Text: "hello"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "aGVsbG8=" {
		t.Fatalf("bad encode: %s", out.Text)
	}
	out, err = dec.Run(context.Background(), tool.Input{Text: out.Text}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "hello" {
		t.Fatalf("bad decode: %s", out.Text)
	}
}

func TestSHA256(t *testing.T) {
	registry := tools.NewRegistry()
	hash, _ := registry.Get("hash.sha256")
	out, err := hash.Run(context.Background(), tool.Input{Text: "hello"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if out.Text != want {
		t.Fatalf("want %s, got %s", want, out.Text)
	}
}

func TestCoreToolRegistryContainsExpectedTools(t *testing.T) {
	registry := tools.NewRegistry()
	for _, id := range []string{
		"json.pretty", "json.minify", "json.validate", "json.get", "json.sort", "json.escape", "json.unescape",
		"b64.encode", "b64.decode", "url.encode", "url.decode", "hex.encode", "hex.decode",
		"hash.md5", "hash.sha256", "hmac.sha256",
		"time.now", "time.from", "time.parse",
		"uuid.v4", "password", "token",
		"text.upper", "text.lower", "text.sort", "text.uniq", "text.count", "regex.test", "regex.replace",
		"url.parse", "query.parse", "dns.lookup", "cidr.parse",
		"json.to_yaml", "yaml.to_json", "csv.to_json", "headers.to_json",
		"jwt.decode",
	} {
		if _, ok := registry.Get(id); !ok {
			t.Fatalf("tool %q is not registered", id)
		}
	}
}

func TestJSONTools(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		input    string
		options  tool.Options
		want     string
		contains []string
	}{
		{name: "minify", id: "json.minify", input: "{\n  \"b\": 2\n}", want: `{"b":2}`},
		{name: "validate", id: "json.validate", input: `{"ok":true}`, want: "valid JSON"},
		{name: "sort", id: "json.sort", input: `{"b":2,"a":1}`, want: "{\n  \"a\": 1,\n  \"b\": 2\n}"},
		{name: "escape", id: "json.escape", input: `{"a":1}`, want: `"{\"a\":1}"`},
		{name: "unescape", id: "json.unescape", input: `"{\"a\":1}"`, want: `{"a":1}`},
		{name: "get", id: "json.get", input: `{"data":{"name":"sharp"}}`, options: tool.Options{"path": "data.name"}, want: "sharp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := runTool(t, tt.id, tt.input, tt.options)
			if tt.want != "" && out.Text != tt.want {
				t.Fatalf("%s output = %q, want %q", tt.id, out.Text, tt.want)
			}
			for _, want := range tt.contains {
				if !strings.Contains(out.Text, want) {
					t.Fatalf("%s output = %q, want to contain %q", tt.id, out.Text, want)
				}
			}
		})
	}
}

func TestEncodeTools(t *testing.T) {
	pairs := []struct {
		encodeID string
		decodeID string
		encoded  string
		input    string
	}{
		{encodeID: "url.encode", decodeID: "url.decode", input: "hello world", encoded: "hello+world"},
		{encodeID: "hex.encode", decodeID: "hex.decode", input: "hello", encoded: "68656c6c6f"},
		{encodeID: "html.escape", decodeID: "html.unescape", input: "<tag>", encoded: "&lt;tag&gt;"},
	}
	for _, pair := range pairs {
		t.Run(pair.encodeID, func(t *testing.T) {
			out := runTool(t, pair.encodeID, pair.input, nil)
			if out.Text != pair.encoded {
				t.Fatalf("%s output = %q, want %q", pair.encodeID, out.Text, pair.encoded)
			}
			out = runTool(t, pair.decodeID, out.Text, nil)
			if out.Text != pair.input {
				t.Fatalf("%s output = %q, want %q", pair.decodeID, out.Text, pair.input)
			}
		})
	}
}

func TestCryptoTools(t *testing.T) {
	cases := []struct {
		id      string
		input   string
		options tool.Options
		want    string
	}{
		{id: "hash.md5", input: "hello", want: "5d41402abc4b2a76b9719d911017c592"},
		{id: "hmac.sha256", input: "hello", options: tool.Options{"key": "secret"}, want: "88aab3ede8d3adf94d26ab90d3bafd4a2083070c3bcce9c014ee04a443847c0b"},
		{id: "hash.crc32", input: "hello", want: "3610a686"},
	}
	for _, tt := range cases {
		t.Run(tt.id, func(t *testing.T) {
			out := runTool(t, tt.id, tt.input, tt.options)
			if out.Text != tt.want {
				t.Fatalf("%s output = %q, want %q", tt.id, out.Text, tt.want)
			}
		})
	}
}

func TestTimeTools(t *testing.T) {
	out := runTool(t, "time.from", "1700000000", nil)
	for _, want := range []string{"Unix seconds: 1700000000", "Unix milliseconds: 1700000000000"} {
		if !strings.Contains(out.Text, want) {
			t.Fatalf("time.from output = %q, want %q", out.Text, want)
		}
	}

	out = runTool(t, "time.parse", "2024-01-02T03:04:05Z", nil)
	if !strings.Contains(out.Text, "Unix seconds: 1704164645") {
		t.Fatalf("time.parse output = %q, want parsed unix seconds", out.Text)
	}

	out = runTool(t, "time.now", "", nil)
	if !strings.Contains(out.Text, "Unix seconds:") || !strings.Contains(out.Text, "UTC:") {
		t.Fatalf("time.now output = %q, want formatted current time", out.Text)
	}
}

func TestGeneratorTools(t *testing.T) {
	out := runTool(t, "uuid.v4", "", nil)
	if len(out.Text) != 36 || strings.Count(out.Text, "-") != 4 {
		t.Fatalf("uuid.v4 output = %q, want UUID shape", out.Text)
	}

	out = runTool(t, "password", "", tool.Options{"length": "12"})
	if len(out.Text) != 12 {
		t.Fatalf("password length = %d, want 12", len(out.Text))
	}

	out = runTool(t, "token", "", tool.Options{"bytes": "16"})
	if _, err := base64.RawURLEncoding.DecodeString(out.Text); err != nil {
		t.Fatalf("token output = %q, want raw base64url: %v", out.Text, err)
	}
}

func TestTextTools(t *testing.T) {
	cases := []struct {
		id      string
		input   string
		options tool.Options
		want    string
	}{
		{id: "text.upper", input: "hello", want: "HELLO"},
		{id: "text.lower", input: "HELLO", want: "hello"},
		{id: "text.trim", input: "  hello\n", want: "hello"},
		{id: "text.snake", input: "Hello world", want: "hello_world"},
		{id: "text.kebab", input: "Hello world", want: "hello-world"},
		{id: "text.camel", input: "hello world", want: "helloWorld"},
		{id: "text.pascal", input: "hello world", want: "HelloWorld"},
		{id: "text.sort", input: "b\na", want: "a\nb"},
		{id: "text.uniq", input: "a\nb\na", want: "a\nb"},
		{id: "regex.test", input: "abc 123", options: tool.Options{"pattern": `\d+`}, want: "123"},
		{id: "regex.replace", input: "abc 123", options: tool.Options{"pattern": `\d+`, "replacement": "NUM"}, want: "abc NUM"},
	}
	for _, tt := range cases {
		t.Run(tt.id, func(t *testing.T) {
			out := runTool(t, tt.id, tt.input, tt.options)
			if out.Text != tt.want {
				t.Fatalf("%s output = %q, want %q", tt.id, out.Text, tt.want)
			}
		})
	}

	out := runTool(t, "text.count", "hello 世界\nagain", nil)
	for _, want := range []string{"Bytes:", "Runes:", "Words: 3", "Lines: 2"} {
		if !strings.Contains(out.Text, want) {
			t.Fatalf("text.count output = %q, want %q", out.Text, want)
		}
	}
}

func TestNetworkTools(t *testing.T) {
	out := runTool(t, "url.parse", "https://example.com/path?q=1#frag", nil)
	assertJSONContains(t, out.Text, map[string]string{
		"scheme":   "https",
		"host":     "example.com",
		"path":     "/path",
		"rawQuery": "q=1",
		"fragment": "frag",
	})

	out = runTool(t, "query.parse", "?a=1&b=2", nil)
	if !strings.Contains(out.Text, `"a":`) || !strings.Contains(out.Text, `"1"`) {
		t.Fatalf("query.parse output = %q, want parsed query JSON", out.Text)
	}

	out = runTool(t, "cidr.parse", "192.168.1.1/24", nil)
	for _, want := range []string{"IP: 192.168.1.1", "Network: 192.168.1.0/24", "Prefix: 24"} {
		if !strings.Contains(out.Text, want) {
			t.Fatalf("cidr.parse output = %q, want %q", out.Text, want)
		}
	}
}

func TestConvertTools(t *testing.T) {
	out := runTool(t, "json.to_yaml", `{"name":"sharp"}`, nil)
	if !strings.Contains(out.Text, "name: sharp") || out.MimeType != "application/yaml" {
		t.Fatalf("json.to_yaml output = %q mime=%q, want YAML", out.Text, out.MimeType)
	}

	out = runTool(t, "yaml.to_json", "name: sharp\n", nil)
	if !strings.Contains(out.Text, `"name": "sharp"`) || out.MimeType != "application/json" {
		t.Fatalf("yaml.to_json output = %q mime=%q, want JSON", out.Text, out.MimeType)
	}

	out = runTool(t, "csv.to_json", "name,age\nsharp,1\n", nil)
	if !strings.Contains(out.Text, `"name": "sharp"`) || !strings.Contains(out.Text, `"age": "1"`) {
		t.Fatalf("csv.to_json output = %q, want JSON array", out.Text)
	}

	out = runTool(t, "headers.to_json", "Content-Type: application/json\nX-Test: yes", nil)
	if !strings.Contains(out.Text, `"Content-Type": "application/json"`) || !strings.Contains(out.Text, `"X-Test": "yes"`) {
		t.Fatalf("headers.to_json output = %q, want header JSON", out.Text)
	}
}

func TestJWTDecode(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoic2hhcnAiLCJpYXQiOjE3MDAwMDAwMDB9.signature"
	out := runTool(t, "jwt.decode", token, nil)
	if !strings.Contains(out.Text, `"alg": "HS256"`) || !strings.Contains(out.Text, `"name": "sharp"`) {
		t.Fatalf("jwt.decode output = %q, want decoded header and payload", out.Text)
	}
}

func TestInvalidInputsReturnErrors(t *testing.T) {
	cases := []struct {
		id      string
		input   string
		options tool.Options
	}{
		{id: "json.validate", input: `{"bad"`},
		{id: "hex.decode", input: "not-hex"},
		{id: "time.from", input: "not-time"},
		{id: "regex.test", input: "abc", options: tool.Options{"pattern": "["}},
		{id: "headers.to_json", input: "missing colon"},
	}
	registry := tools.NewRegistry()
	for _, tt := range cases {
		t.Run(tt.id, func(t *testing.T) {
			current, ok := registry.Get(tt.id)
			if !ok {
				t.Fatalf("tool %q not registered", tt.id)
			}
			if _, err := current.Run(context.Background(), tool.Input{Text: tt.input}, tt.options); err == nil {
				t.Fatalf("%s returned nil error for invalid input", tt.id)
			}
		})
	}
}

func runTool(t *testing.T, id string, input string, options tool.Options) tool.Output {
	t.Helper()
	registry := tools.NewRegistry()
	current, ok := registry.Get(id)
	if !ok {
		t.Fatalf("tool %q not registered", id)
	}
	out, err := current.Run(context.Background(), tool.Input{Text: input}, options)
	if err != nil {
		t.Fatalf("%s returned error: %v", id, err)
	}
	return out
}

func assertJSONContains(t *testing.T, text string, want map[string]string) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", text, err)
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("JSON field %s = %#v, want %q in %s", key, got[key], value, text)
		}
	}
}

func TestTimeParseExpectedUnixSecondsFixture(t *testing.T) {
	parsed, err := time.Parse(time.RFC3339, "2024-01-02T03:04:05Z")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Unix() != 1704164645 {
		t.Fatalf("test fixture unix seconds = %d, want 1704164645", parsed.Unix())
	}
}
