package nonconformity

import "time"

// Формы ответов экранов контролёра (PRD §3a «Контролёр качества», эпик 11):
// очередь «Ждут моего решения», карточка несоответствия, разрешения на
// отклонение. Смысл — только контрактный (NFR-UI-4): «под подозрением» ≠
// «брак», «заблокировано» ≠ «признано дефектным», снятие блока ≠ годность.

// NCReason — основание решения: код и текст.
type NCReason struct {
	Code *string `json:"code,omitempty" doc:"Машинный код причины."`
	Text string  `json:"text" minLength:"1" maxLength:"2000" doc:"Текст основания по-русски."`
}

// NCItemAxes — оси статуса изделия (PRD §3b, AD-30): у каждой оси один модуль-владелец.
type NCItemAxes struct {
	Position      string `json:"position" enum:"in_queue,in_progress,at_inspection,at_presentation_point,in_transit,in_storage,isolated,completed" doc:"Положение в процессе (process)."`
	Quality       string `json:"quality" enum:"not_inspected,conforming,accepted_with_concession,unable_to_assess,signal,nonconforming" doc:"Состояние качества (quality)."`
	Disposition   string `json:"disposition" enum:"none,rework,repair,use_as_is,scrap,return_to_supplier" doc:"Решение по изделию (nonconformity)."`
	Containment   string `json:"containment" enum:"none,observe,additional_check,item_hold,lot_hold" doc:"Сдерживание (nonconformity)."`
	ErpAccounting string `json:"erp_accounting" enum:"not_sent,accepted_into_work,moved,transferred_to_scrap,returned_to_supplier,released" doc:"Учёт в 1С (erp) — только по квитанции."`
}

// NCRecordRef — ссылка на запись журнала в карточке: исходный факт, вывод
// системы или решение человека — раздельно (FR-51, кейс §3.4).
type NCRecordRef struct {
	EventID    string            `json:"event_id"`
	EventType  string            `json:"event_type" doc:"Тип записи каталога."`
	Kind       string            `json:"kind" enum:"fact,reaction,decision,service" doc:"Вид записи (AD-2): исходный факт, вывод системы, решение человека."`
	Seq        *int64            `json:"seq,omitempty" doc:"Позиция в журнале — переход к записи."`
	OccurredAt time.Time         `json:"occurred_at"`
	SourceKind *string           `json:"source_kind,omitempty" enum:"manual_entry,machine,sensor,camera,external_system,import" doc:"Вид источника факта (FR-140)."`
	Author     *string           `json:"author,omitempty" doc:"Псевдоним автора решения или подписанта."`
	Summary    string            `json:"summary" doc:"Краткое содержание для строки."`
	Params     map[string]string `json:"params,omitempty"`
}

// DecisionQueueRow — строка очереди «Ждут моего решения».
type DecisionQueueRow struct {
	Kind          string     `json:"kind" enum:"presentation,signal,isolated" doc:"Точка предъявления, сигнал на рассмотрение, изолированное изделие."`
	ObjectID      string     `json:"object_id" doc:"nc_id, signal_id или id предъявления."`
	NCID          *string    `json:"nc_id,omitempty"`
	ItemID        string     `json:"item_id"`
	ItemLabel     string     `json:"item_label" doc:"Номер детали для людей."`
	StepKey       string     `json:"step_key"`
	Title         string     `json:"title"`
	Severity      string     `json:"severity" enum:"critical,major,minor,unknown"`
	RiskRank      int        `json:"risk_rank" minimum:"0" doc:"Порядок по риску (0 — наибольший); вычисляет сервер."`
	DueAt         *time.Time `json:"due_at,omitempty" doc:"Срок решения; обратный отсчёт на экране."`
	Overdue       bool       `json:"overdue"`
	PresentationN *int       `json:"presentation_no,omitempty" doc:"Номер предъявления (повторное — уровнем выше)."`
	BasisSeq      int64      `json:"basis_seq" doc:"seq, на котором построена строка (для basis_seq команды, AD-39)."`
}

// DecisionQueue — очередь «Ждут моего решения», сортировка по риску и сроку.
type DecisionQueue struct {
	Items      []DecisionQueueRow `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// NCOperationContext — «что произошло» на операции: станок, инструмент,
// программа, исполнитель (зона «что произошло», FR-51).
type NCOperationContext struct {
	OperationRunID string     `json:"operation_run_id"`
	StepKey        string     `json:"step_key"`
	Label          string     `json:"label" doc:"Название операции по описанию процесса."`
	EquipmentID    *string    `json:"equipment_id,omitempty"`
	ToolID         *string    `json:"tool_id,omitempty"`
	ProgramRef     *string    `json:"program_ref,omitempty"`
	PerformerID    *string    `json:"performer_id,omitempty" doc:"Псевдоним исполнителя; нет — неизвестно."`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
}

// NCHappened — зона «что произошло»: до операции / операция / после.
type NCHappened struct {
	Before    []NCRecordRef       `json:"before"`
	Operation *NCOperationContext `json:"operation,omitempty"`
	During    []NCRecordRef       `json:"during" doc:"Состояние оборудования и действия исполнителя на операции."`
	After     []NCRecordRef       `json:"after"`
}

// NCAnalyzerStage — ступень ансамбля VisionQC: где дефект / какой тип (FR-38).
type NCAnalyzerStage struct {
	Stage        string `json:"stage"`
	Version      string `json:"version"`
	ConfidenceBP *int   `json:"confidence_bp,omitempty" minimum:"0" maximum:"10000" doc:"Уверенность ступени, б. п. — не вероятность брака."`
	OutputNote   string `json:"output_note,omitempty"`
}

// NCSourceSignal — исходный сигнал в карточке (отдельно от анализа системы).
type NCSourceSignal struct {
	SignalID             string            `json:"signal_id"`
	BasisKind            string            `json:"basis_kind" enum:"inspection_result,equipment_deviation,check_skipped,damage_on_receipt,leak,special_process_violation,operator_report"`
	DefectTypeCode       *string           `json:"defect_type_code,omitempty"`
	DefectTypeKnown      bool              `json:"defect_type_known"`
	ZoneID               *string           `json:"zone_id,omitempty"`
	Severity             string            `json:"severity" enum:"critical,major,minor,unknown"`
	AnalyzerConfidenceBP *int              `json:"analyzer_confidence_bp,omitempty" minimum:"0" maximum:"10000" doc:"Уверенность анализатора, б. п.; ≠ вероятность брака."`
	ObservationQualityBP *int              `json:"observation_quality_bp,omitempty" minimum:"0" maximum:"10000" doc:"Качество наблюдения, б. п."`
	Stages               []NCAnalyzerStage `json:"stages"`
	Versions             map[string]string `json:"versions,omitempty" doc:"Вектор версий наблюдения (AD-29)."`
	EvidenceRefs         []string          `json:"evidence_refs" doc:"Адреса материалов: кадры, иллюстрация."`
	Record               NCRecordRef       `json:"record"`
}

// NCRequirement — требование: характеристика, допуск, ревизия КД.
type NCRequirement struct {
	Characteristic string  `json:"characteristic"`
	Tolerance      *string `json:"tolerance,omitempty"`
	KDRef          *string `json:"kd_ref,omitempty"`
}

// NCEvidence — зона «доказательства».
type NCEvidence struct {
	Signals      []NCSourceSignal `json:"signals"`
	Requirement  *NCRequirement   `json:"requirement,omitempty"`
	ZoneHistory  []NCRecordRef    `json:"zone_history" doc:"История этой зоны до и после операции."`
	SimilarCount int              `json:"similar_count" minimum:"0" doc:"Похожих случаев (analysis.similar.list)."`
}

// NCConclusionVersion — версия вывода системы по слоту реакции (AD-3): при
// пересвёртке — новая версия с причиной пересмотра; обе видны.
type NCConclusionVersion struct {
	Version        int           `json:"version" minimum:"1"`
	EventID        string        `json:"event_id"`
	RuleID         string        `json:"rule_id"`
	AutomationMode int           `json:"automation_mode" minimum:"1" maximum:"5" doc:"Режим автоматизации правила (FR-50)."`
	Outcome        string        `json:"outcome" enum:"pass_to_next,manual_review,isolate,question_to_technologist"`
	RevisedDueTo   *string       `nullable:"true" json:"revised_due_to" doc:"event_id записи, из-за которой вывод пересмотрен; null — первая версия."`
	RecordedAt     time.Time     `json:"recorded_at"`
	Causes         []NCRecordRef `json:"causes"`
}

// NCSystemAnalysis — анализ системы и блок «почему система это предлагает».
type NCSystemAnalysis struct {
	Versions           []NCConclusionVersion `json:"versions" doc:"По возрастанию версии."`
	Why                []string              `json:"why" doc:"Почему система это предлагает — основания по-русски."`
	Alternatives       []string              `json:"alternatives" doc:"Альтернативные объяснения."`
	MissingInformation []string              `json:"missing_information" doc:"Нехватка сведений."`
}

// NCToDecide — зона «что решить»: допустимые решения и срок.
type NCToDecide struct {
	Decisions          []string   `json:"decisions" doc:"id операций решений, допустимых по состоянию (права — access.permission.list)."`
	DecisionDueAt      *time.Time `json:"decision_due_at,omitempty"`
	ConcessionRequired bool       `json:"concession_required" doc:"Ремонт и «как есть» — только с действующим разрешением на отклонение."`
}

// NCCard — карточка несоответствия (FR-51): три зоны, анализ системы, решения людей.
type NCCard struct {
	NCID           string           `json:"nc_id"`
	Number         string           `json:"number" doc:"Номер для людей."`
	Status         string           `json:"status" enum:"draft,confirmed,disposition_set,verified,closed"`
	ItemID         string           `json:"item_id"`
	ItemLabel      string           `json:"item_label"`
	Axes           NCItemAxes       `json:"axes"`
	Happened       NCHappened       `json:"happened"`
	Evidence       NCEvidence       `json:"evidence"`
	SystemAnalysis NCSystemAnalysis `json:"system_analysis"`
	HumanDecisions []NCRecordRef    `json:"human_decisions" doc:"Решения людей с подписью (отдельно от вывода системы)."`
	ToDecide       NCToDecide       `json:"to_decide"`
	BasisSeq       int64            `json:"basis_seq" doc:"seq, на котором построена карточка (для basis_seq команд, AD-39)."`
}

// NCSummary — несоответствие в списке.
type NCSummary struct {
	NCID           string    `json:"nc_id"`
	Number         string    `json:"number"`
	Status         string    `json:"status" enum:"draft,confirmed,disposition_set,verified,closed"`
	ItemID         string    `json:"item_id"`
	ItemLabel      string    `json:"item_label"`
	DefectTypeCode *string   `json:"defect_type_code,omitempty"`
	Severity       string    `json:"severity" enum:"critical,major,minor,unknown"`
	StepKey        string    `json:"step_key"`
	Disposition    string    `json:"disposition" enum:"none,rework,repair,use_as_is,scrap,return_to_supplier"`
	FoundAt        time.Time `json:"found_at"`
}

// NCList — несоответствия.
type NCList struct {
	Items      []NCSummary `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

// Concession — разрешение на отклонение (FR-54): действует после закрытия
// маршрута подписей, может покрывать серию изделий с лимитом.
type Concession struct {
	ConcessionID string     `json:"concession_id"`
	Title        string     `json:"title"`
	Kind         string     `json:"kind" enum:"repair,use_as_is" doc:"Для какого решения."`
	DocumentID   string     `json:"document_id" doc:"Документ разрешения (закрытый маршрут)."`
	Limit        *int       `json:"limit,omitempty" minimum:"0" doc:"Лимит изделий."`
	Used         int        `json:"used" minimum:"0"`
	ValidUntil   *time.Time `json:"valid_until,omitempty"`
	Status       string     `json:"status" enum:"active,exhausted,expired,revoked"`
}

// ConcessionList — разрешения на отклонение.
type ConcessionList struct {
	Items []Concession `json:"items"`
}
