package nonconformity

// Данные записей семейства decision (contracts/events/decision/*.v1.json) в
// представлении домена: те же поля и имена JSON, время — строкой RFC 3339
// («Соглашения/Время»), чтобы канонический JSON записи не зависел от
// представления времени в сгенерированных типах. Эти структуры — и то, что
// application пишет в журнал, и то, что свёртка читает (Decode).

// Reason — основание решения: код и текст (defs.reason).
type Reason struct {
	Code string `json:"code,omitempty"`
	Text string `json:"text"`
}

// DraftedData — decision.nonconformity.drafted (FR-51): черновик карточки;
// те же поля — полезная нагрузка намерения «черновик несоответствия».
type DraftedData struct {
	NCID               string   `json:"nc_id"`
	SignalIDs          []string `json:"signal_ids"`
	DocumentID         string   `json:"document_id,omitempty"`
	BasisKind          string   `json:"basis_kind,omitempty"`
	Severity           string   `json:"severity,omitempty"`
	DefectTypeCode     string   `json:"defect_type_code,omitempty"`
	ZoneID             string   `json:"zone_id,omitempty"`
	StepKey            string   `json:"step_key,omitempty"`
	OperationRunID     string   `json:"operation_run_id,omitempty"`
	RequirementRef     string   `json:"requirement_ref,omitempty"`
	ReactionOutcome    string   `json:"reaction_outcome,omitempty"`
	ReactionMapRef     string   `json:"reaction_map_ref,omitempty"`
	ClosingPoint       string   `json:"closing_point,omitempty"`
	PresentationNo     int      `json:"presentation_no,omitempty"`
	MissingInformation []string `json:"missing_information,omitempty"`
}

// RegisteredData — decision.nonconformity.registered (FR-151).
type RegisteredData struct {
	NCID                   string `json:"nc_id"`
	ViolationWindowEventID string `json:"violation_window_event_id"`
	OperationRunID         string `json:"operation_run_id"`
	StepKey                string `json:"step_key,omitempty"`
}

// ConfirmedData — decision.nonconformity.confirmed (FR-52).
type ConfirmedData struct {
	NCID                 string   `json:"nc_id"`
	SignalIDs            []string `json:"signal_ids"`
	RequirementRef       string   `json:"requirement_ref,omitempty"`
	Severity             string   `json:"severity"`
	DefectTypeCode       string   `json:"defect_type_code,omitempty"`
	FullAnalysisRequired bool     `json:"full_analysis_required,omitempty"`
	Reason               Reason   `json:"reason"`
}

// SignalRejectedData — decision.signal.rejected (FR-52): причина обязательна.
type SignalRejectedData struct {
	SignalIDs          []string `json:"signal_ids"`
	Reason             Reason   `json:"reason"`
	LabelForAdaptation bool     `json:"label_for_adaptation,omitempty"`
}

// RecheckData — decision.recheck.requested (FR-52).
type RecheckData struct {
	Method          string   `json:"method"`
	ZoneIDs         []string `json:"zone_ids,omitempty"`
	DueAt           string   `json:"due_at,omitempty"`
	SuggestedByRule string   `json:"suggested_by_rule,omitempty"`
	Reason          Reason   `json:"reason"`
}

// IsolatedData — decision.item.isolated (FR-55).
type IsolatedData struct {
	IsolatorLocationID string `json:"isolator_location_id,omitempty"`
	DecisionDueAt      string `json:"decision_due_at,omitempty"`
	Reason             Reason `json:"reason"`
}

// ContainmentAppliedData — decision.containment.applied (FR-49, правило).
type ContainmentAppliedData struct {
	Level      string   `json:"level"`
	IncidentID string   `json:"incident_id,omitempty"`
	LotID      string   `json:"lot_id,omitempty"`
	Basis      []string `json:"basis"`
}

// ContainmentSetData — decision.containment.set (FR-49, человек).
type ContainmentSetData struct {
	Level  string `json:"level"`
	Reason Reason `json:"reason"`
}

// ContainmentReleasedData — decision.containment.released (FR-49, AD-27).
type ContainmentReleasedData struct {
	ReleasedEventIDs []string `json:"released_event_ids"`
	Reason           Reason   `json:"reason"`
}

// DispositionSetData — decision.disposition.set (FR-53).
type DispositionSetData struct {
	NCID            string `json:"nc_id"`
	Disposition     string `json:"disposition"`
	ScrapKind       string `json:"scrap_kind,omitempty"`
	ConcessionID    string `json:"concession_id,omitempty"`
	DocumentID      string `json:"document_id,omitempty"`
	ClaimBasis      string `json:"claim_basis,omitempty"`
	ApprovalsStatus string `json:"approvals_status,omitempty"`
	Reason          Reason `json:"reason"`
}

// DispositionVerifiedData — decision.disposition.verified.
type DispositionVerifiedData struct {
	NCID            string   `json:"nc_id"`
	RecheckEventIDs []string `json:"recheck_event_ids"`
}

// ClosedData — decision.nonconformity.closed.
type ClosedData struct {
	NCID    string `json:"nc_id"`
	Summary string `json:"summary,omitempty"`
}

// PresentationResolvedData — decision.presentation.resolved (FR-19, FR-56).
type PresentationResolvedData struct {
	StepKey        string   `json:"step_key"`
	ClosingPoint   string   `json:"closing_point"`
	Resolution     string   `json:"resolution"`
	PresentationNo int      `json:"presentation_no"`
	ConcessionID   string   `json:"concession_id,omitempty"`
	MethodEventIDs []string `json:"method_event_ids"`
	DocumentID     string   `json:"document_id,omitempty"`
	Reason         *Reason  `json:"reason,omitempty"`
}

// LotResolvedData — decision.lot.resolved (ЗТ-1).
type LotResolvedData struct {
	LotID            string   `json:"lot_id"`
	Resolution       string   `json:"resolution"`
	AcceptedQuantity *int     `json:"accepted_quantity,omitempty"`
	MethodEventIDs   []string `json:"method_event_ids"`
	Reason           *Reason  `json:"reason,omitempty"`
}

// ConcessionGrantedData — decision.concession.granted (FR-54).
type ConcessionGrantedData struct {
	ConcessionID    string   `json:"concession_id"`
	Number          string   `json:"number,omitempty"`
	Title           string   `json:"title,omitempty"`
	Kind            string   `json:"kind"`
	RequirementRef  string   `json:"requirement_ref,omitempty"`
	ScopeItemIDs    []string `json:"scope_item_ids,omitempty"`
	ScopeRangeFrom  string   `json:"scope_range_from,omitempty"`
	ScopeRangeTo    string   `json:"scope_range_to,omitempty"`
	Limit           int      `json:"limit"`
	ValidUntil      string   `json:"valid_until,omitempty"`
	DocumentID      string   `json:"document_id,omitempty"`
	ApprovalsStatus string   `json:"approvals_status,omitempty"`
	Reason          Reason   `json:"reason"`
}

// ConcessionRevokedData — decision.concession.revoked (FR-54).
type ConcessionRevokedData struct {
	ConcessionID string `json:"concession_id"`
	Reason       Reason `json:"reason"`
}

// ReworkWaivedData — decision.rework_limit.waived (FR-18).
type ReworkWaivedData struct {
	ZoneID       string `json:"zone_id"`
	Used         int    `json:"used"`
	Limit        int    `json:"limit"`
	ExtraAllowed int    `json:"extra_allowed"`
	Reason       Reason `json:"reason"`
}

// ProcessHoldSetData — decision.process_hold.set (FR-49).
type ProcessHoldSetData struct {
	HoldID           string `json:"hold_id"`
	Level            string `json:"level"`
	EquipmentID      string `json:"equipment_id,omitempty"`
	ToolID           string `json:"tool_id,omitempty"`
	ProgramRef       string `json:"program_ref,omitempty"`
	StepKey          string `json:"step_key,omitempty"`
	IncidentID       string `json:"incident_id,omitempty"`
	ReleaseCondition string `json:"release_condition,omitempty"`
	Reason           Reason `json:"reason"`
}

// ProcessHoldReleasedData — decision.process_hold.released (FR-49).
type ProcessHoldReleasedData struct {
	HoldID          string `json:"hold_id"`
	CleanPointItems int    `json:"clean_point_items"`
	Reason          Reason `json:"reason"`
}

// CleanPointData — decision.clean_point.assigned (FR-49, точка чистоты).
type CleanPointData struct {
	HoldID         string `json:"hold_id"`
	OperationRunID string `json:"operation_run_id"`
	EquipmentID    string `json:"equipment_id,omitempty"`
	Ordinal        int    `json:"ordinal"`
	Of             int    `json:"of"`
}
