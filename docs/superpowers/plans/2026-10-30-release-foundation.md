# Release foundation

**Goal:** Prepare a repeatable v0.1.0 release without publishing a tag or Release during implementation.

**Design:** Linux/macOS CI runs formatting, vet, tests, race tests and builds using go.mod. GoReleaser v2 packages darwin/linux amd64/arm64 with CGO disabled, version/commit/date injection, README, license and SHA256 checksums. A tag workflow repeats validation on the exact tag and verifies it belongs to main, then creates only a GitHub Release draft. Release body uses a tracked, reviewed v0.1.0 notes file. Development builds identify themselves as dev; go install at a module version reads that version from runtime build info. Release docs cover SemVer, changelog promotion, snapshots, installation, tag/release review, and immutable tags.

**Scope:** Work inline preserving pending Categories navigation changes. No Homebrew tap, Windows promise, repository settings mutation, tagging, pushing or automatic commit. All release notes remain explicitly draft until the first publication.

- [x] Add failing version metadata tests for linker override, local VCS and module install fallback; implement metadata rendering without changing tool commands.
- [x] Configure shared verification, Linux/macOS CI, GoReleaser packaging and a tag-to-draft workflow with minimal permissions and main ancestry/tag validation.
- [x] Add English/Chinese install and release guides plus reviewed first-release notes; keep CHANGELOG Unreleased until the actual release preparation commit.
- [x] Verify tests/vet/race/format/build, actionlint and GoReleaser config; create a local snapshot without publishing, inspect all archives/checksums/platform binaries, and run the native packaged CLI. Report hosted workflow checks that cannot run locally.

## Local verification — 2026-10-30

- `make verify`, uncached `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, formatting and `git diff --check` passed.
- actionlint v1.7.12 and GoReleaser v2.19.0 configuration checks passed. A real snapshot build produced all four archives in ignored `dist/` with SHA256 checksums and the required documentation.
- `scripts/verify-release.py dist` passed; a deliberately corrupted checksum was rejected. Platform binary headers matched all four targets. The native macOS arm64 archive passed version metadata and URL encoding checks.
- The packaged TUI passed navigation/search/action smoke checks at 80×24, 120×40 and 160×50, preserving the pending Categories navigation changes.
- Local Go was 1.27.0. GitHub-hosted Linux/macOS jobs using go.mod's Go version have not run; other target architectures were cross-compiled, not executed locally. Creating a Release draft remains untested until an authorized tag push.
- No commit, tag, push, public Release or repository setting change was performed.
