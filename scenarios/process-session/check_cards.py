#!/usr/bin/env python3
"""Проверка карточек сценариев против файла ожиданий.

Что проверяет:
  1. Каждый ID утверждения (S01-05, S05-N3, I1-08 …), который встречается
     в карточках cards/<ID>/README.md, есть в YAML ожиданий.
  2. В каждой карточке есть все семь разделов.
  3. В разделе «Ожидается» от 3 до 7 пунктов.
  4. Число утверждений в оглавлении scenarios/process-session/README.md совпадает с YAML.

Запуск:
  python3 scenarios/_check_ids.py [путь к scenarios-flange-expected.yaml]

Без аргумента ищет файл рядом по известным путям. Только стандартная
библиотека: YAML разбирается регулярным выражением по строкам {id: …}.
Код выхода 0 — всё сошлось, 1 — есть ошибки.
"""
import re
import sys
from collections import Counter
from pathlib import Path

HERE = Path(__file__).resolve().parent
CANDIDATES = [
    HERE / "expected" / "scenarios-flange-expected.yaml",
    HERE.parent / "scenarios-flange-expected.yaml",
    HERE.parent.parent / "process" / "scenarios-flange-expected.yaml",
]
ID_RE = re.compile(r"\b((?:S\d{2}[AB]?|I1)-N?\d{1,2})\b")
YAML_ID_RE = re.compile(r"\{\s*id:\s*((?:S\d{2}[AB]?|I1)-N?\d{1,2})\s*,")
SECTIONS = ["Проверяет", "Исходное состояние", "Действие", "Ожидается",
            "Чего не должно случиться", "Данные", "Запуск"]


def find_yaml(argv):
    if len(argv) > 1:
        return Path(argv[1])
    for p in CANDIDATES:
        if p.exists():
            return p
    sys.exit("Не найден scenarios-flange-expected.yaml — передайте путь аргументом")


def section(text, name):
    m = re.search(rf"^## {re.escape(name)}\s*$(.*?)(?=^## |\Z)", text, re.M | re.S)
    return m.group(1) if m else None


def main(argv):
    yaml_path = find_yaml(argv)
    yaml_ids = YAML_ID_RE.findall(yaml_path.read_text(encoding="utf-8"))
    known = set(yaml_ids)
    errors = []
    dup = [i for i, n in Counter(yaml_ids).items() if n > 1]
    if dup:
        errors.append(f"YAML: повторяются ID {sorted(dup)}")

    cards = sorted((HERE / "cards").glob("*/README.md"))
    used = set()
    for card in cards:
        name = card.parent.name
        text = card.read_text(encoding="utf-8")
        for s in SECTIONS:
            if section(text, s) is None:
                errors.append(f"{name}: нет раздела «{s}»")
        exp = section(text, "Ожидается") or ""
        n_items = len(re.findall(r"^- ", exp, re.M))
        if not 3 <= n_items <= 7:
            errors.append(f"{name}: в «Ожидается» {n_items} пунктов (нужно 3–7)")
        ids = set(ID_RE.findall(text))
        if not ids:
            errors.append(f"{name}: нет ни одного ID утверждения")
        for i in sorted(ids - known):
            errors.append(f"{name}: ID {i} нет в YAML")
        used |= ids

    per_scn = Counter(i.split("-")[0] for i in yaml_ids)
    toc = (HERE / "README.md").read_text(encoding="utf-8")
    toc_rows = re.findall(r"^\| \[([A-Z0-9-]+)\]\(cards/[^)]+\)[^|]*\|[^|]*\|[^|]*\| *(\d+) *\|", toc, re.M)
    toc_map = {k: int(v) for k, v in toc_rows}
    for scn, n in sorted(per_scn.items()):
        if toc_map.get(scn) != n:
            errors.append(f"оглавление: у {scn} указано {toc_map.get(scn)}, в YAML {n}")
    for scn in toc_map:
        if scn != "MAIN-STORY" and scn not in per_scn:
            errors.append(f"оглавление: сценария {scn} нет в YAML")
    cards_names = {c.parent.name for c in cards}
    for scn in per_scn:
        if scn not in cards_names:
            errors.append(f"нет карточки для {scn}")

    print(f"YAML: {yaml_path}")
    print(f"Утверждений в YAML: {len(yaml_ids)}; сценариев: {len(per_scn)}")
    print(f"Карточек: {len(cards)}; разных ID в карточках: {len(used)}; "
          f"из них нет в YAML: {len(used - known)}")
    if errors:
        print("ОШИБКИ:")
        for e in errors:
            print("  -", e)
        return 1
    print("OK: все ID из карточек есть в YAML, разделы на месте, оглавление сходится")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
