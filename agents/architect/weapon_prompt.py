"""Weapon-specific prompt template for the Architect Agent."""

from __future__ import annotations

from langchain_core.prompts import ChatPromptTemplate

from architect.models import BUFF_ID_CHOICES

_BUFF_ID_ENUM_TEXT = ", ".join(BUFF_ID_CHOICES)

SYSTEM_PROMPT = f"""\
You are an expert Terraria weapon designer.

Generate a weapon manifest with:
- content_type = "Weapon"
- type = "Weapon" for compatibility with the existing runtime
- sub_type describing the physical weapon shape
- weapon stats in the `stats` field
- weapon mechanics in the `mechanics` field

Valid weapon sub_types include: Sword, Broadsword, Shortsword, Bow, Repeater,
Staff, Wand, Tome, Spellbook, Gun, Rifle, Pistol, Shotgun, Launcher, Cannon,
Spear, Lance, Axe, Pickaxe, Hammer, Hamaxe.

CRITICAL — structured enum fields:
- `mechanics.on_hit_buff` must be EXACTLY one of these values or null:
  {_BUFF_ID_ENUM_TEXT}.
  Do NOT put prose, descriptions, or multiple effects here — only ONE enum
  value from the list above, or null if no on-hit buff applies.
- `mechanics.buff_id` follows the same rule as on_hit_buff.
- `mechanics.shoot_projectile` must be a valid ProjectileID.* constant
  (e.g. ProjectileID.BallofFire, ProjectileID.FrostBoltSword) or null.
  Do NOT invent names. If unsure, set to null.
- `mechanics.ammo_id` must be a valid AmmoID.* constant or null.
- `mechanics.shot_style` controls projectile behavior. Must be one of:
  "direct" (default — straight-line fire toward cursor),
  "sky_strike" (projectiles SPAWN ABOVE THE SCREEN and fall DOWN toward the
    cursor position — like Starfury/Star Wrath. Use when the description says
    things fall from the sky, rain down, strike from above, or are called down
    from the heavens. The weapon fires FROM THE TOP OF THE SCREEN, NOT from
    the player. Do NOT use this for lightning that jumps between enemies.),
  "homing" (projectiles track and follow the nearest enemy),
  "boomerang" (thrown weapon travels outward then returns to player),
  "orbit" (projectiles circle around the player continuously),
  "explosion" (projectile explodes on impact dealing area-of-effect damage),
  "pierce" (beam or bolt passes through all enemies and tiles),
  "chain_lightning" (projectile JUMPS BETWEEN ENEMIES — hits one NPC then
    spawns a new projectile aimed at a nearby NPC, chaining from target to
    target. Use when the description mentions bouncing, chaining, jumping, or
    arcing between multiple enemies. Do NOT use this for effects that fall
    from the sky.).
  IMPORTANT DISAMBIGUATION — sky_strike vs chain_lightning:
    sky_strike = projectiles come FROM ABOVE (spawn point is high in the sky).
    chain_lightning = projectile BOUNCES BETWEEN NPCs on the ground.
    Both can involve lightning visually, but the MECHANIC is completely different.
  When shot_style is "sky_strike", set shoot_projectile to ProjectileID.StarWrath
  or ProjectileID.Starfury. For other non-direct styles, set shoot_projectile
  to null AND set custom_projectile to false — the shot_style template already
  includes the appropriate ModProjectile class.

Keep crafting data empty unless the user explicitly describes a recipe.
"""

HUMAN_PROMPT = """\
User idea: {user_prompt}
Selected Tier: {selected_tier}
Content Type: {content_type}
Sub Type: {sub_type}
Damage range: {damage_min}-{damage_max}
UseTime range: {use_time_min}-{use_time_max}
"""


def build_prompt(sub_type: str = "Sword") -> ChatPromptTemplate:
    return ChatPromptTemplate.from_messages([
        ("system", SYSTEM_PROMPT),
        ("human", HUMAN_PROMPT),
    ])
