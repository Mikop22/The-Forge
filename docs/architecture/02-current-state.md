# Current State

## Reality Snapshot
- The repo now has three active surfaces on `main`:
  - normal compile/build flow
  - instant inject flow
  - Forge Director workshop flow
- Direct-inject asset staging/debug hardening is merged on `main`
- Storm Brand runtime/presentation fixes are merged on `main`
- The older minimized handoff set remains useful, but this document now reflects the newer workshop/runtime state

## Implemented Now

### TUI
- Bench-first staging/workshop UI exists
- Command bar and workshop actions exist
- Runtime banner reads connector heartbeat, inject status, and runtime summary
- Variant shelf, bench swap, reinject, and restore flows are implemented

### Orchestrator
- Supports:
  - normal compile path
  - instant path
  - hidden audition path
  - workshop session/variant flow
- Workshop sessions are persisted under `.forge_workshop_sessions`
- Workshop transport uses:
  - `workshop_request.json`
  - `workshop_status.json`

### Runtime bridge
- `ForgeConnector` supports:
  - live item/projectile/buff injection
  - runtime summary output
  - direct-inject texture staging into runtime-owned files
  - last-inject payload/debug artifact preservation
  - hidden-lab telemetry

### Package/runtime presentation
- `storm_brand` is the best-proven runtime path
- bounded presentation routing exists for the live runtime slice that was validated during the Storm Brand work

## Still Bounded / Intentionally Incomplete
- Hidden runtime validation is still narrow compared with the full package surface
- Runtime capability coverage still does not imply every combat package is live-valid
- Workshop/director flow is still V1:
  - no heavy long-running agent orchestration in the UI
  - no full timeline/history UX
  - no generalized compare/review system
- Hidden-audition request extras still rely partly on raw-dict behavior rather than a fully promoted typed contract

## Practical Mental Model
- Treat the system as:
  - package-first authoring
  - file-backed live runtime injection
  - bounded runtime validation
  - workshop-style live iteration surface
- Do not treat the TUI as an IDE
- Do not treat `ForgeConnector` as a general generated-runtime host

## Known Sharp Edges

### IPC/state ownership
- Root status is still shared across multiple actors:
  - orchestrator
  - Gatekeeper
  - `ForgeConnector`
- TUI clears some files optimistically, so race/debug reasoning must account for client cleanup behavior

### Runtime support ambiguity
- Authoring/package surface is broader than live runtime validation support
- Docs and operator expectations can drift if runtime support is not explicitly fenced

### Texture/runtime presentation
- Runtime visuals depend on:
  - sprite generation quality
  - staged asset copy
  - texture assignment into template-backed runtime slots
- Manifest success and visual success are not the same thing

### Workshop durability
- Workshop V1 is real and usable, but not yet deeply hardened with richer history, compare, or background experiment management

## High-Value Next Hardening Work
1. Promote hidden-audition request fields to a fully first-class typed contract if the feature is intended to stay operator-facing
2. Tighten README entrypoints so `docs/architecture/` is the obvious first read
3. Expand or explicitly fence runtime package support matrix to reduce incorrect assumptions
4. Add a compact live smoke-test checklist for workshop/runtime changes
5. Continue runtime presentation hardening package-by-package instead of generalizing all at once

## Read Targets By Goal

### Goal: modify forge/workshop UX
- `BubbleTeaTerminal/screen_forge.go`
- `BubbleTeaTerminal/screen_staging.go`
- `BubbleTeaTerminal/internal/ipc/ipc.go`
- `agents/contracts/workshop.py`
- `agents/core/workshop_session.py`
- `agents/core/workshop_director.py`

### Goal: modify generation
- `agents/orchestrator.py`
- `agents/architect/architect.py`
- `agents/architect/models.py`
- `agents/pixelsmith/pixelsmith.py`
- `agents/forge_master/forge_master.py`
- `agents/gatekeeper/gatekeeper.py`

### Goal: modify live injection/runtime behavior
- `mod/ForgeConnector/ForgeConnectorSystem.cs`
- `mod/ForgeConnector/ForgeManifestStore.cs`
- `mod/ForgeConnector/ForgeItemGlobal.cs`
- `mod/ForgeConnector/ForgeProjectileGlobal.cs`
- `mod/ForgeConnector/ForgeLabTelemetry.cs`

### Goal: modify hidden audition/runtime gate
- `agents/core/runtime_capabilities.py`
- `agents/core/runtime_contracts.py`
- `agents/core/runtime_lab_contract.py`
- `agents/core/cross_consistency.py`
- `agents/core/weapon_lab_archive.py`

## Residual Historical Docs
- `docs/plans/` remains useful for implementation history
- it is no longer the preferred starting point for a new LLM/operator
