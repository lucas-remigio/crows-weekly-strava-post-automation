"""Loads the club athlete roster from a dedicated Google Sheets tab."""

import logging

from gspread.exceptions import WorksheetNotFound

from . import config
from .sheets_client import get_worksheet

logger = logging.getLogger(__name__)

ATHLETES_WORKSHEET_TITLE = "Atletas"


def _index(headers: list[str], options: list[str]) -> int | None:
    for idx, header in enumerate(headers):
        if header in options:
            return idx
    return None


def get_athletes() -> list[dict[str, str]]:
    if not config.GOOGLE_SERVICE_ACCOUNT_JSON or not config.GOOGLE_SHEET_ID:
        logger.warning(
            "Google Sheets config missing for athletes roster "
            "(GOOGLE_SERVICE_ACCOUNT_JSON/GOOGLE_SHEET_ID)."
        )
        return []

    spreadsheet = get_worksheet().spreadsheet
    try:
        ws = spreadsheet.worksheet(ATHLETES_WORKSHEET_TITLE)
    except WorksheetNotFound:
        logger.warning(
            "Athletes worksheet '%s' not found — weekly roast will be skipped.",
            ATHLETES_WORKSHEET_TITLE,
        )
        return []

    values = ws.get_all_values()
    if not values:
        logger.warning(
            "Athletes worksheet '%s' is empty — weekly roast will be skipped.",
            ATHLETES_WORKSHEET_TITLE,
        )
        return []

    headers = [h.strip().lower() for h in values[0]]
    name_idx = _index(headers, ["nome", "name"])
    characteristic_idx = _index(headers, ["caracteristica", "característica", "characteristic"])

    if name_idx is None or characteristic_idx is None:
        logger.warning(
            "Athletes worksheet '%s' must contain 'Nome' and 'Caracteristica' columns.",
            ATHLETES_WORKSHEET_TITLE,
        )
        return []

    athletes: list[dict[str, str]] = []
    for row in values[1:]:
        name = row[name_idx].strip() if len(row) > name_idx else ""
        characteristic = row[characteristic_idx].strip() if len(row) > characteristic_idx else ""
        if not name:
            continue
        athletes.append({"name": name, "characteristic": characteristic})

    if not athletes:
        logger.warning(
            "Athletes worksheet '%s' has no valid rows — weekly roast will be skipped.",
            ATHLETES_WORKSHEET_TITLE,
        )

    return athletes
