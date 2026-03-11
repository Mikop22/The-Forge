# Arcane Forge HUD Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement an arcane-themed visual shell and light animation state for the existing terminal forge flow while preserving current controls and transitions.

**Architecture:** Extend the model with animation and layout state (`animTick`, `heat`, `revealPhase`, `termCompact`) and route all screen rendering through a new shared shell function. Keep state transitions in existing update handlers and increment animation state from spinner/tick messages to avoid custom goroutine loops.

**Tech Stack:** Go, Bubble Tea, Bubbles (`list`, `spinner`, `textinput`), Lip Gloss

---

### Task 1: Add failing tests for new behavior

**Files:**
- Modify: `model_test.go`

**Step 1: Write the failing tests**
- Add test for compact mode threshold from `tea.WindowSizeMsg`.
- Add test for forge heat progression bounds via repeated tick messages.
- Add test for staging reveal phase increment and reset on craft another.

**Step 2: Run test to verify it fails**
Run: `go test ./...`
Expected: FAIL due missing fields/behavior.

### Task 2: Implement model state and transitions

**Files:**
- Modify: `main.go`

**Step 1: Write minimal implementation**
- Add model fields for animation/layout.
- Update `WindowSizeMsg` handling to set compact mode threshold.
- Update forge tick handling to advance animation + bounded heat.
- Update staging enter and craft-another behavior for reveal phase reset.

**Step 2: Run test to verify it passes**
Run: `go test ./...`
Expected: PASS for newly added tests (environment permitting).

### Task 3: Implement visual shell and arcane styling

**Files:**
- Modify: `styles.go`
- Modify: `main.go`

**Step 1: Write minimal implementation**
- Add semantic arcane color/style tokens.
- Add shared shell rendering with ember strip and optional sigil column.
- Add per-screen copy/styling updates (forge verbs, heat meter, reveal behavior).

**Step 2: Run tests again**
Run: `go test ./...`
Expected: PASS (environment permitting).

### Task 4: Verification sweep

**Files:**
- Verify only

**Step 1: Execute verification commands**
Run:
- `go test ./...`

**Step 2: Report exact status with evidence**
- If command fails due environment, report failure details explicitly.
