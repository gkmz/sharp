package cli

import (
	"bytes"
	"testing"

	"github.com/gkmz/sharp/internal/tools"
)

func TestReviewHelpTokensAsOptionValues(t *testing.T) {
	for _, value := range []string{"--help", "-h"} {
		cmd := NewRootCommand(tools.NewRegistry())
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"json", "get", "--path", value, `{"--help":42,"-h":42}`})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if out.String() != "42\n" {
			t.Fatalf("option value %q triggered help: %s", value, out.String())
		}
	}
}
