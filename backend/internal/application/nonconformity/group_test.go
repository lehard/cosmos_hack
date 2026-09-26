package nonconformity_test

import (
	"context"
	"testing"
	"time"

	appjournal "ant/internal/application/journal"
	app "ant/internal/application/nonconformity"
	"ant/internal/application/nonconformity/nctest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	nc "ant/internal/domain/nonconformity"
)

// FR-151, S05 (NC-G1): групповое несоответствие окна спецпроцесса — один id
// на окно; решение комиссии приходит в поток каждого изделия группы одной
// командой, а проверка исполнения — только изделию, чьи результаты в ней.
func TestGroupDispositionReachesEveryItem(t *testing.T) {
	w := nctest.NewWorld(t)
	svc := w.Service(app.DemoRoutes{})
	window := nctest.ID()
	for _, it := range []string{"FL:0001", "FL:0002"} {
		w.Add(nctest.Run(it, nctest.T0, "RUN-"+it, "op-7"), nctest.RegisteredIn(it, nctest.T0.Add(10*time.Minute), "RUN-"+it, window))
	}
	w.Settle()
	ncID := nc.WindowNCID(window)
	ctx := nctest.As("tech-1", "technologist")
	r, err := svc.SetDisposition(ctx, ncID, app.SetDisposition{CommandHeader: hdr(0), Disposition: "rework", Reason: reason("Сварка вне режима ТП: переделка")})
	if err != nil || len(r.EventIDs) != 2 {
		t.Fatalf("групповое решение: %+v %v", r, err)
	}
	w.Settle()
	for _, it := range []string{"FL:0001", "FL:0002"} {
		es, err := w.J.Read(context.Background(), appjournal.ReadQuery{Stream: "item:" + it, EventType: string(catalog.DecisionDispositionSet)})
		if err != nil || len(es) != 1 {
			t.Fatalf("%s: решение не пришло в поток изделия: %d %v", it, len(es), err)
		}
	}
	// Повтор той же команды — прежняя квитанция, без новых записей (AD-7).
	h := hdr(0)
	h.CommandID = r.CommandID
	again, err := svc.SetDisposition(ctx, ncID, app.SetDisposition{CommandHeader: h, Disposition: "rework", Reason: reason("повтор")})
	if err != nil || !again.Replayed || len(again.EventIDs) != 2 {
		t.Fatalf("повтор групповой команды: %+v %v", again, err)
	}
	insp := nctest.Inspection("FL:0002", nctest.T0.Add(2*time.Hour), "Z-1")
	w.Add(insp)
	w.Settle()
	w.Now = w.Now.Add(time.Hour)
	v, err := svc.VerifyDisposition(ctx, ncID, app.VerifyDisposition{CommandHeader: hdr(0), RecheckEventIDs: []string{insp.Entry.EventID}})
	if err != nil || len(v.EventIDs) != 1 {
		t.Fatalf("проверка исполнения: %+v %v", v, err)
	}
	w.Settle()
	for it, want := range map[string]int{"FL:0001": 0, "FL:0002": 1} {
		es, err := w.J.Read(context.Background(), appjournal.ReadQuery{Stream: "item:" + it, EventType: string(catalog.DecisionDispositionVerified)})
		if err != nil || len(es) != want {
			t.Fatalf("%s: проверок исполнения %d, ожидалось %d (%v)", it, len(es), want, err)
		}
	}
	c, err := svc.Card(context.Background(), ncID, platform.Moment{})
	if err != nil || c.Axes.Disposition != "rework" || len(c.GroupItemIDs) != 2 {
		t.Fatalf("карточка группового несоответствия: %+v %v", c.Axes, err)
	}
}
