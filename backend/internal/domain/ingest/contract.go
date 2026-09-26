package ingest

import (
	"slices"
	"strings"

	"ant/internal/contracts/errcodes"
)

// Outcome — итог классификации сообщения по контракту (FR-29, FR-30).
type Outcome string

// Итоги классификации.
const (
	// OutcomeAccepted — сообщение соответствует схеме (в том числе с новыми
	// необязательными полями: схемы событий открыты).
	OutcomeAccepted Outcome = "accepted"
	// OutcomeAcceptedWithFlag — принято с флагом: неизвестное значение
	// перечисления в поле, не важном для безопасности, хранится как UNKNOWN(значение).
	OutcomeAcceptedWithFlag Outcome = "accepted_with_flag"
	// OutcomeQuarantined — карантин с кодом.
	OutcomeQuarantined Outcome = "quarantined"
)

// ViolationKind — вид нарушения схемы, к которому сводится ошибка валидатора.
type ViolationKind string

// Виды нарушений.
const (
	ViolationRequired ViolationKind = "required"
	ViolationEnum     ViolationKind = "enum"
	ViolationOther    ViolationKind = "other"
)

// Violation — одно нарушение схемы в нейтральном виде: валидатор JSON Schema
// (application) переводит свои ошибки сюда, правила пяти случаев — здесь.
type Violation struct {
	Kind ViolationKind
	// Pointer — JSON Pointer места в сообщении (`/data/outcome`); для required —
	// путь объекта, в котором нет поля.
	Pointer string
	// Missing — отсутствующее поле (для required).
	Missing string
	// Value — полученное значение (для enum, строкой).
	Value string
	// Message — пояснение валидатора (для прочих нарушений).
	Message string
}

// Decision — решение приёма по контракту для одного сообщения.
type Decision struct {
	Outcome Outcome
	// Code — код из contracts/errors.yaml (пусто у принятого без флага).
	Code errcodes.Code
	// Field — JSON Pointer поля, из-за которого решение (`/data/outcome`).
	Field string
	// Value — исходное значение для неизвестного перечисления.
	Value string
	// Flags — флаги UNKNOWN(значение) для принятого с флагом (могут быть несколько).
	Flags []EnumFlag
	// Detail — пояснение для problem+json и записи карантина.
	Detail string
}

// EnumFlag — неизвестное значение перечисления, принятое как UNKNOWN(значение).
type EnumFlag struct {
	Field string
	Value string
}

// StoredAs — представление неизвестного значения: `UNKNOWN(значение)` (FR-29,
// «Соглашения/Неизвестность»).
func (f EnumFlag) StoredAs() string { return "UNKNOWN(" + f.Value + ")" }

// TypeInfo — то, что приём знает о типе события из каталога (AD-40).
type TypeInfo struct {
	Known    bool
	Versions []int
	// Fact — тип записи вида «факт»: через приём входят только факты источников
	// (AD-2); решения — командами API, реакции и служебные — ядро.
	Fact bool
}

// ClassifyVersion — случай 1 FR-29 и неизвестный тип: до проверки схемой.
// Возвращает решение и true, если сообщение дальше не проверяется.
func ClassifyVersion(eventType string, version int, t TypeInfo) (Decision, bool) {
	switch {
	case !t.Known:
		return Decision{Outcome: OutcomeQuarantined, Code: errcodes.IngestUnknownEventType, Field: "/event_type",
			Value: eventType, Detail: "тип «" + eventType + "» не объявлен в каталоге"}, true
	case !t.Fact:
		return Decision{Outcome: OutcomeQuarantined, Code: errcodes.IngestUnknownEventType, Field: "/event_type",
			Value: eventType, Detail: "тип «" + eventType + "» — не факт источника: решения входят командами, реакции и служебные записи пишет ядро (AD-2)"}, true
	case !slices.Contains(t.Versions, version):
		// FR-29, кейс §4.7, случай 1: неизвестная версия — карантин до появления схемы и повышателя.
		return Decision{Outcome: OutcomeQuarantined, Code: errcodes.IngestUnknownSchemaVersion, Field: "/schema_version",
			Detail: "версия неизвестна — сообщение в карантине до появления повышателя"}, true
	}
	return Decision{}, false
}

// Classify — случаи 2–5 FR-29 по нарушениям схемы (кейс §4.7, AD-20):
//   - нарушений нет — принято (новое необязательное поле тоже: схемы открыты, случай 2);
//   - нет обязательного поля — карантин ingest.missing_required_field (случай 3);
//     несовместимое изменение без новой мажорной версии проявляется так же (случай 5);
//   - только неизвестные значения перечислений: в поле, важном для безопасности
//     (safety-critical-enums.yaml), — карантин ingest.unknown_enum_value_critical,
//     иначе — принято с флагом UNKNOWN(значение) (случай 4);
//   - прочее — карантин ingest.schema_violation.
//
// Порядок нарушений не влияет на решение: они сортируются по указателю.
func Classify(eventType string, vs []Violation) Decision {
	if len(vs) == 0 {
		return Decision{Outcome: OutcomeAccepted}
	}
	vs = slices.Clone(vs)
	slices.SortFunc(vs, func(a, b Violation) int {
		return strings.Compare(a.Pointer+"/"+a.Missing, b.Pointer+"/"+b.Missing)
	})
	for _, v := range vs {
		if v.Kind == ViolationRequired {
			f := v.Pointer + "/" + v.Missing
			return Decision{Outcome: OutcomeQuarantined, Code: errcodes.IngestMissingRequiredField, Field: f,
				Detail: "нет обязательного поля «" + f + "»"}
		}
	}
	allEnum := true
	for _, v := range vs {
		allEnum = allEnum && v.Kind == ViolationEnum
	}
	if !allEnum {
		var msg []string
		for _, v := range vs {
			if v.Kind == ViolationOther {
				msg = append(msg, v.Pointer+": "+v.Message)
			}
		}
		first := vs[0]
		return Decision{Outcome: OutcomeQuarantined, Code: errcodes.IngestSchemaViolation, Field: first.Pointer,
			Detail: strings.Join(msg, "; ")}
	}
	for _, v := range vs {
		if IsSafetyCritical(eventType, v.Pointer) {
			// FR-29: неизвестное значение в поле, важном для безопасности, — карантин;
			// «годно» не додумывается (FR-123).
			return Decision{Outcome: OutcomeQuarantined, Code: errcodes.IngestUnknownEnumValueCritical, Field: v.Pointer,
				Value: v.Value, Detail: "поле «" + v.Pointer + "»: значение «" + v.Value + "» неизвестно"}
		}
	}
	d := Decision{Outcome: OutcomeAcceptedWithFlag, Code: errcodes.IngestUnknownEnumValue, Field: vs[0].Pointer, Value: vs[0].Value}
	for _, v := range vs {
		d.Flags = append(d.Flags, EnumFlag{Field: v.Pointer, Value: v.Value})
	}
	d.Detail = "поле «" + d.Field + "»: значение «" + d.Value + "» сохранено как " + d.Flags[0].StoredAs()
	return d
}
