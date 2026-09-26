package security

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	app "ant/internal/application/security"
	"ant/internal/application/security/verify"
	cc "ant/internal/contracts/crypto"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	dom "ant/internal/domain/security"
	"ant/internal/infrastructure/security/atrest"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// Интеграционные тесты эпика 29 на своей БД (make dev-db): запись CA в той
// же транзакции, шифрование при хранении, три атаки make tamper и их
// обнаружение верификатором.

type world struct {
	db    *journaltest.DB
	store *journalstore.Store
	kek   *atrest.KEK
}

func newWorld(t *testing.T) world {
	t.Helper()
	db := journaltest.NewDB(t)
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	kek, err := atrest.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	st := journalstore.NewStore(db.AppPool(t), journaltest.SysClock{},
		journalstore.WithCritical(app.CriticalHook{Enc: app.Encoder{DomainBuild: dj.ZeroLink.String(), StagePartition: 4}}),
		journalstore.WithCipher(kek))
	return world{db: db, store: st, kek: kek}
}

func envelope(t *testing.T, eventType, item string, data any) []byte {
	t.Helper()
	return journaltest.Envelope(eventType, data)
}

func decision(t *testing.T, eventType, item string, data any) appjournal.Pending {
	p := journaltest.Entry(eventType, jc.JournalEntryEntryKindDecision, "item:"+item, item, time.Now())
	p.Envelope = envelope(t, eventType, item, data)
	return p
}

func caRecords(t *testing.T, w world) []dom.Record {
	t.Helper()
	var out []dom.Record
	for _, e := range journaltest.ReadAll(t, w.store, "ca") {
		env, err := w.store.Open(context.Background(), e)
		if err != nil {
			t.Fatal(err)
		}
		ev, _, err := app.ParseEnvelope(env.Raw)
		if err != nil {
			t.Fatal(err)
		}
		var r dom.Record
		if err := json.Unmarshal(ev.Data, &r); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

// AD-8, AD-28: запись CA — в той же транзакции, номер — позиция в цепочке
// ca, commit — обязательство основной записи; отмена — только новой
// записью с причиной; некритическая запись CA не даёт.
func TestCriticalActionSameTransaction(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	if _, err := w.store.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{journaltest.Fact("I-1")}}); err != nil {
		t.Fatal(err)
	}
	var res appjournal.AppendResult
	set := decision(t, "decision.containment.set", "I-1", map[string]any{"level": "item_hold", "reason": map[string]string{"text": "брак"}})
	err := app.Execute(ctx, dom.Command{ActorID: "KO-01"}, func(ctx context.Context) error {
		var err error
		res, err = w.store.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{set}})
		return err
	})
	if err != nil || len(res.CARefs) != 1 || res.CARefs[0] != "CA-1" {
		t.Fatalf("CA: %v %v", res.CARefs, err)
	}
	main := journaltest.ReadAll(t, w.store, "main")
	cas := caRecords(t, w)
	if len(cas) != 1 || cas[0].CANo != 1 || cas[0].MainCommit != main[1].Commit || cas[0].ActorID != "KO-01" ||
		cas[0].After != "заблокировано: Блок изделия" || cas[0].MainEventID != main[1].EventID {
		t.Fatalf("запись CA: %+v", cas)
	}
	// Отмена несуществующей CA — отказ, ничего не записано (одна транзакция).
	rel := decision(t, "decision.containment.released", "I-1", map[string]any{"released_event_ids": []string{main[1].EventID}, "reason": map[string]string{"text": "ошибка"}})
	err = app.Execute(ctx, dom.Command{ActorID: "KO-01", Cancels: "CA-7", CancelReason: "ошибочный блок"}, func(ctx context.Context) error {
		_, err := w.store.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{rel}})
		return err
	})
	if !errors.Is(err, app.ErrCancel) || len(journaltest.ReadAll(t, w.store, "main")) != 2 {
		t.Fatalf("отмена несуществующей CA: %v", err)
	}
	if err := app.Execute(ctx, dom.Command{Cancels: "CA-1"}, func(context.Context) error { return nil }); !errors.Is(err, app.ErrCancel) {
		t.Fatalf("отмена без причины: %v", err)
	}
	err = app.Execute(ctx, dom.Command{ActorID: "KO-01", Cancels: "CA-1", CancelReason: "ошибочный блок"}, func(ctx context.Context) error {
		_, err := w.store.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{rel}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	cas = caRecords(t, w)
	if len(cas) != 2 || cas[1].Cancels != "CA-1" || cas[1].CancelReason.Text != "ошибочный блок" || len(cas[1].BasisEventIDs) != 1 {
		t.Fatalf("отмена: %+v", cas)
	}
}

// AD-23: блок записи зашифрован (в БД нет открытого содержимого), обёртки
// DEK — в journal.dek_wraps; без KEK содержимое не читается, commit сверяется.
func TestEncryptionAtRest(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	p := decision(t, "decision.containment.set", "I-2", map[string]any{"level": "item_hold", "reason": map[string]string{"text": "секрет-причина"}})
	if _, err := w.store.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}}); err != nil {
		t.Fatal(err)
	}
	conn := w.db.AdminConn(t)
	rows, err := conn.Query(ctx, "SELECT envelope, salt, dek_id, aead FROM journal.entries")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for rows.Next() {
		var env, salt []byte
		var dek, aead *string
		if err := rows.Scan(&env, &salt, &dek, &aead); err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(env, []byte("payload")) || len(salt) != 0 || dek == nil || aead == nil || *aead != "aes_256_gcm" {
			t.Fatalf("блок не зашифрован: dek=%v aead=%v salt=%d", dek, aead, len(salt))
		}
		n++
	}
	rows.Close()
	var wraps int
	_ = conn.QueryRow(ctx, "SELECT count(*) FROM journal.dek_wraps").Scan(&wraps)
	if n != 2 || wraps != 2 {
		t.Fatalf("записей %d, обёрток %d", n, wraps)
	}
	es := journaltest.ReadAll(t, w.store, "main")
	env, err := w.store.Open(ctx, es[0])
	if err != nil || !bytes.Contains(env.Raw, []byte("payload")) || len(env.Salt) != dj.SaltSize {
		t.Fatalf("Open: %v", err)
	}
	blind := journalstore.NewStore(w.db.AppPool(t), journaltest.SysClock{})
	if _, err := blind.Open(ctx, es[0]); !errors.Is(err, appjournal.ErrSealed) {
		t.Fatalf("без KEK: %v", err)
	}
	if _, err := journaltest.MigrateAgain(w.db); err != nil {
		t.Fatal(err)
	}
}

// fakeProjections — проекций нет, воркер не запускался (проекции не сверяются).
type fakeProjections struct{}

func (fakeProjections) ItemRows(context.Context, string) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}
func (fakeProjections) WorkerCursor(context.Context, int) (int64, error) { return 0, nil }

type lenient struct{ *journalstore.Store }

func (l lenient) Open(ctx context.Context, e jc.JournalEntry) (appjournal.Envelope, error) {
	s, env, err := l.OpenUnverified(ctx, e)
	return appjournal.Envelope{Raw: env, Salt: s}, err
}

func check(t *testing.T, w world, cps []verify.Checkpoint, links map[string]map[int64]string) verify.Report {
	t.Helper()
	in := verify.Input{Journal: w.store, Codec: &engineapp.Codec{Store: lenient{w.store}, Partitions: 4},
		Registry: engineapp.NewRegistry(), Bundles: engineapp.EmptyBundles{}, Projections: fakeProjections{},
		Checkpoints: cps, KeeperLinks: links, Partitions: 4, MaxGap: time.Hour}
	if cps == nil {
		in.CheckpointsErr = errors.New("нет хранителя")
	}
	r, err := verify.Run(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func rejected(r verify.Report, name string) []string {
	var out []string
	for _, c := range r.Checks {
		if string(c.Check) == name && c.Status == "rejected" {
			for _, f := range c.Findings {
				out = append(out, f.Detail)
			}
		}
	}
	return out
}

func seed(t *testing.T, w world) []jc.JournalEntry {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if _, err := w.store.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{journaltest.Fact("I-3")}}); err != nil {
			t.Fatal(err)
		}
	}
	return journaltest.ReadAll(t, w.store, "main")
}

// UJ-5, AD-28 атака 1: правка записи в обход системы — «звено не сходится» с номером записи.
func TestTamperUpdateInPlace(t *testing.T) {
	w := newWorld(t)
	es := seed(t, w)
	if r := check(t, w, nil, nil); r.Verdict == "violated" {
		t.Fatalf("чистый журнал — нарушение: %s", verify.Summary(r))
	}
	tp := &Tamperer{Conn: w.db.AdminConn(t), KEK: w.kek, Profile: "demo"}
	res, err := tp.UpdateInPlace(context.Background(), es[2].EventID, map[string]any{"data/v": 2})
	if err != nil || res.Seq != 3 {
		t.Fatalf("атака 1: %+v %v", res, err)
	}
	r := check(t, w, nil, nil)
	got := rejected(r, "chains")
	if r.Verdict != "violated" || len(got) != 1 || !strings.Contains(got[0], "seq 3") || !strings.Contains(got[0], "звено не сходится") {
		t.Fatalf("атака 1 не найдена: %s", verify.Summary(r))
	}
	if _, err := (&Tamperer{Conn: w.db.AdminConn(t), Profile: "prod"}).UpdateInPlace(context.Background(), "", nil); !errors.Is(err, ErrProfile) {
		t.Fatalf("prod: %v", err)
	}
}

// AD-28 атака 2: правка и пересчёт цепочки — звенья сходятся, но голова
// расходится с контрольной точкой, а звено — с переданным хранителю.
func TestTamperRechain(t *testing.T) {
	w := newWorld(t)
	es := seed(t, w)
	head := es[len(es)-1]
	cp := verify.Checkpoint{Checkpoint: app.Checkpoint{Digest: "d1", Payload: cc.KeeperCheckpoint{CheckpointNo: 1, KeeperTime: dj.FormatTime(time.Now()),
		Heads: []cc.KeeperCheckpointHeadsElem{{Chain: "main", Seq: head.Seq, Link: head.Link}, {Chain: "ca", Seq: 0, Link: dj.ZeroLink.String()}}}}}
	links := map[string]map[int64]string{"main": {}, "ca": {}}
	for _, e := range es {
		links["main"][int64(e.Seq)] = e.Link
	}
	if r := check(t, w, []verify.Checkpoint{cp}, links); len(rejected(r, "checkpoints")) != 0 {
		t.Fatalf("до атаки: %s", verify.Summary(r))
	}
	tp := &Tamperer{Conn: w.db.AdminConn(t), KEK: w.kek, Profile: "demo"}
	res, err := tp.UpdateAndRechain(context.Background(), "seq:2", nil)
	if err != nil || res.Relinked != 4 {
		t.Fatalf("атака 2: %+v %v", res, err)
	}
	r := check(t, w, []verify.Checkpoint{cp}, links)
	if len(rejected(r, "chains")) != 0 {
		t.Fatalf("цепочка пересчитана, но звенья не сходятся: %v", rejected(r, "chains"))
	}
	got := strings.Join(rejected(r, "checkpoints"), "\n")
	if r.Verdict != "violated" || !strings.Contains(got, "контрольной точкой №1") || !strings.Contains(got, "запись seq 2") {
		t.Fatalf("атака 2 не найдена: %s", verify.Summary(r))
	}
}
