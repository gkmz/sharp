# Sharp Theme System Design

Date: 2026-11-16
Status: proposed for implementation

## Summary

Sharp will replace its fixed package-level colors with a semantic theme system. The first release of the system will include five built-in themes, a TUI theme picker with live preview, persistent selection, a startup override, and user-defined color overrides in the Sharp configuration file.

Themes control color roles, not layout, spacing, fonts, border characters, or arbitrary component styles. This keeps the rendering model predictable while allowing a richer lazygit-style interface.

## Goals

- Give categories, groups, actions, focus, selection, shortcuts, and status states distinct visual roles.
- Preserve readable light and dark terminal behavior.
- Let users preview and switch themes without restarting Sharp.
- Persist the selected theme across sessions.
- Let users define a custom theme by extending a built-in theme and overriding selected semantic colors.
- Keep CLI transformations unchanged and free of color when their output is piped or redirected.
- Avoid global mutable styles so tests and multiple model instances remain isolated.

## Non-goals

- A full color editor inside the TUI.
- User-defined layout, spacing, fonts, border characters, or keybindings.
- A plugin or downloadable theme marketplace.
- Generic syntax highlighting for arbitrary Input and Output text.
- Runtime file watching or automatic theme reload after startup.
- Guaranteed reproduction of another product's palette or layout.

Structured syntax highlighting can be designed later for renderers that know the output format. The first theme release keeps document text neutral and uses color for interface hierarchy and state.

## User Experience

### Built-in themes

Sharp ships with these stable theme IDs:

- `sharp`: the current restrained appearance, retained for compatibility.
- `sharp-color`: a balanced semantic palette and the new default.
- `lazygit`: a more expressive high-differentiation palette inspired by terminal Git tools.
- `catppuccin`: a soft, multi-color palette with light and dark variants.
- `high-contrast`: stronger contrast and fewer low-saturation distinctions.

Theme IDs are configuration contracts. Display names may be friendlier, but IDs do not change after release.

### Theme picker

In TUI Normal mode, `t` opens the theme picker from Categories, Actions, or Workspace. The picker lists built-in themes first and valid custom themes afterward.

- Up/Down or `j`/`k` changes the highlighted theme and previews it immediately.
- Enter applies the highlighted theme, writes its ID to configuration, and closes the picker.
- Esc closes the picker and restores the theme that was active when the picker opened.
- `?` includes the theme picker shortcut and picker-specific commands.
- Search and a full color editor are excluded from the first version.

If persistence fails, the selected theme remains active for the current session and Sharp shows an actionable error. A failed save must not terminate the TUI.

### Startup selection

Theme selection at startup uses this precedence:

1. Root `--theme <id>` flag.
2. `SHARP_THEME` environment variable.
3. `theme` in the Sharp configuration file.
4. `sharp-color` default.

The precedence determines only the startup theme. A user may still change the active theme in the picker. Saving a picker selection updates the configuration file; a command-line or environment override will take precedence again on the next launch.

An unknown theme ID falls back to `sharp-color` and produces a non-fatal warning in the TUI status area. CLI commands continue normally.

## Semantic Color Model

Each theme provides light and dark palettes. Lip Gloss chooses the appropriate side using the terminal background and performs its normal color-profile downsampling.

The first palette contains these roles:

| Role | Purpose |
| --- | --- |
| `text` | Normal document and interface text. |
| `muted` | Descriptions, secondary labels, inactive hints. |
| `border` | Inactive panel borders and section rules. |
| `focus` | Focused border, focused panel title, active cursor. |
| `accent` | General interactive accent and retained selection. |
| `category` | Top-level category labels. |
| `group` | Nested category/group labels. |
| `action` | Action labels and action-level emphasis. |
| `shortcut` | Footer and help key labels. |
| `selection_foreground` | Foreground of the active row or action. |
| `selection_background` | Background of the active row or action. |
| `retained_foreground` | Foreground of an inactive retained selection. |
| `retained_background` | Background of an inactive retained selection. |
| `success` | Successful operations and confirmations. |
| `warning` | Stale output, caution, non-fatal fallback. |
| `error` | Failed operations and invalid state. |

Input and Output body text use `text` in the first version. The theme does not parse arbitrary content to guess strings, numbers, tags, users, or commits.

Color is applied by meaning:

- Focus stays consistent across all panels.
- Selection is always readable as a foreground/background pair.
- Category, group, and action colors express hierarchy but do not override active selection colors.
- Success, warning, and error are reserved for state and never reused as decorative category colors.
- Borders remain mostly neutral so the full screen does not become uniformly saturated.

## Configuration

Sharp introduces a general configuration file under the OS user configuration directory:

- macOS: `~/Library/Application Support/sharp/config.yaml`
- Linux: `${XDG_CONFIG_HOME:-~/.config}/sharp/config.yaml`

The theme portion is:

```yaml
theme: sharp-color

themes:
  my-theme:
    extends: sharp-color
    dark:
      category: "#73c7ec"
      group: "#c5a3ff"
      focus: "#a6e36f"
      selection_background: "#31456d"
    light:
      category: "#236c85"
      group: "#7352a3"
      focus: "#416e28"
      selection_background: "#d4e1f5"
```

Custom theme rules:

- A custom theme must have a unique ID that does not replace a built-in ID.
- `extends` is required and must name a built-in theme in the first version.
- Light and dark sections are optional partial overrides.
- Color values must use `#RRGGBB` syntax.
- Unknown role names, invalid colors, duplicate IDs, or invalid inheritance reject that custom theme as a unit.
- One invalid custom theme does not prevent other valid themes or the main configuration from loading.
- Configuration writes are atomic. When updating `theme`, Sharp edits only
  that top-level scalar in the parsed YAML document and preserves all other
  keys and comments, including keys introduced by newer Sharp versions.
- If an existing file is malformed, Sharp reports the error and does not
  overwrite it. New files contain only the supported `theme` and `themes` keys
  that are needed.

The selected built-in theme can be saved even when no custom themes exist. Sharp creates the configuration directory and file when the user first persists a selection.

## Architecture

### `internal/theme`

This package owns:

- `Palette`, `Theme`, and custom `Definition` domain types.
- The built-in theme registry.
- Semantic role validation and `#RRGGBB` validation.
- Custom theme inheritance and resolution.
- Lookup by stable theme ID.

It has no Bubble Tea, Lip Gloss, filesystem, or CLI dependencies.

### `internal/config`

This package owns:

- Resolving `os.UserConfigDir()/sharp/config.yaml`.
- YAML decoding into the supported configuration schema.
- Returning valid custom definitions plus non-fatal diagnostics.
- Atomically saving the selected theme while preserving unrelated YAML keys
  and comments.

It depends on `internal/theme` domain types but not on TUI or CLI packages.

### `internal/tui`

`StyleSet` converts a resolved `theme.Theme` into Lip Gloss adaptive colors and
styles. The model owns the current `StyleSet` value. Generic pages, JSON pages,
option forms, recent view, and help rendering receive that value through their
constructors or an `ApplyStyles` method; no package-level style state remains.

Changing a theme builds a new `StyleSet`, stores it on the model, and propagates
it to every already-created page. `ApplyStyles` also updates Bubble components
and regenerates cached viewport content while preserving input, output,
selection, undo history, and focus. Newly created pages receive the active set.
This prevents visual tests and concurrent model instances from affecting each
other.

Status messages are stored as unstyled text plus a semantic kind (`muted`,
`success`, `warning`, or `error`) and styled during rendering. This avoids
retaining ANSI sequences from the previous theme during live preview.

`theme_picker.go` owns picker state:

- Original theme ID for cancel/revert.
- Highlighted theme ID.
- Ordered built-in and custom theme entries.
- Preview, confirm, cancel, and save behavior.

The picker uses the same overlay and key-routing conventions as Help and Recent
Tools. Only one overlay is active at a time. Pressing `t` is handled before
focus-specific navigation, but only while no overlay, search field, or editor
is active.

### `internal/cli`

The root command accepts `--theme <id>` for TUI startup. Configuration and theme
resolution run only when the root command launches the TUI with no subcommand.
That path reads `SHARP_THEME`, resolves the startup theme, and passes a
`tui.Options` value into `tui.Run`.

Tool subcommands keep their existing output and do not emit color. Theme loading failures do not break transformations such as `sharp json pretty`.

## Data Flow

Startup flow:

1. CLI reads the root flag and environment.
2. Config loader reads the file and returns the saved selection, custom definitions, and diagnostics.
3. Theme registry validates and resolves custom themes.
4. Startup precedence chooses a theme ID.
5. TUI receives the registry, resolved theme, config persistence callback, and warnings.
6. TUI builds a model-local `StyleSet` and renders.

Picker flow:

1. User presses `t`; the picker stores the active ID as its revert point.
2. Navigation changes the highlighted ID.
3. The model resolves that ID and rebuilds its `StyleSet` for live preview.
4. Esc reapplies the stored original ID without writing configuration.
5. Enter keeps the previewed theme and atomically persists its ID.

## Error Handling

- Missing configuration is normal and produces no warning.
- Malformed YAML or an unreadable file falls back to built-ins and `sharp-color`, with a concise status warning.
- Invalid custom themes are excluded individually and summarized without exposing file contents.
- Unknown startup IDs fall back to `sharp-color` and identify the unknown ID in the warning.
- Save failures keep the session theme active and report the target path and underlying error.
- Theme errors never alter tool input, output, recent-tool history, or command exit behavior.

## Accessibility and Terminal Compatibility

- Every built-in theme includes light and dark palettes.
- Active selections use both background and foreground changes, not color alone.
- Focus uses color plus the existing panel border/title treatment.
- Errors and warnings retain textual messages; color is supplementary.
- `high-contrast` avoids subtle neighboring shades.
- Rendering must remain legible when Lip Gloss reduces colors for a limited terminal profile.
- `NO_COLOR` behavior remains governed by the existing terminal styling stack; the theme system must not force ANSI output into redirected CLI output.

## Testing

### Theme domain tests

- All built-ins define every semantic role for light and dark palettes.
- Stable IDs are unique.
- Custom partial overrides inherit correctly from built-ins.
- Invalid colors, roles, IDs, and inheritance reject only the affected custom theme.
- Startup precedence is deterministic.

### Configuration tests

- Missing, valid, malformed, and partially invalid configuration files.
- OS configuration path injection for deterministic tests.
- Atomic save that changes only `theme` while preserving unrelated keys and
  comments.
- A malformed existing file is never overwritten by picker persistence.
- Permission and rename failures return actionable errors.

### TUI tests

- `t` opens the picker only from Normal mode.
- Navigation previews without persisting.
- Esc restores the original theme.
- Enter applies and persists.
- Save failure keeps the theme and shows an error.
- Help includes theme commands.
- Existing focus, recent, search, insert, and overlay routing remain intact.
- Visual tests cover one light and two dark built-ins at narrow and wide terminal sizes.
- Rendered rows retain stable dimensions across theme changes.

### Integration checks

- `sharp --theme lazygit` starts the TUI with the requested theme.
- `SHARP_THEME` and configuration precedence work as documented.
- `sharp json pretty` remains free of ANSI styling when piped.
- `make verify` and release-package verification pass.

## Documentation and Release

- Document built-in theme IDs, picker keys, startup precedence, configuration path, and custom theme schema in both READMEs.
- Add a light/dark screenshot or GIF showing `sharp-color`.
- Add the configuration file to the project structure documentation.
- Record the new default and migration path in both changelogs.
- Keep the prior restrained appearance available as `sharp` for users who prefer it.

## Implementation Boundaries

The implementation is complete when:

- The five built-in themes render correctly in light and dark terminals.
- The picker previews, cancels, applies, and persists themes.
- Startup overrides and custom partial themes resolve predictably.
- The current global style variables are replaced with model-local styles.
- Existing CLI output, TUI workflows, and layout tests continue to pass.
- English and Chinese documentation describe the shipped behavior.
