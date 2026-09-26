package nonconformity

import "ant/internal/application/platform"

// ConfirmNonconformity — подтвердить несоответствие (decision.nonconformity.confirmed, FR-52).
type ConfirmNonconformity struct {
	platform.CommandHeader
	SignalIDs            []string `json:"signal_ids" minItems:"1"`
	Severity             string   `json:"severity" enum:"critical,major,minor,unknown"`
	RequirementRef       string   `json:"requirement_ref,omitempty" maxLength:"256"`
	DefectTypeCode       string   `json:"defect_type_code,omitempty" maxLength:"64"`
	FullAnalysisRequired bool     `json:"full_analysis_required,omitempty"`
	Reason               NCReason `json:"reason"`
}

// RejectSignal — отклонить сигнал; причина обязательна (decision.signal.rejected, FR-52).
type RejectSignal struct {
	platform.CommandHeader
	SignalIDs          []string `json:"signal_ids" minItems:"1"`
	Reason             NCReason `json:"reason"`
	LabelForAdaptation bool     `json:"label_for_adaptation,omitempty" doc:"Пометить для контура адаптации анализатора."`
}

// RequestRecheck — назначить дополнительную проверку (decision.recheck.requested).
type RequestRecheck struct {
	platform.CommandHeader
	Method  string   `json:"method" enum:"camera,cmm,radiography,ultrasonic,penetrant,leak_test,torque,visual_human,supplier_documents,laboratory,other"`
	ZoneIDs []string `json:"zone_ids,omitempty"`
	DueAt   string   `json:"due_at,omitempty" format:"date-time"`
	Reason  NCReason `json:"reason"`
}

// IsolateItem — изолировать изделие со сроком решения (decision.item.isolated, FR-55).
type IsolateItem struct {
	platform.CommandHeader
	IsolatorLocationID string   `json:"isolator_location_id,omitempty" maxLength:"128"`
	DecisionDueAt      string   `json:"decision_due_at,omitempty" format:"date-time"`
	Reason             NCReason `json:"reason"`
}

// ResolvePresentation — решение на точке предъявления: «Принять — передать на
// ‹следующий шаг›» и др. (decision.presentation.resolved, FR-19, FR-56).
type ResolvePresentation struct {
	platform.CommandHeader
	StepKey        string    `json:"step_key"`
	ClosingPoint   string    `json:"closing_point" doc:"Закрывающая точка (ЗТ)."`
	Resolution     string    `json:"resolution" enum:"accept,accept_with_concession,reject,insufficient_data"`
	PresentationNo int       `json:"presentation_no" minimum:"1"`
	ConcessionID   string    `json:"concession_id,omitempty" doc:"Для accept_with_concession."`
	MethodEventIDs []string  `json:"method_event_ids" minItems:"1" doc:"Результаты методов контроля, на которых основано решение."`
	Reason         *NCReason `json:"reason,omitempty"`
}

// ReviewPresentation — пересмотр решения на точке, принятого до новых данных
// (decision.presentation.reviewed, FR-32, FR-146, Д-81): «оставить в силе»
// или «отозвать приёмку»; основание обязательно.
type ReviewPresentation struct {
	platform.CommandHeader
	ReviewedEventID string   `json:"reviewed_event_id" doc:"Пересматриваемое решение — review.decision.event_id из nonconformity.presentation.read."`
	Outcome         string   `json:"outcome" enum:"upheld,revoked" doc:"upheld — оставить в силе; revoked — отозвать приёмку."`
	NewFactIDs      []string `json:"new_fact_ids,omitempty" doc:"Рассмотренные новые факты (review.new_facts); пусто — все новые факты пересмотра."`
	Reason          NCReason `json:"reason" doc:"Основание пересмотра."`
}

// ResolveLot — решение по партии входного контроля (decision.lot.resolved).
type ResolveLot struct {
	platform.CommandHeader
	Resolution       string    `json:"resolution" enum:"accept,accept_partially,reject,insufficient_data"`
	AcceptedQuantity *int      `json:"accepted_quantity,omitempty" minimum:"0"`
	MethodEventIDs   []string  `json:"method_event_ids" minItems:"1"`
	Reason           *NCReason `json:"reason,omitempty"`
}

// SetDisposition — решение по несоответствию: переделка, ремонт, как есть,
// списать, вернуть поставщику (decision.disposition.set, FR-53); ремонт и «как
// есть» — только с действующим разрешением на отклонение (FR-54).
type SetDisposition struct {
	platform.CommandHeader
	Disposition  string   `json:"disposition" enum:"rework,repair,use_as_is,scrap,return_to_supplier"`
	ScrapKind    string   `json:"scrap_kind,omitempty" enum:"writeoff,reprocess"`
	ConcessionID string   `json:"concession_id,omitempty" doc:"Обязательно для repair и use_as_is (nonconformity.concession_required)."`
	ClaimBasis   string   `json:"claim_basis,omitempty" maxLength:"2000" doc:"Основание претензии поставщику."`
	Reason       NCReason `json:"reason"`
}

// VerifyDisposition — подтвердить выполнение решения повторной проверкой (decision.disposition.verified).
type VerifyDisposition struct {
	platform.CommandHeader
	RecheckEventIDs []string `json:"recheck_event_ids" minItems:"1"`
}

// SetContainment — установить уровень сдерживания (decision.containment.set, FR-49).
type SetContainment struct {
	platform.CommandHeader
	Level  string   `json:"level" enum:"none,observe,additional_check,item_hold,lot_hold"`
	Reason NCReason `json:"reason"`
}

// ReleaseContainment — снять сдерживание (decision.containment.released):
// снятие блока ≠ годность; только уполномоченный (AD-27).
type ReleaseContainment struct {
	platform.CommandHeader
	ReleasedEventIDs []string `json:"released_event_ids" minItems:"1" doc:"Какие записи сдерживания снимаются."`
	Reason           NCReason `json:"reason"`
}

// RevokeConcession — отозвать разрешение на отклонение (decision.concession.revoked).
type RevokeConcession struct {
	platform.CommandHeader
	Reason NCReason `json:"reason"`
}

// WaiveReworkLimit — разрешение сверх лимита доработок (decision.rework_limit.waived, FR-18).
type WaiveReworkLimit struct {
	platform.CommandHeader
	ZoneID       string   `json:"zone_id"`
	Used         int      `json:"used" minimum:"0"`
	Limit        int      `json:"limit" minimum:"0"`
	ExtraAllowed int      `json:"extra_allowed" minimum:"1"`
	Reason       NCReason `json:"reason"`
}

// SetProcessHold — стоп точки процесса или критическая остановка (decision.process_hold.set, FR-49).
type SetProcessHold struct {
	platform.CommandHeader
	HoldID           string   `json:"hold_id,omitempty" doc:"Пусто — присвоит сервер."`
	Level            string   `json:"level" enum:"process_point_stop,critical_stop"`
	EquipmentID      string   `json:"equipment_id,omitempty"`
	ToolID           string   `json:"tool_id,omitempty"`
	ProgramRef       string   `json:"program_ref,omitempty"`
	StepKey          string   `json:"step_key,omitempty"`
	IncidentID       string   `json:"incident_id,omitempty"`
	ReleaseCondition string   `json:"release_condition,omitempty" maxLength:"2000"`
	Reason           NCReason `json:"reason"`
}

// ReleaseProcessHold — снять остановку точки процесса (decision.process_hold.released).
type ReleaseProcessHold struct {
	platform.CommandHeader
	CleanPointItems int      `json:"clean_point_items" minimum:"0" doc:"Изделий после точки чистоты."`
	Reason          NCReason `json:"reason"`
}

// CloseNonconformity — закрыть несоответствие (decision.nonconformity.closed).
type CloseNonconformity struct {
	platform.CommandHeader
	Summary string `json:"summary,omitempty" maxLength:"4000"`
}

// GrantConcession — выдать разрешение на отклонение (decision.concession.granted,
// FR-54, Д-24): номер, пункт КД/ТУ, область действия (изделия или диапазон
// номеров), лимит количества, срок; лимит открывается атомарно с записью.
type GrantConcession struct {
	platform.CommandHeader
	ConcessionID   string   `json:"concession_id,omitempty" maxLength:"128" doc:"Пусто — присвоит сервер."`
	Number         string   `json:"number,omitempty" maxLength:"64"`
	Title          string   `json:"title" minLength:"1" maxLength:"256"`
	Kind           string   `json:"kind" enum:"repair,use_as_is"`
	RequirementRef string   `json:"requirement_ref,omitempty" maxLength:"256" doc:"Пункт КД/ТУ."`
	ScopeItemIDs   []string `json:"scope_item_ids,omitempty" doc:"Область действия — перечень изделий."`
	ScopeRangeFrom string   `json:"scope_range_from,omitempty" maxLength:"128"`
	ScopeRangeTo   string   `json:"scope_range_to,omitempty" maxLength:"128"`
	Limit          int      `json:"limit" minimum:"1" doc:"Лимит количества изделий."`
	ValidUntil     string   `json:"valid_until,omitempty" format:"date-time"`
	DocumentID     string   `json:"document_id,omitempty" maxLength:"128" doc:"Документ разрешения с маршрутом подписей."`
	Reason         NCReason `json:"reason"`
}
