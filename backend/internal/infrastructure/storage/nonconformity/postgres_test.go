package nonconformity_test

import (
	"context"
	"sync"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	app "ant/internal/application/nonconformity"
	"ant/internal/application/nonconformity/nctest"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dj "ant/internal/domain/journal"
	store "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

const parts = 4

// copyAPI — отдельная «копия api»: свой пул соединений, свой журнал и свой
// сервис модуля над общей БД (AD-39: гонку решает journal.Append).
func copyAPI(t *testing.T, d *journaltest.DB) (*app.Service, *store.Store) {
	t.Helper()
	pool := d.AppPool(t)
	st := store.NewStore(pool, journaltest.SysClock{})
	codec := &engineapp.Codec{Store: st, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: parts}
	svc := app.NewService(
		app.WithDeps(app.Deps{Journal: st, Codec: codec, Routes: app.DemoRoutes{}}),
		app.WithConfig(app.Config{DomainBuild: dj.ZeroLink.String(), Partitions: parts}),
	)
	return svc, st
}

func codeOf(err error) errcodes.Code {
	if e, ok := platform.AsError(err); ok {
		return e.Code
	}
	return ""
}

func hdr(basis int64) platform.CommandHeader {
	return platform.CommandHeader{CommandID: nctest.ID(), BasisSeq: basis}
}

func reason(s string) app.NCReason { return app.NCReason{Text: s} }

// registered — изделие окна нарушения специального процесса (FR-151):
// несоответствие зарегистрировано правилом, ждёт решения комиссии.
func registered(t *testing.T, st *store.Store, item string) string {
	t.Helper()
	at := time.Now().UTC().Add(-time.Hour)
	run := nctest.RecordP("operation.run.started", item, at, map[string]any{"operation_run_id": "RUN-" + item, "operation_code": "020",
		"step_key": "welding.weld", "equipment_id": "WELD-1", "operator_id": "op-7"}, parts)
	nctest.Partitions = parts
	reg, ncID := nctest.Registered(item, at.Add(10*time.Minute), "RUN-"+item)
	if _, err := st.Append(context.Background(), appjournal.AppendRequest{Batch: []appjournal.Pending{run, reg}}); err != nil {
		t.Fatal(err)
	}
	return ncID
}

// FR-54, AD-39: двойной расход лимита разрешения на отклонение с двух копий
// api (два пула соединений) — одна запись проходит, вторая получает 409
// journal.concession_exhausted; «как есть» без разрешения — 422.
func TestPostgresConcessionDoubleSpend(t *testing.T) {
	d := journaltest.NewDB(t)
	svc1, st1 := copyAPI(t, d)
	svc2, _ := copyAPI(t, d)
	ncA := registered(t, st1, "FL:0101")
	ncB := registered(t, st1, "FL:0102")
	chief := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "chief-1", Role: "head_of_qc"})

	_, err := svc1.SetDisposition(chief, ncA, app.SetDisposition{CommandHeader: hdr(0), Disposition: "use_as_is", Reason: reason("решение комиссии")})
	if codeOf(err) != errcodes.NonconformityConcessionRequired {
		t.Fatalf("«как есть» без разрешения: %v", err)
	}
	if _, err := svc1.GrantConcession(chief, app.GrantConcession{CommandHeader: hdr(0), ConcessionID: "CON-7", Title: "Отклонение режима сварки",
		Kind: "use_as_is", Limit: 1, Reason: reason("решение комиссии, ТУ п. 5.1")}); err != nil {
		t.Fatal(err)
	}
	// Повторная выдача того же разрешения — отказ.
	if _, err := svc2.GrantConcession(chief, app.GrantConcession{CommandHeader: hdr(0), ConcessionID: "CON-7", Title: "дубль",
		Kind: "use_as_is", Limit: 5, Reason: reason("дубль")}); codeOf(err) != errcodes.NonconformityInvalidTransition {
		t.Fatalf("повторная выдача: %v", err)
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, x := range []struct {
		svc *app.Service
		nc  string
	}{{svc1, ncA}, {svc2, ncB}} {
		wg.Go(func() {
			_, errs[i] = x.svc.SetDisposition(chief, x.nc, app.SetDisposition{CommandHeader: hdr(0), Disposition: "use_as_is",
				ConcessionID: "CON-7", Reason: reason("решение комиссии")})
		})
	}
	wg.Wait()
	ok, exhausted := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case codeOf(err) == errcodes.JournalConcessionExhausted:
			exhausted++
			if e, _ := platform.AsError(err); e.Params["remaining"] != "0" || e.Params["concession_id"] != "CON-7" {
				t.Fatalf("параметры отказа: %+v", e.Params)
			}
		default:
			t.Fatalf("неожиданная ошибка: %v", err)
		}
	}
	if ok != 1 || exhausted != 1 {
		t.Fatalf("двойной расход: прошло %d, отказ 409 %d (%v)", ok, exhausted, errs)
	}
	cl, err := svc2.Concessions(chief, "", platform.Moment{})
	if err != nil || len(cl.Items) != 1 || cl.Items[0].Used != 1 || cl.Items[0].Status != "exhausted" {
		t.Fatalf("разрешение после расхода: %+v %v", cl, err)
	}
}

// AD-39, AD-7: решение по устаревшему basis_seq — 409 journal.stale_state;
// повтор команды с тем же command_id — прежняя квитанция.
func TestPostgresStaleStateAndReplay(t *testing.T) {
	d := journaltest.NewDB(t)
	svc, st := copyAPI(t, d)
	nc := registered(t, st, "FL:0201")
	qc := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "qc-1", Role: "quality_inspector"})
	c, err := svc.Card(qc, nc, platform.Moment{})
	if err != nil || !c.Commission || c.Origin != "special_process" || c.Axes.Containment != "item_hold" {
		t.Fatalf("карточка спецпроцесса: %+v %v", c, err)
	}
	h := hdr(c.BasisSeq)
	r1, err := svc.SetContainment(qc, "FL:0201", app.SetContainment{CommandHeader: h, Level: "observe", Reason: reason("наблюдать")})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := svc.SetContainment(qc, "FL:0201", app.SetContainment{CommandHeader: h, Level: "observe", Reason: reason("наблюдать")})
	if err != nil || !r2.Replayed || r2.Seq != r1.Seq {
		t.Fatalf("повтор команды: %+v %v", r2, err)
	}
	_, err = svc.Isolate(qc, "FL:0201", app.IsolateItem{CommandHeader: hdr(c.BasisSeq), Reason: reason("по старой карточке")})
	if codeOf(err) != errcodes.JournalStaleState {
		t.Fatalf("устаревший basis_seq: %v", err)
	}
}
