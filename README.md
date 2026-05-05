# sharp

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

The TUI is organized into four numbered regions:

| Region | Purpose |
| --- | --- |
| `1 Search` | Search categories, subcategories, and concrete tools. |
| `2 Categories` | Select a category or subcategory. Parent categories with subcategories are headers only. |
| `3 Tools` | Select tools in the current subcategory. JSON uses one Workspace page. |
| `4 Workspace` | Work with input, options/path, actions, and output. |

Common shortcuts:

| Key | Action |
| --- | --- |
| `1`, `2`, `3`, `4` | Focus a region. |
| `/` | Open search. |
| `j` / `k` | Move in the category list. |
| `h` / `l` | Move between tools in region 3. |
| `i` or `Enter` | Edit input when the current tool accepts input. |
| `o` | Edit options, or JSON path in the JSON workbench. |
| `r` | Run the current tool or default action. |
| `v` | Paste clipboard data into input. |
| `x` / `X` | Clear input / clear output. |
| `y` | Copy trimmed output. |
| `s` | Save trimmed output to `sharp-output.txt`. |
| `p` | Pipe output back into input for chaining. |
| `ctrl+u` / `ctrl+d` | Scroll output by half a page. |
| `alt+u` / `alt+d` | Scroll input by half a page. |
| `?` | Open searchable command help. |
| `q` | Quit from normal mode. |

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
