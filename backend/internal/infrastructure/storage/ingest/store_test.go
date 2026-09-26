package ingest_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/application/ingest/inmem"
	ingeststore "ant/internal/infrastructure/storage/ingest"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
	"ant/internal/infrastructure/storage/journal/migrator"
)

// Приём на настоящем журнале (эпик 04) и своём хранилище в Postgres: пять
// случаев FR-29, повтор без изменения показателей, реестр и карантин в
// транзакции журнала, миграция ingest повторно ничего не меняет.
// Без ANT_DB_HOST пропускается (make dev-db).
func TestIngestOnPostgres(t *testing.T) {
	db := journaltest.NewDB(t)
	ctx := context.Background()
	set := migrator.Set{Module: "ingest", FS: ingeststore.Migrations, Dir: ingeststore.MigrationsDir}
	if a, err := migrator.Up(ctx, db.Admin.ConnConfig, nil, set); err != nil || len(a) != 1 {
		t.Fatalf("миграция ingest: %v %v", a, err)
	}
	if a, err := migrator.Up(ctx, db.Admin.ConnConfig, nil, set); err != nil || len(a) != 0 {
		t.Fatalf("повтор миграции: %v %v", a, err)
	}
	pool := db.AppPool(t)
	js := journalstore.NewStore(pool, journaltest.SysClock{})
	st := ingeststore.NewStore(pool)
	cfg := app.DefaultConfig()
	svc := app.NewService(app.WithConfig(cfg), app.WithDeps(app.Deps{Journal: js, Registry: st, Quarantine: st,
		DomainClock: inmem.SystemClock{}, InfraClock: inmem.SystemInfra{}}))

	files, _ := filepath.Glob("../../../../../contracts/events/examples/contract-change/*.json")
	if len(files) != 6 {
		t.Fatalf("примеры: %v", files)
	}
	want := map[string]app.Outcome{"01-unknown-version.json": app.OutcomeQuarantined, "02-new-optional-field.json": app.OutcomeAccepted,
		"03-missing-required-field.json": app.OutcomeQuarantined, "04a-unknown-enum-value.json": app.OutcomeAcceptedWithFlag,
		"04b-unknown-enum-value-critical.json": app.OutcomeQuarantined, "05-incompatible-change.json": app.OutcomeQuarantined}
	var accepted []byte
	for _, f := range files {
		b, _ := os.ReadFile(f)
		var ex struct {
			Message json.RawMessage `json:"message"`
		}
		_ = json.Unmarshal(b, &ex)
		r, err := svc.Ingest(ctx, ex.Message)
		if err != nil {
			t.Fatal(err)
		}
		if r.Outcome != want[filepath.Base(f)] {
			t.Fatalf("%s: %+v", filepath.Base(f), r)
		}
		if r.Outcome == app.OutcomeAccepted {
			accepted = ex.Message
		}
	}
	heads, _ := js.Head(ctx)
	for range 3 {
		if r, _ := svc.Ingest(ctx, accepted); r.Outcome != app.OutcomeDuplicate || r.Seq == 0 {
			t.Fatalf("повтор: %+v", r)
		}
	}
	if h2, _ := js.Head(ctx); h2.MainSeq != heads.MainSeq {
		t.Fatalf("повтор записал в журнал: %d → %d", heads.MainSeq, h2.MainSeq)
	}
	// Повтор уже карантинного — без новой записи.
	b, _ := os.ReadFile(files[0])
	var ex struct {
		Message json.RawMessage `json:"message"`
	}
	_ = json.Unmarshal(b, &ex)
	if r, _ := svc.Ingest(ctx, ex.Message); !r.Replayed {
		t.Fatalf("повтор карантина: %+v", r)
	}
	if h2, _ := js.Head(ctx); h2.MainSeq != heads.MainSeq {
		t.Fatal("повтор карантина записал в журнал")
	}
	// Конфликт содержимого: карантин, событие security и запись цепочки CA.
	var ev map[string]any
	_ = json.Unmarshal(accepted, &ev)
	ev["data"].(map[string]any)["station_id"] = "weld-3"
	bad, _ := json.Marshal(ev)
	r, err := svc.Ingest(ctx, bad)
	if err != nil || r.Outcome != app.OutcomeConflict || r.CARef != "CA-1" {
		t.Fatalf("конфликт: %+v %v", r, err)
	}
	list, err := svc.Quarantine(ctx, app.QuarantineFilter{State: "open"}, platformPage())
	if err != nil || len(list.Items) != 5 {
		t.Fatalf("карантин: %+v %v", list, err)
	}
	e, err := svc.QuarantineEntry(ctx, list.Items[0].QuarantineID)
	if err != nil || e.Content == nil || e.QuarantinedAt.After(time.Now()) {
		t.Fatalf("запись карантина: %+v %v", e, err)
	}
	m, err := svc.Metrics(ctx)
	if err != nil || m.Accepted != 2 || m.Duplicates != 3 || m.QuarantineOpen != 5 {
		t.Fatalf("метрики: %+v %v", m, err)
	}
	srcs, err := svc.Sources(ctx, platformPage())
	if err != nil || len(srcs.Items) == 0 {
		t.Fatalf("источники: %+v %v", srcs, err)
	}
}
