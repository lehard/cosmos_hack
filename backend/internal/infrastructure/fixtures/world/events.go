package world

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"ant/internal/infrastructure/fixtures/loader"
)

// Записи журнала мира (AD-2, AD-44): факты источников, реакции движка, решения
// людей и служебные — по времени записи; поздние факты встают на время
// возникновения (occurred_at), но видны только с шага, где их записали (S07).

type evOpt func(*Event)

func withItem(it *Item) evOpt {
	return func(e *Event) { e.Item = it; e.Stream = "item:" + FullID(it.ID) }
}
func withStep(s string) evOpt { return func(e *Event) { e.StepKey = s } }
func withSource(id, kind string) evOpt {
	return func(e *Event) { e.Source, e.SourceKind = id, kind }
}
func withAuthor(p string) evOpt {
	return func(e *Event) {
		e.Author = p
		if e.Source == "" {
			e.Source = "ui:" + p
		}
		e.Provenance = "personal"
	}
}
func withParams(kv ...string) evOpt {
	return func(e *Event) {
		if e.Params == nil {
			e.Params = map[string]string{}
		}
		for i := 0; i+1 < len(kv); i += 2 {
			e.Params[kv[i]] = kv[i+1]
		}
	}
}
func recordedAt(t time.Time) evOpt { return func(e *Event) { e.Recorded = t; e.Late = true } }
func entity(kind, id string) evOpt {
	return func(e *Event) {
		e.Entity = loader.Change{Entity: kind, ID: id}
		if e.Stream == "" {
			e.Stream = kind + ":" + id
		}
	}
}

// ev — новая запись: тип каталога, вид, момент, краткое содержание.
func (m *Model) ev(typ, kind string, at time.Time, summary string, opts ...evOpt) *Event {
	m.nextEv++
	e := &Event{ID: m.eventID(fmt.Sprintf("ev/%d", m.nextEv)), Type: typ, Kind: kind, Occurred: at, Recorded: at, Summary: summary, Stream: "", Provenance: "server_attested"}
	for _, o := range opts {
		o(e)
	}
	if e.Stream == "" {
		e.Stream = "global"
	}
	if e.Source == "" {
		e.Source = "ant"
	}
	if kind == "fact" && e.Provenance == "server_attested" {
		e.Provenance = "device"
	}
	m.Events = append(m.Events, e)
	return e
}

var stepTitle = map[string]string{
	"machining.kt2_camera": "КТ-2 камера", "machining.kt2_cmm": "КТ-2 КИМ", "welding.kt3_camera": "КТ-3 камера",
	"welding.kt3_radiography": "КТ-3 рентген", "final.kt5_camera": "КТ-5 камера",
	"machining.zt2_acceptance": "ЗТ-2", "welding.zt3_acceptance": "ЗТ-3", "assembly.zt4_acceptance": "ЗТ-4",
	"testing.zt5_protocol": "ЗТ-5", "final.zt6_acceptance": "ЗТ-6",
}

func (m *Model) personName(id string) string {
	if m.policy != nil {
		if p, ok := m.policy.Person(id); ok {
			return p.Name
		}
	}
	return id
}

func pct(bp int) string { return fmt.Sprintf("%d,%02d", bp/10000, (bp%10000)/100) }

// buildEvents — записи журнала по маршрутам изделий и сюжету.
func (m *Model) buildEvents() {
	for _, o := range m.Spec.Orders {
		m.ev("erp.order.received", "fact", o.Received.Time(), fmt.Sprintf("Задание 1С %s: %d шт. ФЛ-100.00.000 СБ, срок %s", o.Label, o.Qty, o.Due),
			withSource("onec", "external_system"), withParams("order_id", o.ID, "external_number", o.ERPRef), entity("lot", o.ID))
	}
	for _, l := range m.Spec.Lots {
		m.ev("genealogy.lot.registered", "fact", l.Received.Time(), fmt.Sprintf("Партия %s поступила: %d шт., сертификат %s", l.Label, l.Qty, l.Cert),
			withSource("onec", "external_system"), withParams("lot_id", l.ID, "supplier", l.Supplier), entity("lot", l.ID))
		m.ev("decision.lot.resolved", "decision", l.Accepted.Time(), fmt.Sprintf("ЗТ-1: партия %s годна (документы, камера КТ-1)", l.Label),
			withAuthor("INS-02"), withStep("incoming.zt1_lot_acceptance"), withParams("lot_id", l.ID, "outcome", "accepted"), entity("lot", l.ID))
	}
	for _, it := range m.Items {
		m.itemEvents(it)
	}
	m.storyEvents()
	for _, e := range m.ERP {
		subj := e.Lot
		if len(e.Items) > 0 {
			var ls []string
			for _, it := range e.Items {
				ls = append(ls, it.Label)
			}
			subj = strings.Join(ls, ", ")
		}
		opts := []evOpt{entity("erp_message", e.ID), withParams("business_key", e.ID, "action", e.Action)}
		if len(e.Items) == 1 {
			opts = append(opts, withItem(e.Items[0]))
		}
		m.ev("erp.posting.requested", "reaction", e.At, fmt.Sprintf("В 1С: %s — %s", erpActionTitle(e.Action), subj), opts...)
		for _, a := range e.Attempts {
			sum := "1С подтвердила приём"
			if a.Doc != "" {
				sum += ": " + a.Doc
			}
			if a.Result != "acked" {
				sum = fmt.Sprintf("1С: ошибка %d (%s)", a.HTTP, a.Code)
			}
			m.ev("erp.posting.responded", "fact", a.At.Time(), sum, append(opts, withSource("onec", "external_system"), withParams("result", a.Result))...)
		}
	}
}

func erpActionTitle(a string) string {
	return map[string]string{
		"accept_into_work": "принято в работу", "warehouse_transfer": "перемещение", "scrap_transfer_rework": "перевод в брак (переделка)",
		"scrap_transfer_writeoff": "перевод в брак (списание)", "scrap_transfer_reprocess": "возврат из брака в производство",
		"return_to_supplier": "возврат поставщику", "release": "выпуск годного",
	}[a]
}

// itemEvents — записи маршрута изделия: запуск, операции, контроль, закрывающие точки, перемещения.
func (m *Model) itemEvents(it *Item) {
	w := withItem(it)
	m.ev("item.item.registered", "fact", it.Launch, fmt.Sprintf("Изделие %s запущено: заготовка выдана в механический цех", it.Label),
		w, withSource("term-sk", "manual_entry"), withAuthor("STK-51"), withStep("incoming.issue_blank"), withParams("order_id", it.Order))
	m.ev("item.carrier.applied", "fact", it.Launch.Add(minutes(5)), "Нанесён DataMatrix "+it.ID, w, withSource("term-sk", "manual_entry"), withAuthor("STK-51"))
	for _, r := range it.Runs {
		src, kind := "edge-cnc-1", "machine"
		switch r.Kind {
		case "welding":
			src, kind = "term-"+strings.ToLower(r.Station), "manual_entry"
		case "assembly":
			src, kind = "term-wp-asm-1", "manual_entry"
		case "leak_test":
			src, kind = "edge-leak-1", "machine"
		}
		opts := []evOpt{w, withStep(r.StepKey), withSource(src, kind), withParams("operation_run_id", r.ID, "equipment_id", r.Equipment, "performer", r.Performer)}
		if r.ReworkOf != "" {
			opts = append(opts, withParams("rework_of", r.ReworkOf))
		}
		m.ev("operation.run.started", "fact", r.From, fmt.Sprintf("%s начата: %s, %s", r.Label, r.Equipment, m.personName(r.Performer)), opts...)
		origin := "передано источником"
		if r.Kind == "welding" || r.Kind == "assembly" {
			origin = "вычислено системой"
		}
		m.ev("operation.run.finished", "fact", r.To, fmt.Sprintf("%s завершена: %d мин, %s, активная обработка", r.Label, int(r.To.Sub(r.From).Minutes()), origin), opts...)
		if r.Kind == "welding" {
			m.weldLog(it, r)
		}
	}
	for i, mv := range it.moves {
		switch mv.step {
		case "machining.kt2_camera", "welding.kt3_camera", "final.kt5_camera":
			if mv.pos != "at_inspection" || m.storyInspection(it, mv.at, 15) {
				continue
			}
			q := 9000 + (atoi(it.ID[2:])*37)%500
			m.ev("inspection.result.recorded", "fact", mv.at.Add(minutes(5)), fmt.Sprintf("%s: признаков нет; качество %s; уверенность 0,95", stepTitle[mv.step], pct(q)),
				w, withStep(mv.step), withSource(cameraSource(mv.step), "camera"), withParams("method", "camera", "outcome", "no_defect_indicated", "observation_quality_bp", fmt.Sprint(q), "analyzer_confidence_bp", "9500"))
		case "machining.kt2_cmm":
			m.ev("inspection.result.recorded", "fact", mv.at.Add(minutes(10)), "КТ-2 КИМ: 5 размеров в допуске; калибровка действует",
				w, withStep(mv.step), withSource("gw-cmm", "machine"), withParams("method", "cmm", "outcome", "no_defect_indicated"))
		case "welding.zt3_acceptance":
			if mv.pos != "at_presentation_point" || m.storyInspection(it, mv.at, 5) {
				continue
			}
			m.ev("inspection.result.recorded", "fact", mv.at, "Заключение РК: дефектов нет", w, withStep("welding.kt3_radiography"),
				withSource("gw-ndt", "manual_entry"), withAuthor("NDT-61"), withParams("method", "radiography", "outcome", "no_defect_indicated"))
		case "welding.receive", "assembly.receive":
			from := map[string]string{"welding.receive": "FOR-MC", "assembly.receive": "FOR-WC"}[mv.step]
			to := map[string]string{"welding.receive": "FOR-WC", "assembly.receive": "FOR-AC"}[mv.step]
			if mv.loc == "WS-WC" && i > 0 && strings.HasPrefix(it.moves[i-1].step, "assembly") {
				from = "FOR-AC"
			}
			m.ev("operation.movement.sent", "decision", mv.at.Add(-minutes(5)), "Отправлено: "+m.personName(from), w, withStep(mv.step), withAuthor(from))
			m.ev("operation.movement.received", "decision", mv.at.Add(minutes(5)), "Принято цехом: осмотр при приёмке — повреждений нет", w, withStep(mv.step), withAuthor(to))
		case "final.released":
			m.ev("item.release.recorded", "decision", mv.at, "Сдано на склад готовой продукции", w, withStep(mv.step), withAuthor("STK-51"))
		}
		// Закрывающие точки: решение ОТК при уходе с точки предъявления.
		if strings.HasSuffix(mv.step, "_acceptance") || mv.step == "testing.zt5_protocol" {
			if i+1 >= len(it.moves) || mv.pos != "at_presentation_point" {
				continue
			}
			next := it.moves[i+1]
			if strings.HasSuffix(next.step, "nonconformity") || next.step == mv.step {
				continue
			}
			who := "INS-01"
			if strings.HasPrefix(mv.step, "assembly") || strings.HasPrefix(mv.step, "testing") || strings.HasPrefix(mv.step, "final") {
				who = "INS-02"
			}
			outcome, sum := "accepted", stepTitle[mv.step]+": годно"
			if mv.step == "welding.zt3_acceptance" && it.Spec.ZT3Incomplete && next.at.Equal(it.Spec.ZT3.Time()) {
				outcome, sum = "accepted_incomplete_data", "ЗТ-3: годно при неполных данных — журнал режима ИС-2 недоступен (сбой связи); принято по камере и рентгену"
			}
			if st := it.State(next.at); st.ReworkDone && mv.step == "welding.zt3_acceptance" {
				sum = "ЗТ-3, повторное предъявление: переделка принята"
			}
			opts := []evOpt{w, withStep(mv.step), withAuthor(who), withParams("outcome", outcome)}
			if mv.step == "final.zt6_acceptance" {
				opts = append(opts, withParams("co_signer", "CR-71"))
				sum += " (ОТК и представитель заказчика)"
			}
			m.ev("decision.presentation.resolved", "decision", next.at, sum, opts...)
		}
	}
}

func cameraSource(step string) string {
	switch step {
	case "machining.kt2_camera":
		return "edge-kt2"
	case "final.kt5_camera":
		return "edge-kt5"
	}
	return "edge-kt3"
}

// storyInspection — у изделия есть сюжетный результат контроля рядом с t (сигнал, блик).
func (m *Model) storyInspection(it *Item, t time.Time, window int) bool {
	near := func(x time.Time) bool {
		return !x.IsZero() && x.Sub(t) >= -minutes(window) && x.Sub(t) <= minutes(window+10)
	}
	for _, n := range m.NCs {
		if len(n.Spec.Items) == 0 && slices.Contains(n.Items, it) && near(n.SignalAt) {
			return true
		}
	}
	return false
}

// weldLog — сводка цикла сварки от источника (FR-147): ток против уставки 160 ± 10 А;
// журнал ИС-2 во время сбоя связи пришёл поздно (S07) или потерян (S04).
func (m *Model) weldLog(it *Item, r *OpRun) {
	src := "edge-weld-1"
	if r.Equipment == "IS-2" {
		src = "edge-weld-2"
	}
	lo, hi := 158, 163
	var dev time.Time
	for _, le := range m.Spec.LateEvents {
		dev = le.Occurred.Time()
	}
	out := r.Equipment == "IS-2" && !dev.IsZero() && r.To.After(dev) && r.From.Before(m.clk.at(23, 11, 12))
	if out {
		n := atoi(it.ID[2:])
		lo, hi = 176+(n%3), 179+(n%4)
		if it.ID == "F-017" {
			lo, hi = 176, 182
		}
	}
	recorded := r.To
	for _, s := range m.Spec.Sources {
		if s.Equipment != r.Equipment {
			continue
		}
		lost, restored := s.Lost.Time(), s.Restored.Time()
		if r.To.After(lost) && r.To.Before(restored) {
			recorded = restored
		}
		lw := s.Batch.LostWindow
		if !lw.IsZero() && r.From.Before(lw.To.Time()) && r.To.After(lw.From.Time()) {
			r.LogLost = true
		}
	}
	r.LogReceived, r.CurrentA = recorded, [2]int{lo, hi}
	if r.LogLost {
		m.ev("quality.inspection.missing", "reaction", recorded, fmt.Sprintf("%s: параметры сварки %s отсутствуют — потеря у источника (буфер шлюза)", r.Label, r.Equipment),
			withItem(it), withStep("welding.weld"), withParams("operation_run_id", r.ID, "reason", "equipment_log_missing"), recordedAt(recorded))
		return
	}
	opts := []evOpt{withItem(it), withStep("welding.weld"), withSource(src, "machine"), withParams("operation_run_id", r.ID, "equipment_id", r.Equipment,
		"parameter", "current_a", "value", fmt.Sprintf("%d–%d", lo, hi), "setpoint", "160 ± 10 А", "program", r.Program)}
	if recorded.After(r.To) {
		opts = append(opts, recordedAt(recorded))
	}
	sum := fmt.Sprintf("Сводка цикла %s: ток %d–%d А (уставка 160 ± 10 А)", r.Label, lo, hi)
	m.ev("equipment.cycle.summarized", "fact", r.To, sum, opts...)
	if out {
		m.ev("equipment.deviation.detected", "fact", r.From.Add(minutes(2)), fmt.Sprintf("%s: ток вне уставки, до %d А (%s)", r.Equipment, hi, r.Label), opts...)
	}
}

// storyEvents — записи сюжета: сигналы, несоответствия, области риска, источники, карантин, целостность.
func (m *Model) storyEvents() {
	for _, s := range m.Spec.Signals {
		it := m.itemByID[s.Item]
		w := withItem(it)
		if s.Unable {
			m.ev("decision.recheck.requested", "decision", s.At.Time().Add(-minutes(10)), "Назначена доп. проверка: повторная съёмка шва и рентген", w, withAuthor("INS-01"))
			// Иллюстрация блика — к наблюдению и к производному «оценка невозможна» (FR-102).
			m.ev("inspection.result.recorded", "fact", s.At.Time(), fmt.Sprintf("КТ-3: признаков нет; уверенность %s; качество %s — блик, зона У6–У7 частично закрыта прижимом", pct(s.ConfidenceBP), pct(s.QualityBP)),
				w, withStep("welding.kt3_camera"), withSource("edge-kt3", "camera"), withParams("method", "camera", "outcome", "no_defect_indicated", "analyzer_confidence_bp", fmt.Sprint(s.ConfidenceBP), "observation_quality_bp", fmt.Sprint(s.QualityBP), "zone", s.Zone)).attach(illGlare)
			m.ev("inspection.result.recorded", "reaction", s.At.Time(), "Производный результат: качество наблюдения ниже порога рецепта 0,6 → оценка невозможна для У6–У7; исходное сообщение не меняется",
				w, withStep("welding.kt3_camera"), withParams("method", "camera", "outcome", "unable_to_assess", "derived_from", "observation_quality_bp < 6000")).attach(illGlare)
			m.ev("decision.recheck.requested", "decision", s.At.Time().Add(minutes(5)), "Назначена повторная съёмка: снять прижим, сменить угол света", w, withAuthor("INS-01"))
			m.ev("inspection.result.recorded", "fact", s.Resolved.Time(), "КТ-3, повторный кадр: признаков нет; уверенность 0,93; качество 0,88",
				w, withStep("welding.kt3_camera"), withSource("edge-kt3", "camera"), withParams("method", "camera", "outcome", "no_defect_indicated", "analyzer_confidence_bp", "9300", "observation_quality_bp", "8800"))
			continue
		}
		m.ev("inspection.result.recorded", "fact", s.At.Time().Add(-time.Minute), fmt.Sprintf("КТ-4: нет метки затяжки, болт 7; уверенность %s; качество %s", pct(s.ConfidenceBP), pct(s.QualityBP)),
			w, withStep("assembly.kt4_camera"), withSource("edge-kt4", "camera"), withParams("method", "camera", "outcome", "defect_indicated", "analyzer_confidence_bp", fmt.Sprint(s.ConfidenceBP), "observation_quality_bp", fmt.Sprint(s.QualityBP)))
		m.ev("quality.signal.raised", "reaction", s.At.Time(), "Сигнал: нет метки затяжки (болт 7) — изделие придержано на ЗТ-4 до решения", w, withStep("assembly.zt4_acceptance"), withParams("signal_id", s.ID))
		m.ev("decision.signal.rejected", "decision", s.Rejected.Time(), "Сигнал отклонён: "+s.Reason, w, withAuthor(s.By), withParams("signal_id", s.ID))
	}
	for _, n := range m.NCs {
		m.ncEvents(n)
	}
	for _, in := range m.Incidents {
		m.incidentEvents(in)
	}
	for _, h := range m.Spec.ProcessHolds {
		m.ev("decision.process_hold.set", "decision", h.Set.Time(), fmt.Sprintf("Остановка %s: %s; снять — %s (предложила система)", h.Equipment, h.Reason, h.ReleaseCondition),
			withAuthor(h.By), entity("equipment", h.Equipment), withParams("equipment_id", h.Equipment))
		m.ev("equipment.state.changed", "fact", h.Set.Time().Add(time.Minute), h.Equipment+": остановлен (остановка по качеству)", withSource("edge-weld-2", "machine"), entity("equipment", h.Equipment))
	}
	for _, s := range m.Spec.Sources {
		m.ev("ingest.source.loss_suspected", "reaction", s.Alert.Time(), "Нет данных от ИС-2 15 минут: источник помечен «нет связи»", entity("equipment", s.Equipment), withParams("source_id", s.ID))
		b := s.Batch
		m.ev("ingest.anomaly.flagged", "reaction", s.Restored.Time(), fmt.Sprintf("Связь с ИС-2 восстановлена: дослано %d записей, повторов %d, опоздавших %d, потеряно у источника %d (разрыв %s)", b.Records, b.Duplicates, b.Late, b.Lost, b.Gap),
			entity("equipment", s.Equipment), withParams("source_id", s.ID, "late_write", fmt.Sprint(b.Late), "duplicates", fmt.Sprint(b.Duplicates), "lost", fmt.Sprint(b.Lost)))
	}
	for _, le := range m.Spec.LateEvents {
		it := m.itemByID[le.RunItem]
		e := m.ev("equipment.deviation.detected", "fact", le.Occurred.Time(), fmt.Sprintf("%s: ток %d А при уставке %s — первое отклонение (запись %s)", "ИС-2", le.CurrentA, le.Setpoint, le.ID),
			withItem(it), withSource(le.Source, "machine"), withParams("record", le.ID, "value", fmt.Sprint(le.CurrentA), "setpoint", le.Setpoint), recordedAt(le.Received.Time()))
		e.Params["late"] = "true"
	}
	for _, rv := range m.Spec.Reviews {
		it := m.itemByID[rv.Item]
		m.ev("task.task.created", "reaction", rv.Flagged.Time(), fmt.Sprintf("Решение %s по %s помечено «принято до новых данных — пересмотрите»: основание %s", rv.Gate, it.Label, rv.LateEvent), withItem(it))
		m.ev("decision.presentation.resolved", "decision", rv.Completed.Time(), fmt.Sprintf("Пересмотр приёмки %s: отозвано — изделие в инциденте RS-01, ждёт решения; исходная подпись остаётся", rv.Gate),
			withItem(it), withAuthor(rv.By), withStep("welding.zt3_acceptance"), withParams("outcome", rv.Outcome))
	}
	for _, iv := range m.Spec.Interventions {
		it := m.itemByID[iv.Item]
		m.ev("item.intervention.opened", "decision", iv.Opened.Time(), "Вскрытие открыто: "+iv.Note, withItem(it), withAuthor(iv.By), withStep("assembly.intervention"))
	}
	for _, u := range m.Spec.ComponentUnlinks {
		it := m.itemByID[u.Item]
		m.ev("item.assembly.recorded", "fact", u.At.Time(), "Компонент снят с изделия: "+u.Note, withItem(it), withAuthor("W21"), withParams("component", FullID(u.Component), "action", "unlinked"))
	}
	for _, q := range m.Spec.Quarantine {
		m.ev("ingest.message.quarantined", "service", q.At.Time(), fmt.Sprintf("Карантин: %s (поле %s) от %s", q.Code, q.Field, q.Source), entity("quarantine", q.ID), withParams("code", q.Code, "field", q.Field))
		if !q.Fixed.IsZero() {
			m.ev("ingest.message.reprocessed", "decision", q.Fixed.Time(), "Сообщение исправлено (номер изделия по контексту поста) и принято один раз", entity("quarantine", q.ID), withAuthor(q.By))
		}
	}
	for _, v := range m.Spec.Integrity.Violations {
		m.ev("security.integrity.violated", "service", v.At.Time(), "Нарушение целостности: "+v.Note, entity("integrity", "global"), withParams("record", v.Record))
	}
	// «Забыли прокладку» (S11): шаг остановлен до брака по порядку шагов маршрута.
	if it := m.itemByID["F-003"]; it != nil {
		w := withItem(it)
		m.ev("operator.action.observed", "fact", m.clk.at(22, 11, 50), "OperatorVision: крышка поднесена к фланцу, уплотнения в канавке не видно — пропущен шаг «установка уплотнения»; уверенность 0,78; качество 0,85",
			w, withSource("edge-ov-asm", "camera"), withStep("assembly.cover_install"), withParams("analyzer_confidence_bp", "7800", "observation_quality_bp", "8500"))
		m.ev("operation.precondition.failed", "reaction", m.clk.at(22, 11, 50), "Шаг «установка крышки» заблокирован: нет подтверждения «установка уплотнения» и предъявления зоны ОТК",
			w, withStep("assembly.cover_install"))
		m.ev("operator.step.confirmed", "fact", m.clk.at(22, 11, 52), "Уплотнение установлено, шаг подтверждён", w, withAuthor("A31"), withStep("assembly.seal_install"))
	}
}

func (m *Model) ncEvents(n *NC) {
	s := n.Spec
	var opts []evOpt
	if len(n.Items) > 0 {
		opts = append(opts, withItem(n.Items[0]))
	}
	opts = append(opts, entity("nonconformity", n.ID), withParams("nc_id", n.ID))
	if len(s.Items) == 0 {
		method, src, kind := "radiography", "gw-ndt", "manual_entry"
		if s.Source == "CAM-KT3" {
			method, src, kind = "camera", "edge-kt3", "camera"
		}
		var defects []string
		for _, d := range s.Defects {
			defects = append(defects, defectTitle(d.Kind)+" "+zoneTitle(d.Zone))
		}
		sum := fmt.Sprintf("%s: признаки дефекта — %s", map[string]string{"camera": "КТ-3 камера", "radiography": "Рентген " + s.Report}[method], strings.Join(defects, "; "))
		obsOpts := append(slices.Clone(opts), withStep(n.StepKey), withSource(src, kind), withParams("method", method, "outcome", "defect_indicated"))
		if method == "camera" {
			obsOpts = append(obsOpts, withParams("analyzer_confidence_bp", "8600", "observation_quality_bp", "9000", "analyzer", "vqc-weld 2.3.1"))
		}
		obs := m.ev("inspection.result.recorded", "fact", n.SignalAt.Add(-time.Minute), sum, obsOpts...)
		if n.ID == "NC-01" {
			// Иллюстрации к двум наблюдениям КТ-3 (кадров в заготовках нет, FR-102).
			obs.attach(illBurnGeneral)
			m.ev("inspection.result.recorded", "fact", n.SignalAt.Add(-50*time.Second), "КТ-3 ракурс 2: прожог У2; уверенность 0,81; качество 0,88", append(slices.Clone(opts), withStep(n.StepKey), withSource("edge-kt3-2", "camera"), withParams("method", "camera", "outcome", "defect_indicated", "analyzer_confidence_bp", "8100", "observation_quality_bp", "8800"))...).attach(illBurnClose)
			m.ev("quality.observation.linked", "reaction", n.SignalAt.Add(-50*time.Second), "Наблюдение ракурса 2 связано с известным дефектом: Ф-017 + У2 (один дефект, два наблюдения)", opts...)
		}
		m.ev("quality.signal.raised", "reaction", n.SignalAt, "Сигнал о признаке дефекта: "+strings.Join(defects, "; "), append(slices.Clone(opts), withStep(n.StepKey), withParams("signal_id", n.SignalID))...)
		for _, d := range s.Defects {
			m.ev("quality.defect.identified", "reaction", n.SignalAt, "Дефект: "+defectTitle(d.Kind)+" "+zoneTitle(d.Zone), append(slices.Clone(opts), withParams("zone", d.Zone, "defect_type", d.Kind))...)
		}
		m.ev("decision.nonconformity.drafted", "reaction", n.SignalAt, "Черновик карточки несоответствия с основаниями", opts...)
	}
	confOpts := append(slices.Clone(opts), withAuthor(s.By))
	if len(s.Items) > 0 {
		confOpts = append(confOpts, withParams("items", strings.Join(s.Items, ",")))
		m.ev("decision.nonconformity.registered", "decision", n.ConfirmedAt, fmt.Sprintf("Групповое несоответствие %s «сварка вне режима ТП» на %d изделий", n.Number, len(s.Items)), confOpts...)
	} else {
		m.ev("decision.nonconformity.confirmed", "decision", n.ConfirmedAt, fmt.Sprintf("%s подтверждено: тяжесть «значительный»", n.Number), confOpts...)
	}
	if !s.Isolated.IsZero() {
		m.ev("decision.item.isolated", "decision", s.Isolated.Time(), "Изолировано; перенесено в изолятор сварочного цеха", append(slices.Clone(opts), withAuthor(s.By))...)
	}
	for _, h := range n.Hyps {
		sum := fmt.Sprintf("Гипотезы, версия %d: оборудование — %s; отклонение от процедуры — %s", h.V, strengthTitle(h.Equipment), strengthTitle(h.Performer))
		if s.Cause != nil && s.Cause.Category == "incoming" {
			sum = fmt.Sprintf("Гипотезы, версия %d: входной брак — %s; зону не затрагивала ни одна операция", h.V, strengthTitle(h.Incoming))
		}
		if h.RevisedDueTo != "" {
			sum += "; пересмотрено из-за " + h.RevisedDueTo
		}
		if h.Categorical {
			continue
		}
		m.ev("incident.hypothesis.computed", "reaction", h.At, sum, append(slices.Clone(opts), withParams("version", fmt.Sprint(h.V)))...)
	}
	if d := s.Disposition; d != nil {
		m.ev("decision.disposition.set", "decision", d.At.Time(), fmt.Sprintf("Решение по %s: %s (подписали: %s)", n.Number, dispositionTitle(d.Kind), strings.Join(d.By, ", ")),
			append(slices.Clone(opts), withAuthor(d.By[0]), withParams("disposition", d.Kind, "signers", strings.Join(d.By, ",")))...)
	}
	if c := s.Cause; c != nil {
		title := map[string]string{"equipment": "оборудование: " + c.Ref + ", дрейф регулятора тока (контрольный образец КО-7)", "incoming": "входной брак: " + c.Ref}[c.Category]
		m.ev("incident.cause.concluded", "decision", c.At.Time(), "Причина подтверждена — "+title, append(slices.Clone(opts), withAuthor("TEC-01"), withParams("category", c.Category))...)
	}
	if !n.VerifiedAt.IsZero() {
		m.ev("decision.disposition.verified", "decision", n.VerifiedAt, "Исполнение переделки подтверждено: повторная ЗТ-3 пройдена", append(slices.Clone(opts), withAuthor("INS-01"))...)
	}
}

func (m *Model) incidentEvents(in *Incident) {
	s := in.Spec
	opts := []evOpt{entity("incident", s.ID), withParams("incident_id", s.ID)}
	m.ev("incident.incident.opened", "reaction", s.Opened.Time(), fmt.Sprintf("Инцидент %s открыт по %s", s.Label, s.Trigger), opts...)
	var prev *ScopeState
	for i := range in.Versions {
		v := &in.Versions[i]
		at := v.Spec.At.Time()
		switch {
		case i == 0:
			m.ev("incident.scope.computed", "reaction", at, fmt.Sprintf("Область риска, версия 1: %d изделий — %s", v.Size(), v.Spec.Basis), append(slices.Clone(opts), withParams("scope_version", "1", "size", fmt.Sprint(v.Size())))...)
			for _, id := range sortedKeys(v.Status) {
				if v.Status[id] == "suspect" {
					m.ev("decision.containment.applied", "reaction", at, "Придержано по правилу R-12: блок области риска "+s.ID, withItem(m.itemByID[id]), withParams("rule_id", "R-12", "level", "item_hold"))
				}
			}
		case v.Spec.Rule == "confirm_all":
			for _, id := range sortedKeys(v.Status) {
				if v.Status[id] == "confirmed" && prev.Status[id] != "confirmed" {
					m.ev("incident.item.assessed", "decision", at, "Статус в инциденте: подтверждено — "+v.Spec.Basis, withItem(m.itemByID[id]), withAuthor(v.Spec.By), withParams("incident_id", s.ID, "known", "confirmed"))
				}
			}
		default:
			sum := fmt.Sprintf("Область сужена, версия %d: %d изделий — %s", v.Spec.V, v.Size(), v.Spec.Basis)
			if v.Spec.LateEvent != "" {
				sum += "; история пересчитана из-за позднего события " + v.Spec.LateEvent
			}
			m.ev("incident.scope.narrowed", "decision", at, sum, append(slices.Clone(opts), withAuthor(v.Spec.By), withParams("scope_version", fmt.Sprint(v.Spec.V), "size", fmt.Sprint(v.Size())))...)
			for _, id := range sortedKeys(v.Status) {
				if v.Status[id] == "excluded" && prev.Status[id] != "excluded" {
					m.ev("decision.containment.released", "decision", at, "Блок снят тем же шагом сужения: исключено из инцидента "+s.ID+" (≠ годно)", withItem(m.itemByID[id]), withAuthor(v.Spec.By))
				}
			}
		}
		prev = v
	}
	if !s.Closed.IsZero() {
		last := in.Versions[len(in.Versions)-1]
		m.ev("incident.incident.closed", "decision", s.Closed.Time(), fmt.Sprintf("Область закрыта: было %d; подтверждено %d; исключено %d", len(last.Status), last.Count("confirmed"), last.Count("excluded")), append(slices.Clone(opts), withAuthor("TEC-01"))...)
	}
}

func defectTitle(k string) string {
	return map[string]string{"burn_through": "прожог", "porosity": "поры", "undercut": "подрез", "base_metal_pore": "пора в основном металле", "missing_torque_mark": "нет метки затяжки", "scratch": "царапина"}[k]
}

func zoneTitle(z string) string {
	if strings.HasPrefix(z, "W-1.U") {
		return "У" + z[len("W-1.U"):]
	}
	if z == "RING-BODY" {
		return "в теле кольца"
	}
	return z
}

func strengthTitle(s string) string {
	return map[string]string{"not_assessable": "не оценить", "weak": "слабая", "possible": "возможна", "strong": "сильная", "confirmed": "подтверждена"}[s]
}

func dispositionTitle(k string) string {
	return map[string]string{"rework": "переделка", "repair": "ремонт", "use_as_is": "как есть", "scrap": "списать", "return_to_supplier": "вернуть поставщику", "none": "нет"}[k]
}

// finishEvents — порядок знания (AD-37): записи по времени записи; шаг — первый,
// часы которого не раньше записи; seq — внутри шага по порядку.
func (m *Model) finishEvents() {
	sort.SliceStable(m.Events, func(i, j int) bool {
		a, b := m.Events[i], m.Events[j]
		if !a.Recorded.Equal(b.Recorded) {
			return a.Recorded.Before(b.Recorded)
		}
		return a.Occurred.Before(b.Occurred)
	})
	var out []*Event
	perStep := map[int]int64{}
	for _, e := range m.Events {
		e.Step = m.StepOf(e.Recorded)
		if e.Step >= len(m.Steps) {
			continue
		}
		perStep[e.Step]++
		e.Seq = int64(e.Step)*loader.SeqPerStep + perStep[e.Step]
		out = append(out, e)
	}
	m.Events = out
}
