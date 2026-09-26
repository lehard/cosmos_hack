package world

import (
	"fmt"
	"slices"
	"strings"
	"time"

	analysisapp "ant/internal/application/analysis"
	dom "ant/internal/domain/analysis"
)

// Контур улучшений мира (эпик 42; FR-63, FR-64, FR-138, FR-143): предложения
// генераторов, решения людей по ним и корректирующие меры по инцидентам
// главной истории. Предложения и меры — записи журнала мира (incident.
// suggestion.*, incident.action.*), поэтому ответы analysis.suggestion.list,
// analysis.action.list и analysis.data_deficit.read ссылаются на те же
// event_id, что показывает журнал. Тексты — как у встроенных генераторов live
// (domain/analysis), id предложений — тем же правилом SuggestionID.

// loopStep — решение человека по предложению: передать или принять/отклонить.
type loopStep struct {
	At         time.Time
	By         string
	To, Role   string // передано: кому и в какой роли
	Resolution string // принято в работу / отклонено
	Text       string
	EventID    string
}

// loopSuggestion — предложение генератора в мире.
type loopSuggestion struct {
	ID                                      string
	Generator, Kind, Title, Statement       string
	Estimate, Role, Step, Incident, Missing string
	Dedup                                   string
	At                                      time.Time
	Basis                                   []string
	Forward, Resolve                        *loopStep
	EventID                                 string
}

// loopEval — строка истории меры: внедрена или оценена.
type loopEval struct {
	Type, Result, Text, By, Evidence string
	At                               time.Time
	EventID                          string
}

// loopAction — корректирующая мера по инциденту (FR-64).
type loopAction struct {
	ID, Incident, Type, Direction, Owner, By, Title, Suggestion string
	Assigned, Due                                               time.Time
	Plan                                                        dom.EffectivenessPlan
	History                                                     []loopEval
	EventID                                                     string
}

// stepTitleOf — имя узла BPMN действующей версии по step_key (для людей);
// узла нет — сам step_key.
func (m *Model) stepTitleOf(k string) string {
	if n := m.Bpmn[k]; n != nil && n.Name != "" {
		return n.Name
	}
	return k
}

// pickEvents — event_id записей мира, известных к моменту at, по условию (не больше n).
func (m *Model) pickEvents(at time.Time, n int, pred func(*Event) bool) []string {
	var evs []*Event
	for _, e := range m.Events {
		if !e.Recorded.After(at) && pred(e) {
			evs = append(evs, e)
		}
	}
	slices.SortStableFunc(evs, func(a, b *Event) int { return a.Recorded.Compare(b.Recorded) })
	out := []string{}
	for _, e := range evs {
		if len(out) < n {
			out = append(out, e.ID)
		}
	}
	return out
}

func itemIs(e *Event, ids ...string) bool { return e.Item != nil && slices.Contains(ids, e.Item.ID) }

// buildLoop — предложения и меры главной истории (день 23–24) и их записи журнала.
func (m *Model) buildLoop() {
	at := m.clk.at
	weld := "welding.weld"
	byEntity := func(ids ...string) func(*Event) bool {
		return func(e *Event) bool { return slices.Contains(ids, e.Entity.ID) }
	}
	add := func(s *loopSuggestion) {
		s.ID = dom.SuggestionID(s.Generator, s.Dedup)
		m.Loop.Suggestions = append(m.Loop.Suggestions, s)
	}
	// 1. Область риска RS-01 без единого сужения — сузить по основаниям (FR-61).
	add(&loopSuggestion{Generator: "rules.risk_scope", Kind: dom.SuggestRiskScope, Incident: "RS-01", Step: weld, Role: dom.RoleTechnologist,
		Title:     "Сузить область риска RS-01 (34 изд.)",
		Statement: "Область риска инцидента RS-01 (общий фактор IS-2): 34 изделия, сужений по основаниям не было. Окно сбоя — после последней подтверждённо годной сварки Ф-006. Предлагается проверить изделия у границ окна и исключить подтверждённо годные с доказательствами — область сузится без риска пропуска.",
		Estimate:  "Под подозрением 34 изд.; каждое исключение — с основанием (FR-61)", Dedup: "risk_scope|narrow|RS-01|v1",
		At: at(23, 11, 11), Basis: m.pickEvents(at(23, 11, 11), 6, byEntity("RS-01", "NC-01")),
		Forward: &loopStep{At: at(23, 11, 14), By: "HQC-01", To: "TEC-01", Role: dom.RoleTechnologist, Text: "Сузить по журналам ИС-1 и ИС-2 до решения комиссии"},
		Resolve: &loopStep{At: at(23, 11, 45), By: "TEC-01", Resolution: "accepted", Text: "Область сужена до 13: исключены 21 сварка на ИС-1 (версия 2)"}})
	// 2. Изделие области ушло дальше — известить получателей (отклонено: не отгружено).
	add(&loopSuggestion{Generator: "rules.risk_scope", Kind: dom.SuggestRiskScope, Incident: "RS-01", Step: weld, Role: dom.RoleProductionManager,
		Title:     "Изделия области RS-01 ушли дальше: 1",
		Statement: "Из области риска инцидента RS-01 ушли дальше 1 и отгружены 0 изд. (Ф-015 — на сборке). Предлагается известить получателей и держать изделия под наблюдением до решения по области.",
		Dedup:     "risk_scope|downstream|RS-01|v1", At: at(23, 11, 11), Basis: m.pickEvents(at(23, 11, 11), 4, byEntity("RS-01")),
		Resolve: &loopStep{At: at(23, 12, 20), By: "PM-01", Resolution: "rejected", Text: "Ф-015 придержан на сборке, получателей нет — извещать некого"}})
	// 3. Ограничение линии: ИС-2 остановлен, вся сварка — на ИС-1 (UJ-1).
	add(&loopSuggestion{Generator: "rules.bottleneck", Kind: dom.SuggestBottleneck, Step: weld, Role: dom.RoleSiteForeman,
		Title:     "Ограничение линии: узел «" + m.stepTitleOf(weld) + "»",
		Statement: "Узел «" + m.stepTitleOf(weld) + "» — текущее ограничение линии: наибольшее ожидание при наибольшей загрузке (ИС-2 остановлен, сварка идёт только на ИС-1). Проверить обеспеченность поста людьми по графику смен (мастер участка); если причина в маршруте — новая версия процесса (технолог).",
		Estimate:  "Детали ждут в среднем 1 ч 10 мин; в очереди сейчас 9 дет.", Dedup: "bottleneck|" + weld + "|7",
		At: at(23, 11, 40), Basis: m.pickEvents(at(23, 11, 40), 3, byEntity("IS-2")),
		Forward: &loopStep{At: at(23, 11, 55), By: "PM-01", To: "FOR-WC", Role: dom.RoleSiteForeman, Text: "Вывести второго сварщика на ИС-1 во вторую смену"}})
	// 4. Адаптация VisionQC: камера КТ-3 не видит прожог в корне — нашёл рентген.
	kt3 := "welding.kt3_camera"
	add(&loopSuggestion{Generator: "rules.vision_adaptation", Kind: dom.SuggestAnalyzer, Step: kt3, Role: dom.RoleHeadOfQC,
		Title:     "Адаптация VisionQC: пропуски «прожог» на узле «" + m.stepTitleOf(kt3) + "»",
		Statement: "Камера не показала признаков дефекта «прожог» на узле «" + m.stepTitleOf(kt3) + "», а дефект затем нашли рентгеном: 2 случ. (NC-02, NC-03). Предлагается передать случаи в размеченные примеры и пересмотреть рецепт; новая версия анализатора — только через паспорт допуска.",
		Estimate:  "2 пропуск(ов) камеры", Dedup: "vision|" + kt3 + "|burn_through|2", At: at(23, 13, 25),
		Basis: m.pickEvents(at(23, 13, 25), 8, func(e *Event) bool { return e.Type == "inspection.result.recorded" && itemIs(e, "F-021", "F-023") })})
	// 5–6. Карта дефицита данных (FR-143): журнал оборудования и наблюдение до операции.
	add(&loopSuggestion{Generator: "rules.data_deficit", Kind: dom.SuggestDataDeficit, Missing: dom.MissingEquipmentLog, Role: dom.RoleProductionManager,
		Title:     "Цифровизация: " + dom.MissingText(dom.MissingEquipmentLog) + " — 2 из 4",
		Statement: "В 2 расследованиях из 4 " + dom.MissingText(dom.MissingEquipmentLog) + " (IS-2). Предлагается: журнал оборудования: датчики тока, вибрации, состояния — буфер на шлюзе IS-2 и тревога о потере связи.",
		Estimate:  "Сузило бы область риска в среднем с 34 до 6 дет. (по 1 инцидент(ам) с сужением по основаниям)", Dedup: "deficit|" + dom.MissingEquipmentLog + "|2",
		At: at(23, 13, 51), Basis: m.pickEvents(at(23, 13, 51), 6, byEntity("NC-01", "NC-04")),
		Forward: &loopStep{At: at(23, 14, 10), By: "PM-01", To: "ADM-01", Role: "administrator", Text: "Оценить буфер шлюза и тревогу потери связи"},
		Resolve: &loopStep{At: at(23, 16, 40), By: "PM-01", Resolution: "accepted", Text: "Принято в работу: мера по инциденту RS-01"}})
	add(&loopSuggestion{Generator: "rules.data_deficit", Kind: dom.SuggestDataDeficit, Missing: dom.MissingBefore, Role: dom.RoleProductionManager,
		Title:     "Цифровизация: " + dom.MissingText(dom.MissingBefore) + " — 2 из 5",
		Statement: "В 2 расследованиях из 5 " + dom.MissingText(dom.MissingBefore) + " (incoming.zt1_lot_acceptance). Предлагается: контроль зоны до операции (входной или межоперационный) — выборочный рентген колец на входном контроле.",
		Estimate:  "Сузило бы область риска в среднем с 3 до 1 дет. (по 1 инцидент(ам) с сужением по основаниям)", Dedup: "deficit|" + dom.MissingBefore + "|2",
		At: at(23, 14, 56), Basis: m.pickEvents(at(23, 14, 56), 6, byEntity("NC-04", "NC-05"))})
	// 7. Кандидат в правило реакции: три одинаковых решения «возврат из брака после повторной ЗТ-3».
	add(&loopSuggestion{Generator: "rules.reaction_rules", Kind: dom.SuggestReactionRule, Step: "welding.zt3_acceptance", Role: dom.RoleTechnologist,
		Title:     "Кандидат в правило реакции: возврат из брака после повторной ЗТ-3",
		Statement: "Люди 3 раза приняли одно и то же решение — возврат из брака в производство с основанием «переварка по ТП, повторная ЗТ-3 годно» (Ф-017, Ф-023, Ф-025; инцидент RS-01). Предлагается оформить правило реакции в режиме 2 «предлагать»: система будет предлагать это решение, человек — подтверждать.",
		Estimate:  "3 ручных решения", Dedup: "reaction_rule|decision|rework_return|3", At: at(24, 13, 30),
		Basis: m.pickEvents(at(24, 13, 30), 6, func(e *Event) bool {
			return e.Kind == "decision" && itemIs(e, "F-017", "F-023", "F-025") && e.Recorded.After(at(24, 0, 0))
		})})

	sugDeficitLog := m.Loop.Suggestions[4].ID
	m.Loop.Actions = []*loopAction{
		{Incident: "RS-02", Type: "correction", Direction: "prevent_occurrence", Owner: "HQC-01", By: "TEC-01",
			Title: "Возврат партии колец П-117 поставщику (7 шт.) и претензия", Assigned: at(23, 15, 42), Due: at(23, 18, 0),
			Plan: dom.EffectivenessPlan{Metric: "Кольца партии П-117 в производстве", Baseline: "7 колец на складе, 1 в изделии Ф-019", WindowDays: 1, SuccessCriterion: "Ни одного кольца П-117 в производстве"},
			History: []loopEval{{Type: "implemented", At: at(23, 16, 11), By: "HQC-01", Text: "Возврат проведён в 1С после исправления договора", Evidence: "Возврат товаров поставщику № 0000-000012"},
				{Type: "evaluated", Result: "effective", At: at(24, 16, 20), By: "HQC-01", Text: "Колец П-117 в производстве нет, кольцо К-101 снято с Ф-019"}}},
		{Incident: "RS-02", Type: "preventive_action", Direction: "improve_detection", Owner: "HQC-01", By: "TEC-01",
			Title: "Входной рентген выборки колец каждой партии (3 шт.)", Assigned: at(23, 15, 50), Due: at(25, 18, 0),
			Plan: dom.EffectivenessPlan{Metric: "Поры в теле кольца, найденные после сварки", Baseline: "2 случая за неделю (НС-04, НС-05)", WindowDays: 30,
				SuccessCriterion: "Ни одной поры в теле кольца после сварки за 30 дней", EnhancedControl: "Рентген каждого кольца до трёх чистых партий подряд"}},
		{Incident: "RS-01", Type: "correction", Direction: "prevent_occurrence", Owner: "FOR-WC", By: "HQC-01",
			Title: "Ремонт регулятора тока ИС-2 и проверка контрольным образцом", Assigned: at(23, 16, 25), Due: at(24, 12, 0),
			Plan: dom.EffectivenessPlan{Metric: "Сварки ИС-2 с током вне уставки 160 ± 10 А", Baseline: "7 сварок вне уставки за 22–23.09", WindowDays: 14,
				SuccessCriterion: "Ни одной сварки вне уставки за 14 дней", EnhancedControl: "Рентген каждого шва с ИС-2 до конца окна наблюдения"},
			History: []loopEval{{Type: "implemented", At: at(24, 11, 20), By: "FOR-WC", Text: "Регулятор заменён, ИС-2 пущен после проверки", Evidence: "Контрольный образец КО-8: ток 160 А, шов годен"}}},
		{Incident: "RS-01", Type: "corrective_action", Direction: "prevent_occurrence", Owner: "TEC-01", By: "HQC-01",
			Title: "Контроль тока сварки в реальном времени: остановка при выходе из уставки", Assigned: at(23, 16, 30), Due: at(24, 16, 0),
			Plan: dom.EffectivenessPlan{Metric: "Сварки, законченные с током вне уставки", Baseline: "7 за 22–23.09", WindowDays: 30, SuccessCriterion: "Ни одной сварки, законченной вне уставки"}},
		{Incident: "RS-01", Type: "corrective_action", Direction: "improve_detection", Owner: "ADM-01", By: "PM-01", Suggestion: sugDeficitLog,
			Title: "Буфер журнала на шлюзе ИС-2 и тревога о потере связи за 5 минут", Assigned: at(23, 16, 45), Due: at(30, 18, 0),
			Plan: dom.EffectivenessPlan{Metric: "Расследования без журнала оборудования", Baseline: "2 из 4 за неделю", WindowDays: 30, SuccessCriterion: "Ни одного расследования без журнала сварочных источников"}},
	}
	for _, a := range m.Loop.Actions {
		a.ID = "ACT-" + strings.ToUpper(strings.ReplaceAll(m.eventID("action/"+a.Title), "-", "")[:8])
	}
	m.loopEvents()
}

// Loop — контур улучшений мира: предложения и меры.
type Loop struct {
	Suggestions []*loopSuggestion
	Actions     []*loopAction
}

// loopEvents — записи журнала предложений и мер: факт предложения от сервера
// (класс server_attested), решения людей — личные записи (AD-3).
func (m *Model) loopEvents() {
	for _, s := range m.Loop.Suggestions {
		e := m.ev("incident.suggestion.recorded", "fact", s.At, "Предложение: "+s.Title, entity("suggestion", s.ID), withParams("suggestion_id", s.ID, "kind", s.Kind, "generator", s.Generator))
		e.Source, e.Provenance, e.SourceKind = analysisapp.SourceSuggestions, "server_attested", ""
		s.EventID = e.ID
		if f := s.Forward; f != nil {
			f.EventID = m.ev("incident.suggestion.forwarded", "decision", f.At, fmt.Sprintf("Предложение передано: %s — %s", m.personName(f.To), f.Text),
				entity("suggestion", s.ID), withAuthor(f.By), withParams("suggestion_id", s.ID, "responsible_id", f.To)).ID
		}
		if r := s.Resolve; r != nil {
			verb := map[string]string{"accepted": "принято в работу", "rejected": "отклонено"}[r.Resolution]
			r.EventID = m.ev("incident.suggestion.resolved", "decision", r.At, "Предложение "+verb+": "+r.Text,
				entity("suggestion", s.ID), withAuthor(r.By), withParams("suggestion_id", s.ID, "resolution", r.Resolution)).ID
		}
	}
	for _, a := range m.Loop.Actions {
		a.EventID = m.ev("incident.action.assigned", "decision", a.Assigned, fmt.Sprintf("Мера %s: %s — ответственный %s", a.ID, a.Title, m.personName(a.Owner)),
			entity("incident", a.Incident), withAuthor(a.By), withParams("incident_id", a.Incident, "action_id", a.ID, "action_type", a.Type)).ID
		for i := range a.History {
			h := &a.History[i]
			typ, sum := "incident.action.implemented", "Мера "+a.ID+" внедрена: "+h.Text
			if h.Type == "evaluated" {
				typ, sum = "incident.action.evaluated", "Мера "+a.ID+" — оценка эффективности: "+h.Text
			}
			h.EventID = m.ev(typ, "decision", h.At, sum, entity("incident", a.Incident), withAuthor(h.By), withParams("incident_id", a.Incident, "action_id", a.ID)).ID
		}
	}
}
