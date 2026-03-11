"""
Telegram delivery via the Bot API.

Setup (one-time):
  1. Open Telegram and message @BotFather.
  2. Send /newbot, follow the prompts, copy the token it gives you.
     → Set TELEGRAM_BOT_TOKEN in your .env / GitHub Secrets.
  3. Add the bot to your group (or just start a chat with it personally).
  4. Get the chat_id:
     a. Send any message to the bot / in the group.
     b. Open in browser:
        https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates
     c. Look for "chat": {"id": -123456789} — that number is your chat_id.
        Group IDs are negative; personal chat IDs are positive.
     → Set TELEGRAM_CHAT_IDS in .env as a comma-separated list.
        E.g. for one group + one personal: -123456789,987654321

Docs: https://core.telegram.org/bots/api#sendmessage
"""

import logging

import requests

from . import config

logger = logging.getLogger(__name__)

TELEGRAM_API = "https://api.telegram.org/bot{token}/sendMessage"


def _send_to_one(chat_id: str, message: str) -> None:
    """Send `message` to a single Telegram chat/group."""
    url = TELEGRAM_API.format(token=config.TELEGRAM_BOT_TOKEN)
    payload = {
        "chat_id": chat_id,
        "text": message,
    }
    logger.info("Sending Telegram message to chat_id %s", chat_id)
    resp = requests.post(url, json=payload, timeout=15)
    logger.info("Telegram response [%d]: %s", resp.status_code, resp.text[:200])
    resp.raise_for_status()


def send_message(message: str) -> None:
    """
    Send `message` to all chat IDs in TELEGRAM_CHAT_IDS.
    Logs errors per recipient without aborting the others.
    """
    if not config.TELEGRAM_CHAT_IDS:
        logger.warning("No TELEGRAM_CHAT_IDS configured — skipping Telegram.")
        return

    for chat_id in config.TELEGRAM_CHAT_IDS:
        try:
            _send_to_one(chat_id, message)
        except Exception as exc:
            logger.error("Failed to send Telegram message to %s: %s", chat_id, exc)
