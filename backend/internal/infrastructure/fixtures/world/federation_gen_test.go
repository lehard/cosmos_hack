package world

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dom "ant/internal/domain/federation"
	"ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// ANT_UPDATE_FEDERATION=1 go test -run TestFederationSamples ./internal/infrastructure/fixtures/world
// — пересобрать подписанные выписки сценария (эпик 41, AD-19): ключи
// партнёров создаются заново (закрытые части не сохраняются — фикстура
// самодостаточна: корни в манифесте, акты ключей внутри выписок). После —
// `-update` генератора мира (копия входов).

type sampleKey struct {
	k    *profiles.PrivateKey
	role string
	name string
}

func genKey(t *testing.T, ref, role, name string) sampleKey {
	t.Helper()
	k, err := profiles.Generate(ref, signing.ProfileGost)
	if err != nil {
		t.Fatal(err)
	}
	return sampleKey{k: k, role: role, name: name}
}

func signEnvelope(t *testing.T, payloadType string, payload []byte, keys ...sampleKey) []byte {
	t.Helper()
	var ks []*profiles.PrivateKey
	for _, k := range keys {
		ks = append(ks, k.k)
	}
	env, err := profiles.Signer{Keys: profiles.NewKeyring(ks...)}.Envelope(payloadType, payload, refsOf(keys)...)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.MarshalIndent(env, "", "  ")
	return append(b, '\n')
}

func refsOf(keys []sampleKey) []string {
	var out []string
	for _, k := range keys {
		out = append(out, k.k.Ref)
	}
	return out
}

// buildSample — выписка отправителя enterprise: акты ключей людей подписаны
// корнем root, пакет — людьми и шлюзом gateway.
func buildSample(t *testing.T, x dom.Extract, root, gateway sampleKey, people ...sampleKey) ([]byte, []byte) {
	t.Helper()
	x.Roots = []dom.Root{{KeyRef: root.k.Ref, Profile: root.k.Profile, PublicB64: root.k.PublicB64()}, {KeyRef: gateway.k.Ref, Profile: gateway.k.Profile, PublicB64: gateway.k.PublicB64()}}
	for _, p := range people {
		act, err := signing.CanonicalOf(dom.KeyAct{Enterprise: x.Sender, KeyRef: p.k.Ref, Profile: p.k.Profile, PublicB64: p.k.PublicB64(), Role: p.role})
		if err != nil {
			t.Fatal(err)
		}
		raw := signEnvelope(t, dom.KeyActPayloadType, act, root)
		var compact json.RawMessage = mustCompact(t, raw)
		x.Signers = append(x.Signers, dom.Signer{KeyRef: p.k.Ref, Role: p.role, Name: p.name, KeyAct: compact})
	}
	x.Signers = append(x.Signers, dom.Signer{KeyRef: gateway.k.Ref, Role: "шлюз предприятия", Name: "Ключ шлюза " + x.Sender})
	payload, err := signing.CanonicalOf(x)
	if err != nil {
		t.Fatal(err)
	}
	return signEnvelope(t, dom.ExtractPayloadType, payload, append(people, gateway)...), payload
}

func mustCompact(t *testing.T, raw []byte) []byte {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(v)
	return b
}

func TestFederationSamples(t *testing.T) {
	if os.Getenv("ANT_UPDATE_FEDERATION") != "1" {
		t.Skip("ANT_UPDATE_FEDERATION=1 — пересобрать выписки scenarios/federation")
	}
	dir := filepath.Join(repoRoot, FederationDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name string, b []byte) {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cp := &dom.Checkpoint{Keeper: "keeper.mz01", HeadSeq: 184213, HeadHash: signing.Digest([]byte("mz01 head 184213")), At: "2026-09-16T18:00:00Z"}

	// МЗ-1 — металлургический завод: плавка 5512 для партии заготовок ЗФ-201.
	mzRoot, mzGw := genKey(t, "mz01-root@1", "", ""), genKey(t, "mz01-gateway@1", "", "")
	mzOTK := genKey(t, "mz01-otk-sidorova@1", "контролёр ОТК", "Сидорова А. В.")
	mzLab := genKey(t, "mz01-lab-petrov@1", "начальник ЦЗЛ", "Петров И. С.")
	mz := dom.Extract{FormatVersion: 1, CryptoProfile: signing.ProfileGost, Sender: "MZ01", Recipient: "ENT01", ExtractNo: "МЗ-ВП-2026-0917-5512",
		IssuedAt: "2026-09-16T18:10:00Z",
		Subject:  dom.Subject{Kind: "lot", GlobalID: "MZ01:HEAT-5512-P3", Label: "Поковки Ø220 из плавки 5512 (42 шт.)", Quantity: "42 шт.", RecipientRef: "LOT-ZF-201"},
		Origin: dom.Origin{HeatNo: "5512", Material: "сталь 09Г2С", Standard: "ГОСТ 19281-2014", Certificate: "С-201",
			Chemistry:  map[string]string{"C": "0,10", "Mn": "1,55", "Si": "0,68", "S": "0,021", "P": "0,019"},
			Mechanical: map[string]string{"σв, МПа": "510", "σт, МПа": "355", "δ5, %": "23", "KCU−40, Дж/см²": "44"}},
		Control: []dom.Control{
			{Name: "Химический состав (спектральный анализ)", Result: "соответствует", Norm: "ГОСТ 19281-2014", By: "mz01-lab-petrov@1"},
			{Name: "Механические свойства", Result: "соответствует", Value: "σв 510 МПа", Norm: "≥ 490 МПа", By: "mz01-lab-petrov@1"},
			{Name: "Ультразвуковой контроль поковок", Result: "годен", Norm: "ГОСТ 24507-80, группа 2", By: "mz01-otk-sidorova@1"},
			{Name: "Приёмка ОТК партии", Result: "принято", By: "mz01-otk-sidorova@1"},
		},
		Decisions:  []string{"Несоответствий по партии нет"},
		Checkpoint: cp,
		Sources:    []string{"MZ01:heat/5512", "MZ01:cert/С-201"},
	}
	mzRaw, _ := buildSample(t, mz, mzRoot, mzGw, mzOTK, mzLab)
	write("mz01-heat-5512.extract.json", mzRaw)
	// Изменённая копия: после подписи в химсоставе «подправлена» сера.
	write("mz01-heat-5512.tampered.json", tamper(t, mzRaw, `"S":"0,021"`, `"S":"0,015"`))

	// Поставщик колец ПК-2 — только подпись шлюза: «подтверждено сервером отправителя».
	pkRoot, pkGw := genKey(t, "pk02-root@1", "", ""), genKey(t, "pk02-gateway@1", "", "")
	pk := dom.Extract{FormatVersion: 1, CryptoProfile: signing.ProfileGost, Sender: "PK02", Recipient: "ENT01", ExtractNo: "ПК2-116",
		IssuedAt: "2026-09-16T12:00:00Z",
		Subject:  dom.Subject{Kind: "lot", GlobalID: "PK02:R-116", Label: "Кольца ФЛ-100.01.002 (33 шт.)", Quantity: "33 шт.", RecipientRef: "LOT-R-116"},
		Origin:   dom.Origin{HeatNo: "7741", Material: "сталь 09Г2С", Certificate: "С-116"},
		Control:  []dom.Control{{Name: "Размерный контроль", Result: "годен"}},
	}
	pkRaw, _ := buildSample(t, pk, pkRoot, pkGw)
	write("pk02-lot-r116.extract.json", pkRaw)

	// Наше предприятие ENT01 → сборщик СБ-1: выписка паспорта фланца Ф-001.
	enRoot, enGw := genKey(t, "ent01-root@1", "", ""), genKey(t, "gateway-ingest@1", "", "")
	enOTK := genKey(t, "ent01-ins-01@1", "контролёр ОТК", "Контролёр ОТК INS-01")
	en := dom.Extract{FormatVersion: 1, CryptoProfile: signing.ProfileGost, Sender: "ENT01", Recipient: "SB01", ExtractNo: "ENT01-ВП-F-001",
		IssuedAt: "2026-09-23T11:00:00Z",
		Subject:  dom.Subject{Kind: "item", GlobalID: "ENT01:F-001", Label: "Фланец Ф-001 (ФЛ-100)"},
		Origin:   dom.Origin{HeatNo: "5512", Material: "сталь 09Г2С", Certificate: "МЗ-ВП-2026-0917-5512"},
		Control: []dom.Control{
			{Name: "Рентгенографический контроль сварного шва", Result: "годен", By: "ent01-ins-01@1"},
			{Name: "Гидроиспытание 6,3 МПа", Result: "годен", By: "ent01-ins-01@1"},
			{Name: "Приёмка ОТК (ЗТ-6)", Result: "принято", By: "ent01-ins-01@1"},
		},
		Decisions:  []string{"Выпуск разрешён"},
		Checkpoint: &dom.Checkpoint{Keeper: "keeper.ent01", HeadSeq: 4210, HeadHash: signing.Digest([]byte("ent01 head 4210")), At: "2026-09-23T10:59:00Z"},
		Sources:    []string{"ENT01:item/F-001", "MZ01:HEAT-5512-P3"},
	}
	enRaw, _ := buildSample(t, en, enRoot, enGw, enOTK)
	write("ent01-f-001-to-sb01.extract.json", enRaw)

	fp := func(ks ...sampleKey) []string {
		var out []string
		for _, k := range ks {
			out = append(out, k.k.Fingerprint())
		}
		return out
	}
	man := FederationManifest{
		Self: FederationPartner{Code: "ENT01", Name: "Наше предприятие", Roots: fp(enRoot, enGw)},
		Partners: []FederationPartner{
			{Code: "MZ01", Name: "Металлургический завод МЗ-1", Role: "supplier", Endpoint: "https://mz01.partners.stand/federation/v1", Roots: fp(mzRoot, mzGw), DocumentID: "DOC-PARTNER-MZ01"},
			{Code: "PK02", Name: "Поставщик колец ПК-2", Role: "supplier", Endpoint: "https://pk02.partners.stand/federation/v1", Roots: fp(pkRoot, pkGw), DocumentID: "DOC-PARTNER-PK02"},
			{Code: "SB01", Name: "Сборочное предприятие СБ-1", Role: "assembler", Endpoint: "https://sb01.partners.stand/federation/v1", Roots: []string{signing.Digest([]byte("sb01-root"))}, DocumentID: "DOC-PARTNER-SB01"},
		},
		Incoming: []FederationSample{{File: "mz01-heat-5512.extract.json", Partner: "MZ01"}, {File: "pk02-lot-r116.extract.json", Partner: "PK02"}},
		Outgoing: []FederationSample{{File: "ent01-f-001-to-sb01.extract.json", Partner: "SB01", Item: "F-001", DocumentID: "DOC-EXTRACT-F-001", MessageID: "0192c3a0-0000-7000-8000-00000000f001"}},
		Tampered: "mz01-heat-5512.tampered.json",
	}
	b, _ := json.MarshalIndent(man, "", "  ")
	write("manifest.json", append(b, '\n'))
}

func tamper(t *testing.T, raw []byte, from, to string) []byte {
	t.Helper()
	var env signing.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	_, payload, err := signing.ParseEnvelope(raw)
	if err != nil {
		t.Fatal(err)
	}
	p := strings.Replace(string(payload), from, to, 1)
	if p == string(payload) {
		t.Fatalf("в выписке нет %s", from)
	}
	env.Payload = base64.StdEncoding.EncodeToString([]byte(p))
	b, _ := json.MarshalIndent(env, "", "  ")
	return append(b, '\n')
}
