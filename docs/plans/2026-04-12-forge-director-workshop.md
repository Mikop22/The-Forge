# Forge Director Workshop Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Ship a V1 Forge Director workshop foundation that makes live Terraria item iteration faster and clearer without turning the TUI into an IDE.

**Architecture:** Keep the current initial forge path (`user_request.json` -> `generation_status.json`) and layer a workshop loop on top of the ready/staging state. Add explicit workshop request/status files, a runtime summary file from `ForgeConnector`, a bench-first workshop shell in the TUI, and one narrow variant shelf flow.

**Tech Stack:** Go Bubble Tea TUI, Python orchestrator and contracts, file-based IPC in `ModSources`, C# `ForgeConnector`, pytest, Go tests, live tModLoader runtime verification

---

### Scope Of This Plan

This plan intentionally stops at a smaller V1.

V1 includes:
- explicit workshop IPC wiring
- bench-first workshop shell
- runtime banner with real sources of truth
- one narrow variant shelf flow
- bench reinject and restore baseline / last live

V1 does **not** include:
- rich feedback memory loop
- checkpoints / favorites beyond baseline and last live
- compare summaries
- background variant preparation
- workshop timeline

Those are follow-on phases after the foundation proves stable.

### Variant Definition For V1

For this plan, a `variant` is a materialized preview snapshot with:
- `variant_id`
- manifest snapshot
- item sprite path
- projectile sprite path
- short label
- short rationale
- short change summary

Putting a variant on the bench copies that snapshot into `BenchState`.
Reinjecting always uses the current bench snapshot.

### Transport Definition For V1

Initial forge path remains unchanged:
- TUI writes `user_request.json`
- orchestrator writes `generation_status.json`

Workshop path begins only after the first ready item:
- TUI writes `workshop_request.json`
- orchestrator writes `workshop_status.json`
- `ForgeConnector` writes `forge_runtime_summary.json`

The staging screen becomes the workshop shell instead of spawning a new top-level app surface.

---

### Task 1: Add failing workshop IPC contract tests

**Files:**
- Create: `agents/contracts/workshop.py`
- Create: `agents/tests/test_workshop_contracts.py`
- Read: `agents/contracts/ipc.py`

**Step 1: Write the failing tests**

```python
from contracts.workshop import WorkshopRequest, WorkshopStatus, RuntimeSummary


def test_workshop_request_supports_variants_action() -> None:
    req = WorkshopRequest.model_validate(
        {
            "action": "variants",
            "session_id": "sess-1",
            "bench_item_id": "storm-brand",
            "directive": "make the projectile feel heavier",
        }
    )

    assert req.action == "variants"
    assert req.directive == "make the projectile feel heavier"


def test_workshop_status_carries_bench_and_shelf() -> None:
    status = WorkshopStatus.model_validate(
        {
            "session_id": "sess-1",
            "bench": {"item_id": "storm-brand", "label": "Storm Brand"},
            "shelf": [{"variant_id": "v1", "label": "Heavier Shot"}],
        }
    )

    assert status.bench.item_id == "storm-brand"
    assert status.shelf[0].variant_id == "v1"


def test_runtime_summary_exposes_live_banner_fields() -> None:
    summary = RuntimeSummary.model_validate(
        {
            "bridge_alive": True,
            "world_loaded": True,
            "live_item_name": "Storm Brand",
            "last_inject_status": "item_injected",
            "last_runtime_note": "Ready on bench",
        }
    )

    assert summary.bridge_alive is True
    assert summary.live_item_name == "Storm Brand"
```

**Step 2: Run test to verify it fails**

Run: `pytest -q agents/tests/test_workshop_contracts.py`

Expected: FAIL because the workshop models do not exist yet.

**Step 3: Implement bounded contract models**

Add:
- `WorkshopRequest`
- `WorkshopStatus`
- `BenchState`
- `ShelfVariant`
- `RuntimeSummary`

Keep them workshop-oriented and small.

**Step 4: Re-run test**

Run: `pytest -q agents/tests/test_workshop_contracts.py`

Expected: PASS

**Step 5: Commit**

```bash
git add agents/contracts/workshop.py agents/tests/test_workshop_contracts.py
git commit -m "feat: add workshop ipc contracts"
```

### Task 2: Add orchestrator transport wiring for workshop requests and workshop status

**Files:**
- Modify: `agents/orchestrator.py`
- Create: `agents/core/workshop_session.py`
- Create: `agents/tests/test_workshop_session.py`

**Step 1: Write the failing test**

```python
from core.workshop_session import WorkshopSessionStore


def test_session_store_round_trips_bench_and_shelf(tmp_path) -> None:
    store = WorkshopSessionStore(tmp_path)
    store.save(
        {
            "session_id": "sess-1",
            "bench": {"item_id": "storm-brand", "label": "Storm Brand"},
            "shelf": [{"variant_id": "v1", "label": "Heavier Shot"}],
        }
    )

    loaded = store.load("sess-1")
    assert loaded["bench"]["item_id"] == "storm-brand"
    assert loaded["shelf"][0]["variant_id"] == "v1"
```

**Step 2: Run test to verify it fails**

Run: `pytest -q agents/tests/test_workshop_session.py`

Expected: FAIL because the session store does not exist yet.

**Step 3: Add explicit workshop transport paths**

In `agents/orchestrator.py`, define:
- `WORKSHOP_REQUEST_FILE = _MOD_SOURCES / "workshop_request.json"`
- `WORKSHOP_STATUS_FILE = _MOD_SOURCES / "workshop_status.json"`

Extend the watchdog handler so:
- `user_request.json` still triggers the initial forge path
- `workshop_request.json` triggers workshop actions only after a ready bench item exists

Do not replace the initial forge path.

**Step 4: Add minimal session persistence**

Add a session store that:
- persists active bench snapshot
- persists shelf variants
- persists active session id

**Step 5: Write `workshop_status.json`**

Write the active bench and shelf state whenever workshop actions complete.

**Step 6: Re-run tests**

Run: `pytest -q agents/tests/test_workshop_contracts.py agents/tests/test_workshop_session.py`

Expected: PASS

**Step 7: Commit**

```bash
git add agents/orchestrator.py agents/core/workshop_session.py agents/tests/test_workshop_session.py
git commit -m "feat: wire workshop request and status loop"
```

### Task 3: Add explicit runtime summary output from ForgeConnector

**Files:**
- Modify: `mod/ForgeConnector/ForgeConnectorSystem.cs`
- Create: `agents/tests/test_workshop_runtime_summary_source_contract.py`

**Step 1: Write the failing source-contract test**

```python
from pathlib import Path


def test_connector_writes_runtime_summary_file() -> None:
    source = Path("mod/ForgeConnector/ForgeConnectorSystem.cs").read_text(encoding="utf-8")
    assert "forge_runtime_summary.json" in source
    assert "WriteRuntimeSummary(" in source
    assert "live_item_name" in source
    assert "last_runtime_note" in source
```

**Step 2: Run test to verify it fails**

Run: `pytest -q agents/tests/test_workshop_runtime_summary_source_contract.py`

Expected: FAIL because the runtime summary file does not exist yet.

**Step 3: Implement minimal runtime summary writing**

`ForgeConnector` should write `forge_runtime_summary.json` with:
- `bridge_alive`
- `world_loaded`
- `live_item_name`
- `last_inject_status`
- `last_runtime_note`
- `updated_at`

V1 notes can stay simple and deterministic.

**Step 4: Re-run test**

Run: `pytest -q agents/tests/test_workshop_runtime_summary_source_contract.py`

Expected: PASS

**Step 5: Build connector**

Run: `dotnet build mod/ForgeConnector/ForgeConnector.csproj -v minimal`

Expected: `Build succeeded.`

**Step 6: Commit**

```bash
git add mod/ForgeConnector/ForgeConnectorSystem.cs agents/tests/test_workshop_runtime_summary_source_contract.py
git commit -m "feat: add forge runtime summary output"
```

### Task 4: Refactor the staging screen into a bench-first workshop shell

**Files:**
- Modify: `BubbleTeaTerminal/model.go`
- Modify: `BubbleTeaTerminal/screen_staging.go`
- Create: `BubbleTeaTerminal/workshop_state.go`
- Create: `BubbleTeaTerminal/workshop_state_test.go`

**Step 1: Write the failing state test**

```go
func TestWorkshopStateStartsWithBenchAndEmptyShelf(t *testing.T) {
	ws := newWorkshopState()

	if ws.Bench.ItemID != "" {
		t.Fatalf("bench item = %q, want empty", ws.Bench.ItemID)
	}
	if len(ws.Shelf) != 0 {
		t.Fatalf("shelf len = %d, want 0", len(ws.Shelf))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./BubbleTeaTerminal/...`

Expected: FAIL because the workshop state layer does not exist yet.

**Step 3: Add workshop shell state**

Introduce a bounded UI state struct for:
- bench
- shelf
- runtime banner
- active session id

The staging screen should render this workshop shell after the first item is ready.

**Step 4: Re-run test**

Run: `go test ./BubbleTeaTerminal/...`

Expected: PASS

**Step 5: Commit**

```bash
git add BubbleTeaTerminal/model.go BubbleTeaTerminal/screen_staging.go BubbleTeaTerminal/workshop_state.go BubbleTeaTerminal/workshop_state_test.go
git commit -m "feat: add bench-first workshop shell"
```

### Task 5: Add command bar and slash command parsing for workshop actions

**Files:**
- Create: `BubbleTeaTerminal/command_bar.go`
- Create: `BubbleTeaTerminal/command_bar_test.go`
- Modify: `BubbleTeaTerminal/model.go`
- Modify: `BubbleTeaTerminal/screen_staging.go`

**Step 1: Write the failing parser test**

```go
func TestParseWorkshopCommandVariants(t *testing.T) {
	cmd := parseWorkshopCommand("/variants make the projectile feel heavier")
	if cmd.Name != "variants" {
		t.Fatalf("name = %q, want variants", cmd.Name)
	}
	if cmd.Arg != "make the projectile feel heavier" {
		t.Fatalf("arg = %q", cmd.Arg)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./BubbleTeaTerminal/...`

Expected: FAIL because the command parser does not exist yet.

**Step 3: Implement a small V1 command surface**

Support:
- freeform direction text
- `/variants`
- `/bench`
- `/try`
- `/restore`

Do not add the full future command set yet.

**Step 4: Wire the command bar to `workshop_request.json`**

The command bar must write real workshop requests, not just local UI actions.

**Step 5: Re-run tests**

Run: `go test ./BubbleTeaTerminal/...`

Expected: PASS

**Step 6: Commit**

```bash
git add BubbleTeaTerminal/command_bar.go BubbleTeaTerminal/command_bar_test.go BubbleTeaTerminal/model.go BubbleTeaTerminal/screen_staging.go
git commit -m "feat: add workshop command bar"
```

### Task 6: Add a real runtime banner sourced from connector files

**Files:**
- Modify: `BubbleTeaTerminal/screen_staging.go`
- Modify: `BubbleTeaTerminal/main_test.go`
- Modify: Go-side file reading helpers that currently read connector status / heartbeat

**Step 1: Write the failing view test**

Add a test that expects the workshop screen to show:
- bridge state
- world loaded state
- live item name
- last inject result

**Step 2: Run test to verify it fails**

Run: `go test ./BubbleTeaTerminal/...`

Expected: FAIL because the current screen only shows bridge heartbeat plus inject result.

**Step 3: Render the runtime banner from explicit files**

Use:
- `forge_connector_alive.json`
- `forge_connector_status.json`
- `forge_runtime_summary.json`

Do not infer `Terraria Connected` indirectly.

**Step 4: Re-run tests**

Run: `go test ./BubbleTeaTerminal/...`

Expected: PASS

**Step 5: Commit**

```bash
git add BubbleTeaTerminal/screen_staging.go BubbleTeaTerminal/main_test.go
git commit -m "feat: add runtime banner with explicit summary source"
```

### Task 7: Add one narrow variant shelf flow

**Files:**
- Create: `agents/core/workshop_director.py`
- Create: `agents/tests/test_workshop_director.py`
- Modify: `agents/orchestrator.py`
- Modify: `BubbleTeaTerminal/screen_staging.go`
- Modify: `BubbleTeaTerminal/model.go`

**Step 1: Write the failing director test**

```python
from core.workshop_director import build_variants


def test_build_variants_returns_two_or_three_materialized_variants() -> None:
    variants = build_variants(
        bench_manifest={"item_name": "Storm Brand"},
        directive="make the projectile feel heavier",
    )

    assert 2 <= len(variants) <= 3
    assert all("variant_id" in variant for variant in variants)
    assert all("label" in variant for variant in variants)
    assert all("manifest" in variant for variant in variants)
```

**Step 2: Run test to verify it fails**

Run: `pytest -q agents/tests/test_workshop_director.py`

Expected: FAIL because the director variant builder does not exist yet.

**Step 3: Implement a bounded V1 variant builder**

For V1, a variant should be one of:
- stronger impact
- cleaner read
- bigger spectacle

Each variant should be materialized as a preview snapshot, not just a label.

The orchestrator should:
- read current bench snapshot
- build 2-3 bounded variant requests
- generate preview-ready variant snapshots
- write them into `workshop_status.json`

**Step 4: Add benching and reinject actions**

Support:
- `/bench 1`
- `/bench 2`
- `/bench 3`
- `/try`

`/try` should reinject the current bench snapshot through the existing `forge_inject.json` path.

**Step 5: Re-run tests**

Run: `pytest -q agents/tests/test_workshop_contracts.py agents/tests/test_workshop_session.py agents/tests/test_workshop_director.py`

Run: `go test ./BubbleTeaTerminal/...`

Expected: PASS

**Step 6: Commit**

```bash
git add agents/core/workshop_director.py agents/tests/test_workshop_director.py agents/orchestrator.py BubbleTeaTerminal/screen_staging.go BubbleTeaTerminal/model.go
git commit -m "feat: add v1 variant shelf flow"
```

### Task 8: Add restore baseline and restore last live

**Files:**
- Modify: `agents/core/workshop_session.py`
- Create: `agents/tests/test_workshop_restore.py`
- Modify: `BubbleTeaTerminal/screen_staging.go`

**Step 1: Write the failing restore test**

```python
from core.workshop_session import WorkshopSessionStore


def test_restore_targets_round_trip(tmp_path) -> None:
    store = WorkshopSessionStore(tmp_path)
    store.save(
        {
            "session_id": "sess-1",
            "bench": {"item_id": "current"},
            "baseline": {"item_id": "baseline"},
            "last_live": {"item_id": "live"},
        }
    )

    assert store.load("sess-1")["baseline"]["item_id"] == "baseline"
    assert store.load("sess-1")["last_live"]["item_id"] == "live"
```

**Step 2: Run test to verify it fails**

Run: `pytest -q agents/tests/test_workshop_restore.py`

Expected: FAIL because restore targets do not exist yet.

**Step 3: Implement restore targets**

Support:
- `/restore baseline`
- `/restore live`

Do not add favorites or broader checkpoint systems in V1.

**Step 4: Re-run tests**

Run: `pytest -q agents/tests/test_workshop_restore.py agents/tests/test_workshop_session.py`

Expected: PASS

**Step 5: Commit**

```bash
git add agents/core/workshop_session.py agents/tests/test_workshop_restore.py BubbleTeaTerminal/screen_staging.go
git commit -m "feat: add v1 restore targets"
```

### Task 9: Run focused cross-stack verification and one concrete live workshop slice

**Files:**
- Test: `agents/tests/test_workshop_contracts.py`
- Test: `agents/tests/test_workshop_session.py`
- Test: `agents/tests/test_workshop_runtime_summary_source_contract.py`
- Test: `agents/tests/test_workshop_director.py`
- Test: `agents/tests/test_workshop_restore.py`
- Test: `BubbleTeaTerminal/main_test.go`
- Verify live: TUI + tModLoader runtime

**Step 1: Run the Python workshop regression slice**

Run:

```bash
pytest -q \
  agents/tests/test_workshop_contracts.py \
  agents/tests/test_workshop_session.py \
  agents/tests/test_workshop_runtime_summary_source_contract.py \
  agents/tests/test_workshop_director.py \
  agents/tests/test_workshop_restore.py
```

Expected: PASS

**Step 2: Run the Go test suite**

Run:

```bash
go test ./BubbleTeaTerminal/...
```

Expected: PASS

**Step 3: Run one concrete live verification slice**

Manual acceptance criteria:
- forge one initial item through the existing flow
- staging screen becomes the workshop shell
- runtime banner shows explicit live fields from connector files
- `/variants make the projectile feel heavier` creates 2-3 shelf variants
- `/bench 2` changes the bench item snapshot
- `/try` reinjects the current bench and `forge_connector_status.json` reaches `item_injected`
- `/restore baseline` restores the initial bench snapshot
- `/restore live` restores the last live snapshot

If any of those fail, do not claim V1 complete.

**Step 4: Commit**

```bash
git add .
git commit -m "feat: add forge director workshop v1 foundation"
```

### Follow-On Phase After V1 Holds

Do not start these until Tasks 1-9 are stable:
- feedback memory loop
- favorites and checkpoints
- compare summaries
- background variant preparation
- workshop timeline
- stronger director recommendations
