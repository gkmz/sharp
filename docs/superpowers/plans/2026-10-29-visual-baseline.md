# TUI visual baseline

**Goal:** Apply the approved Lazygit-inspired visual hierarchy to the existing three-region interaction model.

**Design:** Use terminal-default background with readable text, muted secondary text and borders, green focus, blue active selection and a subdued retained selection. Focus is exclusive to the current region, or Help when open. Keep one compact mode badge; shortcut keys use foreground color only. The application does not select a terminal font or size.

Keep the outer Workspace panel but replace its inner boxes with open sections. Actions already identifies the tool group: retain only a one-line description above Input/Options/Output. Allocate the recovered header rows to document content. Empty Input and Output explain the relevant next action; they are view-only and must never enter copy/save/Apply data. Hide the textarea prompt/gutter and active-line decoration when displaying the document in Normal mode.

**Constraints:** Preserve the previous uncommitted mode/Esc work, all commands, previews, undo, secret masking, shared documents and CLI behavior. No new application dependencies. Support dark and light terminal backgrounds. Keep the rendered frame within 80×24, 120×40 and 160×50; do not hide the final output line or footer.

## Tasks

- [x] Add focused layout tests in `internal/tui/visual_test.go`: selected category spans available width, workspace description appears once without duplicate tool header or nested boxes, empty-state hints are view-only, and focus changes visible selection styling. Observe failures before implementation.
- [x] Introduce palette/style roles in `internal/tui/theme.go`. Apply active versus retained selections in Categories/Actions, muted inactive borders, exclusive focus under Help, and plain foreground shortcut styling.
- [x] Use open section rendering and empty-state helpers in `internal/tui/workspace.go`; align generic and JSON page views with the new section geometry. Rebudget the root workspace header and editor sizes together. Remove Normal-mode textarea gutters without modifying document content.
- [x] Update README translations and changelogs. Recheck every registered tool at both standard sizes, run full tests/vet/whitespace checks, and inspect real terminal frames for empty, populated, editing, options, generator and Help states.

## Verification

```sh
go test ./internal/tui -run TestVisual -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

All commands passed. The existing every-tool layout tests now budget one description row instead of the removed four-row header. New frame checks preserve the last Output row at 80×24, 120×40 and 160×50. Existing editing, undo, preview and secret masking tests remain green.

Inspected color-rendered model frames for Categories, empty JSON, Insert, Normal, Query, generators and narrow Help. Real PTY smoke covered 80-column startup and a 120-column colored Pretty → input → preview → Esc → Help → close → exit flow.

Visual inspection exposed pre-existing Help issues: long shortcut names were clipped and overlay slicing counted ANSI bytes as visible columns. Added failing tests and fixed responsive Help rows and ANSI-aware slicing using the already-installed `x/ansi` package. No dependency versions were added or changed; tidy promotes the existing ANSI package and the test color-profile package to direct imports.
