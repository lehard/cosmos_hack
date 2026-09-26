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
	Note               *string   `json:"note,omitempty" doc:"Что увидело правило автоотката, по-русски."`
	Basis              []string  `json:"basis,omitempty" doc:"event_id записей-оснований приостановки (наблюдения, пропуск брака, отчёт проверки)."`
	RunID              *string   `json:"run_id,omitempty" doc:"Прогон сценария, в котором приостановлен паспорт (AD-38)."`
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
	Provenance *string           `json:"provenance,omitempty" doc:"Происхождение записи допуска (AD-2): genesis — демо-затравка без экзамена (не промышленная валидация), personal — решение людей."`
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
	Title              *string             `json:"title,omitempty" doc:"Название анализатора."`
	AnalyzerKind       *string             `json:"analyzer_kind,omitempty" enum:"visionqc,operatorvision" doc:"Визуальный контроль или контроль действий оператора."`
	Provenance         *string             `json:"provenance,omitempty" doc:"Происхождение записи допуска (AD-2): genesis — демо-затравка без экзамена (не промышленная валидация), personal — решение людей."`
	Monitor            *AnalyzerMonitor    `json:"monitor,omitempty" doc:"Контроль дрейфа правила автоотката (FR-101)."`
	History            []AnalyzerStatus    `json:"history,omitempty" doc:"Смены статуса: допуск, приостановка, возврат, вывод."`
}

// AnalyzerStatus — смена статуса паспорта.
type AnalyzerStatus struct {
	Status  string    `json:"status" enum:"active,suspended,retired"`
	At      time.Time `json:"at"`
	EventID string    `json:"event_id"`
	Trigger *string   `json:"trigger,omitempty"`
	Note    *string   `json:"note,omitempty"`
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

// AnalyzerMonitor — контроль дрейфа паспорта (правило автоотката, FR-101):
// последние кадры, на которых анализатор отвечал, и критерии.
type AnalyzerMonitor struct {
	Window          int   `json:"window" doc:"Сколько наблюдений подряд проверяется."`
	QualityMinBP    int   `json:"quality_min_bp" doc:"Качество кадра ниже — «кадр не как при допуске»."`
	Seen            int   `json:"seen" doc:"Сколько ответов анализатора учтено правилом."`
	RecentQualityBP []int `json:"recent_quality_bp" doc:"Качество последних кадров (старые — первыми)."`
}

// MissedObservation — раннее наблюдение «признаков нет», пропустившее дефект (FR-100).
type MissedObservation struct {
	EventID    string            `json:"event_id"`
	ItemID     string            `json:"item_id,omitempty"`
	OccurredAt time.Time         `json:"occurred_at"`
	Point      *string           `json:"point,omitempty"`
	Versions   map[string]string `json:"versions" doc:"Вектор версий наблюдения (AD-29)."`
}

// RecheckItem — изделие на перепроверку: та же версия модели в той же зоне сказала «признаков нет».
type RecheckItem struct {
	ItemID              string    `json:"item_id"`
	ObservationEventIDs []string  `json:"observation_event_ids"`
	LastAt              time.Time `json:"last_at"`
}

// AdaptationEscape — пропуск брака с ранними наблюдениями и списком на перепроверку (FR-100).
type AdaptationEscape struct {
	EventID            string              `json:"event_id"`
	DefectID           string              `json:"defect_id"`
	ItemID             string              `json:"item_id"`
	AnalyzerVersion    *string             `json:"analyzer_version,omitempty"`
	MethodCoversDefect bool                `json:"method_covers_defect" doc:"Метод способен выявить вид — иначе это не ошибка модели."`
	RecordedAt         time.Time           `json:"recorded_at"`
	Missed             []MissedObservation `json:"missed"`
	Recheck            []RecheckItem       `json:"recheck"`
}

// AdaptationEscapeList — пропуски брака.
type AdaptationEscapeList struct {
	Items []AdaptationEscape `json:"items"`
}

// ObservationStage — ступень анализатора.
type ObservationStage struct {
	Name         string  `json:"name"`
	Version      string  `json:"version"`
	ConfidenceBP *int    `json:"confidence_bp,omitempty" minimum:"0" maximum:"10000"`
	Output       *string `json:"output,omitempty"`
}

// ObservationAccount — «какими версиями и почему» по наблюдению анализатора (FR-98).
type ObservationAccount struct {
	EventID            string             `json:"event_id"`
	ItemID             *string            `json:"item_id,omitempty"`
	OccurredAt         time.Time          `json:"occurred_at"`
	Point              *string            `json:"point,omitempty"`
	Outcome            string             `json:"outcome"`
	QualityBP          *int               `json:"quality_bp,omitempty" minimum:"0" maximum:"10000"`
	ConfidenceBP       *int               `json:"confidence_bp,omitempty" minimum:"0" maximum:"10000"`
	Versions           map[string]string  `json:"versions" doc:"Вектор версий наблюдения (AD-29)."`
	MissingVersions    []string           `json:"missing_versions,omitempty"`
	Stages             []ObservationStage `json:"stages,omitempty"`
	PassportID         *string            `json:"passport_id,omitempty"`
	StatusThen         string             `json:"status_then" enum:"active,suspended,retired,not_admitted" doc:"Статус паспорта на момент наблюдения."`
	StatusNow          string             `json:"status_now" enum:"active,suspended,retired,not_admitted"`
	LevelThen          int                `json:"level_then" minimum:"0" maximum:"4" doc:"Уровень доверия, с которым система реагировала."`
	AllowedAutoActions []string           `json:"allowed_auto_actions"`
	Suspicious         bool               `json:"suspicious" doc:"Изделие поставлено «под подозрением» по этому наблюдению."`
	Reasons            []string           `json:"reasons" doc:"Почему — по-русски, по шагам."`
}

// LabeledExample — размеченный пример из подтверждённого решения (FR-99):
// ответ анализатора и ответ эксперта раздельно (AD-29).
type LabeledExample struct {
	DecisionEventID    string            `json:"decision_event_id"`
	Verdict            string            `json:"verdict" enum:"confirmed,rejected" doc:"Эксперт подтвердил признак или отклонил (ложная тревога)."`
	DecidedAt          time.Time         `json:"decided_at"`
	ObservationEventID string            `json:"observation_event_id"`
	ItemID             *string           `json:"item_id,omitempty"`
	AnalyzerOutcome    string            `json:"analyzer_outcome"`
	AnalyzerDefect     *string           `json:"analyzer_defect,omitempty"`
	AnalyzerConfidence *int              `json:"analyzer_confidence_bp,omitempty" minimum:"0" maximum:"10000"`
	ExpertDefect       *string           `json:"expert_defect,omitempty"`
	ExpertReason       *string           `json:"expert_reason,omitempty"`
	Versions           map[string]string `json:"versions"`
	ForAdaptation      bool              `json:"for_adaptation" doc:"Отмечен как кандидат в размеченные данные для настройки модели."`
}

// LabeledExampleList — размеченные примеры.
type LabeledExampleList struct {
	Items []LabeledExample `json:"items"`
}
