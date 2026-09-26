package nonconformity

import (
	"context"
	"sort"

	app "ant/internal/application/nonconformity"
	"ant/internal/application/platform"
)

// Adapter — реализация fixtures ведущих портов модуля nonconformity (AD-36):
// очередь «Ждут моего решения», карточки и список несоответствий, разрешения
// на отклонение — из мира заготовок; решения контролёра, комиссии и главного
// сварщика двигают сценарий, если он ждёт именно их (FR-129), и видны поверх
// мира до сброса прогона (session.go).
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Queue — очередь «Ждут моего решения» (nonconformity.queue.list): ответ по
// роли субъекта, фильтр по виду строки и сортировка по риску или сроку.
func (Adapter) Queue(ctx context.Context, f app.QueueFilter, m platform.Moment, _ platform.Page) (app.DecisionQueue, error) {
	params := map[string]string{"kind": f.Kind, "sort": f.Sort}
	if p := platform.PrincipalFrom(ctx); !p.Anonymous() {
		params["role"] = p.Role
	}
	v, err := respond[app.DecisionQueue](ctx, "nonconformity.queue.list", params, &m)
	if err != nil {
		return v, err
	}
	d := sessionDecisions(ctx, m)
	out := v.Items[:0]
	for _, x := range v.Items {
		x, keep := d.queueRow(x)
		if keep && (f.Kind == "" || x.Kind == f.Kind) {
			out = append(out, x)
		}
	}
	switch f.Sort {
	case "risk":
		sort.SliceStable(out, func(i, j int) bool { return out[i].RiskRank < out[j].RiskRank })
	case "deadline":
		sort.SliceStable(out, func(i, j int) bool {
			a, b := out[i].DueAt, out[j].DueAt
			if a == nil || b == nil {
				return a != nil
			}
			return a.Before(*b)
		})
	}
	v.Items = out
	return v, nil
}

// Presentation — точка предъявления изделия (nonconformity.presentation.read, FR-19).
func (Adapter) Presentation(ctx context.Context, itemID string, m platform.Moment) (app.NCPresentationView, error) {
	v, err := respond[app.NCPresentationView](ctx, "nonconformity.presentation.read", map[string]string{"item_id": itemID}, &m)
	if err != nil {
		return v, err
	}
	return sessionDecisions(ctx, m).presentation(v), nil
}

// Card — карточка несоответствия (nonconformity.card.read, FR-51).
func (Adapter) Card(ctx context.Context, ncID string, m platform.Moment) (app.NCCard, error) {
	c, err := respond[app.NCCard](ctx, "nonconformity.card.read", map[string]string{"nc_id": ncID}, &m)
	if err != nil {
		return c, err
	}
	return sessionDecisions(ctx, m).card(c), nil
}

// List — несоответствия с фильтром по изделию и статусу (nonconformity.nonconformity.list).
func (Adapter) List(ctx context.Context, f app.NCFilter, m platform.Moment, _ platform.Page) (app.NCList, error) {
	v, err := respond[app.NCList](ctx, "nonconformity.nonconformity.list", map[string]string{"item_id": f.ItemID, "status": f.Status}, &m)
	if err != nil {
		return v, err
	}
	d := sessionDecisions(ctx, m)
	out := v.Items[:0]
	for _, x := range v.Items {
		if len(d.of(string(platform.EntityNonconformity), x.NCID))+len(d.of(string(platform.EntityItem), x.ItemID)) > 0 {
			c := d.card(app.NCCard{NCID: x.NCID, ItemID: x.ItemID, Status: x.Status, Axes: app.NCItemAxes{Disposition: x.Disposition}})
			x.Status, x.Disposition = c.Status, c.Axes.Disposition
		}
		if (f.ItemID == "" || x.ItemID == f.ItemID) && (f.Status == "" || x.Status == f.Status) {
			out = append(out, x)
		}
	}
	v.Items = out
	return v, nil
}

// Concessions — разрешения на отклонение (nonconformity.concession.list).
func (Adapter) Concessions(ctx context.Context, itemID string, m platform.Moment) (app.ConcessionList, error) {
	return respond[app.ConcessionList](ctx, "nonconformity.concession.list", map[string]string{"item_id": itemID}, &m)
}

// Confirm — подтвердить несоответствие (nonconformity.nonconformity.confirm).
func (Adapter) Confirm(ctx context.Context, ncID string, in app.ConfirmNonconformity) (platform.Receipt, error) {
	return record(ctx, "nonconformity.nonconformity.confirm", "nonconformity", ncID, in.CommandMeta(), in)
}

// RejectSignal — отклонить сигнал (nonconformity.signal.reject).
func (Adapter) RejectSignal(ctx context.Context, itemID string, in app.RejectSignal) (platform.Receipt, error) {
	return record(ctx, "nonconformity.signal.reject", "item", itemID, in.CommandMeta(), in)
}

// RequestRecheck — назначить доп. проверку (nonconformity.recheck.request).
func (Adapter) RequestRecheck(ctx context.Context, itemID string, in app.RequestRecheck) (platform.Receipt, error) {
	return record(ctx, "nonconformity.recheck.request", "item", itemID, in.CommandMeta(), in)
}

// Isolate — изолировать изделие (nonconformity.item.isolate).
func (Adapter) Isolate(ctx context.Context, itemID string, in app.IsolateItem) (platform.Receipt, error) {
	return record(ctx, "nonconformity.item.isolate", "item", itemID, in.CommandMeta(), in)
}

// ResolvePresentation — решение на точке предъявления (nonconformity.presentation.resolve).
func (Adapter) ResolvePresentation(ctx context.Context, itemID string, in app.ResolvePresentation) (platform.Receipt, error) {
	return record(ctx, "nonconformity.presentation.resolve", "item", itemID, in.CommandMeta(), in)
}

// ReviewPresentation — пересмотр решения на точке (nonconformity.presentation.review).
func (Adapter) ReviewPresentation(ctx context.Context, itemID string, in app.ReviewPresentation) (platform.Receipt, error) {
	return record(ctx, "nonconformity.presentation.review", "item", itemID, in.CommandMeta(), in)
}

// ResolveLot — решение по партии (nonconformity.lot.resolve).
func (Adapter) ResolveLot(ctx context.Context, lotID string, in app.ResolveLot) (platform.Receipt, error) {
	return record(ctx, "nonconformity.lot.resolve", "lot", lotID, in.CommandMeta(), in)
}

// SetDisposition — решение по несоответствию (nonconformity.disposition.set).
func (Adapter) SetDisposition(ctx context.Context, ncID string, in app.SetDisposition) (platform.Receipt, error) {
	return record(ctx, "nonconformity.disposition.set", "nonconformity", ncID, in.CommandMeta(), in)
}

// VerifyDisposition — подтвердить исполнение решения (nonconformity.disposition.verify).
func (Adapter) VerifyDisposition(ctx context.Context, ncID string, in app.VerifyDisposition) (platform.Receipt, error) {
	return record(ctx, "nonconformity.disposition.verify", "nonconformity", ncID, in.CommandMeta(), in)
}

// SetContainment — уровень сдерживания (nonconformity.containment.set).
func (Adapter) SetContainment(ctx context.Context, itemID string, in app.SetContainment) (platform.Receipt, error) {
	return record(ctx, "nonconformity.containment.set", "item", itemID, in.CommandMeta(), in)
}

// ReleaseContainment — снять блок (nonconformity.containment.release).
func (Adapter) ReleaseContainment(ctx context.Context, itemID string, in app.ReleaseContainment) (platform.Receipt, error) {
	return record(ctx, "nonconformity.containment.release", "item", itemID, in.CommandMeta(), in)
}

// RevokeConcession — отозвать разрешение на отклонение (nonconformity.concession.revoke).
func (Adapter) RevokeConcession(ctx context.Context, concessionID string, in app.RevokeConcession) (platform.Receipt, error) {
	return record(ctx, "nonconformity.concession.revoke", "nonconformity", concessionID, in.CommandMeta(), in)
}

// GrantConcession — выдать разрешение на отклонение (nonconformity.concession.grant).
func (Adapter) GrantConcession(ctx context.Context, in app.GrantConcession) (platform.Receipt, error) {
	return record(ctx, "nonconformity.concession.grant", "nonconformity", in.ConcessionID, in.CommandMeta(), in)
}

// WaiveReworkLimit — разрешить сверх лимита доработок (nonconformity.rework_limit.waive).
func (Adapter) WaiveReworkLimit(ctx context.Context, itemID string, in app.WaiveReworkLimit) (platform.Receipt, error) {
	return record(ctx, "nonconformity.rework_limit.waive", "item", itemID, in.CommandMeta(), in)
}

// SetProcessHold — остановить точку процесса (nonconformity.process_hold.set).
func (Adapter) SetProcessHold(ctx context.Context, in app.SetProcessHold) (platform.Receipt, error) {
	return record(ctx, "nonconformity.process_hold.set", "equipment", "", in.CommandMeta(), in)
}

// ReleaseProcessHold — снять остановку (nonconformity.process_hold.release).
func (Adapter) ReleaseProcessHold(ctx context.Context, holdID string, in app.ReleaseProcessHold) (platform.Receipt, error) {
	return record(ctx, "nonconformity.process_hold.release", "equipment", holdID, in.CommandMeta(), in)
}

// Close — закрыть несоответствие (nonconformity.nonconformity.close).
func (Adapter) Close(ctx context.Context, ncID string, in app.CloseNonconformity) (platform.Receipt, error) {
	return record(ctx, "nonconformity.nonconformity.close", "nonconformity", ncID, in.CommandMeta(), in)
}
