package journal

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
)

// Сверка формулы звена и Стрибога-256 с тест-векторами эпика 00
// (contracts/crypto/test-vectors/vectors.v1.json, разделы streebog256 и
// chain_v1): одна реализация у Append, хранителя и верификатора (AD-44).
func TestChainVectorsV1(t *testing.T) {
	raw, err := os.ReadFile("../../../../../contracts/crypto/test-vectors/vectors.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Streebog []struct {
			MessageHex string `json:"message_hex"`
			DigestHex  string `json:"digest_hex"`
		} `json:"streebog256"`
		Chain struct {
			Envelope string `json:"envelope_canonical_utf8"`
			Entries  []struct {
				SaltHex  string `json:"salt_hex"`
				OpenJSON string `json:"open_fields_json"`
				OpenHash string `json:"open_fields_hash"`
				Commit   string `json:"commit"`
				PrevLink string `json:"prev_link"`
				Link     string `json:"link"`
			} `json:"entries"`
		} `json:"chain_v1"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.Streebog) == 0 || len(v.Chain.Entries) == 0 {
		t.Fatal("векторы пусты")
	}
	for _, s := range v.Streebog {
		m, _ := hex.DecodeString(s.MessageHex)
		if got := hex.EncodeToString(dj.H(m).Bytes()); got != s.DigestHex {
			t.Fatalf("Стрибог-256: %s, want %s", got, s.DigestHex)
		}
	}
	prev := dj.ZeroLink
	var chain []jc.JournalEntry
	for i, ve := range v.Chain.Entries {
		salt, _ := hex.DecodeString(ve.SaltHex)
		c, err := dj.Commit(salt, []byte(v.Chain.Envelope))
		if err != nil || c.String() != ve.Commit {
			t.Fatalf("запись %d: commit %s, want %s (%v)", i+1, c, ve.Commit, err)
		}
		if prev.String() != ve.PrevLink {
			t.Fatalf("запись %d: prev_link %s, want %s", i+1, prev, ve.PrevLink)
		}
		// Открытые поля проходят через сгенерированный тип — так их видят все.
		var e jc.JournalEntry
		if err := json.Unmarshal([]byte(ve.OpenJSON), &e); err != nil {
			t.Fatal(err)
		}
		oh, err := dj.OpenHash(e)
		if err != nil || oh.String() != ve.OpenHash {
			t.Fatalf("запись %d: open_hash %s, want %s (%v)", i+1, oh, ve.OpenHash, err)
		}
		link, open, err := dj.Seal(&e, prev, salt, []byte(v.Chain.Envelope))
		if err != nil || link.String() != ve.Link {
			t.Fatalf("запись %d: link %s, want %s (%v)", i+1, link, ve.Link, err)
		}
		if dj.H(open) != oh {
			t.Fatalf("запись %d: заголовок Seal не совпадает с открытыми полями", i+1)
		}
		chain = append(chain, e)
		prev = link
	}
	if _, err := dj.VerifyLinks(dj.ZeroLink, 0, chain); err != nil {
		t.Fatal(err)
	}
}
