package documents_test

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	. "ant/internal/domain/documents" //nolint:revive // правила отпечатка домена под их именами
)

// «Отпечаток сервера = отпечаток агента» (AD-12): правила отпечатка
// (domain/documents) сверяются с эталоном contracts/crypto/test-vectors/vectors.v1.json (раздел
// doc_digest), который независимо собран генератором тест-векторов на
// GoGOST; агент токена и demo-signer проверяют себя по тому же эталону.
func TestDocDigestVector(t *testing.T) {
	b, err := os.ReadFile("../../../../contracts/crypto/test-vectors/vectors.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		DocDigest struct {
			ContentJSON      string `json:"content_json"`
			DigestInput      string `json:"digest_input_canonical_utf8"`
			DocDigest        string `json:"doc_digest"`
			DocFormatVersion string `json:"doc_format_version"`
			QR               string `json:"qr"`
			RenderingHash    string `json:"rendering_hash"`
			Rendering        string `json:"rendering_utf8"`
			TemplateRef      string `json:"template_ref"`
		} `json:"doc_digest"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	x := v.DocDigest
	if got := RenderingHash(x.Rendering); got != x.RenderingHash {
		t.Fatalf("rendering_hash: %s, эталон %s", got, x.RenderingHash)
	}
	fv, _ := strconv.Atoi(x.DocFormatVersion)
	in, err := DigestInput(json.RawMessage(x.ContentJSON), x.RenderingHash, x.TemplateRef, fv)
	if err != nil || string(in) != x.DigestInput {
		t.Fatalf("вход отпечатка:\n%s\nэталон:\n%s (%v)", in, x.DigestInput, err)
	}
	d, err := DocDigest(json.RawMessage(x.ContentJSON), x.RenderingHash, x.TemplateRef, fv)
	if err != nil || d != x.DocDigest {
		t.Fatalf("doc_digest: %s, эталон %s (%v)", d, x.DocDigest, err)
	}
	if QR("DOC-0001", d) != x.QR {
		t.Fatalf("QR: %s", QR("DOC-0001", d))
	}
	if id, dg, ok := ParseQR(x.QR); !ok || id != "DOC-0001" || dg != d {
		t.Fatalf("разбор QR: %s %s %v", id, dg, ok)
	}
	if _, err := DocDigest(json.RawMessage(x.ContentJSON), x.RenderingHash, x.TemplateRef, 2); err == nil {
		t.Fatal("неизвестная версия формата принята")
	}
}

// Канонический JSON не экранирует HTML-символы и сортирует ключи (RFC 8785).
func TestCanonical(t *testing.T) {
	got, err := Canonical(map[string]any{"b": "<a&b>", "a": 1})
	if err != nil || string(got) != `{"a":1,"b":"<a&b>"}` {
		t.Fatalf("%s %v", got, err)
	}
}
