# Normal mode and preview implementation plan

**Goal:** Complete the approved region navigation and automatic local preview workflow without Input/Output pane selection.

**Architecture:** Normal mode owns regions and commands; Insert mode owns only Input or Options/Path. A shared revision-based debounce invalidates obsolete previews and marks old output stale. Existing registry/CLI logic and workspace documents remain intact.

**Tech Stack:** Go, Bubble Tea, Bubbles textarea/textinput/viewport.

**Constraints:** Preserve existing workspace changes. No new tools/dependencies, persisted user data, clipboard writes during tests, commits, or pushes. Network/time/random actions remain explicit. Input Enter remains newline; parameter Enter executes and exits; Esc ends editing, then backs through Workspace → Actions → Categories.

## Tasks

- [ ] Add failing model tests in `internal/tui/normal_mode_test.go`: Tab/Shift+Tab region ring, layered Esc, brackets preserving focus, direct i/o, editing Tab never changes panels, default output scrolling, immediate paste preview, debounced typing, obsolete timer rejection, stale output protection, and manual tools.
- [ ] Modify `internal/tui/app.go` and `navigation.go`: route region keys before category typing; keep search/editor keys isolated; route preview messages to their originating page; paste previews in Normal; update help/footer.
- [ ] Modify `generic_page.go`, `json_page.go`, `workspace.go`: remove pane cycling/output focus; highlight only an active editor, direct i/o; add shared preview revisions and 200ms timer; preview on typing/Esc/paste, preserve Enter newline and parameter submit; reject stale copy/save/Apply and keep errors redacted.
- [ ] Update affected existing tests, README translations and changelogs. Run `gofmt -w cmd internal pkg`, `go test ./...`, `go vet ./...`, `git diff --check`; exercise Pretty → input/paste → auto-preview → Normal → switch action in a real terminal.

## Verification commands

```sh
go test ./internal/tui -run TestNormal -count=1
go test ./internal/tui -count=1
gofmt -w cmd internal pkg
go test ./... && go vet ./... && git diff --check
```
