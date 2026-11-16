# Sharp Theme System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add semantic, switchable Sharp themes with five built-ins, custom YAML overrides, live TUI preview, persistence, and startup overrides.

**Architecture:** A dependency-free `internal/theme` package resolves complete palettes, while `internal/config` loads and atomically edits YAML. The CLI resolves startup precedence and injects immutable theme options into a TUI whose model-local `StyleSet` is propagated to existing pages and Bubble components.

**Tech Stack:** Go 1.25.1, Cobra, Bubble Tea, Bubbles, Lip Gloss, `gopkg.in/yaml.v3`, Go `testing`.

## Global Constraints

- Built-in IDs are exactly `sharp`, `sharp-color`, `lazygit`, `catppuccin`, and `high-contrast`.
- Startup precedence is `--theme` > `SHARP_THEME` > config file > `sharp-color`.
- Every built-in has complete light and dark palettes for all 16 semantic roles in the design.
- Custom themes extend one built-in and use only `#RRGGBB` values; invalid custom themes are rejected independently.
- Themes change colors only. Layout, spacing, fonts, borders, keybindings, tool behavior, and redirected CLI output remain unchanged.
- Live preview must preserve page input, output, focus, options, undo history, and recent-tool history.
- Existing malformed configuration is never overwritten.
- Do not stage unrelated pre-existing worktree changes.

---

## File Map

- `internal/theme/theme.go`: palette, theme, definition, diagnostic, and registry types.
- `internal/theme/builtins.go`: the five complete built-in themes and stable ordering.
- `internal/theme/theme_test.go`: validation, inheritance, isolation, and ordering tests.
- `internal/config/config.go`: YAML load, raw-node preservation, path resolution, and atomic theme save.
- `internal/config/config_test.go`: missing/malformed/partial files, preservation, and write-failure tests.
- `internal/tui/theme.go`: `StyleSet`, semantic status values, and theme-to-Lip-Gloss conversion.
- `internal/tui/theme_picker.go`: picker state, navigation, preview, cancel, confirm, and view.
- `internal/tui/theme_test.go`: style isolation, status rendering, picker behavior, and layout stability.
- `internal/tui/app.go`: options injection, model theme state, overlay routing, and current style usage.
- `internal/tui/generic_page.go`, `json_page.go`, `workspace.go`, `option_form.go`: page-local styles and reapplication to Bubble components/cached content.
- `internal/tui/recent_view.go`, `navigation.go`: model-local styles and semantic status assignments.
- `internal/cli/root.go`: root flag, configuration load, startup precedence, and TUI injection.
- `internal/cli/theme_test.go`: startup precedence and subcommand isolation.
- `README.md`, `docs/README.zh-CN.md`, `docs/CHANGELOG.md`, `docs/CHANGELOG.zh-CN.md`: user-facing theme documentation.
- `docs/assets/demo.tape`, `docs/assets/sharp-demo.gif`: recorded `sharp-color` default-theme demonstration.

---

### Task 1: Theme Domain and Built-in Registry

**Files:**
- Create: `internal/theme/theme.go`
- Create: `internal/theme/builtins.go`
- Create: `internal/theme/theme_test.go`

**Interfaces:**
- Produces: `Palette`, `Theme`, `Definition`, `Diagnostic`, `Registry`, `NewRegistry(map[string]Definition) (*Registry, []Diagnostic)`, `(*Registry).Lookup(string) (Theme, bool)`, and `(*Registry).All() []Theme`.
- Ordering contract: `All()` returns built-ins in the required order, then valid custom themes sorted by ID.

- [ ] **Step 1: Write failing registry tests**

```go
func TestBuiltinsAreCompleteAndStable(t *testing.T) {
    registry, diagnostics := NewRegistry(nil)
    if len(diagnostics) != 0 { t.Fatal(diagnostics) }
    want := []string{"sharp", "sharp-color", "lazygit", "catppuccin", "high-contrast"}
    got := ids(registry.All())
    if !slices.Equal(got, want) { t.Fatalf("IDs = %v, want %v", got, want) }
    for _, current := range registry.All() {
        if err := current.Light.Validate(); err != nil { t.Fatalf("%s light: %v", current.ID, err) }
        if err := current.Dark.Validate(); err != nil { t.Fatalf("%s dark: %v", current.ID, err) }
    }
}

func TestCustomThemeInheritsAndInvalidThemesAreIsolated(t *testing.T) {
    registry, diagnostics := NewRegistry(map[string]Definition{
        "mine": {Extends: "sharp-color", Dark: map[string]string{"focus": "#112233"}},
        "bad":  {Extends: "sharp-color", Dark: map[string]string{"focus": "red"}},
    })
    mine, ok := registry.Lookup("mine")
    if !ok || mine.Dark.Focus != "#112233" || mine.Dark.Text == "" { t.Fatal(mine, ok) }
    if _, ok := registry.Lookup("bad"); ok || len(diagnostics) != 1 { t.Fatal(diagnostics) }
}
```

- [ ] **Step 2: Run tests and confirm the package is missing**

Run: `go test ./internal/theme`

Expected: FAIL because `internal/theme` and its symbols do not exist.

- [ ] **Step 3: Implement palette validation and registry resolution**

```go
type Palette struct {
    Text, Muted, Border, Focus, Accent string
    Category, Group, Action, Shortcut string
    SelectionForeground, SelectionBackground string
    RetainedForeground, RetainedBackground string
    Success, Warning, Error string
}

type Theme struct { ID, Name string; Light, Dark Palette }
type Definition struct { Extends string; Light, Dark map[string]string }
type Diagnostic struct { ThemeID, Message string }

func NewRegistry(definitions map[string]Definition) (*Registry, []Diagnostic)
func (r *Registry) Lookup(id string) (Theme, bool)
func (r *Registry) All() []Theme
```

Validate IDs with `^[a-z0-9][a-z0-9-]*$`, reject built-in replacement, require a built-in `Extends`, reject unknown role keys, and validate colors with `^#[0-9A-Fa-f]{6}$`. Apply partial maps to a copied base palette so definitions cannot mutate built-ins.

- [ ] **Step 4: Define all five light/dark palettes and run tests**

Run: `go test ./internal/theme`

Expected: PASS, including complete role coverage and deterministic order.

- [ ] **Step 5: Commit the theme domain**

```bash
git add internal/theme
git commit -m "feat: add semantic theme registry"
```

---

### Task 2: YAML Configuration and Atomic Persistence

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Interfaces:**
- Consumes: `theme.Definition` from Task 1.
- Produces: `DefaultPath() (string, error)`, `Load(path string) (*Config, error)`, `(*Config).SelectedTheme() string`, `(*Config).ThemeDefinitions() map[string]theme.Definition`, and `(*Config).SaveTheme(id string) error`.

- [ ] **Step 1: Write failing load and preservation tests**

```go
func TestSaveThemePreservesUnknownKeysAndComments(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.yaml")
    original := "# user config\ntheme: sharp\nfuture:\n  enabled: true\n"
    if err := os.WriteFile(path, []byte(original), 0o600); err != nil { t.Fatal(err) }
    cfg, err := Load(path)
    if err != nil { t.Fatal(err) }
    if err := cfg.SaveTheme("lazygit"); err != nil { t.Fatal(err) }
    saved, _ := os.ReadFile(path)
    if !bytes.Contains(saved, []byte("# user config")) ||
       !bytes.Contains(saved, []byte("future:")) ||
       !bytes.Contains(saved, []byte("theme: lazygit")) { t.Fatal(string(saved)) }
}

func TestMalformedConfigIsNotOverwritten(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.yaml")
    original := []byte("theme: [")
    os.WriteFile(path, original, 0o600)
    if _, err := Load(path); err == nil { t.Fatal("expected parse error") }
    saved, _ := os.ReadFile(path)
    if !bytes.Equal(saved, original) { t.Fatalf("file changed: %q", saved) }
}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/config`

Expected: FAIL because the package does not exist.

- [ ] **Step 3: Implement node-preserving YAML load and save**

```go
type Config struct {
    path        string
    document    yaml.Node
    selected    string
    definitions map[string]theme.Definition
}

func DefaultPath() (string, error) {
    dir, err := os.UserConfigDir()
    if err != nil { return "", err }
    return filepath.Join(dir, "sharp", "config.yaml"), nil
}
```

Decode theme definitions into `map[string]theme.Definition` while retaining the original `yaml.Node`. `SaveTheme` updates or appends only the root mapping's `theme` scalar, writes mode `0600` to a temporary file in the same directory, closes it, and renames it over the target. A missing file begins with an empty document; malformed YAML returns no writable `Config`.

- [ ] **Step 4: Add missing-file, custom-theme, and failure-path tests**

Cover a missing path, optional light/dark maps, directory creation mode, temp-write failure, and rename failure through package-local injected functions rather than changing process permissions.

- [ ] **Step 5: Run configuration tests**

Run: `go test ./internal/config`

Expected: PASS with no file left behind after simulated failures.

- [ ] **Step 6: Commit configuration support**

```bash
git add internal/config
git commit -m "feat: add theme configuration persistence"
```

---

### Task 3: Model-local StyleSet and Semantic Status

**Files:**
- Modify: `internal/tui/theme.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/generic_page.go`
- Modify: `internal/tui/json_page.go`
- Modify: `internal/tui/workspace.go`
- Modify: `internal/tui/option_form.go`
- Modify: `internal/tui/recent_view.go`
- Modify: `internal/tui/navigation.go`
- Create: `internal/tui/theme_test.go`

**Interfaces:**
- Consumes: `theme.Theme` from Task 1.
- Produces: `NewStyleSet(theme.Theme) StyleSet`, `statusMessage{text string, kind statusKind}`, `statusMessage.Render(StyleSet) string`, and `toolPage.ApplyStyles(StyleSet)`.

- [ ] **Step 1: Write failing isolation and reapplication tests**

```go
func TestModelsKeepIndependentStyles(t *testing.T) {
    themes, _ := theme.NewRegistry(nil)
    sharp, _ := themes.Lookup("sharp")
    lazy, _ := themes.Lookup("lazygit")
    first := newModelWithOptions(tools.NewRegistry(), Options{Registry: themes, ActiveTheme: sharp})
    second := newModelWithOptions(tools.NewRegistry(), Options{Registry: themes, ActiveTheme: lazy})
    if first.styles.Category.GetForeground() == second.styles.Category.GetForeground() {
        t.Fatal("themes share category color")
    }
}

func TestApplyThemePreservesWorkspaceState(t *testing.T) {
    m := newThemeTestModel(t, "sharp")
    page := m.currentPage().(*genericToolPage)
    page.SetInput("keep me")
    before := page.OutputText()
    m.applyTheme("lazygit")
    if page.input.Value() != "keep me" || page.OutputText() != before { t.Fatal("state changed") }
}
```

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/tui -run 'TestModelsKeepIndependentStyles|TestApplyThemePreservesWorkspaceState'`

Expected: FAIL because `StyleSet`, options, and `ApplyStyles` do not exist.

- [ ] **Step 3: Replace package globals with `StyleSet`**

```go
type StyleSet struct {
    Text, Focus, AppTitle, PanelTitle, SectionTitle lipgloss.Style
    Category, Group, Action, Shortcut lipgloss.Style
    ActiveLabel, RetainedLabel, Dim, Error, Warning, Success lipgloss.Style
    SelectedRow, RetainedRow, SectionRule, Footer lipgloss.Style
    StatusKey, StatusText lipgloss.Style
    Border lipgloss.TerminalColor
}

func NewStyleSet(current theme.Theme) StyleSet
```

Convert both palette sides to `lipgloss.AdaptiveColor`. Active and retained selections must set both foreground and background. Keep the original current palette exactly as built-in `sharp`.

- [ ] **Step 4: Make pages and widgets accept and reapply styles**

Add `styles StyleSet` to model and page/form structs. Change constructors to accept it. Add `ApplyStyles(StyleSet)` to `toolPage`; update textarea/textinput styles and call `renderOutput()` after setting the new styles. Pass styles explicitly to workspace rendering helpers instead of consulting package state.

- [ ] **Step 5: Store status semantically and render late**

```go
type statusKind uint8
const (statusMuted statusKind = iota; statusSuccess; statusWarning; statusError)
type statusMessage struct { text string; kind statusKind }
func (s statusMessage) Render(styles StyleSet) string
```

Replace pre-rendered status assignments with constructors such as `mutedStatus(text)`, `successStatus(text)`, `warningStatus(text)`, and `errorStatus(text)`. Preserve `toolPage.Status() string` for callers by rendering with the page's current styles. Render model notifications in `View()`.

- [ ] **Step 6: Run all TUI tests and fix style plumbing regressions**

Run: `go test ./internal/tui`

Expected: PASS. Existing dimensions, editing, preview, recent view, and workspace tests remain unchanged in behavior.

- [ ] **Step 7: Commit model-local styling**

```bash
git add internal/tui
git commit -m "refactor: make TUI styles model local"
```

---

### Task 4: Theme-aware TUI Options and Live Theme Picker

**Files:**
- Create: `internal/tui/theme_picker.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/theme_test.go`
- Modify: `internal/tui/app_test.go`

**Interfaces:**
- Consumes: `theme.Registry`, `theme.Theme`, and `StyleSet`.
- Produces: `Options`, `RunWithOptions(*tool.Registry, Options) error`, picker open/update/view helpers, and `model.applyTheme(string) bool`.
- Compatibility: existing `Run` and `newModel` remain wrappers using `sharp-color`.

- [ ] **Step 1: Write failing picker behavior tests**

```go
func TestThemePickerPreviewCancelAndConfirm(t *testing.T) {
    var saved []string
    m := newPickerTestModel(t, "sharp-color", func(id string) error { saved = append(saved, id); return nil })
    m, _ = updateModel(t, m, "t")
    original := m.activeThemeID
    m, _ = updateModel(t, m, "down")
    preview := m.activeThemeID
    if preview == original || len(saved) != 0 { t.Fatal(preview, saved) }
    m, _ = updateModel(t, m, "esc")
    if m.activeThemeID != original { t.Fatal(m.activeThemeID) }
    m, _ = updateModel(t, m, "t")
    m, _ = updateModel(t, m, "down")
    m, _ = updateModel(t, m, "enter")
    if len(saved) != 1 || saved[0] != m.activeThemeID { t.Fatal(saved) }
}
```

Also test save failure, built-in/custom ordering, no open while editing/searching/overlayed, help commands, narrow layout, and stable row dimensions.

- [ ] **Step 2: Run picker tests and confirm failure**

Run: `go test ./internal/tui -run ThemePicker`

Expected: FAIL because picker state and routing do not exist.

- [ ] **Step 3: Add options and compatibility wrappers**

```go
type Options struct {
    Registry      *theme.Registry
    ActiveTheme   theme.Theme
    SaveTheme     func(string) error
    StartupAlerts []string
}

func Run(registry *tool.Registry) error
func RunWithOptions(registry *tool.Registry, options Options) error
func newModel(registry *tool.Registry) model
func newModelWithOptions(registry *tool.Registry, options Options) model
```

Defaults use a fresh built-in registry and `sharp-color`; a nil saver returns a clear persistence error only when Enter is pressed.

- [ ] **Step 4: Implement picker state and overlay rendering**

```go
type themePicker struct {
    open, saving bool
    originalID string
    selected int
    entries []theme.Theme
}
```

Open with `t` only in Normal mode. `j/k` and arrows wrap and call `applyTheme`; Esc reapplies `originalID`; Enter closes while keeping the preview and calls `SaveTheme`. On save failure, keep the theme active and set an error status containing the config path/error supplied by the callback.

- [ ] **Step 5: Add help/footer integration and startup alerts**

Add `t Theme` to Normal-mode help and footer when width permits. Display startup alerts as warning status without blocking keyboard input. Do not add the shortcut inside text editors.

- [ ] **Step 6: Run all TUI tests**

Run: `go test ./internal/tui`

Expected: PASS, including picker overlay and existing navigation routing.

- [ ] **Step 7: Commit the picker**

```bash
git add internal/tui
git commit -m "feat: add live theme picker"
```

---

### Task 5: CLI Startup Precedence and Subcommand Isolation

**Files:**
- Modify: `internal/cli/root.go`
- Create: `internal/cli/theme_test.go`

**Interfaces:**
- Consumes: `config.Load`, `config.DefaultPath`, `theme.NewRegistry`, and `tui.RunWithOptions`.
- Produces: `selectThemeID(flagID, envID, savedID string) string` and injectable root dependencies for deterministic tests.

- [ ] **Step 1: Write failing precedence and isolation tests**

```go
func TestThemeStartupPrecedence(t *testing.T) {
    cases := []struct{ flag, env, saved, want string }{
        {"lazygit", "sharp", "catppuccin", "lazygit"},
        {"", "sharp", "catppuccin", "sharp"},
        {"", "", "catppuccin", "catppuccin"},
        {"", "", "", "sharp-color"},
    }
    for _, tc := range cases {
        if got := selectThemeID(tc.flag, tc.env, tc.saved); got != tc.want { t.Fatalf("got %q", got) }
    }
}

func TestToolSubcommandDoesNotLoadThemeConfig(t *testing.T) {
    loads := 0
    cmd := newRootCommand(tools.NewRegistry(), rootDependencies{
        loadConfig: func(string) (*config.Config, error) { loads++; return nil, errors.New("must not run") },
    })
    cmd.SetArgs([]string{"json", "pretty", `{"a":1}`})
    if err := cmd.Execute(); err != nil || loads != 0 { t.Fatal(err, loads) }
}
```

- [ ] **Step 2: Run CLI theme tests and confirm failure**

Run: `go test ./internal/cli -run Theme`

Expected: FAIL because the root flag and dependency seam do not exist.

- [ ] **Step 3: Implement root-only theme initialization**

Add a root `--theme` string flag. In the root `RunE` path with no arguments, resolve the config path, load YAML, build the registry, choose the precedence winner, fall back unknown IDs to `sharp-color`, and pass diagnostics to `tui.Options.StartupAlerts`. Bind `SaveTheme` to the loaded config. Keep subcommand paths unchanged and ANSI-free.

- [ ] **Step 4: Cover unknown IDs and config errors**

Test unknown flag/env/config IDs, malformed YAML, invalid custom themes, and unavailable user config directories. Each case launches the injected TUI with `sharp-color` plus a concise alert; none exits before the TUI runner.

- [ ] **Step 5: Run CLI and integration tests**

Run: `go test ./internal/cli ./internal/config ./internal/theme`

Expected: PASS. Existing command parsing and output tests stay green.

- [ ] **Step 6: Commit CLI integration**

```bash
git add internal/cli
git commit -m "feat: resolve themes at TUI startup"
```

---

### Task 6: Visual Coverage and User Documentation

**Files:**
- Modify: `internal/tui/visual_test.go`
- Modify: `README.md`
- Modify: `docs/README.zh-CN.md`
- Modify: `docs/CHANGELOG.md`
- Modify: `docs/CHANGELOG.zh-CN.md`
- Modify: `docs/assets/demo.tape`
- Regenerate: `docs/assets/sharp-demo.gif`

**Interfaces:**
- Consumes: shipped theme IDs, `t` picker behavior, config schema, and startup precedence.
- Produces: regression coverage and matching English/Chinese documentation.

- [ ] **Step 1: Add visual semantic assertions**

Create table-driven tests that render `sharp-color`, `lazygit`, and `high-contrast` at 80x24 and 140x40. Assert each render fits, selected rows retain fixed widths, category/group/action roles emit distinct true-color ANSI sequences, and changing themes does not alter stripped text or dimensions. Exercise both AdaptiveColor branches by setting Lip Gloss background mode in a test-scoped renderer or environment seam and restoring global terminal state in cleanup.

- [ ] **Step 2: Run visual tests**

Run: `go test ./internal/tui -run 'Visual|Theme'`

Expected: PASS for light and dark palettes at both sizes.

- [ ] **Step 3: Document installation-neutral theme usage in both READMEs**

Add matching sections containing:

```text
t opens the theme picker; j/k previews; Enter saves; Esc restores.
sharp --theme lazygit
SHARP_THEME=catppuccin sharp
Precedence: flag > environment > config > sharp-color.
```

Include the exact macOS/Linux config paths and the complete custom-theme YAML example from the design. Explain that `sharp` restores the original restrained palette and that CLI transformation output is unchanged.

- [ ] **Step 4: Update both changelogs**

Record five built-ins, `sharp-color` becoming the default, the `sharp` compatibility theme, live preview/persistence, and custom partial overrides. Do not describe generic input/output syntax highlighting as shipped.

- [ ] **Step 5: Record the default theme demonstration**

Update `docs/assets/demo.tape` to launch `sharp --theme sharp-color`, open the theme picker, preview `lazygit`, cancel back to `sharp-color`, and continue the existing JSON workflow. Run `vhs docs/assets/demo.tape` when VHS is available and verify the regenerated GIF is nonblank. If VHS is unavailable, keep the deterministic tape change and report that the binary GIF could not be regenerated instead of fabricating an asset.

- [ ] **Step 6: Review only theme-related documentation hunks**

Run: `git diff -- README.md docs/README.zh-CN.md docs/CHANGELOG.md docs/CHANGELOG.zh-CN.md`

Expected: Theme sections are bilingual and consistent; pre-existing unrelated edits remain intact.

- [ ] **Step 7: Commit only theme documentation hunks when they can be isolated**

Stage only the theme-related hunks in the four documentation files. If a hunk overlaps a pre-existing user edit, leave that file unstaged and report it instead of staging unrelated content.

Commit message: `docs: document configurable themes`

---

### Task 7: Full Verification and Release Safety

**Files:**
- Modify only files required by failures attributable to the theme work.

**Interfaces:**
- Consumes: all prior tasks.
- Produces: a formatted, vetted, tested, buildable theme release.

- [ ] **Step 1: Format theme implementation**

Run: `gofmt -w internal/theme internal/config internal/tui internal/cli`

Expected: no output.

- [ ] **Step 2: Run focused race tests**

Run: `go test -race ./internal/theme ./internal/config ./internal/tui ./internal/cli`

Expected: PASS with no data races between independently created models.

- [ ] **Step 3: Run repository verification**

Run: `make verify`

Expected: `fmt-check`, `go vet ./...`, `go test ./...`, and `go build ./cmd/sharp` all pass.

- [ ] **Step 4: Verify plain CLI output**

Run: `go run ./cmd/sharp json pretty '{"theme":"plain"}' | od -An -tx1`

Expected: output contains JSON bytes and newline but no `1b 5b` ANSI escape prefix.

Run: `NO_COLOR=1 go run ./cmd/sharp json pretty '{"theme":"plain"}' | od -An -tx1`

Expected: output is identical and contains no ANSI escape prefix.

- [ ] **Step 5: Manually smoke-test the TUI**

Run: `go run ./cmd/sharp --theme lazygit`

Expected: the picker opens with `t`, previews all themes, Esc restores, Enter persists, and a second `go run ./cmd/sharp` loads the saved theme. Restore the user's prior config value after the smoke test.

- [ ] **Step 6: Review final scope and worktree**

Run: `git status --short && git diff --check`

Expected: no whitespace errors; every remaining change is either theme work or a clearly identified pre-existing change. Do not discard or stage unrelated work.
