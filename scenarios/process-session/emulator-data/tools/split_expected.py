#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Раскладка process/scenarios-flange-expected.yaml по файлам expected/*.yaml — БЕЗ изменений содержания.
Текст режется по ключам верхнего уровня и по ключам сценариев; комментарии сохраняются.
Проверка: объединение разобранных файлов == разобранный исходник (глубокое равенство).
Запуск: python3 tools/split_expected.py
"""
import os
import re

import yaml

BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SRC = os.path.normpath(os.path.join(BASE, "..", "expected", "scenarios-flange-expected.yaml"))
OUT = os.path.join(BASE, "expected")

HDR = ("# =============================================================================\n"
       "# Срез файла process/scenarios-flange-expected.yaml — БЕЗ изменений (наши имена, как есть).\n"
       "# Источник истины — исходный файл; этот файл пересобирается tools/split_expected.py.\n"
       "# Общие словари и ID — expected/_common.yaml; главная история — expected/main-story.yaml.\n"
       "# Времена здесь — МСК (+03:00), как в исходнике; в потоках и определениях — UTC.\n"
       "# =============================================================================\n")


def main():
    text = open(SRC, encoding="utf-8").read()
    lines = text.splitlines(keepends=True)
    top = [(i, m.group(1)) for i, l in enumerate(lines) for m in [re.match(r"^([A-Za-z_][\w-]*):", l)] if m]
    blocks = {}
    for k, (i, key) in enumerate(top):
        j = top[k + 1][0] if k + 1 < len(top) else len(lines)
        # хвостовые комментарии-разделители относим к следующему блоку
        while j > i and (lines[j - 1].startswith("#") or not lines[j - 1].strip()):
            j -= 1
        start = i
        while start > 0 and (lines[start - 1].startswith("#")) and (k == 0 or start - 1 > top[k - 1][0]):
            start -= 1
        blocks[key] = (start if k > 0 else 0, j)
    written = []

    def w(name, body):
        p = os.path.join(OUT, name)
        with open(p, "w", encoding="utf-8") as f:
            f.write(HDR + "\n" + body)
        written.append(p)

    head_end = blocks["main_story"][0]
    w("_common.yaml", "".join(lines[0:head_end]))
    ms = blocks["main_story"]
    rv = blocks["role_views"]
    mt = blocks["main_story_totals"]
    w("main-story.yaml", "".join(lines[ms[0]:ms[1]]) + "\n" + "".join(lines[rv[0]:rv[1]]) + "\n" +
      "".join(lines[mt[0]:mt[1]]))
    sc0, sc1 = blocks["scenarios"]
    body = lines[sc0:sc1]
    subs = [(i, m.group(1)) for i, l in enumerate(body) for m in [re.match(r"^  ([A-Za-z0-9_]+):\s*$", l)] if m]
    for k, (i, key) in enumerate(subs):
        j = subs[k + 1][0] if k + 1 < len(subs) else len(body)
        while j > i and (body[j - 1].strip().startswith("#") or not body[j - 1].strip()):
            j -= 1
        w(key + ".yaml", "scenarios:\n" + "".join(body[i:j]))

    # проверка: объединение == исходник
    orig = yaml.safe_load(text)
    merged = {}
    for p in written:
        d = yaml.safe_load(open(p, encoding="utf-8"))
        for k, v in d.items():
            if k == "scenarios":
                merged.setdefault("scenarios", {}).update(v)
            else:
                assert k not in merged, k
                merged[k] = v
    assert merged == orig, "срез ожиданий не совпадает с исходником"
    print("OK: %d файлов, объединение совпадает с исходником (%s)" % (len(written), ", ".join(os.path.basename(p) for p in written)))


if __name__ == "__main__":
    main()
