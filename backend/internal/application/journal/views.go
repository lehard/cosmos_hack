package journal

import (
	"time"

	"ant/internal/application/platform"
)

// JournalEntryView — запись журнала для общего экрана «Журнал событий» (PRD §3a):
// открытые поля записи (AD-44) и содержимое data, если его можно показать
// пользователю (расшифровано, AD-23). Вид записи различим: факт / реакция /
// решение / служебная (AD-2, кейс §7.2).
type JournalEntryView struct {
	Seq             int64          `json:"seq" minimum:"1"`
	EventID         string         `json:"event_id"`
	EventType       string         `json:"event_type" doc:"Тип записи каталога."`
	SchemaVersion   int            `json:"schema_version" minimum:"1"`
	EntryKind       string         `json:"entry_kind" enum:"fact,reaction,decision,service"`
	Chain           string         `json:"chain" enum:"main,ca"`
	SourceID        string         `json:"source_id"`
	SourceKind      *string        `json:"source_kind,omitempty" doc:"Вид источника факта (FR-140)."`
	ProvenanceClass string         `json:"provenance_class" doc:"Класс происхождения подписи (AD-2)."`
	ItemID          *string        `json:"item_id,omitempty"`
	Stream          string         `json:"stream"`
	RunID           *string        `json:"run_id,omitempty"`
	OccurredAt      time.Time      `json:"occurred_at"`
	ReceivedAt      time.Time      `json:"received_at"`
	RecordedAt      time.Time      `json:"recorded_at"`
	CommittedAt     time.Time      `json:"committed_at"`
	CorrelationID   string         `json:"correlation_id"`
	CausationID     *string        `nullable:"true" json:"causation_id"`
	Corrects        *string        `json:"corrects,omitempty" doc:"Исправляемая запись (FR-122)."`
	BasisSeq        *int64         `json:"basis_seq,omitempty"`
	CARef           *string        `json:"ca_ref,omitempty" doc:"Критическое действие CA-‹n› (AD-28)."`
	Signers         []string       `json:"signers" doc:"key_id@версия подписантов."`
	SignatureStatus string         `json:"signature_status" enum:"valid,invalid,unverifiable,not_checked" doc:"Статус проверки подписи (FR-68); not_checked — профиль demo до эпика 05."`
	Data            map[string]any `json:"data,omitempty" doc:"Содержимое data; нет — содержимое недоступно (нет KEK или прав)."`
}

// JournalEntryList — страница журнала событий.
type JournalEntryList struct {
	Items      []JournalEntryView `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// JournalHead — голова журнала: позиция и время последней записи (AD-37),
// головы цепочек для индикаторов.
type JournalHead struct {
	Seq        int64      `json:"seq" minimum:"0"`
	RecordedAt *time.Time `nullable:"true" json:"recorded_at" doc:"Доменное время последней записи; null — журнал пуст."`
	CASeq      int64      `json:"ca_seq" minimum:"0" doc:"Последний номер CA-‹n›."`
	ClockMode  string     `json:"clock_mode" enum:"system,scenario" doc:"Режим часов журнала (AD-37)."`
}

// TimelineMark — значимое событие на таймлайне (FR-4): эскалация, стоп точки
// процесса, всплеск, новая версия процесса.
type TimelineMark struct {
	MarkID string             `json:"mark_id"`
	At     time.Time          `json:"at" doc:"Момент события на выбранной оси."`
	Kind   string             `json:"kind" enum:"escalation,process_stop,spike,revision"`
	Title  *string            `json:"title,omitempty" doc:"Короткая подпись (узел, изделие)."`
	Ref    *platform.DrillRef `json:"ref,omitempty" doc:"Куда провалиться (FR-7)."`
}

// TimelineData — таймлайн под живой картой: диапазон истории и метки (FR-4, FR-155).
type TimelineData struct {
	From  time.Time      `json:"from" doc:"Начало доступной истории (или прогона)."`
	To    time.Time      `json:"to" doc:"Конец — «сейчас» на сервере."`
	Marks []TimelineMark `json:"marks"`
}

// EntryFilter — фильтр журнала событий.
type EntryFilter struct {
	ItemID    string
	Stream    string
	EventType string
	AfterSeq  int64
	EntryKind string
}
