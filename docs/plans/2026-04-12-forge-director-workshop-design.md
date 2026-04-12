# Forge Director Workshop Design

## Goal
Turn the Bubble Tea TUI into a live-runtime-first Terraria item workshop that feels closer to Claude Code or Cursor in responsiveness and iteration speed, while still reading as a creative forge room rather than an IDE.

## Product Thesis
The TUI should stop feeling like a request form followed by a staging screen. It should feel like a forge room where:

- one item is always on the bench
- 2-3 nearby variants sit on a shelf
- a director turns natural-language direction into new takes
- the user gets back into Terraria quickly

The right metaphor is not "terminal IDE". It is "Terraria workshop with a smart director".

## User Intent
The primary product targets are:

- fast reload and reinject
- fast in-world tuning
- low technical overhead
- natural-language iteration
- safe experimentation with easy restore

The target loop is:

1. Forge an item.
2. Ask for a stronger or cleaner take in plain language.
3. Receive 2-3 workshop variants.
4. Put one on the bench.
5. Reinject and test in Terraria.
6. Give quick feedback.
7. Repeat without ever feeling like a code editor opened.

## Research Inputs

### Cursor / Claude Code patterns worth borrowing
- command-first interaction
- persistent session memory
- resumable state
- asynchronous work where useful
- clear status surfaces
- safe review / restore flows

Relevant references:
- Cursor overview: https://docs.cursor.com/chat/overview
- Cursor background agents: https://docs.cursor.com/en/background-agents
- Claude Code memory: https://docs.anthropic.com/en/docs/claude-code/memory
- Claude Code slash commands: https://docs.anthropic.com/en/docs/claude-code/slash-commands
- Claude Code hooks: https://docs.anthropic.com/en/docs/claude-code/hooks
- Claude Code sub-agents: https://docs.anthropic.com/en/docs/claude-code/sub-agents

### Terraria / tModLoader constraints
- The system is file-backed through `ModSources`.
- The Go TUI is a client, not the orchestrator.
- Live runtime state is owned by `ForgeConnector`.
- Reload and runtime fragility are normal constraints of the environment.

Relevant references:
- [docs/llm-handoff/01-architecture.md](/Users/user/Desktop/the-forge/docs/llm-handoff/01-architecture.md)
- [docs/llm-handoff/02-current-state.md](/Users/user/Desktop/the-forge/docs/llm-handoff/02-current-state.md)
- tModLoader development workflow: https://github-wiki-see.page/m/tModLoader/tModLoader/wiki/Developing-with-Visual-Studio-Code

## Product Shape
The workshop should have three visible layers.

### 1. The Bench
One active item sits at the center.

The bench shows:
- active item name and sprite
- current live/runtime state
- dominant next action
- current feel notes

The bench is the only accepted state. The user should always know which version is:
- on the bench
- live in Terraria
- safe to restore

### 2. The Shelf
The shelf holds 2-3 nearby variants created from a natural-language direction.

Each shelf variant is a real materialized preview snapshot, not a fake suggestion card. In V1, a shelf variant means:
- a stable `variant_id`
- a preview manifest snapshot
- preview sprite / projectile paths
- a short label
- a short rationale
- a short summary of what changed

Putting a variant on the bench copies that variant snapshot into the active bench state. Reinjecting always uses the current bench snapshot.

### 3. The Director
The director is the collaborative runtime-facing layer.

The user can say:
- "make the projectile feel heavier"
- "keep the cashout, make the cast more dramatic"
- "less noisy, more readable"
- "keep this, but make it feel more Terraria"

The director should:
- translate that into bounded variant requests
- remember what the user liked and disliked about the item family
- summarize differences in plain language
- recommend the next few experiments

## What It Must Not Be
- not a source editor
- not a file browser
- not a raw manifest viewer by default
- not a shell console
- not a prompt engineering surface

## UX Principles

### Live-first
Every important action should shorten the loop back to holding the item in Terraria.

### Non-technical language
The UI should say:
- `Try Variant`
- `Put On Bench`
- `Restore Favorite`
- `Reforge`

It should not say:
- `dispatch job`
- `apply diff`
- `inspect manifest`

### Safe experimentation
The user should always be able to restore:
- baseline
- last live version
- favorite version
- last accepted bench version

### One creative thread at a time
The workshop should focus on one active item family. It should avoid turning into a project dashboard.

### Memory without clutter
The director should remember item-local context, but that memory should surface as compact notes rather than logs.

## Architecture
Keep the current system backbone.

- Go TUI stays the workshop surface.
- Python orchestrator stays the director brain.
- `ForgeConnector` stays the live runtime bridge.

The workshop is a new state layer on top of the existing forge flow, not a replacement for the whole repo architecture.

## Explicit Transport Path
The current system already has one transport path:

- TUI writes `user_request.json`
- orchestrator writes `generation_status.json`
- TUI moves from forge to staging
- TUI writes `forge_inject.json`
- `ForgeConnector` writes bridge status files

The workshop must extend that path rather than inventing a disconnected second app.

### Initial forge path remains unchanged
The first item still enters through:
- `user_request.json`
- `generation_status.json`

This keeps the current mode / wizard / prompt flow intact for V1.

### Workshop path begins only after the first ready item
Once the current staging screen has a ready item, it becomes the workshop shell.

At that point:
- the command bar writes `workshop_request.json`
- orchestrator watches `workshop_request.json` alongside `user_request.json`
- orchestrator writes `workshop_status.json`
- the TUI polls `workshop_status.json` while still reading live runtime files from `ForgeConnector`

This keeps workshop iteration separate from the initial forge request while still using the current request loop to create the first bench item.

## Workshop State Model

### BenchState
- current active item snapshot
- live version snapshot
- baseline snapshot
- favorite snapshot
- current runtime status

### VariantShelf
- 2-3 candidate variants
- short explanation for each
- current selection
- readiness for reinject

### DirectorMemory
- item-local notes and preferences
- accepted / rejected directions
- stable constraints such as:
  - "keep the cashout"
  - "projectile still too flat"
  - "cast flash is good"

### WorkshopSession
- ties bench, shelf, memory, history, and runtime feedback together
- resumable when reopening the TUI

## Runtime Summary Source Of Truth
The runtime banner should not rely on vague inference.

V1 should use explicit data sources:
- `forge_connector_alive.json` for bridge heartbeat
- `forge_connector_status.json` for last inject result
- new `forge_runtime_summary.json` for stable live-session facts

`forge_runtime_summary.json` should be owned by `ForgeConnector` and expose only workshop-friendly facts such as:
- `bridge_alive`
- `world_loaded`
- `live_item_name`
- `last_inject_status`
- `last_runtime_note`
- `updated_at`

That gives the TUI a clean banner source instead of forcing it to synthesize runtime state from partial files.

## Terminal UX Direction

### Command bar
The TUI should gain a bottom command bar as the main interaction surface.

It should support:
- natural language
- slash commands
- lightweight suggestion chips

Examples:
- `make the projectile feel heavier`
- `/variants heavier projectile`
- `/bench 2`
- `/try`
- `/restore baseline`
- `/favorite`
- `/history`
- `/notes`

### Slash command philosophy
Slash commands should feel like workshop verbs, not shell verbs.

Good:
- `/forge`
- `/variants`
- `/bench`
- `/try`
- `/restore`
- `/favorite`
- `/history`

Bad:
- `/exec`
- `/diff`
- `/tail-log`

### Runtime banner
The always-visible runtime banner should show:
- Terraria world loaded / not loaded
- bridge online / offline
- last inject result
- current live item
- last runtime note

In V1, these facts must come from the files above, not from UI inference.

## Feature Ladder

### Tier 1: Foundation
Low complexity.

- workshop shell on top of current staging screen
- command bar
- runtime banner
- bench state
- restore baseline / last live
- explicit workshop IPC

### Tier 2: Workshop Core
Medium complexity. Recommended next step after the foundation holds.

- variant shelf
- natural-language variant generation
- benching and reinjecting shelf variants
- simple item-local director memory

### Tier 3: Expansion
Higher complexity.

- feedback memory loop
- checkpoints and favorites
- plain-language compare summaries
- background variant preparation
- workshop timeline
- stronger director recommendations

## Recommended Rollout

### Phase 1
Build the transport and foundation:
- workshop request/status files
- bench shell
- runtime summary file
- command bar
- restore baseline / last live

### Phase 2
Build one narrow shelf flow:
- 2-3 variants from a natural-language ask
- put variant on bench
- reinject bench variant

### Phase 3
Add memory and compare features:
- feedback notes
- favorites / checkpoints
- compare summaries

### Phase 4
Add bigger agentic features:
- background variant preparation
- workshop timeline
- stronger director recommendations

## Risks

### 1. Over-technical drift
The TUI can drift into debug-console language. The main surface must stay workshop-oriented.

### 2. State sprawl
Bench, shelf, live runtime, and session memory can drift out of sync if the workshop state is layered on top of the current linear flow without a single source of truth.

### 3. Runtime trust
If reinject or reload signals are ambiguous, the workshop will feel dishonest. The runtime banner and restore paths are necessary trust mechanisms.

### 4. Variant noise
If the director creates variants that are too similar, the shelf becomes clutter without insight. The variants must be clearly differentiated.

### 5. Transport mismatch
If `workshop_request.json` and `workshop_status.json` are added without watcher and polling support, the feature becomes a design fiction instead of a buildable system.

## Success Criteria
- The user can iterate on one live item without feeling like they are using an IDE.
- The TUI can turn a natural-language direction into 2-3 meaningful variants.
- Reinjecting and restoring versions feels fast and safe.
- The runtime banner reflects explicit live-session facts instead of guesswork.
- The workshop remembers item-local preferences without exposing technical internals on the main surface.
