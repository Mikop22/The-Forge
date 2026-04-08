import unittest
from typing import get_args

import pytest

from architect.models import (
    AMMO_ID_CHOICES,
    AMMO_ID_TUPLE,
    AmmoIDLiteral,
    BUFF_ID_CHOICES,
    BUFF_ID_TUPLE,
    BuffIDLiteral,
    LLMMechanics,
    Mechanics,
    SHOT_STYLE_CHOICES,
    VALID_AMMO_IDS,
    VALID_BUFF_IDS,
    _normalize_buff_id,
)
from forge_master.models import ManifestMechanics


class IdRegistryInvariantTests(unittest.TestCase):
    def test_buff_ids_single_source(self) -> None:
        self.assertEqual(set(BUFF_ID_CHOICES), set(VALID_BUFF_IDS))
        self.assertEqual(len(BUFF_ID_CHOICES), len(VALID_BUFF_IDS))

    def test_ammo_ids_single_source(self) -> None:
        self.assertEqual(set(AMMO_ID_CHOICES), set(VALID_AMMO_IDS))
        self.assertEqual(len(AMMO_ID_CHOICES), len(VALID_AMMO_IDS))

    def test_buff_literal_matches_tuple(self) -> None:
        self.assertEqual(set(get_args(BuffIDLiteral)), set(BUFF_ID_TUPLE))

    def test_ammo_literal_matches_tuple(self) -> None:
        self.assertEqual(set(get_args(AmmoIDLiteral)), set(AMMO_ID_TUPLE))

    def test_shot_style_literal_matches_across_models(self) -> None:
        """shot_style Literal in forge_master must match architect's SHOT_STYLE_CHOICES."""
        fm_choices = set(get_args(ManifestMechanics.model_fields["shot_style"].annotation))
        self.assertEqual(fm_choices, set(SHOT_STYLE_CHOICES))


class BuffNormalizationTests(unittest.TestCase):
    def test_on_hit_buff_accepts_sentence_containing_on_fire(self) -> None:
        mechanics = LLMMechanics(
            on_hit_buff="Inflicts a brief random elemental effect: On Fire! or Chilled"
        )
        self.assertEqual(mechanics.on_hit_buff, "BuffID.OnFire")

    def test_on_hit_buff_accepts_canonical_form(self) -> None:
        mechanics = LLMMechanics(on_hit_buff="BuffID.Frostburn")
        self.assertEqual(mechanics.on_hit_buff, "BuffID.Frostburn")

    def test_on_hit_buff_accepts_bare_name(self) -> None:
        mechanics = LLMMechanics(on_hit_buff="Frostburn")
        self.assertEqual(mechanics.on_hit_buff, "BuffID.Frostburn")

    def test_on_hit_buff_accepts_display_alias(self) -> None:
        mechanics = LLMMechanics(on_hit_buff="On Fire!")
        self.assertEqual(mechanics.on_hit_buff, "BuffID.OnFire")

    def test_on_hit_buff_accepts_weak(self) -> None:
        mechanics = LLMMechanics(on_hit_buff="Weak")
        self.assertEqual(mechanics.on_hit_buff, "BuffID.Weak")

    def test_on_hit_buff_accepts_buffid_weak(self) -> None:
        mechanics = LLMMechanics(on_hit_buff="BuffID.Weak")
        self.assertEqual(mechanics.on_hit_buff, "BuffID.Weak")

    def test_on_hit_buff_prose_with_frostburn(self) -> None:
        mechanics = LLMMechanics(
            on_hit_buff="Applies Frostburn to enemies on contact"
        )
        self.assertEqual(mechanics.on_hit_buff, "BuffID.Frostburn")

    def test_on_hit_buff_prose_with_cursed_inferno(self) -> None:
        mechanics = LLMMechanics(
            on_hit_buff="Inflicts Cursed Inferno debuff for 3 seconds"
        )
        self.assertEqual(mechanics.on_hit_buff, "BuffID.CursedInferno")

    def test_on_hit_buff_prose_with_shadow_flame(self) -> None:
        mechanics = LLMMechanics(
            on_hit_buff="Has a chance to inflict Shadow Flame on hit"
        )
        self.assertEqual(mechanics.on_hit_buff, "BuffID.ShadowFlame")

    def test_on_hit_buff_null_passthrough(self) -> None:
        mechanics = LLMMechanics(on_hit_buff=None)
        self.assertIsNone(mechanics.on_hit_buff)

    def test_on_hit_buff_empty_string_becomes_none(self) -> None:
        mechanics = LLMMechanics(on_hit_buff="")
        self.assertIsNone(mechanics.on_hit_buff)

    def test_on_hit_buff_total_garbage_falls_back_to_none(self) -> None:
        """Prose with no recognizable buff should fall back to None, not crash."""
        mechanics = LLMMechanics(
            on_hit_buff="Makes the enemy feel slightly uncomfortable"
        )
        self.assertIsNone(mechanics.on_hit_buff)

    def test_on_hit_buff_burning_prose_maps_to_on_fire(self) -> None:
        mechanics = LLMMechanics(
            on_hit_buff="On hit, inflicts a brief burning effect"
        )
        self.assertEqual(mechanics.on_hit_buff, "BuffID.OnFire")

    def test_buff_id_accepts_canonical(self) -> None:
        mechanics = LLMMechanics(buff_id="BuffID.WellFed")
        self.assertEqual(mechanics.buff_id, "BuffID.WellFed")

    def test_buff_id_prose_maps_like_on_hit(self) -> None:
        mechanics = LLMMechanics(buff_id="Grants Well Fed when used")
        self.assertEqual(mechanics.buff_id, "BuffID.WellFed")

    def test_on_hit_buff_unknown_buffid_constant_becomes_none(self) -> None:
        mechanics = LLMMechanics(on_hit_buff="BuffID.Chilled")
        self.assertIsNone(mechanics.on_hit_buff)


class MechanicsBuffNormalizationTests(unittest.TestCase):
    """``Mechanics`` duplicates buff validators — mirror key ``LLMMechanics`` cases."""

    def _base(self, **mech_kwargs):
        return Mechanics(
            crafting_material="ItemID.Wood",
            crafting_cost=5,
            crafting_tile="TileID.WorkBenches",
            **mech_kwargs,
        )

    def test_prose_frostburn_matches_llm_mechanics(self) -> None:
        m = self._base(on_hit_buff="Applies Frostburn on hit")
        self.assertEqual(m.on_hit_buff, "BuffID.Frostburn")

    def test_invalid_buffid_string_becomes_none(self) -> None:
        m = self._base(buff_id="BuffID.Venom")
        self.assertIsNone(m.buff_id)


@pytest.mark.parametrize("raw,expected", [
    # Canonical forms — unchanged
    ("BuffID.OnFire",       "BuffID.OnFire"),
    ("BuffID.Frostburn",    "BuffID.Frostburn"),
    # Exact alias — existing coverage
    ("On Fire!",            "BuffID.OnFire"),
    ("On Fire",             "BuffID.OnFire"),
    # Case variants — newly hardened
    ("on fire!",            "BuffID.OnFire"),
    ("ON FIRE!",            "BuffID.OnFire"),
    ("on Fire",             "BuffID.OnFire"),
    ("POISONED",            "BuffID.Poisoned"),
    ("poisoned",            "BuffID.Poisoned"),
    ("SLIMED",              "BuffID.Slimed"),
    ("well fed",            "BuffID.WellFed"),
    ("WELL FED",            "BuffID.WellFed"),
    ("cursed inferno",      "BuffID.CursedInferno"),
    ("CURSED INFERNO",      "BuffID.CursedInferno"),
    ("shadow flame",        "BuffID.ShadowFlame"),
    ("shadowflame",         "BuffID.ShadowFlame"),
    ("SHADOWFLAME",         "BuffID.ShadowFlame"),
    ("frostburn",           "BuffID.Frostburn"),
    ("FROSTBURN",           "BuffID.Frostburn"),
    # Completely unrecognised → None (no crash)
    ("completely_bogus",    None),
    ("BuffID.Chilled",      None),
])
def test_normalize_buff_id_variants(raw: str, expected):
    assert _normalize_buff_id(raw) == expected


class ShotStyleTests(unittest.TestCase):
    """shot_style field on LLMMechanics and Mechanics."""

    def test_llm_mechanics_defaults_to_direct(self) -> None:
        m = LLMMechanics()
        self.assertEqual(m.shot_style, "direct")

    def test_llm_mechanics_accepts_sky_strike(self) -> None:
        m = LLMMechanics(shot_style="sky_strike")
        self.assertEqual(m.shot_style, "sky_strike")

    def test_mechanics_defaults_to_direct(self) -> None:
        m = Mechanics(
            crafting_material="ItemID.Wood",
            crafting_cost=5,
            crafting_tile="TileID.WorkBenches",
        )
        self.assertEqual(m.shot_style, "direct")

    def test_mechanics_accepts_sky_strike(self) -> None:
        m = Mechanics(
            crafting_material="ItemID.Wood",
            crafting_cost=5,
            crafting_tile="TileID.WorkBenches",
            shot_style="sky_strike",
        )
        self.assertEqual(m.shot_style, "sky_strike")

    def test_llm_mechanics_accepts_all_styles(self) -> None:
        for style in ("homing", "boomerang", "orbit", "explosion", "pierce", "chain_lightning"):
            m = LLMMechanics(shot_style=style)
            self.assertEqual(m.shot_style, style)

    def test_mechanics_accepts_all_styles(self) -> None:
        for style in ("homing", "boomerang", "orbit", "explosion", "pierce", "chain_lightning"):
            m = Mechanics(
                crafting_material="ItemID.Wood",
                crafting_cost=5,
                crafting_tile="TileID.WorkBenches",
                shot_style=style,
            )
            self.assertEqual(m.shot_style, style)

    def test_llm_mechanics_rejects_invalid_shot_style(self) -> None:
        from pydantic import ValidationError
        with self.assertRaises(ValidationError):
            LLMMechanics(shot_style="orbital")


if __name__ == "__main__":
    unittest.main()
