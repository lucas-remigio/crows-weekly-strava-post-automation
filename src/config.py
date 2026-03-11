"""
Central configuration — all values come from environment variables.
Locally, put them in a .env file (copy .env.example).
In GitHub Actions, put them in repository Secrets.
"""

import os
from dotenv import load_dotenv

load_dotenv()


def _require(key: str) -> str:
    val = os.getenv(key)
    if not val:
        raise EnvironmentError(
            f"Required environment variable '{key}' is missing. "
            "Copy .env.example to .env and fill in the values."
        )
    return val


# ── Strava ────────────────────────────────────────────────────────────────────
STRAVA_CLIENT_ID: str = _require("STRAVA_CLIENT_ID")
STRAVA_CLIENT_SECRET: str = _require("STRAVA_CLIENT_SECRET")
STRAVA_REFRESH_TOKEN: str = _require("STRAVA_REFRESH_TOKEN")
STRAVA_CLUB_ID: str = _require("STRAVA_CLUB_ID")

# ── Google Sheets ─────────────────────────────────────────────────────────────
# The full JSON content of the service account key file, stored as a secret.
GOOGLE_SERVICE_ACCOUNT_JSON: str = _require("GOOGLE_SERVICE_ACCOUNT_JSON")
GOOGLE_SHEET_ID: str = _require("GOOGLE_SHEET_ID")

# ── WhatsApp (CallMeBot) ──────────────────────────────────────────────────────
# Phone number with country code, no + sign. E.g. 351912345678
CALLMEBOT_PHONE: str = _require("CALLMEBOT_PHONE")
CALLMEBOT_API_KEY: str = _require("CALLMEBOT_API_KEY")

# ── Goal / Formatting ─────────────────────────────────────────────────────────
ANNUAL_GOAL_KM: int = int(os.getenv("ANNUAL_GOAL_KM", "12000"))
TOTAL_WEEKS: int = int(os.getenv("TOTAL_WEEKS", "52"))

# Which sport types to count. Empty list means ALL types.
# Examples: ["Run", "Walk", "Hike", "VirtualRun"]
SPORT_TYPES: list[str] = [
    t.strip()
    for t in os.getenv("SPORT_TYPES", "").split(",")
    if t.strip()
]
