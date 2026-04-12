# Architecture Index

## Purpose
This folder is the canonical LLM/operator entrypoint for the repo. It replaces the older mix of plan docs and ad hoc handoff notes with a short, current architecture set.

## Canonical Docs
1. `docs/architecture/00-index.md`
   - navigation, environment, reading order
2. `docs/architecture/01-architecture.md`
   - topology, contracts, execution paths, state files, coupling
3. `docs/architecture/02-current-state.md`
   - what is actually implemented on `main`, current limits, likely next hardening work
4. `docs/architecture/03-storm-brand-live-bug.md`
   - resolved incident note for the direct-inject asset/render failure

## Minimal Environment
- Target runtime: Terraria + tModLoader `1.4.4`
- Go: `1.24+` for `BubbleTeaTerminal/`
- Python: `3.12+` for `agents/`
- Node: `18+` for `agents/pixelsmith/fal_flux2_runner.mjs`
- Python deps: `agents/requirements.txt` plus `fal-client`, `playwright`, `scikit-learn`
- Browser runtime: `playwright install chromium`
- External keys: `OPENAI_API_KEY`, `FAL_KEY`
- Optional art toggle: `FAL_IMAGE_TO_IMAGE_ENABLED=true`
- Pixelsmith local weights: `agents/pixelsmith/terraria_weights.safetensors`
- Live bridge: copy `mod/ForgeConnector/` into tModLoader `ModSources`, build, enable

## Working Assumptions
- Shared filesystem IPC through tModLoader `ModSources` is still the backbone.
- Python orchestrator owns pipeline state and workshop/director decisions.
- Go TUI is a client/workshop surface, not the orchestrator.
- `ForgeConnector` is a constrained runtime adapter for live injection, telemetry, and presentation.
- Runtime-validated combat remains intentionally bounded even though authoring/package coverage is broader.

## Repo Topology
- `BubbleTeaTerminal/`: workshop UI, request writing, status polling, variant shelf, reinject flows
- `agents/`: Python pipeline and director logic
- `agents/architect/`: prompt-to-manifest and package-first thesis/finalist generation
- `agents/pixelsmith/`: sprite generation, deterministic sprite gates, hidden art audition
- `agents/forge_master/`: manifest-to-C# generation and review
- `agents/gatekeeper/`: compile/build/repair loop for packaged mod output
- `agents/core/`: combat packages, runtime contracts, archive models, workshop session helpers
- `agents/contracts/`: IPC schemas for TUI/orchestrator/runtime files
- `mod/ForgeConnector/`: live inject bridge, slot pool, texture loading, runtime summary, telemetry
- `docs/plans/`: implementation/design history, not the primary architectural entrypoint

## Exact Reading Order For A New LLM
1. `docs/architecture/01-architecture.md`
2. `docs/architecture/02-current-state.md`
3. `agents/orchestrator.py`
4. `agents/contracts/ipc.py`
5. `agents/contracts/workshop.py`
6. `agents/core/workshop_director.py`
7. `agents/core/workshop_session.py`
8. `agents/architect/architect.py`
9. `agents/architect/models.py`
10. `agents/core/combat_packages.py`
11. `agents/core/runtime_capabilities.py`
12. `agents/core/runtime_contracts.py`
13. `agents/core/runtime_lab_contract.py`
14. `agents/pixelsmith/pixelsmith.py`
15. `agents/forge_master/forge_master.py`
16. `agents/gatekeeper/gatekeeper.py`
17. `mod/ForgeConnector/ForgeConnectorSystem.cs`
18. `mod/ForgeConnector/ForgeManifestStore.cs`
19. `mod/ForgeConnector/ForgeLabTelemetry.cs`
20. `mod/ForgeConnector/ForgeItemGlobal.cs`
21. `mod/ForgeConnector/ForgeProjectileGlobal.cs`
22. `BubbleTeaTerminal/screen_forge.go`
23. `BubbleTeaTerminal/screen_staging.go`

## Fast Verification Targets
- Python: `agents/tests/test_workshop_contracts.py`, `agents/tests/test_workshop_session.py`, `agents/tests/test_workshop_runtime_summary_source_contract.py`, `agents/tests/test_direct_inject_asset_guards.py`
- Go: `go test ./...` in `BubbleTeaTerminal/`
- Connector: `dotnet build mod/ForgeConnector/ForgeConnector.csproj -v minimal`

## Notes
- Keep this folder small and current.
- Use `docs/plans/` for speculative or implementation-specific detail, not as the first read target.
