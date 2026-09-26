#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Проверки набора данных имитатора. Запуск: python3 tools/check.py  (код выхода 0 — всё цело).

 1. Каждая строка каждого JSONL — валидный JSON; обязательные поля конверта на месте (кроме намеренно испорченных).
 2. Все YAML разбираются.
 3. event_id уникален внутри источника; повторы — только намеренные дубли (тот же event_id, source_seq и payload)
    и один намеренный конфликт (тот же event_id и source_seq, другой payload).
 4. source_seq строго растёт по источнику в порядке доставки (дубли исключены); в прогонах — без пропусков,
    кроме намеренных разрывов из streams/_manifest.yaml.
 5. Технические ID — ASCII (ключи JSON, event_id, event_type, source_id, item_id, lot_id, *_id, метки, коды).
 6. Ссылки на справочники: источники, люди, оборудование, партии, типы событий и действия шагов людей.
 7. Время: формат RFC 3339 UTC с мс; received_at ≥ occurred_at.
 8. Инварианты сценариев (числа из scenarios-flange-expected.yaml): пачка ИС-2 1240 / 312 / 928, потеря 35,
    EV-WS2-0412, ток эталона 158–163 А, карантин S12 = 3, S06 и др.
 9. Линия: line_id только из справочника линий; у источников поста — линия поста; у событий по фланцу
    с момента запуска — линия поста его последней сварки (до первой — по плану); там, где линия не применима, её нет.
10. Ожидания: ID утверждений уникальны; «выпуск годного» (release_good) — только по изделиям с подписанной ЗТ-6.
11. 1С (erp_outbox_expected в I1, S13, S10B): release_good — только после ЗТ-6; «возврат из брака в производство» —
    только после повторной ЗТ; у каждого сообщения есть ответ, последний — «подтверждено».
12. scenarios-flange.md §6.3: нет строк «Принят после переделки» и «Заблокирован».
13. Экземпляры процессов (схема v0.2, README §2): _sim.process / process_ref — из справочника процессов;
    изделие начинается с item.registered (Е-09), заказ — с erp.order_received, партия — с erp.batch_received;
    кольцо (V6) выдаётся после приёмки фланца сварочным цехом и до скана WJ; скан WJ (Е-41) — до подготовки кромок W1;
    комплект (V7) — после приёмки сборкой (в S13 — заранее по сменному заданию, так схема допускает).
14. Счётчики событий по потокам — печатаются для README.
Волна 2:
15. S15 — сдвиг часов КИМ-1: ровно 5 строк с _sim.noise = clock_skew:+7m (только у них occurred_at > received_at);
    исправленное время каждой записи — до подписи ЗТ-2, неисправленное — после (ловушка для «пересмотрите» есть);
    после синхронизации часов (шаг S15/clock-synced) сдвига нет.
16. S16 — запись пульта ЧПУ без номера выполнения и без исполнителя лежит в перекрытии двух выполнений по отметкам терминала.
17. S17 — Е-21 «проверка пропущена» по F-006, затем Е-23 «проверка выполнена», и только потом ЗТ-СР.
18. Установка оборудования (ред-тим, строка 20): у каждой сборки с крышкой — выполнение EQ: скан клапана, затяжка J-2,
    контровка — до ЗТ-4; у ЗТ-4 в основании журнал ключа по J-2.
19. Перемаркировка (строка 34): у каждой заготовки до мехобработки — бирка (не код); после мехобработки — item.marked
    с кодом DataMatrix до КИМ и до ЗТ-2; ЗТ-2 сверяет маркировку.
20. S09 — три атаки подмены; разбор (S09/resolved) раньше любой ЗТ по затронутым изделиям.
21. S14 — три подреза, все у одного сварщика и на обоих постах; у второго сварщика чисто; ток в уставке; «ошибка исполнителя»
    записана только после объяснения работника (попытка раньше — отказ); остановки источников нет; в 1С — 3 «перевода в брак (переделка)».
Схема v0.3.1 (решения пользователя 26.09):
22. Плоскость после сварки (узел WP): у каждой ЗТ-3 во всех прогонах в основании — запись плоскости по последней сварке,
    сделанная раньше подписи; в главной истории все в допуске.
23. Выборочный рентген колец на входе (узел VX): у каждой партии колец — ровно одна запись, кольцо из партии, до ЗТ-1 и в её
    основании; у П-117 в выборке К-104 (чисто), а пора — в К-101 (после сварки) и К-105: «выборка чистая ≠ партия чистая»;
    К-104 на складе повторно не просвечивают.
24. Фон S08: по одному сигналу на скол, вмятину, царапину и геометрию — фланцы фонового заказа (не Ф-221…Ф-224), до 17.09;
    у каждого решение ОТК; скол и геометрия подтверждены (ЗТ-2 «доработка по ТП» → ЗТ-2 №2 «годно», в 1С — ничего сверх обычного),
    вмятина и царапина отклонены с причиной.
25. В главной истории есть сигналы всех пяти классов автора (справочник defect-classes.yaml → author_class).
26. MES (mes_outbox_expected в I1, S10B, S14): по объекту сообщения чередуются «не выдавать» → «можно выдавать»; «не выдавать» —
    в момент удержания (сигнал: наблюдение + 1 мин; круг — время круга); каждое «можно выдавать» — по принятому шагу человека
    в ту же минуту; после «списать» и «вернуть» — нет; в сообщениях нет гипотез и причин; у каждого изделия с сигналом
    (шаг подтверждения или отклонения) есть «не выдавать»; итог главной истории — 43 / 41, остались F-021 и LOT-R-117.
Нормы, плитки, материалы, карта дефицита (26.09):
27. Нормы шагов (process/step-norms.yaml, tools/norms.py): ключи — узлы BPMN v0.3.1, у каждого шага источник, пометка
    «проектное допущение» и порог задержки; по нормам главная история и S13 дают ровно те задержки и эскалации,
    что в ожиданиях S07-12 и S13-16; Д-1 S13 из тех же измерений совпадает с S13-10.
28. Материалы (definitions/reference/media.yaml): у каждого результата камеры и рентгена и у события OperatorVision
    ссылки есть (пустых evidence_refs нет ни в одном потоке); media_id — из справочника, вид совпадает; снимков изделия
    нет; у иллюстраций — источник, лицензия, пометка «не относится к изделию»; S03-16, S03-17, S11-06 — по данным.
29. Карта дефицита данных (definitions/reference/data-gaps.yaml, tools/gaps.py): записи случаев есть в данных, цену
    считает скрипт; рейтинг = S07-13, случаи S14 = S14-32, у S13 случаев нет (S13-17).
30. Четыре плитки стола руководителя (tools/tiles.py): S05-43…S05-46, S14-30 — по данным; у выводов всех прогонов
    полные основания.
31а. Обязательные показатели О-1…О-11 после сверки аналитики (tools/metrics.py, deliverables/metrics-audit.md, А-1…А-20):
    О-1, О-2 раздельно, лесенка ④ ≤ О-2, О-4 по видам, корзины О-5 и О-6, О-7 по участку, исполнителю, смене и изделию,
    корзины О-9 (с «приостановлено по качеству»), О-10 по станкам и «мало сопоставимых работ» в S14, журнал О-11,
    неизменность незатронутых показателей после опоздавшей пачки, итог по изделию, повторяемость (Д-6) — по данным.
31. «Тающая область»: у каждой исключённой детали свои основания в данных, и она названа в утверждении «почему?»
    (RS-01 — S05-47, RS-02 — S02-06, RS-14 — S14-31).
Нестандартные случаи S18–S22 (схема v0.3.2; отдельные прогоны S18…S21, S22 — продолжение S10B):
32. S18 — старт сварки С-03 после конца его удостоверения (по qualifications.yaml) отклонён: у попытки нет конца, кадров
    и сводок; сварил С-02 с действующим удостоверением после переназначения мастером; записи ключа КЛ-1 позже конца его
    поверки (equipment.yaml) в основании принятой ЗТ-4 нет, попытка ЗТ-4 на них — отказ; основание — контрольный КЛ-2.
33. S19 — ревизия «В» из КОМПАС после начала задела; задел Б придержан (MES) до решения; решения по каждому: доработать
    (Ф-039: новое выполнение со ссылкой, КИМ по D2 = 252, ЗТ-2 №2 с ревизией «В»), по разрешению с подписью ПЗ (Ф-037),
    списать (Ф-038 — в 1С одно «перевод в брак (списание)»); Ф-040 запущен по «В»; для «В» КТ-2 — ручной контроль, камеры нет.
34. S20 — скан «не то изделие» и «код не читается» раньше любой работы по этим фланцам на посту; сварка — только после
    выяснения; история не смешана (записи выполнения — по своему фланцу); ручной ввод подтверждён вторым человеком;
    перемаркировка — до ЗТ-3, ЗТ-3 сверяет маркировку.
35. S21 — К-107 выдано до претензии и вварено в Ф-034 (скан кольца, партия П-117); после претензии скан К-108 — «партия
    заблокирована», выдачи нет, кольцо в изоляторе; Ф-034 в круге по партии и исключён по рентгену; в 1С — возврат 9 колец.
36. S22 — всё после списания Ф-090 в S10B; скан на посту и «переделка» — отказ E_ITEM_SCRAPPED; разрешено только «образец»
    без возврата в маршрут; в S22 нет ни сварки, ни выдачи, ни ЗТ, ни сообщений в 1С.
«Следующее лучшее наблюдение» (камера), S04:
37. Пересъёмка — только по допущенному профилю карты (inspection-recipes.yaml → capture_profiles), номер ≤ лимита карты;
    у назначения есть причина неопределённости, профиль и почему он, предложение (система, не решение), кто назначил
    (человек, если нужен человек — например, снять прижим); кадр пересъёмки — после назначения, ссылается на прежний кадр
    того же изделия и той же КТ (prior_observation_event_id); для изделий, где рентген нашёл класс, который камере не виден
    (корень шва, внутренние поры), пересъёмки нет; S04-13 и S04-12 совпадают с картой и с шагом.
"""
import datetime as dt
import glob
import json
import os
import re
import sys
from collections import defaultdict, Counter

import yaml

BASE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TS = re.compile(r"^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}Z$")
ERR = []


def err(msg):
    ERR.append(msg)


def ascii_ok(s):
    return isinstance(s, str) and all(ord(c) < 128 for c in s)


ID_KEYS_SUFFIX = ("_id", "_ids", "_ref", "_label", "_labels")
ID_KEYS = {"event_id", "event_type", "schema_version", "source_id", "item_id", "lot_id", "correlation_id", "label",
           "gate", "zone", "zones", "zones_inspected", "checkpoint_id", "defect_type", "state", "code", "method",
           "phase", "inspection_result", "action_type", "step", "deviation", "expected_step", "tp_step",
           "from_shop", "to_shop", "from_warehouse", "to_warehouse", "to_station", "position", "seam_segment",
           "persona", "edge_agent", "sign_via", "noise", "scenarios", "for_items", "component_ids", "items",
           "sent_by", "received_by", "issued_by", "registered_by", "confirmed_by", "inspector_id", "operator_id",
           "equipment_id", "run", "result", "verdict", "reason", "container", "link_id", "binding_method",
           "presented_by", "presented_to", "program_id", "program_revision", "recipe_id", "camera_id",
           "calibration_id", "conclusion_no", "certificate_no", "heat_no", "message_id", "order_id", "order_ext_id",
           "lot_ext_id", "supplier_id", "item_type_id", "nomenclature_ext_id", "serial", "batch_id", "route_id",
           "operation_run_id", "operation_code", "station_id", "line_id", "rework_of_run_id", "rework_zones",
           "wire_lot_id", "welder_qualification_id", "gas_lot", "joint_id", "item_hint", "component_type_id",
           "assembly_item_id", "component_id", "purpose", "mode", "reason_code", "trigger", "kd_revision", "tp_id"}
FREE_TEXT_KEYS = {"text", "note", "description", "details", "observed_action", "designation", "reason"}


def walk_ascii(obj, path, where):
    if isinstance(obj, dict):
        for k, v in obj.items():
            if not ascii_ok(k):
                err("%s: не-ASCII ключ %r в %s" % (where, k, path))
            if k in FREE_TEXT_KEYS and k != "reason":
                continue
            if k in ID_KEYS or k.endswith(ID_KEYS_SUFFIX):
                vals = v if isinstance(v, list) else [v]
                for x in vals:
                    if isinstance(x, str) and not ascii_ok(x):
                        # reason — ASCII-код (quality_block, no_part …) либо русская фраза-обоснование: фразы допускаем
                        if k == "reason" and " " in x:
                            continue
                        err("%s: не-ASCII значение ID-поля %s.%s = %r" % (where, path, k, x))
            walk_ascii(v, path + "." + k, where)
    elif isinstance(obj, list):
        for i, x in enumerate(obj):
            walk_ascii(x, path + "[%d]" % i, where)


def load_ref():
    ref = {}
    d = os.path.join(BASE, "definitions", "reference")
    for p in sorted(glob.glob(os.path.join(d, "*.yaml"))):
        ref[os.path.basename(p)] = yaml.safe_load(open(p, encoding="utf-8"))
    sources = {s["source_id"] for s in ref["sources.yaml"]["sources"]}
    people = {p["id"] for p in ref["people.yaml"]["people"]}
    roles = {r["id"] for r in ref["people.yaml"]["roles"]}
    equipment = {e["id"] for e in ref["equipment.yaml"]["equipment"]}
    lots = {l["id"] for l in ref["lots.yaml"]["lots"]}
    stream_types = {t["type"] for t in ref["event-types.yaml"]["stream"]}
    actions = {t["action"] for t in ref["event-types.yaml"]["human_step"]}
    lines = {l["id"] for l in ref["sites.yaml"]["lines"]}
    station_line = {s["id"]: s["line_id"] for s in ref["sites.yaml"]["stations"] if s.get("line_id")}
    return dict(sources=sources, people=people, roles=roles, equipment=equipment, lots=lots,
                stream_types=stream_types, actions=actions, lines=lines, station_line=station_line)


def t_utc(s_):
    """RFC 3339 → datetime (UTC или со смещением)."""
    s_ = s_.replace("Z", "+00:00")
    return dt.datetime.fromisoformat(s_)


def check_lines(fn, rows, ref):
    """Правило линии (README §2): источники поста — линия поста; фланец с запуска — линия последней сварки."""
    post = {}
    for src, st in (("TERM-WS-1", "ST-WC-P1"), ("WS-1", "ST-WC-P1"), ("TERM-WS-2", "ST-WC-P2"), ("WS-2", "ST-WC-P2")):
        post[src] = ref["station_line"][st]
    reg, runs = {}, defaultdict(list)
    uniq = [o for o in rows if not (o["_sim"].get("noise") or "").startswith(("duplicate", "conflict"))]
    for o in uniq:
        it = o.get("item_id")
        if o["event_type"] == "item.registered" and it:
            reg.setdefault(it, o["occurred_at"])
        pl = o.get("payload") if isinstance(o.get("payload"), dict) else {}
        if o["event_type"] == "operation.started" and it and pl.get("operation_code") == "W2":
            runs[it].append((o["occurred_at"], ref["station_line"][pl["station_id"]]))
    for v in runs.values():
        v.sort()
    n_with = 0
    for o in rows:
        ln = o.get("line_id")
        if ln is not None:
            n_with += 1
            if ln not in ref["lines"]:
                err("%s: line_id %s нет в справочнике линий" % (fn, ln))
        src = o["source_id"]
        if src in post:
            if ln != post[src]:
                err("%s: у источника поста %s линия %s ≠ %s (%s)" % (fn, src, ln, post[src], o["_sim"]["label"]))
            continue
        it = o.get("item_id")
        if it and it.startswith("F-") and it in reg and o["occurred_at"] >= reg[it] and runs.get(it):
            want = runs[it][0][1]
            for (t_, l_) in runs[it]:
                if t_ <= o["occurred_at"]:
                    want = l_
            if ln != want:
                err("%s: %s %s — линия %s, ожидалась %s" % (fn, it, o["_sim"]["label"], ln, want))
        elif ln is not None:
            err("%s: линия %s там, где она не применима (%s)" % (fn, ln, o["_sim"]["label"]))
    return n_with


ITEM_RE = re.compile(r"^(F-\d{3}|R-\d{3}|C-\d{3}|CS-\d{2,3})$")
PROC_RE = re.compile(r"^(order/ORD-\d{4}|lot/LOT-[A-Z]+-\d+|item/F-\d{3}|incident/RS-\d{2}|review/ZT-\d/F-\d{3}|nc/NC-[A-Z0-9]+"
                     r"|signal/SG-[A-Za-z0-9.-]+|cause/RS-\d{2}|equipment/[A-Za-z0-9.-]+)$")   # v0.3: сигнал, разбор причин, сбой


def check_processes(fn, rows, steps, s13=False):
    """Экземпляры процессов схемы v0.2 и порядок выдач (README §2). Возвращает сводку."""
    uniq = [o for o in rows if not (o["_sim"].get("noise") or "").startswith(("duplicate", "conflict"))]
    uniq.sort(key=lambda o: (o["occurred_at"], o["_sim"]["gen_no"]))
    first = {}
    per = Counter()
    for o in uniq:
        pk = o["_sim"].get("process")
        if pk is None:
            continue
        per[pk.split("/")[0]] += 1
        if pk not in first:
            first[pk] = o
    starts = {"item": "item.registered", "order": "erp.order_received", "lot": "erp.batch_received"}
    for pk, o in first.items():
        kind = pk.split("/")[0]
        if kind in starts and o["event_type"] != starts[kind]:
            err("%s: экземпляр %s начинается с %s (%s), а не с %s" % (fn, pk, o["event_type"], o["_sim"]["label"], starts[kind]))
        if kind in ("incident", "review", "nc", "signal", "cause"):
            err("%s: %s в потоке — такие экземпляры бывают только в шагах людей" % (fn, pk))
    reg = {o["item_id"]: o["occurred_at"] for o in uniq if o["event_type"] == "item.registered"}
    for o in uniq:
        it = o.get("item_id")
        if it and it.startswith("F-") and o["_sim"].get("process") != "item/" + it:
            if (o["_sim"].get("process") or "").startswith("equipment/") and "item/" + it in (o["_sim"].get("process_links") or []):
                continue   # v0.3: пропуск проверки, ручное вмешательство — экземпляр «Сбой оборудования», изделие — связь
            if it in reg and o["occurred_at"] >= reg[it]:
                err("%s: %s после запуска не в экземпляре изделия (%s)" % (fn, it, o["_sim"]["label"]))
    # порядок выдач: V6 (кольцо) после приёмки фланца СЦ и до скана WJ; WJ до W1; V7 после приёмки СИЦ
    recv = defaultdict(dict)
    comp_item, link_at, prep_at = {}, {}, {}
    for o in uniq:
        pl = o["payload"] if isinstance(o["payload"], dict) else {}
        it = o.get("item_id")
        if o["event_type"] == "movement.received" and it:
            recv[it].setdefault((pl.get("from_shop"), pl.get("to_shop")), o["occurred_at"])
        if o["event_type"] == "assembly.component_linked" and pl.get("component_type_id") == "FL-100-RING":
            comp_item[pl["component_id"]] = pl["assembly_item_id"]
            link_at.setdefault(pl["assembly_item_id"], o["occurred_at"])
        if o["event_type"] == "operator.action" and (pl.get("details") or {}).get("step") == "edge_prep" and it:
            prep_at.setdefault(it, o["occurred_at"])
    n_v6 = n_v7 = n_v7_early = 0
    for o in uniq:
        if o["event_type"] != "batch.issued":
            continue
        pl = o["payload"]
        items = list(pl.get("for_items") or []) + [comp_item[c] for c in pl.get("component_ids") or []
                                                    if c in comp_item and comp_item[c] not in (pl.get("for_items") or [])]
        if pl["lot_id"].startswith("LOT-R-"):
            for it in items:
                n_v6 += 1
                r = recv[it].get(("MC", "WC"))
                if r is None or o["occurred_at"] < r:
                    err("%s: кольцо под %s выдано раньше приёмки фланца сварочным цехом (%s)" % (fn, it, o["_sim"]["label"]))
                if it in link_at and link_at[it] < o["occurred_at"]:
                    err("%s: скан WJ %s раньше выдачи кольца (%s)" % (fn, it, o["_sim"]["label"]))
        elif pl.get("to_station") == "ST-AC-ASM":
            for it in items:
                n_v7 += 1
                r = recv[it].get(("WC", "AC"))
                if r is None or o["occurred_at"] < r:
                    if s13 and "@0810" in o["_sim"]["label"]:
                        n_v7_early += 1     # S13: комплект выдан заранее по сменному заданию — V7 закроется при входе в сборку
                    else:
                        err("%s: комплект под %s выдан раньше приёмки сборкой (%s)" % (fn, it, o["_sim"]["label"]))
    for it, t in link_at.items():
        if it in prep_at and prep_at[it] < t:
            err("%s: %s — скан WJ после подготовки кромок W1" % (fn, it))
    # шаги людей
    nsteps = Counter()
    for s_ in steps:
        if "process_ref" not in s_:
            err("%s: шаг %s без process_ref" % (fn, s_["step_id"]))
            continue
        for pk in [s_["process_ref"]] + list(s_.get("process_links") or []):
            if pk is not None and not PROC_RE.match(pk):
                err("%s: шаг %s — странный экземпляр %r" % (fn, s_["step_id"], pk))
        nsteps[(s_["process_ref"] or "—").split("/")[0]] += 1
    return dict(lines_by_kind=dict(sorted(per.items())), instances=len(first), ring_issues_v6=n_v6,
                kit_issues_v7=n_v7, kit_issues_early_s13=n_v7_early, steps_by_kind=dict(sorted(nsteps.items())))
PERSON_KEYS = ("operator_id", "sent_by", "received_by", "issued_by", "registered_by", "confirmed_by", "inspector_id",
               "presented_by")


def check_stream(path, ref, manifest, kind):
    name = os.path.basename(path)
    lines = []
    with open(path, encoding="utf-8") as f:
        for n, raw in enumerate(f, 1):
            try:
                o = json.loads(raw)
            except Exception as ex:
                err("%s:%d: невалидный JSON: %s" % (name, n, ex))
                continue
            lines.append((n, o))
    seen = {}   # (src, event_id) -> (seq, payload)
    last_seq = {}
    seqs = defaultdict(list)
    for n, o in lines:
        where = "%s:%d" % (name, n)
        sim = o.get("_sim", {})
        noise = sim.get("noise") or ""
        for k in ("event_id", "event_type", "schema_version", "source_id", "source_seq", "occurred_at", "received_at",
                  "recorded_at", "payload", "_sim"):
            if k not in o:
                err("%s: нет поля конверта %s" % (where, k))
        if not TS.match(o.get("occurred_at", "")) or not TS.match(o.get("received_at", "")):
            err("%s: формат времени" % where)
        if o.get("received_at", "") < o.get("occurred_at", "") and not noise.startswith("clock_skew"):
            err("%s: received_at раньше occurred_at" % where)   # у S15 так и задумано: часы источника спешат
        if o.get("recorded_at") is not None:
            err("%s: recorded_at должен быть null (ставит ядро)" % where)
        walk_ascii({k: v for k, v in o.items() if k != "_sim"}, "", where)
        walk_ascii({"label": sim.get("label"), "noise": noise, "persona": sim.get("persona"),
                    "scenarios": sim.get("scenarios"), "process_id": sim.get("process"),
                    "process_ids": sim.get("process_links")}, "_sim", where)
        for pk in [sim.get("process")] + list(sim.get("process_links") or []):
            if pk is not None and not PROC_RE.match(pk):
                err("%s: странный экземпляр процесса %r" % (where, pk))
        if "process" not in sim:
            err("%s: нет _sim.process (экземпляр процесса или null)" % where)
        src = o.get("source_id")
        if src not in ref["sources"]:
            err("%s: неизвестный source_id %s" % (where, src))
        if o.get("event_type") not in ref["stream_types"]:
            err("%s: тип %s нет в event-types.yaml/stream" % (where, o.get("event_type")))
        it = o.get("item_id")
        if it is not None and not ITEM_RE.match(it):
            err("%s: странный item_id %s" % (where, it))
        if o.get("lot_id") and o["lot_id"] not in ref["lots"]:
            err("%s: неизвестная партия %s" % (where, o["lot_id"]))
        pl = o.get("payload") or {}
        if isinstance(pl, dict):
            for k in PERSON_KEYS:
                v = pl.get(k)
                if v and v not in ref["people"]:
                    err("%s: неизвестный человек %s=%s" % (where, k, v))
            eq = pl.get("equipment_id")
            if eq and eq not in ref["equipment"]:
                err("%s: неизвестное оборудование %s" % (where, eq))
            if pl.get("lot_id") and pl["lot_id"] not in ref["lots"]:
                err("%s: неизвестная партия в payload %s" % (where, pl["lot_id"]))
        if sim.get("persona") and sim["persona"] not in ref["people"]:
            err("%s: неизвестная персона %s" % (where, sim["persona"]))
        key = (src, o.get("event_id"))
        seq = o.get("source_seq")
        if key in seen:
            pseq, ppl = seen[key]
            if pseq != seq:
                err("%s: повтор event_id с другим source_seq" % where)
            if noise.startswith("duplicate"):
                if ppl != pl:
                    err("%s: дубль с другим содержимым" % where)
            elif noise.startswith("conflict"):
                if ppl == pl:
                    err("%s: конфликт без отличия содержимого" % where)
            else:
                err("%s: неожиданный повтор event_id %s у %s" % (where, o.get("event_id"), src))
            continue
        if noise.startswith("duplicate") and kind != "injection":
            err("%s: строка помечена дублем, но оригинала раньше нет" % where)
        seen[key] = (seq, pl)
        if kind in ("run", "slice"):
            if src in last_seq and seq <= last_seq[src]:
                err("%s: source_seq не растёт у %s: %s после %s" % (where, src, seq, last_seq[src]))
            last_seq[src] = seq
            seqs[src].append(seq)
    # непрерывность в прогонах
    if kind == "run":
        allowed = defaultdict(set)
        for s in manifest:
            if s["file"] == name:
                for gp in s.get("gaps", []) or []:
                    allowed[gp["source_id"]] |= set(range(gp["source_seq_from"], gp["source_seq_to"] + 1))
        for src, lst in seqs.items():
            full = set(range(1, max(lst) + 1))
            missing = full - set(lst)
            unexpected = missing - allowed[src]
            if unexpected:
                err("%s: пропуски source_seq у %s без объявления: %s…" % (name, src, sorted(unexpected)[:5]))
            if allowed[src] - missing:
                err("%s: объявленный разрыв у %s не совпадает" % (name, src))
    return lines


def _t(s_):
    return t_utc(s_)


def _steps_by_label(d):
    return {s_["label"]: s_ for s_ in d.get("human_steps") or []}


def wave2_checks(data, defs, ms):
    """Проверки волны 2 (пункты 15–21 в шапке)."""
    msk = dt.timezone(dt.timedelta(hours=3))
    by_label = {o["_sim"]["label"]: o for o in ms if not (o["_sim"].get("noise") or "").startswith("duplicate")}
    steps = _steps_by_label(defs["main-story.yaml"])
    # 15. S15 — сдвиг часов
    skew = [o for o in ms if (o["_sim"].get("noise") or "").startswith("clock_skew")]
    labs = sorted(o["_sim"]["label"] for o in skew)
    if labs != ["cmm/F-007", "cmm/F-008", "cmm/F-009", "cmm/F-010", "cmm/F-011"] or {o["source_id"] for o in skew} != {"CMM-1"}:
        err("S15: строки со сдвигом часов %s" % labs)
    sync = steps.get("S15/clock-synced")
    if not sync:
        err("S15: нет шага синхронизации часов")
    for o in skew:
        d_ = (_t(o["occurred_at"]) - _t(o["received_at"])).total_seconds()
        if d_ != 7 * 60 - 2:
            err("S15: %s — расхождение %s с, ожидалось 418" % (o["_sim"]["label"], d_))
        it = o["item_id"]
        zt = steps.get("ZT-2/" + it)
        corrected = _t(o["occurred_at"]) - dt.timedelta(minutes=7)
        if not zt or not (corrected < _t(zt["at"]) < _t(o["occurred_at"])):
            err("S15: у %s нет ловушки «КИМ после подписи ЗТ-2» (исправленное %s, ЗТ-2 %s)" % (it, corrected, zt and zt["at"]))
        if sync and _t(o["received_at"]) > _t(sync["at"]):
            err("S15: сдвиг после синхронизации часов (%s)" % o["_sim"]["label"])
    cmm_seq = [o["source_seq"] for o in ms if o["source_id"] == "CMM-1"]
    if cmm_seq != sorted(cmm_seq):
        err("S15: порядок КИМ-1 по source_seq нарушен")
    # 16. S16 — неоднозначная привязка
    ev = by_label.get("S16/cnc-feed-override")
    if not ev:
        err("S16: нет записи пульта ЧПУ")
    else:
        pl = ev["payload"]
        if ev.get("item_id") or ev.get("correlation_id") or pl.get("operation_run_id") or pl.get("operator_id") is not None:
            err("S16: у записи пульта не должно быть изделия, выполнения и исполнителя")
        a0, a1 = by_label["MO-004-1.start"]["occurred_at"], by_label["MO-004-1.finish"]["occurred_at"]
        b0, b1 = by_label["MO-005-1.start"]["occurred_at"], by_label["MO-005-1.finish"]["occurred_at"]
        if not (a0 <= ev["occurred_at"] <= a1 and b0 <= ev["occurred_at"] <= b1):
            err("S16: запись пульта не в перекрытии выполнений MO-004-1 и MO-005-1")
        if not (by_label["cnc/F-005/running"]["occurred_at"] <= ev["occurred_at"]):
            err("S16: запись пульта раньше начала цикла станка")
    # 17. S17 — пропуск проверки
    sk, dn = by_label.get("S17/check-skipped"), by_label.get("S17/check-done")
    z41 = steps.get("ZT-SR/F-006")
    if not (sk and dn and z41):
        err("S17: нет отметки пропуска, выполненной проверки или ЗТ-СР по F-006")
    else:
        if sk["payload"]["action_type"] != "check_skipped" or sk["item_id"] != "F-006":
            err("S17: отметка пропуска не Е-21 по F-006")
        if not (sk["occurred_at"] < dn["occurred_at"] and _t(dn["occurred_at"]) <= _t(z41["at"])):
            err("S17: ЗТ-СР по F-006 раньше выполненной проверки")
    # 18–19. установка оборудования и перемаркировка — во всех прогонах
    for fn, sfn in (("main-story.jsonl", "main-story.yaml"), ("S13.jsonl", "S13.yaml"), ("S10B.jsonl", "S10B.yaml"),
                    ("S14.jsonl", "S14.yaml"), ("S18.jsonl", "S18.yaml"), ("S19.jsonl", "S19.yaml"), ("S20.jsonl", "S20.yaml"), ("S21.jsonl", "S21.yaml")):
        rows = [o for _, o in data[fn] if not (o["_sim"].get("noise") or "").startswith(("duplicate", "conflict"))]
        lb = {o["_sim"]["label"]: o for o in rows}
        st = _steps_by_label(defs[sfn])
        n_eq = n_rm = 0
        for o in rows:
            pl = o["payload"] if isinstance(o["payload"], dict) else {}
            it = o.get("item_id")
            if o["event_type"] == "operator.action" and (pl.get("details") or {}).get("step") == "cover_install":
                n = it.split("-")[1]
                need = ["EQ-%s-1.start" % n, "EQ-%s-1.link.FL-100-VALVE" % n, "torque/%s/J-2" % it, "EQ-%s-1.lockwire" % n,
                        "EQ-%s-1.finish" % n]
                if any(x not in lb for x in need):
                    err("%s: у сборки %s нет установки оборудования EQ (%s)" % (fn, it, [x for x in need if x not in lb]))
                    continue
                n_eq += 1
                tig = lb.get("AS-%s-1.tightening" % n)
                k4 = lb.get("kt4/" + it)
                if not tig or not k4 or not (tig["occurred_at"] < lb["EQ-%s-1.start" % n]["occurred_at"]
                                             and lb["EQ-%s-1.finish" % n]["occurred_at"] < k4["occurred_at"]):
                    err("%s: установка оборудования %s не между затяжкой и камерой КТ-4 (узел AO после A6)" % (fn, it))
                z4 = st.get("ZT-4/" + it)
                j2 = [x for x in ("torque/%s/J-2" % it, "torque-check/%s/J-2" % it) if x in (z4 or {}).get("params", {}).get("basis_labels", [])]
                if z4 and (not j2 or
                           _t(lb["EQ-%s-1.finish" % n]["occurred_at"]) > _t(z4["at"])):
                    err("%s: ЗТ-4 %s без основания по клапану или раньше установки" % (fn, it))
            if o["event_type"] == "item.marked" and it and it.startswith("F-"):
                car = pl.get("carrier", {}).get("type")
                if pl.get("reason") == "remark_after_machining":
                    n_rm += 1
                    fin = lb.get("cnc/%s/idle" % it)
                    km = lb.get("cmm/" + it)
                    z2 = st.get("ZT-2/" + it)
                    if car != "dpm_datamatrix" or not fin or o["occurred_at"] < fin["occurred_at"] or \
                            (km and km["occurred_at"] < o["occurred_at"] and not (km["_sim"].get("noise") or "").startswith("clock_skew")) or \
                            (z2 and _t(z2["at"]) < _t(o["occurred_at"])):
                        err("%s: перемаркировка %s не на своём месте (после станка, до КИМ и ЗТ-2)" % (fn, it))
                    if z2 and (z2["params"].get("marking_check") or {}).get("current") != "DM:" + it:
                        err("%s: ЗТ-2 %s не сверяет маркировку" % (fn, it))
                elif pl.get("reason") == "remark_unreadable":
                    pass   # S20: перемаркировка по процедуре — проверка в пункте 34
                elif car != "tag_qr":
                    err("%s: заготовка %s до мехобработки помечена не биркой, а %s" % (fn, it, car))
        machined = {o["item_id"] for o in rows if o["event_type"] == "operation.started"
                    and (o["payload"] or {}).get("operation_code") == "M1"}
        if n_rm != len(machined):
            err("%s: перемаркировок %d, а мехобработок %d" % (fn, n_rm, len(machined)))
        if fn not in ("S10B.jsonl", "S14.jsonl", "S19.jsonl", "S20.jsonl", "S21.jsonl") and n_eq == 0:   # до сборки не доходят
            err("%s: нет ни одной установки оборудования" % fn)
    # 20. S09 — три атаки и разбор
    tam = defs["main-story.yaml"].get("tamper_actions") or []
    meth = sorted(t_.get("method", "update_in_place") for t_ in tam)
    if meth != ["projection_update", "update_and_rechain", "update_in_place"]:
        err("S09: атаки подмены %s (ожидалось три разных)" % meth)
    res = steps.get("S09/resolved")
    if not res or any(_t(res["at"]) < _t(t_["at"]) for t_ in tam):
        err("S09: разбор подмен не после всех атак")
    elif any(s_["action"] == "decision.gate_passed" and s_["subject"].get("item_id") in ("F-015", "F-019")
             and _t(t_["at"]) < _t(s_["at"]) < _t(res["at"]) for s_ in defs["main-story.yaml"]["human_steps"] for t_ in tam):
        err("S09: ЗТ по изделию с подменённой записью раньше разбора")
    # 21. S14 — один сварщик
    r14 = [o for _, o in data["S14.jsonl"]]
    st14 = _steps_by_label(defs["S14.yaml"])
    starts = {o["payload"]["operation_run_id"]: o for o in r14 if o["event_type"] == "operation.started"
              and o["payload"].get("operation_code") == "W2" and (o.get("item_id") or "").startswith("F-")}
    bad = [o for o in r14 if o["event_type"] == "inspection.result" and o["payload"].get("method") == "camera"
           and o["payload"].get("checkpoint_id") == "CP-WELD" and o["payload"]["inspection_result"] != "no_defect_found"]
    runs_bad = {o["_sim"]["label"].split("/")[1] for o in bad}
    welders = {starts[r]["payload"]["operator_id"] for r in runs_bad}
    posts = {starts[r]["payload"]["equipment_id"] for r in runs_bad}
    if len(runs_bad) != 3 or welders != {"WLD-03"} or posts != {"WS-1", "WS-2"}:
        err("S14: подрезы %s, сварщики %s, посты %s" % (sorted(runs_bad), welders, posts))
    if any(d_.get("defect_type") != "undercut" for o in bad for d_ in o["payload"]["defects"]):
        err("S14: дефект не «подрез»")
    clean02 = [r for r, o in starts.items() if o["payload"]["operator_id"] == "WLD-02"]
    if len(clean02) != 6 or {starts[r]["payload"]["equipment_id"] for r in clean02} != {"WS-1", "WS-2"} or set(clean02) & runs_bad:
        err("S14: у второго сварщика не 6 чистых швов на обоих постах")
    cur = [o["payload"]["parameters"]["welding_current_a"] for o in r14
           if o["event_type"] == "machine.parameters" and o["source_id"] in ("WS-1", "WS-2") and (o["payload"].get("parameters") or None)]
    if not cur or min(c["min"] for c in cur) < 150 or max(c["max"] for c in cur) > 170:
        err("S14: ток вне уставки 160 ± 10 А")
    if any(o["event_type"] == "machine.state" and o["payload"].get("state") == "warning" for o in r14):
        err("S14: есть предупреждения источника — журналы должны быть в норме")
    ex, cz, rf = st14.get("S14/explanation"), st14.get("S14/cause"), st14.get("S14/cause-before-explanation-refused")
    if not (ex and cz and rf) or not (_t(rf["at"]) < _t(ex["at"]) < _t(cz["at"])) or rf["expect"] != "refused:E_EXPLANATION_REQUIRED" \
            or cz["params"].get("explanation_ref_label") != "S14/explanation":
        err("S14: «ошибка исполнителя» должна записываться только после объяснения работника (и отказ — раньше)")
    if any(s_["action"] == "process_hold.set" for s_ in defs["S14.yaml"]["human_steps"]):
        err("S14: источники останавливать не должны — журналы в норме")
    ob = defs["S14.yaml"].get("erp_outbox_expected") or []
    rw = sorted(x for r in ob if r["type"] == "scrap_transfer_rework" for x in r.get("items", []))
    if rw != ["F-304", "F-306", "F-310"] or sum(1 for r in ob if r["type"] == "scrap_transfer_rework") != 1:
        err("S14: в 1С «перевод в брак (переделка)» по %s (ожидалось одно сообщение на три изделия)" % rw)
    if sorted(r["item_id"] for r in ob if r["type"] == "rework_return_to_production") != ["F-304", "F-306", "F-310"]:
        err("S14: «возврат из брака в производство» — не по трём переделанным")
    wave3_checks(data, defs, ms, st14, r14)


def wave3_checks(data, defs, ms, st14, r14):
    """Сверка со схемой v0.3 (bpmn-v03-changes.md, «Нужно поправить в сценариях и данных»)."""
    order_ = lambda *labs: all(_t(st14[a]["at"]) < _t(st14[b]["at"]) for a, b in zip(labs, labs[1:]))
    # S14: путь схемы — инцидент «сварщик», групповое решение, разбор до закрытия
    need = ["S14/exclude-F-302", "S14/exclude-F-308", "S14/exclude-F-312", "S14/NC-14G", "S14/disposition-rework", "S14/close",
            "S14/explanation", "S14/cause", "S14/capa", "S14/capa-done", "S14/effective", "S14/closed"]
    miss = [x for x in need if x not in st14]
    if miss:
        err("S14: нет шагов пути схемы %s" % miss)
    else:
        if not order_("S14/explanation", "S14/cause", "S14/capa", "S14/capa-done", "S14/effective", "S14/closed"):
            err("S14: разбор причин не по порядку схемы (CA4 → CA5 → CA6 → CA7 → CA8 → CA9)")
        for it in ("F-302", "F-308", "F-312"):
            if not _t(st14["S14/exclude-" + it]["at"]) < _t(st14["ZT-3/" + it]["at"]):
                err("S14: %s — ЗТ-3 раньше исключения из круга RS-14" % it)
        if any(s_["action"] == "cause.confirmed" and s_["expect"] == "accepted" and s_["params"].get("operator_error")
               and _t(s_["at"]) < _t(st14["S14/explanation"]["at"]) for s_ in defs["S14.yaml"]["human_steps"]):
            err("S14: «ошибка исполнителя» записана раньше объяснения работника")
    lb14 = {o["_sim"]["label"]: o for o in r14}
    win = [o for o in r14 if o["event_type"] == "operation.started" and (o["payload"] or {}).get("operation_code") == "W2"
           and o.get("item_id", "").startswith("F-") and o["payload"]["operator_id"] == "WLD-03"
           and not o["payload"].get("rework_of_run_id") and "capa-done" and "S14/capa-done" in st14
           and _t(o["occurred_at"]) > _t(st14["S14/capa-done"]["at"])]
    if len(win) != 8:
        err("S14: в окне наблюдения %d новых швов С-03 (ожидалось 8)" % len(win))
    for o in win:
        run = o["payload"]["operation_run_id"]
        cams = [lb14.get("kt3/%s/CAM-WS-%d" % (run, k)) for k in (1, 2)]
        if not all(c and c["payload"]["inspection_result"] == "no_defect_found" for c in cams):
            err("S14: шов окна %s не чистый" % run)
    # выборка и контроль выборки перед каждой переделкой (WL1 → WL2 → WL3); согласие ГС-01 на 2-ю и 3-ю (S10B)
    for fn, sfn in (("main-story.jsonl", "main-story.yaml"), ("S10B.jsonl", "S10B.yaml"), ("S14.jsonl", "S14.yaml")):
        rows = [o for _, o in data[fn] if not (o["_sim"].get("noise") or "").startswith("duplicate")]
        lb = {o["_sim"]["label"]: o for o in rows}
        wl = defaultdict(list)
        for o in rows:
            p_ = o["payload"] if isinstance(o["payload"], dict) else {}
            if o["event_type"] == "operation.finished" and p_.get("operation_run_id", "").startswith("WL-"):
                chk = lb.get("wl2/" + p_["operation_run_id"])
                wl[o["item_id"]].append((o["occurred_at"], chk["occurred_at"] if chk else None))
        for o in rows:
            p_ = o["payload"] if isinstance(o["payload"], dict) else {}
            if o["event_type"] == "operation.started" and p_.get("operation_code") == "W2" and p_.get("rework_of_run_id"):
                it = o["item_id"]
                ok = [c for (f_, c) in wl.get(it, []) if c and f_ < o["occurred_at"] and c < o["occurred_at"]]
                if not ok:
                    err("%s: переделка %s без выборки и контроля выборки до заварки" % (fn, p_["operation_run_id"]))
    st10 = _steps_by_label(defs["S10B.yaml"])
    r10 = {o["_sim"]["label"]: o for _, o in data["S10B.jsonl"]}
    for k, run in ((2, "SV-090-3"), (3, "SV-090-4")):
        a = st10.get("S10B/chief-welder-approval-%d" % k)
        if not a or not (_t(r10["wl2/WL-090-%d" % k]["occurred_at"]) < _t(a["at"]) < _t(r10[run + ".start"]["occurred_at"])):
            err("S10B: согласие главного сварщика на %d-е исправление не между контролем выборки и заваркой" % k)
    # лаборатория на входе (VL) у заготовок и колец; поле «повреждения» у приёма цехом (Е-31)
    for fn in ("main-story.jsonl", "S13.jsonl", "S10B.jsonl", "S14.jsonl", "S18.jsonl", "S19.jsonl", "S20.jsonl", "S21.jsonl"):
        rows = [o for _, o in data[fn] if not (o["_sim"].get("noise") or "").startswith("duplicate")]
        labs = {o["_sim"]["label"] for o in rows}
        for o in rows:
            if o["event_type"] == "erp.batch_received" and o["payload"].get("item_type_id") in ("FL-100-FLANGE", "FL-100-RING"):
                if "iqc/%s/lab" % o["lot_id"] not in labs:
                    err("%s: у партии %s нет лабораторного контроля" % (fn, o["lot_id"]))
            if o["event_type"] == "movement.received" and o["payload"].get("damage") not in ("none", "found"):
                err("%s: у приёма цехом нет поля «повреждения» (%s)" % (fn, o["_sim"]["label"]))
    # S08: отклонение снимает удержание по правилу тем же решением
    s08 = _steps_by_label(defs["main-story.yaml"]).get("S08/reject")
    if not s08 or "Е-116" not in (s08["params"].get("hold_released") or ""):
        err("S08: отклонение сигнала не снимает удержание (Е-116)")
    # экземпляры v0.3: сигнал, разбор причин, сбой оборудования
    kinds = Counter()
    for fn, sfn in (("main-story.jsonl", "main-story.yaml"), ("S14.jsonl", "S14.yaml"), ("S10B.jsonl", "S10B.yaml")):
        for s_ in defs[sfn]["human_steps"]:
            kinds[(s_.get("process_ref") or "—").split("/")[0]] += 1
        for _, o in data[fn]:
            kinds[(o["_sim"].get("process") or "—").split("/")[0]] += 1
    for k in ("signal", "cause", "equipment"):
        if not kinds[k]:
            err("экземпляров «%s» в данных нет" % k)


def v031_checks(data, defs, ms):
    """Схема v0.3.1: плоскость, выборочный рентген колец, фон по классам автора, MES (пункты 22–26)."""
    # 22. плоскость
    for fn, sfn in (("main-story.jsonl", "main-story.yaml"), ("S13.jsonl", "S13.yaml"), ("S10B.jsonl", "S10B.yaml"),
                    ("S14.jsonl", "S14.yaml"), ("S18.jsonl", "S18.yaml"), ("S19.jsonl", "S19.yaml"), ("S20.jsonl", "S20.yaml"), ("S21.jsonl", "S21.yaml")):
        rows = [o for _, o in data[fn] if not (o["_sim"].get("noise") or "").startswith(("duplicate", "conflict"))]
        lb = {o["_sim"]["label"]: o for o in rows}
        welds = defaultdict(list)
        for o in rows:
            p_ = o["payload"] if isinstance(o["payload"], dict) else {}
            if o["event_type"] == "operation.finished" and str(p_.get("operation_run_id", "")).startswith("SV-") \
                    and (o.get("item_id") or "").startswith("F-"):
                welds[o["item_id"]].append((o["occurred_at"], p_["operation_run_id"]))
        for s_ in defs[sfn]["human_steps"]:
            if s_["action"] != "decision.gate_passed" or s_["subject"].get("gate") != "ZT-3":
                continue
            it = s_["subject"]["item_id"]
            prev = [r for (t_, r) in sorted(welds[it]) if _t(t_) < _t(s_["at"])]
            need = "flat/" + prev[-1] if prev else None
            fl = lb.get(need)
            if not fl or need not in (s_["params"].get("basis_labels") or []) or _t(fl["occurred_at"]) >= _t(s_["at"]):
                err("%s: ЗТ-3 %s без записи плоскости по %s до подписи" % (fn, s_["label"], need))
            elif fl["payload"]["characteristic"]["result"] != "in_tolerance" or fl["payload"]["characteristic"]["actual"] > \
                    fl["payload"]["characteristic"]["tol_plus"]:
                err("%s: плоскость %s вне допуска" % (fn, need))
    # 23. выборочный рентген колец на входе
    for fn, sfn in (("main-story.jsonl", "main-story.yaml"), ("S13.jsonl", "S13.yaml"), ("S10B.jsonl", "S10B.yaml"),
                    ("S14.jsonl", "S14.yaml"), ("S18.jsonl", "S18.yaml"), ("S19.jsonl", "S19.yaml"), ("S20.jsonl", "S20.yaml"), ("S21.jsonl", "S21.yaml")):
        rows = [o for _, o in data[fn] if not (o["_sim"].get("noise") or "").startswith("duplicate")]
        st = _steps_by_label(defs[sfn])
        for o in rows:
            if o["event_type"] == "erp.batch_received" and o["payload"].get("item_type_id") == "FL-100-RING":
                lot = o["lot_id"]
                xs = [x for x in rows if x["_sim"]["label"] == "iqc/%s/xr-sample" % lot]
                z1 = st.get("ZT-1/" + lot)
                if len(xs) != 1 or xs[0]["payload"].get("lot_id") != lot or xs[0]["payload"]["sample"]["size"] != 1:
                    err("%s: у партии колец %s нет одного выборочного рентгена" % (fn, lot))
                elif not z1 or _t(xs[0]["occurred_at"]) >= _t(z1["at"]) or xs[0]["_sim"]["label"] not in z1["params"].get("basis_labels", []):
                    err("%s: выборочный рентген %s не в основании ЗТ-1 или позже неё" % (fn, lot))
    lbm = {o["_sim"]["label"]: o for o in ms if not (o["_sim"].get("noise") or "").startswith("duplicate")}
    smp = lbm.get("iqc/LOT-R-117/xr-sample")
    pore = [o for o in ms if o["event_type"] == "inspection.result" and o["payload"].get("method") == "xray"
            and any(d_.get("defect_type") == "base_metal_pore" for d_ in o["payload"].get("defects") or [])]
    if not smp or smp["item_id"] != "R-104" or smp["payload"]["inspection_result"] != "no_defect_found":
        err("S02: в выборке П-117 должно быть чистое К-104")
    elif sorted(o.get("item_id") for o in pore) != ["F-019", "R-105"] or not all(smp["occurred_at"] < o["occurred_at"] for o in pore):
        err("S02: пора должна найтись после чистой выборки: у F-019 (кольцо R-101) и R-105")
    if any(o["event_type"] == "inspection.result" and o.get("item_id") == "R-104" and o["payload"].get("method") == "xray"
           and o["_sim"]["label"] != "iqc/LOT-R-117/xr-sample"
           for o in ms):
        err("S02: К-104 просвечено повторно — оно уже было в выборке")
    # 24. фон S08: четыре класса
    steps = defs["main-story.yaml"]["human_steps"]
    bg = {"chip": ("F-203", "confirm"), "dimension_out_of_tolerance": ("F-219", "confirm"), "dent": ("F-206", "reject"),
          "scratch": ("F-213", "reject")}
    for cls, (it, outcome) in bg.items():
        sig = [o for o in ms if o.get("item_id") == it and "S08" in o["_sim"]["scenarios"] and
               any(d_.get("defect_type") == cls for d_ in (o["payload"].get("defects") or o["payload"].get("damage_details") or []))]
        if len(sig) != 1 or sig[0]["occurred_at"] >= "2026-09-17":
            err("S08 фон: сигнал «%s» по %s — %d строк или не до 17.09" % (cls, it, len(sig)))
            continue
        dec = [s_ for s_ in steps if s_["subject"].get("item_id") == it and s_["action"] in
               ("decision.signal_confirmed", "decision.signal_rejected") and sig[0]["_sim"]["label"] in s_["subject"].get("signal_basis_labels", [])]
        if len(dec) != 1 or dec[0]["action"] != "decision.signal_" + ("confirmed" if outcome == "confirm" else "rejected") \
                or not (dec[0]["params"].get("reason") or "").strip():
            err("S08 фон: решение ОТК по %s (%s) не то" % (it, cls))
        if outcome == "confirm":
            gz = [s_ for s_ in steps if s_["subject"].get("item_id") == it and s_["subject"].get("gate") == "ZT-2"]
            if [x["params"]["decision"] for x in gz] != ["return_for_rework", "accept"] or gz[1]["params"].get("presentation_no") != 2:
                err("S08 фон: у %s нет «доработка по ТП» → ЗТ-2 №2 «годно»" % it)
    ob = defs["I1.yaml"].get("erp_outbox_expected") or []
    for it in ("F-203", "F-219", "F-206", "F-213"):
        types = {r["type"] for r in ob if r.get("item_id") == it or it in (r.get("items") or [])}
        if types - {"issue_to_production", "transfer", "release_good"}:
            err("S08 фон: по %s в 1С ушло лишнее %s" % (it, types))
    if any(x in ("F-203", "F-219", "F-206", "F-213") for s_ in steps if s_["action"] == "risk_scope.narrowed"
           for x in s_["params"].get("excluded_items", [])):
        err("S08 фон: фоновые фланцы попали в круг инцидента")
    # 25. пять классов автора
    ref_dc = yaml.safe_load(open(os.path.join(BASE, "definitions", "reference", "defect-classes.yaml"), encoding="utf-8"))
    ac = {c["id"]: c.get("author_class") for c in ref_dc["classes"]}
    seen_ac = set()
    for o in ms:
        if not isinstance(o["payload"], dict):
            continue
        for d_ in (o["payload"].get("defects") or []) + (o["payload"].get("damage_details") or []):
            if ac.get(d_.get("defect_type")):
                seen_ac.add(ac[d_["defect_type"]])
    if seen_ac != {"weld", "chip", "dent", "scratch", "geometry"}:
        err("классы автора в главной истории: %s" % sorted(seen_ac))
    # 26. MES
    people = set()
    for dfn, sfn in (("I1.yaml", "main-story.yaml"), ("S10B.yaml", "S10B.yaml"), ("S14.yaml", "S14.yaml"),
                     ("S18.yaml", "S18.yaml"), ("S19.yaml", "S19.yaml"), ("S20.yaml", "S20.yaml"), ("S21.yaml", "S21.yaml")):
        rows_ = defs[dfn].get("mes_outbox_expected")
        if rows_ is None:
            err("%s: нет mes_outbox_expected" % dfn)
            continue
        st_by = {s_["label"]: s_ for s_ in defs[sfn]["human_steps"]}
        state = {}
        for r in rows_:
            obj = r.get("item_id") or r.get("lot_id")
            blob = json.dumps(r, ensure_ascii=False)
            if any(k in blob for k in ("hypothes", "cause", "operator_candidates", "frame")):
                err("%s: в сообщении MES №%s есть гипотезы или причины" % (dfn, r["msg_no"]))
            if r["type"] == "hold":
                if state.get(obj) == "hold":
                    err("%s: второе «не выдавать» по %s без снятия" % (dfn, obj))
                state[obj] = "hold"
                if r["set_by"] != "system":
                    err("%s: «не выдавать» ставит система" % dfn)
            else:
                if state.get(obj) != "hold":
                    err("%s: «можно выдавать» по %s без удержания" % (dfn, obj))
                state[obj] = "release"
                s_ = st_by.get(r["decision_step"])
                if not s_ or s_["expect"] != "accepted" or s_["at_msk"] != r["at_msk"] or s_["persona"] != r["decided_by"]:
                    err("%s: «можно выдавать» по %s не по шагу человека (%s)" % (dfn, obj, r["decision_step"]))
                elif s_["params"].get("disposition") in ("scrap", "return_to_supplier"):
                    err("%s: «можно выдавать» по решению «списать» или «вернуть»" % dfn)
        for s_ in defs[sfn]["human_steps"]:
            if s_["action"] in ("decision.signal_confirmed", "decision.signal_rejected") and s_["expect"] == "accepted" \
                    and not s_.get("optional") and s_["subject"].get("signal_basis_labels") and (s_["subject"].get("item_id") or "").startswith("F-"):
                it = s_["subject"]["item_id"]
                if not any(r["type"] == "hold" and r.get("item_id") == it and r["at_msk"] <= s_["at_msk"] for r in rows_):
                    err("%s: по %s есть сигнал (%s), а «не выдавать» в MES нет" % (dfn, it, s_["label"]))
    i1 = defs["I1.yaml"]
    if i1.get("mes_outbox_counts") != {"hold": 43, "release": 41} or i1.get("mes_held_at_end") != ["F-021", "LOT-R-117"]:
        err("I1: MES %s, остались %s (ожидалось 43 / 41, F-021 и LOT-R-117)" % (i1.get("mes_outbox_counts"), i1.get("mes_held_at_end")))
    r17 = [r for r in i1.get("mes_outbox_expected", []) if r.get("item_id") == "F-017"]
    st = _steps_by_label(defs["main-story.yaml"])
    if not r17 or r17[0]["type"] != "hold" or r17[0]["at_msk"] >= st["S03/confirm-NC-01"]["at_msk"]:
        err("S03: «не выдавать» по F-017 не раньше подтверждения сигнала")


# ----------------------------------------------------------------------------- 32–36. нестандартные случаи S18–S22
def ns_checks(data, defs, exv=None):
    ref_q = yaml.safe_load(open(os.path.join(BASE, "definitions", "reference", "qualifications.yaml"), encoding="utf-8"))
    ref_e = yaml.safe_load(open(os.path.join(BASE, "definitions", "reference", "equipment.yaml"), encoding="utf-8"))
    cert = {q["person"]: q["valid_until"] for q in ref_q["qualifications"] if q.get("kind") == "welder_certification"}
    verif = {e["id"]: (e.get("verification") or {}).get("valid_until") for e in ref_e["equipment"]}
    msk = lambda s_: t_utc(s_).astimezone(dt.timezone(dt.timedelta(hours=3)))
    rows = lambda fn: [o for _, o in data[fn]]
    lbs = lambda fn: {o["_sim"]["label"]: o for o in rows(fn)}
    stp = lambda sfn: _steps_by_label(defs[sfn])
    n = Counter()
    # 32. S18
    r, lb, st = rows("S18.jsonl"), lbs("S18.jsonl"), stp("S18.yaml")
    att = lb.get("S18/start-refused")
    if not att or att["payload"]["operator_id"] != "WLD-03" or \
            msk(att["occurred_at"]).date().isoformat() <= cert["WLD-03"]:
        err("S18: попытки сварки С-03 после конца удостоверения нет")
    else:
        run = att["payload"]["operation_run_id"]
        fins = [o for o in r if o["event_type"] == "operation.finished" and o["payload"].get("operation_run_id") == run]
        starts = [o for o in r if o["event_type"] == "operation.started" and o["payload"].get("operation_run_id") == run
                  and o is not att]
        ra = st.get("S18/reassign")
        if len(fins) != 1 or fins[0]["payload"]["operator_id"] != "WLD-02" or len(starts) != 1 \
                or starts[0]["payload"]["operator_id"] != "WLD-02" or not ra or not (att["occurred_at"] < ra["at"] < starts[0]["occurred_at"]):
            err("S18: сварку должен выполнить С-02 после переназначения; у отклонённой попытки не должно быть конца")
        elif msk(starts[0]["occurred_at"]).date().isoformat() > cert["WLD-02"]:
            err("S18: у С-02 удостоверение не действует на дату сварки")
        n["S18"] += 1
    tw1 = [o for o in r if o["source_id"] == "TW-1"]
    bad = [o["_sim"]["label"] for o in tw1 if msk(o["occurred_at"]).date().isoformat() <= verif["TW-1"]]
    if not tw1 or bad:
        err("S18: записи КЛ-1 должны быть после конца поверки (%s)" % bad)
    tw2 = [o for o in r if o["source_id"] == "TW-2"]
    if len(tw2) != 2 or any(msk(o["occurred_at"]).date().isoformat() > verif["TW-2"] for o in tw2):
        err("S18: контроль момента ключом КЛ-2 — две записи, ключ поверен")
    z4r, z4 = st.get("S18/ZT-4-refused"), st.get("ZT-4/F-042")
    tw1l = {o["_sim"]["label"] for o in tw1}
    if not z4r or not z4r["expect"].startswith("refused") or not (set(z4r["params"]["basis_labels"]) & tw1l):
        err("S18: попытки ЗТ-4 на записях КЛ-1 с отказом нет")
    if not z4 or z4["expect"] != "accepted" or set(z4["params"]["basis_labels"]) & tw1l or \
            not {o["_sim"]["label"] for o in tw2} <= set(z4["params"]["basis_labels"]) or \
            any(o["occurred_at"] > z4["at"] for o in tw2):
        err("S18: принятая ЗТ-4 должна опираться только на контроль КЛ-2")
    else:
        n["S18"] += 1
    ob = defs["S18.yaml"].get("erp_outbox_expected") or []
    if {x["type"] for x in ob} - {"issue_to_production", "transfer"}:
        err("S18: в 1С ушло лишнее")
    # 33. S19
    r, lb, st = rows("S19.jsonl"), lbs("S19.jsonl"), stp("S19.yaml")
    imp = lb.get("import/kompas-FL-100@V")
    reg = {o["item_id"]: o for o in r if o["event_type"] == "item.registered"}
    if not imp or imp["payload"]["version"] != "V" or imp["payload"]["change_notice"]["from_revision"] != "B":
        err("S19: нет импорта ревизии «В» с извещением")
    else:
        for it in ("F-037", "F-038", "F-039"):
            if reg[it]["payload"]["kd_revision"] != "B" or reg[it]["occurred_at"] > imp["occurred_at"]:
                err("S19: %s — задел, должен быть запущен по «Б» до извещения" % it)
        if reg["F-040"]["payload"]["kd_revision"] != "V" or reg["F-040"]["occurred_at"] < imp["occurred_at"]:
            err("S19: F-040 должен быть запущен по «В» после извещения")
        want = {"F-039": "rework_to_new_revision", "F-037": "keep_old_with_concession", "F-038": "scrap"}
        got = {s_["subject"]["item_id"]: s_ for s_ in defs["S19.yaml"]["human_steps"] if s_["action"] == "kd.wip_decision"}
        if {k: v["params"]["decision"] for k, v in got.items()} != want or any(v["at"] < imp["occurred_at"] for v in got.values()):
            err("S19: решения по заделу не те %s" % {k: v["params"]["decision"] for k, v in got.items()})
        pm = st.get("S19/permit-F-037")
        if not pm or "CR-01" not in (pm.get("signers") or []) or pm["at"] < got["F-037"]["at"]:
            err("S19: разрешение на отклонение по F-037 без подписи ПЗ или раньше решения")
        rw = lb.get("MO-039-2.start")
        km = lb.get("cmm/F-039/rev-V")
        z2 = st.get("ZT-2#2/F-039")
        d2 = [c for c in (km or {}).get("payload", {}).get("characteristics", []) if c["char_id"] == "D2"]
        if not rw or rw["payload"].get("rework_of_run_id") != "MO-039-1" or rw["occurred_at"] < got["F-039"]["at"] \
                or not d2 or d2[0]["nominal"] != 252.0 or d2[0]["result"] != "in_tolerance" \
                or not z2 or z2["params"].get("kd_revision") != "V" or z2["at"] < km["occurred_at"]:
            err("S19: доработка F-039 до «В» не по порядку (выполнение со ссылкой → КИМ D2 = 252 → ЗТ-2 №2)")
        cams = [o for o in r if o["source_id"] == "CAM-MO" and o.get("item_id") in ("F-039", "F-040")
                and o["occurred_at"] > imp["occurred_at"]]
        man = [lb.get("kt2-manual/" + x) for x in ("F-039", "F-040")]
        if cams or not all(m and m["payload"]["ai_admission"]["status"] == "not_admitted" for m in man):
            err("S19: для ревизии «В» КТ-2 должна быть ручной (камеры нет, допуска ИИ нет)")
        n["S19"] += 1
    ob = defs["S19.yaml"].get("erp_outbox_expected") or []
    wo = [x for x in ob if x["type"] == "scrap_transfer_writeoff"]
    if len(wo) != 1 or wo[0]["items"] != ["F-038"] or {x["type"] for x in ob} - {"issue_to_production", "transfer",
                                                                                    "scrap_transfer_writeoff"}:
        err("S19: в 1С — только «перевод в брак (списание)» по F-038 сверх обычного")
    mo = defs["S19.yaml"]
    if mo.get("mes_outbox_counts") != {"hold": 3, "release": 2} or mo.get("mes_held_at_end") != ["F-038"]:
        err("S19: MES %s / %s (ожидалось 3 / 2, придержан F-038)" % (mo.get("mes_outbox_counts"), mo.get("mes_held_at_end")))
    # 34. S20
    r, lb, st = rows("S20.jsonl"), lbs("S20.jsonl"), stp("S20.yaml")
    mm, un = lb.get("S20/scan-mismatch"), lb.get("S20/scan-unreadable")
    sw, mn = st.get("S20/swap-resolved"), st.get("S20/manual-F-032")
    if not mm or mm["payload"]["read_result"] != "mismatch" or mm["payload"]["expected_item_id"] != "F-031" or mm["item_id"] != "F-033":
        err("S20: нет скана «ожидался F-031, пришёл F-033»")
    elif not sw or not (mm["occurred_at"] < sw["at"] < lb["SV-031-1.link"]["occurred_at"] < lb["SV-031-1.start"]["occurred_at"]):
        err("S20: сварка F-031 должна идти только после выяснения")
    elif any(o.get("correlation_id", "").startswith("SV-031") and o.get("item_id") not in (None, "F-031") for o in r) or \
            any(o.get("correlation_id", "").startswith("SV-033") and o.get("item_id") not in (None, "F-033") for o in r) or \
            any(o["event_type"] == "operation.started" and o.get("item_id") == "F-033" and o["payload"].get("operation_code") == "W2"
                and o["occurred_at"] < sw["at"] for o in r):
        err("S20: история F-031 и F-033 смешалась")
    else:
        n["S20"] += 1
    if not un or un["payload"]["read_result"] != "unreadable" or "item_id" in un or un["payload"]["expected_item_id"] != "F-032":
        err("S20: нет скана «код не читается» по F-032")
    elif not mn or len(set(mn.get("signers") or [])) < 2 or "MST-02" not in mn["signers"] or \
            not (un["occurred_at"] < mn["at"] < lb["SV-032-1.link"]["occurred_at"]) or \
            lb["SV-032-1.link"]["payload"]["binding_method"] != "manual_confirmed":
        err("S20: ручной ввод по F-032 без подтверждения второго человека или не до привязки кольца")
    else:
        rm, z3 = lb.get("remark2/F-032"), st.get("ZT-3/F-032")
        if not rm or rm["payload"]["reason"] != "remark_unreadable" or not z3 or rm["occurred_at"] > z3["at"] or \
                (z3["params"].get("marking_check") or {}).get("basis_label") != "remark2/F-032":
            err("S20: перемаркировка F-032 не до ЗТ-3 или ЗТ-3 её не сверяет")
        else:
            n["S20"] += 1
    # 35. S21
    r, lb, st = rows("S21.jsonl"), lbs("S21.jsonl"), stp("S21.yaml")
    cl = st.get("S21/claim")
    s107, s108 = lb.get("S21/scan-R-107"), lb.get("S21/scan-R-108-blocked")
    issued = {c: o for o in r if o["event_type"] == "batch.issued" for c in (o["payload"].get("component_ids") or [])}
    if not cl or not s107 or s107["occurred_at"] > cl["at"] or issued.get("R-107", {}).get("payload", {}).get("for_items") != ["F-034"] \
            or lb["SV-034-1.link"]["payload"]["lot_id"] != "LOT-R-117":
        err("S21: К-107 должно быть выдано под F-034 до претензии, генеалогия — партия П-117")
    elif not s108 or s108["occurred_at"] < cl["at"] or s108["payload"]["lot_status"] != "blocked" or "R-108" in issued \
            or not lb.get("S21/isolator-R-108") or issued.get("R-033", {}).get("payload", {}).get("for_items") != ["F-036"]:
        err("S21: К-108 после претензии не должно выдаваться; в изолятор; под F-036 — К-033")
    else:
        ex = st.get("S21/exclude-F-034")
        if not ex or ex["at"] < cl["at"] or ex["params"]["result"] != "excluded" or \
                not any(b.startswith("xr/F-034/") for b in ex["params"]["basis_labels"]):
            err("S21: F-034 должен быть в круге по партии и исключён по рентгену после претензии")
        else:
            n["S21"] += 1
    ob = defs["S21.yaml"].get("erp_outbox_expected") or []
    rt = [x for x in ob if x["type"] == "return_to_supplier"]
    if len(rt) != 1 or rt[0]["qty"] != 9 or any("R-108" in (x.get("components") or []) for x in ob if x["type"] == "issue_to_production"):
        err("S21: в 1С — один возврат 9 колец; К-108 в «принято в работу» не уходит")
    mo = defs["S21.yaml"]
    if mo.get("mes_outbox_counts") != {"hold": 2, "release": 1} or mo.get("mes_held_at_end") != ["LOT-R-117"]:
        err("S21: MES %s / %s" % (mo.get("mes_outbox_counts"), mo.get("mes_held_at_end")))
    # 36. S22
    r = rows("S22.jsonl")
    st10 = stp("S10B.yaml")
    st = stp("S22.yaml")
    sc = st10.get("S10B/scrap")
    if not r or not sc or any(o["occurred_at"] < sc["at"] for o in r):
        err("S22: строки должны идти после списания F-090 в S10B")
    if any(o["event_type"] in ("operation.started", "batch.issued", "movement.sent") for o in r) or \
            any(s_["action"] == "decision.gate_passed" for s_ in defs["S22.yaml"]["human_steps"]) or \
            defs["S22.yaml"].get("erp_outbox_expected"):
        err("S22: по списанному F-090 не должно быть работы, выдачи, ЗТ и сообщений в 1С")
    rf, sm = st.get("S22/rework-refused"), st.get("S22/sample")
    scan = [o for o in r if o["event_type"] == "item.scanned"]
    if not rf or rf["expect"] != "refused:E_ITEM_SCRAPPED" or not sm or sm["expect"] != "accepted" \
            or sm["params"].get("route_return") is not False or len(scan) != 1 or scan[0]["payload"].get("item_status") != "scrapped":
        err("S22: отказ E_ITEM_SCRAPPED и решение «образец без возврата в маршрут» — не так")
    else:
        n["S22"] += 1
    # сверка чисел утверждений с данными
    if exv:
        z3 = _steps_by_label(defs["S18.yaml"]).get("ZT-3/F-042")
        if z3 and exv["S18-05"]["field"]["basis"] != z3["params"]["basis_labels"]:
            err("S18-05: основание ЗТ-3 в данных %s" % z3["params"]["basis_labels"])
        z4 = _steps_by_label(defs["S18.yaml"]).get("ZT-4/F-042")
        if z4 and exv["S18-08"]["field"]["basis"] != z4["params"]["basis_labels"]:
            err("S18-08: основание ЗТ-4 в данных %s" % z4["params"]["basis_labels"])
        h = [r_ for r_ in defs["S19.yaml"]["mes_outbox_expected"] if r_["type"] == "hold"]
        if sorted(r_["item_id"] for r_ in h) != exv["S19-03"]["field"]["items"] or \
                {r_["at_msk"] for r_ in h} != {exv["S19-03"]["field"]["at"][:19].replace("T", " ")}:
            err("S19-03: удержания задела в данных не те")
        rt = [x for x in defs["S21.yaml"]["erp_outbox_expected"] if x["type"] == "return_to_supplier"]
        if not rt or rt[0]["qty"] != exv["S21-02"]["field"]["erp_return_qty"]:
            err("S21-02: число колец в возврате не совпадает с данными")
        h21 = [r_ for r_ in defs["S21.yaml"]["mes_outbox_expected"] if r_.get("item_id") == "F-034" and r_["type"] == "hold"]
        if not h21 or h21[0]["at_msk"] != exv["S21-03"]["field"]["item_hold_at"][:19].replace("T", " "):
            err("S21-03: время удержания Ф-034 не совпадает с данными")
    return dict(n)


# ----------------------------------------------------------------------------- 28. материалы
def next_observation_checks(data, defs, exv):
    """37. «Следующее лучшее наблюдение» (камера): профили карты, лимит, ссылки, правило «класс камере не виден»."""
    rec = yaml.safe_load(open(os.path.join(BASE, "definitions", "reference", "inspection-recipes.yaml"), encoding="utf-8"))
    recipes = {r["id"]: r for r in rec["recipes"]}
    n_steps = n_frames = 0
    seen = set()
    for dfn, d in defs.items():
        for s_ in d.get("human_steps") or []:
            par = s_.get("params") or {}
            if s_["action"] != "decision.recheck_requested" or par.get("method") != "camera_reshoot":
                continue
            n_steps += s_["label"] not in seen
            r = recipes.get(par.get("recipe_id"))
            where = "%s %s" % (dfn, s_["label"])
            if not r or not r.get("capture_profiles"):
                err("%s: пересъёмка без карты с профилями съёмки (%s)" % (where, par.get("recipe_id")))
                continue
            ids = [c["id"] for c in r["capture_profiles"]]
            lim = r["next_observation"]["reshoot_limit_per_zone"]
            if par.get("capture_profile") not in ids:
                err("%s: профиль %s не из карты %s" % (where, par.get("capture_profile"), r["id"]))
            if par.get("reshoot_limit") != lim or not 1 <= (par.get("reshoot_no") or 0) <= lim:
                err("%s: пересъёмка №%s при лимите %s (карта: %s)" % (where, par.get("reshoot_no"), par.get("reshoot_limit"), lim))
            for k in ("uncertainty_reasons", "profile_reason", "prior_observation_label", "assigned_by", "proposal"):
                if not par.get(k):
                    err("%s: у назначения пересъёмки нет поля %s" % (where, k))
            pr = par.get("proposal") or {}
            if pr.get("voice") != "инженерная справка" or pr.get("kind") != "next_observation" or pr.get("status_after") != "accepted":
                err("%s: предложение системы оформлено не так: %s" % (where, pr))
            ab = par.get("assigned_by") or {}
            if ab.get("kind") not in ("person", "rule") or (ab.get("kind") == "person" and ab.get("persona") != s_["persona"]):
                err("%s: кто назначил — %s" % (where, ab))
            if par.get("person_actions") and ab.get("kind") != "person":
                err("%s: нужен человек (%s), а назначено по правилу" % (where, par["person_actions"]))
            # кадр пересъёмки: после назначения, по профилю назначения, со ссылкой на прежний кадр
            item = s_["subject"].get("item_id")
            rows = [o for _, o in data["main-story.jsonl"]] if dfn in ("main-story.yaml", "S04.yaml") else []
            by_id = {o["event_id"]: o for o in rows}
            fr = [o for o in rows if o.get("item_id") == item and o["event_type"] == "inspection.result"
                  and o["payload"].get("trigger") == "reshoot" and o["payload"].get("reshoot_no") == par.get("reshoot_no")]
            if len(fr) != 1:
                err("%s: кадр пересъёмки №%s по %s: %d" % (where, par.get("reshoot_no"), item, len(fr)))
                continue
            f = fr[0]
            n_frames += s_["label"] not in seen
            seen.add(s_["label"])
            prev = by_id.get(f["payload"].get("prior_observation_event_id"))
            if f["payload"].get("capture_profile") != par.get("capture_profile") or f["occurred_at"] <= s_["at"]:
                err("%s: кадр пересъёмки не по назначению (профиль %s, время %s)" % (where, f["payload"].get("capture_profile"), f["occurred_at"]))
            if not prev or prev["item_id"] != item or prev["_sim"]["label"] != par["prior_observation_label"] \
                    or prev["payload"]["checkpoint_id"] != f["payload"]["checkpoint_id"] or prev["occurred_at"] >= f["occurred_at"]:
                err("%s: кадр пересъёмки не ссылается на прежний кадр %s" % (where, par.get("prior_observation_label")))
            elif prev["payload"].get("capture_profile") not in ids:
                err("%s: у прежнего кадра профиль %s не из карты" % (where, prev["payload"].get("capture_profile")))
            if set(f["payload"]["zones_inspected"]) != set(par.get("zones") or []):
                err("%s: пересняты зоны %s, назначены %s" % (where, f["payload"]["zones_inspected"], par.get("zones")))
            if exv and dfn == "main-story.yaml" and item == "F-025":
                e13, e12, e14 = exv["S04-13"], exv["S04-12"]["value"][0], exv["S04-14"]["field"]
                if e13["value"] != ids:
                    err("S04-13: список профилей %s ≠ карта %s" % (e13["value"], ids))
                for k in ("uncertainty_reasons", "capture_profile", "reshoot_no", "reshoot_limit"):
                    if e12[k] != par[k]:
                        err("S04-12: %s = %s, в шаге %s" % (k, e12[k], par[k]))
                if e12["proposal_ref"] != pr.get("proposal_ref") or not par["profile_reason"].startswith(e12["profile_reason"]):
                    err("S04-12: предложение или «почему» не совпадают с шагом")
                if e14["zones_inspected"] != f["payload"]["zones_inspected"] or e14["capture_profile"] != f["payload"]["capture_profile"]:
                    err("S04-14: кадр пересъёмки не совпадает с данными")
    # правило «класс камере не виден → рентген, не пересъёмка»
    blind = []
    for _, o in data["main-story.jsonl"]:
        p = o["payload"]
        if o["event_type"] == "inspection.result" and isinstance(p, dict) and p.get("method") == "xray":
            for d in p.get("defects") or []:
                if d.get("side") == "root" or (d.get("defect_type") == "porosity" and d.get("zone", "").startswith("U")):
                    blind.append(o["item_id"])
    blind = sorted(set(blind))
    for dfn, d in defs.items():
        for s_ in d.get("human_steps") or []:
            if (s_.get("params") or {}).get("method") == "camera_reshoot" and s_["subject"].get("item_id") in blind:
                err("%s %s: пересъёмка по %s, а рентген нашёл класс, который камере не виден" % (dfn, s_["label"], s_["subject"]["item_id"]))
    if exv and not {"F-021", "F-023"} <= set(blind):
        err("S04-N7: у Ф-021 и Ф-023 нет находки рентгена в корне (%s)" % blind)
    if exv and exv["S04-16"]["field"]["root_burn_through"]["conclusion_no"] not in \
            {o["payload"].get("conclusion_no") for _, o in data["main-story.jsonl"] if o.get("item_id") == "F-021"}:
        err("S04-16: заключения по Ф-021 нет в данных")
    s04 = {o["_sim"]["label"] for _, o in data["S04.jsonl"]}
    for lb in ("kt3/F-025/recheck-glare", "kt3/F-025/reshoot", "kt3/F-021/recheck", "xr/F-021/RK-0923-08"):
        if lb not in s04:
            err("S04: в срезе нет строки %s" % lb)
    if n_steps < 1 or n_frames != n_steps:
        err("пересъёмок: назначено %d, кадров %d" % (n_steps, n_frames))
    return n_steps, blind


def media_checks(data, ex):
    ref = yaml.safe_load(open(os.path.join(BASE, "definitions", "reference", "media.yaml"), encoding="utf-8"))
    kinds = set(ref["meta"]["allowed_kinds"])
    med = {m["id"]: m for m in ref["media"]}
    for m in med.values():
        if m["kind"] not in kinds:
            err("media.yaml: %s — вид %s не разрешён" % (m["id"], m["kind"]))
        if m["kind"] == "illustration":
            for k in ("license", "license_url", "source_dataset", "authors", "dataset_url", "notice", "caption"):
                if not m.get(k):
                    err("media.yaml: у иллюстрации %s нет поля %s" % (m["id"], k))
            if m.get("relates_to_item") is not False or "ИЛЛЮСТРАЦИЯ" not in (m.get("notice") or ""):
                err("media.yaml: у иллюстрации %s нет пометки «не относится к изделию»" % m["id"])
    stats = {}
    for fn, rows in data.items():
        c = Counter()
        for _, o in rows:
            p = o["payload"] if isinstance(o["payload"], dict) else {}
            need = (o["event_type"] == "inspection.result" and p.get("method") in ("camera", "xray")) or \
                o["event_type"] == "operator_action.detected"
            refs = p.get("evidence_refs")
            if "evidence_refs" in p and not refs:
                err("%s %s: пустое поле материалов" % (fn, o["_sim"]["label"]))
            if need and not refs:
                err("%s %s: нет ссылки на материалы" % (fn, o["_sim"]["label"]))
            for r in refs or []:
                m = med.get(r.get("media_id"))
                if not m:
                    err("%s %s: media_id %s нет в media.yaml" % (fn, o["_sim"]["label"], r.get("media_id")))
                    continue
                if r.get("kind") != m["kind"]:
                    err("%s %s: вид ссылки %s ≠ %s" % (fn, o["_sim"]["label"], r.get("kind"), m["kind"]))
                if m["kind"] == "illustration" and (r.get("relates_to_item") is not False or r.get("defect_type") != m["defect_type"]):
                    err("%s %s: иллюстрация без пометки или не того класса" % (fn, o["_sim"]["label"]))
                if o["_sim"]["noise"] != "duplicate":
                    c[m["kind"]] += 1
                    if m["kind"] == "illustration":
                        c["ill_items:" + (o.get("item_id") or "?")] += 1
            if need and p.get("method") == "camera" and o["_sim"]["noise"] != "duplicate":
                c["camera"] += 1
            if need and p.get("method") == "xray" and [r.get("media_id") for r in refs or []] != ["MED-DOC-XR-CONCLUSION"]:
                err("%s %s: у рентгена должно быть только заключение-документ" % (fn, o["_sim"]["label"]))
        stats[fn] = c
    ms = stats["main-story.jsonl"]
    got = {"camera_not_provided": ms["not_provided"], "camera_illustrations": ms["illustration"],
           "illustration_items": sorted(k.split(":", 1)[1] for k in ms if k.startswith("ill_items:")),
           "xray_documents": ms["document"], "operator_vision_event_only": ms["event_only"], "empty_evidence_refs": 0,
           "item_images": 0}
    if ms["camera"] != ms["not_provided"]:
        err("материалы: кадров %d, «не предоставлен» %d" % (ms["camera"], ms["not_provided"]))
    if "S03-17" in ex and ex["S03-17"]["field"] != got:
        err("материалы: по данным %s ≠ S03-17 %s" % (got, ex["S03-17"]["field"]))
    by = {o["_sim"]["label"]: o for _, o in data["main-story.jsonl"]}
    for eid, lab in (("S03-16", "kt3/SV-017-1/CAM-WS-1"), ("S11-06", "S11/ov-detected")):
        if eid in ex and by[lab]["payload"]["evidence_refs"] != ex[eid]["value"]:
            err("материалы: %s по данным %s ≠ %s" % (eid, by[lab]["payload"]["evidence_refs"], ex[eid]["value"]))
    return {fn: dict((k, v) for k, v in c.items() if not k.startswith("ill_items")) for fn, c in stats.items()}


# ----------------------------------------------------------------------------- 31. «почему?» у каждой исключённой детали
def why_checks(data, defs, ex):
    WHY = {"RS-01": "S05-47", "RS-02": "S02-06", "RS-14": "S14-31"}
    runs = {"main-story.yaml": "main-story.jsonl", "S14.yaml": "S14.jsonl", "S13.yaml": "S13.jsonl", "S10B.yaml": "S10B.jsonl"}
    n = 0
    for dfn, sfn in runs.items():
        ev = {o["_sim"]["label"]: o for _, o in data[sfn]}
        for s_ in defs[dfn]["human_steps"]:
            if s_.get("expect") != "accepted":
                continue
            rs = s_["subject"].get("risk_scope_ref")
            pairs = []
            if s_["action"] == "risk_scope.narrowed":
                pib = s_["params"].get("per_item_basis") or {}
                pairs = [(it, pib.get(it) or []) for it in s_["params"].get("excluded_items") or []]
            elif s_["action"] == "risk_scope.item_assessed" and s_["params"].get("result") == "excluded":
                pairs = [(s_["subject"]["item_id"], s_["params"].get("basis_labels") or [])]
            for it, labs in pairs:
                n += 1
                own = [x for x in labs if x in ev and (ev[x].get("item_id") == it or ("/%s/" % it) in "/" + x + "/"
                                                     or x.split("/")[1:2] == ["SV-%s-1" % it[2:]])]
                if not own or any(x not in ev for x in labs):
                    err("«почему?» %s: у %s нет своих оснований в данных (%s)" % (s_["label"], it, labs))
                eid = WHY.get(rs)
                if not eid or eid not in ex or it not in json.dumps(ex[eid], ensure_ascii=False):
                    err("«почему?» %s: %s не названа в утверждении %s" % (rs, it, eid))
    f47 = (ex.get("S05-47") or {}).get("field") or {}
    if f47.get("items_with_own_basis") != 28 or f47.get("items_without_own_basis") != 0:
        err("S05-47: число деталей с основаниями")
    st14 = {s_["label"]: s_ for s_ in defs["S14.yaml"]["human_steps"]}
    got = {it: st14["S14/exclude-" + it]["params"]["basis_labels"] for it in ("F-302", "F-308", "F-312")}
    if "S14-31" in ex and ex["S14-31"]["value"] != got:
        err("S14-31: основания по данным %s ≠ %s" % (got, ex["S14-31"]["value"]))
    return n


def main():
    ref = load_ref()
    # YAML
    ymls = sorted(glob.glob(os.path.join(BASE, "**", "*.yaml"), recursive=True))
    for p in ymls:
        try:
            yaml.safe_load(open(p, encoding="utf-8"))
        except Exception as ex:
            err("YAML %s: %s" % (p, ex))
    manifest = yaml.safe_load(open(os.path.join(BASE, "streams", "_manifest.yaml"), encoding="utf-8"))["streams"]
    kinds = {s["file"]: s["kind"] for s in manifest}
    data = {}
    for p in sorted(glob.glob(os.path.join(BASE, "streams", "*.jsonl"))):
        k = kinds.get(os.path.basename(p), "?")
        kk = "run" if k == "run" else ("slice" if k.startswith("slice") else "injection")
        data[os.path.basename(p)] = check_stream(p, ref, manifest, kk)

    ms = [o for _, o in data["main-story.jsonl"]]
    labels = {o["_sim"]["label"] for o in ms}
    line_counts = {}
    for fn in ("main-story.jsonl", "S13.jsonl", "S10B.jsonl", "S14.jsonl", "S18.jsonl", "S19.jsonl", "S20.jsonl", "S21.jsonl"):
        line_counts[fn] = check_lines(fn, [o for _, o in data[fn]], ref)
    line_counts["S22.jsonl"] = check_lines("S10B+S22", [o for _, o in data["S10B.jsonl"]] + [o for _, o in data["S22.jsonl"]],
                                           ref) - line_counts["S10B.jsonl"]
    for fn in data:
        if fn in line_counts:
            continue
        for _, o in data[fn]:
            if o.get("line_id") is not None and o["line_id"] not in ref["lines"]:
                err("%s: line_id %s нет в справочнике" % (fn, o["line_id"]))

    # ---- инварианты главной истории
    b = [o for o in ms if o["source_id"] == "WS-2" and "2026-09-23T09:05:00" <= o["received_at"] < "2026-09-23T09:10:00"]
    nd = sum(1 for o in b if o["_sim"]["noise"] == "duplicate")
    if (len(b), nd, len(b) - nd) != (1240, 312, 928):
        err("пачка ИС-2: %s строк, %s дублей, %s уникальных (ожидалось 1240/312/928)" % (len(b), nd, len(b) - nd))
    late = [o for o in b if o["_sim"]["noise"] == "late"]
    if len(late) != 928:
        err("опоздавших в пачке %d ≠ 928" % len(late))
    mx = max((o["received_at"], o["occurred_at"]) for o in late)
    first_occ = min(o["occurred_at"] for o in late)
    if first_occ != "2026-09-22T06:50:00.000Z":
        err("первая запись пачки occurred_at %s ≠ Вт 09:50 МСК" % first_occ)
    key = [o for o in ms if o["_sim"]["label"] == "EV-WS2-0412"]
    if not key or key[0]["occurred_at"] != "2026-09-22T07:20:14.000Z" or key[0]["received_at"] != "2026-09-23T09:05:07.000Z" \
            or key[0]["payload"]["value"] != 176:
        err("EV-WS2-0412: не то время/значение: %s" % (key[:1],))
    gaps = [g for s in manifest if s["file"] == "main-story.jsonl" for g in s["gaps"]]
    if len(gaps) != 1 or gaps[0]["count"] != 35:
        err("разрыв ИС-2: %s" % gaps)
    cur = [o["payload"]["parameters"]["welding_current_a"] for o in ms
           if o["source_id"] == "WS-2" and o["event_type"] == "machine.parameters" and "S01" in o["_sim"]["scenarios"]
           and o["payload"].get("parameters")]
    if cur:
        lo, hi = min(c["min"] for c in cur), max(c["max"] for c in cur)
        if (lo, hi) != (158, 163):
            err("ток SV-001-1 %s–%s ≠ 158–163" % (lo, hi))
    else:
        err("нет сводок ИС-2 по F-001")
    mal = Counter(o["_sim"]["noise"] for o in ms if (o["_sim"]["noise"] or "").startswith("malformed"))
    if sorted(mal) != ["malformed:E_MISSING_FIELD", "malformed:E_UNKNOWN_ENUM", "malformed:E_UNSUPPORTED_VERSION"] or \
            sum(mal.values()) != 3:
        err("S12: испорченных сообщений %s (ожидалось по одному из трёх кодов)" % dict(mal))
    miss = [o for o in ms if o["_sim"]["noise"] == "malformed:E_MISSING_FIELD"]
    if not miss or "item_id" in miss[0]:
        err("S12 сообщение 1 содержит item_id")
    v11 = [o for o in ms if o["schema_version"] == "1.1"]
    if len(v11) != 1 or "lens_temp_c" not in v11[0]["payload"]:
        err("S12 сообщение 4 (1.1 + lens_temp_c) не найдено")
    if not any(d.get("defect_type") == "spatter_x" for o in ms for d in (o["payload"].get("defects") or [])
               if isinstance(o["payload"], dict)):
        err("S12 сообщение 5 (spatter_x) не найдено")
    camdup = [o for o in ms if o["source_id"] == "CAM-WS-1" and o["_sim"]["noise"] == "duplicate"]
    if len(camdup) != 1 or camdup[0]["received_at"] != "2026-09-23T08:05:40.000Z":
        err("S06: дубль камеры 11:05:40 не найден")
    f025 = [o for o in ms if o["_sim"]["label"] == "kt3/F-025/recheck-glare"]
    if not f025 or (f025[0]["payload"]["observation_quality"], f025[0]["payload"]["confidence"],
                    f025[0]["payload"]["inspection_result"]) != (0.34, 0.91, "no_defect_found"):
        err("S04: кадр F-025 с бликом неверен")
    f017 = [o for o in ms if o["_sim"]["label"] == "kt3/SV-017-1/CAM-WS-1"]
    if not f017 or f017[0]["payload"]["defects"][0]["zone"] != "U2":
        err("S03: прожог F-017 У2 не найден")
    # WS-2: у F-019 нет ни одной сводки с током
    f019 = [o for o in ms if o["source_id"] == "WS-2" and "2026-09-22T13:40:00.000Z" < o["occurred_at"] < "2026-09-22T14:15:00.000Z"]
    if f019:
        err("S04: в окне потери ИС-2 есть записи: %d" % len(f019))
    # эталонная длительность мехобработки
    mo = [o for o in ms if o["_sim"]["label"] == "MO-001-1.finish"]
    if not mo or mo[0]["payload"]["reported_duration"]["value"] != 62:
        err("S01: длительность MO-001-1 ≠ 62 мин")
    # S06 инъекция
    s06 = [o for _, o in data["S06.jsonl"]]
    if len(s06) != 1241 or sum(1 for o in s06 if (o["_sim"]["noise"] or "").startswith("conflict")) != 1:
        err("S06.jsonl: %d строк" % len(s06))
    ms_ids = {(o["source_id"], o["event_id"]): o["source_seq"] for o in ms}
    for o in s06:
        if ms_ids.get((o["source_id"], o["event_id"])) != o["source_seq"]:
            err("S06.jsonl: строка не повторяет событие главной истории")
            break

    # ---- определения: шаги людей
    defs = {}
    for p in sorted(glob.glob(os.path.join(BASE, "definitions", "scenarios", "*.yaml"))):
        defs[os.path.basename(p)] = yaml.safe_load(open(p, encoding="utf-8"))
    stream_labels = {
        "main-story.yaml": labels,
        "S13.yaml": {o["_sim"]["label"] for _, o in data["S13.jsonl"]},
        "S10B.yaml": {o["_sim"]["label"] for _, o in data["S10B.jsonl"]},
        "S14.yaml": {o["_sim"]["label"] for _, o in data["S14.jsonl"]},
        "S22.yaml": {o["_sim"]["label"] for _, o in data["S10B.jsonl"] + data["S22.jsonl"]},
    }
    for sid in ("S18", "S19", "S20", "S21"):
        stream_labels[sid + ".yaml"] = {o["_sim"]["label"] for _, o in data[sid + ".jsonl"]}
    for fn, d in defs.items():
        steps = d.get("human_steps") or []
        sl = stream_labels.get(fn, labels)
        seen_ids = set()
        for s in steps:
            if s["step_id"] in seen_ids and fn in stream_labels:
                err("%s: повтор step_id %s" % (fn, s["step_id"]))
            seen_ids.add(s["step_id"])
            if s["action"] not in ref["actions"]:
                err("%s: действие %s нет в event-types.yaml/human_step" % (fn, s["action"]))
            if s["persona"] not in ref["people"]:
                err("%s: персона %s" % (fn, s["persona"]))
            for x in s.get("signers", []) or []:
                if x not in ref["people"]:
                    err("%s: подписант %s" % (fn, x))
            if s["role"] not in ref["roles"]:
                err("%s: роль %s" % (fn, s["role"]))
            for k in ("signal_basis_labels",):
                for lb in (s["subject"].get(k) or []):
                    if lb not in sl:
                        err("%s: %s ссылается на нет такой строки %s" % (fn, s["step_id"], lb))
            walk_ascii({"step_id": s["step_id"], "label": s["label"], "subject": s["subject"]}, "", fn)
        # after_step из потока должен ссылаться на существующий шаг
    step_labels_main = {s["label"] for s in defs["main-story.yaml"]["human_steps"]}
    for o in ms:
        a = o["_sim"].get("after_step")
        if a and a not in step_labels_main:
            err("main-story: after_step %s не найден среди шагов" % a)
    stops = [s for s in defs["main-story.yaml"]["human_steps"] if s["mode"] == "stop"]
    if len(stops) != 7:
        err("MS-1: стоп-шагов %d (ожидалось 7: 6 точек expected, в 15:30 — два шага)" % len(stops))

    # ---- экземпляры процессов схемы v0.2
    proc_stats = {}
    for fn, sfn in (("main-story.jsonl", "main-story.yaml"), ("S13.jsonl", "S13.yaml"), ("S10B.jsonl", "S10B.yaml"),
                    ("S14.jsonl", "S14.yaml"), ("S18.jsonl", "S18.yaml"), ("S19.jsonl", "S19.yaml"), ("S20.jsonl", "S20.yaml"), ("S21.jsonl", "S21.yaml")):
        proc_stats[fn] = check_processes(fn, [o for _, o in data[fn]], defs[sfn]["human_steps"], s13=(fn == "S13.jsonl"))
    proc_stats["S22.jsonl"] = check_processes("S10B+S22", [o for _, o in data["S10B.jsonl"] + data["S22.jsonl"]],
                                              defs["S10B.yaml"]["human_steps"] + defs["S22.yaml"]["human_steps"])

    # ---- порядок «решение на ЗТ → движение дальше» в прогонах
    for fn, sfn in (("main-story.jsonl", "main-story.yaml"), ("S13.jsonl", "S13.yaml"), ("S10B.jsonl", "S10B.yaml"),
                    ("S14.jsonl", "S14.yaml"), ("S18.jsonl", "S18.yaml"), ("S19.jsonl", "S19.yaml"), ("S20.jsonl", "S20.yaml"), ("S21.jsonl", "S21.yaml")):
        gates = defaultdict(dict)
        for s_ in defs[sfn]["human_steps"]:
            if s_["action"] == "decision.gate_passed" and s_["subject"].get("item_id") and s_["expect"] == "accepted" \
                    and s_["params"].get("decision") != "recheck":
                g_ = s_["subject"]["gate"] + ("#2" if s_["params"].get("presentation_no") == 2 else "")
                gates[s_["subject"]["item_id"]][g_] = s_["at"]
        for _, o in data[fn]:
            it = o.get("item_id")
            if not it or it not in gates:
                continue
            p_ = o["payload"] if isinstance(o["payload"], dict) else {}
            need = None
            if o["event_type"] == "movement.sent" and p_.get("from_shop") == "MC":
                need = "ZT-2"
            elif o["event_type"] == "movement.sent" and p_.get("from_shop") == "WC" and p_.get("to_shop") == "AC":
                need = "ZT-3"
            elif o["event_type"] == "operation.started" and p_.get("operation_code") == "T1":
                need = "ZT-4"
            elif o["event_type"] == "item.released":
                need = "ZT-6"
            elif o["event_type"] == "operator.action" and (p_.get("details") or {}).get("step") == "cover_install":
                need = "ZT-SR"
            if need:
                t_ = gates[it].get(need)
                if t_ is None or t_ > o["occurred_at"]:
                    err("%s: %s %s раньше решения %s (%s)" % (fn, it, o["event_type"], need, t_))

    # ---- ожидания: уникальность ID и «выпуск годного» только после ЗТ-6
    def zt_steps(sfn):
        out = defaultdict(list)
        for s_ in defs[sfn]["human_steps"]:
            if s_["action"] == "decision.gate_passed" and s_["subject"].get("item_id"):
                out[(s_["subject"]["item_id"], s_["subject"]["gate"], s_["params"].get("presentation_no", 1))].append(t_utc(s_["at"]))
        return out
    zt_main = zt_steps("main-story.yaml")
    exp_ids = Counter()
    n_assert = 0

    def walk_exp(obj, where, cp_at, in_must_not):
        nonlocal n_assert
        if isinstance(obj, dict):
            if "id" in obj and "what" in obj:
                n_assert += 1
                exp_ids[obj["id"]] += 1
                blob = json.dumps(obj, ensure_ascii=False)
                if "release_good" in blob and not in_must_not and cp_at:
                    items = []
                    flt = obj.get("filter") or {}
                    if isinstance(flt.get("item"), str):
                        items.append(flt["item"])
                    fld = obj.get("field")
                    if isinstance(fld, dict) and isinstance(fld.get("items"), list):
                        items += fld["items"]
                    for it in items:
                        ts = [t for t in zt_main.get((it, "ZT-6", 1), []) if t <= cp_at]
                        if not ts:
                            err("%s %s: «выпуск годного» по %s без подписанной ЗТ-6 до %s" % (where, obj["id"], it, cp_at))
                return
            cp = t_utc(obj["at"]) if isinstance(obj.get("at"), str) and obj["at"][:2] == "20" else cp_at
            for k, v in obj.items():
                walk_exp(v, where, cp, in_must_not or k == "must_not")
        elif isinstance(obj, list):
            for x in obj:
                walk_exp(x, where, cp_at, in_must_not)
    for p in sorted(glob.glob(os.path.join(BASE, "expected", "*.yaml"))):
        walk_exp(yaml.safe_load(open(p, encoding="utf-8")), os.path.basename(p), None, False)
    dup_ids = [k for k, v in exp_ids.items() if v > 1]
    if dup_ids:
        err("ожидания: повторяются ID %s" % dup_ids)

    # ---- 1С: ожидаемые сообщения и ответы
    for dfn in ("I1.yaml", "S13.yaml", "S10B.yaml", "S14.yaml", "S18.yaml", "S19.yaml", "S20.yaml", "S21.yaml"):
        ob = defs[dfn].get("erp_outbox_expected")
        if not ob:
            err("%s: нет erp_outbox_expected" % dfn)
            continue
        zt = zt_steps({"I1.yaml": "main-story.yaml"}.get(dfn, dfn))
        for r in ob:
            at_ = dt.datetime.strptime(r["at_msk"], "%Y-%m-%d %H:%M:%S").replace(tzinfo=dt.timezone(dt.timedelta(hours=3)))
            resp = r["response"]
            last = resp if isinstance(resp, str) else resp[-1]["result"]
            if last != "acked":
                err("%s: сообщение 1С №%s без подтверждения" % (dfn, r["msg_no"]))
            if r["type"] == "release_good":
                if not [t for t in zt.get((r["item_id"], "ZT-6", 1), []) if t <= at_]:
                    err("%s: «выпуск годного» %s раньше ЗТ-6" % (dfn, r["item_id"]))
            if r["type"] == "rework_return_to_production":
                g_ = r["gate"].split("#")[0]
                if not [t for t in zt.get((r["item_id"], g_, 2), []) if t <= at_]:
                    err("%s: «возврат из брака» %s раньше повторной %s" % (dfn, r["item_id"], g_))
        if dfn == "I1.yaml":
            six = {"F-015", "F-017", "F-019", "F-021", "F-023", "F-025"}
            rr = sorted(r["item_id"] for r in ob if r["type"] == "rework_return_to_production")
            if rr != ["F-015", "F-017", "F-023", "F-025"]:
                err("I1: «возврат из брака в производство» по %s (ожидалось F-015, F-017, F-023, F-025)" % rr)
            if six & {r["item_id"] for r in ob if r["type"] == "release_good"}:
                err("I1: «выпуск годного» по фланцам инцидента в главной истории")
            if sum(1 for r in ob if r["type"] == "scrap_transfer_rework") != 1:
                err("I1: «перевод в брак (переделка)» должен уйти один раз")

    # ---- волна 2: новые эпизоды и реализм
    wave2_checks(data, defs, ms)
    # ---- схема v0.3.1: плоскость, выборочный рентген, фон по классам автора, MES
    v031_checks(data, defs, ms)

    # ---- 27–31: нормы шагов, материалы, карта дефицита, плитки, «почему?»
    sys.path.insert(0, os.path.join(BASE, "tools"))
    import norms as norms_mod
    import tiles as tiles_mod
    import gaps as gaps_mod
    import metrics as metrics_mod
    exv = norms_mod.expected_values()
    extra_lines = []
    extra_lines += norms_mod.check(err)[0]
    media_stats = media_checks(data, exv)
    extra_lines += gaps_mod.check(err)[0]
    extra_lines += tiles_mod.check(err)[0]
    extra_lines += metrics_mod.check(err)[0]
    n_why = why_checks(data, defs, exv)
    ns = ns_checks(data, defs, exv)
    n_reshoot, blind = next_observation_checks(data, defs, exv)
    extra_lines.append("следующее наблюдение (камера, S04): назначений пересъёмки %d, все по профилям карты; класс камере не виден "
                       "(рентген) — %s, пересъёмки по ним нет" % (n_reshoot, ", ".join(blind)))
    extra_lines.append("нестандартные случаи S18–S22: проверено эпизодов %s" % ns)
    extra_lines.append("«почему?» у исключённых деталей: %d исключений, у всех свои основания" % n_why)

    # ---- scenarios-flange.md §6.3: нет запрещённых строк
    md = open(os.path.normpath(os.path.join(BASE, "..", "scenarios-flange.md")), encoding="utf-8").read()
    i63 = md.find("### 6.3.")
    sec = md[i63:md.find("\n### ", i63 + 5)] if i63 >= 0 else ""
    if not sec:
        err("scenarios-flange.md: нет раздела 6.3")
    for bad in ("Принят после переделки", "Заблокирован"):
        if bad in sec:
            err("scenarios-flange.md §6.3: запрещённая строка «%s»" % bad)

    # ---- сводка
    print("Потоки (строк / дублей / уникальных):")
    for fn in sorted(data):
        lst = [o for _, o in data[fn]]
        d = sum(1 for o in lst if (o["_sim"]["noise"] or "").startswith(("duplicate", "conflict")))
        print("  %-18s %6d  %5d  %6d  [%s]" % (fn, len(lst), d, len(lst) - d, kinds.get(fn)))
    per = Counter()
    for o in ms:
        for s in o["_sim"]["scenarios"]:
            per[s] += 1
    print("Строк главной истории с тегом сценария:", dict(sorted(per.items())))
    print("Шаги людей:", {fn: len(d.get("human_steps") or []) for fn, d in defs.items()})
    print("YAML-файлов разобрано:", len(ymls))
    print("Строк с line_id:", line_counts)
    print("Экземпляры процессов:")
    for fn, st in proc_stats.items():
        print("  %-18s %s" % (fn, st))
    print("Утверждений в ожиданиях: %d, ID уникальны: %s" % (n_assert, not dup_ids))
    print("1С (I1, весь прогон MS-1):", defs["I1.yaml"].get("erp_outbox_counts"))
    print("MES (I1, весь прогон MS-1):", defs["I1.yaml"].get("mes_outbox_counts"), "остались придержаны:", defs["I1.yaml"].get("mes_held_at_end"))
    print("Материалы (ссылки без дублей):", {k.split(".")[0]: dict(v) for k, v in media_stats.items() if v})
    for ln in extra_lines:
        print(ln)
    if ERR:
        print("\nОШИБКИ (%d):" % len(ERR))
        for e in ERR[:80]:
            print("  -", e)
        sys.exit(1)
    print("\nВСЕ ПРОВЕРКИ ПРОЙДЕНЫ")


if __name__ == "__main__":
    main()
