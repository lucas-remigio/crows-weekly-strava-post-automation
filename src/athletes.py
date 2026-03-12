"""
Loads the club athlete roster from athletes.json at the project root.

The file is excluded from git (matched by *.json in .gitignore) so each
deployment can have its own private list. Copy athletes.json.example to
athletes.json and fill in your real club members before running.
"""

import json
import logging
from pathlib import Path

logger = logging.getLogger(__name__)

_ATHLETES_FILE = Path(__file__).parent.parent / "athletes.json"


def _load() -> list[dict[str, str]]:
    if not _ATHLETES_FILE.exists():
        logger.warning(
            "athletes.json not found at %s — weekly roast will be skipped. "
            "Copy athletes.json.example to athletes.json to enable it.",
            _ATHLETES_FILE,
        )
        return []
    with _ATHLETES_FILE.open(encoding="utf-8") as fh:
        return json.load(fh)


ATHLETES: list[dict[str, str]] = _load()
