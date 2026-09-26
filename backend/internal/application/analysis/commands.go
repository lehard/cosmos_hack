package analysis

import "ant/internal/application/platform"

// RecordHypothesis — записать гипотезу причины человеком (incident.hypothesis.recorded, FR-59).
type RecordHypothesis struct {
	platform.CommandHeader
	IncidentID         string   `json:"incident_id" doc:"Инцидент."`
	Branch             string   `json:"branch" enum:"why_made,why_missed" doc:"Почему возник / почему пропустили."`
	Category           string   `json:"category" enum:"incoming,equipment,performer,handling,assembly,documentation,not_established"`
	Statement          string   `json:"statement" minLength:"1" maxLength:"2000"`
	SupportingEventIDs []string `json:"supporting_event_ids,omitempty"`
}

// ConcludeCause — подтвердить гипотезу как причину или «причина не установлена»
// (incident.cause.concluded, FR-59; необратимое, критическое — группа cause).
type ConcludeCause struct {
	platform.CommandHeader
	HypothesisID string   `json:"hypothesis_id,omitempty" doc:"Подтверждаемая гипотеза."`
	NCIDs        []string `json:"nc_ids" minItems:"1"`
	Conclusion   string   `json:"conclusion" enum:"confirmed,not_established"`
	Category     string   `json:"category,omitempty" enum:"incoming,equipment,performer,handling,assembly,documentation,not_established"`
	Verification string   `json:"verification" minLength:"1" maxLength:"2000" doc:"Чем проверено."`
	Reason       Reason   `json:"reason"`
}

// RejectHypothesis — отклонить гипотезу с основанием.
type RejectHypothesis struct {
	platform.CommandHeader
	HypothesisID string `json:"hypothesis_id"`
	Reason       Reason `json:"reason"`
}

// RequestMeasurement — запросить измерение для проверки гипотезы (задачу ставит notifications).
type RequestMeasurement struct {
	platform.CommandHeader
	HypothesisID string `json:"hypothesis_id"`
	What         string `json:"what" minLength:"1" maxLength:"1000" doc:"Что измерить."`
	AssigneeID   string `json:"assignee_id,omitempty" doc:"Кому (псевдоним); пусто — по правилу."`
}

// ChangeScope — сузить или расширить область риска (incident.scope.narrowed /
// expanded, FR-61): каждая правка — новая версия с основанием.
type ChangeScope struct {
	platform.CommandHeader
	ItemIDs            []string `json:"item_ids" minItems:"1"`
	EvidenceEventIDs   []string `json:"evidence_event_ids,omitempty" doc:"Для сужения — обязательны (AD-27: исключение — разрешающее действие)."`
	ReleaseContainment bool     `json:"release_containment,omitempty" doc:"Сужение: снять блок исключённых (отдельное разрешающее решение nonconformity)."`
	Reason             Reason   `json:"reason"`
}

// AssessItem — оценка изделия в инциденте: подтверждено / исключено (incident.item.assessed, FR-62).
type AssessItem struct {
	platform.CommandHeader
	ItemID           string   `json:"item_id"`
	Assessment       string   `json:"assessment" enum:"confirmed,excluded"`
	EvidenceEventIDs []string `json:"evidence_event_ids" minItems:"1"`
}

// ScopeAnalysis — решение о глубине разбора (incident.analysis.scoped, П-02).
type ScopeAnalysis struct {
	platform.CommandHeader
	FullAnalysis bool   `json:"full_analysis"`
	Reason       Reason `json:"reason"`
}

// CloseIncident — закрыть инцидент (incident.incident.closed).
type CloseIncident struct {
	platform.CommandHeader
	Summary string `json:"summary,omitempty" maxLength:"4000"`
}

// AssignAction — назначить корректирующее действие (incident.action.assigned, FR-64).
type AssignAction struct {
	platform.CommandHeader
	ActionType        string         `json:"action_type" enum:"correction,corrective_action,preventive_action"`
	Direction         string         `json:"direction" enum:"prevent_occurrence,improve_detection"`
	OwnerID           string         `json:"owner_id"`
	DueAt             string         `json:"due_at,omitempty" format:"date-time"`
	EffectivenessPlan map[string]any `json:"effectiveness_plan"`
	// Title — что делается словами (эпик 42, FR-138: организационная память).
	Title string `json:"title,omitempty" maxLength:"256" doc:"Что делается словами — основа организационной памяти."`
	// SuggestionID — предложение, из которого родилась мера (эпик 42, FR-63).
	SuggestionID string `json:"suggestion_id,omitempty" maxLength:"128" doc:"Предложение, из которого родилась мера."`
}

// ImplementAction — отметить действие выполненным (incident.action.implemented).
type ImplementAction struct {
	platform.CommandHeader
	Note string `json:"note,omitempty" maxLength:"2000"`
}

// EvaluateAction — оценить эффективность действия (incident.action.evaluated).
type EvaluateAction struct {
	platform.CommandHeader
	Result   string `json:"result" enum:"effective,failed"`
	Evidence string `json:"evidence,omitempty" maxLength:"4000"`
}
