#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Обязательные показатели кейса (О-1…О-11) по данным имитатора — утверждения, добавленные после сверки аналитики
(deliverables/metrics-audit.md, раздел 5, А-1…А-20). Считает по потокам и шагам людей и сверяет с ожиданиями.

  О-1  проверенные изделия (фланцы) за главную историю                                         → S05-56
  О-2  раздельно: производственное / входной брак; итог по изделию                           → S05-51, S10A-20
  О-3  лесенка по RS-01: ④ изделия с дефектом ≤ О-2, разница — несоответствие без дефекта     → S05-52
  О-4  по видам дефекта (сварка) и строка «входной брак»                                      → S10A-19
  О-5  корзины причин (установлена / только гипотеза / …) в моменты S05 и S14                 → S05-49, S05-53, S05-57, S14-33, S14-34
  О-6  столбец «подтверждено» по категориям; «гипотеза» = корзина «только гипотеза» О-5      → S05-50, S05-54
  О-7  сварка по участку, исполнителю, смене; повторы отдельно; по изделию Ф-001              → S05-58, S01-25
  О-9  корзины незавершённых: «приостановлено по качеству», «прервано», «зависло»            → S05-48, S05-55, S13-18
  О-10 срез «станок» в окне инцидента; S14 — мало сопоставимых работ (6 < 10)                  → S05-59, S14-35
  О-11 журнал изменений показателя: было → стало, из-за чего, кто и когда                    → S03-18, S08-15, S07-14
  позднее событие не меняет незатронутое (О-1, О-2, О-3, О-8, О-10: 12:04 → 12:10)             → S07-N5
  повторяемость проблемы (Д-6 [П]) по виду дефекта × операции × оборудованию                 → S05-61, S14-36
  правила без чисел (З-6, З-9, происхождение времени) — проверяется наличие и форма          → S14-N7, S01-N4, S05-60

Правила, которых нет в данных как событий (движок соседей):
  - гипотезы пишет система при каждом подтверждённом несоответствии (S03-07: версия 1 сразу), поэтому подтверждённое
    несоответствие без подтверждённой причины — в корзине О-5 «есть только гипотеза»; «разбор не начат» — пусто;
  - групповое несоответствие (НС-Г1, НС-14Г) — решение по группе изделий, а не отдельный случай: в О-5 и О-6 его нет
    (как в S14-22: «исполнитель — подтверждено» +3, а не +4), в Д-4, Д-5 и в дефектах лесенки О-3 — тоже;
    в О-2 его изделия считаются (несоответствие без найденного дефекта — S05-52);
  - изделие в О-1 и О-2 — фланец маршрута; кольцо на складе (НС-05, К-105) — компонент, идёт в Р-4.
Время — «определено системой»: occurred_at (исправленное у S15), до минут вниз; снимок «на момент» — по received_at.
Запуск отдельно: python3 tools/metrics.py        Из check.py: metrics.check(err).
"""
import datetime as dt
import json
import math
import os
import sys
from collections import defaultdict

import yaml

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import norms  # noqa: E402

MSK = norms.MSK
CLEAR_Q = 0.6                      # порог качества наблюдения (S04-02)
ROUTE_CPS = ("CP-MO", "CP-WELD", "CP-ASM", "CP-LEAK", "CP-FQC")   # КТ маршрута фланца после входа


def T(s_):
    return dt.datetime.fromisoformat(s_)


def acc(steps):
    return [s for s in steps if s.get("expect", "accepted") == "accepted"]


def st_time(s):
    return norms.t_msk(s["at"])


def recv(o):
    return norms.t_msk(o["received_at"])


def median(v):
    v = sorted(v)
    n = len(v)
    m = None if not n else (v[n // 2] if n % 2 else (v[n // 2 - 1] + v[n // 2]) / 2)
    return int(m) if m is not None and m == int(m) else m


def p90(v):
    """90-й процентиль, ближайший ранг."""
    v = sorted(v)
    return v[max(0, math.ceil(0.9 * len(v)) - 1)] if v else None


# ----------------------------------------------------------------------------- несоответствия и причины
def ncs(steps, until=None):
    """Подтверждённые несоответствия: nc → {items, at, group, defects}."""
    out = {}
    for s in acc(steps):
        if s["action"] != "decision.signal_confirmed" or not s["subject"].get("nc_ref"):
            continue
        if until and st_time(s) > until:
            continue
        sub, p = s["subject"], s["params"]
        group = sub.get("nc_kind") == "group"
        items = sub.get("items") or [sub.get("item_id")]
        defects = [] if group else (p.get("defects") or [{"defect_type": p.get("defect_type"), "zone": p.get("zone")}])
        out[sub["nc_ref"]] = {"items": items, "at": st_time(s), "group": group, "defects": defects,
                              "label": s["label"], "by": s.get("persona"), "run": p.get("run")}
    return out


def causes(steps, until=None):
    out = {}
    for s in acc(steps):
        if s["action"] == "cause.confirmed" and (until is None or st_time(s) <= until):
            for nc in s["subject"].get("nc_refs") or []:
                out[nc] = {"category": s["params"].get("category"), "at": st_time(s)}
    return out


def incoming_set(steps):
    return {nc for nc, c in causes(steps).items() if c["category"] == "incoming"}


# ----------------------------------------------------------------------------- О-5, О-6
def o5(steps, at):
    n, c = {k: v for k, v in ncs(steps, at).items() if not v["group"]}, causes(steps, at)
    conf = sorted(k for k in n if k in c)
    return {"cause_confirmed": conf, "hypothesis_only": sorted(k for k in n if k not in c),
            "undetermined": [], "not_needed_by_policy": [], "not_started": [],
            "confirmed_category": {k: c[k]["category"] for k in conf}}


CATEGORIES = ("incoming", "equipment", "operator_procedure", "handling", "undetermined")


def o6_confirmed(steps, at):
    n, c = {k: v for k, v in ncs(steps, at).items() if not v["group"]}, causes(steps, at)
    out = {k: 0 for k in CATEGORIES}
    for k in n:
        if k in c:
            out[c[k]["category"]] += 1
    return out


# ----------------------------------------------------------------------------- О-1 … О-3
def is_clear(o):
    p = o["payload"]
    q = p.get("observation_quality")
    return p["inspection_result"] == "defect_found" or (p["inspection_result"] == "no_defect_found" and (q is None or q >= CLEAR_Q))


def o1(rows, fr, to, by_received=None):
    ins = [o for o in rows if o["event_type"] == "inspection.result" and (o.get("item_id") or "").startswith("F-")
           and (by_received is None or recv(o) <= by_received)]
    win = [o for o in ins if fr <= norms.rec_time(o) <= to]
    items = sorted({o["item_id"] for o in win if is_clear(o)})
    unclear_only = sorted({o["item_id"] for o in win} - set(items))
    unable = sorted(o["_sim"]["label"] for o in win if not is_clear(o))
    cps = defaultdict(set)
    for o in ins:
        if is_clear(o) and norms.rec_time(o) <= to:
            cps[o["item_id"]].add(o["payload"]["checkpoint_id"])
    full = sorted(i for i in items if set(ROUTE_CPS) <= cps[i])
    return {"items_inspected": len(items), "full_route": len(full), "only_unable_or_no_data": len(unclear_only),
            "unable_to_assess_observations": [lab.split("/")[1] for lab in unable]}


def o2(steps, fr, at):
    n = {k: v for k, v in ncs(steps, at).items() if v["at"] >= fr}
    inc = incoming_set(steps)
    prod, incoming, comp = set(), set(), set()
    for k, v in n.items():
        for it in v["items"]:
            if not it.startswith("F-"):
                comp.add(it)
            elif k in inc:
                incoming.add(it)
            else:
                prod.add(it)
    return {"production": sorted(prod), "incoming": sorted(incoming), "total": len(prod | incoming),
            "components_to_R-4": sorted(comp)}


def ladder_rs01(steps, at):
    n = ncs(steps, at)
    inc = incoming_set(steps)
    g = n.get("NC-G1")
    rs = set(g["items"]) if g else set()
    weld = {k: v for k, v in n.items() if not v["group"] and k not in inc and v["items"][0] in rs}
    with_def = sorted({v["items"][0] for v in weld.values()})
    return {"confirmed_defects": sum(len(v["defects"]) for v in weld.values()), "items_with_defects": len(with_def),
            "items_with_defects_list": with_def, "O-2": len(rs), "nc_without_found_defect": sorted(rs - set(with_def))}


def o4_by_type(steps, at):
    n = ncs(steps, at)
    inc = incoming_set(steps)
    weld, incoming = defaultdict(int), defaultdict(int)
    for k, v in n.items():
        if v["group"]:
            continue
        for d in v["defects"]:
            (incoming if k in inc else weld)[d["defect_type"]] += 1
    return {"welding_detected": dict(sorted(weld.items())), "incoming_row": dict(sorted(incoming.items()))}


# ----------------------------------------------------------------------------- выполнения операций
def runs(rows):
    st, fin, pa = {}, {}, defaultdict(list)
    for o in rows:
        p = o["payload"] if isinstance(o["payload"], dict) else {}
        et = o["event_type"]
        if et == "operation.started":
            st[p["operation_run_id"]] = o
        elif et == "operation.finished":
            fin[p["operation_run_id"]] = o
        elif et in ("operation.paused", "operation.resumed"):
            pa[p.get("operation_run_id")].append(o)
    return st, fin, pa


def active_min(so, fo, pauses):
    t0, t1 = norms.rec_time(so), norms.rec_time(fo)
    paused = 0.0
    pp = sorted(pauses, key=norms.rec_time)
    for i, x in enumerate(pp):
        if x["event_type"] == "operation.paused":
            nxt = next((norms.rec_time(y) for y in pp[i + 1:] if y["event_type"] == "operation.resumed"), t1)
            paused += (nxt - norms.rec_time(x)).total_seconds() / 60
    return int((norms.floor_min(t1) - norms.floor_min(t0)).total_seconds() / 60 - paused)


def shift_of(t):
    hm = t.hour * 60 + t.minute
    return "A" if 8 * 60 <= hm < 16 * 60 + 30 else "B"


def o7_welding(rows, fr, to):
    st, fin, pa = runs(rows)
    first, rep = [], []
    for r, so in st.items():
        p = so["payload"]
        if p.get("operation_code") != "W2" or not (so.get("item_id") or "").startswith("F-") or r not in fin:
            continue
        if not (fr <= norms.rec_time(so) <= to) or fin[r]["payload"].get("result") != "completed":
            continue
        d = active_min(so, fin[r], pa.get(r, []))
        rec = (p["station_id"], p["operator_id"], shift_of(norms.rec_time(so)), d)
        (rep if p.get("rework_of_run_id") else first).append(rec)

    def agg(recs, key):
        g = defaultdict(list)
        for r_ in recs:
            g[r_[key]].append(r_[3])
        return {k: {"runs": len(v), "median_min": median(v), "p90_min": p90(v)} for k, v in sorted(g.items())}
    return {"labels": {"source": "computed", "meaning": "active"},
            "first_runs": {"all": {"runs": len(first), "median_min": median([x[3] for x in first]),
                                   "p90_min": p90([x[3] for x in first])},
                           "by_station": agg(first, 0), "by_operator": agg(first, 1), "by_shift": agg(first, 2)},
            "repeats": {"runs": len(rep), "median_min": median([x[3] for x in rep]), "p90_min": p90([x[3] for x in rep])}}


def o7_item(rows, item):
    st, fin, pa = runs(rows)
    mine = [(r, so) for r, so in st.items() if so.get("item_id") == item and r in fin]
    spans = {r: (norms.rec_time(so), norms.rec_time(fin[r])) for r, so in mine}
    nested = sorted(r for r, (a, b) in spans.items()
                    if any(r2 != r and a2 <= a and b <= b2 for r2, (a2, b2) in spans.items()))
    per = {r: active_min(so, fin[r], pa.get(r, [])) for r, so in mine}
    reg = next(norms.rec_time(o) for o in rows if o["event_type"] == "item.registered" and o.get("item_id") == item)
    rel = next(norms.rec_time(o) for o in rows if o["event_type"] == "item.released" and o.get("item_id") == item)
    active = sum(v for r, v in per.items() if r not in nested)
    lead = int((norms.floor_min(rel) - norms.floor_min(reg)).total_seconds() // 60)
    return {"runs_min": dict(sorted(per.items())), "nested_not_added": nested, "active_work_min": active,
            "lead_time_min": lead, "not_in_operation_min": lead - active, "source": "computed"}


def o9(rows, at, items=None):
    """Корзины незавершённых выполнений на момент at (по времени события)."""
    st, fin, pa = runs(rows)
    nrm = yaml.safe_load(open(norms.NORMS, encoding="utf-8"))["steps"]
    delay = {s["data"]["operation_code"]: s["delay_after"]["value"] for s in nrm.values()
             if s["measure"] == "operation" and s.get("data")}
    out = {"in_progress": [], "stuck": [], "quality_hold": [], "aborted": []}
    for r, so in st.items():
        t0 = norms.rec_time(so)
        if t0 > at or (items and so.get("item_id") not in items):
            continue
        fo = fin.get(r)
        code = so["payload"]["operation_code"]
        if fo and norms.rec_time(fo) <= at:
            if fo["payload"].get("result") == "aborted":
                again = any(x.get("item_id") == so.get("item_id") and x["payload"]["operation_code"] == code and
                            norms.rec_time(so) < norms.rec_time(x) <= at for x in st.values())
                if not again:
                    out["aborted"].append(r)
            continue
        pp = sorted((x for x in pa.get(r, []) if norms.rec_time(x) <= at), key=norms.rec_time)
        if pp and pp[-1]["event_type"] == "operation.paused" and pp[-1]["payload"].get("reason") == "quality_block":
            out["quality_hold"].append(r)
        elif code in delay and (at - t0).total_seconds() / 60 > delay[code]:
            out["stuck"].append(r)
        else:
            out["in_progress"].append(r)
    return {k: sorted(v) for k, v in out.items()}


# ----------------------------------------------------------------------------- О-10
def o10_equipment(rows, steps):
    """Окно инцидента RS-01: первые сварки фланцев после отсчёта (конец SV-006-1) до создания круга (11:10)."""
    st, fin, _ = runs(rows)
    anchor = norms.rec_time(fin["SV-006-1"])
    created = T("2026-09-23T11:10:00+03:00")
    welds = {r: so for r, so in st.items() if so["payload"].get("operation_code") == "W2" and
             (so.get("item_id") or "").startswith("F-") and not so["payload"].get("rework_of_run_id") and
             anchor < norms.rec_time(so) <= created}
    inc = incoming_set(steps)
    run_of_item = {so["item_id"]: r for r, so in welds.items()}
    defects = defaultdict(int)
    for k, v in ncs(steps).items():
        if v["group"] or k in inc:
            continue
        r = v["run"] or run_of_item.get(v["items"][0])
        if r in welds:
            defects[r] += len(v["defects"])
    by_eq, by_person = defaultdict(lambda: {"welds": 0, "welds_with_confirmed_defects": 0, "confirmed_defects": 0}), defaultdict(int)
    for r, so in welds.items():
        e = by_eq[so["payload"]["equipment_id"]]
        e["welds"] += 1
        e["welds_with_confirmed_defects"] += 1 if defects[r] else 0
        e["confirmed_defects"] += defects[r]
        by_person[so["payload"]["operator_id"]] += 1
    errs = {p: 0 for p in by_person}
    for s in acc(steps):
        if s["action"] == "cause.confirmed" and (s["params"].get("operator_error") or {}).get("confirmed"):
            errs[s["params"]["operator_error"]["person"]] = errs.get(s["params"]["operator_error"]["person"], 0) + 1
    return {"by_equipment": dict(sorted(by_eq.items())), "comparable_runs_by_person": dict(sorted(by_person.items())),
            "confirmed_errors_by_person": dict(sorted(errs.items()))}


def o10_s14(rows, day="2026-09-29"):
    st, fin, _ = runs(rows)
    by = defaultdict(int)
    for r, so in st.items():
        p = so["payload"]
        if p.get("operation_code") == "W2" and (so.get("item_id") or "").startswith("F-") and not p.get("rework_of_run_id") \
                and norms.rec_time(so).date().isoformat() == day:
            by[p["operator_id"]] += 1
    return dict(sorted(by.items()))


# ----------------------------------------------------------------------------- снимок и журнал изменений
def snapshot(rows, steps, fr, at):
    """О-1, О-2, О-3, О-8, О-10 по тому, что система получила к моменту at."""
    got = [o for o in rows if recv(o) <= at]
    obs = [o for o in got if o["event_type"] == "inspection.result" and o["payload"]["inspection_result"] == "defect_found"
           and fr <= norms.rec_time(o)]
    uniq = {(o.get("item_id"), d.get("zone")) for o in obs for d in (o["payload"].get("defects") or [{}])}
    n = {k: v for k, v in ncs(steps, at).items() if v["at"] >= fr}
    rep = [o for o in got if o["event_type"] == "operation.started" and o["payload"].get("rework_of_run_id")
           and fr <= norms.rec_time(o)]
    errs = sum(1 for s in acc(steps) if s["action"] == "cause.confirmed" and st_time(s) <= at
               and (s["params"].get("operator_error") or {}).get("confirmed"))
    return {"O-1.items": o1(rows, fr, at, by_received=at)["items_inspected"], "O-2.items": o2(steps, fr, at)["total"],
            "O-3.ladder": {"observations": len(obs), "unique_defects": len(uniq),
                           "confirmed_defects": sum(len(v["defects"]) for v in n.values())},
            "O-8.repeated_runs": len(rep), "O-10.confirmed_errors": errs}


def out_of_setpoint_by_day(rows, at):
    """Сварки ИС-2 с током вне уставки по сводкам журнала, которые система получила к моменту at."""
    st, fin, _ = runs(rows)
    spans = [(r, norms.rec_time(so), norms.rec_time(fin[r])) for r, so in st.items()
             if so["payload"].get("equipment_id") == "WS-2" and so["payload"].get("operation_code") == "W2" and r in fin]
    bad = set()
    for o in rows:
        if o["event_type"] != "machine.parameters" or o["source_id"] != "WS-2" or recv(o) > at:
            continue
        p = o["payload"]
        cur = ((p.get("parameters") or {}).get("welding_current_a") or {})
        sp = (p.get("setpoints") or {}).get("welding_current_a")
        if not cur or not sp:
            continue
        lo, hi = sp["nominal"] - sp["tolerance"], sp["nominal"] + sp["tolerance"]
        if cur.get("max", 0) > hi or cur.get("min", 1e9) < lo:
            t = norms.rec_time(o)
            for r, a, b in spans:
                if a <= t <= b:
                    bad.add((a.date().isoformat(), r))
    out = defaultdict(int)
    for d, _ in bad:
        out[d] += 1
    return dict(sorted(out.items()))


def false_alarms(rows, steps, cp, at):
    by_label = {o["_sim"]["label"]: o for o in rows}
    n = 0
    for s in acc(steps):
        if s["action"] == "decision.signal_rejected" and st_time(s) <= at:
            c = s["subject"].get("checkpoint_id") or next(
                (by_label[b]["payload"].get("checkpoint_id") for b in s["subject"].get("signal_basis_labels", []) if b in by_label), None)
            n += c == cp
    return n


def journal(rows, steps, fr):
    """Три записи журнала изменений (О-11): S03 — О-2, S08 — Н-4, S07 — сварки вне режима."""
    by = {s["label"]: s for s in acc(steps)}
    s3 = by["S03/confirm-NC-01"]
    t3 = st_time(s3)
    e3 = {"metric": "O-2", "before": o2(steps, fr, t3 - dt.timedelta(minutes=1))["total"], "after": o2(steps, fr, t3)["total"],
          "due_to": s3["label"], "by": s3["persona"], "at": t3.isoformat()}
    s8 = by["S08/reject"]
    t8 = st_time(s8)
    e8 = {"metric": "N-4", "checkpoint": "CP-ASM", "before": false_alarms(rows, steps, "CP-ASM", t8 - dt.timedelta(minutes=1)),
          "after": false_alarms(rows, steps, "CP-ASM", t8), "due_to": s8["label"], "by": s8["persona"], "at": t8.isoformat(),
          "O-2_entry": o2(steps, fr, t8 - dt.timedelta(minutes=1))["total"] != o2(steps, fr, t8)["total"]}
    late = next(o for o in rows if o["_sim"]["label"] == "EV-WS2-0412")
    ta = recv(late)
    b0 = out_of_setpoint_by_day(rows, ta - dt.timedelta(minutes=1))
    b1 = out_of_setpoint_by_day(rows, ta + dt.timedelta(minutes=5))
    days = sorted(set(b0) | set(b1))
    e7 = {"metric": "runs_out_of_setpoint", "by_day": {d: {"before": b0.get(d, 0), "after": b1.get(d, 0)} for d in days},
          "due_to": "EV-WS2-0412", "by": "system", "at": norms.floor_min(ta).isoformat()}
    return {"S03": e3, "S08": e8, "S07": e7}


# ----------------------------------------------------------------------------- итог по изделию, повторяемость
def outcomes(steps, items):
    disp, verified = {}, set()
    for s in sorted(acc(steps), key=lambda s: s["at"]):
        sub = s["subject"]
        its = sub.get("items") or [sub.get("item_id")]
        if s["action"] == "decision.disposition_set":
            for it in its:
                if it in items:
                    disp[it] = s["params"]["disposition"]
        elif s["action"] == "decision.disposition_verified":
            verified |= set(its) & set(items)
    return {"rework_verified": sorted(i for i in items if disp.get(i) == "rework" and i in verified),
            "in_rework": sorted(i for i in items if disp.get(i) == "rework" and i not in verified),
            "scrapped": sorted(i for i in items if disp.get(i) == "scrap")}


def recurrence(rows, steps, fr, to):
    st, _, _ = runs(rows)
    inc = incoming_set(steps)
    by_op, by_eq = defaultdict(list), defaultdict(list)
    item_run = {}
    for r, so in st.items():
        if so["payload"].get("operation_code") == "W2" and not so["payload"].get("rework_of_run_id"):
            item_run.setdefault(so.get("item_id"), r)
    for k, v in ncs(steps, to).items():
        if v["group"] or k in inc or v["at"] < fr:
            continue
        r = v["run"] or item_run.get(v["items"][0])
        so = st[r]
        for d in v["defects"]:
            by_op["%s@%s" % (d["defect_type"], so["payload"]["operation_code"])].append(k)
            by_eq["%s@%s" % (d["defect_type"], so["payload"]["equipment_id"])].append(k)
    return {"by_operation": {k: len(v) for k, v in sorted(by_op.items())},
            "by_equipment": {k: len(v) for k, v in sorted(by_eq.items())},
            "recurring": sorted(k for k, v in list(by_op.items()) + list(by_eq.items()) if len(v) >= 2)}


# ----------------------------------------------------------------------------- расчёт и сверка
def compute():
    fr, to = T("2026-09-17T08:00:00+03:00"), T("2026-09-24T16:30:00+03:00")
    rows, steps = norms.load_run("main-story.jsonl", "main-story.yaml")
    r14, s14 = norms.load_run("S14.jsonl", "S14.yaml")
    r13, _ = norms.load_run("S13.jsonl", "S13.yaml")
    t1331, t1645 = T("2026-09-23T13:31:00+03:00"), T("2026-09-23T16:45:00+03:00")
    rs01 = ["F-015", "F-017", "F-019", "F-021", "F-023", "F-025"]
    q13 = ["F-030", "F-031", "F-032", "F-033", "F-034", "F-035", "F-036", "F-223", "F-224"]
    return {
        "S05-56": o1(rows, fr, to),
        "S05-51": o2(steps, fr, t1645),
        "S05-52": ladder_rs01(steps, t1645),
        "S10A-19": o4_by_type(steps, to),
        "S05-49": o5(steps, t1331), "S05-53": o5(steps, t1645), "S05-57": o5(steps, to),
        "S14-33": o5(s14, T("2026-09-29T16:10:00+03:00")), "S14-34": o5(s14, T("2026-09-30T11:15:00+03:00")),
        "S05-50": o6_confirmed(steps, t1331), "S05-54": o6_confirmed(steps, t1645),
        "S05-58": o7_welding(rows, fr, to), "S01-25": o7_item(rows, "F-001"),
        "S05-48": o9(rows, T("2026-09-23T12:30:00+03:00")), "S05-55": o9(rows, t1645, rs01),
        "S13-18": o9(r13, T("2026-09-25T10:30:00+03:00"), q13),
        "S05-59": o10_equipment(rows, steps), "S14-35": o10_s14(r14),
        "S07-N5": [snapshot(rows, steps, fr, T("2026-09-23T12:04:00+03:00")),
                   snapshot(rows, steps, fr, T("2026-09-23T12:10:00+03:00"))],
        "journal": journal(rows, steps, fr),
        "S10A-20": outcomes(steps, rs01),
        "S05-61": recurrence(rows, steps, fr, to),
        "S14-36": recurrence(r14, s14, T("2026-09-29T00:00:00+03:00"), T("2026-10-01T23:59:00+03:00")),
        "MO-001-1.reported": next((o["payload"].get("reported_duration") for o in rows if o["event_type"] == "operation.finished"
                                   and o["payload"].get("operation_run_id") == "MO-001-1"), None),
    }


def _j(x):
    return json.dumps(x, sort_keys=True, ensure_ascii=False)


def check(err):
    res = compute()
    ex = norms.expected_values()
    need = ["S01-25", "S01-N4", "S03-18", "S05-48", "S05-49", "S05-50", "S05-51", "S05-52", "S05-53", "S05-54", "S05-55",
            "S05-56", "S05-57", "S05-58", "S05-59", "S05-60", "S05-61", "S07-14", "S07-N5", "S08-15", "S10A-19", "S10A-20",
            "S13-18", "S14-33", "S14-34", "S14-35", "S14-36", "S14-N7"]
    miss = [i for i in need if i not in ex]
    for i in miss:
        err("показатели: нет утверждения %s" % i)
    if miss:
        return [], res

    def cmp(eid, got, want):
        if _j(got) != _j(want):
            err("показатели %s: по данным %s ≠ ожидания %s" % (eid, _j(got), _j(want)))

    f = {i: ex[i].get("field") for i in need}
    cmp("S05-56", res["S05-56"], f["S05-56"])
    cmp("S05-51", res["S05-51"], f["S05-51"])
    cmp("S05-52", res["S05-52"], f["S05-52"])
    if f["S05-52"]["items_with_defects"] > f["S05-52"]["O-2"]:
        err("S05-52: ④ больше О-2")
    cmp("S10A-19", res["S10A-19"], f["S10A-19"])
    for i in ("S05-49", "S05-53", "S05-57", "S14-33", "S14-34"):
        cmp(i, res[i], f[i])
    # О-6: «подтверждено» — по причинам; «гипотеза» — те же случаи, что в корзине О-5 «только гипотеза»
    for i, o5id in (("S05-50", "S05-49"), ("S05-54", "S05-53")):
        cmp(i + " (подтверждено)", res[i], f[i]["confirmed"])
        hyp = sorted(x for v in f[i]["hypothesis"].values() for x in v)
        if sorted(set(hyp)) != res[o5id]["hypothesis_only"]:
            err("%s: в столбце «гипотеза» %s, а в О-5 «только гипотеза» %s" % (i, sorted(set(hyp)), res[o5id]["hypothesis_only"]))
    cmp("S05-58", res["S05-58"], f["S05-58"])
    cmp("S01-25", res["S01-25"], f["S01-25"])
    cmp("S05-48", {k: res["S05-48"][k] for k in ("quality_hold", "stuck", "aborted")}, f["S05-48"])
    cmp("S05-55", {k: res["S05-55"][k] for k in ("quality_hold", "stuck", "aborted")}, f["S05-55"])
    cmp("S13-18", res["S13-18"], f["S13-18"])
    cmp("S05-59", res["S05-59"], f["S05-59"])
    s35 = f["S14-35"]
    cmp("S14-35", res["S14-35"], s35["comparable_runs"])
    mn = s35["min_runs"]
    for p, n in res["S14-35"].items():
        want_note = "мало сопоставимых работ (%d < %d), вывод предварительный" % (n, mn) if n < mn else None
        if (s35.get("note_by_person") or {}).get(p) != want_note:
            err("S14-35: у %s пометка %s, нужна %s" % (p, (s35.get("note_by_person") or {}).get(p), want_note))
    if ex["S14-21"]["value"] != {"WLD-03": 3, "WLD-02": 0}:
        err("S14-21: подтверждённые ошибки должны остаться 3 и 0")
    a, b = res["S07-N5"]
    if _j(a) != _j(b):
        err("S07-N5: опоздавшая пачка изменила незатронутые показатели: %s → %s" % (_j(a), _j(b)))
    cmp("S07-N5", a, ex["S07-N5"]["value"])
    j = res["journal"]
    cmp("S03-18", j["S03"], f["S03-18"])
    cmp("S08-15", j["S08"], f["S08-15"])
    cmp("S07-14", j["S07"], f["S07-14"])
    s707 = ex["S07-07"]["value"]
    for d, v in j["S07"]["by_day"].items():
        if s707.get(d) != v:
            err("S07-14 и S07-07 расходятся за %s: %s ≠ %s" % (d, v, s707.get(d)))
    cmp("S10A-20", res["S10A-20"], f["S10A-20"])
    if _j(res["S10A-20"]) != _j({k: v for k, v in ex["S10A-08"]["value"].items()}):
        err("S10A-20 и S10A-08 расходятся")
    cmp("S05-61", res["S05-61"], f["S05-61"])
    cmp("S14-36", res["S14-36"], f["S14-36"])
    # правила без чисел: форма
    rep = res["MO-001-1.reported"]
    if not rep or (f["S05-60"] or {}).get("O-7@MO-001-1", {}).get("time_origin") != "reported" or \
            (f["S05-60"] or {}).get("D-4", {}).get("time_origin") != "computed" or \
            (f["S05-60"] or {}).get("D-5", {}).get("time_origin") != "computed":
        err("S05-60: происхождение времени у Д-4, Д-5, О-7 мехобработки не то (в данных у MO-001-1 передано: %s)" % rep)
    if ex["S14-N7"].get("op") != "not_exists" or ex["S01-N4"].get("op") != "eq":
        err("S14-N7 / S01-N4: форма утверждения")
    m = res["S05-58"]["first_runs"]
    lines = ["показатели О-1…О-11 (сверка аналитики): О-1 %d фланцев, О-2 %d (вход %s), лесенка ④ %d ≤ О-2 %d, "
             "О-7 сварка медиана %s мин (90-й %s), О-10 ИС-2 %s из %s сварок с дефектом, S14 — %s сопоставимых"
             % (res["S05-56"]["items_inspected"], res["S05-51"]["total"], res["S05-51"]["incoming"],
                res["S05-52"]["items_with_defects"], res["S05-52"]["O-2"], m["all"]["median_min"], m["all"]["p90_min"],
                res["S05-59"]["by_equipment"]["WS-2"]["welds_with_confirmed_defects"], res["S05-59"]["by_equipment"]["WS-2"]["welds"],
                res["S14-35"])]
    return lines, res


if __name__ == "__main__":
    print(json.dumps(compute(), ensure_ascii=False, indent=1, default=str))
