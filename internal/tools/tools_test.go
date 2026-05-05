package tools_test

import (
	"context"
	"strings"
	"testing"

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
