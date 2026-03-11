"""
setup/get_strava_token.py

Run this ONCE locally to complete the Strava OAuth flow and obtain your
refresh_token. After that, store the refresh_token as a GitHub Secret and
never run this script again (the token doesn't expire unless you revoke it).

Usage:
    python setup/get_strava_token.py

Requirements:
    pip install requests

Steps:
  1. Go to https://www.strava.com/settings/api and create an app.
     - "Authorization Callback Domain" → set to  localhost
  2. Fill in CLIENT_ID and CLIENT_SECRET below (or in your .env file).
  3. Run this script. A browser tab will open for you to authorize.
  4. After authorization, Strava redirects to localhost — the script
     captures the code automatically via a tiny local HTTP server.
  5. The script prints the refresh_token. Copy it to GitHub Secrets.
"""

import http.server
import os
import sys
import threading
import urllib.parse
import webbrowser

import requests
from dotenv import load_dotenv

load_dotenv()

CLIENT_ID = os.environ.get("STRAVA_CLIENT_ID", "")
CLIENT_SECRET = os.environ.get("STRAVA_CLIENT_SECRET", "")

if not CLIENT_ID or not CLIENT_SECRET:
    print(
        "ERROR: Set STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET in your .env file "
        "(or as environment variables) before running this script."
    )
    sys.exit(1)

REDIRECT_PORT = 8888
REDIRECT_URI = f"http://localhost:{REDIRECT_PORT}/callback"
AUTH_URL = (
    "https://www.strava.com/oauth/authorize"
    f"?client_id={CLIENT_ID}"
    "&response_type=code"
    f"&redirect_uri={urllib.parse.quote(REDIRECT_URI)}"
    "&approval_prompt=force"
    "&scope=read,activity:read"
)

captured_code: list[str] = []


class CallbackHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        parsed = urllib.parse.urlparse(self.path)
        params = urllib.parse.parse_qs(parsed.query)

        if "code" in params:
            captured_code.append(params["code"][0])
            self.send_response(200)
            self.end_headers()
            self.wfile.write(
                b"<h2>Authorization successful! You can close this tab.</h2>"
            )
        else:
            self.send_response(400)
            self.end_headers()
            error = params.get("error", ["unknown"])[0]
            self.wfile.write(f"<h2>Error: {error}</h2>".encode())

    def log_message(self, *args):
        pass  # suppress request logs


def main():
    server = http.server.HTTPServer(("localhost", REDIRECT_PORT), CallbackHandler)
    thread = threading.Thread(target=server.handle_request)
    thread.start()

    print(f"\nOpening Strava authorization page...")
    print(f"If the browser doesn't open, visit:\n  {AUTH_URL}\n")
    webbrowser.open(AUTH_URL)

    thread.join(timeout=120)

    if not captured_code:
        print("ERROR: No authorization code received within 2 minutes.")
        sys.exit(1)

    code = captured_code[0]
    print(f"Authorization code received. Exchanging for tokens...")

    resp = requests.post(
        "https://www.strava.com/oauth/token",
        data={
            "client_id": CLIENT_ID,
            "client_secret": CLIENT_SECRET,
            "code": code,
            "grant_type": "authorization_code",
        },
        timeout=15,
    )
    resp.raise_for_status()
    data = resp.json()

    athlete = data.get("athlete", {})
    print(f"\nAuthorized as: {athlete.get('firstname', '')} {athlete.get('lastname', '')}")
    print(f"\n{'='*60}")
    print("Add these to your .env file and GitHub Secrets:")
    print(f"{'='*60}")
    print(f"STRAVA_CLIENT_ID={CLIENT_ID}")
    print(f"STRAVA_CLIENT_SECRET={CLIENT_SECRET}")
    print(f"STRAVA_REFRESH_TOKEN={data['refresh_token']}")
    print(f"{'='*60}\n")
    print("Keep the REFRESH TOKEN secret — it grants ongoing access to your Strava.")


if __name__ == "__main__":
    main()
