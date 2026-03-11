"""
Central configuration — all values come from environment variables.
Locally, put them in a .env file (copy .env.example).
In GitHub Actions, put them in repository Secrets.
"""

import os
from dotenv import load_dotenv

load_dotenv()


# ── Strava ────────────────────────────────────────────────────────────────────
STRAVA_CLIENT_ID: str = os.getenv("STRAVA_CLIENT_ID", "")
STRAVA_CLIENT_SECRET: str = os.getenv("STRAVA_CLIENT_SECRET", "")
STRAVA_REFRESH_TOKEN: str = os.getenv("STRAVA_REFRESH_TOKEN", "")
STRAVA_CLUB_ID: str = os.getenv("STRAVA_CLUB_ID", "")

# ── Google Sheets ─────────────────────────────────────────────────────────────
# The full JSON content of the service account key file, stored as a secret.
GOOGLE_SERVICE_ACCOUNT_JSON: str = os.getenv("GOOGLE_SERVICE_ACCOUNT_JSON", "")
GOOGLE_SHEET_ID: str = os.getenv("GOOGLE_SHEET_ID", "")

# ── WhatsApp (CallMeBot) ──────────────────────────────────────────────────────
# Phone number with country code, no + sign. E.g. 351912345678
CALLMEBOT_PHONE: str = os.getenv("CALLMEBOT_PHONE", "")
CALLMEBOT_API_KEY: str = os.getenv("CALLMEBOT_API_KEY", "")


_REQUIRED_KEYS = [
    "STRAVA_CLIENT_ID",
    "STRAVA_CLIENT_SECRET",
    "STRAVA_REFRESH_TOKEN",
    "STRAVA_CLUB_ID",
    "GOOGLE_SERVICE_ACCOUNT_JSON",
    "GOOGLE_SHEET_ID",
    "CALLMEBOT_PHONE",
    "CALLMEBOT_API_KEY",
]


def validate() -> None:
    """Call this at the start of a full run to catch missing variables early."""
    missing = [k for k in _REQUIRED_KEYS if not os.getenv(k)]
    if missing:
        raise EnvironmentError(
            f"Missing required environment variables: {', '.join(missing)}\n"
            "Copy .env.example to .env and fill in the values."
        )

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
