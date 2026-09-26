package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	erpapp "ant/internal/application/erp"
	appingest "ant/internal/application/ingest"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/erp"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	"ant/internal/infrastructure/integration/erp/onec"
	onecstand "ant/internal/infrastructure/integration/erp/onec/stand"
	"ant/internal/infrastructure/integration/ingest/stands"
	enginestore "ant/internal/infrastructure/storage/engine"
	erpstore "ant/internal/infrastructure/storage/erp"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/feed"
	"ant/internal/infrastructure/storage/journal/journaltest"
	"ant/internal/infrastructure/storage/journal/migrator"
)

// Сквозной путь модуля erp на своей БД (make dev-db), эпик 30:
// событие-сообщение процесса «в 1С» → реакция erp.posting.requested (роль
// projector) → очередь исходящих и отправка (роль outbox) → stand 1С (OData +
// HTTP-сервис qc.v1, состояние в схеме stand_onec) → квитанция
// erp.posting.responded в журнале → ось «учёт в 1С» и журнал обмена (проекции,
// операции erp.*). Сбои stand-а: 503 → повтор с тем же номером; 422 →
// карантин и ручная переотправка; дубль (потерянный ответ) → та же квитанция;
// несовместимые метаданные → канал degraded, отправки нет. Повторный проход
// реакции по журналу (воспроизведение) ничего не добавляет и не отправляет.

type erpRig struct {
	t        *testing.T
	ctx      context.Context
	journal  *journalstore.Store
	codec    *engineapp.Codec
	engine   *enginestore.Store
	stand    *onecstand.Stand
	registry *stands.Registry
	outbox   *erpapp.Outbox
	service  *erpapp.Service
	reactor  *erpapp.Reactor
	n        int
}

var erpT0 = time.Date(2026, 9, 23, 7, 0, 0, 0, time.UTC)

func newERPRig(t *testing.T) *erpRig {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)
	db := journaltest.NewDB(t)
	if _, err := migrator.Up(ctx, db.Admin.ConnConfig, nil,
		migrator.Set{Module: "erp", FS: erpstore.Migrations, Dir: erpstore.MigrationsDir},
		migrator.Set{Module: "stand_onec", FS: onecstand.Migrations, Dir: onecstand.MigrationsDir}); err != nil {
		t.Fatal(err)
	}
	pool := db.AppPool(t)
	infra := journaltest.SysClock{}
	r := &erpRig{t: t, ctx: ctx}
	r.journal = journalstore.NewStore(pool, infra, journalstore.WithEffects(erpstore.ApplyEffect))
	leases, listener := journalstore.NewLeases(pool, infra), journalstore.NewListener(pool, nil)
	go func() { _ = listener.Run(ctx) }()
	r.engine = &enginestore.Store{Pool: pool}
	t.Cleanup(r.engine.Close)
	r.codec = &engineapp.Codec{Store: r.journal, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1",
		DomainBuild: dj.ZeroLink.String(), Partitions: 4}

	st, err := onecstand.New(onecstand.Options{Store: onecstand.Postgres{Pool: pool}})
	if err != nil {
		t.Fatal(err)
	}
	r.stand = st
	r.registry = stands.NewRegistry(st, onecstand.Alias(st))
	srv := httptest.NewServer(r.registry.Handler())
	t.Cleanup(srv.Close)
	client, err := onec.New(onec.Config{BaseURL: srv.URL + "/stand/1c/erp", Stand: true, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	env, err := erpEnv()
	if err != nil {
		t.Fatal(err)
	}
	opts := func(role string) feed.Options { return feed.Options{Holder: "erp-test/" + role, TTL: 3 * time.Second} }
	store := erpstore.NewStore(pool)

	// Роль projector: реакция erp и проекции erp.*.
	r.reactor = &erpapp.Reactor{Consumer: feed.NewConsumer(r.journal, leases, listener, opts("projector")), Codec: r.codec, Store: r.engine, Env: env}
	go func() { _ = r.reactor.Run(ctx) }()
	reg := engineapp.NewRegistry()
	erpapp.MustRegister(reg)
	pr := &engineapp.Projector{Consumer: feed.NewConsumer(r.journal, leases, listener, opts("projector")), Codec: r.codec, Store: r.engine, Registry: reg}
	lead := func(name string) engineapp.Leader {
		return engineapp.Leader{Leases: leases, Name: name, Holder: "erp-test", TTL: 3 * time.Second}
	}
	go func() { _ = lead("projector").Run(ctx, pr.Run) }()

	// Роль outbox: быстрые повторы и сверки для теста.
	r.outbox = &erpapp.Outbox{Journal: r.journal, Consumer: feed.NewConsumer(r.journal, leases, listener, opts("outbox")), Codec: r.codec,
		Store: store, Ledger: client, Retry: erpapp.RetryPolicy{Delays: []time.Duration{150 * time.Millisecond}, Max: 5},
		Poll: 50 * time.Millisecond, Recheck: 200 * time.Millisecond, Log: slog.New(slog.DiscardHandler)}
	go func() { _ = lead("outbox").Run(ctx, r.outbox.Run) }()

	r.service = erpapp.NewLive(erpapp.Config{Projections: r.engine, Outbox: store,
		Decisions: erpapp.JournalDecisions{Journal: r.journal, DomainBuild: dj.ZeroLink.String()}, Channels: []erpapp.LedgerInfo{client.Info()}})
	return r
}

func (r *erpRig) id() string {
	r.n++
	return kernel.UUIDv5(constants.NsAnt, "erp-e2e\x1f"+r.t.Name()+"\x1f"+string(rune('a'+r.n%26))+time.Now().Format(time.RFC3339Nano))
}

func (r *erpRig) put(o engineapp.Out) string {
	r.t.Helper()
	if o.EventID == "" {
		o.EventID = r.id()
	}
	if o.OccurredAt.IsZero() {
		r.n++
		o.OccurredAt = erpT0.Add(time.Duration(r.n) * time.Minute)
	}
	pd, err := r.codec.Encode(r.ctx, o)
	if err != nil {
		r.t.Fatal(err)
	}
	if _, err := r.journal.Append(r.ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{pd}}); err != nil {
		r.t.Fatal(err)
	}
	return o.EventID
}

func (r *erpRig) register(item string) {
	oid := ev.ObjectID("ORD-0917")
	r.put(engineapp.Out{Type: catalog.ItemItemRegistered, Kind: catalog.KindFact, Stream: "item:" + item, ItemID: item,
		Data: ev.ItemItemRegisteredV1{ItemID: ev.ItemID(item), ItemTypeID: "FL-100.00.000", ItemRevision: "Б", OrderID: &oid,
			ProcessVersionHash: ev.Digest("streebog256:" + strings.Repeat("0", 64)), NormativeRev: "r1"}})
}

// thrown — событие-сообщение процесса «в 1С» (реакция process, эпик 17).
func (r *erpRig) thrown(item, step string, a dom.Action) {
	slot := engineapp.SlotMeta{RuleID: "process.message", Subject: "item:" + item, TriggerKey: step}
	r.put(engineapp.Out{Type: catalog.OperationMessageThrown, Kind: catalog.KindReaction, Stream: "item:" + item, ItemID: item,
		Reaction: &engineapp.ReactionMeta{RuleID: "process.message", AutomationMode: 1, Slot: slot, Version: 1, Causes: []string{}},
		Data:     ev.OperationMessageThrownV1{StepKey: ev.StepKey(step), MessageRef: "Msg", ErpAction: ev.OperationMessageThrownV1ErpAction(a), ClosingBasis: []ev.UUID{}}})
}

type exchange struct {
	Type    catalog.Type
	Key     string
	Outcome string
	Cause   string
	Attempt int
	Seq     int64
}

// exchanges — записи обмена в журнале по порядку.
func (r *erpRig) exchanges() []exchange {
	var out []exchange
	for _, e := range journaltest.ReadAll(r.t, r.journal, "main") {
		if !dom.Is(catalog.Type(e.EventType), dom.Exchange) {
			continue
		}
		d, err := r.codec.Decode(r.ctx, e)
		if err != nil {
			r.t.Fatal(err)
		}
		var x struct {
			BusinessKey string `json:"business_key"`
			Outcome     string `json:"outcome"`
			Cause       string `json:"cause"`
			Attempt     int    `json:"attempt"`
		}
		_ = json.Unmarshal(d.Record.Data, &x)
		out = append(out, exchange{Type: d.Record.Type, Key: x.BusinessKey, Outcome: x.Outcome, Cause: x.Cause, Attempt: x.Attempt, Seq: d.Record.Seq})
	}
	return out
}

func (r *erpRig) count(t catalog.Type, key, outcome string) int {
	n := 0
	for _, x := range r.exchanges() {
		if x.Type == t && x.Key == key && (outcome == "" || x.Outcome == outcome || x.Cause == outcome) {
			n++
		}
	}
	return n
}

func (r *erpRig) wait(what string, cond func() bool) {
	r.t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for !cond() {
		if time.Now().After(deadline) || r.ctx.Err() != nil {
			r.t.Fatalf("не дождались: %s; обмен: %+v", what, r.exchanges())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (r *erpRig) fault(f appingest.Fault) {
	r.t.Helper()
	if err := r.registry.SetFault(r.ctx, onecstand.AliasName, f); err != nil {
		r.t.Fatal(err)
	}
}

func (r *erpRig) clear() { _ = r.registry.ClearFaults(r.ctx, onecstand.Name) }

func (r *erpRig) standMessage(key string) (onecstand.Message, int) {
	snap, err := r.stand.Snapshot(r.ctx)
	if err != nil {
		r.t.Fatal(err)
	}
	var m onecstand.Message
	docs := 0
	for _, x := range snap.Messages {
		if x.BusinessKey == key {
			m = x
		}
	}
	for _, d := range snap.Documents {
		if d.MessageID == m.MessageID && m.MessageID != "" {
			docs++
		}
	}
	return m, docs
}

func TestERPEndToEnd(t *testing.T) {
	r := newERPRig(t)
	const item = "ENT01:F-001"
	transfer := item + "/warehouse_transfer/welding.erp_transfer_in"
	release := item + "/release/final.erp_release"
	r.wait("канал ok", func() bool {
		ch, _ := r.service.Channels(r.ctx)
		return len(ch.Items) == 1 && ch.Items[0].State == "ok"
	})

	// 1. Смена склада и выпуск; на выпуск Ф-001 stand отвечает 503 (сбой по
	// образцу из сценария S01) — повтор с тем же номером, затем квитанция.
	r.fault(appingest.Fault{Kind: appingest.FaultError, Param: 503, Match: "release_good:F-001", Detail: "503 Service Unavailable"})
	r.register(item)
	r.thrown(item, "welding.erp_transfer_in", dom.WarehouseTransfer)
	r.thrown(item, "final.erp_release", dom.Release)
	r.wait("квитанция на смену склада", func() bool { return r.count(catalog.ErpPostingResponded, transfer, "accepted") == 1 })
	r.wait("503 на выпуск", func() bool { m, _ := r.standMessage(release); return len(m.Faults) > 0 })
	r.clear()
	r.wait("квитанция на выпуск после повтора", func() bool { return r.count(catalog.ErpPostingResponded, release, "accepted") == 1 })
	for _, x := range r.exchanges() {
		if x.Key == release && x.Type == catalog.ErpPostingResponded && x.Attempt < 2 {
			t.Fatalf("выпуск подтверждён с попытки %d — повтора после 503 не было", x.Attempt)
		}
		if x.Type == catalog.ErpPostingQuarantined && x.Key == release {
			t.Fatal("503 — повтор, а не карантин")
		}
	}
	if m, docs := r.standMessage(release); docs != 1 || m.Receipt == nil {
		t.Fatalf("выпуск в 1С: документов %d, %+v", docs, m)
	}

	// 2. Возврат поставщику партии LOT-R-117: 422 «не найден договор» → карантин;
	// после исправления — ручная переотправка тем же номером → квитанция.
	lot := "LOT-R-117"
	ret := lot + "/return_to_supplier/ZT-1"
	r.fault(appingest.Fault{Kind: appingest.FaultError, Param: 422, Match: "return_to_supplier:" + lot, Detail: "422 не найден договор с контрагентом ctr-0b19-0003"})
	r.put(engineapp.Out{Type: catalog.ErpLotReceived, Kind: catalog.KindFact, Stream: "lot:" + lot,
		Data: ev.ErpLotReceivedV1{LotID: ev.ObjectID(lot), ExternalSystem: "onec", ExternalNumber: "ПТ00-000217", SupplierID: "SUP-3", ItemTypeID: "FL-100.01.003", Quantity: 10}})
	r.put(engineapp.Out{Type: catalog.DecisionLotResolved, Kind: catalog.KindDecision, Stream: "lot:" + lot,
		Data: ev.DecisionLotResolvedV1{LotID: ev.ObjectID(lot), Resolution: "reject", MethodEventIds: []ev.UUID{}, Reason: &ev.Reason{Text: "Входной брак партии П-117"}}})
	r.wait("карантин возврата поставщику", func() bool {
		return r.count(catalog.ErpPostingResponded, ret, "rejected") == 1 && r.count(catalog.ErpPostingQuarantined, ret, "data_error") == 1
	})
	if _, docs := r.standMessage(ret); docs != 0 {
		t.Fatal("отклонённое сообщение не создаёт документ 1С")
	}
	r.wait("результат контроля партии", func() bool {
		return r.count(catalog.ErpPostingResponded, lot+"/inspection_result/ZT-1", "accepted") == 1
	})
	r.clear()
	var view ev.ErpPostingRequestedV1
	r.wait("проекция сообщения в карантине", func() bool {
		m, err := r.service.Message(r.ctx, ret, platform.Moment{})
		view.BusinessKey = m.BusinessKey
		return err == nil && m.Status == "quarantined"
	})
	m, _ := r.service.Message(r.ctx, ret, platform.Moment{})
	if _, err := r.service.ResendPosting(r.ctx, ret, erpapp.ResendPosting{RequestEventID: m.RequestEventID, Reason: erpapp.ErpReason{Text: "исправлено соответствие договора"}}); err != nil {
		t.Fatal(err)
	}
	r.wait("квитанция после ручной переотправки", func() bool { return r.count(catalog.ErpPostingResponded, ret, "accepted") == 1 })
	if sm, docs := r.standMessage(ret); docs != 1 || sm.MessageID != dom.MessageID(ret, 1) {
		t.Fatalf("переотправка — тем же номером и один документ: %d %+v", docs, sm)
	}
	if _, err := r.service.ResendPosting(r.ctx, release, erpapp.ResendPosting{Reason: erpapp.ErpReason{Text: "лишняя"}}); err == nil {
		t.Fatal("переотправка подтверждённого сообщения — отказ erp.not_quarantined")
	}

	// 3. Дубль: stand проводит документ и теряет ответ — повтор получает ту же
	// квитанцию (200), второго документа нет.
	r.fault(appingest.Fault{Kind: appingest.FaultDuplicate})
	asm := item + "/warehouse_transfer/assembly.erp_transfer_in"
	r.thrown(item, "assembly.erp_transfer_in", dom.WarehouseTransfer)
	r.wait("повтор после потерянного ответа", func() bool { return r.count(catalog.ErpPostingResponded, asm, "duplicate") == 1 })
	r.clear()
	if sm, docs := r.standMessage(asm); docs != 1 || sm.Deliveries != 2 || sm.Lost != 1 {
		t.Fatalf("дубль задвоил учёт: документов %d, %+v", docs, sm)
	}

	// 4. Несовместимые метаданные: канал degraded, отправки нет; контракт
	// восстановлен — канал ok, сообщение уходит.
	r.fault(appingest.Fault{Kind: appingest.FaultCorrupt})
	r.wait("канал degraded", func() bool {
		ch, _ := r.service.Channels(r.ctx)
		return len(ch.Items) == 1 && ch.Items[0].State == "degraded"
	})
	const other = "ENT01:F-002"
	r.register(other)
	scrap := other + "/scrap_transfer_rework/defect.1"
	r.thrown(other, "nc.erp_scrap_rework", dom.ScrapRework)
	r.wait("сообщение в очереди", func() bool {
		m, err := r.service.Message(r.ctx, scrap, platform.Moment{})
		return err == nil && m.Status == "queued"
	})
	time.Sleep(500 * time.Millisecond)
	if sm, _ := r.standMessage(scrap); sm.MessageID != "" {
		t.Fatal("при degraded ничего не отправляется")
	}
	r.clear()
	r.wait("отправка после восстановления контракта", func() bool { return r.count(catalog.ErpPostingResponded, scrap, "accepted") == 1 })

	// 5. Воспроизведение: та же реакция с нуля над всем журналом даёт те же
	// записи с теми же id (UUIDv5 слота и версии) — журнал их не примет
	// повторно; очередь пуста, в 1С больше ничего не уходит (AD-7, AD-18).
	before, _ := r.stand.Snapshot(r.ctx)
	all := journaltest.ReadAll(t, r.journal, "main")
	have := map[string]bool{}
	requested := 0
	for _, e := range all {
		if e.EventType == string(catalog.ErpPostingRequested) {
			have[e.EventID] = true
			requested++
		}
	}
	replay := &erpapp.Reactor{Codec: r.codec, Env: r.reactor.Env}
	rq, err := replay.Apply(r.ctx, all)
	if err != nil {
		t.Fatal(err)
	}
	if len(rq.Batch) != requested {
		t.Fatalf("воспроизведение дало %d сообщений, в журнале %d", len(rq.Batch), requested)
	}
	for _, p := range rq.Batch {
		if !have[p.Entry.EventID] {
			t.Fatalf("воспроизведение дало новое сообщение %s", p.Entry.EventID)
		}
	}
	if _, err := r.journal.Append(r.ctx, appjournal.AppendRequest{Batch: rq.Batch[:1]}); err == nil {
		t.Fatal("журнал принял повтор того же сообщения")
	}
	time.Sleep(400 * time.Millisecond)
	after, _ := r.stand.Snapshot(r.ctx)
	if len(after.Messages) != len(before.Messages) || len(after.Documents) != len(before.Documents) {
		t.Fatal("после воспроизведения в 1С ушли сообщения")
	}
	if r.count(catalog.ErpPostingRequested, release, "") != 1 {
		t.Fatal("выпуск сформирован дважды")
	}

	// 6. Чтение: журнал обмена, ось «учёт в 1С», каналы.
	r.wait("ось «учёт в 1С»: выпущено на склад готовой продукции", func() bool {
		raw, ok, _ := r.engine.Get(r.ctx, erpapp.ProjectionItemAccounting, item)
		var a dom.ItemAccounting
		_ = json.Unmarshal(raw, &a)
		return ok && a.State == "moved" && a.Warehouse == "WH-AC" && len(a.Keys) == 3
	})
	list, err := r.service.Messages(r.ctx, erpapp.MessageFilter{ItemID: item}, platform.Moment{}, platform.Page{})
	if err != nil || len(list.Items) != 3 {
		t.Fatalf("сообщения изделия: %d %v", len(list.Items), err)
	}
	for _, x := range list.Items {
		if x.Status != "acknowledged" || x.AccountingState == nil {
			t.Fatalf("сообщение %s: %s", x.BusinessKey, x.Status)
		}
	}
	rel, _ := r.service.Message(r.ctx, release, platform.Moment{})
	if *rel.AccountingState != "released" || len(rel.Attempts) != 1 || rel.Attempts[0].Outcome != "accepted" {
		t.Fatalf("выпуск: %+v", rel)
	}
	ch, _ := r.service.Channels(r.ctx)
	if ch.Items[0].State != "ok" || ch.Items[0].Quarantined != 0 || ch.Items[0].LastExchangeAt == nil {
		t.Fatalf("канал: %+v", ch.Items[0])
	}
}
