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
		{name: "get null", id: "json.get", input: `{"data":{"name":null}}`, options: tool.Options{"path": "data.name"}, want: "null"},
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

func TestLossyJSONTransformsRejectDuplicateObjectKeys(t *testing.T) {
	cases := []struct {
		id    string
		input string
	}{
		{id: "json.sort", input: `{"a":1,"a":2}`},
		{id: "json.sort", input: `{"nested":{"a":1,"a":2}}`},
		{id: "json.to_yaml", input: `{"a":1,"a":2}`},
	}
	registry := tools.NewRegistry()
	for _, tt := range cases {
		t.Run(tt.id, func(t *testing.T) {
			current, ok := registry.Get(tt.id)
			if !ok {
				t.Fatalf("tool %q not registered", tt.id)
			}
			_, err := current.Run(context.Background(), tool.Input{Text: tt.input}, nil)
			if err == nil || !strings.Contains(err.Error(), `duplicate JSON object key "a"`) {
				t.Fatalf("%s error = %v, want duplicate key error", tt.id, err)
			}
		})
	}
}

func TestJSONGetRejectsDuplicateObjectKeys(t *testing.T) {
	registry := tools.NewRegistry()
	current, ok := registry.Get("json.get")
	if !ok {
		t.Fatal("json.get not registered")
	}
	_, err := current.Run(context.Background(), tool.Input{Text: `{"a":1,"a":2}`}, tool.Options{"path": "a"})
	if err == nil || !strings.Contains(err.Error(), `duplicate JSON object key "a"`) {
		t.Fatalf("json.get error = %v, want duplicate key error", err)
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

func TestDecodeToolsIgnoreFormattingWhitespace(t *testing.T) {
	cases := []struct {
		id    string
		input string
		want  string
	}{
		{id: "b64.decode", input: "aGVs bG8=", want: "hello"},
		{id: "b64.decode", input: "aGVs\nbG8=", want: "hello"},
		{id: "hex.decode", input: "68 65 6c 6c 6f", want: "hello"},
		{id: "hex.decode", input: "68\t65\n6c 6c 6f", want: "hello"},
	}
	for _, tt := range cases {
		t.Run(tt.id+" whitespace", func(t *testing.T) {
			out := runTool(t, tt.id, tt.input, nil)
			if out.Text != tt.want {
				t.Fatalf("%s output = %q, want %q", tt.id, out.Text, tt.want)
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

func TestHMACRejectsEmptyKey(t *testing.T) {
	registry := tools.NewRegistry()
	current, ok := registry.Get("hmac.sha256")
	if !ok {
		t.Fatal("hmac.sha256 not registered")
	}
	for _, options := range []tool.Options{nil, {"key": ""}, {"key": "   "}} {
		_, err := current.Run(context.Background(), tool.Input{Text: "hello"}, options)
		if err == nil || !strings.Contains(err.Error(), "missing required option: key") {
			t.Fatalf("hmac.sha256 error = %v, want missing key error", err)
		}
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

func TestTimeFromDetectsUnitsByDigitCount(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "positive seconds", input: "999999999999", want: []string{"Unix seconds: 999999999999", "Unix milliseconds: 999999999999000", "Unix nanoseconds: 999999999999000000000"}},
		{name: "negative seconds", input: "-999999999999", want: []string{"Unix seconds: -999999999999", "Unix milliseconds: -999999999999000", "Unix nanoseconds: -999999999999000000000"}},
		{name: "zero-padded positive seconds", input: "0000000000001", want: []string{"Unix seconds: 1", "Unix milliseconds: 1000"}},
		{name: "zero-padded negative seconds", input: "-0000000000001", want: []string{"Unix seconds: -1", "Unix milliseconds: -1000"}},
		{name: "zero timestamp", input: "0000000000000", want: []string{"Unix seconds: 0", "Unix milliseconds: 0"}},
		{name: "positive milliseconds", input: "1000000000000", want: []string{"Unix seconds: 1000000000", "Unix milliseconds: 1000000000000"}},
		{name: "negative milliseconds", input: "-1000000000000", want: []string{"Unix seconds: -1000000000", "Unix milliseconds: -1000000000000"}},
		{name: "positive nanoseconds", input: "1000000000000000000", want: []string{"Unix seconds: 1000000000", "Unix nanoseconds: 1000000000000000000"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			out := runTool(t, "time.from", tt.input, nil)
			for _, want := range tt.want {
				if !strings.Contains(out.Text, want) {
					t.Fatalf("time.from output = %q, want %q", out.Text, want)
				}
			}
		})
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

func TestGeneratorInvalidOptionsReturnErrors(t *testing.T) {
	cases := []struct {
		id      string
		options tool.Options
	}{
		{id: "password", options: tool.Options{"length": "abc"}},
		{id: "password", options: tool.Options{"length": "0"}},
		{id: "token", options: tool.Options{"bytes": "abc"}},
		{id: "token", options: tool.Options{"bytes": "-1"}},
	}
	registry := tools.NewRegistry()
	for _, tt := range cases {
		t.Run(tt.id+" invalid option", func(t *testing.T) {
			current, ok := registry.Get(tt.id)
			if !ok {
				t.Fatalf("tool %q not registered", tt.id)
			}
			if _, err := current.Run(context.Background(), tool.Input{}, tt.options); err == nil {
				t.Fatalf("%s returned nil error for options %#v", tt.id, tt.options)
			}
		})
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
		{id: "text.snake", input: "用户 Name Café", want: "用户_name_café"},
		{id: "text.kebab", input: "用户 Name Café", want: "用户-name-café"},
		{id: "text.camel", input: "用户 name café", want: "用户NameCafé"},
		{id: "text.sort", input: "b\na", want: "a\nb"},
		{id: "text.sort", input: "b\na\n", want: "a\nb"},
		{id: "text.sort", input: "b\r\na\r\n", want: "a\nb"},
		{id: "text.sort", input: "b\n\na\n", want: "\na\nb"},
		{id: "text.uniq", input: "a\nb\na", want: "a\nb"},
		{id: "text.uniq", input: "a\nb\na\n", want: "a\nb"},
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

	out = runTool(t, "url.parse", "example.com/path?q=1", nil)
	assertJSONContains(t, out.Text, map[string]string{
		"host":     "example.com",
		"path":     "/path",
		"rawQuery": "q=1",
	})

	out = runTool(t, "url.parse", "example.com:8080/path", nil)
	assertJSONContains(t, out.Text, map[string]string{
		"host": "example.com:8080",
		"path": "/path",
	})

	out = runTool(t, "url.parse", "localhost:8080/path", nil)
	assertJSONContains(t, out.Text, map[string]string{
		"host": "localhost:8080",
		"path": "/path",
	})

	out = runTool(t, "url.parse", "[::1]:8080/path", nil)
	assertJSONContains(t, out.Text, map[string]string{
		"host": "[::1]:8080",
		"path": "/path",
	})

	out = runTool(t, "url.parse", "mailto:user@example.com", nil)
	assertJSONContains(t, out.Text, map[string]string{
		"scheme": "mailto",
		"opaque": "user@example.com",
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

func TestNetworkEmptyInputsReturnActionableErrors(t *testing.T) {
	cases := []struct {
		id    string
		input string
		want  string
	}{
		{id: "url.parse", input: "", want: "URL is required"},
		{id: "url.parse", input: "just-a-path", want: "URL must include a scheme or host"},
		{id: "url.parse", input: "http:///path", want: `URL scheme "http" requires a host`},
		{id: "url.parse", input: "https:?q=1", want: `URL scheme "https" requires a host`},
		{id: "query.parse", input: "?", want: "query string is required"},
		{id: "dns.lookup", input: "", want: "host is required"},
		{id: "cidr.parse", input: "", want: "CIDR is required"},
	}
	registry := tools.NewRegistry()
	for _, tt := range cases {
		t.Run(tt.id+" empty input", func(t *testing.T) {
			current, ok := registry.Get(tt.id)
			if !ok {
				t.Fatalf("tool %q not registered", tt.id)
			}
			_, err := current.Run(context.Background(), tool.Input{Text: tt.input}, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("%s error = %v, want %q", tt.id, err, tt.want)
			}
		})
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

	out = runTool(t, "headers.to_json", "Content-Type: application/json\nX-Test: yes\nSet-Cookie: a\nSet-Cookie: b", nil)
	if !strings.Contains(out.Text, `"Content-Type": "application/json"`) || !strings.Contains(out.Text, `"X-Test": "yes"`) {
		t.Fatalf("headers.to_json output = %q, want header JSON", out.Text)
	}
	if !strings.Contains(out.Text, `"Set-Cookie": [`) || !strings.Contains(out.Text, `"a"`) || !strings.Contains(out.Text, `"b"`) {
		t.Fatalf("headers.to_json output = %q, want duplicate headers preserved as array", out.Text)
	}

	out = runTool(t, "headers.to_json", "content-type: application/json\nContent-Type: text/plain\nX-Test: a\nx-test: b", nil)
	if !strings.Contains(out.Text, `"Content-Type": [`) || strings.Contains(out.Text, `"content-type"`) {
		t.Fatalf("headers.to_json output = %q, want canonical content-type array", out.Text)
	}
	if !strings.Contains(out.Text, `"X-Test": [`) || strings.Contains(out.Text, `"x-test"`) {
		t.Fatalf("headers.to_json output = %q, want canonical x-test array", out.Text)
	}
}

func TestYAMLToJSONNormalizesScalarKeys(t *testing.T) {
	out := runTool(t, "yaml.to_json", "1: one\ntrue: ok\nnested:\n  2: two\nitems:\n  - 3: three\n", nil)
	var got map[string]any
	if err := json.Unmarshal([]byte(out.Text), &got); err != nil {
		t.Fatalf("yaml.to_json output = %q, want valid JSON: %v", out.Text, err)
	}
	if got["1"] != "one" || got["true"] != "ok" {
		t.Fatalf("yaml.to_json output = %q, want scalar map keys normalized as JSON keys", out.Text)
	}
	nested, ok := got["nested"].(map[string]any)
	if !ok || nested["2"] != "two" {
		t.Fatalf("yaml.to_json output = %q, want nested scalar map keys normalized", out.Text)
	}
	items, ok := got["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("yaml.to_json output = %q, want list item preserved", out.Text)
	}
	item, ok := items[0].(map[string]any)
	if !ok || item["3"] != "three" {
		t.Fatalf("yaml.to_json output = %q, want scalar keys normalized inside lists", out.Text)
	}
}

func TestYAMLToJSONRejectsNormalizedDuplicateKeys(t *testing.T) {
	registry := tools.NewRegistry()
	current, ok := registry.Get("yaml.to_json")
	if !ok {
		t.Fatal("yaml.to_json not registered")
	}
	_, err := current.Run(context.Background(), tool.Input{Text: "1: one\n\"1\": uno\n"}, nil)
	if err == nil || !strings.Contains(err.Error(), `YAML map key "1" is duplicated after JSON normalization`) {
		t.Fatalf("yaml.to_json error = %v, want normalized duplicate key error", err)
	}
}

func TestYAMLToJSONRejectsAliasCycle(t *testing.T) {
	registry := tools.NewRegistry()
	current, ok := registry.Get("yaml.to_json")
	if !ok {
		t.Fatal("yaml.to_json not registered")
	}
	_, err := current.Run(context.Background(), tool.Input{Text: "a: &a [*a]\n"}, nil)
	if err == nil || !strings.Contains(err.Error(), "YAML alias cycle is not supported") {
		t.Fatalf("yaml.to_json error = %v, want alias cycle error", err)
	}
}

func TestYAMLToJSONRejectsMultipleDocuments(t *testing.T) {
	registry := tools.NewRegistry()
	current, ok := registry.Get("yaml.to_json")
	if !ok {
		t.Fatal("yaml.to_json not registered")
	}
	_, err := current.Run(context.Background(), tool.Input{Text: "a: 1\n---\nb: 2\n"}, nil)
	if err == nil || !strings.Contains(err.Error(), "YAML stream contains multiple documents") {
		t.Fatalf("yaml.to_json error = %v, want multiple document error", err)
	}
}

func TestCSVToJSONRejectsUnsafeHeaders(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: "a,a\n1,2\n", want: `CSV header "a" is duplicated`},
		{input: "a,\n1,2\n", want: "CSV header 2 is empty"},
	}
	registry := tools.NewRegistry()
	current, ok := registry.Get("csv.to_json")
	if !ok {
		t.Fatal("csv.to_json not registered")
	}
	for _, tt := range cases {
		t.Run(tt.want, func(t *testing.T) {
			_, err := current.Run(context.Background(), tool.Input{Text: tt.input}, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("csv.to_json error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestCSVToJSONTrimsHeaderNames(t *testing.T) {
	out := runTool(t, "csv.to_json", " name , age \nsharp,1\n", nil)
	var got []map[string]string
	if err := json.Unmarshal([]byte(out.Text), &got); err != nil {
		t.Fatalf("csv.to_json output = %q, want valid JSON: %v", out.Text, err)
	}
	if len(got) != 1 || got[0]["name"] != "sharp" || got[0]["age"] != "1" {
		t.Fatalf("csv.to_json output = %q, want trimmed header names", out.Text)
	}
	if _, ok := got[0][" name "]; ok {
		t.Fatalf("csv.to_json output = %q, should not keep untrimmed header name", out.Text)
	}
}

func TestJWTDecode(t *testing.T) {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoic2hhcnAiLCJpYXQiOjE3MDAwMDAwMDB9.signature"
	out := runTool(t, "jwt.decode", token, nil)
	var got struct {
		Header            map[string]any `json:"header"`
		Payload           map[string]any `json:"payload"`
		SignatureVerified bool           `json:"signatureVerified"`
	}
	if err := json.Unmarshal([]byte(out.Text), &got); err != nil {
		t.Fatalf("jwt.decode output = %q, want valid JSON: %v", out.Text, err)
	}
	if got.Header["alg"] != "HS256" || got.Payload["name"] != "sharp" || got.SignatureVerified {
		t.Fatalf("jwt.decode output = %q, want decoded header and payload", out.Text)
	}
	if out.MimeType != "application/json" {
		t.Fatalf("jwt.decode mime = %q, want application/json", out.MimeType)
	}
}

func TestJWTDecodeRejectsMalformedTokenShape(t *testing.T) {
	registry := tools.NewRegistry()
	current, ok := registry.Get("jwt.decode")
	if !ok {
		t.Fatal("jwt.decode not registered")
	}
	for _, input := range []string{"header.payload", "header.payload.signature.extra", ".payload.signature"} {
		t.Run(input, func(t *testing.T) {
			if _, err := current.Run(context.Background(), tool.Input{Text: input}, nil); err == nil {
				t.Fatalf("jwt.decode returned nil error for malformed token %q", input)
			}
		})
	}
}

func TestJWTDecodeRejectsNonJSONParts(t *testing.T) {
	registry := tools.NewRegistry()
	current, ok := registry.Get("jwt.decode")
	if !ok {
		t.Fatal("jwt.decode not registered")
	}
	if _, err := current.Run(context.Background(), tool.Input{Text: "bm90anNvbg.bm90anNvbg.sig"}, nil); err == nil {
		t.Fatal("jwt.decode returned nil error for non-JSON header and payload")
	}
}

func TestJWTDecodeRejectsNonObjectParts(t *testing.T) {
	registry := tools.NewRegistry()
	current, ok := registry.Get("jwt.decode")
	if !ok {
		t.Fatal("jwt.decode not registered")
	}
	encode := base64.RawURLEncoding.EncodeToString
	cases := []struct {
		name  string
		token string
		want  string
	}{
		{name: "header string", token: encode([]byte(`"str"`)) + "." + encode([]byte(`{"a":1}`)) + ".sig", want: "invalid header: part must be a JSON object"},
		{name: "payload array", token: encode([]byte(`{"alg":"none"}`)) + "." + encode([]byte(`[]`)) + ".sig", want: "invalid payload: part must be a JSON object"},
		{name: "duplicate header key", token: encode([]byte(`{"a":1,"a":2}`)) + "." + encode([]byte(`{"a":1}`)) + ".sig", want: `invalid header: duplicate JSON object key "a"`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := current.Run(context.Background(), tool.Input{Text: tt.token}, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("jwt.decode error = %v, want %q", err, tt.want)
			}
		})
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
