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
