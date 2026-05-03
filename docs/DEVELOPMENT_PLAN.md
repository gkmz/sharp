# sharp Development Plan

## 1. Development Strategy

Build `sharp` in thin vertical slices:

1. Establish project structure and the shared tool interface.
2. Implement CLI mode first for deterministic testing.
3. Add TUI mode on top of the same tool registry.
4. Add tools in priority order.
5. Add workflow features such as clipboard, history, and chaining.

This keeps the core tool logic independent from Bubble Tea and makes every feature testable without a terminal UI.

## 2. Milestones

## Milestone 0: Project Bootstrap

Goal: create a maintainable Go project foundation.

Tasks:

- Initialize Go module.
- Add Cobra command root.
- Add Bubble Tea app skeleton.
- Add package structure.
- Add basic CI-friendly test setup.
- Add lint/format commands in `Makefile`.
- Add tool registry abstraction.

Suggested output:

```text
cmd/sharp/main.go
internal/cli/root.go
internal/tui/app.go
internal/tools/registry.go
pkg/tool/tool.go
Makefile
README.md
```

Acceptance criteria:

- `go test ./...` passes.
- `go run ./cmd/sharp --help` works.
- `go run ./cmd/sharp` opens a minimal TUI.
- Tool registry can register and retrieve a sample tool.

## Milestone 1: Core Tool Runtime

Goal: define the reusable execution model used by CLI and TUI.

Tasks:

- Define `Tool`, `Input`, `Output`, `Option`, and `Options` types.
- Define categories and metadata.
- Implement input resolvers:
  - direct argument
  - stdin
  - file path
- Implement output rendering for:
  - text
  - JSON
  - error
- Add unit tests for input and output behavior.

Acceptance criteria:

- A sample `echo` tool runs from CLI and TUI.
- CLI mode reads stdin correctly.
- Tool errors include actionable messages.

## Milestone 2: JSON MVP

Goal: ship the first high-value developer tool group.

Tasks:

- Add JSON category.
- Implement `json pretty`.
- Implement `json minify`.
- Implement `json validate`.
- Implement `json get` using `gjson` syntax.
- Add JSON error display with parse position when available.
- Add unit tests with valid, invalid, nested, and array JSON.

CLI examples:

```bash
sharp json pretty file.json
sharp json minify file.json
sharp json validate file.json
sharp json get file.json data.user.name
echo '{"data":{"id":1}}' | sharp json get data.id
```

Acceptance criteria:

- Pretty and minify preserve JSON semantics.
- Invalid JSON returns non-zero exit code in CLI mode.
- Query result is stable and script-friendly.
- TUI can select JSON tools and show output.

## Milestone 3: Encoding MVP

Goal: support common encoding and decoding tasks.

Tasks:

- Implement Base64 standard encode/decode.
- Implement Base64 URL-safe encode/decode.
- Implement raw URL-safe Base64 encode/decode.
- Implement URL encode/decode.
- Add clear errors for malformed input.
- Add tests for Unicode, padding, empty input, and invalid input.

CLI examples:

```bash
sharp b64 encode "hello"
sharp b64 decode "aGVsbG8="
sharp b64 encode --url "hello?"
sharp url encode "a=b&c=d"
sharp url decode "a%3Db%26c%3Dd"
```

Acceptance criteria:

- Standard and URL-safe modes are explicit.
- Decode failures return non-zero exit code.
- TUI options can switch encoding mode.

## Milestone 4: Hash and Time MVP

Goal: cover two daily developer utility groups.

Tasks:

- Implement MD5.
- Implement SHA1.
- Implement SHA256.
- Implement SHA512.
- Implement current timestamp display.
- Implement Unix seconds to datetime.
- Implement Unix milliseconds to datetime.
- Implement datetime to Unix seconds/milliseconds.
- Add UTC/local display option.
- Add tests for known hash vectors and timestamp conversions.

CLI examples:

```bash
sharp hash sha256 "hello"
sharp hash md5 file.txt
sharp time now
sharp time from 1714723200
sharp time parse "2026-05-03 12:00:00"
```

Acceptance criteria:

- Hash output matches standard command-line tools.
- Time conversion is deterministic in tests.
- CLI supports both direct input and stdin.

## Milestone 5: Generators MVP

Goal: add common value generation tools.

Tasks:

- Implement UUID v4.
- Implement random password generator.
- Implement random token generator.
- Add options for length and character sets.
- Use cryptographically secure randomness.
- Add tests for length, charset, and uniqueness basics.

CLI examples:

```bash
sharp uuid
sharp password --length 24
sharp token --bytes 32 --encoding base64url
```

Acceptance criteria:

- Random output uses `crypto/rand`.
- Password options are validated.
- TUI can regenerate values with one key.

## Milestone 6: TUI Usability

Goal: make the interactive app genuinely useful.

Tasks:

- Add category navigation.
- Add tool search.
- Add input textarea.
- Add output viewport.
- Add option controls for selected tool.
- Add copy output action.
- Add save output action.
- Add help modal.
- Add error panel.

Acceptance criteria:

- P0 tools are usable without CLI flags.
- Output can be copied to clipboard.
- Long output scrolls correctly.
- Keyboard shortcuts are visible and consistent.

## Milestone 7: Tool Chaining

Goal: make `sharp` more powerful than standalone web utilities.

Tasks:

- Add pipe action in TUI.
- Add tool selector for next step.
- Pass current output as next tool input.
- Track current chain in memory.
- Add recent chains list.

Acceptance criteria:

- User can run `Base64 decode -> JSON pretty -> JSON get` inside TUI.
- CLI chaining works naturally through stdout/stdin.
- Chaining does not persist sensitive input by default.

## Milestone 8: P1 Tool Expansion

Goal: broaden day-to-day coverage after MVP.

Tasks:

- Text case conversion.
- Sort and unique lines.
- Regex test and replace.
- URL parser.
- Query string parser/builder.
- DNS lookup.
- JSON to YAML and YAML to JSON.
- JWT decode.
- HMAC-SHA256.

Acceptance criteria:

- Each tool has CLI command, TUI entry, and tests.
- Tool metadata is searchable.
- Errors are written for humans in TUI and scripts in CLI.

## Milestone 9: Config, History, and Polish

Goal: make the tool comfortable for daily use.

Tasks:

- Add config file support.
- Add favorite tools.
- Add recent tools.
- Add optional history.
- Add per-tool sensitive-history policy.
- Add theme configuration.
- Add shell completion.
- Improve README with examples.

Acceptance criteria:

- `sharp config path` shows config location.
- History is disabled for sensitive tools by default.
- Shell completion works for supported shells.
- README covers install, TUI, CLI, and examples.

## 3. Recommended First Implementation Slice

Start with Milestones 0 to 2:

1. Project bootstrap.
2. Shared tool runtime.
3. JSON MVP.

Reason:

- JSON is the highest-frequency feature.
- It validates the core abstraction.
- It gives both CLI and TUI modes a meaningful first workflow.
- It forces input handling, output handling, errors, and tests to be solved early.

## 4. Initial Technical Decisions

Recommended stack:

- TUI: `github.com/charmbracelet/bubbletea`
- TUI components: `github.com/charmbracelet/bubbles`
- Styling: `github.com/charmbracelet/lipgloss`
- CLI: `github.com/spf13/cobra`
- JSON query: `github.com/tidwall/gjson`
- JSON parsing: Go standard library first, add `github.com/bytedance/sonic` after profiling or large-payload needs
- Clipboard: `github.com/atotto/clipboard`

Recommended principle:

- Use standard library implementations first when they are good enough.
- Add specialized dependencies when they clearly improve performance, compatibility, or UX.

## 5. Testing Plan

Test layers:

- Unit tests for each tool implementation.
- CLI tests for command behavior and exit codes.
- Golden tests for formatted outputs.
- TUI model tests for state transitions where practical.

Minimum coverage expectation for MVP:

- JSON tools: valid, invalid, nested, array, empty input.
- Encoding tools: normal, URL-safe, raw, invalid, Unicode.
- Hash tools: standard test vectors.
- Time tools: fixed time zone and fixed timestamp.
- Generators: length and charset constraints.

## 6. Release Plan

## v0.1.0

Contents:

- CLI root
- TUI shell
- Tool registry
- JSON pretty/minify/validate/get
- Base64 standard and URL-safe encode/decode
- URL encode/decode
- Hash MD5/SHA1/SHA256/SHA512
- Time now/from/parse
- UUID/password/token

Release quality:

- Works on macOS and Linux.
- `go test ./...` passes.
- README has install and usage examples.

## v0.2.0

Contents:

- Better TUI navigation and search
- Tool chaining
- JWT decode
- HMAC-SHA256
- Text tools
- JSON to YAML/YAML to JSON

## v0.3.0

Contents:

- Network tools
- HTTP requester
- DNS lookup
- URL parser
- Config, favorites, and history

## 7. Immediate Next Task

Implement Milestone 0:

- Initialize the Go module.
- Add Cobra root command.
- Add Bubble Tea minimal app.
- Add shared tool registry.
- Add one sample tool.
- Verify with `go test ./...` and `go run ./cmd/sharp --help`.

