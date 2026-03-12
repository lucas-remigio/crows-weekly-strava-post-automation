# Copilot Workspace Instructions

## Project Overview

Python automation that runs weekly via GitHub Actions: fetches Strava club activities, accumulates an annual km total in Google Sheets, builds a formatted post, and delivers it via Telegram.

**Entry point:** `python -m src.main` (or `--dry-run` to skip writes/sends)

---

## Architecture

```
src/
  config.py          # Single source of truth for all env vars. Import config.* everywhere.
  strava_client.py   # Token refresh + paginated club activities fetch + distance sum
  sheets_client.py   # Google Sheets read/write via service account
  telegram_client.py # Telegram Bot API delivery (multi-chat)
  main.py            # Pipeline orchestration + post text generation

setup/               # One-time scripts. Import from src.* — never duplicate logic here.
  get_strava_token.py    # OAuth flow to obtain refresh token
  init_sheet.py          # Bootstrap sheet header and optional seed row
  fetch_historical_km.py # Sum historical km to find the correct annual-total seed value
```

The pipeline in `main.py::run()` is the single canonical flow:

1. Validate config → guard duplicate week → fetch Strava → sum km → read sheet total → build post → write sheet → send Telegram.

---

## Commands

```bash
# Install
pip install -r requirements.txt
cp .env.example .env   # fill in secrets

# Run
python -m src.main             # full run
python -m src.main --dry-run   # fetch + print only, skip sheet write & Telegram

# Setup (one-time)
python setup/get_strava_token.py
python setup/init_sheet.py [--week N --annual-total X]
python setup/fetch_historical_km.py [--sport-types Run,Walk]
```

No test suite exists yet. Validate changes with `--dry-run`.

---

## Code Quality Principles

This codebase prioritises **simplicity, readability, and maintainability** above all. Write like a senior engineer:

- **DRY** — Never duplicate logic between `src/` and `setup/`. Setup scripts must import from `src.*`.
- **SOLID (Single Responsibility)** — Each module owns one concern. `config.py` reads env vars; clients perform I/O; `main.py` orchestrates. Do not mix concerns.
- **Thin functions** — One function = one job. If a function needs a comment explaining what a section does, split it.
- **No defensive over-engineering** — Don't add error handling, retries, or fallbacks for scenarios that can't occur at a given call site. Validate only at system boundaries (env on startup via `config.validate()`, at HTTP edges with `resp.raise_for_status()`).
- **No premature abstractions** — Don't create helpers, base classes, or protocols unless the same logic appears in ≥2 places.
- **Readable names over comments** — Prefer a clear function/variable name to an explanatory comment.

---

## Conventions

### Configuration

- All env vars live in `src/config.py`. Never call `os.getenv()` outside that file.
- `config.validate()` is called once at the start of a full run — it is the only place that checks for missing variables.

### External I/O

- All HTTP calls use `timeout=15` and `resp.raise_for_status()`.
- Google Sheets auth always goes through `sheets_client.get_worksheet()` — never re-implement credential loading.
- Strava token refresh always goes through `strava_client.refresh_access_token()`.

### Shared utilities in `sheets_client.py`

- `get_worksheet()` — authenticates and returns the sheet (use this everywhere)
- `fmt_date(d)` — formats a `date` as `dd-mm-yyyy`
- `ensure_header_exists()` — idempotent header writer
- `HEADER_ROW` — single source of truth for column names

### Logging

- Use `logging.getLogger(__name__)` in every module. Never use bare `print()` in `src/`.
- Setup scripts may use `print()` since they are interactive CLI tools.

### Date handling

- Week bounds (ISO week number, monday, sunday) are computed in `main.py::get_week_bounds()`.
- Epoch conversion for Strava's `after=` param lives in `strava_client._week_start_epoch()`.

### Post text

- All copy/formatting is in `main.py::build_post_text()`. Keep Portuguese copy there.

---

## Adding a New Delivery Channel

Pattern: create `src/<channel>_client.py` with a single `send_message(message: str) -> None` public function. Import and call it in `main.py::run()` after the Telegram call. Register any new env vars in `config.py` and add them to `_REQUIRED_KEYS`.

---

## Anti-Patterns to Avoid

- **Reimplementing `get_worksheet()`, `fmt_date()`, or `HEADER_ROW`** in setup scripts — import them.
- **Reading env vars directly** with `os.environ.get()` outside `config.py`.
- **Hardcoding API URLs** — use the constants in the relevant client (`STRAVA_AUTH_URL`, `STRAVA_API_BASE`).
- **Adding new columns** to the sheet without updating `HEADER_ROW` and `append_weekly_entry()` together.
- **Catching broad exceptions** that hide real problems — let unexpected errors propagate unless you have a deliberate per-recipient fault-isolation reason (see `telegram_client.send_message`).
- **Unnecessary abstractions** — a class is not needed anywhere in this codebase today.
