"""Shared setup script bootstrap utilities."""

import os
import sys
from pathlib import Path

from dotenv import load_dotenv


def bootstrap_setup() -> None:
    """Enable imports from src and load .env for setup scripts."""
    repo_root = Path(__file__).resolve().parent.parent
    sys.path.insert(0, os.fspath(repo_root))
    load_dotenv()
