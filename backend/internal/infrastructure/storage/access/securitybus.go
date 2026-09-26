package access

import (
	"context"
	"sync"
	"time"
	"uuid"

	app "ant/internal/application/access"
	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
)

// SecurityBus — порт application/access.SecurityEvents над журналом (AD-24:
// события безопасности — записи журнала семейства security; подписчики —
// потребители журнала, источники о них не знают). Запись собирает кодек
// движка (класс происхождения server_attested, поток global); эмитент типов —
// модуль security по каталогу (AD-40): этот адаптер — временный писатель от
// его имени до шины безопасности эпика 29.
//
// Отказы в доступе одного субъекта по одному действию и объекту пишутся не
// чаще раза в DeniedEvery: виджет, который опрашивает недоступную операцию,
// не засоряет журнал; неудачные входы пишутся все (их ограничивает частота входа).
type SecurityBus struct {
	Journal appjournal.JournalStore
	Codec   *engineapp.Codec
	// DeniedEvery — наименьший интервал между одинаковыми отказами (0 — минута).
	DeniedEvery time.Duration

	mu   sync.Mutex
	last map[string]time.Time
}

// NewSecurityBus — шина безопасности над журналом j и кодеком c.
func NewSecurityBus(j appjournal.JournalStore, c *engineapp.Codec) *SecurityBus {
	return &SecurityBus{Journal: j, Codec: c, DeniedEvery: time.Minute, last: map[string]time.Time{}}
}

var _ app.SecurityEvents = (*SecurityBus)(nil)

// AuthFailed — security.auth.failed (FR-118, AD-15: неудачный вход).
func (b *SecurityBus) AuthFailed(ctx context.Context, f app.AuthFailure) error {
	d := ev.SecurityAuthFailedV1{Reason: ev.SecurityAuthFailedV1Reason(f.Reason)}
	if f.Login != "" {
		l := f.Login
		if len(l) > 64 {
			l = l[:64]
		}
		d.Login = &l
	}
	if f.ClientIP != "" {
		ip := f.ClientIP
		d.ClientIp = &ip
	}
	return b.write(ctx, catalog.SecurityAuthFailed, f.At, d)
}

// AccessDenied — security.access.denied (FR-85, AD-24: отказ в доступе).
func (b *SecurityBus) AccessDenied(ctx context.Context, x app.Denial) error {
	if !b.fresh(x.PersonID + "|" + x.ActionID + "|" + x.Object.Kind + ":" + x.Object.ID + "|" + x.Code) {
		return nil
	}
	d := ev.SecurityAccessDeniedV1{ActionID: x.ActionID, ProblemCode: x.Code}
	if x.PersonID != "" {
		p := ev.PersonRef(x.PersonID)
		d.PersonID = &p
	}
	if x.Object.Kind != "" && x.Object.ID != "" {
		ref := ev.StreamRef(x.Object.Kind + ":" + x.Object.ID)
		d.ObjectRef = &ref
	}
	return b.write(ctx, catalog.SecurityAccessDenied, x.At, d)
}

// fresh — такого отказа не было за DeniedEvery (InfraClock, AD-37).
func (b *SecurityBus) fresh(key string) bool {
	now := b.now()
	every := b.DeniedEvery
	if every <= 0 {
		every = time.Minute
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.last == nil {
		b.last = map[string]time.Time{}
	}
	if len(b.last) > 10000 {
		for k, t := range b.last {
			if now.Sub(t) > every {
				delete(b.last, k)
			}
		}
	}
	if t, ok := b.last[key]; ok && now.Sub(t) < every {
		return false
	}
	b.last[key] = now
	return true
}

func (b *SecurityBus) now() time.Time {
	if b.Codec != nil && b.Codec.Now != nil {
		return b.Codec.Now()
	}
	return time.Now()
}

func (b *SecurityBus) write(ctx context.Context, t catalog.Type, at time.Time, data any) error {
	if at.IsZero() {
		at = b.now()
	}
	p, err := b.Codec.Encode(ctx, engineapp.Out{EventID: uuid.NewV7().String(), Type: t, Kind: catalog.KindService, Stream: "global", OccurredAt: at, Data: data})
	if err != nil {
		return err
	}
	_, err = b.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}})
	return err
}
