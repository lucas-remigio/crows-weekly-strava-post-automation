"""
main.py — Weekly Strava club post generator.

Run this script every Sunday night (via GitHub Actions cron or manually):
    python -m src.main

What it does:
  1. Fetches all club activities for the current ISO week from Strava.
  2. Sums the total distance (km).
  3. Reads last week's annual total from Google Sheets.
  4. Calculates the new running annual total.
  5. Generates the formatted post text.
  6. Appends a row to the Google Sheet.
  7. Sends the post text to WhatsApp via CallMeBot.
  8. Prints the post text to stdout (for GitHub Actions logs / copy-paste).
"""

import logging
import sys
from datetime import date, timedelta

from . import config
from . import strava_client
from . import sheets_client
from . import telegram_client

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s — %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


# ── Date helpers ──────────────────────────────────────────────────────────────

def get_week_bounds(for_date: date = None) -> tuple[int, date, date]:
    """Return (iso_week_number, monday, sunday) for the given date."""
    if for_date is None:
        for_date = date.today()

    iso_week = for_date.isocalendar()[1]
    monday = for_date - timedelta(days=for_date.weekday())
    sunday = monday + timedelta(days=6)
    return iso_week, monday, sunday


# ── Post text generation ──────────────────────────────────────────────────────

def build_post_text(
    week_number: int,
    weekly_km: float,
    annual_km: float,
    goal_km: int,
    total_weeks: int,
) -> str:
    annual_pct = (annual_km / goal_km * 100) if goal_km else 0
    week_pct = (week_number / total_weeks * 100) if total_weeks else 0
    on_pace_km = (goal_km / total_weeks) * week_number

    lines = [
        f"Semana {week_number}/{total_weeks} ({week_pct:.1f}%)",
        f"",
        f"Total semanal: {weekly_km:.1f} km",
        f"Total anual: {annual_km:.1f} / {goal_km} km ({annual_pct:.1f}%)",
        f"",
        f"Por esta altura devíamos ter feito {on_pace_km:.0f} km",
    ]

    if annual_km >= on_pace_km:
        diff = annual_km - on_pace_km
        lines.append(f"Estamos +{diff:.1f} km acima do ritmo. Muito bom!")
    else:
        diff = on_pace_km - annual_km
        lines.append(f"Estamos -{diff:.1f} km abaixo do ritmo. Vamos lá!")

    return "\n".join(lines)


# ── Main flow ─────────────────────────────────────────────────────────────────

def run(dry_run: bool = False) -> None:
    """
    Execute the full weekly post pipeline.

    Args:
        dry_run: If True, skip writing to Sheets and sending WhatsApp.
                 Useful for testing the Strava fetch and post text locally.
    """
    if not dry_run:
        config.validate()

    today = date.today()
    week_number, week_start, week_end = get_week_bounds(today)

    logger.info(
        "Running for week %d (%s → %s)",
        week_number,
        week_start.isoformat(),
        week_end.isoformat(),
    )

    # ── Guard: already processed this week? ──────────────────────────────────
    if not dry_run and sheets_client.has_entry_for_week(week_number):
        logger.warning(
            "Week %d already exists in the sheet. Exiting to avoid duplicate.",
            week_number,
        )
        sys.exit(0)

    # ── Step 1: Fetch Strava activities ───────────────────────────────────────
    access_token = strava_client.refresh_access_token()
    activities = strava_client.get_club_weekly_activities(
        access_token, config.STRAVA_CLUB_ID, for_date=today
    )
    weekly_km = strava_client.sum_weekly_distance_km(activities)

    # ── Step 2: Read annual total from Sheets ─────────────────────────────────
    last_annual_km = 0.0 if dry_run else sheets_client.get_last_annual_total()
    new_annual_km = round(last_annual_km + weekly_km, 2)

    logger.info(
        "Weekly: %.2f km | Previous annual: %.2f km | New annual: %.2f km",
        weekly_km,
        last_annual_km,
        new_annual_km,
    )

    # ── Step 3: Build post text ───────────────────────────────────────────────
    post_text = build_post_text(
        week_number=week_number,
        weekly_km=weekly_km,
        annual_km=new_annual_km,
        goal_km=config.ANNUAL_GOAL_KM,
        total_weeks=config.TOTAL_WEEKS,
    )

    print("\n" + "=" * 50)
    print("WEEKLY POST TEXT")
    print("=" * 50)
    print(post_text)
    print("=" * 50 + "\n")

    if dry_run:
        logger.info("DRY RUN — skipping Sheets write and Telegram send.")
        return

    # ── Step 4: Write to Google Sheets ────────────────────────────────────────
    sheets_client.ensure_header_exists()
    sheets_client.append_weekly_entry(
        week_number=week_number,
        week_start=week_start,
        week_end=week_end,
        weekly_km=weekly_km,
        annual_km=new_annual_km,
        post_text=post_text,
    )

    # ── Step 5: Send Telegram message ────────────────────────────────────────
    telegram_client.send_message(post_text)

    logger.info("Done.")


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Generate weekly Strava club post.")
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Fetch Strava data and print the post, but skip Sheets and Telegram.",
    )
    args = parser.parse_args()
    run(dry_run=args.dry_run)
