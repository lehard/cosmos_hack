package signing

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"uuid"

	"ant/internal/application/ingest/inmem"
	"ant/internal/application/journal"
	"ant/internal/application/platform"
	app "ant/internal/application/signing"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// world — модуль signing над журналом в памяти, криптографией профилей и
// затравкой реестра: ключи людей демо-политики (у подписантов актов — два
// ключа, gost и pq, AD-32), ключ устройства.
type world struct {
	t     *testing.T
	j     *inmem.Journal
	clock *inmem.Clock
	svc   *app.Service
	keys  *profiles.Keyring
	scans fakeScans
}

// fakeScans — хранилище сканов и «чтение QR»: скан — байты `QR:‹текст›`
// (настоящее чтение gozxing проверяют integration/signing/paperscan и cmd/demo-signer).
type fakeScans map[string][]byte

func (f fakeScans) Scan(_ context.Context, addr string) ([]byte, error) {
	b, ok := f[addr]
	if !ok {
		return nil, errors.New("нет скана")
	}
	return b, nil
}

type fakeQR struct{}

func (fakeQR) ReadQR(img []byte) (string, error) {
	s, ok := strings.CutPrefix(string(img), "QR:")
	if !ok {
		return "", errors.New("QR не найден")
	}
	return s, nil
}

var t0 = time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)

func newWorld(t *testing.T, allowUnsigned bool) *world {
	t.Helper()
	w := &world{t: t, clock: inmem.NewClock(t0), keys: profiles.NewKeyring(), scans: fakeScans{}}
	w.j = inmem.NewJournal(w.clock.At)
	var boot []dom.Registration
	add := func(ref, kind, subject, profile string, classes ...string) {
		k, err := profiles.Generate(ref, profile)
		if err != nil {
			t.Fatal(err)
		}
		w.keys = profiles.NewKeyring(append(w.all(), k)...)
		g, _ := BootstrapOf(ref, kind, subject, profile, k.PublicB64(), k.Fingerprint(), dom.NaturalProvenance(kind), classes, t0.Add(-time.Hour)).Registration()
		g.Provenance = dom.NaturalProvenance(kind)
		boot = append(boot, g)
	}
	all := []string{dom.ClassEvent, dom.ClassKeyAct, dom.ClassPaperAttestation, dom.ClassShiftReport, dom.ClassDocumentSignature}
	for _, p := range []string{"ADM-01", "HQC-01", "AUD-01", "PM-01"} {
		add(strings.ToLower(p)+"@1", dom.SubjectPerson, p, dom.ProfileGost, all...)
		add(strings.ToLower(p)+"-pq@1", dom.SubjectPerson, p, dom.ProfilePQ, all...)
	}
	for _, p := range []string{"INS-01", "INS-02", "FOR-WC", "O17"} {
		add(strings.ToLower(p)+"@1", dom.SubjectPerson, p, dom.ProfileGost, all...)
	}
	add("ins-01-pq@1", dom.SubjectPerson, "INS-01", dom.ProfilePQ, all...)
	add("dev-ws2@1", dom.SubjectDevice, "ws-2", dom.ProfileGost, dom.ClassEvent)
	// Ключ, открытой части которого нет (том не смонтирован, KEK нет): «недоступный ключ».
	boot = append(boot, dom.Registration{KeyRef: "lost@1", SubjectKind: dom.SubjectPerson, SubjectID: "INS-01", ProfileID: dom.ProfileGost,
		Algorithm: dom.AlgGost, PayloadClasses: all, Provenance: dom.ProvPersonal})
	if k, err := profiles.Generate("lost@1", dom.ProfileGost); err == nil {
		w.keys = profiles.NewKeyring(append(w.all(), k)...)
	}
	w.svc = app.NewService(
		app.WithConfig(app.Config{Profile: "prod", AllowUnsigned: allowUnsigned, Partitions: 16, DomainBuild: dom.Digest([]byte("test"))}),
		app.WithDeps(app.Deps{Journal: w.j, Registry: app.NewRegistry(w.j, boot, nil), Authorities: app.DefaultAuthorities(),
			Scans: w.scans, QR: fakeQR{}, Now: w.clock.At}))
	w.svc.UseCrypto(profiles.Verifier{Keys: w.svc.Registry()})
	w.svc.UseAlerts(app.JournalKeyAlerts{Service: w.svc})
	return w
}

func (w *world) all() []*profiles.PrivateKey {
	var out []*profiles.PrivateKey
	for _, r := range w.keys.Refs() {
		k, _ := w.keys.Key(r)
		out = append(out, k)
	}
	return out
}

func as(person string) context.Context {
	return platform.WithPrincipal(context.Background(), platform.Principal{PersonID: person})
}

// sign — конверт класса class над payload ключами refs.
func (w *world) sign(class string, payload []byte, refs ...string) []byte {
	env, err := profiles.Signer{Keys: w.keys}.Envelope(dom.PayloadType(class, 1), payload, refs...)
	if err != nil {
		w.t.Fatal(err)
	}
	return env.Marshal()
}

// decision — событие-команда «годен» контролёра и её ожидание сервером.
func decision(item string, level int, profile string, signers ...string) (dom.CommandEvent, app.Expect) {
	id := uuid.NewV7().String()
	data := json.RawMessage(`{"inspection_point":"weld.zt3","item_id":"` + item + `","outcome":"accept"}`)
	ce := dom.CommandEvent{EventType: "inspection.result.recorded", CommandID: id, ItemID: item, SourceID: "INS-01", Data: data,
		Level: level, ClientSignedAt: t0.Format(app.TimeLayout), Profile: profile, Signers: signers}
	return ce, app.Expect{Class: dom.ClassEvent, EventType: ce.EventType, CommandID: id, ItemID: item, Data: data, Actor: "INS-01",
		Level: dom.Level2, Critical: true, PaperAllowed: true, Signer: "INS-01", Stage: 1}
}

func (w *world) signCmd(ce dom.CommandEvent, refs ...string) []byte {
	c, err := ce.Canonical()
	if err != nil {
		w.t.Fatal(err)
	}
	return w.sign(dom.ClassEvent, c, refs...)
}

func code(err error) string {
	var pe *platform.Error
	if errors.As(err, &pe) {
		return string(pe.Code)
	}
	if err == nil {
		return ""
	}
	return "?" + err.Error()
}

func (w *world) alerts() []string {
	es, _ := w.j.Read(context.Background(), journal.ReadQuery{EventType: string(catalog.SecurityKeyAlert)})
	var out []string
	for _, e := range es {
		env, _ := w.j.Open(context.Background(), e)
		d, _ := app.EventData(env.Raw)
		var a struct{ Alert string }
		_ = json.Unmarshal(d, &a)
		out = append(out, a.Alert)
	}
	return out
}

// appendDecision — записать решение как его записал бы модуль-исполнитель:
// подписанный конверт как есть (для сменного рапорта и защитной реакции).
func (w *world) appendDecision(t catalog.Type, item string, raw []byte) int64 {
	id := uuid.NewV7().String()
	it := item
	res, err := w.j.Append(context.Background(), journal.AppendRequest{Batch: []journal.Pending{{Entry: jc.JournalEntry{
		EntryKind: jc.JournalEntryEntryKindDecision, EventType: string(t), SchemaVersion: 1, EventID: id, SourceID: "ant-api",
		ItemID: &it, Stream: "item:" + item, OccurredAt: w.clock.At().Format(app.TimeLayout), ReceivedAt: w.clock.At().Format(app.TimeLayout),
		CorrelationID: id, ProvenanceClass: jc.JournalEntryProvenanceClassPersonal}, Envelope: raw}}})
	if err != nil {
		w.t.Fatal(err)
	}
	return res.Seqs[0]
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

var _ = errcodes.SigningQrMismatch
