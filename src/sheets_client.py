"""
Google Sheets client.

Uses a Service Account (no user interaction needed at runtime).

Sheet layout (set up by setup/init_sheet.py):
  A: Week Number   B: Week Start   C: Week End
  D: Weekly KM     E: Annual Total  F: Annual Goal   G: Post Text
"""

import json
import logging
from datetime import date
from typing import Any

import gspread
from google.oauth2.service_account import Credentials

from . import config

logger = logging.getLogger(__name__)

SCOPES = [
    "https://www.googleapis.com/auth/spreadsheets",
]

HEADER_ROW = [
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


def _get_worksheet() -> gspread.Worksheet:
    """Authenticate with the service account and return the first worksheet."""
    sa_info = json.loads(config.GOOGLE_SERVICE_ACCOUNT_JSON)
    creds = Credentials.from_service_account_info(sa_info, scopes=SCOPES)
    gc = gspread.authorize(creds)
    sheet = gc.open_by_key(config.GOOGLE_SHEET_ID)
    return sheet.sheet1


def get_last_annual_total() -> float:
    """
    Read the most recent Annual Total from the sheet.
    Returns 0.0 if the sheet has no data rows yet.
    """
    ws = _get_worksheet()
    all_values = ws.get_all_values()

    data_rows = [row for row in all_values if row and row[0] not in ("", HEADER_ROW[0])]
    if not data_rows:
        logger.info("No existing rows in sheet — starting from 0 km.")
        return 0.0

    last_row = data_rows[-1]
    try:
        total = float(last_row[4])  # Column E
    except (IndexError, ValueError):
        logger.warning("Could not read Annual Total from last row: %s", last_row)
        total = 0.0

    logger.info("Last annual total from sheet: %.2f km", total)
    return total


def has_entry_for_week(week_number: int) -> bool:
    """
    Return True if the sheet already contains a row for this ISO week number.
    Prevents accidental duplicate entries if the script runs twice.
    """
    ws = _get_worksheet()
    col_a = ws.col_values(1)  # Week Number column

    for cell in col_a[1:]:  # skip header
        try:
            if int(cell) == week_number:
                return True
        except ValueError:
            continue

    return False


def append_weekly_entry(
    week_number: int,
    week_start: date,
    week_end: date,
    weekly_km: float,
    annual_km: float,
    post_text: str,
) -> None:
    """Append a new data row for the given week."""
    ws = _get_worksheet()

    row: list[Any] = [
        week_number,
        _fmt(week_start),
        _fmt(week_end),
        round(weekly_km, 2),
        round(annual_km, 2),
        config.ANNUAL_GOAL_KM,
        post_text,
    ]

    ws.append_row(row, value_input_option="USER_ENTERED")
    logger.info("Appended row for week %d to sheet.", week_number)


def ensure_header_exists() -> None:
    """Write the header row if the sheet is completely empty."""
    ws = _get_worksheet()
    first_cell = ws.acell("A1").value

    if not first_cell:
        ws.append_row(HEADER_ROW)
        logger.info("Header row written to sheet.")
