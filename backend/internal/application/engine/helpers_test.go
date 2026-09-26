package engine_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	"ant/internal/domain/engine/enginetest"
)

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

// world — журнал в памяти, воркер с пустышкой и одна партиция.
type world struct {
	t      *testing.T
	j      *enginemem.Journal
	feed   *enginemem.Feed
	codec  *engineapp.Codec
	worker *engineapp.WorkerService
	part   engineapp.Partition
	tel    *telemetry
	n      int
}

func newWorld(t *testing.T, fold engine.Folder) *world {
	t.Helper()
	if fold == nil {
		fold = enginetest.Fold
	}
	j := enginemem.New(nil)
	part := engineapp.Partition{Number: 0, Epoch: 1}
	j.SetEpoch(appjournal.PartitionLease(0), 1)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}
	tel := &telemetry{}
	feed := &enginemem.Feed{J: j, Parts: []engineapp.Partition{part}}
	w := engineapp.NewWorker(engineapp.WorkerConfig{Feed: feed, Codec: codec, Fold: fold, Telemetry: tel, Refresh: 50 * time.Millisecond, Backoff: 10 * time.Millisecond})
	return &world{t: t, j: j, feed: feed, codec: codec, worker: w, part: part, tel: tel}
}

// fact пишет результат контроля изделия (как это сделал бы приём, эпик 06).
func (w *world) fact(item, id string, occurred time.Duration, outcome, corrects string) {
	w.t.Helper()
	w.n++
	p, err := w.codec.Encode(context.Background(), engineapp.Out{
		EventID: id, Type: catalog.InspectionResultRecorded, Kind: catalog.KindFact, Stream: "item:" + item, ItemID: item,
		OccurredAt: t0.Add(occurred), Correlation: id, Data: map[string]string{"outcome": outcome},
	})
	if err != nil {
		w.t.Fatal(err)
	}
	if corrects != "" {
		p.Envelope = patchEnvelope(w.t, p.Envelope, func(m map[string]any) {
			m["corrects"] = map[string]any{"event_id": corrects, "reason": map[string]any{"text": "исправление"}}
		})
	}
	if _, err := w.j.Append(context.Background(), appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
		w.t.Fatal(err)
	}
}

func patchEnvelope(t *testing.T, raw []byte, f func(map[string]any)) []byte {
	t.Helper()
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	payload, _ := base64.StdEncoding.DecodeString(d["payload"].(string))
	var env map[string]any
	if err := json.Unmarshal(payload, &env); err != nil {
		t.Fatal(err)
	}
	f(env)
	b, _ := json.Marshal(env)
	d["payload"] = base64.StdEncoding.EncodeToString(b)
	out, _ := json.Marshal(d)
	return out
}

// step — одна итерация воркера по партиции: подача работ и обработка.
func (w *world) step() {
	w.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	works, err := w.feed.Next(ctx, w.part)
	if err != nil {
		w.t.Fatalf("нет работы: %v", err)
	}
	if err := w.worker.Process(context.Background(), w.part, works); err != nil {
		w.t.Fatal(err)
	}
}

// of — записи журнала типа t.
func (w *world) of(t catalog.Type) []jc.JournalEntry {
	var out []jc.JournalEntry
	for _, e := range w.j.Entries() {
		if e.EventType == string(t) {
			out = append(out, e)
		}
	}
	return out
}

// reaction — метаданные реакции из конверта записи.
func (w *world) reaction(e jc.JournalEntry) engineapp.ReactionMeta {
	w.t.Helper()
	d, err := w.codec.Decode(context.Background(), e)
	if err != nil || d.Reaction == nil {
		w.t.Fatalf("конверт реакции seq %d: %v", e.Seq, err)
	}
	return *d.Reaction
}

// telemetry — телеметрия для проверки метрик.
type telemetry struct {
	mu       sync.Mutex
	observed map[string][]time.Duration
	counters map[string]int64
}

func (t *telemetry) Counter(name string, d int64, _ ...string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.counters == nil {
		t.counters = map[string]int64{}
	}
	t.counters[name] += d
}

func (t *telemetry) Observe(name string, v time.Duration, _ ...string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.observed == nil {
		t.observed = map[string][]time.Duration{}
	}
	t.observed[name] = append(t.observed[name], v)
}

func (t *telemetry) Gauge(string, int64, ...string) {}

func (t *telemetry) values(name string) []time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]time.Duration(nil), t.observed[name]...)
}
