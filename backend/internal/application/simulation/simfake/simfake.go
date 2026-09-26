// Пакет simfake — заготовки ведомых портов модуля simulation для автосверки без
// движка (make sim-check): приём в памяти (дедупликация по источнику и номеру
// события, конфликт содержимого, карантин, разрывы номеров, сдвиг часов), его
// чтение (ingest.metrics.read, ingest.source.list, ingest.quarantine.list),
// регистрация изделий, журнал служебных записей прогона и часы.
//
// Это не макет системы: заготовка отвечает только за то, что делает приём, —
// остальные операции честно «пока не отвечают», и строки табло по ним ждут
// движка (AD-26: сверка идёт теми же операциями, что и в живой системе).
//
// Слой: application (заготовки портов для тестов и make sim-check).
package simfake

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"

	app "ant/internal/application/simulation"
	sim "ant/internal/domain/simulation"
)

// Ingest — приём в памяти: Gateway и Probe для операций приёма.
type Ingest struct {
	mu        sync.Mutex
	seen      map[string]string // источник|событие → отпечаток содержимого
	received  int
	accepted  int
	dups      int
	rejected  int
	quar      []quarantine
	sources   map[string]*source
	items     map[string]string // item_id по номеру регистрации
	itemSeq   int64
	journal   map[int64]map[string]any
	decisions map[string]int
}

type quarantine struct {
	ID, SourceID, EventID, Code string
	At                          time.Time
}

type source struct {
	seqs  map[int64]bool
	skews []int64
	last  time.Time
}

// NewIngest — пустой приём.
func NewIngest() *Ingest {
	return &Ingest{seen: map[string]string{}, sources: map[string]*source{}, items: map[string]string{},
		journal: map[int64]map[string]any{}, decisions: map[string]int{}}
}

var _ app.Gateway = (*Ingest)(nil)

// Deliver — приём событий источника (AD-7: ключ идемпотентности — источник и event_id).
func (g *Ingest) Deliver(_ context.Context, runID string, batch []sim.Emission) ([]app.Delivered, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]app.Delivered, 0, len(batch))
	for _, e := range batch {
		g.received++
		var ev map[string]any
		dec := json.NewDecoder(bytes.NewReader(e.Event))
		dec.UseNumber()
		if err := dec.Decode(&ev); err != nil {
			return nil, err
		}
		src := g.sources[e.SourceID]
		if src == nil {
			src = &source{seqs: map[int64]bool{}}
			g.sources[e.SourceID] = src
		}
		src.last = e.DeliverAt
		key := e.SourceID + "|" + e.EventID
		fp := string(e.Event)
		if prev, ok := g.seen[key]; ok {
			if prev == fp {
				g.dups++
				out = append(out, app.Delivered{EventID: e.EventID, Status: app.DeliveryDuplicate})
				continue
			}
			g.quar = append(g.quar, quarantine{ID: fmt.Sprintf("q-%d", len(g.quar)+1), SourceID: e.SourceID, EventID: e.EventID, Code: "ingest.duplicate_conflict", At: e.DeliverAt})
			out = append(out, app.Delivered{EventID: e.EventID, Status: app.DeliveryQuarantined, Code: "ingest.duplicate_conflict"})
			continue
		}
		// номер источника занят и сообщением в карантине: дыра — только если
		// номера нет ни в журнале, ни в карантине (AD-7, AD-9)
		src.seqs[e.SourceSeq] = true
		if e.Quarantine {
			code := "ingest.unknown_enum_value_critical"
			switch {
			case ev["occurred_at"] == nil:
				code = "ingest.missing_required_field"
			case fmt.Sprint(ev["schema_version"]) != "1" && fmt.Sprint(ev["schema_version"]) != "2":
				code = "ingest.unknown_schema_version"
			}
			g.quar = append(g.quar, quarantine{ID: fmt.Sprintf("q-%d", len(g.quar)+1), SourceID: e.SourceID, EventID: e.EventID, Code: code, At: e.DeliverAt})
			out = append(out, app.Delivered{EventID: e.EventID, Status: app.DeliveryQuarantined, Code: code})
			continue
		}
		g.seen[key] = fp
		g.accepted++
		if occ, ok := ev["occurred_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, occ); err == nil && t.After(e.DeliverAt) {
				src.skews = append(src.skews, t.Sub(e.DeliverAt).Milliseconds())
			}
		}
		out = append(out, app.Delivered{EventID: e.EventID, Status: app.DeliveryAccepted})
	}
	return out, nil
}

// Read — чтение операций приёма; остальное — «пока не отвечает».
func (g *Ingest) Read(_ context.Context, op string, params map[string]string, runID string) (any, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var v any
	switch op {
	case "ingest.metrics.read":
		completeness := int64(10000)
		exp, have := int64(0), int64(0)
		for _, s := range g.sources {
			lo, hi := bounds(s.seqs)
			exp += hi - lo + 1
			have += int64(len(s.seqs))
		}
		if exp > 0 {
			completeness = have * 10000 / exp
		}
		v = map[string]any{"received": g.received, "accepted": g.accepted, "duplicates": g.dups, "rejected": g.rejected,
			"quarantined": len(g.quar), "quarantine_open": len(g.quar), "completeness_bp": completeness}
	case "ingest.source.list":
		var items []any
		ids := make([]string, 0, len(g.sources))
		for id := range g.sources {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		for _, id := range ids {
			s := g.sources[id]
			lo, hi := bounds(s.seqs)
			item := map[string]any{"source_id": id, "last_seq": hi, "gap_count": hi - lo + 1 - int64(len(s.seqs)), "state": "active",
				"last_received_at": s.last.UTC().Format(sim.TimeLayout), "quarantined": 0}
			if len(s.skews) > 0 {
				sk := slices.Clone(s.skews)
				sort.Slice(sk, func(i, j int) bool { return sk[i] < sk[j] })
				item["clock_skew_ms"] = sk[len(sk)/2]
			}
			for _, q := range g.quar {
				if q.SourceID == id {
					item["quarantined"] = item["quarantined"].(int) + 1
				}
			}
			items = append(items, item)
		}
		v = map[string]any{"items": items}
	case "ingest.quarantine.list":
		var items []any
		for _, q := range g.quar {
			if src := params["source_id"]; src != "" && q.SourceID != src {
				continue
			}
			if code := params["code"]; code != "" && q.Code != code {
				continue
			}
			items = append(items, map[string]any{"quarantine_id": q.ID, "source_id": q.SourceID, "event_id": q.EventID, "problem_code": q.Code,
				"state": "open", "quarantined_at": q.At.UTC().Format(sim.TimeLayout)})
		}
		v = map[string]any{"items": items}
	case "journal.entry.read":
		var seq int64
		fmt.Sscan(params["seq"], &seq)
		e, ok := g.journal[seq]
		if !ok {
			return nil, app.ErrNotFound
		}
		v = e
	default:
		return nil, app.ErrUnavailable
	}
	// ответ как у клиента: JSON с числами json.Number
	b, _ := json.Marshal(v)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var doc any
	err := dec.Decode(&doc)
	return doc, err
}

func bounds(m map[int64]bool) (int64, int64) {
	var lo, hi int64
	first := true
	for k := range m {
		if first || k < lo {
			lo = k
		}
		if first || k > hi {
			hi = k
		}
		first = false
	}
	if first {
		return 1, 0
	}
	return lo, hi
}

// Actor — решения людей: регистрация изделий и нанесение носителя принимаются
// (без них не узнать изделия прогона), прочие операции «пока не отвечают» —
// их исполнит настоящий API с движком.
type Actor struct{ In *Ingest }

var _ app.Actor = Actor{}

// Act — команда от имени персоны.
func (a Actor) Act(_ context.Context, persona, op string, params map[string]string, body map[string]any) (app.ActResult, error) {
	g := a.In
	g.mu.Lock()
	defer g.mu.Unlock()
	g.decisions[op]++
	switch op {
	case "item.item.register":
		g.itemSeq++
		seq := 1_000_000 + g.itemSeq
		id := fmt.Sprintf("ENT01:fake/%d", g.itemSeq)
		g.journal[seq] = map[string]any{"seq": seq, "item_id": id, "event_type": "item.item.registered"}
		return app.ActResult{Seq: seq, Status: 200}, nil
	case "item.carrier.apply":
		return app.ActResult{Seq: 1, Status: 200}, nil
	}
	return app.ActResult{}, app.ErrUnavailable
}

// Decided — в заготовке решение на столе не принимается само.
func (a Actor) Decided(context.Context, string, string, int64) (bool, int64, error) {
	return false, 0, nil
}

// Recorder — служебные записи прогона в памяти; проверяет, что доменное время
// записей не убывает (AD-37).
type Recorder struct {
	mu      sync.Mutex
	Records []app.Record
	last    time.Time
}

// Record — запись.
func (r *Recorder) Record(_ context.Context, rec app.Record) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec.OccurredAt.Before(r.last) {
		return 0, fmt.Errorf("recorded_at убывает: %s после %s (AD-37)", rec.OccurredAt, r.last)
	}
	r.last = rec.OccurredAt
	r.Records = append(r.Records, rec)
	return int64(len(r.Records)), nil
}

// Clock — часы: реальное время двигается вручную.
type Clock struct {
	mu sync.Mutex
	T  time.Time
}

// Now — текущее время.
func (c *Clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.T }

// Advance — сдвинуть часы.
func (c *Clock) Advance(d time.Duration) { c.mu.Lock(); c.T = c.T.Add(d); c.mu.Unlock() }
