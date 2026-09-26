package crossitem

import (
	"encoding/json"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
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
// analysis; поздний модуль видит адресованные записи ранних в том же проходе
// (Pass, решение Д-40). Слои: изделие → стадия → изделие. Неподвижную точку
// обеспечивает Settle.
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

// ModuleStep — подключённая функция модуля в стадии: имя модуля (по нему
// модуль не получает собственных адресованных записей) и шаг над его частью
// Stage.
type ModuleStep struct {
	Module kernel.Module
	Step   StepFunc
}

// Steps — подключённые функции стадии в фиксированном порядке (AD-42, AD-40):
// модули наполняют свои Stage, не трогая рамку.
var Steps = []ModuleStep{
	{Module, func(s Stage, r kernel.Record) (Stage, []kernel.Addressed) {
		var a []kernel.Addressed
		s.Own, a = own(s.Own, r)
		return s, a
	}},
	{"reference", func(s Stage, r kernel.Record) (Stage, []kernel.Addressed) {
		var a []kernel.Addressed
		s.Reference, a = reference.Stage(s.Reference, r)
		return s, a
	}},
	{machinelogs.Module, func(s Stage, r kernel.Record) (Stage, []kernel.Addressed) {
		var a []kernel.Addressed
		s.Machinelogs, a = machinelogs.Stage(s.Machinelogs, r)
		return s, a
	}},
	{documents.Module, func(s Stage, r kernel.Record) (Stage, []kernel.Addressed) {
		var a []kernel.Addressed
		s.Documents, a = documents.Stage(s.Documents, r)
		return s, a
	}},
	{nonconformity.Module, func(s Stage, r kernel.Record) (Stage, []kernel.Addressed) {
		var a []kernel.Addressed
		s.Nonconformity, a = nonconformity.Stage(s.Nonconformity, r)
		return s, a
	}},
	{analysis.Module, func(s Stage, r kernel.Record) (Stage, []kernel.Addressed) {
		var a []kernel.Addressed
		s.Analysis, a = analysis.Stage(s.Analysis, r)
		return s, a
	}},
}

// Modules — один проход подключённых функций стадии Steps по записи r (Pass).
func Modules(s Stage, r kernel.Record) (Stage, []kernel.Addressed) {
	return Pass(s, r, Steps)
}

// Pass — проход модулей стадии в фиксированном порядке по записи r (решение
// Д-40): каждый модуль получает r, а затем — адресованные записи модулей,
// стоящих раньше него, в порядке их выдачи, как записи стадии (AddressedRecord).
// Так окно нарушения machinelogs и несоответствия окна доходят до analysis в
// том же проходе, и анализ открывает или расширяет область.
//
// Правила:
//   - модуль не получает собственных записей: ни своих выходов этого прохода,
//     ни записей, эмитентом которых он указан (например, несоответствие окна,
//     построенное функцией-намерением nonconformity внутри шага machinelogs,
//     не приходит в nonconformity);
//   - доставляется только то, что будет выдано: запись, вызванная адресованной
//     записью самой стадии, выхода не даёт (Settle) — её выходы не
//     доставляются; уже выданная (id в Own.Emitted) и повтор в проходе — тоже;
//   - адресованные записи стадии, прочитанные потом из журнала, стадия заново
//     не толкует (application/crossitem.IsStageInput: роль crossitem) — каждый
//     модуль видит адресованную запись ровно один раз, в проходе её выдачи.
//
// Выход упорядочен: порядок модулей, внутри модуля — по порядку входа, затем
// как вернула функция.
func Pass(s Stage, r kernel.Record, steps []ModuleStep) (Stage, []kernel.Addressed) {
	type delivered struct {
		from, emitter kernel.Module // модуль, в шаге которого выдана; эмитент типа
		rec           kernel.Record
	}
	quiet := s.Own.Emitted[r.CausationID] // выходы будут подавлены Settle
	var out []kernel.Addressed
	var derived []delivered
	seen := map[string]bool{}
	for _, m := range steps {
		var a []kernel.Addressed
		s, a = m.Step(s, r)
		produced := a
		// Входы модуля — адресованные записи ранних модулей; свои выходы этого
		// прохода добавляются в derived после его шага, и модуль их не видит.
		for _, d := range derived {
			if d.from == m.Module || d.emitter == m.Module {
				continue
			}
			s, a = m.Step(s, d.rec)
			produced = append(produced, a...)
		}
		out = append(out, produced...)
		if quiet {
			continue
		}
		for _, x := range produced {
			id := AddressedID(x)
			if seen[id] || s.Own.Emitted[id] {
				continue
			}
			seen[id] = true
			if rec, ok := AddressedRecord(x, r); ok {
				derived = append(derived, delivered{from: m.Module, emitter: x.Module, rec: rec})
			}
		}
	}
	return s, out
}

// AddressedRecord — адресованная запись стадии как запись для поздних модулей
// того же прохода: те же event_id (AddressedID), тип, поток, occurred_at,
// причина и basis_seq, что у записи, которую потом запишет
// application/crossitem (addressedOut); Seq — seq записи-основания (своего
// seq у записи ещё нет). ok = false — данные не сериализуются (такую запись не
// запишет и кодек стадии).
func AddressedRecord(a kernel.Addressed, cause kernel.Record) (kernel.Record, bool) {
	data, err := json.Marshal(a.Data)
	if err != nil {
		return kernel.Record{}, false
	}
	info, _ := catalog.Lookup(a.Type)
	rec := kernel.Record{
		Seq: cause.Seq, EventID: AddressedID(a), Type: a.Type, SchemaVersion: info.CurrentVersion, Kind: info.Kind,
		SourceID: cause.SourceID, SourceKind: cause.SourceKind, Provenance: "server_attested",
		Stream: a.Stream, RunID: cause.RunID, OccurredAt: a.OccurredAt, ReceivedAt: cause.ReceivedAt, RecordedAt: cause.RecordedAt,
		CorrelationID: cause.CorrelationID, CausationID: cause.EventID, BasisSeq: cause.Seq, Data: data,
	}
	if rec.OccurredAt.IsZero() {
		rec.OccurredAt = cause.OccurredAt
	}
	if item, ok := strings.CutPrefix(a.Stream, "item:"); ok {
		rec.ItemID = item
	}
	return rec, true
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
