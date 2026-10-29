# TUI workflow implementation plan

**Goal:** Preserve category browsing while making search transient and JSON actions visible and consistent.

**Architecture:** Keep the shared tool registry and existing in-memory pages. Put category-search handling in `internal/tui/navigation.go`; the root routes navigation and shared actions, while JSON owns the selected operation and document. Keep existing uncommitted work and the tool roadmap.

**Constraints:** No new tools or dependencies. No persistent input/history. CLI behavior stays unchanged. English and Chinese usage instructions must match the interface.

## Tasks

- [x] Add failing model-level tests for typing `json` (including initial `j`), moving filtered results, clearing search with Esc, and protecting input editing from search. Remove the standalone search panel; show the search field only inside Categories while filtering. Use arrows for category navigation and reserve alphabetic keys for search there. Number Categories/Tools/Workspace 1/2/3; `/` remains a global search shortcut outside editors.
- [x] Replace the JSON Workspace placeholder tab with ordered actions: Pretty, Minify, Sort, Query, Escape, Unescape, Check. Preserve the JSON document when selecting actions, run the selected action, and show Path only when querying. Test search-to-action execution, shared input, and query routing through the root model.
- [x] Unify selection versus editing, shared copy/save/apply keys, and event forwarding. Keep errors and validation notifications out of promotable JSON results. Test output scrolling, safe result application, status visibility, and 80x24/120x40/160x50 layouts. Bound footer width and fix output height accounting.
- [x] Update README usage in both languages, run `gofmt` on changed Go files, `go test ./...`, `go vet ./...`, and `git diff --check`. Review the final diff without committing unrelated changes.

## Acceptance sequences

1. Launch → type `json` in Categories → arrows select a result → Enter opens actions/workspace → Esc clears the filter on return.
2. Search `minify` → Enter → `r` produces compact JSON using the existing shared JSON input.
3. Choose Query → edit Path → Enter queries; change actions without losing input or Path.
4. Invalid JSON → run → apply result preserves input; Check reports status without replacing the previous data result.
5. Focus Output → Down/Ctrl+D scroll; notification text remains visible and every rendered line fits the terminal.

## Verification

- Model-level regression tests cover category typing, filtering, result selection, selected-action execution, input preservation, Query, safe Apply, viewport scrolling, bounded rendering, async input messages, and generator hints.
- A code review identified async search/help message routing and generator hints; both received regression tests and fixes.
- Terminal smoke check confirmed startup, inline `json` filtering, and clean exit.
- Full Go tests, Go vet, and whitespace checks are the final validation commands.
