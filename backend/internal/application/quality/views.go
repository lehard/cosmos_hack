package quality

import "time"

// Формы ответов модуля quality: сигналы, результаты контроля, полнота, дефекты,
// пропуски брака, карта реакций (FR-14, FR-35…FR-38, FR-48, FR-50; эпики 11, 20).
// Смысл — только контрактный (NFR-UI-4): «оценка невозможна» ≠ «годно»,
// уверенность анализатора ≠ вероятность брака, сигнал ≠ брак.

// QualityAnalyzerStage — ступень ансамбля VisionQC: где дефект / какой тип (FR-38).
type QualityAnalyzerStage struct {
	Stage        string `json:"stage" doc:"Ступень ансамбля."`
	Version      string `json:"version" doc:"Версия анализатора ступени."`
	ConfidenceBP *int   `json:"confidence_bp,omitempty" minimum:"0" maximum:"10000" doc:"Уверенность ступени, б. п."`
	OutputNote   string `json:"output_note,omitempty"`
}

// QualitySignal — сигнал о признаке дефекта (quality.signal.raised) с исходным
// наблюдением: ступени, уверенность и качество наблюдения, вектор версий (AD-29).
type QualitySignal struct {
	SignalID             string                 `json:"signal_id"`
	ItemID               string                 `json:"item_id"`
	ItemLabel            string                 `json:"item_label"`
	StepKey              string                 `json:"step_key"`
	BasisKind            string                 `json:"basis_kind" enum:"inspection_result,equipment_deviation,check_skipped,damage_on_receipt,leak,special_process_violation,operator_report"`
	DefectID             *string                `json:"defect_id,omitempty"`
	ZoneID               *string                `json:"zone_id,omitempty"`
	DefectTypeCode       *string                `json:"defect_type_code,omitempty"`
	DefectTypeKnown      bool                   `json:"defect_type_known"`
	Severity             string                 `json:"severity" enum:"critical,major,minor,unknown"`
	AnalyzerConfidenceBP *int                   `json:"analyzer_confidence_bp,omitempty" minimum:"0" maximum:"10000" doc:"Уверенность анализатора, б. п. — не вероятность брака."`
	ObservationQualityBP *int                   `json:"observation_quality_bp,omitempty" minimum:"0" maximum:"10000" doc:"Качество наблюдения, б. п."`
	ReactionOutcome      string                 `json:"reaction_outcome" enum:"pass_to_next,manual_review,isolate,question_to_technologist" doc:"Исход по карте реакций."`
	ReactionMapRef       string                 `json:"reaction_map_ref"`
	TrustLevel           *int                   `json:"trust_level,omitempty" minimum:"0" maximum:"4" doc:"Уровень доверия паспорта анализатора (AD-29)."`
	State                string                 `json:"state" enum:"open,confirmed,rejected" doc:"Рассмотрен ли сигнал человеком."`
	RaisedAt             time.Time              `json:"raised_at"`
	ObservationEventID   *string                `json:"observation_event_id,omitempty" doc:"Исходный результат контроля."`
	Stages               []QualityAnalyzerStage `json:"stages"`
	Versions             map[string]string      `json:"versions,omitempty" doc:"Вектор версий наблюдения (AD-29): ревизия изделия, карта контроля, камера, калибровка, анализатор, профиль порогов, контракт, приложение."`
	EvidenceRefs         []string               `json:"evidence_refs" doc:"Адреса материалов: кадр, иллюстрация (если есть)."`
	BasisSeq             int64                  `json:"basis_seq" doc:"seq, на котором построен ответ (AD-39)."`
}

// QualitySignalList — сигналы.
type QualitySignalList struct {
	Items      []QualitySignal `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

// InspectionDefect — дефект в результате контроля (FR-37): без вида дефекта в ключе.
type InspectionDefect struct {
	DefectTypeCode *string `json:"defect_type_code,omitempty"`
	Description    string  `json:"description,omitempty"`
	ComponentID    *string `json:"component_id,omitempty"`
	ZoneID         *string `json:"zone_id,omitempty"`
	Location       string  `json:"location,omitempty"`
	Severity       string  `json:"severity" enum:"critical,major,minor,unknown"`
}

// InspectionResult — результат контроля любого метода (inspection.result.recorded,
// FR-36): три исхода — признак дефекта / признака нет / оценка невозможна.
type InspectionResult struct {
	EventID              string             `json:"event_id"`
	Seq                  *int64             `json:"seq,omitempty"`
	ItemID               string             `json:"item_id"`
	StepKey              string             `json:"step_key"`
	OperationRunID       *string            `json:"operation_run_id,omitempty"`
	Method               string             `json:"method" enum:"camera,cmm,radiography,ultrasonic,penetrant,leak_test,torque,visual_human,supplier_documents,laboratory,other"`
	Outcome              string             `json:"outcome" enum:"defect_indicated,no_defect_indicated,unable_to_assess" doc:"«Оценка невозможна» ≠ «годно»."`
	ProcessingState      string             `json:"processing_state" enum:"completed,aborted,failed"`
	Defects              []InspectionDefect `json:"defects"`
	AnalyzerConfidenceBP *int               `json:"analyzer_confidence_bp,omitempty" minimum:"0" maximum:"10000"`
	ObservationQualityBP *int               `json:"observation_quality_bp,omitempty" minimum:"0" maximum:"10000"`
	Limitations          []string           `json:"limitations,omitempty"`
	SourceKind           string             `json:"source_kind" enum:"manual_entry,machine,sensor,camera,external_system,import" doc:"Пометка источника (FR-140)."`
	Reliability          string             `json:"reliability" enum:"high,medium,low,unknown"`
	OccurredAt           time.Time          `json:"occurred_at"`
	EvidenceRefs         []string           `json:"evidence_refs"`
}

// InspectionResultList — результаты контроля изделия.
type InspectionResultList struct {
	Items      []InspectionResult `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// CoveragePoint — точка контроля плана: получен ли результат (FR-35).
type CoveragePoint struct {
	StepKey         string  `json:"step_key"`
	InspectionPoint string  `json:"inspection_point"`
	Method          string  `json:"method" enum:"camera,cmm,radiography,ultrasonic,penetrant,leak_test,torque,visual_human,supplier_documents,laboratory,other"`
	Required        bool    `json:"required"`
	Status          string  `json:"status" enum:"received,pending,missing" doc:"Результат получен / ещё ждём / нет (quality.inspection.missing)."`
	MissingReason   *string `json:"missing_reason,omitempty" enum:"result_not_received,check_skipped,point_manual_mode,not_covered_by_method,unknown"`
	EventID         *string `json:"event_id,omitempty" doc:"Результат или запись о пропуске."`
}

// InspectionCoverage — полнота контроля изделия (FR-35, FR-14).
type InspectionCoverage struct {
	ItemID   string          `json:"item_id"`
	Complete bool            `json:"complete" doc:"Все обязательные результаты получены."`
	Points   []CoveragePoint `json:"points"`
	BasisSeq int64           `json:"basis_seq"`
}

// QualityDefect — физический дефект (FR-37): ключ — изделие, зона и место в
// пределах выполнения операции, без вида дефекта; повторные наблюдения не множат.
type QualityDefect struct {
	DefectID                string    `json:"defect_id"`
	ItemID                  string    `json:"item_id"`
	ZoneID                  string    `json:"zone_id"`
	Location                string    `json:"location,omitempty"`
	DefectTypeCode          *string   `json:"defect_type_code,omitempty"`
	FirstObservationEventID string    `json:"first_observation_event_id"`
	Observations            int       `json:"observations" minimum:"1" doc:"Наблюдений этого дефекта."`
	IdentifiedAt            time.Time `json:"identified_at"`
}

// QualityDefectList — дефекты; дефекты и изделия с дефектами считаются раздельно (FR-37).
type QualityDefectList struct {
	Items           []QualityDefect `json:"items"`
	DefectCount     int             `json:"defect_count" minimum:"0"`
	ItemsWithDefect int             `json:"items_with_defect" minimum:"0"`
	NextCursor      string          `json:"next_cursor,omitempty"`
}

// QualityEscape — пропуск брака (quality.escape.recorded): возврат в контур адаптации.
type QualityEscape struct {
	EventID                   string    `json:"event_id"`
	DefectID                  string    `json:"defect_id"`
	ItemID                    string    `json:"item_id"`
	MissedObservationEventIDs []string  `json:"missed_observation_event_ids"`
	MethodCoversDefect        bool      `json:"method_covers_defect"`
	AnalyzerVersion           *string   `json:"analyzer_version,omitempty"`
	RecordedAt                time.Time `json:"recorded_at"`
}

// QualityEscapeList — пропуски брака.
type QualityEscapeList struct {
	Items      []QualityEscape `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

// ReactionRule — правило карты реакций с режимом автоматизации (FR-48, FR-50, AD-3).
type ReactionRule struct {
	RuleID         string     `json:"rule_id"`
	Title          string     `json:"title"`
	Trigger        string     `json:"trigger" doc:"Условие срабатывания по-русски."`
	DefectTypeCode *string    `json:"defect_type_code,omitempty"`
	Severity       *string    `json:"severity,omitempty" enum:"critical,major,minor,unknown"`
	Outcome        string     `json:"outcome" enum:"pass_to_next,manual_review,isolate,question_to_technologist"`
	AutomationMode int        `json:"automation_mode" minimum:"1" maximum:"5" doc:"Режим автоматизации 1–5 (FR-50)."`
	ActionClass    string     `json:"action_class" enum:"record,protective,permissive,irreversible" doc:"Класс действия (AD-27)."`
	Owner          string     `json:"owner" doc:"Владелец правила."`
	ApprovedBy     *string    `json:"approved_by,omitempty"`
	ValidUntil     *time.Time `json:"valid_until,omitempty"`
	ThresholdBP    *int       `json:"threshold_bp,omitempty" minimum:"0" maximum:"10000" doc:"Порог уверенности, б. п."`
}

// ReactionMap — действующая карта реакций (normative/reactions, FR-48).
type ReactionMap struct {
	Ref     string         `json:"ref" doc:"Ссылка на версию карты реакций."`
	Version string         `json:"version"`
	Rules   []ReactionRule `json:"rules"`
}
