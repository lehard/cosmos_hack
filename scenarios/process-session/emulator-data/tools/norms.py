#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Нормы времени шагов схемы (process/step-norms.yaml) против данных имитатора.

Считает по прогону (поток + шаги людей) все измерения, у которых в файле норм есть порог:
длительность операций, ожидание на ЗТ, ожидание решения по сигналу, путь между цехами,
окно кромок, задержку результата камеры КТ-3, ожидание рентгена, срок решения по несоответствию.
Уровень каждого измерения — ok / attention / delay. Для ЗТ — уровни эскалации.

Время — «определено системой»: occurred_at (у записей со сдвигом часов S15 — исправленное),
до целых минут вниз. Ожидание на ЗТ и по сигналу — в минутах смены, в которой решение.

Запуск отдельно: python3 tools/norms.py   (печатает задержки по окнам главной истории и S13)
Из check.py: norms.check(err) — сверка с ожиданиями S07-12 и S13-16.
"""
import datetime as dt
import json
import os
from collections import defaultdict

import yaml

BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
NORMS = os.path.normpath(os.path.join(BASE, "..", "step-norms.yaml"))
MSK = dt.timezone(dt.timedelta(hours=3))


def t_msk(s_):
    return dt.datetime.strptime(s_[:19], "%Y-%m-%dT%H:%M:%S").replace(tzinfo=dt.timezone.utc).astimezone(MSK)


def floor_min(t):
    return t.replace(second=0, microsecond=0)


def rec_time(o):
    """Время записи для расчёта: occurred_at; у записей со сдвигом часов — исправленное (S15)."""
    t = t_msk(o["occurred_at"])
    nz = o["_sim"].get("noise") or ""
    if nz.startswith("clock_skew:+"):
        t -= dt.timedelta(minutes=int(nz.split("+")[1].rstrip("m")))
    return t


def load_calendar():
    c = yaml.safe_load(open(os.path.join(BASE, "definitions", "reference", "calendar.yaml"), encoding="utf-8"))
    sh = {s["id"]: s for s in c["shifts"]}
    off = {dt.date.fromisoformat(d) for d in c["production_calendar"]["non_working_days"]}
    return sh, off


def shift_start(t, shifts):
    a = shifts["A"]
    h, m = map(int, a["start"].split(":"))
    return t.replace(hour=h, minute=m, second=0, microsecond=0)


def working_days_between(d0, d1, off):
    n, d = 0, d0
    while d < d1:
        d += dt.timedelta(days=1)
        if d.weekday() < 5 and d not in off:
            n += 1
    return n


def level(v, st):
    w = st.get("wait") or {}
    lim = st["delay_after"]["value"]
    if v > lim:
        return "delay"
    if "attention_lt" in w:
        return "ok" if v < w["norm_lt"] else "attention"
    dur = st.get("duration") or {}
    if isinstance(dur.get("norm"), (int, float)):
        return "ok" if v <= dur.get("max", dur["norm"]) else "attention"
    return "ok"


def load_run(stream, scen):
    rows = []
    for line in open(os.path.join(BASE, "streams", stream), encoding="utf-8"):
        o = json.loads(line)
        nz = o["_sim"].get("noise") or ""
        if nz == "duplicate" or nz.startswith("malformed") or nz.startswith("conflict"):
            continue
        rows.append(o)
    steps = yaml.safe_load(open(os.path.join(BASE, "definitions", "scenarios", scen), encoding="utf-8"))["human_steps"]
    return rows, steps


def measure(rows, steps, norms):
    shifts, off = load_calendar()
    st = norms["steps"]
    out = []
    by_label = {o["_sim"]["label"]: o for o in rows}

    def add(key, obj, frm, to, val, unit, extra=None):
        d = dict(step=key, object=obj, frm=frm, to=to, value=val, unit=unit, level=level(val, st[key]))
        if extra:
            d.update(extra)
        out.append(d)

    # ---- операции
    op_steps = {s["data"]["operation_code"]: k for k, s in st.items() if s["measure"] == "operation" and s.get("data")}
    starts, pauses, finish = {}, defaultdict(list), {}
    for o in rows:
        p = o["payload"] if isinstance(o["payload"], dict) else {}
        et = o["event_type"]
        if et == "operation.started":
            starts[p["operation_run_id"]] = (o, rec_time(o))
        elif et in ("operation.paused", "operation.resumed"):
            pauses[p.get("operation_run_id") or o.get("correlation_id")].append((et, rec_time(o)))
        elif et == "operation.finished":
            finish[p["operation_run_id"]] = (o, rec_time(o))
    for run, (so, t0) in starts.items():
        code = so["payload"]["operation_code"]
        if code not in op_steps or run not in finish:
            continue
        fo, t1 = finish[run]
        if fo["payload"].get("result") != "completed":
            continue
        paused = 0.0
        pp = sorted(pauses.get(run, []), key=lambda x: x[1])
        for i, (et, t) in enumerate(pp):
            if et == "operation.paused":
                nxt = next((x[1] for x in pp[i + 1:] if x[0] == "operation.resumed"), t1)
                paused += (nxt - t).total_seconds() / 60
        val = int(((floor_min(t1) - floor_min(t0)).total_seconds() / 60) - paused)
        add(op_steps[code], so.get("item_id") or run, t0, t1, val, "min", {"run": run})

    # ---- ожидание на ЗТ
    gate_step = {s["data"]["gate"]: k for k, s in st.items() if s["measure"] == "gate_wait"}
    facts = defaultdict(list)
    for o in rows:
        pr = o["_sim"].get("process") or ""
        k = pr[4:] if pr.startswith("lot/") else o.get("item_id")
        if k:
            facts[k].append(o)

    def is_ready(gate, o):
        p = o["payload"] if isinstance(o["payload"], dict) else {}
        et = o["event_type"]
        if gate == "ZT-1":
            return et == "inspection.result"
        if gate == "ZT-2":
            return et == "inspection.result" and p.get("method") == "cmm"
        if gate == "ZT-3":
            return et == "inspection.result" and p.get("method") == "xray" and p.get("checkpoint_id") == "CP-WELD"
        if gate == "ZT-SR":
            return et == "inspection.result" and o["source_id"] == "CAM-UV"
        if gate == "ZT-4":
            return et == "operation.finished" and p.get("operation_run_id", "").startswith("AS-")
        if gate == "ZT-5":
            return et == "operation.finished" and p.get("operation_run_id", "").startswith("HT-")
        if gate == "ZT-6":
            return et == "inspection.result" and o["source_id"] == "CAM-FQC"
        return False

    gate_waits = []
    for s in steps:
        if s["action"] != "decision.gate_passed" or s.get("expect", "accepted") != "accepted":
            continue
        g = s["subject"]["gate"]
        if g not in gate_step:
            continue
        obj = s["subject"].get("item_id") or s["subject"].get("lot_id")
        at = floor_min(t_msk(s["at"]))
        pno = s["params"].get("presentation_no", 1)
        if pno >= 2:
            cand = [rec_time(o) for o in facts[obj] if o["event_type"] == "item.presented"
                    and (o["payload"].get("gate") == g and o["payload"].get("presentation_no") == pno)]
        else:
            cand = []
        if not cand:
            cand = [rec_time(o) for o in facts[obj] if is_ready(g, o)]
        cand = [c for c in cand if floor_min(c) <= at]
        if not cand:
            continue
        ready = floor_min(max(cand))
        start = max(ready, shift_start(at, shifts))
        val = int((at - start).total_seconds() / 60)
        key = gate_step[g]
        esc = {}
        for lvl, lim in ((st[key].get("escalation") or {}).items()):
            if lvl in ("unit", "to"):
                continue
            fire = start + dt.timedelta(minutes=lim)
            if fire < at:
                esc[lvl] = fire
        add(key, obj, ready, at, val, "min", {"gate": g, "presentation_no": pno, "counted_from": start,
                                               "escalation": esc, "signer": s["persona"]})
        gate_waits.append(out[-1])

    # ---- решение по сигналу
    for s in steps:
        if s["action"] not in ("decision.signal_confirmed", "decision.signal_rejected") or s.get("expect", "accepted") != "accepted":
            continue
        bl = [by_label[b] for b in (s["subject"].get("signal_basis_labels") or []) if b in by_label]
        if not bl:
            continue
        obs = floor_min(min(rec_time(o) for o in bl))
        at = floor_min(t_msk(s["at"]))
        start = max(obs, shift_start(at, shifts))
        add("SG3", s["subject"].get("item_id"), obs, at, int((at - start).total_seconds() / 60), "min",
            {"decision": s["action"].split("_")[-1]})

    # ---- путь между цехами
    route_step = {s["data"]["route"]: k for k, s in st.items() if s["measure"] == "transit"}
    sent = {}
    for o in rows:
        p = o["payload"] if isinstance(o["payload"], dict) else {}
        if o["event_type"] == "movement.sent":
            sent[(o["item_id"], p["from_shop"], p["to_shop"])] = rec_time(o)
        elif o["event_type"] == "movement.received":
            k = (o["item_id"], p["from_shop"], p["to_shop"])
            if k in sent:
                r = "%s->%s" % k[1:]
                if r in route_step:
                    t0 = sent.pop(k)
                    t1 = rec_time(o)
                    add(route_step[r], o["item_id"], t0, t1, round((t1 - t0).total_seconds() / 3600, 1), "h")
    unreceived = [(k, t0) for k, t0 in sent.items() if "%s->%s" % k[1:] in route_step]

    # ---- окно кромок, камера КТ-3, рентген
    prep = {}
    for o in rows:
        p = o["payload"] if isinstance(o["payload"], dict) else {}
        if o["event_type"] == "operator.action" and (p.get("details") or {}).get("step") == "edge_prep":
            prep[o["_sim"]["label"].rsplit(".", 1)[0]] = rec_time(o)
    welds = sorted([(t1, run, starts[run][0].get("item_id"), starts[run][1]) for run, (fo, t1) in finish.items()
                    if run in starts and starts[run][0]["payload"]["operation_code"] == "W2"
                    and fo["payload"].get("result") == "completed"])
    for t1, run, it, t0 in welds:
        if run in prep:
            add("WW", it, prep[run], t0, round((t0 - prep[run]).total_seconds() / 3600, 1), "h", {"run": run})
    kt3 = defaultdict(list)
    xr = defaultdict(list)
    for o in rows:
        p = o["payload"] if isinstance(o["payload"], dict) else {}
        lb = o["_sim"]["label"]
        if o["event_type"] == "inspection.result" and lb.startswith("kt3/"):
            kt3[lb.split("/")[1]].append(rec_time(o))
        if o["event_type"] == "inspection.result" and p.get("method") == "xray" and p.get("checkpoint_id") == "CP-WELD" \
                and o.get("item_id"):
            xr[o["item_id"]].append(rec_time(o))
    for t1, run, it, t0 in welds:
        if kt3.get(run):
            c = min(kt3[run])
            add("W3", it, t1, c, int((floor_min(c) - floor_min(t1)).total_seconds() / 60), "min", {"run": run})
        nxt_weld = min([w[0] for w in welds if w[2] == it and w[0] > t1], default=None)
        xs = [x for x in xr.get(it, []) if x >= t1 and (nxt_weld is None or x < nxt_weld)]
        if xs:
            add("W4", it, t1, min(xs), round((min(xs) - t1).total_seconds() / 3600, 1), "h", {"run": run})

    # ---- решение по несоответствию
    conf = {}
    for s in steps:
        if s["action"] == "decision.signal_confirmed" and s["subject"].get("nc_ref") and s.get("expect", "accepted") == "accepted":
            conf[s["subject"]["nc_ref"]] = (t_msk(s["at"]), s["subject"].get("items") or [s["subject"].get("item_id")])
    decided = {}
    for s in steps:
        if s["action"] == "decision.disposition_set" and s.get("expect", "accepted") == "accepted":
            refs = s["subject"].get("nc_refs") or [s["subject"].get("nc_ref")]
            items = set(s["subject"].get("items") or [s["subject"].get("item_id")])
            for nc, (tc, its) in conf.items():
                if nc in decided:
                    continue
                if nc in refs or (items & set(its) and any(r and r.endswith(("G1", "14G")) for r in refs)):
                    decided[nc] = t_msk(s["at"])
    for nc, (tc, its) in conf.items():
        if nc in decided:
            add("N2", nc, tc, decided[nc], working_days_between(tc.date(), decided[nc].date(), off), "wd")
    return out, unreceived


def delays_in(meas, unreceived, win, norms):
    fr, to = dt.datetime.fromisoformat(win["from"]), dt.datetime.fromisoformat(win["to"])
    inside = [m for m in meas if fr <= m["to"] <= to]
    d = [m for m in inside if m["level"] == "delay"]
    # не принятые к концу окна: в отправке указана кладовая назначения (to_warehouse) — место известно,
    # розыска (таймер «нет скана места 4 ч») нет; изделие стоит в очереди перед цехом — это не задержка шага
    before = [m for m in meas if m["to"] < fr and m["level"] == "delay"]
    return inside, d, before


def escalation_episodes(gate_meas):
    """Эпизоды эскалации на узле: связная цепочка ожиданий (очередь не пустела); уровень — самое раннее срабатывание
    среди изделий эпизода; закрытие — когда очередь пуста (конец цепочки). Ключ — узел и время первого срабатывания."""
    out = {}
    by_node = defaultdict(list)
    for m in gate_meas:
        by_node[m["step"]].append(m)
    for node, ms in by_node.items():
        chain = []
        for m in sorted(ms, key=lambda m: (m["counted_from"], m["to"])) + [None]:
            if m is not None and chain and m["counted_from"] <= max(x["to"] for x in chain):
                chain.append(m)
                continue
            if chain:
                lv = {}
                for x in chain:
                    for k, t in x.get("escalation", {}).items():
                        lv[k] = min(lv.get(k, t), t)
                if lv:
                    out["%s@%s" % (node, fmt(min(lv.values())))] = {"levels": lv, "closed_at": max(x["to"] for x in chain)}
            chain = [m] if m is not None else []
    return out


def fmt(t):
    return t.strftime("%Y-%m-%dT%H:%M:00+03:00")


def summarize(d):
    out = []
    for m in d:
        x = {"step": m["step"], "object": m["object"], "value": m["value"], "unit": m["unit"]}
        if "presentation_no" in m:
            x["presentation_no"] = m["presentation_no"]
        out.append(x)
    return sorted(out, key=lambda x: (x["step"], x["object"], x.get("presentation_no", 0)))


def compute():
    norms = yaml.safe_load(open(NORMS, encoding="utf-8"))
    res = {}
    for name, stream, scen in (("main_story", "main-story.jsonl", "main-story.yaml"), ("S13", "S13.jsonl", "S13.yaml")):
        rows, steps = load_run(stream, scen)
        meas, unrec = measure(rows, steps, norms)
        win = norms["meta"]["scenario_windows"][name]
        inside, d, before = delays_in(meas, unrec, win, norms)
        esc = escalation_episodes([m for m in inside if m.get("gate")])
        res[name] = dict(norms=norms, meas=meas, inside=inside, delays=d, before=before, escalations=esc, win=win)
    return res


def expected_values():
    src = os.path.normpath(os.path.join(BASE, "..", "expected", "scenarios-flange-expected.yaml"))
    d = yaml.safe_load(open(src, encoding="utf-8"))
    ex = {}
    for sc in d["scenarios"].values():
        for cp in sc.get("checkpoints", []):
            for e in cp["expect"]:
                ex[e["id"]] = e
        for e in sc.get("must_not", []) or []:
            ex[e["id"]] = e
    return ex


def check(err):
    res = compute()
    ex = expected_values()
    lines = []
    for name, r in res.items():
        eid = r["win"]["expect_id"]
        e = ex.get(eid)
        got = summarize(r["delays"])
        if not e:
            err("нормы: нет утверждения %s" % eid)
            continue
        want = e["value"]["delays"]
        if sorted(json.dumps(x, sort_keys=True) for x in want) != sorted(json.dumps(x, sort_keys=True) for x in got):
            err("нормы %s: задержки по данным %s ≠ ожидания %s %s" % (name, got, eid, want))
        want_esc = e["value"].get("escalations") or {}
        got_esc = {k: {"levels": {lv: fmt(t) for lv, t in v["levels"].items()}, "closed_at": fmt(v["closed_at"])}
                   for k, v in r["escalations"].items()}
        if want_esc != got_esc:
            err("нормы %s: эскалации по данным %s ≠ ожидания %s %s" % (name, got_esc, eid, want_esc))
        n_steps = len({m["step"] for m in r["inside"]})
        lines.append("%s: измерений %d по %d шагам, задержек %d, эскалаций %d" % (
            name, len(r["inside"]), n_steps, len(got), len(got_esc)))
    # S13-10: Д-1 «ждёт подписи на ЗТ» за Пт 25.09 — из тех же измерений
    r = res["S13"]
    zt3 = [m for m in r["inside"] if m["step"] == "W5"]
    ws = sorted(m["value"] for m in zt3)
    if ws:
        med = (ws[len(ws) // 2 - 1] + ws[len(ws) // 2]) / 2 if len(ws) % 2 == 0 else ws[len(ws) // 2]
        mx = max(zt3, key=lambda m: m["value"])
        got = {"decisions": len(ws), "over_due": sum(1 for w in ws if w > 60), "median_min": med, "max_min": mx["value"],
               "max_item": mx["object"], "over_norm_sum_min": sum(w - 30 for w in ws if w > 30)}
        want = {k: v for k, v in ex["S13-10"]["field"].items() if k in got}
        if got != want:
            err("нормы: Д-1 S13 по данным %s ≠ S13-10 %s" % (got, want))
    # в файле норм — у каждого шага ключ узла схемы, источник и пометка
    bpmn = open(os.path.normpath(os.path.join(BASE, "..", "..", "..", "docs", "process", "flange-process-v0.3.2.bpmn")), encoding="utf-8").read()
    for k, s in res["S13"]["norms"]["steps"].items():
        if ('id="%s"' % k) not in bpmn:
            err("нормы: ключа %s нет среди узлов flange-process.bpmn" % k)
        if not s.get("source") or s.get("project_assumption") is not True or "delay_after" not in s:
            err("нормы: у шага %s нет источника, пометки «проектное допущение» или порога задержки" % k)
    return lines, res


if __name__ == "__main__":
    res = compute()
    for name, r in res.items():
        print("==", name, r["win"]["from"], "…", r["win"]["to"])
        print("  задержки:", summarize(r["delays"]))
        for k, v in r["escalations"].items():
            print("  эскалация", k, {lv: fmt(t) for lv, t in v["levels"].items()}, "закрыта", fmt(v["closed_at"]))
        att = [m for m in r["inside"] if m["level"] == "attention"]
        print("  внимание:", [(m["step"], m["object"], m["value"]) for m in att])
        print("  до окна (предыстория), задержки:", summarize(r["before"]))
        cnt = defaultdict(int)
        for m in r["inside"]:
            cnt[m["step"]] += 1
        print("  измерений по шагам:", dict(cnt))
