# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project currently uses pre-1.0 development versions.

## Unreleased

## [v0.1.0] - 2026-10-31

### Added

- Added a draft-only GoReleaser release pipeline for macOS/Linux amd64/arm64 archives and SHA256 checksums, Linux/macOS verification with race tests, PR package smoke checks, build metadata, first-release notes, and bilingual release/installation guides.
- URL Encode/Decode now supports `mode=component` for URI values such as database passwords, preserving literal plus on decode; `mode=query` remains the default.
- Token generation now supports `format=hex|base64|base64url` with explicit raw-byte sizing; the existing 32-byte unpadded Base64URL default is unchanged.
- Added a Ctrl+R recent-tools picker for the five most recently used actions, with persistent ID-only history, exact-action navigation without execution, and session fallback when storage is unavailable.

- Lazygit-inspired TUI layout with numbered focus regions.
- Category and subcategory navigation.
- Search across categories, subcategories, and concrete tools.
- JSON workbench with pretty, minify, validate, sort, escape, unescape, path query, and apply-output chaining.
- Input/output workspaces with visible borders.
- Contextual shortcut hints for common tool commands.
- Help overlay with grouped commands and filtering.
- Output and input half-page scrolling shortcuts.
- Undo/redo support for input editors.
- Registry-level tests for core built-in tools.
- Open source project documentation: README, contributing guide, code of conduct, security policy, and changelog.

### Changed

- Categories now uses j/k or arrows for Normal navigation and s to enter Search; / remains available across Normal regions. Search keeps letters literal and stays active when the query becomes empty. Updated footer/help hints; s still saves Output in Actions/Workspace.
- Consolidated Convert into four explicit conversion directions, Base64 into six variant-specific actions, and Generator into Password/Token/UUID v4. Categories now has 18 selectable entries for the same 57 actions; shared workspaces preserve input, drafts and output provenance, and search/history retain exact action navigation.
- Replaced the TUI options line with a shared form: fixed labels, individual value editors and secret masks, Tab/Shift+Tab field navigation, and Left/Right enum selection. Values accept spaces and quotes literally; per-action drafts preserve the selected field.
- Category borders now indicate additional rows above/below. Group and current-result headings avoid repeated action labels; retained results keep their source, and execution feedback uses readable tool names.
- Refined document heading contrast and text weight; Categories now uses readable subgroup names and right-aligned counts, with primary guidance in Input and a quieter waiting hint in Output.
- Reworked the TUI visual hierarchy with adaptive colors, green focus borders, full-width category selection, subdued retained selections, and shortcuts without filled backgrounds.
- Workspace now uses open sections, one description row, contextual empty-state guidance, and a clean document view without editor gutters in Normal mode.
- Mode badges now consistently identify Normal, Insert, Search, and Help. Workspace titles name the active editor, and Insert hints explain Enter and Esc.
- Esc now follows one shared priority order: close Help, clear search, finish editing, then return through Workspace → Actions → Categories. Idle Categories preserves operation feedback.

- Normal mode now navigates regions with Tab/Shift+Tab and actions with brackets; Input/Output no longer require pane selection. Paste previews immediately, edits preview after 200ms, and stale results cannot be copied, saved, or applied. Search opens the complete action group.
- All tool groups now share the JSON workspace interaction model: input/undo/output survive action switches and option drafts are kept separately per action in memory.
- Known local transformations preview on action selection; network lookups, clocks, and generators require explicit execution.
- Parameter editing now runs on Enter across every workspace; Apply consistently clears successful output and rejects errors.
- Workspace sizing, footer hints, group/action labels, and output provenance use shared rules.

- Tool search opens inline in Categories with s or /; category navigation uses j/k or arrows and the permanent Search panel is removed.
- TUI focus shortcuts now map to three regions: Categories (1), Tools/Actions (2), and Workspace (3).
- JSON now exposes selectable actions sharing one document, previews action changes, and shows Path only for Query.
- Field selection and editing are separate across all workspaces; `i` edits, Enter runs, `s` saves, and `p` applies output.
- Common action hints moved to the footer, leaving more space for input and output.

- README rewritten as professional open source documentation.
- Generic tool UI now adapts to tools that do not require input.
- JSON category now uses a single workbench page instead of separate atomic JSON tabs.
- CLI runtime errors now print a single actionable error without duplicate Cobra usage output.
- Product and development docs now use the current command syntax for JSON path, Base64 URL, time, UUID, and token examples.

### Fixed

- Fixed YAML merge precedence and scalar key aliases; bounded raw input before parsing, limited expanded node visits, and honored conversion cancellation. Aligned CLI JSON duplicate-key validation, preserved help-like option values, and stopped acronym lookahead at separators.
- Fixed shared JSON action registration and execution of custom validators. Restricted automatic previews to known local tool IDs, skipped unchanged Escape previews, and preserved feedback/footer rows on short terminals. Clarified recent-tool ID retention and timestamp unit inference.
- Help adapts to its viewport width without truncating long key names; overlay clipping preserves ANSI color sequences.
- Undo/redo while editing Options or JSON Path no longer changes Input; malformed secret option errors no longer reveal secret fragments.
- Search results now execute the selected JSON action rather than always formatting JSON.
- Output arrow keys reach the active viewport, operation notifications remain visible, and shortcut hints stay within terminal width.
- JSON output headers no longer hide the final viewport line; errors cannot replace input through Apply.
- JSON Check reports validation separately from data results and consistently rejects duplicate keys.

- Copy/save/pipe operations use trimmed raw output instead of rendered viewport text.
- Long input and output content wraps to visible pane width.
- JSON validation shortcut no longer conflicts with paste.
- Generator options now reject invalid numeric values instead of silently falling back to defaults.
- TUI option parsing now validates required and unknown options before running a tool.
- TUI option values can contain spaces when quoted, for example `replacement="hello world"`.
- Secret options are masked in the normal TUI workspace view.
- Text case conversion now preserves Unicode letters and digits instead of dropping non-ASCII words.
- JWT decode now rejects malformed token shapes instead of accepting partial or extra segments.
- JWT decode now requires JSON header and payload segments.
- Network parsers now return clear errors for empty or path-only input.
- URL parse now accepts common scheme-less host input such as `example.com/path`.
- URL parse now preserves opaque URL data for schemes such as `mailto:`.
- HTTP header conversion now preserves duplicate header values as arrays.
- CSV to JSON now rejects empty or duplicated headers instead of producing ambiguous output.
- CLI tool commands now accept dash-prefixed input such as `-hello` and negative timestamps without requiring `--`.
- Time timestamp conversion now detects seconds, milliseconds, and nanoseconds by digit count without treating signs as digits.
- Time output now avoids undefined `UnixNano` overflow values for timestamps outside Go's native nanosecond range.
- Base64 and hex decoders now ignore formatting whitespace from copied or wrapped encoded text.
- CLI tools without options now accept double-dash-prefixed input such as `--hello` without requiring an extra `--` separator.
- TUI no-input tools now hide the `options` shortcut when the selected tool has no configurable options.
- HTTP header conversion now canonicalizes case-insensitive header names before merging duplicate values.
- JSON path queries now print `null` for existing null values instead of an empty line.
- Lossy JSON transforms now reject duplicate object keys before sorting JSON or converting JSON to YAML.
- JWT decode now emits a valid JSON object so its output can be chained into JSON tools.
- JWT decode now rejects header or payload parts that are not JSON objects, including duplicate object keys.
- Text sort and unique-line tools no longer treat a final file newline as an extra empty line.
- URL parse now accepts scheme-less bracketed IPv6 hosts such as `[::1]:8080/path`.
- Timestamp unit detection now ignores leading zero padding when choosing seconds, milliseconds, or nanoseconds.
- YAML to JSON now normalizes scalar map keys before encoding JSON and rejects normalized duplicate keys or alias cycles.
- YAML to JSON now rejects multi-document YAML streams instead of silently converting only the first document.
- JSON path queries now reject duplicate object keys instead of returning an ambiguous value.
- URL parse now rejects network URL schemes such as `http` or `https` when no host is present.
- HMAC tools now reject empty keys in the shared tool implementation, not only in CLI/TUI option parsing.
- CSV to JSON now trims header names before using them as JSON object keys.
- Text naming-style conversion now preserves camelCase and acronym word boundaries.
- Added the missing `RTK.md` referenced by project agent instructions.
