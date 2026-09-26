package vision

import (
	"slices"
	"strconv"

	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Действия над паспортом (x-ant-action, AD-40).
const (
	ActionAdmit     = "vision.passport.admit"
	ActionReinstate = "vision.passport.reinstate"
	ActionRetire    = "vision.passport.retire"
)

// RoleHeadOfQC — роль начальника ОТК: только он возвращает анализатор после
// отката (FR-101, AD-27; normative/policy: head_of_qc).
const RoleHeadOfQC = "head_of_qc"

// Admission — команда допуска версии анализатора (FR-98) в представлении домена.
type Admission struct {
	PassportID         string
	AnalyzerID         string
	AnalyzerKind       string
	Stage              string
	TrustLevel         int
	RecipeRef          string
	Versions           Versions
	PreviousPassportID string
	DocumentID         string
}

func invalid(field, reason string) *kernel.Refusal {
	r := kernel.Refuse(errcodes.ApiValidationFailed, "field", field, "reason", reason)
	r.Detail = field + ": " + reason
	return r
}

func transition(p Passport, action string) *kernel.Refusal {
	return kernel.Refuse(errcodes.AnalyzerInvalidTransition, "passport_id", p.PassportID, "status", p.Current().Status, "action", action)
}

// GuardAdmit — гард допуска (AD-39): id паспорта новый, предыдущий паспорт
// существует, карта контроля совпадает с вектором версий, уровень 0…4,
// в векторе есть версия анализатора и контракта.
func GuardAdmit(reg Registry, a Admission) error {
	if p, ok := reg.Passport(a.PassportID); ok {
		return transition(p, ActionAdmit)
	}
	if a.TrustLevel < 0 || a.TrustLevel > 4 {
		return invalid("trust_level", "уровень доверия 0…4")
	}
	if !slices.Contains([]string{"shadow", "pilot", "active"}, a.Stage) {
		return invalid("stage", "стадия shadow | pilot | active")
	}
	if a.AnalyzerKind != "" && a.AnalyzerKind != KindVisionQC && a.AnalyzerKind != KindOperatorVision {
		return invalid("analyzer_kind", "visionqc | operatorvision")
	}
	if a.RecipeRef == "" || a.Versions.RecipeRef != a.RecipeRef {
		return invalid("versions.recipe_ref", "карта контроля в векторе версий должна совпадать с recipe_ref")
	}
	for _, c := range []struct{ k, v string }{{"versions.analyzer_version", a.Versions.AnalyzerVersion}, {"versions.contract_version", a.Versions.ContractVersion}} {
		if c.v == "" || c.v == Unknown {
			return invalid(c.k, "допускается конкретная версия, а не «неизвестно»")
		}
	}
	if a.DocumentID == "" {
		return invalid("document_id", "нужен протокол допуска")
	}
	if a.PreviousPassportID != "" {
		if _, ok := reg.Passport(a.PreviousPassportID); !ok {
			return kernel.Refuse(errcodes.ApiNotFound, "object", "analyzer_passport", "id", a.PreviousPassportID)
		}
	}
	return nil
}

// GuardReinstate — гард возврата после отката (FR-101, AD-27): паспорт
// приостановлен именно этой приостановкой; вернуть может только начальник ОТК.
// headOfQC — субъект действует в роли начальника ОТК с наследованием ролей
// (Principal.HasRole: роль-наследник начальника ОТК тоже вправе; эпик 26 —
// сравнение строки активной роли не учитывало наследование); actorRole — для
// текста отказа.
func GuardReinstate(reg Registry, passportID, suspensionEventID, actorRole string, headOfQC bool) (Passport, error) {
	p, ok := reg.Passport(passportID)
	if !ok {
		return p, kernel.Refuse(errcodes.ApiNotFound, "object", "analyzer_passport", "id", passportID)
	}
	if !headOfQC {
		return p, kernel.Refuse(errcodes.AnalyzerReinstateRequiresHeadOfQc, "role", actorRole)
	}
	s, suspended := p.LastSuspension()
	if !suspended {
		return p, transition(p, ActionReinstate)
	}
	if s.EventID != suspensionEventID {
		return p, invalid("suspension_event_id", "действующая приостановка паспорта — "+s.EventID+" (seq "+strconv.FormatInt(s.Seq, 10)+")")
	}
	return p, nil
}

// GuardRetire — гард вывода паспорта: выведенный повторно не выводится.
func GuardRetire(reg Registry, passportID string) (Passport, error) {
	p, ok := reg.Passport(passportID)
	if !ok {
		return p, kernel.Refuse(errcodes.ApiNotFound, "object", "analyzer_passport", "id", passportID)
	}
	if p.Current().Status == StatusRetired {
		return p, transition(p, ActionRetire)
	}
	return p, nil
}
