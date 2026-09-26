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
	// Reading — числа режима оборудования (совместимое дополнение): у
	// equipment.deviation.detected и equipment.cycle.summarized.
	Reading *NCParameterReading `json:"reading,omitempty" doc:"Параметр режима числами: уставка и наблюдённые значения (отклонение режима, сводка цикла)."`
}

// NCParameterReading — параметр режима числами (AD-4: целые с масштабом, без
// float): значение = число × 10^(−scale) в единице unit. Нет границы или
// значения в записи — поля нет.
type NCParameterReading struct {
	Parameter       string `json:"parameter" doc:"Параметр у источника (current_a — ток сварки, …)."`
	Unit            string `json:"unit" doc:"Единица, код UCUM (A, V, mm, …)."`
	Scale           int    `json:"scale" minimum:"0" maximum:"12" doc:"Знаков после запятой: значение = число × 10^(−scale)."`
	SetpointNominal *int64 `json:"setpoint_nominal,omitempty" doc:"Уставка: номинал."`
	SetpointMin     *int64 `json:"setpoint_min,omitempty" doc:"Уставка: нижняя граница."`
	SetpointMax     *int64 `json:"setpoint_max,omitempty" doc:"Уставка: верхняя граница."`
	ObservedMin     *int64 `json:"observed_min,omitempty" doc:"Наблюдённый минимум (у отклонения — значение)."`
	ObservedMax     *int64 `json:"observed_max,omitempty" doc:"Наблюдённый максимум (у отклонения — значение)."`
}

// DecisionQueueRow — строка очереди «Ждут моего решения».
type DecisionQueueRow struct {
	Kind          string     `json:"kind" enum:"presentation,signal,isolated,review" doc:"Точка предъявления, сигнал на рассмотрение, изолированное изделие, пересмотр решения, принятого до новых данных (AD-3)."`
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
	// Совместимые дополнения (стол контролёра): код записи — не в тексте для людей.
	SourceEventID *string    `json:"source_event_id,omitempty" doc:"Запись, из-за которой появилась строка (у пересмотра — пришедшая после решения запись); переход к записи журнала."`
	ReviewSince   *time.Time `json:"review_since,omitempty" doc:"Пересмотр: с какого момента решение помечено «принято до новых данных» (kind = review)."`
	Reason        *string    `json:"reason,omitempty" doc:"Суть строки несоответствия: вид дефекта и зона по-русски — из справочников (нет в справочнике — не называются)."`
}

// DecisionQueue — очередь «Ждут моего решения», сортировка по риску и сроку.
type DecisionQueue struct {
	Items      []DecisionQueueRow `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// NCOperationContext — «что произошло» на операции: станок, инструмент,
// программа, исполнитель (зона «что произошло», FR-51).
type NCOperationContext struct {
	OperationRunID string  `json:"operation_run_id"`
	StepKey        string  `json:"step_key"`
	Label          string  `json:"label" doc:"Название операции по описанию процесса."`
	EquipmentID    *string `json:"equipment_id,omitempty"`
	// EquipmentLabel — название оборудования (совместимое дополнение).
	EquipmentLabel *string    `json:"equipment_label,omitempty" doc:"Название оборудования — из справочника оборудования; нет в справочнике — поля нет."`
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
	SignalID        string  `json:"signal_id"`
	BasisKind       string  `json:"basis_kind" enum:"inspection_result,equipment_deviation,check_skipped,damage_on_receipt,leak,special_process_violation,operator_report"`
	DefectTypeCode  *string `json:"defect_type_code,omitempty"`
	DefectTypeKnown bool    `json:"defect_type_known"`
	ZoneID          *string `json:"zone_id,omitempty"`
	// Названия рядом с кодами (совместимое дополнение): из справочников; кода
	// нет в справочнике — поля нет.
	DefectTypeLabel      *string           `json:"defect_type_label,omitempty" doc:"Вид дефекта по-русски — из классификатора видов дефектов; нет в классификаторе — поля нет."`
	ZoneLabel            *string           `json:"zone_label,omitempty" doc:"Зона по-русски — из зон типа изделия по КД (справочник номенклатуры); нет в справочнике — поля нет."`
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
	RuleRev        *string       `json:"rule_rev,omitempty" doc:"Ревизия нормативного слоя правила (FR-50): карта реакций, версия."`
	RevisedDueTo   *string       `nullable:"true" json:"revised_due_to" doc:"event_id записи, из-за которой вывод пересмотрен; null — первая версия."`
	RecordedAt     time.Time     `json:"recorded_at"`
	Causes         []NCRecordRef `json:"causes"`
}

// NCSystemAnalysis — анализ системы и блок «почему система это предлагает».
type NCSystemAnalysis struct {
	Versions           []NCConclusionVersion `json:"versions" doc:"По возрастанию версии."`
	Why                []string              `json:"why" doc:"Почему система это предлагает — основания по-русски."`
	Alternatives       []string              `json:"alternatives" doc:"Альтернативные объяснения."`
	MissingInformation []string              `json:"missing_information" doc:"Нехватка сведений — коды (те же, что missing_information_codes)."`
	// MissingInformationCodes — то же перечислением (совместимое дополнение
	// эпика 21: у missing_information перечисление добавить нельзя — oasdiff).
	MissingInformationCodes []string `json:"missing_information_codes,omitempty" enum:"tool_unknown,cycle_end_time_unknown,no_observation_after_operation,no_observation_before_operation,operator_unknown,equipment_log_missing,other" doc:"Нехватка сведений — перечисление (как missing_information в incident.hypothesis.computed)."`
}

// NCToDecide — зона «что решить»: допустимые решения и срок.
type NCToDecide struct {
	Decisions          []string   `json:"decisions" doc:"id операций решений, допустимых по состоянию (права — access.permission.list)."`
	DecisionDueAt      *time.Time `json:"decision_due_at,omitempty"`
	ConcessionRequired bool       `json:"concession_required" doc:"Ремонт и «как есть» — только с действующим разрешением на отклонение."`
}

// NCPresentationContext — контекст точки предъявления (FR-19, FR-56): для
// решения «Принять — передать дальше» с карточки.
type NCPresentationContext struct {
	StepKey        string `json:"step_key"`
	ClosingPoint   string `json:"closing_point" doc:"Закрывающая точка (ЗТ)."`
	PresentationNo int    `json:"presentation_no" minimum:"1" doc:"Номер предъявления (повторное — больше 1)."`
	EventID        string `json:"event_id" doc:"Запись предъявления (item.presentation.recorded)."`
	// MethodEventIDs — результаты контроля изделия, на которых можно основать
	// решение (method_event_ids команды nonconformity.presentation.resolve).
	MethodEventIDs []string `json:"method_event_ids,omitempty" doc:"Результаты методов контроля изделия (для method_event_ids решения)."`
}

// NCIsolation — изоляция изделия (FR-55): срок решения по производственному
// календарю и расхождение «изолировано в системе, физически не перемещено».
type NCIsolation struct {
	EventID            string     `json:"event_id" doc:"Решение «изолировать»."`
	IsolatedAt         time.Time  `json:"isolated_at"`
	DecisionDueAt      *time.Time `json:"decision_due_at,omitempty"`
	IsolatorLocationID *string    `json:"isolator_location_id,omitempty"`
	PhysicallyMoved    bool       `json:"physically_moved" doc:"Перемещение в изолятор подтверждено приёмкой."`
	Overdue            bool       `json:"overdue" doc:"Срок решения истёк."`
}

// NCContainmentSource — действующее основание сдерживания (FR-49, FR-62):
// правило или человек; снимает только человек (AD-27).
type NCContainmentSource struct {
	Key    string  `json:"key" doc:"Запись-основание — её указывают в released_event_ids при снятии."`
	Level  string  `json:"level" enum:"none,observe,additional_check,item_hold,lot_hold"`
	By     string  `json:"by" enum:"rule,human"`
	RuleID *string `json:"rule_id,omitempty"`
	Reason string  `json:"reason"`
	// BasisGone — основание правила ушло (снятие не делегировано): блок
	// остаётся до решения человека, движок поставил задачу пересмотра (AD-3).
	BasisGone bool `json:"basis_gone"`
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

	// Совместимые дополнения эпика 21 (пробелы, найденные эпиком 11).
	InvestigationStatus string                 `json:"investigation_status,omitempty" enum:"none,open,closed" doc:"Второй статус несоответствия — «системное расследование» (FR-51): закрытие по изделию его не закрывает."`
	Resolution          *string                `json:"resolution,omitempty" enum:"signal_rejected" doc:"Исход несоответствия, закрытого без решения по изделию."`
	Origin              string                 `json:"origin,omitempty" enum:"signal,special_process" doc:"Черновик по сигналу или регистрация окна нарушения специального процесса (FR-151)."`
	Commission          bool                   `json:"commission,omitempty" doc:"Решение принимает комиссия (специальный процесс, FR-151)."`
	RuleRev             *string                `json:"rule_rev,omitempty" doc:"Ревизия правила, построившего черновик (карта реакций)."`
	Presentation        *NCPresentationContext `json:"presentation,omitempty" doc:"Контекст точки предъявления, если изделие ждёт решения на ней."`
	Isolation           *NCIsolation           `json:"isolation,omitempty"`
	PhysicallyNotMoved  bool                   `json:"physically_not_moved,omitempty" doc:"«Изолировано в системе, физически не перемещено» (FR-55)."`
	ApprovalsStatus     *string                `json:"approvals_status,omitempty" enum:"route_closed,pending,demo_stub" doc:"Подписи маршрута решения (режим 4): pending — решение не исполняется; demo_stub — демо, подписи не проверялись."`
	Containment         []NCContainmentSource  `json:"containment,omitempty" doc:"Действующие основания сдерживания."`
	GroupItemIDs        []string               `json:"group_item_ids,omitempty" doc:"Изделия группового несоответствия: окно нарушения специального процесса — одно несоответствие на все изделия окна, решение комиссии приходит каждому (FR-151)."`
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
	// Совместимые дополнения эпика 21.
	InvestigationStatus string `json:"investigation_status,omitempty" enum:"none,open,closed" doc:"Системное расследование (FR-51)."`
	Containment         string `json:"containment,omitempty" enum:"none,observe,additional_check,item_hold,lot_hold" doc:"Сдерживание изделия."`
	Commission          bool   `json:"commission,omitempty" doc:"Решение — комиссия (FR-151)."`
	// DefectTypeLabel — вид дефекта по-русски (совместимое дополнение).
	DefectTypeLabel *string `json:"defect_type_label,omitempty" doc:"Вид дефекта по-русски — из классификатора видов дефектов; нет в классификаторе — поля нет."`
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
	// Совместимые дополнения эпика 21 (FR-54).
	Number         *string  `json:"number,omitempty" doc:"Номер по стандарту предприятия."`
	RequirementRef *string  `json:"requirement_ref,omitempty" doc:"Пункт КД/ТУ."`
	ScopeItemIDs   []string `json:"scope_item_ids,omitempty" doc:"Область действия — изделия."`
	ScopeRangeFrom *string  `json:"scope_range_from,omitempty"`
	ScopeRangeTo   *string  `json:"scope_range_to,omitempty"`
	BasisSeq       int64    `json:"basis_seq,omitempty" doc:"seq последней записи потока разрешения (basis_seq отзыва, AD-39)."`
}

// ConcessionList — разрешения на отклонение.
type ConcessionList struct {
	Items []Concession `json:"items"`
}
