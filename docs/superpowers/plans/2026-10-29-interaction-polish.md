# Interaction polish implementation plan

**Goal:** Progress through the approved interaction priorities in reviewable increments, starting with visible modes and predictable Esc behavior.

**Architecture:** Keep page editors and preview logic intact. The root model owns Esc precedence and region navigation. Both page types expose the current editing target so the workspace title and shared footer describe the same state without adding a second mode flag.

**Constraints:** No new tools or dependencies, persistent inputs, clipboard writes in tests, automatic network requests, or changes to CLI behavior. Reuse the two footer rows and existing panel borders; render within 80×24 and 120×40. Mode labels must be understandable without color. Existing document, action, preview, undo and secret masking behavior must remain covered.

## Increment 1: modes and Esc

- [x] Add model tests in `internal/tui/interaction_test.go`: mode/area labels in every region, INSERT Input/Options/Path in both workspace title and footer, text/command isolation, parameter submit returning to Normal, Esc precedence and document preservation, mode rendering at both terminal sizes.
- [x] Run the new tests and confirm failures demonstrate the missing mode labels and idle Esc behavior.
- [x] Add `EditTarget() string` to `toolPage` in `internal/tui/app.go`, implemented by both pages: empty outside editing, Input for input editing, Options/Path for parameter editing. Add a shared styled mode footer in `workspace.go` and put the active mode in the Workspace border. Input hints say Enter newline / Esc normal; parameter hints say Enter run / Esc normal.
- [x] Centralize root Esc handling: Help closes without changing its underlying region; search clears and returns to Categories; an editor exits and flushes preview without changing region; Workspace returns to Actions; Actions returns to Categories; idle Categories does nothing. Keep page-level Esc for editor cleanup and preview invalidation.
- [x] Update help and both README/changelog languages. Run focused tests, full Go tests, vet, whitespace checks and a real PTY smoke flow before reporting completion.

## Remaining approved increments

- [ ] Action switching feedback: distinguish selected action, preview, and explicit execution; explain when manual execution is needed.
- [ ] JSON Query flow: visible empty/missing Path guidance, parameter execution, and clear rules for continuing Query output through other JSON actions.
- [ ] Output lifecycle: unify preview/stale/error/ready presentation and unavailable copy/save/Apply feedback.
- [ ] Contextual shortcut density: prioritize common commands at 80 columns and retain full help access.
- [ ] Search result context: category/action identification, full group restoration and predictable clearing.
- [ ] Repeatable PTY coverage for JSON, Regex, Base64 and generators at both terminal sizes.

## Verification commands

```sh
go test ./internal/tui -run TestInteraction -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

## Increment 1 verification

- New mode and idle-Esc tests failed before implementation; the completed interaction tests and existing TUI suite pass.
- `go test ./... -count=1`, `go vet ./...`, and `git diff --check` passed.
- Real 80×24 PTY flow: search Pretty → Input Insert → type JSON → live preview → Esc Normal → Help → close Help → Esc Actions → Esc Categories. Then search Query → edit Path → Enter returns data and Normal; Ctrl+C exits cleanly.
- Model tests cover both 80×24 and 120×40 with Normal, Insert, Search and Help views, plus Input/Options/Path text isolation and parameter submission.
