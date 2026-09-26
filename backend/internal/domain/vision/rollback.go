package vision

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Автоматический откат версии анализатора (FR-101, AD-27, AD-29): делегированное
// правило режима 2 (FR-50) — ограничивает, а не разрешает. Триггеры: дрейф
// входных данных, провал эталонного набора, рост расхождений с людьми,
// пропуск брака. Откат — запись analyzer.passport.suspended («паспорт
// приостановлен»): действует на будущее, старые наблюдения сохраняют
// «проанализировано версией …»; вернуть анализатор — только решение
// начальника ОТК (GuardReinstate). Критерии заданы до испытаний и не
// меняются по ходу (FR-101): константы ниже — часть этого правила.

// RuleRollback — правило автоотката (слот реакции analyzer.passport.suspended).
const RuleRollback = "vision.rollback"

// Критерии автоотката.
const (
	// DriftWindow — сколько наблюдений подряд проверяет контроль дрейфа.
	DriftWindow = 3
	// DriftQualityBP — качество кадра ниже этого (0,70) при ответе анализатора —
	// кадр не похож на кадры допуска (свет, оптика, положение): DriftWindow
	// таких наблюдений подряд — дрейф входных данных.
	DriftQualityBP = 7000
	// DisagreementMaxBP — доля расхождений с людьми выше 20 % в отчёте проверки —
	// «рост расхождений с людьми».
	DisagreementMaxBP = 2000
)

// Триггеры отката (contracts/events/analyzer/analyzer.passport.suspended.v1.json).
const (
	TriggerDrift        = "drift"
	TriggerReferenceSet = "reference_set_failed"
	TriggerDisagreement = "disagreement_growth"
	TriggerEscape       = "escape_detected"
)

// Что действует вместо приостановленного паспорта.
const (
	FallbackPrevious = "previous_passport"
	FallbackManual   = "manual_control"
)

// Frame — наблюдение анализатора в окне контроля дрейфа.
type Frame struct {
	EventID   string    `json:"event_id"`
	ItemID    string    `json:"item_id,omitempty"`
	QualityBP int       `json:"quality_bp"`
	At        time.Time `json:"at"`
}

// RunWatch — контроль паспорта в одном прогоне сценария (AD-38; "" — живая
// работа): приостановка и окно последних наблюдений.
type RunWatch struct {
	RunID             string  `json:"run_id"`
	Suspended         bool    `json:"suspended"`
	SuspensionEventID string  `json:"suspension_event_id,omitempty"`
	Suspensions       int     `json:"suspensions"`
	Recent            []Frame `json:"recent,omitempty"`
	// Seen — сколько наблюдений анализатора этого паспорта учтено.
	Seen int `json:"seen"`
}

// Watched — паспорт под контролем автоотката.
type Watched struct {
	PassportID         string     `json:"passport_id"`
	RecipeRef          string     `json:"recipe_ref"`
	AnalyzerVersion    string     `json:"analyzer_version"`
	PreviousPassportID string     `json:"previous_passport_id,omitempty"`
	AdmittedAt         time.Time  `json:"admitted_at"`
	Retired            bool       `json:"retired,omitempty"`
	Runs               []RunWatch `json:"runs"`
}

// run — контроль в прогоне (создаётся при первом обращении).
func (w *Watched) run(id string) *RunWatch {
	for i := range w.Runs {
		if w.Runs[i].RunID == id {
			return &w.Runs[i]
		}
	}
	w.Runs = append(w.Runs, RunWatch{RunID: id})
	slices.SortFunc(w.Runs, func(a, b RunWatch) int { return strings.Compare(a.RunID, b.RunID) })
	for i := range w.Runs {
		if w.Runs[i].RunID == id {
			return &w.Runs[i]
		}
	}
	return nil
}

// RunOf — контроль в прогоне без создания.
func (w Watched) RunOf(id string) (RunWatch, bool) {
	for _, r := range w.Runs {
		if r.RunID == id {
			return r, true
		}
	}
	return RunWatch{RunID: id}, false
}

// Watch — состояние правила автоотката по всем паспортам: свёртка записей
// журнала в порядке seq (глобальный потребитель роли projector, AD-45).
type Watch struct {
	Passports []Watched `json:"passports"`
}

func (s *Watch) find(id string) *Watched {
	for i := range s.Passports {
		if s.Passports[i].PassportID == id {
			return &s.Passports[i]
		}
	}
	return nil
}

// Watches — записи, которые меняют состояние правила автоотката.
var Watches = []catalog.Type{
	catalog.AnalyzerPassportAdmitted, catalog.AnalyzerPassportSuspended, catalog.AnalyzerPassportReinstated,
	catalog.AnalyzerPassportRetired, catalog.AnalyzerCheckRecorded, catalog.InspectionResultRecorded, catalog.QualityEscapeRecorded,
}

// Step — одна запись журнала: новое состояние и приостановки, которые правило
// должно записать (реакции модуля vision, версия 1; повтор даёт тот же id).
// Чистая функция (AD-4): время — только из записей.
func (s Watch) Step(r kernel.Record) (Watch, []kernel.Reaction) {
	s = s.clone()
	switch r.Type {
	case catalog.AnalyzerPassportAdmitted:
		d, err := kernel.Decode[ev.AnalyzerPassportAdmittedV1](r)
		if err != nil {
			return s, nil
		}
		w := Watched{PassportID: string(d.PassportID), RecipeRef: d.RecipeRef, AnalyzerVersion: d.Versions.AnalyzerVersion, AdmittedAt: r.OccurredAt}
		if d.PreviousPassportID != nil {
			w.PreviousPassportID = string(*d.PreviousPassportID)
		}
		if p := s.find(w.PassportID); p != nil {
			w.Runs = p.Runs
			*p = w
		} else {
			s.Passports = append(s.Passports, w)
			slices.SortFunc(s.Passports, func(a, b Watched) int { return strings.Compare(a.PassportID, b.PassportID) })
		}
	case catalog.AnalyzerPassportSuspended:
		d, err := kernel.Decode[ev.AnalyzerPassportSuspendedV1](r)
		if err != nil {
			return s, nil
		}
		if p := s.find(string(d.PassportID)); p != nil {
			rw := p.run(r.RunID)
			if !rw.Suspended || rw.SuspensionEventID != r.EventID {
				rw.Suspensions++
			}
			rw.Suspended, rw.SuspensionEventID, rw.Recent = true, r.EventID, nil
		}
	case catalog.AnalyzerPassportReinstated:
		d, err := kernel.Decode[ev.AnalyzerPassportReinstatedV1](r)
		if err != nil {
			return s, nil
		}
		if p := s.find(string(d.PassportID)); p != nil {
			rw := p.run(r.RunID)
			rw.Suspended, rw.SuspensionEventID, rw.Recent = false, "", nil
		}
	case catalog.AnalyzerPassportRetired:
		d, err := kernel.Decode[ev.AnalyzerPassportRetiredV1](r)
		if err != nil {
			return s, nil
		}
		if p := s.find(string(d.PassportID)); p != nil {
			p.Retired = true
		}
	case catalog.InspectionResultRecorded:
		return s.frame(r)
	case catalog.QualityEscapeRecorded:
		return s.escape(r)
	case catalog.AnalyzerCheckRecorded:
		return s.check(r)
	}
	return s, nil
}

// clone — копия состояния: Step не меняет вход (чистая функция).
func (s Watch) clone() Watch {
	out := Watch{Passports: make([]Watched, len(s.Passports))}
	for i, p := range s.Passports {
		p.Runs = slices.Clone(p.Runs)
		for j := range p.Runs {
			p.Runs[j].Recent = slices.Clone(p.Runs[j].Recent)
		}
		out.Passports[i] = p
	}
	return out
}

// watching — паспорт, чья версия анализатора дала наблюдение: карта контроля и
// версия анализатора совпадают, паспорт допущен раньше наблюдения и не выведен.
func (s *Watch) watching(recipe, analyzer string, at time.Time) *Watched {
	for i := range s.Passports {
		p := &s.Passports[i]
		if p.Retired || p.AdmittedAt.After(at) || p.AnalyzerVersion != analyzer || (recipe != "" && p.RecipeRef != recipe) {
			continue
		}
		return p
	}
	return nil
}

// frame — контроль дрейфа: DriftWindow ответов анализатора подряд на кадрах
// хуже DriftQualityBP (кейс: «эмулятор меняет свет → дрейф», FR-101).
func (s Watch) frame(r kernel.Record) (Watch, []kernel.Reaction) {
	d, err := kernel.Decode[ev.InspectionResultRecordedV1](r)
	if err != nil || d.Method != ev.InspectionMethodCamera || d.Versions == nil || d.ObservationQualityBp == nil {
		return s, nil
	}
	// Анализатор ответил: «оценить нельзя» или прерванная обработка — не ответ
	// (система сама признала кадр плохим — это не дрейф, а честный отказ).
	if d.ProcessingState != ev.ProcessingStateCompleted || d.Outcome == ev.InspectionOutcomeUnableToAssess {
		return s, nil
	}
	p := s.watching(d.Versions.RecipeRef, d.Versions.AnalyzerVersion, r.OccurredAt)
	if p == nil {
		return s, nil
	}
	rw := p.run(r.RunID)
	if rw.Suspended {
		return s, nil
	}
	rw.Seen++
	rw.Recent = append(rw.Recent, Frame{EventID: r.EventID, ItemID: r.ItemID, QualityBP: int(*d.ObservationQualityBp), At: r.OccurredAt})
	if len(rw.Recent) > DriftWindow {
		rw.Recent = rw.Recent[len(rw.Recent)-DriftWindow:]
	}
	if len(rw.Recent) < DriftWindow {
		return s, nil
	}
	basis, worst := []string{}, 10000
	for _, f := range rw.Recent {
		if f.QualityBP >= DriftQualityBP {
			return s, nil
		}
		basis = append(basis, f.EventID)
		worst = min(worst, f.QualityBP)
	}
	note := fmt.Sprintf("дрейф входных данных: качество кадра ниже %s в %d наблюдениях подряд (худшее %s), анализатор %s продолжал отвечать",
		Share(DriftQualityBP), DriftWindow, Share(worst), p.AnalyzerVersion)
	return s, s.suspend(p, r, TriggerDrift, basis, note)
}

// escape — пропуск брака (FR-100): дефект найден позже, ранние «признаков нет»
// дала эта версия анализатора, и метод способен выявить этот вид — ошибка модели.
func (s Watch) escape(r kernel.Record) (Watch, []kernel.Reaction) {
	d, err := kernel.Decode[ev.QualityEscapeRecordedV1](r)
	if err != nil || !d.MethodCoversDefect || d.AnalyzerVersion == nil || *d.AnalyzerVersion == "" {
		return s, nil
	}
	var out []kernel.Reaction
	basis := []string{}
	for _, m := range d.MissedObservationEventIds {
		basis = append(basis, string(m))
	}
	basis = append(basis, r.EventID)
	for i := range s.Passports {
		p := &s.Passports[i]
		if p.Retired || p.AnalyzerVersion != *d.AnalyzerVersion || p.AdmittedAt.After(r.OccurredAt) {
			continue
		}
		if rw, _ := p.RunOf(r.RunID); rw.Suspended {
			continue
		}
		note := fmt.Sprintf("пропуск брака: дефект %s найден позже, ранние наблюдения «признаков нет» (%d) дала версия %s, метод способен выявить этот вид",
			d.DefectID, len(d.MissedObservationEventIds), p.AnalyzerVersion)
		out = append(out, s.suspend(p, r, TriggerEscape, basis, note)...)
	}
	return s, out
}

// check — отчёт проверки анализатора: провал эталонного набора, рост
// расхождений с людьми, провал контроля дрейфа.
func (s Watch) check(r kernel.Record) (Watch, []kernel.Reaction) {
	d, err := kernel.Decode[ev.AnalyzerCheckRecordedV1](r)
	if err != nil {
		return s, nil
	}
	p := s.find(string(d.PassportID))
	if p == nil || p.Retired {
		return s, nil
	}
	if rw, _ := p.RunOf(r.RunID); rw.Suspended {
		return s, nil
	}
	trigger, note := "", ""
	switch {
	case d.CheckKind == ev.AnalyzerCheckRecordedV1CheckKindReferenceSet && !d.Passed:
		trigger, note = TriggerReferenceSet, "провал эталонного набора: критерии допуска не выполнены"
	case d.DisagreementRateBp != nil && int(*d.DisagreementRateBp) > DisagreementMaxBP:
		trigger, note = TriggerDisagreement, fmt.Sprintf("расхождения с людьми %s — выше порога %s", Share(int(*d.DisagreementRateBp)), Share(DisagreementMaxBP))
	case d.CheckKind == ev.AnalyzerCheckRecordedV1CheckKindDriftMonitor && !d.Passed:
		trigger, note = TriggerDrift, "контроль дрейфа не пройден"
	default:
		return s, nil
	}
	return s, s.suspend(p, r, trigger, []string{r.EventID}, note)
}

// suspend — реакция «паспорт приостановлен» в прогоне записи-триггера: откат
// к предыдущему допущенному паспорту, если он в действии, иначе 100 % ручной
// контроль (FR-101). Состояние сразу помечается приостановленным: повторный
// триггер в той же пачке не пишет вторую приостановку.
func (s *Watch) suspend(p *Watched, r kernel.Record, trigger string, basis []string, note string) []kernel.Reaction {
	rw := p.run(r.RunID)
	n := rw.Suspensions + 1
	fallback, prev := FallbackManual, ""
	if q := s.find(p.PreviousPassportID); q != nil && !q.Retired {
		if qr, _ := q.RunOf(r.RunID); !qr.Suspended {
			fallback, prev = FallbackPrevious, q.PassportID
		}
	}
	slices.Sort(basis)
	basis = slices.Compact(basis)
	d := ev.AnalyzerPassportSuspendedV1{PassportID: ev.ObjectID(p.PassportID), Trigger: ev.AnalyzerPassportSuspendedV1Trigger(trigger),
		Fallback: ev.AnalyzerPassportSuspendedV1Fallback(fallback)}
	for _, b := range basis {
		d.Basis = append(d.Basis, ev.UUID(b))
	}
	if prev != "" {
		v := ev.ObjectID(prev)
		d.FallbackPassportID = &v
	}
	if note != "" {
		n := clip(note, 512)
		d.Note = &n
	}
	slot := kernel.Slot{RuleID: RuleRollback, Subject: Stream(p.PassportID), TriggerKey: r.RunID + "#" + strconv.Itoa(n)}
	re, err := kernel.NewReaction(Module, catalog.AnalyzerPassportSuspended, slot, d, r)
	if err != nil {
		panic(err) // тип и эмитент — из каталога; ошибка — дефект сборки
	}
	re.AutomationMode = 2
	rw.Suspended, rw.SuspensionEventID, rw.Suspensions, rw.Recent = true, re.ID(1), n, nil
	return []kernel.Reaction{re}
}

// Share — доля в б. п. как «0,70» (интерфейс и пояснения по-русски).
func Share(bp int) string {
	return strconv.Itoa(bp/10000) + "," + fmt.Sprintf("%02d", (bp%10000)/100)
}

// MarshalState, UnmarshalState — состояние правила в проекции (JSON).
func (s Watch) MarshalState() (json.RawMessage, error) { return json.Marshal(s) }

// UnmarshalState — состояние из проекции; пусто — начальное.
func UnmarshalState(raw json.RawMessage) (Watch, error) {
	var s Watch
	if len(raw) == 0 {
		return Watch{Passports: []Watched{}}, nil
	}
	err := json.Unmarshal(raw, &s)
	return s, err
}
