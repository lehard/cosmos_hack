package hybrid

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.stargrave.org/gogost/v7/gost3410"
)

// AD-10, AD-32: пакет hybrid проверяется обеими подписями; удаление любой —
// понижение профиля; подмена payload — «подпись не сходится».
func TestHybridSignVerify(t *testing.T) {
	dir := t.TempDir()
	keys, err := Generate(dir, "keeper")
	if err != nil {
		t.Fatal(err)
	}
	s, err := Load(dir, "keeper")
	if err != nil {
		t.Fatal(err)
	}
	a := Anchors{FormatVersion: 1, Keys: keys}
	env, err := s.Sign("checkpoint", []byte(`{"checkpoint_no":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if p, err := a.Verify(env, "checkpoint", "keeper"); err != nil || string(p) != `{"checkpoint_no":1}` {
		t.Fatalf("проверка: %v", err)
	}
	if _, err := a.Verify(env, "verifier-report", "keeper"); !errors.Is(err, ErrClass) {
		t.Fatalf("класс: %v", err)
	}
	var e Envelope
	_ = json.Unmarshal(env, &e)
	e1 := e
	e1.Signatures = e.Signatures[:1]
	b, _ := json.Marshal(e1)
	if _, err := a.Verify(b, "checkpoint", "keeper"); !errors.Is(err, ErrDowngrade) {
		t.Fatalf("понижение профиля: %v", err)
	}
	e2 := e
	e2.Payload = base64.StdEncoding.EncodeToString([]byte(`{"checkpoint_no":2}`))
	b, _ = json.Marshal(e2)
	if _, err := a.Verify(b, "checkpoint", "keeper"); !errors.Is(err, ErrBadSig) {
		t.Fatalf("подмена: %v", err)
	}
	if _, err := a.Verify(env, "checkpoint", "verifier"); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("чужой субъект: %v", err)
	}
}

// Кодирование ГОСТ совпадает с тест-векторами контракта (contracts/crypto/test-vectors).
func TestGostVector(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "contracts", "crypto", "test-vectors", "vectors.v1.json"))
	if err != nil {
		t.Skip(err)
	}
	var v struct {
		DssePae struct {
			PaeHex string `json:"pae_hex"`
		} `json:"dsse_pae"`
		Gost struct {
			PublicKeyLEHex string `json:"public_key_le_hex"`
			SignatureHex   string `json:"signature_hex"`
		} `json:"gost3410_2012_256"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	pae, _ := hex.DecodeString(v.DssePae.PaeHex)
	pubRaw, _ := hex.DecodeString(v.Gost.PublicKeyLEHex)
	sig, _ := hex.DecodeString(v.Gost.SignatureHex)
	pub, err := gost3410.NewPublicKeyLE(curve(), pubRaw)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := (gost3410.PublicKeyReverseDigest{Pub: pub}).VerifyDigest(Streebog(pae), sig)
	if err != nil || !ok {
		t.Fatalf("вектор ГОСТ не проверился: %v", err)
	}
}
