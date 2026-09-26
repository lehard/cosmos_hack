package vision

import (
	"slices"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Статусы паспорта допуска (AD-29, FR-101).
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusRetired   = "retired"
	// StatusNotAdmitted — паспорта нет или он ещё не действовал на момент.
	StatusNotAdmitted = "not_admitted"
)

// Виды анализаторов (Д-10): «Визуальный контроль» и «Контроль действий оператора».
const (
	KindVisionQC       = "visionqc"
	KindOperatorVision = "operatorvision"
)

// Status — смена статуса паспорта: допуск, приостановка (откат), возврат, вывод.
type Status struct {
	Status  string    `json:"status"`
	At      time.Time `json:"at"`
	EventID string    `json:"event_id"`
	Seq     int64     `json:"seq"`
	// Trigger, Fallback, FallbackPassportID — у приостановки (analyzer.passport.suspended).
	Trigger            string `json:"trigger,omitempty"`
	Fallback           string `json:"fallback,omitempty"`
	FallbackPassportID string `json:"fallback_passport_id,omitempty"`
	// Note — пояснение правила автоотката; RunID — прогон сценария записи (AD-38).
	Note  string   `json:"note,omitempty"`
	RunID string   `json:"run_id,omitempty"`
	Basis []string `json:"basis,omitempty"`
}

// Passport — паспорт допуска карты контроля (нормативный слой, AD-29) в
// представлении модуля vision: что допущено, кем записано, история статусов.
type Passport struct {
	PassportID   string `json:"passport_id"`
	AnalyzerID   string `json:"analyzer_id"`
	AnalyzerKind string `json:"analyzer_kind"`
	Title        string `json:"title"`
	// Stage — shadow | pilot | active; TrustLevel — 0…4 (тень всегда действует как 0).
	Stage      string   `json:"stage"`
	TrustLevel int      `json:"trust_level"`
	RecipeRef  string   `json:"recipe_ref"`
	Versions   Versions `json:"versions"`
	DocumentID string   `json:"document_id"`
	// PreviousPassportID — предыдущий допущенный паспорт для отката.
	PreviousPassportID string    `json:"previous_passport_id,omitempty"`
	AdmittedAt         time.Time `json:"admitted_at"`
	// Provenance — класс происхождения записи допуска (AD-2): genesis у демо-затравки.
	Provenance string   `json:"provenance"`
	History    []Status `json:"history"`
	// Seq — seq последней записи потока паспорта (версия потока, AD-39).
	Seq int64 `json:"seq"`
}

// Stream — поток паспорта `analyzer_passport:‹id›` (catalog.yaml, streams).
func Stream(passportID string) string { return "analyzer_passport:" + passportID }

// Current — действующий статус паспорта (последняя смена).
func (p Passport) Current() Status {
	if len(p.History) == 0 {
		return Status{Status: StatusNotAdmitted}
	}
	return p.History[len(p.History)-1]
}

// StatusAt — статус на момент t (по occurred_at смен): откат действует на
// будущее, наблюдения до него сохраняют прежний статус (FR-101).
func (p Passport) StatusAt(t time.Time) string {
	st := StatusNotAdmitted
	for _, h := range p.History {
		if h.At.After(t) {
			break
		}
		st = h.Status
	}
	return st
}

// LastSuspension — последняя приостановка, если паспорт сейчас приостановлен.
func (p Passport) LastSuspension() (Status, bool) {
	c := p.Current()
	return c, c.Status == StatusSuspended
}

// EffectiveLevel — уровень доверия, который действует сейчас: паспорт не в
// действии или в тени — 0 (только запись, AD-29).
func (p Passport) EffectiveLevel() int {
	if p.Current().Status != StatusActive || p.Stage == "shadow" {
		return 0
	}
	return min(max(p.TrustLevel, 0), 4)
}

// Check — отчёт проверки анализатора (analyzer.check.recorded, FR-101).
type Check struct {
	EventID            string    `json:"event_id"`
	Seq                int64     `json:"seq"`
	PassportID         string    `json:"passport_id"`
	Kind               string    `json:"check_kind"`
	EscapeRateBP       *int      `json:"escape_rate_bp,omitempty"`
	FalseAlarmRateBP   *int      `json:"false_alarm_rate_bp,omitempty"`
	DisagreementRateBP *int      `json:"disagreement_rate_bp,omitempty"`
	Passed             bool      `json:"passed"`
	OccurredAt         time.Time `json:"occurred_at"`
}

// Registry — паспорта и проверки анализаторов по записям семейства analyzer.
type Registry struct {
	Passports []Passport `json:"passports"`
	Checks    []Check    `json:"checks"`
	// BasisSeq — наибольший seq учтённых записей (AD-39).
	BasisSeq int64 `json:"basis_seq"`
}

// Passport — паспорт по id.
func (r Registry) Passport(id string) (Passport, bool) {
	for _, p := range r.Passports {
		if p.PassportID == id {
			return p, true
		}
	}
	return Passport{}, false
}

// ChecksOf — проверки паспорта по порядку записи.
func (r Registry) ChecksOf(id string) []Check {
	out := []Check{}
	for _, c := range r.Checks {
		if c.PassportID == id {
			out = append(out, c)
		}
	}
	return out
}

// analyzerOf — анализатор по версии, если запись допуска его не называет:
// первое слово версии («vqc-weld 2.3.1» → vqc-weld).
func analyzerOf(version string) string {
	f := strings.Fields(version)
	if len(f) == 0 {
		return Unknown
	}
	return f[0]
}

func bp(p *ev.Bp) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

// Fold — реестр паспортов из записей analyzer.* в порядке знания (seq, AD-37).
// Чистая функция: её вызывают операции чтения на момент (AD-22) и гарды
// команд. Нечитаемая запись пропускается — без паспорта уровень доверия 0 (строже).
func Fold(records []kernel.Record) Registry {
	in := slices.Clone(records)
	slices.SortStableFunc(in, func(a, b kernel.Record) int {
		switch {
		case a.Seq < b.Seq:
			return -1
		case a.Seq > b.Seq:
			return 1
		}
		return 0
	})
	reg := Registry{Passports: []Passport{}, Checks: []Check{}}
	idx := map[string]int{}
	status := func(r kernel.Record, id string, st Status) {
		i, ok := idx[id]
		if !ok {
			return
		}
		st.At, st.EventID, st.Seq, st.RunID = r.OccurredAt, r.EventID, r.Seq, r.RunID
		reg.Passports[i].History = append(reg.Passports[i].History, st)
		reg.Passports[i].Seq = r.Seq
	}
	for _, r := range in {
		reg.BasisSeq = max(reg.BasisSeq, r.Seq)
		switch r.Type {
		case catalog.AnalyzerPassportAdmitted:
			d, err := kernel.Decode[ev.AnalyzerPassportAdmittedV1](r)
			if err != nil {
				continue
			}
			p := Passport{PassportID: string(d.PassportID), Stage: string(d.Stage), TrustLevel: d.TrustLevel,
				RecipeRef: d.RecipeRef, Versions: FromContract(d.Versions), DocumentID: string(d.DocumentID),
				AdmittedAt: r.OccurredAt, Provenance: r.Provenance, AnalyzerKind: KindVisionQC}
			p.AnalyzerID = analyzerOf(d.Versions.AnalyzerVersion)
			if d.AnalyzerID != nil {
				p.AnalyzerID = string(*d.AnalyzerID)
			}
			if d.AnalyzerKind != nil {
				p.AnalyzerKind = string(*d.AnalyzerKind)
			}
			p.Title = p.AnalyzerID
			if d.Title != nil && *d.Title != "" {
				p.Title = *d.Title
			}
			if d.PreviousPassportID != nil {
				p.PreviousPassportID = string(*d.PreviousPassportID)
			}
			if i, ok := idx[p.PassportID]; ok {
				// Повторный допуск того же id гард не пропускает; запись всё же
				// есть в журнале — последняя конфигурация, история сохраняется.
				p.History = reg.Passports[i].History
				reg.Passports[i] = p
			} else {
				idx[p.PassportID] = len(reg.Passports)
				reg.Passports = append(reg.Passports, p)
			}
			status(r, p.PassportID, Status{Status: StatusActive})
		case catalog.AnalyzerPassportSuspended:
			d, err := kernel.Decode[ev.AnalyzerPassportSuspendedV1](r)
			if err != nil {
				continue
			}
			st := Status{Status: StatusSuspended, Trigger: string(d.Trigger), Fallback: string(d.Fallback)}
			if d.FallbackPassportID != nil {
				st.FallbackPassportID = string(*d.FallbackPassportID)
			}
			if d.Note != nil {
				st.Note = *d.Note
			}
			for _, b := range d.Basis {
				st.Basis = append(st.Basis, string(b))
			}
			status(r, string(d.PassportID), st)
		case catalog.AnalyzerPassportReinstated:
			d, err := kernel.Decode[ev.AnalyzerPassportReinstatedV1](r)
			if err != nil {
				continue
			}
			status(r, string(d.PassportID), Status{Status: StatusActive})
		case catalog.AnalyzerPassportRetired:
			d, err := kernel.Decode[ev.AnalyzerPassportRetiredV1](r)
			if err != nil {
				continue
			}
			status(r, string(d.PassportID), Status{Status: StatusRetired})
		case catalog.AnalyzerCheckRecorded:
			d, err := kernel.Decode[ev.AnalyzerCheckRecordedV1](r)
			if err != nil {
				continue
			}
			reg.Checks = append(reg.Checks, Check{EventID: r.EventID, Seq: r.Seq, PassportID: string(d.PassportID),
				Kind: string(d.CheckKind), EscapeRateBP: bp(d.EscapeRateBp), FalseAlarmRateBP: bp(d.FalseAlarmRateBp),
				DisagreementRateBP: bp(d.DisagreementRateBp), Passed: d.Passed, OccurredAt: r.OccurredAt})
		}
	}
	slices.SortFunc(reg.Passports, func(a, b Passport) int { return strings.Compare(a.PassportID, b.PassportID) })
	return reg
}

// autoActions — допустимые автоматические действия по уровням доверия
// (contracts/analyzer-trust-levels.yaml; уровни накопительные, AD-29).
// Совпадение с таблицей контракта проверяет тест application/vision.
var autoActions = [5][]string{
	{"record_observation"},
	{"recommend_to_inspector"},
	{"mark_suspect", "additional_check"},
	{"item_hold", "isolate"},
	{"auto_pass_confident"},
}

// AllowedAutoActions — что движок может делать сам по сигналу анализатора
// этого уровня: всё с уровней ≤ level (AD-29). Окончательный выпуск, ремонт,
// «как есть», списание и снятие блока не разрешены никаким уровнем (FR-98).
func AllowedAutoActions(level int) []string {
	level = min(max(level, 0), 4)
	out := []string{}
	for l := 0; l <= level; l++ {
		out = append(out, autoActions[l]...)
	}
	return out
}

// TrustLevels — число уровней доверия (0…4).
func TrustLevels() int { return len(autoActions) }
