# URL encoding modes and token formats

**Goal:** Extend the approved URL and Token tools through existing options, categories, CLI commands, and shared TUI workspaces.

**Design:** `url.encode` and `url.decode` gain `mode=query|component`, defaulting to query for compatibility. Query uses form/query rules, including space ↔ plus. Component encodes every byte except RFC 3986 unreserved ASCII (`A-Z a-z 0-9 -._~`); spaces become `%20`, literal plus becomes `%2B`, and decoding preserves literal plus. Encode individual URI values such as database passwords, not the entire connection string. Invalid modes/percent escapes return errors without exposing source input in mode errors.

`token` gains `format=hex|base64|base64url`, defaulting to the existing unpadded Base64URL. `bytes` remains 1–4096, default 32, and measures original random bytes. Hex uses lowercase, standard Base64 includes padding, Base64URL omits padding; returned data is a single line. Validate options before obtaining random bytes from `crypto/rand`. Generation remains explicit; parameter edits, history opening, and action navigation do not regenerate tokens.

**Files:** `internal/tools/encode/encode.go` adds shared mode metadata and encoding/decoding branches; `internal/tools/generator/generator.go` adds format metadata and encoding selection. Registry-level coverage lives in `internal/tools/url_token_test.go`; CLI coverage in `internal/cli/root_test.go`; integration coverage in `internal/tui/url_token_test.go`. Update README and changelog in both languages. No dependency, category, command ID, or broad UI changes.

**Execution:** Work in the current feature branch. Preserve all existing interactions and defaults. Do not automatically commit this new feature round.

- [x] Write registry tests for component special characters/UTF-8/byte round-trips, query defaults and plus semantics, malformed escapes/unknown modes, real database-URI password round-trip, and token format lengths/padding/default/invalid options. Observe failures before implementation.
- [x] Implement URL mode and token format options in the existing shared tool registrations; rerun the registry tests.
- [x] Verify CLI examples, help/defaults, invalid options and explicit `--` input. Verify TUI mode editing/previews, independent action drafts, token generation only on explicit submit/run, selected history, and narrow-frame layout.
- [x] Document CLI/TUI usage and raw-byte/output-length distinctions; run `go test ./... -count=1`, `go vet ./...`, and `git diff --check`. Smoke the examples and inspect results before marking complete.

**Verification:** Registry tests first failed for component-space/plus semantics, database password corruption, unsupported format handling, and token encoded lengths; all passed after implementation. CLI/TUI integration checks passed, including the existing every-tool layout/footer checks. The new TUI test initially retained the suffix of a multi-option draft because Ctrl+U deletes only before the retained cursor; the test now sends Ctrl+E then Ctrl+U to replace the full line. No production editor behavior changed.

`go test ./... -count=1`, `go vet ./...`, and `git diff --check` passed. Built `/tmp/sharp-url-token-20261030/sharp` and checked URL examples/defaults plus 32/64-byte token formats using an independent Python decoder. Character counts were 64/44/43 and 128/88/86 for Hex/Base64/Base64URL; invalid options exited with no stdout. Generated tokens were not printed or saved by the smoke script. No new dependencies or automatic commits.
