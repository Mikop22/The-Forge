# Storm Brand Presentation Polish Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add bounded runtime presentation polish for `storm_brand` staff projectiles and marked targets without weakening direct inject, hidden-audition identity, or the existing 3-hit starfall cashout.

**Architecture:** Preserve package and FX-profile identity from the existing manifest fields, route that identity through a bounded C# presentation registry, and trigger cast, travel, hit, and target-mark effects from `ForgeItemGlobal`, `ForgeProjectileGlobal`, and a new NPC visual global. Verify the routing with source-contract tests, then verify the runtime with `dotnet build` and live reinject.

**Tech Stack:** Python manifest pipeline, C# `ForgeConnector`, pytest, `dotnet build`, live tModLoader direct inject

---

### Task 1: Add failing source-contract tests for presentation routing

**Files:**
- Create: `agents/tests/test_storm_brand_presentation_source_contract.py`
- Read: `mod/ForgeConnector/ForgeManifestStore.cs`
- Read: `mod/ForgeConnector/ForgeConnectorSystem.cs`
- Read: `mod/ForgeConnector/ForgeItemGlobal.cs`
- Read: `mod/ForgeConnector/ForgeProjectileGlobal.cs`

**Step 1: Write the failing test**

```python
from pathlib import Path


def _read(path: str) -> str:
    return Path(path).read_text(encoding="utf-8")


def test_runtime_data_preserves_package_and_presentation_identity() -> None:
    source = _read("mod/ForgeConnector/ForgeManifestStore.cs")
    assert "CombatPackageKey" in source
    assert "PresentationProfile" in source
    assert "PresentationModule" in source


def test_connector_parses_presentation_fields_from_existing_manifest_shape() -> None:
    source = _read("mod/ForgeConnector/ForgeConnectorSystem.cs")
    assert 'GetStr(mechanics, "combat_package"' in source
    assert 'GetStr(presentation, "fx_profile"' in source
    assert 'GetStr(resolvedCombat, "presentation_module"' in source


def test_storm_brand_routes_through_runtime_presentation_helpers() -> None:
    item_source = _read("mod/ForgeConnector/ForgeItemGlobal.cs")
    projectile_source = _read("mod/ForgeConnector/ForgeProjectileGlobal.cs")
    assert "ApplyCastPresentation" in item_source
    assert "ApplyTravelPresentation" in projectile_source
    assert "ApplyHitPresentation" in projectile_source
```

**Step 2: Run test to verify it fails**

Run: `pytest -q agents/tests/test_storm_brand_presentation_source_contract.py`

Expected: FAIL because the presentation fields and helpers do not exist yet.

**Step 3: Commit the failing test**

```bash
git add agents/tests/test_storm_brand_presentation_source_contract.py
git commit -m "test: add storm brand presentation routing contract"
```

### Task 2: Add bounded presentation identity to runtime data objects

**Files:**
- Modify: `mod/ForgeConnector/ForgeManifestStore.cs`
- Modify: `mod/ForgeConnector/ForgeConnectorSystem.cs`
- Test: `agents/tests/test_storm_brand_presentation_source_contract.py`

**Step 1: Write the minimal runtime fields**

Add fields to `ForgeItemData` and `ForgeProjectileData`:

```csharp
public string CombatPackageKey { get; set; } = "";
public string PresentationProfile { get; set; } = "";
public string PresentationModule { get; set; } = "";
```

**Step 2: Parse existing manifest fields into those runtime fields**

In `ParseManifest(...)` and `ParseProjectile(...)`, read:

```csharp
JsonElement presentation = TryGetObject(manifest, "presentation", out var presentationValue) ? presentationValue : default;
JsonElement resolvedCombat = TryGetObject(manifest, "resolved_combat", out var resolvedCombatValue) ? resolvedCombatValue : default;

data.CombatPackageKey = GetStr(resolvedCombat, "package_key", GetStr(mechanics, "combat_package", ""));
data.PresentationProfile = GetStr(presentation, "fx_profile", "");
data.PresentationModule = GetStr(resolvedCombat, "presentation_module", "");
```

**Step 3: Run test to verify partial progress**

Run: `pytest -q agents/tests/test_storm_brand_presentation_source_contract.py::test_runtime_data_preserves_package_and_presentation_identity agents/tests/test_storm_brand_presentation_source_contract.py::test_connector_parses_presentation_fields_from_existing_manifest_shape`

Expected: PASS

**Step 4: Commit**

```bash
git add mod/ForgeConnector/ForgeManifestStore.cs mod/ForgeConnector/ForgeConnectorSystem.cs agents/tests/test_storm_brand_presentation_source_contract.py
git commit -m "feat: preserve presentation identity in forge runtime data"
```

### Task 3: Add a bounded C# presentation profile registry

**Files:**
- Create: `mod/ForgeConnector/ForgePresentationProfiles.cs`
- Modify: `mod/ForgeConnector/ForgeItemGlobal.cs`
- Modify: `mod/ForgeConnector/ForgeProjectileGlobal.cs`
- Test: `agents/tests/test_storm_brand_presentation_source_contract.py`

**Step 1: Write the minimal registry**

Create a bounded registry:

```csharp
internal readonly record struct ForgePresentationProfile(
    int CastDustId,
    int TrailDustId,
    int HitDustId,
    float ProjectileLight,
    float PulseAmplitude);

internal static class ForgePresentationProfiles
{
    public static bool TryResolve(string packageKey, string profile, out ForgePresentationProfile resolved)
    {
        if (string.Equals(packageKey, "storm_brand", StringComparison.OrdinalIgnoreCase)
            && string.Equals(profile, "celestial_shock", StringComparison.OrdinalIgnoreCase))
        {
            resolved = new ForgePresentationProfile(
                CastDustId: DustID.Electric,
                TrailDustId: DustID.Electric,
                HitDustId: DustID.MagicMirror,
                ProjectileLight: 0.55f,
                PulseAmplitude: 0.06f);
            return true;
        }

        resolved = default;
        return false;
    }
}
```

**Step 2: Add minimal helper methods without wiring them fully**

Add method stubs:

```csharp
private static void ApplyCastPresentation(...) { }
private static void ApplyTravelPresentation(...) { }
private static void ApplyHitPresentation(...) { }
```

**Step 3: Run full source-contract test**

Run: `pytest -q agents/tests/test_storm_brand_presentation_source_contract.py`

Expected: PASS

**Step 4: Commit**

```bash
git add mod/ForgeConnector/ForgePresentationProfiles.cs mod/ForgeConnector/ForgeItemGlobal.cs mod/ForgeConnector/ForgeProjectileGlobal.cs agents/tests/test_storm_brand_presentation_source_contract.py
git commit -m "feat: add bounded storm brand presentation registry"
```

### Task 4: Add cast flash and firing sound on item use

**Files:**
- Modify: `mod/ForgeConnector/ForgeItemGlobal.cs`
- Test: `agents/tests/test_storm_brand_presentation_source_contract.py`

**Step 1: Route `Shoot(...)` through a cast presentation helper**

In `Shoot(...)`, after telemetry emit, call:

```csharp
ApplyCastPresentation(player, position, velocity, data);
```

**Step 2: Implement the minimal cast presentation**

Inside `ApplyCastPresentation(...)`:

```csharp
if (!ForgePresentationProfiles.TryResolve(data.CombatPackageKey, data.PresentationProfile, out var profile))
    return;

Vector2 muzzle = position + Vector2.Normalize(velocity) * 12f;
for (int i = 0; i < 8; i++)
{
    Dust dust = Dust.NewDustDirect(muzzle - new Vector2(4f), 8, 8, profile.CastDustId);
    dust.noGravity = true;
    dust.velocity *= 0.2f;
    dust.scale = 1.1f;
}

SoundEngine.PlaySound(SoundID.Item43, muzzle);
Lighting.AddLight(muzzle, 0.2f, 0.3f, 0.55f);
```

**Step 3: Run source-contract test**

Run: `pytest -q agents/tests/test_storm_brand_presentation_source_contract.py`

Expected: PASS

**Step 4: Build connector**

Run: `dotnet build mod/ForgeConnector/ForgeConnector.csproj -v minimal`

Expected: `Build succeeded.`

**Step 5: Commit**

```bash
git add mod/ForgeConnector/ForgeItemGlobal.cs
git commit -m "feat: add storm brand cast flash and fire sound"
```

### Task 5: Add travel light, trail, and pulse to projectile flight

**Files:**
- Modify: `mod/ForgeConnector/ForgeProjectileGlobal.cs`
- Test: `agents/tests/test_storm_brand_presentation_source_contract.py`

**Step 1: Wire travel presentation into `AI(...)`**

Add:

```csharp
ApplyTravelPresentation(projectile, data);
```

before the AI mode switch returns for normal projectiles.

**Step 2: Implement the minimal travel presentation**

```csharp
private static void ApplyTravelPresentation(Projectile projectile, ForgeProjectileData data)
{
    if (!ForgePresentationProfiles.TryResolve(data.CombatPackageKey, data.PresentationProfile, out var profile))
        return;

    projectile.light = Math.Max(projectile.light, profile.ProjectileLight);

    if (Main.rand.NextBool(2))
    {
        Dust dust = Dust.NewDustDirect(projectile.position, projectile.width, projectile.height, profile.TrailDustId);
        dust.noGravity = true;
        dust.scale = 1.0f;
        dust.velocity = projectile.velocity * 0.15f;
    }
}
```

**Step 3: Add pulse/wobble in `PreDraw(...)`**

Adjust draw scale and rotation slightly:

```csharp
float pulse = 1f + (float)Math.Sin(Main.GameUpdateCount * 0.35f) * profile.PulseAmplitude;
float drawRotation = projectile.rotation + (float)Math.Sin(Main.GameUpdateCount * 0.2f) * 0.05f;
```

Use `pulse` and `drawRotation` in the custom draw call.

**Step 4: Run tests**

Run: `pytest -q agents/tests/test_storm_brand_presentation_source_contract.py`

Expected: PASS

**Step 5: Build**

Run: `dotnet build mod/ForgeConnector/ForgeConnector.csproj -v minimal`

Expected: `Build succeeded.`

**Step 6: Commit**

```bash
git add mod/ForgeConnector/ForgeProjectileGlobal.cs
git commit -m "feat: add storm brand projectile travel polish"
```

### Task 6: Add hit burst on contact

**Files:**
- Modify: `mod/ForgeConnector/ForgeProjectileGlobal.cs`
- Test: `agents/tests/test_runtime_lab_contract.py`

**Step 1: Extend `OnHitNPC(...)` with a hit presentation helper**

Add:

```csharp
ApplyHitPresentation(projectile, target, stackCount, data);
```

before the cashout return path and before the escalate telemetry emit.

**Step 2: Implement the minimal hit burst**

```csharp
private static void ApplyHitPresentation(Projectile projectile, NPC target, int stackCount, ForgeProjectileData data)
{
    if (!ForgePresentationProfiles.TryResolve(data.CombatPackageKey, data.PresentationProfile, out var profile))
        return;

    for (int i = 0; i < 10; i++)
    {
        Vector2 speed = Main.rand.NextVector2Circular(2.5f, 2.5f);
        Dust dust = Dust.NewDustPerfect(target.Center, profile.HitDustId, speed);
        dust.noGravity = true;
        dust.scale = stackCount >= 3 ? 1.4f : 1.0f;
    }

    Lighting.AddLight(target.Center, 0.15f, 0.25f, 0.45f);
}
```

**Step 3: Run targeted runtime lab regression**

Run: `pytest -q agents/tests/test_runtime_lab_contract.py`

Expected: PASS

**Step 4: Build**

Run: `dotnet build mod/ForgeConnector/ForgeConnector.csproj -v minimal`

Expected: `Build succeeded.`

**Step 5: Commit**

```bash
git add mod/ForgeConnector/ForgeProjectileGlobal.cs
git commit -m "feat: add storm brand projectile hit burst"
```

### Task 7: Expose stack count for visuals and add a target-mark draw hook

**Files:**
- Modify: `mod/ForgeConnector/ForgeLabTelemetry.cs`
- Create: `mod/ForgeConnector/ForgeNpcVisualGlobal.cs`
- Test: `agents/tests/test_storm_brand_presentation_source_contract.py`

**Step 1: Expose a read-only stack query**

Add a helper in `ForgeLabTelemetry`:

```csharp
public static int GetTargetStackCount(ForgeLabTelemetryContext context, NPC target)
{
    string key = BuildTargetKey(context, target.whoAmI);
    lock (_sync)
    {
        return _targetStacks.TryGetValue(key, out var state) ? state.StackCount : 0;
    }
}
```

**Step 2: Create an NPC visual global**

Add a new `GlobalNPC` that draws a small sigil above marked targets:

```csharp
public override void PostDraw(NPC npc, SpriteBatch spriteBatch, Vector2 screenPos, Color drawColor)
{
    int stackCount = ResolveStormBrandStackCount(npc);
    if (stackCount <= 0)
        return;

    Vector2 center = npc.Top - screenPos + new Vector2(0f, -12f);
    float scale = 0.7f + stackCount * 0.15f;
    Color color = Color.Lerp(Color.Cyan, Color.White, 0.35f);
    Utils.DrawBorderStringFourWay(spriteBatch, FontAssets.ItemStack.Value, stackCount.ToString(), center.X, center.Y, color, Color.Transparent, Vector2.Zero, scale);
}
```

Use a simple first version. Do not block on a perfect custom glyph.

**Step 3: Run source-contract test**

Run: `pytest -q agents/tests/test_storm_brand_presentation_source_contract.py`

Expected: PASS

**Step 4: Build**

Run: `dotnet build mod/ForgeConnector/ForgeConnector.csproj -v minimal`

Expected: `Build succeeded.`

**Step 5: Commit**

```bash
git add mod/ForgeConnector/ForgeLabTelemetry.cs mod/ForgeConnector/ForgeNpcVisualGlobal.cs
git commit -m "feat: add storm brand target mark visuals"
```

### Task 8: Run regression suites and live direct-inject verification

**Files:**
- Test: `agents/tests/test_storm_brand_presentation_source_contract.py`
- Test: `agents/tests/test_hidden_thesis_to_manifest.py`
- Test: `agents/tests/test_runtime_lab_contract.py`
- Test: `agents/tests/test_pixelsmith_hidden_audition.py`
- Test: `agents/tests/test_projectile_pipeline_fixes.py`
- Verify live: `/Users/user/Library/Application Support/Terraria/tModLoader/ModSources/forge_connector_status.json`
- Verify live: `/Users/user/Library/Application Support/Steam/steamapps/common/tModLoader/tModLoader-Logs/client.log`

**Step 1: Run the Python regression slice**

Run:

```bash
pytest -q \
  agents/tests/test_storm_brand_presentation_source_contract.py \
  agents/tests/test_hidden_thesis_to_manifest.py \
  agents/tests/test_runtime_lab_contract.py \
  agents/tests/test_pixelsmith_hidden_audition.py \
  agents/tests/test_projectile_pipeline_fixes.py
```

Expected: PASS

**Step 2: Build the connector**

Run: `dotnet build mod/ForgeConnector/ForgeConnector.csproj -v minimal`

Expected: `Build succeeded.`

**Step 3: Sync the updated connector into live `ModSources` and reinject**

Run the existing live sync + reinject flow used for `StormBrandStaffLive`.

Expected live verification:
- cast flash visible
- firing sound more distinct than generic `Item11`
- projectile emits visible light/trail in flight
- hit burst appears on contact
- target shows visible stack indicator on hit 1 and hit 2
- 3-hit `Starfury` cashout still appears

**Step 4: Commit**

```bash
git add mod/ForgeConnector agents/tests docs/plans
git commit -m "feat: add storm brand runtime presentation polish"
```
