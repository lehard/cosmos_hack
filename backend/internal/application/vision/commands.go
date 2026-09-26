package vision

import "ant/internal/application/platform"

// AdmitPassport — допустить версию анализатора (analyzer.passport.admitted,
// FR-98): после закрытия маршрута протокола допуска (document_id).
type AdmitPassport struct {
	platform.CommandHeader
	PassportID         string            `json:"passport_id" maxLength:"128"`
	AnalyzerID         string            `json:"analyzer_id" maxLength:"128"`
	Stage              string            `json:"stage" enum:"shadow,pilot,active"`
	TrustLevel         int               `json:"trust_level" minimum:"0" maximum:"4"`
	RecipeRef          string            `json:"recipe_ref"`
	Versions           map[string]string `json:"versions"`
	PreviousPassportID string            `json:"previous_passport_id,omitempty"`
	DocumentID         string            `json:"document_id" doc:"Протокол допуска с закрытым маршрутом."`
	AnalyzerKind       string            `json:"analyzer_kind,omitempty" enum:"visionqc,operatorvision" doc:"Визуальный контроль или контроль действий оператора; по умолчанию visionqc."`
	Title              string            `json:"title,omitempty" maxLength:"256" doc:"Название анализатора."`
}

// ReinstatePassport — вернуть анализатор после отката (analyzer.passport.reinstated,
// FR-101): разрешающее действие начальника ОТК.
type ReinstatePassport struct {
	platform.CommandHeader
	SuspensionEventID string         `json:"suspension_event_id" format:"uuid"`
	Reason            AnalyzerReason `json:"reason"`
}

// RetirePassport — вывести паспорт из действия (analyzer.passport.retired).
type RetirePassport struct {
	platform.CommandHeader
	Reason AnalyzerReason `json:"reason"`
}
