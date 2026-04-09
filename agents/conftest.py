"""Pytest: ensure `agents/` is importable as top-level (orchestrator-style imports)."""

from __future__ import annotations

import sys
from pathlib import Path

_AGENTS = Path(__file__).resolve().parent
if str(_AGENTS) not in sys.path:
    sys.path.insert(0, str(_AGENTS))
