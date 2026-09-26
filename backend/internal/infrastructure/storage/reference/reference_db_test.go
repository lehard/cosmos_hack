package reference_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	notifapp "ant/internal/application/notifications"
	"ant/internal/application/platform"
	processapp "ant/internal/application/process"
	"ant/internal/application/process/proctest"
	app "ant/internal/application/reference"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/engine"
	dj "ant/internal/domain/journal"
	"ant/internal/domain/kernel"
	dp "ant/internal/domain/process"
	dom "ant/internal/domain/reference"
	store "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/feed"
	jt "ant/internal/infrastructure/storage/journal/journaltest"
	storage "ant/internal/infrastructure/storage/reference"
)

// Сквозные проверки эпика 19 на своей БД (make dev-db): справочники —
// записи журнала на Postgres, срез на basis_seq изделия — внешний слой
// нормативного слоя свёртки (как у воркера в cmd/ant).

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now(context.Context) (time.Time, error) { return c.t, nil }

type env struct {
	t     *testing.T
	ctx   context.Context
	j     *store.Store
	codec *engineapp.Codec
	src   *app.JournalSource
	w     app.JournalWriter
	clock *fakeClock
	svc   *app.Service
	lsn   *store.Listener
	leas  *store.Leases
}

func msk(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, dom.Local)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

func newEnv(t *testing.T) *env {
	t.Helper()
	d := jt.NewDB(t)
	pool := d.AppPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)
	j := store.NewStore(pool, jt.SysClock{})
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: 1}
	e := &env{t: t, ctx: ctx, j: j, codec: codec, src: &app.JournalSource{Journal: j, Codec: codec},
		w: app.JournalWriter{Journal: j, DomainBuild: dj.ZeroLink.String()}, clock: &fakeClock{t: msk("2026-09-20 10:00")}}
	e.lsn = store.NewListener(pool, nil)
	go func() { _ = e.lsn.Run(ctx) }()
	e.leas = store.NewLeases(pool, jt.SysClock{})
	e.svc = app.NewService(app.WithSource(e.src), app.WithWriter(e.w), app.WithClock(e.clock))
	recs, err := storage.SeedRecords()
	if err != nil {
		t.Fatal(err)
	}
	n, err := app.Seed(ctx, e.w, recs)
	if err != nil || n != len(recs) {
		t.Fatalf("затравка: %d из %d, %v", n, len(recs), err)
	}
	// Повторная затравка — дубли, новых записей нет (migrate идемпотентен).
	if n, err := app.Seed(ctx, e.w, recs); err != nil || n != 0 {
		t.Fatalf("повторная затравка: %d, %v", n, err)
	}
	return e
}

func cmdID(name string) string { return kernel.UUIDv5(constants.NsAnt, "reference-test/"+name) }

// append — записи изделия так, как их пишет приём: факты — device, решения — personal.
func (e *env) append(outs []engineapp.Out) {
	e.t.Helper()
	if err := proctest.Append(e.ctx, e.codec, e.j, outs); err != nil {
		e.t.Fatal(err)
	}
}

// input — вход свёртки изделия из журнала (AD-5).
func (e *env) input(item string) []kernel.Record {
	e.t.Helper()
	es, err := e.j.Read(e.ctx, appjournal.ReadQuery{ItemID: item, Limit: 1000})
	if err != nil {
		e.t.Fatal(err)
	}
	var in []kernel.Record
	for _, x := range es {
		d, err := e.codec.Decode(e.ctx, x)
		if err != nil {
			e.t.Fatal(err)
		}
		if d.Record.Kind != catalog.KindReaction {
			in = append(in, d.Record)
		}
	}
	return in
}

// processBundles — нормативный слой процесса фланца (стартовая версия) под
// срезом справочника, как bundleSource в cmd/ant.
func (e *env) processBundles() (*app.Bundles, string) {
	e.t.Helper()
	xml, err := os.ReadFile("../../../../../normative/process/flange-process.bpmn")
	if err != nil {
		e.t.Fatal(err)
	}
	vs := &processapp.MemVersions{}
	seed, err := processapp.EnsureSeed(e.ctx, vs, xml, msk("2026-01-01 00:00"))
	if err != nil {
		e.t.Fatal(err)
	}
	return &app.Bundles{Next: &processapp.Bundles{Store: vs, TTL: time.Nanosecond}, Source: e.src}, seed.Hash
}

// FR-17, AD-31 (эпик 19, «Что ожидаем в итоге»): операция на оборудовании с
// истёкшей на дату операции поверкой блокируется — гардом команды и реакцией
// на факт; до истечения — разрешена. Поверка — решение метролога в журнале.
func TestExpiredVerificationBlocksOperationOnPostgres(t *testing.T) {
	e := newEnv(t)
	// Метролог 20.09 записывает калибровку станка до 30.09 включительно.
	if _, err := e.svc.VerifyEquipment(e.ctx, "CNC-1", app.VerifyEquipment{CommandHeader: platform.CommandHeader{CommandID: cmdID("verify")},
		Kind: "calibration", Result: "valid", ValidUntil: "2026-09-30", CertificateRef: "СК-1"}); err != nil {
		t.Fatal(err)
	}
	// Справочник на момент: 30.09 — годен, 1.10 — срок истёк.
	for _, c := range []struct {
		at     string
		usable bool
		reason string
	}{{"2026-09-30 23:00", true, ""}, {"2026-10-01 00:00", false, dom.ReasonExpired}} {
		at := msk(c.at)
		l, err := e.svc.Equipment(e.ctx, platform.Moment{Axis: platform.AxisOccurred, AsOf: &at})
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(l.Items, func(x app.RefEquipment) bool { return x.EquipmentID == "CNC-1" })
		if i < 0 || l.Items[i].Usable != c.usable || l.Items[i].UnusableReason != c.reason || l.Items[i].VerificationResult != "valid" {
			t.Fatalf("%s: CNC-1 %+v", c.at, l.Items[i])
		}
	}

	rb, hash := e.processBundles()
	item := "ENT01:FL-0901"
	j := proctest.New(item, hash, msk("2026-10-01 08:00"))
	j.Register(0)
	j.Decide("incoming.zt1_lot_acceptance", "accept", 1)
	j.Run("RUN-MK-1", "incoming.marking", 1.2, 1.4)
	j.Received("incoming.issue_blank", 1.6)
	e.append(j.Facts)
	in := e.input(item)
	bd, _, err := rb.Bundle(e.ctx, item, in)
	if err != nil {
		t.Fatal(err)
	}
	if bd.Process.Reference.Equipment == nil || bd.Process.Reference.Qualifications != nil {
		t.Fatalf("срез справочника для предусловий: %+v", bd.Process.Reference)
	}
	s, _ := engine.Fold(bd, in)
	if s.Process.TokenAt("machining.cnc") == nil {
		t.Fatalf("изделие не на мехобработке: %v", s.Process.Steps())
	}
	start := func(at time.Time) error {
		return engine.Guard(s, bd, kernel.Command{Action: "process.operation.start", OccurredAt: at,
			Payload: dp.StartCommand{StepKey: "machining.cnc", RunID: "RUN-M1-1", OperatorID: "O17", EquipmentID: "CNC-1"}})
	}
	var r *kernel.Refusal
	if err := start(msk("2026-10-01 10:30")); !errors.As(err, &r) || r.Code != errcodes.ProcessPreconditionFailed ||
		!strings.Contains(r.Params["condition"], "CNC-1") {
		t.Fatalf("операция на станке с истёкшей поверкой не заблокирована: %v %v %s", err, r.Params, r.Detail)
	}
	if err := start(msk("2026-09-30 10:30")); err != nil {
		t.Fatalf("до истечения поверки операция разрешена: %v", err)
	}

	// Факт начала операции с терминала на станке с истёкшей поверкой —
	// реакция «нарушено предусловие» (факт не отвергается, AD-30).
	j.Add(catalog.OperationRunStarted, 2.5, map[string]any{"operation_run_id": "RUN-M1-1", "operation_code": "010",
		"step_key": "machining.cnc", "operator_id": "O17", "equipment_id": "CNC-1"})
	e.append(j.Facts[len(j.Facts)-1:])
	in = e.input(item)
	bd, _, err = rb.Bundle(e.ctx, item, in)
	if err != nil {
		t.Fatal(err)
	}
	_, rs := engine.Fold(bd, in)
	if !slices.ContainsFunc(rs, func(x kernel.Reaction) bool {
		b, _ := json.Marshal(x.Data)
		return x.Type == catalog.OperationPreconditionFailed && strings.Contains(string(b), "equipment_verification")
	}) {
		t.Fatalf("нет operation.precondition.failed по поверке: %+v", rs)
	}

	// Срез на basis_seq: изделие, чей вход закончился до записи поверки,
	// поверки не видит (станок не СИ, поверки не было — годен).
	early, err := e.src.Book(e.ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if st := early.Slice(dom.Filter{BasisSeq: 1}).EquipmentStatusAt("CNC-1", msk("2026-10-01 10:30")); st.Valid {
		t.Fatalf("срез на seq 1 видит записи позже: %+v", st)
	}
}

// FR-55 (эпик 19, «Что ожидаем в итоге»): срок решения по изолированному
// изделию — 3 рабочих дня по производственному календарю с праздниками. Воркер
// сворачивает изделие с нормативным слоем под срезом справочника; изоляция
// в четверг 30.04 10:00 МСК: 1 мая — праздник, 2–3 — выходные, срок — среда
// 6 мая 10:00 (по «пн–пт» было бы 5 мая).
func TestIsolationDeadlineByProductionCalendarOnPostgres(t *testing.T) {
	e := newEnv(t)
	wf := feed.NewWorkFeed(e.j, e.leas, e.lsn, 1, feed.Options{Holder: "t", TTL: 30 * time.Second})
	parts, err := wf.Partitions(e.ctx)
	if err != nil || len(parts) != 1 {
		t.Fatalf("партиции: %v %v", parts, err)
	}
	reg := engineapp.NewRegistry()
	if err := notifapp.Register(reg); err != nil {
		t.Fatal(err)
	}
	worker := engineapp.NewWorker(engineapp.WorkerConfig{Feed: wf, Codec: e.codec, Projections: reg,
		Bundles: &app.Bundles{Next: notifapp.Bundles{}, Source: e.src}})

	item := "ENT01:FL-0902"
	isoAt := msk("2026-04-30 10:00")
	j := proctest.New(item, "", isoAt.Add(-time.Hour))
	j.Add(catalog.OperationRunStarted, 0, map[string]any{"operation_run_id": "RUN-W-1", "operation_code": "030", "step_key": "welding.weld",
		"operator_id": "W21", "workplace_id": "WP-WELD-1"})
	j.Add(catalog.DecisionItemIsolated, 1, map[string]any{"reason": map[string]string{"code": "burn_through", "text": "прожог"}})
	e.append(j.Facts)
	for {
		ctx, cancel := context.WithTimeout(e.ctx, 500*time.Millisecond)
		works, err := wf.Next(ctx, parts[0])
		cancel()
		if err != nil || len(works) == 0 {
			break
		}
		if err := worker.Process(e.ctx, parts[0], works); err != nil {
			t.Fatal(err)
		}
	}
	es, err := e.j.Read(e.ctx, appjournal.ReadQuery{Stream: "item:" + item, EventType: string(catalog.ObligationDueSet)})
	if err != nil {
		t.Fatal(err)
	}
	var dues []string
	for _, x := range es {
		d, err := e.codec.Decode(e.ctx, x)
		if err != nil {
			t.Fatal(err)
		}
		var o struct {
			Kind  string `json:"kind"`
			DueAt string `json:"due_at"`
		}
		_ = json.Unmarshal(d.Record.Data, &o)
		if o.Kind == "nc_disposition" {
			dues = append(dues, o.DueAt)
		}
	}
	want := engineapp.FormatTime(msk("2026-05-06 10:00"))
	if len(dues) != 1 || dues[0] != want {
		t.Fatalf("срок решения по изолированному изделию: %v, ждали %s", dues, want)
	}
	// Тот же календарь — для срока, который nonconformity ставит при решении
	// «изолировать» (порт nonconformity.Calendar).
	if got := (app.WorkingCalendar{Source: e.src}).AddWorkingDays(isoAt, 3); !got.Equal(msk("2026-05-06 10:00")) {
		t.Fatalf("календарь nonconformity: %s", got.In(dom.Local))
	}
}

// AD-31, FR-95: живые операции reference на журнале — справочник на момент,
// команды с гардом, соответствия внешних ID с конфликтом.
func TestLiveOperationsOnPostgres(t *testing.T) {
	e := newEnv(t)
	now := platform.Moment{Axis: platform.AxisOccurred}
	its, err := e.svc.ItemTypes(e.ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if i := slices.IndexFunc(its.Items, func(x app.RefItemType) bool { return x.ItemTypeID == "FL-100.00.000" }); i < 0 ||
		!slices.ContainsFunc(its.Items[i].Zones, func(z app.RefZone) bool { return z.ZoneID == "W-1.EDGE" }) {
		t.Fatalf("номенклатура: %+v", its.Items)
	}
	locs, err := e.svc.Locations(e.ctx, now)
	if err != nil || !slices.ContainsFunc(locs.Items, func(x app.RefLocation) bool { return x.LocationID == "LINE-FL-2" && x.ParentID == "LINE-FL" }) {
		t.Fatalf("места: %v %+v", err, locs.Items)
	}
	cal, err := e.svc.Calendar(e.ctx, 2026, now)
	if err != nil || !slices.Contains(cal.NonWorkingDays, "2026-01-09") || !slices.Contains(cal.ShortenedDays, "2026-11-03") {
		t.Fatalf("календарь 2026: %v %+v", err, cal)
	}
	if _, err := e.svc.Calendar(e.ctx, 2030, now); err == nil {
		t.Fatal("календарь несуществующего года")
	}
	sh, err := e.svc.Shifts(e.ctx, "WS-WC", now)
	if err != nil || len(sh.Items) == 0 || !strings.HasPrefix(sh.Items[0].ShiftID, "SHIFT-") {
		t.Fatalf("смены: %v %+v", err, sh.Items)
	}

	// Гард: поверка неизвестного оборудования, место с неизвестным родителем.
	var pe *platform.Error
	if _, err := e.svc.VerifyEquipment(e.ctx, "NOPE-1", app.VerifyEquipment{CommandHeader: platform.CommandHeader{CommandID: cmdID("v-nope")},
		Kind: "verification", Result: "valid", ValidUntil: "2027-01-01"}); !errors.As(err, &pe) || pe.Code != errcodes.ReferenceNotFound {
		t.Fatalf("поверка неизвестного оборудования: %v", err)
	}
	if _, err := e.svc.DefineLocation(e.ctx, app.DefineLocation{CommandHeader: platform.CommandHeader{CommandID: cmdID("loc-bad")},
		RefLocation: app.RefLocation{LocationID: "WP-X", Kind: "workplace", ParentID: "NOPE", Scope: "ent01/x", Name: "X", ValidFrom: e.clock.t}}); !errors.As(err, &pe) || pe.Code != errcodes.ReferenceNotFound {
		t.Fatalf("место с неизвестным родителем: %v", err)
	}
	// Новое рабочее место с датой действия; повтор команды — та же квитанция (AD-7).
	def := app.DefineLocation{CommandHeader: platform.CommandHeader{CommandID: cmdID("loc-ok")},
		RefLocation: app.RefLocation{LocationID: "WP-WELD-3", Kind: "workplace", ParentID: "ST-WELD", Scope: "ent01/b1/wc/weld/wp3",
			Name: "Пост сварки 3", ValidFrom: msk("2026-10-01 00:00")}}
	rc, err := e.svc.DefineLocation(e.ctx, def)
	if err != nil {
		t.Fatal(err)
	}
	if rc2, err := e.svc.DefineLocation(e.ctx, def); err != nil || !rc2.Replayed || rc2.Seq != rc.Seq {
		t.Fatalf("повтор команды: %+v %v", rc2, err)
	}
	// Устаревший basis_seq — 409 journal.stale_state (AD-39).
	def.CommandID, def.BasisSeq, def.Name = cmdID("loc-stale"), rc.Seq-1, "Пост сварки 3 (переименован)"
	if _, err := e.svc.DefineLocation(e.ctx, def); !errors.As(err, &pe) || pe.Code != errcodes.JournalStaleState {
		t.Fatalf("устаревший basis_seq: %v", err)
	}
	before, after := msk("2026-09-30 12:00"), msk("2026-10-01 12:00")
	for _, c := range []struct {
		at   *time.Time
		want bool
	}{{&before, false}, {&after, true}} {
		l, err := e.svc.Locations(e.ctx, platform.Moment{Axis: platform.AxisOccurred, AsOf: c.at})
		if err != nil || slices.ContainsFunc(l.Items, func(x app.RefLocation) bool { return x.LocationID == "WP-WELD-3" }) != c.want {
			t.Fatalf("место с датой действия на %s: %v", c.at, err)
		}
	}

	// Соответствия внешних ID: конфликт — сигнал, не перезапись (FR-95).
	for i, internal := range []string{"ORD-0917", "ORD-0917", "ORD-9999"} {
		if _, err := e.w.Write(e.ctx, app.Record{EventID: cmdID("map-" + string(rune('a'+i))), Type: catalog.ReferenceExternalIdMapped,
			Stream: app.Stream("external_id", "onec:order:ЗП-0917"), OccurredAt: e.clock.t, SourceID: "onec-stand",
			Provenance: jc.JournalEntryProvenanceClassServerAttested,
			Data:       map[string]string{"system": "onec", "object_kind": "order", "external_id": "ЗП-0917", "internal_id": internal}}); err != nil {
			t.Fatal(err)
		}
	}
	m, err := e.svc.ExternalIDs(e.ctx, "onec", now, platform.Page{Limit: 10})
	if err != nil || len(m.Items) != 2 || !m.Items[0].Conflict || !m.Items[0].Effective || m.Items[0].InternalID != "ORD-0917" || m.Items[1].Effective {
		t.Fatalf("соответствия: %v %+v", err, m.Items)
	}
}
