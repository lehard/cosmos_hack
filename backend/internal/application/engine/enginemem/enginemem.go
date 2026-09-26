// Пакет enginemem — фейки в памяти ведомых портов журнала и движка
// (JournalStore, WorkFeed, Consumer, LeaseStore, ProjectionStore, ChangeLog,
// Sealer) для тестов движка, воркера, стадии, проекций и живых обновлений,
// пока адаптеры Postgres эпика 04 не готовы. Семантика — как в AD-6, AD-39,
// AD-44, AD-45: seq по порядку, проверка эпохи аренды (ErrFenced), проверки
// конкурентности команд AD-39 (journal.stale_state, journal.stale_policy,
// journal.concession_exhausted) и повтор event_id (journal.duplicate) — как в
// адаптере Postgres (infrastructure/storage/journal), эффекты и курсор в одной
// «транзакции» Append, сигнал «есть новое» после записи.
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
	"strconv"
	"strings"
	"sync"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
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
	// ids — записанные event_id по цепочкам (UNIQUE (chain, event_id) адаптера).
	ids map[string]bool
	// ledger — леджер разрешений на отклонение: открытия (+лимит) и расходы
	// (−количество) по concession_id (journal.concession_ledger адаптера).
	ledger map[string][]int64
	signal chan struct{}
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
		contrib: map[string][]engineapp.Contribution{}, ids: map[string]bool{}, ledger: map[string][]int64{},
		signal: make(chan struct{})}
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
	// Проверки конкурентности команды (AD-39) — до записи, как в адаптере:
	// отказ — ни записей, ни расхода лимита.
	for _, c := range rq.Checks {
		if err := j.check(c); err != nil {
			return appjournal.AppendResult{}, err
		}
	}
	for _, g := range rq.ConcessionGrants {
		if g.ConcessionID == "" || g.Limit <= 0 {
			return appjournal.AppendResult{}, fmt.Errorf("%w: лимит разрешения %q = %d", appjournal.ErrInvalidEntry, g.ConcessionID, g.Limit)
		}
	}
	seen := map[string]bool{}
	for _, p := range rq.Batch {
		k := idKey(p.Entry)
		if j.ids[k] || seen[k] {
			return appjournal.AppendResult{}, fmt.Errorf("%w: event_id %s", appjournal.ErrDuplicate, p.Entry.EventID)
		}
		seen[k] = true
	}
	now := j.now().UTC()
	var res appjournal.AppendResult
	for _, p := range rq.Batch {
		e := p.Entry
		e.Seq = len(j.entries) + 1
		if e.Chain == "" {
			e.Chain = jc.JournalEntryChainMain
		}
		j.ids[idKey(e)] = true
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
	// Леджер разрешений: открытие и расход атомарно с записью решения (AD-39).
	for _, g := range rq.ConcessionGrants {
		j.ledger[g.ConcessionID] = append(j.ledger[g.ConcessionID], g.Limit)
	}
	for _, c := range rq.Checks {
		if c.ConcessionID != "" && c.Consume > 0 {
			j.ledger[c.ConcessionID] = append(j.ledger[c.ConcessionID], -c.Consume)
		}
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
			for _, c := range x.Changes {
				c.Pos = int64(len(j.changes) + 1)
				j.changes = append(j.changes, c)
			}
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

// idKey — ключ уникальности event_id в цепочке (пустая цепочка — main).
func idKey(e jc.JournalEntry) string {
	chain := e.Chain
	if chain == "" {
		chain = jc.JournalEntryChainMain
	}
	return string(chain) + "/" + e.EventID
}

// check — проверка AD-39 над записанным журналом; коды и параметры отказов —
// как у адаптера Postgres (check в infrastructure/storage/journal).
func (j *Journal) check(c appjournal.Check) error {
	basis := strconv.FormatInt(c.BasisSeq, 10)
	if c.Stream != "" {
		for _, e := range j.entries {
			if e.Chain == jc.JournalEntryChainMain && e.Stream == c.Stream && int64(e.Seq) > c.BasisSeq && dj.Classify(e).GuardRelevant {
				return appjournal.Reject(appjournal.ErrStaleState, c.BasisSeq, "stream", c.Stream, "basis_seq", basis)
			}
		}
	}
	if c.ItemProcessed && strings.HasPrefix(c.Stream, "item:") {
		for _, e := range j.entries {
			if e.Chain == jc.JournalEntryChainMain && e.Stream == c.Stream && dj.Classify(e).Trigger &&
				int64(e.Seq) > j.cursors[cursorKey(appjournal.WorkerConsumer, e.Partition)] {
				return appjournal.Reject(appjournal.ErrStaleState, c.BasisSeq, "stream", c.Stream, "basis_seq", basis, "reason", "unprocessed_input")
			}
		}
	}
	if c.PolicyStream != "" {
		for _, e := range j.entries {
			if e.Chain == jc.JournalEntryChainMain && e.Stream == c.PolicyStream && int64(e.Seq) > c.PolicySeq {
				return appjournal.Reject(appjournal.ErrStalePolicy, c.PolicySeq, "policy_seq", strconv.FormatInt(c.PolicySeq, 10))
			}
		}
	}
	if c.ConcessionID != "" && c.Consume > 0 {
		var remaining, limit int64
		for _, d := range j.ledger[c.ConcessionID] {
			remaining += d
			if d > 0 {
				limit += d
			}
		}
		if remaining < c.Consume {
			return appjournal.Reject(appjournal.ErrConcessionExhausted, 0, "concession_id", c.ConcessionID,
				"remaining", strconv.FormatInt(remaining, 10), "limit", strconv.FormatInt(limit, 10))
		}
	}
	return nil
}

// Read — записи по фильтрам ReadQuery (цепочка, поток, изделие, партиция,
// тип, после AfterSeq, в обратном порядке); момент и прогон не учитываются.
func (j *Journal) Read(_ context.Context, q appjournal.ReadQuery) ([]jc.JournalEntry, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	chain := jc.JournalEntryChain(q.Chain)
	if chain == "" {
		chain = jc.JournalEntryChainMain
	}
	entries := slices.Clone(j.entries)
	if q.Backward {
		slices.Reverse(entries)
	}
	var out []jc.JournalEntry
	for _, e := range entries {
		switch {
		case e.Chain != chain,
			!q.Backward && int64(e.Seq) <= q.AfterSeq,
			q.Stream != "" && e.Stream != q.Stream,
			q.ItemID != "" && (e.ItemID == nil || *e.ItemID != q.ItemID),
			q.Partition != nil && e.Partition != *q.Partition,
			q.EventType != "" && e.EventType != q.EventType:
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

// After — изменения журнала изменений по запросу (порт ChangeLog).
func (j *Journal) After(_ context.Context, q engineapp.ChangeQuery) ([]engineapp.Change, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	var out []engineapp.Change
	for _, c := range j.changes {
		if c.Pos <= q.AfterPos || q.UpToPos > 0 && c.Pos > q.UpToPos || c.Seq <= q.AfterSeq {
			continue
		}
		out = append(out, c)
		if q.Limit > 0 && len(out) == q.Limit {
			break
		}
	}
	return out, nil
}

// Tail — позиция последнего изменения (порт ChangeLog).
func (j *Journal) Tail(context.Context) (int64, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return int64(len(j.changes)), nil
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
		part = appjournal.GlobalPartition
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
	cur := j.cursors[cursorKey(appjournal.WorkerConsumer, p.Number)]
	byItem := map[string]engineapp.Work{}
	for _, e := range j.entries {
		if int64(e.Seq) <= cur || e.Partition != p.Number || e.ItemID == nil {
			continue
		}
		if info, ok := catalog.Lookup(catalog.Type(e.EventType)); ok && info.Role == engineapp.RoleWorker {
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
