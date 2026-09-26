#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Блок «быстрее нашли — сохранили основания — не повторили брак» на столе руководителя: четыре плитки.
Считает по данным имитатора (потоки + шаги людей) за окно главной истории и сверяет с ожиданиями.

  а) Д-4 задержка обнаружения — от завершения операции, где дефект мог возникнуть, до первого наблюдения с признаками
     (сигнал Е-80 ставится по нему); входной брак — отдельно (места возникновения на участке нет).      → S05-43
  б) Д-5 время разбора — от сигнала (наблюдение + 1 мин) до решения по изделию и до подтверждённой причины. → S05-44
  в) доля выводов с полными основаниями — у вывода есть ссылки, и все они ведут к строкам потока
     (прямо или через прежние выводы).                                                                  → S05-45
  г) повторный брак после меры — окно наблюдения после корректирующей меры: повторов за N единиц.        → S05-46, S14-30

Время — «определено системой»: occurred_at, до целых минут вниз.
Запуск отдельно: python3 tools/tiles.py        Из check.py: tiles.check(err).
"""
import datetime as dt
import json
import os
import sys

import yaml

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import norms  # noqa: E402

BASE = norms.BASE
CONCLUSIONS = ("decision.gate_passed", "decision.signal_confirmed", "decision.signal_rejected", "decision.item_isolated",
               "process_hold.set", "risk_scope.narrowed", "risk_scope.item_assessed", "risk_scope.closed",
               "decision.disposition_set", "decision.disposition_verified", "decision.review_completed",
               "cause.confirmed", "capa.action_assigned", "capa.action_implemented", "capa.effectiveness_confirmed",
               "cause.investigation_closed", "rework.chief_welder_approved")
BASIS_KEYS = ("basis_labels", "signal_basis_labels", "evidence_labels", "claim_basis_labels", "basis_label",
              "supersedes_label", "original_decision_label", "recalculated_due_to_label")


def refs_of(s):
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


def mins(a, b):
    return int((norms.floor_min(b) - norms.floor_min(a)).total_seconds() // 60)


def median(v):
    v = sorted(v)
    n = len(v)
    return None if not n else (v[n // 2] if n % 2 else (v[n // 2 - 1] + v[n // 2]) / 2)


def window(name):
    w = yaml.safe_load(open(norms.NORMS, encoding="utf-8"))["meta"]["scenario_windows"][name]
    return dt.datetime.fromisoformat(w["from"]), dt.datetime.fromisoformat(w["to"])


def acc(steps):
    return [s for s in steps if s.get("expect", "accepted") == "accepted"]


def nc_list(steps, fr, to):
    out = []
    for s in acc(steps):
        sub = s["subject"]
        if s["action"] == "decision.signal_confirmed" and sub.get("nc_ref") and sub.get("nc_kind") != "group" \
                and fr <= norms.t_msk(s["at"]) <= to:
            out.append(s)
    return out


def incoming_ncs(steps):
    return {n for s in acc(steps) if s["action"] == "cause.confirmed" and s["params"].get("category") == "incoming"
            for n in s["subject"].get("nc_refs") or []}


def tile_a(rows, steps, fr, to):
    by_label = {o["_sim"]["label"]: o for o in rows}
    inc = incoming_ncs(steps)
    fin = [o for o in rows if o["event_type"] == "operation.finished" and o["payload"].get("result") == "completed"]
    by_nc, incoming = {}, []
    for s in nc_list(steps, fr, to):
        nc, it = s["subject"]["nc_ref"], s["subject"].get("item_id")
        if nc in inc:
            incoming.append(nc)
            continue
        obs = min(norms.rec_time(by_label[b]) for b in s["subject"]["signal_basis_labels"] if b in by_label)
        ops = [norms.rec_time(o) for o in fin if o.get("item_id") == it and norms.rec_time(o) <= obs]
        if ops:
            by_nc[nc] = mins(max(ops), obs)
    v = list(by_nc.values())
    return {"by_nc_min": by_nc, "median_min": median(v), "min_min": min(v), "max_min": max(v),
            "incoming_separately": sorted(incoming)}


def tile_b(rows, steps, fr, to):
    by_label = {o["_sim"]["label"]: o for o in rows}
    out = {}
    disp = [s for s in acc(steps) if s["action"] == "decision.disposition_set"]
    cause = [s for s in acc(steps) if s["action"] == "cause.confirmed"]
    for s in nc_list(steps, fr, to):
        nc, it = s["subject"]["nc_ref"], s["subject"].get("item_id")
        t_conf = norms.t_msk(s["at"])
        sig = min(norms.rec_time(by_label[b]) for b in s["subject"]["signal_basis_labels"] if b in by_label) \
            + dt.timedelta(minutes=1)
        d = [norms.t_msk(x["at"]) for x in disp if norms.t_msk(x["at"]) >= t_conf and
             (nc in ([x["subject"].get("nc_ref")] + list(x["subject"].get("nc_refs") or []))
              or it in (x["subject"].get("items") or [x["subject"].get("item_id")]))]
        c = [norms.t_msk(x["at"]) for x in cause if nc in (x["subject"].get("nc_refs") or [])]
        out[nc] = {"to_disposition_min": mins(sig, min(d)) if d else None,
                   "to_confirmed_cause_min": mins(sig, min(c)) if c else None}
    td = [v["to_disposition_min"] for v in out.values() if v["to_disposition_min"] is not None]
    tc = [v["to_confirmed_cause_min"] for v in out.values() if v["to_confirmed_cause_min"] is not None]
    return {"by_nc": out, "median_to_disposition_min": median(td), "median_to_confirmed_cause_min": median(tc),
            "without_disposition": sorted(k for k, v in out.items() if v["to_disposition_min"] is None),
            "without_confirmed_cause": sorted(k for k, v in out.items() if v["to_confirmed_cause_min"] is None)}


def tile_c(rows, steps, fr, to):
    labels = {o["_sim"]["label"] for o in rows}
    full, n, ok, bad = set(), 0, 0, []
    for s in sorted(acc(steps), key=lambda s: s["at"]):
        if s["action"] not in CONCLUSIONS:
            continue
        r = refs_of(s)
        good = bool(r) and all(x in labels or x in full for x in r)
        if good:
            full.add(s["label"])
        if fr <= norms.t_msk(s["at"]) <= to:
            n += 1
            ok += good
            if not good:
                bad.append(s["label"])
    return {"conclusions": n, "with_full_basis": ok, "share_pct": round(100.0 * ok / n, 1) if n else None,
            "without_full_basis": bad}


def tile_d(rows, steps, fr, to):
    """Меры с окном наблюдения: что наблюдали после меры и сколько повторов той же причины."""
    out = []
    for s in acc(steps):
        if s["action"] != "capa.action_assigned" or not (fr <= norms.t_msk(s["at"]) <= to):
            continue
        ncs = set(s["subject"].get("nc_refs") or [])
        t0 = norms.t_msk(s["at"])
        impl = [norms.t_msk(x["at"]) for x in acc(steps) if x["action"] == "capa.action_implemented"
                and ncs & set(x["subject"].get("nc_refs") or [])]
        eff = [x for x in acc(steps) if x["action"] == "capa.effectiveness_confirmed" and ncs & set(x["subject"].get("nc_refs") or [])]
        rec = {"measure": s["label"], "cause_ref": s.get("process_ref")}
        if s["params"].get("kind") == "improve_detection":       # вход: следующие партии того же поставщика
            lot = s["params"].get("lot_id")
            sup = next((o["payload"].get("supplier_id") for o in rows if o["event_type"] == "erp.batch_received"
                        and o["payload"].get("lot_id") == lot), None)
            typ = next((o["payload"].get("item_type_id") for o in rows if o["event_type"] == "erp.batch_received"
                        and o["payload"].get("lot_id") == lot), None)
            nxt = sorted(o["payload"]["lot_id"] for o in rows if o["event_type"] == "erp.batch_received"
                         and o["payload"].get("supplier_id") == sup and o["payload"].get("item_type_id") == typ
                         and norms.rec_time(o) > t0)
            plan = (s["params"].get("effectiveness_plan") or {}).get("window", "")
            need = int(plan.split()[1]) if plan.split()[:1] == ["следующие"] and plan.split()[1].isdigit() else None
            rec.update(window_unit="lot", window_needed=need, observed=len(nxt), repeats=0, closed=bool(eff))
        else:                                                     # сварщик: первые N новых швов после внедрения
            who = next((x["subject"].get("person") for x in acc(steps) if x["action"] == "cause.explanation_recorded"
                        and ncs & set(x["subject"].get("nc_refs") or [])), None)
            t_impl = min(impl) if impl else None
            starts = sorted((norms.rec_time(o), o["item_id"], o["payload"]["operation_run_id"]) for o in rows
                            if o["event_type"] == "operation.started" and o["payload"].get("operation_code") == "W2"
                            and o["payload"].get("operator_id") == who and not o["payload"].get("rework_of_run_id")
                            and not (o.get("item_id") or "").startswith("CS-") and t_impl and norms.rec_time(o) > t_impl)
            win = starts[:8]
            runs = {r for _, _, r in win}
            items = {i for _, i, _ in win}
            rep = 0
            for o in rows:
                p = o["payload"] if isinstance(o["payload"], dict) else {}
                if o["event_type"] == "inspection.result" and p.get("inspection_result") == "defect_found" \
                        and (o.get("item_id") in items or any(r in o["_sim"]["label"] for r in runs)):
                    rep += 1
            rec.update(window_unit="weld", window_needed=8, observed=len(win), window_items=sorted(items), repeats=rep,
                       closed=bool(eff))
        out.append(rec)
    causes_no_measure = sorted({s.get("process_ref") for s in acc(steps) if s["action"] == "cause.confirmed"
                                and fr <= norms.t_msk(s["at"]) <= to}
                               - {r["cause_ref"] for r in out})
    return {"measures": out, "causes_without_measure": causes_no_measure}


def compute():
    res = {}
    fr, to = window("main_story")
    rows, steps = norms.load_run("main-story.jsonl", "main-story.yaml")
    res["main_story"] = {"a": tile_a(rows, steps, fr, to), "b": tile_b(rows, steps, fr, to),
                         "c": tile_c(rows, steps, fr, to), "d": tile_d(rows, steps, fr, to)}
    rows14, steps14 = norms.load_run("S14.jsonl", "S14.yaml")
    far = (dt.datetime(2026, 9, 1, tzinfo=norms.MSK), dt.datetime(2026, 10, 31, tzinfo=norms.MSK))
    res["S14"] = {"d": tile_d(rows14, steps14, *far), "c": tile_c(rows14, steps14, *far)}
    for name, st, sc in (("S13", "S13.jsonl", "S13.yaml"), ("S10B", "S10B.jsonl", "S10B.yaml")):
        r_, s_ = norms.load_run(st, sc)
        res[name] = {"c": tile_c(r_, s_, *far)}
    return res


def _cmp(err, eid, got, want):
    if json.dumps(got, sort_keys=True, ensure_ascii=False) != json.dumps(want, sort_keys=True, ensure_ascii=False):
        err("плитки %s: по данным %s ≠ ожидания %s" % (eid, got, want))


def check(err):
    res = compute()
    ex = norms.expected_values()
    m = res["main_story"]
    need = ("S05-43", "S05-44", "S05-45", "S05-46", "S14-30")
    for i in need:
        if i not in ex:
            err("плитки: нет утверждения %s" % i)
    if any(i not in ex for i in need):
        return [], res
    a = m["a"]
    _cmp(err, "S05-43", {"by_nc_min": a["by_nc_min"], "median_min": a["median_min"], "max_min": a["max_min"],
                         "incoming_separately": a["incoming_separately"]}, ex["S05-43"]["field"])
    b = m["b"]
    _cmp(err, "S05-44", {"by_nc": b["by_nc"], "median_to_disposition_min": b["median_to_disposition_min"],
                         "median_to_confirmed_cause_min": b["median_to_confirmed_cause_min"],
                         "without_confirmed_cause": b["without_confirmed_cause"]}, ex["S05-44"]["field"])
    c = m["c"]
    _cmp(err, "S05-45", {k: c[k] for k in ("conclusions", "with_full_basis", "share_pct")}, ex["S05-45"]["field"])
    d = m["d"]
    _cmp(err, "S05-46", {"measures": [{k: x[k] for k in ("measure", "window_unit", "window_needed", "observed", "repeats", "closed")}
                                      for x in d["measures"]], "causes_without_measure": d["causes_without_measure"]},
         ex["S05-46"]["field"])
    d14 = [x for x in res["S14"]["d"]["measures"] if x["window_unit"] == "weld"]
    _cmp(err, "S14-30", {k: d14[0][k] for k in ("window_items", "observed", "repeats", "closed")} if d14 else None,
         ex["S14-30"]["field"])
    for name in ("S14", "S13", "S10B"):
        cc = res[name]["c"]
        if cc["with_full_basis"] != cc["conclusions"]:
            err("плитка «в» %s: выводы без полных оснований %s" % (name, cc["without_full_basis"][:5]))
    lines = ["плитки (главная история): Д-4 медиана %s мин, Д-5 медиана до решения %s мин, до причины %s мин, "
             "основания %s/%s, мер с окном %d" % (a["median_min"], b["median_to_disposition_min"],
                                                 b["median_to_confirmed_cause_min"], c["with_full_basis"], c["conclusions"],
                                                 len(d["measures"]))]
    return lines, res


if __name__ == "__main__":
    print(json.dumps(compute(), ensure_ascii=False, indent=1, default=str))
