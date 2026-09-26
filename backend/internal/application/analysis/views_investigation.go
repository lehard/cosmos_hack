package analysis

import "time"

// Поля стола технолога (эпик 12, «Бэкенд для интерфейса 4»): расследование
// как главный объект — связь инцидента с разбором, стадия, ступени области
// риска с поводом и разницей изделий, история уверенности гипотез, «что
// проверить следующим», качество данных дорожек. Всё — поля чтения к
// существующим ответам, совместимо (FR-29).

// IncidentLink — связь инцидента с разбором несоответствий (FR-59, FR-61).
type IncidentLink struct {
	NCIDs       []string `json:"nc_ids" doc:"Несоответствия инцидента: по ним — /nonconformities/{nc_id}/circumstances и /hypotheses."`
	GroupKey    *string  `json:"group_key,omitempty" doc:"Ключ группы /analysis/groups (вид дефекта × операция × оборудование) ведущего несоответствия."`
	PrimaryNCID *string  `json:"primary_nc_id,omitempty" doc:"Несоответствие с гипотезами — вход «Почему могло произойти»."`
}

// KnownCountsView — изделия текущей области по оси «что известно» (FR-62).
type KnownCountsView struct {
	Confirmed int `json:"confirmed" minimum:"0" doc:"Подтверждено."`
	Suspect   int `json:"suspect" minimum:"0" doc:"Под подозрением."`
	Unknown   int `json:"unknown" minimum:"0" doc:"Нет данных — исключать нельзя."`
	Excluded  int `json:"excluded" minimum:"0" doc:"Исключено с основанием."`
}

// InvestigationState — стадия расследования для шапки (словарь investigation_stage).
type InvestigationState struct {
	Stage       string          `json:"stage" enum:"scope_defined,hypothesis,cause_confirmed,action_assigned,effectiveness_check,closed" doc:"Стадия расследования: словарь investigation_stage (contracts/statuses.yaml)."`
	Counts      KnownCountsView `json:"counts" doc:"Изделия текущей версии области по оси «что известно»."`
	LastEventAt *time.Time      `json:"last_event_at,omitempty" doc:"Время последней записи по инциденту (версия области, гипотеза, причина, мера)."`
	NextStep    *string         `nullable:"true" json:"next_step" doc:"Что сделать следующим — текст сервера; null — делать нечего."`
	// CloseBlockers — почему расследование ещё нельзя закрыть (кнопка объясняет).
	CloseBlockers []CloseBlocker `json:"close_blockers" doc:"Почему расследование нельзя закрыть: те же коды, что вернёт POST /incidents/{id}/close со scope=investigation (422). Пусто — можно."`
}

// CloseBlocker — причина, по которой расследование нельзя закрыть.
type CloseBlocker struct {
	Code string `json:"code" doc:"Код отказа contracts/errors.yaml: incident.cause_branch_open, incident.effectiveness_unchecked, incident.investigation_closed."`
	Text string `json:"text" doc:"Почему словами."`
}

// ScopeTrigger — повод ступени области риска: что её вызвало.
type ScopeTrigger struct {
	Kind       string     `json:"kind" enum:"late_event,violation_window,human,computed" doc:"Опоздавшие данные, окно нарушения режима, решение человека, правило системы."`
	Label      string     `json:"label" doc:"Повод словами: «пришёл журнал «Сварочный источник ИС-2»: ток 176 А при уставке 160 ± 10 А»."`
	EventID    *string    `json:"event_id,omitempty" doc:"Запись-повод (для окна записи)."`
	ReceivedAt *time.Time `json:"received_at,omitempty" doc:"Когда данные пришли (ось «что мы знали»)."`
	OccurredAt *time.Time `json:"occurred_at,omitempty" doc:"Когда это произошло (ось «как было»)."`
}

// ScopeVersionDiff — что изменила ступень области и кто её подписал.
type ScopeVersionDiff struct {
	ItemsAdded   []string           `json:"items_added" doc:"Изделия, вошедшие в область этой версией (item_id); у первой версии — все."`
	ItemsRemoved []string           `json:"items_removed" doc:"Изделия, исключённые этой версией (item_id)."`
	Trigger      *ScopeTrigger      `json:"trigger,omitempty" doc:"Повод ступени — вместо разбора reason.code."`
	AuthorName   *string            `json:"author_name,omitempty" doc:"Имя автора версии из справочника людей; нет — правило системы или справочник недоступен."`
	SignedBy     *string            `json:"signed_by,omitempty" doc:"Кто подписал сужение (псевдоним)."`
	KeyClass     *string            `json:"key_class,omitempty" enum:"personal,device,server_attested,scenario" doc:"Класс ключа подписи сужения (AD-10)."`
	Evidence     []JournalRecordRef `json:"evidence" doc:"Доказательства версии словами (те же записи, что evidence_event_ids)."`
}

// HypothesisChange — что изменило уверенность гипотезы.
type HypothesisChange struct {
	At           time.Time `json:"at" doc:"Когда."`
	ConfidenceBP *int      `json:"confidence_bp,omitempty" minimum:"0" maximum:"10000" doc:"Уверенность после изменения; нет — не оценить."`
	EventID      *string   `json:"event_id,omitempty" doc:"Запись, изменившая уверенность."`
	Text         string    `json:"text" doc:"Что изменилось словами."`
}

// NextCheck — «что проверить следующим» (стол технолога, killer №2).
type NextCheck struct {
	Text            string `json:"text" doc:"Проверка словами."`
	MeasurementKind string `json:"measurement_kind" enum:"control_sample,radiography,camera_reshoot,equipment_log,sample_inspection,document_check,explanation,other" doc:"Вид проверки — для кнопки «Запросить измерение»."`
	UnlocksText     string `json:"unlocks_text" doc:"Что проверка разблокирует: подтверждение причины, сужение области."`
	CouldExclude    int    `json:"could_exclude" minimum:"0" doc:"Сколько изделий области проверка может исключить (оценка)."`
	ScopeSize       int    `json:"scope_size" minimum:"0" doc:"Размер текущей области риска."`
}

// DataGap — пропуск данных на дорожке.
type DataGap struct {
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	Source string    `json:"source" doc:"Источник (id)."`
	Text   string    `json:"text" doc:"Что пропало словами; пропуск — «исключать нельзя»."`
}

// LaneQuality — качество данных одной дорожки.
type LaneQuality struct {
	LateCount   int       `json:"late_count" minimum:"0" doc:"Записей, пришедших позже, чем произошли (опоздания)."`
	MaxDelayMin int       `json:"max_delay_min" minimum:"0" doc:"Наибольшее опоздание, минут."`
	Gaps        []DataGap `json:"gaps" doc:"Пропуски данных."`
}

// LaneQualities — качество данных трёх дорожек разбора.
type LaneQualities struct {
	Item      LaneQuality `json:"item"`
	Person    LaneQuality `json:"person"`
	Equipment LaneQuality `json:"equipment"`
}
