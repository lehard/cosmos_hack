package signing

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appingest "ant/internal/application/ingest"
	"ant/internal/application/ingest/inmem"
	"ant/internal/application/journal"
	"ant/internal/application/platform"
	processapp "ant/internal/application/process"
	app "ant/internal/application/signing"
	appvision "ant/internal/application/vision"
	"ant/internal/contracts/catalog"
	domingest "ant/internal/domain/ingest"
	dj "ant/internal/domain/journal"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/clock"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// Встроенная копия затравки совпадает с normative/ репозитория (кроме README).
func TestSeedMatchesRepo(t *testing.T) {
	repo := os.DirFS("../../../../..")
	n := 0
	err := fs.WalkDir(repo, "normative", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(p) == "README.md" {
			return err
		}
		n++
		want, _ := fs.ReadFile(repo, p)
		got, err := fs.ReadFile(Seed(), p)
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s: встроенная копия расходится с репозиторием — скопируйте файл в seed/", p)
		}
		return nil
	})
	if err != nil || n < 20 {
		t.Fatalf("файлов %d: %v", n, err)
	}
	s, err := app.LoadSeed(Seed())
	if err != nil || len(s.Files) != n || len(s.Policy.Persons) == 0 {
		t.Fatalf("затравка: %d файлов, %v", len(s.Files), err)
	}
}

// volumes — тома ant init во временном каталоге.
func volumes(t *testing.T) Volumes {
	d := t.TempDir()
	return Volumes{AntKeys: filepath.Join(d, "ant-keys"), Keeper: filepath.Join(d, "keeper"), Verifier: filepath.Join(d, "verifier"),
		AntPKI: filepath.Join(d, "ant-pki"), DemoSigner: filepath.Join(d, "demo-signer"), TokenAgent: filepath.Join(d, "demo-keys", "token-agent"),
		Devices: filepath.Join(d, "edge"), Partner: filepath.Join(d, "ant-keys", "partner")}
}

func provisionConfig(v Volumes) ProvisionConfig {
	return ProvisionConfig{Volumes: v, Profile: "demo", ClockMode: app.ClockScenario, ProcessVersionID: processapp.SeedVersionID,
		DomainBuild: dom.Digest([]byte("test")), Partitions: 16}
}

// snapshot — отпечаток всех файлов томов: путь, права, содержимое.
func snapshot(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		fi, _ := d.Info()
		b, _ := os.ReadFile(p)
		h := sha256.Sum256(b)
		lines = append(lines, p+" "+fi.Mode().String()+" "+base64.StdEncoding.EncodeToString(h[:]))
		return nil
	})
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// AD-33, FR-109, критерий Т1: на чистой базе ant init создаёт ключи в тома
// (0400), сертификаты mTLS, блок генезиса seq 1…k и trust-anchors; повтор
// ничего не меняет; блок проходит проверку по закреплённому якорю, его записи
// соответствуют схемам контракта; демо-путь после генезиса: ключи персон,
// шлюза и устройств в реестре, подпись шлюза, паспорта анализаторов,
// стартовая версия процесса, режим часов.
func TestProvisionOnDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	db := journaltest.NewDB(t)
	j := journalstore.NewStore(db.AppPool(t), journaltest.SysClock{}, journalstore.WithBatchMax(4096))
	v := volumes(t)
	res, err := Provision(ctx, j, provisionConfig(v))
	if err != nil || !res.Written {
		t.Fatalf("init: %+v %v", res, err)
	}
	all := journaltest.ReadAll(t, j, "main")
	if len(all) != res.Header.BlockSize || all[0].Seq != 1 || all[0].EventType != string(catalog.JournalGenesisRecorded) ||
		all[len(all)-1].EventType != string(catalog.JournalAnchorDestroyed) {
		t.Fatalf("блок: %d записей, заголовок %+v", len(all), res.Header)
	}
	for _, e := range all {
		if e.ProvenanceClass != "genesis" {
			t.Fatalf("seq %d: класс %s", e.Seq, e.ProvenanceClass)
		}
	}
	// Ключи — 0400, якоря на диске нет; trust-anchors закрепляет якорь и генезис.
	for _, p := range []string{filepath.Join(v.AntKeys, profiles.FileName(GatewayKeyRef)), filepath.Join(v.AntKeys, profiles.FileName(EngineKeyRef)),
		filepath.Join(v.Keeper, "keys", "keeper.gost"), filepath.Join(v.Verifier, "keys", "verifier.pq"),
		filepath.Join(v.DemoSigner, profiles.FileName("tec-01@1")), filepath.Join(v.TokenAgent, profiles.FileName("ins-01-ta@1")),
		filepath.Join(v.Devices, "edge-weld-1.key"), filepath.Join(v.AntPKI, "pki", "ant.key"), filepath.Join(v.Verifier, "pki", "verifier.key")} {
		fi, err := os.Stat(p)
		if err != nil || fi.Mode().Perm() != 0o400 {
			t.Errorf("%s: %v %v", p, fi, err)
		}
	}
	if _, err := os.Stat(filepath.Join(v.Keeper, "pki", "ant.key")); !errors.Is(err, os.ErrNotExist) {
		t.Error("закрытый ключ mTLS ant остался в томе хранителя")
	}
	root := filepath.Dir(v.AntKeys)
	if out := snapshot(t, root); strings.Contains(out, "anchor-") {
		t.Error("ключ-якорь на диске")
	}
	ta, err := profiles.LoadTrustAnchors(filepath.Join(v.Keeper, AnchorsFile))
	if err != nil || ta.GenesisDigest != res.Digest || ta.AnchorFingerprint != res.Header.AnchorFingerprint || len(ta.Keys) != 6 {
		t.Fatalf("trust-anchors: %+v %v", ta, err)
	}
	if tv, _ := profiles.LoadTrustAnchors(filepath.Join(v.Verifier, AnchorsFile)); tv.GenesisDigest != res.Digest {
		t.Fatal("копия trust-anchors у верификатора")
	}
	// Повтор ничего не меняет: ни журнал, ни тома.
	before := snapshot(t, root)
	again, err := Provision(ctx, j, provisionConfig(v))
	if err != nil || again.Written || again.Digest != res.Digest {
		t.Fatalf("повтор: %+v %v", again, err)
	}
	if n := len(journaltest.ReadAll(t, j, "main")); n != len(all) {
		t.Fatalf("повтор дописал журнал: %d → %d", len(all), n)
	}
	if snapshot(t, root) != before {
		t.Fatal("повтор изменил тома")
	}
	// Блок — по закреплённому якорю; записи — по схемам контракта.
	rep, err := app.VerifyGenesis(ctx, j, profiles.Verifier{}, ta.AnchorFingerprint)
	if err != nil || rep.Digest != res.Digest {
		t.Fatalf("проверка: %v", err)
	}
	for _, e := range all {
		env, _ := j.Open(ctx, e)
		_, payload, err := dom.ParseEnvelope(env.Raw)
		if err != nil {
			t.Fatal(err)
		}
		d, err := appingest.ValidateEnvelope(payload)
		if err != nil || d.Outcome != domingest.OutcomeAccepted {
			t.Errorf("seq %d %s: схема: %+v %v", e.Seq, e.EventType, d, err)
		}
	}
	// Реестр ключей из генезиса: демо-персоны (scenario), шлюз и устройства.
	reg := app.NewRegistry(j, nil, nil)
	keys, book, _, err := reg.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if k, ok := keys.Key("tec-01@1"); !ok || k.Provenance != dom.ProvScenario || k.SubjectKind != dom.SubjectDemoPersona {
		t.Fatalf("персона: %+v", k)
	}
	if book.Required(dom.ClassGenesis, 1) != dom.ProfileHybrid || book.Required(dom.ClassEvent, 10) != dom.ProfileGost {
		t.Fatal("реестр профилей")
	}
	ver, ik, signer, err := IngestSigning(j, v.AntKeys)
	if err != nil || signer == nil {
		t.Fatalf("подпись шлюза: %v", err)
	}
	if info, err := ik.Key(ctx, GatewayKeyRef); err != nil || info.SourceID != GatewaySource || info.Revoked {
		t.Fatalf("ключ шлюза: %+v %v", info, err)
	}
	if err := ik.Source(ctx, "edge-weld-1"); err != nil {
		t.Fatalf("источник устройства: %v", err)
	}
	raw, err := signer.Sign(ctx, appingest.PayloadTypeEvent, []byte(`{"a":1}`), GatewayKeyRef)
	if err != nil {
		t.Fatal(err)
	}
	if vr, err := ver.Verify(ctx, raw); err != nil || len(vr.Signers) != 1 || vr.Signers[0].KeyRef != GatewayKeyRef {
		t.Fatalf("проверка подписи шлюза: %+v %v", vr, err)
	}
	// Паспорта анализаторов — из генезиса.
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: EngineKeyRef}, KeyRef: EngineKeyRef, DomainBuild: dj.ZeroLink.String(), Partitions: 16}
	svc := appvision.NewService(appvision.WithDeps(appvision.Deps{Journal: j, Codec: codec, Routes: appvision.DemoRoutes{}}),
		appvision.WithConfig(appvision.Config{DomainBuild: dj.ZeroLink.String(), Partitions: 16}))
	pp, err := svc.Passport(ctx, "AP-KT3-WELD-1", platform.Moment{})
	if err != nil || pp.TrustLevel != 3 || pp.Status != "active" || *pp.Provenance != "genesis" {
		t.Fatalf("паспорт: %+v %v", pp, err)
	}
	// Стартовая версия процесса — отпечаток из генезиса.
	id, xml, err := GenesisProcess(ctx, j, nil)
	if err != nil || id != processapp.SeedVersionID || dom.Digest(xml) != res.Header.NormativeVersionHash {
		t.Fatalf("процесс: %s %v", id, err)
	}
	vs := &processapp.MemVersions{}
	if sv, err := processapp.EnsureSeed(ctx, vs, xml, time.Now()); err != nil || !sv.Genesis || sv.Hash != res.Header.NormativeVersionHash {
		t.Fatalf("версия: %+v %v", sv, err)
	}
	// Режим часов — свойство журнала из генезиса.
	if mode, err := clock.NewJournal(j).Mode(ctx); err != nil || mode != clock.ModeScenario {
		t.Fatalf("часы: %s %v", mode, err)
	}
	// Подделка в обход системы: изменить data записи блока суперпользователем БД.
	conn := db.AdminConn(t)
	var envRaw []byte
	if err := conn.QueryRow(ctx, "SELECT envelope FROM journal.entries WHERE chain = 'main' AND seq = 20").Scan(&envRaw); err != nil {
		t.Fatal(err)
	}
	var env dom.Envelope
	_ = json.Unmarshal(envRaw, &env)
	p, _ := base64.StdEncoding.DecodeString(env.Payload)
	env.Payload = base64.StdEncoding.EncodeToString(bytes.Replace(p, []byte(`"genesis"`), []byte(`"genesix"`), 1))
	for _, q := range []string{"ALTER TABLE journal.entries DISABLE TRIGGER ALL",
		"UPDATE journal.entries SET envelope = $1 WHERE chain = 'main' AND seq = 20", "ALTER TABLE journal.entries ENABLE TRIGGER ALL"} {
		var args []any
		if strings.HasPrefix(q, "UPDATE") {
			args = append(args, env.Marshal())
		}
		if _, err := conn.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := app.VerifyGenesis(ctx, j, profiles.Verifier{}, ta.AnchorFingerprint); err == nil {
		t.Fatal("изменённый генезис принят")
	}
	if _, err := Provision(ctx, j, provisionConfig(v)); err == nil {
		t.Fatal("init принял изменённый генезис")
	}
}

// Журнал не пуст, а генезиса нет — init отказывается и ключей не создаёт.
func TestProvisionRefusesNonEmptyJournal(t *testing.T) {
	ctx := context.Background()
	j := inmem.NewJournal(time.Now)
	if _, err := j.Append(ctx, journal.AppendRequest{Batch: []journal.Pending{journaltest.Fact("ENT01:X-1")}}); err != nil {
		t.Fatal(err)
	}
	v := volumes(t)
	if _, err := Provision(ctx, j, provisionConfig(v)); !errors.Is(err, app.ErrJournalNotEmpty) {
		t.Fatalf("%v", err)
	}
	if _, err := os.Stat(v.AntKeys); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("ключи созданы при отказе")
	}
}

func keep(ps []journal.Pending) []journal.Pending { return ps }

// genesisOnMemory — блок генезиса в журнале в памяти и его пакеты.
func genesisOnMemory(t *testing.T, mutate func(ps []journal.Pending) []journal.Pending) (*inmem.Journal, error) {
	t.Helper()
	ctx := context.Background()
	src := inmem.NewJournal(time.Now)
	cfg := provisionConfig(volumes(t))
	if _, err := Provision(ctx, src, cfg); err != nil {
		t.Fatal(err)
	}
	var ps []journal.Pending
	for _, s := range src.Main() {
		e := s.Entry
		e.Seq, e.Commit, e.Link, e.CommittedAt, e.RecordedAt = 0, "", "", "", ""
		ps = append(ps, journal.Pending{Entry: e, Envelope: s.Envelope})
	}
	ps = mutate(ps)
	dst := inmem.NewJournal(time.Now)
	if _, err := dst.Append(ctx, journal.AppendRequest{Batch: ps}); err != nil {
		t.Fatal(err)
	}
	_, err := app.VerifyGenesis(ctx, dst, profiles.Verifier{}, "")
	return dst, err
}

// Изменённый пакет генезиса отвергается (AD-10, AD-33): другое содержимое
// под теми же подписями, снятая подпись ML-DSA (понижение hybrid), удалённая
// запись блока, запись класса не genesis, второй генезис.
func TestTamperedGenesisRejected(t *testing.T) {
	if _, err := genesisOnMemory(t, keep); err != nil {
		t.Fatalf("неизменённый блок: %v", err)
	}
	edit := func(p *journal.Pending, f func(env *dom.Envelope, payload []byte) []byte) {
		var env dom.Envelope
		_ = json.Unmarshal(p.Envelope, &env)
		pl, _ := base64.StdEncoding.DecodeString(env.Payload)
		env.Payload = base64.StdEncoding.EncodeToString(f(&env, pl))
		p.Envelope = env.Marshal()
	}
	cases := map[string]func(ps []journal.Pending) []journal.Pending{
		"содержимое": func(ps []journal.Pending) []journal.Pending {
			edit(&ps[10], func(_ *dom.Envelope, pl []byte) []byte {
				return bytes.Replace(pl, []byte(`"data":{`), []byte(`"data":{"x":1,`), 1)
			})
			return ps
		},
		"понижение": func(ps []journal.Pending) []journal.Pending {
			edit(&ps[5], func(env *dom.Envelope, pl []byte) []byte { env.Signatures = env.Signatures[:1]; return pl })
			return ps
		},
		"удаление": func(ps []journal.Pending) []journal.Pending { return append(ps[:7:7], ps[8:]...) },
		"класс": func(ps []journal.Pending) []journal.Pending {
			ps[3].Entry.ProvenanceClass = "server_attested"
			return ps
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := genesisOnMemory(t, mutate); !errors.Is(err, dom.ErrGenesis) {
				t.Fatalf("изменённый блок принят: %v", err)
			}
		})
	}
	t.Run("второй генезис", func(t *testing.T) {
		j, err := genesisOnMemory(t, keep)
		if err != nil {
			t.Fatal(err)
		}
		second, _ := genesisOnMemory(t, keep)
		for _, s := range second.Main()[:1] {
			e := s.Entry
			e.Seq, e.Commit, e.Link, e.CommittedAt, e.RecordedAt = 0, "", "", "", ""
			if _, err := j.Append(context.Background(), journal.AppendRequest{Batch: []journal.Pending{{Entry: e, Envelope: s.Envelope}}}); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := app.VerifyGenesis(context.Background(), j, profiles.Verifier{}, ""); !errors.Is(err, dom.ErrSecondGenesis) {
			t.Fatalf("второй генезис: %v", err)
		}
	})
}
