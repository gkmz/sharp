# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project currently uses pre-1.0 development versions.

## Unreleased

### Added

- Lazygit-inspired TUI layout with numbered focus regions.
- Category and subcategory navigation.
- Search across categories, subcategories, and concrete tools.
- JSON workbench with pretty, minify, validate, sort, escape, unescape, path query, and apply-output chaining.
- Input/output workspaces with visible borders.
- Actions area for common tool commands.
- Help overlay with grouped commands and filtering.
- Output and input half-page scrolling shortcuts.
- Undo/redo support for input editors.
- Registry-level tests for core built-in tools.
- Open source project documentation: README, contributing guide, code of conduct, security policy, and changelog.

### Changed

- README rewritten as professional open source documentation.
- Generic tool UI now adapts to tools that do not require input.
- JSON category now uses a single workbench page instead of separate atomic JSON tabs.
- CLI runtime errors now print a single actionable error without duplicate Cobra usage output.
- Product and development docs now use the current command syntax for JSON path, Base64 URL, time, UUID, and token examples.

### Fixed

- Copy/save/pipe operations use trimmed raw output instead of rendered viewport text.
- Long input and output content wraps to visible pane width.
- JSON validation shortcut no longer conflicts with paste.
- Generator options now reject invalid numeric values instead of silently falling back to defaults.
- TUI option parsing now validates required and unknown options before running a tool.
- TUI option values can contain spaces when quoted, for example `replacement="hello world"`.
- Secret options are masked in the normal TUI workspace view.
- Text case conversion now preserves Unicode letters and digits instead of dropping non-ASCII words.
- JWT decode now rejects malformed token shapes instead of accepting partial or extra segments.
- Network parsers now return clear errors for empty or path-only input.
- Added the missing `RTK.md` referenced by project agent instructions.
