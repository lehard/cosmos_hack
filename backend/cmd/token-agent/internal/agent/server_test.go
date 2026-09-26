package agent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"ant/internal/application/ingest/inmem"
	"ant/internal/application/platform"
	app "ant/internal/application/signing"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// Сервер принимает подпись ключом в браузере (AD-12, AD-14, Д-72): конверт,
// собранный ядром агента (тем же, что в WASM), проходит проверку команды
// модуля signing (CheckCommand с ожиданием ExpectRequest) — подписано ровно
// исполняемое; класс хранения ключа берётся из акта регистрации и виден в
// итоге проверки; подпись другим человеком — «чужой ключ».
func TestServerAcceptsBrowserSignature(t *testing.T) {
	k, _ := profiles.FromSecret("ins-01-ta@1", dom.ProfileGost, bytes.Repeat([]byte{7}, 32))
	t0 := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	clock := inmem.NewClock(t0)
	j := inmem.NewJournal(clock.At)
	reg := app.RegistrationData{KeyRef: k.Ref, SubjectKind: dom.SubjectDemoPersona, SubjectID: "INS-01", ProfileID: dom.ProfileGost,
		Algorithm: k.Algorithm(), PublicKeyB64: k.PublicB64(), Fingerprint: k.Fingerprint(), PayloadClasses: []string{dom.ClassEvent},
		SubjectConfirmation: dom.ConfirmGenesis, ValidFrom: t0.Add(-time.Hour).Format(app.TimeLayout),
		KeyStorage: dom.StorageSoftwareBrowser, StorageVariant: dom.VariantExtension}
	g, err := reg.Registration()
	if err != nil {
		t.Fatal(err)
	}
	g.Provenance = dom.ProvScenario
	svc := app.NewService(app.WithConfig(app.Config{Profile: "demo", Partitions: 16, DomainBuild: dom.Digest([]byte("test"))}),
		app.WithDeps(app.Deps{Journal: j, Registry: app.NewRegistry(j, []dom.Registration{g}, nil), Authorities: app.DefaultAuthorities(), Now: clock.At}))
	svc.UseCrypto(profiles.Verifier{Keys: svc.Registry()})

	f, _ := json.Marshal(KeyFile{KeyRef: k.Ref, Profile: k.Profile, SecretHex: hex.EncodeToString(k.Secret())})
	b, err := NewBundle([][]byte{f}, "")
	if err != nil {
		t.Fatal(err)
	}
	var keys []KeyInfo
	for _, x := range b.Keys {
		keys = append(keys, KeyInfo{KeyRef: x.KeyRef, Profile: x.Profile, Fingerprint: x.Fingerprint})
	}
	blk := decisionBlock(2)
	p, err := Prepare(blk, b.PersonID, keys, Context{Now: t0})
	if err != nil {
		t.Fatal(err)
	}
	env, err := Sign(p, b)
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
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "INS-01"})
	a, err := svc.CheckCommand(ctx, env.Marshal(), exp)
	if err != nil {
		t.Fatalf("сервер не принял подпись ключом в браузере: %v", err)
	}
	if a.Status != dom.StatusValid || a.KeyStorage != dom.StorageSoftwareBrowser || a.StorageVariant != dom.VariantExtension ||
		a.Method != dom.MethodTokenAgent || a.SignerPersonID != "INS-01" {
		t.Fatalf("итог проверки: %+v", a)
	}
	// Другой пользователь сеанса — «чужой ключ».
	other := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "INS-02"})
	exp.Actor = "INS-02"
	if _, err := svc.CheckCommand(other, env.Marshal(), exp); err == nil {
		t.Fatal("подпись чужим ключом принята")
	}
}

