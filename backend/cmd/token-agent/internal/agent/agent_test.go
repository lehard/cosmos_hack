package agent

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/signing"
	"ant/internal/contracts/procs"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// Детерминированные ключи персоны INS-01 (как файлы .demo-keys/token-agent/).
func keyFile(t *testing.T, ref, profile string, fill byte) []byte {
	t.Helper()
	k, err := profiles.FromSecret(ref, profile, bytes.Repeat([]byte{fill}, 32))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(KeyFile{KeyRef: ref, Profile: profile, SecretHex: hex.EncodeToString(k.Secret()), PublicKeyB64: k.PublicB64(),
		Fingerprint: k.Fingerprint()})
	return b
}

var fastKDF = KDF{Name: "argon2id", Time: 1, MemoryKiB: 64, Threads: 1}

func sealed(t *testing.T, pin string, files ...[]byte) (Sealed, Bundle) {
	t.Helper()
	b, err := NewBundle(files, "")
	if err != nil {
		t.Fatal(err)
	}
	k := fastKDF
	s, err := Seal(b, pin, SealOptions{KeyStorage: dom.StorageSoftwareBrowser, StorageVariant: dom.VariantExtension, KDF: &k}, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return s, b
}

// Хранилище под PIN: верный PIN открывает, неверный — signing.pin_wrong;
// подмена открытых сведений (владельца) без PIN не проходит (AAD).
func TestVault(t *testing.T) {
	g := keyFile(t, "ins-01-ta@1", dom.ProfileGost, 7)
	s, b := sealed(t, "1234", g)
	if s.PersonID != "INS-01" || len(s.Keys) != 1 || s.Keys[0].KeyRef != "ins-01-ta@1" || s.KeyStorage != dom.StorageSoftwareBrowser {
		t.Fatalf("открытые сведения: %+v", s)
	}
	if strings.Contains(s.CipherB64, hex.EncodeToString(bytes.Repeat([]byte{7}, 32))) {
		t.Fatal("закрытый ключ в открытом виде")
	}
	got, err := Open(s, DeriveKey(s, "1234"))
	if err != nil || got.PersonID != b.PersonID {
		t.Fatalf("верный PIN: %v", err)
	}
	if _, err := Open(s, DeriveKey(s, "0000")); !errors.Is(err, ErrPIN) {
		t.Fatalf("неверный PIN: %v", err)
	}
	s2 := s
	s2.PersonID = "HQC-01"
	if _, err := Open(s2, DeriveKey(s2, "1234")); !errors.Is(err, ErrPIN) {
		t.Fatalf("подмена владельца: %v", err)
	}
	if _, err := Seal(b, "12", SealOptions{}, rand.Reader); err == nil {
		t.Fatal("короткий PIN принят")
	}
	if _, err := NewBundle([][]byte{g, keyFile(t, "hqc-01-ta-pq@1", dom.ProfilePQ, 9)}, ""); err == nil {
		t.Fatal("ключи двух людей в одном хранилище")
	}
	if PersonOf("for-wc-ta-pq@1") != "FOR-WC" || PersonOf("ins-01-ta@1") != "INS-01" {
		t.Fatal("PersonOf")
	}
}

// Запрос решения контролёра: тело команды как у страницы (JSON.stringify).
func decisionBlock(level int) procs.SignBlock {
	body := `{"basis_seq":41,"command_id":"0192f1a0-0000-7000-8000-000000000001","policy_seq":3,"reason":{"text":"Непровар шва 2"},` +
		`"severity":"major","signal_ids":["sig-1"],"workplace_id":"WP-QC-1"}`
	et := "decision.nonconformity.confirmed"
	item := "ITEM-017"
	return procs.SignBlock{Level: level, PayloadType: dom.PayloadType(dom.ClassEvent, 1), PayloadB64: base64.StdEncoding.EncodeToString([]byte(body)),
		EventType: &et, CommandRequest: &procs.SignBlockCommandRequest{Operation: "nonconformity.nonconformity.confirm",
			Params: procs.SignBlockCommandRequestParams{"nc_id": "NC-ITEM-017-1"}, ItemID: &item}}
}

var at = time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)

// AD-12: отпечаток агента = отпечаток сервера. Событие-команда агента
// собрано тем же RequestData, что ожидает сервер (ExpectRequest), и тем же
// CommandEvent, что у demo-signer; подпись проверяется открытым ключом.
func TestPrepareMatchesServer(t *testing.T) {
	g := keyFile(t, "ins-01-ta@1", dom.ProfileGost, 7)
	s, b := sealed(t, "1234", g)
	blk := decisionBlock(2)
	p, err := Prepare(blk, s.PersonID, s.Keys, Context{Now: at})
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	raw, _ := base64.StdEncoding.DecodeString(blk.PayloadB64)
	_ = json.Unmarshal(raw, &body)
	exp, err := app.ExpectRequest("nonconformity.nonconformity.confirm", map[string]string{"nc_id": "NC-ITEM-017-1"}, body,
		"decision.nonconformity.confirmed", "0192f1a0-0000-7000-8000-000000000001", "ITEM-017", "INS-01", 2, true)
	if err != nil {
		t.Fatal(err)
	}
	var ev struct {
		EventID   string          `json:"event_id"`
		EventType string          `json:"event_type"`
		ItemID    string          `json:"item_id"`
		Data      json.RawMessage `json:"data"`
		Command   map[string]any  `json:"command"`
	}
	if err := json.Unmarshal(p.Payload(), &ev); err != nil {
		t.Fatal(err)
	}
	a, _ := dom.Canonical(ev.Data)
	e, _ := dom.Canonical(exp.Data)
	if !bytes.Equal(a, e) || ev.EventType != exp.EventType || ev.EventID != exp.CommandID || ev.ItemID != exp.ItemID {
		t.Fatalf("подписанное расходится с ожиданием сервера:\n%s\n%s", a, e)
	}
	if ev.Command["workplace_id"] != "WP-QC-1" || ev.Command["client_signed_at"] != "2026-09-26T10:00:00.000Z" {
		t.Fatalf("workplace_id и client_signed_at под подписью: %v", ev.Command)
	}
	if err := dom.CheckLevel(ev.EventType, exp.Level, 2, exp.Critical); err != nil {
		t.Fatal(err)
	}
	// Тот же CommandEvent, что строит demo-signer.
	ce := dom.CommandEvent{EventType: exp.EventType, CommandID: exp.CommandID, ItemID: "ITEM-017", SourceID: "INS-01", Data: exp.Data,
		Level: 2, BasisSeq: 41, PolicySeq: 3, WorkplaceID: "WP-QC-1", ClientSignedAt: "2026-09-26T10:00:00.000Z", Profile: dom.ProfileGost,
		Signers: []string{"ins-01-ta@1"}}
	c, _ := ce.Canonical()
	if dom.Digest(c) != p.DocDigest {
		t.Fatalf("отпечаток агента %s ≠ сервера %s", p.DocDigest, dom.Digest(c))
	}
	if len(p.Summary) < 3 || len(p.Summary) > 7 || p.Summary[0].Value == "" {
		t.Fatalf("сводка: %+v", p.Summary)
	}
	env, err := Sign(p, b)
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := base64.StdEncoding.DecodeString(b.Keys[0].PublicKeyB64)
	sig, _ := base64.StdEncoding.DecodeString(env.Signatures[0].Sig)
	if !profiles.Verify(dom.ProfileGost, pub, p.PayloadType, dom.PAE(p.PayloadType, p.Payload()), sig) {
		t.Fatal("подпись не проверяется открытым ключом")
	}
	// Сервер ожидает другой отпечаток — агент не подписывает.
	other := "streebog256:" + strings.Repeat("0", 64)
	blk.ExpectedDocDigest = &other
	if _, err := Prepare(blk, s.PersonID, s.Keys, Context{Now: at}); CodeOf(err) != CodeChanged {
		t.Fatalf("чужой отпечаток: %v", err)
	}
}

// PRD §11.16, AD-13: «годен» и решения ОТК уровнем 1 агент отклоняет сам;
// факты исполнителя из перечня — подписывает; частота уровня 1 ограничена.
func TestLevel1(t *testing.T) {
	g := keyFile(t, "ins-01-ta@1", dom.ProfileGost, 7)
	s, _ := sealed(t, "1234", g)
	if _, err := Prepare(decisionBlock(1), s.PersonID, s.Keys, Context{Now: at}); CodeOf(err) != CodeLevel {
		t.Fatalf("решение уровнем 1: %v", err)
	}
	blk := decisionBlock(1)
	pass := "inspection.result.recorded"
	blk.EventType = &pass
	if _, err := Prepare(blk, s.PersonID, s.Keys, Context{Now: at}); CodeOf(err) != CodeLevel {
		t.Fatalf("«годен» уровнем 1: %v", err)
	}
	started := "operation.run.started"
	blk.EventType = &started
	blk.CommandRequest.Operation = "mes.operation.start"
	if _, err := Prepare(blk, s.PersonID, s.Keys, Context{Now: at}); err != nil {
		t.Fatalf("факт исполнителя уровнем 1: %v", err)
	}
	var j []Entry
	for i := range DefaultRate.Max {
		j = append(j, Entry{Seq: int64(i + 1), Level: 1, SignedAt: at.Add(time.Duration(i) * time.Second).Format(TimeLayout)})
	}
	if _, ok := RateCheck(j, at.Add(30*time.Second), DefaultRate); ok {
		t.Fatal("частота уровня 1 не ограничена")
	}
	if _, ok := RateCheck(j, at.Add(2*time.Minute), DefaultRate); !ok {
		t.Fatal("окно частоты не сдвигается")
	}
}

// Операции для оболочек: подпись требует подтверждённого отпечатка, журнал
// растёт, сменный рапорт считает корень по журналу.
func TestAPISign(t *testing.T) {
	g := keyFile(t, "ins-01-ta@1", dom.ProfileGost, 7)
	s, _ := sealed(t, "1234", g)
	sj, _ := json.Marshal(s)
	un := APIUnlock(string(sj), "1234")
	if un["ok"] != true {
		t.Fatalf("unlock: %v", un)
	}
	if r := APIUnlock(string(sj), "9999"); r["code"] != CodePIN {
		t.Fatalf("неверный PIN: %v", r)
	}
	blk, _ := json.Marshal(decisionBlock(2))
	ctx := CallContext{Now: "2026-09-26T10:00:00.000Z"}
	cj, _ := json.Marshal(ctx)
	pr := APIPrepare(string(blk), string(sj), string(cj))
	if pr["ok"] != true {
		t.Fatalf("prepare: %v", pr)
	}
	p := pr["prepared"].(Prepared)
	call := SignCall{Block: blk, Sealed: sj, DKB64: un["dk_b64"].(string), Context: ctx, ConfirmedDigest: "streebog256:" + strings.Repeat("1", 64)}
	b, _ := json.Marshal(call)
	if r := APISign(string(b)); r["code"] != CodeChanged {
		t.Fatalf("не тот отпечаток подтверждён: %v", r)
	}
	call.ConfirmedDigest = p.DocDigest
	b, _ = json.Marshal(call)
	r := APISign(string(b))
	if r["ok"] != true || len(r["journal"].([]Entry)) != 1 || r["key_storage"] != dom.StorageSoftwareBrowser {
		t.Fatalf("sign: %v", r)
	}
	j, _ := json.Marshal(r["journal"])
	base, _ := json.Marshal(dom.ShiftReport{FormatVersion: 1, CryptoProfile: "gost", Signers: []string{"ins-01-ta@1"}, PersonID: "INS-01",
		ShiftID: "S-1", WindowFrom: "2026-09-26T08:00:00.000Z", WindowTo: "2026-09-26T20:00:00.000Z"})
	sr := APIShiftReport(string(j), string(base))
	if sr["ok"] != true || sr["report"].(dom.ShiftReport).LeafCount != 1 {
		t.Fatalf("сменный рапорт: %v", sr)
	}
}

// Вектор для проверки сборки WASM (extension/test/wasm-vectors.mjs): тот же
// запрос в WASM обязан дать тот же отпечаток, что посчитал здесь Go —
// сервер и агент из одного коммита (AD-12). ANT_UPDATE_VECTORS=1 — переписать.
func TestWasmVectors(t *testing.T) {
	g := keyFile(t, "ins-01-ta@1", dom.ProfileGost, 7)
	s, _ := sealed(t, "1234", g)
	blk := decisionBlock(2)
	ctx := CallContext{Now: "2026-09-26T10:00:00.000Z"}
	p, err := Prepare(blk, s.PersonID, s.Keys, ctx.context())
	if err != nil {
		t.Fatal(err)
	}
	type vector struct {
		Name      string          `json:"name"`
		Block     procs.SignBlock `json:"block"`
		Sealed    Sealed          `json:"sealed"`
		PIN       string          `json:"pin"`
		Context   CallContext     `json:"context"`
		DocDigest string          `json:"doc_digest"`
		Payload   string          `json:"payload_b64"`
		Level1    string          `json:"level1_refusal_code"`
		// KeyFiles — файлы ключей нескольких персон (тестовые, детерминированные):
		// WASM грузит их разом под один PIN и подписывает ключами одного человека.
		KeyFiles []string `json:"key_files,omitempty"`
	}
	path := filepath.Join("..", "..", "..", "..", "..", "extension", "test", "vectors.json")
	if os.Getenv("ANT_UPDATE_VECTORS") == "1" {
		kf := []string{string(g), string(keyFile(t, "ins-01-ta-pq@1", dom.ProfilePQ, 8)), string(keyFile(t, "hqc-01-ta@1", dom.ProfileGost, 9))}
		v := []vector{{Name: "решение контролёра, уровень 2", Block: blk, Sealed: s, PIN: "1234", Context: ctx, DocDigest: p.DocDigest,
			Payload: p.PayloadB64, Level1: CodeLevel, KeyFiles: kf}}
		b, _ := json.MarshalIndent(v, "", "  ")
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v — сгенерируйте: ANT_UPDATE_VECTORS=1 go test ./cmd/token-agent/...", err)
	}
	var vs []vector
	if err := json.Unmarshal(raw, &vs); err != nil || len(vs) == 0 {
		t.Fatal(err)
	}
	for _, v := range vs {
		got, err := Prepare(v.Block, v.Sealed.PersonID, v.Sealed.Keys, v.Context.context())
		if err != nil || got.DocDigest != v.DocDigest || got.PayloadB64 != v.Payload {
			t.Fatalf("%s: вектор разошёлся с Go (%v): %s ≠ %s", v.Name, err, got.DocDigest, v.DocDigest)
		}
		if _, err := Open(v.Sealed, DeriveKey(v.Sealed, v.PIN)); err != nil {
			t.Fatalf("%s: хранилище вектора: %v", v.Name, err)
		}
	}
}

// AD-14: без содержимого (только ожидаемый отпечаток) агент не подписывает.
func TestNoBlindSigning(t *testing.T) {
	g := keyFile(t, "ins-01-ta@1", dom.ProfileGost, 7)
	s, _ := sealed(t, "1234", g)
	d := "streebog256:" + strings.Repeat("a", 64)
	blk := procs.SignBlock{Level: 2, PayloadType: dom.PayloadType(dom.ClassDocumentSignature, 1), PayloadB64: "", ExpectedDocDigest: &d}
	if _, err := Prepare(blk, s.PersonID, s.Keys, Context{Now: at}); CodeOf(err) != CodeNoPath {
		t.Fatalf("подпись по отпечатку: %v", err)
	}
}

// Д-72, демо из одного браузера: ключи многих персон под одним PIN — argon2id
// один раз (общая соль), по хранилищу на ключ; подпись — только ключами
// одного человека, «суперключа» нет.
func TestSealEach(t *testing.T) {
	files := [][]byte{
		keyFile(t, "ins-01-ta@1", dom.ProfileGost, 7),
		keyFile(t, "ins-01-ta-pq@1", dom.ProfilePQ, 8),
		keyFile(t, "hqc-01-ta@1", dom.ProfileGost, 9),
		keyFile(t, "ins-01-ta@1", dom.ProfileGost, 7),
	}
	k := fastKDF
	o := SealOptions{KeyStorage: dom.StorageSoftwareBrowser, StorageVariant: dom.VariantExtension, KDF: &k}
	list, dk, err := SealEach(files, "1234", o, nil, rand.Reader)
	if err != nil || len(list) != 3 {
		t.Fatalf("SealEach: %v, %d хранилищ", err, len(list))
	}
	for _, s := range list {
		if s.KDF.SaltB64 != list[0].KDF.SaltB64 || len(s.Keys) != 1 || s.PersonID != PersonOf(s.Keys[0].KeyRef) {
			t.Fatalf("хранилище: %+v", s)
		}
		if _, err := Open(s, dk); err != nil {
			t.Fatalf("один ключ из PIN открывает все: %v", err)
		}
	}
	// Дозагрузка: тот же PIN — та же соль; другой PIN — отказ.
	more, dk2, err := SealEach([][]byte{keyFile(t, "tec-01-ta@1", dom.ProfileGost, 10)}, "1234", o, &list[0], rand.Reader)
	if err != nil || more[0].KDF.SaltB64 != list[0].KDF.SaltB64 || !bytes.Equal(dk, dk2) {
		t.Fatalf("дозагрузка под тем же PIN: %v", err)
	}
	if _, _, err := SealEach(files[:1], "9999", o, &list[0], rand.Reader); !errors.Is(err, ErrPIN) {
		t.Fatalf("дозагрузка под другим PIN: %v", err)
	}

	// Подпись hybrid ключами INS-01 из двух хранилищ.
	ins, _ := json.Marshal([]Sealed{list[0], list[1]})
	blk, _ := json.Marshal(decisionBlock(2))
	ctx := CallContext{Now: "2026-09-26T10:00:00.000Z", Profile: dom.ProfileHybrid}
	cj, _ := json.Marshal(ctx)
	pr := APIPrepare(string(blk), string(ins), string(cj))
	if pr["ok"] != true || len(pr["prepared"].(Prepared).Signers) != 2 || pr["prepared"].(Prepared).PersonID != "INS-01" {
		t.Fatalf("prepare hybrid: %v", pr)
	}
	if un := APIUnlock(string(ins), "1234"); un["ok"] != true {
		t.Fatalf("unlock набора: %v", un)
	}
	call := SignCall{Block: blk, Sealed: ins, DKB64: base64.StdEncoding.EncodeToString(dk), Context: ctx,
		ConfirmedDigest: pr["prepared"].(Prepared).DocDigest}
	cb, _ := json.Marshal(call)
	if r := APISign(string(cb)); r["ok"] != true || len(r["envelope"].(dom.Envelope).Signatures) != 2 {
		t.Fatalf("sign hybrid: %v", r)
	}
	// Ключи разных людей в одной подписи — отказ.
	mixed, _ := json.Marshal([]Sealed{list[1], list[2]})
	if r := APIPrepare(string(blk), string(mixed), string(cj)); r["ok"] != false || r["code"] != CodeInvalid {
		t.Fatalf("чужой ключ в подписи: %v", r)
	}
	// Через API: ответ — хранилища и ключ сеанса.
	req, _ := json.Marshal(SealRequest{Files: []string{string(files[0])}, StorageVariant: dom.VariantExtension, Now: "2026-09-26T10:00:00.000Z"})
	if r := APISealEach(string(req), "12"); r["ok"] != false {
		t.Fatalf("короткий PIN: %v", r)
	}
}
