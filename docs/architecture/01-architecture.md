# Architecture Map

## System Shape
`BubbleTeaTerminal` writes file-backed requests into tModLoader `ModSources`. `agents/orchestrator.py` watches those files, runs generation or workshop actions, mirrors status back into `ModSources`, and uses `ForgeConnector` for live injection and runtime telemetry inside tModLoader.

## Main Layers

### TUI / workshop surface
- Entry: `BubbleTeaTerminal/main.go`
- Forge request path:
  - `screen_forge.go`
  - `ipc.WriteUserRequest(...)`
  - `user_request.json`
- Workshop path:
  - `screen_staging.go`
  - `ipc.WriteWorkshopRequest(...)`
  - `workshop_request.json`
- Live inject path:
  - `screen_staging.go`
  - `ipc.WriteInjectFile(...)`
  - `forge_inject.json`

The TUI owns interaction flow, variant shelf rendering, and runtime banners. It does not own generation logic.

### Python orchestrator
- Entry/daemon: `agents/orchestrator.py`
- Normal compile path:
  - `ArchitectAgent`
  - parallel `CoderAgent` + `ArtistAgent`
  - `Gatekeeper Integrator`
- Instant path:
  - `ArchitectAgent`
  - `ArtistAgent`
  - `_set_ready(..., inject_mode=True)`
- Workshop/director path:
  - reads `workshop_request.json`
  - loads/saves workshop sessions
  - materializes bounded variants
  - writes `workshop_status.json`
- Hidden audition path remains orchestrator-owned end-to-end.

### Runtime bridge
- Active code: `mod/ForgeConnector/`
- `ForgeConnectorSystem.cs`
  - watches inject/runtime files
  - parses manifests
  - allocates template slots
  - stages and loads runtime textures
  - writes connector status and runtime summary
  - handles hidden-lab request/result flow
- `ForgeManifestStore.cs`
  - in-memory slot/type/texture registries
- `ForgeLabTelemetry.cs`
  - emits runtime events for supported hidden-lab validation
- `ForgeItemGlobal.cs` / `ForgeProjectileGlobal.cs`
  - runtime behavior/presentation hooks

## Contracts

### TUI <-> orchestrator
- `agents/contracts/ipc.py::UserRequest`
  - forge request contract
- `agents/contracts/ipc.py::GenerationStatus`
  - root build/ready/error status
- `agents/contracts/workshop.py::WorkshopRequest`
  - workshop actions: `variants`, `bench`, `try`, `restore`
- `agents/contracts/workshop.py::WorkshopStatus`
  - bench snapshot, shelf variants, last action, error

### Runtime-facing files
- `forge_inject.json`
  - TUI -> `ForgeConnector`
- `forge_connector_status.json`
  - `ForgeConnector` -> TUI
- `forge_runtime_summary.json`
  - `ForgeConnector` -> TUI runtime banner state
- `forge_connector_alive.json`
  - bridge heartbeat

### Hidden runtime gate
- `agents/core/runtime_contracts.py::HiddenLabRequest`
- `agents/core/runtime_lab_contract.py::RuntimeLabResult`

## File IPC Surface

### Primary forge flow
- `user_request.json`
- `generation_status.json`
- `orchestrator_alive.json`
- `.forge_orchestrator.lock`

### Workshop flow
- `workshop_request.json`
- `workshop_status.json`
- `.forge_workshop_sessions/`

### Live inject / runtime flow
- `forge_inject.json`
- `forge_connector_status.json`
- `forge_runtime_summary.json`
- `forge_connector_alive.json`
- `forge_last_inject.json`
- `forge_last_inject_debug.json`
- `ForgeConnectorInjectedAssets/`

### Hidden audition flow
- `forge_lab_hidden_request.json`
- `forge_lab_hidden_result.json`
- `forge_lab_runtime_events.jsonl`

## Execution Paths

### 1. Normal compile path
1. TUI writes `user_request.json`
2. Orchestrator validates request and writes `generation_status.json`
3. Architect produces manifest
4. Pixelsmith produces item/projectile art
5. Forge Master produces generated mod code
6. Gatekeeper stages/builds/repairs under `ForgeGeneratedMod`
7. Ready state is mirrored back to root `generation_status.json`
8. TUI reaches staging
9. User accepts
10. TUI writes `forge_inject.json`
11. `ForgeConnector` injects the item into the live runtime

### 2. Instant path
1. Same request ingress
2. Orchestrator runs Architect + Pixelsmith only
3. Orchestrator returns `ready` with preview manifest and sprite paths
4. TUI still uses `forge_inject.json` for runtime behavior
5. No Gatekeeper build occurs

### 3. Workshop / Forge Director path
1. A ready item becomes the bench
2. TUI sends `workshop_request.json`
3. Orchestrator updates workshop session state
4. `workshop_status.json` refreshes the bench and shelf
5. User can place a shelf variant on the bench or reinject the bench directly

### 4. Hidden audition path
1. Orchestrator sees `hidden_audition=true`
2. Architect generates theses/finalists
3. Pixelsmith runs hidden art audition with deterministic gates
4. Cross-consistency filters drifted candidates
5. Orchestrator emits one `HiddenLabRequest` at a time
6. `ForgeConnector` runs the bounded runtime validation path
7. Orchestrator evaluates terminal evidence and archives results
8. Winner reveal happens only after art + runtime pass

## Current Boundaries
- Workshop/director flow is a live-iteration surface, not a general-purpose IDE.
- Runtime validation is still intentionally narrow.
- `ForgeConnector` can host bounded live mechanics and presentation, but it is still a template-slot runtime rather than arbitrary generated code.

## Highest-Risk Coupling Points
1. Identity drift across manifest/package/runtime fields
2. Small runtime support matrix relative to authoring/package surface
3. File-backed status ownership across TUI, orchestrator, Gatekeeper, and `ForgeConnector`
4. Reflection-backed texture injection and runtime asset staging
5. Template-slot reuse and ephemeral runtime state
