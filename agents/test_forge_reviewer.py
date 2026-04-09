import pytest
from unittest.mock import patch, MagicMock

import sys
from pathlib import Path
_AGENTS_DIR = Path(__file__).resolve().parent
if str(_AGENTS_DIR) not in sys.path:
    sys.path.insert(0, str(_AGENTS_DIR))

from forge_master.reviewer import WeaponReviewer, ReviewOutput, ReviewIssue


def _make_reviewer() -> WeaponReviewer:
    """Construct a WeaponReviewer without hitting the OpenAI API key check."""
    with patch("forge_master.reviewer.ChatOpenAI", return_value=MagicMock()):
        return WeaponReviewer(model_name="gpt-4o-mini")


def test_reviewer_approves_good_code():
    reviewer = _make_reviewer()

    mock_review_output = ReviewOutput(
        approved=True,
        issues=[],
        summary="Looks good!"
    )
    reviewer._review_chain = MagicMock()
    reviewer._review_chain.invoke.return_value = mock_review_output

    manifest = {
        "item_name": "TestSword",
        "mechanics": {"shot_style": "direct", "custom_projectile": False}
    }
    cs_code = "class TestSword {}"

    new_code, final_review = reviewer.review(manifest, cs_code)

    assert final_review.approved is True
    assert final_review.summary == "Looks good!"
    assert new_code == cs_code
    reviewer._review_chain.invoke.assert_called_once()


def test_reviewer_fixes_bad_code():
    reviewer = _make_reviewer()

    reviewer._review_chain = MagicMock()
    reviewer._fix_chain = MagicMock()

    bad_review = ReviewOutput(
        approved=False,
        issues=[
            ReviewIssue(
                severity="critical",
                category="shot_style",
                description="Missing Item.channel",
                suggested_fix="Add Item.channel=true to SetDefaults"
            )
        ],
        summary="Missing channel"
    )

    good_review = ReviewOutput(
        approved=True,
        issues=[],
        summary="Fixed"
    )

    # First call returns bad, second call returns good
    reviewer._review_chain.invoke.side_effect = [bad_review, good_review]
    reviewer._fix_chain.invoke.return_value = "class FixedTestSword {}"

    manifest = {
        "item_name": "TestSword",
        "mechanics": {"shot_style": "channeled"}
    }
    cs_code = "class TestSword {}"

    new_code, final_review = reviewer.review(manifest, cs_code)

    assert new_code == "class FixedTestSword {}"
    assert final_review.approved is True
    assert reviewer._review_chain.invoke.call_count == 2
    assert reviewer._fix_chain.invoke.call_count == 1


def test_reviewer_info_only_issues_approved():
    """Info-only issues should approve without mutating the original ReviewOutput."""
    reviewer = _make_reviewer()

    info_review = ReviewOutput(
        approved=False,
        issues=[
            ReviewIssue(
                severity="info",
                category="style",
                description="Could use a helper method",
                suggested_fix="Extract method"
            )
        ],
        summary="Only style issues"
    )
    reviewer._review_chain = MagicMock()
    reviewer._review_chain.invoke.return_value = info_review

    manifest = {"item_name": "TestSword", "mechanics": {"shot_style": "direct"}}
    _, final_review = reviewer.review(manifest, "class TestSword {}")

    assert final_review.approved is True
    # Original LLM result should not have been mutated
    assert info_review.approved is False
