package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"

	"github.com/gkmz/sharp/internal/tools"
)

func TestURLCommandModes(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"url", "encode", "a b+c"}, "a+b%2Bc"},
		{[]string{"url", "encode", "--mode", "component", "^@#$ a+b"}, "%5E%40%23%24%20a%2Bb"},
		{[]string{"url", "decode", "--mode=component", "a+b%2Bc"}, "a+b+c"},
		{[]string{"url", "decode", "a+b%2Bc"}, "a b+c"},
		{[]string{"url", "encode", "--mode=component", "--", "--password"}, "--password"},
		{[]string{"url", "encode", "--", "--mode=component"}, "--mode%3Dcomponent"},
	} {
		cmd := NewRootCommand(tools.NewRegistry())
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if got := out.String(); got != tc.want+"\n" {
			t.Fatalf("%v = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestTokenCommandFormats(t *testing.T) {
	for _, n := range []int{32, 64} {
		for _, format := range []string{"hex", "base64", "base64url"} {
			cmd := NewRootCommand(tools.NewRegistry())
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{"token", "--bytes", strconv.Itoa(n), "--format", format, ""})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			value := strings.TrimSuffix(out.String(), "\n")
			var decoded []byte
			var err error
			switch format {
			case "hex":
				decoded, err = hex.DecodeString(value)
			case "base64":
				decoded, err = base64.StdEncoding.Strict().DecodeString(value)
			case "base64url":
				decoded, err = base64.RawURLEncoding.Strict().DecodeString(value)
			}
			if err != nil || len(decoded) != n || strings.Count(out.String(), "\n") != 1 {
				t.Fatalf("%s/%d: decoded bytes=%d, error=%v", format, n, len(decoded), err)
			}
		}
	}
}

func TestURLTokenCommandHelpAndInvalidOptions(t *testing.T) {
	for _, tc := range []struct{ args, want []string }{
		{[]string{"url", "encode", "--help"}, []string{"--mode", "component", "query"}},
		{[]string{"url", "decode", "--help"}, []string{"--mode", "literal +"}},
		{[]string{"token", "--help"}, []string{"--bytes", "--format", "base64url", "padding", "output characters"}},
	} {
		cmd := NewRootCommand(tools.NewRegistry())
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs(tc.args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		for _, want := range tc.want {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("help for %v omits %q", tc.args, want)
			}
		}
	}
	for _, args := range [][]string{
		{"url", "encode", "--mode=bad", "secret"},
		{"url", "decode", "--mode=bad", "secret"},
		{"url", "decode", "--mode=component", "%GG"},
		{"token", "--format=bad", ""},
		{"token", "--format=hex", "--bytes=4097", ""},
	} {
		cmd := NewRootCommand(tools.NewRegistry())
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil || out.Len() != 0 {
			t.Fatalf("invalid %v: error=%v, output=%q", args, err, out.String())
		}
	}
}

func TestHelpIncludesLogo(t *testing.T) {
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"local dev toolkit", "Usage:", "Available Commands:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output = %q, want %q", got, want)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	oldVersion := Version
	Version = "1.2.3"
	defer func() {
		Version = oldVersion
	}()

	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"local dev toolkit", "version: 1.2.3"} {
		if !strings.Contains(got, want) {
			t.Fatalf("version output = %q, want %q", got, want)
		}
	}
	for _, unwanted := range []string{"commit:", "date:"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("version output = %q, should not contain %q", got, unwanted)
		}
	}
}

func TestRuntimeErrorsDoNotPrintUsage(t *testing.T) {
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"password", "--length", "abc"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() returned nil error, want invalid length error")
	}
	if got := err.Error(); got != "length must be an integer" {
		t.Fatalf("error = %q, want invalid length message", got)
	}
	if got := out.String(); strings.Contains(got, "Usage:") || strings.Contains(got, "Error:") {
		t.Fatalf("output = %q, should not include cobra usage or duplicate error", got)
	}
}

func TestToolCommandAcceptsDashPrefixedInput(t *testing.T) {
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"hash", "sha256", "-hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	const want = "9806e72c3ed3362b7ded0142804aaec6469b7f85cb973d6f5edd66d782448973"
	if got := strings.TrimSpace(out.String()); got != want {
		t.Fatalf("hash output = %q, want %q", got, want)
	}
}

func TestToolCommandAcceptsDoubleDashPrefixedInputWithoutOptions(t *testing.T) {
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"hash", "sha256", "--hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	const want = "d53b07197ae503c756ef08a6bc88a4dbd7443092abf117e44fb4b50ff1826e6b"
	if got := strings.TrimSpace(out.String()); got != want {
		t.Fatalf("hash output = %q, want %q", got, want)
	}
}

func TestToolCommandAcceptsNegativeInput(t *testing.T) {
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"time", "from", "-1000"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "Unix seconds: -1000") {
		t.Fatalf("time.from output = %q, want negative timestamp result", got)
	}
}

func TestToolCommandParsesLongOptions(t *testing.T) {
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"json", "get", "--path=data.id", `{"data":{"id":42}}`})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "42" {
		t.Fatalf("json.get output = %q, want 42", got)
	}
}

func TestToolCommandRejectsUnknownLongOption(t *testing.T) {
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"json", "get", "--bad", "hello"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() returned nil error, want unknown option error")
	}
	if got := err.Error(); got != "unknown option: --bad" {
		t.Fatalf("error = %q, want unknown option message", got)
	}
}

func TestToolCommandRequiresLongOptionValue(t *testing.T) {
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"json", "get", "--path"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() returned nil error, want missing value error")
	}
	if got := err.Error(); got != "missing value for option: --path" {
		t.Fatalf("error = %q, want missing value message", got)
	}
}
