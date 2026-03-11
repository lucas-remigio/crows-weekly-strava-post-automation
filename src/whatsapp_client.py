"""
WhatsApp delivery via CallMeBot.

Setup (one-time, per phone number):
  1. Add +34 644 59 78 12 to your WhatsApp contacts as "CallMeBot".
  2. Send this message to that contact:
       I allow callmebot to send me messages
  3. You will receive your personal API key in response.
  4. Set CALLMEBOT_PHONE and CALLMEBOT_API_KEY in your .env / GitHub Secrets.

Docs: https://www.callmebot.com/blog/free-api-whatsapp-messages/
"""

import logging
import urllib.parse

import requests

from . import config

logger = logging.getLogger(__name__)

CALLMEBOT_ENDPOINT = "https://api.callmebot.com/whatsapp.php"


def send_whatsapp_message(message: str) -> None:
    """
    Send `message` to the configured WhatsApp number via CallMeBot.
    Raises requests.HTTPError on failure.
    """
    params = {
        "phone": config.CALLMEBOT_PHONE,
        "text": message,
        "apikey": config.CALLMEBOT_API_KEY,
    }

    # Build URL manually so we can log it (with API key redacted)
    safe_params = {**params, "apikey": "***"}
    logger.info(
        "Sending WhatsApp message to %s | URL params: %s",
        config.CALLMEBOT_PHONE,
        safe_params,
    )

    resp = requests.get(CALLMEBOT_ENDPOINT, params=params, timeout=15)

    # CallMeBot returns 200 even on some errors, so log the body
    logger.info("CallMeBot response [%d]: %s", resp.status_code, resp.text[:200])
    resp.raise_for_status()
