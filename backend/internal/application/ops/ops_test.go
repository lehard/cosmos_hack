package ops_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	opsapp "ant/internal/application/ops"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/engine"
	"ant/internal/domain/engine/enginetest"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/ops"
)

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

const item = "ENT01:FL-0042"

// rig — журнал в памяти, воркер с ломающейся свёрткой и ops вживую.
type rig struct {
	t      *testing.T
	j      *enginemem.Journal
	codec  *engineapp.Codec
	feed   *enginemem.Feed
	worker *engineapp.WorkerService
	part   engineapp.Partition
	ops    *opsapp.Service
	broken atomic.Bool
}

func newRig(t *testing.T) *rig {
	t.Helper()
	r := &rig{t: t, j: enginemem.New(nil)}
	r.part = engineapp.Partition{Number: 0, Epoch: 1}
	r.j.SetEpoch(appjournal.PartitionLease(0), 1)
	r.codec = &engineapp.Codec{Store: r.j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}
	r.feed = &enginemem.Feed{J: r.j, Parts: []engineapp.Partition{r.part}}
	fold := func(b engine.Bundle, in []kernel.Record) (engine.Snapshot, []kernel.Reaction) {
		if r.broken.Load() {
			panic("правило модуля упало на записи")
		}
		return enginetest.Fold(b, in)
	}
	r.worker = engineapp.NewWorker(engineapp.WorkerConfig{Feed: r.feed, Codec: r.codec, Fold: fold})
	r.ops = opsapp.NewLive(opsapp.Config{Journal: r.j, Codec: r.codec, Partitions: 1, Enabled: []string{"onec"},
		Profile: "demo", Version: "test", Mode: platform.ModeLive, Adapters: map[string]string{"telemetry": "prometheus"}})
	return r
}

func (r *rig) fact(id string) {
	r.t.Helper()
	p, err := r.codec.Encode(context.Background(), engineapp.Out{EventID: id, Type: catalog.InspectionResultRecorded, Kind: catalog.KindFact,
		Stream: "item:" + item, ItemID: item, OccurredAt: t0, Correlation: id, Data: map[string]string{"outcome": "defect_indicated"}})
	if err != nil {
		r.t.Fatal(err)
	}
	if _, err := r.j.Append(context.Background(), appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) step() {
	r.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	works, err := r.feed.Next(ctx, r.part)
	if err != nil {
		r.t.Fatalf("нет работы: %v", err)
	}
	if err := r.worker.Process(context.Background(), r.part, works); err != nil {
		r.t.Fatal(err)
	}
}

// AD-45, FR-127: администратор видит остановленное изделие и повторяет его
// обработку; после исправления воркер сворачивает изделие заново.
func TestStoppedItemAndRetry(t *testing.T) {
	r := newRig(t)
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "ADM-1"})
	r.broken.Store(true)
	r.fact("01929a2b-7c3d-7e4f-8a5b-000000000001")
	r.step()

	list, err := r.ops.StoppedItems(ctx, platform.Page{})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("остановленные: %+v %v", list, err)
	}
	st := list.Items[0]
	if st.ItemID != item || st.Consumer != appjournal.WorkerConsumer || !strings.Contains(st.Error, "паника свёртки") || st.FailedSeq != 1 {
		t.Fatalf("строка: %+v", st)
	}
	h, err := r.ops.Health(ctx)
	if err != nil || h.StoppedItems != 1 {
		t.Fatalf("здоровье: %+v %v", h.StoppedItems, err)
	}

	// Не тот сбой — 409; не остановленное изделие — 404.
	cmd := opsapp.RetryProcessing{CommandHeader: platform.CommandHeader{CommandID: "01929a2b-7c3d-7e4f-8a5b-00000000aa01"},
		FailureEventID: "01929a2b-7c3d-7e4f-8a5b-00000000ffff"}
	if _, err := r.ops.RetryProcessing(ctx, item, cmd); code(err) != errcodes.JournalStaleState {
		t.Fatalf("чужой сбой: %v", err)
	}
	if _, err := r.ops.RetryProcessing(ctx, "ENT01:NOPE", opsapp.RetryProcessing{}); code(err) != errcodes.ApiNotFound {
		t.Fatalf("не остановлено: %v", err)
	}

	// Исправили правило — повтор.
	r.broken.Store(false)
	cmd.FailureEventID = st.FailureEventID
	cmd.Reason = &opsapp.OpsReason{Text: "исправлено правило модуля"}
	rc, err := r.ops.RetryProcessing(ctx, item, cmd)
	if err != nil || rc.Seq == 0 || rc.CommandID != cmd.CommandID {
		t.Fatalf("повтор: %+v %v", rc, err)
	}
	again, err := r.ops.RetryProcessing(ctx, item, cmd)
	if err != nil || !again.Replayed || again.Seq != rc.Seq {
		t.Fatalf("повтор команды (AD-7): %+v %v", again, err)
	}
	var retried int
	for _, e := range r.j.Entries() {
		if e.EventType == string(catalog.OpsProcessingRetried) {
			retried++
			if e.ItemID == nil || *e.ItemID != item || e.EntryKind != "decision" {
				t.Fatalf("запись повтора: %+v", e)
			}
		}
	}
	if retried != 1 {
		t.Fatalf("записей повтора: %d", retried)
	}
	r.step() // воркер снимает «обработка остановлена» и сворачивает изделие
	in, err := r.codec.LoadItem(context.Background(), item, 0)
	if err != nil || in.Stopped {
		t.Fatalf("после повтора: stopped=%v %v", in.Stopped, err)
	}
	if _, ok := r.j.Projection(engineapp.ItemStateProjection)[item]; !ok {
		t.Fatal("проекция изделия построена после повтора")
	}
	list, _ = r.ops.StoppedItems(ctx, platform.Page{})
	if len(list.Items) != 0 {
		t.Fatalf("после повтора остановленных нет: %+v", list.Items)
	}
}

func code(err error) errcodes.Code {
	if e, ok := platform.AsError(err); ok {
		return e.Code
	}
	return ""
}

// Эпик 30: ops.integration.degraded через порт ops — только переходы канала.
func TestReportIntegration(t *testing.T) {
	r := newRig(t)
	rep := &opsapp.Reporter{Journal: r.j, Codec: r.codec, Now: func() time.Time { return t0 }}
	ctx := context.Background()
	steps := []struct {
		state string
		wrote bool
	}{{"ok", false}, {"degraded", true}, {"degraded", false}, {"ok", true}, {"ok", false}, {"disabled", false}}
	for i, s := range steps {
		wrote, err := rep.ReportIntegration(ctx, opsapp.IntegrationReport{System: "onec", State: s.state, Detail: "метаданные 1С не совпали"})
		if err != nil || wrote != s.wrote {
			t.Fatalf("шаг %d (%s): записано %v, ошибка %v", i, s.state, wrote, err)
		}
	}
	if _, err := rep.ReportIntegration(ctx, opsapp.IntegrationReport{System: "sap", State: "degraded"}); err == nil {
		t.Fatal("система вне контракта")
	}
	var n int
	for _, e := range r.j.Entries() {
		if e.EventType == string(catalog.OpsIntegrationDegraded) {
			n++
			if e.Stream != "source:onec" || e.EntryKind != "service" {
				t.Fatalf("запись: %+v", e)
			}
		}
	}
	if n != 2 {
		t.Fatalf("переходов записано %d", n)
	}
	// Стол администратора: последняя запись — «ok».
	h, err := r.ops.Health(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if h.Integrations[0].System != "onec" || h.Integrations[0].State != "ok" || h.Integrations[1].State != "disabled" {
		t.Fatalf("%+v", h.Integrations)
	}
	// Сбой потребителя через порт ops — та же запись, что пишет воркер.
	rq, err := rep.ReportFailure(ctx, opsapp.FailureReport{Consumer: "projector:x", ItemID: item, Seq: 7, Cause: errors.New("сломалась проекция")})
	if err != nil || len(rq.Batch) != 1 || rq.Batch[0].Entry.EventID != dom.FailureEventID("projector:x", item, 7) {
		t.Fatalf("%+v %v", rq, err)
	}
}

// fakeDB — база для самопроверки: миграции и роли по сценарию теста.
type fakeDB struct {
	migrations []dom.Migration
	missing    []string
}

func (fakeDB) Ping(context.Context) error                   { return nil }
func (d fakeDB) Migrations(context.Context) []dom.Migration { return d.migrations }
func (d fakeDB) DBRoles(context.Context) ([]string, []dom.Privilege, error) {
	return d.missing, []dom.Privilege{{Role: "ant_app", Privilege: "UPDATE", Object: "journal.entries", Want: false}}, nil
}

// FR-109: самопроверка находит отсутствие генезиса и неприменённую миграцию.
func TestSelfCheck(t *testing.T) {
	r := newRig(t)
	r.fact("01929a2b-7c3d-7e4f-8a5b-000000000001")
	ok := fakeDB{migrations: []dom.Migration{{Module: "journal", Current: 2, Target: 2}}}
	sc := &opsapp.SelfCheck{Journal: r.j, Database: ok, Roles: []string{"api"}}
	v := sc.Run(context.Background())
	if !v.OK || v.Summary != dom.MsgOK || len(v.Findings) != 1 || v.Findings[0].Check != dom.CheckGenesis {
		t.Fatalf("без обязательного генезиса — предупреждение: %+v", v)
	}
	sc.RequireGenesis = true
	sc.Database = fakeDB{migrations: []dom.Migration{{Module: "journal", Current: 2, Target: 2}, {Module: "erp", Current: 0, Target: 20260927, Pending: 3}}}
	v = sc.Run(context.Background())
	if v.OK || !strings.Contains(v.Summary, "генезиса нет") || !strings.Contains(v.Summary, "модуль erp") {
		t.Fatalf("находки: %+v", v)
	}
	sc.Database = fakeDB{missing: []string{"ant_verifier"}}
	sc.Roles, sc.Pending = []string{"api", "init"}, []string{"init"}
	sc.RequireGenesis = false
	v = sc.Run(context.Background())
	if v.OK || !strings.Contains(v.Summary, "ant_verifier") {
		t.Fatalf("роли БД: %+v", v)
	}
}
