# Releasing Sharp

[中文](RELEASING.zh-CN.md)

The release pipeline is prepared for v0.1.0. No release is implied by these instructions or the prepared release notes. Publishing is a maintainer action.

## Version policy

Use SemVer tags prefixed with v. Start at v0.1.0. For pre-1.0 releases, patch versions fix bugs without intentionally breaking existing usage; minor versions add capabilities or make documented incompatible changes. Once v1.0.0 ships, backward-incompatible public CLI/configuration changes require a major version. Pre-releases use tags such as v0.2.0-rc.1. Never move or reuse a published tag; ship a new version for fixes.

## Validate locally

Use the Go version in go.mod or newer. CI uses go.mod and verifies Linux/macOS. Install GoReleaser v2.19.0 from its official release (or Homebrew and verify the version); it is a release-development dependency, not needed to run Sharp. Package smoke verification also needs Python 3.9+.

```sh
make verify
go test -race ./...
goreleaser check
goreleaser release --snapshot --clean
python3 scripts/verify-release.py dist
```

Snapshots are local, unpublished development packages. They can be built in an untagged/dirty checkout, so they are for validation and must not be presented as a tagged release. The packager produces macOS/Linux × amd64/arm64 tar.gz archives, LICENSE, both READMEs, and checksums.txt; the Python check verifies every checksum/archive and runs the host package's version and URL commands. Cross-compilation does not prove execution on every target: native CI runs Linux/macOS on its hosted runner architecture; manually smoke other target architectures before claiming support.

PR CI also runs the snapshot/package check. The release workflow repeats Go verification and race tests on the exact tag on both operating systems. Packaging runs only after both checks pass.

## Prepare the release commit

1. Finish and commit pending work, open a PR, and pass CI before merging into main. The worktree used to tag must be clean.
2. Review docs/releases/v0.1.0.md. For later versions create docs/releases/<tag>.md, with English/Chinese highlights, installation, limits and migration notes. Remove the preparation-status sentences once ready for publication.
3. Move the appropriate Unreleased entries into `## [v0.1.0] - YYYY-MM-DD` in both changelogs, using the actual release preparation date. Keep an empty Unreleased section for later work. Do not claim the draft notes are a published release.
4. Merge this release-preparation change into main and wait for all CI jobs, including package verification.
5. Ensure GitHub Actions is enabled and repository/organization policy permits the workflow's contents:write token to create Releases. Only the packaging job requests write permission. The GoReleaser version is pinned in both workflows; update them together and validate locally.

## Tag and generate the draft

The following commands publish a tag and trigger a remote workflow. Run them only when ready to start the release, after reviewing the exact main commit:

```sh
git switch main
git pull --ff-only origin main
test -z "$(git status --porcelain)"
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

`Release draft` checks that the tagged commit is on origin/main, that a non-empty notes file exists, and that both changelogs contain the dated tag heading. It then builds with version, commit and UTC build date, and creates a **draft** GitHub Release. RC tags are marked pre-release. No Homebrew tap or other external repository is updated.

A pushed tag is already visible and can be used by Go module tooling, even while the GitHub Release remains a draft. Treat tag creation as a publishing step. If the workflow fails, fix the cause and rerun the failed workflow where appropriate; do not force-move a published tag. A code change needs a new version.

## Review and publish

1. Open the draft under https://github.com/gkmz/sharp/releases and verify its target commit, version and reviewed notes.
2. Download the four target archives and checksums.txt to an empty directory. Run the appropriate checksum command and inspect their contents:

```sh
# macOS
shasum -a 256 -c checksums.txt
# Linux
sha256sum -c checksums.txt
```

3. Extract your native archive, run `./sharp version`, test representative CLI commands, and smoke the TUI (j/k, s search, h/l, forms, recent tools, clipboard). Confirm version/commit match the tag. Obtain native checks on the remaining supported targets.
4. Click **Publish release** only after this review. Confirm the download links and pinned-version go install command from a separate installation. Keep release notes and changelogs aligned.

SHA256 detects file corruption; the checksum file and binaries are not signed. macOS binaries are not Developer ID signed or notarized. Document these facts; signing/provenance and Homebrew distribution can be added later.

## Repository settings outside the code

Maintainers should configure main branch protection to require PRs and the CI checks `Verify (ubuntu-latest)`, `Verify (macos-latest)` and `Verify release packages`. Configure a tag ruleset restricting v* creation/deletion, allowing the maintainer release workflow. Consider private vulnerability reporting under Settings → Security; the current security policy must only advertise a working channel. These are repository settings and are not applied by the checked-in workflows.
