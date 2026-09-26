package process

import (
	"time"

	"ant/internal/application/platform"
)

// ── живая карта (FR-1…FR-5, FR-9, FR-130, FR-155; AD-21, AD-22) ──
// Форма — frontend/src/entities/live-map/index.ts (LiveMapData). Счётчики
// узлов и ограничение линии считает сервер (analytics), отдаёт этот ответ.

// MapNodeCounters — счётчики узла по step_key за период (FR-2, FR-3).
type MapNodeCounters struct {
	StepKey         string `json:"step_key"`
	Queue           int    `json:"queue" minimum:"0"`
	InProgress      int    `json:"in_progress" minimum:"0"`
	Passed          int    `json:"passed" minimum:"0"`
	Defects         int    `json:"defects" minimum:"0" doc:"Физические дефекты за период, не наблюдения (соглашение «Дефект»)."`
	Nonconformities *int   `json:"nonconformities,omitempty" minimum:"0" doc:"Открытые несоответствия узла (FR-154)."`
}

// MapNodeAnomaly — аномалия узла (FR-5).
type MapNodeAnomaly struct {
	StepKey   string `json:"step_key"`
	Kind      string `json:"kind" enum:"queue_above_norm,wait_above_norm,downtime_over_threshold,output_spike,defect_rate_out_of_control"`
	Threshold string `json:"threshold,omitempty" doc:"Порог текстом с единицей."`
}

// MapItem — изделие-точка в текущем узле (FR-2).
type MapItem struct {
	ItemID           string `json:"item_id"`
	Label            string `json:"label" doc:"Номер детали для подписи точки."`
	StepKey          string `json:"step_key"`
	Position         string `json:"position" enum:"in_queue,in_progress,at_inspection,at_presentation_point,in_transit,in_storage,isolated,completed"`
	Summary          string `json:"summary" enum:"in_process,suspect,reinspection_required,hold,pending_decision,nonconforming,cleared,released,in_rework,in_repair,accepted_with_concession,scrapped,returned"`
	ProcessVersionID string `json:"process_version_id" doc:"Версия процесса, по которой изделие запущено (AD-17)."`
	IncidentStatus   string `json:"incident_status,omitempty" enum:"confirmed,suspect,excluded,unknown" doc:"Статус в выбранном инциденте; нет — вне области (FR-9)."`
}

// MapVersionRef — версия процесса на карте.
type MapVersionRef struct {
	ProcessVersionID string `json:"process_version_id"`
	Label            string `json:"label"`
	IsCurrent        bool   `json:"is_current"`
	Items            int    `json:"items" minimum:"0" doc:"Изделий в работе по этой версии."`
}

// MapBottleneck — узел-ограничение линии (FR-5).
type MapBottleneck struct {
	StepKey string `json:"step_key"`
	Wait    string `json:"wait,omitempty" doc:"Среднее ожидание текстом с единицей."`
}

// MapIncident — выбранный инцидент на карте (FR-9, FR-61).
type MapIncident struct {
	IncidentID     string `json:"incident_id"`
	Label          string `json:"label"`
	ScopeVersion   int    `json:"scope_version" minimum:"0"`
	SizeAtCreation int    `json:"size_at_creation" minimum:"0"`
	Size           int    `json:"size" minimum:"0" doc:"Размер области сейчас (34 → 13 → 6)."`
	Basis          string `json:"basis,omitempty" doc:"Основание последнего изменения области."`
}

// LiveMap — состояние живой карты на момент (AD-22).
type LiveMap struct {
	ProcessVersion MapVersionRef     `json:"process_version"`
	Versions       []MapVersionRef   `json:"versions"`
	BpmnXML        string            `json:"bpmn_xml" doc:"BPMN 2.0 XML показанной версии: BPMNDI, documentation, ant:properties/@stepKey."`
	Counters       []MapNodeCounters `json:"counters"`
	Items          []MapItem         `json:"items"`
	Bottleneck     *MapBottleneck    `json:"bottleneck,omitempty" doc:"Нет — ограничение не выявлено."`
	Anomalies      []MapNodeAnomaly  `json:"anomalies"`
	DataGaps       []string          `json:"data_gaps" doc:"step_key узлов, где оценка невозможна: нет данных источника (не «норма»)."`
	Incident       *MapIncident      `json:"incident,omitempty"`
	BasisSeq       int64             `json:"basis_seq"`
}

// LiveMapQuery — параметры живой карты помимо момента.
type LiveMapQuery struct {
	ProcessVersionID string
	Period           string
	From, To         *time.Time
	IncidentID       string
}

// ── карточка узла (FR-154, FR-156) ──

// NormRef — нормативная опора шага (ant:normRef, FR-156).
type NormRef struct {
	Standard     string `json:"standard"`
	Clause       string `json:"clause"`
	Check        string `json:"check,omitempty" doc:"Что проверить на шаге."`
	SystemAction string `json:"system_action,omitempty" doc:"Что делает система на шаге."`
}

// ProcessNodeCard — карточка узла живой карты (FR-154): описание шага из
// documentation версии изделия, свойства, опоры и счётчики с переходом к
// изделиям и несоответствиям.
type ProcessNodeCard struct {
	ProcessVersionID string              `json:"process_version_id"`
	StepKey          string              `json:"step_key"`
	ElementID        string              `json:"element_id"`
	Name             string              `json:"name"`
	Kind             string              `json:"kind" enum:"automatedInspection,operation,humanInspection,parallelGateway,inclusiveGateway,startEvent,endEvent,intermediateEvent,timer,subprocess,callActivity,conditionalFlow,lane"`
	Lane             string              `json:"lane,omitempty"`
	Documentation    string              `json:"documentation" doc:"<bpmn:documentation> элемента."`
	Properties       map[string]any      `json:"properties" doc:"Наши свойства (ключи process.properties.*)."`
	NormRefs         []NormRef           `json:"norm_refs"`
	Counters         MapNodeCounters     `json:"counters"`
	Items            []platform.DrillRef `json:"items" doc:"Изделия в узле."`
	Nonconformities  []platform.DrillRef `json:"nonconformities" doc:"Открытые несоответствия узла."`
}

// ── версии в читаемом виде (FR-22…FR-24) ──
// Форма — frontend/src/entities/process-version/model/types.ts.

// ProcessElement — элемент описания процесса.
type ProcessElement struct {
	ID         string         `json:"id" doc:"id элемента BPMN."`
	StepKey    *string        `json:"step_key,omitempty" doc:"ant:properties/@stepKey — сопоставляет элементы версий."`
	Kind       string         `json:"kind" enum:"automatedInspection,operation,humanInspection,parallelGateway,inclusiveGateway,startEvent,endEvent,intermediateEvent,timer,subprocess,callActivity,conditionalFlow,lane"`
	Name       string         `json:"name"`
	Lane       *string        `json:"lane,omitempty" doc:"Цех (дорожка)."`
	Properties map[string]any `json:"properties" doc:"Ключи process.properties.*; значения — строка, число, логическое или null."`
	Thresholds map[string]int `json:"thresholds,omitempty" doc:"Пороги уверенности карты реакций по видам дефектов, б. п."`
	Next       []string       `json:"next,omitempty" doc:"id следующих элементов."`
}

// VersionQuorum — кворум утверждения (FR-23).
type VersionQuorum struct {
	Have int `json:"have" minimum:"0"`
	Need int `json:"need" minimum:"0"`
}

// ProcessVersion — версия процесса в читаемом виде (FR-24).
type ProcessVersion struct {
	VersionID     string           `json:"version_id"`
	Label         string           `json:"label"`
	Status        string           `json:"status" enum:"draft,on_approval,active,retired"`
	Hash          string           `json:"hash,omitempty" doc:"Хеш версии — H(байты XML как загружены), AD-17."`
	Author        *string          `nullable:"true" json:"author,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	EffectiveFrom *time.Time       `nullable:"true" json:"effective_from,omitempty"`
	Quorum        *VersionQuorum   `json:"quorum,omitempty"`
	Elements      []ProcessElement `json:"elements" doc:"Элементы в порядке маршрута."`
	BasisSeq      int64            `json:"basis_seq"`
}

// ProcessVersionSummary — версия в списке.
type ProcessVersionSummary struct {
	VersionID     string         `json:"version_id"`
	Label         string         `json:"label"`
	Status        string         `json:"status" enum:"draft,on_approval,active,retired"`
	Hash          string         `json:"hash,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	EffectiveFrom *time.Time     `nullable:"true" json:"effective_from,omitempty"`
	Quorum        *VersionQuorum `json:"quorum,omitempty"`
	ItemsInWork   int            `json:"items_in_work" minimum:"0"`
}

// ProcessVersionList — версии процесса.
type ProcessVersionList struct {
	Items []ProcessVersionSummary `json:"items"`
}

// ProcessDiffEntry — отличие версии от действующей (строки process.diff.*, FR-24):
// дискриминатор kind; поля — по виду записи.
type ProcessDiffEntry struct {
	Kind       string `json:"kind" enum:"elementAdded,elementRemoved,presentationPointAdded,thresholdChanged,propertyChanged"`
	Element    string `json:"element,omitempty" doc:"elementAdded, elementRemoved, thresholdChanged, propertyChanged."`
	Step       string `json:"step,omitempty" doc:"presentationPointAdded."`
	DefectType string `json:"defectType,omitempty" doc:"thresholdChanged."`
	Property   string `json:"property,omitempty" doc:"propertyChanged: ключ process.properties.*."`
	From       any    `json:"from,omitempty" doc:"Было (thresholdChanged, propertyChanged)."`
	To         any    `json:"to,omitempty" doc:"Стало."`
}

// ProcessVersionDiff — читаемая разница двух версий (FR-24).
type ProcessVersionDiff struct {
	VersionID string             `json:"version_id"`
	AgainstID string             `json:"against_id" doc:"С чем сравнивается (по умолчанию — действующая)."`
	Entries   []ProcessDiffEntry `json:"entries"`
}

// ProcessBpmn — подписываемая версия целиком: BPMN XML как загружен (AD-17).
type ProcessBpmn struct {
	VersionID string `json:"version_id"`
	Hash      string `json:"hash" doc:"H(байты XML как загружены), без канонизации XML."`
	BpmnXML   string `json:"bpmn_xml"`
}
