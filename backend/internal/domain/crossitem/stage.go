package crossitem

import (
	"time"

	"ant/internal/contracts/constants"
	"ant/internal/domain/analysis"
	"ant/internal/domain/documents"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
	"ant/internal/domain/nonconformity"
	"ant/internal/domain/reference"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "crossitem"

// Own — собственное состояние стадии: генеалогия, партии, садки, плавки,
// временные группы, привязка событий без изделия, расход разрешений на
// отклонение (проекция), точки процесса (AD-42).
type Own struct {
	// Emitted — event_id адресованных записей, уже выданных стадией (рамка
	// эпика 07): повтор той же записи подавляется, записи, вызванные своими
	// адресованными, не порождают новых (неподвижная точка, AD-42).
	Emitted map[string]bool `json:"emitted,omitempty"`
}

// Stage — состояние межизделийной стадии целиком: своё и подключённых модулей.
type Stage struct {
	Own           Own
	Reference     reference.StageState
	Machinelogs   machinelogs.StageState
	Documents     documents.StageState
	Nonconformity nonconformity.StageState
	Analysis      analysis.StageState
}

// Fold — шаг межизделийной стадии (AD-42): запись стадии → адресованные
// записи. Порядок подключённых функций фиксирован: своё (привязка,
// генеалогия, партии) → reference → machinelogs → documents → nonconformity →
// analysis. Слои: изделие → стадия → изделие. Неподвижную точку обеспечивает
// Settle.
func Fold(s Stage, r kernel.Record) (Stage, []kernel.Addressed) {
	return Settle(s, r, Modules)
}

// StepFunc — шаг подключённых функций стадии без правил неподвижной точки.
type StepFunc func(s Stage, r kernel.Record) (Stage, []kernel.Addressed)

// AddressedID — event_id адресованной записи стадии: UUIDv5(NS_ANT, эмитент ‖
// тип ‖ поток-адресат ‖ ключ) (AD-42, «Соглашения/Идентификаторы»). Один и тот
// же вывод стадии при повторе даёт тот же id.
func AddressedID(a kernel.Addressed) string {
	return kernel.UUIDv5(constants.NsAnt, "stage\x1f"+string(a.Module)+"\x1f"+string(a.Type)+"\x1f"+a.Stream+"\x1f"+a.Key)
}

// Settle — шаг стадии с правилами неподвижной точки (AD-42):
//   - запись, непосредственно вызванная адресованной записью самой стадии
//     (causation_id ∈ выданных), меняет состояние, но выхода не даёт — «без
//     нового факта стадия не реагирует на записи, вызванные её же
//     адресованными записями»;
//   - адресованная запись с уже выданным id не повторяется.
//
// Выход упорядочен: порядок подключённых функций, внутри — как вернула функция.
func Settle(s Stage, r kernel.Record, step StepFunc) (Stage, []kernel.Addressed) {
	s, out := step(s, r)
	if s.Own.Emitted[r.CausationID] {
		return s, nil
	}
	var kept []kernel.Addressed
	for _, a := range out {
		id := AddressedID(a)
		if s.Own.Emitted[id] {
			continue
		}
		if s.Own.Emitted == nil {
			s.Own.Emitted = map[string]bool{}
		}
		s.Own.Emitted[id] = true
		kept = append(kept, a)
	}
	return s, kept
}

// Modules — подключённые функции стадии в фиксированном порядке (AD-42):
// модули волны 4 наполняют свои Stage, не трогая рамку.
func Modules(s Stage, r kernel.Record) (Stage, []kernel.Addressed) {
	var out, a []kernel.Addressed
	s.Own, a = own(s.Own, r)
	out = append(out, a...)
	s.Reference, a = reference.Stage(s.Reference, r)
	out = append(out, a...)
	s.Machinelogs, a = machinelogs.StageWith(machinelogs.StagePorts{Register: nonconformity.RegisterWindowNC})(s.Machinelogs, r)
	out = append(out, a...)
	s.Documents, a = documents.Stage(s.Documents, r)
	out = append(out, a...)
	s.Nonconformity, a = nonconformity.Stage(s.Nonconformity, r)
	out = append(out, a...)
	s.Analysis, a = analysis.Stage(s.Analysis, r)
	out = append(out, a...)
	return s, out
}

// own — собственные правила стадии: binding.link.resolved, genealogy.link.added
// (обоим изделиям), партии и группы.
func own(s Own, r kernel.Record) (Own, []kernel.Addressed) {
	_ = r
	return s, nil
}

// CarrierRef — носитель из события источника: тип и значение (AD-16).
type CarrierRef struct {
	Type  string
	Value string
}

// Resolution — результат разрешения носителя (AD-41): изделие, основание и
// надёжность привязки; неоднозначное — кандидаты и поток стадии.
type Resolution struct {
	ItemID      string
	Basis       string
	Reliability string
	Candidates  []string
	// Ambiguous — событие идёт в поток межизделийной стадии; стадия пишет
	// binding.link.resolved с кандидатами или одним изделием.
	Ambiguous bool
}

// CarrierRegistry — реестр носителей на момент (с учётом «снят / заменён»),
// построенный из журнала до basis_seq.
type CarrierRegistry interface {
	// Lookup — изделия, у которых носитель ref действовал на момент at.
	Lookup(ref CarrierRef, at time.Time) []string
}

// ResolveCarrier — разрешение носителя (AD-41): чистая функция над реестром
// носителей на occurred_at события. Приём (application/ingest) вызывает её до
// выбора партиции и записывает результат; верификатор проверяет повторным
// вычислением.
func ResolveCarrier(reg CarrierRegistry, ref CarrierRef, at time.Time) Resolution {
	c := reg.Lookup(ref, at)
	switch len(c) {
	case 1:
		return Resolution{ItemID: c[0], Basis: "carrier"}
	case 0:
		return Resolution{Ambiguous: true}
	default:
		return Resolution{Candidates: c, Ambiguous: true}
	}
}
