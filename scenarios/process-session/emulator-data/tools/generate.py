#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Генератор машинных данных имитатора: сценарии на фланце люка (S01–S22; волна 2 сверена со схемой v0.3; нестандартные
случаи S18–S22 — со схемой v0.3.2).

Детерминирован: один и тот же запуск даёт те же файлы байт в байт (seed в каждом потоке,
случайные значения — random.Random от строкового ключа события, без часов и без hash()).

Что строит:
  streams/main-story.jsonl        — главная история MS-1 «Плохой день ИС-2» (S01–S12, S10A), один прогон
  streams/S01…S11.jsonl, S10A     — срезы главной истории по сценарию (те же event_id и source_seq)
  streams/S06.jsonl               — инъекция в тот же прогон после главной истории (повтор пачки + конфликт)
  streams/S12.jsonl               — срез: пять сообщений на границе контракта (+ исправленная повторная отправка)
  streams/S10B.jsonl, S13.jsonl,
          S14.jsonl               — самостоятельные прогоны (свой seed, своя нумерация source_seq)
  streams/S15…S17.jsonl           — срезы главной истории (волна 2: часы источника, привязка к операции, пропуск проверки)
  streams/S18…S21.jsonl           — нестандартные случаи: короткие прогоны в своих мирах (свой seed и source_seq)
  streams/S22.jsonl               — продолжение прогона S10B (попытка вернуть списанный F-090), подаётся после S10B.jsonl
  streams/_manifest.yaml          — описание потоков, счётчики, намеренные разрывы и дубли
  definitions/scenarios/*.yaml    — определения сценариев: цель, поток, шаги людей, сбои стенда, ожидания;
                                    в I1, S10A, S13, S10B — erp_outbox_expected (что уходит в 1С и ответ stand-а)
Линия: line_id в конверте по правилу POST_LINE (две линии ФЛ-100, расходятся на сварочных постах).

Имена событий и полей — НАШИ (process-events-catalog.md, scenarios-flange*.{md,yaml}, docs/plan.md §4).
Форма — из спайна соседей: JSONL, одно исходное (неподписанное) событие в строке; конверт с
source_id, event_id, source_seq и тремя временами; ID в ASCII; детерминизм по seed.
Запуск: python3 tools/generate.py   (из папки emulator-data или откуда угодно)
"""
import copy
import datetime as dt
import json
import os
import random
import uuid
from collections import defaultdict, OrderedDict

import yaml

BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MSK = dt.timezone(dt.timedelta(hours=3))
UTC = dt.timezone.utc
# Пространство имён для event_id шаблонных файлов (предварительно; у соседей в прогоне
# event_id = UUIDv5(run_id, seed ‖ gen_no) — пересчитывается загрузчиком, см. README).
NS_TEMPLATE = uuid.uuid5(uuid.NAMESPACE_URL, "urn:kosmo-flange:emulator-data:v0")
CONTRACT = "1.0"

# ----------------------------------------------------------------------------- время
def T(s):
    """МСК-время из строки 'YYYY-MM-DD HH:MM[:SS]'."""
    fmt = "%Y-%m-%d %H:%M:%S" if s.count(":") == 2 else "%Y-%m-%d %H:%M"
    return dt.datetime.strptime(s, fmt).replace(tzinfo=MSK)


def iso(t):
    """RFC 3339 UTC, ровно три знака после секунд (соглашение спайна)."""
    u = t.astimezone(UTC)
    return u.strftime("%Y-%m-%dT%H:%M:%S.") + "%03dZ" % (u.microsecond // 1000)


def mskstr(t):
    return t.astimezone(MSK).strftime("%Y-%m-%d %H:%M:%S")


def M(n):
    return dt.timedelta(minutes=n)


def S(n):
    return dt.timedelta(seconds=n)


def day(t):
    return t.astimezone(MSK).date()


def at(d, hm):
    """Время hm (строка 'HH:MM[:SS]') в день d (date), МСК."""
    return T(d.isoformat() + " " + hm)


WEEKEND = {dt.date(2026, 9, 5), dt.date(2026, 9, 6), dt.date(2026, 9, 12), dt.date(2026, 9, 13),
           dt.date(2026, 9, 19), dt.date(2026, 9, 20), dt.date(2026, 9, 26), dt.date(2026, 9, 27),
           dt.date(2026, 10, 3), dt.date(2026, 10, 4), dt.date(2026, 10, 10), dt.date(2026, 10, 11)}   # октябрь — мир S18


def next_workday(d):
    d = d + dt.timedelta(days=1)
    while d in WEEKEND:
        d = d + dt.timedelta(days=1)
    return d


def shift_of(t):
    """'A' 08:00–16:30, 'B' 16:30–01:00; иначе None (ночь)."""
    h = t.astimezone(MSK)
    m = h.hour * 60 + h.minute
    if 8 * 60 <= m < 16 * 60 + 30:
        return "A"
    if m >= 16 * 60 + 30 or m < 60:
        return "B"
    return None


def shift_key(t):
    """(дата начала смены, 'A'|'B')."""
    h = t.astimezone(MSK)
    s = shift_of(t)
    d = h.date()
    if s == "B" and h.hour < 1:
        d = d - dt.timedelta(days=1)
    return (d, s)


def rnd(key):
    return random.Random("flange|" + key)


def r2(x):
    return round(x + 0.0, 2)


# ----------------------------------------------------------------------------- источники
SOURCES = {
    "ERP-1C": ("external_system", "stand", 3), "CAD-KOMPAS": ("import", "stand", 5),
    "TERM-IQC": ("manual_entry", "terminal", 1), "TERM-STK": ("manual_entry", "terminal", 1),
    "TERM-MC-1": ("manual_entry", "terminal", 1), "TERM-MC": ("manual_entry", "terminal", 1),
    "TERM-WS-1": ("manual_entry", "terminal", 1), "TERM-WS-2": ("manual_entry", "terminal", 1),
    "TERM-WC": ("manual_entry", "terminal", 1), "TERM-ASM": ("manual_entry", "terminal", 1),
    "TERM-LT": ("manual_entry", "terminal", 1), "TERM-AC": ("manual_entry", "terminal", 1),
    "TERM-QC": ("manual_entry", "terminal", 1),
    "CNC-1": ("machine", "edge-agent", 1), "CMM-1": ("machine", "edge-agent", 2),
    "WS-1": ("machine", "edge-agent", 1), "WS-2": ("machine", "edge-agent", 1),
    "TW-1": ("sensor", "edge-agent", 1), "LT-1": ("machine", "edge-agent", 2),
    "TW-2": ("sensor", "edge-agent", 1),   # S18: контрольный ключ ОТК КЛ-2 [ПП]
    "XR-1": ("external_system", "edge-agent", 5),
    "CAM-IQC": ("camera", "edge-agent", 2), "CAM-MO": ("camera", "edge-agent", 2),
    "CAM-WS-1": ("camera", "edge-agent", 2), "CAM-WS-2": ("camera", "edge-agent", 2),
    "CAM-UV": ("camera", "edge-agent", 2), "CAM-ASM": ("camera", "edge-agent", 2),
    "CAM-FQC": ("camera", "edge-agent", 2), "OV-ASM": ("camera", "edge-agent", 1),
}
EDGE = {"WS-1": "GW-1", "WS-2": "GW-2", "CNC-1": "EA-MC", "CMM-1": "EA-MC", "TW-1": "EA-AC", "TW-2": "EA-AC", "LT-1": "EA-AC",
        "XR-1": "EA-NDT", "CAM-IQC": "EA-KT1", "CAM-MO": "EA-KT2", "CAM-WS-1": "EA-KT3", "CAM-WS-2": "EA-KT3",
        "CAM-UV": "EA-KT4", "CAM-ASM": "EA-KT4", "OV-ASM": "EA-KT4", "CAM-FQC": "EA-KT5"}

SETPOINT_CURRENT = {"nominal": 160, "tolerance": 10, "unit": "A"}
ANALYZERS = {"CP-IQC": "vqc-iqc 1.2.0", "CP-MO": "vqc-mo 1.4.0", "CP-WELD": "vqc-weld 2.3.1",
             "CP-ASM-ZONE": "vqc-fod 1.0.2", "CP-ASM": "vqc-asm 1.3.0", "CP-FQC": "vqc-fqc 1.1.0"}
RECIPES = {"CP-IQC": "RCP-CP-IQC@1", "CP-MO": "RCP-CP-MO@2", "CP-WELD": "RCP-CP-WELD@3",
           "CP-ASM-ZONE": "RCP-CP-ASM-ZONE@1", "CP-ASM": "RCP-CP-ASM@2", "CP-FQC": "RCP-CP-FQC@1"}
CALIB = {"CAM-IQC": "CAL-CAM-IQC-2026-09", "CAM-MO": "CAL-CAM-MO-2026-09", "CAM-WS-1": "CAL-CAM-WS-2026-09",
         "CAM-WS-2": "CAL-CAM-WS-2026-09", "CAM-UV": "CAL-CAM-UV-2026-09", "CAM-ASM": "CAL-CAM-ASM-2026-09",
         "CAM-FQC": "CAL-CAM-FQC-2026-09"}
SEGS = ["U1", "U2", "U3", "U4", "U5", "U6", "U7", "U8"]

# Линии [П] (definitions/reference/sites.yaml → lines). Две линии ФЛ-100 расходятся только на сварке:
# у каждой свой сварочный пост; остальные участки общие. line_id в конверте:
#   1) источники поста (терминал поста, сварочный источник) — всегда линия своего поста;
#   2) событие по фланцу (item_id F-…) с момента запуска (item.registered) — линия изделия: линия поста,
#      где сделана его последняя на этот момент сварка; до первой сварки — линия поста из плана запуска;
#   3) остальное (1С, КОМПАС, партии и компоненты на складе, общее оборудование без item_id) — без линии.
POST_LINE = {"TERM-WS-1": "L-FL-1", "WS-1": "L-FL-1", "TERM-WS-2": "L-FL-2", "WS-2": "L-FL-2"}
STATION_LINE = {"ST-WC-P1": "L-FL-1", "ST-WC-P2": "L-FL-2"}


def num(item):
    return item.split("-")[1]


# ----------------------------------------------------------------------------- генератор
class Gen:
    def __init__(self, stream_id, seed):
        self.stream_id = stream_id
        self.seed = seed
        self.events = []
        self.steps = []
        self.welds = []       # сварки для телеметрии сварочных источников
        self.power = []       # явные окна питания сварочных источников (src, on, off, reason, after)
        self.power_override = set()   # (src, shift_key) — окна заданы явно
        self.labels = set()
        self.step_labels = set()
        self.n = 0
        self.stand_faults = []
        self.tamper = []
        self.lost_windows = []  # (src, from_excl, to_excl, expect_count, note)
        self.batches = []       # досылка пачкой: dict(src, from_incl, to_incl, deliver_start, dup_first, key_label, note)
        self.engine_holds = []  # v0.3.1: удержания, которые ставит движок без шага человека (круг, отзыв приёмки)

    # --- событие-факт (строка потока)
    def ev(self, occ, src, etype, item=None, lot=None, payload=None, tags=(), label=None, schema=CONTRACT,
           corr=None, after=None, drop_item=False, hints=(), noise=None, note=None, persona=None):
        assert src in SOURCES, src
        self.n += 1
        if label is None:
            label = "%s/%s/%s" % (etype, item or lot or src, occ.astimezone(MSK).strftime("%m%d-%H%M%S"))
        base, k = label, 2
        while label in self.labels:
            label = "%s#%d" % (base, k)
            k += 1
        self.labels.add(label)
        h = set(hints)
        if item:
            h.add(item)
        e = dict(i=self.n, occ=occ, src=src, type=etype, item=item, lot=lot, payload=payload or {}, tags=set(tags),
                 label=label, schema=schema, corr=corr, after=after, drop_item=drop_item, hints=h, lost=False,
                 noise=noise, note=note, persona=persona, deliver=None, seq=None)
        self.events.append(e)
        return e

    # --- шаг человека (решение; в поток не входит, выполняет роль или demo-signer)
    def step(self, at_, persona, role, action, subject, params=None, mode="auto", tags=(), catalog=None,
             expect="accepted", signers=None, note=None, label=None, hints=(), optional=False):
        if label is None:
            label = "%s/%s/%s" % (action, subject.get("item_id") or subject.get("lot_id") or subject.get("nc_ref") or persona,
                                  at_.astimezone(MSK).strftime("%m%d-%H%M%S"))
        base, k = label, 2
        while label in self.step_labels:
            label = "%s#%d" % (base, k)
            k += 1
        self.step_labels.add(label)
        h = set(hints)
        for key in ("item_id",):
            if subject.get(key):
                h.add(subject[key])
        for it in subject.get("items", []) or []:
            h.add(it)
        self.n += 1
        s = dict(i=self.n, at=at_, persona=persona, role=role, action=action, subject=subject, params=params or {}, mode=mode,
                 tags=set(tags), catalog=catalog, expect=expect, signers=signers or [persona], note=note, label=label,
                 hints=h, optional=optional)
        self.steps.append(s)
        return s


# ----------------------------------------------------------------------------- типовые шаги маршрута
def gate(g, at_, persona, gate_id, item=None, lot=None, decision="accept", tags=(), basis=None, signers=None,
         note=None, mode="auto", presentation=1, extra=None, label=None, expect="accepted"):
    cat = {"ZT-1": "E-08", "ZT-2": "E-19", "ZT-3": "E-47", "ZT-SR": "E-55", "ZT-4": "E-55", "ZT-5": "E-63", "ZT-6": "E-73"}[gate_id]
    subj = {"gate": gate_id}
    if item:
        subj["item_id"] = item
    if lot:
        subj["lot_id"] = lot
    params = {"decision": decision, "presentation_no": presentation}
    if basis:
        params["basis_labels"] = basis
    if gate_id == "ZT-2" and item:
        # перемаркировка подтверждается ОТК на ЗТ-2: номер на бирке = номер в коде, код читается [П] (строка 34)
        params["marking_check"] = {"previous": carrier_before_machining(item)["value"], "current": carrier_dm(item)["value"],
                                   "result": "match", "readable": True, "basis_label": "remark/" + item}
    if extra:
        params.update(extra)
    role = "QC"
    return g.step(at_, persona, role, "decision.gate_passed", subj, params, mode=mode, tags=tags, catalog=cat,
                  signers=signers, note=note, label=label, expect=expect)


def camera(g, t, src, item, cp, phase, zones, result="no_defect_found", q=None, conf=None, defects=None,
           limitations=None, tags=(), label=None, schema=CONTRACT, extra=None, drop_item=False, lot=None,
           hints=(), noise=None, note=None, reasons=None):
    r = rnd("cam|%s|%s|%s" % (src, item or lot, t.isoformat()))
    if q is None:
        q = r2(r.uniform(0.88, 0.95))
    if conf is None:
        conf = r2(r.uniform(0.90, 0.97))
    p = OrderedDict()
    p["checkpoint_id"] = cp
    p["phase"] = phase
    p["method"] = "camera"
    p["recipe_id"] = RECIPES[cp]
    p["camera_id"] = src
    p["calibration_id"] = CALIB[src]
    p["analyzer_version"] = ANALYZERS[cp]
    p["processing_state"] = "completed"
    p["inspection_result"] = result
    p["observation_quality"] = q
    if reasons:
        p["observation_quality_reasons"] = reasons
    p["confidence"] = conf
    p["zones_inspected"] = zones
    p["defects"] = defects or []
    p["limitations"] = limitations or []
    p["evidence_refs"] = []
    p["is_simulated"] = False
    if extra:
        p.update(extra)
    return g.ev(t, src, "inspection.result", item=None if drop_item else item, lot=lot, payload=p, tags=tags,
                label=label, schema=schema, drop_item=drop_item, hints=hints or ([item] if item else []),
                noise=noise, note=note)


def op_start(g, t, src, item, run, code, station, operator, equipment=None, tags=(), extra=None, rework_of=None,
             label=None, note=None):
    p = OrderedDict([("operation_run_id", run), ("operation_code", code), ("station_id", station),
                     ("operator_id", operator)])
    if equipment:
        p["equipment_id"] = equipment
    if rework_of:
        p["rework_of_run_id"] = rework_of
    if extra:
        p.update(extra)
    return g.ev(t, src, "operation.started", item=item, payload=p, tags=tags, corr=run, label=label or "%s.start" % run,
                persona=operator, note=note)


def op_finish(g, t, src, item, run, operator, result="completed", reported=None, tags=(), label=None, extra=None):
    p = OrderedDict([("operation_run_id", run), ("result", result), ("operator_id", operator)])
    if reported:
        p["reported_duration"] = reported
    if extra:
        p.update(extra)
    return g.ev(t, src, "operation.finished", item=item, payload=p, tags=tags, corr=run, label=label or "%s.finish" % run,
                persona=operator)


def op_action(g, t, src, item, operator, action_type, details, run=None, tags=(), label=None, after=None):
    p = OrderedDict([("action_type", action_type), ("operator_id", operator)])
    if run:
        p["operation_run_id"] = run
    p["details"] = details
    return g.ev(t, src, "operator.action", item=item, payload=p, tags=tags, corr=run, label=label, persona=operator,
                after=after)


def move(g, t_sent, item, frm, to, sender, receiver, recv_delay=10, tags=(), received=True, label=None,
         after_recv=None, receipt_check="no_damage", damage_details=None, recv_tags=()):
    shops = {"MC": ("TERM-MC", "WH-MC"), "WC": ("TERM-WC", "WH-WC"), "AC": ("TERM-AC", "WH-AC"),
             "SK": ("TERM-STK", "WH-SK"), "QC": ("TERM-QC", "WH-FG")}
    src_s, wh_from = shops[frm]
    src_r, wh_to = shops[to]
    g.ev(t_sent, src_s, "movement.sent", item=item,
         payload=OrderedDict([("from_shop", frm), ("to_shop", to), ("from_warehouse", wh_from), ("to_warehouse", wh_to),
                              ("sent_by", sender), ("container", "TARA-" + frm + "-" + to)]),
         tags=tags, label=(label + ".sent") if label else None, persona=sender)
    if received:
        g.ev(t_sent + M(recv_delay), src_r, "movement.received", item=item,
             payload=OrderedDict([("from_shop", frm), ("to_shop", to), ("received_by", receiver),
                                  ("inspection_at_receipt", receipt_check),
                                  ("damage", "none" if receipt_check == "no_damage" else "found")]
                                 + ([("damage_details", damage_details)] if damage_details else [])),
             tags=tuple(tags) + tuple(recv_tags), label=(label + ".received") if label else None, persona=receiver,
             after=after_recv)


# ----------------------------------------------------------------------------- 1С, импорт, партии, входной контроль
def imports(g, t, tags=()):
    g.ev(t, "CAD-KOMPAS", "assembly.structure_imported",
         payload=OrderedDict([("file", "process/kompas-assembly-flange.json"), ("format_version", "1.0"),
                              ("assembly_id", "FL-100.00.000"), ("item_type_id", "FL-100-ASSY"), ("version", "B"),
                              ("components", [{"position": 1, "item_type_id": "FL-100-FLANGE", "qty": 1},
                                              {"position": 2, "item_type_id": "FL-100-RING", "qty": 1},
                                              {"position": 3, "item_type_id": "FL-100-COVER", "qty": 1},
                                              {"position": 4, "item_type_id": "FL-100-SEAL", "qty": 1},
                                              {"position": 5, "item_type_id": "FL-100-FASTENER", "qty": 12},
                                              {"position": 6, "item_type_id": "FL-100-VALVE", "qty": 1}]),
                              ("links", ["W-1", "J-1", "S-1", "J-2"]), ("geometry", None),
                              ("note", "геометрия отсутствует явно; состав и связи — для нормативного слоя")]),
         tags=tags, label="import/kompas-FL-100")
    g.ev(t + M(10), "ERP-1C", "erp.nomenclature_synced",
         payload=OrderedDict([("message_id", "1c-in-000290"), ("nomenclature_ext_id", "nom-7a1e-0001"),
                              ("item_type_id", "FL-100-ASSY"), ("designation", "ФЛ-100.00.000 СБ"),
                              ("kd_revision", "B"), ("tp_id", "TP-FL-100"), ("tp_revision", 3)]),
         tags=tags, label="erp/nomenclature/FL-100")


def order(g, t, oid, ext, msg, qty, due, first, last, tags=()):
    g.ev(t, "ERP-1C", "erp.order_received",
         payload=OrderedDict([("message_id", msg), ("order_id", oid), ("order_ext_id", ext),
                              ("nomenclature_ext_id", "nom-7a1e-0001"), ("item_type_id", "FL-100-ASSY"),
                              ("designation", "ФЛ-100.00.000 СБ"), ("qty", qty), ("due_date", due),
                              ("kd_revision", "B"), ("tp_id", "TP-FL-100"), ("tp_revision", 3),
                              ("expected_items", {"first": first, "last": last})]),
         corr=oid, tags=tags, label="erp/order/" + oid)


LOTS = {  # lot: (item_type, supplier, qty, cert, heat, ext, msg)
    "LOT-ZF-201": ("FL-100-FLANGE", "SUP-1", 42, "C-201", "5512", "rcp-3f44-0201", "1c-in-000316"),
    "LOT-R-116": ("FL-100-RING", "SUP-2", 33, "C-116", None, "rcp-3f44-0116", "1c-in-000317"),
    "LOT-C-301": ("FL-100-COVER", "SUP-4", 40, "C-301", None, "rcp-3f44-0301", "1c-in-000318"),
    "LOT-SL-401": ("FL-100-SEAL", "SUP-5", 50, "C-401", None, "rcp-3f44-0401", "1c-in-000319"),
    "LOT-FS-501": ("FL-100-FASTENER", "SUP-6", 500, "C-501", None, "rcp-3f44-0501", "1c-in-000320"),
    "LOT-W-88": ("WELD-WIRE", "SUP-7", 20, "C-088", None, "rcp-3f44-0088", "1c-in-000321"),
    "LOT-R-117": ("FL-100-RING", "SUP-3", 10, "C-117", None, "rcp-3f44-0117", "1c-in-000322"),
    "LOT-ZF-111": ("FL-100-FLANGE", "SUP-1", 24, "C-111", "5498", "rcp-3f44-0111", "1c-in-000302"),
    "LOT-R-108": ("FL-100-RING", "SUP-2", 24, "C-108", None, "rcp-3f44-0108", "1c-in-000303"),
    "LOT-C-211": ("FL-100-COVER", "SUP-4", 24, "C-211", None, "rcp-3f44-0211", "1c-in-000304"),
    "LOT-SL-391": ("FL-100-SEAL", "SUP-5", 30, "C-391", None, "rcp-3f44-0391", "1c-in-000305"),
    "LOT-FS-481": ("FL-100-FASTENER", "SUP-6", 300, "C-481", None, "rcp-3f44-0481", "1c-in-000306"),
    "LOT-W-87": ("WELD-WIRE", "SUP-7", 20, "C-087", None, "rcp-3f44-0087", "1c-in-000307"),
    "LOT-R-119": ("FL-100-RING", "SUP-2", 5, "C-119", None, "rcp-3f44-0119", "1c-in-000331"),
    # волна 2 [П]: клапаны выравнивания давления (установка оборудования, кейс §1.1) и партии прогона S14
    "LOT-V-601": ("FL-100-VALVE", "SUP-8", 40, "C-601", None, "rcp-3f44-0601", "1c-in-000323"),
    "LOT-V-591": ("FL-100-VALVE", "SUP-8", 24, "C-591", None, "rcp-3f44-0591", "1c-in-000308"),
    "LOT-ZF-202": ("FL-100-FLANGE", "SUP-1", 14, "C-202", "5530", "rcp-3f44-0202", "1c-in-000341"),
    "LOT-R-118": ("FL-100-RING", "SUP-2", 12, "C-118", None, "rcp-3f44-0118", "1c-in-000342"),
}
# Маркировка (строка 34 ред-тима) [П]: до мехобработки у заготовки фланца — бирка на таре (станок срезал бы код);
# код DataMatrix наносят после мехобработки на нерабочую поверхность — перемаркировка (Е-07 ещё раз), ОТК сверяет на ЗТ-2.
BLANK_TYPES = ("FL-100-FLANGE",)


def carrier_before_machining(item):
    return {"type": "tag_qr", "value": "TAG:" + item}   # бирка с QR на таре (схема v0.3, узел V4)


def carrier_dm(item):
    return {"type": "dpm_datamatrix", "value": "DM:" + item}


def lot_received(g, t, lot, tags=()):
    it, sup, qty, cert, heat, ext, msg = LOTS[lot]
    p = OrderedDict([("message_id", msg), ("lot_id", lot), ("lot_ext_id", ext), ("supplier_id", sup),
                     ("item_type_id", it), ("qty", qty), ("certificate_no", cert)])
    if heat:
        p["heat_no"] = heat
    p["warehouse_id"] = "WH-SK"
    g.ev(t, "ERP-1C", "erp.batch_received", lot=lot, payload=p, tags=tags, label="erp/lot/" + lot)


# v0.3.1 (решение пользователя 26.09): выборочный рентген колец новой партии на входе — 1 кольцо из партии [ПП].
# Какое кольцо попало в выборку — задано явно (П-117 — К-104: без поры; пора потом найдена в К-101 после сварки).
XR_SAMPLE = {"LOT-R-117": "R-104"}


def iqc(g, t0, lot, pieces, camera_each, mark, zt_at, tags=(), q_range=(0.88, 0.95), zones=None, cam_q=None):
    """Входной контроль партии: Е-04, Е-05 (документы, лаборатория, у колец — рентген 1 кольца [ПП]),
    Е-06 (камера по детали или выборка), Е-07, ЗТ-1 (шаг ОТК-2)."""
    it, sup, qty, cert, heat, ext, msg = LOTS[lot]
    g.ev(t0, "TERM-IQC", "batch.registered", lot=lot,
         payload=OrderedDict([("lot_id", lot), ("qty_actual", qty), ("packaging", "intact"),
                              ("certificate_present", True), ("operator_id", "QC-02")]),
         tags=tags, label="iqc/%s/registered" % lot, persona="QC-02")
    p = OrderedDict([("checkpoint_id", "CP-IQC"), ("phase", "incoming"), ("method", "documents"),
                     ("inspection_result", "no_defect_found"), ("certificate_no", cert)])
    if heat:
        p["heat_no"] = heat
    p["discrepancies"] = []
    p["operator_id"] = "QC-02"
    docs = g.ev(t0 + M(5), "TERM-IQC", "inspection.result", lot=lot, payload=p, tags=tags,
                label="iqc/%s/documents" % lot, persona="QC-02")
    basis = [docs["label"]]
    if it in ("FL-100-FLANGE", "FL-100-RING"):
        # лабораторный контроль образцов по плану входного контроля [ПП] (ред-тим, строка 48; схема v0.3, узел VL).
        # Объёмного контроля колец в плане нет — это честная граница, из неё мера в S02
        tests = [OrderedDict([("test", "chemical_composition"), ("result", "in_spec")]),
                 OrderedDict([("test", "mechanical_properties"), ("result", "in_spec")])]
        if it == "FL-100-FLANGE":
            tests.append(OrderedDict([("test", "ultrasonic_per_kd"), ("result", "no_defect_found")]))
        lab = g.ev(t0 + M(7), "TERM-IQC", "inspection.result", lot=lot,
                   payload=OrderedDict([("checkpoint_id", "CP-IQC"), ("phase", "incoming"), ("method", "lab"),
                                        ("protocol_no", "LAB-" + lot[4:]), ("sample_size", 2), ("tests", tests),
                                        ("volumetric_control", "per_plan" if it == "FL-100-FLANGE" else "xray_sample_1_per_lot"),
                                        ("inspection_result", "no_defect_found"), ("operator_id", "QC-02")]),
                   tags=tags, label="iqc/%s/lab" % lot, persona="QC-02")
        basis.append(lab["label"])
    if it == "FL-100-RING" and pieces:
        # узел VX: рентген тела одного кольца партии; камера КТ-1 пор не видит. Выборка чистая ≠ вся партия чистая
        smp = XR_SAMPLE.get(lot, pieces[0])
        xs = g.ev(t0 + M(9), "XR-1", "inspection.result", item=smp,
                  payload=OrderedDict([("checkpoint_id", "CP-IQC"), ("phase", "incoming"), ("method", "xray"),
                                       ("recipe_id", "RCP-CP-IQC-XR@1"), ("lot_id", lot),
                                       ("conclusion_no", "RK-%s-IQC" % t0.astimezone(MSK).strftime("%m%d")),
                                       ("inspector_id", "NDT-01"), ("inspector_level", "II"), ("standard", "GOST 7512"),
                                       ("sample", OrderedDict([("size", 1), ("lot_qty", qty), ("rule", "1 кольцо из новой партии [ПП]")])),
                                       ("inspection_result", "no_defect_found"), ("zones_inspected", ["RING-BODY"]),
                                       ("defects", [])]),
                  tags=tags, label="iqc/%s/xr-sample" % lot,
                  note="узел VX схемы v0.3.1: выборочный рентген 1 кольца [ПП]; остальные кольца партии рентгеном не проверены")
        basis.insert(2, xs["label"])
    t = t0 + M(10)
    zones = zones or {"FL-100-FLANGE": ["EDGE", "FACE", "BORE"], "FL-100-RING": ["RING-BODY"],
                      "FL-100-COVER": ["COVER"]}.get(it, ["SURFACE"])
    if camera_each and pieces:
        for pc in pieces:
            c = camera(g, t, "CAM-IQC", pc, "CP-IQC", "incoming", zones, tags=tags, lot=None,
                       q=cam_q, label="iqc/%s/kt1" % pc, extra={"lot_id": lot})
            basis.append(c["label"])
            t = t + S(90)
    elif it not in ("WELD-WIRE",):
        c = camera(g, t, "CAM-IQC", None, "CP-IQC", "incoming", zones, tags=tags, lot=lot,
                   label="iqc/%s/kt1-sample" % lot, extra={"lot_id": lot, "sample_size": min(5, qty)})
        basis.append(c["label"])
        t = t + M(5)
    if mark and pieces:
        for pc in pieces:
            if it in BLANK_TYPES:
                # заготовка фланца: до мехобработки — бирка на таре (код станок срезал бы) [П]
                car, meth = carrier_before_machining(pc), "tag_on_container"
            else:
                car, meth = carrier_dm(pc), "laser_dpm"
            g.ev(t, "TERM-STK", "item.marked", item=pc,
                 payload=OrderedDict([("item_id", pc), ("lot_id", lot), ("carrier", car),
                                      ("method", meth), ("operator_id", "MRK-01")]),
                 tags=tags, label="mark/" + pc, persona="MRK-01")
            t = t + S(60)
    gate(g, max(zt_at, t + M(2)), "QC-02", "ZT-1", lot=lot, tags=tags, basis=basis[:3] + (["…"] if len(basis) > 3 else []),
         label="ZT-1/" + lot)
    return t


def issue(g, t, lot, qty, to_station, items=None, components=None, tags=(), label=None, note=None):
    p = OrderedDict([("lot_id", lot), ("qty", qty), ("to_station", to_station), ("issued_by", "STK-01")])
    if components:
        p["component_ids"] = components
    if items:
        p["for_items"] = items
    return g.ev(t, "TERM-STK", "batch.issued", lot=lot, payload=p, tags=tags, label=label, persona="STK-01",
                hints=(items or []) + (components or []), note=note)


def register(g, t, item, order_id, lot, tags=(), kd="B", tp=3):
    g.ev(t, "TERM-STK", "item.registered", item=item,
         payload=OrderedDict([("item_id", item), ("item_type_id", "FL-100-ASSY"), ("serial", item),
                              ("order_id", order_id), ("batch_id", lot), ("route_id", "ROUTE-FL-100@%d" % tp),
                              ("kd_revision", kd), ("tp_revision", tp),
                              ("carrier", carrier_before_machining(item)),
                              ("registered_by", "STK-01")]),
         tags=tags, label="reg/" + item, persona="STK-01")


# ----------------------------------------------------------------------------- мехобработка
def machining(g, item, start, dur, tags=(), kt2_at=None, reported=True, tool_state=None, finish_at=None, kt2_kwargs=None,
              prog_rev="3", kt2_manual=False):
    run = "MO-%s-1" % num(item)
    oper = "OP-CNC-11" if shift_of(start) == "A" else "OP-CNC-12"
    end = start + M(dur)
    op_start(g, start, "TERM-MC-1", item, run, "M1", "ST-MC-CNC", oper, "CNC-1", tags=tags,
             extra={"program_id": "UP-FL-100-01", "program_revision": prog_rev})
    r = rnd("cnc|" + item)
    used = tool_state[0] if tool_state else 70
    g.ev(start + S(30), "CNC-1", "machine.state",
         payload=OrderedDict([("equipment_id", "CNC-1"), ("state", "running"), ("mode", "auto"),
                              ("program_id", "UP-FL-100-01"), ("program_revision", prog_rev),
                              ("tool", {"tool_id": "T12", "life_used": used, "life_limit": 75})]),
         tags=tags, hints=[item], label="cnc/%s/running" % item)
    g.ev(end, "CNC-1", "machine.parameters",
         payload=OrderedDict([("equipment_id", "CNC-1"), ("window_start", iso(start + S(30))), ("window_s", dur * 60 - 30),
                              ("parameters", {"spindle_load_pct": {"avg": r.randint(52, 60), "max": r.randint(74, 86)},
                                              "feed_override_pct": 100})]),
         tags=tags, hints=[item], label="cnc/%s/summary" % item)
    g.ev(end + S(5), "CNC-1", "machine.state",
         payload=OrderedDict([("equipment_id", "CNC-1"), ("state", "idle"), ("reason", "cycle_complete"),
                              ("reported_cycle_time", {"value": dur, "unit": "min"})]),
         tags=tags, hints=[item], label="cnc/%s/idle" % item)
    rep = OrderedDict([("value", dur), ("unit", "min"), ("meaning", "active"), ("source_equipment_id", "CNC-1")]) \
        if reported else None
    # finish_at — отметка «закончил» на терминале позже обычного (S16: оператор нажал с опозданием)
    op_finish(g, finish_at or end + M(1), "TERM-MC-1", item, run, oper, reported=rep, tags=tags)
    # перемаркировка после мехобработки [П] (строка 34 ред-тима): бирка → код DataMatrix на нерабочей поверхности
    g.ev(end + M(4), "TERM-MC", "item.marked", item=item,
         payload=OrderedDict([("item_id", item), ("carrier", carrier_dm(item)),
                              ("previous_carrier", carrier_before_machining(item)), ("method", "laser_dpm"),
                              ("zone", "MARKING"), ("reason", "remark_after_machining"),
                              ("read_back", OrderedDict([("decoded", carrier_dm(item)["value"]), ("grade", "B"),
                                                         ("match_previous", True)])),
                              ("operator_id", "MRK-01")]),
         tags=tags, label="remark/" + item, persona="MRK-01")
    k = kt2_at or end + M(8)
    if kt2_manual:
        manual_kt2(g, item, k, tags)
        if tool_state is not None:
            tool_state[0] = used + 1 if used < 75 else 1
        return end
    kw = dict(q=0.94 if item == "F-001" else (0.93 if item == "F-017" else None),
              conf=0.96 if item == "F-001" else None, label="kt2/" + item,
              extra={"comparison_to_before": "no_change", "lighting": "low_angle"})
    kw.update(kt2_kwargs or {})
    camera(g, k, "CAM-MO", item, "CP-MO", "after_operation", ["EDGE", "FACE", "BORE"], tags=tuple(tags) + tuple(kw.pop("tags", ())),
           **kw)
    if tool_state is not None:
        tool_state[0] = used + 1 if used < 75 else 1
    return end


def cmm(g, item, t_end, tags=(), override=None, label=None, extra=None, nominals=None):
    """override: {char_id: actual} — значение вне допуска (фон v0.3.1, геометрия); label — повторный замер;
    nominals: {char_id: (номинал, минус, плюс)} — другая ревизия КД (S19: D2 = 252,00 по ревизии «В»)."""
    r = rnd("cmm|" + item + ("" if label is None else "|" + label))
    chars = []
    base = [("D1", 320.00, 0.10, 0.10), ("D2", 250.00, 0.0, 0.05), ("H1", 12.00, 0.05, 0.05),
            ("F1", 0.00, 0.0, 0.05), ("P1", 0.00, 0.0, 0.10)]
    if nominals:
        base = [(c,) + tuple(nominals[c]) if c in nominals else (c, n, a, b) for (c, n, a, b) in base]
    for cid, nom, tm, tp in base:
        lo, hi = nom - tm, nom + tp
        act = round(r.uniform(lo + (hi - lo) * 0.2, hi - (hi - lo) * 0.2), 3)
        if override and cid in override:
            act = override[cid]
        res_c = "in_tolerance" if lo - 1e-9 <= act <= hi + 1e-9 else "out_of_tolerance"
        chars.append(OrderedDict([("char_id", cid), ("nominal", nom), ("tol_minus", tm), ("tol_plus", tp),
                                  ("actual", act), ("unit", "mm"), ("result", res_c)]))
    bad = [c for c in chars if c["result"] != "in_tolerance"]
    pl = OrderedDict([("checkpoint_id", "CP-MO"), ("phase", "after_operation"), ("method", "cmm"),
                      ("recipe_id", "RCP-CP-MO-CMM@1"), ("operator_id", "OP-CMM-01"),
                      ("item_identification", OrderedDict([("method", "pick_from_expected_list"),
                                                           ("checked_against", carrier_dm(item)["value"])])),
                      ("calibration_id", "CAL-CMM-1-2026-06"), ("calibration_valid", True),
                      ("inspection_result", "defect_found" if bad else "no_defect_found"), ("characteristics", chars)])
    if bad:
        pl["defects"] = [OrderedDict([("defect_type", "dimension_out_of_tolerance"), ("char_id", c["char_id"]),
                                      ("severity", "major"), ("measurable", True)]) for c in bad]
    if extra:
        pl.update(extra)
    if override is not None or label is not None or nominals is not None:
        return g.ev(t_end, "CMM-1", "inspection.result", item=item, payload=pl, tags=tags, label=label or "cmm/" + item)
    return g.ev(t_end, "CMM-1", "inspection.result", item=item,
                payload=OrderedDict([("checkpoint_id", "CP-MO"), ("phase", "after_operation"), ("method", "cmm"),
                                     ("recipe_id", "RCP-CP-MO-CMM@1"), ("operator_id", "OP-CMM-01"),
                                     # номер фланца на КИМ выбирают из списка ожидаемых изделий, а не вводят руками;
                                     # код на детали сверяют [П] (строка 34; схема v0.3, узел M3)
                                     ("item_identification", OrderedDict([("method", "pick_from_expected_list"),
                                                                          ("checked_against", carrier_dm(item)["value"])])),
                                     ("calibration_id", "CAL-CMM-1-2026-06"), ("calibration_valid", True),
                                     ("inspection_result", "no_defect_found"), ("characteristics", chars)]),
                tags=tags, label="cmm/" + item)


# ----------------------------------------------------------------------------- сварка
def weld(g, w, tags=(), s12=None, cam_override=None):
    """w: dict(item, src, welder, start, end, arc=(a0,a1), ring, ring_lot, wire, prep, run, rework_of, profile,...)"""
    item, src, welder = w["item"], w["src"], w["welder"]
    run = w["run"]
    term = "TERM-WS-1" if src == "WS-1" else "TERM-WS-2"
    station = "ST-WC-P1" if src == "WS-1" else "ST-WC-P2"
    if w.get("prep"):
        op_action(g, w["prep"], term, item, welder, "confirmation",
                  OrderedDict([("step", w.get("prep_step", "edge_prep")), ("text", w.get("prep_text", "кромки подготовлены"))]),
                  tags=tags, label="%s.prep" % run)
    op_action(g, w["start"] - M(2), term, item, welder, "confirmation",
              OrderedDict([("step", "weld_mode_check"), ("text", "режим по карте сверен"), ("program_id", "PS-4")]),
              tags=tags, label="%s.modecheck" % run)
    extra = OrderedDict([("program_id", "PS-4"), ("wire_lot_id", w["wire"]), ("gas_lot", "AR-2026-09"),
                         ("welder_qualification_id", "Q-" + welder)])
    if w.get("rework_zones"):
        extra["rework_zones"] = w["rework_zones"]
    op_start(g, w["start"], term, item, run, "W2", station, welder, src, tags=tags, extra=extra,
             rework_of=w.get("rework_of"))
    if not w.get("rework_of") and w.get("ring"):
        # Е-41: скан фланца и кольца (узел WJ схемы v0.2) — до подготовки кромок W1
        t_scan = (w["prep"] - M(2)) if w.get("prep") else (w["start"] - M(4))
        g.ev(t_scan, term, "assembly.component_linked", item=item,
             payload=OrderedDict([("assembly_item_id", item), ("component_id", w["ring"]),
                                  ("component_type_id", "FL-100-RING"), ("lot_id", w["ring_lot"]), ("position", 2),
                                  ("link_id", "W-1"), ("binding_method", w.get("link_method", "scan")), ("operator_id", welder)]),
             tags=tags, label="%s.link" % run, persona=welder, hints=[item, w["ring"]])
    for (ps, pe) in w.get("pauses", []):
        g.ev(ps, term, "operation.paused", item=item,
             payload=OrderedDict([("operation_run_id", run), ("reason", w.get("pause_reason", "setup")),
                                  ("operator_id", welder)]), tags=tags, corr=run, label="%s.pause" % run, persona=welder)
        if w.get("manual_override"):
            op_action(g, ps + M(1), term, item, welder, "manual_override", w["manual_override"], run=run, tags=tags,
                      label="%s.override" % run)
        g.ev(pe, term, "operation.resumed", item=item,
             payload=OrderedDict([("operation_run_id", run), ("operator_id", welder)]), tags=tags, corr=run,
             label="%s.resume" % run, persona=welder)
    op_finish(g, w["end"], term, item, run, welder, tags=tags)
    g.welds.append(w)
    # КТ-3: два ракурса, через 5 минут после окончания
    t = w["end"] + M(5)
    if cam_override:
        cam_override(g, t)
    else:
        for k, cam in enumerate(("CAM-WS-1", "CAM-WS-2")):
            kw = dict(s12[cam]) if (s12 and cam in s12) else {}
            if item == "F-001" and not w.get("rework_of"):
                kw.update(q=0.93, conf=0.95)   # S01: «качество 0,93; уверенность 0,95» (S01-07)
            ex = {"comparison_to_before": "new_zone_created_by_operation"}
            ex.update(kw.pop("extra", {}) or {})
            camera(g, t + S(10 * k), cam, item, "CP-WELD", "after_operation", SEGS, tags=tuple(tags) + tuple(kw.pop("tags", ())),
                   label="kt3/%s/%s" % (run, cam), extra=ex, **kw)
    return run


def excavation(g, item, t0, welder, of_run, zones, what, t_check, tags=(), dur=25, source="TERM-WC"):
    """Схема v0.3: WL1 «разметка и выборка дефекта» (новое выполнение WL-…) и WL2 «контроль выборки» (ОТК-1, осмотр)."""
    n = sum(1 for e in g.events if e["type"] == "operation.started" and e["payload"].get("operation_code") == "WL1"
            and e["item"] == item) + 1
    run = "WL-%s-%d" % (num(item), n)
    op_start(g, t0, source, item, run, "WL1", "ST-WC-ZT3", welder, tags=tags,
             extra={"excavation_of_run_id": of_run, "zones": zones, "what": what})
    op_finish(g, t0 + M(dur), source, item, run, welder, tags=tags)
    g.ev(t_check, "TERM-WC", "inspection.result", item=item,
         payload=OrderedDict([("checkpoint_id", "CP-WELD"), ("phase", "after_excavation"), ("method", "visual"),
                              ("operator_id", "QC-01"), ("excavation_run_id", run), ("zones_inspected", zones),
                              ("inspection_result", "no_defect_found"),
                              ("observation", "выборка чистая: дефект удалён полностью")]),
         tags=tags, label="wl2/%s" % run, persona="QC-01")
    return run


def xray(g, t, item, no, defects=None, zones=None, tags=(), label=None, note=None, extra=None):
    res = "defect_found" if defects else "no_defect_found"
    p = OrderedDict([("checkpoint_id", "CP-WELD"), ("phase", "after_operation"), ("method", "xray"),
                     ("recipe_id", "RCP-CP-WELD-XR@1"), ("conclusion_no", no), ("inspector_id", "NDT-01"),
                     ("inspector_level", "II"), ("standard", "GOST 7512"), ("inspection_result", res),
                     ("zones_inspected", zones or SEGS + ["RING-BODY"]), ("defects", defects or [])])
    if extra:
        p.update(extra)
    return g.ev(t, "XR-1", "inspection.result", item=item, payload=p, tags=tags, label=label or "xr/%s/%s" % (item, no),
                note=note)


# ----------------------------------------------------------------------------- сборка, испытание, выпуск
def kit_issue(g, t, item, cover, seal_lot, fs_lot, cover_lot, tags=(), label=None):
    issue(g, t, cover_lot, 1, "ST-AC-ASM", items=[item], components=[cover], tags=tags,
          label=label or "kit/%s/cover" % item)
    issue(g, t + S(20), seal_lot, 1, "ST-AC-ASM", items=[item], tags=tags, label=(label or "kit/%s" % item) + "/seal")
    issue(g, t + S(40), fs_lot, 12, "ST-AC-ASM", items=[item], tags=tags, label=(label or "kit/%s" % item) + "/fasteners")
    valve, vlot = valve_of(item)   # клапан выравнивания давления — установка оборудования [П] (строка 20)
    issue(g, t + S(50), vlot, 1, "ST-AC-ASM", items=[item], components=[valve], tags=tags,
          label=(label or "kit/%s" % item) + "/valve")


VALVE_OVERRIDE = {"F-042": ("V-021", "LOT-V-601")}   # S18: Ф-042 взамен списанного Ф-021 — его клапан и крышка не ставились


def valve_of(item):
    if item in VALVE_OVERRIDE:
        return VALVE_OVERRIDE[item]
    n = int(num(item))
    return ("V-%03d" % n, "LOT-V-591") if n >= 200 else ("V-%03d" % n, "LOT-V-601")


def equipment_install(g, item, te, tags=()):
    """Установка оборудования (кейс §1.1) [П, ПП]: клапан выравнивания давления КВД в штуцер крышки.
    Отдельное выполнение EQ-xxx-1 внутри сборки: скан номера клапана до установки, затяжка J-2 ключом КЛ-1,
    контровка проволокой; отметка ОТК — на ЗТ-4 (основание — журнал ключа и камера КТ-4, зона VALVE)."""
    run = "EQ-%s-1" % num(item)
    valve, vlot = valve_of(item)
    op_start(g, te, "TERM-ASM", item, run, "EQ", "ST-AC-ASM", "ASM-01", tags=tags,
             extra={"equipment_item_type_id": "FL-100-VALVE", "link_id": "J-2"})
    g.ev(te + S(20), "TERM-ASM", "assembly.component_linked", item=item,
         payload=OrderedDict([("assembly_item_id", item), ("component_id", valve), ("component_type_id", "FL-100-VALVE"),
                              ("lot_id", vlot), ("qty", 1), ("position", 6), ("link_id", "J-2"),
                              ("binding_method", "scan_before_install"), ("operator_id", "ASM-01")]),
         tags=tags, label="%s.link.FL-100-VALVE" % run, persona="ASM-01", hints=[item, valve])
    r = rnd("twv|" + item)
    g.ev(te + M(3), "TW-1", "machine.parameters",
         payload=OrderedDict([("equipment_id", "TW-1"), ("joint_id", "J-2"), ("item_hint", item),
                              ("setpoint", {"torque_nm": 12.0, "tolerance_pct": 10}),
                              ("result", OrderedDict([("torque_nm", round(r.uniform(11.6, 12.5), 1)), ("status", "ok")]))]),
         tags=tags, hints=[item], label="torque/%s/J-2" % item)
    op_action(g, te + M(4), "TERM-ASM", item, "ASM-01", "confirmation",
              OrderedDict([("step", "valve_lockwire"), ("text", "клапан затянут, законтрен проволокой")]), run=run,
              tags=tags, label="%s.lockwire" % run)
    op_finish(g, te + M(5), "TERM-ASM", item, run, "ASM-01", tags=tags)
    return run


def assembly(g, item, t0, cover, cover_lot, seal_lot, fs_lot, tags=(), zt41=None, zt4=None, kt4=None, plan=None,
             stop_after_uv=False):
    """Сборка AS-xxx-1: сканы до установки → уплотнение → КТ-СР → ЗТ-СР → крышка → крепёж → затяжка → установка
    оборудования (EQ, узел AO) → КТ-4 → ЗТ-4 (схема v0.3)."""
    run = "AS-%s-1" % num(item)
    p = plan or {}
    op_start(g, t0, "TERM-ASM", item, run, "ASM", "ST-AC-ASM", "ASM-01", tags=tags)
    t = t0 + M(1)
    for comp, ctype, lot, pos, link, qty in ((cover, "FL-100-COVER", cover_lot, 3, "J-1", 1),
                                             (None, "FL-100-SEAL", seal_lot, 4, "S-1", 1),
                                             (None, "FL-100-FASTENER", fs_lot, 5, "J-1", 12)):
        pl = OrderedDict([("assembly_item_id", item)])
        if comp:
            pl["component_id"] = comp
        pl["component_type_id"] = ctype
        pl["lot_id"] = lot
        pl["qty"] = qty
        pl["position"] = pos
        pl["link_id"] = link
        pl["binding_method"] = "scan_before_install"
        pl["operator_id"] = "ASM-01"
        g.ev(t, "TERM-ASM", "assembly.component_linked", item=item, payload=pl, tags=tags,
             label="%s.link.%s" % (run, ctype), persona="ASM-01", hints=[item] + ([comp] if comp else []))
        t = t + S(20)
    ts = p.get("seal", t0 + M(10))
    op_action(g, ts, "TERM-ASM", item, "ASM-01", "confirmation", OrderedDict([("step", "seal_install"),
              ("text", "уплотнение установлено")]), run=run, tags=tags, label="%s.seal" % run)
    sk = p.get("skip_check")
    if sk:
        # S17: сборщик сам отметил, что обязательную проверку сделать не смог (Е-21); проверку делают позже (Е-23)
        stg = tuple(tags) + ("S17",)
        op_action(g, sk["skip_at"], "TERM-ASM", item, "ASM-01", "check_skipped",
                  OrderedDict([("step", "seal_seating_check"), ("check", "посадка уплотнения в канавке, щуп 0,05 мм"),
                               ("mandatory", True), ("reason", sk["reason"])]), run=run, tags=stg, label="S17/check-skipped")
        op_action(g, sk["done_at"], "TERM-ASM", item, "ASM-01", "confirmation",
                  OrderedDict([("step", "seal_seating_check"), ("text", "посадка уплотнения проверена щупом 0,05 мм: в норме"),
                               ("closes_skip_label", "S17/check-skipped")]), run=run, tags=stg, label="S17/check-done")
    tu = p.get("uv", ts + M(5))
    camera(g, tu, "CAM-UV", item, "CP-ASM-ZONE", "before_zone_closure", ["SEAL-GROOVE"], tags=tags,
           label="ktsr/" + item, extra={"lighting": "uv"})
    if stop_after_uv:
        return run
    tz1 = zt41 or tu + M(10)
    # ЗТ-СР (скрытые работы, схема v0.3, узел A3): зона до крышки + счёт инструмента, заходившего в полость
    gate(g, tz1, "QC-02", "ZT-SR", item=item, tags=tags,
         extra={"zone": "SEAL-GROOVE", "tools_counted": {"entered_cavity": 2, "returned": 2}},
         basis=["ktsr/" + item], label="ZT-SR/" + item)
    tc = p.get("cover", tz1 + M(5))
    op_action(g, tc, "TERM-ASM", item, "ASM-01", "confirmation", OrderedDict([("step", "cover_install"),
              ("text", "крышка установлена")]), run=run, tags=tags, label="%s.cover" % run)
    tf = p.get("fasteners", tc + M(8))
    op_action(g, tf, "TERM-ASM", item, "ASM-01", "confirmation", OrderedDict([("step", "fasteners_install"),
              ("text", "крепёж установлен, 12 шт.")]), run=run, tags=tags, label="%s.fasteners" % run)
    tt = p.get("torque", tf + M(12))
    r = rnd("tw|" + item)
    bolts = [OrderedDict([("position", "BOLT-%d" % b), ("torque_nm", round(r.uniform(7.7, 8.4), 1)),
                          ("angle_deg", r.randint(24, 36)), ("status", "ok")]) for b in range(1, 13)]
    g.ev(tt, "TW-1", "machine.parameters",
         payload=OrderedDict([("equipment_id", "TW-1"), ("joint_id", "J-1"), ("item_hint", item),
                              ("pattern", "cross"), ("passes_pct", [30, 70, 100]),
                              ("setpoint", {"torque_nm": 8.0, "tolerance_pct": 10}), ("bolts", bolts)]),
         tags=tags, hints=[item], label="torque/" + item)
    op_action(g, tt + M(1), "TERM-ASM", item, "ASM-01", "confirmation", OrderedDict([("step", "tightening"),
              ("text", "затяжка по схеме крест-накрест, метки нанесены")]), run=run, tags=tags,
              label="%s.tightening" % run)
    # установка оборудования (узел AO схемы v0.3) — после затяжки крепежа, до камеры КТ-4
    te = tt + M(2)
    equipment_install(g, item, te, tags=tags)
    tk = kt4 or te + M(7)
    assert te + M(5) < tk, ("установка оборудования не помещается до камеры КТ-4", item)
    k4 = dict(p.get("kt4_kwargs", {}))
    zones4 = ["BOLT-%d" % b for b in range(1, 13)] + ["COVER", "VALVE"]
    camera(g, tk, "CAM-ASM", item, "CP-ASM", "after_assembly", zones4, tags=tuple(tags) + tuple(k4.pop("tags", ())),
           label="kt4/" + item, **k4)
    if p.get("kt4_resend"):
        camera(g, p["kt4_resend"], "CAM-ASM", item, "CP-ASM", "after_assembly", zones4, tags=tuple(tags) + ("S12",),
               label="kt4/%s/resend" % item, note="повторная отправка анализатора после сообщения с исходом maybe (S12)")
    tfin = p.get("finish", tk + M(2))
    op_finish(g, tfin, "TERM-ASM", item, run, "ASM-01", tags=tags)
    if zt4 is not False:
        gate(g, zt4 or tfin + M(5), "QC-02", "ZT-4", item=item, tags=tags,
             basis=[("kt4/%s/resend" % item) if p.get("kt4_resend") else ("kt4/" + item), "torque/" + item,
                    "torque/%s/J-2" % item], label="ZT-4/" + item)   # сообщение «maybe» ушло в карантин — основание повторная отправка
    return run


def leak_test(g, item, t0, tags=(), zt5=None, dur=60, signers=None):
    run = "HT-%s-1" % num(item)
    op_start(g, t0, "TERM-LT", item, run, "T1", "ST-AC-LT", "TST-01", "LT-1", tags=tags,
             extra={"stand_verification_valid": True, "control_leak_checked": True})
    r = rnd("lt|" + item)
    tp = t0 + M(dur - 10)
    g.ev(tp, "LT-1", "machine.parameters",
         payload=OrderedDict([("equipment_id", "LT-1"), ("item_hint", item), ("test_pressure_mpa", 0.15),
                              ("hold_time_min", 30), ("background_pa_m3_s", 2.0e-09)]), tags=tags, hints=[item],
         label="lt/%s/params" % item)
    leak = round(r.uniform(1.5, 3.5), 1)
    g.ev(tp + M(5), "LT-1", "inspection.result", item=item,
         payload=OrderedDict([("checkpoint_id", "CP-LEAK"), ("phase", "test"), ("method", "leak_test"),
                              ("recipe_id", "RCP-CP-LEAK@1"), ("operator_id", "TST-01"),
                              ("inspection_result", "no_defect_found"), ("verdict", "tight"),
                              ("leak_rate_pa_m3_s", float("%.1fe-07" % leak)), ("leak_rate_norm_pa_m3_s", 1.0e-06),
                              ("defects", [])]),
         tags=tags, label="lt/%s/result" % item)
    op_finish(g, t0 + M(dur), "TERM-LT", item, run, "TST-01", tags=tags)
    if zt5 is not False:
        gate(g, zt5 or t0 + M(dur + 5), "QC-02", "ZT-5", item=item, tags=tags, signers=signers or ["TST-01", "QC-02"],
             basis=["lt/%s/result" % item], label="ZT-5/" + item)
    return t0 + M(dur)


def final(g, item, t_kt5, t_pres, t_zt6, t_rel, tags=(), q=None):
    camera(g, t_kt5, "CAM-FQC", item, "CP-FQC", "final", ["COVER", "MARKING", "FACE"], tags=tags, q=q,
           label="kt5/" + item, extra={"views": 4})
    g.ev(t_pres, "TERM-AC", "item.presented", item=item,
         payload=OrderedDict([("item_id", item), ("presentation_no", 1), ("presented_to", ["QC", "CUSTOMER_REP"]),
                              ("presented_by", "MST-03"), ("gate", "ZT-6")]), tags=tags, label="present/%s/ZT-6" % item,
         persona="MST-03")
    gate(g, t_zt6, "QC-02", "ZT-6", item=item, tags=tags, signers=["QC-02", "CR-01"], basis=["kt5/" + item],
         label="ZT-6/" + item)
    g.ev(t_rel, "TERM-STK", "item.released", item=item,
         payload=OrderedDict([("item_id", item), ("warehouse_id", "WH-FG"), ("released_by", "STK-01"),
                              ("concession_no", None)]), tags=tags, label="release/" + item, persona="STK-01")


# ----------------------------------------------------------------------------- расписания
def ring_of(item):
    n = int(num(item))
    if n >= 200:
        return "R-%03d" % (n + 100), "LOT-R-108"
    special = {19: "R-101", 27: "R-102", 29: "R-103"}
    if n in special:
        return special[n], "LOT-R-117"
    seq = [k for k in range(1, 37) if k not in special]
    return "R-%03d" % (seq.index(n) + 1), "LOT-R-116"


def cover_of(item):
    n = int(num(item))
    return ("C-%03d" % n, "LOT-C-211") if n >= 200 else ("C-%03d" % n, "LOT-C-301")


def lots_of(item):
    n = int(num(item))
    if n >= 200:
        return dict(blank="LOT-ZF-111", seal="LOT-SL-391", fs="LOT-FS-481", wire="LOT-W-87", order="ORD-0911")
    return dict(blank="LOT-ZF-201", seal="LOT-SL-401", fs="LOT-FS-501", wire="LOT-W-88", order="ORD-0917")


def cmm_queue(items_end, first_day_start=None, pins=None):
    """Очередь КИМ: только смена А, 25 мин, обед 12:30–13:00. Возвращает {item: (start, end)}."""
    pins = pins or {}
    free = None
    out = {}
    for item, mend in items_end:
        t = mend + M(10)
        if item in pins:
            s = pins[item]
            out[item] = (s, s + M(25))
            free = max(free or s, s + M(25))
            continue
        if free and free > t:
            t = free
        while True:
            d = day(t)
            a0, a1 = at(d, "08:00"), at(d, "16:30")
            l0, l1 = at(d, "12:30"), at(d, "13:00")
            if d in WEEKEND or t >= a1:
                t = at(next_workday(d), "08:00")
                continue
            if t < a0:
                t = a0
            if t < l1 and t + M(25) > l0:
                t = l1
                continue
            if t + M(25) > a1:
                t = at(next_workday(d), "08:00")
                continue
            break
        out[item] = (t, t + M(25))
        free = t + M(25)
    return out


def zt_time(t):
    """Решение ОТК в смене А: если вне смены — следующее утро 08:05."""
    d = day(t)
    if d in WEEKEND or t >= at(d, "16:25"):
        return at(next_workday(d), "08:05")
    if t < at(d, "08:00"):
        return at(d, "08:05")
    return t


def transfer_run(t, fri_morning_only=True):
    """Ближайший рейс межцехового транспорта не раньше t (+5 мин на оформление)."""
    t = t + M(5)
    d = day(t)
    while True:
        if d not in WEEKEND:
            runs = ["07:50", "10:00"] if (fri_morning_only and d.weekday() == 4) else ["07:50", "10:00", "14:00", "16:00"]
            for hm in runs:
                r = at(d, hm)
                if r >= t:
                    return r
        d = d + dt.timedelta(days=1)
        t = at(d, "00:00")


# Сварки главной недели (scenarios-flange.md §2.6, S01–S05); интервалы, которых нет в описании, — [П].
MAIN_WELDS = [
    # item, src, welder, start, end
    ("F-001", "WS-2", "WLD-02", "2026-09-21 09:00", "2026-09-21 09:45"),
    ("F-002", "WS-1", "WLD-02", "2026-09-21 09:50", "2026-09-21 10:30"),
    ("F-003", "WS-2", "WLD-02", "2026-09-21 10:35", "2026-09-21 11:15"),
    ("F-004", "WS-1", "WLD-02", "2026-09-21 11:20", "2026-09-21 12:00"),
    ("F-005", "WS-2", "WLD-02", "2026-09-21 12:40", "2026-09-21 13:20"),
    ("F-006", "WS-2", "WLD-02", "2026-09-21 14:10", "2026-09-21 14:55"),
    ("F-007", "WS-1", "WLD-02", "2026-09-21 14:58", "2026-09-21 15:35"),
    ("F-008", "WS-2", "WLD-02", "2026-09-21 15:40", "2026-09-21 16:20"),
    ("F-009", "WS-1", "WLD-03", "2026-09-21 16:35", "2026-09-21 17:15"),
    ("F-010", "WS-2", "WLD-03", "2026-09-21 17:30", "2026-09-21 18:10"),
    ("F-011", "WS-1", "WLD-03", "2026-09-21 18:20", "2026-09-21 19:00"),
    ("F-221", "WS-2", "WLD-03", "2026-09-21 19:50", "2026-09-21 20:30"),
    ("F-013", "WS-1", "WLD-03", "2026-09-21 20:40", "2026-09-21 21:20"),
    ("F-012", "WS-2", "WLD-03", "2026-09-21 22:10", "2026-09-21 22:50"),
    ("F-018", "WS-1", "WLD-03", "2026-09-21 23:00", "2026-09-21 23:40"),
    ("F-014", "WS-2", "WLD-02", "2026-09-22 08:20", "2026-09-22 08:50"),
    ("F-222", "WS-2", "WLD-02", "2026-09-22 08:55", "2026-09-22 09:15"),
    ("F-016", "WS-2", "WLD-02", "2026-09-22 09:20", "2026-09-22 10:05"),
    ("F-015", "WS-2", "WLD-02", "2026-09-22 10:15", "2026-09-22 11:00"),
    ("F-020", "WS-1", "WLD-02", "2026-09-22 11:05", "2026-09-22 11:45"),
    ("F-022", "WS-1", "WLD-02", "2026-09-22 11:50", "2026-09-22 12:30"),
    ("F-024", "WS-1", "WLD-02", "2026-09-22 13:15", "2026-09-22 13:55"),
    ("F-026", "WS-1", "WLD-02", "2026-09-22 14:00", "2026-09-22 14:40"),
    ("F-028", "WS-1", "WLD-02", "2026-09-22 14:45", "2026-09-22 15:25"),
    ("F-223", "WS-1", "WLD-02", "2026-09-22 15:28", "2026-09-22 16:00"),
    ("F-030", "WS-1", "WLD-02", "2026-09-22 16:02", "2026-09-22 16:28"),
    ("F-019", "WS-2", "WLD-03", "2026-09-22 16:35", "2026-09-22 17:20"),
    ("F-036", "WS-1", "WLD-03", "2026-09-22 17:22", "2026-09-22 17:58"),
    ("F-027", "WS-1", "WLD-03", "2026-09-22 18:00", "2026-09-22 18:40"),
    ("F-031", "WS-1", "WLD-03", "2026-09-22 18:45", "2026-09-22 19:25"),
    ("F-021", "WS-2", "WLD-03", "2026-09-22 19:40", "2026-09-22 20:25"),
    ("F-032", "WS-1", "WLD-03", "2026-09-22 20:30", "2026-09-22 21:10"),
    ("F-029", "WS-1", "WLD-03", "2026-09-22 21:30", "2026-09-22 22:10"),
    ("F-224", "WS-1", "WLD-03", "2026-09-22 22:15", "2026-09-22 22:50"),
    ("F-033", "WS-1", "WLD-03", "2026-09-22 22:55", "2026-09-22 23:30"),
    ("F-034", "WS-1", "WLD-03", "2026-09-22 23:35", "2026-09-23 00:10"),
    ("F-035", "WS-1", "WLD-03", "2026-09-23 00:12", "2026-09-23 00:47"),
    ("F-023", "WS-2", "WLD-02", "2026-09-23 08:30", "2026-09-23 09:15"),
    ("F-025", "WS-2", "WLD-02", "2026-09-23 09:35", "2026-09-23 10:20"),
    ("F-017", "WS-2", "WLD-02", "2026-09-23 10:40", "2026-09-23 11:00"),
]
# Время подготовки кромок, если оно задано в сценариях (окно до сварки)
PREP = {"F-001": "2026-09-21 08:15", "F-017": "2026-09-23 08:50", "F-021": "2026-09-22 17:30",
        "F-023": "2026-09-23 08:05"}   # F-023: в таблице общих факторов 1 ч 30 мин → в данных 25 мин (смена А с 08:00) [П]
# Токовый профиль: (с какого момента, мин, макс) — ИС-2 «уплыл» с Вт 22.09 10:20:14 (EV-WS2-0412)
DRIFT = {"F-015": [("2026-09-22 10:17:00", 164, 169), ("2026-09-22 10:20:14", 176, 180)],
         "F-019": [("2026-09-22 16:40:20", 177, 181)],
         "F-021": [("2026-09-22 19:42:00", 178, 181)],
         "F-023": [("2026-09-23 08:32:00", 177, 180)],
         "F-025": [("2026-09-23 09:37:00", 176, 181)],
         "F-017": [("2026-09-23 10:41:00", 176, 182), ("2026-09-23 10:54:00", 172, 176)]}
S13_ITEMS = ["F-024", "F-026", "F-028", "F-030", "F-031", "F-032", "F-033", "F-034", "F-035", "F-036", "F-223", "F-224"]
MAIN_MACH = ["F-001", "F-017", "F-002", "F-003", "F-004", "F-005", "F-006", "F-007", "F-008", "F-009", "F-010",
             "F-011", "F-013", "F-012", "F-018", "F-014", "F-016",
             "F-015", "F-020", "F-022", "F-024", "F-026", "F-028", "F-030", "F-019", "F-023",
             "F-025", "F-036", "F-027", "F-031", "F-035", "F-021", "F-032", "F-029", "F-033",
             "F-034"]


def machining_plan(items, day0_start, dur=50, pins_dur=None, gap=5):
    """Станок ЧПУ-1: слоты по порядку; окно рабочего дня D — [D 08:00, D+1 01:00); выходные пропускаются."""
    pins_dur = pins_dur or {}
    plan = {}
    t = day0_start
    for it in items:
        d = pins_dur.get(it, dur)
        while True:
            h = t.astimezone(MSK)
            if h.hour < 1:
                D = h.date() - dt.timedelta(days=1)
            elif h.hour < 8:
                t = at(h.date(), "08:00")
                continue
            else:
                D = h.date()
            if D in WEEKEND:
                t = at(next_workday(D), "08:00")
                continue
            if t + M(d) > at(D + dt.timedelta(days=1), "01:00"):
                t = at(next_workday(D), "08:00")
                continue
            break
        plan[it] = (t, t + M(d))
        t = t + M(d + gap)
        if it == "F-001":
            t = T("2026-09-18 10:10")
    return plan


def reg_times(plan, order_, first_day, first_hm="08:30"):
    """Регистрация (выдача заготовки в производство, Е-09): кладовщик, смена А.
    Первый день — утром того же дня; далее — накануне в 15:00 (выдача впрок) [П]."""
    out = {}
    by_day = defaultdict(list)
    for it in order_:
        s0 = plan[it][0]
        D = shift_key(s0)[0]
        by_day[D].append(it)
    for D, lst in by_day.items():
        if D == first_day:
            base = at(D, first_hm)
        else:
            prev = D - dt.timedelta(days=1)
            while prev in WEEKEND:
                prev = prev - dt.timedelta(days=1)
            base = at(prev, "15:00")
        for k, it in enumerate(lst):
            out[it] = base + M(k)
    return out


# ----------------------------------------------------------------------------- фон v0.3.1: классы дефектов автора
# Решение пользователя 26.09: по одному сигналу на класс, которого нет в главной истории — скол, вмятина, царапина,
# геометрия вне допуска. Фланцы фонового заказа ЗП-0911, неделя 09–14.09 — главную историю не трогают. Путь каждого —
# обычный «сигнал → решение ОТК» (узлы M2S, M3S, W0S, A0S). Два подтверждены — скол и геометрия (доработка по ТП на ЗТ-2,
# без изолятора и 1С), два отклонены с причиной — вмятина и царапина при приёме цехом. Удержание по сигналу → в MES сразу «не выдавать»; снимает решение человека.
BG_CHIP, BG_GEOM, BG_DENT, BG_SCRATCH = "F-203", "F-219", "F-206", "F-213"
BG_CHIP_KT2 = dict(result="defect_found", q=0.92, conf=0.83, tags=("S08",),
                   defects=[OrderedDict([("defect_type", "chip"), ("zone", "EDGE"), ("severity", "minor"),
                                         ("size_estimate_mm", 0.8), ("measurable", False), ("confidence", 0.83)])],
                   extra={"comparison_to_before": "new_after", "lighting": "low_angle"})


def bg_chip(g, it, z):
    """Скол на кромке после станка (КТ-2): подтверждён → ЗТ-2 «доработка по ТП» → зачистка → КТ-2 → ЗТ-2 №2 «годно»."""
    k2 = "kt2/" + it
    t_sig = [e["occ"] for e in g.events if e["label"] == k2][0]
    g.step(t_sig + M(12), "QC-01", "QC", "decision.signal_confirmed", {"item_id": it, "signal_basis_labels": [k2]},
           {"defect_type": "chip", "zone": "EDGE", "severity": "minor",
            "reason": "скол 0,8 мм на кромке под сварку подтверждён осмотром с лупой; общего фактора нет — решение на ЗТ-2",
            "decision_at": "ZT-2"}, tags=("S08",), catalog="E-90", label="S08/bg/confirm-chip")
    gate(g, z, "QC-01", "ZT-2", item=it, basis=[k2, "cmm/" + it], decision="return_for_rework", tags=("S08",),
         extra={"rework": "зачистка скола на кромке по ТП (узел MG → M1): без изолятора и без 1С",
                "hold_released": "containment.hold_released (Е-116): удержание по сигналу снято решением ЗТ-2 → в MES «можно выдавать»"},
         label="ZT-2/" + it)
    run = "MO-%s-2" % num(it)
    op_start(g, z + M(5), "TERM-MC-1", it, run, "M1", "ST-MC-CNC", "OP-CNC-11", tags=("S08",), rework_of="MO-%s-1" % num(it),
             extra={"purpose": "rework_per_tp", "what": "зачистка скола на кромке под сварку"})
    op_finish(g, z + M(20), "TERM-MC-1", it, run, "OP-CNC-11", tags=("S08",))
    camera(g, z + M(25), "CAM-MO", it, "CP-MO", "after_operation", ["EDGE", "FACE", "BORE"], tags=("S08",),
           label="kt2/%s/after-rework" % it, extra={"comparison_to_before": "repaired", "lighting": "low_angle"})
    g.ev(z + M(28), "TERM-MC", "item.presented", item=it,
         payload=OrderedDict([("item_id", it), ("presentation_no", 2), ("presented_to", ["QC"]), ("presented_by", "MST-01"),
                              ("gate", "ZT-2")]), tags=("S08",), label="present/%s/ZT-2#2" % it, persona="MST-01")
    z2 = z + M(30)
    gate(g, z2, "QC-01", "ZT-2", item=it, basis=["kt2/%s/after-rework" % it, "cmm/" + it], presentation=2, tags=("S08",),
         label="ZT-2#2/" + it)
    return z2


def bg_geometry(g, it, t_cmm):
    """Геометрия вне допуска на КИМ: подтверждена → ЗТ-2 «доработка по ТП» (доточка D1 на ЧПУ) → КИМ в допуске → ЗТ-2 №2."""
    cmm(g, it, t_cmm, override={"D1": 320.14}, tags=("S08",))
    g.step(t_cmm + M(7), "QC-01", "QC", "decision.signal_confirmed", {"item_id": it, "signal_basis_labels": ["cmm/" + it]},
           {"defect_type": "dimension_out_of_tolerance", "char_id": "D1", "actual": 320.14, "tolerance": "320 ± 0,10",
            "reason": "КИМ с действующей калибровкой: D1 больше допуска на 0,04 мм; припуск есть — доточка по ТП; общего фактора нет "
                      "(журнал ЧПУ-1: инструмент в ресурсе, программа та же)", "decision_at": "ZT-2"},
           tags=("S08",), catalog="E-90", label="S08/bg/confirm-geometry")
    z = t_cmm + M(10)
    gate(g, z, "QC-01", "ZT-2", item=it, basis=["kt2/" + it, "cmm/" + it], decision="return_for_rework", tags=("S08",),
         extra={"rework": "доточка D1 на ЧПУ по ТП (узел MG → M1): без изолятора и без 1С",
                "hold_released": "containment.hold_released (Е-116): удержание по сигналу снято решением ЗТ-2 → в MES «можно выдавать»"},
         label="ZT-2/" + it)
    run = "MO-%s-2" % num(it)
    t0 = z + M(5)
    op_start(g, t0, "TERM-MC-1", it, run, "M1", "ST-MC-CNC", "OP-CNC-11", "CNC-1", tags=("S08",), rework_of="MO-%s-1" % num(it),
             extra={"purpose": "rework_per_tp", "what": "доточка D1 до допуска", "program_id": "UP-FL-100-01", "program_revision": "3"})
    g.ev(t0 + S(30), "CNC-1", "machine.state",
         payload=OrderedDict([("equipment_id", "CNC-1"), ("state", "running"), ("mode", "auto"), ("program_id", "UP-FL-100-01"),
                              ("program_revision", "3")]), tags=("S08",), hints=[it], label="cnc/%s/running#rework" % it)
    g.ev(t0 + M(15), "CNC-1", "machine.state",
         payload=OrderedDict([("equipment_id", "CNC-1"), ("state", "idle"), ("reason", "cycle_complete")]),
         tags=("S08",), hints=[it], label="cnc/%s/idle#rework" % it)
    op_finish(g, t0 + M(16), "TERM-MC-1", it, run, "OP-CNC-11", tags=("S08",))
    t2 = T("2026-09-10 15:55")
    cmm(g, it, t2, tags=("S08",), label="cmm/%s/after-rework" % it, extra={"trigger": "after_rework"})
    g.ev(t2 + M(3), "TERM-MC", "item.presented", item=it,
         payload=OrderedDict([("item_id", it), ("presentation_no", 2), ("presented_to", ["QC"]), ("presented_by", "MST-01"),
                              ("gate", "ZT-2")]), tags=("S08",), label="present/%s/ZT-2#2" % it, persona="MST-01")
    z2 = t2 + M(5)
    gate(g, z2, "QC-01", "ZT-2", item=it, basis=["kt2/" + it, "cmm/%s/after-rework" % it], presentation=2, tags=("S08",),
         label="ZT-2#2/" + it)
    return z2


def bg_dent(g, it, t_recv):
    """«Вмятина» при приёме сварочным цехом (источник — перемещение): ОТК отклонил — след тары на плёнке."""
    g.step(t_recv + M(20), "QC-01", "QC", "decision.signal_rejected",
           {"item_id": it, "signal_basis_labels": ["mv/%s/MC-WC.received" % it]},
           {"reason": "след ремня тары на защитной плёнке; под плёнкой металл без вмятины — линейка и щуп 0,05 мм",
            "source": "movement",
            "hold_released": "containment.hold_released (Е-116): удержание по сигналу снято этим решением → в MES «можно выдавать»"},
           tags=("S08",), catalog="E-91", label="S08/bg/reject-dent")


def bg_scratch(g, it, t_recv):
    """«Царапина» при приёме сборочным цехом: ОТК отклонил — след СОЖ, под УФ светится [ПП]."""
    g.step(t_recv + M(25), "QC-02", "QC", "decision.signal_rejected",
           {"item_id": it, "signal_basis_labels": ["mv/%s/WC-AC.received" % it]},
           {"reason": "под УФ светится — след СОЖ, не царапина; после протирки поверхность чистая [ПП]", "source": "movement",
            "hold_released": "containment.hold_released (Е-116): удержание по сигналу снято этим решением → в MES «можно выдавать»"},
           tags=("S08",), catalog="E-91", label="S08/bg/reject-scratch")


# ----------------------------------------------------------------------------- «истина» одного мира
def build(world):
    """world: 'main' — главная история; 's13' — самостоятельный прогон S13 (только 12 фланцев)."""
    if world == "main":
        g = Gen("main-story", 20260921)
        ITEMS = None
    else:
        g = Gen("S13", 20260925)
        ITEMS = set(S13_ITEMS)

    def inc(item):
        return ITEMS is None or item in ITEMS

    MAIN = world == "main"
    tool = [60]

    # ---------------- справочные импорты и фоновый заказ ЗП-0911 (прошлая неделя)
    imports(g, T("2026-09-07 09:00"), tags=("S01",) if MAIN else ())
    order(g, T("2026-09-07 10:00"), "ORD-0911", "ord-5c2b-0911", "1c-in-000301", 24, "2026-09-25", "F-201", "F-224")
    bg_items = ["F-%03d" % n for n in range(201, 225)]
    bg_inc = [x for x in bg_items if inc(x)]
    t = T("2026-09-08 08:00")
    for lot in ("LOT-ZF-111", "LOT-R-108", "LOT-C-211", "LOT-SL-391", "LOT-FS-481", "LOT-W-87", "LOT-V-591"):
        lot_received(g, t, lot)
        t = t + S(30)
    tt = iqc(g, T("2026-09-08 09:00"), "LOT-ZF-111", bg_inc, True, True, T("2026-09-08 09:00"))
    tt = iqc(g, tt + M(5), "LOT-R-108", [ring_of(x)[0] for x in bg_inc], True, True, tt)
    tt = iqc(g, tt + M(5), "LOT-C-211", [cover_of(x)[0] for x in bg_inc], False, True, tt)
    tt = iqc(g, tt + M(5), "LOT-SL-391", None, False, False, tt)
    tt = iqc(g, tt + M(5), "LOT-FS-481", None, False, False, tt)
    tt = iqc(g, tt + M(5), "LOT-W-87", None, False, False, tt)
    tt = iqc(g, tt + M(5), "LOT-V-591", None, False, False, tt)   # клапаны: заводские номера, выборка камерой [П]
    bplan = machining_plan(bg_items, T("2026-09-09 08:00"))
    breg = reg_times(bplan, bg_items, None)
    for it in bg_items:
        if inc(it):
            register(g, breg[it], it, "ORD-0911", "LOT-ZF-111")
            kw = BG_CHIP_KT2 if (MAIN and it == BG_CHIP) else None
            machining(g, it, bplan[it][0], 50, tool_state=tool, kt2_kwargs=kw)
    bcmm = cmm_queue([(it, bplan[it][1]) for it in bg_items])
    b_in_wc = {}
    for it in bg_items:
        s0, s1 = bcmm[it]
        if not inc(it):
            continue
        if MAIN and it == BG_GEOM:
            z = bg_geometry(g, it, s1)
        else:
            cmm(g, it, s1)
            z = zt_time(s1 + M(5))
            if MAIN and it == BG_CHIP:
                z = bg_chip(g, it, z)
            else:
                gate(g, z, "QC-01", "ZT-2", item=it, basis=["kt2/" + it, "cmm/" + it], label="ZT-2/" + it)
        r = transfer_run(z)
        if MAIN and it == BG_DENT:
            move(g, r, it, "MC", "WC", "MST-01", "MST-02", label="mv/%s/MC-WC" % it, receipt_check="damage_found",
                 damage_details=[OrderedDict([("defect_type", "dent"), ("zone", "FACE-OUTER"), ("size_estimate_mm", 3),
                                              ("text", "похоже на вмятину на наружной кромке")])], recv_tags=("S08",))
            bg_dent(g, it, r + M(10))
        else:
            move(g, r, it, "MC", "WC", "MST-01", "MST-02", label="mv/%s/MC-WC" % it)
        b_in_wc[it] = r + M(10)
    # фоновые сварки F-201…F-220 (Пн 14 – Вт 15.09), сварщики прошлой недели: С-03 — смена А, С-02 — смена Б [П]
    bg_welds = []
    for k, it in enumerate(["F-%03d" % n for n in range(211, 216)]):
        bg_welds.append((it, "WS-1" if k % 2 == 0 else "WS-2", "WLD-03", T("2026-09-14 08:30") + M(50 * k)))
    for k, it in enumerate(["F-%03d" % n for n in range(201, 206)]):
        bg_welds.append((it, "WS-2" if k % 2 == 0 else "WS-1", "WLD-02", T("2026-09-14 17:00") + M(50 * k)))
    for k, it in enumerate(["F-%03d" % n for n in range(216, 221)]):
        bg_welds.append((it, "WS-2" if k % 2 == 0 else "WS-1", "WLD-03", T("2026-09-15 08:30") + M(50 * k)))
    for k, it in enumerate(["F-%03d" % n for n in range(206, 211)]):
        bg_welds.append((it, "WS-1" if k % 2 == 0 else "WS-2", "WLD-02", T("2026-09-15 17:00") + M(50 * k)))
    if MAIN:
        issue(g, T("2026-09-14 07:55"), "LOT-W-87", 1, "ST-WC-P1", label="issue/LOT-W-87")
        for dd in ("2026-09-14", "2026-09-15"):
            rings = [ring_of(it)[0] for (it, s, w, st) in bg_welds if day(st).isoformat() == dd]
            # до скана кольца (WJ) перед подготовкой кромок первой сварки дня (08:00)
            issue(g, T(dd + " 07:50"), "LOT-R-108", len(rings), "ST-WC-P1", components=rings,
                  label="issue/rings/" + dd)
    bg_weld_end = {}
    for (it, src, welder, st) in bg_welds:
        if not inc(it):
            continue
        assert b_in_wc[it] <= st - M(30), (it, b_in_wc[it], st)
        ring, rlot = ring_of(it)
        w = dict(item=it, src=src, welder=welder, start=st, end=st + M(40), arc=(st + M(2), st + M(37)),
                 ring=ring, ring_lot=rlot, wire="LOT-W-87", prep=st - M(30), run="SV-%s-1" % num(it), profile=None)
        weld(g, w)
        bg_weld_end[it] = w["end"]
    # фон: РК, ЗТ-3, передача в СИЦ, сборка, испытание, выпуск — последовательный планировщик смены А
    if MAIN:
        nd_free = T("2026-09-14 13:00")
        rk_no = defaultdict(int)
        bg_ready_ac = {}
        for (it, src, welder, st) in sorted(bg_welds, key=lambda x: x[3]):
            t = max(bg_weld_end[it] + M(60), nd_free)
            t = zt_time(t) if shift_of(t) != "A" else t
            if t >= at(day(t), "12:30") and t < at(day(t), "13:00"):
                t = at(day(t), "13:00")
            if t + M(20) > at(day(t), "16:20"):
                t = at(next_workday(day(t)), "08:10")
            rk_no[day(t)] += 1
            no = "RK-%s-%02d" % (day(t).strftime("%m%d"), rk_no[day(t)])
            xray(g, t + M(20), it, no)
            nd_free = t + M(20)
            z = zt_time(t + M(30))
            gate(g, z, "QC-01", "ZT-3", item=it, basis=["kt3/SV-%s-1/CAM-WS-1" % num(it), "xr/%s/%s" % (it, no)],
                 label="ZT-3/" + it)
            r = transfer_run(z, fri_morning_only=False)
            if it == BG_SCRATCH:
                move(g, r, it, "WC", "AC", "MST-02", "MST-03", label="mv/%s/WC-AC" % it, receipt_check="damage_found",
                     damage_details=[OrderedDict([("defect_type", "scratch"), ("zone", "FACE-OUTER"), ("length_mm", 12),
                                                  ("text", "похоже на царапину на наружной поверхности")])], recv_tags=("S08",))
                bg_scratch(g, it, r + M(10))
            else:
                move(g, r, it, "WC", "AC", "MST-02", "MST-03", label="mv/%s/WC-AC" % it)
            bg_ready_ac[it] = r + M(10)
        asm_free = T("2026-09-15 08:00")
        lt_free = T("2026-09-15 08:00")
        fin_free = T("2026-09-15 08:00")
        for it in sorted(bg_ready_ac, key=lambda x: bg_ready_ac[x]):
            t = max(bg_ready_ac[it], asm_free)
            while True:
                d = day(t)
                if d in WEEKEND or t + M(70) > at(d, "16:25"):
                    t = at(next_workday(d), "08:00")
                    continue
                if t < at(d, "08:00"):
                    t = at(d, "08:00")
                if t < at(d, "13:00") and t + M(70) > at(d, "12:30"):
                    t = at(d, "13:00")
                    continue
                break
            cov, clot = cover_of(it)
            kit_issue(g, t - M(5), it, cov, "LOT-SL-391", "LOT-FS-481", clot)
            assembly(g, it, t, cov, clot, "LOT-SL-391", "LOT-FS-481")
            asm_free = t + M(70)   # волна 2: в сборку вошла установка оборудования (клапан), слот 70 мин
            t2 = max(t + M(70), lt_free)
            while True:
                d = day(t2)
                if d in WEEKEND or t2 + M(65) > at(d, "16:25"):
                    t2 = at(next_workday(d), "08:00")
                    continue
                if t2 < at(d, "08:00"):
                    t2 = at(d, "08:00")
                break
            leak_test(g, it, t2, dur=55)
            lt_free = t2 + M(60)
            t3 = max(t2 + M(70), fin_free)
            if t3 + M(40) > at(day(t3), "16:25"):
                t3 = at(next_workday(day(t3)), "08:00")
            final(g, it, t3, t3 + M(10), t3 + M(15), t3 + M(30))
            fin_free = t3 + M(20)

    # ---------------- главный заказ ЗП-0917
    order(g, T("2026-09-14 11:00"), "ORD-0917", "ord-5c2b-0917", "1c-in-000314", 40, "2026-10-09", "F-001", "F-040",
          tags=("S01", "I1") if MAIN else ())
    items_main = ["F-%03d" % n for n in range(1, 37)]
    t = T("2026-09-17 08:00")
    for lot in ("LOT-ZF-201", "LOT-R-116", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", "LOT-W-88", "LOT-V-601"):
        lot_received(g, t, lot, tags=("S01",) if MAIN else ())
        t = t + S(30)
    inc_main = [x for x in items_main if inc(x)]
    tags01 = ("S01",) if MAIN else ()
    tt = iqc(g, T("2026-09-17 09:00"), "LOT-ZF-201", inc_main, True, True, T("2026-09-17 09:00"), tags=tags01)
    rings116 = [ring_of(x)[0] for x in inc_main if ring_of(x)[1] == "LOT-R-116"]
    tt = iqc(g, tt + M(5), "LOT-R-116", rings116, True, True, tt, tags=tags01)
    tt = iqc(g, tt + M(5), "LOT-C-301", [cover_of(x)[0] for x in inc_main], False, True, tt, tags=tags01)
    tt = iqc(g, tt + M(5), "LOT-SL-401", None, False, False, tt, tags=tags01)
    tt = iqc(g, tt + M(5), "LOT-FS-501", None, False, False, tt, tags=tags01)
    tt = iqc(g, tt + M(5), "LOT-W-88", None, False, False, tt, tags=tags01)
    tt = iqc(g, tt + M(5), "LOT-V-601", None, False, False, tt, tags=tags01)   # клапаны КВД [П]
    # партия П-117 (S02): Вт 22.09 08:00; входной контроль 10:30–11:00
    s02 = ("S02",) if MAIN else ()
    if MAIN:
        lot_received(g, T("2026-09-22 08:00"), "LOT-R-117", tags=s02)
        iqc(g, T("2026-09-22 10:30"), "LOT-R-117", ["R-%03d" % n for n in range(101, 111)], True, True,
            T("2026-09-22 11:00"), tags=s02, cam_q=0.90)
    # мехобработка
    mplan = machining_plan(MAIN_MACH, T("2026-09-18 09:00"), pins_dur={"F-001": 62})
    mreg = reg_times(mplan, MAIN_MACH, dt.date(2026, 9, 18))
    for it in MAIN_MACH:
        s0, s1 = mplan[it]
        if not inc(it):
            continue
        tg = ("S01",) if it == "F-001" else (("S03",) if it == "F-017" else ())
        tg = tg if MAIN else ()
        register(g, mreg[it], it, "ORD-0917", "LOT-ZF-201", tags=tg)
        fin = None
        if MAIN and it == "F-004":
            # S16: оператор нажал «закончил» по MO-004-1 с опозданием — уже после «начал» по MO-005-1 [П]
            fin = mplan["F-005"][0] + S(70)
        machining(g, it, s0, s1_min(s0, s1), tags=tg, kt2_at=T("2026-09-18 11:20") if it == "F-017" else None,
                  tool_state=tool, finish_at=fin)
    mcmm = cmm_queue([(it, mplan[it][1]) for it in MAIN_MACH], pins={"F-001": T("2026-09-18 10:15")})
    in_wc = {}
    for it in MAIN_MACH:
        s0, s1 = mcmm[it]
        if not inc(it):
            continue
        tg = (("S01",) if it == "F-001" else ()) if MAIN else ()
        cmm(g, it, s1, tags=tg)
        z = T("2026-09-18 11:00") if it == "F-001" else zt_time(s1 + M(5))
        gate(g, z, "QC-01", "ZT-2", item=it, tags=tg, basis=["kt2/" + it, "cmm/" + it], label="ZT-2/" + it)
        r = transfer_run(z)
        tgm = tg + (("S03",) if (it == "F-017" and MAIN) else ())
        move(g, r, it, "MC", "WC", "MST-01", "MST-02", tags=tgm, label="mv/%s/MC-WC" % it)
        in_wc[it] = r + M(10)
    if MAIN:
        s16_and_s15(g, mplan)
    for it in ("F-221", "F-222", "F-223", "F-224"):
        if it in b_in_wc:
            in_wc[it] = b_in_wc[it]
    # выдача проволоки и колец
    issue(g, T("2026-09-21 08:00"), "LOT-W-88", 1, "ST-WC-P1", tags=("S01", "S05") if MAIN else (), label="issue/LOT-W-88",
          note="проволока П-88 на участке с Пн 21.09 08:00")
    for dd in ("2026-09-21", "2026-09-22", "2026-09-23"):
        rows = [x for x in MAIN_WELDS if x[3].startswith(dd) or (dd == "2026-09-22" and x[3].startswith("2026-09-23 00"))]
        rows = [x for x in rows if not (dd == "2026-09-23" and x[3].startswith("2026-09-23 00"))]
        by_lot = defaultdict(list)
        for x in rows:
            if not inc(x[0]):
                continue
            rg, rl = ring_of(x[0])
            if rl == "LOT-R-117":
                continue
            by_lot[rl].append((x[0], rg))
        # V6 (схема v0.2): кольцо выдаётся под фланец, когда фланец принят сварочным цехом (не раньше приёмки
        # + 2 мин), и до подготовки кромок (скан WJ → W1): не позже prep − 5 мин; обычно — утренняя выдача 08:10.
        prep_of = {x[0]: (T(PREP[x[0]]) if x[0] in PREP else T(x[3]) - M(25)) for x in MAIN_WELDS}
        for rl, lst in sorted(by_lot.items()):
            by_t = defaultdict(list)
            for (i, r_) in lst:
                ti = max(in_wc[i] + M(2), min(T(dd + " 08:10"), prep_of[i] - M(5)))
                by_t[ti].append((i, r_))
            for ti, sub in sorted(by_t.items()):
                tg = ("S01",) if (MAIN and any(i == "F-001" for i, _ in sub)) else ()
                lb = "issue/rings/%s/%s" % (dd, rl)
                if ti != T(dd + " 08:10"):
                    lb += "@" + ti.astimezone(MSK).strftime("%H%M")
                issue(g, ti, rl, len(sub), "ST-WC-P1", items=[i for i, _ in sub],
                      components=[r_ for _, r_ in sub], tags=tg, label=lb)
    if MAIN:
        t117 = T("2026-09-22 14:15")
        assert all(in_wc[i] + M(2) <= t117 for i in ("F-019", "F-027", "F-029")), "П-117: кольца раньше приёмки фланцев"
        issue(g, t117, "LOT-R-117", 3, "ST-WC-P1", items=["F-019", "F-027", "F-029"],
              components=["R-101", "R-102", "R-103"], tags=s02, label="issue/rings/LOT-R-117")
    # сварки
    weld_by_item = {}
    for (it, src, welder, s, e) in MAIN_WELDS:
        if not inc(it):
            continue
        st, en = T(s), T(e)
        ring, rlot = ring_of(it)
        prep = T(PREP[it]) if it in PREP else st - M(25)
        assert prep >= in_wc[it], (it, prep, in_wc[it])
        arc = (st + M(2), en - M(3))
        w = dict(item=it, src=src, welder=welder, start=st, end=en, arc=arc, ring=ring, ring_lot=rlot,
                 wire=lots_of(it)["wire"], prep=prep, run="SV-%s-1" % num(it), profile=None)
        tg = ()
        if MAIN:
            if it == "F-001":
                tg = ("S01",)
            if it == "F-019":
                w["arc"] = (T("2026-09-22 16:40:20"), T("2026-09-22 17:13:50"))
                tg = ("S02", "S04", "S05")
            if it == "F-017":
                w["arc"] = (T("2026-09-23 10:41:00"), T("2026-09-23 10:59:00"))
                w["pauses"] = [(T("2026-09-23 10:50"), T("2026-09-23 10:52"))]
                w["arc_gaps"] = [(T("2026-09-23 10:50"), T("2026-09-23 10:52"))]
                w["seg_bounds"] = [("U1", "10:41:00", "10:45:00"), ("U2", "10:45:00", "10:50:00"),
                                   ("U3", "10:52:00", "10:54:00"), ("U4", "10:54:00", "10:55:15"),
                                   ("U5", "10:55:15", "10:56:30"), ("U6", "10:56:30", "10:57:45"),
                                   ("U7", "10:57:45", "10:58:30"), ("U8", "10:58:30", "10:59:00")]
                w["manual_override"] = OrderedDict([("what", "deviation_from_fixture_position"),
                                                    ("text", "деталь переставлена в приспособлении"),
                                                    ("near_zone", "U2")])
                w["pause_reason"] = "manual_repositioning"
                tg = ("S03", "S05")
            if it == "F-015":
                w["arc"] = (T("2026-09-22 10:17:00"), T("2026-09-22 10:57:00"))
                tg = ("S05", "S07")
            if it in ("F-021", "F-023", "F-025"):
                tg = ("S05",) + (("S07",) if True else ())
            if it in ("F-016",):
                tg = ("S07",)
        if it in DRIFT:
            w["profile"] = [(T(a), lo, hi) for (a, lo, hi) in DRIFT[it]]
        if MAIN and it != "F-001":
            tg = tuple(sorted(set(tg) | {"S05"})) if T(s) >= T("2026-09-21 14:55") else tg
        s12 = None
        cam_override = None
        if MAIN and it == "F-024":
            s12 = {"CAM-WS-1": dict(schema="1.1", extra={"comparison_to_before": "new_zone_created_by_operation",
                                                         "lens_temp_c": 41.5},
                                    tags=("S12",), note="S12 сообщение 4: версия 1.1 с новым необязательным полем lens_temp_c"),
                   "CAM-WS-2": dict(drop_item=True, tags=("S12",), noise="malformed:E_MISSING_FIELD",
                                    note="S12 сообщение 1: нет обязательного поля item_id (контекст поста — F-024)")}
        if MAIN and it == "F-026":
            s12 = {"CAM-WS-2": dict(result="defect_found", q=0.9, conf=0.62,
                                    defects=[OrderedDict([("defect_type", "spatter_x"), ("zone", "U3"),
                                                          ("severity", "minor"), ("confidence", 0.62),
                                                          ("description", "класс spatter_x отсутствует в классификаторе")])],
                                    tags=("S12",), noise="edge:unknown_enum_noncritical",
                                    note="S12 сообщение 5: неизвестный класс дефекта в некритичном поле")}
        if MAIN and it == "F-017":
            def cam_override(gg, t0):
                camera(gg, T("2026-09-23 11:05:00"), "CAM-WS-1", "F-017", "CP-WELD", "after_operation", SEGS,
                       result="defect_found", q=0.90, conf=0.86, tags=("S03", "S06", "S05"),
                       defects=[OrderedDict([("defect_type", "burn_through"), ("zone", "U2"), ("side", "face"),
                                             ("severity", "major"), ("size_estimate_mm", 3.0), ("measurable", False),
                                             ("confidence", 0.86)])],
                       label="kt3/SV-017-1/CAM-WS-1", extra={"comparison_to_before": "new_after"})
                camera(gg, T("2026-09-23 11:05:10"), "CAM-WS-2", "F-017", "CP-WELD", "after_operation", SEGS,
                       result="defect_found", q=0.88, conf=0.81, tags=("S03", "S06", "S05"),
                       defects=[OrderedDict([("defect_type", "burn_through"), ("zone", "U2"), ("side", "face"),
                                             ("severity", "major"), ("size_estimate_mm", 3.0), ("measurable", False),
                                             ("confidence", 0.81)])],
                       label="kt3/SV-017-1/CAM-WS-2", extra={"comparison_to_before": "new_after"})
        weld(g, w, tags=tg, s12=s12, cam_override=cam_override)
        weld_by_item[it] = w
    # явные окна питания ИС-2 (под счётчики S04/S06/S07); остальные — автоматически по сменам
    if MAIN:
        g.power_override |= {("WS-2", (dt.date(2026, 9, 22), "A")), ("WS-2", (dt.date(2026, 9, 22), "B")),
                             ("WS-2", (dt.date(2026, 9, 23), "A"))}
        g.power.append(("WS-2", T("2026-09-22 08:15"), "SOLVE_P2", "end_of_work", None))
        g.power.append(("WS-2", T("2026-09-23 08:25"), T("2026-09-23 11:12:30"), "process_hold", "hold/WS-2"))
        g.lost_windows.append(("WS-2", T("2026-09-22 16:40:00"), T("2026-09-22 17:15:00"), 35,
                               "переполнение буфера ШЛ-2: записи Вт 16:40–17:15 потеряны навсегда (S04)"))
        g.batches.append(dict(src="WS-2", frm=T("2026-09-22 09:50:00"), to=T("2026-09-23 11:12:30"),
                              deliver_start=T("2026-09-23 12:05:00"), dup_first=312, key_label="EV-WS2-0412",
                              key_deliver=T("2026-09-23 12:05:07"), expect_unique=928,
                              note="ШЛ-2 потерял связь Вт 09:50, дослал журнал Ср 12:05; первые 312 записей отправил дважды"))

    # ---------------- РК, ЗТ-3, передачи в СИЦ (главная неделя)
    RK_MON = [("F-001", "13:00"), ("F-002", "13:20"), ("F-003", "13:40"), ("F-004", "14:00"), ("F-005", "14:20"),
              ("F-006", "15:40")]
    ZT3_MON = {"F-001": "14:00", "F-002": "14:10", "F-003": "14:20", "F-004": "14:30", "F-005": "14:40", "F-006": "16:05"}
    RK_TUE = [("F-007", "08:30"), ("F-008", "08:50"), ("F-009", "09:10"), ("F-010", "09:30"), ("F-011", "09:50"),
              ("F-221", "10:10"), ("F-013", "10:30"), ("F-012", "10:50"), ("F-018", "11:10"), ("F-014", "11:30"),
              ("F-222", "11:50"), ("F-016", "13:10"), ("F-015", "13:30")]
    zt3_at = {}
    for k, (it, hm) in enumerate(RK_MON):
        if not inc(it):
            continue
        tg = ("S01",) if (MAIN and it == "F-001") else ()
        xray(g, at(dt.date(2026, 9, 21), hm), it, "RK-0921-%02d" % (k + 1), tags=tg)
        z = at(dt.date(2026, 9, 21), ZT3_MON[it])
        gate(g, z, "QC-01", "ZT-3", item=it, tags=tg, basis=["kt3/SV-%s-1/CAM-WS-1" % num(it), "xr/%s/RK-0921-%02d" % (it, k + 1)],
             label="ZT-3/" + it)
        zt3_at[it] = z
    for k, (it, hm) in enumerate(RK_TUE):
        if not inc(it):
            continue
        tg = ("S05",) + (("S07",) if it in ("F-015", "F-016") else ())
        tg = tg if MAIN else ()
        no = "RK-0922-%02d" % (k + 1)
        xray(g, at(dt.date(2026, 9, 22), hm), it, no, tags=tg)
        if it == "F-015":
            z = T("2026-09-22 15:30")
            # схема v0.2: W5 → WG «годно при неполных данных» → WZ2 [П]
            gate(g, z, "QC-01", "ZT-3", item=it, tags=("S07", "S05"), basis=["kt3/SV-015-1/CAM-WS-1", "xr/F-015/" + no],
                 decision="accept_incomplete_data",
                 note="итог «годно при неполных данных» [П]: журнал режима недоступен (сбой связи), принято по камере и рентгену",
                 extra={"missing": ["журнал режима WS-2 за сварку SV-015-1 (нет связи со Вт 09:50)"],
                        "remark": "журнал режима недоступен (сбой связи), принято по камере и рентгену",
                        "bpmn_node": "WZ2"},
                 label="ZT-3/F-015")
        else:
            z = at(dt.date(2026, 9, 22), hm) + M(10)
            gate(g, z, "QC-01", "ZT-3", item=it, tags=tg, basis=["kt3/SV-%s-1/CAM-WS-1" % num(it), "xr/%s/%s" % (it, no)],
                 label="ZT-3/" + it)
        zt3_at[it] = z
    # передачи СЦ → СИЦ
    STORAGE8 = ["F-011", "F-221", "F-013", "F-012", "F-018", "F-014", "F-222", "F-016"]
    moves_wc_ac = [("F-001", "2026-09-21 16:00", True), ("F-002", "2026-09-21 16:00", True),
                   ("F-003", "2026-09-21 16:00", True), ("F-004", "2026-09-21 16:00", True),
                   ("F-005", "2026-09-21 16:00", True), ("F-006", "2026-09-22 07:50", True),
                   ("F-007", "2026-09-22 10:00", True), ("F-008", "2026-09-22 10:00", True),
                   ("F-009", "2026-09-22 10:00", True), ("F-010", "2026-09-22 10:00", True),
                   ("F-015", "2026-09-22 16:00", True)] + [(x, "2026-09-22 14:00", False) for x in STORAGE8]
    for it, ts, rec in moves_wc_ac:
        if not inc(it):
            continue
        assert T(ts) >= zt3_at[it], it
        tg = ("S01",) if it == "F-001" else (("S05",) if it not in ("F-002", "F-003", "F-004", "F-005", "F-006") else ())
        move(g, T(ts), it, "WC", "AC", "MST-02", "MST-03", tags=tg if MAIN else (), received=rec,
             label="mv/%s/WC-AC" % it)

    # ---------------- сборка, испытания, выпуск (главная история)
    if MAIN:
        a = dict(seal=T("2026-09-22 09:20"), uv=T("2026-09-22 09:25"), cover=T("2026-09-22 09:45"),
                 fasteners=T("2026-09-22 09:55"), torque=T("2026-09-22 10:10"), finish=T("2026-09-22 10:25"))
        kit_issue(g, T("2026-09-22 09:00"), "F-001", "C-001", "LOT-SL-401", "LOT-FS-501", "LOT-C-301", tags=("S01",))
        assembly(g, "F-001", T("2026-09-22 09:10"), "C-001", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", tags=("S01",),
                 zt41=T("2026-09-22 09:35"), zt4=T("2026-09-22 10:30"), kt4=T("2026-09-22 10:20"), plan=a)
        leak_test(g, "F-001", T("2026-09-22 13:00"), tags=("S01",), zt5=T("2026-09-22 14:30"))
        final(g, "F-001", T("2026-09-23 09:30"), T("2026-09-23 10:10"), T("2026-09-23 10:15"), T("2026-09-23 10:30"),
              tags=("S01",), q=0.92)
        g.stand_faults.append(OrderedDict([
            ("id", "FAULT-1C-503-F001"), ("system", "erp_1c"), ("scenarios", ["S01", "I1"]),
            ("match", {"posting_type": "release_good", "item_id": "F-001", "attempt": 1}),
            ("response", {"http_status": 503, "text": "1С недоступна"}),
            ("expected_behavior", "временная ошибка → повтор с тем же номером сообщения; подтверждение в Ср 23.09 10:31"),
            ("retry_ack_at", iso(T("2026-09-23 10:31")))]))
        # S08: Ф-002, ложная «нет метки затяжки»
        kit_issue(g, T("2026-09-22 10:30"), "F-002", "C-002", "LOT-SL-401", "LOT-FS-501", "LOT-C-301", tags=("S08",))
        assembly(g, "F-002", T("2026-09-22 10:35"), "C-002", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", tags=("S08",),
                 zt41=T("2026-09-22 10:55"), zt4=False, kt4=T("2026-09-22 11:20"),
                 plan=dict(seal=T("2026-09-22 10:45"), uv=T("2026-09-22 10:48"), cover=T("2026-09-22 11:00"),
                           fasteners=T("2026-09-22 11:05"), torque=T("2026-09-22 11:10"), finish=T("2026-09-22 11:22"),
                           kt4_kwargs=dict(result="defect_found", q=0.82, conf=0.55,
                                           defects=[OrderedDict([("defect_type", "missing_torque_mark"),
                                                                 ("zone", "BOLT-7"), ("severity", "minor"),
                                                                 ("confidence", 0.55)])])))
        g.step(T("2026-09-22 11:35"), "QC-02", "QC", "decision.signal_rejected",
               {"item_id": "F-002", "signal_basis_labels": ["kt4/F-002"], "checkpoint_id": "CP-ASM", "zone": "BOLT-7"},
               {"reason": "метка есть, закрыта тенью от шайбы; момент по ключу в норме",
                "evidence_labels": ["torque/F-002"], "model_tuning_candidate": True,
                "hold_released": "containment.hold_released (Е-116): удержание Е-84 по правилу R-16 [П] (уровень «доп. проверка») "
                                 "снято этим же решением ОТК-2"},
               tags=("S08",), catalog="E-91", label="S08/reject")
        gate(g, T("2026-09-22 11:40"), "QC-02", "ZT-4", item="F-002", tags=("S08",),
             basis=["kt4/F-002", "torque/F-002", "torque/F-002/J-2"], label="ZT-4/F-002")
        g.step(T("2026-09-22 11:30"), "QC-02", "QC", "decision.signal_rejected",
               {"item_id": "F-002", "signal_basis_labels": ["kt4/F-002"]},
               {"reason": ""}, tags=("S08",), catalog="E-91", expect="refused", mode="auto", optional=True,
               note="проверка S08-N2: отклонение без причины не принимается (ожидается отказ)", label="S08/reject-no-reason")
        leak_test(g, "F-002", T("2026-09-22 14:35"), zt5=T("2026-09-22 15:40"))
        # S11: Ф-003, «забыли прокладку»
        kit_issue(g, T("2026-09-22 11:40"), "F-003", "C-003", "LOT-SL-401", "LOT-FS-501", "LOT-C-301", tags=("S11",))
        g.ev(T("2026-09-22 11:50:05"), "OV-ASM", "operator_action.detected", item="F-003",
             payload=OrderedDict([("station_id", "ST-AC-ASM"), ("tp_step", "cover_install"),
                                  ("observed_action", "крышка поднесена к фланцу; уплотнения в канавке не видно"),
                                  ("deviation", "step_skipped"), ("expected_step", "seal_install"),
                                  ("confidence", 0.78), ("observation_quality", 0.85),
                                  ("operator_id", None), ("operator_source", "terminal_login"),
                                  ("evidence_refs", [{"kind": "pose_skeleton", "available_centrally": False}]),
                                  ("video_transferred", False)]),
             tags=("S11",), label="S11/ov-detected")
        assembly(g, "F-003", T("2026-09-22 11:44"), "C-003", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", tags=("S11",),
                 zt41=T("2026-09-22 12:05"), zt4=T("2026-09-22 12:40"), kt4=T("2026-09-22 12:30"),
                 plan=dict(seal=T("2026-09-22 11:52"), uv=T("2026-09-22 11:55"), cover=T("2026-09-22 12:10"),
                           fasteners=T("2026-09-22 12:15"), torque=T("2026-09-22 12:20"), finish=T("2026-09-22 12:35")))
        leak_test(g, "F-003", T("2026-09-23 08:00"), zt5=T("2026-09-23 09:05"))
        # F-004 (S12: сообщение 3 «maybe» на КТ-4, затем исправная повторная отправка), F-005
        kit_issue(g, T("2026-09-22 12:55"), "F-004", "C-004", "LOT-SL-401", "LOT-FS-501", "LOT-C-301")
        assembly(g, "F-004", T("2026-09-22 13:00"), "C-004", "LOT-C-301", "LOT-SL-401", "LOT-FS-501",
                 zt41=T("2026-09-22 13:30"), zt4=T("2026-09-22 14:10"), kt4=T("2026-09-22 14:05"),
                 plan=dict(seal=T("2026-09-22 13:12"), uv=T("2026-09-22 13:18"), cover=T("2026-09-22 13:35"),
                           fasteners=T("2026-09-22 13:42"), torque=T("2026-09-22 13:55"),
                           finish=T("2026-09-22 14:07"), kt4_resend=T("2026-09-22 14:06"),
                           kt4_kwargs=dict(result="maybe", tags=("S12",), noise="malformed:E_UNKNOWN_ENUM",
                                           note="S12 сообщение 3: исход «maybe» — неизвестное значение критичного перечисления")))
        leak_test(g, "F-004", T("2026-09-23 09:05"), zt5=T("2026-09-23 10:10"))
        kit_issue(g, T("2026-09-22 14:12"), "F-005", "C-005", "LOT-SL-401", "LOT-FS-501", "LOT-C-301")
        assembly(g, "F-005", T("2026-09-22 14:15"), "C-005", "LOT-C-301", "LOT-SL-401", "LOT-FS-501",
                 zt41=T("2026-09-22 14:45"), zt4=T("2026-09-22 15:30"))
        # Ср: F-006, F-010 (на стенде в 11:10), F-015 (сборка начата до инцидента)
        kit_issue(g, T("2026-09-23 07:55"), "F-006", "C-006", "LOT-SL-401", "LOT-FS-501", "LOT-C-301")
        assembly(g, "F-006", T("2026-09-23 08:00"), "C-006", "LOT-C-301", "LOT-SL-401", "LOT-FS-501",
                 zt41=T("2026-09-23 08:30"), zt4=T("2026-09-23 09:12"),
                 plan=dict(skip_check=dict(skip_at=T("2026-09-23 08:11"), done_at=T("2026-09-23 08:22"),
                                           reason="щуп 0,05 мм не найден на рабочем месте")))
        for e in g.events:
            if e["label"] in ("AS-006-1.seal", "ktsr/F-006"):
                e["tags"].add("S17")
        for s in g.steps:
            if s["label"] == "ZT-SR/F-006":
                s["tags"].add("S17")
        kit_issue(g, T("2026-09-23 09:12"), "F-010", "C-010", "LOT-SL-401", "LOT-FS-501", "LOT-C-301", tags=("S05",))
        assembly(g, "F-010", T("2026-09-23 09:15"), "C-010", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", tags=("S05",),
                 zt41=T("2026-09-23 09:45"), zt4=T("2026-09-23 10:30"),
                 kt4=T("2026-09-23 10:22"), plan=dict(finish=T("2026-09-23 10:25")))
        leak_test(g, "F-010", T("2026-09-23 10:45"), tags=("S05",), zt5=T("2026-09-23 12:30"))
        leak_test(g, "F-005", T("2026-09-23 11:50"), zt5=T("2026-09-23 12:55"))
        leak_test(g, "F-006", T("2026-09-23 12:55"), zt5=T("2026-09-23 14:00"))
        kit_issue(g, T("2026-09-23 10:30"), "F-015", "C-015", "LOT-SL-401", "LOT-FS-501", "LOT-C-301", tags=("S05",))
        assembly(g, "F-015", T("2026-09-23 10:35"), "C-015", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", tags=("S05",),
                 plan=dict(seal=T("2026-09-23 10:50"), uv=T("2026-09-23 10:55")), stop_after_uv=True)
        g.ev(T("2026-09-23 12:25"), "TERM-ASM", "operation.paused", item="F-015",
             payload=OrderedDict([("operation_run_id", "AS-015-1"), ("reason", "quality_block"),
                                  ("operator_id", "ASM-01"), ("details", "крышка ещё не установлена")]),
             tags=("S05",), corr="AS-015-1", label="AS-015-1.pause", persona="ASM-01", after="S05/v3")
        # схема v0.2, BB → B1: уплотнение уже стояло (10:50) — вскрытие (Е-120); закроется на ЗТ-4 после новой сборки
        g.step(T("2026-09-23 15:50"), "MST-03", "MASTER", "intervention.opened", {"item_id": "F-015"},
               {"removed": ["seal"], "not_installed": ["cover", "fasteners", "valve"], "zone": "SEAL-GROOVE",
                "reason": "возврат в сварочный цех на переделку шва по решению комиссии (узел B1)",
                "consent_by": "QC-02", "closes_at": "ZT-4 после повторной сборки (вскрытие закрывают на ЗТ-4)"},
               tags=("S05", "S10A"), catalog="E-120", signers=["MST-03", "QC-02"], label="S05/F-015/intervention")
        op_finish(g, T("2026-09-23 15:55"), "TERM-ASM", "F-015", "AS-015-1", "ASM-01", result="aborted", tags=("S05", "S10A"),
                  extra={"reason": "возврат в сварочный цех на переделку по решению комиссии"})
        move(g, T("2026-09-23 16:00"), "F-015", "AC", "WC", "MST-03", "MST-02", tags=("S05", "S10A"),
             label="mv/F-015/AC-WC", after_recv="S05/disposition-rework")

        # ---------------- сценарные решения и события Ср 23.09
        main_story_scenarios(g, weld_by_item)
        # v0.3.1: удержания движка (процесс «Удержание», узлы I2 и R3) — для ожидаемых сообщений MES
        rs01 = ["F-%03d" % n for n in range(7, 37)] + ["F-221", "F-222", "F-223", "F-224"]
        g.engine_holds += [
            dict(at=T("2026-09-23 11:10"), targets=rs01, basis="risk_scope:RS-01", node="I2"),
            dict(at=T("2026-09-23 13:55"), targets=["F-019", "F-027", "F-029", "LOT-R-117"], basis="risk_scope:RS-02", node="I2",
                 note="LOT-R-117 — остаток партии на складе: кольца R-104…R-110"),
            dict(at=T("2026-09-23 14:05"), targets=["F-015"], basis="review:ZT-3/F-015", node="R3")]
    else:
        s13_world(g, weld_by_item)
    return g


def s1_min(s0, s1):
    return int((s1 - s0).total_seconds() // 60)


# S15 [П]: часы КИМ-1 спешат на 7 мин с утра Пн 21.09 до синхронизации администратором в 10:15.
# Порядок записей источника (source_seq) не нарушен: «соврали часы» ≠ «нарушили порядок».
CLOCK_SKEW = dict(src="CMM-1", frm="2026-09-21 08:00", to="2026-09-21 10:15", offset_min=7)
S15_ITEMS = ["F-007", "F-008", "F-009", "F-010", "F-011"]
S16_ITEMS = ["F-004", "F-005"]


def s16_and_s15(g, mplan):
    # ---- S16: запись журнала ЧПУ без номера выполнения на стыке двух выполнений
    s5 = mplan["F-005"][0]
    g.ev(s5 + S(50), "CNC-1", "operator.action",
         payload=OrderedDict([("action_type", "mode_change"), ("operator_id", None), ("equipment_id", "CNC-1"),
                              ("details", OrderedDict([("parameter", "feed_override_pct"), ("from", 100), ("to", 120),
                                                       ("duration_s", 80), ("via", "machine_panel"),
                                                       ("program_id", "UP-FL-100-01")]))]),
         tags=("S16",), hints=S16_ITEMS, label="S16/cnc-feed-override",
         note="S16: журнал пульта ЧПУ-1 — без номера выполнения и без исполнителя; по отметкам терминала MO-004-1 закрыт "
              "в 13:51:10, MO-005-1 открыт в 13:50:00 — выполнения перекрываются, привязка неоднозначна")
    s16_labels = {"MO-004-1.finish", "MO-005-1.start", "cnc/F-004/idle", "cnc/F-005/running", "cnc/F-005/summary",
                  "kt2/F-004", "kt2/F-005", "cmm/F-004", "cmm/F-005", "remark/F-004"}
    for e in g.events:
        if e["label"] in s16_labels:
            e["tags"].add("S16")
    for s in g.steps:
        if s["label"] in ("ZT-2/F-004", "ZT-2/F-005"):
            s["tags"].add("S16")
    # ---- S15: сдвиг часов источника
    a, b = T(CLOCK_SKEW["frm"]), T(CLOCK_SKEW["to"])
    n = 0
    for e in g.events:
        if e["src"] == CLOCK_SKEW["src"] and a <= e["occ"] < b:
            e["skew"] = M(CLOCK_SKEW["offset_min"])
            e["noise"] = "clock_skew:+7m"
            e["tags"].add("S15")
            e["note"] = ("S15: часы КИМ-1 спешат на 7 мин (сбой синхронизации после выходных); в occurred_at — время "
                         "по часам источника, как прислано; порядок по source_seq не нарушен")
            n += 1
    assert n == len(S15_ITEMS), n
    for s in g.steps:
        if s["label"] in ["ZT-2/" + x for x in S15_ITEMS]:
            s["tags"].add("S15")
    g.step(b, "ADM-01", "DATA_ADMIN", "source.clock_synced", {"source_id": "CMM-1", "equipment_id": "CMM-1"},
           {"offset_before_s": 420, "detected_at": iso(T("2026-09-21 08:25:02")),
            "method": "часы КИМ-1 синхронизированы с сервером времени цеха",
            "records_corrected_by_rule": ["cmm/" + x for x in S15_ITEMS],
            "note": "исходные occurred_at не меняются; исправленное время — производная запись по правилу"},
           tags=("S15",), catalog="Е-88 [П]", label="S15/clock-synced")


# ----------------------------------------------------------------------------- сценарии главной истории
def main_story_scenarios(g, W):
    RS01_34 = ["F-%03d" % n for n in range(7, 37)] + ["F-221", "F-222", "F-223", "F-224"]
    WS2_13 = ["F-008", "F-010", "F-221", "F-012", "F-014", "F-222", "F-016", "F-015", "F-019", "F-021", "F-023",
              "F-025", "F-017"]
    SIX = ["F-015", "F-017", "F-019", "F-021", "F-023", "F-025"]
    # S03: НС-01 (СТОП), изоляция
    g.step(T("2026-09-23 11:08"), "QC-01", "QC", "decision.signal_confirmed",
           {"item_id": "F-017", "signal_basis_labels": ["kt3/SV-017-1/CAM-WS-1", "kt3/SV-017-1/CAM-WS-2"],
            "nc_ref": "NC-01"},
           {"defect_type": "burn_through", "zone": "U2", "severity": "major",
            "reason": "прожог на участке У2 подтверждён по двум ракурсам КТ-3; деталь не соответствует КД"},
           mode="stop", tags=("S03",), catalog="E-90", label="S03/confirm-NC-01")
    g.step(T("2026-09-23 11:20"), "QC-01", "QC", "decision.item_isolated", {"item_id": "F-017", "nc_ref": "NC-01"},
           {"location": "ST-ISO-WC", "reason": "подтверждённое несоответствие НС-01, ждёт решения"},
           tags=("S03",), catalog="E-93", label="S03/isolate")
    g.ev(T("2026-09-23 11:20:30"), "TERM-WC", "movement.isolator_confirmed", item="F-017",
         payload=OrderedDict([("item_id", "F-017"), ("to_station", "ST-ISO-WC"), ("physically_moved", True),
                              ("confirmed_by", "MST-02")]), tags=("S03",), label="S03/isolator-confirmed",
         persona="MST-02", after="S03/isolate")
    # S05: остановка ИС-2, круги 34 → 13 → 6
    g.step(T("2026-09-23 11:12"), "MST-02", "MASTER", "process_hold.set",
           {"equipment_id": "WS-2", "hold_scope": "equipment"},
           {"reason": "НС-01 и нет журнала ИС-2 со Вт 09:50", "release_condition": "ремонт и проверка источника",
            "proposed_by": "system"}, tags=("S05",), catalog="E-114", label="hold/WS-2")
    g.step(T("2026-09-23 11:45"), "TECH-01", "TECHNOLOGIST", "risk_scope.narrowed",
           {"risk_scope_ref": "RS-01", "to_version": 2},
           {"size_after": 13, "excluded_items": [x for x in RS01_34 if x not in WS2_13],
            "basis": ["журнал WS-1 непрерывный, в уставке", "CP-WELD no_defect_found при качестве ≥ 0,6",
                      "WLD-02 и LOT-W-88 на WS-1 без отклонений"],
            "holds_released": "у исключённых — containment.hold_released (Е-116 [П], узел I5): блок снимает человек этим шагом"},
           mode="stop", tags=("S05",), catalog="E-112", label="S05/v2")
    g.step(T("2026-09-23 11:47"), "QC-01", "QC", "risk_scope.narrowed", {"risk_scope_ref": "RS-01", "to_version": 3},
           {"size_after": 12, "excluded_items": ["F-010"], "basis": ["попытка контролёра"]},
           tags=("S05",), catalog="E-112", expect="refused", optional=True,
           note="проверка S05-N3: сужать круг может только технолог — ожидается отказ и запись в журнал критических действий",
           label="S05/narrow-by-qc-refused")
    g.step(T("2026-09-23 12:15"), "TECH-01", "TECHNOLOGIST", "risk_scope.narrowed",
           {"risk_scope_ref": "RS-01", "to_version": 3},
           {"size_after": 6, "excluded_items": ["F-008", "F-010", "F-012", "F-014", "F-016", "F-221", "F-222"],
            "unknown_items": ["F-019"],
            "basis": ["опоздавший журнал ИС-2: ток впервые вышел за уставку во Вт 10:20 (EV-WS2-0412)",
                      "семь сварок до этого — в уставке, записи есть", "у F-019 записей нет — «нет данных»"],
            "recalculated_due_to_label": "EV-WS2-0412",
            "holds_released": "у исключённых — containment.hold_released (Е-116 [П], узел I5); у F-019 блок остаётся"},
           mode="stop", tags=("S05", "S07", "S04"), catalog="E-112", label="S05/v3")
    g.step(T("2026-09-23 12:30"), "QC-01", "QC", "decision.recheck_requested",
           {"items": ["F-015", "F-019", "F-021", "F-023", "F-025"], "risk_scope_ref": "RS-01"},
           {"methods": ["camera", "xray"], "zones": SEGS}, tags=("S05", "S04"), catalog="E-92", label="S05/recheck5")
    # S05/S04 повторные кадры и рентген
    # S04 (правило «класс камере не виден»): Ф-021 — чистый кадр камеры, прожог в корне находит рентген (ниже, 12:50);
    # пересъёмку для корня камера не предлагает — следующее наблюдение для корня и внутренних пор только рентген
    camera(g, T("2026-09-23 12:35"), "CAM-WS-1", "F-021", "CP-WELD", "after_operation", SEGS, tags=("S05", "S04"),
           label="kt3/F-021/recheck", extra={"trigger": "recheck"})
    camera(g, T("2026-09-23 12:37"), "CAM-WS-1", "F-023", "CP-WELD", "after_operation", SEGS, tags=("S05",),
           label="kt3/F-023/recheck", extra={"trigger": "recheck"})
    camera(g, T("2026-09-23 12:39"), "CAM-WS-1", "F-019", "CP-WELD", "after_operation", SEGS, tags=("S05", "S02"),
           label="kt3/F-019/recheck", extra={"trigger": "recheck"})
    f025_first = camera(g, T("2026-09-23 12:40"), "CAM-WS-1", "F-025", "CP-WELD", "after_operation", SEGS, q=0.34, conf=0.91,
           tags=("S04", "S05"), reasons=["glare", "zone_occluded_by_clamp"],
           limitations=[OrderedDict([("zones", ["U6", "U7"]), ("text", "блик; зона частично закрыта прижимом")])],
           label="kt3/F-025/recheck-glare", extra={"trigger": "recheck", "capture_profile": "KT3-STD"})
    # S04 «следующее лучшее наблюдение» [П]: профиль — только из допущенных в RCP-CP-WELD@3 (inspection-recipes.yaml →
    # capture_profiles); предложение системы (Е-79, реакция — только в ожиданиях) ≠ решение; назначает ОТК-1,
    # потому что прижим снимает человек; лимит — 2 пересъёмки участка
    g.step(T("2026-09-23 12:45"), "QC-01", "QC", "decision.recheck_requested", {"item_id": "F-025"},
           OrderedDict([("method", "camera_reshoot"), ("zones", ["U6", "U7"]),
                        ("instructions", "снять прижим, сменить угол света"),
                        ("uncertainty_reasons", ["glare", "zone_occluded_by_clamp"]),
                        ("uncertainty_text", "блик на У6–У7; участок перекрыт прижимом"),
                        ("recipe_id", "RCP-CP-WELD@3"), ("capture_profile", "KT3-SIDE"),
                        ("profile_reason", "отделить блик от геометрии: зеркальный блик уходит при смене стороны света, рельеф шва остаётся"),
                        ("person_actions", ["снять прижим"]),
                        ("prior_observation_label", "kt3/F-025/recheck-glare"),
                        ("assigned_by", OrderedDict([("kind", "person"), ("persona", "QC-01"),
                                                     ("why_not_rule", "прижим снимает человек: по правилу край сам не переснимет")])),
                        ("reshoot_no", 1), ("reshoot_limit", 2),
                        ("proposal", OrderedDict([("proposal_ref", "PR-F025-01"), ("catalog", "E-79 [П]"),
                                                  ("type", "additional_check"), ("kind", "next_observation"),
                                                  ("voice", "инженерная справка"), ("proposed_at", "2026-09-23 12:41"),
                                                  ("status_after", "accepted")])),
                        ("suggested_by", "system")]), tags=("S04",), catalog="E-92", label="S04/reshoot")
    camera(g, T("2026-09-23 12:55"), "CAM-WS-1", "F-025", "CP-WELD", "after_operation", ["U6", "U7"], q=0.88, conf=0.93,
           tags=("S04", "S05"), label="kt3/F-025/reshoot",
           extra=OrderedDict([("trigger", "reshoot"), ("capture_profile", "KT3-SIDE"), ("reshoot_no", 1),
                              ("prior_observation_event_id", None)]))["prior_ref"] = f025_first
    xray(g, T("2026-09-23 12:50"), "F-021", "RK-0923-08", tags=("S05", "S04"),
         defects=[OrderedDict([("defect_type", "burn_through"), ("zone", "U5"), ("side", "root"), ("severity", "major")]),
                  OrderedDict([("defect_type", "porosity"), ("zone", "U6"), ("severity", "major")])])
    g.step(T("2026-09-23 13:05"), "QC-01", "QC", "decision.signal_confirmed",
           {"item_id": "F-021", "signal_basis_labels": ["xr/F-021/RK-0923-08"], "nc_ref": "NC-02"},
           {"defects": [{"defect_type": "burn_through", "zone": "U5", "side": "root"},
                        {"defect_type": "porosity", "zone": "U6"}], "severity": "major",
            "reason": "одно заключение РК, два дефекта"}, tags=("S05",), catalog="E-90", label="S05/confirm-NC-02")
    xray(g, T("2026-09-23 13:10"), "F-023", "RK-0923-10", tags=("S05",),
         defects=[OrderedDict([("defect_type", "burn_through"), ("zone", "U3"), ("side", "root"), ("severity", "major")])])
    g.step(T("2026-09-23 13:20"), "QC-01", "QC", "decision.signal_confirmed",
           {"item_id": "F-023", "signal_basis_labels": ["xr/F-023/RK-0923-10"], "nc_ref": "NC-03"},
           {"defects": [{"defect_type": "burn_through", "zone": "U3", "side": "root"}], "severity": "major",
            "reason": "прожог в корне шва по заключению РК"}, tags=("S05",), catalog="E-90", label="S05/confirm-NC-03")
    xray(g, T("2026-09-23 13:25"), "F-025", "RK-0923-09", tags=("S04", "S05"))
    # S02: пора в теле кольца Ф-019
    xray(g, T("2026-09-23 13:40"), "F-019", "RK-0923-07", tags=("S02", "S05"),
         defects=[OrderedDict([("defect_type", "base_metal_pore"), ("zone", "RING-BODY"), ("severity", "major"),
                               ("size_mm", 1.2), ("distance_from_weld_mm", 25), ("in_heat_affected_zone", False)])],
         extra={"weld_segments_result": "no_defect_found"})
    g.step(T("2026-09-23 13:50"), "QC-01", "QC", "decision.signal_confirmed",
           {"item_id": "F-019", "signal_basis_labels": ["xr/F-019/RK-0923-07"], "nc_ref": "NC-04"},
           {"defect_type": "base_metal_pore", "zone": "RING-BODY", "severity": "major",
            "reason": "пора 1,2 мм в теле кольца вне зоны нагрева"}, tags=("S02",), catalog="E-90",
           label="S02/confirm-NC-04")
    camera(g, T("2026-09-23 13:55"), "CAM-WS-2", "F-015", "CP-WELD", "after_operation", SEGS, tags=("S05",),
           label="kt3/F-015/recheck", extra={"trigger": "recheck", "note": "повторная камера по перепроверке"})
    g.step(T("2026-09-23 14:00"), "QC-01", "QC", "risk_scope.item_assessed",
           {"risk_scope_ref": "RS-01", "item_id": "F-015"},
           {"result": "no_defects_found", "group_after": "suspected",
            "basis_labels": ["xr/F-015/RK-0922-13", "kt3/F-015/recheck"],
            "note": "дефектов нет, но сварка вне режима — решение на уровне инцидента"},
           tags=("S05",), catalog="E-113", label="S05/assess-F-015")
    g.step(T("2026-09-23 14:05"), "QC-01", "QC", "decision.review_completed", {"gate": "ZT-3", "item_id": "F-015"},
           {"outcome": "revoked", "text": "решение ЗТ-3 пересмотрено: приёмка отозвана, изделие в инциденте RS-01, ждёт решения",
            "original_decision_label": "ZT-3/F-015", "original_decision": "accept_incomplete_data",
            "flag": "decision.review_flagged (Е-122 [П], реакция движка, Ср 12:06, узел R1)",
            "due_to_label": "EV-WS2-0412",
            "then": "R3 — блок изделия; R4 — в инцидент RS-01 (изделие уже в круге); исходная подпись остаётся"},
           tags=("S07",), catalog="E-123 [П]", label="S07/review-ZT3-F015")
    xray(g, T("2026-09-23 14:20"), "F-027", "RK-0923-12", tags=("S02",))
    g.step(T("2026-09-23 14:25"), "TECH-01", "TECHNOLOGIST", "risk_scope.item_assessed",
           {"risk_scope_ref": "RS-02", "item_id": "F-027"}, {"result": "excluded", "basis_labels": ["xr/F-027/RK-0923-12"],
            "holds_released": "containment.hold_released (Е-116 [П], узел I5)"},
           tags=("S02",), catalog="E-113", label="S02/exclude-F-027")
    xray(g, T("2026-09-23 14:40"), "F-029", "RK-0923-13", tags=("S02",))
    g.step(T("2026-09-23 14:45"), "TECH-01", "TECHNOLOGIST", "risk_scope.item_assessed",
           {"risk_scope_ref": "RS-02", "item_id": "F-029"}, {"result": "excluded", "basis_labels": ["xr/F-029/RK-0923-13"],
            "holds_released": "containment.hold_released (Е-116 [П], узел I5)"},
           tags=("S02",), catalog="E-113", label="S02/exclude-F-029")
    g.step(T("2026-09-23 14:50"), "QC-02", "QC", "decision.recheck_requested",
           {"lot_id": "LOT-R-117", "items": ["R-105", "R-106"]},
           {"method": "xray", "sampling": "выборка со склада; К-104 уже было в выборке на входе (Вт 10:39, чисто) — его не повторяем"},
           tags=("S02",), catalog="E-92", label="S02/recheck-rings")
    xray(g, T("2026-09-23 15:00"), "R-106", "RK-0923-14", zones=["RING-BODY"], tags=("S02",))
    xray(g, T("2026-09-23 15:15"), "R-105", "RK-0923-11", zones=["RING-BODY"], tags=("S02",),
         defects=[OrderedDict([("defect_type", "base_metal_pore"), ("zone", "RING-BODY"), ("severity", "major"),
                               ("size_mm", 0.9)])])
    g.step(T("2026-09-23 15:25"), "QC-02", "QC", "decision.signal_confirmed",
           {"item_id": "R-105", "signal_basis_labels": ["xr/R-105/RK-0923-11"], "nc_ref": "NC-05"},
           {"defect_type": "base_metal_pore", "zone": "RING-BODY", "reason": "пора 0,9 мм в теле кольца"},
           tags=("S02",), catalog="E-90", label="S02/confirm-NC-05")
    # S05: решение по инциденту (СТОП) — комиссия
    for it in SIX:
        g.step(T("2026-09-23 15:30"), "TECH-01", "COMMISSION", "risk_scope.item_assessed",
               {"risk_scope_ref": "RS-01", "item_id": it},
               {"result": "confirmed", "reason": "сварка вне режима ТП" if it != "F-019" else "соответствие режиму не доказано"},
               tags=("S05",), catalog="E-113", signers=["TECH-01", "QC-HEAD-01", "CHW-01"], label="S05/assess/" + it)
    g.step(T("2026-09-23 15:30"), "TECH-01", "COMMISSION", "decision.signal_confirmed",
           {"items": SIX, "nc_ref": "NC-G1", "nc_kind": "group", "risk_scope_ref": "RS-01"},
           {"reason": "сварка вне режима ТП (спецпроцесс)", "special_process": True},
           mode="stop", tags=("S05",), catalog="E-90", signers=["TECH-01", "QC-HEAD-01", "CHW-01"], label="S05/NC-G1")
    g.step(T("2026-09-23 15:30"), "TECH-01", "COMMISSION", "decision.disposition_set",
           {"items": SIX, "nc_ref": "NC-G1"},
           {"disposition": "rework", "per_item_notes": {"F-019": "переделка с заменой кольца"},
            "reason": "переварка шва на исправном источнике ИС-1",
            "item_messages": [
                {"item_id": "F-015", "block": "BA", "caught_by": "BAD", "route": "BAG → BB (B1 вскрытие → B2…B4) → BW (WSG → WL1 → WL2 → WLG → WL3)"},
                {"item_id": "F-017", "block": "BW", "caught_by": "BWD", "route": "BWG → BW (WSG → WL1 → WL2 → WLG → WL3)"},
                {"item_id": "F-019", "block": "BW", "caught_by": "BWD", "route": "BWG → BW (WSG → WRX → V6: ждёт кольцо)"},
                {"item_id": "F-021", "block": "BW", "caught_by": "BWD", "route": "BWG → BW (WSG → WL1 → WL2 → WLG → WL3)"},
                {"item_id": "F-023", "block": "BW", "caught_by": "BWD", "route": "BWG → BW (WSG → WL1 → WL2 → WLG → WL3)"},
                {"item_id": "F-025", "block": "BW", "caught_by": "BWD", "route": "BWG → BW (WSG → WL1 → WL2 → WLG → WL3)"}],
            "item_messages_note": "реакция движка (узел I7, сообщение Msg_ItemDisposition): одно групповое решение — "
                                  "шесть сообщений, по одному каждому изделию; в 1С — одно сообщение на список"},
           mode="stop", tags=("S05",), catalog="E-94", signers=["TECH-01", "QC-HEAD-01", "CHW-01"],
           label="S05/disposition-rework")
    g.step(T("2026-09-23 15:35"), "TECH-01", "TECHNOLOGIST", "risk_scope.closed", {"risk_scope_ref": "RS-01"},
           {"was": 34, "confirmed": 6, "excluded": 28}, tags=("S05",), catalog="E-115", label="S05/close")
    # S02: причина и возврат поставщику, ошибка 1С 422 и исправление ID
    g.step(T("2026-09-23 15:40"), "TECH-01", "TECHNOLOGIST", "cause.confirmed", {"nc_refs": ["NC-04", "NC-05"]},
           {"category": "incoming", "supplier_id": "SUP-3", "lot_id": "LOT-R-117",
            "verified_by": "две находки в теле колец одной партии, зоны без операций"},
           tags=("S02",), catalog="П-04", signers=["TECH-01", "QC-HEAD-01"], label="S02/cause")
    # S02, строка 48 ред-тима: честная граница входного контроля — мера «улучшить обнаружение» с проверкой результативности
    g.step(T("2026-09-23 15:50"), "QC-HEAD-01", "QC_HEAD", "capa.action_assigned", {"nc_refs": ["NC-04", "NC-05"]},
           {"kind": "improve_detection", "lot_id": "LOT-R-117",
            "text": "выборочный рентген 1 кольца из партии на входе [ПП] пору не поймал (в выборку попало чистое К-104): "
                    "выборка чистая ≠ вся партия чистая. Для колец Поставщика-3 — рентген каждого кольца до сварки, "
                    "пока три партии подряд не будут чистыми [П]",
            "owner": "QC-HEAD-01", "due": "2026-10-09",
            "effectiveness_plan": {"metric": "поры в теле кольца, найденные после сварки", "baseline": 2,
                                   "window": "следующие 3 партии колец", "success": 0,
                                   "on_fail": "мера снова открыта, разбор заново"}},
           tags=("S02",), catalog="П-05", label="S02/capa")
    g.step(T("2026-09-23 15:45"), "TECH-01", "TECHNOLOGIST", "decision.disposition_set",
           {"lot_id": "LOT-R-117", "items": ["R-%03d" % n for n in range(104, 111)], "nc_refs": ["NC-04", "NC-05"]},
           {"disposition": "return_to_supplier", "qty": 7,
            "claim_basis_labels": ["xr/F-019/RK-0923-07", "xr/R-105/RK-0923-11"]},
           tags=("S02", "I1"), catalog="E-94", label="S02/return")
    g.stand_faults.append(OrderedDict([
        ("id", "FAULT-1C-422-LOT-R-117"), ("system", "erp_1c"), ("scenarios", ["S02", "I1"]),
        ("match", {"posting_type": "return_to_supplier", "lot_id": "LOT-R-117"}),
        ("response", {"http_status": 422, "text": "не найден договор с контрагентом ctr-0b19-0003"}),
        ("active_until_step", "S02/id-map"),
        ("expected_behavior", "ошибка данных → без автоповтора, задача А-01; после соответствия — повтор с тем же номером; подтверждение Ср 23.09 16:11"),
        ("retry_ack_at", iso(T("2026-09-23 16:11")))]))
    g.step(T("2026-09-23 16:10"), "ADM-01", "DATA_ADMIN", "erp.id_map_added",
           {"system": "erp_1c", "entity": "supplier_contract", "internal": "SUP-3"},
           {"external": "dog-0b19-0003-2026", "then": "повторить отправку с тем же номером сообщения"},
           tags=("S02", "I1"), catalog="[П] нет в каталоге", label="S02/id-map")
    # КО-7 и причина по инциденту (СТОП)
    g.ev(T("2026-09-23 15:58"), "TERM-WS-2", "operation.started", item="CS-07",
         payload=OrderedDict([("operation_run_id", "SV-CS07-1"), ("operation_code", "W2"), ("station_id", "ST-WC-P2"),
                              ("operator_id", "CHW-01"), ("equipment_id", "WS-2"),
                              ("program_id", "PS-4"), ("purpose", "control_sample"), ("item_type_id", "CONTROL-SAMPLE")]),
         tags=("S05",), corr="SV-CS07-1", label="SV-CS07-1.start", persona="CHW-01")
    g.ev(T("2026-09-23 16:12"), "TERM-WS-2", "operation.finished", item="CS-07",
         payload=OrderedDict([("operation_run_id", "SV-CS07-1"), ("result", "completed"), ("operator_id", "CHW-01")]),
         tags=("S05",), corr="SV-CS07-1", label="SV-CS07-1.finish", persona="CHW-01")
    g.welds.append(dict(item="CS-07", src="WS-2", welder="CHW-01", start=T("2026-09-23 15:58"), end=T("2026-09-23 16:12"),
                        arc=(T("2026-09-23 16:00:00"), T("2026-09-23 16:10:00")), run="SV-CS07-1",
                        profile=[(T("2026-09-23 16:00:00"), 180, 182)], cs=True))
    g.power.append(("WS-2", T("2026-09-23 15:55"), T("2026-09-23 16:15"), "end_of_work", None))
    g.ev(T("2026-09-23 16:15"), "TERM-WS-2", "inspection.result", item="CS-07",
         payload=OrderedDict([("checkpoint_id", "CP-WELD"), ("phase", "after_operation"), ("method", "visual"),
                              ("operator_id", "CHW-01"), ("inspection_result", "defect_found"), ("purpose", "control_sample"),
                              ("defects", [OrderedDict([("defect_type", "burn_through"), ("zone", "U2"),
                                                        ("severity", "major"), ("description", "прожог повторился")])])]),
         tags=("S05",), label="CS-07/visual", persona="CHW-01")
    g.step(T("2026-09-23 16:20"), "TECH-01", "TECHNOLOGIST", "cause.confirmed",
           {"nc_refs": ["NC-01", "NC-02", "NC-03", "NC-G1"]},
           {"category": "equipment", "equipment_id": "WS-2", "text": "дрейф регулятора тока ИС-2",
            "verified_by": "контрольный образец CS-07 + журнал", "evidence_labels": ["CS-07/visual", "EV-WS2-0412"]},
           mode="stop", tags=("S05",), catalog="П-04", signers=["TECH-01", "CHW-01"], label="S05/cause")
    # S09: подмена записи (инструмент вне системы) и попытка «исправить» через API
    g.tamper.append(OrderedDict([
        ("id", "TAMPER-EV-WS2-0412"), ("scenario", "S09"), ("at", iso(T("2026-09-23 16:30"))),
        ("tool", "cmd/tamper (только профили fixtures/demo; прямое подключение к БД в обход системы)"),
        ("target_label", "EV-WS2-0412"), ("field", "payload.value"), ("from", 176), ("to", 166),
        ("expected", "нарушение звена цепочки на этой записи; журнал критических действий; уведомления ADM-01, QC-HEAD-01")]))
    g.step(T("2026-09-23 16:31"), "ADM-01", "DATA_ADMIN", "integrity.check_requested", {"scope": "journal"},
           {"trigger": "button"}, tags=("S09",), catalog="[П] нет в каталоге", optional=True,
           note="если проверка целостности у соседей идёт по расписанию — шаг не нужен", label="S09/check")
    g.step(T("2026-09-23 16:35"), "ADM-01", "DATA_ADMIN", "event.corrected",
           {"target_label": "EV-WS2-0412", "method": "update_in_place"},
           {"field": "payload.value", "from": 166, "to": 176, "reason": "попытка вернуть значение правкой записи"},
           tags=("S09",), catalog="E-100", expect="refused",
           note="ожидается отказ: «исправление — только новой записью (event.corrected)»; правки на месте нет",
           label="S09/update-refused")
    # S09, волна 2: ещё две атаки (ред-тим, строка 26; ревью безопасности В11 — сверка Н8)
    g.tamper.append(OrderedDict([
        ("id", "TAMPER-RECHAIN-RK-0923-07"), ("scenario", "S09"), ("at", iso(T("2026-09-23 16:50"))),
        ("tool", "cmd/tamper (только профили fixtures/demo; «администратор БД»: прямое подключение к БД)"),
        ("method", "update_and_rechain"),
        ("target_label", "xr/F-019/RK-0923-07"), ("field", "payload.defects"),
        ("from", "пора 1,2 мм в теле кольца (base_metal_pore, RING-BODY)"), ("to", []),
        ("also", "пересчитаны звенья цепочки от этой записи до конца журнала — простая проверка звеньев проходит"),
        ("expected", "нарушение найдено: подпись источника (edge-агент EA-NDT) не сходится с изменённым содержимым; "
                     "голова цепочки не совпадает с контрольной отметкой, которую хранитель держит вне базы (16:45) [П]; "
                     "журнал критических действий; уведомления ADM-01, QC-HEAD-01")]))
    g.step(T("2026-09-23 16:51"), "ADM-01", "DATA_ADMIN", "integrity.check_requested", {"scope": "journal_and_anchors"},
           {"trigger": "button"}, tags=("S09",), catalog="Е-105 [П]", optional=True,
           note="если сверка с контрольными отметками идёт по расписанию — шаг не нужен", label="S09/check-2")
    g.tamper.append(OrderedDict([
        ("id", "TAMPER-PROJECTION-F-023"), ("scenario", "S09"), ("at", iso(T("2026-09-23 17:00"))),
        ("tool", "cmd/tamper (только профили fixtures/demo; прямое изменение таблицы представлений)"),
        ("method", "projection_update"),
        ("target", "projection: item_status / F-023"), ("field", "status"), ("from", "rework"), ("to", "accepted"),
        ("expected", "сверка представлений с журналом: пересборка даёт «в переделке», в таблице «годно» → расхождение; "
                     "представление пересобрано из журнала; журнал событий не тронут; F-023 никуда не двинулся")]))
    g.step(T("2026-09-23 17:01"), "ADM-01", "DATA_ADMIN", "integrity.check_requested", {"scope": "projections"},
           {"trigger": "button"}, tags=("S09",), catalog="Е-105 [П]", optional=True,
           note="если сверка представлений идёт по расписанию — шаг не нужен", label="S09/check-3")
    g.step(T("2026-09-24 07:45"), "QC-HEAD-01", "QC_HEAD", "integrity.violation_resolved",
           {"targets": ["EV-WS2-0412", "xr/F-019/RK-0923-07"]},
           {"authentic_version": "версия, с которой сходится подпись источника (из реплики или резервной копии; способ — у соседей)",
            "result": {"EV-WS2-0412": "подлинное значение 176 А; подменённое 166 А отвергнуто",
                       "xr/F-019/RK-0923-07": "подлинное заключение — пора 1,2 мм в теле кольца; подмена отвергнута"},
            "flags_removed": "«основание под вопросом» снято с зависимых выводов; изделия снова могут проходить ЗТ",
            "projection_F-023": "пересобрано из журнала в Ср 17:01, отдельного разбора не нужно"},
           tags=("S09",), catalog="Е-103 [П]", signers=["QC-HEAD-01", "ADM-01"], label="S09/resolved")

    # S12: исправление сообщения 1 из карантина и (по желанию) отклонение сигнала по сообщению 5
    g.step(T("2026-09-22 14:30"), "ADM-01", "DATA_ADMIN", "item.binding_corrected",
           {"quarantine_ref_label": "kt3/SV-024-1/CAM-WS-2"},
           {"item_id": "F-024", "method": "post_context", "then": "quarantine.reprocessed [П] — повторная обработка из карантина"},
           tags=("S12",), catalog="E-101 + [П] quarantine.reprocessed", label="S12/reprocess", hints=["F-024"])
    g.step(T("2026-09-22 15:00"), "QC-01", "QC", "decision.signal_rejected",
           {"item_id": "F-026", "signal_basis_labels": ["kt3/SV-026-1/CAM-WS-2"]},
           {"reason": "брызги вне шва; класс spatter_x неизвестен словарю — передано владельцу классификатора"},
           tags=("S12",), catalog="E-91", optional=True, note="нужен, только если движок поднял сигнал по сообщению 5 [П]",
           label="S12/reject-spatter")

    # ---------------- S10A: Чт 24.09 — переварка на ИС-1, повторный контроль
    rework = [("F-017", "2026-09-24 08:20", "2026-09-24 09:00", "08:10", "2026-09-23 16:40", "08:00"),
              ("F-023", "2026-09-24 09:10", "2026-09-24 09:50", "08:12", "2026-09-23 17:15", "08:02"),
              ("F-025", "2026-09-24 10:00", "2026-09-24 10:40", "08:14", "2026-09-23 17:50", "08:04"),
              ("F-015", "2026-09-24 10:50", "2026-09-24 11:30", "08:16", "2026-09-23 18:25", "08:06"),
              ("F-021", "2026-09-24 12:50", "2026-09-24 13:30", "12:30", "2026-09-23 19:00", "08:08")]
    for it, s, e, prep, wl1, wl2 in rework:
        # схема v0.3 (WL1 → WL2 → WLG → WL3): выборка — удаление всего шва (С-03, смена Б в среду),
        # контроль выборки — ОТК-1 до заварки (Чт утром); первая переделка — без согласия главного сварщика
        excavation(g, it, T(wl1), "WLD-03", "SV-%s-1" % num(it), SEGS, "весь шов (переварка по решению НС-И1)",
                   T("2026-09-24 " + wl2), tags=("S10A",))
        w = dict(item=it, src="WS-1", welder="WLD-02", start=T(s), end=T(e), arc=(T(s) + M(2), T(e) - M(3)),
                 wire="LOT-W-88", prep=T("2026-09-24 " + prep), prep_step="edge_prep_after_excavation",
                 prep_text="кромки после выборки подготовлены", run="SV-%s-2" % num(it),
                 rework_of="SV-%s-1" % num(it), rework_zones=SEGS, profile=None)
        weld(g, w, tags=("S10A",))
    # Ф-019: схема v0.2, WRX — шов удалён, кольцо К-101 снято и отложено в претензию; V6 ждёт новое кольцо (пятница)
    g.ev(T("2026-09-24 13:40"), "TERM-WC", "assembly.component_unlinked", item="F-019",
         payload=OrderedDict([("assembly_item_id", "F-019"), ("component_id", "R-101"), ("component_type_id", "FL-100-RING"),
                              ("lot_id", "LOT-R-117"), ("link_id", "W-1"), ("binding_method", "scan"),
                              ("weld_removed", True), ("reason_nc_ref", "NC-G1"),
                              ("reason", "решение ЗТ-Р по НС-И1: переделка с заменой кольца"),
                              ("destination", "claim"), ("claim_basis", "NC-04"), ("operator_id", "WLD-02")]),
         tags=("S10A", "S05"), label="F-019/unlink-R-101", persona="WLD-02", hints=["F-019", "R-101"],
         note="Е-49 [П] assembly.component_unlinked: связь «кольцо → изделие» закрыта, изделие то же; новое кольцо — снова Е-41")
    XR_THU = [("F-017", "11:00", "RK-0924-01", None), ("F-023", "11:40", "RK-0924-02", None),
              ("F-025", "12:20", "RK-0924-03", None), ("F-015", "13:00", "RK-0924-05", None),
              ("F-021", "14:30", "RK-0924-04", [OrderedDict([("defect_type", "porosity"), ("zone", "U5"),
                                                             ("severity", "major")])])]
    for it, hm, no, dfx in XR_THU:
        xray(g, at(dt.date(2026, 9, 24), hm), it, no, defects=dfx, tags=("S10A",))
    for it, hm in (("F-017", "12:00"), ("F-023", "12:40"), ("F-025", "13:20"), ("F-015", "15:10")):
        tp = at(dt.date(2026, 9, 24), hm) - M(5)
        g.ev(tp, "TERM-WC", "item.presented", item=it,
             payload=OrderedDict([("item_id", it), ("presentation_no", 2), ("presented_to", ["QC"]),
                                  ("presented_by", "MST-02"), ("gate", "ZT-3")]),
             tags=("S10A",), label="present/%s/ZT-3#2" % it, persona="MST-02")
        g.step(at(dt.date(2026, 9, 24), hm), "QC-01", "QC", "decision.disposition_verified",
               {"item_id": it, "nc_ref": "NC-G1"},
               {"result": "passed", "methods": ["camera", "xray"], "rework_run": "SV-%s-2" % num(it)},
               tags=("S10A",), catalog="E-96", label="S10A/verified/" + it)
        gate(g, at(dt.date(2026, 9, 24), hm) + S(30), "QC-01", "ZT-3", item=it, tags=("S10A",), presentation=2,
             basis=["kt3/SV-%s-2/CAM-WS-1" % num(it)], label="ZT-3#2/" + it)
    g.step(T("2026-09-24 14:35"), "QC-01", "QC", "decision.signal_confirmed",
           {"item_id": "F-021", "signal_basis_labels": ["xr/F-021/RK-0924-04"], "nc_ref": "NC-06"},
           {"defect_type": "porosity", "zone": "U5", "run": "SV-021-2", "reason": "поры в новом шве"},
           tags=("S10A",), catalog="E-90", label="S10A/confirm-NC-06")
    g.step(T("2026-09-24 14:35:30"), "QC-01", "QC", "decision.disposition_verified", {"item_id": "F-021", "nc_ref": "NC-G1"},
           {"result": "failed", "reason": "повторный контроль не пройден: НС-06"}, tags=("S10A",), catalog="E-96",
           label="S10A/verify-failed-F-021")
    g.step(T("2026-09-24 15:30"), "CHW-01", "CHIEF_WELDER", "decision.disposition_set",
           {"item_id": "F-021", "nc_refs": ["NC-06", "NC-G1"]},
           {"disposition": "scrap", "supersedes_label": "S05/disposition-rework",
            "reason": "после удаления шва стенка у У5 на нижней границе; повторную переделку того же места не назначаем"},
           mode="stop", tags=("S10A",), catalog="E-94", signers=["CHW-01", "TECH-01"], label="S10A/scrap-F-021")


# ----------------------------------------------------------------------------- самостоятельный мир S13
def s13_world(g, W):
    D24, D25 = dt.date(2026, 9, 24), dt.date(2026, 9, 25)
    tg = ("S13",)
    # Чт 24.09 — пролог: РК, ЗТ-3 для F-026/F-028, передачи, сборка F-026, предъявления
    for k, (it, hm) in enumerate((("F-026", "08:20"), ("F-028", "08:40"), ("F-024", "09:00"), ("F-030", "09:20"),
                                  ("F-031", "09:40"), ("F-032", "10:00"))):
        xray(g, at(D24, hm), it, "RK-0924-%02d" % (21 + k), tags=tg)
    gate(g, at(D24, "10:30"), "QC-01", "ZT-3", item="F-026", tags=tg, basis=["xr/F-026/RK-0924-21"], label="ZT-3/F-026")
    gate(g, at(D24, "10:45"), "QC-01", "ZT-3", item="F-028", tags=tg, basis=["xr/F-028/RK-0924-22"], label="ZT-3/F-028")
    for it in ("F-026", "F-028"):
        move(g, at(D24, "14:00"), it, "WC", "AC", "MST-02", "MST-03", tags=tg, label="mv/%s/WC-AC" % it)
    kit_issue(g, at(D24, "14:10"), "F-026", "C-026", "LOT-SL-401", "LOT-FS-501", "LOT-C-301", tags=tg)
    assembly(g, "F-026", at(D24, "14:15"), "C-026", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", tags=tg,
             zt41=at(D24, "14:40"), zt4=at(D24, "15:30"))
    for k, it in enumerate(("F-024", "F-030", "F-031", "F-032")):
        g.ev(at(D24, "16:05") + M(5 * k), "TERM-WC", "item.presented", item=it,
             payload=OrderedDict([("item_id", it), ("presentation_no", 1), ("presented_to", ["QC"]),
                                  ("presented_by", "MST-02"), ("gate", "ZT-3")]), tags=tg,
             label="present/%s/ZT-3" % it, persona="MST-02")
    # Пт 25.09
    g.ev(at(D25, "07:55"), "TERM-STK", "batch.issued", lot="LOT-C-301",
         payload=OrderedDict([("lot_id", "LOT-C-301"), ("qty", 1), ("to_station", "ST-AC-ASM"), ("issued_by", "STK-01"),
                              ("component_ids", ["C-028"]), ("for_items", ["F-028"])]), tags=tg, label="kit/F-028/cover",
         persona="STK-01", hints=["F-028"])
    issue(g, at(D25, "07:55:20"), "LOT-SL-401", 1, "ST-AC-ASM", items=["F-028"], tags=tg, label="kit/F-028/seal")
    issue(g, at(D25, "07:55:40"), "LOT-FS-501", 12, "ST-AC-ASM", items=["F-028"], tags=tg, label="kit/F-028/fasteners")
    issue(g, at(D25, "07:55:50"), "LOT-V-601", 1, "ST-AC-ASM", items=["F-028"], components=["V-028"], tags=tg,
          label="kit/F-028/valve")
    assembly(g, "F-028", at(D25, "08:00"), "C-028", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", tags=tg,
             zt41=at(D25, "08:30"), zt4=at(D25, "09:25"),
             plan=dict(seal=at(D25, "08:15"), uv=at(D25, "08:20"), cover=at(D25, "08:40"), fasteners=at(D25, "08:50"),
                       torque=at(D25, "09:05"), finish=at(D25, "09:20")))
    leak_test(g, "F-026", at(D25, "08:00"), tags=tg, dur=75, zt5=at(D25, "09:20"))
    # комплекты по сменному заданию под F-024, F-030, F-031 (08:10)
    for k, it in enumerate(("F-024", "F-030", "F-031")):
        kit_issue(g, at(D25, "08:10") + M(k), it, cover_of(it)[0], "LOT-SL-401", "LOT-FS-501", "LOT-C-301", tags=tg,
                  label="kit/%s@0810" % it)
    gate(g, at(D25, "08:15"), "QC-01", "ZT-3", item="F-024", tags=tg, basis=["xr/F-024/RK-0924-23"], label="ZT-3/F-024")
    move(g, at(D25, "08:20"), "F-024", "WC", "AC", "MST-02", "MST-03", recv_delay=15, tags=tg, label="mv/F-024/WC-AC",
         after_recv=None)
    g.step(at(D25, "09:05"), "MST-02", "MASTER", "escalation.acknowledged", {"gate": "ZT-3", "escalation_level": "master"},
           {"text": "Ищу контролёра", "stops_deadlines": False}, tags=tg, catalog="[П] S13", label="S13/ack")
    for k, it in enumerate(("F-033", "F-034", "F-035")):
        xray(g, at(D25, "09:20") + S(20 * k), it, "RK-0925-%02d" % (k + 1), tags=tg)
        g.ev(at(D25, "09:21") + S(20 * k), "TERM-WC", "item.presented", item=it,
             payload=OrderedDict([("item_id", it), ("presentation_no", 1), ("presented_to", ["QC"]),
                                  ("presented_by", "MST-02"), ("gate", "ZT-3")]), tags=tg,
             label="present/%s/ZT-3" % it, persona="MST-02")
    leak_test(g, "F-028", at(D25, "09:30"), tags=tg, dur=70, zt5=at(D25, "10:42"))
    assembly(g, "F-024", at(D25, "09:20"), "C-024", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", tags=tg,
             zt41=at(D25, "09:50"), zt4=at(D25, "10:40"),
             plan=dict(seal=at(D25, "09:35"), uv=at(D25, "09:40"), cover=at(D25, "09:55"), fasteners=at(D25, "10:05"),
                       torque=at(D25, "10:18"), finish=at(D25, "10:30")))
    for k, it in enumerate(("F-036", "F-223", "F-224")):
        xray(g, at(D25, "10:20") + S(20 * k), it, "RK-0925-%02d" % (k + 4), tags=tg)
        g.ev(at(D25, "10:21") + S(20 * k), "TERM-WC", "item.presented", item=it,
             payload=OrderedDict([("item_id", it), ("presentation_no", 1), ("presented_to", ["QC"]),
                                  ("presented_by", "MST-02"), ("gate", "ZT-3")]), tags=tg,
             label="present/%s/ZT-3" % it, persona="MST-02")
    op_action(g, at(D25, "10:30:30"), "TERM-ASM", None, "ASM-01", "idle_reported",
              OrderedDict([("station_id", "ST-AC-ASM"), ("reason", "no_part"),
                           ("text", "Ф-024 собран; следующего фланца нет")]), tags=tg, label="S13/asm-no-part")
    leak_test(g, "F-024", at(D25, "10:45"), tags=tg, dur=75, zt5=at(D25, "12:05"))
    g.step(at(D25, "10:40"), "MST-02", "MASTER", "post.assignment_requested", {"post": "ZT-3", "station": "ST-WC-ZT3"},
           {"option": "second_controller", "assignee": "QC-02", "until": "очередь пуста"},
           mode="stop", tags=tg, catalog="[П] S13", label="S13/request")
    g.step(at(D25, "10:45"), "QC-HEAD-01", "QC_HEAD", "post.assignment_changed", {"post": "ZT-3", "station": "ST-WC-ZT3"},
           {"assignee": "QC-02", "from": iso(at(D25, "10:45")), "requested_by": "MST-02",
            "checks": ["допуск к приёмке сварных швов (STAMP-QC-02-WELD)", "не участвовала в изготовлении"]},
           mode="stop", tags=tg, catalog="[П] S13", label="S13/approve")
    s_ids = {}
    for k, it in enumerate(("F-030", "F-031", "F-032", "F-033", "F-034")):
        z = at(D25, "11:00") + M(10 * k)
        s_ids[it] = gate(g, z, "QC-02", "ZT-3", item=it, tags=tg, mode="stop", basis=["xr/%s" % it],
                         label="ZT-3/" + it)["label"]
        move(g, z + M(5), it, "WC", "AC", "MST-02", "MST-03", tags=tg, label="mv/%s/WC-AC" % it)
    for k, it in enumerate(("F-035", "F-036", "F-223", "F-224")):
        z = at(D25, "11:40") + M(10 * k)
        gate(g, z, "QC-01", "ZT-3", item=it, tags=tg, mode="stop", basis=["xr/%s" % it], label="ZT-3/" + it)
        move(g, z + M(5), it, "WC", "AC", "MST-02", "MST-03", tags=tg, label="mv/%s/WC-AC" % it)
    g.step(at(D25, "11:40:30"), "MST-02", "MASTER", "post.assignment_changed", {"post": "ZT-3", "station": "ST-WC-ZT3"},
           {"assignee": "QC-02", "to": iso(at(D25, "11:40")), "text": "назначение закончено, ОТК-2 возвращается в СИЦ"},
           tags=tg, catalog="[П] S13", label="S13/end")
    assembly(g, "F-030", at(D25, "11:20"), "C-030", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", tags=tg,
             zt41=at(D25, "11:45"), zt4=at(D25, "12:40"),
             plan=dict(seal=at(D25, "11:32"), uv=at(D25, "11:38"), cover=at(D25, "11:50"), fasteners=at(D25, "11:58"),
                       torque=at(D25, "12:10"), finish=at(D25, "12:35")))
    g.ev(at(D25, "12:00"), "LT-1", "machine.state",
         payload=OrderedDict([("equipment_id", "LT-1"), ("state", "idle"), ("reason", "no_part"),
                              ("text", "Ф-024 испытан; следующего фланца нет")]), tags=tg, label="S13/lt-idle")
    issue(g, at(D25, "12:15"), "LOT-SL-401", 1, "ST-AC-ASM", items=["F-031"], tags=tg, label="S13/seal-reissue",
          note="по правилу R-21 (по желанию): прежнее уплотнение — на склад для оценки, не списано")
    leak_test(g, "F-030", at(D25, "12:45"), tags=tg, dur=75, zt5=False)


# ----------------------------------------------------------------------------- S10B: самостоятельный прогон
def build_s10b():
    g = Gen("S10B", 20260928)
    D = dt.date(2026, 9, 28)
    tg = ("S10B",)
    order(g, at(D, "07:30"), "ORD-0928", "ord-5c2b-0928", "1c-in-000330", 1, "2026-10-09", "F-090", "F-090", tags=tg)
    for k, lot in enumerate(("LOT-ZF-201", "LOT-R-119", "LOT-W-88")):
        lot_received(g, at(D, "07:40") + S(30 * k), lot, tags=tg)
    t = iqc(g, at(D, "07:45"), "LOT-ZF-201", ["F-090"], True, True, at(D, "07:45"), tags=tg)
    t = iqc(g, t + M(1), "LOT-R-119", ["R-901"], True, True, t, tags=tg)
    t = iqc(g, t + M(1), "LOT-W-88", None, False, False, t, tags=tg)
    register(g, at(D, "08:05"), "F-090", "ORD-0928", "LOT-ZF-201", tags=tg)
    machining(g, "F-090", at(D, "08:10"), 50, tags=tg)
    cmm(g, "F-090", at(D, "09:25"), tags=tg)
    gate(g, at(D, "09:30"), "QC-01", "ZT-2", item="F-090", tags=tg, basis=["kt2/F-090", "cmm/F-090"], label="ZT-2/F-090")
    move(g, at(D, "09:35"), "F-090", "MC", "WC", "MST-01", "MST-02", tags=tg, label="mv/F-090/MC-WC")
    issue(g, at(D, "09:40"), "LOT-W-88", 1, "ST-WC-P1", tags=tg, label="issue/wire")
    issue(g, at(D, "09:47"), "LOT-R-119", 1, "ST-WC-P1", items=["F-090"], components=["R-901"], tags=tg,
          label="issue/ring")   # V6: после приёмки фланца сварочным цехом (09:45)
    undercut = lambda conf: [OrderedDict([("defect_type", "undercut"), ("zone", "U4"), ("side", "face"),
                                          ("severity", "major"), ("confidence", conf)])]
    runs = [("SV-090-1", None, "09:55", "10:00", "10:40", None),
            ("SV-090-2", "SV-090-1", "11:14", "11:20", "11:40", "NC-87"),
            ("SV-090-3", "SV-090-2", "12:40", "13:00", "13:20", "NC-88"),
            ("SV-090-4", "SV-090-3", "14:04", "14:10", "14:30", "NC-89")]
    # схема v0.3 (WL1 → WL2 → WLG → WLA → WL3): перед каждой переделкой — выборка и контроль выборки;
    # второе и третье исправление участка — согласие главного сварщика отдельной записью (Е-99 [П]) после контроля выборки
    wl = {"SV-090-2": ("11:02", "11:12", None), "SV-090-3": ("12:02", "12:30", "12:35"),
          "SV-090-4": ("13:42", "13:52", "13:56")}
    for k, (run, prev, prep, s, e, prev_nc) in enumerate(runs):
        if prev:
            t_wl1, t_wl2, t_wla = wl[run]
            excavation(g, "F-090", at(D, t_wl1), "WLD-02", prev, ["U4"], "участок У4 по снимку и заключению", at(D, t_wl2),
                       tags=tg, dur=8, source="TERM-WS-1")
            if t_wla:
                g.step(at(D, t_wla), "CHW-01", "CHIEF_WELDER", "rework.chief_welder_approved",
                       {"item_id": "F-090", "zone": "U4"},
                       {"rework_no": k, "of_limit": 3,
                        "basis": "выборка проконтролирована ОТК-1; переделка участка по ТП, %d-я" % k},
                       tags=tg, catalog="Е-99 [П]", label="S10B/chief-welder-approval-%d" % k)
        w = dict(item="F-090", src="WS-1", welder="WLD-02", start=at(D, s), end=at(D, e),
                 arc=(at(D, s) + M(2), at(D, e) - M(3)), ring="R-901", ring_lot="LOT-R-119", wire="LOT-W-88",
                 prep=at(D, prep), run=run, rework_of=prev, profile=None,
                 rework_zones=["U4"] if prev else None,
                 prep_step="edge_prep" if not prev else "rework_zone_prep",
                 prep_text="кромки подготовлены" if not prev else "участок У4 после выборки подготовлен к заварке")

        def ov(gg, t0, run=run, k=k):
            camera(gg, t0, "CAM-WS-1", "F-090", "CP-WELD", "after_operation", SEGS, result="defect_found",
                   q=0.91, conf=[0.88, 0.86, 0.87, 0.89][k], defects=undercut([0.88, 0.86, 0.87, 0.89][k]), tags=tg,
                   label="kt3/%s/CAM-WS-1" % run)
        weld(g, w, tags=tg, cam_override=ov)
        nc = ["NC-87", "NC-88", "NC-89", "NC-90"][k]
        tc = at(D, e) + M(10)
        g.step(tc, "QC-01", "QC", "decision.signal_confirmed",
               {"item_id": "F-090", "signal_basis_labels": ["kt3/%s/CAM-WS-1" % run], "nc_ref": nc},
               {"defect_type": "undercut", "zone": "U4", "severity": "major", "reason": "подрез на участке У4"},
               mode="stop" if nc == "NC-90" else "auto", tags=tg, catalog="E-90", label="S10B/confirm-" + nc)
        if k < 3:
            g.step(tc + M(10), "TECH-01", "TECHNOLOGIST",
                   "decision.disposition_set", {"item_id": "F-090", "nc_ref": nc},
                   {"disposition": "rework", "rework_zones": ["U4"], "rework_no": k + 1,
                    "reason": "переделка участка У4 по ТП" + ("" if k == 0 else "; согласие главного сварщика — отдельной "
                                                                          "записью после контроля выборки (узел WLA)")},
                   tags=tg, catalog="E-94", label="S10B/rework-%d" % (k + 1))
    # попытка пятого выполнения
    excavation(g, "F-090", at(D, "14:45"), "WLD-02", "SV-090-4", ["U4"], "участок У4 — четвёртая выборка (без решения)",
               at(D, "14:55"), tags=tg, dur=8, source="TERM-WS-1")
    op_action(g, at(D, "15:00"), "TERM-WS-1", "F-090", "WLD-02", "confirmation",
              OrderedDict([("step", "rework_zone_prep"), ("text", "участок У4 после выборки подготовлен к заварке")]), tags=tg,
              label="SV-090-5.prep")
    op_start(g, at(D, "15:05"), "TERM-WS-1", "F-090", "SV-090-5", "W2", "ST-WC-P1", "WLD-02", "WS-1", tags=tg,
             rework_of="SV-090-4", extra={"program_id": "PS-4", "rework_zones": ["U4"]},
             note="попытка пятого выполнения (четвёртой переделки) на У4: лимит 3 из 3 — заварка блокируется на развилке «какое исправление участка?» (WLG схемы v0.3; Е-46, Е-13)")
    g.step(at(D, "15:15"), "TECH-01", "TECHNOLOGIST", "decision.disposition_set", {"item_id": "F-090", "nc_ref": "NC-90"},
           {"disposition": "rework", "rework_zones": ["U4"]}, mode="stop", tags=tg, catalog="E-94",
           expect="refused:E_REWORK_LIMIT", label="S10B/rework-refused",
           note="отказ — проверка движка полномочий соседей; в схеме v0.2 N2 переделку сверх лимита не предлагает, "
                "а если её выбрать — развилка итога WNG вернёт в решение")
    g.step(at(D, "15:20"), "TECH-01", "TECHNOLOGIST", "decision.disposition_set", {"item_id": "F-090", "nc_ref": "NC-90"},
           {"disposition": "repair", "concession_no": None}, mode="stop", tags=tg, catalog="E-94",
           expect="refused:E_PERMIT_REQUIRED", label="S10B/repair-refused")
    g.step(at(D, "15:30"), "TECH-01", "TECHNOLOGIST", "decision.disposition_set", {"item_id": "F-090", "nc_ref": "NC-90"},
           {"disposition": "scrap", "reason": "лимит переделок участка У4 исчерпан; разрешения на отклонение нет"},
           mode="stop", tags=tg, catalog="E-94", label="S10B/scrap")
    return g


# ----------------------------------------------------------------------------- S14: самостоятельный прогон
# «Общий фактор — один сварщик» [П]. Отдельный мир (seed 20260929): заказ ЗП-0929 на 20 фланцев Ф-301…Ф-320.
# Путь по схеме v0.3 (check_paths.py, S14): сигнал на камере шва → инцидент по общему фактору «сварщик» → групповое
# решение «переделка» через Н → переделка (выборка → контроль выборки → заварка) → разбор причин: версии → проверка →
# объяснение работника и решение комиссии → мера «проверка навыка и инструктаж» → внедрена → окно наблюдения → закрыт.
# Вт 29.09 оба сварщика в смене А (С-03 подменяет); оба источника исправны, журналы в уставке.
S14_ITEMS = ["F-%03d" % n for n in range(301, 321)]
S14_DAY1 = S14_ITEMS[:12]
S14_WINDOW = S14_ITEMS[12:]
S14_WELDS = [  # item, src, welder, start, end, дефект (участок подреза)
    ("F-301", "WS-2", "WLD-02", "08:30", "09:10", None), ("F-302", "WS-1", "WLD-03", "08:30", "09:10", None),
    ("F-303", "WS-1", "WLD-02", "09:20", "10:00", None), ("F-304", "WS-2", "WLD-03", "09:20", "10:00", "U3"),
    ("F-305", "WS-2", "WLD-02", "10:10", "10:50", None), ("F-306", "WS-1", "WLD-03", "10:10", "10:50", "U6"),
    ("F-307", "WS-1", "WLD-02", "11:00", "11:40", None), ("F-308", "WS-2", "WLD-03", "11:00", "11:40", None),
    ("F-309", "WS-2", "WLD-02", "13:00", "13:40", None), ("F-310", "WS-1", "WLD-03", "13:00", "13:40", "U2"),
    ("F-311", "WS-1", "WLD-02", "13:50", "14:30", None), ("F-312", "WS-2", "WLD-03", "13:50", "14:30", None),
]
S14_NC = {"F-304": "NC-141", "F-306": "NC-142", "F-310": "NC-143"}
S14_XR = {"F-301": "11:30", "F-302": "11:50", "F-303": "12:10", "F-304": "13:00", "F-305": "13:20", "F-306": "13:40",
          "F-307": "14:00", "F-308": "14:20", "F-309": "14:40", "F-310": "15:00", "F-311": "15:20", "F-312": "15:40"}
S14_ZT3 = {"F-301": "11:40", "F-302": "12:10", "F-303": "12:20", "F-305": "13:30", "F-307": "14:10", "F-308": "14:30",
           "F-309": "14:50", "F-311": "15:30", "F-312": "15:50"}
S14_EXCL = {"F-302": ("12:05", ["vt/F-302", "xr/F-302/RK-0929-02"]), "F-308": ("14:25", ["vt/F-308", "xr/F-308/RK-0929-08"]),
            "F-312": ("15:45", ["vt/F-312", "xr/F-312/RK-0929-12"])}
S14_REDO = [  # переделка участка по ТП после проверки навыка, Ср 30.09: item, пост, WL1, WL2, prep, start, end
    ("F-304", "WS-1", "13:30", "13:45", "13:48", "13:50", "14:10"),
    ("F-306", "WS-2", "14:20", "14:35", "14:38", "14:40", "15:00"),
    ("F-310", "WS-1", "15:10", "15:25", "15:28", "15:30", "15:50")]
S14_WIN = [  # окно наблюдения после меры, Чт 01.10: С-03, 8 швов по очереди на обоих постах; рентген; ЗТ-3
    ("F-313", "WS-1", "08:30", "09:10", "10:00"), ("F-314", "WS-2", "09:15", "09:55", "10:30"),
    ("F-315", "WS-1", "10:00", "10:40", "11:00"), ("F-316", "WS-2", "10:45", "11:25", "11:40"),
    ("F-317", "WS-1", "11:30", "12:10", "13:00"), ("F-318", "WS-2", "13:00", "13:40", "14:00"),
    ("F-319", "WS-1", "13:45", "14:25", "14:50"), ("F-320", "WS-2", "14:30", "15:10", "15:30")]


def build_s14():
    g = Gen("S14", 20260929)
    tg = ("S14",)
    D0, D1, D3, D4, D5 = (dt.date(2026, 9, 24), dt.date(2026, 9, 25), dt.date(2026, 9, 29), dt.date(2026, 9, 30),
                          dt.date(2026, 10, 1))
    ring = lambda it: "R-%03d" % (int(num(it)) + 100)
    order(g, at(D0, "10:00"), "ORD-0929", "ord-5c2b-0929", "1c-in-000340", 20, "2026-10-16", "F-301", "F-320", tags=tg)
    for k, lot in enumerate(("LOT-ZF-202", "LOT-R-118", "LOT-W-88")):
        lot_received(g, at(D0, "10:10") + S(30 * k), lot, tags=tg)
    t = iqc(g, at(D0, "10:30"), "LOT-ZF-202", S14_ITEMS, True, True, at(D0, "10:30"), tags=tg)
    t = iqc(g, t + M(5), "LOT-R-118", [ring(x) for x in S14_ITEMS], True, True, t, tags=tg)
    iqc(g, t + M(5), "LOT-W-88", None, False, False, t, tags=tg)
    # Пт 25.09 – Пн 28.09: запуск, мехобработка, перемаркировка, КИМ, ЗТ-2, передача в сварочный цех
    plan = machining_plan(S14_ITEMS, at(D1, "08:40"))
    reg = reg_times(plan, S14_ITEMS, D1)
    tool = [20]
    for it in S14_ITEMS:
        register(g, reg[it], it, "ORD-0929", "LOT-ZF-202", tags=tg)
        machining(g, it, plan[it][0], s1_min(*plan[it]), tags=tg, tool_state=tool)
    q = cmm_queue([(it, plan[it][1]) for it in S14_ITEMS])
    in_wc = {}
    for it in S14_ITEMS:
        cmm(g, it, q[it][1], tags=tg)
        z = zt_time(q[it][1] + M(5))
        gate(g, z, "QC-01", "ZT-2", item=it, tags=tg, basis=["kt2/" + it, "cmm/" + it], label="ZT-2/" + it)
        r = transfer_run(z)
        move(g, r, it, "MC", "WC", "MST-01", "MST-02", tags=tg, label="mv/%s/MC-WC" % it)
        in_wc[it] = r + M(10)
    # ---- Вт 29.09: сварка 12 фланцев на обоих постах
    issue(g, at(D3, "08:00"), "LOT-W-88", 1, "ST-WC-P1", tags=tg, label="issue/LOT-W-88@0929")
    t_ring = at(D3, "08:05")
    assert all(in_wc[x] + M(2) <= t_ring for x in S14_DAY1), "S14: кольца раньше приёмки фланцев"
    issue(g, t_ring, "LOT-R-118", 12, "ST-WC-P1", items=list(S14_DAY1), components=[ring(x) for x in S14_DAY1], tags=tg,
          label="issue/rings/LOT-R-118@0929")
    undercut = lambda zone, conf: [OrderedDict([("defect_type", "undercut"), ("zone", zone), ("side", "face"),
                                                ("severity", "major"), ("depth_estimate_mm", 0.6), ("measurable", False),
                                                ("confidence", conf)])]
    for (it, src, welder, s_, e_, dz) in S14_WELDS:
        st, en = at(D3, s_), at(D3, e_)
        w = dict(item=it, src=src, welder=welder, start=st, end=en, arc=(st + M(2), en - M(3)), ring=ring(it),
                 ring_lot="LOT-R-118", wire="LOT-W-88", prep=st - M(8), run="SV-%s-1" % num(it), profile=None)
        assert w["prep"] >= in_wc[it], it
        ov = None
        if dz:
            def ov(gg, t0, it=it, dz=dz, run=w["run"]):
                for k, (cam, q_, c_) in enumerate((("CAM-WS-1", 0.91, 0.84), ("CAM-WS-2", 0.89, 0.80))):
                    camera(gg, t0 + S(10 * k), cam, it, "CP-WELD", "after_operation", SEGS, result="defect_found",
                           q=q_, conf=c_, defects=undercut(dz, c_), tags=tg, label="kt3/%s/%s" % (run, cam),
                           extra={"comparison_to_before": "new_zone_created_by_operation"})
        weld(g, w, tags=tg, cam_override=ov)
    # сигналы: подтверждение ОТК-1 (Е-90) и изоляция. НС-141 — решение на своей ЗТ (общего фактора ещё нет);
    # с НС-142 общий фактор — сварщик С-03 (2 из 2, аппараты разные) → инцидент RS-14 (реакция Е-110, в потоке нет)
    for it, hm_c, hm_i, mode in (("F-304", "10:12", "10:15", "auto"), ("F-306", "11:02", "11:05", "auto"),
                                 ("F-310", "13:52", "13:55", "stop")):
        run = "SV-%s-1" % num(it)
        dz = [x[5] for x in S14_WELDS if x[0] == it][0]
        g.step(at(D3, hm_c), "QC-01", "QC", "decision.signal_confirmed",
               {"item_id": it, "signal_basis_labels": ["kt3/%s/CAM-WS-1" % run, "kt3/%s/CAM-WS-2" % run], "nc_ref": S14_NC[it]},
               {"defect_type": "undercut", "zone": dz, "severity": "major",
                "reason": "подрез на участке %s подтверждён по двум ракурсам КТ-3 и осмотру с шаблоном" % dz.replace("U", "У"),
                "incident_basis": None if it == "F-304" else "RS-14 (общий фактор — сварщик С-03)"},
               mode=mode, tags=tg, catalog="E-90", label="S14/confirm-" + S14_NC[it])
        g.step(at(D3, hm_i), "QC-01", "QC", "decision.item_isolated", {"item_id": it, "nc_ref": S14_NC[it]},
               {"location": "ST-ISO-WC", "reason": "подтверждённое несоответствие %s, ждёт решения" % S14_NC[it]},
               tags=tg, catalog="E-93", label="S14/isolate-" + it)
        g.ev(at(D3, hm_i) + S(40), "TERM-WC", "movement.isolator_confirmed", item=it,
             payload=OrderedDict([("item_id", it), ("to_station", "ST-ISO-WC"), ("physically_moved", True),
                                  ("confirmed_by", "MST-02")]), tags=tg, label="S14/isolator-" + it, persona="MST-02",
             after="S14/isolate-" + it)
    # инцидент RS-14: подсказки «сначала аппараты» (журналы — в уставке), затем швы С-03 в круге — осмотр с шаблоном
    g.step(at(D3, "11:08"), "QC-01", "QC", "decision.recheck_requested", {"items": ["F-302"], "risk_scope_ref": "RS-14"},
           {"method": "visual_template", "reason": "в круге RS-14 (общий фактор — сварщик С-03): осмотреть шов с шаблоном",
            "suggested_by": "system", "suggestion_order": ["журналы ИС-1 и ИС-2 — в уставке (готово)",
                                                           "сравнение с С-02 на тех же постах (готово)",
                                                           "осмотр с шаблоном и рентген швов С-03 в круге",
                                                           "контрольный образец С-03 под наблюдением ГС-01"]},
           tags=tg, catalog="E-92", label="S14/recheck-F-302")
    g.step(at(D3, "14:00"), "QC-01", "QC", "decision.recheck_requested", {"items": ["F-308", "F-312"], "risk_scope_ref": "RS-14"},
           {"method": "visual_template", "reason": "в круге RS-14: осмотреть швы С-03 с шаблоном", "suggested_by": "system"},
           tags=tg, catalog="E-92", label="S14/recheck-F-308-F-312")
    for it, hm in (("F-302", "11:20"), ("F-308", "14:10"), ("F-312", "14:45")):
        g.ev(at(D3, hm), "TERM-WC", "inspection.result", item=it,
             payload=OrderedDict([("checkpoint_id", "CP-WELD"), ("phase", "after_operation"), ("method", "visual"),
                                  ("tool", "шаблон сварщика УШС-3 [ПП]"), ("operator_id", "QC-01"),
                                  ("inspection_result", "no_defect_found"), ("zones_inspected", SEGS), ("defects", [])]),
             tags=tg, label="vt/%s" % it, persona="QC-01")
    for k, it in enumerate(S14_DAY1):
        dz = [x[5] for x in S14_WELDS if x[0] == it][0]
        dfx = [OrderedDict([("defect_type", "undercut"), ("zone", dz), ("side", "face"), ("severity", "major")])] if dz else None
        xray(g, at(D3, S14_XR[it]), it, "RK-0929-%02d" % (k + 1), defects=dfx, tags=tg)
    for it, (hm, basis) in S14_EXCL.items():
        g.step(at(D3, hm), "TECH-01", "TECHNOLOGIST", "risk_scope.item_assessed", {"risk_scope_ref": "RS-14", "item_id": it},
               {"result": "excluded", "basis_labels": basis, "holds_released": "containment.hold_released (Е-116) этим шагом"},
               tags=tg, catalog="E-113", label="S14/exclude-" + it)
    for it, hm in S14_ZT3.items():
        k = S14_DAY1.index(it) + 1
        basis = ["kt3/SV-%s-1/CAM-WS-1" % num(it), "xr/%s/RK-0929-%02d" % (it, k)]
        if it in S14_EXCL:
            basis.append("vt/" + it)
        gate(g, at(D3, hm), "QC-01", "ZT-3", item=it, tags=tg, basis=basis, label="ZT-3/" + it)
    BAD = ["F-304", "F-306", "F-310"]
    for it in BAD:
        g.step(at(D3, "16:00"), "TECH-01", "COMMISSION", "risk_scope.item_assessed", {"risk_scope_ref": "RS-14", "item_id": it},
               {"result": "confirmed", "reason": "подрез подтверждён камерой, осмотром и рентгеном"},
               tags=tg, catalog="E-113", signers=["TECH-01", "QC-HEAD-01", "CHW-01"], label="S14/assess/" + it)
    g.step(at(D3, "16:00"), "TECH-01", "COMMISSION", "decision.signal_confirmed",
           {"items": BAD, "nc_ref": "NC-14G", "nc_kind": "group", "risk_scope_ref": "RS-14"},
           {"reason": "подрез, общий фактор — сварщик С-03 (3 из 3); НС-141…НС-143 сведены в групповое"},
           mode="stop", tags=tg, catalog="E-90", signers=["TECH-01", "QC-HEAD-01", "CHW-01"], label="S14/NC-14G")
    g.step(at(D3, "16:00"), "TECH-01", "COMMISSION", "decision.disposition_set", {"items": BAD, "nc_ref": "NC-14G"},
           {"disposition": "rework", "per_item_zones": {"F-304": ["U3"], "F-306": ["U6"], "F-310": ["U2"]}, "rework_no": 1,
            "reason": "переделка участка шва по ТП (первая — без согласия главного сварщика); варить — после проверки навыка",
            "item_messages": [{"item_id": x, "block": "BW", "caught_by": "BWD", "route": "BWG → BW (WSG → WL1 → WL2 → WLG → WL3)"}
                              for x in BAD]},
           mode="stop", tags=tg, catalog="E-94", signers=["TECH-01", "QC-HEAD-01", "CHW-01"], label="S14/disposition-rework")
    # v0.3.1: круг RS-14 (узел I2): швы С-03 за смену — придержаны; новые швы С-03 входят в круг по окончании сварки
    g.engine_holds += [dict(at=at(D3, "11:02"), targets=["F-302", "F-304", "F-306"], basis="risk_scope:RS-14", node="I2"),
                       dict(at=at(D3, "11:40"), targets=["F-308"], basis="risk_scope:RS-14", node="I2"),
                       dict(at=at(D3, "13:40"), targets=["F-310"], basis="risk_scope:RS-14", node="I2"),
                       dict(at=at(D3, "14:30"), targets=["F-312"], basis="risk_scope:RS-14", node="I2")]
    g.step(at(D3, "16:05"), "TECH-01", "TECHNOLOGIST", "risk_scope.closed", {"risk_scope_ref": "RS-14"},
           {"was": 6, "confirmed": 3, "excluded": 3, "note": "Изделия: решено · Расследование: открыто"},
           tags=tg, catalog="E-115", label="S14/close")
    # ---- Ср 30.09: проверка версии контрольными образцами; объяснение; комиссия; мера
    g.power.append(("WS-1", at(D4, "08:25"), at(D4, "09:06"), "end_of_work", None))
    g.power.append(("WS-1", at(D4, "12:25"), at(D4, "12:46"), "end_of_work", None))
    samples = [("CS-141", "SV-CS141-1", "08:30", "08:40", "08:45", "как обычно, без подсказок", True,
                "подрез повторился; угол наклона горелки ≈ 45° при 70–80° по карте режимов [ПП]"),
               ("CS-142", "SV-CS142-1", "08:50", "09:00", "09:05", "угол горелки — по карте (≈ 75°)", False, "подреза нет"),
               ("CS-143", "SV-CS143-1", "12:30", "12:40", "12:45", "внеочередная проверка навыка после инструктажа", False,
                "подреза нет; проверка навыка сдана")]
    for (cs, run, s_, e_, tv, technique, defect, text) in samples:
        g.ev(at(D4, s_), "TERM-WS-1", "operation.started", item=cs,
             payload=OrderedDict([("operation_run_id", run), ("operation_code", "W2"), ("station_id", "ST-WC-P1"),
                                  ("operator_id", "WLD-03"), ("equipment_id", "WS-1"), ("program_id", "PS-4"),
                                  ("purpose", "control_sample"), ("item_type_id", "CONTROL-SAMPLE"),
                                  ("observer_id", "CHW-01"), ("technique_note", technique)]),
             tags=tg, corr=run, label=run + ".start", persona="WLD-03")
        g.ev(at(D4, e_), "TERM-WS-1", "operation.finished", item=cs,
             payload=OrderedDict([("operation_run_id", run), ("result", "completed"), ("operator_id", "WLD-03")]),
             tags=tg, corr=run, label=run + ".finish", persona="WLD-03")
        g.welds.append(dict(item=cs, src="WS-1", welder="WLD-03", start=at(D4, s_), end=at(D4, e_),
                            arc=(at(D4, s_) + M(1), at(D4, e_) - M(1)), run=run, profile=None, cs=True))
        g.ev(at(D4, tv), "TERM-WS-1", "inspection.result", item=cs,
             payload=OrderedDict([("checkpoint_id", "CP-WELD"), ("phase", "after_operation"), ("method", "visual"),
                                  ("operator_id", "CHW-01"), ("purpose", "control_sample"),
                                  ("inspection_result", "defect_found" if defect else "no_defect_found"),
                                  ("defects", [OrderedDict([("defect_type", "undercut"), ("zone", "SAMPLE"),
                                                            ("severity", "major"), ("description", text)])] if defect else []),
                                  ("observation", text)]),
             tags=tg, label="%s/visual" % cs, persona="CHW-01")
    NCS = ["NC-141", "NC-142", "NC-143", "NC-14G"]
    g.step(at(D4, "09:10"), "TECH-01", "COMMISSION", "cause.confirmed", {"nc_refs": NCS},
           {"category": "operator_procedure", "operator_error": {"confirmed": True, "person": "WLD-03"},
            "verified_by": "контрольные образцы CS-141, CS-142"},
           tags=tg, catalog="П-04", signers=["TECH-01", "QC-HEAD-01", "CHW-01"], expect="refused:E_EXPLANATION_REQUIRED",
           optional=True, label="S14/cause-before-explanation-refused",
           note="проверка правила: «ошибку исполнителя» нельзя записать без письменного объяснения работника — ожидается отказ [П]")
    g.step(at(D4, "09:30"), "WLD-03", "WELDER", "cause.explanation_recorded", {"nc_refs": NCS, "person": "WLD-03"},
           {"form": "письменное объяснение", "received_by": "QC-HEAD-01", "hypothesis": "operator_procedure",
            "text": "с понедельника на постах новые горелки с другим держателем; держал горелку ниже, чем по карте, — "
                    "так удобнее; угол не проверял [П]"},
           tags=tg, catalog="П-09 [П]", label="S14/explanation")
    g.step(at(D4, "11:00"), "TECH-01", "COMMISSION", "cause.confirmed", {"nc_refs": NCS},
           {"category": "operator_procedure",
            "text": "отклонение от техники выполнения: угол наклона горелки ниже карты режимов (≈ 45° при 70–80°) [ПП]",
            "verified_by": "контрольные образцы: CS-141 с прежним углом — подрез, CS-142 по карте — чисто; журналы ИС-1 и ИС-2 "
                           "в уставке; у С-02 на тех же постах 6 из 6 швов чистые",
            "evidence_labels": ["CS-141/visual", "CS-142/visual"],
            "explanation_ref_label": "S14/explanation",
            "hypotheses_final": {"equipment": "weak", "material": "weak", "operator_procedure": "confirmed"},
            "contributing_factors": ["новая горелка с другим держателем на постах 1 и 2 с Пн 28.09"],
            "operator_error": {"confirmed": True, "person": "WLD-03",
                               "basis": "решение комиссии после разбора и письменного объяснения работника"},
            "not_decided_here": "дисциплинарные вопросы — вне системы: решает работодатель по ТК РФ; система их не готовит"},
           mode="stop", tags=tg, catalog="П-04", signers=["TECH-01", "QC-HEAD-01", "CHW-01"], label="S14/cause")
    g.step(at(D4, "11:20"), "QC-HEAD-01", "QC_HEAD", "capa.action_assigned", {"nc_refs": NCS},
           {"actions": [
               {"kind": "correction", "text": "внеочередная проверка навыка С-03: контрольный образец под наблюдением ГС-01",
                "owner": "CHW-01", "due": "2026-09-30"},
               {"kind": "corrective", "text": "внеплановый инструктаж по технике сварки кольцевого шва: угол горелки",
                "owner": "CHW-01", "due": "2026-09-30"},
               {"kind": "corrective", "text": "проверить держатель новой горелки на постах 1 и 2, при необходимости заменить",
                "owner": "MST-02", "due": "2026-10-02"}],
            "effectiveness_plan": {"metric": "подрезы в швах С-03", "baseline": "3 из 6 швов С-03 за Вт 29.09",
                                   "window": "первые 8 новых швов С-03 после меры (точка чистоты)", "success": "0 подрезов",
                                   "on_fail": "мера снова открыта, разбор заново"},
            "not_a_penalty": True},
           mode="stop", tags=tg, catalog="П-05", signers=["QC-HEAD-01", "CHW-01"], label="S14/capa")
    g.step(at(D4, "13:00"), "CHW-01", "CHIEF_WELDER", "capa.action_implemented", {"nc_refs": NCS},
           {"done": ["инструктаж проведён (подписи С-03 и ГС-01)", "внеочередная проверка навыка сдана: образец CS-143 чистый",
                     "держатель горелки на посту 1 заменён"],
            "evidence_labels": ["CS-143/visual"], "window_opened": "первые 8 новых швов С-03"},
           tags=tg, catalog="П-06", label="S14/capa-done")
    # ---- Ср 30.09, после проверки навыка: переделка трёх участков (выборка → контроль выборки → заварка)
    issue(g, at(D4, "13:20"), "LOT-W-88", 1, "ST-WC-P1", tags=tg, label="issue/LOT-W-88@0930")
    for it, src, wl1, wl2, prep, s_, e_ in S14_REDO:
        dz = [x[5] for x in S14_WELDS if x[0] == it][0]
        excavation(g, it, at(D4, wl1), "WLD-03", "SV-%s-1" % num(it), [dz], "участок %s по снимку и заключению" % dz,
                   at(D4, wl2), tags=tg, dur=10, source="TERM-WS-1" if src == "WS-1" else "TERM-WS-2")
        w = dict(item=it, src=src, welder="WLD-03", start=at(D4, s_), end=at(D4, e_),
                 arc=(at(D4, s_) + M(1), at(D4, e_) - M(1)), wire="LOT-W-88", prep=at(D4, prep),
                 prep_step="rework_zone_prep", prep_text="участок после выборки подготовлен к заварке",
                 run="SV-%s-2" % num(it), rework_of="SV-%s-1" % num(it), rework_zones=[dz], profile=None)
        weld(g, w, tags=tg)
    # ---- Чт 01.10: рентген переделанных участков, повторная ЗТ-3; окно наблюдения — 8 новых швов С-03
    for k, (it, hm_x, hm_z) in enumerate((("F-304", "08:10", "08:20"), ("F-306", "08:30", "08:40"),
                                          ("F-310", "08:50", "09:00"))):
        dz = [x[5] for x in S14_WELDS if x[0] == it][0]
        xray(g, at(D5, hm_x), it, "RK-1001-%02d" % (k + 1), zones=[dz], tags=tg)
        g.ev(at(D5, hm_z) - M(5), "TERM-WC", "item.presented", item=it,
             payload=OrderedDict([("item_id", it), ("presentation_no", 2), ("presented_to", ["QC"]),
                                  ("presented_by", "MST-02"), ("gate", "ZT-3")]),
             tags=tg, label="present/%s/ZT-3#2" % it, persona="MST-02")
        g.step(at(D5, hm_z), "QC-01", "QC", "decision.disposition_verified", {"item_id": it, "nc_ref": "NC-14G"},
               {"result": "passed", "methods": ["camera", "xray"], "rework_run": "SV-%s-2" % num(it)},
               tags=tg, catalog="E-96", label="S14/verified/" + it)
        gate(g, at(D5, hm_z) + S(30), "QC-01", "ZT-3", item=it, tags=tg, presentation=2,
             basis=["kt3/SV-%s-2/CAM-WS-1" % num(it), "xr/%s/RK-1001-%02d" % (it, k + 1)], label="ZT-3#2/" + it)
    issue(g, at(D5, "08:00"), "LOT-W-88", 1, "ST-WC-P1", tags=tg, label="issue/LOT-W-88@1001")
    assert all(in_wc[x] + M(2) <= at(D5, "08:05") for x in S14_WINDOW)
    issue(g, at(D5, "08:05"), "LOT-R-118", 8, "ST-WC-P1", items=list(S14_WINDOW), components=[ring(x) for x in S14_WINDOW],
          tags=tg, label="issue/rings/LOT-R-118@1001")
    for k, (it, src, s_, e_, hm_x) in enumerate(S14_WIN):
        st, en = at(D5, s_), at(D5, e_)
        prep = st - M(3) if k else st - M(8)
        w = dict(item=it, src=src, welder="WLD-03", start=st, end=en, arc=(st + M(2), en - M(3)), ring=ring(it),
                 ring_lot="LOT-R-118", wire="LOT-W-88", prep=prep, run="SV-%s-1" % num(it), profile=None)
        weld(g, w, tags=tg)
        no = "RK-1001-%02d" % (k + 4)
        xray(g, at(D5, hm_x), it, no, tags=tg)
        gate(g, at(D5, hm_x) + M(10), "QC-01", "ZT-3", item=it, tags=tg,
             basis=["kt3/SV-%s-1/CAM-WS-1" % num(it), "xr/%s/%s" % (it, no)], label="ZT-3/" + it)
    g.step(at(D5, "16:00"), "QC-HEAD-01", "QC_HEAD", "capa.effectiveness_confirmed", {"nc_refs": NCS},
           {"window": "первые 8 новых швов С-03 после меры", "result": "8 из 8 без подреза (камера КТ-3 и рентген)",
            "evidence_labels": ["kt3/SV-%s-1/CAM-WS-1" % num(x) for x in S14_WINDOW]},
           tags=tg, catalog="П-07", label="S14/effective")
    g.step(at(D5, "16:05"), "QC-HEAD-01", "QC_HEAD", "cause.investigation_closed", {"nc_refs": NCS},
           {"cause": "отклонение от техники выполнения (угол горелки); сопутствующее — новая горелка",
            "measures": ["проверка навыка", "инструктаж", "держатель горелки"], "window_result": "результативна",
            "status": "Расследование: закрыто"},
           tags=tg, catalog="П-10 [П]", label="S14/closed")
    return g


# ----------------------------------------------------------------------------- нестандартные случаи S18–S22 (отдельные миры) [П]
# Каждый случай — свой короткий прогон со своим seed и своей нумерацией source_seq (S18–S21) или продолжение прогона
# S10B (S22, подаётся после него). Главную историю не трогают. Люди и партии — из набора §2 сценариев; чем мир
# отличается от главной истории — в карточке сценария (scenarios-flange.md §14) и в определении (precondition).
CN_ID, CN_DISPLAY = "CN-FL-100-07", "ИИ-ФЛ-100-07"   # S19: извещение об изменении КД [П]
D2_REV_V = {"D2": (252.00, 0.0, 0.05)}             # S19: ревизия «В» — отверстие D2 252,00 под уплотнение новой конструкции [П]


def manual_kt2(g, item, t, tags, why=None):
    """КТ-2 без камеры: для ревизии «В» нет допущенного контура ИИ (AI_ADMISSION) — осмотр контролёром (узел M2H)."""
    return g.ev(t, "TERM-MC", "inspection.result", item=item,
                payload=OrderedDict([("checkpoint_id", "CP-MO"), ("phase", "after_operation"), ("method", "visual"),
                                     ("operator_id", "QC-01"), ("kd_revision", "V"),
                                     ("ai_admission", OrderedDict([("status", "not_admitted"), ("admitted_for_kd_revision", "B"),
                                                                   ("mode", "manual")])),
                                     ("zones_inspected", ["EDGE", "FACE", "BORE"]), ("inspection_result", "no_defect_found"),
                                     ("defects", []),
                                     ("observation", why or "ручной контроль: для ревизии «В» нет допущенного анализатора — "
                                                            "камера КТ-2 в основание не идёт")]),
                tags=tags, label="kt2-manual/" + item, persona="QC-01")


def scanned(g, t, src, item, persona, purpose, read_result, tags, label, expected=None, decoded=None, note=None, **extra):
    """Е-126 [П]: скан кода изделия или компонента на посту или при выдаче. Вывод (Е-13) делает движок — не поток."""
    p = OrderedDict([("purpose", purpose), ("operator_id", persona)])
    if expected:
        p["expected_item_id"] = expected
    p["decoded"] = decoded
    p["read_result"] = read_result
    p.update(extra)
    return g.ev(t, src, "item.scanned", item=item, payload=p, tags=tags, label=label, persona=persona,
                hints=[x for x in (item, expected, extra.get("for_item")) if x], note=note)


def ns_lots(g, t, lots, pieces, tg, when=None):
    """Партии мира: Е-03 и входной контроль (компактно). pieces — экземпляры по партии; when — своё время по партии."""
    when = when or {}
    tt = t
    for k, lot in enumerate(lots):
        t0 = when.get(lot, t + S(30 * k))
        lot_received(g, t0, lot, tags=tg)
    for lot in lots:
        t0 = when.get(lot)
        it = LOTS[lot][0]
        pcs = pieces.get(lot)
        start = (t0 + M(5)) if t0 else tt + M(3)
        end = iqc(g, start, lot, pcs, bool(pcs) and it in ("FL-100-FLANGE", "FL-100-RING"), bool(pcs), start, tags=tg)
        if not t0:
            tt = end + M(3)
    return tt


def ns_front(g, items, reg_at, mach_at, tg, order_id="ORD-0917", blank_lot="LOT-ZF-201", send=True, kd="B", tp=3, prog="3",
             kt2_manual=False, cmm_nominals=None):
    """Запуск, мехобработка (перемаркировка, КТ-2), КИМ, ЗТ-2, передача в сварочный цех. Возвращает {изделие: приём СЦ}."""
    out = {}
    t = mach_at
    for k, it in enumerate(items):
        register(g, reg_at + M(k), it, order_id, blank_lot, tags=tg, kd=kd, tp=tp)
    for it in items:
        end = machining(g, it, t, 50, tags=tg, prog_rev=prog, kt2_manual=kt2_manual)
        tc = end + M(25)
        cmm(g, it, tc, tags=tg, nominals=cmm_nominals)
        z = tc + M(5)
        k2 = ("kt2-manual/" if kt2_manual else "kt2/") + it
        gate(g, z, "QC-01", "ZT-2", item=it, tags=tg, basis=[k2, "cmm/" + it], label="ZT-2/" + it,
             extra={"kd_revision": kd} if kd != "B" else None)
        if send:
            r = transfer_run(z)
            move(g, r, it, "MC", "WC", "MST-01", "MST-02", tags=tg, label="mv/%s/MC-WC" % it)
            out[it] = r + M(10)
        t = end + M(5)
    return out


def ns_weld(g, it, src, welder, s_, e_, ring, ring_lot, tg, prep=None, link_method="scan", run=None):
    w = dict(item=it, src=src, welder=welder, start=s_, end=e_, arc=(s_ + M(2), e_ - M(3)), ring=ring, ring_lot=ring_lot,
             wire="LOT-W-88", prep=prep, run=run or "SV-%s-1" % num(it), profile=None, link_method=link_method)
    return weld(g, w, tags=tg)


def build_s18():
    """S18 «Истёк допуск на дату операции»: а) удостоверение С-03 истекло 02.10 → старт сварки отклонён, система предлагает
    С-02; б) поверка ключа КЛ-1 истекла 02.10 → моменты затяжки «недостоверно», ЗТ-4 на них не подписывают, ОТК-2
    проверяет моменты поверенным ключом КЛ-2. Мир: Ф-042 — взамен списанного Ф-021 (запасная заготовка ЗФ-201)."""
    g = Gen("S18", 20261018)
    tg = ("S18",)
    D0, D1 = dt.date(2026, 10, 2), dt.date(2026, 10, 5)
    it = "F-042"
    ns_lots(g, at(D0, "07:30"), ["LOT-ZF-201", "LOT-R-119", "LOT-W-88", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", "LOT-V-601"],
            {"LOT-ZF-201": [it], "LOT-R-119": ["R-904"], "LOT-C-301": ["C-021"]}, tg)
    in_wc = ns_front(g, [it], at(D0, "09:30"), at(D0, "09:40"), tg)
    # ---- Пн 05.10: а) допуск сварщика
    t_in = in_wc[it]
    assert t_in <= at(D1, "08:00"), t_in
    issue(g, at(D1, "08:02"), "LOT-W-88", 1, "ST-WC-P2", tags=tg, label="issue/LOT-W-88@1005")
    issue(g, at(D1, "08:05"), "LOT-R-119", 1, "ST-WC-P2", items=[it], components=["R-904"], tags=tg, label="issue/ring/" + it)
    g.ev(at(D1, "08:10"), "TERM-WS-2", "assembly.component_linked", item=it,
         payload=OrderedDict([("assembly_item_id", it), ("component_id", "R-904"), ("component_type_id", "FL-100-RING"),
                              ("lot_id", "LOT-R-119"), ("position", 2), ("link_id", "W-1"), ("binding_method", "scan"),
                              ("operator_id", "WLD-03")]),
         tags=tg, label="SV-042-1.link", persona="WLD-03", hints=[it, "R-904"])
    op_action(g, at(D1, "08:12"), "TERM-WS-2", it, "WLD-03", "confirmation",
              OrderedDict([("step", "edge_prep"), ("text", "кромки подготовлены")]), tags=tg, label="S18/edge-prep")
    g.ev(at(D1, "08:20"), "TERM-WS-2", "operation.started", item=it,
         payload=OrderedDict([("operation_run_id", "SV-042-1"), ("operation_code", "W2"), ("station_id", "ST-WC-P2"),
                              ("operator_id", "WLD-03"), ("equipment_id", "WS-2"), ("program_id", "PS-4"),
                              ("wire_lot_id", "LOT-W-88"), ("gas_lot", "AR-2026-09"), ("welder_qualification_id", "Q-WLD-03")]),
         tags=tg, corr="SV-042-1", label="S18/start-refused", persona="WLD-03",
         note="Е-13 E_QUALIFICATION_EXPIRED: удостоверение С-03 (НАКС-УСЛ-0003) действовало до 02.10.2026 — сварка не начата; "
              "запись в журнал критических действий; система предлагает С-02 (допуск до 01.03.2027)")
    g.step(at(D1, "08:25"), "MST-02", "MASTER", "post.assignment_changed",
           {"station_id": "ST-WC-P2", "item_id": it, "operation_code": "W2"},
           {"from": "WLD-03", "to": "WLD-02", "reason": "у С-03 удостоверение истекло 02.10.2026; система предложила С-02 — "
                                                       "его удостоверение действует до 01.03.2027, он в смене А и свободен",
            "suggested_by": "system", "basis_labels": ["S18/start-refused"],
            "what_next_for_WLD-03": "направлен на продление аттестации; до продления — работы без сварки"},
           tags=tg, catalog="[П] назначение исполнителя", label="S18/reassign")
    ns_weld(g, it, "WS-2", "WLD-02", at(D1, "08:40"), at(D1, "09:20"), None, None, tg)
    xray(g, at(D1, "10:30"), it, "RK-1005-01", tags=tg)
    gate(g, at(D1, "10:50"), "QC-01", "ZT-3", item=it, tags=tg, basis=["kt3/SV-042-1/CAM-WS-1", "xr/%s/RK-1005-01" % it],
         label="ZT-3/" + it)
    move(g, at(D1, "11:00"), it, "WC", "AC", "MST-02", "MST-03", tags=tg, label="mv/%s/WC-AC" % it)
    # ---- б) поверка ключа КЛ-1
    kit_issue(g, at(D1, "11:15"), it, "C-021", "LOT-SL-401", "LOT-FS-501", "LOT-C-301", tags=tg)
    assembly(g, it, at(D1, "11:20"), "C-021", "LOT-C-301", "LOT-SL-401", "LOT-FS-501", tags=tg, zt4=False)
    gate(g, at(D1, "12:25"), "QC-02", "ZT-4", item=it, tags=tg,
         basis=["kt4/" + it, "torque/" + it, "torque/%s/J-2" % it], expect="refused:E_INSTRUMENT_NOT_VERIFIED",
         mode="stop", label="S18/ZT-4-refused",
         note="проверка правила: моменты записаны ключом КЛ-1, поверка которого кончилась 02.10.2026 — «недостоверно» (Е-128); "
              "на таком основании ЗТ-4 не подписывается")
    gate(g, at(D1, "12:28"), "QC-02", "ZT-4", item=it, tags=tg, decision="recheck", basis=["kt4/" + it],
         extra={"method": "контроль момента поверенным ключом КЛ-2 (TW-2): J-1 — 12 болтов, J-2 — клапан",
                "reason": "записи КЛ-1 — «недостоверно: средство не поверено на дату»; нужен повторный замер поверенным средством",
                "invalidated_basis": ["torque/" + it, "torque/%s/J-2" % it]},
         label="S18/ZT-4-recheck")
    r = rnd("tw2|" + it)
    bolts = [OrderedDict([("position", "BOLT-%d" % b), ("torque_nm", round(r.uniform(7.8, 8.3), 1)), ("status", "ok")])
             for b in range(1, 13)]
    g.ev(at(D1, "13:05"), "TW-2", "machine.parameters",
         payload=OrderedDict([("equipment_id", "TW-2"), ("joint_id", "J-1"), ("item_hint", it), ("purpose", "control_check"),
                              ("method", "breakaway_check"), ("method_note", "проверка момента дотяжкой до страгивания [ПП]"),
                              ("operator_id", "QC-02"),
                              ("setpoint", {"torque_nm": 8.0, "tolerance_pct": 10}), ("bolts", bolts)]),
         tags=tg, hints=[it], label="torque-check/" + it)
    g.ev(at(D1, "13:08"), "TW-2", "machine.parameters",
         payload=OrderedDict([("equipment_id", "TW-2"), ("joint_id", "J-2"), ("item_hint", it), ("purpose", "control_check"),
                              ("operator_id", "QC-02"), ("setpoint", {"torque_nm": 12.0, "tolerance_pct": 10}),
                              ("result", OrderedDict([("torque_nm", round(r.uniform(11.7, 12.3), 1)), ("status", "ok")]))]),
         tags=tg, hints=[it], label="torque-check/%s/J-2" % it)
    gate(g, at(D1, "13:15"), "QC-02", "ZT-4", item=it, tags=tg,
         basis=["kt4/" + it, "torque-check/" + it, "torque-check/%s/J-2" % it],
         extra={"invalidated_basis": ["torque/" + it, "torque/%s/J-2" % it],
                "note": "записи КЛ-1 — «недостоверно» (поверка до 02.10.2026); основание — контрольный ключ КЛ-2, поверен до 01.04.2027"},
         label="ZT-4/" + it)
    return g


def s19_rework(g, it, t0, tg):
    """S19: доработка Ф-039 до ревизии «В» — расточка D2 до 252,00 на ЧПУ-1 (новое выполнение со ссылкой на прежнее)."""
    run = "MO-%s-2" % num(it)
    op_start(g, t0, "TERM-MC-1", it, run, "M1", "ST-MC-CNC", "OP-CNC-11", "CNC-1", tags=tg, rework_of="MO-%s-1" % num(it),
             extra={"purpose": "rework_to_new_revision", "change_notice": CN_ID, "what": "расточка D2 с 250,00 до 252,00",
                    "program_id": "UP-FL-100-01", "program_revision": "4"})
    g.ev(t0 + S(30), "CNC-1", "machine.state",
         payload=OrderedDict([("equipment_id", "CNC-1"), ("state", "running"), ("mode", "auto"), ("program_id", "UP-FL-100-01"),
                              ("program_revision", "4")]), tags=tg, hints=[it], label="cnc/%s/running#rev-V" % it)
    g.ev(t0 + M(25), "CNC-1", "machine.state",
         payload=OrderedDict([("equipment_id", "CNC-1"), ("state", "idle"), ("reason", "cycle_complete")]),
         tags=tg, hints=[it], label="cnc/%s/idle#rev-V" % it)
    op_finish(g, t0 + M(26), "TERM-MC-1", it, run, "OP-CNC-11", tags=tg)
    return t0 + M(26)


def build_s19():
    """S19 «Изменили КД посреди партии»: извещение ИИ-ФЛ-100-07, ревизия «Б» → «В» (D2 250 → 252 [П]). Новые изделия — по «В»;
    задел придержан и решается по каждому: Ф-039 (мехобработан) — доработать до «В»; Ф-037 (сварен) — оставить по «Б» по
    разрешению на отклонение (одно уплотнение старой конструкции на складе); Ф-038 (сварен) — списать. Ф-040 запущен по «В»;
    КТ-2 для «В» — ручной контроль, пока нет допуска ИИ. Мир: остаток заказа ЗП-0917."""
    g = Gen("S19", 20261019)
    tg = ("S19",)
    D0, D1 = dt.date(2026, 9, 25), dt.date(2026, 9, 28)
    imports(g, T("2026-09-07 09:00"), tags=tg)
    order(g, T("2026-09-14 11:00"), "ORD-0917", "ord-5c2b-0917", "1c-in-000314", 40, "2026-10-09", "F-001", "F-040", tags=tg)
    ns_lots(g, at(D0, "07:30"), ["LOT-ZF-201", "LOT-R-119", "LOT-W-88"],
            {"LOT-ZF-201": ["F-037", "F-038", "F-039", "F-040"], "LOT-R-119": ["R-902", "R-903"]}, tg)
    in_wc = ns_front(g, ["F-037", "F-038"], at(D0, "08:30"), at(D0, "08:40"), tg)
    ns_front(g, ["F-039"], at(D0, "08:32"), at(D0, "10:30"), tg, send=False)
    issue(g, at(D1, "08:02"), "LOT-W-88", 1, "ST-WC-P1", tags=tg, label="issue/LOT-W-88@0928")
    for it, ring, st, hm in (("F-037", "R-902", "ST-WC-P1", "08:05"), ("F-038", "R-903", "ST-WC-P2", "08:06")):
        assert in_wc[it] <= at(D1, hm), (it, in_wc[it])
        issue(g, at(D1, hm), "LOT-R-119", 1, st, items=[it], components=[ring], tags=tg, label="issue/ring/" + it)
    ns_weld(g, "F-037", "WS-1", "WLD-02", at(D1, "08:30"), at(D1, "09:10"), "R-902", "LOT-R-119", tg, prep=at(D1, "08:20"))
    ns_weld(g, "F-038", "WS-2", "WLD-02", at(D1, "09:20"), at(D1, "10:00"), "R-903", "LOT-R-119", tg, prep=at(D1, "09:14"))
    for it, no, hx, hz in (("F-037", "RK-0928-41", "10:30", "11:00"), ("F-038", "RK-0928-42", "10:50", "11:15")):
        xray(g, at(D1, hx), it, no, tags=tg)
        gate(g, at(D1, hz), "QC-01", "ZT-3", item=it, tags=tg, basis=["kt3/SV-%s-1/CAM-WS-1" % num(it), "xr/%s/%s" % (it, no)],
             label="ZT-3/" + it)
    # ---- 12:40 извещение об изменении КД: новая ревизия из КОМПАС, номенклатура из 1С
    g.ev(at(D1, "12:40"), "CAD-KOMPAS", "assembly.structure_imported",
         payload=OrderedDict([("file", "process/kompas-assembly-flange.json"), ("format_version", "1.0"),
                              ("assembly_id", "FL-100.00.000"), ("item_type_id", "FL-100-ASSY"), ("version", "V"),
                              ("previous_version", "B"),
                              ("change_notice", OrderedDict([
                                  ("id", CN_ID), ("display", CN_DISPLAY), ("from_revision", "B"), ("to_revision", "V"),
                                  ("changes", [OrderedDict([("char_id", "D2"), ("from_mm", 250.00), ("to_mm", 252.00),
                                                            ("why", "под уплотнение новой конструкции [П]")])]),
                                  ("effective", "изделия, запущенные после извещения"),
                                  ("wip", "задел — по решению технолога: доработать до «В», оставить по «Б» по разрешению на "
                                          "отклонение или списать")])),
                              ("geometry", None)]),
         tags=tg, label="import/kompas-FL-100@V")
    g.ev(at(D1, "12:45"), "ERP-1C", "erp.nomenclature_synced",
         payload=OrderedDict([("message_id", "1c-in-000350"), ("nomenclature_ext_id", "nom-7a1e-0001"),
                              ("item_type_id", "FL-100-ASSY"), ("designation", "ФЛ-100.00.000 СБ"), ("kd_revision", "V"),
                              ("tp_id", "TP-FL-100"), ("tp_revision", 4), ("change_notice", CN_ID)]),
         tags=tg, label="erp/nomenclature/FL-100@V")
    WIP = ["F-037", "F-038", "F-039"]
    g.engine_holds.append(dict(at=at(D1, "12:46"), targets=list(WIP), basis="kd_change:" + CN_ID, node="OC2"))
    subj = lambda it: {"order_id": "ORD-0917", "item_id": it, "change_notice": CN_ID}
    g.step(at(D1, "13:30"), "TECH-01", "TECHNOLOGIST", "kd.wip_decision", subj("F-039"),
           {"decision": "rework_to_new_revision", "state": "мехобработан по «Б», ЗТ-2 пройдена, в механическом цехе",
            "how": "расточить D2 с 250,00 до 252,00 по ТП ред. 4 (УП ред. 4), КИМ, повторная ЗТ-2", "revision_after": "V",
            "basis_labels": ["import/kompas-FL-100@V", "ZT-2/F-039"]},
           mode="stop", tags=tg, catalog="Е-127 [П]", label="S19/wip-F-039")
    g.step(at(D1, "13:32"), "TECH-01", "TECHNOLOGIST", "kd.wip_decision", subj("F-037"),
           {"decision": "keep_old_with_concession", "state": "сварен по «Б», ЗТ-3 пройдена",
            "why": "D2 после сварки не растачивают; уплотнение старой конструкции (УП-401) на складе — одно, хватает на одно изделие",
            "needs": "разрешение на отклонение с подписью ПЗ", "revision_after": "B",
            "basis_labels": ["import/kompas-FL-100@V", "ZT-3/F-037"]},
           tags=tg, catalog="Е-127 [П]", label="S19/wip-F-037")
    g.step(at(D1, "13:34"), "TECH-01", "TECHNOLOGIST", "kd.wip_decision", subj("F-038"),
           {"decision": "scrap", "state": "сварен по «Б», ЗТ-3 пройдена",
            "why": "доработать до «В» нельзя — D2 после сварки не растачивают; оставить по «Б» нельзя — второго уплотнения "
                   "старой конструкции нет", "basis_labels": ["import/kompas-FL-100@V", "ZT-3/F-038"]},
           tags=tg, catalog="Е-127 [П]", signers=["TECH-01", "QC-HEAD-01"], label="S19/wip-F-038")
    g.step(at(D1, "13:45"), "TECH-01", "TECHNOLOGIST", "permit.approved",
           {"order_id": "ORD-0917", "item_id": "F-037", "permit_no": "RO-0928-01"},
           {"display_no": "РО-0928-01 [П]", "what": "Ф-037 — по ревизии «Б» (D2 = 250,00) с уплотнением УП-401 старой конструкции",
            "scope": "только Ф-037", "reason": "извещение " + CN_DISPLAY + ": задел, доработка невозможна",
            "basis_labels": ["S19/wip-F-037"]},
           tags=tg, catalog="E-95", signers=["TECH-01", "QC-HEAD-01", "CR-01"], label="S19/permit-F-037")
    # ---- доработка Ф-039 до «В» и запуск Ф-040 по «В»
    t = s19_rework(g, "F-039", at(D1, "14:00"), tg)
    manual_kt2(g, "F-039", t + M(4), tg)
    cmm(g, "F-039", t + M(19), tags=tg, label="cmm/F-039/rev-V", nominals=D2_REV_V,
        extra={"kd_revision": "V", "trigger": "after_rework_to_new_revision"})
    g.ev(t + M(24), "TERM-MC", "item.presented", item="F-039",
         payload=OrderedDict([("item_id", "F-039"), ("presentation_no", 2), ("presented_to", ["QC"]), ("presented_by", "MST-01"),
                              ("gate", "ZT-2")]), tags=tg, label="present/F-039/ZT-2#2", persona="MST-01")
    gate(g, t + M(29), "QC-01", "ZT-2", item="F-039", tags=tg, presentation=2,
         basis=["kt2-manual/F-039", "cmm/F-039/rev-V"],
         extra={"kd_revision": "V", "revision_history": "Б → В: доработано по " + CN_DISPLAY}, label="ZT-2#2/F-039")
    ns_front(g, ["F-040"], at(D1, "14:05"), at(D1, "14:35"), tg, send=False, kd="V", tp=4, prog="4", kt2_manual=True,
             cmm_nominals=D2_REV_V)
    return g


def build_s20():
    """S20 «Маркировка не читается / детали перепутали». Скан на посту: ожидался Ф-031, пришёл Ф-033 → шаг остановлен,
    мастер выясняет (фланцы лежали в таре друг друга), история не смешивается. У Ф-032 код не читается → ручной ввод
    с подтверждением мастера, затем перемаркировка по процедуре; ОТК сверяет на ЗТ-3. Мир: Ф-031…Ф-033 варят в Ср 30.09."""
    g = Gen("S20", 20261020)
    tg = ("S20",)
    D0, D1 = dt.date(2026, 9, 29), dt.date(2026, 9, 30)
    items = ["F-031", "F-032", "F-033"]
    rings = {it: ring_of(it)[0] for it in items}
    ns_lots(g, at(D0, "07:30"), ["LOT-ZF-201", "LOT-R-116", "LOT-W-88"],
            {"LOT-ZF-201": items, "LOT-R-116": [rings[x] for x in items]}, tg)
    in_wc = ns_front(g, items, at(D0, "08:40"), at(D0, "08:50"), tg)
    issue(g, at(D1, "08:00"), "LOT-W-88", 1, "ST-WC-P1", tags=tg, label="issue/LOT-W-88@0930")
    for k, it in enumerate(items):
        assert in_wc[it] <= at(D1, "08:00")
        issue(g, at(D1, "08:05") + M(k), "LOT-R-116", 1, "ST-WC-P1", items=[it], components=[rings[it]], tags=tg,
              label="issue/ring/" + it)
    scanned(g, at(D1, "08:12"), "TERM-WS-1", "F-033", "WLD-02", "weld_link", "mismatch", tg, "S20/scan-mismatch",
            expected="F-031", decoded="DM:F-033", station_id="ST-WC-P1", tare_label="F-031", ring_in_tare=rings["F-031"],
            note="Е-13 E_ITEM_MISMATCH: ожидался Ф-031 (сменное задание, бирка тары), прочитан Ф-033 — шаг остановлен, "
                 "запись в журнал критических действий, задача мастеру М-02")
    scanned(g, at(D1, "08:20"), "TERM-WC", "F-031", "MST-02", "identity_check", "ok", tg, "S20/scan-found-F-031",
            decoded="DM:F-031", station_id="ST-WC-P1", tare_label="F-033")
    g.step(at(D1, "08:25"), "MST-02", "MASTER", "item.binding_corrected", {"items": ["F-031", "F-033"]},
           {"resolution": "swap_found", "expected": "F-031", "found": "F-033",
            "where": "Ф-031 лежал в таре «Ф-033», Ф-033 — в таре «Ф-031»",
            "cause": "при приёмке во вторник фланцы положили в тару друг друга; бирки тары — по сменному заданию",
            "fixed": "фланцы переложены в свою тару; кольца К-028 и К-030 — к своим фланцам", "history_mixed": False,
            "events_rebound": 0, "basis_labels": ["S20/scan-mismatch", "S20/scan-found-F-031"]},
           mode="stop", tags=tg, catalog="E-101", label="S20/swap-resolved")
    ns_weld(g, "F-031", "WS-1", "WLD-02", at(D1, "08:40"), at(D1, "09:20"), rings["F-031"], "LOT-R-116", tg, prep=at(D1, "08:32"))
    scanned(g, at(D1, "09:25"), "TERM-WS-1", None, "WLD-02", "weld_link", "unreadable", tg, "S20/scan-unreadable",
            expected="F-032", decoded=None, station_id="ST-WC-P1", tare_label="F-032",
            reason="код DataMatrix повреждён — царапина по полю кода [П]",
            note="Е-13 E_ITEM_NOT_IDENTIFIED: код не читается — шаг остановлен; ручной ввод с подтверждением второго или перемаркировка")
    g.step(at(D1, "09:28"), "MST-02", "MASTER", "item.binding_corrected", {"item_id": "F-032"},
           {"method": "manual_entry", "entered_by": "WLD-02", "confirmed_by": "MST-02",
            "checked_against": ["бирка тары «Ф-032»", "сопроводительная карта Ф-032", "кольцо К-029 выдано под Ф-032",
                                "последнее чтение кода — ЗТ-2 во вторник"],
            "reliability": "manual_confirmed", "event_label": "S20/scan-unreadable",
            "then": "перемаркировка по процедуре до ЗТ-3; ОТК-1 сверяет на ЗТ-3"},
           tags=tg, catalog="E-101", signers=["WLD-02", "MST-02"], label="S20/manual-F-032")
    ns_weld(g, "F-032", "WS-1", "WLD-02", at(D1, "09:40"), at(D1, "10:20"), rings["F-032"], "LOT-R-116", tg,
            prep=at(D1, "09:32"), link_method="manual_confirmed")
    ns_weld(g, "F-033", "WS-1", "WLD-02", at(D1, "10:35"), at(D1, "11:15"), rings["F-033"], "LOT-R-116", tg, prep=at(D1, "10:27"))
    g.ev(at(D1, "11:05"), "TERM-WC", "item.marked", item="F-032",
         payload=OrderedDict([("item_id", "F-032"), ("carrier", carrier_dm("F-032")),
                              ("previous_carrier", OrderedDict([("type", "dpm_datamatrix"), ("value", "DM:F-032"),
                                                                ("state", "unreadable")])),
                              ("method", "laser_dpm_portable"), ("method_note", "переносной лазерный маркёр [ПП]"), ("zone", "MARKING-2"),
                              ("reason", "remark_unreadable"),
                              ("identity_basis", "ручной ввод 09:28, подтверждён М-02; сопроводительная карта"),
                              ("read_back", OrderedDict([("decoded", "DM:F-032"), ("grade", "B"), ("match_previous", True)])),
                              ("witness_id", "QC-01"), ("operator_id", "MRK-01")]),
         tags=tg, label="remark2/F-032", persona="MRK-01")
    for it, no, hx, hz in (("F-031", "RK-0930-51", "11:30", "12:10"), ("F-032", "RK-0930-52", "11:45", "12:20"),
                           ("F-033", "RK-0930-53", "12:00", "13:05")):
        xray(g, at(D1, hx), it, no, tags=tg)
        ex = None
        if it == "F-032":
            ex = {"marking_check": {"previous": "DM:F-032 (не читается)", "current": "DM:F-032", "result": "match",
                                    "readable": True, "basis_label": "remark2/F-032"}}
        gate(g, at(D1, hz), "QC-01", "ZT-3", item=it, tags=tg, basis=["kt3/SV-%s-1/CAM-WS-1" % num(it), "xr/%s/%s" % (it, no)],
             extra=ex, label="ZT-3/" + it)
    return g


def build_s21():
    """S21 «Смешение партий»: кольца П-117 (претензия) оказались в таре П-116. К-107 выдано под Ф-034 до блокировки и вварено —
    после претензии генеалогия ставит Ф-034 в круг по партии, рентген тела кольца чист — исключён. К-108 при выдаче под
    Ф-036: скан — «партия заблокирована», выдача не проходит; ОТК-2 проверяет тару сканом. Мир: вторая половина заказа
    ЗП-0917 варится позже, поэтому у П-116 есть кольца К-031…К-033."""
    g = Gen("S21", 20261021)
    tg = ("S21",)
    D1, D2, D3, D4 = dt.date(2026, 9, 21), dt.date(2026, 9, 22), dt.date(2026, 9, 23), dt.date(2026, 9, 24)
    items = ["F-034", "F-035", "F-036"]
    r117 = ["R-%03d" % n for n in range(101, 111)]
    ns_lots(g, T("2026-09-17 07:30"), ["LOT-ZF-201", "LOT-R-116", "LOT-W-88", "LOT-R-117"],
            {"LOT-ZF-201": items, "LOT-R-116": ["R-031", "R-032", "R-033"], "LOT-R-117": r117}, tg,
            when={"LOT-R-117": T("2026-09-22 08:00")})
    in_wc = ns_front(g, items, at(D1, "08:30"), at(D1, "08:40"), tg)
    # Вт 22.09: кольцо под Ф-034 — из тары П-116, по коду кольца — К-107 партии П-117 (партия ещё годна)
    assert in_wc["F-034"] <= at(D2, "16:00")
    scanned(g, at(D2, "16:00"), "TERM-STK", "R-107", "STK-01", "issue", "ok", tg, "S21/scan-R-107",
            decoded="DM:R-107", for_item="F-034", tare_label="LOT-R-116", lot_id="LOT-R-117", lot_status="released")
    issue(g, at(D2, "16:01"), "LOT-R-117", 1, "ST-WC-P1", items=["F-034"], components=["R-107"], tags=tg,
          label="issue/ring/F-034")
    ns_weld(g, "F-034", "WS-1", "WLD-03", at(D2, "16:45"), at(D2, "17:25"), "R-107", "LOT-R-117", tg, prep=at(D2, "16:38"))
    xray(g, at(D3, "09:00"), "F-034", "RK-0923-21", tags=tg)
    gate(g, at(D3, "09:30"), "QC-01", "ZT-3", item="F-034", tags=tg, basis=["kt3/SV-034-1/CAM-WS-1", "xr/F-034/RK-0923-21"],
         label="ZT-3/F-034")
    # Ср 15:45 — претензия по П-117 (как в главной истории): партия заблокирована; по генеалогии Ф-034 — в круг по партии
    back = [x for x in r117 if x != "R-107"]
    g.step(at(D3, "15:45"), "TECH-01", "TECHNOLOGIST", "decision.disposition_set", {"lot_id": "LOT-R-117", "nc_ref": "NC-21"},
           {"disposition": "return_to_supplier", "qty": len(back), "components": back,
            "reason": "претензия Поставщику-3 по П-117: поры в теле колец (главная история, S02: К-101, К-105) [П]",
            "genealogy": "из партии уже в изделиях: К-107 → Ф-034 (выдано Вт 16:01 по скану кольца)"},
           mode="stop", tags=tg, catalog="E-94", signers=["TECH-01", "QC-HEAD-01"], label="S21/claim")
    g.engine_holds.append(dict(at=at(D3, "15:45"), targets=["LOT-R-117"], basis="nc:NC-21", node="N1"))
    g.engine_holds.append(dict(at=at(D3, "15:46"), targets=["F-034"], basis="risk_scope:RS-21", node="I2"))
    g.step(at(D3, "16:00"), "TECH-01", "TECHNOLOGIST", "risk_scope.item_assessed", {"risk_scope_ref": "RS-21", "item_id": "F-034"},
           {"result": "excluded", "basis_labels": ["xr/F-034/RK-0923-21"],
            "why": "кольцо К-107 партии П-117 — по генеалогии; рентген шва и тела кольца в зоне снимка — чисто (как у Ф-027 и "
                   "Ф-029 в S02)", "holds_released": "containment.hold_released (Е-116) этим шагом"},
           tags=tg, catalog="E-113", label="S21/exclude-F-034")
    g.step(at(D3, "16:05"), "TECH-01", "TECHNOLOGIST", "risk_scope.closed", {"risk_scope_ref": "RS-21"},
           {"was": 1, "confirmed": 0, "excluded": 1, "note": "круг по партии П-117: изделия с её кольцами — Ф-034"},
           tags=tg, catalog="E-115", label="S21/close")
    # Чт 24.09: выдача колец под Ф-035 и Ф-036 из той же тары
    issue(g, at(D4, "08:00"), "LOT-W-88", 1, "ST-WC-P1", tags=tg, label="issue/LOT-W-88@0924")
    scanned(g, at(D4, "08:05"), "TERM-STK", "R-032", "STK-01", "issue", "ok", tg, "S21/scan-R-032",
            decoded="DM:R-032", for_item="F-035", tare_label="LOT-R-116", lot_id="LOT-R-116", lot_status="released")
    issue(g, at(D4, "08:06"), "LOT-R-116", 1, "ST-WC-P1", items=["F-035"], components=["R-032"], tags=tg, label="issue/ring/F-035")
    scanned(g, at(D4, "08:10"), "TERM-STK", "R-108", "STK-01", "issue", "ok", tg, "S21/scan-R-108-blocked",
            decoded="DM:R-108", for_item="F-036", tare_label="LOT-R-116", lot_id="LOT-R-117", lot_status="blocked",
            note="Е-13 E_LOT_BLOCKED: партия П-117 заблокирована (претензия, возврат поставщику) — выдача не проходит; "
                 "запись в журнал критических действий; задача ОТК-2 — проверить тару П-116 сканом")
    for k, rr in enumerate(("R-031", "R-033")):
        scanned(g, at(D4, "08:15") + M(k), "TERM-STK", rr, "QC-02", "tare_check", "ok", tg, "S21/tare-" + rr,
                decoded="DM:" + rr, tare_label="LOT-R-116", lot_id="LOT-R-116", lot_status="released")
    g.ev(at(D4, "08:18"), "TERM-STK", "movement.isolator_confirmed", item="R-108",
         payload=OrderedDict([("item_id", "R-108"), ("to_station", "ST-ISO-WC"), ("physically_moved", True),
                              ("confirmed_by", "QC-02"),
                              ("reason", "кольцо партии П-117 (претензия) в таре П-116 — к остатку партии до отгрузки поставщику")]),
         tags=tg, label="S21/isolator-R-108", persona="QC-02", after="S21/claim")
    scanned(g, at(D4, "08:25"), "TERM-STK", "R-033", "STK-01", "issue", "ok", tg, "S21/scan-R-033",
            decoded="DM:R-033", for_item="F-036", tare_label="LOT-R-116", lot_id="LOT-R-116", lot_status="released")
    issue(g, at(D4, "08:26"), "LOT-R-116", 1, "ST-WC-P1", items=["F-036"], components=["R-033"], tags=tg, label="issue/ring/F-036")
    ns_weld(g, "F-035", "WS-1", "WLD-02", at(D4, "08:40"), at(D4, "09:20"), "R-032", "LOT-R-116", tg, prep=at(D4, "08:32"))
    ns_weld(g, "F-036", "WS-1", "WLD-02", at(D4, "09:30"), at(D4, "10:10"), "R-033", "LOT-R-116", tg, prep=at(D4, "09:24"))
    for it, no, hx, hz in (("F-035", "RK-0924-21", "11:00", "11:40"), ("F-036", "RK-0924-22", "11:20", "12:00")):
        xray(g, at(D4, hx), it, no, tags=tg)
        gate(g, at(D4, hz), "QC-01", "ZT-3", item=it, tags=tg, basis=["kt3/SV-%s-1/CAM-WS-1" % num(it), "xr/%s/%s" % (it, no)],
             label="ZT-3/" + it)
    return g


def add_s22(g):
    """S22 «Попытка вернуть в работу списанную деталь» — продолжение прогона S10B (Ф-090 списан Пн 28.09 15:30).
    Вт 29.09: скан на посту и решение «переделка» — отказ (статус «списано» конечный), журнал критических действий.
    Разрешено только отдельное решение «образец на разрушающий контроль» — макрошлиф У4, без возврата в маршрут."""
    tg = ("S22",)
    D = dt.date(2026, 9, 29)
    scanned(g, at(D, "08:20"), "TERM-WS-1", "F-090", "WLD-02", "operation_start", "ok", tg, "S22/scan-F-090",
            decoded="DM:F-090", station_id="ST-WC-P1", operation_code="W2", item_status="scrapped",
            note="Е-13 E_ITEM_SCRAPPED: изделие списано (НС-90, Пн 28.09 15:30) — начать работу нельзя; запись в журнал "
                 "критических действий")
    g.step(at(D, "08:30"), "TECH-01", "TECHNOLOGIST", "decision.disposition_set", {"item_id": "F-090", "nc_ref": "NC-90"},
           {"disposition": "rework", "rework_zones": ["U4"], "reason": "вернуть в работу: ещё одна попытка на У4 с новым держателем горелки"},
           mode="stop", tags=tg, catalog="E-94", expect="refused:E_ITEM_SCRAPPED", label="S22/rework-refused",
           note="проверка правила: «списано» — конечный статус; возврат в маршрут — отказ и запись в журнал критических действий")
    g.step(at(D, "09:00"), "TECH-01", "TECHNOLOGIST", "item.sample_designated", {"item_id": "F-090"},
           {"purpose": "destructive_test", "test": "макрошлиф участка У4",
            "why": "три подреза на У4 подряд — посмотреть форму и глубину подреза и провар корня; основание для разбора причин "
                   "по НС-87…НС-90", "route_return": False, "status_after": "списано — образец",
            "erp": "ничего нового: в 1С изделие уже списано"},
           mode="stop", tags=tg, catalog="Е-129 [П]", signers=["TECH-01", "QC-HEAD-01"], label="S22/sample")
    g.ev(at(D, "13:00"), "TERM-IQC", "inspection.result", item="F-090",
         payload=OrderedDict([("checkpoint_id", "CP-WELD"), ("phase", "destructive_test"), ("method", "macro_section"),
                              ("sample_of", "F-090"), ("item_status", "scrapped_sample"), ("operator_id", "QC-02"),
                              ("zones_inspected", ["U4"]), ("inspection_result", "defect_found"),
                              ("defects", [OrderedDict([("defect_type", "undercut"), ("zone", "U4"), ("depth_mm", 0.6),
                                                        ("severity", "major")])]),
                              ("observation", "подрез 0,6 мм по краю шва на слоях переделок; провар корня полный [ПП]"),
                              ("signal", "не заводится: изделие списано, это образец")]),
         tags=tg, label="S22/macro-U4", persona="QC-02", after="S22/sample")


# ----------------------------------------------------------------------------- телеметрия сварочных источников
def auto_power(g):
    wins = {}
    for w in g.welds:
        if w.get("cs"):
            continue
        k = (w["src"], shift_key(w["arc"][0]))
        if k in g.power_override:
            continue
        a0, a1 = w["arc"]
        lo, hi = wins.get(k, (a0, a1))
        wins[k] = (min(lo, a0), max(hi, a1))
    out = []
    for (src, sk), (a0, a1) in wins.items():
        on = (a0 - M(5)).replace(second=0, microsecond=0)
        off = (a1 + M(5)).replace(second=0, microsecond=0) + M(1)
        out.append((src, on, off, "end_of_work", None))
    return out


def current_at(w, t):
    prof = w.get("profile")
    r = rnd("cur|%s|%s" % (w["run"], t.isoformat()))
    if not prof:
        return None
    cur = None
    for (a, lo, hi) in prof:
        if t >= a:
            cur = (lo, hi)
    return cur


def seg_at(w, t):
    a0, a1 = w["arc"]
    if w.get("seg_bounds"):
        d = t.astimezone(MSK).date()
        for sname, s, e in w["seg_bounds"]:
            if at(d, s) <= t < at(d, e):
                return sname
        return "U8"
    frac = (t - a0).total_seconds() / max(1.0, (a1 - a0).total_seconds())
    return SEGS[min(7, max(0, int(frac * 8)))]


WARN_AT = {"F-015": "2026-09-22 10:20:14", "F-021": "2026-09-22 19:45:00", "F-023": "2026-09-23 08:35:00",
           "F-025": "2026-09-23 09:40:00", "F-017": "2026-09-23 10:42:00"}   # S07: «встают на своё время»
WARN_VAL = {"F-015": 176, "F-019": 178, "F-021": 178, "F-023": 177, "F-025": 176, "F-017": 176, "CS-07": 181}


def telemetry(g, p2=None, only_src=None):
    """Сводки на окно 60 с, пока источник включён; предупреждение при первом выходе тока за уставку в выполнении."""
    recs = []
    wins = auto_power(g)
    for (src, on, off, reason, after) in g.power:
        if off == "SOLVE_P2":
            off = p2
        wins.append((src, on, off, reason, after))
    if only_src:
        wins = [w for w in wins if w[0] == only_src]
    welds_by_src = defaultdict(list)
    for w in g.welds:
        welds_by_src[w["src"]].append(w)
    for (src, on, off, reason, after) in sorted(wins, key=lambda x: (x[0], x[1])):
        recs.append(dict(occ=on, src=src, type="machine.state",
                         payload=OrderedDict([("equipment_id", src), ("state", "idle"), ("reason", "power_on")]),
                         hints=set(), label="%s/power-on/%s" % (src, on.astimezone(MSK).strftime("%m%d-%H%M"))))
        t = on.replace(second=0, microsecond=0) + M(1)
        while t <= off:
            w0 = t - M(1)
            arc_s = 0
            wref = None
            for w in welds_by_src[src]:
                a0, a1 = w["arc"]
                lo, hi = max(a0, w0), min(a1, t)
                if hi > lo:
                    ov = (hi - lo).total_seconds()
                    for (g0, g1) in w.get("arc_gaps", []):
                        gl, gh = max(g0, lo), min(g1, hi)
                        if gh > gl:
                            ov -= (gh - gl).total_seconds()
                    if ov > 0:
                        arc_s += int(ov)
                        wref = w
            p = OrderedDict([("equipment_id", src), ("window_start", iso(w0)), ("window_s", 60),
                             ("state", "running" if arc_s else "idle"), ("arc_on_s", arc_s)])
            hints = set()
            if wref:
                mid = max(wref["arc"][0], w0) + (min(wref["arc"][1], t) - max(wref["arc"][0], w0)) / 2
                rng = rnd("tele|%s|%s" % (wref["run"], t.isoformat()))
                prof = current_at(wref, mid)
                if prof:
                    lo, hi = prof
                else:
                    lo, hi = 158, 163
                avg = rng.randint(lo + 1, hi - 1) if hi - lo >= 2 else lo
                mn = max(lo, avg - rng.randint(1, 2))
                mx = min(hi, avg + rng.randint(1, 2))
                a0 = wref["arc"][0]
                first_w = (t - M(1) <= a0 < t)
                last_w = (t - M(1) < wref["arc"][1] <= t)
                if not prof and first_w:
                    mn = 158
                if not prof and last_w:
                    mx = 163
                p["program_id"] = "PS-4"
                p["seam_segment"] = seg_at(wref, mid) if not wref.get("cs") else None
                p["parameters"] = OrderedDict([
                    ("welding_current_a", OrderedDict([("avg", avg), ("min", mn), ("max", mx)])),
                    ("arc_voltage_v", OrderedDict([("avg", round(rng.uniform(13.6, 14.4), 1))])),
                    ("wire_feed_m_min", OrderedDict([("avg", round(rng.uniform(5.8, 6.2), 1))])),
                    ("gas_flow_l_min", OrderedDict([("avg", rng.randint(11, 13))]))])
                p["setpoints"] = OrderedDict([("welding_current_a", SETPOINT_CURRENT)])
                hints.add(wref["item"])
            else:
                p["parameters"] = None
            recs.append(dict(occ=t, src=src, type="machine.parameters", payload=p, hints=hints,
                             label="%s/sum/%s" % (src, t.astimezone(MSK).strftime("%m%d-%H%M"))))
            t = t + M(1)
        recs.append(dict(occ=off, src=src, type="machine.state",
                         payload=OrderedDict([("equipment_id", src), ("state", "stopped"), ("reason", reason)]),
                         hints=set(), label="%s/stop/%s" % (src, off.astimezone(MSK).strftime("%m%d-%H%M%S")),
                         after=after))
    # предупреждения: первый выход тока за уставку в каждом выполнении
    for w in g.welds:
        prof = w.get("profile")
        if not prof or (only_src and w["src"] != only_src):
            continue
        first = None
        for (a, lo, hi) in prof:
            if lo > 170 or hi > 170:
                first = a if first is None else first
                break
        if first is None:
            continue
        tw = T(WARN_AT[w["item"]]) if w["item"] in WARN_AT else first + S(20)
        val = WARN_VAL[w["item"]]
        label = "EV-WS2-0412" if w["item"] == "F-015" else "%s/warning/%s" % (w["src"], w["run"])
        recs.append(dict(occ=tw, src=w["src"], type="machine.state",
                         payload=OrderedDict([("equipment_id", w["src"]), ("state", "warning"),
                                              ("code", "WELD_CURRENT_HIGH"), ("parameter", "welding_current_a"),
                                              ("value", val), ("setpoint", SETPOINT_CURRENT), ("program_id", "PS-4"),
                                              ("seam_segment", seg_at(w, tw) if not w.get("cs") else None)]),
                         hints={w["item"]}, label=label))
    return recs


def add_telemetry(g):
    p2 = None
    if any(x[2] == "SOLVE_P2" for x in g.power):
        b = g.batches[0]
        lw = g.lost_windows[0]

        def uniq(p2c):
            recs = telemetry(g, p2c, only_src=b["src"])
            inwin = [r for r in recs if b["frm"] <= r["occ"] <= b["to"]]
            lost = [r for r in inwin if lw[1] < r["occ"] < lw[2]]
            return len(inwin) - len(lost)
        lo, hi = 0, 260          # минуты от Вт 20:40 (после сварки F-021) до 01:00
        base = T("2026-09-22 20:40")
        assert uniq(base + M(lo)) <= b["expect_unique"] <= uniq(base + M(hi)), (uniq(base + M(lo)), uniq(base + M(hi)))
        while lo < hi:
            mid = (lo + hi) // 2
            if uniq(base + M(mid)) < b["expect_unique"]:
                lo = mid + 1
            else:
                hi = mid
        p2 = base + M(lo)
        assert uniq(p2) == b["expect_unique"], ("не удалось подобрать выключение ИС-2", uniq(p2))
    recs = telemetry(g, p2)
    for r in recs:
        e = g.ev(r["occ"], r["src"], r["type"], payload=r["payload"], hints=r["hints"], label=r["label"],
                 after=r.get("after"))
        e["tele"] = True
    if g.stream_id == "main-story":
        hit = [e for e in g.events if e["src"] == "WS-1" and e["type"] == "machine.parameters"
               and e["occ"] == T("2026-09-22 14:00")]
        assert len(hit) == 1, len(hit)
        e = hit[0]
        e["schema"] = "9.0"
        e["noise"] = "malformed:E_UNSUPPORTED_VERSION"
        e["tags"].add("S12")
        e["note"] = "S12 сообщение 2: сводка ИС-1 с версией контракта 9.0 (неизвестная мажорная версия)"
        pl = e["payload"]
        e["payload"] = OrderedDict([("equipment", {"id": pl["equipment_id"], "kind": "welding_source"}),
                                    ("window", {"start": pl["window_start"], "seconds": 60}),
                                    ("state", pl["state"]), ("arc_on_s", pl["arc_on_s"]), ("measurements", [])])
    return p2


# ----------------------------------------------------------------------------- линия в конверте
def assign_lines(g):
    """line_id по правилу из POST_LINE (см. комментарий там). Вызывать после телеметрии, до finalize."""
    reg = {}
    runs = defaultdict(list)
    for e in g.events:
        if e["type"] == "item.registered" and e["item"]:
            reg.setdefault(e["item"], e["occ"])
        if e["type"] == "operation.started" and e["item"] and e["payload"].get("operation_code") == "W2":
            runs[e["item"]].append((e["occ"], STATION_LINE[e["payload"]["station_id"]]))
    for v in runs.values():
        v.sort()
    for e in g.events:
        e["line"] = None
        if e["src"] in POST_LINE:
            e["line"] = POST_LINE[e["src"]]
            continue
        it = e["item"]
        if not it or e["drop_item"] or not it.startswith("F-") or it not in reg or e["occ"] < reg[it] or not runs.get(it):
            continue
        cur = runs[it][0][1]           # до первой сварки — план запуска
        for (t, ln) in runs[it]:
            if t <= e["occ"]:
                cur = ln
        e["line"] = cur


# ----------------------------------------------------------------------------- экземпляры процессов (схема v0.2)
# [П] Схема v0.2 — шесть процессов, у каждого свои экземпляры (bpmn-v02-changes.md, «Коротко», п. 1):
#   order/<ORD>  «Заказ из 1С»            старт S0 — erp.order_received (Е-01)
#   lot/<LOT>    «Партия»                 старт S1 — erp.batch_received (Е-03); Е-04…Е-08, маркировка, склад, выдача
#   item/<F>     «Фланец в сборе»         старт F0/V5 — item.registered (Е-09, выдача заготовки); всё по изделию дальше
#   incident/<RS> «Инцидент»              старт I0 — реакция движка (Е-110); в потоках нет, только шаги людей
#   review/<ZT>/<F> «Пересмотр приёмки»   старт R0 — реакция движка decision.review_flagged (Е-122 [П])
#   nc/<NC>      «Решение по несоответствию (Н)» — вызывается из изделия, партии или инцидента (callActivity)
# Общих фишек нет: изделие связано с партией сканом при выдаче, с инцидентом — статусом блока и сообщением о решении.
# В строке потока: _sim.process — экземпляр, которому принадлежит факт; _sim.process_links — кому ещё он нужен
# (сообщение между процессами). Это ожидание для проверки корреляции, а не поле контракта: ядро выводит экземпляр
# само по деловым ключам (item_id, lot_id, order_id). Оборудование без item_id, КОМПАС, номенклатура, контрольный
# образец и сообщение без item_id (S12) — вне экземпляров (null).
PROCESS_KINDS = ("order", "lot", "item", "incident", "review", "nc", "signal", "cause", "equipment")
# Схема v0.3 [П] добавила три процесса (bpmn-v03-changes.md, «Коротко», п. 1):
#   signal/SG-<основание>   «Сигнал» — один экземпляр на сигнал; в данных — шаги Е-90, Е-91 (реакции Е-80, Е-84 — в ожиданиях)
#   cause/<RS>              «Разбор причин и корректирующие действия» — П-02…П-10 (S02 — RS-02, S05 — RS-01, S14 — RS-14)
#   equipment/<метка>       «Сбой оборудования, нарушение режима, пропуск проверки» — один экземпляр на событие:
#                           Е-43 (предупреждение источника), Е-21 (пропуск проверки), Е-22 (ручное вмешательство),
#                           запись пульта станка без номера выполнения (S16)


def slug(x):
    return x.replace("/", "-").replace("#", "-")


def assign_process(g):
    """_sim.process / _sim.process_links для строк потока. Вызывать до finalize (дубли копируют поля)."""
    reg = {}
    piece_lot = {}
    comp_item = {}
    for e in sorted(g.events, key=lambda e: (e["occ"], e["i"])):
        p = e["payload"] if isinstance(e["payload"], dict) else {}
        if e["type"] == "item.registered" and e["item"]:
            reg.setdefault(e["item"], e["occ"])
        if e["type"] == "item.marked" and e["item"] and p.get("lot_id"):
            piece_lot.setdefault(e["item"], p["lot_id"])
        if e["type"] == "inspection.result" and e["item"] and p.get("lot_id"):
            piece_lot.setdefault(e["item"], p["lot_id"])
        if e["type"] == "assembly.component_linked" and p.get("component_id"):
            comp_item.setdefault(p["component_id"], p.get("assembly_item_id"))
    for e in g.events:
        p = e["payload"] if isinstance(e["payload"], dict) else {}
        it = None if e["drop_item"] else e["item"]
        proc, links = None, []
        t = e["type"]
        if (t == "machine.state" and p.get("state") == "warning") or \
                (t == "operator.action" and p.get("action_type") in ("check_skipped", "manual_override")) or \
                (t == "operator.action" and e["src"] in ("CNC-1",)):
            # схема v0.3: «Сбой оборудования, нарушение режима, пропуск проверки» — экземпляр на событие; изделие — связь
            proc = "equipment/" + slug(e["label"])
            links = ["item/" + it] if it and it.startswith("F-") and it in reg and e["occ"] >= reg[it] else []
        elif t == "erp.order_received":
            proc = "order/" + p["order_id"]
        elif t in ("erp.nomenclature_synced", "assembly.structure_imported"):
            proc = None
        elif t == "erp.batch_received":
            proc = "lot/" + e["lot"]
        elif t == "item.registered":
            proc = "item/" + it
            links = ["order/" + p["order_id"], "lot/" + p["batch_id"]]
        elif t == "item.scanned":
            # Е-126 [П]: скан на посту или при выдаче — экземпляр того, чей код прочитан; ожидавшееся изделие — связь
            want = [x for x in (p.get("expected_item_id"), p.get("for_item")) if x and x in reg and e["occ"] >= reg[x]]
            if it and it.startswith("F-"):
                proc = "item/" + it
            elif it and it.startswith(("R-", "C-")):
                proc = "lot/" + piece_lot[it]
            elif want:
                proc = "item/" + want[0]
                want = want[1:]
            links = ["item/" + x for x in want if "item/" + x != proc]
        elif t == "batch.issued":
            proc = "lot/" + p["lot_id"]
            fi = list(p.get("for_items") or [])
            for c in p.get("component_ids") or []:
                if comp_item.get(c) and comp_item[c] not in fi:
                    fi.append(comp_item[c])
            links = ["item/" + x for x in fi]
        elif it and it.startswith("F-"):
            if it in reg and e["occ"] >= reg[it]:
                proc = "item/" + it
                if t == "assembly.component_unlinked" and p.get("lot_id"):
                    links = ["lot/" + p["lot_id"]]
            else:
                proc = "lot/" + piece_lot[it]          # маркировка и КТ-1 заготовки до запуска — процесс «Партия»
        elif it and it.startswith(("R-", "C-")):
            proc = "lot/" + piece_lot[it]              # компонент на складе — процесс «Партия»
        elif it:
            proc = None                                # CS-07: контрольный образец — разбор причин (волна 2)
        elif e["lot"]:
            proc = "lot/" + e["lot"]
        e["proc"] = proc
        e["proc_links"] = links


# шаги людей: экземпляр, чью задачу закрывает шаг (правило + уточнения по меткам шагов)
STEP_PROCESS_OVERRIDE = {
    "hold/WS-2": ("incident/RS-01", []),                                          # I3 — остановка источника
    "S02/recheck-rings": ("incident/RS-02", ["lot/LOT-R-117"]),                   # I5 — кольца на складе в круге
    # схема v0.3: «Разбор причин» (CA0…CAZ2) — отдельный экземпляр
    "S05/cause": ("cause/RS-01", ["incident/RS-01", "nc/NC-01", "nc/NC-02", "nc/NC-03", "nc/NC-G1"]),
    "S02/cause": ("cause/RS-02", ["incident/RS-02", "nc/NC-04", "nc/NC-05"]),
    "S02/capa": ("cause/RS-02", ["nc/NC-04", "nc/NC-05"]),
}
CAUSE_ACTIONS = {"cause.confirmed", "cause.explanation_recorded", "capa.action_assigned", "capa.action_implemented",
                 "capa.effectiveness_confirmed", "cause.investigation_closed"}
STEP_EXTRA_LINKS = {
    "S03/confirm-NC-01": ["incident/RS-01"],       # несоответствие вне ЗТ → основание инцидента (I0)
    "S02/confirm-NC-04": ["incident/RS-02"],
    "S02/confirm-NC-05": ["incident/RS-02"],
    "S05/confirm-NC-02": ["incident/RS-01"],       # найдено проверкой в круге (I5)
    "S05/confirm-NC-03": ["incident/RS-01"],
    "S05/disposition-rework": ["incident/RS-01"],  # Н вызван из I6
    "S02/return": ["incident/RS-02"],
    "S14/confirm-NC-142": ["incident/RS-14"],      # с НС-142 общий фактор — сварщик: основание инцидента RS-14
    "S14/confirm-NC-143": ["incident/RS-14"],
    "S14/disposition-rework": ["incident/RS-14"],  # Н вызван из I6
}
STEP_NO_PROCESS = {"item.binding_corrected", "erp.id_map_added", "integrity.check_requested", "event.corrected",
                   "cause.confirmed", "escalation.acknowledged", "post.assignment_requested", "post.assignment_changed",
                   "source.clock_synced", "integrity.violation_resolved"}


def _obj_key(x):
    if x.startswith("F-"):
        return "item/" + x
    if x.startswith("LOT-"):
        return "lot/" + x
    return None


def step_process(s, piece_lot):
    subj, act, lab = s["subject"], s["action"], s["label"]
    if lab in STEP_PROCESS_OVERRIDE:
        proc, links = STEP_PROCESS_OVERRIDE[lab]
        return proc, list(links)
    if act in CAUSE_ACTIONS and lab.startswith("S14/"):
        return "cause/RS-14", ["incident/RS-14"] + ["nc/" + x for x in subj.get("nc_refs") or []]
    if act in STEP_NO_PROCESS:
        return None, []
    if act in ("kd.wip_decision", "permit.approved") and subj.get("order_id"):
        # схема v0.3.2: решение по заделу (OC3) и разрешение (OC4) — в экземпляре «Заказ», изделие — связь
        return "order/" + subj["order_id"], ["item/" + subj["item_id"]] if subj.get("item_id") else []
    it = subj.get("item_id")
    objs = ([it] if it else []) + list(subj.get("items") or [])

    def obj(x):
        if x.startswith(("R-", "C-")):
            return "lot/" + piece_lot[x]
        return _obj_key(x)
    nc = subj.get("nc_ref") or (subj.get("nc_refs") or [None])[0]
    links = []
    if act == "decision.review_completed":
        proc = "review/%s/%s" % (subj["gate"], it)
        links = ["item/" + it, "incident/RS-01"]
    elif act in ("decision.signal_confirmed", "decision.signal_rejected") and subj.get("signal_basis_labels") \
            and not subj.get("items"):
        # схема v0.3: «Сигнал» — один экземпляр на сигнал (ключ — первое основание)
        proc = "signal/SG-" + slug(subj["signal_basis_labels"][0])
        links = [obj(x) for x in objs]
        if nc:
            links.append("nc/" + nc)
    elif subj.get("risk_scope_ref"):
        proc = "incident/" + subj["risk_scope_ref"]
        links = [obj(x) for x in objs]
        if nc:
            links.append("nc/" + nc)
    elif act == "decision.disposition_set":
        proc = "nc/" + nc
        links = [obj(x) for x in objs]
        if subj.get("lot_id"):
            links.append("lot/" + subj["lot_id"])
    elif act == "decision.gate_passed" and subj.get("lot_id"):
        proc = "lot/" + subj["lot_id"]
    elif objs:
        proc = obj(objs[0])
        links = [obj(x) for x in objs[1:]]
        if nc:
            links.append("nc/" + nc)
    elif subj.get("lot_id"):
        proc = "lot/" + subj["lot_id"]
    else:
        proc = None
    links += STEP_EXTRA_LINKS.get(lab, [])
    out = []
    for x in links:
        if x and x != proc and x not in out:
            out.append(x)
    return proc, out


def piece_lots(g):
    m = {}
    for e in g.events:
        p = e["payload"] if isinstance(e["payload"], dict) else {}
        if e["type"] in ("item.marked", "inspection.result") and e["item"] and p.get("lot_id"):
            m.setdefault(e["item"], p["lot_id"])
    return m


# ----------------------------------------------------------------------------- что уходит в 1С (вывод, не поток)
ERP_REWORK_RETURN = "rework_return_to_production"   # [П] «возврат из брака в производство»


def erp_outbox(g):
    """Ожидаемые учётные сообщения в 1С и ответы stand-а — выводятся из фактов и шагов людей прогона.
    В потоки не входят (их отправляет движок соседей); нужны для I1/табло и для check.py.
    Правила (scenarios-flange.md §6): принято в работу — запуск изделия и выдача комплектующих;
    перемещение — приёмка цехом-получателем; перевод в брак (переделка) — решение ЗТ-Р «переделка»,
    один раз, пока изделие числится в браке; возврат из брака в производство [П] — повторная ЗТ после
    удачной переделки; перевод в брак (списание); возврат поставщику; выпуск годного — только item.released
    после ЗТ-6. Блокировки, сигналы, круг, гипотезы — не уходят."""
    cand = []
    for e in g.events:
        if e.get("tele"):
            continue
        cand.append((e["occ"], e["i"], "ev", e))
    for s in g.steps:
        if s["expect"] != "accepted":
            continue
        cand.append((s["at"], s["i"], "step", s))
    cand.sort(key=lambda x: (x[0], x[1]))
    rows = []
    in_defect = set()
    reworked = set()
    kits = {}
    for (t, _, kind, o) in cand:
        row = None
        if kind == "ev":
            p = o["payload"] if isinstance(o["payload"], dict) else {}
            if o["type"] == "item.registered" and (o["item"] or "").startswith("F-"):
                row = OrderedDict([("type", "issue_to_production"), ("item_id", o["item"]), ("lot_id", p.get("batch_id")),
                                   ("what", "заготовка фланца")])
            elif o["type"] == "batch.issued" and (p.get("component_ids") or p.get("for_items")):
                items = tuple(p.get("for_items") or [])
                if p.get("to_station") == "ST-AC-ASM" and items:
                    k = kits.get(items)
                    if k and (t - k[0]) <= S(60):
                        k[1]["lots"].append(p["lot_id"])
                        continue
                    row = OrderedDict([("type", "issue_to_production"), ("items", list(items)),
                                       ("lots", [p["lot_id"]]), ("what", "комплект на сборку")])
                    kits[items] = (t, row)
                else:
                    row = OrderedDict([("type", "issue_to_production"), ("lot_id", p["lot_id"])])
                    if items:
                        row["items"] = list(items)
                    if p.get("component_ids"):
                        row["components"] = p["component_ids"]
                    row["what"] = "кольца в сварку"
            elif o["type"] == "movement.received" and p.get("from_shop") != p.get("to_shop"):
                row = OrderedDict([("type", "transfer"), ("item_id", o["item"]),
                                   ("route", "%s->%s" % (p["from_shop"], p["to_shop"]))])
            elif o["type"] == "item.released":
                row = OrderedDict([("type", "release_good"), ("item_id", o["item"]),
                                   ("after_rework", o["item"] in reworked), ("concession_no", p.get("concession_no"))])
            if row is not None:
                row["basis"] = o["label"]
        else:
            subj, par = o["subject"], o["params"]
            items = list(subj.get("items") or ([subj["item_id"]] if subj.get("item_id") else []))
            if o["action"] == "decision.disposition_set":
                d = par.get("disposition")
                if d == "rework":
                    new = [x for x in items if x not in in_defect]
                    if new:
                        row = OrderedDict([("type", "scrap_transfer_rework"), ("items", new)])
                        in_defect |= set(new)
                elif d == "scrap":
                    row = OrderedDict([("type", "scrap_transfer_writeoff"), ("items", items)])
                    in_defect -= set(items)
                elif d == "return_to_supplier":
                    row = OrderedDict([("type", "return_to_supplier"), ("lot_id", subj.get("lot_id")),
                                       ("qty", par.get("qty")), ("items", items)])
            elif o["action"] == "kd.wip_decision" and par.get("decision") == "scrap":
                row = OrderedDict([("type", "scrap_transfer_writeoff"), ("items", items),
                                   ("why", "задел при смене ревизии КД (S19)")])   # доработка и «по разрешению» в 1С не уходят
            elif o["action"] == "decision.gate_passed" and par.get("presentation_no") == 2 \
                    and subj.get("item_id") in in_defect:
                it = subj["item_id"]
                row = OrderedDict([("type", ERP_REWORK_RETURN), ("item_id", it),
                                   ("gate", "%s#2" % subj["gate"]), ("what", "переделка выполнена и проверена")])
                in_defect.discard(it)
                reworked.add(it)
            if row is not None:
                row["basis"] = o["label"]
        if row is None:
            continue
        row["at_msk"] = mskstr(t)
        resp = "acked"
        for f in g.stand_faults:
            m = f["match"]
            if m.get("posting_type") == row["type"] and m.get("item_id", row.get("item_id")) == row.get("item_id") \
                    and m.get("lot_id", row.get("lot_id")) == row.get("lot_id"):
                st = f["response"]["http_status"]
                if st >= 500:
                    resp = [OrderedDict([("attempt", 1), ("http_status", st), ("result", "error_retryable")]),
                            OrderedDict([("attempt", 2), ("same_message_id", True), ("result", "acked"),
                                         ("at_msk", iso_to_msk(f["retry_ack_at"]))])]
                else:
                    resp = [OrderedDict([("attempt", 1), ("http_status", st), ("result", "error_final"),
                                         ("then", "без автоповтора; задача ADM-01")]),
                            OrderedDict([("attempt", 2), ("same_message_id", True), ("after_step", f["active_until_step"]),
                                         ("result", "acked"), ("at_msk", iso_to_msk(f["retry_ack_at"]))])]
        row["response"] = resp
        rows.append(row)
    out = []
    for k, r in enumerate(rows):
        o = OrderedDict([("msg_no", k + 1), ("at_msk", r.pop("at_msk")), ("type", r.pop("type"))])
        o.update(r)
        out.append(o)
    return out


def iso_to_msk(s_):
    return mskstr(dt.datetime.strptime(s_, "%Y-%m-%dT%H:%M:%S.%fZ").replace(tzinfo=UTC))


def outbox_counts(rows):
    c = defaultdict(int)
    for r in rows:
        c[r["type"]] += 1
    return OrderedDict(sorted(c.items()))


# ----------------------------------------------------------------------------- плоскость после сварки (v0.3.1, узел WP)
FLAT_TOL = 0.05   # мм, плоскость уплотнительной поверхности [ПП]


def add_flatness(g):
    """Узел WP схемы v0.3.1: после рентгена шва и до ЗТ-3 — плоскость уплотнительной поверхности (плита со щупом [ПП]).
    Измеряет мастер сварочного цеха (М-02): это КТ, не решение. В главной истории и в прогонах все фланцы в допуске.
    Время — через 5 мин после первого рентгена шва после сварки, но не позже чем за 2 мин до ближайшей ЗТ-3."""
    welds = sorted([e for e in g.events if e["type"] == "operation.finished" and (e["item"] or "").startswith("F-")
                    and str(e["payload"].get("operation_run_id", "")).startswith("SV-") and e["payload"].get("result") == "completed"],
                   key=lambda e: e["occ"])
    xr = defaultdict(list)
    for e in g.events:
        p_ = e["payload"] if isinstance(e["payload"], dict) else {}
        if e["type"] == "inspection.result" and p_.get("method") == "xray" and p_.get("checkpoint_id") == "CP-WELD" \
                and (e["item"] or "").startswith("F-"):
            xr[e["item"]].append(e)
    zt3 = defaultdict(list)
    for st_ in g.steps:
        if st_["action"] == "decision.gate_passed" and st_["subject"].get("gate") == "ZT-3" and st_["subject"].get("item_id"):
            zt3[st_["subject"]["item_id"]].append(st_)
    runs_by_item = defaultdict(list)
    for w in welds:
        runs_by_item[w["item"]].append(w)
    for it, rl in runs_by_item.items():
        for k, w in enumerate(rl):
            nxt = rl[k + 1]["occ"] if k + 1 < len(rl) else None
            xs = sorted([x for x in xr[it] if x["occ"] > w["occ"] and (nxt is None or x["occ"] < nxt)], key=lambda x: x["occ"])
            if not xs:
                continue
            t = xs[0]["occ"] + M(5)
            gates = sorted([z for z in zt3[it] if z["at"] > xs[0]["occ"]], key=lambda z: z["at"])
            if gates and gates[0]["at"] - M(2) < t:
                t = gates[0]["at"] - M(2)
            run = w["payload"]["operation_run_id"]
            r = rnd("flat|" + run)
            val = round(r.uniform(0.010, 0.032), 3)
            lab = "flat/" + run
            e = g.ev(t, "TERM-WC", "inspection.result", item=it,
                     payload=OrderedDict([("checkpoint_id", "CP-WELD"), ("phase", "after_operation"), ("method", "flatness"),
                                          ("operation_run_id", run), ("tool", "поверочная плита и щуп [ПП]"),
                                          ("tool_calibration_id", "CAL-PLATE-WC-2026-08"), ("calibration_valid", True),
                                          ("characteristic", OrderedDict([("char_id", "F2"), ("zone", "SEAL-FACE"),
                                                                          ("nominal", 0.0), ("tol_plus", FLAT_TOL),
                                                                          ("actual", val), ("unit", "mm"),
                                                                          ("result", "in_tolerance")])),
                                          ("inspection_result", "no_defect_found"), ("operator_id", "MST-02")]),
                     tags=set(xs[0]["tags"]) & {"S01", "S10A", "S10B", "S14", "S13", "S18", "S19", "S20", "S21"}, label=lab, persona="MST-02", hints=[it])
            for z in gates[:1]:
                bl = z["params"].setdefault("basis_labels", [])
                if lab not in bl:
                    bl.append(lab)


# ----------------------------------------------------------------------------- MES: «не выдавать» и «можно выдавать» (вывод)
RELEASE_DISPOSITIONS = {"rework", "repair", "use_as_is", "replace_component", "component_replacement"}
RELEASE_GATE_DECISIONS = {"accept", "accept_with_concession", "return_for_rework", "accept_incomplete_data"}


def mes_outbox(g, by_label):
    """Ожидаемые сообщения в MES (процесс «Удержание» схемы v0.3.1) — вывод движка, в потоки не входит.
    «Не выдавать в работу» — в момент удержания, если по объекту в MES ещё «можно»: удержание по правилу при сигнале
    (сигнал = первое наблюдение-основание шага подтверждения или отклонения + 1 мин), круг инцидента, отзыв приёмки.
    «Можно выдавать» — только по шагу человека и только когда у объекта не осталось оснований: отклонение сигнала,
    исключение из круга, решение на ЗТ (годно, по разрешению, доработка), решение по несоответствию (переделка, ремонт,
    как есть). Списано и возвращено — удержание остаётся. Гипотез, причин, кадров в сообщениях нет."""
    ev = []
    for s_ in g.steps:
        if s_["expect"] != "accepted" or s_["optional"]:
            continue
        subj = s_["subject"]
        if s_["action"] in ("decision.signal_confirmed", "decision.signal_rejected") and subj.get("signal_basis_labels"):
            items = [subj["item_id"]] if subj.get("item_id") else list(subj.get("items") or [])
            occ = min(by_label[b]["occ"] for b in subj["signal_basis_labels"])
            sig = "signal:" + subj["signal_basis_labels"][0]
            for it in items:
                ev.append((occ + M(1), 0, "hold", it, sig, "SG2", None))
    for h in g.engine_holds:
        for it in h["targets"]:
            ev.append((h["at"], 0, "hold", it, h["basis"], h["node"], None))
    for s_ in g.steps:
        if s_["expect"] != "accepted" or s_["optional"]:
            continue
        subj, par, act = s_["subject"], s_["params"], s_["action"]
        items = [subj["item_id"]] if subj.get("item_id") else list(subj.get("items") or [])
        if act == "decision.signal_rejected":
            for it in items:
                ev.append((s_["at"], 1, "release", it, "signal:" + subj["signal_basis_labels"][0], s_, None))
        elif act == "risk_scope.narrowed":
            for it in par.get("excluded_items") or []:
                ev.append((s_["at"], 1, "release", it, "risk_scope:" + subj["risk_scope_ref"], s_, None))
        elif act == "risk_scope.item_assessed" and par.get("result") == "excluded":
            ev.append((s_["at"], 1, "release", subj["item_id"], "risk_scope:" + subj["risk_scope_ref"], s_, None))
        elif act == "decision.disposition_set":
            d = par.get("disposition")
            tg = items + ([subj["lot_id"]] if subj.get("lot_id") else [])
            for it in tg:
                ev.append((s_["at"], 1, "release" if d in RELEASE_DISPOSITIONS else "stays", it, "*", s_, d))
        elif act == "kd.wip_decision":
            # S19: задел придержан по извещению; снимает решение «доработать» или подписанное разрешение; «списать» — остаётся
            d = par.get("decision")
            if d == "rework_to_new_revision":
                ev.append((s_["at"], 1, "release", subj["item_id"], "kd_change:" + subj["change_notice"], s_, d))
            elif d == "scrap":
                ev.append((s_["at"], 1, "stays", subj["item_id"], "*", s_, d))
        elif act == "permit.approved" and subj.get("item_id"):
            ev.append((s_["at"], 1, "release", subj["item_id"], "kd_change:*", s_, "permit"))
        elif act == "decision.gate_passed" and subj.get("item_id") and par.get("decision") in RELEASE_GATE_DECISIONS:
            ev.append((s_["at"], 1, "release", subj["item_id"], "signal:*", s_, par.get("decision")))
    ev.sort(key=lambda x: (x[0], x[1]))
    bases = defaultdict(set)
    gone = set()
    rows = []
    for (t, _, kind, obj, basis, who, extra) in ev:
        if obj in gone or not obj.startswith(("F-", "LOT-")):
            continue   # кольцо на складе (R-105) держит удержание партии LOT-R-117; в MES — партия
        if kind == "hold":
            was = bool(bases[obj])
            bases[obj].add(basis)
            if not was:
                rows.append(OrderedDict([("at_msk", mskstr(t)), ("type", "hold"),
                                         ("item_id" if obj.startswith("F-") else "lot_id", obj),
                                         ("text", "не выдавать в работу"), ("basis", basis), ("node", who),
                                         ("set_by", "system")]))
            continue
        if kind == "stays":
            gone.add(obj)
            continue
        before = set(bases[obj])
        if not before:
            continue
        if basis == "*":
            bases[obj] = set()
        elif basis == "signal:*":
            bases[obj] = {b for b in bases[obj] if not b.startswith("signal:")}
        elif basis == "kd_change:*":
            bases[obj] = {b for b in bases[obj] if not b.startswith("kd_change:")}
        else:
            bases[obj].discard(basis)
        if before and not bases[obj]:
            rows.append(OrderedDict([("at_msk", mskstr(t)), ("type", "release"),
                                     ("item_id" if obj.startswith("F-") else "lot_id", obj),
                                     ("text", "можно выдавать"), ("released_bases", sorted(before)),
                                     ("decision_step", who["label"]), ("decided_by", who["persona"]), ("action", who["action"])]))
    out = []
    for k, r in enumerate(rows):
        o = OrderedDict([("msg_no", k + 1)])
        o.update(r)
        out.append(o)
    held_end = sorted(o for o, b in bases.items() if b)
    return out, held_end


# ----------------------------------------------------------------------------- сборка потока
def finalize(g, renumber_from=None):
    """Потери, source_seq, время доставки, дубли, gen_no, event_id. Возвращает (строки, сведения)."""
    info = dict(lost=[], dups=0, batches=[], p2=None)
    # потери
    for (src, a, b, expect, note) in g.lost_windows:
        lost = [e for e in g.events if e["src"] == src and a < e["occ"] < b]
        assert len(lost) == expect, ("lost", len(lost), expect)
        for e in lost:
            e["lost"] = True
    # source_seq по источнику в порядке возникновения
    by_src = defaultdict(list)
    for e in g.events:
        by_src[e["src"]].append(e)
    for src, lst in by_src.items():
        lst.sort(key=lambda e: (e["occ"], e["i"]))
        for k, e in enumerate(lst):
            e["seq"] = k + 1
    for (src, a, b, expect, note) in g.lost_windows:
        seqs = sorted(e["seq"] for e in g.events if e["src"] == src and e["lost"])
        info["lost"].append(OrderedDict([("source_id", src), ("source_seq_from", seqs[0]), ("source_seq_to", seqs[-1]),
                                         ("count", len(seqs)), ("occurred_between", [iso(a), iso(b)]), ("note", note)]))
    # время доставки
    for e in g.events:
        e["deliver"] = e["occ"] + S(SOURCES[e["src"]][2])
    lines = [e for e in g.events if not e["lost"]]
    dups = []
    for b in g.batches:
        buf = sorted([e for e in lines if e["src"] == b["src"] and b["frm"] <= e["occ"] <= b["to"]],
                     key=lambda e: e["seq"])
        assert len(buf) == b["expect_unique"], ("batch", len(buf), b["expect_unique"])
        order = buf[:b["dup_first"]] + ["DUP%d" % k for k in range(b["dup_first"])] + buf[b["dup_first"]:]
        key_idx = [k for k, x in enumerate(order) if not isinstance(x, str) and x["label"] == b["key_label"]][0]
        step_ms = (b["key_deliver"] - b["deliver_start"]).total_seconds() * 1000.0 / key_idx
        for k, x in enumerate(order):
            if k <= key_idx:
                tdel = b["deliver_start"] + dt.timedelta(milliseconds=round(step_ms * k))
            else:
                tdel = b["key_deliver"] + dt.timedelta(milliseconds=20 * (k - key_idx))
            if isinstance(x, str):
                orig = buf[int(x[3:])]
                d = copy.copy(orig)
                d["dup_of"] = orig
                d["deliver"] = tdel
                d["noise"] = "duplicate"
                d["tags"] = set(orig["tags"])
                dups.append(d)
            else:
                x["deliver"] = tdel
                x["noise"] = x["noise"] or "late"
        info["batches"].append(OrderedDict([("source_id", b["src"]), ("occurred_from", iso(b["frm"])),
                                            ("occurred_to", iso(b["to"])), ("delivered_from", iso(b["deliver_start"])),
                                            ("delivered_to", iso(max([d["deliver"] for d in dups] + [x["deliver"] for x in buf]))),
                                            ("lines", len(order)), ("unique", len(buf)), ("duplicates", b["dup_first"]),
                                            ("key_label", b["key_label"]), ("key_received_at", iso(b["key_deliver"])),
                                            ("note", b["note"])]))
    # дубль камеры (S06): тот же номер события пришёл второй раз в 11:05:40
    for e in lines:
        if e["label"] == "kt3/SV-017-1/CAM-WS-1" and g.stream_id == "main-story":
            d = copy.copy(e)
            d["dup_of"] = e
            d["deliver"] = T("2026-09-23 11:05:40")
            d["noise"] = "duplicate"
            d["tags"] = set(e["tags"]) | {"S06"}
            dups.append(d)
    info["dups"] = len(dups)
    allx = lines + dups
    allx.sort(key=lambda e: (e["deliver"], e["occ"], e["src"], e["seq"], 1 if e.get("dup_of") else 0))
    gen = 0
    for e in allx:
        if e.get("dup_of") is None:
            gen += 1
            e["gen_no"] = gen
    for e in allx:
        if e.get("dup_of") is not None:
            e["gen_no"] = e["dup_of"]["gen_no"]
        e["event_id"] = str(uuid.uuid5(NS_TEMPLATE, "%s:%d:%d" % (g.stream_id, g.seed, e["gen_no"])))
    return allx, info


# ----------------------------------------------------------------------------- материалы (evidence_refs) — справочник media.yaml
# Правило (кейс §5.4): никаких фиктивных снимков изделия. У кадра камеры — «материал не предоставлен»; если класс дефекта
# есть в открытом наборе — ещё и ИЛЛЮСТРАЦИЯ класса с источником и лицензией (не относится к изделию).
# Рентген — заключение как документ (снимок в центр не передаётся). OperatorVision — только событие, видео не передаётся.
MEDIA_NOT_PROVIDED = "MED-NP-CAMERA-FRAME"
MEDIA_ILLUSTRATION = {"burn_through": "MED-ILL-TIG5083-BURN-THROUGH"}   # класс → иллюстрация из TIG Aluminium 5083
MEDIA_ILL_CAMERAS = {"CAM-WS-1", "CAM-WS-2"}                             # набор снят на сварке — только камеры КТ-3
MEDIA_XR_DOC = "MED-DOC-XR-CONCLUSION"
MEDIA_OV = "MED-OV-EVENT-ONLY"


def with_media(e):
    p = e["payload"]
    if not isinstance(p, dict):
        return p
    if e["type"] == "inspection.result" and p.get("method") == "camera":
        refs = [OrderedDict([("media_id", MEDIA_NOT_PROVIDED), ("kind", "not_provided")])]
        if e["src"] in MEDIA_ILL_CAMERAS:
            for d in p.get("defects") or []:
                m = MEDIA_ILLUSTRATION.get(d.get("defect_type"))
                if m and all(r["media_id"] != m for r in refs):
                    refs.append(OrderedDict([("media_id", m), ("kind", "illustration"), ("defect_type", d["defect_type"]),
                                             ("relates_to_item", False),
                                             ("notice", "ИЛЛЮСТРАЦИЯ, не относится к изделию")]))
        q = OrderedDict(p)
        q["evidence_refs"] = refs
        if e.get("prior_ref") is not None:        # S04: пересъёмка ссылается на прежнее наблюдение (event_id)
            q["prior_observation_event_id"] = e["prior_ref"]["event_id"]
        return q
    if e["type"] == "inspection.result" and p.get("method") == "xray":
        q = OrderedDict(p)
        q["evidence_refs"] = [OrderedDict([("media_id", MEDIA_XR_DOC), ("kind", "document"),
                                           ("document_no", p.get("conclusion_no") or p.get("sample_conclusion_no")),
                                           ("image_transferred", False)])]
        return q
    if e["type"] == "operator_action.detected":
        q = OrderedDict(p)
        q["evidence_refs"] = [OrderedDict([("media_id", MEDIA_OV), ("kind", "event_only"), ("video_transferred", False)])]
        return q
    return p


# ----------------------------------------------------------------------------- основания решений людей (плитка «в»)
# Вывод = подтверждение или отклонение сигнала, решение на ЗТ, изоляция, остановка источника, сужение, оценка и закрытие
# круга, решение по изделию, проверка исполнения, пересмотр приёмки, причина, мера. У каждого — ссылки на записи:
# на строки потока (метки) или на прежние решения (метки шагов), которые сами стоят на строках потока.
CONCLUSION_ACTIONS = ("decision.gate_passed", "decision.signal_confirmed", "decision.signal_rejected", "decision.item_isolated",
                      "process_hold.set", "risk_scope.narrowed", "risk_scope.item_assessed", "risk_scope.closed",
                      "decision.disposition_set", "decision.disposition_verified", "decision.review_completed",
                      "cause.confirmed", "capa.action_assigned", "capa.action_implemented", "capa.effectiveness_confirmed",
                      "cause.investigation_closed", "rework.chief_welder_approved")
BASIS_KEYS = ("basis_labels", "signal_basis_labels", "evidence_labels", "claim_basis_labels", "basis_label",
              "supersedes_label", "original_decision_label", "recalculated_due_to_label")


def step_refs(s):
    out = []

    def walk(x):
        if isinstance(x, dict):
            for k, v in x.items():
                if k in BASIS_KEYS:
                    out.extend(v if isinstance(v, list) else [v])
                elif k == "per_item_basis":
                    for v2 in v.values():
                        out.extend(v2)
                else:
                    walk(v)
        elif isinstance(x, list):
            for v in x:
                walk(v)
    walk(s["subject"])
    walk(s["params"])
    return [r for r in out if r and r != "…"]


def add_basis_refs(g):
    """Ссылки-основания у выводов, где их не было: из тех же записей, на которые опирается решение. В поток не входит."""
    live = [e for e in g.events if not e.get("lost")]          # потерянные у источника в поток не попали
    ev_labels = [e["label"] for e in sorted(live, key=lambda e: e["i"])]
    labset = set(ev_labels)
    ev_by_label = {e["label"]: e for e in live}

    def full(prefix):   # «xr/F-030» → единственная полная метка «xr/F-030/RK-…»
        c = [x for x in ev_labels if x.startswith(prefix + "/")]
        return c[0] if len(c) == 1 else None
    steps = sorted(g.steps, key=lambda s: (s["at"], s["i"]))
    for s in steps:                               # префиксы → полные метки
        for k in ("basis_labels",):
            bl = s["params"].get(k)
            if bl:
                s["params"][k] = [b if (b in labset or b == "…" or not full(b)) else full(b) for b in bl]
    confirm_of = {}
    assess_of = defaultdict(list)
    cause_of = {}
    for s in steps:
        subj = s["subject"]
        if s["action"] == "decision.signal_confirmed" and subj.get("nc_ref") and s["expect"] == "accepted":
            confirm_of[subj["nc_ref"]] = s
        if s["action"] == "risk_scope.item_assessed" and s["expect"] == "accepted":
            assess_of[subj.get("risk_scope_ref")].append(s)
        if s["action"] == "cause.confirmed" and s["expect"] == "accepted":
            for nc in subj.get("nc_refs") or []:
                cause_of[nc] = s

    def ev_before(pred, t):
        return [e["label"] for e in sorted(live, key=lambda e: e["occ"]) if e["occ"] <= t and pred(e)]

    def last_weld_frames(it, t):
        runs = [w["run"] for w in sorted(g.welds, key=lambda w: w["start"]) if w["item"] == it and w["start"] <= t]
        if not runs:
            return []
        return [x for x in ev_labels if x.startswith("kt3/%s/" % runs[-1])][:1]
    def run_summaries(w):     # записи журнала источника за эту сварку: первая и последняя сводка с током
        c = [e["label"] for e in sorted(live, key=lambda e: e["occ"]) if e["src"] == w["src"]
             and e["type"] == "machine.parameters" and w["start"] <= e["occ"] <= w["end"]
             and isinstance(e["payload"], dict) and e["payload"].get("parameters")]
        return c[:1] + c[-1:] if len(c) > 1 else c
    for s in steps:                               # «почему?» у каждой исключённой детали — свои записи
        if s["action"] != "risk_scope.narrowed" or s["expect"] != "accepted":
            continue
        pib = OrderedDict()
        for it in s["params"].get("excluded_items") or []:
            ws = [w for w in sorted(g.welds, key=lambda w: w["start"]) if w["item"] == it and w["start"] <= s["at"]]
            lab = last_weld_frames(it, s["at"])
            if ws:
                lab = lab + run_summaries(ws[-1])
            pib[it] = lab
        s["params"]["per_item_basis"] = pib
    for s in steps:
        if s["action"] not in CONCLUSION_ACTIONS or s["expect"] != "accepted" or step_refs(s):
            continue
        subj, par, a = s["subject"], s["params"], s["action"]
        refs = []
        ncs = [subj["nc_ref"]] if subj.get("nc_ref") else list(subj.get("nc_refs") or [])
        if a == "decision.item_isolated" or a == "process_hold.set":
            src = confirm_of.get(ncs[0]) if ncs else max((c for c in confirm_of.values() if c["at"] <= s["at"]),
                                                        key=lambda c: c["at"], default=None)
            if src:
                refs = [src["label"]] + list(src["subject"].get("signal_basis_labels") or [])
        elif a == "risk_scope.narrowed":
            for it in par.get("excluded_items") or []:
                refs += last_weld_frames(it, s["at"])
        elif a == "risk_scope.item_assessed":
            it = subj["item_id"]
            warn = [x for x in ev_labels if x.startswith("WS-2/warning/") and ev_by_label[x].get("item") == it]
            key = [x for x in ("EV-WS2-0412",) if x in labset and ev_by_label[x].get("item") == it]
            refs = key + warn
            if not refs:
                refs = ev_before(lambda e: e["item"] == it and (e["label"].startswith("xr/") or e["label"].startswith("kt3/")
                                                               or (e["type"] == "operation.finished" and e["label"].startswith("SV-"))),
                                 s["at"])[-3:]
        elif a == "risk_scope.closed":
            refs = [x["label"] for x in assess_of.get(subj.get("risk_scope_ref"), []) if x["at"] <= s["at"]]
        elif a == "decision.signal_confirmed":            # групповое несоответствие — по оценкам изделий в круге
            refs = [x["label"] for x in assess_of.get(subj.get("risk_scope_ref"), []) if x["at"] <= s["at"]
                    and x["params"].get("result") == "confirmed"]
        elif a == "decision.disposition_set":
            refs = [confirm_of[n]["label"] for n in ncs if n in confirm_of]
        elif a == "cause.confirmed":
            for n in ncs:
                if n in confirm_of:
                    refs += list(confirm_of[n]["subject"].get("signal_basis_labels") or [])
        elif a == "capa.action_assigned":
            refs = sorted({cause_of[n]["label"] for n in ncs if n in cause_of})
        elif a == "decision.disposition_verified":
            it = subj["item_id"]
            run = par.get("rework_run")
            if run:
                refs = [x for x in ev_labels if x.startswith("kt3/%s/" % run)][:1]
                t_run = min((w["start"] for w in g.welds if w["run"] == run), default=s["at"])
                refs += [x for x in ev_before(lambda e: e["item"] == it and e["label"].startswith("xr/"), s["at"])
                         if ev_by_label[x]["occ"] >= t_run]
            else:
                refs = [x["label"] for x in steps if x["action"] == "decision.signal_confirmed" and x["at"] <= s["at"]
                        and x["subject"].get("item_id") == it][-1:]
        elif a in ("cause.investigation_closed", "capa.action_implemented", "capa.effectiveness_confirmed"):
            refs = [x["label"] for x in steps if x["at"] <= s["at"] and x is not s and x["expect"] == "accepted"
                    and x["action"] in ("cause.confirmed", "capa.action_assigned", "capa.action_implemented",
                                        "capa.effectiveness_confirmed") and set(x["subject"].get("nc_refs") or []) & set(ncs)][-2:]
        elif a == "rework.chief_welder_approved":
            refs = ev_before(lambda e: e["item"] == subj.get("item_id") and e["label"].startswith("wl2/"), s["at"])[-1:]
        if refs:
            par["basis_labels"] = refs


def to_line(e, stream_id):
    o = OrderedDict()
    o["event_id"] = e["event_id"]
    o["event_type"] = e["type"]
    o["schema_version"] = e["schema"]
    o["source_id"] = e["src"]
    o["source_seq"] = e["seq"]
    o["source_kind"] = SOURCES[e["src"]][0]
    # S15: occurred_at — время по часам источника, как он его прислал (у КИМ-1 часы спешат); доставка и
    # source_seq считаются по настоящему времени — порядок записей источника не нарушен
    o["occurred_at"] = iso(e["occ"] + e.get("skew", dt.timedelta(0)))
    o["received_at"] = iso(e["deliver"])
    o["recorded_at"] = None
    if e["item"] and not e["drop_item"]:
        o["item_id"] = e["item"]
    if e.get("line"):
        o["line_id"] = e["line"]
    if e["lot"]:
        o["lot_id"] = e["lot"]
    if e["corr"]:
        o["correlation_id"] = e["corr"]
    o["payload"] = with_media(e)
    sim = OrderedDict()
    sim["gen_no"] = e["gen_no"]
    sim["label"] = e["label"]
    sim["scenarios"] = sorted(e["tags"])
    sim["process"] = e.get("proc")
    if e.get("proc_links"):
        sim["process_links"] = e["proc_links"]
    sim["noise"] = e["noise"]
    sim["sign_via"] = SOURCES[e["src"]][1]
    if e["src"] in EDGE:
        sim["edge_agent"] = EDGE[e["src"]]
    if SOURCES[e["src"]][1] == "terminal":
        sim["persona"] = e["persona"] or (e["payload"].get("operator_id") if isinstance(e["payload"], dict) else None)
    if e["after"]:
        sim["after_step"] = e["after"]
    if e["note"]:
        sim["note"] = e["note"]
    o["_sim"] = sim
    return json.dumps(o, ensure_ascii=False, separators=(",", ":"))


# ----------------------------------------------------------------------------- теги срезов по сценариям главной истории
def tag_slices(g):
    WS2_WIN = (T("2026-09-22 09:50"), T("2026-09-23 11:12:30"))
    RS34 = set(["F-%03d" % n for n in range(7, 37)] + ["F-221", "F-222", "F-223", "F-224"])
    for e in g.events:
        h = e["hints"]
        o = e["occ"]
        if e["src"] in ("WS-1", "WS-2"):
            if h & {"F-001"}:
                e["tags"].add("S01")
            if e["src"] == "WS-2" and WS2_WIN[0] <= o <= WS2_WIN[1]:
                e["tags"] |= {"S07", "S06", "S05"}
            if e["src"] == "WS-2" and T("2026-09-22 16:30") <= o <= T("2026-09-22 17:25"):
                e["tags"] |= {"S04", "S02"}
            if e["src"] == "WS-1" and T("2026-09-21 14:50") <= o <= T("2026-09-23 01:00"):
                e["tags"].add("S05")
            if o >= T("2026-09-24 07:00"):
                e["tags"].add("S10A")
            if "CS-07" in h:
                e["tags"].add("S05")
            if e["label"] == "EV-WS2-0412":
                e["tags"] |= {"S09", "S07", "S05", "S06"}
            if e["src"] == "WS-1" and e["schema"] == "9.0":
                e["tags"].add("S12")
        if h & {"F-017"} and o >= T("2026-09-23 08:00") and o < T("2026-09-24 00:00"):
            e["tags"].add("S03")
        if h & {"F-025"} and o >= T("2026-09-23 12:00") and o < T("2026-09-24 00:00"):
            e["tags"].add("S04")
        if h & {"F-019", "F-027", "F-029"} and o >= T("2026-09-22 14:00") and o < T("2026-09-24 00:00"):
            e["tags"].add("S02")
        if h & RS34 and T("2026-09-21 14:50") <= o < T("2026-09-24 00:00") and e["src"] not in ("CAM-IQC", "TERM-STK"):
            e["tags"].add("S05")
        if h & {"F-001", "R-001", "C-001"}:
            e["tags"].add("S01")
        if e["label"] in ("EV-WS2-0412",):
            e["tags"].add("S09")
        if o >= T("2026-09-14 00:00") and (e["src"] == "ERP-1C" or e["type"] in (
                "item.registered", "batch.issued", "movement.received", "item.released")):
            if not (e["lot"] or "").startswith(("LOT-ZF-111", "LOT-R-108", "LOT-C-211", "LOT-SL-391", "LOT-FS-481", "LOT-W-87",
                                                 "LOT-V-591")) \
                    and not any(x.startswith("F-2") and x not in ("F-221", "F-222", "F-223", "F-224") for x in h):
                e["tags"].add("I1")


# ----------------------------------------------------------------------------- вывод
def dump_yaml(path, header, obj):
    with open(path, "w", encoding="utf-8") as f:
        f.write(header.rstrip() + "\n\n")
        yaml.safe_dump(json.loads(json.dumps(obj, ensure_ascii=False)), f, allow_unicode=True, sort_keys=False,
                       width=180, default_flow_style=False)


def steps_out(g, prefix):
    out = []
    pl = piece_lots(g)
    for k, s in enumerate(sorted(g.steps, key=lambda s: (s["at"], s["i"]))):
        o = OrderedDict()
        o["step_id"] = "%s-H%04d" % (prefix, k + 1)
        o["label"] = s["label"]
        o["at"] = iso(s["at"])
        o["at_msk"] = mskstr(s["at"])
        o["mode"] = s["mode"]
        o["role"] = s["role"]
        o["persona"] = s["persona"]
        if s["signers"] != [s["persona"]]:
            o["signers"] = s["signers"]
        o["action"] = s["action"]
        o["catalog_ref"] = s["catalog"]
        o["subject"] = s["subject"]
        proc, links = step_process(s, pl)
        o["process_ref"] = proc
        if links:
            o["process_links"] = links
        o["params"] = s["params"]
        o["expect"] = s["expect"]
        if s["optional"]:
            o["optional"] = True
        o["scenarios"] = sorted(s["tags"])
        if s["note"]:
            o["note"] = s["note"]
        out.append(o)
    return out


def write_stream(path, lines_json):
    with open(path, "w", encoding="utf-8") as f:
        for l in lines_json:
            f.write(l + "\n")


def stats(evs):
    c = defaultdict(int)
    for e in evs:
        c[e["type"]] += 1
    return OrderedDict(sorted(c.items()))


def main():
    out_streams = os.path.join(BASE, "streams")
    out_defs = os.path.join(BASE, "definitions", "scenarios")
    os.makedirs(out_streams, exist_ok=True)  # потоки не хранятся в репозитории — генератор создаёт их заново
    manifest = OrderedDict()
    manifest["meta"] = OrderedDict([
        ("kind", "streams.manifest"), ("version", "0.1-draft"), ("generator", "tools/generate.py"),
        ("event_contract_version", CONTRACT),
        ("event_id_rule_in_files", "UUIDv5(NS_TEMPLATE, '<stream_id>:<seed>:<gen_no>'), NS_TEMPLATE = UUIDv5(URL, 'urn:kosmo-flange:emulator-data:v0')"),
        ("event_id_rule_in_run", "соседи пересчитывают при загрузке: UUIDv5(run_id, seed ‖ gen_no) (AD-38); дубли сохраняют gen_no → тот же event_id"),
        ("times", "UTC, RFC 3339, 3 знака мс; received_at — плановое время доставки по доменным часам сценария; recorded_at — null (ставит ядро)")])
    manifest["streams"] = []

    # ---- главная история
    g = build("main")
    add_flatness(g)
    p2 = add_telemetry(g)
    assign_lines(g)
    assign_process(g)
    tag_slices(g)
    allx, info = finalize(g)
    lines = [to_line(e, g.stream_id) for e in allx]
    write_stream(os.path.join(out_streams, "main-story.jsonl"), lines)
    main_by_label = {e["label"]: e for e in allx if e.get("dup_of") is None}
    add_basis_refs(g)
    steps_main = steps_out(g, "MS1")
    manifest["streams"].append(OrderedDict([
        ("file", "main-story.jsonl"), ("stream_id", "main-story"), ("kind", "run"), ("seed", g.seed),
        ("scenarios", ["S01", "S02", "S03", "S04", "S05", "S06", "S07", "S08", "S09", "S10A", "S11", "S12", "S15", "S16", "S17", "I1"]),
        ("occurred_from", iso(min(e["occ"] for e in allx))), ("occurred_to", iso(max(e["occ"] for e in allx))),
        ("lines", len(allx)), ("unique_events", len(allx) - info["dups"]), ("duplicates", info["dups"]),
        ("lost_not_emitted", sum(x["count"] for x in info["lost"])),
        ("malformed_for_quarantine", sum(1 for e in allx if (e["noise"] or "").startswith("malformed"))),
        ("human_steps", len(steps_main)), ("stop_steps", sum(1 for s in steps_main if s["mode"] == "stop")),
        ("ws2_power_off_tue", mskstr(p2) + " МСК (подобрано под 928 уникальных записей в пачке)"),
        ("gaps", info["lost"]), ("batches", info["batches"]), ("by_type", stats([e for e in allx]))]))

    # ---- срезы главной истории
    slice_ids = ["S01", "S02", "S03", "S04", "S05", "S07", "S08", "S09", "S10A", "S11", "S12", "S15", "S16", "S17", "I1"]
    slice_counts = {}
    for sid in slice_ids:
        sl = [e for e in allx if sid in e["tags"]]
        write_stream(os.path.join(out_streams, sid + ".jsonl"), [to_line(e, g.stream_id) for e in sl])
        slice_counts[sid] = (len(sl), sum(1 for e in sl if e.get("dup_of") is not None))
        manifest["streams"].append(OrderedDict([
            ("file", sid + ".jsonl"), ("stream_id", "main-story"), ("kind", "slice_of_main_story"),
            ("note", "выборка строк main-story.jsonl с _sim.scenarios ∋ %s; те же event_id и source_seq; для чтения, отладки и инъекций цифрового стенда; прогон = main-story" % sid),
            ("lines", len(sl)), ("duplicates", slice_counts[sid][1]), ("by_type", stats(sl))]))

    # ---- S06: инъекция после главной истории (повтор пачки + конфликт номера)
    batch = [e for e in allx if e["src"] == "WS-2" and e["noise"] in ("late", "duplicate") and
             T("2026-09-23 12:00") <= e["deliver"] <= T("2026-09-23 12:10")]
    batch.sort(key=lambda e: e["deliver"])
    s06 = []
    t0 = T("2026-09-24 16:40:00")
    for k, e in enumerate(batch):
        d = copy.copy(e)
        d["deliver"] = t0 + dt.timedelta(milliseconds=20 * k)
        d["noise"] = "duplicate_resend"
        d["tags"] = {"S06"}
        d["note"] = "S06 отдельно: повторная отправка той же пачки ИС-2 после главной истории"
        s06.append(d)
    key = main_by_label["EV-WS2-0412"]
    c = copy.deepcopy({k: v for k, v in key.items() if k not in ("dup_of",)})
    c["payload"] = copy.deepcopy(key["payload"])
    c["payload"]["value"] = 171
    c["deliver"] = t0 + dt.timedelta(milliseconds=20 * len(batch) + 1000)
    c["noise"] = "conflict:E_ID_CONFLICT"
    c["tags"] = {"S06"}
    c["label"] = "EV-WS2-0412"
    c["note"] = "S06 вариант: тот же source_id + event_id + source_seq, другое содержимое (171 А вместо 176 А) → конфликт"
    s06.append(c)
    write_stream(os.path.join(out_streams, "S06.jsonl"), [to_line(e, g.stream_id) for e in s06])
    s06_main_slice = [e for e in allx if "S06" in e["tags"]]
    manifest["streams"].append(OrderedDict([
        ("file", "S06.jsonl"), ("stream_id", "main-story"), ("kind", "injection_after_main_story"),
        ("note", "подавать в тот же прогон после main-story (Чт 24.09 16:40 МСК): 1240 строк пачки ИС-2 повторно + 1 конфликт номера; в самой главной истории S06 = дубль камеры 11:05:40 + 312 повторов в пачке"),
        ("lines", len(s06)), ("duplicates", len(s06) - 1), ("conflicts", 1),
        ("main_story_lines_tagged_S06", len(s06_main_slice))]))

    # ---- S13: самостоятельный прогон
    g13 = build("s13")
    add_flatness(g13)
    add_telemetry(g13)
    assign_lines(g13)
    assign_process(g13)
    for e in g13.events:
        e["tags"].add("S13")
    a13, i13 = finalize(g13)
    write_stream(os.path.join(out_streams, "S13.jsonl"), [to_line(e, g13.stream_id) for e in a13])
    add_basis_refs(g13)
    steps13 = steps_out(g13, "S13")
    manifest["streams"].append(OrderedDict([
        ("file", "S13.jsonl"), ("stream_id", "S13"), ("kind", "run"), ("seed", g13.seed),
        ("note", "самостоятельный прогон: 12 фланцев S13 (история с 07.09 + Чт 24.09 + Пт 25.09); без инцидента RS-01; своя нумерация source_seq"),
        ("occurred_from", iso(min(e["occ"] for e in a13))), ("occurred_to", iso(max(e["occ"] for e in a13))),
        ("lines", len(a13)), ("duplicates", 0), ("human_steps", len(steps13)),
        ("stop_steps", sum(1 for s in steps13 if s["mode"] == "stop")), ("by_type", stats(a13))]))

    # ---- S10B: самостоятельный прогон (+ S22 — продолжение того же прогона, отдельным файлом)
    g10 = build_s10b()
    add_s22(g10)
    add_flatness(g10)
    add_telemetry(g10)
    assign_lines(g10)
    assign_process(g10)
    a10all, i10 = finalize(g10)
    a10 = [e for e in a10all if "S22" not in e["tags"]]
    a22 = [e for e in a10all if "S22" in e["tags"]]
    assert a22 and max(e["deliver"] for e in a10) < min(e["deliver"] for e in a22), "S22 должен идти после S10B"
    write_stream(os.path.join(out_streams, "S10B.jsonl"), [to_line(e, g10.stream_id) for e in a10])
    write_stream(os.path.join(out_streams, "S22.jsonl"), [to_line(e, g10.stream_id) for e in a22])
    add_basis_refs(g10)
    v10, v22 = copy.copy(g10), copy.copy(g10)
    v10.steps = [x for x in g10.steps if "S22" not in x["tags"]]
    v22.steps = [x for x in g10.steps if "S22" in x["tags"]]
    steps10 = steps_out(v10, "S10B")
    steps22 = steps_out(v22, "S22")
    manifest["streams"].append(OrderedDict([
        ("file", "S10B.jsonl"), ("stream_id", "S10B"), ("kind", "run"), ("seed", g10.seed),
        ("note", "самостоятельный прогон F-090 (Пн 28.09): история трёх переделок У4 по ТП + попытка пятого выполнения (четвёртой переделки)"),
        ("occurred_from", iso(min(e["occ"] for e in a10))), ("occurred_to", iso(max(e["occ"] for e in a10))),
        ("lines", len(a10)), ("duplicates", 0), ("human_steps", len(steps10)),
        ("stop_steps", sum(1 for s in steps10 if s["mode"] == "stop")), ("by_type", stats(a10))]))

    # ---- S14: самостоятельный прогон «общий фактор — один сварщик»
    g14 = build_s14()
    add_flatness(g14)
    add_telemetry(g14)
    assign_lines(g14)
    assign_process(g14)
    a14, i14 = finalize(g14)
    write_stream(os.path.join(out_streams, "S14.jsonl"), [to_line(e, g14.stream_id) for e in a14])
    add_basis_refs(g14)
    steps14 = steps_out(g14, "S14")
    manifest["streams"].append(OrderedDict([
        ("file", "S14.jsonl"), ("stream_id", "S14"), ("kind", "run"), ("seed", g14.seed),
        ("note", "самостоятельный прогон: 12 фланцев ЗП-0929 (Чт 24.09 – Ср 30.09); три подреза у одного сварщика на обоих постах, "
                 "журналы источников в уставке; своя нумерация source_seq"),
        ("occurred_from", iso(min(e["occ"] for e in a14))), ("occurred_to", iso(max(e["occ"] for e in a14))),
        ("lines", len(a14)), ("duplicates", 0), ("human_steps", len(steps14)),
        ("stop_steps", sum(1 for s in steps14 if s["mode"] == "stop")), ("by_type", stats(a14))]))

    manifest["streams"].append(OrderedDict([
        ("file", "S22.jsonl"), ("stream_id", "S10B"), ("kind", "injection_after_run"),
        ("note", "S22: продолжение прогона S10B (Вт 29.09) — подавать в тот же прогон после S10B.jsonl; source_seq и gen_no "
                 "продолжают S10B; попытка вернуть в работу списанный F-090 и макрошлиф У4 (образец)"),
        ("occurred_from", iso(min(e["occ"] for e in a22))), ("occurred_to", iso(max(e["occ"] for e in a22))),
        ("lines", len(a22)), ("duplicates", 0), ("human_steps", len(steps22)),
        ("stop_steps", sum(1 for s in steps22 if s["mode"] == "stop")), ("by_type", stats(a22))]))

    # ---- нестандартные случаи S18–S21: отдельные короткие прогоны
    extra = OrderedDict()
    for sid, builder, note in (
            ("S18", build_s18, "мир S18 (Пт 02.10 – Пн 05.10): Ф-042 взамен списанного Ф-021; у С-03 истекло удостоверение, у ключа "
                               "КЛ-1 — поверка; контроль момента ключом КЛ-2"),
            ("S19", build_s19, "мир S19 (остаток ЗП-0917, Пт 25.09 – Пн 28.09): извещение ИИ-ФЛ-100-07, ревизия «Б» → «В»; "
                               "задел Ф-037…Ф-039, новый запуск Ф-040"),
            ("S20", build_s20, "мир S20 (Вт 29.09 – Ср 30.09): Ф-031…Ф-033 на посту 1; перепутанные фланцы и нечитаемый код"),
            ("S21", build_s21, "мир S21 (Чт 17.09 – Чт 24.09): кольца П-117 в таре П-116; К-107 вварено до блокировки, К-108 "
                               "отклонено при выдаче")):
        gg = builder()
        add_flatness(gg)
        add_telemetry(gg)
        assign_lines(gg)
        assign_process(gg)
        for e in gg.events:
            e["tags"].add(sid)
        aa, _ = finalize(gg)
        write_stream(os.path.join(out_streams, sid + ".jsonl"), [to_line(e, gg.stream_id) for e in aa])
        add_basis_refs(gg)
        st = steps_out(gg, sid)
        extra[sid] = (gg, st)
        manifest["streams"].append(OrderedDict([
            ("file", sid + ".jsonl"), ("stream_id", sid), ("kind", "run"), ("seed", gg.seed),
            ("note", "самостоятельный короткий прогон — " + note + "; своя нумерация source_seq"),
            ("occurred_from", iso(min(e["occ"] for e in aa))), ("occurred_to", iso(max(e["occ"] for e in aa))),
            ("lines", len(aa)), ("duplicates", 0), ("human_steps", len(st)),
            ("stop_steps", sum(1 for s in st if s["mode"] == "stop")), ("by_type", stats(aa))]))

    dump_yaml(os.path.join(out_streams, "_manifest.yaml"),
              "# Манифест потоков. Генерируется tools/generate.py — руками не править.\n"
              "# Счётчики, намеренные разрывы source_seq (потери), пачки с опозданием и дублями.", manifest)

    write_definitions(out_defs, g, steps_main, g13, steps13, g10, steps10, manifest, slice_counts, len(s06), p2,
                      g14=g14, steps14=steps14, extra=extra, steps22=steps22)
    print("OK: main-story %d строк (%d дублей), S13 %d, S10B %d, S14 %d, S06 %d; шагов людей: MS1 %d, S13 %d, S10B %d, S14 %d; "
          "ИС-2 выкл. Вт %s"
          % (len(allx), info["dups"], len(a13), len(a10), len(a14), len(s06), len(steps_main), len(steps13), len(steps10),
             len(steps14), mskstr(p2)))
    print("    нестандартные случаи: " + ", ".join("%s %d строк / %d шагов" % (k, next(x["lines"] for x in manifest["streams"]
                                                                              if x["file"] == k + ".jsonl"), len(v[1]))
                                              for k, v in extra.items()) + ", S22 %d строк / %d шагов" % (len(a22), len(steps22)))


# ----------------------------------------------------------------------------- определения сценариев
SCEN = OrderedDict([
    ("S01", dict(title="Нормальное изготовление — эталонный фланец Ф-001", kind="slice_of_main_story",
                 goal="Целая история одного фланца без замечаний: все проверки чистые при хорошем качестве, журнал ИС-2 в норме, все шаги подтверждены, все ЗТ подписаны, 1С всё приняла; эталон для сравнения.",
                 case={"s4_2": ["нормальное изготовление"], "s5_1": ["Нормальное производство"], "other": ["§1.5", "§5.2 происхождение времени"]},
                 window=("2026-09-14 11:00", "2026-09-23 10:31"), items=["F-001"],
                 erp=["issue_to_production ×3 (заготовка, кольцо, крышка с уплотнением и крепежом)", "transfer MC→WC", "transfer WC→AC", "release_good (1-я попытка 503 → повтор → acked)"])),
    ("S02", dict(title="Входной брак — пора в металле кольца (партия П-117)", kind="slice_of_main_story",
                 goal="«Где нашли» ≠ «где возникло»: рентген после сварки находит пору в теле кольца — входной брак поставщика; сварщик и источник ни при чём.",
                 case={"s4_2": ["входной дефект"], "s5_1": ["Входной брак"], "other": ["§5.2 входной брак отдельно"]},
                 window=("2026-09-22 08:00", "2026-09-23 16:11"), items=["F-019", "F-027", "F-029", "R-101", "R-102", "R-103", "R-104", "R-105", "R-106"],
                 erp=["issue_to_production (кольца R-101…R-103)", "return_to_supplier ×1 (7 колец; 422 → соответствие ID → acked)"])),
    ("S03", dict(title="Новый дефект после сварки — прожог на Ф-017", kind="slice_of_main_story",
                 goal="Связать дефект с прошлой чистой проверкой и операцией, собрать обстоятельства и честно написать «причина не установлена», пока данных мало; сварщик — обстоятельство, не виновник.",
                 case={"s4_2": ["дефект после операции"], "s5_1": ["Новый дефект", "Решение пользователя (подтверждение)"], "other": ["§2.3", "§5.4"]},
                 window=("2026-09-18 10:10", "2026-09-23 11:20"), items=["F-017"], erp=["ничего"])),
    ("S04", dict(title="Недостаток сведений — блик на Ф-025 и потерянные данные Ф-019", kind="slice_of_main_story",
                 goal="Не делать категоричных выводов при нехватке данных: высокая уверенность на плохом кадре — не «годно»; потерянный кусок журнала — «нет данных», а не «в норме».",
                 case={"s4_2": ["недостаток сведений для вывода"], "s5_1": ["Неопределённость"], "other": ["§4.5", "§4.6"]},
                 window=("2026-09-22 16:35", "2026-09-23 13:25"), items=["F-025", "F-019", "F-021"], erp=["ничего"])),
    ("S05", dict(title="Событие оборудования — ток ИС-2 вне уставки, тающая область 34 → 13 → 6", kind="slice_of_main_story",
                 goal="Главный кадр защиты: круг берётся широко и осторожно, сужается только по доказательствам с кнопкой «почему?»; решение на уровне инцидента; гипотеза про сварщика слабеет по фактам; причина подтверждается контрольным образцом.",
                 case={"s4_2": ["отклонение оборудования"], "s5_1": ["Событие станка"], "other": ["§2.3", "§5.2"]},
                 window=("2026-09-21 14:55", "2026-09-23 16:40"), items=["F-007…F-036", "F-221…F-224", "CS-07"],
                 erp=["scrap_transfer_rework ×1 документ на 6 изделий (acked)", "transfer AC→WC для F-015 (acked)"])),
    ("S06", dict(title="Повторная доставка — дубли не считаются дважды", kind="injection_after_main_story",
                 goal="Повторно пришедшие сообщения и второе наблюдение того же дефекта не увеличивают показатели.",
                 case={"s4_2": ["повторная доставка"], "s5_1": ["Повтор"], "other": ["§4.5", "§5.2"]},
                 window=("2026-09-23 11:05", "2026-09-24 16:41"), items=["F-017"], erp=["ничего"])),
    ("S07", dict(title="Позднее событие — опоздавший журнал ИС-2 пересматривает вывод", kind="slice_of_main_story",
                 goal="Опоздавшие данные встают на своё время, выводы пересчитываются с пометкой, решения людей не переписываются — только помечаются «пересмотрите» (Е-122 [П]); пересмотр — новая запись ОТК (Е-123 [П]). ЗТ-3 по Ф-015 — «годно при неполных данных».",
                 case={"s4_2": ["поздняя доставка"], "s5_1": ["Задержка"], "other": ["§4.6", "§5.2"]},
                 window=("2026-09-22 09:50", "2026-09-23 14:05"), items=["F-015", "F-016", "F-017", "F-021", "F-023", "F-025"], erp=["ничего"])),
    ("S08", dict(title="Решение контролёра — ложное срабатывание отклонено (Ф-002)", kind="slice_of_main_story",
                 goal="Решение человека — новая запись, исходный сигнал сохраняется.",
                 case={"s4_2": ["решение контролёра"], "s5_1": ["Решение пользователя"], "other": ["§4.5", "§5.2"]},
                 window=("2026-09-22 10:30", "2026-09-22 11:40"), items=["F-002"], erp=["issue_to_production (крышка, уплотнение, крепёж)"])),
    ("S09", dict(title="Подмена записи — обнаружена (три атаки)", kind="slice_of_main_story",
                 goal="Запись нельзя тихо переписать: 1) правку записи в обход системы находит проверка звеньев цепочки; "
                      "2) правку с пересчётом цепочки («как администратор БД») — подпись источника и контрольная отметка вне базы; "
                      "3) правку представления — сверка с журналом, представление пересобирается из журнала. Разбор — Чт 07:45.",
                 case={"s4_2": ["контрольное изменение записи"], "s5_1": ["Изменение данных"], "other": ["§3.4"]},
                 window=("2026-09-23 16:30", "2026-09-24 07:45"), items=["F-019", "F-023"], erp=["ничего"])),
    ("S10A", dict(title="Переделка шва — новое выполнение операции; переделка помогает не всем", kind="slice_of_main_story",
                  goal="Переделка — новое выполнение со ссылкой на прежнее; «исполнено» ≠ «проверено»; один фланец не проходит повторный контроль и получает новое решение.",
                  case={"other": ["§4.3 повторная обработка", "§4.6 новый ID повторного выполнения", "§5.2 повторные операции"]},
                  window=("2026-09-24 08:00", "2026-09-24 16:30"), items=["F-015", "F-017", "F-021", "F-023", "F-025"],
                  erp=["rework_return_to_production ×4 [П] «возврат из брака в производство» — на повторной ЗТ-3 (acked)",
                       "scrap_transfer_writeoff ×1 (F-021, acked)",
                       "release_good по переделанным — НЕТ: до ЗТ-6 (сборка, испытание, окончательный контроль) в главной истории они не доходят",
                       "блокировка F-021 и НС-06 в 1С НЕ уходят: в 1С F-021 и так числится в браке (переделка)"])),
    ("S10B", dict(title="Лимит переделок участка шва — отдельный прогон на Ф-090", kind="run",
                  goal="Жёсткий лимит: у участка У4 уже три переделки по ТП; пятое выполнение (четвёртая переделка) заблокировано; доступны только ремонт по разрешению на отклонение или списание.",
                  case={"other": ["§4.3 повторная обработка", "лимит доработок из BPMN"]},
                  window=("2026-09-28 07:30", "2026-09-28 15:30"), items=["F-090"],
                  erp=["issue_to_production (заготовка, кольцо)", "transfer MC→WC",
                       "scrap_transfer_rework ×1 (первое решение о переделке; переделки 2–3 — без нового движения: изделие уже числится в браке)",
                       "scrap_transfer_writeoff ×1 (итог)"])),
    ("S11", dict(title="«Забыли прокладку» — шаг остановлен до брака (Ф-003)", kind="slice_of_main_story",
                 goal="Предупреждение до брака: крышку взяли раньше уплотнения — шаг остановлен; наблюдение за техпроцессом, а не слежка за человеком.",
                 case={"other": ["§4.3 действия: пропуск шага", "OperatorVision: действия и сопоставление с маршрутом"]},
                 window=("2026-09-22 11:40", "2026-09-22 12:40"), items=["F-003"], erp=["issue_to_production (крышка, уплотнение, крепёж)"])),
    ("S12", dict(title="Некорректные сообщения — карантин с кодами", kind="slice_of_main_story",
                 goal="Нарушение контракта: неправильные сообщения не теряются и не искажают данные, лежат в карантине с понятной причиной.",
                 case={"other": ["§4.7 контракты и версии", "версия контракта ≠ версия анализатора"]},
                 window=("2026-09-22 14:00", "2026-09-22 14:35"), items=["F-024", "F-004", "F-026"], erp=["ничего"])),
    ("I1", dict(title="Обмен с 1С: задание, результаты на ЗТ, подтверждения, ошибка и повтор, ID", kind="slice_of_main_story",
                goal="Двусторонний обмен с 1С через stand: входящее задание и партии, исходящие учётные сообщения только на ЗТ, подтверждения, ошибка 503 с повтором, ошибка 422 с исправлением соответствия ID.",
                case={"other": ["§5.3 интеграционные данные", "§3.3 интеграции"]},
                window=("2026-09-14 11:00", "2026-09-24 16:30"), items=["F-001…F-036", "LOT-R-117"],
                erp=["issue_to_production", "transfer", "scrap_transfer_rework", "scrap_transfer_writeoff", "return_to_supplier",
                     "rework_return_to_production [П] (после удачной переделки, на повторной ЗТ-3)",
                     "release_good — только после ЗТ-6 (Ф-001 и фоновый заказ)",
                     "ответы stand 1С: acked / 503 / 422 — см. stand_faults и erp_outbox_expected",
                     "предложение [П, ждёт решения соседей]: control_result «результат контроля» (кейс §3.3) — в outbox не входит"])),
    ("S13", dict(title="«Специалист заснул» — очередь на ЗТ-3 без решения, эскалация, перераспределение", kind="run",
                 goal="На закрывающей точке никто не принимает решения; очередь растёт; система не пропускает детали сама — срабатывают сроки и эскалация с ценой задержки; мастер перераспределяет работу.",
                 case={"other": ["§2.4 ожидание и простой", "§1.2", "§5.2 изделия на ЗТ без подписи (О-9)", "соседи: FR-5, FR-8, FR-19, FR-56, FR-57, FR-81, FR-105, FR-129"]},
                 window=("2026-09-25 08:00", "2026-09-25 13:00"), items=S13_ITEMS,
                 erp=["transfer WC→AC ×10 — каждое после своей подписи ЗТ-3", "issue_to_production (комплекты под F-024, F-030, F-031; новое уплотнение под F-031)"])),
    ("S14", dict(title="Общий фактор — один сварщик: система не ставит «ошибку исполнителя» сама", kind="run",
                 goal="Три подреза у одного сварщика на обоих постах при исправных источниках. Путь схемы v0.3: сигнал → инцидент "
                      "по общему фактору «сварщик» (круг 6 → 3) → групповое решение «переделка» через Н → разбор причин: версия "
                      "«отклонение от техники выполнения» сильная, но «ошибку исполнителя» записывает только комиссия после "
                      "письменного объяснения работника → мера «проверка навыка и инструктаж» → окно наблюдения (8 швов) → закрыт.",
                 case={"other": ["§2.3 обоснованность выводов", "§5.2 подтверждённые ошибки отдельно от гипотез",
                                 "§5.2 сравнение сопоставимых работ", "вопрос жюри «а если виноват человек?» (матрица приёмки, В1)"]},
                 window=("2026-09-24 10:00", "2026-10-01 16:05"), items=S14_ITEMS + ["CS-141", "CS-142", "CS-143"],
                 erp=["issue_to_production (заготовки ×20, кольца R-401…R-420 — две выдачи)", "transfer MC→WC ×20",
                      "scrap_transfer_rework ×1 документ на 3 изделия (F-304, F-306, F-310 — групповое решение по RS-14)",
                      "rework_return_to_production ×3 [П] — на повторной ЗТ-3 в Чт 01.10",
                      "гипотезы, причина, объяснение работника и мера в 1С не уходят"])),
    ("S15", dict(title="Сдвиг часов источника: часы КИМ-1 спешат на 7 минут", kind="slice_of_main_story",
                 goal="Часы источника соврали, а порядок записей (source_seq) сохранён: «соврали часы» ≠ «нарушили порядок». "
                      "Система исправляет время по правилу, помечает запись и не трогает исходное значение; решения ЗТ-2 "
                      "не получают ложную метку «пересмотрите».",
                 case={"other": ["§4.6 расхождения времени источников", "§5.2 происхождение показателя времени"]},
                 window=("2026-09-21 08:25", "2026-09-21 10:15"), items=S15_ITEMS, erp=["ничего сверх обычного"])),
    ("S16", dict(title="Ненадёжная привязка события к выполнению операции", kind="slice_of_main_story",
                 goal="Запись журнала ЧПУ без номера выполнения попала на стык двух выполнений: уровень привязки хранится "
                      "(«неоднозначно», два кандидата), неопределённость мешает только зависящим от неё выводам.",
                 case={"other": ["§4.6 ненадёжная идентификация операции — явно", "§4.3 действия: изменение режима"]},
                 window=("2026-09-18 12:55", "2026-09-18 15:20"), items=S16_ITEMS, erp=["ничего сверх обычного"])),
    ("S17", dict(title="Исполнитель честно отметил пропуск обязательной проверки", kind="slice_of_main_story",
                 goal="Сборщик не смог проверить посадку уплотнения (нет щупа) и сам отметил пропуск: запись в журнале "
                      "критических действий, «нет данных» по проверке, ЗТ-СР закрыта, пока проверку не сделают. "
                      "Это не наказание: честно отметивший не хуже промолчавшего.",
                 case={"other": ["§4.3 действия: пропуски проверки", "§3.2 CriticalActions", "критерий Т3: журнал критических действий операторов"]},
                 window=("2026-09-23 08:00", "2026-09-23 09:12"), items=["F-006"], erp=["ничего сверх обычного"])),
    # ---- нестандартные случаи (решение пользователя 26.09; схема v0.3.2) [П]
    ("S18", dict(title="Истёк допуск на дату операции: удостоверение сварщика и поверка ключа", kind="run",
                 goal="Допуск проверяется на дату операции. а) У С-03 удостоверение истекло 02.10 — старт сварки отклонён "
                      "(предусловие шага, Е-13), запись в журнал критических действий, система предлагает С-02. б) У ключа КЛ-1 "
                      "поверка истекла 02.10 — моменты затяжки «недостоверно» (Е-128), ЗТ-4 на них не подписывается; ОТК-2 "
                      "проверяет моменты поверенным ключом КЛ-2.",
                 case={"other": ["§4.3 условия выполнения операции", "§3.4 права и разграничение", "§3.2 журнал критических действий",
                                 "критерий О1: особенности отрасли — допуск к спецпроцессу, поверка средств измерений"]},
                 window=("2026-10-02 07:30", "2026-10-05 13:15"), items=["F-042"],
                 precondition=("мир S18 [П]: Ф-042 — взамен списанного Ф-021 (запасная заготовка ЗФ-201, его крышка К-021 и клапан "
                               "V-021 не ставились); удостоверение С-03 (НАКС-УСЛ-0003) — до 02.10.2026; поверка ключа КЛ-1 — "
                               "до 02.10.2026 (qualifications.yaml, equipment.yaml); неделя 05.10 — оба сварщика в смене А"),
                 new_events_p=["inspection.result_invalidated (Е-128 [П], реакция) — «недостоверно: средство не поверено на дату»",
                               "operation.precondition_failed (Е-13, реакция) с кодом E_QUALIFICATION_EXPIRED",
                               "отказ E_INSTRUMENT_NOT_VERIFIED [П] — ЗТ-4 на записях непроверенного ключа"],
                 erp=["issue_to_production (заготовка, кольцо, комплект)", "transfer MC→WC, WC→AC",
                      "отказ в сварке, «недостоверно» и повторный контроль в 1С не уходят"])),
    ("S19", dict(title="Изменили КД посреди партии: ревизия «Б» → «В»", kind="run",
                 goal="Извещение ИИ-ФЛ-100-07: новые изделия — по ревизии «В»; задел придержан, по каждому решает человек: "
                      "Ф-039 — доработать до «В», Ф-037 — оставить по «Б» по разрешению на отклонение, Ф-038 — списать. "
                      "Изделие хранит, по какой ревизии сделано. Карта контроля для «В» — ручной контроль, пока нет допуска ИИ. "
                      "В 1С — только «перевод в брак (списание)».",
                 case={"other": ["§4.6 версии: ревизия КД у изделия", "§3.3 КОМПАС-3D и 1С", "AI_ADMISSION: новая ревизия — ручной контроль",
                                 "критерий О1: работа с заделом при изменении КД"]},
                 window=("2026-09-25 07:30", "2026-09-28 16:00"), items=["F-037", "F-038", "F-039", "F-040"],
                 precondition=("мир S19 [П]: остаток заказа ЗП-0917; Ф-037 и Ф-038 сварены по «Б» в Пн 28.09 утром, Ф-039 "
                               "мехобработан по «Б» и ждёт рейса, Ф-040 не запущен; уплотнений старой конструкции УП-401 — одно"),
                 new_events_p=["kd.wip_decision (Е-127 [П], шаг человека) — решение по заделу", "permit.approved (Е-95) — разрешение на отклонение",
                               "удержание задела по извещению (узел OC2, реакция) — в MES «не выдавать» сразу"],
                 erp=["issue_to_production (заготовки ×4, кольца ×2)", "transfer MC→WC ×2",
                      "scrap_transfer_writeoff ×1 (F-038)", "доработка до новой ревизии и «по разрешению» в 1С не уходят"])),
    ("S20", dict(title="Маркировка не читается, детали перепутали", kind="run",
                 goal="Скан на посту: ожидался Ф-031, пришёл Ф-033 — шаг остановлен (Е-13), мастер выясняет, история не "
                      "смешивается. У Ф-032 код не читается — ручной ввод с подтверждением второго человека, потом "
                      "перемаркировка по процедуре; ОТК сверяет на ЗТ-3.",
                 case={"other": ["§4.6 ненадёжная идентификация объекта — явно", "§4.6 уникальные ID и сборка ↔ компоненты",
                                 "§3.2 журнал критических действий"]},
                 window=("2026-09-29 07:30", "2026-09-30 13:05"), items=["F-031", "F-032", "F-033"],
                 precondition=("мир S20 [П]: Ф-031…Ф-033 варят в Ср 30.09 на посту 1 (С-02, смена А); при приёмке во вторник "
                               "Ф-031 и Ф-033 положили в тару друг друга; код Ф-032 поцарапан при перемещении"),
                 new_events_p=["item.scanned (Е-126 [П], поток) — скан на посту: ok / не читается / не то изделие",
                               "operation.precondition_failed (Е-13, реакция) с кодами E_ITEM_MISMATCH, E_ITEM_NOT_IDENTIFIED"],
                 erp=["issue_to_production (заготовки ×3, кольца ×3)", "transfer MC→WC ×3", "стоп, выяснение и перемаркировка в 1С не уходят"])),
    ("S21", dict(title="Смешение партий: кольца П-116 и П-117 (после претензии)", kind="run",
                 goal="Кольца заблокированной партии П-117 оказались в таре П-116. Скан кольца при выдаче ловит «партия "
                      "заблокирована» (К-108), выдача не проходит. Кольцо К-107 выдано раньше блокировки и вварено в Ф-034 — "
                      "генеалогия показывает партию, и после претензии Ф-034 попадает в круг по партии; исключён по рентгену.",
                 case={"other": ["§4.6 сборка ↔ компоненты (генеалогия)", "§4.2 входной дефект: круг по партии",
                                 "§3.2 журнал критических действий", "критерий О1: прослеживаемость партий"]},
                 window=("2026-09-17 07:30", "2026-09-24 12:00"), items=["F-034", "F-035", "F-036", "R-107", "R-108"],
                 precondition=("мир S21 [П]: вторая половина заказа варится позже, у П-116 есть кольца К-031…К-033; при "
                               "маркировке П-117 во вторник кольца К-107 и К-108 положили в тару П-116; претензия по П-117 — "
                               "Ср 23.09 15:45, как в главной истории"),
                 new_events_p=["item.scanned (Е-126 [П]) — скан кольца при выдаче и проверка тары",
                               "operation.precondition_failed (Е-13) с кодом E_LOT_BLOCKED",
                               "основание в инцидент из партии (узел VI схемы v0.3.2): изделия с её кольцами — круг RS-21"],
                 erp=["issue_to_production (заготовки ×3, кольца ×3 — в том числе К-107 партии П-117)", "transfer MC→WC ×3",
                      "return_to_supplier ×1 (9 колец П-117)", "отказ в выдаче К-108 в 1С не уходит"])),
    ("S22", dict(title="Попытка вернуть в работу списанную деталь (Ф-090)", kind="injection_after_run",
                 goal="«Списано» — конечный статус: скан на посту и решение «переделка» по списанному Ф-090 — отказ и запись в "
                      "журнал критических действий. Разрешено только отдельное решение «образец на разрушающий контроль» — "
                      "макрошлиф участка У4, без возврата в маршрут; в 1С ничего нового.",
                 case={"other": ["§4.3 повторная обработка: лимит и конечный статус", "§3.4 права: отказ по правилу",
                                 "§3.2 журнал критических действий"]},
                 window=("2026-09-29 08:20", "2026-09-29 13:00"), items=["F-090"],
                 precondition="Ф-090 списан в прогоне S10B (Пн 28.09 15:30, НС-90); S22.jsonl подаётся после S10B.jsonl в тот же прогон",
                 new_events_p=["item.sample_designated (Е-129 [П], шаг человека) — образец на разрушающий контроль",
                               "отказ E_ITEM_SCRAPPED [П]; Е-13 с тем же кодом на скан поста"],
                 erp=["ничего: изделие уже списано в S10B"])),
])

NEW_EVENTS_P = ["gate.wait_overdue (реакция)", "escalation.acknowledged (шаг человека)", "escalation.closed (реакция)",
                "post.assignment_requested (шаг человека)", "post.assignment_changed (шаг человека)"]


def write_definitions(out_defs, g, steps_main, g13, steps13, g10, steps10, manifest, slice_counts, n_s06, p2,
                      g14=None, steps14=None, extra=None, steps22=None):
    extra = extra or {}
    hdr = ("# =============================================================================\n"
           "# Определение сценария. Генерируется tools/generate.py — руками не править (правьте генератор).\n"
           "# Имена событий, действий и полей — НАШИ (process-events-catalog.md, scenarios-flange*.{md,yaml}).\n"
           "# [П] — имя или значение предварительное (нет в каталоге) / наше допущение; [ПП] — проектное предположение.\n"
           "# Время: at — UTC (RFC 3339, мс); at_msk — то же в МСК для чтения.\n"
           "# human_steps.mode: stop — сценарий ждёт решения на столе роли (интерактивный режим; в автосверке подписывает\n"
           "#   demo-signer); auto — demo-signer подписывает сам в указанное доменное время.\n"
           "# expect: accepted | refused[:код] — ожидаемый ответ системы на шаг (проверка отказов по правам и правилам).\n"
           "# =============================================================================")
    ms = OrderedDict()
    ms["id"] = "MS-1"
    ms["title"] = "Главная история «Плохой день ИС-2»"
    ms["kind"] = "run"
    ms["stream"] = "streams/main-story.jsonl"
    ms["seed"] = g.seed
    ms["clock"] = OrderedDict([("mode", "scenario"), ("start", iso(T("2026-09-07 09:00"))), ("end", iso(T("2026-09-24 16:30"))),
                               ("story_window_msk", ["2026-09-17 08:00", "2026-09-24 16:30"]),
                               ("default_speed", "x600 (минута за 0,1 с) [П]; фон до Пн 21.09 можно проматывать"),
                               ("pause", "останавливает поток и доменные часы")])
    ms["scenarios"] = ["S01", "S02", "S03", "S04", "S05", "S06", "S07", "S08", "S09", "S10A", "S11", "S12", "S15", "S16", "S17", "I1"]
    ms["order_in_story"] = ["S01", "S16", "S15", "S07", "S08", "S11", "S12", "S17", "S03", "S05", "S06", "S04", "S02", "S09",
                            "S10A"]
    ms["expected"] = ["expected/main-story.yaml", "expected/_common.yaml"] + ["expected/%s.yaml" % s for s in ms["scenarios"]]
    ms["stand_faults"] = g.stand_faults
    ms["tamper_actions"] = g.tamper
    ms["ws2_power_off_tue_msk"] = mskstr(p2)
    ms["counts"] = OrderedDict([("stream_lines", manifest["streams"][0]["lines"]),
                                ("unique_events", manifest["streams"][0]["unique_events"]),
                                ("duplicates", manifest["streams"][0]["duplicates"]),
                                ("lost_not_emitted", manifest["streams"][0]["lost_not_emitted"]),
                                ("human_steps", len(steps_main)),
                                ("stop_steps", sum(1 for s in steps_main if s["mode"] == "stop"))])
    ms["reactions_not_in_stream"] = [
        "quality.signal_raised (Е-80)", "quality.observation_linked (Е-81)", "quality.inspection_missing (Е-82)",
        "nc.draft_created (Е-83)", "containment.hold_applied (Е-84)", "analysis.hypotheses_updated (Е-85)",
        "operation.precondition_failed (Е-13)", "rework.limit_reached (Е-46)", "item.inspection_summary (Е-71)",
        "risk_scope.created (Е-110)", "nc.disposition_overdue (Е-97)", "erp.posting_sent (И)",
        "integration.ack / integration.error (Е-75, отвечает stand 1С)",
        "decision.review_flagged (Е-122 [П]): Ср 12:06, метка «пересмотрите» на ЗТ-3 F-015 (узел R1)",
        "сообщение о решении каждому изделию (узел I7, Msg_ItemDisposition): Ср 15:30, шесть изделий RS-01 — "
        "см. шаг S05/disposition-rework → params.item_messages",
        "source.clock_offset_detected (Е-87 [П]) и event.time_corrected (Е-89 [П]): S15, КИМ-1 +7 мин, Пн 08:25…10:05",
        "уровень привязки к выполнению «неоднозначно» (поле, не событие [П]): S16, запись S16/cnc-feed-override",
        "quality.inspection_missing (Е-82) по пропущенной проверке и запись в журнал критических действий: S17, Ср 08:11",
        "integrity.violation_detected (Е-102 [П]) ×3 и projection.rebuilt (Е-104 [П]): S09, Ср 16:31, 16:51, 17:01"]
    ms["line_rule"] = ("line_id в конверте [П]: источники поста (TERM-WS-n, WS-n) — линия своего поста; событие по фланцу "
                       "с момента запуска — линия поста его последней сварки (до первой сварки — по плану запуска); "
                       "1С, КОМПАС, партии и компоненты на складе, общее оборудование без item_id — без линии. "
                       "Справочник — definitions/reference/sites.yaml → lines")
    ms["process_rule"] = ("экземпляры процессов схемы v0.2 [П]: order/<ORD> (старт Е-01), lot/<LOT> (старт Е-03; входной "
                          "контроль, маркировка до запуска, склад, выдача), item/<F> (старт Е-09 — выдача заготовки; всё по "
                          "изделию дальше), incident/<RS>, review/<ЗТ>/<F>, nc/<НС> — только в шагах людей. В строке потока — "
                          "_sim.process и _sim.process_links, в шаге — process_ref и process_links. Это ожидание корреляции, "
                          "не поле контракта. Справочник — definitions/reference/sites.yaml → bpmn_processes")
    ob_main, ob13, ob10 = erp_outbox(g), erp_outbox(g13), erp_outbox(g10)
    ms["erp_outbox_counts"] = outbox_counts(ob_main)
    mes = {}
    for key, gg in (("I1", g), ("S13", g13), ("S10B", g10), ("S14", g14)) + tuple((k, v[0]) for k, v in extra.items()):
        mes[key] = mes_outbox(gg, {e["label"]: e for e in gg.events})
    ms["mes_outbox_counts"] = outbox_counts(mes["I1"][0])
    ms["mes_outbox_ref"] = "definitions/scenarios/I1.yaml → mes_outbox_expected"
    ms["erp_outbox_ref"] = "definitions/scenarios/I1.yaml → erp_outbox_expected"
    ms["human_steps"] = steps_main
    dump_yaml(os.path.join(out_defs, "main-story.yaml"), hdr, ms)
    for sid, meta in SCEN.items():
        d = OrderedDict()
        d["id"] = sid
        d["title"] = meta["title"]
        d["goal"] = meta["goal"]
        d["case"] = meta["case"]
        d["kind"] = meta["kind"]
        if meta["kind"] == "slice_of_main_story":
            d["run"] = "main_story (MS-1)"
            d["stream"] = "streams/main-story.jsonl"
            d["slice"] = "streams/%s.jsonl" % sid
            d["slice_lines"] = slice_counts.get(sid, (0, 0))[0]
            d["seed"] = g.seed
        elif sid == "S06":
            d["run"] = "main_story (MS-1) + инъекция после конца истории"
            d["stream"] = "streams/main-story.jsonl"
            d["injection"] = "streams/S06.jsonl"
            d["injection_lines"] = n_s06
            d["seed"] = g.seed
        elif sid == "S13":
            d["run"] = "standalone"
            d["stream"] = "streams/S13.jsonl"
            d["seed"] = g13.seed
        elif sid == "S10B":
            d["run"] = "standalone"
            d["stream"] = "streams/S10B.jsonl"
            d["seed"] = g10.seed
        elif sid == "S14":
            d["run"] = "standalone"
            d["stream"] = "streams/S14.jsonl"
            d["seed"] = g14.seed
        elif sid in extra:
            d["run"] = "standalone"
            d["stream"] = "streams/%s.jsonl" % sid
            d["seed"] = extra[sid][0].seed
        elif sid == "S22":
            d["run"] = "S10B + продолжение: S22.jsonl подаётся в тот же прогон после S10B.jsonl"
            d["stream"] = "streams/S10B.jsonl"
            d["injection"] = "streams/S22.jsonl"
            d["seed"] = g10.seed
        d["window_msk"] = list(meta["window"])
        d["items"] = meta["items"]
        d["erp_postings_expected"] = meta["erp"]
        d["erp_note"] = ("Сообщения 1С — по словарю scenarios-flange-expected.yaml (erp_posting_type). Полный список с ответами stand-а — "
                         "I1.yaml → erp_outbox_expected (для S13 и S10B — в их файлах). «Выпуск годного» — только после ЗТ-6.")
        d["expected"] = "expected/%s.yaml" % sid
        if sid == "S13":
            d["precondition"] = "Пт 25.09 08:00: на ЗТ-3 ждут F-024, F-030, F-031, F-032; РК по F-033…F-035 выйдут в 09:20, по F-036, F-223, F-224 — в 10:20; в сборке F-028, на стенде LT-1 — F-026; пост ЗТ-3 — QC-01"
            d["clock_note"] = "имитатор не присылает решений QC-01 с 08:15 до 11:30; часы идут — ожидание и есть предмет проверки"
            d["new_events_p"] = NEW_EVENTS_P
            d["optional_rule"] = "R-21 (по желанию): уплотнение под F-031, выданное в 08:10, в 12:10 блокируется; в 12:15 выдано новое"
            d["human_steps"] = steps13
        elif sid == "S10B":
            d["precondition"] = "У4 фланца F-090 уже переделан по ТП 3 раза: SV-090-2, SV-090-3, SV-090-4 (история в том же потоке)"
            d["human_steps"] = steps10
        elif sid in extra or sid == "S22":
            d["precondition"] = meta["precondition"]
            if meta.get("new_events_p"):
                d["new_events_p"] = meta["new_events_p"]
            d["human_steps"] = extra[sid][1] if sid in extra else steps22
        elif sid == "S14":
            d["precondition"] = ("отдельный мир: оба источника исправны, журналы в уставке; 29.09–01.10 — С-03 в смене А "
                                 "(подменяет) [П]; история фланцев с Чт 24.09 — в том же потоке")
            d["new_events_p"] = ["cause.explanation_recorded (П-09 [П], шаг человека)",
                                 "capa.effectiveness_confirmed (П-07), cause.investigation_closed (П-10 [П])",
                                 "отказ E_EXPLANATION_REQUIRED [П] — «ошибку исполнителя» без объяснения работника не записать",
                                 "реакции: risk_scope.created RS-14 (фактор «сварщик»), analysis.check_suggested (Е-79 [П]) — "
                                 "«сначала аппараты»; остановку поста система не предлагает"]
            d["human_steps"] = steps14
        else:
            d["human_steps_ref"] = "definitions/scenarios/main-story.yaml"
            d["human_steps"] = [s for s in steps_main if sid in s["scenarios"]]
        if sid == "S06":
            d["noise"] = ["дубль камеры CAM-WS-1 (тот же event_id) — Ср 23.09 11:05:40",
                          "пачка ИС-2 Ср 12:05: 1240 строк, 312 повторов", "инъекция S06.jsonl: повтор всей пачки + конфликт номера EV-WS2-0412 (171 А)"]
        if sid == "S07":
            d["noise"] = ["пачка ИС-2: occurred_at Вт 09:50 … Ср 11:12, received_at Ср 12:05; ключевая запись EV-WS2-0412 (Вт 10:20:14 → получена Ср 12:05:07)"]
        if sid == "S04":
            d["noise"] = ["разрыв source_seq ИС-2: 35 записей Вт 16:40–17:15 не отправлены (см. streams/_manifest.yaml → gaps)",
                          "кадр F-025 Ср 12:40: inspection_result no_defect_found, confidence 0.91, observation_quality 0.34",
                          "пересъёмка F-025 Ср 12:55: профиль KT3-SIDE «боковой свет» (RCP-CP-WELD@3), У6–У7, ссылка на кадр 12:40 (prior_observation_event_id)",
                          "F-021: кадр камеры 12:35 чистый, прожог в корне У5 — рентген RK-0923-08 12:50; пересъёмки для корня нет"]
        if sid == "S12":
            d["noise"] = ["сообщение 1: CAM-WS-2, F-024, нет item_id → E_MISSING_FIELD",
                          "сообщение 2: WS-1, сводка Вт 14:00, schema_version 9.0 → E_UNSUPPORTED_VERSION",
                          "сообщение 3: CAM-ASM, F-004, inspection_result maybe → E_UNKNOWN_ENUM (затем исправная повторная отправка 14:06)",
                          "сообщение 4: CAM-WS-1, F-024, schema_version 1.1 + lens_temp_c → принято",
                          "сообщение 5: CAM-WS-2, F-026, defect_type spatter_x → принято как неизвестно (spatter_x) с флагом",
                          "сообщение 6: шаг ADM-01 — привязка к F-024 по контексту поста и повторная обработка из карантина"]
        if sid == "S09":
            d["tamper_actions"] = g.tamper
        ob = {"I1": ob_main, "S13": ob13, "S10B": ob10, "S14": erp_outbox(g14),
              "S10A": [r for r in ob_main if r["at_msk"].startswith("2026-09-24")]}.get(sid)
        if sid in extra:
            ob = erp_outbox(extra[sid][0])
        if sid == "S22":
            ob = [r for r in ob10 if r["at_msk"] >= "2026-09-29"]
        if ob is not None:
            d["erp_outbox_note"] = ("Ожидаемые учётные сообщения в 1С и ответы stand-а. Вывод движка — в потоки не входит; "
                                    "выведено генератором из фактов и шагов людей прогона [П]. "
                                    + ("Весь прогон MS-1." if sid == "I1" else ("Только Чт 24.09." if sid == "S10A" else "Весь прогон.")))
            d["erp_outbox_counts"] = outbox_counts(ob)
            d["erp_outbox_expected"] = ob
        if sid in mes:
            rows_, held_ = mes[sid]
            d["mes_outbox_note"] = ("Ожидаемые сообщения в MES (схема v0.3.1, процесс «Удержание»; вывод движка, в потоки не входит) [П]: "
                                    "«не выдавать в работу» — в момент удержания (правило при сигнале, круг, отзыв приёмки), "
                                    "один раз на объект; «можно выдавать» — только по шагу человека, когда оснований не осталось. "
                                    "Гипотез, причин и кадров нет. Кольцо на складе держит удержание партии."
                                    + (" В прогоне удержаний нет." if not rows_ else ""))
            d["mes_outbox_counts"] = outbox_counts(rows_)
            d["mes_held_at_end"] = held_
            d["mes_outbox_expected"] = rows_
        faults = [f for f in g.stand_faults if sid in f.get("scenarios", [])]
        if faults:
            d["stand_faults"] = faults
        dump_yaml(os.path.join(out_defs, sid + ".yaml"), hdr, d)


if __name__ == "__main__":
    main()
