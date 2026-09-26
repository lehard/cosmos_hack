package item

import "time"

// ItemLookup — результат поиска изделия по носителю (AD-16, AD-41).
type ItemLookup struct {
	ItemID     string `json:"item_id" doc:"Внутренний ID изделия: код_предприятия:локальный_id (AD-16)."`
	CarrierRef string `json:"carrier_ref,omitempty" doc:"Носитель, по которому найдено: ant:carrier:‹тип›:‹значение›."`
	// Candidates — носитель действовал у нескольких изделий: не угадываем (FR-34).
	Candidates []string `json:"candidates,omitempty" doc:"Все изделия с этим носителем на момент, если их несколько (FR-34): выбор — за человеком."`
}

// ItemStatus — оси статуса изделия (PRD §3b, AD-30): у каждой оси один
// модуль-владелец; сводный статус вычисляется из осей и сам осью не является.
type ItemStatus struct {
	Position      string `json:"position" enum:"in_queue,in_progress,at_inspection,at_presentation_point,in_transit,in_storage,isolated,completed" doc:"Положение в процессе (process)."`
	Quality       string `json:"quality" enum:"not_inspected,conforming,accepted_with_concession,unable_to_assess,signal,nonconforming" doc:"Состояние качества (quality)."`
	Disposition   string `json:"disposition" enum:"none,rework,repair,use_as_is,scrap,return_to_supplier" doc:"Решение по изделию (nonconformity)."`
	Containment   string `json:"containment" enum:"none,observe,additional_check,item_hold,lot_hold" doc:"Сдерживание (nonconformity); снятие блока ≠ годность."`
	ErpAccounting string `json:"erp_accounting" enum:"not_sent,accepted_into_work,moved,transferred_to_scrap,returned_to_supplier,released" doc:"Учёт в 1С (erp) — только по квитанции."`
	Summary       string `json:"summary" enum:"in_process,suspect,reinspection_required,hold,pending_decision,nonconforming,cleared,released,in_rework,in_repair,accepted_with_concession,scrapped,returned" doc:"Сводный статус для списков и карты (словарь item_summary)."`
}

// ItemRow — изделие в списке (проваливание из узла карты, очереди, партии).
type ItemRow struct {
	ItemID       string     `json:"item_id"`
	Label        string     `json:"label" doc:"Номер детали для людей."`
	ItemTypeID   string     `json:"item_type_id"`
	StepKey      string     `json:"step_key,omitempty" doc:"Текущий шаг (AD-17)."`
	VersionLabel string     `json:"version_label,omitempty" doc:"Версия процесса изделия (изделия прежних версий — с пометкой)."`
	LotID        string     `json:"lot_id,omitempty"`
	Status       ItemStatus `json:"status"`
	RunID        string     `json:"run_id,omitempty" doc:"Прогон сценария (AD-38)."`
}

// ItemList — список изделий.
type ItemList struct {
	Items      []ItemRow `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

// ItemSignature — подпись под записью паспорта и статус её проверки (FR-68, AD-2).
type ItemSignature struct {
	SignerID   string `json:"signer_id" doc:"Псевдоним подписанта или id устройства."`
	Class      string `json:"class" enum:"device,personal,paper,partner,server_attested,scenario,genesis" doc:"Класс происхождения подписи (AD-2)."`
	Level      int    `json:"level" minimum:"0" maximum:"3" doc:"Уровень подписи (AD-13)."`
	Check      string `json:"check" enum:"valid,rejected,not_verifiable,unchecked" doc:"Статус проверки: цело / отвергнуто / не проверяемо / не проверялась (демо без подписей)."`
	AttestedBy string `json:"attested_by,omitempty" doc:"Заверитель бумажной подписи (AD-43)."`
	// PaperOriginalRef, ScanAddress — бумага с заверением (AD-43): учётный
	// номер бумажного оригинала в архиве ОТК и адрес скана в хранилище материалов.
	PaperOriginalRef string `json:"paper_original_ref,omitempty" doc:"Учётный номер бумажного оригинала в архиве ОТК (AD-43)."`
	ScanAddress      string `json:"scan_address,omitempty" doc:"Адрес скана в хранилище материалов: streebog256:… (AD-23, AD-43)."`
}

// PassportEntry — запись паспорта: факт, решение или реакция — раздельно (кейс §7.2, FR-42).
type PassportEntry struct {
	EventID     string          `json:"event_id"`
	EventType   string          `json:"event_type" doc:"Тип записи каталога."`
	Kind        string          `json:"kind" enum:"fact,reaction,decision,service" doc:"Вид записи (AD-2): исходный сигнал, анализ системы, решение человека."`
	Seq         int64           `json:"seq"`
	OccurredAt  time.Time       `json:"occurred_at"`
	RecordedAt  time.Time       `json:"recorded_at"`
	StepKey     string          `json:"step_key,omitempty"`
	Author      string          `json:"author,omitempty" doc:"Автор (псевдоним) или источник."`
	SourceKind  string          `json:"source_kind,omitempty" enum:"manual_entry,machine,sensor,camera,external_system,import" doc:"Пометка источника факта (FR-140)."`
	Reliability string          `json:"reliability,omitempty" enum:"high,medium,low,unknown"`
	Summary     string          `json:"summary" doc:"Краткое содержание для ленты истории."`
	Signatures  []ItemSignature `json:"signatures"`
	Corrects    string          `json:"corrects,omitempty" doc:"Исправляемая запись (FR-122)."`
	CARef       string          `json:"ca_ref,omitempty" doc:"Критическое действие (AD-28)."`
	// BindingBasis, BindingReliability — привязка события к изделию (AD-41,
	// FR-34): как привязано и насколько надёжно; у событий, привязанных
	// межизделийной стадией, — основание стадии.
	BindingBasis       string `json:"binding_basis,omitempty" doc:"Как событие привязано к изделию: internal_id, carrier, post_context, time_window, manual (AD-41)."`
	BindingReliability string `json:"binding_reliability,omitempty" doc:"Надёжность привязки: unique, probable, ambiguous, unidentified (FR-34)."`
	// Candidates — кандидаты неоднозначной привязки (FR-34): событие не угадано.
	Candidates []string `json:"candidates,omitempty" doc:"Кандидаты при неоднозначной привязке события (FR-34)."`
	// Bound — запись-копия события без изделия, привязанного стадией или человеком.
	Bound bool `json:"bound,omitempty" doc:"Событие пришло без изделия и привязано позже (AD-41)."`
}

// ItemDocumentRef — документ изделия (FR-65: «документов собрано из истории»).
type ItemDocumentRef struct {
	DocumentID string `json:"document_id"`
	Template   string `json:"template" doc:"Шаблон@версия."`
	Title      string `json:"title"`
	Status     string `json:"status" enum:"drafted,in_route,closed,annulled"`
	Digest     string `json:"digest" doc:"Отпечаток документа (AD-12)."`
}

// ItemZone — зона изделия (FR-46): доступ, закрыта ли, скрытые работы.
type ItemZone struct {
	ZoneID           string `json:"zone_id"`
	Title            string `json:"title"`
	Closed           bool   `json:"closed" doc:"Доступ к зоне закрыт (FR-20)."`
	ClosedBy         string `json:"closed_by,omitempty" doc:"Шаг, закрывший доступ."`
	OpenIntervention string `json:"open_intervention,omitempty" doc:"Открытое вмешательство (FR-21)."`
	InspectionStatus string `json:"inspection_status,omitempty" doc:"Проверка зоны (FR-46): not_inspected — не проверялась, inspected — проверена, stale — устарела после вмешательства."`
	LastInspection   string `json:"last_inspection,omitempty" doc:"Последний результат контроля зоны (event_id)."`
}

// ItemCarrier — носитель идентификатора (AD-16).
type ItemCarrier struct {
	CarrierType string     `json:"carrier_type" enum:"dpm_datamatrix,tag_qr,container_cell,route_card,post_context,manual_entry,internal_id"`
	Value       string     `json:"value"`
	ZoneID      string     `json:"zone_id,omitempty"`
	Temporary   bool       `json:"temporary"`
	State       string     `json:"state" enum:"applied,verified,unreadable,removed,replaced"`
	AppliedAt   time.Time  `json:"applied_at"`
	RemovedAt   *time.Time `json:"removed_at,omitempty"`
}

// ItemPassport — паспорт изделия (FR-42, кейс «история изделия»): исходные
// сигналы, анализ системы, решения людей и итоговый статус — раздельно.
type ItemPassport struct {
	ItemID         string            `json:"item_id"`
	Label          string            `json:"label"`
	ItemTypeID     string            `json:"item_type_id"`
	ItemRevision   string            `json:"item_revision"`
	ProcessVersion string            `json:"process_version" doc:"Закреплённая версия процесса (хеш, AD-17)."`
	StepKey        string            `json:"step_key,omitempty"`
	OrderID        string            `json:"order_id,omitempty" doc:"Задание 1С."`
	LotIDs         []string          `json:"lot_ids"`
	Identification string            `json:"identification" enum:"unique,probable,ambiguous,unidentified" doc:"Идентификация (AD-16): под сомнением — изоляция до повторной идентификации."`
	Status         ItemStatus        `json:"status"`
	Entries        []PassportEntry   `json:"entries" doc:"Записи паспорта по времени."`
	Documents      []ItemDocumentRef `json:"documents"`
	Zones          []ItemZone        `json:"zones"`
	Carriers       []ItemCarrier     `json:"carriers"`
	Incidents      []string          `json:"incidents" doc:"Инциденты, в области которых изделие."`
	// IncidentStatuses — статус изделия в каждом инциденте по двум осям (FR-62).
	IncidentStatuses []ItemIncident `json:"incident_statuses,omitempty" doc:"Статус изделия в каждом инциденте: что известно и что делать (FR-62)."`
	// Questions — «идентификация под сомнением» (AD-16).
	Questions []IdentificationQuestion `json:"identification_questions,omitempty" doc:"Идентификация под сомнением: причина, кандидаты; открытое — изоляция до повторной идентификации (AD-16)."`
	// Witnesses — результаты образца-свидетеля групп изделия (FR-15).
	Witnesses []WitnessResult `json:"witness_results,omitempty" doc:"Результаты образца-свидетеля садки или групповой операции (FR-15)."`
	// Holds — сдерживание, пришедшее по генеалогии (AD-42).
	Holds []GenealogyHold `json:"genealogy_holds,omitempty" doc:"Сдерживание по генеалогии: блок партии или компонента (AD-42)."`
	// Interventions — вмешательства в собранное изделие (FR-21).
	Interventions []ItemIntervention `json:"interventions,omitempty" doc:"Вмешательства в собранное изделие (FR-21)."`
	// RefChanges — изменения справочника с действием в прошлом (AD-31).
	RefChanges      []ItemRefChange `json:"reference_changes,omitempty" doc:"Изменения справочника с действием в прошлом, затронувшие изделие (AD-31)."`
	SplitFrom       string          `json:"split_from,omitempty" doc:"Изделие, из которого выделено разделением 1→N (FR-15)."`
	Nonconformities []string        `json:"nonconformities"`
	BasisSeq        int64           `json:"basis_seq" doc:"seq, на котором построен паспорт (для команд, AD-39)."`
}

// ItemHistoryEntry — строка журнала изменений паспорта (FR-43): было / стало / кто / причина.
type ItemHistoryEntry struct {
	Seq        int64     `json:"seq"`
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	RecordedAt time.Time `json:"recorded_at"`
	Field      string    `json:"field" doc:"Что изменилось (ось статуса, носитель, зона…)."`
	Before     string    `json:"before,omitempty"`
	After      string    `json:"after,omitempty"`
	Author     string    `json:"author,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}

// ItemHistory — журнал изменений паспорта.
type ItemHistory struct {
	Items      []ItemHistoryEntry `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// GenealogyNode — узел дерева генеалогии: изделие или партия (FR-45; владелец данных — crossitem).
type GenealogyNode struct {
	Ref        string `json:"ref" doc:"item_id или lot_id."`
	Kind       string `json:"kind" enum:"item,lot,partner_extract"`
	Label      string `json:"label"`
	ParentRef  string `json:"parent_ref,omitempty" doc:"Куда вошёл (сборка)."`
	Position   string `json:"position,omitempty" doc:"Позиция в сборке."`
	Summary    string `json:"summary,omitempty" enum:"in_process,suspect,reinspection_required,hold,pending_decision,nonconforming,cleared,released,in_rework,in_repair,accepted_with_concession,scrapped,returned"`
	Provenance string `json:"provenance,omitempty" doc:"Для выписки партнёра — «происхождение подтверждено / не подтверждено» (AD-19)."`
	Relation   string `json:"relation,omitempty" doc:"Связь с изделием паспорта: component_of, assembly, made_from_lot, split_from, split_into, grouped_with."`
	Depth      int    `json:"depth,omitempty" doc:"Уровень от изделия паспорта: вниз — положительный, вверх — отрицательный."`
}

// ItemGenealogy — генеалогия изделия вверх (из чего собрано) и вниз (куда вошло).
type ItemGenealogy struct {
	ItemID string          `json:"item_id"`
	Nodes  []GenealogyNode `json:"nodes"`
	// Up, Down — запросы вверх и вниз по дереву сборки (FR-45).
	Up     []string `json:"up,omitempty" doc:"Сборки, в которые вошло изделие, снизу вверх (FR-45)."`
	Down   []string `json:"down,omitempty" doc:"Компоненты-экземпляры на всех уровнях (FR-45)."`
	Lots   []string `json:"lots,omitempty" doc:"Партии изделия и партионных компонентов."`
	Groups []string `json:"groups,omitempty" doc:"Временные группы (садки) изделия."`
}

// ItemIncident — статус изделия в инциденте по двум осям (FR-62).
type ItemIncident struct {
	IncidentID    string `json:"incident_id"`
	Status        string `json:"status" enum:"confirmed,suspect,excluded,unknown" doc:"Что известно."`
	Action        string `json:"action" enum:"observe,check,block,release" doc:"Что делать."`
	ScopeVersion  int    `json:"scope_version" minimum:"0"`
	ViaAssemblyOf string `json:"via_assembly_of,omitempty" doc:"Попало в область через компонент."`
}

// IdentificationQuestion — «идентификация под сомнением» (AD-16).
type IdentificationQuestion struct {
	Cause          string    `json:"cause" enum:"carrier_unreadable,carrier_mismatch,ambiguous_binding,carrier_missing"`
	Candidates     []string  `json:"candidates,omitempty"`
	Basis          []string  `json:"basis"`
	At             time.Time `json:"at"`
	QuestionedID   string    `json:"questioned_event_id" doc:"Запись item.identification.questioned — её снимает подтверждение."`
	SubjectEventID string    `json:"subject_event_id,omitempty" doc:"Событие с неоднозначной привязкой."`
	Open           bool      `json:"open"`
	ClosedBy       string    `json:"closed_by,omitempty"`
}

// WitnessResult — результат образца-свидетеля группы (FR-15).
type WitnessResult struct {
	GroupID           string    `json:"group_id"`
	GroupKind         string    `json:"group_kind"`
	WitnessItemID     string    `json:"witness_item_id"`
	InspectionEventID string    `json:"inspection_event_id"`
	Outcome           string    `json:"outcome"`
	Method            string    `json:"method,omitempty"`
	ConclusionRef     string    `json:"conclusion_ref,omitempty"`
	At                time.Time `json:"at"`
}

// GenealogyHold — сдерживание, пришедшее по генеалогии (AD-42).
type GenealogyHold struct {
	Level         string   `json:"level"`
	Source        string   `json:"source" doc:"lot — блок партии; component — блок компонента; split_parent — блок исходного изделия."`
	SourceEventID string   `json:"source_event_id"`
	LotID         string   `json:"lot_id,omitempty"`
	SourceItemID  string   `json:"source_item_id,omitempty"`
	Path          []string `json:"path,omitempty"`
	Released      bool     `json:"released" doc:"Основание снято у источника; блок снимает человек (AD-27)."`
}

// ItemIntervention — вмешательство (FR-21).
type ItemIntervention struct {
	InterventionID string     `json:"intervention_id"`
	ZoneIDs        []string   `json:"zone_ids"`
	Purpose        string     `json:"purpose,omitempty"`
	OpenedAt       time.Time  `json:"opened_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	RetestRequired bool       `json:"retest_required,omitempty"`
}

// ItemRefChange — изменение справочника с действием в прошлом (AD-31).
type ItemRefChange struct {
	ReferenceEventID string `json:"reference_event_id"`
	Kind             string `json:"kind"`
	ValidFrom        string `json:"valid_from"`
}
