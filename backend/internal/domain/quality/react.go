package quality

import (
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Правила слотов реакций модуля (AD-3): слот — (правило, субъект, ключ
// срабатывания). Правило в слоте — стабильное имя вывода модуля, а не строка
// карты реакций: смена строки карты при пересвёртке — новая версия того же
// слота («пересмотрен из-за записи ‹id›»), а не исчезновение защиты.
const (
	RuleSignal   = "quality.signal"
	RuleDefect   = "quality.defect"
	RuleLinked   = "quality.observation"
	RuleMissing  = "quality.missing"
	RuleEscape   = "quality.escape"
	RuleAutoPass = "quality.auto_pass"
)

// SignalRaised — реакция quality.signal.raised в слоте slot (AD-40: запись
// строит функция модуля-эмитента). data — ev.QualitySignalRaisedV1.
func SignalRaised(slot kernel.Slot, data any, causes ...kernel.Record) (kernel.Reaction, error) {
	return kernel.NewReaction(Module, catalog.QualitySignalRaised, slot, data, causes...)
}

// reactions — реакции модуля по выводам состояния.
func reactions(s State, env Env) []kernel.Reaction {
	subject := "item:" + s.ItemID
	out := []kernel.Reaction{}
	add := func(t catalog.Type, rule, key string, mode int, data any, causes []string) {
		r, err := kernel.NewReaction(Module, t, kernel.Slot{RuleID: rule, Subject: subject, TriggerKey: key}, data, s.records(causes)...)
		if err != nil {
			// Типы модуля — из каталога: ошибка здесь — ошибка сборки, её ловят тесты.
			panic(err)
		}
		r.RuleRev, r.AutomationMode = env.RuleRev, max(mode, 1)
		out = append(out, r)
	}
	for _, d := range s.Defects {
		data := ev.QualityDefectIdentifiedV1{DefectID: ev.ObjectID(d.DefectID), ZoneID: ev.ObjectID(nz(d.Zone, "unknown")),
			FirstObservationEventID: ev.UUID(d.FirstObservation), Location: strp(d.Location), DefectTypeCode: strp(d.TypeCode)}
		causes := []string{d.FirstObservation}
		if d.TypeSource != "" {
			causes = append(causes, d.TypeSource)
		}
		add(catalog.QualityDefectIdentified, RuleDefect, d.DefectID, 1, data, causes)
		changed := map[string]string{}
		for _, tc := range d.TypeChanges {
			changed[tc.ObservationID] = tc.TypeCode
		}
		for _, oid := range d.Observations[1:] {
			ld := ev.QualityObservationLinkedV1{DefectID: ev.ObjectID(d.DefectID), ObservationEventID: ev.UUID(oid)}
			if c, ok := changed[oid]; ok {
				ld.DefectTypeCode = strp(c)
			}
			add(catalog.QualityObservationLinked, RuleLinked, oid+"/"+d.DefectID, 1, ld, []string{oid})
		}
	}
	for _, sg := range s.Signals {
		if !sg.Raised {
			continue
		}
		a := sg.Assessment
		data := ev.QualitySignalRaisedV1{
			SignalID: ev.ObjectID(sg.SignalID), BasisKind: ev.QualitySignalRaisedV1BasisKind(sg.Basis),
			Severity: ev.Severity(nz(sg.Severity, "unknown")), ReactionOutcome: ev.QualitySignalRaisedV1ReactionOutcome(a.Outcome),
			ReactionMapRef: a.MapRef, DefectTypeCode: strp(sg.TypeCode), RequirementRef: strp(sg.RequirementRef),
			AnalyzerConfidenceBp: bp(sg.ConfidenceBP), ObservationQualityBp: bp(sg.QualityBP), TrustLevel: sg.TrustLevel,
		}
		if sg.DefectID != "" {
			id := ev.ObjectID(sg.DefectID)
			data.DefectID = &id
		}
		if sg.Zone != "" {
			z := ev.ObjectID(sg.Zone)
			data.ZoneID = &z
		}
		if sg.TypeCode != "" {
			k := sg.TypeKnown
			data.DefectTypeKnown = &k
		}
		add(catalog.QualitySignalRaised, RuleSignal, sg.SignalID, a.Mode, data, sg.Causes)
	}
	for _, p := range s.Points {
		if p.Status != "missing" || !p.Required {
			continue
		}
		data := ev.QualityInspectionMissingV1{StepKey: ev.StepKey(p.StepKey), MissingReason: ev.QualityInspectionMissingV1MissingReason(p.MissingReason),
			InspectionPoint: strp(nz(p.InspectionPoint, p.ClosingPoint))}
		m := ev.InspectionMethod(p.Method)
		data.Method = &m
		if p.Window != "" {
			w := ev.ObjectID(p.Window)
			data.AfterOperationRunID = &w
		}
		add(catalog.QualityInspectionMissing, RuleMissing, p.StepKey+"/"+p.Method+"@"+p.Window, 1, data, p.Causes)
	}
	for _, e := range s.Escapes {
		missed := make([]ev.UUID, 0, len(e.Missed))
		for _, m := range e.Missed {
			missed = append(missed, ev.UUID(m))
		}
		add(catalog.QualityEscapeRecorded, RuleEscape, e.DefectID, 1, ev.QualityEscapeRecordedV1{DefectID: ev.ObjectID(e.DefectID),
			MissedObservationEventIds: missed, MethodCoversDefect: e.MethodCovers, AnalyzerVersion: strp(e.AnalyzerVersion)}, e.Causes)
	}
	for _, ap := range s.AutoPasses {
		// Разрешающее действие — по явно делегированному правилу режима 2 (AD-27).
		add(catalog.QualityInspectionAutoPassed, RuleAutoPass, ap.ObservationID, 2, ev.QualityInspectionAutoPassedV1{
			ObservationEventID: ev.UUID(ap.ObservationID), PassportID: ev.ObjectID(nz(ap.PassportID, "unknown")),
			TrustLevel: ap.TrustLevel, SampledForHumanRecheck: ap.Sampled}, []string{ap.ObservationID})
	}
	return out
}

// records — записи-основания по event_id с occurred_at (реакция получает
// наибольший occurred_at причин, AD-37).
func (s State) records(ids []string) []kernel.Record {
	out := make([]kernel.Record, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.record(id))
	}
	return out
}

// record — запись входа по event_id (только event_id и occurred_at).
func (s State) record(id string) kernel.Record {
	return kernel.Record{EventID: id, OccurredAt: s.occurred(id)}
}

func (s State) occurred(id string) time.Time {
	for _, o := range s.Observations {
		if o.EventID == id {
			return o.OccurredAt
		}
	}
	for _, r := range s.Runs {
		if r.EventID == id {
			return r.OccurredAt
		}
	}
	for _, x := range s.Skips {
		if x.EventID == id {
			return x.OccurredAt
		}
	}
	for _, x := range s.Reports {
		if x.EventID == id {
			return x.OccurredAt
		}
	}
	for _, x := range s.Presentations {
		if x.EventID == id {
			return x.OccurredAt
		}
	}
	for _, x := range s.Reviews {
		if x.EventID == id {
			return x.OccurredAt
		}
	}
	for _, x := range s.Rechecks {
		if x.EventID == id {
			return x.OccurredAt
		}
	}
	return time.Time{}
}

func strp(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func nz(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func bp(v *int) *ev.Bp {
	if v == nil {
		return nil
	}
	b := ev.Bp(*v)
	return &b
}
