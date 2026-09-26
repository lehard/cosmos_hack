package engine_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	"ant/internal/domain/engine/enginetest"
	"ant/internal/domain/kernel"
)

const item = "ENT01:I-1"

// Сценарий воркера (AD-5): факт → пересвёртка → реакция v1 с basis_seq;
// позднее событие → v2 «пересмотрен из-за записи ‹id›», v1 остаётся в журнале
// (FR-32); курсор, проекция изделия и вклады — в той же записи.
func TestWorkerLateEventRevision(t *testing.T) {
	w := newWorld(t, nil)
	w.fact(item, "e1", 10*time.Minute, "defect_indicated", "")
	w.step()

	sig := w.of(catalog.QualitySignalRaised)
	if len(sig) != 1 {
		t.Fatalf("ждали одну реакцию, есть %d", len(sig))
	}
	v1 := sig[0]
	if v1.BasisSeq == nil || *v1.BasisSeq != 1 || v1.Version == nil || *v1.Version != 1 || v1.Supersedes != nil {
		t.Fatalf("v1: basis=%v version=%v supersedes=%v", v1.BasisSeq, v1.Version, v1.Supersedes)
	}
	if v1.CausationID == nil || *v1.CausationID != "e1" || *v1.ItemID != item || v1.Stream != "item:"+item {
		t.Fatalf("v1: causation/stream: %+v", v1)
	}
	if w.j.Cursor(engineapp.ConsumerWorker, 0) != 1 {
		t.Fatalf("курсор: %d", w.j.Cursor(engineapp.ConsumerWorker, 0))
	}
	var st engineapp.ItemState
	if err := json.Unmarshal(w.j.Projection(engineapp.ItemStateProjection)[item], &st); err != nil || st.BasisSeq != 1 || st.Reactions != 1 {
		t.Fatalf("проекция изделия: %+v %v", st, err)
	}
	if rows := w.j.Contributions(item); rows == nil {
		t.Fatal("вклады изделия должны быть заменены (пустой набор)")
	}

	// Повторная подача без нового входа ничего не пишет (идемпотентность, AD-5).
	before := len(w.j.Entries())
	if err := w.worker.Process(context.Background(), w.part, []engineapp.Work{{ItemID: item, UpToSeq: 1}}); err != nil {
		t.Fatal(err)
	}
	if len(w.j.Entries()) != before {
		t.Fatal("повторная пересвёртка не должна дописывать реакции")
	}

	// Позднее событие: возникло раньше e1, записано позже.
	w.fact(item, "e0", 5*time.Minute, "defect_indicated", "")
	w.step()
	sig = w.of(catalog.QualitySignalRaised)
	if len(sig) != 2 {
		t.Fatalf("ждали две версии слота, есть %d", len(sig))
	}
	v2 := sig[1]
	m := w.reaction(v2)
	if m.Version != 2 || m.Supersedes == nil || *m.Supersedes != v1.EventID || m.RevisedDueTo == nil || *m.RevisedDueTo != "e0" {
		t.Fatalf("v2: %+v", m)
	}
	if *v2.BasisSeq != 3 || len(m.Causes) != 2 {
		t.Fatalf("v2: basis=%d causes=%v", *v2.BasisSeq, m.Causes)
	}
	if v2.EventID != kernel.UUIDv5("4b82fbf1-fc9e-5a06-91c6-8c100ebac4ef", "test.signal\x1fitem:"+item+"\x1fsignal\x1f2") {
		t.Fatalf("reaction_id v2 = UUIDv5(NS_ANT, слот ‖ версия): %s", v2.EventID)
	}
	// Прежняя версия осталась в журнале без изменений.
	if w.of(catalog.QualitySignalRaised)[0].EventID != v1.EventID {
		t.Fatal("v1 должна остаться в журнале")
	}
}

// Защита не снимается при пересвёртке: задача «основание защиты изменилось» (AD-3).
func TestWorkerProtectiveKept(t *testing.T) {
	w := newWorld(t, nil)
	w.fact(item, "e1", time.Minute, "defect_indicated", "")
	w.step()
	w.fact(item, "e2", 2*time.Minute, "no_defect_indicated", "e1")
	w.step()
	if n := len(w.of(catalog.QualitySignalRaised)); n != 1 {
		t.Fatalf("сигнал не снимается и не пересматривается: %d", n)
	}
	tasks := w.of(catalog.TaskTaskCreated)
	if len(tasks) != 1 || *tasks[0].RuleID != engine.RuleProtectionBasisChanged {
		t.Fatalf("задача пересмотра защиты: %+v", tasks)
	}
	if len(w.of(catalog.TaskTaskWithdrawn)) != 0 {
		t.Fatal("задачу не снимать, пока основание не вернулось")
	}
}

// Ошибка на записи изделия → ops.processing.failed, изделие «обработка
// остановлена», партиция продолжает; `ant rebuild --item` повторяет (AD-45).
func TestWorkerFailureStopsItemOnly(t *testing.T) {
	broken := true
	fold := func(b engine.Bundle, in []kernel.Record) (engine.Snapshot, []kernel.Reaction) {
		if broken && len(in) > 0 && in[0].ItemID == "ENT01:BAD" {
			panic("ошибка правила")
		}
		return enginetest.Fold(b, in)
	}
	w := newWorld(t, fold)
	w.fact("ENT01:BAD", "b1", time.Minute, "defect_indicated", "")
	w.fact(item, "g1", time.Minute, "defect_indicated", "")
	w.step()

	failed := w.of(catalog.OpsProcessingFailed)
	if len(failed) != 1 || *failed[0].ItemID != "ENT01:BAD" {
		t.Fatalf("ждали сбой одного изделия: %+v", failed)
	}
	if len(w.of(catalog.QualitySignalRaised)) != 1 {
		t.Fatal("соседнее изделие партиции должно быть обработано")
	}
	if w.j.Cursor(engineapp.ConsumerWorker, 0) != 2 {
		t.Fatalf("курсор партиции продолжает: %d", w.j.Cursor(engineapp.ConsumerWorker, 0))
	}
	if w.tel.counters[engineapp.MetricProcessingFailed] != 1 {
		t.Fatal("метрика сбоев")
	}

	// Новый вход остановленного изделия не сворачивается.
	broken = false
	w.fact("ENT01:BAD", "b2", 2*time.Minute, "defect_indicated", "")
	w.step()
	if n := len(w.of(catalog.QualitySignalRaised)); n != 1 {
		t.Fatalf("остановленное изделие не сворачивается: %d", n)
	}

	// Повтор по `ant rebuild --item`.
	rb := &engineapp.Rebuilder{Codec: w.codec, Fold: fold, Registry: engineapp.NewRegistry(), Partitions: []int{0}}
	rep, err := rb.RebuildItem(context.Background(), "ENT01:BAD", "")
	if err != nil || !rep.Retried {
		t.Fatalf("rebuild --item: %+v %v", rep, err)
	}
	w.step()
	sig := w.of(catalog.QualitySignalRaised)
	if len(sig) != 2 || *sig[1].ItemID != "ENT01:BAD" {
		t.Fatalf("после повтора изделие свёрнуто: %d", len(sig))
	}
	if m := w.reaction(sig[1]); len(m.Causes) != 2 {
		t.Fatalf("свёрнут весь вход, включая пришедший во время остановки: %v", m.Causes)
	}
}

// Запись от копии, потерявшей аренду, отвергается (AD-6).
func TestWorkerFenced(t *testing.T) {
	w := newWorld(t, nil)
	w.fact(item, "e1", time.Minute, "defect_indicated", "")
	w.j.SetEpoch("worker/0", 2)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	works, _ := w.feed.Next(ctx, w.part)
	if err := w.worker.Process(ctx, w.part, works); !errors.Is(err, appjournal.ErrFenced) {
		t.Fatalf("ждали ErrFenced: %v", err)
	}
	if len(w.of(catalog.QualitySignalRaised)) != 0 {
		t.Fatal("ничего не записано")
	}
}

// Сквозной путь «журнал → воркер → проекция → SSE» не дольше 2 с и метрика
// ant_event_to_sse_seconds (FR-2, AD-6).
func TestEndToEndSSE(t *testing.T) {
	w := newWorld(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	live := engineapp.NewLiveUpdates(engineapp.LiveConfig{Log: w.j, Telemetry: w.tel})
	go func() { _ = live.Run(ctx) }()
	go func() { _ = w.worker.Run(ctx) }()

	sub, err := live.Subscribe(ctx, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	start := time.Now()
	w.fact(item, "e1", time.Minute, "defect_indicated", "")
	wctx, wcancel := context.WithTimeout(ctx, 2*time.Second)
	defer wcancel()
	got := map[platform.EntityKind]bool{}
	for !got[platform.EntityItem] || !got[platform.EntityLiveMap] {
		ch, err := sub.Next(wctx)
		if err != nil {
			t.Fatalf("SSE не пришло за 2 с: %v", err)
		}
		got[ch.Entity] = true
		if ch.Entity == platform.EntityItem && (ch.ID != item || ch.Seq != 1 || ch.Mode != platform.ModeLive) {
			t.Fatalf("сообщение: %+v", ch)
		}
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("событие → SSE: %v", d)
	}
	if _, ok := w.j.Projection(engineapp.ItemStateProjection)[item]; !ok {
		t.Fatal("к моменту SSE проекция уже записана")
	}
	obs := w.tel.values(engineapp.MetricEventToSSE)
	if len(obs) == 0 || obs[0] > 2*time.Second || obs[0] < 0 {
		t.Fatalf("метрика %s: %v", engineapp.MetricEventToSSE, obs)
	}

	// Переподключение с Last-Event-ID догоняет по seq.
	again, err := live.Subscribe(ctx, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	c1, err := again.Next(wctx)
	if err != nil || c1.Seq != 1 {
		t.Fatalf("догоняющая подписка: %+v %v", c1, err)
	}
	// Фильтр прогона (AD-38).
	other, _ := live.Subscribe(ctx, 0, "run-X")
	defer other.Close()
	sctx, scancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer scancel()
	if _, err := other.Next(sctx); err == nil {
		t.Fatal("изменения чужого прогона не приходят")
	}
}

// Запрос состояния на момент — свёртка префикса без записи (AD-22).
func TestStateAt(t *testing.T) {
	w := newWorld(t, nil)
	w.fact(item, "e1", time.Minute, "defect_indicated", "")
	w.fact(item, "e2", 2*time.Minute, "no_defect_indicated", "e1")
	n := len(w.j.Entries())
	q := engineapp.StateQueries{Codec: w.codec, Fold: enginetest.Fold}

	now, err := q.Item(context.Background(), item, platform.Moment{Axis: platform.AxisOccurred})
	if err != nil || len(now.Reactions) != 0 {
		t.Fatalf("сейчас: исправлено, сигнала нет: %+v %v", now.Reactions, err)
	}
	at := t0.Add(90 * time.Second)
	was, err := q.Item(context.Background(), item, platform.Moment{Axis: platform.AxisOccurred, AsOf: &at})
	if err != nil || len(was.Reactions) != 1 || was.StateHash == now.StateHash {
		t.Fatalf("как было на t0+1.5м: %+v %v", was.Reactions, err)
	}
	if len(w.j.Entries()) != n {
		t.Fatal("запрос на момент ничего не пишет")
	}
	if _, err := q.Item(context.Background(), "ENT01:NONE", platform.Moment{}); err == nil {
		t.Fatal("нет изделия — ошибка")
	}
}

// Полная пересборка проекций даёт те же значения и тот же rebuild_hash (AD-6, FR-115).
func TestRebuildAll(t *testing.T) {
	w := newWorld(t, nil)
	reg := engineapp.NewRegistry()
	count := engineapp.GlobalProjection{
		Name: "test.count", Writer: "ops",
		Keys: func(r kernel.Record) []string { return []string{r.ItemID} },
		Step: func(_ string, prev json.RawMessage, _ kernel.Record) (json.RawMessage, error) {
			var n int
			_ = json.Unmarshal(prev, &n)
			return json.Marshal(n + 1)
		},
	}
	if err := reg.AddGlobal(count); err != nil {
		t.Fatal(err)
	}
	if err := reg.AddGlobal(count); err == nil || !strings.Contains(err.Error(), "один писатель") {
		t.Fatalf("второй писатель проекции: %v", err)
	}
	w.fact(item, "e1", time.Minute, "defect_indicated", "")
	w.fact("ENT01:I-2", "e2", time.Minute, "no_defect_indicated", "")
	w.step()
	before := w.j.Projection(engineapp.ItemStateProjection)

	rb := &engineapp.Rebuilder{Codec: w.codec, Fold: enginetest.Fold, Registry: reg, Partitions: []int{0}}
	r1, err := rb.RebuildAll(context.Background())
	if err != nil || r1.Items != 2 || r1.Globals != 1 {
		t.Fatalf("rebuild: %+v %v", r1, err)
	}
	after := w.j.Projection(engineapp.ItemStateProjection)
	for k, v := range before {
		if string(after[k]) != string(v) {
			t.Fatalf("проекция %s разошлась: %s != %s", k, after[k], v)
		}
	}
	if got := string(w.j.Projection("test.count")[item]); got != "2" {
		t.Fatalf("глобальная проекция: %s", got)
	}
	r2, err := rb.RebuildAll(context.Background())
	if err != nil || r2.RebuildHash != r1.RebuildHash {
		t.Fatalf("rebuild_hash: %s != %s (%v)", r2.RebuildHash, r1.RebuildHash, err)
	}
	if c := w.j.Cursor(engineapp.ConsumerName("test.count"), engineapp.GlobalPartition); c != int64(len(w.j.Entries())) {
		t.Fatalf("курсор глобальной проекции после пересборки: %d", c)
	}
}

// Роль projector: глобальная проекция под арендой лидера, курсор и значения атомарно.
func TestProjectorAndLeader(t *testing.T) {
	w := newWorld(t, nil)
	reg := engineapp.NewRegistry()
	_ = reg.AddGlobal(engineapp.GlobalProjection{
		Name: "test.last", Writer: "ops",
		Keys: func(r kernel.Record) []string { return []string{r.ItemID} },
		Step: func(_ string, _ json.RawMessage, r kernel.Record) (json.RawMessage, error) {
			return json.Marshal(r.EventID)
		},
		Entity: func(k string) (platform.EntityKind, string, bool) { return platform.EntityItem, k, true },
	})
	w.fact(item, "e1", time.Minute, "no_defect_indicated", "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pr := &engineapp.Projector{Consumer: w.j, Codec: w.codec, Store: w.j, Registry: reg}
	a := engineapp.Leader{Leases: w.j, Name: "projector", Holder: "a", TTL: 300 * time.Millisecond}
	b := engineapp.Leader{Leases: w.j, Name: "projector", Holder: "b", TTL: 300 * time.Millisecond}
	go func() { _ = a.Run(ctx, pr.Run) }()
	go func() { _ = b.Run(ctx, pr.Run) }()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if v := w.j.Projection("test.last")[item]; string(v) == `"e1"` {
			if w.j.Cursor(engineapp.ConsumerName("test.last"), engineapp.GlobalPartition) < 1 {
				t.Fatal("курсор не сдвинулся с проекцией")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("глобальная проекция не построена")
}
