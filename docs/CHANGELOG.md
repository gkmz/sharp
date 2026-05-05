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

### Fixed

- Copy/save/pipe operations use trimmed raw output instead of rendered viewport text.
- Long input and output content wraps to visible pane width.
- JSON validation shortcut no longer conflicts with paste.
