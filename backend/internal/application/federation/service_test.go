package federation_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	app "ant/internal/application/federation"
	"ant/internal/application/ingest/inmem"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/federation"
	"ant/internal/domain/signing"
)

// Поддельная криптография: «подпись» = H(pub ‖ PAE) (настоящий ГОСТ — в тестах мира заготовок).
func fakeVerify(_ string, pub []byte, _ string, pae, sig []byte) bool {
	return bytes.Equal(sig, signing.Hash(pub, pae))
}

func extract(t *testing.T, heat string) ([]byte, string) {
	t.Helper()
	pub := []byte("mz-root-pub")
	x := dom.Extract{FormatVersion: 1, CryptoProfile: "gost", Sender: "MZ01", ExtractNo: "1", IssuedAt: "2026-09-16T00:00:00Z",
		Subject: dom.Subject{Kind: "lot", GlobalID: "MZ01:H-" + heat, RecipientRef: "LOT-ZF-201"}, Origin: dom.Origin{HeatNo: heat},
		Roots: []dom.Root{{KeyRef: "mz01-root@1", Profile: "gost", PublicB64: base64.StdEncoding.EncodeToString(pub)}}}
	p, err := signing.CanonicalOf(x)
	if err != nil {
		t.Fatal(err)
	}
	e := signing.Seal(dom.ExtractPayloadType, p)
	e.Signatures = []signing.Signature{{KeyID: "mz01-root@1", Sig: base64.StdEncoding.EncodeToString(signing.Hash(pub, signing.PAE(dom.ExtractPayloadType, p)))}}
	return e.Marshal(), signing.Digest(pub)
}

func header() platform.CommandHeader {
	return platform.CommandHeader{CommandID: "0192c3a0-0000-7000-8000-000000000001"}
}

// TestLiveReceive — FR-132, AD-19: партнёр зарегистрирован актом; выписка
// проверена его корнем и принята (подтверждено сервером отправителя);
// изменённая — 422 federation.extract_tampered; повтор приёма — та же запись.
func TestLiveReceive(t *testing.T) {
	ctx := context.Background()
	now := func() time.Time { return time.Date(2026, 9, 21, 9, 0, 0, 0, time.UTC) }
	j := inmem.NewJournal(now)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", Partitions: 4}
	s := app.NewService(app.Deps{Journal: j, Codec: codec, Crypto: fakeVerify, Enterprise: "ENT01", Now: now})

	raw, root := extract(t, "5512")
	h := header()
	if _, err := s.RegisterPartner(ctx, app.RegisterPartner{CommandHeader: h, PartnerCode: "MZ01", Name: "МЗ-1", RootFingerprints: []string{root}, DocumentID: "DOC-PARTNER-MZ01"}); err != nil {
		t.Fatal(err)
	}
	ps, err := s.Partners(ctx, platform.Moment{})
	if err != nil || len(ps.Items) != 1 || ps.Items[0].RootFingerprints[0] != root {
		t.Fatalf("партнёры: %v %+v", err, ps)
	}
	rc, err := s.ReceiveExtract(ctx, app.ReceiveExtract{PartnerCode: "MZ01", Envelope: string(raw)})
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.ReceiveExtract(ctx, app.ReceiveExtract{PartnerCode: "MZ01", Envelope: string(raw)})
	if err != nil || !again.Replayed || again.CommandID != rc.CommandID {
		t.Fatalf("повтор приёма: %v %+v", err, again)
	}
	l, err := s.Extracts(ctx, "incoming", "", platform.Moment{}, platform.Page{})
	if err != nil || len(l.Items) != 1 {
		t.Fatalf("выписки: %v %+v", err, l)
	}
	e := l.Items[0]
	if e.OriginStatus != dom.OriginServerOnly || e.HeatNo != "5512" || e.Subject == nil || e.Subject.ID != "LOT-ZF-201" {
		t.Fatalf("выписка: %+v", e)
	}
	// Изменённая после подписи — отказ.
	env, p, _ := signing.ParseEnvelope(raw)
	env.Payload = base64.StdEncoding.EncodeToString(bytes.Replace(p, []byte("5512"), []byte("5513"), 1))
	_, err = s.ReceiveExtract(ctx, app.ReceiveExtract{PartnerCode: "MZ01", Envelope: string(env.Marshal())})
	var pe *platform.Error
	if !errors.As(err, &pe) || pe.Code != errcodes.FederationExtractTampered {
		t.Fatalf("изменённая выписка: %v", err)
	}
	// Отправка выписки зарегистрированному партнёру — запись federation.message.sent.
	if _, err := s.SendExtract(ctx, app.SendExtract{CommandHeader: platform.CommandHeader{CommandID: "0192c3a0-0000-7000-8000-000000000002"}, PartnerCode: "MZ01", Kind: "passport_extract",
		Subject: platform.DrillRef{Entity: platform.EntityItem, ID: "ENT01:F-001"}, DocumentID: "DOC-1"}); err != nil {
		t.Fatal(err)
	}
	out, err := s.Extracts(ctx, "outgoing", "", platform.Moment{}, platform.Page{})
	if err != nil || len(out.Items) != 1 || out.Items[0].Subject.ID != "ENT01:F-001" || out.Items[0].Acknowledged != nil {
		t.Fatalf("исходящие: %v %+v", err, out)
	}
	if _, err := s.SendExtract(ctx, app.SendExtract{PartnerCode: "XX", Kind: "passport_extract"}); err == nil {
		t.Fatal("отправка незарегистрированному партнёру прошла")
	}
}
