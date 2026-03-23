# The Forge — Deep Technical Reference

> A complete technical breakdown of every subsystem, data contract, IPC mechanism,
> external API, and error-handling strategy in The Forge.

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Technology Stack](#2-technology-stack)
3. [Architecture & Pipeline DAG](#3-architecture--pipeline-dag)
4. [IPC: The JSON File Handshake](#4-ipc-the-json-file-handshake)
5. [Go TUI — BubbleTeaTerminal](#5-go-tui--bubbleteeterminal)
6. [Python Orchestrator](#6-python-orchestrator)
7. [Architect Agent](#7-architect-agent)
8. [Forge Master Agent](#8-forge-master-agent)
9. [Pixelsmith Agent](#9-pixelsmith-agent)
10. [Gatekeeper](#10-gatekeeper)
11. [ForgeConnector (tModLoader Mod)](#11-forgeconnector-tmodloader-mod)
12. [Data Contracts & Schemas](#12-data-contracts--schemas)
13. [Error Handling & Retry Strategies](#13-error-handling--retry-strategies)
14. [External Services & APIs](#14-external-services--apis)
15. [Environment Variables](#15-environment-variables)
16. [File I/O Map](#16-file-io-map)

---

## 1. System Overview

The Forge is an AI-powered Terraria item generator that takes a natural-language weapon
description and produces a fully compiled, in-game-ready tModLoader mod — including C#
source code, a pixel art sprite, and localization files — entirely from the terminal.

The system is split into four physical layers that communicate through a shared filesystem
directory (the tModLoader `ModSources/` folder):

```
┌───────────────────────────────────────────────────────┐
│  Layer 1: Go TUI  (BubbleTeaTerminal/)                │
│  User-facing terminal application                     │
├───────────────────────────────────────────────────────┤
│  Layer 2: Python Orchestrator  (agents/)              │
│  Async daemon that drives the AI pipeline             │
├───────────────────────────────────────────────────────┤
│  Layer 3: tModLoader  (external)                      │
│  Headless C# build target                             │
├───────────────────────────────────────────────────────┤
│  Layer 4: ForgeConnector  (mod/)                      │
│  In-game bridge mod that hot-reloads new items        │
└───────────────────────────────────────────────────────┘
```

All inter-layer communication happens by reading and writing JSON files inside the
`ModSources/` directory — no sockets, no HTTP, no shared memory.

---

## 2. Technology Stack

| Layer | Language | Key Libraries |
|-------|----------|---------------|
| TUI | Go 1.24+ | [BubbleTea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Orchestrator | Python 3.12+ | asyncio, watchdog |
| Architect | Python 3.12+ | LangChain (LCEL), OpenAI, Pydantic v2, Playwright |
| Forge Master | Python 3.12+ | LangChain (LCEL), OpenAI, Pydantic v2 |
| Pixelsmith | Python 3.12+ | fal-ai, Pillow, rembg (U²-Net), scikit-learn |
| Pixelsmith bridge | Node.js 18+ | @fal-ai/client |
| Gatekeeper | Python 3.12+ | subprocess (tModLoader CLI) |
| ForgeConnector | C# (.NET 6) | tModLoader 1.4.4 API |

---

## 3. Architecture & Pipeline DAG

```
┌───────────────────┐
│  Go TUI           │
│  (BubbleTea)      │
│                   │
│  · prompt entry   │
│  · wizard steps   │
│  · heat bar anim  │
│  · staging screen │
└────────┬──────────┘
         │  writes user_request.json
         ▼
┌──────────────────────────────────────┐
│  orchestrator.py (asyncio daemon)    │
│  FileSystemWatcher on ModSources/    │
└──┬──────────────────────────────────┘
   │
   │  (5%)  _set_building()
   ▼
┌─────────────────────────────────────────────┐
│  Architect Agent  [sequential, 1 LLM call]  │
│  · tier-based stat clamping (Pydantic)      │
│  · keyword-based crafting resolution        │
│  · optional reference image search (Bing)  │
│  Output: ItemManifest dict                  │
└───────────────┬─────────────────────────────┘
                │  (40%)  asyncio.gather
       ┌────────┴────────────┐
       ▼                     ▼
┌──────────────┐   ┌───────────────────────┐
│ Forge Master │   │ Pixelsmith            │
│ [parallel]   │   │ [parallel]            │
│              │   │                       │
│ gpt LLM      │   │ FLUX.2 Klein LoRA     │
│ → C# code    │   │ → 512×512 PNG         │
│ → HJSON      │   │ → background removal  │
│              │   │ → downscale           │
└──────┬───────┘   └───────────┬───────────┘
       └────────┬──────────────┘
                │  (80%)
                ▼
┌─────────────────────────────────────────────┐
│  Gatekeeper  [sequential]                   │
│  · stage .cs / .png / .hjson to ModSources  │
│  · tModLoader headless build                │
│  · self-heal loop (up to 3 retries)         │
└──────────────────┬──────────────────────────┘
                   │  (100%)  _set_ready()
                   ▼
         generation_status.json
              {"status":"ready"}
                   │
                   │  Go TUI polls every 2 s
                   ▼
         User presses ENTER → command_trigger.json
                   │
                   ▼
┌─────────────────────────────────────────────┐
│  ForgeConnector (Terraria game process)     │
│  FileSystemWatcher → Interlocked signal     │
│  → PostUpdateEverything() main-thread hook  │
│  → TriggerReload() via reflection           │
└─────────────────────────────────────────────┘
```

**Key architectural invariants:**
- Agents exchange data exclusively via `dict` (`Pydantic.model_dump()` on send,
  `model_validate()` on receive). No shared objects.
- Steps are sequential at the pipeline level; only Forge Master and Pixelsmith run in
  parallel (via `asyncio.gather` wrapping `ThreadPoolExecutor.run_in_executor`).
- Every unhandled exception anywhere in the pipeline is caught by the orchestrator and
  written to `generation_status.json` as `{"status":"error"}`.

---

## 4. IPC: The JSON File Handshake

All inter-process communication uses plain JSON files written to and read from the
tModLoader `ModSources/` directory. There are four distinct files:

### 4.1 `user_request.json` — TUI → Orchestrator

Written by the Go TUI when the user submits a request. The orchestrator's `watchdog`
file watcher fires `on_modified`, which reads this file to start the pipeline.

```json
{
  "prompt": "A frost katana that shoots ice shards",
  "tier": "Tier1_Starter",
  "crafting_station": "Workbench"
}
```

`crafting_station` is only present when the user chose Manual Override in the wizard.
It is omitted in Auto mode so the Architect can decide.

### 4.2 `generation_status.json` — Orchestrator → TUI

Written atomically by the orchestrator (`.tmp` rename) throughout the pipeline.
The Go TUI polls this file every 2 seconds and drives the heat bar animation.

```json
// Building state
{ "status": "building", "stage_label": "Architect — Designing item...", "stage_pct": 15 }

// Success state
{ "status": "ready", "batch_list": ["GelatinousBlade"], "stage_pct": 100,
  "message": "Compilation successful. Waiting for user..." }

// Error state
{ "status": "error", "error_code": "PIPELINE_FAIL",
  "message": "The pipeline collapsed: CoderAgent failed after 3 attempts" }
```

The `stage_pct` integer (0–100) is mapped to the heat bar width in the TUI.

### 4.3 `command_trigger.json` — TUI → ForgeConnector

Written by the Go TUI when the user presses Enter on the staging screen.
ForgeConnector's `FileSystemWatcher` picks this up and signals the main game thread
to hot-reload mods.

```json
{ "action": "execute", "timestamp": "2026-01-15T10:23:45.123Z" }
```

The file is deleted by ForgeConnector after reading to prevent duplicate triggers
(necessary on APFS which fires two change events per write).

### 4.4 `forge_connector_alive.json` — ForgeConnector → TUI

Written by ForgeConnector every time it loads. The Go TUI polls this file to determine
whether Terraria is running with the bridge mod active and shows a "⬡ Bridge Online"
indicator on the staging screen.

```json
{ "status": "listening", "pid": 12345, "loaded_at": "2026-01-15T10:20:00Z" }
```

---

## 5. Go TUI — BubbleTeaTerminal

**Files:** `BubbleTeaTerminal/main.go`, `BubbleTeaTerminal/styles.go`

The TUI follows the standard BubbleTea `Model / Update / View` architecture.

### 5.1 Screen State Machine

```
screenInput
  │  user types prompt, presses Enter
  ▼
screenMode
  │  "Auto-Forge" branch:            "Manual Override" branch:
  │    → writes user_request.json      → screenWizard
  │    → screenForge                        │ (5 wizard steps)
  ▼                                         │ → writes user_request.json
screenForge ◄──────────────────────────────┘ → screenForge
  │  heat bar animates, polls status every 2 s
  │  on "ready":
  ▼
screenStaging
  │  Enter: writes command_trigger.json (inject)
  │  C:     clears state → screenInput (craft another)
  └─ Esc:   exits TUI
```

The five wizard steps (only shown in Manual Override):

| Step | Options |
|------|---------|
| Tier | Starter / Dungeon / Hardmode / Endgame |
| Damage Class | Melee / Ranged / Magic |
| Use Style | Swing / Stab / Hold |
| Projectile | None / Standard Shot / Beam Slash / Thrown |
| Crafting Station | Auto / By Hand / Workbench / Iron Anvil / Mythril Anvil / Ancient Manipulator |

### 5.2 Core Data Structure

```go
type model struct {
    state           screen
    width, height   int

    // Wizard
    prompt          string
    tier            string
    damageClass     string
    styleChoice     string
    projectile      string
    craftingStation string

    // Forge screen
    heat            int        // 0–100, drives heat bar
    targetHeat      int        // Smooth animation target
    stageLabel      string
    forgeItemName   string
    forgeErr        string

    // Staging screen
    bridgeAlive     bool       // ForgeConnector heartbeat present?
    injecting       bool       // Injection in progress
}
```

### 5.3 Filesystem Polling

The `checkStatusCmd` BubbleTea `tea.Cmd` runs every 2 seconds. It reads
`generation_status.json`, parses the JSON, and returns a `statusMsg` that drives
`Update()`. Heat animation is achieved by interpolating `heat` toward `targetHeat`
by a fixed step each tick (60 ms timer), so the bar glides smoothly even during
long stages.

The orchestrator is launched as a subprocess at startup:

```go
cmd := exec.Command("python3", "orchestrator.py")
cmd.Dir = "../agents"
cmd.Start()
```

The PID is tracked so the TUI can kill the daemon on exit.

### 5.4 UI Styling

Defined in `styles.go` using Lip Gloss:

| Element | Color / Style |
|---------|---------------|
| Background | `#0B0A10` |
| Rune accent | `#4DDB80` (green) |
| Gold accent | `#C8A14A` |
| Heat bar (hot) | `#FF4500` → `#FF8C00` gradient |
| Heat bar (cool) | `#1A1A2E` |
| Item name reveal | Phases: hidden → partial (`?` chars) → full |

Terminal fallback: compact single-column layout when `width < 84` or `height < 24`.

---

## 6. Python Orchestrator

**File:** `agents/orchestrator.py`

### 6.1 Startup

```python
main()
  → asyncio.new_event_loop()
  → watchdog.Observer watching ModSources/
      filter: user_request.json modifications
  → loop.run_forever()
```

A single `asyncio.Lock` serialises all pipeline runs. If a second request arrives
while a pipeline is in progress, it queues and runs after the first completes.

### 6.2 Request Handling

```python
class _RequestHandler(FileSystemEventHandler):
    def on_modified(event):
        if event.src_path != str(REQUEST_FILE): return
        if now - self._last_trigger < 1.0: return  # 1 s debounce
        payload = json.loads(REQUEST_FILE.read_text())
        asyncio.run_coroutine_threadsafe(self._run_safe(payload), self._loop)

    async def _run_safe(request):
        async with self._lock:
            try:
                await run_pipeline(request)
            except Exception as exc:
                _set_error(str(exc))
```

### 6.3 Pipeline Execution

```python
async def run_pipeline(request: dict) -> None:
    prompt = request["prompt"]
    tier   = request.get("tier", "Tier1_Starter")
    crafting_station = request.get("crafting_station")   # None = Auto

    _set_stage("Kindling the Forge...", 5)

    # Sequential: Architect
    manifest = ArchitectAgent().generate_manifest(prompt, tier, crafting_station)

    _set_stage("Smithing code and art...", 40)

    # Parallel: Coder + Artist (run in ThreadPoolExecutor)
    coder  = CoderAgent()
    artist = ArtistAgent()
    loop   = asyncio.get_running_loop()
    code_result, art_result = await asyncio.gather(
        loop.run_in_executor(None, coder.write_code,    manifest),
        loop.run_in_executor(None, artist.generate_asset, manifest),
    )

    if code_result["status"] != "success": raise RuntimeError(...)
    if art_result["status"]  != "success": raise RuntimeError(...)

    _set_stage("Gatekeeper — Compiling mod...", 80)

    # Sequential: Gatekeeper
    gate_result = Integrator(coder=coder).build_and_verify(
        forge_output=code_result,
        sprite_path=art_result["item_sprite_path"],
        projectile_sprite_path=art_result.get("projectile_sprite_path"),
    )
    if gate_result["status"] != "success": raise RuntimeError(...)

    _set_ready(gate_result["item_name"])
```

### 6.4 Atomic Status Writes

```python
def _write_status(payload: dict) -> None:
    tmp = STATUS_FILE.with_suffix(".tmp")
    tmp.write_text(json.dumps(payload, indent=2))
    tmp.replace(STATUS_FILE)   # atomic rename — TUI never sees a partial file
```

---

## 7. Architect Agent

**Files:** `agents/architect/architect.py`, `models.py`, `prompts.py`,
`reference_finder.py`, `reference_policy.py`

### 7.1 Responsibilities

1. Parse user intent from the natural-language prompt.
2. Query an LLM to design balanced item stats, visuals, and mechanics.
3. Deterministically resolve crafting materials and stats from a tier table.
4. Optionally fetch and approve a reference image (for recognized subjects).
5. Return a validated, clamped `ItemManifest` dict.

### 7.2 LLM Chain

```python
chain = (
    ChatPromptTemplate.from_messages([system_msg, human_msg])
    | ChatOpenAI(model="gpt-5.4")
      .with_structured_output(LLMItemOutput)   # Pydantic schema enforced by OpenAI
)
result: LLMItemOutput = chain.invoke({
    "user_prompt":     prompt,
    "selected_tier":   tier,
    "damage_min":      TIER_TABLE[tier]["damage"][0],
    "damage_max":      TIER_TABLE[tier]["damage"][1],
    "use_time_min":    TIER_TABLE[tier]["use_time"][0],
    "use_time_max":    TIER_TABLE[tier]["use_time"][1],
})
```

The system prompt instructs the LLM to produce:
- PascalCase `item_name` and human-readable `display_name`
- `type` (`Weapon`, `Armor`, etc.) and `sub_type` (physical shape: `Sword`, `Gun`, etc.)
- `stats`: damage, knockback, crit, use_time, auto_reuse, rarity
- `visuals`: 2–4 hex color palette, SD image description, icon size
- `mechanics`: shoot_projectile, on_hit_buff, crafting details
- `reference_needed` (bool) and `reference_subject` (string | null)

The prompt explicitly forbids using damage class names as `sub_type`
(e.g., `"Melee"` is illegal; `"Sword"` is correct).

### 7.3 Tier Balance Table

| Tier | Damage | Use Time | Default Rarity | Crafting Tile |
|------|--------|----------|----------------|---------------|
| `Tier1_Starter`  | 8–15    | 20–30 | White  | WorkBenches |
| `Tier2_Dungeon`  | 25–40   | 18–25 | Orange | Anvils |
| `Tier3_Hardmode` | 45–65   | 15–22 | Pink   | Anvils |
| `Tier4_Endgame`  | 150–300 | 8–15  | Red    | Anvils |

Stats that exceed the tier bounds are silently clamped by a Pydantic `model_validator`:

```python
@model_validator(mode="before")
def clamp_to_tier(cls, values, info):
    tier = info.context.get("tier")
    dmg_lo, dmg_hi = TIER_TABLE[tier]["damage"]
    values["damage"] = max(dmg_lo, min(dmg_hi, values["damage"]))
    # same for use_time
    return values
```

### 7.4 Deterministic Crafting Resolution

Before calling the LLM, `resolve_crafting()` runs a keyword scan on the user prompt
and maps thematic words to Terraria material IDs:

```python
THEME_MATERIAL_MAP = {
    frozenset(["fire", "magma", "hell", "flame"]):   "ItemID.HellstoneBar",
    frozenset(["ice",  "frost", "cryo", "snow"]):    "ItemID.IceBlock",
    frozenset(["dark", "evil", "shadow", "corrupt"]): "ItemID.DemoniteBar",
    frozenset(["light","holy", "angel", "hallow"]):  "ItemID.HallowedBar",
    frozenset(["jungle","spore","vine","mushroom"]):  "ItemID.JungleSpores",
    # ...
}
```

This output is merged into the LLM manifest after generation, overriding whatever
the LLM chose for crafting materials.

### 7.5 Item Name Sanitization

```python
def _to_pascal_case(name: str) -> str:
    cleaned = re.sub(r"[^a-zA-Z0-9]", " ", name)
    cleaned = re.sub(r"([a-z])([A-Z])", r"\1 \2", cleaned)
    return "".join(w.capitalize() for w in cleaned.split())
# "slime sword of bouncing" → "SlimeSwordOfBouncing"
# "IceBlade_v2"             → "IceBladeV2"
```

### 7.6 Reference Image Pipeline

Triggered when `reference_needed=True` (LLM decision).

```
BrowserReferenceFinder.find_candidates(subject, prompt)
  │
  ├── Build search queries:
  │     "{subject}"
  │     "{subject} pixel art"
  │     "{subject} sprite illustration"
  │
  ├── Playwright headless Chromium → Bing Images
  │     https://www.bing.com/images/search?q={query}
  │     Extract <a class="iusc"> card metadata (JSON in "m" attribute):
  │       murl (original URL), t (title), purl (page), mw/mh (dimensions)
  │
  └── Score and rank candidates:
        +30  pixel_art in title/URL
        +24  sprite in title/URL
        +18  illustration / concept_art
        +14  plain background hint
        +10  single subject (no group/collage)
        +6   "official"/"art"/"render" keywords
        +5   square aspect ratio (≤1.5)
        +3   .png extension
        −6   oversized (>2000 px)
        −8   clutter hint per occurrence
        −12  extreme aspect ratio (>3.0)
        Reject entirely: blocked sources, photos of real objects

HybridReferenceApprover.approve(candidates, prompt, subject, ...)
  │
  ├── For each candidate: download thumbnail → base64 encode
  ├── LLM (gpt-4o vision) picks best index or returns null
  │     Criteria: isolated item, illustration/concept/pixel art,
  │               no UI cards, no watermarks, no merchandise
  │
  └── Fallback (if thumbnails unavailable): accept top if score ≥ 25
```

The approver runs up to `max_retries + 1` (default: 2) search+approve cycles.
On every failure the subject query is refined with `refine_subject()`.
If all attempts fail, `generation_mode` falls back to `"text_to_image"` silently.

### 7.7 Output: ItemManifest

The final validated dict has this shape (see [Section 12](#12-data-contracts--schemas)
for the full schema).

---

## 8. Forge Master Agent

**Files:** `agents/forge_master/forge_master.py`, `models.py`, `prompts.py`,
`templates.py`, `compilation_harness.py`

### 8.1 Responsibilities

Generate a compilable tModLoader 1.4.4 C# source file and an HJSON localization file
from the `ItemManifest`. Self-heal using an LLM repair chain when static validation
finds banned or missing patterns.

### 8.2 Sub-type Mappings

```python
DAMAGE_CLASS_MAP = {
    "Sword":   "DamageClass.Melee",     "Gun":   "DamageClass.Ranged",
    "Bow":     "DamageClass.Ranged",    "Staff": "DamageClass.Magic",
    "Summon":  "DamageClass.Summon",    "Whip":  "DamageClass.SummonMeleeSpeed",
    # default → DamageClass.Melee
}

USE_STYLE_MAP = {
    "Sword":   "ItemUseStyleID.Swing",  "Gun":    "ItemUseStyleID.Shoot",
    "Bow":     "ItemUseStyleID.Shoot",  "Staff":  "ItemUseStyleID.Shoot",
    "Summon":  "ItemUseStyleID.Swing",  "Whip":   "ItemUseStyleID.Swing",
    # default → ItemUseStyleID.Swing
}
```

### 8.3 Inline Reference Snippets (RAG)

`templates.py` contains six complete, correct C# examples (one per weapon archetype)
that are injected verbatim into the LLM system prompt as ground truth:

- **Sword** — `ModItem` with `ItemUseStyleID.Swing`, `DamageClass.Melee`, `OnHitNPC`
- **Gun** — `ModItem` with `ModifyShootStats` for ammo velocity
- **Bow** — similar to Gun with arrow-specific projectile logic
- **Staff** — `DamageClass.Magic`, `ItemUseStyleID.Shoot`
- **Summon** — `ModItem` + `ModBuff` + `ModProjectile` (minion)
- **Whip** — `DefaultToWhip()` helper with a custom `ModProjectile`
- **Custom Projectile** — appended to any template when `custom_projectile=True`

### 8.4 LangChain Code Generation Chain

```python
gen_chain = (
    build_codegen_prompt()   # ChatPromptTemplate with system + human messages
    | ChatOpenAI(model="gpt-5.4").with_structured_output(CSharpOutput)
)

result: CSharpOutput = gen_chain.invoke({
    "manifest_json":    json.dumps(manifest, indent=2),
    "damage_class":     damage_class,
    "use_style":        use_style,
    "reference_snippet": reference_snippet,
})
```

### 8.5 Post-Generation Validation

After code generation (and after each repair attempt), `validate_cs()` scans the
source for banned and required patterns using compiled regexes:

**Banned patterns (raise validation error if found):**

| Pattern | Reason |
|---------|--------|
| `new ModRecipe` | tModLoader 1.3 API |
| `item\.melee\s*=` / `item\.ranged\s*=` / etc. | 1.3 DamageType API |
| `using System\.Drawing` | Crashes on Linux tModLoader |
| Old `OnHitNPC(NPC, int, float, bool)` signature | 1.3 signature |
| `mod\.GetItem<T>` | 1.3 API |
| Minion `penetrate` > 0 | Minions must have `penetrate = -1` |
| `DefaultToWhip(ProjectileID\.)` | Whips require a custom `ModProjectile` |

**Required patterns (raise validation error if absent):**

```
using Terraria;
using Terraria.ID;
using Terraria.ModLoader;
: ModItem
void SetDefaults()
```

### 8.6 Repair Loop

```
attempt = 1
while violations and attempt ≤ 3:
    repair_chain.invoke({
        "original_code": cs_code,
        "error_summary": "VALIDATION ERRORS:\n" + "\n".join(violations)
    })
    cs_code = _strip_markdown_fences(result)
    violations = validate_cs(cs_code)
    attempt += 1

if violations:
    return ForgeOutput(status="error", error=ForgeError("VALIDATION", ...))
```

### 8.7 HJSON Localization Output

Generated deterministically (no LLM) from the manifest fields:

```hjson
Mods: {
    ForgeGeneratedMod: {
        Items: {
            GelatinousBlade: {
                DisplayName: Gelatinous Blade
                Tooltip: A blade made of solidified slime.
            }
        }
    }
}
```

The Gatekeeper merges this block into the mod's `Localization/en-US.hjson` file,
preserving any existing entries.

### 8.8 `fix_code()` — Used by Gatekeeper

```python
def fix_code(self, error_log: str, original_code: str) -> dict:
    # Same repair chain, up to 3 attempts
    # Returns ForgeOutput dict (status = "success" or "error")
```

---

## 9. Pixelsmith Agent

**Files:** `agents/pixelsmith/pixelsmith.py`, `models.py`, `image_processing.py`,
`color_extraction.py`, `armor_compositor.py`, `variant_selector.py`,
`fal_flux2_runner.mjs`

### 9.1 Responsibilities

Generate game-ready pixel art PNG sprites by calling FLUX.2 Klein via fal-ai,
then post-process the output for Terraria's sprite format.

### 9.2 Generation Mode Resolution

```
generation_mode = manifest["generation_mode"]  # "text_to_image" or "image_to_image"
reference_url   = manifest["reference_image_url"]

if generation_mode == "image_to_image":
    if not reference_url or not env.FAL_IMAGE_TO_IMAGE_ENABLED:
        generation_mode = "text_to_image"   # silent fallback
        endpoint = FAL_MODEL_ENDPOINT       # text-to-image endpoint
    else:
        endpoint = FAL_IMG2IMG_ENDPOINT     # image-to-image endpoint
```

### 9.3 Weapon Orientation Mapping

The prompt is augmented with an orientation clause based on `sub_type` so the generated
sprite faces the correct direction for Terraria's rendering:

```python
WEAPON_ORIENTATION_MAP = {
    # 45° diagonal (held like a sword)
    "Sword":  "diagonal orientation tilted 45 degrees pointing upper-right",
    "Spear":  "diagonal orientation tilted 45 degrees pointing upper-right",
    # Vertical (bows, staves)
    "Bow":    "vertical orientation pointing straight up",
    "Staff":  "vertical orientation pointing straight up, slight tilt",
    # Horizontal (guns)
    "Gun":    "horizontal orientation pointing right",
    "Launcher": "horizontal orientation pointing right",
}
```

### 9.4 Prompt Construction

**Text-to-image prompt template:**
```
pixel art sprite, {description}, plain white background, centered,
{orientation}, hard edges, terraria game style, 2D, no anti-aliasing,
clean silhouette, high contrast{lora_trigger}
```

`{lora_trigger}` is `, aziib` when the local LoRA weights are loaded, otherwise empty.

**Image-to-image prompt (reference-aware):**
1. Download reference image, extract dominant colors via k-means (`color_extraction.py`)
2. Send reference thumbnail to an LLM (gpt-4o vision) to describe the weapon shape
   using the extracted color names
3. Append any accent colors the description missed
4. Use that description in the same `POSITIVE_TEMPLATE` above

When generating 4 variants (image-to-image mode), `variant_selector.py` picks the
best match by comparing each candidate against the reference image using a perceptual
metric.

### 9.5 fal-ai Bridge Architecture

The fal-ai JavaScript SDK is the only officially supported client for their LoRA
endpoints. Pixelsmith bridges Python → Node.js as follows:

```
Python: pixelsmith.py
  │  1. Build JSON payload (prompt, lora path, image size, steps, …)
  │  2. Write to temp file: /tmp/forge_payload_<uuid>.json
  │  3. subprocess.run(["node", "fal_flux2_runner.mjs", payload_path])
  │     env: FAL_KEY=...

Node.js: fal_flux2_runner.mjs
  │  1. Read payload JSON from argv[1]
  │  2. fal.subscribe(endpoint, { input })  →  polls fal-ai cloud queue
  │  3. Download result image URL → write PNG to output_path (from payload)
  │  4. Exit 0 on success, 1 on error

Python: pixelsmith.py
  └─ 4. Open PIL Image from output_path → return RGBA Image object
```

**fal-ai input payload:**
```json
{
  "prompt":               "pixel art sprite, ...",
  "guidance_scale":       5,
  "num_inference_steps":  28,
  "image_size":           { "width": 512, "height": 512 },
  "num_images":           1,
  "output_format":        "png",
  "loras":                [{ "path": "terraria_weights.safetensors", "scale": 0.85 }],
  "image_urls":           ["https://..."]   // img2img only
}
```

### 9.6 Post-Processing Pipeline

```
raw 512×512 PNG (white background)
  │
  ▼
rembg (U²-Net model)
  Remove background → transparent RGBA
  │
  ▼
Downscale (nearest-neighbor, Pillow)
  → target size from manifest visuals.icon_size (e.g., [32, 32])
  │
  ▼ (weapons / accessories)
Save: output/{ItemName}.png
```

**Armor path (different post-processing):**
```
raw 512×512 PNG
  │ remove background → downscale to 40×56
  ▼
armor_compositor.py
  Duplicate and transform into a 20-frame sprite sheet (40×1120 px)
  Save: output/{ItemName}_Body.png
```

---

## 10. Gatekeeper

**Files:** `agents/gatekeeper/gatekeeper.py`, `models.py`

### 10.1 Responsibilities

Stage generated files into the tModLoader `ModSources` directory, compile the mod
with a headless tModLoader build, and self-heal any Roslyn compiler errors using the
`CoderAgent.fix_code()` loop.

### 10.2 tModLoader Discovery

```python
SEARCH_PATHS = {
    "Darwin": [
        Path.home() / "Library/Application Support/Steam/steamapps/common/tModLoader",
    ],
    "Windows": [
        Path("C:/Program Files (x86)/Steam/steamapps/common/tModLoader"),
        Path("C:/Program Files/Steam/steamapps/common/tModLoader"),
    ],
    "Linux": [
        Path.home() / ".steam/steam/steamapps/common/tModLoader",
        Path.home() / ".local/share/Steam/steamapps/common/tModLoader",
    ],
}
# TMODLOADER_PATH env var overrides discovery entirely
```

### 10.3 File Staging

```python
mod_root = ~/Library/.../ModSources/ForgeGeneratedMod   # (platform-specific)

Content/Items/{ItemName}.cs        ← C# source
Content/Items/{ItemName}.png       ← item sprite
Content/Items/{ProjectileName}.png ← projectile sprite (if any)
Localization/en-US.hjson           ← merged localization block
build.txt                          ← mod metadata (name, author, version)
description.txt                    ← mod description
```

HJSON is merged (not replaced): existing entries for other items are preserved.

### 10.4 Compilation

```python
subprocess.run([
    str(tmod_dll),
    "-build", mod_name,
    "-tmlsavedirectory", str(tmod_saves_dir),
], capture_output=True, timeout=120)
```

The build runs `dotnet` under the hood. Success is detected by exit code 0;
Roslyn errors are extracted from stdout/stderr with:

```
regex: r"(\S+\.cs)\((\d+),(\d+)\):\s+error\s+(CS\d+):\s+(.+)"
```

### 10.5 Compilation Retry Loop

```
attempt = 1
loop:
    run tModLoader build
    if success: enable mod, return GatekeeperResult(status="success", attempts=attempt)
    
    errors = parse_roslyn_errors(build_output)
    if attempt > MAX_RETRIES (3): return GatekeeperResult(status="error", ...)
    
    repair_result = coder.fix_code(error_log=build_output, original_code=cs_code)
    if repair_result["status"] != "success": return GatekeeperResult(status="error", ...)
    
    cs_code = repair_result["cs_code"]
    stage cs_code back to disk
    attempt += 1
```

### 10.6 Mod Enabling

After a successful build the Gatekeeper appends `"ForgeGeneratedMod"` to
`{tModLoaderSaves}/Mods/enabled.json`, so the mod is active on next game load.

---

## 11. ForgeConnector (tModLoader Mod)

**Files:** `mod/ForgeConnector/ForgeConnectorSystem.cs`

### 11.1 Purpose

A minimal tModLoader `ModSystem` that:
1. Writes a heartbeat file so the TUI knows Terraria is running with it loaded.
2. Watches for `command_trigger.json` and hot-reloads all mods when triggered.

### 11.2 Lifecycle

```csharp
public override void PostSetupContent() {
    _modSourcesDir = GetModSourcesDir();   // Platform-aware path
    _triggerPath   = Path.Combine(_modSourcesDir, "command_trigger.json");
    _heartbeatPath = Path.Combine(_modSourcesDir, "forge_connector_alive.json");

    WriteHeartbeat();   // {"status":"listening","pid":...,"loaded_at":"..."}
    StartWatcher();     // FileSystemWatcher on command_trigger.json
}

public override void Unload() {
    _watcher?.Dispose();
    File.Delete(_heartbeatPath);  // Best-effort cleanup
}
```

### 11.3 File Watch & Thread-Safety

```csharp
// File watcher fires on a threadpool thread
private void OnTriggerEvent(object sender, FileSystemEventArgs e) {
    Thread.Sleep(50);  // Let writer finish flushing
    try {
        string json = File.ReadAllText(_triggerPath);
        using JsonDocument doc = JsonDocument.Parse(json);
        if (doc.RootElement.TryGetProperty("action", out var el)
            && el.GetString() == "execute") {
            File.Delete(_triggerPath);               // Prevent APFS double-fire
            Interlocked.Exchange(ref _reloadRequested, 1);  // Signal main thread
        }
    } catch { /* file mid-write — ignore */ }
}

// Main game thread hook — runs every frame
public override void PostUpdateEverything() {
    if (Interlocked.Exchange(ref _reloadRequested, 0) == 1) {
        TriggerReload();
    }
}
```

### 11.4 Hot-Reload via Reflection

tModLoader does not expose a public `Reload()` API, so `TriggerReload()` uses
reflection with two fallback strategies:

```csharp
// Strategy 1: Set Main.menuMode to tModLoader's internal reload menu ID
var interfaceType = Type.GetType("Terraria.ModLoader.UI.Interface, tModLoader");
int reloadId = (int)interfaceType.GetField("reloadModsID", BindingFlags.Static | ...).GetValue(null);
Main.menuMode = reloadId;

// Strategy 2 (fallback): Directly invoke ModLoader.Reload()
typeof(ModLoader)
    .GetMethod("Reload", BindingFlags.Static | BindingFlags.NonPublic)
    ?.Invoke(null, null);
```

Both strategies trigger tModLoader's standard "Reloading mods…" screen, which
recompiles and relinks `ForgeGeneratedMod` in memory without restarting Terraria.

---

## 12. Data Contracts & Schemas

### 12.1 ItemManifest (Architect → Coder + Artist)

```json
{
  "item_name":     "GelatinousBlade",
  "display_name":  "Gelatinous Blade",
  "tooltip":       "A blade made of solidified slime.",
  "type":          "Weapon",
  "sub_type":      "Sword",
  "stats": {
    "damage":      18,
    "knockback":   4.0,
    "crit_chance": 4,
    "use_time":    25,
    "auto_reuse":  true,
    "rarity":      "ItemRarityID.Green"
  },
  "visuals": {
    "color_palette": ["#00FF00", "#FFFFFF"],
    "description":   "A translucent green sword with dripping slime effects.",
    "icon_size":     [32, 32]
  },
  "mechanics": {
    "shoot_projectile":   "ModContent.ProjectileType<SpinningSlimeStar>()",
    "on_hit_buff":        "BuffID.Slimed",
    "custom_projectile":  true,
    "crafting_material":  "ItemID.Gel",
    "crafting_cost":      20,
    "crafting_tile":      "TileID.Anvils"
  },
  "projectile_visuals": {
    "description": "A spinning green star with slime trails",
    "icon_size":   [16, 16]
  },
  "reference_needed":    true,
  "reference_subject":   "Master Sword",
  "reference_image_url": "https://imgur.com/xyz.png",
  "generation_mode":     "image_to_image",
  "reference_attempts":  1,
  "reference_notes":     "llm_approved"
}
```

### 12.2 ForgeOutput (Forge Master → Gatekeeper)

```json
{
  "cs_code":   "using Terraria;\nusing Terraria.ID;\n...",
  "hjson_code": "Mods: { ForgeGeneratedMod: { Items: { ... } } }",
  "status":    "success",
  "error":     null
}
```

### 12.3 PixelsmithOutput (Pixelsmith → Gatekeeper)

```json
{
  "item_sprite_path":       "/path/to/output/GelatinousBlade.png",
  "projectile_sprite_path": "/path/to/output/SpinningSlimeStar.png",
  "status": "success",
  "error":  null
}
```

### 12.4 GatekeeperResult (Gatekeeper → Orchestrator)

```json
{
  "status":        "success",
  "item_name":     "GelatinousBlade",
  "attempts":      1,
  "errors":        null,
  "error_message": null
}
```

---

## 13. Error Handling & Retry Strategies

### 13.1 Orchestrator (top-level)

All exceptions from any pipeline step are caught by `_run_safe()` and written to
`generation_status.json` as `{"status":"error","message":"..."}`. The TUI reads
this and shows the error on the forge screen. There are no automatic retries at
the orchestrator level.

### 13.2 Architect — Reference Pipeline Fallback

| Failure | Recovery |
|---------|----------|
| Playwright / Bing search error | `find_candidates()` returns `[]`, next attempt uses refined query |
| LLM rejects all image candidates | `refine_subject()` + retry (max 1 retry) |
| All retries exhausted | `generation_mode` silently falls back to `"text_to_image"` |
| Thumbnail download fails | Fallback: accept top candidate if score ≥ 25 |

### 13.3 Forge Master — Validation Repair Loop

Up to **3 LLM repair attempts** for static validation failures. Each attempt feeds
the original code plus a numbered list of violations back to the model. After 3
failures, `write_code()` returns `status="error"` (no exception).

### 13.4 Pixelsmith — Generation Fallback

| Failure | Recovery |
|---------|----------|
| img2img requested, no URL | Silent fallback to text-to-image |
| img2img requested, env flag off | Silent fallback to text-to-image |
| Node.js subprocess non-zero exit | `RuntimeError` → orchestrator catches → pipeline error |

### 13.5 Gatekeeper — Compilation Retry Loop

Up to **3 Roslyn compiler error repair cycles** (4 total build attempts). Each cycle:
1. Runs the tModLoader headless build.
2. Parses Roslyn error lines from stdout/stderr.
3. Calls `coder.fix_code(error_log, original_code)` to get patched C#.
4. Writes patched C# back to disk and retries.

On the 4th build failure (or if `fix_code` itself returns an error), the Gatekeeper
returns `status="error"` with the full error log attached.

### 13.6 ForgeConnector — Thread Safety

The `_reloadRequested` int field uses `Interlocked.Exchange` to safely communicate
between the `FileSystemWatcher` threadpool thread and the main game thread without
locks or blocking.

---

## 14. External Services & APIs

### 14.1 OpenAI

| Agent | Model | Purpose |
|-------|-------|---------|
| Architect | gpt-5.4 | Generate `ItemManifest` (structured output) |
| Architect | gpt-5.4 | Approve reference images (structured output) |
| Forge Master | gpt-5.4 | Generate C# code (structured output) |
| Forge Master | gpt-5.4 | Repair invalid C# (up to 3 calls) |
| Pixelsmith | gpt-4o (vision) | Describe weapon shape from reference image |
| Pixelsmith | gpt-4o (vision) | Select best image variant |
| Gatekeeper | gpt-5.4 (via Coder) | Repair compiler errors (up to 3 calls) |

All calls use LangChain's `ChatOpenAI` wrapper with structured output where applicable.

### 14.2 fal-ai (FLUX.2 Klein)

| Endpoint | Mode |
|----------|------|
| `fal-ai/flux-2/klein/9b/base/lora` | Text-to-image |
| `fal-ai/flux-2/klein/9b/base/edit/lora` | Image-to-image |

Requires: `FAL_KEY` env var. Bridged via a Node.js subprocess
(`fal_flux2_runner.mjs`) using the `@fal-ai/client` npm package because the official
SDK is JavaScript-only.

The Terraria LoRA weights (`terraria_weights.safetensors`) are injected at
`scale=0.85` to steer generation toward Terraria's art style.

### 14.3 Bing Images (Reference Search)

**URL:** `https://www.bing.com/images/search?q={query}`

**Method:** Playwright headless Chromium (no API key required)

**Data extracted** from each `<a class="iusc">` card's `m` JSON attribute:
- `murl` — original image URL
- `t` — title
- `purl` — source page URL
- `mw`, `mh` — image dimensions

### 14.4 tModLoader CLI (Compilation)

**Command:**
```bash
dotnet tModLoader.dll -build ForgeGeneratedMod \
    -tmlsavedirectory ~/Library/.../tModLoader
```

**Output:** Roslyn compiler errors (`file(line,col): error CSxxxx: message`).

**Discovery:** Scanned from common Steam install paths (macOS / Windows / Linux).

---

## 15. Environment Variables

| Variable | Required | Default | Purpose |
|----------|----------|---------|---------|
| `OPENAI_API_KEY` | ✅ | — | All LLM calls |
| `FAL_KEY` or `FAL_API_KEY` | ✅ | — | Image generation |
| `FAL_IMAGE_TO_IMAGE_ENABLED` | ❌ | `false` | Enable img2img mode |
| `FAL_IMAGE_TO_IMAGE_ENDPOINT` | ❌ | Built-in URL | Override img2img endpoint |
| `TMODLOADER_PATH` | ❌ | Auto-discovered | Override tModLoader install path |
| `MOD_SOURCE_PATH` | ❌ | Platform default | Override ModSources directory |

All variables can be loaded from `agents/.env` via `python-dotenv`.

---

## 16. File I/O Map

All runtime files live under the tModLoader `ModSources/` directory.

| File | Writer | Reader | Purpose |
|------|--------|--------|---------|
| `user_request.json` | Go TUI | Orchestrator (watchdog) | Pipeline trigger + user input |
| `generation_status.json` | Orchestrator (atomic) | Go TUI (2 s poll) | Build progress + final result |
| `command_trigger.json` | Go TUI | ForgeConnector (FSW) | Hot-reload signal |
| `forge_connector_alive.json` | ForgeConnector | Go TUI (2 s poll) | Bridge heartbeat |
| `ForgeGeneratedMod/Content/Items/{Name}.cs` | Gatekeeper | tModLoader CLI | ModItem C# source |
| `ForgeGeneratedMod/Content/Items/{Name}.png` | Gatekeeper | tModLoader | Item sprite |
| `ForgeGeneratedMod/Content/Items/{ProjName}.png` | Gatekeeper | tModLoader | Projectile sprite |
| `ForgeGeneratedMod/Localization/en-US.hjson` | Gatekeeper (merge) | tModLoader | Display names + tooltips |
| `ForgeGeneratedMod/build.txt` | Gatekeeper | tModLoader | Mod metadata (name, author) |
| `ForgeGeneratedMod/description.txt` | Gatekeeper | tModLoader | Mod Steam description |
| `{tMLSaves}/Mods/enabled.json` | Gatekeeper (append) | tModLoader | Enabled mods list |

Intermediate/temporary files:

| File | Location | Purpose |
|------|----------|---------|
| `forge_payload_<uuid>.json` | `/tmp/` | fal-ai call input (Node.js bridge) |
| `forge_output_<uuid>.png` | `/tmp/` | fal-ai raw PNG output |
| `generation_status.tmp` | `ModSources/` | Atomic rename buffer for status writes |

---

## Further Reading

- [`agents/docs/pipeline-flow.md`](agents/docs/pipeline-flow.md) — step-by-step agent
  walkthrough with inline code excerpts
- [`ReferenceDesign.MD`](ReferenceDesign.MD) — original design document for the
  reference image feature
- [`agents/README.md`](agents/README.md) — runtime setup notes (weights, env vars)
