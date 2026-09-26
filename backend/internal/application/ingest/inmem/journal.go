package inmem

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
)

// Journal — фейк порта JournalStore в памяти: seq по основной цепочке и
// номера CA-‹n› по цепочке критических действий; конверты хранятся как есть.
// Звенья и шифрование — эпик 04; здесь только порядок и содержимое.
type Journal struct {
	mu       sync.Mutex
	main     []Stored
	ca       []Stored
	now      func() time.Time
	FailNext error
}

// Stored — записанная запись с конвертом.
type Stored struct {
	Entry    jc.JournalEntry
	Envelope []byte
}

// NewJournal создаёт журнал; now — часы committed_at (nil — системные).
func NewJournal(now func() time.Time) *Journal {
	if now == nil {
		now = time.Now
	}
	return &Journal{now: now}
}

// Append — пачка основной цепочки и записи CA одной «транзакцией».
func (j *Journal) Append(_ context.Context, rq journal.AppendRequest) (journal.AppendResult, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.FailNext; err != nil {
		j.FailNext = nil
		return journal.AppendResult{}, err
	}
	t := j.now().UTC()
	var res journal.AppendResult
	for _, p := range rq.Batch {
		e := p.Entry
		e.Seq = len(j.main) + 1
		e.Chain = jc.JournalEntryChainMain
		e.CommittedAt = t.Format("2006-01-02T15:04:05.000Z")
		j.main = append(j.main, Stored{Entry: e, Envelope: slices.Clone(p.Envelope)})
		res.Seqs = append(res.Seqs, int64(e.Seq))
	}
	for _, p := range rq.Critical {
		e := p.Entry
		e.Seq = len(j.ca) + 1
		e.Chain = jc.JournalEntryChainCa
		e.CommittedAt = t.Format("2006-01-02T15:04:05.000Z")
		j.ca = append(j.ca, Stored{Entry: e, Envelope: slices.Clone(p.Envelope)})
		res.CARefs = append(res.CARefs, fmt.Sprintf("CA-%d", e.Seq))
	}
	res.Committed = t
	return res, nil
}

// Read — записи потока (пусто — все) после AfterSeq.
func (j *Journal) Read(_ context.Context, q journal.ReadQuery) ([]jc.JournalEntry, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	var out []jc.JournalEntry
	for _, s := range j.main {
		if int64(s.Entry.Seq) <= q.AfterSeq || (q.Stream != "" && s.Entry.Stream != q.Stream) {
			continue
		}
		out = append(out, s.Entry)
		if q.Limit > 0 && len(out) >= q.Limit {
			break
		}
	}
	return out, nil
}

// Head — головы цепочек.
func (j *Journal) Head(context.Context) (journal.Heads, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return journal.Heads{MainSeq: int64(len(j.main)), CASeq: int64(len(j.ca))}, nil
}

// Open — конверт записи.
func (j *Journal) Open(_ context.Context, e jc.JournalEntry) (journal.Envelope, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	src := j.main
	if e.Chain == jc.JournalEntryChainCa {
		src = j.ca
	}
	if e.Seq < 1 || e.Seq > len(src) {
		return journal.Envelope{}, fmt.Errorf("нет записи %s/%d", e.Chain, e.Seq)
	}
	return journal.Envelope{Raw: slices.Clone(src[e.Seq-1].Envelope)}, nil
}

// Main — копия основной цепочки.
func (j *Journal) Main() []Stored {
	j.mu.Lock()
	defer j.mu.Unlock()
	return slices.Clone(j.main)
}

// CA — копия цепочки критических действий.
func (j *Journal) CA() []Stored {
	j.mu.Lock()
	defer j.mu.Unlock()
	return slices.Clone(j.ca)
}

// Count — число записей типа t в основной цепочке.
func (j *Journal) Count(t string) int {
	n := 0
	for _, s := range j.Main() {
		if s.Entry.EventType == t {
			n++
		}
	}
	return n
}

var _ journal.JournalStore = (*Journal)(nil)
