package analysis

import "time"

// Формы ответов разбора (эпик 12: frontend/src/entities/incident/model/types.ts).
// Поля — как в контракте семейства incident и проекции analysis.circumstances
// (AD-29); смысл — только контрактный (NFR-UI-4): уверенность ≠ вероятность
// вины, «под подозрением» ≠ брак, гипотеза ≠ причина.

// JournalRecordRef — ссылка на запись журнала с данными для подписи.
type JournalRecordRef struct {
	EventID    string            `json:"event_id" doc:"event_id записи."`
	EventType  string            `json:"event_type" doc:"Тип записи каталога, например equipment.deviation.detected."`
	Variant    *string           `json:"variant,omitempty" doc:"Уточнение внутри типа: outcome контроля, deviation_kind отклонения, cycle_started / cycle_finished, condition."`
	OccurredAt time.Time         `json:"occurred_at" doc:"Время возникновения (AD-37)."`
	Params     map[string]string `json:"params,omitempty" doc:"Параметры подписи: метод, параметр, значение, уставка, шаг…"`
}

// CircumstanceRecord — строка проекции analysis.circumstances на дорожке разбора (FR-153).
type CircumstanceRecord struct {
	JournalRecordRef
	Lane            string     `json:"lane" enum:"item,person,equipment" doc:"Дорожка: изделие, человек, оборудование."`
	EndedAt         *time.Time `json:"ended_at,omitempty" doc:"Конец интервала (цикл, отклонение)."`
	JournalSeq      *int64     `json:"journal_seq,omitempty" doc:"Позиция записи в журнале — для перехода к записи."`
	EvidenceRefs    []string   `json:"evidence_refs,omitempty" doc:"Адреса материалов: кадры, протоколы."`
	RelatedEventIDs []string   `json:"related_event_ids,omitempty" doc:"Связанные записи — подсвечиваются вместе с выбранной."`
	SourceKind      *string    `json:"source_kind,omitempty" doc:"Вид источника факта (FR-140)."`
}

// CausalWindow — окно возможного возникновения (FR-58).
type CausalWindow struct {
	Start             time.Time `json:"start" doc:"causal_window_start — последнее подтверждённо нормальное состояние."`
	End               time.Time `json:"end" doc:"causal_window_end — первая находка."`
	LowerBoundEventID *string   `json:"lower_bound_event_id,omitempty"`
	UpperBoundEventID *string   `json:"upper_bound_event_id,omitempty"`
}

// OperationSpan — выполнение операции, вокруг которого идёт разбор (FR-148).
type OperationSpan struct {
	OperationRunID string     `json:"operation_run_id"`
	Label          string     `json:"label" doc:"Название операции по описанию процесса."`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `nullable:"true" json:"finished_at" doc:"null — не завершена."`
}

// Circumstances — «Разбор обстоятельств» по несоответствию (FR-58, FR-153):
// «возможные обстоятельства», не «причина».
type Circumstances struct {
	NCID                    string               `json:"nc_id"`
	Operation               *OperationSpan       `json:"operation,omitempty" doc:"Нет — выполнение операции не установлено."`
	Window                  *CausalWindow        `json:"window,omitempty" doc:"Нет — окно определить нельзя."`
	Records                 []CircumstanceRecord `json:"records"`
	MissingInformation      []string             `json:"missing_information" enum:"tool_unknown,cycle_end_time_unknown,no_observation_after_operation,no_observation_before_operation,operator_unknown,equipment_log_missing,other"`
	ConclusionIsCategorical bool                 `json:"conclusion_is_categorical"`
	BasisSeq                int64                `json:"basis_seq" doc:"seq, на котором построен ответ (для basis_seq команд, AD-39)."`
}

// CommonFactorRow — строка таблицы общих факторов «сколько из N» (FR-135).
type CommonFactorRow struct {
	Factor         string  `json:"factor" enum:"machine,tool,fixture,program,performer,material_batch"`
	Value          *string `nullable:"true" json:"value" doc:"Самое частое значение; null — неизвестно."`
	Matches        int     `json:"matches" minimum:"0"`
	DistinctValues int     `json:"distinct_values" minimum:"0"`
}

// CommonFactors — общие факторы по группе несоответствий.
type CommonFactors struct {
	GroupKey   string            `json:"group_key" doc:"Вид дефекта × операция × оборудование."`
	GroupLabel string            `json:"group_label"`
	NCCount    int               `json:"nc_count" minimum:"0"`
	Rows       []CommonFactorRow `json:"rows"`
}

// Hypothesis — гипотеза причины с доводами «за» и «против» (FR-59).
type Hypothesis struct {
	HypothesisID    string             `json:"hypothesis_id"`
	Category        string             `json:"category" enum:"incoming,equipment,performer,handling,assembly,documentation,not_established"`
	Branch          *string            `json:"branch,omitempty" enum:"why_made,why_missed"`
	Statement       *string            `json:"statement,omitempty"`
	Status          string             `json:"status" enum:"proposed_by_system,recorded,confirmed,rejected"`
	ConfidenceBP    *int               `json:"confidence_bp,omitempty" minimum:"0" maximum:"10000" doc:"Уверенность вывода в базисных пунктах — не вероятность вины."`
	Supporting      []JournalRecordRef `json:"supporting"`
	Contradicting   []JournalRecordRef `json:"contradicting"`
	MeasurementHint *string            `json:"measurement_hint,omitempty" doc:"Что измерить, чтобы проверить гипотезу."`
}

// SimilarCase — похожий прошлый случай (FR-60).
type SimilarCase struct {
	NCID           string  `json:"nc_id"`
	Number         string  `json:"number"`
	CauseCategory  *string `nullable:"true" json:"cause_category" enum:"incoming,equipment,performer,handling,assembly,documentation,not_established"`
	CauseConfirmed bool    `json:"cause_confirmed"`
	Measure        *string `nullable:"true" json:"measure"`
	Result         *string `nullable:"true" json:"result" enum:"assigned,implemented,effective,failed"`
}

// Hypotheses — гипотезы по несоответствию: версия вывода incident.hypothesis.computed и записи людей.
type Hypotheses struct {
	NCID                    string        `json:"nc_id"`
	Version                 int           `json:"version" minimum:"0"`
	Hypotheses              []Hypothesis  `json:"hypotheses"`
	MissingInformation      []string      `json:"missing_information" enum:"tool_unknown,cycle_end_time_unknown,no_observation_after_operation,no_observation_before_operation,operator_unknown,equipment_log_missing,other"`
	ConclusionIsCategorical bool          `json:"conclusion_is_categorical"`
	SimilarCases            []SimilarCase `json:"similar_cases"`
	BasisSeq                int64         `json:"basis_seq"`
}

// SimilarCaseList — похожие случаи отдельно (FR-60).
type SimilarCaseList struct {
	Items []SimilarCase `json:"items"`
}

// ScopeBreakdown — разбивка области (FR-61).
type ScopeBreakdown struct {
	InProduction int `json:"in_production" minimum:"0"`
	MovedOn      int `json:"moved_on" minimum:"0"`
	Assembled    int `json:"assembled" minimum:"0"`
	Shipped      int `json:"shipped" minimum:"0"`
}

// Reason — основание: код и текст.
type Reason struct {
	Code *string `json:"code,omitempty"`
	Text string  `json:"text"`
}

// ScopeVersion — версия области риска: каждая правка новая (FR-61).
type ScopeVersion struct {
	ScopeVersion     int            `json:"scope_version" minimum:"1"`
	Change           string         `json:"change" enum:"computed,expanded,narrowed" doc:"incident.scope.computed / expanded / narrowed."`
	Size             int            `json:"size" minimum:"0"`
	RecordedAt       time.Time      `json:"recorded_at"`
	Author           *string        `nullable:"true" json:"author" doc:"Псевдоним; null — правило системы."`
	Reason           *Reason        `json:"reason,omitempty"`
	EvidenceEventIDs []string       `json:"evidence_event_ids"`
	Breakdown        ScopeBreakdown `json:"breakdown"`
}

// ScopeItem — изделие в области риска: две оси статуса и место (FR-62).
type ScopeItem struct {
	ItemID   string `json:"item_id"`
	Label    string `json:"label"`
	Known    string `json:"known" enum:"confirmed,suspect,excluded,unknown" doc:"Ось incident словаря статусов."`
	Action   string `json:"action" enum:"observe,check,block,release" doc:"Словарь incident_action."`
	Location string `json:"location" enum:"in_production,moved_on,assembled,shipped"`
}

// FactorRef — общий фактор, по которому собрана область.
type FactorRef struct {
	Factor string `json:"factor" enum:"machine,tool,fixture,program,performer,material_batch"`
	Value  string `json:"value"`
}

// TimeWindow — окно времени.
type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// KnownGood — последнее подтверждённо нормальное состояние.
type KnownGood struct {
	Label string    `json:"label"`
	At    time.Time `json:"at"`
}

// RiskScope — область риска инцидента с версиями (FR-61, FR-62): «тающая область».
type RiskScope struct {
	IncidentID        string         `json:"incident_id"`
	IncidentLabel     string         `json:"incident_label"`
	CommonFactor      *FactorRef     `json:"common_factor,omitempty"`
	Window            *TimeWindow    `json:"window,omitempty"`
	LastKnownGood     *KnownGood     `json:"last_known_good,omitempty"`
	Versions          []ScopeVersion `json:"versions" doc:"По возрастанию."`
	Items             []ScopeItem    `json:"items" doc:"Изделия текущей версии."`
	ShippedToPartners *int           `json:"shipped_to_partners,omitempty"`
	BasisSeq          int64          `json:"basis_seq"`
}

// NcGroup — группа несоответствий: вид дефекта × операция × оборудование.
type NcGroup struct {
	GroupKey      string    `json:"group_key"`
	DefectType    string    `json:"defect_type"`
	Operation     string    `json:"operation"`
	Equipment     string    `json:"equipment"`
	NCCount       int       `json:"nc_count" minimum:"0"`
	Investigation string    `json:"investigation" enum:"not_required,not_started,in_progress,hypothesis_only,cause_confirmed,cause_not_established,measures_assigned,effectiveness_check,closed"`
	LastFoundAt   time.Time `json:"last_found_at"`
}

// NcGroupList — группы несоответствий («Разбор причин», стол технолога).
type NcGroupList struct {
	Items []NcGroup `json:"items"`
}

// IncidentSummary — инцидент в списке.
type IncidentSummary struct {
	IncidentID   string     `json:"incident_id"`
	Label        string     `json:"label"`
	CommonFactor *FactorRef `json:"common_factor,omitempty"`
	Size         int        `json:"size" minimum:"0"`
	InitialSize  int        `json:"initial_size" minimum:"0"`
	ScopeVersion int        `json:"scope_version" minimum:"0"`
	Status       string     `json:"status" enum:"open,closed"`
	OpenedAt     time.Time  `json:"opened_at"`
}

// IncidentList — инциденты.
type IncidentList struct {
	Items      []IncidentSummary `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}
