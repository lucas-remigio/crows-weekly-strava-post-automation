# Strava Crows Weekly Post

Automates the weekly Strava club progress post.

Every Sunday night a GitHub Action:

1. Fetches all club activities for the week from the Strava API
2. Sums the total distance
3. Adds it to a running annual total stored in Google Sheets
4. Generates a formatted post text
5. Sends it to Telegram via a bot

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

> The script requests `read` and `activity:read` scopes.

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

### 6. Set up Telegram

You already have the bot token. Now you just need the chat ID(s) of where to send the message.

#### 6a. Add the bot to your chat

- **Personal chat:** Open Telegram, search for your bot by username, and press **Start**.
- **Group:** Open the group → Add members → search for your bot and add it.

#### 6b. Get the chat ID

1. Send any message in the chat (or group) where the bot was added.
2. Open this URL in your browser (replace with your actual token):
   ```
   https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates
   ```
3. Find the `"chat"` object in the response:
   ```json
   "chat": { "id": -123456789, "type": "group" }
   ```
   That number is your chat ID. **Group IDs are negative**, personal chat IDs are positive.

#### 6c. Set the values in `.env`

```
TELEGRAM_BOT_TOKEN=123456789:ABC-your-token-here
TELEGRAM_CHAT_IDS=-123456789
```

To send to multiple chats (e.g. a group and your personal chat), separate them with a comma:

```
TELEGRAM_CHAT_IDS=-123456789,987654321
```

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
| `TELEGRAM_BOT_TOKEN`          | From @BotFather                  |
| `TELEGRAM_CHAT_IDS`           | Comma-separated chat IDs         |

Add these **Variables** (non-sensitive config):

| Variable         | Value                                   |
| ---------------- | --------------------------------------- |
| `ANNUAL_GOAL_KM` | `12000` (or your goal)                  |
| `TOTAL_WEEKS`    | `52` or `53` depending on the year      |
| `SPORT_TYPES`    | Leave empty for all, or e.g. `Run,Walk` |

---

## Running locally

```bash
# Full run (writes to Sheets, sends Telegram message):
python -m src.main

# Dry run (fetches Strava + reads current Sheet total, prints post — skips write & Telegram):
python -m src.main --dry-run
```

---

## Schedule

The GitHub Action runs **every Sunday at 22:00 UTC** (midnight Portugal winter time, 23:00 Portugal summer time).

You can also trigger it manually from the **Actions** tab → **Weekly Strava Club Post** → **Run workflow**.

---

## Post format

```
Semana 11/53 (20.8%)

Total semanal: 84.4 km
Total anual: 1101.7 / 12000 km (9.2%)

Por esta altura devíamos ter feito 2491 km
Estamos -1389.3 km abaixo do ritmo. Vamos lá!
```

Customize the text in `src/main.py → build_post_text()`.

---

## Project structure

```
.
├── .github/workflows/weekly_post.yml   # GitHub Actions cron job
├── setup/
│   ├── get_strava_token.py             # One-time OAuth helper
│   ├── init_sheet.py                   # One-time Sheet bootstrapper
│   └── fetch_historical_km.py          # One-time: sum all km from Jan 1 to last week
├── src/
│   ├── config.py                       # Reads all env vars
│   ├── strava_client.py                # Strava API (token refresh + activities)
│   ├── sheets_client.py                # Google Sheets read/write
│   ├── telegram_client.py              # Telegram bot sender
│   └── main.py                         # Orchestration & post generation
├── .env.example                        # Template — copy to .env
└── requirements.txt
```
