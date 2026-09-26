package analysis

import (
	"context"
	"fmt"
	"slices"
	"strings"

	dom "ant/internal/domain/analysis"
)

// Расследование на столе технолога в режиме live (эпик 12, «Бэкенд для
// интерфейса 4»): связь инцидента с разбором несоответствий, стадия, счётчики,
// «что сделать следующим», повод и разница ступеней области, история
// уверенности гипотез, «что проверить следующим», качество данных дорожек.

// Names — названия для людей из справочников нормативного слоя: имя человека
// по псевдониму (author_name версий области) и значение общего фактора по его
// виду (common_factor.label). nil в Config — названия не подставляются.
type Names interface {
	// PersonName — имя человека по псевдониму (справочник людей).
	PersonName(ctx context.Context, personID string) (string, bool)
	// FactorLabel — значение общего фактора словами: machine, tool, fixture —
	// справочник оборудования; performer — людей; material_batch — партий.
	FactorLabel(ctx context.Context, factor, value string) (string, bool)
	// DefectLabel — вид дефекта по-русски (классификатор видов дефектов).
	DefectLabel(ctx context.Context, code string) (string, bool)
	// StepName — имя шага BPMN действующей версии процесса.
	StepName(ctx context.Context, stepKey string) (string, bool)
}

// name — название из справочника через f; справочник не подключён или кода нет — nil.
func (s *Service) name(f func(Names) (string, bool)) *string {
	if s.cfg.Names == nil {
		return nil
	}
	if v, ok := f(s.cfg.Names); ok && v != "" {
		return &v
	}
	return nil
}

// labeled — общий фактор с названием из справочника (если подключён).
func (s *Service) labeled(ctx context.Context, f *FactorRef) *FactorRef {
	if f == nil || s.cfg.Names == nil {
		return f
	}
	if l, ok := s.cfg.Names.FactorLabel(ctx, f.Factor, f.Value); ok {
		f.Label = strp(l)
	}
	return f
}

// ncIndex — несоответствия и их профили для связи с инцидентами (один проход на запрос).
type ncIndex struct {
	ids      []string
	ncs      map[string]dom.NCRecord
	profiles map[string]dom.Profile
}

func (s *Service) ncIndex(ctx context.Context) (ncIndex, error) {
	ps, ncs, err := s.profiles(ctx)
	if err != nil {
		return ncIndex{}, err
	}
	x := ncIndex{ncs: ncs, profiles: map[string]dom.Profile{}}
	for _, p := range ps {
		x.ids = append(x.ids, p.NCID)
		x.profiles[p.NCID] = p
	}
	return x, nil
}

// link — несоответствия инцидента, ключ группы и ведущее НС: НС, отнесённые к
// инциденту решениями людей, или НС изделий его области; ведущее — НС,
// открывшее инцидент (trigger_event_ids), иначе первое с гипотезами.
func (x ncIndex) link(v dom.IncidentRecord) IncidentLink {
	out := IncidentLink{NCIDs: []string{}}
	var cause []string
	if v.Cause != nil {
		cause = v.Cause.NCIDs
	}
	for _, id := range x.ids {
		n := x.ncs[id]
		_, member := v.Members[n.ItemID]
		if slices.Contains(n.IncidentIDs, v.IncidentID) || slices.Contains(cause, id) || n.ItemID != "" && member {
			out.NCIDs = append(out.NCIDs, id)
		}
	}
	primary := ""
	for _, id := range out.NCIDs {
		if slices.Contains(v.TriggerEventIDs, x.ncs[id].EventID) {
			primary = id
			break
		}
	}
	if primary == "" {
		for _, id := range out.NCIDs {
			if x.ncs[id].Versions > 0 || len(x.ncs[id].Hypotheses) > 0 {
				primary = id
				break
			}
		}
	}
	if primary == "" && len(out.NCIDs) > 0 {
		primary = out.NCIDs[0]
	}
	if primary != "" {
		out.PrimaryNCID = strp(primary)
		if p, ok := x.profiles[primary]; ok {
			out.GroupKey = strp(dom.GroupKey(p))
		}
	}
	return out
}

// facts — факты расследования для стадии, «что дальше» и гарда закрытия.
func (s *Service) facts(ctx context.Context, v dom.IncidentRecord, primary *string) dom.InvestigationFacts {
	f := dom.InvestigationFacts{IncidentID: v.IncidentID, Closed: v.InvestigationClosed, ScopeClosed: v.Closed, Causes: map[string]string{}}
	for _, m := range v.Members {
		f.Counts.Add(m.Status)
	}
	for _, x := range v.Versions {
		f.Narrowed = f.Narrowed || x.Change == dom.ChangeNarrowed
	}
	for _, h := range v.Hypotheses {
		if h.Verdict != "rejected" {
			f.Hypotheses++
		}
	}
	for b, c := range v.Causes {
		f.Causes[b] = c.Conclusion
	}
	if c := v.Cause; c != nil && len(v.Causes) == 0 {
		f.Causes[dom.BranchWhyMade] = c.Conclusion
	}
	for _, a := range v.Actions {
		f.Actions = append(f.Actions, a.Status)
	}
	if primary != nil {
		if n, err := s.nc(ctx, *primary); err == nil {
			if a, _, err := s.analyze(ctx, n); err == nil {
				best := -1
				for _, h := range a.Hypotheses {
					f.Hypotheses++
					c := 0
					if h.ConfidenceBP != nil {
						c = *h.ConfidenceBP
					}
					if r := measured(n, h.ID); r != nil && r.Outcome != dom.MeasurementInconclusive {
						continue // проверка выполнена — дальше вывод о причине человеком
					}
					if h.MeasurementHint != "" && c > best {
						best, f.NextCheck = c, h.MeasurementHint
					}
				}
			}
		}
	}
	return f
}

// investigationState — шапка расследования (stage, counts, last_event_at, next_step).
func investigationState(v dom.IncidentRecord, f dom.InvestigationFacts) InvestigationState {
	st := InvestigationState{Stage: dom.InvestigationStage(f), Counts: KnownCountsView(f.Counts), LastEventAt: v.LastEventAt}
	st.NextStep = strp(dom.InvestigationNextStep(f))
	st.CloseBlockers = closeBlockers(f)
	return st
}

// versionDiff — повод и разница ступени области (live): решение человека или
// правило системы; подписант — автор решения (личный ключ, AD-3).
func (s *Service) versionDiff(ctx context.Context, v dom.IncidentRecord, x dom.VersionRecord) ScopeVersionDiff {
	d := ScopeVersionDiff{ItemsAdded: append([]string{}, x.Added...), ItemsRemoved: append([]string{}, x.Removed...), Evidence: []JournalRecordRef{}}
	at := x.RecordedAt
	if dec, ok := v.Decisions[x.DecisionID]; ok && x.DecisionID != "" {
		label := "Решение " + dec.Actor
		if t := strings.TrimSpace(dec.Reason.Text); t != "" {
			label += ": " + t
		}
		occurred := dec.At
		d.Trigger = &ScopeTrigger{Kind: "human", Label: label, EventID: strp(x.DecisionID), ReceivedAt: &at, OccurredAt: &occurred}
		d.SignedBy, d.KeyClass = strp(dec.Actor), strp("personal")
		d.Evidence = append(d.Evidence, JournalRecordRef{EventID: x.DecisionID, EventType: dec.Type, OccurredAt: dec.At, Text: strp(dec.Reason.Text), SourceLabel: strp(dec.Actor)})
	} else {
		label := "Правило системы"
		if x.Reason != nil && x.Reason.Text != "" {
			label = x.Reason.Text
		}
		d.Trigger = &ScopeTrigger{Kind: "computed", Label: label, EventID: strp(x.EventID), ReceivedAt: &at}
	}
	if x.Author != "" && s.cfg.Names != nil {
		if name, ok := s.cfg.Names.PersonName(ctx, x.Author); ok {
			d.AuthorName = strp(name)
		}
	}
	return d
}

// hypothesisHistory — что меняло уверенность системной гипотезы: версии вывода
// (incident.hypothesis.computed), отклонение и подтверждение причины людьми.
func hypothesisHistory(n dom.NCRecord, id, category string) []HypothesisChange {
	out := []HypothesisChange{}
	prev := -1
	for i, cv := range n.Computed {
		c, ok := cv.Confidence[category]
		if !ok || c == prev {
			continue
		}
		text := fmt.Sprintf("Версия вывода %d: уверенность %s", i+1, bpText(c))
		if prev >= 0 {
			text += " (было " + bpText(prev) + ")"
		}
		c2 := c
		out = append(out, HypothesisChange{At: cv.At, ConfidenceBP: &c2, EventID: strp(cv.EventID), Text: text})
		prev = c
	}
	for _, h := range n.Hypotheses {
		if h.HypothesisID == id && h.Verdict == "rejected" {
			text := "Отклонена человеком (" + h.Actor + ")"
			if h.Reason != nil && h.Reason.Text != "" {
				text += ": " + h.Reason.Text
			}
			out = append(out, HypothesisChange{At: h.At, EventID: strp(h.EventID), Text: text})
		}
	}
	// Проверка гипотезы измерением: запрос и результат (FR-59, FR-135).
	for _, m := range n.Measurements {
		if m.HypothesisID != id {
			continue
		}
		out = append(out, HypothesisChange{At: m.At, EventID: strp(m.EventID), Text: "Запрошена проверка: " + m.What})
		if r := m.Result; r != nil {
			var c *int
			if bp, ok := resultConfidence(r.Outcome, prev); ok {
				c = &bp
				prev = bp
			}
			out = append(out, HypothesisChange{At: r.At, ConfidenceBP: c, EventID: strp(r.EventID),
				Text: "Результат проверки «" + m.What + "» — " + outcomeText(r.Outcome) + ": " + r.Text})
		}
	}
	if c := n.Cause; c != nil && c.Conclusion == "confirmed" && c.Category == category {
		out = append(out, HypothesisChange{At: c.At, EventID: strp(c.EventID), Text: "Причина подтверждена (" + c.Actor + "): " + c.Reason.Text})
	}
	slices.SortStableFunc(out, func(a, b HypothesisChange) int { return a.At.Compare(b.At) })
	return out
}

// Уверенность гипотезы после результата измерения (базисные пункты):
// подтверждение — сильная, опровержение — слабая; «оценить нельзя» не меняет.
const (
	confidenceMeasuredSupports = 9500
	confidenceMeasuredRefutes  = 500
)

// resultConfidence — уверенность после результата измерения; ok=false — не меняется.
func resultConfidence(outcome string, prev int) (int, bool) {
	switch outcome {
	case dom.MeasurementSupports:
		return max(prev, confidenceMeasuredSupports), true
	case dom.MeasurementRefutes:
		return confidenceMeasuredRefutes, true
	}
	return 0, false
}

// measured — последний результат измерения по гипотезе (nil — нет).
func measured(n dom.NCRecord, id string) *dom.MeasurementResult {
	var out *dom.MeasurementResult
	for _, m := range n.Measurements {
		if m.HypothesisID == id && m.Result != nil {
			out = m.Result
		}
	}
	return out
}

func outcomeText(o string) string {
	switch o {
	case dom.MeasurementSupports:
		return "подтверждает гипотезу"
	case dom.MeasurementRefutes:
		return "опровергает гипотезу"
	}
	return "оценить нельзя"
}

// bpText — базисные пункты долей словами: 8500 → «0,85».
func bpText(bp int) string {
	return strings.Replace(fmt.Sprintf("%.2f", float64(bp)/10000), ".", ",", 1)
}

// measurementKind — вид проверки по категории гипотезы.
func measurementKind(category string) string {
	switch category {
	case dom.CatEquipment:
		return "control_sample"
	case dom.CatIncoming:
		return "sample_inspection"
	case dom.CatPerformer:
		return "explanation"
	case dom.CatDocumentation:
		return "document_check"
	}
	return "other"
}

// nextCheck — «что проверить следующим» по подсказке гипотезы; оценка
// исключаемых — изделия области под подозрением и без данных.
func nextCheck(h dom.Hypothesis, inc *dom.IncidentRecord) *NextCheck {
	if h.MeasurementHint == "" {
		return nil
	}
	x := &NextCheck{Text: h.MeasurementHint, MeasurementKind: measurementKind(h.Category),
		UnlocksText: "Подтвердить или отклонить гипотезу: подтверждение открывает вывод о причине, отклонение — сужение области по основаниям"}
	if inc != nil {
		var c dom.KnownCounts
		for _, m := range inc.Members {
			c.Add(m.Status)
		}
		x.CouldExclude, x.ScopeSize = c.Suspect+c.Unknown, inc.Size()
	}
	return x
}

// laneQualities — качество данных дорожек (live): пропуски — из нехватки
// сведений разбора; опоздания live пока не считает (нужно время записи в
// отметках дорожек).
func laneQualities(a dom.Analysis) *LaneQualities {
	q := &LaneQualities{Item: LaneQuality{Gaps: []DataGap{}}, Person: LaneQuality{Gaps: []DataGap{}}, Equipment: LaneQuality{Gaps: []DataGap{}}}
	if a.Window == nil {
		return q
	}
	from, to := a.Window.Start, a.Window.End
	src := ""
	if r := a.Operation; r != nil {
		from, src = r.Started, r.Equipment
		if r.Finished != nil {
			to = *r.Finished
		}
	}
	for _, m := range a.Missing {
		gap := DataGap{From: from, To: to, Source: src}
		switch m {
		case dom.MissingEquipmentLog:
			gap.Text = "Журнал оборудования за время операции не пришёл — исключать нельзя"
			q.Equipment.Gaps = append(q.Equipment.Gaps, gap)
		case dom.MissingOperator, dom.MissingTool:
			gap.Text = "Исполнитель или инструмент неизвестен — не додумываем"
			q.Person.Gaps = append(q.Person.Gaps, gap)
		case dom.MissingAfter, dom.MissingBefore:
			gap.Text = "Нет наблюдения зоны до или после операции — исключать нельзя"
			q.Item.Gaps = append(q.Item.Gaps, gap)
		}
	}
	return q
}

// markText — запись дорожки словами (live): вид записи и значение против уставки.
func markText(mk dom.Mark) string {
	p := mk.Params
	switch {
	case strings.HasPrefix(mk.EventType, "equipment."):
		s := "Оборудование"
		if mk.Variant != "" {
			s += " (" + mk.Variant + ")"
		}
		if p["parameter"] != "" && p["value"] != "" {
			s += ": " + p["parameter"] + " " + p["value"]
			if p["setpoint"] != "" {
				s += " при уставке " + p["setpoint"]
			}
		}
		return s
	case mk.EventType == "inspection.result.recorded":
		switch mk.Variant {
		case "defect_indicated":
			return "Контроль: признак дефекта"
		case "no_defect_indicated":
			return "Контроль: признаков нет"
		case "unable_to_assess":
			return "Контроль: оценка невозможна"
		}
		return "Результат контроля"
	case strings.HasPrefix(mk.EventType, "operation.run"):
		return "Выполнение операции"
	case strings.HasPrefix(mk.EventType, "decision."):
		return "Решение человека"
	}
	return ""
}

// closeBlockers — блокеры закрытия расследования для шапки.
func closeBlockers(f dom.InvestigationFacts) []CloseBlocker {
	out := []CloseBlocker{}
	for _, b := range dom.CloseBlockers(f) {
		out = append(out, CloseBlocker{Code: string(b.Code), Text: b.Text})
	}
	return out
}
