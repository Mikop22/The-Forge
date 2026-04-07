# The Polished Forge — Design Spec

**Date:** 2026-04-02
**Status:** Draft — pending user review

## Vision

Transform The Forge from a single-item weapon generator into an interactive content playground. Users generate weapons, accessories, summons, consumables, and tools, preview them with ASCII animations in the TUI, iterate on sprites and stats before injecting, and test everything in-game.

The experience target: **30 minutes of fun** — generate a loadout, tweak it, inject it, go fight.

## What's NOT in scope

- Armor sets (head/body/legs with set bonuses) — requires cross-item linking
- Bosses and biomes — deferred to a future "Arena" expansion
- Browser-based preview — staying terminal-only
- Template pool scaling — existing 50 item / 25 projectile slots are sufficient for now

---

## 1. New Content Types

### 1.1 Accessories

**What:** Wings, shields, movement modifiers, stat-boosting accessories.

**Manifest fields:**
- `type: "Accessory"`
- `sub_type`: "Wings", "Shield", "Movement", "StatBoost"
- `stats`: `defense`, `movement_speed_bonus`, `jump_boost`, `flight_time` (wings)
- `mechanics`: `on_equip_buff` (e.g. BuffID.Swiftness), `dash` (boolean), `immunity_frames`

**Template pool:** Reuse existing `ForgeItem` templates. In `ForgeItemGlobal.SetDefaults`, when `data.Accessory == true`:
- Set `item.accessory = true`
- Set `item.defense`, `item.wingTimeMax`, etc. based on manifest
- Skip damage/useStyle/shoot configuration

**Architect prompt:** Dedicated accessory prompt. No damage stats. Focus on passive effects, movement, and defense.

### 1.2 Summon Weapons (Minion Staves)

**What:** Staves that summon a persistent minion companion.

**Manifest fields:**
- `type: "Weapon"`, `sub_type: "SummonStaff"`
- `stats`: `damage`, `knockback`, `mana_cost`, `use_time`
- `mechanics`: `minion_ai_type` ("follower", "orbiter"), `minion_slots` (default 1), `chase_speed`, `idle_offset`, `detection_range`
- `projectile_visuals`: describes the minion sprite (not a projectile — the flying companion)

**Template pool changes:**
- Summon items use existing `ForgeItem` pool with `DamageType = Summon`, `useStyle = Swing`, `noMelee = true`, `buffType` linked to a template buff
- **New: Template buff pool** — `ForgeBuff_001` through `ForgeBuff_025`. Minimal buff that keeps the minion alive (identical pattern to ExampleSimpleMinionBuff)
- Minion projectiles use existing `ForgeProjectile` pool but with AI injected

**Minion AI in ForgeProjectileGlobal:**
When `ForgeProjectileData.AiMode == "minion_follower"`:
- `SetStaticDefaults`: set `MinionTargettingFeature`, `projPet`, `MinionSacrificable`
- `SetDefaults`: set `minion = true`, `minionSlots`, `penetrate = -1`, `tileCollide = false`, `DamageType = Summon`
- `AI()`: implement the standard idle → search → chase → contact loop using manifest parameters (`chase_speed`, `idle_offset`, `detection_range`, `inertia`)

**Architect prompt:** Dedicated summon prompt. Generates mana cost, minion behavior parameters, and a minion visual description.

### 1.3 Consumables

**What:** Potions (healing, mana, buffs), thrown weapons, ammo.

**Manifest fields:**
- `type: "Consumable"`
- `sub_type`: "HealPotion", "ManaPotion", "BuffPotion", "ThrownWeapon", "Ammo"
- `stats`: `heal_life`, `heal_mana`, `buff_type`, `buff_duration`
- `mechanics`: `max_stack` (default 30 for potions, 999 for ammo), `consumable: true`

**Template pool:** Reuse `ForgeItem` templates. `ForgeItemGlobal.SetDefaults` handles:
- `item.consumable = true`, `item.maxStack`, `item.healLife`, `item.healMana`
- `item.buffType`, `item.buffTime` for buff potions
- `item.useStyle = DrinkLiquid` for potions, `Swing` for thrown
- `item.ammo = AmmoID.X` for ammo types

**Architect prompt:** Dedicated consumable prompt. Focus on healing amounts, buff selection from valid BuffIDs, stack sizes.

### 1.4 Tools with Special Effects

**What:** Grappling hooks, mounts, fishing rods.

**Feasibility note:** Mounts and grappling hooks require dedicated ModMount/ModProjectile subclasses with complex AI. For v1, limit to:
- **Grappling hooks** — use vanilla `AIStyle = 7` (hook AI) on the projectile template
- **Mounts** — defer to future version (requires ModMount template pool, mount texture sheets with multiple frames)
- **Fishing rods** — simple, just a weapon with `fishingPole` stat set

**Manifest fields:**
- `type: "Tool"`, `sub_type`: "Hook", "FishingRod"
- `stats`: hook — `hook_reach`, `hook_speed`, `hook_count`; fishing — `fishing_power`

**Architect prompt:** Dedicated tool prompt.

---

## 2. Specialized Architect Prompts

Replace the single monolithic prompt with a **prompt router**. The TUI's content-type selector determines which prompt is used.

```
User selects content type in wizard
  → "Weapon"     → weapon_prompt.py
  → "Accessory"  → accessory_prompt.py
  → "Summon"     → summon_prompt.py
  → "Consumable" → consumable_prompt.py
  → "Tool"       → tool_prompt.py
```

Each prompt:
- Contains only the fields relevant to that content type
- Has its own Pydantic output model (no unused fields)
- Includes type-specific balance tables (e.g. wing flight times per tier, potion heal amounts per tier)
- References valid Terraria IDs for its domain (BuffIDs for potions, AmmoIDs for ammo)

The Architect agent's `generate_manifest()` method accepts a `content_type` parameter and dispatches to the correct prompt chain.

---

## 3. TUI Preview & Reprompt Loop

### 3.1 Current flow
```
Prompt → Tier → Generate → [sprite + stats card] → Inject (Y/N)
```

### 3.2 New flow
```
Prompt → Content Type → Tier → Generate → [Preview Screen] → Action Menu
                                                ↑                  |
                                                |   ← Reprompt ←   |
                                                                   ↓
                                                              Inject / Discard
```

### 3.3 Preview Screen

The preview screen shows:
- **ASCII sprite** — the item/minion sprite rendered as colored Unicode block characters (▀▄█)
- **Stats card** — damage, knockback, use time, rarity, special effects (already exists)
- **Swing/use animation** — a small looping ASCII animation showing the weapon's use style:
  - Swing: sprite rotates through 4-6 frames in a swing arc
  - Shoot: sprite held out + projectile ASCII dot moving rightward
  - Thrust: sprite jabs forward and back
  - Summon: minion sprite bobs up and down near a player icon
  - Accessory: static sprite with stat overlay (no animation)
  - Consumable: sprite with "gulp" text flash

### 3.4 ASCII Sprite Rendering

Convert PNG sprite to terminal block characters:
1. Load PNG, downscale to ~16-24 columns wide (maintaining aspect ratio with half-block ▀▄ characters for 2:1 pixel density)
2. Map each pixel to nearest ANSI 256-color
3. Transparent pixels → terminal background
4. Cache the converted frames

Go library options: process in Go using `image/png` stdlib + ANSI escape codes. No external dependency needed.

### 3.5 Action Menu

After preview, the user sees:
```
[R] Reprompt sprite    [S] Tweak stats    [A] Accept & Inject    [D] Discard
```

- **Reprompt sprite**: opens a mini text input — "What should change?" — sends the original manifest + user feedback back to the Pixelsmith with the feedback appended to the art prompt. Re-renders preview.
- **Tweak stats**: opens an inline editor for key numeric stats (damage, use time, knockback). No LLM call — direct manifest edit.
- **Accept & Inject**: writes `forge_inject.json` and injects.
- **Discard**: returns to prompt screen.

---

## 4. Code Reliability

The template pool approach is inherently reliable — no free-form C# is generated for injection. All behavior comes from pre-tested template classes (`ForgeItem_XXX`, `ForgeProjectile_XXX`, `ForgeBuff_XXX`) with stats applied at runtime via `GlobalItem`/`GlobalProjectile`/`GlobalBuff`.

### What could still go wrong:
- **Invalid BuffIDs/AmmoIDs** from the Architect → validate against a whitelist in the manifest Pydantic model
- **Out-of-range stats** → already handled by tier clamping
- **Minion AI edge cases** (stuck in walls, infinite chase) → the AI template is pre-tested; only speed/range parameters change

### New validation:
- Add `VALID_BUFF_IDS` and `VALID_AMMO_IDS` sets to the Architect models
- Pydantic validators reject invalid IDs before they reach the game
- Log warnings for any clamped or rejected values

---

## 5. ForgeConnectorSystem Changes

### 5.1 New template pools needed:
- **Buff templates**: `ForgeBuff_001` through `ForgeBuff_025` — generic `ModBuff` subclasses
- Buff template registered in `ForgeManifestStore` like items/projectiles

### 5.2 ParseManifest expansion:
- Read `type` field to determine content category
- Branch parsing: Weapon → current path, Accessory → accessory path, Consumable → consumable path, etc.
- Each branch populates the relevant `ForgeItemData` fields

### 5.3 ForgeItemGlobal.SetDefaults expansion:
- Check `data.Type` to apply content-type-specific properties
- Accessory: `item.accessory = true`, skip damage/useStyle
- Consumable: `item.consumable = true`, `item.maxStack`, healing, buffs
- Summon: `item.DamageType = Summon`, `item.buffType`, `item.mana`
- Tool: `item.fishingPole` for rods, hook projectile setup

### 5.4 ForgeProjectileGlobal.AI() expansion:
- Currently no AI (projectiles fly straight)
- Add AI mode dispatch based on `ForgeProjectileData.AiMode`:
  - `"straight"` — current behavior (default)
  - `"minion_follower"` — idle → search → chase → contact damage loop
  - `"hook"` — vanilla hook AI (`Projectile.aiStyle = 7`)

---

## 6. File Structure (new/modified)

```
agents/architect/
  prompts.py              → becomes prompt router
  weapon_prompt.py        (new) — weapon-specific prompt
  accessory_prompt.py     (new) — accessory-specific prompt
  summon_prompt.py        (new) — summon-specific prompt
  consumable_prompt.py    (new) — consumable-specific prompt
  tool_prompt.py          (new) — tool-specific prompt
  models.py               → expanded with content-type models

mod/ForgeConnector/
  Content/Buffs/
    ForgeTemplateBuff.cs  (new) — 25 buff template slots
  ForgeConnectorSystem.cs → expanded ParseManifest
  ForgeItemGlobal.cs      → expanded SetDefaults
  ForgeProjectileGlobal.cs → minion AI
  ForgeManifestStore.cs   → buff pool + expanded data fields
  ForgeBuff Global.cs     (new) — GlobalBuff for template buffs

BubbleTeaTerminal/
  main.go                 → content type wizard, preview screen, action menu, ASCII renderer
```

---

## 7. TUI Wizard Flow (updated)

```
Step 1: "What do you want to forge?"
        [1] Weapon  [2] Accessory  [3] Summon  [4] Consumable  [5] Tool

Step 2: Sub-type (context-dependent)
        Weapon:     Sword / Bow / Staff / Gun / Cannon / Spear / ...
        Accessory:  Wings / Shield / Movement / StatBoost
        Summon:     Minion Staff
        Consumable: Heal Potion / Buff Potion / Thrown / Ammo
        Tool:       Hook / Fishing Rod

Step 3: Tier selection (same as now)

Step 4: Describe your item (free text prompt)

Step 5: [Generating...] → Preview Screen → Action Menu
```

---

## Success Criteria

1. User can generate and inject at least 5 content types (weapons, accessories, summons, consumables, hooks)
2. ASCII preview renders the sprite recognizably in the terminal
3. Swing/shoot animation plays in the preview
4. User can reprompt the sprite and tweak stats before injecting
5. Summon minions follow the player and attack enemies in-game
6. No game crashes from injected content (template pool guarantees this)
7. Specialized prompts produce higher quality output than the current monolithic prompt
