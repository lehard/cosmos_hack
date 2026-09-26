package profiles

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	dom "ant/internal/domain/signing"
)

type pubs map[string][2]any

func (p pubs) PublicKey(_ context.Context, ref string) (string, []byte, error) {
	v, ok := p[ref]
	if !ok {
		return "", nil, ErrKeyUnavailable
	}
	return v[0].(string), v[1].([]byte), nil
}

// Кодирование ГОСТ и ML-DSA совпадает с тест-векторами контракта (AD-12:
// «отпечаток сервера = отпечаток агента»); гибридный конверт вектора
// проверяется обеими подписями.
func TestVectors(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "contracts", "crypto", "test-vectors", "vectors.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Pae struct {
			Hex string `json:"pae_hex"`
		} `json:"dsse_pae"`
		G struct {
			Priv     string `json:"private_key_le_hex"`
			Pub      string `json:"public_key_le_hex"`
			Sig      string `json:"signature_hex"`
			Tampered string `json:"tampered_pae_hex"`
		} `json:"gost3410_2012_256"`
		M struct {
			Seed string `json:"seed_hex"`
			Pub  string `json:"public_key_b64"`
			Sig  string `json:"signature_b64"`
		} `json:"mldsa65"`
		Hybrid json.RawMessage `json:"dsse_envelope_hybrid"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	pae, _ := hex.DecodeString(v.Pae.Hex)
	priv, _ := hex.DecodeString(v.G.Priv)
	g, err := FromSecret("person-welder-o17@1", dom.ProfileGost, priv)
	if err != nil || hex.EncodeToString(g.Public()) != v.G.Pub {
		t.Fatalf("открытый ключ ГОСТ: %v", err)
	}
	sig, _ := hex.DecodeString(v.G.Sig)
	pt := "application/vnd.ant.event+json; v=1"
	if !Verify(dom.ProfileGost, g.Public(), pt, pae, sig) {
		t.Fatal("подпись ГОСТ вектора")
	}
	tp, _ := hex.DecodeString(v.G.Tampered)
	if Verify(dom.ProfileGost, g.Public(), pt, tp, sig) {
		t.Fatal("изменённый пакет прошёл")
	}
	seed, _ := hex.DecodeString(v.M.Seed)
	m, err := FromSecret("person-welder-o17-pq@1", dom.ProfilePQ, seed)
	if err != nil || m.PublicB64() != v.M.Pub {
		t.Fatalf("открытый ключ ML-DSA: %v", err)
	}
	msig, _ := base64.StdEncoding.DecodeString(v.M.Sig)
	if !Verify(dom.ProfilePQ, m.Public(), pt, pae, msig) || Verify(dom.ProfilePQ, m.Public(), "application/vnd.ant.key-act+json; v=1", pae, msig) {
		t.Fatal("ML-DSA: контекст ant/‹payloadType›")
	}
	ver := Verifier{Keys: pubs{g.Ref: {dom.ProfileGost, g.Public()}, m.Ref: {dom.ProfilePQ, m.Public()}}}
	_, _, checks, err := ver.Check(context.Background(), v.Hybrid)
	if err != nil || len(checks) != 2 || checks[0].Result != dom.CryptoOK || checks[1].Result != dom.CryptoOK {
		t.Fatalf("гибрид вектора: %+v %v", checks, err)
	}
}

// Подпись и проверка своим набором ключей; изменённый пакет и недоступный ключ.
func TestSignVerify(t *testing.T) {
	dir := t.TempDir()
	kr, _ := LoadDir(dir)
	g, err := kr.Ensure(dir, "gateway-ingest@1", dom.ProfileGost)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := kr.Ensure(dir, "gateway-ingest-pq@1", dom.ProfilePQ)
	again, _ := LoadDir(dir)
	if k, ok := again.Key("gateway-ingest@1"); !ok || k.Fingerprint() != g.Fingerprint() {
		t.Fatal("ключ из тома")
	}
	s := Signer{Keys: again}
	payload := []byte(`{"a":1}`)
	pt := dom.PayloadType(dom.ClassEvent, 1)
	raw, err := s.Sign(context.Background(), pt, payload, "gateway-ingest@1,gateway-ingest-pq@1")
	if err != nil {
		t.Fatal(err)
	}
	ver := Verifier{Keys: pubs{g.Ref: {dom.ProfileGost, g.Public()}, p.Ref: {dom.ProfilePQ, p.Public()}}}
	res, err := ver.Verify(context.Background(), raw)
	if err != nil || len(res.Signers) != 2 || string(res.Payload) != `{"a":1}` {
		t.Fatalf("%+v %v", res, err)
	}
	var env dom.Envelope
	_ = json.Unmarshal(raw, &env)
	env.Payload = base64.StdEncoding.EncodeToString([]byte(`{"a":2}`))
	if _, err := ver.Verify(context.Background(), env.Marshal()); err == nil {
		t.Fatal("изменённый пакет прошёл")
	}
	only := Verifier{Keys: pubs{}}
	if _, err := only.Verify(context.Background(), raw); !errors.Is(err, ErrKeyUnavailable) {
		t.Fatalf("недоступный ключ: %v", err)
	}
}

// FromSeed (Д-82): одинаковый материал — одинаковый ключ; скаляр ГОСТ по
// модулю q, нулевой (0 и q) — ErrZeroScalar; ML-DSA — материал как зерно.
func TestFromSeed(t *testing.T) {
	m := make([]byte, 32)
	for i := range m {
		m[i] = byte(i + 1)
	}
	for _, p := range []string{dom.ProfileGost, dom.ProfilePQ} {
		a, err := FromSeed("x@1", p, m)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := FromSeed("x@1", p, m)
		if a.PublicB64() != b.PublicB64() || len(a.Secret()) != 32 {
			t.Fatalf("%s: ключ из одного материала различается", p)
		}
	}
	q := curve().Q.FillBytes(make([]byte, 32))
	for i, j := 0, len(q)-1; i < j; i, j = i+1, j-1 {
		q[i], q[j] = q[j], q[i]
	}
	for _, z := range [][]byte{make([]byte, 32), q} {
		if _, err := FromSeed("x@1", dom.ProfileGost, z); !errors.Is(err, ErrZeroScalar) {
			t.Fatalf("нулевой скаляр принят: %v", err)
		}
	}
	if _, err := FromSeed("x@1", dom.ProfilePQ, m[:16]); err == nil {
		t.Fatal("материал 16 байт принят")
	}
}
