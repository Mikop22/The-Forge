# Storm Brand Presentation Design

## Goal
Add bounded runtime presentation polish to `storm_brand` staff projectiles so direct-injected and package-backed weapons feel more like native Terraria magic weapons without weakening the package-first pipeline.

## Problem
The current `storm_brand` loop succeeds mechanically, especially once the 3-hit `Starfury` cashout lands, but the seed projectile still feels flat. Right now the runtime mostly provides:

- a projectile sprite
- generic shoot sound
- basic travel
- stack/cashout gameplay telemetry

That makes the payoff feel good while the shot itself still feels under-authored.

## Design Choice
Use a bounded runtime presentation registry driven by existing package metadata.

We should not ask the LLM to invent low-level effect instructions such as dust IDs, sound names, animation amplitudes, or draw formulas. The existing pipeline already has the right selector:

- `mechanics.combat_package`
- `presentation.fx_profile`
- `resolved_combat.presentation_module`

The runtime should interpret those bounded identities and map them to authored effect behavior.

## Why This Fits The Current Architecture
The current system already splits responsibilities cleanly:

- Architect chooses package identity and `fx_profile`
- Pixelsmith generates static raster art
- `ForgeConnector` handles runtime behavior and presentation

That means runtime polish belongs in `ForgeConnector`, not in Pixelsmith prompts and not in new free-form manifest fields.

## Runtime Presentation Ladder

### 1. Cast
Triggered from item fire.

Add:
- muzzle flash at the staff tip / fire point
- a package-specific firing sound
- a small electric dust burst

### 2. Travel
Triggered every projectile update and draw.

Add:
- soft blue-white light emission
- dust or spark trail
- slight pulse, wobble, or spin so the bolt feels active

### 3. Hit
Triggered on NPC contact.

Add:
- compact hit burst
- a sharper hit sound or ping
- stronger visual confirmation on mark application

### 4. Marked Target State
Triggered while a target carries 1-2 stacks.

Add:
- visible mark glyph or orbit above the NPC
- brightness/scale tied to stack count
- quick reset after cashout

### 5. Cashout
Keep the current real `Starfury` follow-up as the finisher. The main work is improving the setup and stack readability before the cashout.

## Data Model Changes
Keep schema changes minimal and bounded.

Add C# runtime fields to carry already-existing manifest identity:

- `CombatPackageKey`
- `PresentationProfile`
- `PresentationModule`

These should be parsed from existing manifest fields:

- `mechanics.combat_package`
- `presentation.fx_profile`
- `resolved_combat.presentation_module`

No new open-ended effect schema should be introduced.

## Runtime Registry
Create a C# presentation profile registry that maps bounded profile identity to authored behavior.

Initial scope:
- `storm_brand`
- `celestial_shock`

The registry should answer questions such as:

- which cast sound to play
- which dust family to use
- how much projectile light to emit
- whether to pulse, wobble, or rotate
- how marked targets should be drawn

This keeps direct inject deterministic and reviewable.

## Rendering Strategy

### Item-side
Use `ForgeItemGlobal.Shoot(...)` as the fire-time entrypoint for cast flash and sound.

### Projectile-side
Use:
- `ForgeProjectileGlobal.AI(...)` for movement-time effects
- `ForgeProjectileGlobal.PreDraw(...)` for pulse/wobble/afterimage drawing
- `ForgeProjectileGlobal.OnHitNPC(...)` for hit burst and stack escalation feedback

### NPC-side
Add a new `GlobalNPC` draw hook that reads the existing stack state and renders the current mark indicator.

## Reuse Existing Stack State
Do not create a second mark-tracking system.

`ForgeLabTelemetry` already keeps per-target stack state for `storm_brand`. That state should be exposed through a read-only helper so the NPC draw hook can render the mark indicator using the same source of truth that drives cashout.

## Non-Goals
- no free-form VFX DSL
- no LLM-authored dust/sound constants
- no generated audio for live inject
- no baked trails or flashes inside projectile PNGs
- no broad generalization to all packages in the first pass

## Rollout

### Phase 1
Ship bounded runtime presentation support for `storm_brand` only.

### Phase 2
Verify live feel and no regression to:
- direct inject
- hidden-audition identity
- 3-hit starfall cashout

### Phase 3
Generalize the same runtime registry pattern to `orbit_furnace` and `frost_shatter` if needed.

## Testing Strategy

### Source-contract / routing tests
Add Python tests that assert:
- runtime data objects preserve package/profile identity
- connector parsing lowers manifest presentation data into runtime fields
- `storm_brand` routes to the presentation helpers

### Existing regression suites
Keep these passing:
- hidden-audition identity tests
- runtime lab contract tests
- projectile pipeline fix tests

### Build verification
`dotnet build mod/ForgeConnector/ForgeConnector.csproj -v minimal`

### Live verification
Direct inject a `storm_brand` staff and confirm:
- visible cast flash
- improved firing sound
- visible traveling bolt polish
- hit burst on contact
- visible stack mark on NPC
- 3-hit `Starfury` cashout still appears
