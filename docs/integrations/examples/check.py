#!/usr/bin/env python3
"""Проверка эталонных сообщений интеграций: каждый пример против своей схемы + отрицательные случаи.

Запуск:  python3 docs/integrations/examples/check.py
Нужно:   pip install jsonschema lxml   (jsonschema >= 3.2, draft-07; lxml — для XSD Галактики)
Код выхода 0 — всё сошлось. Любое расхождение — код 1 и список.
"""
import copy
import hashlib
import json
import pathlib
import sys

import jsonschema

try:
    from lxml import etree
except ImportError:  # XML тогда проверяется только на правильность разметки
    etree = None
    import xml.etree.ElementTree as ET

ROOT = pathlib.Path(__file__).resolve().parent

# пример → схема (путь от каталога этого файла)
JSON_PAIRS = {
    "1c/control-result.request.F-001.ZT-6.accepted.json": "1c/schema/control-result.request.schema.json",
    "1c/control-result.request.NS-I1.ZT-R.rework.json": "1c/schema/control-result.request.schema.json",
    "1c/control-result.request.LOT-R-117.ZT-1.accepted.json": "1c/schema/control-result.request.schema.json",
    "1c/control-result.response.201-accepted.json": "1c/schema/response.schema.json",
    "1c/control-result.response.200-duplicate.json": "1c/schema/response.schema.json",
    "1c/control-result.response.422-ref-not-found.json": "1c/schema/response.schema.json",
    "1c/control-result.response.409-message-id-conflict.json": "1c/schema/response.schema.json",
    "1c/control-result.response.503-unavailable.json": "1c/schema/response.schema.json",
    "mes/in.job-order.SV-017-2.json": "mes/schema/job-order.schema.json",
    "mes/in.job-response.SV-017-2.completed.json": "mes/schema/job-response.schema.json",
    "mes/out.ack.job-order.accepted.json": "mes/schema/acknowledge.schema.json",
    "mes/out.material-sublot.hold.F-017.json": "mes/schema/material-sublot-hold.schema.json",
    "mes/out.material-sublot.release.F-008.json": "mes/schema/material-sublot-hold.schema.json",
    "mes/in.ack.hold.accepted.json": "mes/schema/acknowledge.schema.json",
    "mes/in.ack.hold.rejected-unknown-sublot.json": "mes/schema/acknowledge.schema.json",
    "mes/out.test-result.F-001.ZT-2.accepted.json": "mes/schema/test-result.schema.json",
    "mes/out.test-result.NS-I1.rework.json": "mes/schema/test-result.schema.json",
    "kompas/assembly.FL-100.00.000-B.json": "kompas/schema/assembly.schema.json",
    "kompas/discrepancy-report.FL-100.00.000-B.json": "kompas/schema/discrepancy-report.schema.json",
}
XSD = "galaktika/schema/qc-galaktika-1.0.xsd"


def load(rel):
    return json.loads((ROOT / rel).read_text(encoding="utf-8"))


FORMATS = jsonschema.FormatChecker()
if "date-time" not in FORMATS.checkers:  # без rfc3339-пакетов jsonschema формат не проверяет — проверяем сами
    import datetime
    import re

    @FORMATS.checks("date-time", raises=ValueError)
    def _date_time(value):
        if not isinstance(value, str):
            return True
        if not re.fullmatch(r"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?(Z|[+-]\d\d:\d\d)", value):
            raise ValueError("нужно RFC 3339 со смещением, например 2026-09-23T10:15:00+03:00")
        datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))
        return True


def validator(schema_rel):
    schema = load(schema_rel)
    jsonschema.Draft7Validator.check_schema(schema)
    return jsonschema.Draft7Validator(schema, format_checker=FORMATS)


def errors_of(v, doc):
    return [f"{'/'.join(map(str, e.absolute_path)) or '(корень)'}: {e.message}" for e in v.iter_errors(doc)]


# Отрицательные случаи: схема ОБЯЗАНА отвергнуть (постановка §4.7)
def neg_cases():
    def drop(doc, *path):
        d = doc
        for p in path[:-1]:
            d = d[p]
        del d[path[-1]]
        return doc

    def setv(doc, value, *path):
        d = doc
        for p in path[:-1]:
            d = d[p]
        d[path[-1]] = value
        return doc

    f001 = "1c/control-result.request.F-001.ZT-6.accepted.json"
    nsi1 = "1c/control-result.request.NS-I1.ZT-R.rework.json"
    hold = "mes/out.material-sublot.hold.F-017.json"
    rel = "mes/out.material-sublot.release.F-008.json"
    jo = "mes/in.job-order.SV-017-2.json"
    asm = "kompas/assembly.FL-100.00.000-B.json"
    return [
        ("1С: нет обязательного поля message_id", f001, lambda d: drop(d, "message_id")),
        ("1С: неизвестное значение итога «maybe»", f001, lambda d: setv(d, "maybe", "outcome")),
        ("1С: «оценка невозможна» — не итог ЗТ", f001, lambda d: setv(d, "unable_to_assess", "outcome")),
        ("1С: старшая версия контракта 2.0", f001, lambda d: setv(d, "2.0", "contract_version")),
        ("1С: «годно по разрешению» без номера разрешения", f001, lambda d: setv(d, "accepted_by_permit", "outcome")),
        ("1С: ЗТ-Р без решения по изделию", nsi1, lambda d: setv(d, None, "disposition")),
        ("1С: в сообщение попало лишнее поле (гипотеза)", nsi1, lambda d: setv(d, "дрейф тока ИС-2", "hypothesis")),
        ("1С: время подписи без смещения часового пояса", f001, lambda d: setv(d, "2026-09-23 10:15", "signed", "at")),
        ("MES: удержание уровня «наблюдать»", hold, lambda d: setv(d, "watch", "DataArea", "MaterialSubLot", 0, "Hold", "Level")),
        ("MES: в удержании текст причины", hold, lambda d: setv(d, "подозрение на дрейф ИС-2", "DataArea", "MaterialSubLot", 0, "Hold", "Reason")),
        ("MES: снятие удержания без человека", rel, lambda d: drop(d, "DataArea", "MaterialSubLot", 0, "Hold", "ReleasedBy")),
        ("MES: наряд без изделия", jo, lambda d: drop(d, "DataArea", "JobOrder", 0, "MaterialRequirement")),
        ("КОМПАС: геометрия не null", asm, lambda d: setv(d, {"step": "ФЛ-100.stp"}, "assembly", "geometry")),
        ("КОМПАС: нет версии КД", asm, lambda d: drop(d, "assembly", "version")),
    ]


def main():
    fails, n_ok = [], 0
    cache = {}
    for ex, sch in JSON_PAIRS.items():
        v = cache.setdefault(sch, validator(sch))
        errs = errors_of(v, load(ex))
        if errs:
            fails.append(f"ПРИМЕР НЕ ПРОШЁЛ {ex}: " + "; ".join(errs))
        else:
            n_ok += 1
    n_neg = 0
    for title, ex, mutate in neg_cases():
        doc = mutate(copy.deepcopy(load(ex)))
        if errors_of(cache[JSON_PAIRS[ex]], doc):
            n_neg += 1
        else:
            fails.append(f"СХЕМА НЕ ОТВЕРГЛА: {title}")

    # целостность: хеш файла сборки в отчёте о расхождениях
    rep = load("kompas/discrepancy-report.FL-100.00.000-B.json")
    sha = hashlib.sha256((ROOT / "kompas/assembly.FL-100.00.000-B.json").read_bytes()).hexdigest()
    if rep["source_file"]["sha256"] != sha:
        fails.append("kompas: sha256 в отчёте не совпадает с файлом сборки")

    xml_files = sorted((ROOT / "galaktika").glob("*.xml"))
    n_xml = 0
    if etree is not None:
        xsd = etree.XMLSchema(etree.parse(str(ROOT / XSD)))
        for f in xml_files:
            if xsd.validate(etree.parse(str(f))):
                n_xml += 1
            else:
                fails.append(f"XML НЕ ПРОШЁЛ XSD {f.name}: {xsd.error_log.last_error}")
        # отрицательный: неизвестная сущность в заголовке
        bad = etree.parse(str(ROOT / "galaktika/out.defect-act.NS-I1.xml"))
        ns = {"g": "urn:sergey-palych:integration:galaktika"}
        bad.find(".//g:Header/g:Entity", ns).text = "Hypothesis"
        if xsd.validate(bad):
            fails.append("XSD НЕ ОТВЕРГЛА неизвестную сущность Hypothesis")
        else:
            n_neg += 1
    else:
        for f in xml_files:
            ET.parse(f)
            n_xml += 1
        print("ВНИМАНИЕ: lxml нет — XML проверен только на правильность разметки, без XSD")

    print(f"JSON: {n_ok}/{len(JSON_PAIRS)} примеров прошли схемы; XML: {n_xml}/{len(xml_files)} прошли XSD; "
          f"отрицательных случаев отвергнуто: {n_neg}")
    for f in fails:
        print(" -", f)
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
