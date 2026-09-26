package vision

import "time"

// Формы ответов модуля vision: анализаторы VisionQC и OperatorVision, паспорта
// допуска, отчёты проверки (FR-97…FR-101, FR-126; AD-29).

// AnalyzerReason — основание решения по паспорту: код и текст.
type AnalyzerReason struct {
	Code *string `json:"code,omitempty"`
	Text string  `json:"text" minLength:"1" maxLength:"2000"`
}

// AnalyzerSuspension — приостановка паспорта (analyzer.passport.suspended):
// действует на будущее и сильнее закреплённой версии (AD-17, AD-29).
type AnalyzerSuspension struct {
	EventID            string    `json:"event_id"`
	Trigger            string    `json:"trigger" enum:"drift,reference_set_failed,disagreement_growth,escape_detected"`
	Fallback           string    `json:"fallback" enum:"previous_passport,manual_control"`
	FallbackPassportID *string   `json:"fallback_passport_id,omitempty"`
	At                 time.Time `json:"at"`
}

// AnalyzerSummary — анализатор в списке: версии и уровень доверия паспорта.
type AnalyzerSummary struct {
	AnalyzerID string            `json:"analyzer_id"`
	Title      string            `json:"title"`
	Kind       string            `json:"kind" enum:"visionqc,operatorvision" doc:"Визуальный контроль или контроль действий оператора."`
	PassportID *string           `json:"passport_id,omitempty" doc:"Действующий паспорт допуска."`
	Stage      *string           `json:"stage,omitempty" enum:"shadow,pilot,active" doc:"Стадия допуска: тень, пилот, работа."`
	TrustLevel *int              `json:"trust_level,omitempty" minimum:"0" maximum:"4" doc:"Уровень доверия паспорта → допустимые автоматические действия (AD-29)."`
	Status     string            `json:"status" enum:"active,suspended,retired,not_admitted"`
	Versions   map[string]string `json:"versions,omitempty" doc:"Вектор версий (AD-29)."`
}

// AnalyzerList — анализаторы.
type AnalyzerList struct {
	Items []AnalyzerSummary `json:"items"`
}

// AnalyzerPassport — паспорт допуска карты контроля (нормативный слой, документ с маршрутом, AD-29).
type AnalyzerPassport struct {
	PassportID         string              `json:"passport_id"`
	AnalyzerID         string              `json:"analyzer_id"`
	Stage              string              `json:"stage" enum:"shadow,pilot,active"`
	TrustLevel         int                 `json:"trust_level" minimum:"0" maximum:"4"`
	RecipeRef          string              `json:"recipe_ref" doc:"Карта контроля."`
	Versions           map[string]string   `json:"versions" doc:"Вектор версий допуска."`
	Status             string              `json:"status" enum:"active,suspended,retired"`
	AdmittedAt         time.Time           `json:"admitted_at"`
	DocumentID         string              `json:"document_id" doc:"Протокол допуска (закрытый маршрут подписей)."`
	PreviousPassportID *string             `json:"previous_passport_id,omitempty"`
	Suspension         *AnalyzerSuspension `json:"suspension,omitempty"`
	AllowedAutoActions []string            `json:"allowed_auto_actions" doc:"Допустимые автоматические действия уровня доверия (contracts/analyzer-trust-levels.yaml)."`
	BasisSeq           int64               `json:"basis_seq" doc:"seq, на котором построен ответ (AD-39)."`
}

// AnalyzerCheck — отчёт проверки анализатора (analyzer.check.recorded).
type AnalyzerCheck struct {
	EventID            string    `json:"event_id"`
	PassportID         string    `json:"passport_id"`
	CheckKind          string    `json:"check_kind" enum:"exam,reference_set,shadow_comparison,drift_monitor"`
	EscapeRateBP       *int      `json:"escape_rate_bp,omitempty" minimum:"0" maximum:"10000"`
	FalseAlarmRateBP   *int      `json:"false_alarm_rate_bp,omitempty" minimum:"0" maximum:"10000"`
	DisagreementRateBP *int      `json:"disagreement_rate_bp,omitempty" minimum:"0" maximum:"10000"`
	Passed             bool      `json:"passed"`
	OccurredAt         time.Time `json:"occurred_at"`
}

// AnalyzerCheckList — отчёты проверки.
type AnalyzerCheckList struct {
	Items      []AnalyzerCheck `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}
