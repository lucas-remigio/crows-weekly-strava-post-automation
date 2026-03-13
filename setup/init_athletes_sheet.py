"""
setup/init_athletes_sheet.py

Run this once to create the athletes table on a second worksheet tab.

Usage:
    python setup/init_athletes_sheet.py

Optional:
    python setup/init_athletes_sheet.py --force-header
    python setup/init_athletes_sheet.py --no-seed

Requirements:
    pip install -r requirements.txt
    GOOGLE_SERVICE_ACCOUNT_JSON and GOOGLE_SHEET_ID must be set in .env
"""

import argparse
import json
from pathlib import Path

from gspread.exceptions import WorksheetNotFound

from _bootstrap import bootstrap_setup

bootstrap_setup()

from src import config
from src.sheets_client import apply_header_style, get_worksheet

ATHLETES_HEADER = ["Nome", "Caracteristica"]
WORKSHEET_TITLE = "Atletas"
DEFAULT_ROWS = 300


def _ensure_tab():
    spreadsheet = get_worksheet().spreadsheet

    try:
        ws = spreadsheet.worksheet(WORKSHEET_TITLE)
        created = False
    except WorksheetNotFound:
        ws = spreadsheet.add_worksheet(
            title=WORKSHEET_TITLE,
            rows=DEFAULT_ROWS,
            cols=len(ATHLETES_HEADER),
        )
        created = True

    return ws, created


def _ensure_header(ws, force_header: bool) -> None:
    first_row = ws.row_values(1)

    has_header = first_row and any(cell.strip() for cell in first_row)
    if not has_header or force_header:
        ws.update(
            range_name="A1:B1",
            values=[ATHLETES_HEADER],
            value_input_option="USER_ENTERED",
        )

    # Always apply styling so existing tabs can be reformatted safely.
    apply_header_style(ws, len(ATHLETES_HEADER))


def _ensure_characteristic_multiline(ws) -> None:
    ws.format(
        "B:B",
        {
            "wrapStrategy": "WRAP",
            "verticalAlignment": "TOP",
        },
    )


def _autosize_rows(ws) -> None:
    ws.spreadsheet.batch_update(
        {
            "requests": [
                {
                    "autoResizeDimensions": {
                        "dimensions": {
                            "sheetId": ws.id,
                            "dimension": "ROWS",
                            "startIndex": 1,
                            "endIndex": ws.row_count,
                        }
                    }
                }
            ]
        }
    )


def _validate_required_google_config() -> None:
    missing = []
    if not config.GOOGLE_SERVICE_ACCOUNT_JSON:
        missing.append("GOOGLE_SERVICE_ACCOUNT_JSON")
    if not config.GOOGLE_SHEET_ID:
        missing.append("GOOGLE_SHEET_ID")

    if missing:
        raise EnvironmentError(
            "Missing required environment variables: " + ", ".join(missing)
        )


def _load_athletes_json_rows() -> list[list[str]]:
    athletes_file = Path(__file__).parent.parent / "athletes.json"
    if not athletes_file.exists():
        return []

    with athletes_file.open(encoding="utf-8") as fh:
        data = json.load(fh)

    if not isinstance(data, list):
        raise ValueError("athletes.json must contain a list of athlete objects")

    rows: list[list[str]] = []
    for item in data:
        if not isinstance(item, dict):
            continue
        name = str(item.get("name", "")).strip()
        characteristic = str(item.get("characteristic", "")).strip()
        if not name:
            continue
        rows.append([name, characteristic])

    return rows


def _has_data_rows(ws) -> bool:
    values = ws.get_all_values()
    if len(values) <= 1:
        return False

    for row in values[1:]:
        if any(cell.strip() for cell in row):
            return True
    return False


def _seed_from_json_if_empty(ws) -> int:
    if _has_data_rows(ws):
        return 0

    rows = _load_athletes_json_rows()
    if not rows:
        return 0

    ws.append_rows(rows, value_input_option="USER_ENTERED")
    return len(rows)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Create the athletes table in the 'Atletas' Google Sheets tab."
    )
    parser.add_argument(
        "--force-header",
        action="store_true",
        help="Rewrite row 1 with the expected athletes header.",
    )
    parser.add_argument(
        "--no-seed",
        action="store_true",
        help="Do not seed athletes from athletes.json.",
    )
    args = parser.parse_args()

    _validate_required_google_config()

    ws, created = _ensure_tab()
    _ensure_header(ws, force_header=args.force_header)
    _ensure_characteristic_multiline(ws)
    _autosize_rows(ws)
    seeded_count = 0
    if not args.no_seed:
        seeded_count = _seed_from_json_if_empty(ws)

    print("Done.")
    print(f"Worksheet: {ws.title}")
    print(f"Created new tab: {'yes' if created else 'no'}")
    print(f"Header: {', '.join(ATHLETES_HEADER)}")
    if args.no_seed:
        print("Seeding: skipped by --no-seed")
    elif seeded_count:
        print(f"Seeding: inserted {seeded_count} athletes from athletes.json")
    else:
        print("Seeding: nothing inserted (athletes.json missing/empty or tab already had data)")


if __name__ == "__main__":
    main()
