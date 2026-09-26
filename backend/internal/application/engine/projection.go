package engine

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// Каркас проекций (AD-2, AD-45): у каждой проекции один модуль-писатель;
// проекция — чистая функция журнала; пишется только эффектами в транзакции
// Append вместе с курсором; пересобирается командой `ant rebuild` целиком или
// по изделию (`--item`), FR-115, FR-124.
//
// Два вида проекций:
//   - проекция изделия — функция итога свёртки изделия (Snapshot): воркер
//     заменяет её целиком при каждой пересвёртке, как и вклады показателей;
//   - глобальная проекция — чистая свёртка `(состояние, запись) → состояние`
//     по ключу; её ведёт роль projector (одна копия-лидер).

// Change — изменение сущности для живых обновлений (AD-21): сообщение SSE
// (сущность, id, seq) плюс время приёма записи-триггера для метрики
// ant_event_to_sse_seconds (FR-2, AD-6).
type Change struct {
	Entity platform.EntityKind
	ID     string
	Seq    int64
	RunID  string
	// ReceivedAt — время приёма записи-триггера (received_at); в SSE не уходит.
	ReceivedAt time.Time
	// Pos — позиция в журнале изменений (порядок фиксации); ставит
	// хранилище ChangeLog, в SSE не уходит.
	Pos int64
}

// Эффекты транзакции Append (application/journal.Effect).

// ProjectionPut — записать значение проекции Name по ключу Key; ItemID —
// изделие, к которому относится строка (для `rebuild --item`).
type ProjectionPut struct {
	Name   string
	Key    string
	ItemID string
	Value  json.RawMessage
}

// EffectKind — вид эффекта.
func (ProjectionPut) EffectKind() string { return "engine.projection.put" }

// ProjectionReset — очистить проекцию Name целиком или строки одного изделия.
type ProjectionReset struct {
	Name   string
	ItemID string
}

// EffectKind — вид эффекта.
func (ProjectionReset) EffectKind() string { return "engine.projection.reset" }

// Contribution — строка вклада изделия в показатель (AD-45): агрегат —
// сумма вкладов, инкременты запрещены; Sources — id исходных записей для
// раскрытия («Соглашения/Показатели»). Значение — целое с масштабом (AD-4).
type Contribution struct {
	ItemID  string   `json:"item_id"`
	Metric  string   `json:"metric"`
	Slice   string   `json:"slice"`
	Value   int64    `json:"value"`
	Scale   int      `json:"scale,omitempty"`
	Sources []string `json:"sources,omitempty"`
}

// ContributionsReplace — заменить вклады изделия целиком (AD-45).
type ContributionsReplace struct {
	ItemID string
	Rows   []Contribution
}

// EffectKind — вид эффекта.
func (ContributionsReplace) EffectKind() string { return "engine.contributions.replace" }

// Notify — строки журнала изменений для SSE и сигнал NOTIFY после фиксации (AD-6).
type Notify struct {
	Changes []Change
}

// EffectKind — вид эффекта.
func (Notify) EffectKind() string { return "engine.notify" }

// ItemProjection — проекция изделия: значение по итогу свёртки (nil — строки нет).
type ItemProjection struct {
	Name   string
	Writer kernel.Module
	// View — чистая функция итога свёртки изделия.
	View func(itemID string, s engine.Snapshot, reactions []kernel.Reaction) (any, error)
}

// GlobalProjection — глобальная проекция: чистая свёртка по ключам (AD-45).
type GlobalProjection struct {
	Name   string
	Writer kernel.Module
	// Keys — ключи, которые меняет запись (пусто — запись не относится к проекции).
	Keys func(r kernel.Record) []string
	// Step — чистая функция (состояние по ключу, запись) → состояние.
	Step func(key string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error)
	// Entity — сущность SSE для ключа (ok=false — не сообщать).
	Entity func(key string) (platform.EntityKind, string, bool)
}

// Contributor — строки вклада изделия в показатели по итогу свёртки (analytics, эпик 25).
type Contributor func(itemID string, s engine.Snapshot) ([]Contribution, error)

// Registry — реестр проекций: одно имя — один писатель (AD-2, AD-45).
type Registry struct {
	mu           sync.RWMutex
	writers      map[string]kernel.Module
	items        []ItemProjection
	globals      []GlobalProjection
	contributors []Contributor
}

// ItemStateProjection — проекция движка «состояние изделия»: basis_seq,
// хеш итога свёртки и число вычисленных реакций; ею пользуются запросы
// состояния (как кэш, AD-22), сверка проекций верификатором и rebuild_hash.
const ItemStateProjection = "engine.item_state"

// ItemState — значение проекции engine.item_state.
type ItemState struct {
	ItemID    string `json:"item_id"`
	BasisSeq  int64  `json:"basis_seq"`
	StateHash string `json:"state_hash"`
	Reactions int    `json:"reactions"`
}

// NewRegistry — реестр с проекцией движка engine.item_state.
func NewRegistry() *Registry {
	r := &Registry{writers: map[string]kernel.Module{}}
	if err := r.AddItem(ItemProjection{Name: ItemStateProjection, Writer: "engine", View: itemStateView}); err != nil {
		panic(err)
	}
	return r
}

func itemStateView(itemID string, s engine.Snapshot, rs []kernel.Reaction) (any, error) {
	h, err := engine.StateHash(s, rs)
	if err != nil {
		return nil, err
	}
	return ItemState{ItemID: itemID, BasisSeq: s.BasisSeq, StateHash: h, Reactions: len(rs)}, nil
}

func (r *Registry) claim(name string, w kernel.Module) error {
	if name == "" {
		return fmt.Errorf("проекция без имени")
	}
	if !knownModule(w) {
		return fmt.Errorf("проекция %s: писатель %q — не модуль каталога", name, w)
	}
	if prev, ok := r.writers[name]; ok {
		return fmt.Errorf("проекция %s уже зарегистрирована писателем %s: у проекции один писатель (AD-45)", name, prev)
	}
	r.writers[name] = w
	return nil
}

func knownModule(m kernel.Module) bool {
	for _, mi := range catalog.Modules {
		if mi.Name == string(m) {
			return true
		}
	}
	return false
}

// AddItem регистрирует проекцию изделия.
func (r *Registry) AddItem(p ItemProjection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.claim(p.Name, p.Writer); err != nil {
		return err
	}
	r.items = append(r.items, p)
	return nil
}

// AddGlobal регистрирует глобальную проекцию.
func (r *Registry) AddGlobal(p GlobalProjection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.claim(p.Name, p.Writer); err != nil {
		return err
	}
	r.globals = append(r.globals, p)
	return nil
}

// AddContributor регистрирует источник вкладов показателей.
func (r *Registry) AddContributor(c Contributor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contributors = append(r.contributors, c)
}

// Items — проекции изделия в порядке регистрации.
func (r *Registry) Items() []ItemProjection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.items)
}

// Globals — глобальные проекции в порядке регистрации.
func (r *Registry) Globals() []GlobalProjection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.globals)
}

// ItemEffects — эффекты проекций изделия по итогу свёртки: значения всех
// проекций изделия и замена вкладов целиком (AD-45).
func (r *Registry) ItemEffects(itemID string, s engine.Snapshot, rs []kernel.Reaction) ([]appjournal.Effect, error) {
	r.mu.RLock()
	items, contributors := slices.Clone(r.items), slices.Clone(r.contributors)
	r.mu.RUnlock()
	var out []appjournal.Effect
	for _, p := range items {
		v, err := p.View(itemID, s, rs)
		if err != nil {
			return nil, fmt.Errorf("проекция %s: %w", p.Name, err)
		}
		if v == nil {
			out = append(out, ProjectionReset{Name: p.Name, ItemID: itemID})
			continue
		}
		b, err := engine.Canonical(v)
		if err != nil {
			return nil, fmt.Errorf("проекция %s: %w", p.Name, err)
		}
		out = append(out, ProjectionPut{Name: p.Name, Key: itemID, ItemID: itemID, Value: b})
	}
	rows := []Contribution{}
	for _, c := range contributors {
		cs, err := c(itemID, s)
		if err != nil {
			return nil, fmt.Errorf("вклады изделия: %w", err)
		}
		rows = append(rows, cs...)
	}
	out = append(out, ContributionsReplace{ItemID: itemID, Rows: rows})
	return out, nil
}
