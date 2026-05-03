# sharp PRD

## 1. Product Summary

`sharp` is an interactive terminal toolkit for developers. It brings common web-based developer utilities into a fast, keyboard-first CLI/TUI experience.

The product should feel closer to `lazygit` than to a loose collection of shell commands: fast launch, searchable tools, editable input, live output preview, copy/save actions, and smooth chaining between tools.

## 2. Target Users

- Backend developers
- Full-stack developers
- SRE, DevOps, and platform engineers
- Security and testing engineers
- Terminal-heavy developers who frequently use browser-based utility sites

## 3. Core Problems

Developers often need to:

- Format, validate, inspect, and query JSON responses.
- Encode and decode Base64, URL strings, HTML entities, and hex values.
- Generate hashes, HMACs, passwords, UUIDs, and tokens.
- Convert timestamps across formats and time zones.
- Inspect JWTs, URLs, HTTP headers, DNS records, and certificates.
- Chain several transformations together quickly.

Browser tools solve these tasks, but they interrupt terminal workflows, may be unsafe for sensitive data, and are hard to compose.

## 4. Product Goals

- Provide a fast local alternative to browser-based developer tools.
- Support both interactive TUI mode and scriptable CLI mode.
- Make common tasks reachable with only a few keystrokes.
- Keep all sensitive operations local by default.
- Allow output from one tool to become input to another tool.
- Build a modular architecture so new tools are easy to add.

## 5. Non-Goals

- Replace full-featured API clients like Postman in the first release.
- Replace `jq` completely in the first release.
- Provide cloud sync, teams, accounts, or hosted services.
- Store secrets, keys, or sensitive operation history by default.

## 6. Product Modes

### 6.1 Interactive TUI Mode

Run:

```bash
sharp
```

Expected experience:

- Full-screen terminal UI.
- Tool categories on the left.
- Current tool input, options, and output on the right.
- Search tools with `/`.
- Copy output with `y`.
- Save output with `s`.
- Pipe current output into another tool with `p`.

### 6.2 CLI Command Mode

Examples:

```bash
sharp json pretty file.json
sharp json minify file.json
sharp json get file.json data.user.name
echo '{"a":1}' | sharp json pretty
sharp b64 encode "hello"
sharp b64 decode --url "aGVsbG8"
sharp hash sha256 "hello"
sharp time 1714723200
sharp uuid
sharp password --length 24
```

CLI mode must be useful for shell scripts and terminal pipelines.

## 7. Information Architecture

Primary categories:

- JSON
- Encode
- Crypto
- Time
- Text
- Network
- Generator
- Inspector
- Convert

Suggested TUI layout:

```text
+--------------------------------------------------+
| sharp                                      /      |
+----------------+---------------------------------+
| Categories     | Tool input                      |
|                |                                 |
| JSON           +---------------------------------+
| Encode         | Tool options                    |
| Crypto         +---------------------------------+
| Time           | Tool output                     |
| Text           |                                 |
| Network        |                                 |
+----------------+---------------------------------+
| Tab focus | Enter run | y copy | p pipe | ? help |
+--------------------------------------------------+
```

## 8. Global Interaction Requirements

### 8.1 Keyboard Shortcuts

- `/`: search tools
- `j/k` or arrow keys: move selection
- `Enter`: select or run
- `Tab`: switch focus
- `Esc`: go back
- `q`: quit current view
- `Ctrl+C`: exit app
- `y`: copy output
- `s`: save output
- `r`: rerun current tool
- `c`: clear current input
- `p`: pipe output to another tool
- `?`: show help for current tool

### 8.2 Input Sources

Tools should accept:

- Manual text input
- Clipboard paste
- File path input
- Standard input
- Output piped from another `sharp` tool

### 8.3 Output Actions

Users should be able to:

- Copy output to clipboard
- Save output to file
- Pipe output to another tool
- View errors with actionable messages

## 9. Feature Requirements

## 9.1 JSON Tools

Priority: P0

Features:

- Pretty print JSON
- Minify JSON into one line
- Validate JSON
- Query JSON by path
- Extract a field
- Sort object keys
- Escape and unescape JSON strings
- Convert JSON to Go struct
- Convert JSON to TypeScript interface
- Convert JSON to YAML
- Compare two JSON documents

Recommended libraries:

- `github.com/tidwall/gjson` for JSON path query
- `github.com/tidwall/sjson` for JSON mutation
- `github.com/bytedance/sonic` for high-performance JSON parsing
- `github.com/itchyny/gojq` if jq-compatible expressions are needed later

MVP scope:

- Pretty
- Minify
- Validate
- Query with `gjson` path syntax
- Extract query result

## 9.2 Encoding Tools

Priority: P0

Features:

- Standard Base64 encode/decode
- URL-safe Base64 encode/decode
- Raw URL-safe Base64 encode/decode
- URL encode/decode
- Hex encode/decode
- HTML escape/unescape
- Unicode escape/unescape
- JWT decode

MVP scope:

- Standard Base64 encode/decode
- URL-safe Base64 encode/decode
- URL encode/decode
- JWT decode without signature verification

## 9.3 Crypto and Hash Tools

Priority: P0 for hash, P1 for encryption

Features:

- MD5
- SHA1
- SHA224
- SHA256
- SHA384
- SHA512
- CRC32
- HMAC-MD5
- HMAC-SHA1
- HMAC-SHA256
- HMAC-SHA512
- bcrypt generate
- bcrypt verify
- AES-GCM encrypt/decrypt
- AES-CBC encrypt/decrypt
- ChaCha20-Poly1305 encrypt/decrypt

MVP scope:

- MD5
- SHA1
- SHA256
- SHA512
- HMAC-SHA256

Security requirements:

- Secret input should be masked in TUI.
- Secret values must not be stored in history by default.
- Tool output should clearly label encoding format.

## 9.4 Time Tools

Priority: P0

Features:

- Unix seconds to datetime
- Unix milliseconds to datetime
- Unix nanoseconds to datetime
- Datetime to Unix seconds
- Datetime to Unix milliseconds
- UTC and local time conversion
- Time zone conversion
- RFC3339 and ISO8601 parsing
- Relative time parsing such as `now+1h` and `now-7d`
- Time difference calculator
- Cron expression explanation
- Next cron run times

MVP scope:

- Unix seconds/milliseconds to local and UTC datetime
- Datetime to Unix seconds/milliseconds
- Current timestamp display

## 9.5 Text Tools

Priority: P1

Features:

- Uppercase/lowercase
- camelCase, PascalCase, snake_case, kebab-case
- Trim whitespace
- Sort lines
- Unique lines
- Count lines, words, runes, and bytes
- Regex tester
- Regex replace
- Text diff
- SQL formatter
- Markdown table formatter

MVP scope:

- Case conversion
- Sort lines
- Unique lines
- Count text

## 9.6 Network Tools

Priority: P1

Features:

- URL parser
- Query parameter parser
- Query parameter builder
- HTTP GET/POST requester
- cURL command parser
- cURL command generator
- DNS lookup
- CIDR calculator
- IP information
- Port connectivity check
- TLS certificate inspector

MVP scope:

- URL parser
- Query encode/decode
- DNS lookup

## 9.7 Generator Tools

Priority: P0

Features:

- UUID v4
- ULID
- NanoID
- Random password
- Random token
- RSA key pair
- Ed25519 key pair
- Fake email/name/IP data

MVP scope:

- UUID v4
- Random password
- Random token

## 9.8 Format Conversion Tools

Priority: P1

Features:

- JSON to YAML
- YAML to JSON
- JSON to TOML
- TOML to JSON
- CSV to JSON
- JSON to CSV
- ENV to JSON
- URL query to JSON
- HTTP headers to JSON

## 9.9 Inspector Tools

Priority: P2

Features:

- JWT inspector
- TLS certificate inspector
- HTTP status code lookup
- MIME type lookup
- SemVer comparator
- Data URI parser
- Docker image name parser
- Kubernetes YAML viewer

## 10. Tool Chaining

Tool chaining is a key differentiator.

Examples:

```text
Base64 decode -> JSON pretty -> JSON query -> copy
JWT decode -> JSON pretty -> extract field
URL decode -> JSON pretty -> save
```

Requirements:

- `p` should open a tool selector and pass current output as next input.
- The app should remember recent chains.
- CLI mode should support shell-native chaining through stdout/stdin.

## 11. History and Privacy

History can improve productivity, but it must be conservative.

Requirements:

- Do not store secrets by default.
- Do not store crypto keys by default.
- Allow users to disable history globally.
- Allow per-tool history policy.
- Mark sensitive tools explicitly.

## 12. Configuration

Config file:

```text
~/.config/sharp/config.yaml
```

Possible settings:

- Theme
- Default time zone
- Default output format
- Clipboard behavior
- History enabled/disabled
- Custom keybindings
- Favorite tools

## 13. Technical Architecture

Suggested package layout:

```text
cmd/sharp/
internal/app/
internal/cli/
internal/tui/
internal/tui/components/
internal/tools/
internal/tools/json/
internal/tools/encode/
internal/tools/crypto/
internal/tools/time/
internal/tools/text/
internal/tools/network/
internal/tools/generator/
internal/config/
internal/clipboard/
internal/history/
pkg/tool/
```

Core tool interface:

```go
type Tool interface {
    ID() string
    Name() string
    Category() string
    Description() string
    Options() []Option
    Run(ctx context.Context, input Input, options Options) (Output, error)
}
```

The same tool implementation should be used by both TUI mode and CLI mode.

Recommended libraries:

- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/bubbles`
- `github.com/spf13/cobra`
- `github.com/tidwall/gjson`
- `github.com/tidwall/sjson`
- `github.com/bytedance/sonic`
- `github.com/atotto/clipboard`
- `github.com/spf13/viper`

## 14. Success Metrics

Early product metrics:

- App cold start under 100 ms for basic startup.
- P0 tools reachable within three keystrokes from TUI home.
- CLI commands work correctly with stdin/stdout.
- JSON formatting and query handle common API payloads smoothly.
- No sensitive input is persisted by default.

## 15. MVP Definition

The first usable version should include:

- TUI shell with categories, search, input, output, and help.
- CLI command shell with Cobra.
- Unified tool registry.
- Input from stdin, file, and direct argument.
- Output to stdout in CLI mode.
- Copy output from TUI mode.
- JSON pretty/minify/validate/query.
- Base64 standard and URL-safe encode/decode.
- URL encode/decode.
- Hash MD5/SHA1/SHA256/SHA512.
- Unix timestamp conversion.
- UUID v4 generation.
- Random password/token generation.

