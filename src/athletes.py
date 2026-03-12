"""
Loads the club athlete roster from athletes.json at the project root,
or from the ATHLETES_JSON environment variable (used in GitHub Actions
where the file is not committed to the repo).

Locally: copy athletes.json.example to athletes.json and fill in your members.
CI: store the JSON content as the ATHLETES_JSON GitHub Actions Secret.
"""

import json
import logging
import os
from pathlib import Path

logger = logging.getLogger(__name__)

_ATHLETES_FILE = Path(__file__).parent.parent / "athletes.json"


def _load() -> list[dict[str, str]]:
    # Prefer the file (local dev); fall back to env var (CI/Actions).
    if _ATHLETES_FILE.exists():
        with _ATHLETES_FILE.open(encoding="utf-8") as fh:
            return json.load(fh)

    env_json = os.getenv("ATHLETES_JSON", "")
    if env_json:
        return json.loads(env_json)

    logger.warning(
        "No athletes roster found (checked %s and ATHLETES_JSON env var) "
        "— weekly roast will be skipped. "
        "Copy athletes.json.example to athletes.json to enable it.",
        _ATHLETES_FILE,
    )
    return []


ATHLETES: list[dict[str, str]] = _load()
