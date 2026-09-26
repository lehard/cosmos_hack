package access

import (
	"context"
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
type SecurityBus struct {
	Journal appjournal.JournalStore
	Codec   *engineapp.Codec
}

var _ app.SecurityEvents = SecurityBus{}

// AuthFailed — security.auth.failed (FR-118, AD-15: неудачный вход).
func (b SecurityBus) AuthFailed(ctx context.Context, f app.AuthFailure) error {
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
func (b SecurityBus) AccessDenied(ctx context.Context, x app.Denial) error {
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

func (b SecurityBus) write(ctx context.Context, t catalog.Type, at time.Time, data any) error {
	if at.IsZero() {
		at = time.Now()
		if b.Codec.Now != nil {
			at = b.Codec.Now()
		}
	}
	p, err := b.Codec.Encode(ctx, engineapp.Out{EventID: uuid.NewV7().String(), Type: t, Kind: catalog.KindService, Stream: "global", OccurredAt: at, Data: data})
	if err != nil {
		return err
	}
	_, err = b.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}})
	return err
}
