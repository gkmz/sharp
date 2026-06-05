package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gkmz/sharp/internal/tools"
)

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
	cmd.SetArgs([]string{"hash", "sha256", "--bad", "hello"})

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
