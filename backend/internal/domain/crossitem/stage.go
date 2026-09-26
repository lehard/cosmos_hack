package crossitem

import (
	"time"

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
type Own struct{}

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
// analysis. Слои: изделие → стадия → изделие.
func Fold(s Stage, r kernel.Record) (Stage, []kernel.Addressed) {
	var out, a []kernel.Addressed
	s.Own, a = own(s.Own, r)
	out = append(out, a...)
	s.Reference, a = reference.Stage(s.Reference, r)
	out = append(out, a...)
	s.Machinelogs, a = machinelogs.Stage(s.Machinelogs, r)
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
