// Пакет enginemem — фейки в памяти ведомых портов журнала и движка
// (JournalStore, WorkFeed, Consumer, LeaseStore, ProjectionStore, ChangeLog,
// Sealer) для тестов движка, воркера, стадии, проекций и живых обновлений,
// пока адаптеры Postgres эпика 04 не готовы. Семантика — как в AD-6, AD-44,
// AD-45: seq по порядку, проверка эпохи аренды (ErrFenced), эффекты и курсор
// в одной «транзакции» Append, сигнал «есть новое» после записи.
//
// Слой: application (тестовая опора; без драйверов и сети). В сборку ролей
// не входит.
package enginemem

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
)

// Journal — журнал в памяти с проекциями, вкладами, курсорами, арендами и
// журналом изменений.
type Journal struct {
	mu        sync.Mutex
	now       func() time.Time
	entries   []jc.JournalEntry
	envelopes map[int][]byte
	cursors   map[string]int64
	epochs    map[string]int64
	leases    map[string]lease
	proj      map[string]map[string]json.RawMessage
	projItem  map[string]map[string]string
	contrib   map[string][]engineapp.Contribution
	changes   []engineapp.Change
	signal    chan struct{}
	// FailAppend — если задано, Append возвращает эту ошибку (проверка повторов).
	FailAppend error
	// Appends — число успешных Append.
	Appends int
}

type lease struct {
	holder string
	epoch  int64
	until  time.Time
}

// New создаёт журнал; now — часы (nil — time.Now).
func New(now func() time.Time) *Journal {
	if now == nil {
		now = time.Now
	}
	return &Journal{now: now, envelopes: map[int][]byte{}, cursors: map[string]int64{}, epochs: map[string]int64{},
		leases: map[string]lease{}, proj: map[string]map[string]json.RawMessage{}, projItem: map[string]map[string]string{},
		contrib: map[string][]engineapp.Contribution{}, signal: make(chan struct{})}
}

var (
	_ appjournal.JournalStore   = (*Journal)(nil)
	_ appjournal.LeaseStore     = (*Journal)(nil)
	_ appjournal.Consumer       = (*Journal)(nil)
	_ engineapp.ChangeLog       = (*Journal)(nil)
	_ engineapp.ProjectionStore = (*Journal)(nil)
)

func cursorKey(name string, part int) string { return fmt.Sprintf("%s/%d", name, part) }

// SetEpoch задаёт текущую эпоху аренды lease (для проверки ErrFenced).
func (j *Journal) SetEpoch(lease string, epoch int64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.epochs[lease] = epoch
}

// Append — единственная функция записи (AD-44) в памяти.
func (j *Journal) Append(_ context.Context, rq appjournal.AppendRequest) (appjournal.AppendResult, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.FailAppend != nil {
		return appjournal.AppendResult{}, j.FailAppend
	}
	if f := rq.Fence; f != nil {
		if cur, ok := j.epochs[f.Lease]; ok && cur != f.Epoch {
			return appjournal.AppendResult{}, appjournal.ErrFenced
		}
	}
	now := j.now().UTC()
	var res appjournal.AppendResult
	for _, p := range rq.Batch {
		e := p.Entry
		e.Seq = len(j.entries) + 1
		if e.Chain == "" {
			e.Chain = jc.JournalEntryChainMain
		}
		e.CommittedAt = engineapp.FormatTime(now)
		e.RecordedAt = engineapp.FormatTime(now)
		if e.ReceivedAt == "" {
			e.ReceivedAt = engineapp.FormatTime(now)
		}
		e.Commit, e.Link = fmt.Sprintf("c%d", e.Seq), fmt.Sprintf("l%d", e.Seq)
		j.entries = append(j.entries, e)
		j.envelopes[e.Seq] = p.Envelope
		res.Seqs = append(res.Seqs, int64(e.Seq))
	}
	for _, ef := range rq.Effects {
		switch x := ef.(type) {
		case engineapp.ProjectionPut:
			if j.proj[x.Name] == nil {
				j.proj[x.Name], j.projItem[x.Name] = map[string]json.RawMessage{}, map[string]string{}
			}
			j.proj[x.Name][x.Key] = x.Value
			j.projItem[x.Name][x.Key] = x.ItemID
		case engineapp.ProjectionReset:
			for k, item := range j.projItem[x.Name] {
				if x.ItemID == "" || item == x.ItemID {
					delete(j.proj[x.Name], k)
					delete(j.projItem[x.Name], k)
				}
			}
		case engineapp.ContributionsReplace:
			j.contrib[x.ItemID] = slices.Clone(x.Rows)
		case engineapp.Notify:
			j.changes = append(j.changes, x.Changes...)
		default:
			return appjournal.AppendResult{}, fmt.Errorf("enginemem: неизвестный эффект %s", ef.EffectKind())
		}
	}
	if c := rq.Consumer; c != nil {
		j.cursors[cursorKey(c.Name, c.Partition)] = c.Seq
	}
	res.Committed = now
	j.Appends++
	close(j.signal)
	j.signal = make(chan struct{})
	return res, nil
}

// Read — записи потока (Stream) или партиции (Partition) после AfterSeq.
func (j *Journal) Read(_ context.Context, q appjournal.ReadQuery) ([]jc.JournalEntry, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	var out []jc.JournalEntry
	for _, e := range j.entries {
		if int64(e.Seq) <= q.AfterSeq {
			continue
		}
		if q.Stream != "" && e.Stream != q.Stream && !(e.ItemID != nil && q.Stream == "item:"+*e.ItemID) {
			continue
		}
		if q.Stream == "" && e.Partition != q.Partition {
			continue
		}
		out = append(out, e)
		if q.Limit > 0 && len(out) == q.Limit {
			break
		}
	}
	return out, nil
}

// Head — головы цепочек.
func (j *Journal) Head(context.Context) (appjournal.Heads, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return appjournal.Heads{MainSeq: int64(len(j.entries))}, nil
}

// Open — «расшифрованный» конверт записи.
func (j *Journal) Open(_ context.Context, e jc.JournalEntry) (appjournal.Envelope, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	raw, ok := j.envelopes[e.Seq]
	if !ok {
		return appjournal.Envelope{}, fmt.Errorf("enginemem: нет конверта seq %d", e.Seq)
	}
	return appjournal.Envelope{Raw: raw}, nil
}

// Entries — копия всех записей.
func (j *Journal) Entries() []jc.JournalEntry {
	j.mu.Lock()
	defer j.mu.Unlock()
	return slices.Clone(j.entries)
}

// Cursor — курсор потребителя.
func (j *Journal) Cursor(name string, part int) int64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.cursors[cursorKey(name, part)]
}

// Contributions — вклады изделия.
func (j *Journal) Contributions(item string) []engineapp.Contribution {
	j.mu.Lock()
	defer j.mu.Unlock()
	return slices.Clone(j.contrib[item])
}

// Projection — все значения проекции name, отсортированные по ключу.
func (j *Journal) Projection(name string) map[string]json.RawMessage {
	j.mu.Lock()
	defer j.mu.Unlock()
	return maps.Clone(j.proj[name])
}

// Get — значение проекции (порт ProjectionStore).
func (j *Journal) Get(_ context.Context, name, key string) (json.RawMessage, bool, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	v, ok := j.proj[name][key]
	return v, ok, nil
}

// After — изменения после seq (порт ChangeLog).
func (j *Journal) After(_ context.Context, afterSeq int64, limit int) ([]engineapp.Change, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	var out []engineapp.Change
	for _, c := range j.changes {
		if c.Seq > afterSeq {
			out = append(out, c)
		}
	}
	slices.SortStableFunc(out, func(a, b engineapp.Change) int { return int(a.Seq - b.Seq) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Wait — ждать следующей записи (сигнал «есть новое»).
func (j *Journal) Wait(ctx context.Context) error {
	j.mu.Lock()
	ch := j.signal
	j.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}

// Acquire — аренда с эпохой (порт LeaseStore).
func (j *Journal) Acquire(_ context.Context, name, holder string, ttl time.Duration) (appjournal.Fence, bool, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := j.now()
	l, ok := j.leases[name]
	if ok && l.holder != holder && now.Before(l.until) {
		return appjournal.Fence{}, false, nil
	}
	if !ok || l.holder != holder || !now.Before(l.until) {
		l = lease{holder: holder, epoch: j.epochs[name] + 1}
		j.epochs[name] = l.epoch
	}
	l.until = now.Add(ttl)
	j.leases[name] = l
	return appjournal.Fence{Lease: name, Epoch: l.epoch}, true, nil
}

// Release — отдать аренду.
func (j *Journal) Release(_ context.Context, f appjournal.Fence) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if l, ok := j.leases[f.Lease]; ok && l.epoch == f.Epoch {
		delete(j.leases, f.Lease)
	}
	return nil
}

// Consume — потребитель журнала (порт Consumer): пачки после курсора, выход и
// курсор — одним Append.
func (j *Journal) Consume(ctx context.Context, name string, scope appjournal.Scope, handle func(context.Context, []jc.JournalEntry) (appjournal.AppendRequest, error)) error {
	part := scope.Partition
	if scope.Global {
		part = engineapp.GlobalPartition
	}
	for ctx.Err() == nil {
		j.mu.Lock()
		cur := j.cursors[cursorKey(name, part)]
		var batch []jc.JournalEntry
		for _, e := range j.entries {
			if int64(e.Seq) > cur && (scope.Global || e.Partition == scope.Partition) {
				batch = append(batch, e)
				if len(batch) == 100 {
					break
				}
			}
		}
		ch := j.signal
		j.mu.Unlock()
		if len(batch) == 0 {
			select {
			case <-ctx.Done():
				return nil
			case <-ch:
			}
			continue
		}
		rq, err := handle(ctx, batch)
		if err != nil {
			return err
		}
		rq.Consumer = &appjournal.CursorAdvance{Name: name, Partition: part, Seq: int64(batch[len(batch)-1].Seq)}
		if _, err := j.Append(ctx, rq); err != nil {
			return err
		}
	}
	return nil
}

// Feed — WorkFeed в памяти над журналом: фиксированные партиции с эпохами.
type Feed struct {
	J     *Journal
	Parts []engineapp.Partition
}

var _ engineapp.WorkFeed = (*Feed)(nil)

// Partitions — арендованные партиции.
func (f *Feed) Partitions(context.Context) ([]engineapp.Partition, error) {
	return slices.Clone(f.Parts), nil
}

// Next — изделия партиции с триггерами после курсора воркера (AD-5):
// записи роли worker триггером не являются.
func (f *Feed) Next(ctx context.Context, p engineapp.Partition) ([]engineapp.Work, error) {
	for {
		works, ch := f.pending(p)
		if len(works) > 0 {
			return works, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ch:
		}
	}
}

func (f *Feed) pending(p engineapp.Partition) ([]engineapp.Work, chan struct{}) {
	j := f.J
	j.mu.Lock()
	defer j.mu.Unlock()
	cur := j.cursors[cursorKey(engineapp.ConsumerWorker, p.Number)]
	byItem := map[string]engineapp.Work{}
	for _, e := range j.entries {
		if int64(e.Seq) <= cur || e.Partition != p.Number || e.ItemID == nil {
			continue
		}
		if info, ok := catalog.Lookup(catalog.Type(e.EventType)); ok && info.Role == engineapp.ConsumerWorker {
			continue
		}
		byItem[*e.ItemID] = engineapp.Work{ItemID: *e.ItemID, UpToSeq: int64(e.Seq), Trigger: e}
	}
	works := make([]engineapp.Work, 0, len(byItem))
	for _, k := range slices.Sorted(maps.Keys(byItem)) {
		works = append(works, byItem[k])
	}
	return works, j.signal
}

// Sealer — «подпись» в памяти: DSSE с payload и пустой подписью ключа.
type Sealer struct{ KeyRef string }

// Seal упаковывает payload в DSSE.
func (s Sealer) Seal(_ context.Context, payload []byte) ([]byte, error) {
	return json.Marshal(map[string]any{
		"payload":     base64.StdEncoding.EncodeToString(payload),
		"payloadType": engineapp.PayloadTypeEvent,
		"signatures":  []map[string]string{{"keyid": s.KeyRef, "sig": ""}},
	})
}
