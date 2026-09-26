package security

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"testing"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/ingest"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
)

// memJournal — журнал в памяти для шины: записи с конвертами.
type memJournal struct {
	appjournal.JournalStore
	mu  sync.Mutex
	es  []jc.JournalEntry
	env map[string][]byte
}

func (m *memJournal) Append(_ context.Context, rq appjournal.AppendRequest) (appjournal.AppendResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res appjournal.AppendResult
	for _, p := range rq.Batch {
		e := p.Entry
		e.Seq = len(m.es) + 1
		m.es = append(m.es, e)
		if m.env == nil {
			m.env = map[string][]byte{}
		}
		m.env[e.EventID] = p.Envelope
		res.Seqs = append(res.Seqs, int64(e.Seq))
	}
	return res, nil
}

func (m *memJournal) Open(_ context.Context, e jc.JournalEntry) (appjournal.Envelope, error) {
	return appjournal.Envelope{Raw: m.env[e.EventID]}, nil
}

// replay — потребитель: отдаёт весь журнал в handle один раз на имя курсора.
type replay struct {
	j     *memJournal
	mu    sync.Mutex
	names []string
}

func (r *replay) Consume(ctx context.Context, name string, _ appjournal.Scope, handle func(context.Context, []jc.JournalEntry) (appjournal.AppendRequest, error)) error {
	r.mu.Lock()
	r.names = append(r.names, name)
	r.mu.Unlock()
	_, err := handle(ctx, r.j.es)
	return err
}

type collect struct {
	name string
	mu   sync.Mutex
	got  []SecurityEvent
}

func (c *collect) Name() string { return c.name }
func (c *collect) Deliver(_ context.Context, evs []SecurityEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.got = append(c.got, evs...)
	return nil
}

// FR-118, AD-24: новый подписчик шины подключается без изменения источников:
// источник (приём) пишет события как раньше, подписчик со своим курсором
// получает все события безопасности с начала журнала; прочие записи и журнал
// критических действий шиной не идут.
func TestNewSubscriberWithoutSourceChanges(t *testing.T) {
	j := &memJournal{}
	ctx := context.Background()
	enc := Encoder{DomainBuild: dj.ZeroLink.String(), Now: func() time.Time { return time.Unix(1_790_000_000, 0) }}
	src := IngestBus{Enc: enc}
	m, _, err := src.SignatureInvalid(ctx, ingest.SignatureFailure{SourceID: "edge-ws2", Failure: "bad_signature", OccurredAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	c, _, err := src.IdempotencyConflict(ctx, ingest.IdempotencyConflict{SourceID: "edge-ws2", EventID: "0190a8c4-1111-7000-8000-000000000001",
		FirstDigest: dj.ZeroLink.String(), ConflictDigest: dj.ZeroLink.String(), OccurredAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	other := m[0]
	other.Entry.EventType, other.Entry.EventID = string(catalog.InspectionResultRecorded), "0190a8c4-1111-7000-8000-000000000009"
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: append(append(m, other), c...)}); err != nil {
		t.Fatal(err)
	}
	emit := Emitter{Journal: j, Enc: enc}
	if _, err := emit.Emit(ctx, Out{Type: catalog.SecurityIntegrityViolated, OccurredAt: time.Now(),
		Data: map[string]any{"violation": "projection_mismatch", "detail": "проекция расходится с журналом", "ca_ref": "CA-3"}}); err != nil {
		t.Fatal(err)
	}
	first := &collect{name: "first"}
	rp := &replay{j: j}
	if err := (&Bus{Consumer: rp, Journal: j, Subscribers: []Subscriber{first}}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	// Новый подписчик — строкой сборки, источники те же.
	second := &collect{name: "siem"}
	if err := (&Bus{Consumer: rp, Journal: j, Subscribers: []Subscriber{first, second}}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(second.got) != 3 || second.got[0].EventType != string(catalog.SecuritySignatureInvalid) || second.got[2].CARef != "CA-3" ||
		second.got[1].Severity != "alarm" {
		b, _ := json.Marshal(second.got)
		t.Fatalf("новый подписчик: %s", b)
	}
	if !slices.Contains(rp.names, ConsumerName("siem")) || ConsumerName("siem") == ConsumerName("first") {
		t.Fatalf("курсоры: %v", rp.names)
	}
}
