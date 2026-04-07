import unittest

from architect.models import (
    AMMO_ID_CHOICES,
    BUFF_ID_CHOICES,
    LLMMechanics,
    Mechanics,
    VALID_AMMO_IDS,
    VALID_BUFF_IDS,
)


class IdRegistryInvariantTests(unittest.TestCase):
    def test_buff_ids_single_source(self) -> None:
        self.assertEqual(set(BUFF_ID_CHOICES), set(VALID_BUFF_IDS))
        self.assertEqual(len(BUFF_ID_CHOICES), len(VALID_BUFF_IDS))

    def test_ammo_ids_single_source(self) -> None:
        self.assertEqual(set(AMMO_ID_CHOICES), set(VALID_AMMO_IDS))
        self.assertEqual(len(AMMO_ID_CHOICES), len(VALID_AMMO_IDS))


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


if __name__ == "__main__":
    unittest.main()
