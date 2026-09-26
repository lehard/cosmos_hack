package item

import (
	"context"
	"strings"

	app "ant/internal/application/item"
	"ant/internal/application/platform"
)

// Adapter — реализация fixtures ведущих портов модуля item (AD-36): поиск,
// список, паспорт, журнал изменений и генеалогия изделий — из мира заготовок;
// команды двигают сценарий, если он ждёт именно этого решения.
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

func itemP(id string) map[string]string { return map[string]string{"item_id": id} }

// Lookup — изделие по номеру детали или DataMatrix (item.item.lookup, AD-41).
// Номер нормализуется: носитель ant:carrier:‹тип›:‹значение› — по значению,
// код предприятия снимается, кириллица номера («Ф-017», «К-101», «КР-001»)
// переводится в латиницу машинных ID (F-017, R-101, C-001).
func (Adapter) Lookup(ctx context.Context, q string, m platform.Moment) (app.ItemLookup, error) {
	return respond[app.ItemLookup](ctx, "item.item.lookup", map[string]string{"q": NormalizeNumber(q)}, &m)
}

// NormalizeNumber приводит номер детали или содержимое скана к локальному
// машинному номеру изделия.
func NormalizeNumber(q string) string {
	q = strings.TrimSpace(q)
	if strings.HasPrefix(q, "ant:carrier:") {
		q = q[strings.LastIndexByte(q, ':')+1:]
	}
	if i := strings.IndexByte(q, ':'); i >= 0 {
		q = q[i+1:]
	}
	// Префикс прогона (‹run_id›/…) — в нижнем регистре, его снимает загрузчик.
	run, local := "", q
	if i := strings.LastIndexByte(q, '/'); i >= 0 {
		run, local = q[:i+1], q[i+1:]
	}
	local = strings.ToUpper(local)
	for _, r := range [][2]string{{"КР-", "C-"}, {"Ф-", "F-"}, {"К-", "R-"}} {
		if strings.HasPrefix(local, r[0]) {
			local = r[1] + local[len(r[0]):]
			break
		}
	}
	return run + local
}

// List — изделия с фильтром (item.item.list).
func (Adapter) List(ctx context.Context, f app.ItemFilter, m platform.Moment, _ platform.Page) (app.ItemList, error) {
	v, err := respond[app.ItemList](ctx, "item.item.list",
		map[string]string{"step_key": f.StepKey, "summary": f.Summary, "lot_id": f.LotID, "order_id": f.OrderID}, &m)
	if err != nil {
		return v, err
	}
	out := v.Items[:0]
	for _, x := range v.Items {
		if (f.StepKey != "" && x.StepKey != f.StepKey) || (f.Summary != "" && x.Status.Summary != f.Summary) || (f.LotID != "" && x.LotID != f.LotID) {
			continue
		}
		out = append(out, x)
	}
	v.Items = out
	return v, nil
}

// Passport — паспорт изделия (item.passport.read, FR-42).
func (Adapter) Passport(ctx context.Context, itemID string, m platform.Moment) (app.ItemPassport, error) {
	return respond[app.ItemPassport](ctx, "item.passport.read", itemP(itemID), &m)
}

// History — журнал изменений паспорта (item.history.list, FR-43).
func (Adapter) History(ctx context.Context, itemID string, m platform.Moment, _ platform.Page) (app.ItemHistory, error) {
	return respond[app.ItemHistory](ctx, "item.history.list", itemP(itemID), &m)
}

// Genealogy — генеалогия изделия (item.genealogy.read, FR-45).
func (Adapter) Genealogy(ctx context.Context, itemID string, m platform.Moment) (app.ItemGenealogy, error) {
	return respond[app.ItemGenealogy](ctx, "item.genealogy.read", itemP(itemID), &m)
}

// Register — зарегистрировать изделие (item.item.register).
func (Adapter) Register(ctx context.Context, in app.RegisterItem) (platform.Receipt, error) {
	return decide(ctx, "item.item.register", "item", "", in.CommandMeta())
}

// ApplyCarrier — нанести носитель (item.carrier.apply).
func (Adapter) ApplyCarrier(ctx context.Context, itemID string, in app.ApplyCarrier) (platform.Receipt, error) {
	return decide(ctx, "item.carrier.apply", "item", itemID, in.CommandMeta())
}

// RemoveCarrier — снять носитель (item.carrier.remove).
func (Adapter) RemoveCarrier(ctx context.Context, itemID string, in app.RemoveCarrier) (platform.Receipt, error) {
	return decide(ctx, "item.carrier.remove", "item", itemID, in.CommandMeta())
}

// RecordPresentation — предъявить ОТК (item.presentation.record).
func (Adapter) RecordPresentation(ctx context.Context, itemID string, in app.RecordPresentation) (platform.Receipt, error) {
	return decide(ctx, "item.presentation.record", "item", itemID, in.CommandMeta())
}

// OpenIntervention — открыть вмешательство (item.intervention.open).
func (Adapter) OpenIntervention(ctx context.Context, itemID string, in app.OpenIntervention) (platform.Receipt, error) {
	return decide(ctx, "item.intervention.open", "item", itemID, in.CommandMeta())
}

// CloseIntervention — закрыть вмешательство (item.intervention.close).
func (Adapter) CloseIntervention(ctx context.Context, itemID, _ string, in app.CloseIntervention) (platform.Receipt, error) {
	return decide(ctx, "item.intervention.close", "item", itemID, in.CommandMeta())
}

// ConfirmIdentification — подтвердить идентификацию (item.identification.confirm).
func (Adapter) ConfirmIdentification(ctx context.Context, itemID string, in app.ConfirmIdentification) (platform.Receipt, error) {
	return decide(ctx, "item.identification.confirm", "item", itemID, in.CommandMeta())
}

// RecordAssembly — установить компонент (item.assembly.record).
func (Adapter) RecordAssembly(ctx context.Context, itemID string, in app.RecordAssembly) (platform.Receipt, error) {
	return decide(ctx, "item.assembly.record", "item", itemID, in.CommandMeta())
}

// RecordRelease — принять на склад выпуска (item.release.record).
func (Adapter) RecordRelease(ctx context.Context, itemID string, in app.RecordRelease) (platform.Receipt, error) {
	return decide(ctx, "item.release.record", "item", itemID, in.CommandMeta())
}

// Split — разделение 1→N (item.item.split, FR-15).
func (Adapter) Split(ctx context.Context, itemID string, in app.SplitItem) (platform.Receipt, error) {
	return decide(ctx, "item.item.split", "item", itemID, in.CommandMeta())
}
