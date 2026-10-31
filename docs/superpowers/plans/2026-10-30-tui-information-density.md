# TUI information density

**Goal:** Apply the approved reduction in repeated labels and visible category scrolling through shared UI code.

**Design:** Region 2 names the group without repeating “Actions”. Output uses a plain heading for results from the selected action; retained results from a different action keep their source name. Stale markers remain visible. Execution feedback uses readable names, with generators reporting “Generated …”, local previews “Updated …”, and other explicit operations “Completed …”. Copy success reads “Copied”. Keep the single description row and existing layout budgets. Category border legends indicate hidden rows above/below without consuming list rows, including during search.

**Scope:** Shared TUI presentation only. Preserve CLI, tool implementations, result data, previews, output lifecycle, undo, secret masking, and all existing shortcuts. Work on the existing feature branch without committing the previous uncommitted changes. No new dependencies, transient notifications, or per-tool layout changes.

- [x] Add navigation coverage for top/middle/bottom category windows, search, and fully visible lists; verify selected rows and frame dimensions remain intact. Observe the missing-indicator failure first.
- [x] Share category window calculations between list rendering and border legends; add optional bottom legend rendering to the shared panel frame.
- [x] Simplify group/output headings and centralize readable execution feedback in the workspace UI helpers. Preserve producing-action labels when output is retained across action selection.
- [x] Update changelogs; inspect colored UUID and JSON frames plus a short category list. Run `go test ./... -count=1`, `go vet ./...`, and `git diff --check`.

**Verification:** All commands passed. Scroll tests first failed for missing indicators, then passed for first/middle/last windows with and without a search row; fully visible, filtered, and empty lists omit the markers. Output tests first failed for repeated current-action names, then passed for JSON, text, and UUID while preserving the source label on retained JSON/text results. Existing output geometry, stale-result, and interaction tests remain passing. Inspected rendered model ANSI frames for UUID at 80×18 in dark/light themes and populated JSON at 120×40. Preview artifacts live outside the repository at `/tmp/sharp-ui-density-20261030`; the temporary capture test was removed.
