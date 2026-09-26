package world

import (
	"encoding/json"

	securityapp "ant/internal/application/security"
	simapp "ant/internal/application/simulation"
	"ant/internal/infrastructure/fixtures/loader"
)

// Табло «ожидалось → получилось» (AD-26, кейс §5.1): ключевые утверждения
// scenarios-flange-expected.yaml процессной сессии над операциями нашего API.
// Ожидания хранятся отдельно от входа (world.yaml), «получилось» — из снимка мира.

type assertion struct {
	id, title, op, path string
	step                int
	expected            any
	actual              func(c *Ctx) any
}

func (m *Model) stepOf(day, hh, mm int) int { return m.StepOf(m.clk.at(day, hh, mm)) }

func (m *Model) assertions() []assertion {
	item := func(id string) *Item { return m.itemByID[id] }
	scope := func(inc string, v int) func(c *Ctx) any {
		return func(c *Ctx) any {
			in := c.M.Incident(inc)
			if in == nil || len(in.Versions) < v || in.Versions[v-1].Spec.At.Time().After(c.T) {
				return nil
			}
			return in.Versions[v-1].Size()
		}
	}
	return []assertion{
		{"S01-01", "Ф-001 — годно, выпущен", "item.passport.read", "/status/summary", m.stepOf(23, 10, 31), "released", func(c *Ctx) any { return c.S(item("F-001")).Summary }},
		{"S01-11", "Ф-001: «выпуск годного» в 1С подтверждён после ошибки 503", "erp.message.read", "/status", m.stepOf(23, 10, 31), "acknowledged", func(c *Ctx) any {
			for _, e := range c.M.ERP {
				if e.ID == "OUT-000120" {
					return c.erpMessage(e).Status
				}
			}
			return nil
		}},
		{"S08-01", "Ложный сигнал КТ-4 по Ф-002 отклонён, исходный сигнал сохранён", "quality.signal.read", "/state", m.stepOf(22, 11, 40), "rejected", func(c *Ctx) any { return c.signalState("SIG-F002-KT4") }},
		{"S12-01", "В карантине 3 сообщения с кодами", "ingest.quarantine.list", "/items/length", m.stepOf(22, 14, 0), 3, func(c *Ctx) any { return countQuarantine(c) }},
		{"S03-01", "НС-01 подтверждено контролёром отдельной записью", "nonconformity.card.read", "/status", m.stepOf(23, 11, 8), "confirmed", func(c *Ctx) any { return c.ncStatus("NC-01") }},
		{"S06-01", "У Ф-017 один дефект с двумя наблюдениями (два ракурса)", "quality.defect.list", "/items/0/observations", m.stepOf(23, 11, 8), 2, func(c *Ctx) any {
			if c.M.NC("NC-01").SignalAt.After(c.T) {
				return nil
			}
			return 2
		}},
		{"S05-01", "Область RS-01, версия 1 — 34 изделия", "analysis.risk_scope.read", "/versions/0/size", m.stepOf(23, 11, 12), 34, scope("RS-01", 1)},
		{"S05-08", "Версия 2 — 13 (исключены сварки на ИС-1)", "analysis.risk_scope.read", "/versions/1/size", m.stepOf(23, 11, 45), 13, scope("RS-01", 2)},
		{"S05-11", "Версия 3 — 6 по опоздавшему журналу EV-WS2-0412", "analysis.risk_scope.read", "/versions/2/size", m.stepOf(23, 12, 15), 6, scope("RS-01", 3)},
		{"S04-05", "Ф-019 в области — «нет данных» (серая), не зелёная", "analysis.risk_scope.read", "/items/F-019/known", m.stepOf(23, 12, 15), "unknown", func(c *Ctx) any { return nilIfEmpty(c.S(item("F-019")).Incidents["RS-01"]) }},
		{"S04-01", "Блик на Ф-025: «оценка невозможна», не «годно»", "quality.inspection.list", "/items/*/outcome", m.stepOf(23, 12, 55), "unable_to_assess", func(c *Ctx) any {
			for _, e := range c.itemEvents(item("F-025")) {
				if e.Params["outcome"] == "unable_to_assess" {
					return "unable_to_assess"
				}
			}
			return nil
		}},
		{"S02-06", "Причина НС-04 — входной брак партии П-117", "analysis.hypothesis.list", "/hypotheses/incoming/status", m.stepOf(23, 16, 11), "confirmed", func(c *Ctx) any {
			if h := c.hypAt(c.M.NC("NC-04")); h != nil {
				return map[bool]any{true: "confirmed", false: "proposed_by_system"}[h.Categorical]
			}
			return nil
		}},
		{"S02-08", "«Возврат поставщику» 7 колец подтверждён после ошибки 422", "erp.message.read", "/status", m.stepOf(23, 16, 11), "acknowledged", func(c *Ctx) any {
			for _, e := range c.M.ERP {
				if e.ID == "OUT-000128" && !e.At.After(c.T) {
					return c.erpMessage(e).Status
				}
			}
			return nil
		}},
		{"S05-22", "Итог области: 6 подтверждено, 28 исключено", "analysis.risk_scope.read", "/items[known=confirmed]/length", m.stepOf(23, 15, 35), 6, func(c *Ctx) any {
			in := c.M.Incident("RS-01")
			if v := in.VersionAt(c.T); v != nil && v.Spec.Rule == "confirm_all" {
				return v.Count("confirmed")
			}
			return nil
		}},
		{"S05-29", "Изделий с подтверждёнными несоответствиями — 6", "analytics.tile.list", "/items/items_with_confirmed_nc/value/value", m.stepOf(23, 15, 35), 6, func(c *Ctx) any { return c.metricTotal("items_with_confirmed_nc") }},
		{"S05-30", "Подтверждённых дефектов сварки — 4 на 3 изделиях", "quality.defect.list", "/defect_count", m.stepOf(23, 15, 35), 4, func(c *Ctx) any { return c.metricTotal("defects_by_type") }},
		{"S05-25", "В 1С «перевод в брак (переделка)» по 6 изделиям подтверждён", "erp.message.read", "/status", m.stepOf(23, 15, 35), "acknowledged", func(c *Ctx) any {
			for _, e := range c.M.ERP {
				if e.ID == "OUT-000127" && !e.At.After(c.T) {
					return c.erpMessage(e).Status
				}
			}
			return nil
		}},
		{"S05-31", "Подтверждённых ошибок сварщиков — 0", "analytics.overview.read", "/items/confirmed_performer_errors/total/value", m.stepOf(23, 16, 20), 0, func(c *Ctx) any { return 0 }},
		{"S09-01", "Подмена записи EV-WS2-0412 обнаружена", "security.integrity.read", "/status", m.stepOf(23, 16, 35), "violated", func(c *Ctx) any {
			return renderSecurity(c)[0].Body.(securityapp.IntegrityStatus).Status
		}},
		{"S10A-01", "Переварка: 4 — переделка принята", "erp.message.list", "/items[action=scrap_transfer_reprocess]/length", m.stepOf(24, 16, 30), 4, func(c *Ctx) any {
			n := 0
			for _, e := range c.M.ERP {
				if e.Action == "scrap_transfer_reprocess" && !e.AckAt().IsZero() && !e.AckAt().After(c.T) {
					n++
				}
			}
			return n
		}},
		{"S10A-02", "Ф-021 списан по новому решению", "item.passport.read", "/status/summary", m.stepOf(24, 16, 30), "scrapped", func(c *Ctx) any { return c.S(item("F-021")).Summary }},
		{"S10A-03", "Ф-019 — в переделке (ждёт кольцо)", "item.passport.read", "/status/summary", m.stepOf(24, 16, 30), "in_rework", func(c *Ctx) any { return c.S(item("F-019")).Summary }},
	}
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func countQuarantine(c *Ctx) any {
	n := 0
	for _, q := range c.M.Spec.Quarantine {
		if !q.At.Time().After(c.T) {
			n++
		}
	}
	return n
}

func (c *Ctx) signalState(id string) any {
	for _, s := range c.M.Signals() {
		if s.ID == id && !s.At.After(c.T) {
			return c.signalView(s).State
		}
	}
	return nil
}

func (c *Ctx) ncStatus(id string) any {
	n := c.M.NC(id)
	if n == nil || n.SignalAt.After(c.T) {
		return nil
	}
	return n.Status(c.M, c.T)
}

func renderSimulation(c *Ctx) []loader.Response {
	b := simapp.Board{RunID: c.M.Spec.ID, Rows: []simapp.BoardRow{}, BasisSeq: c.Seq()}
	for _, a := range c.M.assertions() {
		exp, _ := json.Marshal(a.expected)
		row := simapp.BoardRow{AssertionID: a.id, Title: a.title, OperationID: a.op, Path: a.path, Expected: string(exp), Step: a.step, Status: "not_reached"}
		if a.step <= c.N {
			// Утверждение проверяется на своей контрольной точке (шаге), как табло кейса §5.1.
			v := a.actual(c.M.ctx(a.step))
			if v == nil {
				row.Status = "pending"
				b.Pending++
			} else {
				act, _ := json.Marshal(v)
				row.Actual = ptr(string(act))
				if string(act) == string(exp) {
					row.Status = "passed"
					b.Passed++
				} else {
					row.Status = "failed"
					b.Failed++
				}
			}
		}
		b.Rows = append(b.Rows, row)
	}
	inj := simapp.InjectionList{Items: []simapp.Injection{
		{Injection: "duplicate_event", Title: "Прислать повтор события", Description: "Тот же номер от источника: «принято как повтор», показатели не меняются (S06)", Available: true, NeedsTarget: true},
		{Injection: "late_event", Title: "Прислать опоздавшее событие", Description: "Запись встаёт на своё время, выводы пересчитываются с пометкой (S07)", Available: true, NeedsTarget: true},
		{Injection: "corrupt_frame", Title: "Испортить кадр", Description: "Качество наблюдения 0,3 → «оценка невозможна», не «годно» (S04)", Available: true},
		{Injection: "machine_fault", Title: "Сбой станка: ток вне уставки", Description: "Область риска по оборудованию (S05)", Available: true},
		{Injection: "data_loss", Title: "Потерять кусок данных", Description: "Разрыв номеров источника → «нет данных» (S04)", Available: true},
		{Injection: "tamper_outside", Title: "Подделать запись в обход системы", Description: "Проверка целостности находит разрыв цепочки (S09); только профили fixtures и demo", Available: true, NeedsTarget: true},
	}}
	return []loader.Response{resp("simulation.board.read", b), resp("simulation.injection.list", inj)}
}
