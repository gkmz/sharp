# Recent tools implementation plan

**Goal:** Add the approved Ctrl+R picker for the five most recently used tool actions, persisted between TUI sessions.

**Architecture:** Keep tool identity/order and atomic JSON persistence in `internal/tui/recent.go`, separate from picker navigation/view code in `recent_view.go`. Successful executions and valid previews caused by document/option interaction publish a tool ID through the shared workspace state; the root update boundary consumes these signals once. Rendering, browsing categories/actions, search-opening previews, failed/incomplete previews, and opening recent items do not record usage. Navigation keeps its existing preview behavior without adding history. JSON Check records successful validation despite having no output. Inputless and network tools remain manual.

**Constraints:** Persist only registered tool IDs, never documents, results, options, or secrets. Deduplicate by action ID, newest first, at most five. Use `os.UserConfigDir()/sharp/recent-tools.json`, private permissions and atomic replacement. Unavailable/corrupt storage degrades to in-memory history and a concise explanation in the picker. Preserve all existing uncommitted work. No new dependencies, automatic commits, or CLI history integration.

**Picker:** Ctrl+R opens only in Normal mode in any of the three regions; editors, search, and Help retain their keys. Arrow keys select, Enter restores the unfiltered category/group and exact action in Workspace without executing/previewing, Esc or Ctrl+R closes and preserves the original context. Freeze the displayed IDs while open. The picker exclusively owns focus and ignores unrelated commands. Empty history explains how to populate it. A compact centered panel uses existing semantic styles and remains within 80×24. Add a visible footer hint and Help entry; keep shortcuts legible at 80 columns.

## Tasks

- [x] Write failing storage tests covering MRU deduplication/limit, restart round-trip, missing/deleted tools, malformed file, and private atomic storage. Implement the history model/persistence in `recent.go`.
- [x] Write failing root interaction tests for actual success vs browsing/failure, explicit/automatic preview and validation, context-safe Ctrl+R, fixed picker order, exact-action restoration without execution, empty history, frame bounds, and persistence failure. Add usage signals to the two shared page types and consume them at the root update boundary.
- [x] Implement picker routing and rendering in `recent_view.go`, update shared modal placement for a compact panel, deactivate background focus, and add footer/Help discoverability.
- [x] Update English/Chinese README and changelogs. Run `go test ./... -count=1`, `go vet ./...`, `git diff --check`, inspect rendered picker frames, and smoke the actual terminal flow. Mark checks complete only after inspecting results.

## Verification commands

```sh
go test ./internal/tui -run TestRecent -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

**Results:** All commands passed. New root tests first failed for missing usage collection/picker, then passed. A further regression test caught action-navigation previews displacing used tools; navigation now consumes those usage signals without adding history. Existing every-tool frame/footer checks remain passing; the Recent hint is appended to Workspace only when space permits, preserving Help on narrow screens.

Inspected five-item and empty pickers at 80×24, a 120×40 picker, and the light theme. Real PTY smoke used the production model with an isolated history path: empty picker → UUID search/open → explicit generate → Recent → exit/restart → Recent → open UUID with no result generated → quit. The persisted file contained only `{"tools":["uuid.v4"]}`. Preview images and the isolated test history are in `/tmp/sharp-ui-recent-20261030`; the temporary capture/PTY harness source was removed. No production history was written by the smoke test.
