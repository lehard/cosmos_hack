package signing

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
)

// block — блок генезиса из трёх записей с «проверенными» подписями якоря.
func block(t *testing.T) []GenesisEntry {
	t.Helper()
	anchor := []AnchorKey{{KeyRef: AnchorRefs[0], ProfileID: ProfileGost, PublicB64: base64.StdEncoding.EncodeToString([]byte("g"))},
		{KeyRef: AnchorRefs[1], ProfileID: ProfilePQ, PublicB64: base64.StdEncoding.EncodeToString([]byte("p"))}}
	fp := AnchorFingerprint(anchor)
	ev := func(t string, data any, quorum ...string) []byte {
		b, _ := CanonicalOf(map[string]any{"event_type": t, "data": data,
			"integrity": map[string]any{"format_version": 1, "crypto_profile": ProfileHybrid, "signers": append(append([]string{}, AnchorRefs...), quorum...)}})
		return b
	}
	body := [][]byte{ev("time.clock.mode_set", map[string]any{"mode": "system"}, "tec-01@1"), ev(TypeAnchorDestroyed, map[string]any{"anchor_fingerprint": fp})}
	h := GenesisHeader{EnterpriseCode: "ENT01", ChainFormatVersion: 1, AnchorFingerprint: fp, BlockSize: 3, AnchorKeys: anchor, BlockDigest: BlockDigest(body)}
	all := append([][]byte{ev(TypeGenesis, h)}, body...)
	types := []string{TypeGenesis, "time.clock.mode_set", TypeAnchorDestroyed}
	var out []GenesisEntry
	for i, p := range all {
		e := GenesisEntry{Seq: int64(i + 1), EventType: types[i], Provenance: ProvGenesis, PayloadType: PayloadType(ClassGenesis, 1), Payload: p,
			Crypto: []CryptoCheck{{KeyRef: AnchorRefs[0], Result: CryptoOK}, {KeyRef: AnchorRefs[1], Result: CryptoOK}}}
		if i == 1 {
			e.Crypto = append(e.Crypto, CryptoCheck{KeyRef: "tec-01@1", Result: CryptoOK})
		}
		out = append(out, e)
	}
	return out
}

// AD-33: блок принимается целиком по якорю; любое отступление — «отвергнуто».
func TestCheckGenesisBlock(t *testing.T) {
	b := block(t)
	h, err := CheckGenesisBlock(GenesisCheck{Block: b, GenesisCount: 1})
	if err != nil || h.EnterpriseCode != "ENT01" {
		t.Fatalf("целый блок: %v", err)
	}
	if GenesisDigest(b[0].Payload) == "" || !ValidKeyRef(AnchorRefs[1]) {
		t.Fatal("отпечаток генезиса")
	}
	cases := []struct {
		name string
		f    func(b []GenesisEntry) GenesisCheck
	}{
		{"закреплён другой якорь", func(b []GenesisEntry) GenesisCheck {
			return GenesisCheck{Block: b, Pinned: Digest([]byte("x")), GenesisCount: 1}
		}},
		{"изменено содержимое", func(b []GenesisEntry) GenesisCheck {
			var m map[string]any
			_ = json.Unmarshal(b[1].Payload, &m)
			m["data"] = map[string]any{"mode": "scenario"}
			b[1].Payload, _ = CanonicalOf(m)
			return GenesisCheck{Block: b, GenesisCount: 1}
		}},
		{"подпись не сходится", func(b []GenesisEntry) GenesisCheck {
			b[2].Crypto[0].Result = CryptoBad
			return GenesisCheck{Block: b, GenesisCount: 1}
		}},
		{"понижение hybrid", func(b []GenesisEntry) GenesisCheck {
			b[1].Crypto = b[1].Crypto[:1]
			return GenesisCheck{Block: b, GenesisCount: 1}
		}},
		{"нет подписи кворума", func(b []GenesisEntry) GenesisCheck {
			b[1].Crypto = b[1].Crypto[:2]
			return GenesisCheck{Block: b, GenesisCount: 1}
		}},
		{"не с seq 1", func(b []GenesisEntry) GenesisCheck {
			for i := range b {
				b[i].Seq += 5
			}
			return GenesisCheck{Block: b, GenesisCount: 1}
		}},
		{"не класса genesis", func(b []GenesisEntry) GenesisCheck {
			b[1].Provenance = ProvServerAttested
			return GenesisCheck{Block: b, GenesisCount: 1}
		}},
		{"не пакет genesis", func(b []GenesisEntry) GenesisCheck {
			b[1].PayloadType = PayloadType(ClassEvent, 1)
			return GenesisCheck{Block: b, GenesisCount: 1}
		}},
		{"без «якорь уничтожен»", func(b []GenesisEntry) GenesisCheck {
			return GenesisCheck{Block: b[:2], GenesisCount: 1}
		}},
	}
	for _, c := range cases {
		if _, err := CheckGenesisBlock(c.f(block(t))); !errors.Is(err, ErrGenesis) {
			t.Errorf("%s: принято (%v)", c.name, err)
		}
	}
	if _, err := CheckGenesisBlock(GenesisCheck{Block: block(t), GenesisCount: 2}); !errors.Is(err, ErrSecondGenesis) {
		t.Errorf("второй генезис: %v", err)
	}
}
