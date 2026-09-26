package journal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
)

// Чтение журнала событий в режиме live (эпик 16): journal.entry.list,
// journal.entry.read, journal.head.read над портом JournalStore. Ими
// пользуются экран журнала и пульт тестовых сценариев (привязка изделия,
// рождённого регистрацией, basis_seq демо-подписанта, «решение принято на
// столе роли», AD-26).

// ClockModeFunc — режим часов журнала (AD-37: system | scenario).
type ClockModeFunc func(ctx context.Context) (string, error)

// WithClockMode — источник режима часов для journal.head.read.
func WithClockMode(f ClockModeFunc) ServiceOption { return func(s *Service) { s.clockMode = f } }

func (s *Service) readable() bool { return s.store != nil }

// Entries — журнал событий (journal.entry.list): фильтры по изделию, потоку,
// типу (или префиксу семейства «item.»), виду записи и seq; в пределах
// прогона m.RunID (AD-38) и момента (AD-22).
func (s *Service) Entries(ctx context.Context, f EntryFilter, m platform.Moment, p platform.Page) (JournalEntryList, error) {
	if !s.readable() {
		return s.Unimplemented.Entries(ctx, f, m, p)
	}
	limit := p.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	desc := f.Order == OrderDesc
	cursor, _ := strconv.ParseInt(p.Cursor, 10, 64)
	q := ReadQuery{Stream: f.Stream, ItemID: f.ItemID, EventID: f.EventID, Limit: limit, Moment: m, RunID: m.RunID, Backward: desc}
	if desc {
		q.BeforeSeq = cursor
	} else {
		q.AfterSeq = max(f.AfterSeq, cursor)
	}
	prefix := strings.HasSuffix(f.EventType, ".")
	if f.EventType != "" && !prefix {
		q.EventType = f.EventType
	}
	out := JournalEntryList{Items: []JournalEntryView{}}
	last := int64(0)
	for len(out.Items) < limit {
		page, err := s.store.Read(ctx, q)
		if err != nil {
			return JournalEntryList{}, err
		}
		for _, e := range page {
			last = int64(e.Seq)
			if desc {
				q.BeforeSeq = last
			} else {
				q.AfterSeq = last
			}
			if !Matches(f, e.EventType, string(e.EntryKind), int64(e.Seq)) {
				continue
			}
			out.Items = append(out.Items, s.view(ctx, e))
			if len(out.Items) == limit {
				break
			}
		}
		if len(page) < q.Limit {
			return out, nil
		}
	}
	out.NextCursor = strconv.FormatInt(last, 10)
	return out, nil
}

// OrderDesc — журнал «новые сверху» (параметр order=desc).
const OrderDesc = "desc"

// Matches — запись проходит отбор журнала по типу (или префиксу семейства
// «quality.»), виду записи и after_seq (остальные условия — у чтения журнала).
func Matches(f EntryFilter, eventType, kind string, seq int64) bool {
	switch {
	case strings.HasSuffix(f.EventType, "."):
		if !strings.HasPrefix(eventType, f.EventType) {
			return false
		}
	case f.EventType != "" && eventType != f.EventType:
		return false
	}
	return (f.EntryKind == "" || kind == f.EntryKind) && seq > f.AfterSeq
}

// PageEntries — страница журнала из всех записей в памяти (заготовки): тот же
// отбор, порядок и курсор, что у live (Entries).
func PageEntries(all []JournalEntryView, f EntryFilter, p platform.Page) JournalEntryList {
	limit := p.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	desc := f.Order == OrderDesc
	cursor, _ := strconv.ParseInt(p.Cursor, 10, 64)
	slices.SortFunc(all, func(a, b JournalEntryView) int {
		if desc {
			return int(b.Seq - a.Seq)
		}
		return int(a.Seq - b.Seq)
	})
	out := JournalEntryList{Items: []JournalEntryView{}}
	for _, e := range all {
		if cursor > 0 && (desc && e.Seq >= cursor || !desc && e.Seq <= cursor) {
			continue
		}
		if !Matches(f, e.EventType, e.EntryKind, e.Seq) || f.EventID != "" && e.EventID != f.EventID ||
			f.Stream != "" && e.Stream != f.Stream || f.ItemID != "" && (e.ItemID == nil || *e.ItemID != f.ItemID) {
			continue
		}
		if len(out.Items) == limit {
			out.NextCursor = strconv.FormatInt(out.Items[len(out.Items)-1].Seq, 10)
			break
		}
		out.Items = append(out.Items, e)
	}
	return out
}

// Entry — запись основной цепочки по seq (journal.entry.read).
func (s *Service) Entry(ctx context.Context, seq int64) (JournalEntryView, error) {
	if !s.readable() {
		return s.Unimplemented.Entry(ctx, seq)
	}
	page, err := s.store.Read(ctx, ReadQuery{AfterSeq: seq - 1, Limit: 1})
	if err != nil {
		return JournalEntryView{}, err
	}
	if len(page) == 0 || int64(page[0].Seq) != seq {
		return JournalEntryView{}, platform.Fail(errcodes.ApiNotFound, "object", "Запись журнала", "id", strconv.FormatInt(seq, 10))
	}
	return s.view(ctx, page[0]), nil
}

// Head — голова журнала (journal.head.read): seq, recorded_at последней записи
// (доменное время, AD-37), номер CA и режим часов.
func (s *Service) Head(ctx context.Context, m platform.Moment) (JournalHead, error) {
	if !s.readable() {
		return s.Unimplemented.Head(ctx, m)
	}
	h, err := s.store.Head(ctx)
	if err != nil {
		return JournalHead{}, err
	}
	out := JournalHead{Seq: h.MainSeq, CASeq: h.CASeq, ClockMode: "system"}
	if last, err := s.store.Read(ctx, ReadQuery{Backward: true, Limit: 1}); err == nil && len(last) > 0 {
		if t, err := time.Parse(time.RFC3339Nano, last[0].RecordedAt); err == nil {
			t = t.UTC()
			out.RecordedAt = &t
		}
	}
	if s.clockMode != nil {
		if mode, err := s.clockMode(ctx); err == nil && mode != "" {
			out.ClockMode = mode
		}
	}
	return out, nil
}

// view — представление записи: открытые поля, подписанты и data из конверта.
func (s *Service) view(ctx context.Context, e jc.JournalEntry) JournalEntryView {
	v := JournalEntryView{Seq: int64(e.Seq), EventID: e.EventID, EventType: e.EventType, SchemaVersion: e.SchemaVersion,
		EntryKind: string(e.EntryKind), Chain: string(e.Chain), SourceID: e.SourceID, ProvenanceClass: string(e.ProvenanceClass),
		ItemID: e.ItemID, Stream: e.Stream, RunID: e.RunID, CorrelationID: e.CorrelationID, Signers: []string{},
		SignatureStatus: "not_checked"}
	v.OccurredAt = parseTS(e.OccurredAt)
	v.ReceivedAt = parseTS(e.ReceivedAt)
	v.RecordedAt = parseTS(e.RecordedAt)
	v.CommittedAt = parseTS(e.CommittedAt)
	if e.CausationID != nil {
		c := string(*e.CausationID)
		v.CausationID = &c
	}
	if e.BasisSeq != nil {
		b := int64(*e.BasisSeq)
		v.BasisSeq = &b
	}
	env, err := s.store.Open(ctx, e)
	if err != nil {
		return v
	}
	var dsse struct {
		Payload    string            `json:"payload"`
		Signatures []json.RawMessage `json:"signatures"`
	}
	if json.Unmarshal(env.Raw, &dsse) != nil {
		return v
	}
	payload, err := base64.StdEncoding.DecodeString(dsse.Payload)
	if err != nil {
		return v
	}
	var ev struct {
		SourceKind string         `json:"source_kind"`
		Corrects   string         `json:"corrects_event"`
		Data       map[string]any `json:"data"`
		Integrity  struct {
			Signers []string `json:"signers"`
		} `json:"integrity"`
	}
	if json.Unmarshal(payload, &ev) != nil {
		return v
	}
	v.Data = ev.Data
	if ev.SourceKind != "" {
		v.SourceKind = &ev.SourceKind
	}
	if ev.Corrects != "" {
		v.Corrects = &ev.Corrects
	}
	if len(ev.Integrity.Signers) > 0 {
		v.Signers = ev.Integrity.Signers
	}
	return v
}

func parseTS(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t.UTC()
}
