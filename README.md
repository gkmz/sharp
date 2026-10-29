# Sharp

```text
   _____ __  _____    ____  ____
  / ___// / / /   |  / __ \/ __ \
  \__ \/ /_/ / /| | / /_/ / /_/ /
 ___/ / __  / ___ |/ _, _/ ____/
/____/_/ /_/_/  |_/_/ |_/_/
        local dev toolkit
```

[![Go Version](https://img.shields.io/badge/Go-1.25.1-00ADD8?style=for-the-badge&logo=go)](go.mod)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=for-the-badge)](LICENSE)
[![TUI](https://img.shields.io/badge/TUI-Bubble%20Tea-FF75B7?style=for-the-badge)](https://github.com/charmbracelet/bubbletea)
[![CLI](https://img.shields.io/badge/CLI-Cobra-6F42C1?style=for-the-badge)](https://github.com/spf13/cobra)
[![Status](https://img.shields.io/badge/Status-Pre--1.0-orange?style=for-the-badge)](docs/CHANGELOG.md)
[![Tests](https://img.shields.io/badge/Tests-go%20test%20%2F%20go%20vet-success?style=for-the-badge)](#development)

[Chinese Documentation](docs/README.zh-CN.md)

`sharp` is a local, terminal-first developer toolkit. It combines a keyboard-driven TUI inspired by lazygit with scriptable CLI commands for JSON, encoding, crypto, time, text, network, conversion, JWT inspection, and generators.

The project is designed for everyday developer workflows: paste data, transform it, inspect the output, chain the result into another action, or call the same tool from shell scripts.

## Highlights

| Capability | Description |
| --- | --- |
| ![TUI](https://img.shields.io/badge/TUI-lazygit--style-FF75B7) | Numbered regions, bordered panes, keyboard-first navigation, status hints, and searchable help. |
| ![CLI](https://img.shields.io/badge/CLI-scriptable-6F42C1) | Every registered tool can be called from shell commands, files, or stdin. |
| ![JSON](https://img.shields.io/badge/JSON-workbench-2EA44F) | Pretty, minify, validate, sort, escape, unescape, path query, and apply-output chaining in one workspace. |
| ![Tools](https://img.shields.io/badge/Tools-built--in-blue) | JSON, Encode, Crypto, Time, Text, Network, Convert, Inspector, and Generator workflows. |
| ![Local](https://img.shields.io/badge/Local-first-lightgrey) | Runs locally and avoids service dependencies for core transformations. |

## Installation

Install the latest version from source:

```bash
go install github.com/gkmz/sharp/cmd/sharp@latest
```

Install from a local checkout:

```bash
git clone https://github.com/gkmz/sharp.git
cd sharp
go install ./cmd/sharp
```

Run directly during development:

```bash
go run ./cmd/sharp
```

## TUI Usage

Start the interactive app:

```bash
sharp
```

The TUI has three numbered regions:

| Region | Purpose |
| --- | --- |
| `1 Categories` | Browse categories and subcategories. Type to filter tools directly here. |
| `2 Actions` | Select an action in the current tool group, such as Encode/Decode or Pretty/Minify. |
| `3 Workspace` | Edit input/options, preview or run the selected action, and inspect output. |

There is no permanent search box. In Categories, type `json`, `minify`, or another tool name to filter immediately. Use Up/Down to select a result, Enter to open it, and Esc to clear the filter. A single tool opens directly in the workspace. Opening a result clears the filter and restores its full action group. `/` starts search from other regions outside edit mode. Letters such as `j`, `k`, and `q` are search text in Categories.

All tool groups use the same workspace rules. Actions within a group share input, output, and input undo history: Base64 Encode/Decode, text transformations, hash algorithms, and regex Match/Replace all keep the current document when switching. Each action remembers its own options in memory; unrelated categories and subgroups keep separate documents. Output titles identify the action that produced the result.

Normal mode handles commands and navigation; only `i` or `o` enters an editor. Tab/Shift+Tab cycle Categories → Actions → Workspace. Use `[`/`]` from any region to switch actions without moving focus, or Left/Right and `h/l` in Actions. Known local transformations preview immediately after `v` paste or action changes, and about 200ms after editing stops; Esc ends editing and flushes the preview. DNS lookup, current time, and random generators run only on `r` or Enter. Options appear only when the action has parameters: `o` edits them, Enter runs, and Esc finishes editing. Inputless actions hide Input while preserving the group's document for other actions.

JSON follows the same rules with Pretty, Minify, Sort, Query, Escape, Unescape, and Check. Path appears only for Query; Enter in Path queries the shared input. Check reports validation in the status line. Across all groups, `p`/`a` applies a successful result to Input and clears Output; errors cannot be applied. While input or options are incomplete, the previous result stays visible as `stale`; copy, save, and Apply are blocked until a fresh result is available.

Input and Output do not need to be selected. Press `i` to edit Input directly, or `o` for Options/Path. Enter inserts a newline in Input; in Options/Path it executes and returns to Normal. While editing, letters are text and Tab does not navigate; use Esc first. In Normal mode, arrows in Workspace scroll Output. Tool options use `key=value` pairs; quote values containing spaces, for example `replacement="hello world"`. Secret options, such as HMAC keys, are masked. `?` lists available commands and option descriptions. Operation feedback appears above the shortcut hints.

Common shortcuts outside edit mode:

| Key | Action |
| --- | --- |
| Tab / Shift+Tab | Cycle regions forward / backward in Normal mode. |
| `[` / `]` | Select the previous / next action without changing region. |
| `1`, `2`, `3` | Focus Categories, Actions, or Workspace. |
| Type in Categories / `/` | Start inline tool search. |
| Up / Down | Select a category or filtered result; scroll Output in Workspace. |
| Left / Right or `h` / `l` | Select an action in region 2. |
| Enter | Open the category selection; run the selected tool/action in regions 2 and 3. |
| `i` | Edit Input directly. |
| `o` | Edit options, or Path when JSON Query is selected. |
| `r` | Run the selected tool or action. |
| `v` | Paste clipboard data into Input and preview, staying in Normal mode. |
| `x` / `X` | Clear input / clear output. |
| `y` | Copy trimmed output. |
| `s` | Save trimmed output to `sharp-output.txt`. |
| `p` / `a` | Apply output to input for chaining. |
| `z` | Undo input changes; Ctrl+Z/Ctrl+Y undo/redo while editing Input. |
| `ctrl+u` / `ctrl+d` | Scroll output by half a page. |
| `alt+u` / `alt+d` | Scroll input by half a page. |
| Esc | Leave editing → Workspace → Actions → Categories; clear an active search filter. |
| `?` | Open searchable command help. |
| `ctrl+c` / `ctrl+q` | Quit from any region; `q` also quits outside Categories, search, and editors. |

## CLI Usage

List tools:

```bash
sharp list
```

Print version information:

```bash
sharp version
```

Examples:

```bash
echo '{"data":{"id":1}}' | sharp json pretty
echo '{"data":{"id":1}}' | sharp json get --path data.id
sharp b64 encode hello
sharp b64 decode aGVsbG8=
sharp url encode "a=b&c=d"
sharp hash sha256 hello
sharp hmac sha256 --key secret hello
sharp time now
sharp time from 1714723200
sharp uuid v4
sharp password --length 32
sharp jwt decode "$JWT"
```

CLI input can come from an argument, a file path, or stdin.

## Tool Categories

| Category | Tools |
| --- | --- |
| ![JSON](https://img.shields.io/badge/JSON-workbench-2EA44F) | `pretty`, `minify`, `validate`, `get`, `sort`, `escape`, `unescape` |
| ![Encode](https://img.shields.io/badge/Encode-code-blue) | Base64, Base64 URL, raw Base64 URL, URL, Hex, HTML, Unicode |
| ![Crypto](https://img.shields.io/badge/Crypto-hash-critical) | CRC32, MD5, SHA1, SHA224, SHA256, SHA384, SHA512, HMAC SHA256, HMAC SHA512 |
| ![Time](https://img.shields.io/badge/Time-timestamp-yellow) | current time, timestamp to time, time to timestamp |
| ![Text](https://img.shields.io/badge/Text-transform-informational) | uppercase, lowercase, trim, case conversion, sort, unique, count, regex test, regex replace |
| ![Network](https://img.shields.io/badge/Network-parse-9cf) | URL parse, query parse, DNS lookup, CIDR parse |
| ![Convert](https://img.shields.io/badge/Convert-format-success) | JSON to YAML, YAML to JSON, CSV to JSON, HTTP headers to JSON |
| ![Inspector](https://img.shields.io/badge/Inspector-JWT-lightgrey) | JWT decode without signature verification |
| ![Generator](https://img.shields.io/badge/Generator-random-orange) | UUID v4, random password, random token |

## Tool Roadmap

The checklist below tracks future tool expansion. Existing capabilities are listed in **Tool Categories** above; unchecked items are planned and are not available yet. Priorities indicate the suggested implementation order, not release commitments.

Keep sharp focused on local, composable data and text workflows. A tool is complete when its shared implementation works in both CLI and TUI, relevant tests pass, and the English and Chinese documentation is updated. Check off each item as it ships; proposed command names may change during implementation.

### P0 — Everyday workflows

- [ ] Text diff: compare two inputs and show additions, deletions, and unchanged context.
- [ ] Number base converter: binary, octal, decimal, and hexadecimal.
- [ ] JSON to CSV: complement the existing CSV to JSON converter.
- [ ] String inspector: extend text counts with Unicode code points, invisible characters, and byte details.
- [ ] Backslash escape/unescape: define supported escape rules explicitly alongside existing JSON and Go-style Unicode tools.
- [ ] Random string generator: configurable length and character set, alongside passwords and tokens.
- [ ] Cron parser: explain expressions and show upcoming runs with an explicit time zone.
- [ ] Color converter: HEX, RGB, HSL, HSV, and CMYK.
- [ ] X.509 certificate decoder: inspect PEM/DER certificate fields, validity, SANs, and fingerprints locally.
- [ ] cURL parser: extract method, URL, headers, and body without executing the command.
- [ ] Query string builder: build a query string from structured parameters, complementing query parsing.
- [ ] JWT signature verification: extend the existing decoder with explicit algorithm and key options.

### P1 — Formatting and structured data

- [ ] HTML beautify/minify.
- [ ] CSS beautify/minify.
- [ ] JavaScript/TypeScript beautify/minify.
- [ ] XML beautify/minify.
- [ ] SQL formatter with explicit dialect selection.
- [ ] JSON to code: start with Go structs and TypeScript interfaces; consider Java and Python later.
- [ ] TOML ↔ JSON conversion.
- [ ] ENV ↔ JSON conversion.
- [ ] INI ↔ JSON conversion.
- [ ] Markdown terminal preview.
- [ ] HTTP request builder and sender with timeouts and readable response output.
- [ ] TLS certificate inspector: fetch and inspect certificates from a remote endpoint.
- [ ] IP inspector: local address classification and subnet details.
- [ ] Port connectivity checker.
- [ ] MIME type lookup.
- [ ] Data URI parser.
- [ ] cURL to code: generate request snippets from parsed commands.
- [ ] UUID/ULID tools: add ULID generation and UUID/ULID inspection alongside UUID v4 generation.

### P2 — Specialized and visual tools

- [ ] QR code generator.
- [ ] QR code reader from image files.
- [ ] Base64 image encode/decode with file and Data URI support.
- [ ] HTML browser preview.
- [ ] HTML to JSX conversion.
- [ ] SVG to CSS conversion.
- [ ] Lorem Ipsum generator.
- [ ] PHP ↔ JSON conversion.
- [ ] PHP serialize/unserialize.
- [ ] ERB beautify/minify.
- [ ] LESS beautify/minify.
- [ ] SCSS beautify/minify.

### Supporting capabilities

Implement these as needed by the tools above:

- [ ] Two-input CLI/TUI workspace for text diff and other comparison tools.
- [ ] Explicit text/file/binary input handling and file output that preserves raw bytes.
- [ ] MIME-aware output handling for structured text, images, and previews.

## Development

Requirements:

- Go 1.25.1 or newer matching `go.mod`.

Useful commands:

```bash
make test
make run
make build
make fmt
make tidy
```

Build with a release version:

```bash
go build -ldflags "-X github.com/gkmz/sharp/internal/cli.Version=0.1.0" ./cmd/sharp
```

In sandboxed environments where the default Go build cache is not writable, use:

```bash
GOCACHE=/private/tmp/sharp-gocache go test ./...
GOCACHE=/private/tmp/sharp-gocache go vet ./...
```

Project layout:

```text
cmd/sharp              CLI entrypoint
internal/cli           Cobra command tree
internal/tui           Bubble Tea TUI
internal/tools         Built-in tools and default registry
pkg/tool               Public tool contract and registry
docs                   Product and development notes
```

## Documentation

- [Contributing Guide](./docs/CONTRIBUTING.md)
- [Code of Conduct](./docs/CODE_OF_CONDUCT.md)
- [Security Policy](./docs/SECURITY.md)
- [Changelog](./docs/CHANGELOG.md)
- [Product Requirements](docs/PRD.md)
- [Development Plan](docs/DEVELOPMENT_PLAN.md)

## License

Apache License 2.0. See [LICENSE](LICENSE).
