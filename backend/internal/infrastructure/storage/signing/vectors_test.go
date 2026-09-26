package signing

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	. "ant/internal/domain/signing"
)

// Тест-векторы контракта для чистых функций домена signing (файлы читаются
// здесь, а не в домене: домен не делает ввода-вывода, AD-4).

// vectors — contracts/crypto/test-vectors/vectors.v1.json (общие с агентом
// токена и верификатором: «отпечаток сервера = отпечаток агента», AD-12).
func vectors(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "contracts", "crypto", "test-vectors", "vectors.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]json.RawMessage
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestVectorsPAEJCSMerkleQR(t *testing.T) {
	v := vectors(t)
	var pae struct {
		PaeHex      string `json:"pae_hex"`
		PayloadB64  string `json:"payload_b64"`
		PayloadType string `json:"payload_type"`
	}
	_ = json.Unmarshal(v["dsse_pae"], &pae)
	p, _ := base64.StdEncoding.DecodeString(pae.PayloadB64)
	if got := hex.EncodeToString(PAE(pae.PayloadType, p)); got != pae.PaeHex {
		t.Fatalf("PAE расходится с вектором")
	}
	if err := CheckCanonical(p); err != nil {
		t.Fatalf("payload вектора должен быть каноническим: %v", err)
	}
	var jcs []struct {
		Input     string `json:"input"`
		Canonical string `json:"canonical_utf8"`
	}
	_ = json.Unmarshal(v["jcs"], &jcs)
	for _, c := range jcs {
		got, err := Canonical([]byte(c.Input))
		if err != nil || string(got) != c.Canonical {
			t.Fatalf("JCS %s: %s", c.Input, got)
		}
		if CheckCanonical([]byte(c.Input)) == nil {
			t.Fatalf("неканоническое принято: %s", c.Input)
		}
	}
	var mk []struct {
		Leaves []string `json:"leaves_hex"`
		Root   string   `json:"root"`
	}
	_ = json.Unmarshal(v["merkle_rfc6962"], &mk)
	for _, c := range mk {
		var ls [][]byte
		for _, h := range c.Leaves {
			b, _ := hex.DecodeString(h)
			ls = append(ls, b)
		}
		if got := MerkleRootString(ls); got != c.Root {
			t.Fatalf("корень Меркла %d листьев: %s ≠ %s", len(ls), got, c.Root)
		}
	}
	var doc struct {
		QR     string `json:"qr"`
		Digest string `json:"doc_digest"`
	}
	_ = json.Unmarshal(v["doc_digest"], &doc)
	id, dg, err := ParseQR(doc.QR)
	if err != nil || id != "DOC-0001" || dg != doc.Digest || QRText(id, dg) != doc.QR {
		t.Fatalf("QR: %s %s %v", id, dg, err)
	}
	var sb []struct {
		Msg    string `json:"message_hex"`
		Digest string `json:"digest_hex"`
	}
	_ = json.Unmarshal(v["streebog256"], &sb)
	for _, c := range sb {
		m, _ := hex.DecodeString(c.Msg)
		if Digest(m) != HashPrefix+c.Digest {
			t.Fatalf("Стрибог %q", c.Msg)
		}
	}
}

// Перечень уровня 1 — копия контракта token-agent/level1-actions.yaml.
func TestLevel1MatchesContract(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "contracts", "internal", "token-agent", "level1-actions.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, l := range strings.Split(string(raw), "\n") {
		l = strings.TrimSpace(l)
		if s, ok := strings.CutPrefix(l, "- "); ok {
			got = append(got, strings.Fields(s)[0])
		}
	}
	slices.Sort(got)
	if !slices.Equal(got, Level1Actions()) || !slices.IsSorted(Level1Actions()) {
		t.Fatalf("перечень уровня 1 расходится с контрактом:\n%v\n%v", got, Level1Actions())
	}
}
