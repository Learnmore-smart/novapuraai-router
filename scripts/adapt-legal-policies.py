#!/usr/bin/env python3
"""Normalize LearnX-derived legal policy JSON for NovaPuraAI."""

from __future__ import annotations

import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
POLICY_DIR = ROOT / "web" / "default" / "src" / "i18n" / "policies"


def load(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8-sig"))


def save(path: Path, data: dict) -> None:
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def adapt_terms_en(data: dict) -> dict:
    terms = data.setdefault("terms", {})
    terms["lastUpdated"] = "Last updated July 15, 2026"
    intro = terms.setdefault("intro", {})
    intro["p1"] = (
        'We are NovaPuraAI ("Company," "we," "us," or "our"), operating an AI API gateway '
        "and prepaid balance platform."
    )
    intro["p2"] = (
        'We operate the website {websiteUrl} (the "Site"), as well as any other related '
        'products and services that refer or link to these legal terms (the "Legal Terms") '
        '(collectively, the "Services").'
    )
    intro["p3"] = "You can contact us by email at support@novapuraai.com."
    intro["p4"] = (
        'These Legal Terms constitute a legally binding agreement made between you, whether '
        'personally or on behalf of an entity ("you"), and NovaPuraAI, concerning your access '
        "to and use of the Services. You agree that by accessing the Services, you have read, "
        "understood, and agreed to be bound by all of these Legal Terms. IF YOU DO NOT AGREE "
        "WITH ALL OF THESE LEGAL TERMS, THEN YOU ARE EXPRESSLY PROHIBITED FROM USING THE "
        "SERVICES AND YOU MUST DISCONTINUE USE IMMEDIATELY."
    )
    # Keep remaining intro paragraphs if present; scrub LearnX leftovers.
    for key, value in list(intro.items()):
        if isinstance(value, str):
            intro[key] = scrub_string(value)
    return data


def adapt_privacy_en(data: dict) -> dict:
    privacy = data.setdefault("privacy", {})
    privacy["lastUpdated"] = "Last updated July 15, 2026"
    intro = privacy.setdefault("intro", {})
    for key, value in list(intro.items()):
        if isinstance(value, str):
            intro[key] = scrub_string(value)
    if "p1" in intro:
        intro["p1"] = (
            'This Privacy Notice for NovaPuraAI ("we," "us," or "our"), describes how and why '
            'we might access, collect, store, use, and/or share ("process") your personal '
            'information when you use our services ("Services"), including when you:'
        )
    if "p2" in intro:
        intro["p2"] = (
            "Questions or concerns? Reading this Privacy Notice will help you understand your "
            "privacy rights and choices. We are responsible for making decisions about how your "
            "personal information is processed. If you do not agree with our policies and "
            "practices, please do not use our Services. If you still have any questions or "
            "concerns, please contact us at support@novapuraai.com."
        )
    return data


def scrub_string(value: str) -> str:
    """Normalize brand/domain leftovers to NovaPuraAI + novapuraai.com."""
    out = (
        value.replace("LearnX", "NovaPuraAI")
        .replace("learnx.pro", "novapuraai.com")
        .replace("support@learnx.pro", "support@novapuraai.com")
    )
    # Case-insensitive support@NovaPuraAI.pro → support@novapuraai.com
    out = re.sub(
        r"support@novapuraai\.pro",
        "support@novapuraai.com",
        out,
        flags=re.IGNORECASE,
    )
    out = re.sub(
        r"novapuraai\.pro",
        "novapuraai.com",
        out,
        flags=re.IGNORECASE,
    )
    return out


def scrub_all(obj):
    if isinstance(obj, dict):
        return {k: scrub_all(v) for k, v in obj.items()}
    if isinstance(obj, list):
        return [scrub_all(v) for v in obj]
    if isinstance(obj, str):
        return scrub_string(obj)
    return obj


def main() -> None:
    files = [
        POLICY_DIR / "privacy" / "en.json",
        POLICY_DIR / "privacy" / "zh.json",
        POLICY_DIR / "terms" / "en.json",
        POLICY_DIR / "terms" / "zh.json",
    ]
    for path in files:
        data = scrub_all(load(path))
        if path.name == "en.json" and path.parent.name == "terms":
            data = adapt_terms_en(data)
        if path.name == "en.json" and path.parent.name == "privacy":
            data = adapt_privacy_en(data)
        save(path, data)
        print(f"updated {path.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
