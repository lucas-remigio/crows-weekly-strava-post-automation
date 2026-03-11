# Strava Crows Weekly Post

Automates the weekly Strava club progress post.

Every Sunday night a GitHub Action:

1. Fetches all club activities for the week from the Strava API
2. Sums the total distance
3. Adds it to a running annual total stored in Google Sheets
4. Generates a formatted post text
5. Sends it to a WhatsApp group via CallMeBot

Your only manual step: **copy the text and paste it into the Strava club post**.

---

## Setup (do this once)

### 1. Clone and install

```bash
git clone <your-repo>
cd strava_crows_weekly_post
pip install -r requirements.txt
cp .env.example .env
```

---

### 2. Create a Strava API application

1. Go to [strava.com/settings/api](https://www.strava.com/settings/api) and create an app.
2. Set **Authorization Callback Domain** to `localhost`.
3. Copy your **Client ID** and **Client Secret** into `.env`.

---

### 3. Get your Strava refresh token

```bash
python setup/get_strava_token.py
```

A browser tab opens. Authorize the app. The script prints your `STRAVA_REFRESH_TOKEN`. Copy it into `.env`.

> The script requests `activity:read` and `club:read` scopes.

---

### 4. Find your Club ID

Go to your club on Strava. The URL looks like:

```
https://www.strava.com/clubs/123456
```

`123456` is your `STRAVA_CLUB_ID`.

---

### 5. Set up Google Sheets

#### 5a. Create a Google Cloud project & service account

1. Go to [console.cloud.google.com](https://console.cloud.google.com).
2. Create a new project (e.g., `strava-weekly-post`).
3. Enable the **Google Sheets API** for the project.
4. Go to **IAM & Admin → Service Accounts** → Create a service account.
5. On the service account page, go to **Keys → Add Key → JSON**. Download the file.
6. Open the downloaded JSON file and paste its **entire contents** (on one line) as your `GOOGLE_SERVICE_ACCOUNT_JSON` value.

> Tip on one-lining the JSON: `cat key.json | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin)))"`

#### 5b. Create the Google Sheet

1. Create a blank Google Sheet at [sheets.google.com](https://sheets.google.com).
2. Copy its ID from the URL into `GOOGLE_SHEET_ID`.
3. Click **Share** and grant **Editor** access to the service account email (found in the JSON under `"client_email"`).

#### 5c. Bootstrap the sheet

```bash
# Just create the header row:
python setup/init_sheet.py

# OR, if you already have a running total (e.g., Week 5, 670 km):
python setup/init_sheet.py --week 5 --annual-total 670
```

---

### 6. Set up WhatsApp (CallMeBot)

1. Add **+34 644 59 78 12** to your WhatsApp contacts as "CallMeBot".
2. Send it this exact message: `I allow callmebot to send me messages`
3. You will receive a reply with your personal API key.
4. Set `CALLMEBOT_PHONE` (your number with country code, no `+`) and `CALLMEBOT_API_KEY` in `.env`.

---

### 7. Deploy to GitHub Actions

1. Push the repo to GitHub.
2. Go to **Settings → Secrets and variables → Actions**.

Add these **Secrets** (sensitive values):

| Secret                        | Value                            |
| ----------------------------- | -------------------------------- |
| `STRAVA_CLIENT_ID`            | From Strava API settings         |
| `STRAVA_CLIENT_SECRET`        | From Strava API settings         |
| `STRAVA_REFRESH_TOKEN`        | From `setup/get_strava_token.py` |
| `STRAVA_CLUB_ID`              | Your club's numeric ID           |
| `GOOGLE_SERVICE_ACCOUNT_JSON` | Full JSON content (one line)     |
| `GOOGLE_SHEET_ID`             | From the Sheet URL               |
| `CALLMEBOT_PHONE`             | Your phone with country code     |
| `CALLMEBOT_API_KEY`           | From CallMeBot WhatsApp reply    |

Add these **Variables** (non-sensitive config):

| Variable         | Value                                   |
| ---------------- | --------------------------------------- |
| `ANNUAL_GOAL_KM` | `12000` (or your goal)                  |
| `TOTAL_WEEKS`    | `52`                                    |
| `SPORT_TYPES`    | Leave empty for all, or e.g. `Run,Walk` |

---

## Running locally

```bash
# Full run (writes to Sheets, sends WhatsApp):
python -m src.main

# Dry run (only fetches Strava, prints post text — safe for testing):
python -m src.main --dry-run
```

---

## Schedule

The GitHub Action runs **every Sunday at 23:00 UTC** (midnight Portugal winter time, 00:00 Portugal summer time).

You can also trigger it manually from the **Actions** tab → **Weekly Strava Club Post** → **Run workflow**.

---

## Post format

```
Semana 5/52

Total semanal: 80.0 km
Total anual: 750.0 / 12000 km (6.3%)

Ritmo para o objetivo: 1154 km
Estamos -404.0 km abaixo do ritmo. Vamos la!
```

Customize the text in `src/main.py → build_post_text()`.

---

## Project structure

```
.
├── .github/workflows/weekly_post.yml   # GitHub Actions cron job
├── setup/
│   ├── get_strava_token.py             # One-time OAuth helper
│   └── init_sheet.py                   # One-time Sheet bootstrapper
├── src/
│   ├── config.py                       # Reads all env vars
│   ├── strava_client.py                # Strava API (token refresh + activities)
│   ├── sheets_client.py                # Google Sheets read/write
│   ├── whatsapp_client.py              # CallMeBot WhatsApp sender
│   └── main.py                         # Orchestration & post generation
├── .env.example                        # Template — copy to .env
└── requirements.txt
```
