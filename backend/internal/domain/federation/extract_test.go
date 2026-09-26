package federation

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	"ant/internal/domain/signing"
)

// Поддельная криптография для доменного теста: «подпись» = H(pub ‖ PAE).
func fakeSign(pub, pae []byte) string { return base64.StdEncoding.EncodeToString(signing.Hash(pub, pae)) }

func fakeVerify(_ string, pub []byte, _ string, pae, sig []byte) bool {
	return bytes.Equal(sig, signing.Hash(pub, pae))
}

func env(t *testing.T, pt string, v any, keys map[string][]byte, refs ...string) []byte {
	t.Helper()
	p, err := signing.CanonicalOf(v)
	if err != nil {
		t.Fatal(err)
	}
	e := signing.Seal(pt, p)
	for _, r := range refs {
		e.Signatures = append(e.Signatures, signing.Signature{KeyID: r, Sig: fakeSign(keys[r], signing.PAE(pt, p))})
	}
	return e.Marshal()
}

func sample(t *testing.T, checkpoint bool, people bool) ([]byte, []string) {
	keys := map[string][]byte{"mz-root@1": []byte("root-pub"), "mz-otk@1": []byte("otk-pub")}
	x := Extract{FormatVersion: 1, CryptoProfile: "gost", Sender: "MZ", ExtractNo: "1", IssuedAt: "2026-09-16T00:00:00Z",
		Subject: Subject{Kind: "lot", GlobalID: "MZ:H-1", RecipientRef: "LOT-1"}, Origin: Origin{HeatNo: "5512"},
		Roots: []Root{{KeyRef: "mz-root@1", Profile: "gost", PublicB64: base64.StdEncoding.EncodeToString(keys["mz-root@1"])}}}
	refs := []string{"mz-root@1"}
	if people {
		act := env(t, KeyActPayloadType, KeyAct{Enterprise: "MZ", KeyRef: "mz-otk@1", Profile: "gost", PublicB64: base64.StdEncoding.EncodeToString(keys["mz-otk@1"])}, keys, "mz-root@1")
		x.Signers = []Signer{{KeyRef: "mz-otk@1", Role: "ОТК", KeyAct: json.RawMessage(act)}}
		refs = append([]string{"mz-otk@1"}, refs...)
	}
	if checkpoint {
		x.Checkpoint = &Checkpoint{Keeper: "k", HeadSeq: 1, HeadHash: "streebog256:00"}
	}
	return env(t, ExtractPayloadType, x, keys, refs...), []string{signing.Digest(keys["mz-root@1"])}
}

func TestVerifyExtractStatuses(t *testing.T) {
	raw, roots := sample(t, true, true)
	if v, err := VerifyExtract(raw, "MZ", roots, fakeVerify); err != nil || v.Origin != OriginVerified {
		t.Fatalf("подтверждено: %v %s", err, v.Origin)
	}
	noCP, _ := sample(t, false, true)
	if v, err := VerifyExtract(noCP, "MZ", roots, fakeVerify); err != nil || v.Origin != OriginServerOnly {
		t.Fatalf("без контрольной точки: %v %s", err, v.Origin)
	}
	gw, _ := sample(t, true, false)
	if v, err := VerifyExtract(gw, "MZ", roots, fakeVerify); err != nil || v.Origin != OriginServerOnly {
		t.Fatalf("только шлюз: %v %s", err, v.Origin)
	}
	if v, err := VerifyExtract(raw, "MZ", []string{"streebog256:" + string(bytes.Repeat([]byte("0"), 64))}, fakeVerify); err != nil || v.Origin != OriginUnverified {
		t.Fatalf("чужие корни: %v %s", err, v.Origin)
	}
	if _, err := VerifyExtract(raw, "PK", roots, fakeVerify); !errors.Is(err, ErrFormat) {
		t.Fatalf("канал другого партнёра: %v", err)
	}
	// Изменённое содержимое: подпись известным ключом не сходится.
	var e signing.Envelope
	_ = json.Unmarshal(raw, &e)
	p, _ := base64.StdEncoding.DecodeString(e.Payload)
	e.Payload = base64.StdEncoding.EncodeToString(bytes.Replace(p, []byte("5512"), []byte("5513"), 1))
	if _, err := VerifyExtract(e.Marshal(), "MZ", roots, fakeVerify); !errors.Is(err, ErrTampered) {
		t.Fatalf("изменённая: %v", err)
	}
	// Не канонический JSON — тоже отказ (AD-10).
	e.Payload = base64.StdEncoding.EncodeToString(append([]byte(" "), p...))
	if _, err := VerifyExtract(e.Marshal(), "MZ", roots, fakeVerify); !errors.Is(err, ErrTampered) {
		t.Fatalf("не канонический: %v", err)
	}
}
