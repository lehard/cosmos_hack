package signing

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
	"uuid"

	"ant/internal/application/ingest/inmem"
	"ant/internal/application/platform"
	app "ant/internal/application/signing"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// Проверка подписи команды общим декоратором (Д-59): модуль signing как порт
// platform.SignatureChecker — подписан запрос JCS{operation, params, body},
// класс хранения ключа — из акта регистрации, конверт клиента — в блок
// command записи (platform.SignRecord), его находят верификатор и сменный
// рапорт (domain/signing.RecordSignatures).

// confirmAct — операция подтверждения несоответствия (уровень 2, критическое).
var confirmAct = platform.Action{ID: "nonconformity.nonconformity.confirm", Class: platform.ClassProtective, Critical: true,
	Owner: "nonconformity", Emits: []catalog.Type{catalog.DecisionNonconformityConfirmed}, SignatureLevel: 2}

func requestWorld(t *testing.T, profile string) (*app.Service, *profiles.Keyring) {
	t.Helper()
	k, err := profiles.Generate("ins-01-ta@1", dom.ProfileGost)
	if err != nil {
		t.Fatal(err)
	}
	reg := app.RegistrationData{KeyRef: k.Ref, SubjectKind: dom.SubjectDemoPersona, SubjectID: "INS-01", ProfileID: dom.ProfileGost,
		Algorithm: k.Algorithm(), PublicKeyB64: k.PublicB64(), Fingerprint: k.Fingerprint(), PayloadClasses: []string{dom.ClassEvent},
		SubjectConfirmation: dom.ConfirmGenesis, ValidFrom: t0.Add(-time.Hour).Format(app.TimeLayout),
		KeyStorage: dom.StorageSoftwareBrowser, StorageVariant: dom.VariantExtension}
	g, err := reg.Registration()
	if err != nil {
		t.Fatal(err)
	}
	g.Provenance = dom.ProvScenario
	clock := inmem.NewClock(t0)
	j := inmem.NewJournal(clock.At)
	svc := app.NewService(app.WithConfig(app.Config{Profile: profile, AllowUnsigned: app.SignatureModeFor(profile), Partitions: 16}),
		app.WithDeps(app.Deps{Journal: j, Registry: app.NewRegistry(j, []dom.Registration{g}, nil), Authorities: app.DefaultAuthorities(), Now: clock.At}))
	svc.UseCrypto(profiles.Verifier{Keys: svc.Registry()})
	return svc, profiles.NewKeyring(k)
}

// signedRequest — тело команды и подпись запроса ключом персоны (как demo-signer и расширение).
func signedRequest(t *testing.T, keys *profiles.Keyring, params map[string]string, body map[string]any, item string) platform.SignedRequest {
	t.Helper()
	data, err := dom.RequestData(confirmAct.ID, params, body)
	if err != nil {
		t.Fatal(err)
	}
	ce := dom.CommandEvent{EventType: string(catalog.DecisionNonconformityConfirmed), CommandID: body["command_id"].(string), ItemID: item,
		SourceID: "INS-01", Data: data, Level: 2, ClientSignedAt: t0.Format(app.TimeLayout), Signers: []string{"ins-01-ta@1"}}
	c, err := ce.Canonical()
	if err != nil {
		t.Fatal(err)
	}
	env, err := profiles.Signer{Keys: keys}.Envelope(dom.PayloadType(dom.ClassEvent, 1), c, "ins-01-ta@1")
	if err != nil {
		t.Fatal(err)
	}
	withSig := map[string]any{}
	for k, v := range body {
		withSig[k] = v
	}
	withSig["signature"] = env
	raw, _ := json.Marshal(withSig)
	return platform.SignedRequest{Params: params, Body: raw,
		Meta: platform.CommandMeta{CommandID: ce.CommandID, Signature: env.Marshal()}}
}

func errCodeOf(err error) errcodes.Code {
	var pe *platform.Error
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

func TestCheckRequestSignedDecision(t *testing.T) {
	svc, keys := requestWorld(t, "prod")
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "INS-01"})
	params := map[string]string{"nc_id": "NC-ITEM-017-1"}
	body := map[string]any{"command_id": uuid.NewV7().String(), "basis_seq": 12, "policy_seq": 3, "reason": map[string]any{"text": "подтверждаю"}}
	rq := signedRequest(t, keys, params, body, "ITEM-017")

	sig, skip, err := svc.CheckRequest(ctx, confirmAct, rq)
	if err != nil || skip {
		t.Fatalf("подписанный запрос не принят: skip=%v err=%v", skip, err)
	}
	if !sig.Signed() || sig.KeyStorage != dom.StorageSoftwareBrowser || sig.Method != dom.MethodTokenAgent || sig.ItemID != "ITEM-017" {
		t.Fatalf("итог: %+v", sig)
	}
	// Подпись — в блок command записи; изделие команды другое — отказ.
	cmd := map[string]any{"command_id": body["command_id"]}
	sctx := platform.WithSignature(ctx, sig)
	if _, err := platform.SignRecord(sctx, cmd, "ITEM-018"); errCodeOf(err) != errcodes.SigningDocumentChanged {
		t.Fatalf("другое изделие: %v", err)
	}
	prov, err := platform.SignRecord(sctx, cmd, "ITEM-017")
	if err != nil || prov != dom.ProvScenario || cmd["key_storage"] != dom.StorageSoftwareBrowser || cmd["signature"] == nil {
		t.Fatalf("блок command: prov=%q err=%v cmd=%v", prov, err, cmd)
	}
	// Верификатор и сменный рапорт находят подпись рядом с записью.
	ev, _ := dom.CanonicalOf(map[string]any{"event_id": body["command_id"], "command": cmd, "data": map[string]any{}})
	_, _, sigs, nested := dom.RecordSignatures(dom.Seal(dom.PayloadType(dom.ClassEvent, 1), ev).Marshal())
	if !nested || len(sigs) != 1 || sigs[0].KeyID != "ins-01-ta@1" {
		t.Fatalf("подписи записи: nested=%v %v", nested, sigs)
	}

	// Тело изменено после подписи — «подписанное расходится с командой».
	changed := signedRequest(t, keys, params, body, "ITEM-017")
	var m map[string]any
	_ = json.Unmarshal(changed.Body, &m)
	m["reason"] = map[string]any{"text": "другое основание"}
	changed.Body, _ = json.Marshal(m)
	if _, _, err := svc.CheckRequest(ctx, confirmAct, changed); errCodeOf(err) != errcodes.SigningDocumentChanged {
		t.Fatalf("изменённое тело: %v", err)
	}
	// Подпись чужим ключом — «чужой ключ».
	other := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "INS-02"})
	if _, _, err := svc.CheckRequest(other, confirmAct, rq); errCodeOf(err) != errcodes.SigningForeignKey {
		t.Fatalf("чужой ключ: %v", err)
	}
}

func TestCheckRequestUnsigned(t *testing.T) {
	params := map[string]string{"nc_id": "NC-ITEM-017-1"}
	raw, _ := json.Marshal(map[string]any{"command_id": uuid.NewV7().String(), "basis_seq": 1, "policy_seq": 1})
	rq := platform.SignedRequest{Params: params, Body: raw}
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "INS-01"})

	prod, _ := requestWorld(t, "prod")
	if _, _, err := prod.CheckRequest(ctx, confirmAct, rq); errCodeOf(err) != errcodes.SigningNoSignaturePath {
		t.Fatalf("prod без подписи: %v", err)
	}
	demo, _ := requestWorld(t, "demo")
	sig, skip, err := demo.CheckRequest(ctx, confirmAct, rq)
	if err != nil || skip || sig.Signed() || sig.Status != string(dom.StatusUnsigned) {
		t.Fatalf("демо без подписи: %+v skip=%v err=%v", sig, skip, err)
	}
	cmd := map[string]any{}
	if prov, err := platform.SignRecord(platform.WithSignature(ctx, sig), cmd, "ITEM-017"); err != nil || prov != "" || len(cmd) != 0 {
		t.Fatalf("неподписанная команда в блоке command: %q %v %v", prov, err, cmd)
	}
	// Акты ключей и подписи документов проверяют подпись сами.
	for _, act := range []platform.Action{
		{ID: "signing.key.register", Owner: "signing", Emits: []catalog.Type{catalog.KeyRegistrationRecorded}, SignatureLevel: 2},
		{ID: "documents.signature.record", Owner: "documents", Emits: []catalog.Type{catalog.DocumentSignatureRecorded}, SignatureLevel: 2},
	} {
		if _, skip, err := prod.CheckRequest(ctx, act, rq); !skip || err != nil {
			t.Fatalf("%s: skip=%v err=%v", act.ID, skip, err)
		}
	}
}
