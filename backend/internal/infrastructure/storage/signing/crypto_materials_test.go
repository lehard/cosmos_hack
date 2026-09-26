package signing

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/signing"
	"ant/internal/contracts/catalog"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// Материалы scenarios/crypto (FR-76, AD-26, AD-32, кейс §6.3, критерий О9):
// данные до и после смены ключа и профиля, изменённый пакет, понижение
// профиля, недоступный ключ, компрометация. В репозитории — только открытые
// ключи и подписанные пакеты; закрытые ключи генерации выбрасываются.
// Пересборка: ANT_UPDATE_CRYPTO=1 go test -run TestCryptoMaterials ./internal/infrastructure/storage/signing/
// (make crypto-materials). Проверка — этот же тест без переменной: каждый
// пакет получает ожидаемый вердикт той же функцией, что у сервера и верификатора.

var cryptoDir = filepath.Join("..", "..", "..", "..", "..", "scenarios", "crypto")

// regLine — строка registry.jsonl: запись key.* журнала (seq, committed_at, тип, data).
type regLine struct {
	Seq         int64           `json:"seq"`
	CommittedAt string          `json:"committed_at"`
	EventType   string          `json:"event_type"`
	Note        string          `json:"note"`
	Data        json.RawMessage `json:"data"`
}

// pkg — подписанный пакет и ожидаемый вердикт на моменте подписи.
type pkg struct {
	Name        string          `json:"name"`
	Title       string          `json:"title"`
	Seq         int64           `json:"seq"`
	CommittedAt string          `json:"committed_at"`
	Envelope    json.RawMessage `json:"envelope"`
	Expected    struct {
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	} `json:"expected"`
}

func ts(seq int64) time.Time {
	return time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC).Add(time.Duration(seq) * time.Minute)
}

func generateCrypto(t *testing.T) {
	var lines []regLine
	keys := map[string]*profiles.PrivateKey{}
	gen := func(ref, profile string) *profiles.PrivateKey {
		k, err := profiles.Generate(ref, profile)
		if err != nil {
			t.Fatal(err)
		}
		keys[ref] = k
		return k
	}
	register := func(seq int64, k *profiles.PrivateKey, subject, rotates, note string, withPublic bool) {
		d := app.RegistrationData{KeyRef: k.Ref, SubjectKind: dom.SubjectPerson, SubjectID: subject, ProfileID: k.Profile,
			Algorithm: k.Algorithm(), Fingerprint: k.Fingerprint(), PayloadClasses: []string{dom.ClassEvent},
			SubjectConfirmation: dom.ConfirmPaper, ValidFrom: ts(0).Format(app.TimeLayout), Rotates: rotates}
		if rotates != "" {
			d.SubjectConfirmation = dom.ConfirmRotation
		}
		if withPublic {
			d.PublicKeyB64 = k.PublicB64()
		}
		raw, _ := json.Marshal(d)
		lines = append(lines, regLine{Seq: seq, CommittedAt: ts(seq).Format(app.TimeLayout), EventType: string(catalog.KeyRegistrationRecorded), Note: note, Data: raw})
	}
	g1, p1 := gen("ins-01@1", dom.ProfileGost), gen("ins-01-pq@1", dom.ProfilePQ)
	lost := gen("ins-09@1", dom.ProfileGost)
	g2 := gen("ins-01@2", dom.ProfileGost)
	x := gen("ins-02@1", dom.ProfileGost)
	register(1, g1, "INS-01", "", "ключ ГОСТ контролёра", true)
	register(2, p1, "INS-01", "", "ключ ML-DSA-65 того же контролёра (для профиля hybrid)", true)
	register(3, lost, "INS-09", "", "ключ зарегистрирован, но его открытая часть недоступна (том не смонтирован, KEK нет) — только отпечаток", false)
	register(4, x, "INS-02", "", "ключ второго контролёра — будет скомпрометирован", true)
	lines = append(lines, regLine{Seq: 100, CommittedAt: ts(100).Format(app.TimeLayout), EventType: string(catalog.KeyProfileRegistered),
		Note: "смена профиля: события — hybrid с позиции 100 (AD-32)", Data: json.RawMessage(`{"object_classes":["event"],"profile_id":"hybrid"}`)})
	register(200, g2, "INS-01", "ins-01@1", "ротация: ins-01@2 заменяет ins-01@1 с позиции 200", true)
	cs := ts(80).Format(app.TimeLayout)
	lines = append(lines, regLine{Seq: 300, CommittedAt: ts(300).Format(app.TimeLayout), EventType: string(catalog.KeyRevocationRecorded),
		Note: "отзыв ins-02@1 на позиции 300: скомпрометирован с позиции 80 (раньше отзыва)",
		Data: json.RawMessage(`{"compromised_since":"` + cs + `","compromised_since_seq":80,"key_ref":"ins-02@1","reason":{"code":"token_lost","text":"токен утерян"}}`)})

	event := func(n int, profile string, signers ...string) []byte {
		ce := dom.CommandEvent{EventType: "inspection.result.recorded", CommandID: fmt.Sprintf("01929a2b-7c3d-7e4f-8a5b-%012d", n),
			ItemID: "ENT01:F-001", SourceID: "INS-01", Data: json.RawMessage(`{"item_id":"ENT01:F-001","outcome":"accept"}`),
			Level: 2, ClientSignedAt: ts(int64(n)).Format(app.TimeLayout), Profile: profile, Signers: signers}
		c, _ := ce.Canonical()
		return c
	}
	sign := func(payload []byte, refs ...string) []byte {
		var ks []*profiles.PrivateKey
		for _, r := range refs {
			ks = append(ks, keys[r])
		}
		env, err := profiles.Signer{Keys: profiles.NewKeyring(ks...)}.Envelope(dom.PayloadType(dom.ClassEvent, 1), payload, refs...)
		if err != nil {
			t.Fatal(err)
		}
		return env.Marshal()
	}
	var pk []pkg
	add := func(name, title string, seq int64, env []byte, status dom.Status, reason string) {
		p := pkg{Name: name, Title: title, Seq: seq, CommittedAt: ts(seq).Format(app.TimeLayout), Envelope: env}
		p.Expected.Status, p.Expected.Reason = string(status), reason
		pk = append(pk, p)
	}
	add("before-profile-switch", "До смены профиля: ГОСТ — цело (проверка по профилю на момент подписи)", 50,
		sign(event(50, dom.ProfileGost, "ins-01@1"), "ins-01@1"), dom.StatusValid, "")
	add("after-profile-switch-hybrid", "После смены профиля: ГОСТ + ML-DSA-65 — цело", 150,
		sign(event(150, dom.ProfileHybrid, "ins-01@1", "ins-01-pq@1"), "ins-01@1", "ins-01-pq@1"), dom.StatusValid, "")
	add("after-profile-switch-gost", "После смены профиля подписано только ГОСТ — отвергнуто: понижение профиля", 151,
		sign(event(151, dom.ProfileGost, "ins-01@1"), "ins-01@1"), dom.StatusRejected, dom.ReasonDowngrade)
	h := sign(event(152, dom.ProfileHybrid, "ins-01@1", "ins-01-pq@1"), "ins-01@1", "ins-01-pq@1")
	var he dom.Envelope
	_ = json.Unmarshal(h, &he)
	he.Signatures = he.Signatures[:1]
	add("hybrid-signature-removed", "Из гибридного пакета удалена подпись ML-DSA-65 — отвергнуто: понижение профиля", 152, he.Marshal(), dom.StatusRejected, dom.ReasonDowngrade)
	tp := sign(event(60, dom.ProfileGost, "ins-01@1"), "ins-01@1")
	var te dom.Envelope
	_ = json.Unmarshal(tp, &te)
	pl, _ := base64.StdEncoding.DecodeString(te.Payload)
	te.Payload = base64.StdEncoding.EncodeToString(bytes.Replace(pl, []byte(`"accept"`), []byte(`"reject"`), 1))
	add("tampered-package", "Изменённый пакет: «годен» заменено на «брак» после подписи — отвергнуто", 60, te.Marshal(), dom.StatusRejected, dom.ReasonTampered)
	add("key-unavailable", "Ключ недоступен — «не проверяемо», никогда не «валидно»", 61,
		sign(event(61, dom.ProfileGost, "ins-09@1"), "ins-09@1"), dom.StatusUnverifiable, dom.ReasonUnavailable)
	add("before-key-rotation", "До ротации: подпись ключом ins-01@1 (гибрид) — цело", 199,
		sign(event(199, dom.ProfileHybrid, "ins-01@1", "ins-01-pq@1"), "ins-01@1", "ins-01-pq@1"), dom.StatusValid, "")
	add("after-key-rotation-old-key", "После ротации старым ключом ins-01@1 — отвергнуто: ключ заменён", 201,
		sign(event(201, dom.ProfileHybrid, "ins-01@1", "ins-01-pq@1"), "ins-01@1", "ins-01-pq@1"), dom.StatusRejected, dom.ReasonKeyInactive)
	add("after-key-rotation-new-key", "После ротации новым ключом ins-01@2 (гибрид) — цело", 202,
		sign(event(202, dom.ProfileHybrid, "ins-01@2", "ins-01-pq@1"), "ins-01@2", "ins-01-pq@1"), dom.StatusValid, "")
	add("compromised-key", "Подписано после «скомпрометирован с X», до отзыва — под сомнением (защитная реакция: сдерживание и переподписание)", 90,
		sign(event(90, dom.ProfileGost, "ins-02@1"), "ins-02@1"), dom.StatusDoubtful, dom.ReasonCompromised)
	add("before-compromise", "Тем же ключом до X — цело", 70,
		sign(event(70, dom.ProfileGost, "ins-02@1"), "ins-02@1"), dom.StatusValid, "")

	if err := os.MkdirAll(filepath.Join(cryptoDir, "packages"), 0o755); err != nil {
		t.Fatal(err)
	}
	var reg bytes.Buffer
	for _, l := range lines {
		b, _ := json.Marshal(l)
		reg.Write(append(b, '\n'))
	}
	if err := os.WriteFile(filepath.Join(cryptoDir, "registry.jsonl"), reg.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	old, _ := filepath.Glob(filepath.Join(cryptoDir, "packages", "*.json"))
	for _, f := range old {
		_ = os.Remove(f)
	}
	for i, p := range pk {
		b, _ := json.MarshalIndent(p, "", "  ")
		if err := os.WriteFile(filepath.Join(cryptoDir, "packages", fmt.Sprintf("%02d-%s.json", i+1, p.Name)), append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// loadRegistry — реестр ключей и профилей из registry.jsonl свёрткой домена.
func loadRegistry(t *testing.T) (*dom.Registry, *dom.ProfileBook, map[string][]byte) {
	f, err := os.Open(filepath.Join(cryptoDir, "registry.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	reg, book, pubs := dom.NewRegistry(), dom.NewProfileBook(nil), map[string][]byte{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var l regLine
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			t.Fatal(err)
		}
		at, _ := time.Parse(app.TimeLayout, l.CommittedAt)
		switch catalog.Type(l.EventType) {
		case catalog.KeyRegistrationRecorded:
			g, err := app.RegistrationFromData(l.Data)
			if err != nil {
				t.Fatal(err)
			}
			g.Seq, g.CommittedAt, g.Provenance = l.Seq, at, dom.ProvPersonal
			reg.ApplyRegistration(g)
			pubs[g.KeyRef] = g.PublicKey()
		case catalog.KeyRevocationRecorded:
			v, err := app.RevocationFromData(l.Data)
			if err != nil {
				t.Fatal(err)
			}
			v.Seq, v.CommittedAt = l.Seq, at
			reg.ApplyRevocation(v)
		case catalog.KeyProfileRegistered:
			var p struct {
				ProfileID     string   `json:"profile_id"`
				ObjectClasses []string `json:"object_classes"`
			}
			_ = json.Unmarshal(l.Data, &p)
			book.Apply(dom.ProfileChange{Profile: p.ProfileID, Classes: p.ObjectClasses, Seq: l.Seq})
		}
	}
	return reg, book, pubs
}

type pubMap struct {
	reg  *dom.Registry
	pubs map[string][]byte
}

func (p pubMap) PublicKey(_ context.Context, ref string) (string, []byte, error) {
	k, ok := p.reg.Key(ref)
	if !ok || len(p.pubs[ref]) == 0 {
		return k.ProfileID, nil, profiles.ErrKeyUnavailable
	}
	return k.ProfileID, p.pubs[ref], nil
}

// TestCryptoMaterials — каждый пакет scenarios/crypto получает ожидаемый вердикт.
func TestCryptoMaterials(t *testing.T) {
	if os.Getenv("ANT_UPDATE_CRYPTO") == "1" {
		generateCrypto(t)
	}
	reg, book, pubs := loadRegistry(t)
	ver := profiles.Verifier{Keys: pubMap{reg, pubs}}
	files, _ := filepath.Glob(filepath.Join(cryptoDir, "packages", "*.json"))
	if len(files) < 9 {
		t.Fatalf("материалов scenarios/crypto мало: %d", len(files))
	}
	var seen []string
	for _, f := range files {
		b, _ := os.ReadFile(f)
		var p pkg
		if err := json.Unmarshal(b, &p); err != nil {
			t.Fatal(err)
		}
		env, payload, checks, err := ver.Check(context.Background(), p.Envelope)
		if err != nil {
			t.Fatalf("%s: %v", p.Name, err)
		}
		ig, _ := dom.IntegrityOf(payload)
		var present []string
		for _, s := range env.Signatures {
			present = append(present, s.KeyID)
		}
		at, _ := time.Parse(app.TimeLayout, p.CommittedAt)
		v := dom.Judge(reg, book, dom.Judgement{Class: dom.ClassEvent, Declared: ig, Present: present, Crypto: checks, Seq: p.Seq, At: at})
		if string(v.Status) != p.Expected.Status || v.Reason != p.Expected.Reason {
			t.Errorf("%s (%s): получено %s/%s, ожидалось %s/%s — %s", p.Name, p.Title, v.Status, v.Reason, p.Expected.Status, p.Expected.Reason, v.Detail)
		}
		seen = append(seen, p.Expected.Status+"/"+p.Expected.Reason)
	}
	for _, need := range []string{"valid/", "rejected/" + dom.ReasonDowngrade, "rejected/" + dom.ReasonTampered, "unverifiable/" + dom.ReasonUnavailable} {
		if !slices.Contains(seen, need) {
			t.Errorf("нет материала %s", need)
		}
	}
	if s := strings.Join(seen, " "); !strings.Contains(s, "valid/") {
		t.Error(s)
	}
}
