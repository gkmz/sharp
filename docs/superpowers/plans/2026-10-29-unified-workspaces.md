# Unified workspaces implementation plan

**Goal:** Extend the JSON workspace interaction rules to every existing tool group.

**Architecture:** Keep the registry and CLI unchanged. Cache workspaces by category and subgroup, with per-action option drafts; reuse a common panel layout and footer. Give every page the same selection, preview, activation, and safe Apply contract. Retain the dedicated JSON validation and Path behavior.

**Constraints:** Preserve existing uncommitted work. No new tools, dependencies, persisted inputs, or secrets. Preview only known local transformations with nonempty input and valid options. DNS, clocks, and random generators run explicitly.

## Tasks

- [x] Add model-level failing tests in `internal/tui/unified_workspaces_test.go`: Text/Base64/hash actions share input and undo; search and category changes return to the same group; unrelated categories remain isolated; option drafts are restored per action and secrets remain masked.
- [x] Extract generic-page code to `internal/tui/generic_page.go`. Add group identity and human action labels to navigation; introduce `SelectTool`, `Preview`, `SetActive`, and `ApplyOutput` on the shared page contract. Root navigation and shortcuts use this contract for all tools.
- [x] Add failing tests for parameter Enter execution, safe Apply after failures, hidden-field focus, result provenance, and equivalent panel sizing/footers. Implement shared rendering/layout helpers in `internal/tui/workspace.go`, and align generic-page editing and Apply with JSON.
- [x] Cover every registered tool at 80x24 and 120x40. Verify a real terminal session, update English/Chinese usage and changelogs, run `gofmt`, `go test ./...`, `go vet ./...`, and `git diff --check`.

## Acceptance flows

1. Text → Uppercase → paste input → execute → switch Lowercase: same input, new preview; Apply and Undo work across actions.
2. Base64 → Encode → Apply → Decode: the shared input follows the user’s explicit Apply, and decoding recovers the original text.
3. Regex → Match → edit pattern → Enter; switch Replace → edit replacement → Enter; return to Match restores its own options.
4. Failed conversion → Apply: original input remains intact. Every Output title identifies the action that produced it.
5. Generator → options → edit → Enter generates; switching generator actions preserves their own settings without generating automatically.
6. Search any existing action → Enter → execute; clear filter → leave and return to group: selected action and input remain intact.

## Verification

- `gofmt -w cmd internal pkg`, `go test ./...`, `go vet ./...`, and `git diff --check` completed successfully.
- Model coverage checks every registered tool at 80×24 and 120×40, shared documents, action drafts, safe Apply, manual generators, search, secret masking, and parameter undo isolation.
- Real terminal smoke: searched Text Uppercase, entered `Hello Sharp`, ran to `HELLO SHARP`; searched Lowercase, confirmed the same input remained, ran to `hello sharp`, and exited cleanly.
- Code review identified malformed secret option error disclosure and parameter undo mutating Input. Added reproducing tests, observed failures, fixed both, and reran the complete checks.
