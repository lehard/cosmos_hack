// Генератор тест-векторов contracts/crypto/test-vectors (эпик 00, AD-10, AD-12, AD-44).
// Запуск (Go на хосте не нужен):
//   cd contracts/crypto/test-vectors/generator && docker run --rm -v "$PWD":/w -w /w -e GOFLAGS=-mod=mod golang:1.27.1 go run . > ../vectors.v1.json
// Библиотека ГОСТ: github.com/deckhouse/gogost/v6 (код GoGOST; в продукте — third_party/gogost 7.0.0).
package main

import (
	"bytes"
	"crypto/mldsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"os"
	"strconv"

	"github.com/deckhouse/gogost/v6/gost3410"
	"github.com/deckhouse/gogost/v6/gost34112012256"
)

func H(b []byte) []byte { h := gost34112012256.New(); h.Write(b); return h.Sum(nil) }
func HX(b []byte) string { return "streebog256:" + hex.EncodeToString(H(b)) }
func jcs(s string) []byte {
	v := jsontext.Value([]byte(s))
	if err := v.Canonicalize(); err != nil { panic(err) }
	return []byte(v)
}
func pae(t string, body []byte) []byte {
	return []byte("DSSEv1 " + strconv.Itoa(len(t)) + " " + t + " " + strconv.Itoa(len(body)) + " " + string(body))
}
func must(b []byte, err error) []byte { if err != nil { panic(err) }; return b }
func unhex(s string) []byte { b, err := hex.DecodeString(s); if err != nil { panic(err) }; return b }

type fixedReader struct{ b []byte }
func (r *fixedReader) Read(p []byte) (int, error) { n := copy(p, r.b); return n, nil }

func uuid5(ns string, name []byte) string {
	nsb := unhex(ns[0:8] + ns[9:13] + ns[14:18] + ns[19:23] + ns[24:])
	h := sha1.New(); h.Write(nsb); h.Write(name); s := h.Sum(nil)[:16]
	s[6] = (s[6] & 0x0f) | 0x50; s[8] = (s[8] & 0x3f) | 0x80
	x := hex.EncodeToString(s)
	return x[0:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:]
}

func merkle(leaves [][]byte) []byte {
	if len(leaves) == 1 { return H(append([]byte{0x00}, leaves[0]...)) }
	k := 1
	for k*2 < len(leaves) { k *= 2 }
	l := merkle(leaves[:k]); r := merkle(leaves[k:])
	return H(append(append([]byte{0x01}, l...), r...))
}

const NS_ANT = "4b82fbf1-fc9e-5a06-91c6-8c100ebac4ef"
const PT = "application/vnd.ant.event+json; v=1"

func main() {
	out := map[string]any{}
	out["generator"] = "contracts/crypto/test-vectors/generator (go1.27.1, github.com/deckhouse/gogost/v6 v6.2.0, crypto/mldsa)"

	// 1. Стрибог-256: пример 1 ГОСТ Р 34.11-2012 (M1) — порядок байтов как у hash.Sum().
	m1 := []byte("012345678901234567890123456789012345678901234567890123456789012")
	d1 := hex.EncodeToString(H(m1))
	if d1 != "9d151eefd8590b89daa6ba6cb74af9275dd051026bb149a452fd84e5e57b5500" { panic("streebog M1 " + d1) }
	out["streebog256"] = []map[string]string{
		{"name": "ГОСТ Р 34.11-2012, пример 1 (M1)", "message_hex": hex.EncodeToString(m1), "digest_hex": d1,
			"note": "порядок байтов hash.Sum(): сообщение — байты ASCII по порядку, дайджест — как в распространённой записи; в нотации ГОСТ (вектор старшим разрядом вперёд) оба записаны в обратном порядке: M1 = 0x3231…3130, H = 0x00557be5…159d"},
		{"name": "пустое сообщение", "message_hex": "", "digest_hex": hex.EncodeToString(H(nil))},
		{"name": "строка «ant» (UTF-8)", "message_hex": hex.EncodeToString([]byte("ant")), "digest_hex": hex.EncodeToString(H([]byte("ant")))},
	}

	// 2. JCS (RFC 8785) — подмножество контракта: целые, строки NFC, объекты, массивы.
	jcsIn := []string{
		`{"b": 2, "a": 1, "c": {"z": true, "y": null}}`,
		`{"event_type":"operation.run.started","data":{"operator_id":null,"values":[3,-1,9007199254740991]},"note":"Сварка фланца \/ шов"}`,
		`{"é":1,"e":2,"€":3,"1":4}`,
	}
	var jv []map[string]string
	for _, s := range jcsIn { jv = append(jv, map[string]string{"input": s, "canonical_utf8": string(jcs(s))}) }
	out["jcs"] = jv

	// Пример события (конверт v1) — им же проверяются схемы в check.sh.
	event := `{
  "event_id": "01929a2b-7c3d-7e4f-8a5b-6c7d8e9f0a1b",
  "event_type": "operation.run.started",
  "schema_version": 1,
  "source_id": "terminal-weld-2",
  "source_seq": 42,
  "source_kind": "manual_entry",
  "reliability": "high",
  "occurred_at": "2026-09-25T10:15:30.123Z",
  "correlation_id": "01929a2b-7c3d-7e4f-8a5b-000000000001",
  "causation_id": null,
  "item_id": "ENT01:FL-0007",
  "integrity": {"format_version": 1, "crypto_profile": "gost", "signers": ["person-welder-o17@1"]},
  "data": {
    "operation_run_id": "run-W2-FL-0007-1",
    "operation_code": "030",
    "step_key": "welding.weld",
    "station_id": "weld-2",
    "equipment_id": "IS-1",
    "operator_id": "O17"
  }
}`
	evc := jcs(event)
	out["sample_event"] = map[string]any{"json": json.RawMessage(event), "canonical_utf8": string(evc), "payload_type": PT}

	// 3. DSSE PAE.
	p := pae(PT, evc)
	out["dsse_pae"] = map[string]string{"payload_type": PT, "payload_b64": base64.StdEncoding.EncodeToString(evc), "pae_hex": hex.EncodeToString(p)}

	// 4. ГОСТ Р 34.10-2012, 256 бит, id-tc26-gost-3410-12-256-paramSetA.
	c := gost3410.CurveIdtc26gost341012256paramSetA()
	prvLE := unhex("7a929ade789bb9be10ed359dd39a72c11b60961f49397eee1d19ce9891ec3b28")
	prv := must2(gost3410.NewPrivateKeyLE(c, prvLE))
	pub := must2(prv.PublicKey())
	kBE := unhex("77105c9b20bcd3122823c8cf6fcc7b956de33814e95b7fe64fed924594dceab3")
	dg := H(p)
	sig := must((&gost3410.PrivateKeyReverseDigest{Prv: prv}).Sign(&fixedReader{kBE}, dg, nil))
	ok, err := (gost3410.PublicKeyReverseDigest{Pub: pub}).VerifyDigest(dg, sig)
	if err != nil || !ok { panic("gost verify") }
	bad := append([]byte{}, p...); bad[len(bad)-2] ^= 0x01
	okBad, _ := (gost3410.PublicKeyReverseDigest{Pub: pub}).VerifyDigest(H(bad), sig)
	if okBad { panic("gost tampered verified") }
	out["gost3410_2012_256"] = map[string]any{
		"curve": "id-tc26-gost-3410-12-256-paramSetA",
		"private_key_le_hex": hex.EncodeToString(prvLE),
		"public_key_le_hex": hex.EncodeToString(pub.RawLE()),
		"public_key_b64": base64.StdEncoding.EncodeToString(pub.RawLE()),
		"message": "dsse_pae.pae_hex",
		"digest_hex": hex.EncodeToString(dg),
		"k_be_hex": hex.EncodeToString(kBE),
		"signature_hex": hex.EncodeToString(sig),
		"signature_b64": base64.StdEncoding.EncodeToString(sig),
		"verify": ok,
		"tampered_pae_hex": hex.EncodeToString(bad),
		"tampered_verify": okBad,
	}

	// 5. ML-DSA-65 (профиль pq, демонстрационный): контекст "ant/" + payloadType.
	seed := make([]byte, 32); for i := range seed { seed[i] = byte(i) }
	sk := must2(mldsa.NewPrivateKey(mldsa.MLDSA65(), seed))
	ctx := "ant/" + PT
	msig := must(sk.SignDeterministic(p, &mldsa.Options{Context: ctx}))
	if err := mldsa.Verify(sk.PublicKey(), p, msig, &mldsa.Options{Context: ctx}); err != nil { panic(err) }
	errWrongCtx := mldsa.Verify(sk.PublicKey(), p, msig, &mldsa.Options{Context: "ant/other"})
	out["mldsa65"] = map[string]any{
		"seed_hex": hex.EncodeToString(seed),
		"public_key_b64": base64.StdEncoding.EncodeToString(sk.PublicKey().Bytes()),
		"context": ctx,
		"message": "dsse_pae.pae_hex",
		"deterministic": true,
		"signature_b64": base64.StdEncoding.EncodeToString(msig),
		"verify": true,
		"verify_with_other_context": errWrongCtx == nil,
	}

	// 6. DSSE-конверт профиля hybrid: две подписи над одним PAE.
	out["dsse_envelope_hybrid"] = map[string]any{
		"payloadType": PT,
		"payload": base64.StdEncoding.EncodeToString(evc),
		"signatures": []map[string]string{
			{"keyid": "person-welder-o17@1", "sig": base64.StdEncoding.EncodeToString(sig)},
			{"keyid": "person-welder-o17-pq@1", "sig": base64.StdEncoding.EncodeToString(msig)},
		},
	}

	// 7. Формат цепочки v1: commit = H(salt ‖ JCS(конверт DSSE)); link = H(prev_link ‖ commit ‖ H(JCS(открытые поля без link))).
	envJSON := must(json.Marshal(out["dsse_envelope_hybrid"]))
	envC := jcs(string(envJSON))
	salt := unhex("000102030405060708090a0b0c0d0e0f")
	commit := H(append(append([]byte{}, salt...), envC...))
	open1 := fmt.Sprintf(`{"seq":1,"chain":"main","entry_kind":"fact","event_type":"operation.run.started","schema_version":1,"event_id":"01929a2b-7c3d-7e4f-8a5b-6c7d8e9f0a1b","source_id":"terminal-weld-2","source_seq":42,"item_id":"ENT01:FL-0007","stream":"item:ENT01:FL-0007","partition":3,"occurred_at":"2026-09-25T10:15:30.123Z","received_at":"2026-09-25T10:15:30.200Z","recorded_at":"2026-09-25T10:15:30.210Z","committed_at":"2026-09-25T10:15:30.210Z","correlation_id":"01929a2b-7c3d-7e4f-8a5b-000000000001","causation_id":null,"provenance_class":"personal","domain_build":"streebog256:%064x","commit":"streebog256:%s"}`, 0, hex.EncodeToString(commit))
	openHash := H(jcs(open1))
	prev := make([]byte, 32)
	link1 := H(bytes.Join([][]byte{prev, commit, openHash}, nil))
	salt2 := unhex("f0e0d0c0b0a090807060504030201000")
	commit2 := H(append(append([]byte{}, salt2...), envC...))
	open2 := fmt.Sprintf(`{"seq":2,"chain":"main","entry_kind":"fact","event_type":"operation.run.started","schema_version":1,"event_id":"01929a2b-7c3d-7e4f-8a5b-6c7d8e9f0a1c","source_id":"terminal-weld-2","source_seq":43,"stream":"item:ENT01:FL-0007","partition":3,"occurred_at":"2026-09-25T10:16:00.000Z","received_at":"2026-09-25T10:16:00.050Z","recorded_at":"2026-09-25T10:16:00.060Z","committed_at":"2026-09-25T10:16:00.060Z","correlation_id":"01929a2b-7c3d-7e4f-8a5b-000000000001","causation_id":null,"provenance_class":"personal","domain_build":"streebog256:%064x","commit":"streebog256:%s"}`, 0, hex.EncodeToString(commit2))
	openHash2 := H(jcs(open2))
	link2 := H(bytes.Join([][]byte{link1, commit2, openHash2}, nil))
	out["chain_v1"] = map[string]any{
		"note": "prev_link первой записи — 32 нулевых байта; все хеши — сырые 32 байта hash.Sum() при склейке; строки — с префиксом streebog256:",
		"envelope_canonical_utf8": string(envC),
		"entries": []map[string]string{
			{"salt_hex": hex.EncodeToString(salt), "open_fields_json": open1, "commit": "streebog256:" + hex.EncodeToString(commit), "open_fields_hash": "streebog256:" + hex.EncodeToString(openHash), "prev_link": "streebog256:" + hex.EncodeToString(prev), "link": "streebog256:" + hex.EncodeToString(link1)},
			{"salt_hex": hex.EncodeToString(salt2), "open_fields_json": open2, "commit": "streebog256:" + hex.EncodeToString(commit2), "open_fields_hash": "streebog256:" + hex.EncodeToString(openHash2), "prev_link": "streebog256:" + hex.EncodeToString(link1), "link": "streebog256:" + hex.EncodeToString(link2)},
		},
	}

	// 8. Отпечаток документа (AD-12): rendering_hash = H(отрисовка); doc_digest = H(JCS({content, rendering_hash, template_ref, doc_format_version})).
	content := `{"item_id":"ENT01:FL-0007","decision":"accept","closing_point":"ZT-3","signers":[{"stage":1,"authority_id":"qc_acceptance"}]}`
	rendering := []byte("<article><h1>Журнал предъявления ОТК</h1><p>ФЛ-0007 — ЗТ-3 — принять</p></article>")
	rh := HX(rendering)
	docObj := fmt.Sprintf(`{"content":%s,"rendering_hash":"%s","template_ref":"presentation-log@1","doc_format_version":1}`, content, rh)
	out["doc_digest"] = map[string]string{
		"content_json": content, "rendering_utf8": string(rendering), "rendering_hash": rh,
		"template_ref": "presentation-log@1", "doc_format_version": "1",
		"digest_input_canonical_utf8": string(jcs(docObj)), "doc_digest": HX(jcs(docObj)),
		"qr": "ant:doc:DOC-0001:" + HX(jcs(docObj)),
	}

	// 9. Дерево Меркла RFC 6962 для сменного рапорта (уровень 3): лист H(0x00‖d), узел H(0x01‖l‖r).
	var leaves [][]byte
	for i := 0; i < 5; i++ { leaves = append(leaves, H([]byte(fmt.Sprintf("signature-%d", i)))) }
	var mr []map[string]any
	for _, n := range []int{1, 2, 3, 5} {
		var lh []string
		for _, l := range leaves[:n] { lh = append(lh, hex.EncodeToString(l)) }
		mr = append(mr, map[string]any{"leaves_hex": lh, "root": "streebog256:" + hex.EncodeToString(merkle(leaves[:n]))})
	}
	out["merkle_rfc6962"] = mr

	// 10. UUIDv5 от NS_ANT (RFC 9562 §5.5): reaction_id = UUIDv5(NS_ANT, слот ‖ версия).
	slot := "rule:R-07|item:ENT01:FL-0007|signal:sig-1|1"
	out["uuid5"] = []map[string]string{
		{"namespace": NS_ANT, "name_utf8": slot, "uuid": uuid5(NS_ANT, []byte(slot))},
		{"namespace": "6ba7b811-9dad-11d1-80b4-00c04fd430c8", "name_utf8": "urn:ant:ns:1", "uuid": uuid5("6ba7b811-9dad-11d1-80b4-00c04fd430c8", []byte("urn:ant:ns:1")), "note": "так получено NS_ANT"},
	}

	enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil { panic(err) }
}

func must2[T any](v T, err error) T { if err != nil { panic(err) }; return v }
