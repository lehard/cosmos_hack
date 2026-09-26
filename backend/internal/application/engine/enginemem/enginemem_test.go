package enginemem

// Проверки AD-39 фейка журнала — те же случаи и коды, что у адаптера Postgres
// (TestConcurrencyChecks в infrastructure/storage/journal): устаревший
// basis_seq, необработанный вход изделия, устаревшая политика, лимит
// разрешения на отклонение, повтор event_id, откат пачки при отказе.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	appjournal "ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
)

var nextID int

func entry(eventType string, kind jc.JournalEntryEntryKind, stream, item string) appjournal.Pending {
	nextID++
	e := jc.JournalEntry{EntryKind: kind, EventType: eventType, SchemaVersion: 1,
		EventID:  fmt.Sprintf("01900000-0000-7000-8000-%012d", nextID),
		SourceID: "test-source", Stream: stream, OccurredAt: "2026-09-26T08:00:00.000Z"}
	if item != "" {
		it := item
		e.ItemID = &it
		e.Partition = kernel.PartitionOf(item, 4)
	}
	return appjournal.Pending{Entry: e, Envelope: []byte(`{}`)}
}

func TestAppendConcurrencyChecks(t *testing.T) {
	ctx := context.Background()
	j := New(func() time.Time { return time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC) })
	item := "ENT01:K-1"
	stream := dj.ItemStream(item)
	fact := func() appjournal.Pending {
		return entry("inspection.result.recorded", jc.JournalEntryEntryKindFact, stream, item)
	}
	decision := func() appjournal.Pending {
		return entry("decision.disposition.set", jc.JournalEntryEntryKindDecision, stream, item)
	}
	r1, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{fact()}})
	if err != nil {
		t.Fatal(err)
	}
	basis := r1.Seqs[0]
	// Реакция воркера не guard_relevant — версия потока не меняется.
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{entry("decision.nonconformity.drafted", jc.JournalEntryEntryKindReaction, stream, item)}}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{decision()}, Checks: []appjournal.Check{{Stream: stream, BasisSeq: basis}}}); err != nil {
		t.Fatalf("гард на актуальном basis_seq: %v", err)
	}
	// Решение на старом basis_seq — после него уже есть guard_relevant запись.
	before := len(j.Entries())
	_, err = j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{decision()}, Checks: []appjournal.Check{{Stream: stream, BasisSeq: basis}}})
	if !errors.Is(err, appjournal.ErrStaleState) {
		t.Fatalf("устаревший basis_seq: %v", err)
	}
	if pe, ok := appjournal.Problem(err); !ok || pe.Code != "journal.stale_state" || pe.Params["stream"] != stream || pe.BasisSeq != basis {
		t.Fatalf("problem: %+v", pe)
	}
	if n := len(j.Entries()); n != before {
		t.Fatalf("отказ проверки записал %d записей", n-before)
	}
	// Необработанный вход изделия: курсора воркера нет — вход не обработан.
	h, _ := j.Head(ctx)
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{decision()}, Checks: []appjournal.Check{{Stream: stream, BasisSeq: h.MainSeq, ItemProcessed: true}}}); !errors.Is(err, appjournal.ErrStaleState) {
		t.Fatalf("необработанный вход: %v", err)
	}
	part := kernel.PartitionOf(item, 4)
	if _, err := j.Append(ctx, appjournal.AppendRequest{Consumer: &appjournal.CursorAdvance{Name: appjournal.WorkerConsumer, Partition: part, Seq: h.MainSeq}}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{decision()}, Checks: []appjournal.Check{{Stream: stream, BasisSeq: h.MainSeq, ItemProcessed: true}}}); err != nil {
		t.Fatalf("вход обработан: %v", err)
	}
	// Политика изменилась после policy_seq.
	pr, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{entry("policy.role.assigned", jc.JournalEntryEntryKindDecision, "policy:global", "")}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{decision()}, Checks: []appjournal.Check{{PolicyStream: "policy:global", PolicySeq: pr.Seqs[0] - 1}}})
	if pe, ok := appjournal.Problem(err); !errors.Is(err, appjournal.ErrStalePolicy) || !ok || pe.Code != "journal.stale_policy" {
		t.Fatalf("политика: %v", err)
	}
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{decision()}, Checks: []appjournal.Check{{PolicyStream: "policy:global", PolicySeq: pr.Seqs[0]}}}); err != nil {
		t.Fatalf("политика не менялась: %v", err)
	}
	// Лимит разрешения: 2 изделия; третий расход — отказ, расход атомарен с решением.
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{decision()}, Checks: []appjournal.Check{{ConcessionID: "CON-1", Consume: 1}}}); !errors.Is(err, appjournal.ErrConcessionExhausted) {
		t.Fatalf("расход без открытого лимита: %v", err)
	}
	if _, err := j.Append(ctx, appjournal.AppendRequest{ConcessionGrants: []appjournal.ConcessionGrant{{ConcessionID: "CON-1", Limit: 0}}}); !errors.Is(err, appjournal.ErrInvalidEntry) {
		t.Fatalf("нулевой лимит: %v", err)
	}
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{decision()}, ConcessionGrants: []appjournal.ConcessionGrant{{ConcessionID: "CON-1", Limit: 2}}}); err != nil {
		t.Fatal(err)
	}
	var okN, exhaustedN int
	for range 3 {
		_, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{decision()}, Checks: []appjournal.Check{{ConcessionID: "CON-1", Consume: 1}}})
		switch {
		case err == nil:
			okN++
		case errors.Is(err, appjournal.ErrConcessionExhausted):
			exhaustedN++
			pe, _ := appjournal.Problem(err)
			if pe.Code != "journal.concession_exhausted" || pe.Params["remaining"] != "0" || pe.Params["limit"] != "2" {
				t.Fatalf("problem: %+v", pe)
			}
		default:
			t.Fatal(err)
		}
	}
	if okN != 2 || exhaustedN != 1 {
		t.Fatalf("расход лимита: принято %d, отказов %d (ожидалось 2 и 1)", okN, exhaustedN)
	}
	// Повтор event_id — journal.duplicate; повтор внутри одной пачки — тоже.
	p := fact()
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); !errors.Is(err, appjournal.ErrDuplicate) {
		t.Fatalf("повтор event_id: %v", err)
	} else if pe, ok := appjournal.Problem(err); !ok || pe.Code != "journal.duplicate" {
		t.Fatalf("problem повтора: %+v", pe)
	}
	q := fact()
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{q, q}}); !errors.Is(err, appjournal.ErrDuplicate) {
		t.Fatalf("повтор в пачке: %v", err)
	}
}
