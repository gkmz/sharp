# Contributing to sharp

Thanks for considering a contribution. `sharp` is a terminal-first developer toolkit, so changes should keep the interface fast, predictable, and keyboard-friendly.

## How to Contribute

1. Open an issue for substantial feature work or behavior changes.
2. Fork the repository and create a topic branch.
3. Keep changes focused. Avoid unrelated refactors in feature or bug-fix PRs.
4. Add or update tests for changed behavior.
5. Run formatting and verification before opening a PR.

## Development Setup

```bash
git clone https://github.com/gkmz/sharp.git
cd sharp
go mod download
```

Useful commands:

```bash
make fmt
make test
go vet ./...
```

If your environment cannot write to the default Go build cache:

```bash
GOCACHE=/private/tmp/sharp-gocache go test ./...
GOCACHE=/private/tmp/sharp-gocache go vet ./...
```

## Code Guidelines

- Prefer small tools with clear IDs, names, descriptions, and deterministic behavior.
- Register new tools in `internal/tools/<category>` and wire them through `internal/tools/registry.go`.
- Add registry-level tests in `internal/tools/tools_test.go` for new core behavior.
- Keep TUI interactions keyboard-first and consistent with existing focus, help, and status bar patterns.
- Public Go symbols must have doc comments.
- Prefer structured parsing over ad hoc string manipulation when a standard parser exists.

## Adding a Tool

1. Choose a stable ID, usually `<group>.<action>`, for example `json.pretty`.
2. Implement it as a `tool.SimpleTool` or a `tool.Tool`.
3. Assign the correct `tool.Category`.
4. Define `tool.Option` values for runtime parameters.
5. Add tests for normal and invalid input.
6. Update README tool lists if the new tool is user-facing.

## Pull Request Checklist

- [ ] Code is formatted with `gofmt`.
- [ ] Tests pass with `go test ./...`.
- [ ] `go vet ./...` passes.
- [ ] New behavior is covered by tests.
- [ ] Public API additions have Go doc comments.
- [ ] README or docs are updated when user-facing behavior changes.
