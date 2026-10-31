package cli

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/gkmz/sharp/internal/tools"
)

func TestBuildVersionResolution(t *testing.T) {
	for _, tc := range []struct {
		name, version, commit   string
		info                    *debug.BuildInfo
		wantVersion, wantCommit string
	}{
		{"linker", "0.1.0", "abc123", &debug.BuildInfo{Main: debug.Module{Version: "v9.0.0"}}, "0.1.0", "abc123"},
		{"module install", "dev", "", &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}}, "v0.1.0", "unknown"},
		{"local checkout", "dev", "", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}, {Key: "vcs.modified", Value: "true"}}}, "dev", "abc123-dirty"},
		{"local Go pseudo-version", "dev", "", &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20261030083507-abc123+dirty"}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}, {Key: "vcs.modified", Value: "true"}}}, "dev", "abc123-dirty"},
		{"no build info", "dev", "", nil, "dev", "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, c := resolveBuildVersion(tc.version, tc.commit, tc.info)
			if v != tc.wantVersion || c != tc.wantCommit {
				t.Fatalf("got %q %q, want %q %q", v, c, tc.wantVersion, tc.wantCommit)
			}
		})
	}
}

func TestVersionIncludesReleaseMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, BuildDate
	t.Cleanup(func() { Version, Commit, BuildDate = oldVersion, oldCommit, oldDate })
	Version, Commit, BuildDate = "0.1.0", "abc123", "2026-10-30T00:00:00Z"
	cmd := NewRootCommand(tools.NewRegistry())
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"version: 0.1.0", "commit: abc123", "built: 2026-10-30T00:00:00Z"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in %q", want, out.String())
		}
	}
}
