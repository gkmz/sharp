# TUI visual refinement

**Goal:** Improve hierarchy in the approved three-region design without changing interactions.

**Design:** Make document headings readable independently of region focus. Reserve bold for group headings and current selections; retain green focus and blue selection. Use readable subgroup names shared by Categories and Actions, with right-aligned tool counts. Keep the primary paste/edit guidance in Input and shorten automatic Output guidance to a muted waiting message. Terminal font settings remain controlled by the terminal.

**Implementation:** Work in the existing feature branch and preserve all previous uncommitted changes. No new dependencies or commits are required for this round.

- [x] Share subgroup display names in `navigation.go`; fit category labels and counts to the actual available width in `app.go`. Verify count alignment, truncation, readable names, and existing search/navigation.
- [x] Refine semantic text/accent colors and font weights in `theme.go`; use readable open-section headings and primary Input guidance in `workspace.go`. Shorten the automatic Output hint without changing document contents or manual-run guidance.
- [x] Update changelogs, inspect colored frames in dark/light themes and narrow/wide sizes, and run `go test ./... -count=1`, `go vet ./...`, and `git diff --check`.

**Verification:** All commands passed. Inspected model-generated ANSI frames rendered at 120×40 (Categories and populated JSON) and 80×24 (dark/light empty states). Count geometry is checked at widths 3, 20, 30, and 48; existing full-frame and interaction coverage remains passing. Preview files are outside the repository at `/tmp/sharp-ui-refinement-20261030`; the temporary capture test was removed.
