"""
setup/fetch_historical_km.py

ONE-TIME USE: Fetches all club activities from January 1st of the current year
up to (but not including) the current ISO week, and prints the total distance.

Use this to find the correct --annual-total value to pass to init_sheet.py
if you're bootstrapping mid-year.

Usage:
    python setup/fetch_historical_km.py

Optional: restrict to specific sport types (same filter as the main script):
    python setup/fetch_historical_km.py --sport-types Run,Walk
"""

import argparse
import os
import sys
import time
from datetime import date, datetime, timezone

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from dotenv import load_dotenv
load_dotenv()

from src import config
from src.strava_client import STRAVA_API_BASE, refresh_access_token

import requests


def _epoch(d: date) -> int:
    return int(datetime(d.year, d.month, d.day, tzinfo=timezone.utc).timestamp())


def fetch_all_activities(
    access_token: str,
    club_id: str,
    after: int,
    before: int,
) -> list[dict]:
    """
    Fetch club activities and filter client-side to the [after, before) window.

    The club activities endpoint does not support before/after query params,
    so we paginate until we've passed the start of our window.
    """
    headers = {"Authorization": f"Bearer {access_token}"}
    all_activities = []
    page = 1

    after_dt = datetime.utcfromtimestamp(after)
    before_dt = datetime.utcfromtimestamp(before)
    print(f"Fetching activities between "
          f"{after_dt.strftime('%Y-%m-%d')} and "
          f"{before_dt.strftime('%Y-%m-%d')} ...")

    while True:
        resp = requests.get(
            f"{STRAVA_API_BASE}/clubs/{club_id}/activities",
            headers=headers,
            params={"page": page, "per_page": 200},
            timeout=15,
        )
        resp.raise_for_status()
        page_data = resp.json()

        if not page_data:
            break

        for act in page_data:
            start = act.get("start_date") or act.get("start_date_local", "")
            if not start:
                # No date info — include it conservatively
                all_activities.append(act)
                continue

            # start_date is ISO 8601, e.g. "2026-03-05T07:30:00Z"
            act_dt = datetime.strptime(start[:19], "%Y-%m-%dT%H:%M:%S")
            if after_dt <= act_dt < before_dt:
                all_activities.append(act)

        print(f"  Page {page} — {len(all_activities)} matching activities so far")

        if len(page_data) < 200:
            break

        page += 1
        time.sleep(0.5)

    return all_activities


def main():
    parser = argparse.ArgumentParser(
        description="Fetch historical club km from Jan 1 to start of current week."
    )
    parser.add_argument(
        "--sport-types",
        default="",
        help="Comma-separated sport types to count (e.g. Run,Walk). Default: all.",
    )
    args = parser.parse_args()

    if not all([config.STRAVA_CLIENT_ID, config.STRAVA_CLIENT_SECRET,
                 config.STRAVA_REFRESH_TOKEN, config.STRAVA_CLUB_ID]):
        print("ERROR: STRAVA_CLIENT_ID, STRAVA_CLIENT_SECRET, STRAVA_REFRESH_TOKEN "
              "and STRAVA_CLUB_ID must be set in .env")
        sys.exit(1)

    sport_filter = [t.strip() for t in args.sport_types.split(",") if t.strip()]

    today = date.today()
    year_start = date(today.year, 1, 1)
    # Monday of the current week = start of current (incomplete) week
    current_week_monday = today - __import__("datetime").timedelta(days=today.weekday())

    after_epoch = _epoch(year_start)
    before_epoch = _epoch(current_week_monday)

    access_token = refresh_access_token()
    activities = fetch_all_activities(access_token, config.STRAVA_CLUB_ID, after_epoch, before_epoch)

    total_meters = 0.0
    skipped = 0
    for act in activities:
        sport_type = act.get("sport_type") or act.get("type") or ""
        if sport_filter and sport_type not in sport_filter:
            skipped += 1
            continue
        total_meters += act.get("distance", 0.0)

    total_km = total_meters / 1000

    print(f"\n{'='*50}")
    print(f"Total activities fetched : {len(activities)}")
    if sport_filter:
        print(f"Filtered to             : {', '.join(sport_filter)}")
        print(f"Activities skipped      : {skipped}")
    print(f"Total distance          : {total_km:.2f} km")
    print(f"Period                  : {year_start} → {current_week_monday} (exclusive)")
    print(f"{'='*50}")
    print(f"\nTo seed this into your sheet, run:")
    print(f"  python setup/init_sheet.py "
          f"--week {current_week_monday.isocalendar()[1] - 1} "
          f"--annual-total {total_km:.2f}")


if __name__ == "__main__":
    main()
