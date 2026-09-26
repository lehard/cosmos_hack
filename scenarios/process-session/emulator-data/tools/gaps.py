#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Карта дефицита данных (Н-6 [П]): пробел → сколько раз мешал → цена → что закрывает.

Разметка случаев — definitions/reference/data-gaps.yaml; цену каждого случая считает этот скрипт по потокам,
шагам людей, сообщениям в MES (mes_outbox_expected) и нормам (process/step-norms.yaml).
Рейтинг сверяется с ожиданиями S07-13 (весь рейтинг), S14-32 (случаи S14) и S13-17 (в S13 пробелов нет).

Запуск отдельно: python3 tools/gaps.py        Из check.py: gaps.check(err).
"""
import datetime as dt
import json
import os
import sys

import yaml

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import norms  # noqa: E402

BASE = norms.BASE
RUNS = {"main_story": ("main-story.jsonl", "main-story.yaml", "I1.yaml"), "S14": ("S14.jsonl", "S14.yaml", "S14.yaml"),
        "S13": ("S13.jsonl", "S13.yaml", "S13.yaml")}


def load():
    return yaml.safe_load(open(os.path.join(BASE, "definitions", "reference", "data-gaps.yaml"), encoding="utf-8"))


def run_data(run, cache={}):
    if run not in cache:
        st, sc, mes = RUNS[run]
        rows, steps = norms.load_run(st, sc)
        allrows = [json.loads(line) for line in open(os.path.join(BASE, "streams", st), encoding="utf-8")]
        ob = yaml.safe_load(open(os.path.join(BASE, "definitions", "scenarios", mes), encoding="utf-8")).get("mes_outbox_expected") or []
        cache[run] = dict(rows=rows, steps=steps, ev={o["_sim"]["label"]: o for o in allrows if o["_sim"]["noise"] != "duplicate"},
                          st={s["label"]: s for s in steps}, mes=ob)
    return cache[run]


def t_of(d, label, received=False):
    if label in d["st"]:
        return norms.floor_min(norms.t_msk(d["st"][label]["at"]))
    o = d["ev"][label]
    return norms.floor_min(norms.t_msk(o["received_at"]) if received else norms.rec_time(o))


def obj(r):
    return r.get("item_id") or r.get("lot_id")


def msk(s_):
    return dt.datetime.strptime(s_, "%Y-%m-%d %H:%M:%S").replace(tzinfo=norms.MSK)


def cost_of(case, d, norm_meas):
    c = case.get("cost") or {}
    out = {"item_hours": 0.0, "extra_checks": 0, "conclusion_delay_h": 0.0}
    ih = c.get("item_hours") or {}
    if "gate_wait" in ih:
        g = ih["gate_wait"]
        m = [x for x in norm_meas if x["step"] == g["step"] and x["object"] == g["object"]
             and x.get("presentation_no") == g["presentation_no"]]
        out["item_hours"] = m[0]["value"] / 60.0
    if "held_until" in ih:
        h = ih["held_until"]
        steps = h["step"] if isinstance(h["step"], list) else [h["step"]]
        tot = 0.0
        for r in d["mes"]:
            if r["type"] == "release" and r.get("decision_step") in steps and h["basis"] in (r.get("released_bases") or []):
                hold = max(msk(x["at_msk"]) for x in d["mes"] if x["type"] == "hold" and obj(x) == obj(r)
                           and msk(x["at_msk"]) <= msk(r["at_msk"]))
                tot += (msk(r["at_msk"]) - hold).total_seconds() / 3600
        out["item_hours"] = tot
    if "between" in ih:
        b = ih["between"]
        out["item_hours"] = b.get("items", 1) * (t_of(d, b["to"]) - t_of(d, b["from"])).total_seconds() / 3600
    cd = c.get("conclusion_delay_h") or {}
    if "between" in cd:
        b = cd["between"]
        out["conclusion_delay_h"] = (t_of(d, b["to"], b.get("to_received")) - t_of(d, b["from"])).total_seconds() / 3600
    out["extra_checks"] = len(c.get("extra") or [])
    return {k: round(v, 2) if isinstance(v, float) else v for k, v in out.items()}


def compute(err=None):
    dg = load()
    nres = norms.compute()
    meas = {"main_story": nres["main_story"]["meas"], "S13": nres["S13"]["meas"]}
    cases = []
    for case in dg["cases"]:
        d = run_data(case["run"])
        for lb in list(case.get("refs") or []) + list((case.get("cost") or {}).get("extra") or []):
            if lb not in d["ev"] and lb not in d["st"] and err:
                err("карта дефицита %s: записи %s нет в данных прогона %s" % (case["id"], lb, case["run"]))
        cost = cost_of(case, d, meas.get(case["run"], []))
        cases.append(dict(id=case["id"], gap=case["gap"], scenario=case["scenario"], hindered=case["hindered"], **cost))
    rating = []
    for g in dg["gaps"]:
        cs = [c for c in cases if c["gap"] == g["id"]]
        hit = [c for c in cs if c["hindered"] != "none"]
        rating.append(dict(gap=g["id"], hindered=len(hit), hindered_what=sorted({c["hindered"] for c in hit}),
                           item_hours=round(sum(c["item_hours"] for c in cs), 2),
                           extra_checks=sum(c["extra_checks"] for c in cs),
                           conclusion_delay_h=round(sum(c["conclusion_delay_h"] for c in cs), 2),
                           closes_step=g["closes"]["ladder_step"], cases=[c["id"] for c in cs]))
    rating.sort(key=lambda r: (-r["hindered"], -r["item_hours"], -r["extra_checks"], r["gap"]))
    for k, r in enumerate(rating):
        r["rank"] = k + 1
    return dg, cases, rating


def check(err):
    dg, cases, rating = compute(err)
    ex = norms.expected_values()
    gap_ids = {g["id"] for g in dg["gaps"]}
    ladder = {x["step"] for x in dg["ladder"]}
    for c in dg["cases"]:
        if c["gap"] not in gap_ids:
            err("карта дефицита %s: пробела %s нет в справочнике" % (c["id"], c["gap"]))
        for i in c.get("expect_refs") or []:
            if i not in ex:
                err("карта дефицита %s: утверждения %s нет в ожиданиях" % (c["id"], i))
    for g in dg["gaps"]:
        if g["closes"]["ladder_step"] not in ladder:
            err("карта дефицита %s: ступени %s нет" % (g["id"], g["closes"]["ladder_step"]))
    want = (ex.get("S07-13") or {}).get("field", {}).get("rating")
    got = [{k: r[k] for k in ("rank", "gap", "hindered", "item_hours", "extra_checks", "conclusion_delay_h", "closes_step")}
           for r in rating]
    if want != got:
        err("карта дефицита: рейтинг по данным %s ≠ S07-13 %s" % (got, want))
    s14 = [{k: c[k] for k in ("id", "hindered", "item_hours", "extra_checks")} for c in cases if c["scenario"] == "S14"]
    want14 = (ex.get("S14-32") or {}).get("field", {}).get("data_gap_cases")
    if want14 != s14:
        err("карта дефицита: случаи S14 по данным %s ≠ S14-32 %s" % (s14, want14))
    n13 = sum(1 for c in cases if c["scenario"] == "S13")
    if (ex.get("S13-17") or {}).get("field", {}).get("data_gap_cases") != n13 or "S13" not in dg["meta"]["no_gaps"]:
        err("карта дефицита: у S13 случаев %d, в S13-17 — %s" % (n13, (ex.get("S13-17") or {}).get("field")))
    top = rating[0]
    return ["карта дефицита: случаев %d, пробелов %d; первое место — %s: мешал %d раз, %.2f изделие-ч" % (
        len(cases), len(rating), top["gap"], top["hindered"], top["item_hours"])], rating


if __name__ == "__main__":
    dg, cases, rating = compute(lambda m: print("ОШИБКА:", m))
    for c in cases:
        print(c)
    print()
    for r in rating:
        print(r)
