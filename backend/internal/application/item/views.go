package item

import "time"

// ItemLookup — результат поиска изделия по носителю (AD-16, AD-41).
type ItemLookup struct {
	ItemID     string `json:"item_id" doc:"Внутренний ID изделия: код_предприятия:локальный_id (AD-16)."`
	CarrierRef string `json:"carrier_ref,omitempty" doc:"Носитель, по которому найдено: ant:carrier:‹тип›:‹значение›."`
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
	ItemID          string            `json:"item_id"`
	Label           string            `json:"label"`
	ItemTypeID      string            `json:"item_type_id"`
	ItemRevision    string            `json:"item_revision"`
	ProcessVersion  string            `json:"process_version" doc:"Закреплённая версия процесса (хеш, AD-17)."`
	StepKey         string            `json:"step_key,omitempty"`
	OrderID         string            `json:"order_id,omitempty" doc:"Задание 1С."`
	LotIDs          []string          `json:"lot_ids"`
	Identification  string            `json:"identification" enum:"unique,probable,ambiguous,unidentified" doc:"Идентификация (AD-16): под сомнением — изоляция до повторной идентификации."`
	Status          ItemStatus        `json:"status"`
	Entries         []PassportEntry   `json:"entries" doc:"Записи паспорта по времени."`
	Documents       []ItemDocumentRef `json:"documents"`
	Zones           []ItemZone        `json:"zones"`
	Carriers        []ItemCarrier     `json:"carriers"`
	Incidents       []string          `json:"incidents" doc:"Инциденты, в области которых изделие."`
	Nonconformities []string          `json:"nonconformities"`
	BasisSeq        int64             `json:"basis_seq" doc:"seq, на котором построен паспорт (для команд, AD-39)."`
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
}

// ItemGenealogy — генеалогия изделия вверх (из чего собрано) и вниз (куда вошло).
type ItemGenealogy struct {
	ItemID string          `json:"item_id"`
	Nodes  []GenealogyNode `json:"nodes"`
}
