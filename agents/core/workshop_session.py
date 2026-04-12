"""Persistence helpers for Forge Director workshop sessions."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from core.atomic_io import atomic_write_text


class WorkshopSessionStore:
    def __init__(self, root: Path) -> None:
        self._root = Path(root)
        self._active_path = self._root / "active_session.txt"

    def _session_path(self, session_id: str) -> Path:
        cleaned = session_id.strip()
        if not cleaned:
            raise ValueError("session_id is required")
        return self._root / f"{cleaned}.json"

    def save(self, payload: dict[str, Any]) -> None:
        session_id = str(payload.get("session_id", "")).strip()
        path = self._session_path(session_id)
        text = json.dumps(payload, indent=2, sort_keys=True) + "\n"
        atomic_write_text(path, text)
        atomic_write_text(self._active_path, session_id + "\n")

    def load(self, session_id: str) -> dict[str, Any]:
        path = self._session_path(session_id)
        if not path.exists():
            return {}
        return json.loads(path.read_text(encoding="utf-8"))

    def active_session_id(self) -> str | None:
        try:
            session_id = self._active_path.read_text(encoding="utf-8").strip()
        except OSError:
            return None
        return session_id or None

    def load_active(self) -> dict[str, Any]:
        session_id = self.active_session_id()
        if not session_id:
            return {}
        return self.load(session_id)
