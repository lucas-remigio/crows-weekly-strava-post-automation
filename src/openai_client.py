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
from .athletes import get_athletes

logger = logging.getLogger(__name__)

OPENAI_API_URL = "https://api.openai.com/v1/chat/completions"


def _system_prompt() -> str:
    return (
        "O teu humor é inspirado em Ricardo Araújo Pereira: inteligente, irónico e absolutamente certeiro. "
        "Usas o absurdo com precisão cirúrgica. As tuas frases têm sempre uma lógica interna impecável que "
        "torna o disparate completamente inevitável. Não explicas, não exageras, não usas pontos de exclamação. "
        "O humor nasce da observação fria de factos ridículos, dita com a seriedade de quem está a ler uma acta. "
        "Escreves em português europeu, culto mas acessível, sem calão e sem emojis."
    )


def generate_weekly_roast(above_pace: bool, diff_km: float) -> str | None:
    """
    Return a one-sentence funny roast in Portuguese, or None if the feature
    is not configured or the athlete list is empty.
    """
    if not config.OPENAI_API_KEY:
        logger.info("OPENAI_API_KEY not configured — skipping weekly roast.")
        return None

    athletes = get_athletes()
    if not athletes:
        logger.info("Athletes list is empty — skipping weekly roast.")
        return None

    athlete = random.choice(athletes)
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
                    "content": _system_prompt(),
                },
                {"role": "user", "content": _build_prompt(athlete, above_pace, diff_km)},
            ],
            "max_tokens": 120,
            "temperature": 1.1,
        },
        timeout=config.HTTP_TIMEOUT_SECONDS,
    )
    resp.raise_for_status()

    roast = resp.json()["choices"][0]["message"]["content"].strip()
    logger.info("Roast generated: %s", roast)
    return roast


def _build_prompt(athlete: dict[str, str], above_pace: bool, diff_km: float) -> str:
    situation = (
        f"O clube está {diff_km:.0f} km acima do ritmo anual."
        if above_pace
        else f"O clube está {diff_km:.0f} km abaixo do ritmo anual."
    )
    return (
        f"{situation} "
        f"{athlete['name']} é conhecido por {athlete['characteristic']}. "
        f"Escreve uma única frase sobre {athlete['name']} que relacione a sua personalidade com este resultado. "
        f"Não expliques a piada. Não uses fórmulas como 'não é surpresa' ou 'é culpa de'. "
        f"Surpreende-nos. Responde apenas com a frase."
    )
