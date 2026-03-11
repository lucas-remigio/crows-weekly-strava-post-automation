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

# ── Telegram ──────────────────────────────────────────────────────────────────
# Bot token from @BotFather.
TELEGRAM_BOT_TOKEN: str = os.getenv("TELEGRAM_BOT_TOKEN", "")

# Comma-separated list of chat IDs to send the post to.
# Group IDs are negative (e.g. -123456789); personal chats are positive.
# E.g.: -123456789,987654321
TELEGRAM_CHAT_IDS: list[str] = [
    cid.strip()
    for cid in os.getenv("TELEGRAM_CHAT_IDS", "").split(",")
    if cid.strip()
]


_REQUIRED_KEYS = [
    "STRAVA_CLIENT_ID",
    "STRAVA_CLIENT_SECRET",
    "STRAVA_REFRESH_TOKEN",
    "STRAVA_CLUB_ID",
    "GOOGLE_SERVICE_ACCOUNT_JSON",
    "GOOGLE_SHEET_ID",
    "TELEGRAM_BOT_TOKEN",
    "TELEGRAM_CHAT_IDS",
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
