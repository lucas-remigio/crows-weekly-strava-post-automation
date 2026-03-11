"""
Strava API client.

Handles token refresh and fetching the club's weekly activity list.
The `GET /clubs/{id}/activities` endpoint returns activities visible to the
authenticated user. Members who have set their activities to "Only Me" will not
appear — this is a Strava API limitation and cannot be worked around.
"""

import time
import logging
from datetime import date, datetime, timezone
from typing import Any

import requests

from . import config

logger = logging.getLogger(__name__)

STRAVA_AUTH_URL = "https://www.strava.com/oauth/token"
STRAVA_API_BASE = "https://www.strava.com/api/v3"


# ── Authentication ────────────────────────────────────────────────────────────

def refresh_access_token() -> str:
    """Exchange the stored refresh token for a fresh access token."""
    resp = requests.post(
        STRAVA_AUTH_URL,
        data={
            "client_id": config.STRAVA_CLIENT_ID,
            "client_secret": config.STRAVA_CLIENT_SECRET,
            "grant_type": "refresh_token",
            "refresh_token": config.STRAVA_REFRESH_TOKEN,
        },
        timeout=15,
    )
    resp.raise_for_status()
    data = resp.json()
    logger.info("Strava token refreshed, expires at %s", data.get("expires_at"))
    return data["access_token"]


# ── Activities ────────────────────────────────────────────────────────────────

def _week_start_epoch(for_date: date) -> int:
    """Return the Unix timestamp of Monday 00:00:00 UTC for the week of `for_date`."""
    monday = for_date - __import__("datetime").timedelta(days=for_date.weekday())
    dt = datetime(monday.year, monday.month, monday.day, tzinfo=timezone.utc)
    return int(dt.timestamp())


def get_club_weekly_activities(
    access_token: str,
    club_id: str,
    for_date: date | None = None,
) -> list[dict[str, Any]]:
    """
    Fetch all club activities recorded from Monday 00:00 UTC of the
    current (or given) week up to now.

    Paginates automatically to handle clubs with many weekly activities.
    """
    if for_date is None:
        for_date = date.today()

    after_epoch = _week_start_epoch(for_date)
    headers = {"Authorization": f"Bearer {access_token}"}
    all_activities: list[dict[str, Any]] = []
    page = 1

    logger.info(
        "Fetching club %s activities after epoch %s (%s)",
        club_id,
        after_epoch,
        datetime.utcfromtimestamp(after_epoch).strftime("%Y-%m-%d"),
    )

    while True:
        resp = requests.get(
            f"{STRAVA_API_BASE}/clubs/{club_id}/activities",
            headers=headers,
            params={"after": after_epoch, "page": page, "per_page": 200},
            timeout=15,
        )
        resp.raise_for_status()
        page_data: list[dict] = resp.json()

        if not page_data:
            break

        all_activities.extend(page_data)
        logger.info("  Page %d — %d activities fetched so far", page, len(all_activities))

        if len(page_data) < 200:
            # Last page
            break

        page += 1
        time.sleep(0.5)  # be polite to the API

    logger.info("Total activities this week: %d", len(all_activities))
    return all_activities


# ── Distance calculation ──────────────────────────────────────────────────────

def sum_weekly_distance_km(activities: list[dict[str, Any]]) -> float:
    """
    Sum distances (meters → km) across all activities.

    If `config.SPORT_TYPES` is non-empty, only activities whose `sport_type`
    (or legacy `type`) matches the list are counted.
    """
    total_meters = 0.0

    for act in activities:
        sport_type = act.get("sport_type") or act.get("type") or ""

        if config.SPORT_TYPES and sport_type not in config.SPORT_TYPES:
            logger.debug(
                "Skipping activity type '%s' (not in SPORT_TYPES filter)", sport_type
            )
            continue

        total_meters += act.get("distance", 0.0)

    total_km = total_meters / 1000
    logger.info(
        "Summed distance: %.2f km (%s)",
        total_km,
        f"filtered to {config.SPORT_TYPES}" if config.SPORT_TYPES else "all types",
    )
    return round(total_km, 2)
