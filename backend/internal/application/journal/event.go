package journal

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
)

// Одна запись журнала по event_id для окна ?open=event:‹id› (интерфейс 6,
// стол технолога: доказательства ступеней области, доводы гипотез, отметки
// дорожек; Д-70): те же поля, что у ссылки на запись в разборе (текст для
// людей, источник словами), плюс позиция в журнале, материалы и числа режима.

// JournalEventReading — параметр режима числами (AD-4: целое с масштабом):
// значение = число × 10^(−scale) в единице unit.
type JournalEventReading struct {
	Parameter       string `json:"parameter" doc:"Параметр у источника (current_a — ток сварки, …)."`
	Unit            string `json:"unit" doc:"Единица, код UCUM (A, V, mm, …)."`
	Scale           int    `json:"scale" minimum:"0" maximum:"12" doc:"Знаков после запятой: значение = число × 10^(−scale)."`
	SetpointNominal *int64 `json:"setpoint_nominal,omitempty" doc:"Уставка: номинал."`
	SetpointMin     *int64 `json:"setpoint_min,omitempty" doc:"Уставка: нижняя граница."`
	SetpointMax     *int64 `json:"setpoint_max,omitempty" doc:"Уставка: верхняя граница."`
	ObservedMin     *int64 `json:"observed_min,omitempty" doc:"Наблюдённый минимум (у отклонения — значение)."`
	ObservedMax     *int64 `json:"observed_max,omitempty" doc:"Наблюдённый максимум (у отклонения — значение)."`
}

// JournalEventView — запись журнала для окна записи (journal.event.read):
// поля ссылки на запись разбора (event_id, event_type, variant, occurred_at,
// params, text, source_label) и позиция, вид, материалы, числа режима.
type JournalEventView struct {
	EventID     string            `json:"event_id" doc:"event_id записи."`
	EventType   string            `json:"event_type" doc:"Тип записи каталога."`
	EventTitle  string            `json:"event_title,omitempty" doc:"Тип записи словами (каталог типов)."`
	Variant     *string           `json:"variant,omitempty" doc:"Уточнение внутри типа: outcome контроля, вид отклонения…"`
	OccurredAt  time.Time         `json:"occurred_at" doc:"Время возникновения (AD-37)."`
	RecordedAt  time.Time         `json:"recorded_at" doc:"Время записи в журнал; позже возникновения — запись опоздала."`
	Late        bool              `json:"late,omitempty" doc:"Запись пришла с опозданием (задержка записи)."`
	Params      map[string]string `json:"params,omitempty" doc:"Параметры записи: метод, параметр, значение, уставка, шаг…"`
	Text        *string           `json:"text,omitempty" doc:"Запись словами для людей: что произошло, значение против уставки, источник."`
	SourceLabel *string           `json:"source_label,omitempty" doc:"Источник словами: журнал оборудования, камера, человек."`
	Kind        string            `json:"kind" enum:"fact,reaction,decision,service" doc:"Вид записи (AD-2): исходный факт, вывод системы, решение человека, служебная."`
	JournalSeq  int64             `json:"journal_seq" minimum:"1" doc:"Позиция записи в основной цепочке — переход к журналу (journal.entry.read)."`
	SourceKind  *string           `json:"source_kind,omitempty" doc:"Вид источника факта (FR-140)."`
	Author      *string           `json:"author,omitempty" doc:"Автор решения (псевдоним)."`
	ItemID      *string           `json:"item_id,omitempty" doc:"Изделие записи."`
	Stream      string            `json:"stream" doc:"Поток записи."`
	Signers     []string          `json:"signers" doc:"key_id@версия подписантов."`
	// EvidenceRefs — адреса материалов наблюдения (кадры, протоколы).
	EvidenceRefs []string `json:"evidence_refs" doc:"Адреса материалов: кадры, протоколы (материалы — materials.material.read)."`
	// Reading — числа режима (отклонение режима, сводка цикла).
	Reading *JournalEventReading `json:"reading,omitempty" doc:"Параметр режима числами: уставка и наблюдённые значения."`
}

// Event — запись по event_id (journal.event.read): live — основная цепочка
// журнала; текст — тип записи словами и её параметры.
func (s *Service) Event(ctx context.Context, eventID string, m platform.Moment) (JournalEventView, error) {
	if !s.readable() {
		return s.Unimplemented.Event(ctx, eventID, m)
	}
	es, err := s.store.Read(ctx, ReadQuery{EventID: eventID, Limit: 1, Moment: m})
	if err != nil {
		return JournalEventView{}, err
	}
	if len(es) == 0 {
		return JournalEventView{}, platform.Fail(errcodes.ApiNotFound, "object", "Запись журнала", "id", eventID)
	}
	e := s.view(ctx, es[0])
	return EventFromEntry(e), nil
}

// EventFromEntry — окно записи из представления записи журнала: параметры —
// простые поля data, материалы — поля *_refs, числа режима — reading.
func EventFromEntry(e JournalEntryView) JournalEventView {
	v := JournalEventView{EventID: e.EventID, EventType: e.EventType, OccurredAt: e.OccurredAt, RecordedAt: e.RecordedAt,
		Kind: e.EntryKind, JournalSeq: e.Seq, SourceKind: e.SourceKind, ItemID: e.ItemID, Stream: e.Stream,
		Signers: e.Signers, EvidenceRefs: []string{}, Params: map[string]string{}}
	if v.Signers == nil {
		v.Signers = []string{}
	}
	v.Late = e.RecordedAt.Sub(e.OccurredAt) > 15*time.Minute
	if info, ok := catalog.Lookup(catalog.Type(e.EventType)); ok {
		v.EventTitle = info.Title
	}
	for k, x := range e.Data {
		switch t := x.(type) {
		case string:
			v.Params[k] = t
		case float64, bool, int, int64:
			v.Params[k] = fmt.Sprint(t)
		case []any:
			if strings.HasSuffix(k, "_refs") || k == "evidence" {
				for _, r := range t {
					if s, ok := r.(string); ok {
						v.EvidenceRefs = append(v.EvidenceRefs, s)
					}
				}
			}
		}
	}
	for _, k := range []string{"outcome", "deviation_kind", "condition", "result", "verdict"} {
		if x := v.Params[k]; x != "" {
			v.Variant = &x
			break
		}
	}
	if len(v.Params) == 0 {
		v.Params = nil
	}
	text := v.EventTitle
	if text == "" {
		text = e.EventType
	}
	if sum := v.Params["summary"]; sum != "" {
		text = sum
	} else {
		var kv []string
		keys := make([]string, 0, len(v.Params))
		for k := range v.Params {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			if len(kv) < 4 && !strings.HasSuffix(k, "_id") {
				kv = append(kv, k+" "+v.Params[k])
			}
		}
		if len(kv) > 0 {
			text += ": " + strings.Join(kv, ", ")
		}
	}
	v.Text = &text
	src := e.SourceID
	if len(e.Signers) > 0 && e.EntryKind == "decision" {
		src, _, _ = strings.Cut(e.Signers[0], "@")
		v.Author = &src
	}
	if src == "ant" || src == "" {
		src = "система"
	}
	v.SourceLabel = &src
	return v
}
