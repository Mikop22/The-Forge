# Shell UX Enhancements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the BubbleTeaTerminal shell UX to parity with Claude Code: slash-command autocomplete drawer, user prompt echo, operation timing, keyboard hints, top status bar, and Esc-to-cancel.

**Architecture:** All changes live inside `BubbleTeaTerminal/`. The autocomplete drawer renders as additional lines appended after the prompt; because `lipgloss.Place` bottom-anchors the whole panel, those extra lines naturally push content upward with no special layout code. Every other feature is an additive change to `session_shell.go` or `model.go` — no new screens are introduced.

**Tech Stack:** Go 1.24, BubbleTea v1.3, Lipgloss v1.1, Bubbles textinput/spinner. Tests use the standard `go test ./...` harness — no external deps.

---

## File Map

| File | Change |
|---|---|
| `model.go` | Add `autocompleteIndex int`; add `operationCancelled bool` |
| `screen_autocomplete.go` (new) | Command registry, filter logic, renderer, keyboard handler |
| `session_shell.go` | `renderTopStrip`, `renderCommandBar` (hint line), `renderFeedContainer` (autocomplete drawer + user echo), `renderOperationLine` (elapsed time + stage) |
| `session_feed.go` | `renderEventRow`: styled error block; add `sessionEventKindUser` kind |
| `session_events.go` | Add `sessionEventKindUser` constant |
| `main.go` | `Update`: route Up/Down/Tab to autocomplete; Esc cancels in-flight forge |
| `main_test.go` + new `autocomplete_test.go` | Tests per feature |

---

## Task 1: User kind + prompt echo in feed

**Files:**
- Modify: `BubbleTeaTerminal/session_events.go`
- Modify: `BubbleTeaTerminal/session_feed.go`
- Modify: `BubbleTeaTerminal/command_router.go` (one line)
- Test: `BubbleTeaTerminal/main_test.go`

- [ ] **Step 1: Write the failing test**

```go
// In main_test.go
func TestUserPromptAppearsInFeedAfterSubmit(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.commandInput.SetValue("glowing war axe")

    updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
    next := updated.(model)

    got := next.View()
    if !strings.Contains(got, "glowing war axe") {
        t.Fatalf("view = %q, want user prompt echoed in feed", got)
    }
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
cd BubbleTeaTerminal && go test -run TestUserPromptAppearsInFeedAfterSubmit -v
```
Expected: FAIL — the prompt text doesn't appear in the feed after submit.

- [ ] **Step 3: Add `sessionEventKindUser` to `session_events.go`**

```go
const (
    sessionEventKindPrompt  sessionEventKind = "prompt"
    sessionEventKindSystem  sessionEventKind = "system"
    sessionEventKindRuntime sessionEventKind = "runtime"
    sessionEventKindFailure sessionEventKind = "failure"
    sessionEventKindHistory sessionEventKind = "history"
    sessionEventKindMemory  sessionEventKind = "memory"
    sessionEventKindUser    sessionEventKind = "user"   // ← add this line
)
```

- [ ] **Step 4: Make user events visible in `session_feed.go`**

In `isVisibleSessionEventKind`, add `sessionEventKindUser` to the visible set:

```go
func isVisibleSessionEventKind(kind sessionEventKind) bool {
    switch kind {
    case sessionEventKindMemory, sessionEventKindFailure, sessionEventKindUser:
        return true
    default:
        return false
    }
}
```

- [ ] **Step 5: Style the user event row in `session_feed.go`**

In `renderEventRow`, add a case before the default label assignment:

```go
case sessionEventKindUser:
    promptArrow := lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(">")
    return promptArrow + " " + styles.Body.Render(event.Message)
```

And update the `label` switch so "USER" is never shown as a plain label:

```go
case sessionEventKindUser:
    // rendered inline above — this branch is unreachable but keeps the switch exhaustive
    label = ">"
```

- [ ] **Step 6: Echo the prompt in `handleShellCommand` (`command_router.go`)**

At the top of `handleShellCommand`, after the empty-check, append the user event before routing:

```go
func (m model) handleShellCommand(raw string) (tea.Model, tea.Cmd) {
    prompt := strings.TrimSpace(raw)
    m.shellNotice = ""
    m.shellError = ""
    m.errMsg = ""
    m.workshopNotice = ""

    if prompt == "" {
        m.shellError = "Prompt cannot be empty."
        m.errMsg = m.shellError
        return m, nil
    }

    // Echo the raw input into the feed before routing.
    m.appendFeedEvent(sessionEventKindUser, prompt)
    m.commandInput.SetValue("")

    route := routeWorkshopCommand(prompt, m.hasActiveWorkshopBench(), m.workshop.Shelf)
    // … rest unchanged …
```

- [ ] **Step 7: Run tests**

```bash
go test ./... -v 2>&1 | tail -20
```
Expected: all pass including `TestUserPromptAppearsInFeedAfterSubmit`.

- [ ] **Step 8: Commit**

```bash
git add BubbleTeaTerminal/session_events.go BubbleTeaTerminal/session_feed.go BubbleTeaTerminal/command_router.go BubbleTeaTerminal/main_test.go
git commit -m "feat: echo user prompt into session feed on submit"
```

---

## Task 2: Styled error blocks

**Files:**
- Modify: `BubbleTeaTerminal/session_feed.go`
- Test: `BubbleTeaTerminal/main_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestErrorEventRendersWithPrefix(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.sessionShell.appendEvent(sessionEventKindFailure, "pipeline collapsed: ArtistAgent failed")

    got := m.View()
    if !strings.Contains(got, "✗") && !strings.Contains(got, "ERROR") {
        t.Fatalf("view = %q, want error event rendered with a visible error prefix", got)
    }
    if !strings.Contains(got, "pipeline collapsed") {
        t.Fatalf("view = %q, want error message text in view", got)
    }
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
go test -run TestErrorEventRendersWithPrefix -v
```
Expected: FAIL — error renders as a plain `ERROR  <text>` line without the styled prefix.

- [ ] **Step 3: Update `renderEventRow` in `session_feed.go`**

Replace the `sessionEventKindFailure` rendering. Find the `renderEventRow` function and add a case:

```go
case sessionEventKindFailure:
    icon := lipgloss.NewStyle().Foreground(colorError).Bold(true).Render("✗")
    label := lipgloss.NewStyle().Foreground(colorError).Render("Error")
    msg := styles.Body.Render(event.Message)
    return icon + " " + label + "  " + msg
```

- [ ] **Step 4: Run tests**

```bash
go test ./...
```
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add BubbleTeaTerminal/session_feed.go BubbleTeaTerminal/main_test.go
git commit -m "feat: style error feed events with icon and label"
```

---

## Task 3: Operation elapsed time + stage label

**Files:**
- Modify: `BubbleTeaTerminal/session_shell.go`
- Test: `BubbleTeaTerminal/main_test.go`

The `operationStartedAt time.Time` field already exists on `model`. This task just formats and renders it.

- [ ] **Step 1: Write the failing test**

```go
func TestOperationLineShowsElapsedTime(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.operationKind = operationForging
    m.operationLabel = "iron sword"
    m.operationStartedAt = time.Now().Add(-15 * time.Second)

    got := m.View()
    if !strings.Contains(got, "15s") && !strings.Contains(got, "14s") {
        t.Fatalf("view = %q, want elapsed seconds in operation line", got)
    }
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
go test -run TestOperationLineShowsElapsedTime -v
```
Expected: FAIL — no elapsed time in the operation line.

- [ ] **Step 3: Add `fmtElapsed` helper in `session_shell.go`**

Add this function anywhere in `session_shell.go`:

```go
func fmtElapsed(start time.Time) string {
    if start.IsZero() {
        return ""
    }
    secs := int(time.Since(start).Seconds())
    if secs < 60 {
        return fmt.Sprintf("%ds", secs)
    }
    return fmt.Sprintf("%dm%ds", secs/60, secs%60)
}
```

Add `"fmt"` to the import block if not already present.

- [ ] **Step 4: Use `fmtElapsed` in `renderOperationLine`**

Replace the existing `renderOperationLine` function:

```go
func renderOperationLine(m model) string {
    label := strings.TrimSpace(m.operationLabel)
    elapsed := fmtElapsed(m.operationStartedAt)
    elapsedSuffix := ""
    if elapsed != "" {
        elapsedSuffix = " " + lipgloss.NewStyle().Foreground(colorDim).Render(elapsed)
    }

    switch m.operationKind {
    case operationForging:
        if label == "" {
            label = "item"
        }
        if m.operationStale {
            return styles.Hint.Render("Still waiting for the forge...") + elapsedSuffix
        }
        stageLabel := strings.TrimSpace(m.stageLabel)
        detail := label
        if stageLabel != "" {
            detail = label + " · " + stageLabel
        }
        return styles.Injecting.Render("⟳ Forging "+detail) + elapsedSuffix
    case operationDirector:
        if label == "" {
            label = "director"
        }
        return styles.Injecting.Render("⟳ Waiting on "+label) + elapsedSuffix
    case operationInjecting:
        if label == "" {
            label = "item"
        }
        return styles.Injecting.Render("⟳ Injecting "+label+" into Terraria") + elapsedSuffix
    default:
        return ""
    }
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./...
```
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add BubbleTeaTerminal/session_shell.go BubbleTeaTerminal/main_test.go
git commit -m "feat: show elapsed time and stage label in operation line"
```

---

## Task 4: Keyboard hint strip

**Files:**
- Modify: `BubbleTeaTerminal/session_shell.go`
- Test: `BubbleTeaTerminal/main_test.go`

The hint strip renders as a dim line directly above the separator line in `renderCommandBar`.

- [ ] **Step 1: Write the failing test**

```go
func TestCommandBarRendersKeyboardHintStrip(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()

    got := m.View()
    if !strings.Contains(got, "/ for commands") {
        t.Fatalf("view = %q, want keyboard hint strip above separator", got)
    }
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
go test -run TestCommandBarRendersKeyboardHintStrip -v
```
Expected: FAIL.

- [ ] **Step 3: Add hint strip to `renderCommandBar` in `session_shell.go`**

Replace the existing `renderCommandBar`:

```go
func (s sessionShellState) renderCommandBar(m model) string {
    hint := styles.Hint.Render("/ for commands · esc to cancel · ctrl+c to quit")
    sep := styles.Hint.Render(strings.Repeat("─", shellContentWidth(m)))
    prompt := styles.Meta.Render(">") + " " + m.commandInput.View()
    return strings.Join([]string{hint, sep, prompt}, "\n")
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./...
```
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add BubbleTeaTerminal/session_shell.go BubbleTeaTerminal/main_test.go
git commit -m "feat: add keyboard hint strip above prompt separator"
```

---

## Task 5: Top status bar

**Files:**
- Modify: `BubbleTeaTerminal/session_shell.go`
- Test: `BubbleTeaTerminal/main_test.go`

`renderTopStrip` currently returns `""`. This task fills it with a one-line status bar: bench name (if active) and runtime status.

- [ ] **Step 1: Write the failing test**

```go
func TestTopStatusBarShowsBenchName(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.workshop.Bench = workshopBench{ItemID: "iron-sword", Label: "Iron Sword"}
    m.width = 120
    m.height = 40

    got := m.View()
    if !strings.Contains(got, "Iron Sword") {
        t.Fatalf("view = %q, want bench label in top status bar", got)
    }
}

func TestTopStatusBarShowsRuntimeOfflineWhenBridgeDown(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.bridgeAlive = false
    m.width = 120
    m.height = 40

    got := m.View()
    if !strings.Contains(got, "offline") {
        t.Fatalf("view = %q, want 'offline' in status bar when bridge is down", got)
    }
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test -run "TestTopStatusBar" -v
```
Expected: FAIL.

- [ ] **Step 3: Implement `renderTopStrip` in `session_shell.go`**

Replace the stub:

```go
func (s sessionShellState) renderTopStrip(m model) string {
    // Don't render a status bar when the terminal hasn't reported its size yet.
    if m.width <= 0 {
        return ""
    }

    dimStyle := lipgloss.NewStyle().Foreground(colorDim)
    boldStyle := lipgloss.NewStyle().Foreground(colorText).Bold(true)

    var parts []string

    if bench := strings.TrimSpace(activeBenchLabel(m)); bench != "" {
        parts = append(parts, boldStyle.Render(bench))
    }

    if m.bridgeAlive || m.workshop.Runtime.BridgeAlive {
        parts = append(parts, dimStyle.Render("runtime online"))
    } else {
        parts = append(parts, dimStyle.Render("runtime offline"))
    }

    return dimStyle.Render(strings.Join(parts, "  ·  "))
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./...
```
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add BubbleTeaTerminal/session_shell.go BubbleTeaTerminal/main_test.go
git commit -m "feat: implement top status bar with bench name and runtime state"
```

---

## Task 6: Esc cancels in-flight forge

**Files:**
- Modify: `BubbleTeaTerminal/screen_forge.go`
- Modify: `BubbleTeaTerminal/session_feed.go` (add cancel event)
- Test: `BubbleTeaTerminal/main_test.go`

Currently `Esc` in `updateForge` only works when `forgeErr != ""`. This task makes Esc work at any point during an active forge.

- [ ] **Step 1: Write the failing test**

```go
func TestEscCancelsInFlightForge(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.state = screenForge
    m.operationKind = operationForging
    m.operationLabel = "iron sword"

    updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
    next := updated.(model)

    if next.state != screenInput {
        t.Fatalf("state after Esc = %v, want screenInput", next.state)
    }
    if next.operationKind != operationIdle {
        t.Fatalf("operationKind after Esc = %v, want operationIdle", next.operationKind)
    }
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
go test -run TestEscCancelsInFlightForge -v
```
Expected: FAIL — Esc only works in the error state, not mid-forge.

- [ ] **Step 3: Expand the Esc handler in `screen_forge.go`**

Find the Esc handler at the top of `updateForge` and replace it:

```go
// Allow escaping both the error state and any active in-flight forge.
if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEsc {
    if m.forgeErr != "" || m.operationKind == operationForging {
        m.state = screenInput
        m.forgeErr = ""
        m.operationKind = operationIdle
        m.operationStale = false
        m.appendFeedEvent(sessionEventKindSystem, "Forge cancelled.")
        m.commandInput.Focus()
        return m, nil
    }
}
```

`sessionEventKindSystem` is already defined and handled as a non-visible event; the cancel message won't clutter the feed. If you want it visible, use `sessionEventKindFailure` instead. Prefer `sessionEventKindSystem` (silent) so the feed stays clean after a manual cancel.

- [ ] **Step 4: Run tests**

```bash
go test ./...
```
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add BubbleTeaTerminal/screen_forge.go BubbleTeaTerminal/main_test.go
git commit -m "feat: Esc cancels in-flight forge and returns to shell"
```

---

## Task 7: Slash-command autocomplete — data and filter

**Files:**
- Create: `BubbleTeaTerminal/screen_autocomplete.go`
- Test: `BubbleTeaTerminal/autocomplete_test.go` (new file)

This task defines the command registry and the filter function. No UI yet.

- [ ] **Step 1: Create `autocomplete_test.go` with failing tests**

```go
package main

import "testing"

func TestAutocompleteFiltersOnPrefix(t *testing.T) {
    items := filterAutocomplete("/fo")
    if len(items) == 0 {
        t.Fatal("filterAutocomplete(\"/fo\") = empty, want at least /forge")
    }
    if items[0].Slash != "/forge" {
        t.Fatalf("first match = %q, want /forge", items[0].Slash)
    }
}

func TestAutocompleteReturnsAllOnSlashOnly(t *testing.T) {
    items := filterAutocomplete("/")
    if len(items) < 8 {
        t.Fatalf("filterAutocomplete(\"/\") = %d items, want all commands (>=8)", len(items))
    }
}

func TestAutocompleteReturnsNilOnNonSlash(t *testing.T) {
    items := filterAutocomplete("radiant spear")
    if items != nil {
        t.Fatalf("filterAutocomplete(non-slash) = %v, want nil", items)
    }
}

func TestAutocompleteReturnsNilOnEmptyInput(t *testing.T) {
    items := filterAutocomplete("")
    if items != nil {
        t.Fatalf("filterAutocomplete(\"\") = %v, want nil", items)
    }
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test -run "TestAutocomplete" -v
```
Expected: FAIL — `filterAutocomplete` undefined.

- [ ] **Step 3: Create `screen_autocomplete.go`**

```go
package main

import "strings"

type autocompleteEntry struct {
    Slash   string // "/forge"
    ArgHint string // "<prompt>"
    Desc    string // one-line description
}

var autocompleteRegistry = []autocompleteEntry{
    {"/forge", "<prompt>", "Generate a new item from scratch"},
    {"/variants", "<direction>", "Generate shelf variants from the bench"},
    {"/bench", "<id or number>", "Set a shelf variant as the active bench"},
    {"/try", "", "Reinject the current bench item into Terraria"},
    {"/restore", "baseline | live", "Restore bench to a previous state"},
    {"/status", "", "Show bench label and runtime state"},
    {"/memory", "", "Show pinned memory notes"},
    {"/what-changed", "", "Summarise changes since last bench"},
    {"/help", "", "List all available commands"},
}

// filterAutocomplete returns matching entries for the given raw input value.
// Returns nil when input is empty or does not start with "/".
func filterAutocomplete(input string) []autocompleteEntry {
    if input == "" || !strings.HasPrefix(input, "/") {
        return nil
    }
    lower := strings.ToLower(input)
    var out []autocompleteEntry
    for _, e := range autocompleteRegistry {
        if strings.HasPrefix(e.Slash, lower) {
            out = append(out, e)
        }
    }
    return out
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./...
```
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add BubbleTeaTerminal/screen_autocomplete.go BubbleTeaTerminal/autocomplete_test.go
git commit -m "feat: add slash-command autocomplete registry and filter"
```

---

## Task 8: Slash-command autocomplete — renderer

**Files:**
- Modify: `BubbleTeaTerminal/screen_autocomplete.go`
- Modify: `BubbleTeaTerminal/model.go`
- Modify: `BubbleTeaTerminal/session_shell.go`
- Test: `BubbleTeaTerminal/autocomplete_test.go`

The drawer renders as lines appended after the prompt. Because `lipgloss.Place` bottom-anchors the full panel, adding lines here naturally shifts all content upward.

- [ ] **Step 1: Write failing render test in `autocomplete_test.go`**

```go
func TestAutocompleteDrawerRendersWhenSlashTyped(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.commandInput.SetValue("/")
    m.width = 120
    m.height = 40

    got := m.View()
    if !strings.Contains(got, "/forge") {
        t.Fatalf("view = %q, want autocomplete drawer showing /forge when '/' typed", got)
    }
    if !strings.Contains(got, "Generate a new item") {
        t.Fatalf("view = %q, want command description in autocomplete drawer", got)
    }
}

func TestAutocompleteDrawerHiddenWhenNoMatch(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.commandInput.SetValue("/zzz")

    got := m.View()
    if strings.Contains(got, "Generate a new item") {
        t.Fatalf("view = %q, want autocomplete hidden when no matches", got)
    }
}

func TestAutocompleteDrawerHighlightsSelectedRow(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.commandInput.SetValue("/")
    m.autocompleteIndex = 1 // second entry: /variants
    m.width = 120

    got := m.View()
    if !strings.Contains(got, "/variants") {
        t.Fatalf("view = %q, want /variants in drawer", got)
    }
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test -run "TestAutocompleteDrawer" -v
```
Expected: FAIL — `m.autocompleteIndex` field undefined, no drawer rendered.

- [ ] **Step 3: Add `autocompleteIndex` to `model.go`**

Inside the `model` struct, add after `commandMode bool`:

```go
autocompleteIndex int
```

- [ ] **Step 4: Add `renderAutocompleteDrawer` to `screen_autocomplete.go`**

```go
// renderAutocompleteDrawer renders the two-column command list below the prompt.
// Returns "" when there are no matches for the current input.
func renderAutocompleteDrawer(m model) string {
    input := m.commandInput.Value()
    entries := filterAutocomplete(input)
    if len(entries) == 0 {
        return ""
    }

    // Clamp the selected index to valid range.
    idx := m.autocompleteIndex
    if idx < 0 {
        idx = 0
    }
    if idx >= len(entries) {
        idx = len(entries) - 1
    }

    slashWidth := 20
    dimStyle := lipgloss.NewStyle().Foreground(colorDim)
    selectedSlash := lipgloss.NewStyle().Foreground(colorRune).Bold(true)
    selectedDesc := lipgloss.NewStyle().Foreground(colorText)

    lines := make([]string, 0, len(entries))
    for i, e := range entries {
        nameCol := e.Slash
        if e.ArgHint != "" {
            nameCol += " " + e.ArgHint
        }
        if i == idx {
            left := selectedSlash.Width(slashWidth).Render(nameCol)
            right := selectedDesc.Render(e.Desc)
            lines = append(lines, left+right)
        } else {
            left := dimStyle.Width(slashWidth).Render(nameCol)
            right := dimStyle.Render(e.Desc)
            lines = append(lines, left+right)
        }
    }
    return strings.Join(lines, "\n")
}
```

Add `"github.com/charmbracelet/lipgloss"` to the import block in `screen_autocomplete.go`:

```go
import (
    "strings"
    "github.com/charmbracelet/lipgloss"
)
```

- [ ] **Step 5: Wire the drawer into `renderFeedContainer` in `session_shell.go`**

At the bottom of `renderFeedContainer`, before the final `strings.Join`, append the drawer as the last body element:

```go
func (s sessionShellState) renderFeedContainer(m model, content string) string {
    feed := s.renderEventRows(m)
    body := []string{renderSplash(m), feed}
    if m.shellError != "" {
        body = append(body, styles.Error.Render(m.shellError))
    } else if m.shellNotice != "" {
        body = append(body, styles.Hint.Render(m.shellNotice))
    }
    if operation := renderOperationLine(m); operation != "" {
        body = append(body, operation)
    }
    if pinned := s.renderPinnedMemoryBlock(); pinned != "" {
        body = append(body, pinned)
    }
    if trimmed := strings.TrimSpace(content); trimmed != "" {
        body = append(body, trimmed)
    }
    return strings.Join(body, "\n")
}
```

Then, in `render`, insert the autocomplete drawer between the feed container and the command bar:

```go
func (s sessionShellState) render(m model, content string) string {
    s.pinnedNotes = loadPinnedMemoryNotes()
    top := s.renderTopStrip(m)
    feed := s.renderFeedContainer(m, content)
    command := s.renderCommandBar(m)
    drawer := renderAutocompleteDrawer(m)

    parts := make([]string, 0, 5)
    if strings.TrimSpace(top) != "" {
        parts = append(parts, top)
    }
    if strings.TrimSpace(feed) != "" {
        parts = append(parts, feed)
    }
    parts = append(parts, command)
    if drawer != "" {
        parts = append(parts, drawer)
    }
    return strings.Join(parts, "\n")
}
```

- [ ] **Step 6: Run tests**

```bash
go test ./...
```
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add BubbleTeaTerminal/screen_autocomplete.go BubbleTeaTerminal/model.go BubbleTeaTerminal/session_shell.go BubbleTeaTerminal/autocomplete_test.go
git commit -m "feat: render slash-command autocomplete drawer below prompt"
```

---

## Task 9: Slash-command autocomplete — keyboard navigation and selection

**Files:**
- Modify: `BubbleTeaTerminal/main.go`
- Test: `BubbleTeaTerminal/autocomplete_test.go`

- [ ] **Step 1: Write failing keyboard tests in `autocomplete_test.go`**

```go
func TestAutocompleteDownArrowMovesSelection(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.commandInput.SetValue("/")
    m.autocompleteIndex = 0

    updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
    next := updated.(model)

    if next.autocompleteIndex != 1 {
        t.Fatalf("autocompleteIndex after Down = %d, want 1", next.autocompleteIndex)
    }
}

func TestAutocompleteUpArrowMovesSelection(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.commandInput.SetValue("/")
    m.autocompleteIndex = 2

    updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
    next := updated.(model)

    if next.autocompleteIndex != 1 {
        t.Fatalf("autocompleteIndex after Up = %d, want 1", next.autocompleteIndex)
    }
}

func TestAutocompleteTabCompletesCommand(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.commandInput.SetValue("/fo")
    m.autocompleteIndex = 0 // /forge is the only match

    updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
    next := updated.(model)

    if next.commandInput.Value() != "/forge " {
        t.Fatalf("input after Tab = %q, want \"/forge \"", next.commandInput.Value())
    }
}

func TestAutocompleteEscDismissesDrawer(t *testing.T) {
    t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
    m := initialModel()
    m.commandInput.SetValue("/")

    updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
    next := updated.(model)

    // After Esc the input should be cleared, drawer gone.
    if got := filterAutocomplete(next.commandInput.Value()); got != nil {
        t.Fatalf("autocomplete still active after Esc, input = %q", next.commandInput.Value())
    }
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test -run "TestAutocomplete(Down|Up|Tab|Esc)" -v
```
Expected: FAIL — Up/Down/Tab don't affect autocomplete state.

- [ ] **Step 3: Add autocomplete key routing to `updateInput` in `screen_input.go`**

The `updateInput` function handles `screenInput`. Add autocomplete routing before the default textinput update:

```go
func (m model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
    if key, ok := msg.(tea.KeyMsg); ok {
        // Route Up/Down/Tab when autocomplete is active.
        if entries := filterAutocomplete(m.commandInput.Value()); entries != nil {
            switch key.Type {
            case tea.KeyDown:
                m.autocompleteIndex++
                if m.autocompleteIndex >= len(entries) {
                    m.autocompleteIndex = len(entries) - 1
                }
                return m, nil
            case tea.KeyUp:
                m.autocompleteIndex--
                if m.autocompleteIndex < 0 {
                    m.autocompleteIndex = 0
                }
                return m, nil
            case tea.KeyTab:
                if m.autocompleteIndex < len(entries) {
                    m.commandInput.SetValue(entries[m.autocompleteIndex].Slash + " ")
                    m.commandInput.CursorEnd()
                    m.autocompleteIndex = 0
                }
                return m, nil
            case tea.KeyEsc:
                m.commandInput.SetValue("")
                m.autocompleteIndex = 0
                return m, nil
            }
        }

        switch key.Type {
        case tea.KeyEsc:
            m.state = screenMode
            return m, nil
        case tea.KeyEnter:
            prompt := strings.TrimSpace(m.commandInput.Value())
            if prompt == "" {
                prompt = strings.TrimSpace(m.textInput.Value())
            }
            m.autocompleteIndex = 0
            return m.handleShellCommand(prompt)
        }
    }

    var cmd tea.Cmd
    m.commandInput, cmd = m.commandInput.Update(msg)
    // Reset autocomplete index when the input changes.
    if filterAutocomplete(m.commandInput.Value()) == nil {
        m.autocompleteIndex = 0
    }
    return m, cmd
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./...
```
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add BubbleTeaTerminal/screen_input.go BubbleTeaTerminal/autocomplete_test.go
git commit -m "feat: wire autocomplete keyboard navigation and Tab-completion"
```

---

## Self-Review Checklist

**Spec coverage:**
- ✅ Slash command autocomplete drawer (Tasks 7–9)
- ✅ Terminal shifts up when autocomplete opens (Task 8 — drawer appended after prompt, bottom-anchored panel handles layout)
- ✅ User prompt echoed in feed (Task 1)
- ✅ Styled error blocks (Task 2)
- ✅ Operation elapsed time + stage label (Task 3)
- ✅ Keyboard hint strip (Task 4)
- ✅ Top status bar (Task 5)
- ✅ Esc cancels in-flight forge (Task 6)

**Placeholder scan:** None found. All steps include code.

**Type consistency:**
- `autocompleteEntry` defined in Task 7, used in Tasks 8–9 ✅
- `autocompleteIndex` added to model in Task 8, used in Task 9 ✅
- `filterAutocomplete` defined in Task 7, used in Tasks 8–9 ✅
- `renderAutocompleteDrawer` defined in Task 8, used in Task 8 ✅
- `fmtElapsed` defined and used in Task 3 ✅
- `sessionEventKindUser` defined in Task 1, used in Task 1 ✅
