package tools_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gkmz/sharp/internal/tools"
	"github.com/gkmz/sharp/pkg/tool"
)

func TestReviewYAMLMergeAndScalarAliases(t *testing.T) {
	current, _ := tools.NewRegistry().Get("yaml.to_json")
	for _, tc := range []struct{ name, input, want string }{
		{"merge", "base: &base {host: localhost, port: 80}\napp: {<<: *base, port: 5432}", `{"base":{"host":"localhost","port":80},"app":{"host":"localhost","port":5432}}`},
		{"explicit before merge", "base: &base {port: 80}\napp: {port: 5432, <<: *base}", `{"base":{"port":80},"app":{"port":5432}}`},
		{"merge sequence precedence", "a: &a {x: 1}\nb: &b {x: 2, y: 3}\napp: {<<: [*a, *b]}", `{"a":{"x":1},"b":{"x":2,"y":3},"app":{"x":1,"y":3}}`},
		{"inline merge", "app: {<<: {host: localhost}}", `{"app":{"host":"localhost"}}`},
		{"quoted merge is literal", "app: {'<<': 42}", `{"app":{"<<":42}}`},
		{"scalar alias key", "base: &k foo\n*k: bar", `{"base":"foo","foo":"bar"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := current.Run(context.Background(), tool.Input{Text: tc.input}, nil)
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal([]byte(out.Text), &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %s, want %s", out.Text, tc.want)
			}
		})
	}
	for _, input := range []string{"app: {<<: 1}", "app: {<<: [{x: 1}, 2]}", "a: &k foo\nfoo: 1\n*k: 2", "a: &a {<<: *a}", "a: &a {child: &b {<<: *a}}"} {
		if _, err := current.Run(context.Background(), tool.Input{Text: input}, nil); err == nil {
			t.Fatalf("accepted invalid YAML %q", input)
		}
	}
}

func TestReviewYAMLExpansionBudgetAndCancellation(t *testing.T) {
	current, _ := tools.NewRegistry().Get("yaml.to_json")
	var input strings.Builder
	input.WriteString("a0: &a0 [x,x,x,x,x,x,x,x,x,x]\n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&input, "a%d: &a%d [%s]\n", i, i, strings.TrimSuffix(strings.Repeat(fmt.Sprintf("*a%d,", i-1), 10), ","))
	}
	if _, err := current.Run(context.Background(), tool.Input{Text: input.String()}, nil); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("unbounded alias expansion: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := current.Run(ctx, tool.Input{Text: "a: 1"}, nil); err != context.Canceled {
		t.Fatalf("ignored cancellation: %v", err)
	}
}

func TestReviewYAMLRepeatedLargeScalarBudget(t *testing.T) {
	current, _ := tools.NewRegistry().Get("yaml.to_json")
	input := "base: &base '" + strings.Repeat("x", 4096) + "'\nitems: [" + strings.TrimSuffix(strings.Repeat("*base,", 5000), ",") + "]"
	if _, err := current.Run(context.Background(), tool.Input{Text: input}, nil); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("unbounded scalar expansion: %v", err)
	}
}

func TestReviewYAMLDepthBudget(t *testing.T) {
	current, _ := tools.NewRegistry().Get("yaml.to_json")
	input := strings.Repeat("[", 300) + "x" + strings.Repeat("]", 300)
	if _, err := current.Run(context.Background(), tool.Input{Text: input}, nil); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("unbounded nesting: %v", err)
	}
}

func TestReviewYAMLInputSizeBeforeParsing(t *testing.T) {
	current, _ := tools.NewRegistry().Get("yaml.to_json")
	const limit = 16 << 20
	// A large comment has no expanded scalar cost, so the raw input needs its own cap.
	input := "#" + strings.Repeat("x", limit-3) + "\n0"
	out, err := current.Run(context.Background(), tool.Input{Text: input}, nil)
	if err != nil || out.Text != "0" {
		t.Fatalf("rejected input at the size limit: %q, %v", out.Text, err)
	}
	for _, oversized := range []string{input + " ", "[" + strings.Repeat("x", limit)} {
		if _, err := current.Run(context.Background(), tool.Input{Text: oversized}, nil); err == nil || !strings.Contains(err.Error(), "YAML input size limit") {
			t.Fatalf("oversized input reached parser: %v", err)
		}
	}
}

func TestReviewJSONValidateRejectsDuplicates(t *testing.T) {
	current, _ := tools.NewRegistry().Get("json.validate")
	for _, input := range []string{`{"a":1,"a":2}`, `{"nested":{"a":1,"a":2}}`} {
		if _, err := current.Run(context.Background(), tool.Input{Text: input}, nil); err == nil {
			t.Fatalf("accepted duplicate keys: %s", input)
		}
	}
}

func TestReviewAcronymLookaheadStopsAtSeparators(t *testing.T) {
	current, _ := tools.NewRegistry().Get("text.snake")
	for _, input := range []string{"ABC def", "ABC\tdef", "ABC\ndef", "ABC-def"} {
		out, err := current.Run(context.Background(), tool.Input{Text: input}, nil)
		if err != nil || out.Text != "abc_def" {
			t.Fatalf("%q: %q, %v", input, out.Text, err)
		}
	}
}

func TestReviewTokensAreNotDeterministic(t *testing.T) {
	current, _ := tools.NewRegistry().Get("token")
	for _, format := range []string{"hex", "base64", "base64url"} {
		opts := tool.Options{"bytes": "32", "format": format}
		first, err := current.Run(context.Background(), tool.Input{}, opts)
		if err != nil {
			t.Fatal(err)
		}
		second, err := current.Run(context.Background(), tool.Input{}, opts)
		if err != nil {
			t.Fatal(err)
		}
		if first.Text == second.Text {
			t.Fatalf("%s tokens are identical", format)
		}
	}
}
