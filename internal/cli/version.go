package cli

import (
	"fmt"
	"runtime/debug"
)

// Version is injected by release builds; local source builds default to dev.
var Version = "dev"

// Commit is the source revision injected by the release builder.
var Commit string

// BuildDate is the UTC build timestamp injected by the release builder.
var BuildDate string

func resolveBuildVersion(version, commit string, info *debug.BuildInfo) (string, string) {
	if info != nil {
		// Newer Go versions can embed a pseudo-version even for a local
		// checkout. VCS settings distinguish it from a module installation.
		local := false
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				local = true
			}
		}
		if version == "dev" && !local && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
		if commit == "" {
			dirty := false
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					commit = setting.Value
				case "vcs.modified":
					dirty = setting.Value == "true"
				}
			}
			if dirty && commit != "" {
				commit += "-dirty"
			}
		}
	}
	if commit == "" {
		commit = "unknown"
	}
	return version, commit
}

func buildVersionText() string {
	info, _ := debug.ReadBuildInfo()
	version, commit := resolveBuildVersion(Version, Commit, info)
	text := fmt.Sprintf("version: %s\ncommit: %s", version, commit)
	if BuildDate != "" {
		text += "\nbuilt: " + BuildDate
	}
	return text
}
