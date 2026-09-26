// Пакет clock — адаптеры часов (AD-35, AD-37): InfraClock (ключ infra_clock:
// system) и DomainClock (ключ domain_clock: system | journal).
//
// Слой: infrastructure/storage, модуль journal. Отдельный пакет, чтобы аренды
// и Append (storage/journal) не импортировали доменные часы: у них только
// InfraClock (AD-37).
//
// Режим часов — свойство журнала (AD-37): его фиксирует генезис записью
// time.clock.mode_set (system | scenario). В режиме scenario доменное
// «сейчас» — последняя запись time.clock.ticked прогона (её ведёт simulation);
// все процессы читают её из журнала, поэтому «сейчас» у копий одинаково.
package clock

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	app "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
)

// System — системные часы: InfraClock и DomainClock режима system.
type System struct{}

// Now — реальное время UTC.
func (System) Now() time.Time { return time.Now().UTC() }

// SystemDomain — DomainClock режима system: доменное время = реальное.
type SystemDomain struct{}

// Now — реальное время UTC.
func (SystemDomain) Now(context.Context) (time.Time, error) { return time.Now().UTC(), nil }

var (
	_ app.InfraClock  = System{}
	_ app.DomainClock = SystemDomain{}
)

// Режимы часов (time.clock.mode_set).
const (
	ModeSystem   = "system"
	ModeScenario = "scenario"
)

// ErrNoTick — режим scenario, но у прогона ещё нет ни одного тика.
var ErrNoTick = errors.New("часы сценария: нет записи time.clock.ticked")

// Journal — DomainClock из журнала (ключ domain_clock: journal): режим — из
// time.clock.mode_set (нет записи — system), «сейчас» сценария — последняя
// time.clock.ticked прогона из контекста (application/journal.WithRun).
type Journal struct {
	store app.JournalStore

	mu   sync.Mutex
	mode string
}

// NewJournal создаёт доменные часы над журналом.
func NewJournal(store app.JournalStore) *Journal { return &Journal{store: store} }

var _ app.DomainClock = (*Journal)(nil)

// Now — доменное «сейчас» (AD-37).
func (j *Journal) Now(ctx context.Context) (time.Time, error) {
	mode, err := j.Mode(ctx)
	if err != nil {
		return time.Time{}, err
	}
	if mode != ModeScenario {
		return time.Now().UTC(), nil
	}
	ticks, err := j.store.Read(ctx, app.ReadQuery{EventType: string(catalog.TimeClockTicked), RunID: app.RunFrom(ctx), Backward: true, Limit: 1})
	if err != nil {
		return time.Time{}, err
	}
	if len(ticks) == 0 {
		return time.Time{}, ErrNoTick
	}
	var data struct {
		Now string `json:"now"`
	}
	if err := j.data(ctx, ticks[0], &data); err != nil {
		return time.Time{}, err
	}
	return dj.ParseTime(data.Now)
}

// Mode — режим часов журнала; найденный режим кэшируется (меняется только
// сбросом окружения, AD-37).
func (j *Journal) Mode(ctx context.Context) (string, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.mode != "" {
		return j.mode, nil
	}
	set, err := j.store.Read(ctx, app.ReadQuery{EventType: string(catalog.TimeClockModeSet), Limit: 1})
	if err != nil {
		return "", err
	}
	if len(set) == 0 {
		return ModeSystem, nil // генезиса ещё нет — не кэшируем
	}
	var data struct {
		Mode string `json:"mode"`
	}
	if err := j.data(ctx, set[0], &data); err != nil {
		return "", err
	}
	if data.Mode != ModeSystem && data.Mode != ModeScenario {
		return "", fmt.Errorf("time.clock.mode_set: режим %q", data.Mode)
	}
	j.mode = data.Mode
	return j.mode, nil
}

// data — поле data из payload конверта DSSE записи.
func (j *Journal) data(ctx context.Context, e jc.JournalEntry, v any) error {
	env, err := j.store.Open(ctx, e)
	if err != nil {
		return err
	}
	var dsse struct {
		Payload string `json:"payload"`
	}
	if err := json.Unmarshal(env.Raw, &dsse); err != nil {
		return fmt.Errorf("конверт %s: %w", e.EventID, err)
	}
	payload, err := base64.StdEncoding.DecodeString(dsse.Payload)
	if err != nil {
		return fmt.Errorf("payload %s: %w", e.EventID, err)
	}
	var ev struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil {
		return fmt.Errorf("payload %s: %w", e.EventID, err)
	}
	return json.Unmarshal(ev.Data, v)
}

// Fake — управляемые часы для тестов и прогонов: InfraClock с ручным сдвигом.
type Fake struct {
	mu sync.Mutex
	t  time.Time
}

// NewFake — часы, стоящие на t.
func NewFake(t time.Time) *Fake { return &Fake{t: t.UTC()} }

// Now — текущее значение.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

// Advance сдвигает часы на d.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}

var _ app.InfraClock = (*Fake)(nil)
