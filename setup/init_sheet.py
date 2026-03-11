"""
setup/init_sheet.py

Run this ONCE to bootstrap your Google Sheet with the correct header row.
Optionally, it can seed a starting row if you already have a running total
from before the automation (i.e., your brother's manually accumulated sum).

Usage:
    # Just create the header:
    python setup/init_sheet.py

    # Also seed the running total so far (e.g., after week 5 with 670 km):
    python setup/init_sheet.py --week 5 --annual-total 670

Requirements:
    pip install -r requirements.txt
    GOOGLE_SERVICE_ACCOUNT_JSON and GOOGLE_SHEET_ID must be set in .env
"""

import argparse
import json
import os
import sys
from datetime import date, timedelta

# Allow running from the repo root
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from dotenv import load_dotenv
load_dotenv()

import gspread
from google.oauth2.service_account import Credentials


def get_worksheet():
    from src import config
    sa_info = json.loads(config.GOOGLE_SERVICE_ACCOUNT_JSON)
    creds = Credentials.from_service_account_info(
        sa_info,
        scopes=["https://www.googleapis.com/auth/spreadsheets"],
    )
    gc = gspread.authorize(creds)
    return gc.open_by_key(config.GOOGLE_SHEET_ID).sheet1


HEADER = [
    "Semana",
    "Início da Semana",
    "Fim da Semana",
    "KM Semanal",
    "Total Anual KM",
    "Objetivo Anual KM",
    "Texto do Post",
]


def _fmt(d: date) -> str:
    """Format a date as dd-mm-yyyy."""
    return d.strftime("%d-%m-%Y")


def week_bounds(week_number: int, year: int = None) -> tuple[date, date]:
    """Return (monday, sunday) for ISO week `week_number` in `year`."""
    if year is None:
        year = date.today().year
    monday = date.fromisocalendar(year, week_number, 1)
    sunday = monday + timedelta(days=6)
    return monday, sunday


def main():
    parser = argparse.ArgumentParser(
        description="Bootstrap the Google Sheet for the Strava weekly tracker."
    )
    parser.add_argument(
        "--week",
        type=int,
        default=None,
        help="ISO week number of the most recent completed week (to seed a starting total).",
    )
    parser.add_argument(
        "--annual-total",
        type=float,
        default=None,
        dest="annual_total",
        help="Running annual total (km) at the end of --week.",
    )
    args = parser.parse_args()

    ws = get_worksheet()

    # Check if sheet is empty (filter out blank rows left by manual deletes)
    existing = [row for row in ws.get_all_values() if any(cell.strip() for cell in row)]
    if existing:
        print(f"Sheet already has {len(existing)} row(s). Header will be skipped.")
    else:
        ws.append_row(HEADER)
        print("Header row written.")

    if args.week is not None and args.annual_total is not None:
        from src import config
        monday, sunday = week_bounds(args.week)
        row = [
            args.week,
            _fmt(monday),
            _fmt(sunday),
            "",  # KM semanal desconhecido na linha inicial
            round(args.annual_total, 2),
            config.ANNUAL_GOAL_KM,
            f"[Linha inicial — entrada manual até à semana {args.week}]",
        ]
        ws.append_row(row, value_input_option="USER_ENTERED")
        print(
            f"Seed row written: week {args.week}, annual total {args.annual_total} km."
        )
    elif args.week is not None or args.annual_total is not None:
        print("WARNING: Provide both --week and --annual-total to seed a starting row.")

    print("\nDone. Your Google Sheet is ready.")
    print("Make sure the sheet is shared (edit access) with your service account email.")


if __name__ == "__main__":
    main()
