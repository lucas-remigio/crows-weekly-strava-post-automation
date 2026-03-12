"""
OpenAI client — generates a weekly athlete roast for the post.

Picks a random athlete from src/athletes.py and asks GPT to write a short
funny sentence in European Portuguese either praising or blaming them based
on whether the club is above or below the annual pace.

Requires OPENAI_API_KEY to be set. If the key is absent, the roast is skipped
silently so the rest of the pipeline is unaffected.
"""

import logging
import random

import requests

from . import config
from .athletes import ATHLETES

logger = logging.getLogger(__name__)

OPENAI_API_URL = "https://api.openai.com/v1/chat/completions"


def generate_weekly_roast(above_pace: bool, diff_km: float) -> str | None:
    """
    Return a one-sentence funny roast in Portuguese, or None if the feature
    is not configured or the athlete list is empty.
    """
    if not config.OPENAI_API_KEY:
        logger.info("OPENAI_API_KEY not configured — skipping weekly roast.")
        return None

    if not ATHLETES:
        logger.info("ATHLETES list is empty — skipping weekly roast.")
        return None

    athlete = random.choice(ATHLETES)
    logger.info("Generating roast for athlete: %s", athlete["name"])

    resp = requests.post(
        OPENAI_API_URL,
        headers={
            "Authorization": f"Bearer {config.OPENAI_API_KEY}",
            "Content-Type": "application/json",
        },
        json={
            "model": "gpt-4o-mini",
            "messages": [
                {
                    "role": "system",
                    "content": (
                        "És um comentador desportivo bem-humorado de um clube de corrida. "
                        "Escreves frases curtas e engraçadas em português europeu informal. "
                        "Usa humor ligeiro e criativo, sem seres ofensivo."
                    ),
                },
                {"role": "user", "content": _build_prompt(athlete, above_pace, diff_km)},
            ],
            "max_tokens": 120,
            "temperature": 0.9,
        },
        timeout=15,
    )
    resp.raise_for_status()

    roast = resp.json()["choices"][0]["message"]["content"].strip()
    logger.info("Roast generated: %s", roast)
    return roast


def _build_prompt(athlete: dict[str, str], above_pace: bool, diff_km: float) -> str:
    stance = "acima" if above_pace else "abaixo"
    angle = (
        "elogiando-o como o herói da semana que puxou o clube para a frente"
        if above_pace
        else "culpando-o de forma engraçada por ser o responsável pelo atraso"
    )
    return (
        f"O atleta {athlete['name']} é conhecido por {athlete['characteristic']}. "
        f"O clube está {stance} do ritmo anual em {diff_km:.1f} km. "
        f"Escreve uma única frase engraçada {angle}, "
        f"relacionando com a sua característica. "
        f"Responde apenas com a frase, sem introdução nem explicação."
    )
