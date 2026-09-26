package inmem

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/application/materials"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/ingest"
)

// Registry — реестр приёма в памяти.
type Registry struct {
	mu      sync.Mutex
	seen    map[string]dom.Seen
	sources map[string]dom.SourceState
}

// NewRegistry создаёт пустой реестр.
func NewRegistry() *Registry {
	return &Registry{seen: map[string]dom.Seen{}, sources: map[string]dom.SourceState{}}
}

func key(src, id string) string { return src + "\x1f" + id }

// Seen — запись по ключу.
func (r *Registry) Seen(_ context.Context, src, id string) (*dom.Seen, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.seen[key(src, id)]
	if !ok {
		return nil, nil
	}
	return &s, nil
}

// SourceState — учёт номеров источника.
func (r *Registry) SourceState(_ context.Context, src string) (dom.SourceState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := r.sources[src]
	st.SourceID = src
	st.Gaps = slices.Clone(st.Gaps)
	return st, nil
}

// Commit — принятое сообщение и состояние источника.
func (r *Registry) Commit(_ context.Context, s dom.Seen, st dom.SourceState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen[key(s.SourceID, s.EventID)] = s
	r.sources[st.SourceID] = st
	return nil
}

// SaveSourceState — состояние источника.
func (r *Registry) SaveSourceState(_ context.Context, st dom.SourceState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[st.SourceID] = st
	return nil
}

// Sources — все источники по source_id.
func (r *Registry) Sources(context.Context) ([]dom.SourceState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]dom.SourceState, 0, len(r.sources))
	for _, k := range slices.Sorted(mapsKeys(r.sources)) {
		out = append(out, r.sources[k])
	}
	return out, nil
}

// Quarantine — карантин в памяти.
type Quarantine struct {
	mu   sync.Mutex
	recs []app.QuarantineRecord
}

// NewQuarantine создаёт пустой карантин.
func NewQuarantine() *Quarantine { return &Quarantine{} }

// Find — по отпечатку и коду.
func (q *Quarantine) Find(_ context.Context, fp string, code errcodes.Code) (*app.QuarantineRecord, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, r := range q.recs {
		if r.Fingerprint == fp && r.Code == code {
			return &r, nil
		}
	}
	return nil, nil
}

// Put — новая запись.
func (q *Quarantine) Put(_ context.Context, r app.QuarantineRecord) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.recs = append(q.recs, r)
	return nil
}

// Get — по ID.
func (q *Quarantine) Get(_ context.Context, id string) (app.QuarantineRecord, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, r := range q.recs {
		if r.ID == id {
			return r, nil
		}
	}
	return app.QuarantineRecord{}, app.ErrNotFound
}

// List — по фильтру, новые сначала.
func (q *Quarantine) List(_ context.Context, f app.QuarantineFilter) ([]app.QuarantineRecord, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var out []app.QuarantineRecord
	for i := len(q.recs) - 1; i >= 0; i-- {
		r := q.recs[i]
		if (f.Status != "" && r.Status != f.Status) || (f.SourceID != "" && r.SourceID != f.SourceID) || (f.Code != "" && r.Code != f.Code) {
			continue
		}
		out = append(out, r)
		if f.Limit > 0 && len(out) >= f.Limit {
			break
		}
	}
	return out, nil
}

// Count — число записей в состоянии.
func (q *Quarantine) Count(_ context.Context, st app.QuarantineStatus) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var n int64
	for _, r := range q.recs {
		if st == "" || r.Status == st {
			n++
		}
	}
	return n, nil
}

// Resolve — итог переобработки.
func (q *Quarantine) Resolve(_ context.Context, id string, st app.QuarantineStatus, by string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.recs {
		if q.recs[i].ID == id {
			q.recs[i].Status, q.recs[i].ResolvedBy = st, by
			return nil
		}
	}
	return app.ErrNotFound
}

// Materials — хранилище материалов в памяти по адресу H(байты).
type Materials struct {
	mu   sync.Mutex
	objs map[string][]byte
	meta map[string]materials.Meta
}

// NewMaterials создаёт пустое хранилище.
func NewMaterials() *Materials {
	return &Materials{objs: map[string][]byte{}, meta: map[string]materials.Meta{}}
}

// Put — сохранить; повтор — тот же адрес.
func (m *Materials) Put(_ context.Context, r io.Reader, meta materials.Meta) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	addr := dom.Digest(b)
	m.mu.Lock()
	m.objs[addr], m.meta[addr] = b, meta
	m.mu.Unlock()
	return addr, nil
}

// Get — по адресу.
func (m *Materials) Get(_ context.Context, addr string) (io.ReadCloser, materials.Meta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.objs[addr]
	if !ok {
		return nil, materials.Meta{}, app.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(b)), m.meta[addr], nil
}

// Has — есть ли материал.
func (m *Materials) Has(_ context.Context, addr string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objs[addr]
	return ok, nil
}

// Keys — реестр источников и ключей в памяти (демо-генезис до эпиков 05, 27).
type Keys struct {
	mu      sync.Mutex
	sources map[string]bool
	keys    map[string]app.KeyInfo
}

// NewKeys создаёт пустой реестр.
func NewKeys() *Keys { return &Keys{sources: map[string]bool{}, keys: map[string]app.KeyInfo{}} }

// Register регистрирует ключ источника (акт регистрации).
func (k *Keys) Register(info app.KeyInfo) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.sources[info.SourceID] = true
	k.keys[info.KeyRef] = info
}

// Revoke отзывает ключ (акт отзыва).
func (k *Keys) Revoke(ref string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	i := k.keys[ref]
	i.Revoked = true
	k.keys[ref] = i
}

// Source — зарегистрирован ли источник.
func (k *Keys) Source(_ context.Context, id string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if !k.sources[id] {
		return app.ErrUnknownSource
	}
	return nil
}

// Key — ключ по ссылке.
func (k *Keys) Key(_ context.Context, ref string) (app.KeyInfo, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	i, ok := k.keys[ref]
	if !ok {
		return app.KeyInfo{}, app.ErrUnknownKey
	}
	return i, nil
}

func mapsKeys[V any](m map[string]V) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// Telemetry — счётчики и показатели в памяти (проверка метрик FR-41).
type Telemetry struct {
	mu       sync.Mutex
	Counters map[string]int64
	Gauges   map[string]int64
	Observed map[string]int
}

// NewTelemetry создаёт пустую телеметрию.
func NewTelemetry() *Telemetry {
	return &Telemetry{Counters: map[string]int64{}, Gauges: map[string]int64{}, Observed: map[string]int{}}
}

func name(n string, labels []string) string { return n + "{" + strings.Join(labels, ",") + "}" }

// Counter — счётчик.
func (t *Telemetry) Counter(n string, d int64, labels ...string) {
	t.mu.Lock()
	t.Counters[name(n, labels)] += d
	t.mu.Unlock()
}

// Observe — наблюдение гистограммы (учитывается число наблюдений).
func (t *Telemetry) Observe(n string, _ time.Duration, labels ...string) {
	t.mu.Lock()
	t.Observed[name(n, labels)]++
	t.mu.Unlock()
}

// Gauge — показатель.
func (t *Telemetry) Gauge(n string, v int64, labels ...string) {
	t.mu.Lock()
	t.Gauges[name(n, labels)] = v
	t.mu.Unlock()
}

// Sum — сумма счётчика n по всем меткам, содержащим все подстроки want.
func (t *Telemetry) Sum(n string, want ...string) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	var s int64
	for k, v := range t.Counters {
		if !strings.HasPrefix(k, n+"{") {
			continue
		}
		ok := true
		for _, w := range want {
			ok = ok && strings.Contains(k, w)
		}
		if ok {
			s += v
		}
	}
	return s
}

var (
	_ app.Registry            = (*Registry)(nil)
	_ app.QuarantineStore     = (*Quarantine)(nil)
	_ materials.MaterialStore = (*Materials)(nil)
	_ app.KeyRegistry         = (*Keys)(nil)
)
