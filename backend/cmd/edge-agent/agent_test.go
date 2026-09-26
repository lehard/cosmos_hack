package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	app "ant/internal/application/ingest"
	"ant/internal/application/ingest/inmem"
	"ant/internal/contracts/procs"
)

// Самопоказ как проверка: пять случаев FR-29, повтор, конфликт и
// «недоступность и восстановление» (FR-39) — без потерь и дублей.
func TestDemo(t *testing.T) {
	var out bytes.Buffer
	s, err := runDemo(context.Background(), &out, "../../../contracts/events/examples/contract-change", t.TempDir())
	t.Log("\n" + out.String())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"01-unknown-version.json":              "quarantined ingest.unknown_schema_version",
		"02-new-optional-field.json":           "accepted",
		"03-missing-required-field.json":       "quarantined ingest.missing_required_field",
		"04a-unknown-enum-value.json":          "accepted ingest.unknown_enum_value",
		"04b-unknown-enum-value-critical.json": "quarantined ingest.unknown_enum_value_critical",
		"05-incompatible-change.json":          "quarantined ingest.missing_required_field",
	}
	for f, w := range want {
		if s.Cases[f] != w {
			t.Fatalf("%s: %q, ждали %q", f, s.Cases[f], w)
		}
	}
	if !s.RepeatSeqSame || s.JournalBefore != s.JournalAfter || s.Duplicates != 3 {
		t.Fatalf("повтор: %+v", s)
	}
	if s.ConflictCA != 1 || s.ConflictSecEvt != 1 {
		t.Fatalf("конфликт: %+v", s)
	}
	if s.Buffered != 5 || s.Facts != 5 || s.DupAfterLoss != 3 || s.Completeness != 10000 {
		t.Fatalf("недоступность и восстановление: %+v", s)
	}
}

// Буфер и номер переживают перезапуск агента: неотправленное досылается,
// номера не повторяются.
func TestAgentRestartKeepsBufferAndSeq(t *testing.T) {
	dir := t.TempDir()
	cfg := app.DefaultConfig()
	core := inmem.NewCore(cfg, nil, nil)
	srv := httptest.NewServer(coreHandler(core.Service))
	defer srv.Close()
	down := Config{SourceID: "edge-cnc-1", CoreURL: "http://127.0.0.1:1", StateDir: dir, SourceKind: "machine", Reliability: "high"}
	a, err := NewAgent(down, Unsigned{Ref: "device-edge-cnc-1@1"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err := a.Enqueue(stateEvent("running")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.Flush(context.Background()); err == nil {
		t.Fatal("ядро недоступно — ждали ошибку")
	}
	// «Сбой»: номер не сохранён, буфер на диске.
	_ = os.Remove(filepath.Join(dir, "next_seq"))
	up := down
	up.CoreURL = srv.URL
	b, err := NewAgent(up, Unsigned{Ref: "device-edge-cnc-1@1"}, srv.Client(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if st := b.Status(); st.NextSeq != 4 || st.Buffered != 3 {
		t.Fatalf("после перезапуска: %+v", st)
	}
	e, _ := b.Enqueue(stateEvent("idle"))
	if e.SourceSeq != 4 {
		t.Fatalf("номер %d", e.SourceSeq)
	}
	if n, err := b.Drain(context.Background()); err != nil || n != 4 {
		t.Fatalf("досылка %d %v", n, err)
	}
	st, _ := core.Service.Stats(context.Background())
	if st.Accepted != 4 || st.Sources[0].CompletenessBP != 10000 {
		t.Fatalf("%+v", st)
	}
}

func stateEvent(exec string) map[string]any {
	return map[string]any{"event_type": "equipment.state.changed", "schema_version": 2,
		"data": map[string]any{"equipment_id": "CNC-1", "execution": exec, "controller_mode": "automatic", "condition": "normal"}}
}

// Кодирование подписи ключа устройства совпадает с тест-векторами контракта
// (contracts/crypto/test-vectors, раздел gost3410_2012_256).
func TestGostSignerMatchesVectors(t *testing.T) {
	raw, err := os.ReadFile("../../../contracts/crypto/test-vectors/vectors.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		G struct {
			Prv string `json:"private_key_le_hex"`
			Pub string `json:"public_key_le_hex"`
			Sig string `json:"signature_hex"`
		} `json:"gost3410_2012_256"`
		P struct {
			PAE string `json:"pae_hex"`
		} `json:"dsse_pae"`
		E struct {
			Canonical   string `json:"canonical_utf8"`
			PayloadType string `json:"payload_type"`
		} `json:"sample_event"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	pae, _ := hex.DecodeString(v.P.PAE)
	if got := PAE(v.E.PayloadType, []byte(v.E.Canonical)); !bytes.Equal(got, pae) {
		t.Fatal("PAE не совпадает с вектором")
	}
	pub, _ := hex.DecodeString(v.G.Pub)
	sig, _ := hex.DecodeString(v.G.Sig)
	if ok, err := VerifyGost(pub, pae, sig); err != nil || !ok {
		t.Fatalf("подпись вектора не проверилась нашим кодированием: %v", err)
	}
	kf := filepath.Join(t.TempDir(), "k")
	if err := os.WriteFile(kf, []byte(v.G.Prv), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGostKey(kf, "device-edge-weld-1@1")
	if err != nil {
		t.Fatal(err)
	}
	env, err := g.Sign([]byte(v.E.Canonical))
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Payload    string `json:"payload"`
		Signatures []struct {
			KeyID, Sig string
		} `json:"signatures"`
	}
	_ = json.Unmarshal(env, &d)
	s, _ := base64.StdEncoding.DecodeString(d.Signatures[0].Sig)
	if ok, _ := VerifyGost(pub, pae, s); !ok || d.Signatures[0].KeyID != "device-edge-weld-1@1" {
		t.Fatal("наша подпись не проверяется открытым ключом вектора")
	}
	pk, _ := g.prv.PublicKey()
	if hex.EncodeToString(pk.RawLE()) != v.G.Pub {
		t.Fatal("открытый ключ не совпадает")
	}
}

// Выделитель: смена режима stand-а → equipment.state.changed v2 по контракту.
func TestExtractorAndLocalHandler(t *testing.T) {
	cfg := app.DefaultConfig()
	core := inmem.NewCore(cfg, nil, nil)
	srv := httptest.NewServer(coreHandler(core.Service))
	defer srv.Close()
	a, _ := NewAgent(Config{SourceID: "edge-weld-3", CoreURL: srv.URL, StateDir: t.TempDir()}, Unsigned{Ref: "device-edge-weld-3@1"}, srv.Client(), nil, nil)
	h := LocalHandler(a, NewExtractor())
	tel := procs.StandTelemetryV1{StandID: "stand-weld", EquipmentID: "IS-1", Seq: 1, SentAt: "2026-09-25T10:00:00.000Z",
		Events: []procs.StandTelemetryV1EventsElem{
			{Category: "execution", Value: "ACTIVE", At: "2026-09-25T10:00:00.000Z"},
			{Category: "execution", Value: "ACTIVE", At: "2026-09-25T10:00:01.000Z"},
			{Category: "condition", Value: "WARNING", At: "2026-09-25T10:00:02.000Z"},
		}}
	b, _ := json.Marshal(tel)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(b)))
	if rec.Code != http.StatusAccepted || strings.Count(rec.Body.String(), "source_seq") != 2 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if n, err := a.Drain(context.Background()); err != nil || n != 2 {
		t.Fatalf("%d %v", n, err)
	}
	st, _ := core.Service.Stats(context.Background())
	if st.Accepted != 2 || st.Quarantined != 0 {
		t.Fatalf("%+v", st)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/telemetry", strings.NewReader(`{"stand_id":"x","equipment_id":"IS-1","seq":1,"sent_at":"2026-09-25T10:00:00.000Z","extra":1}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("закрытая схема телеметрии: %d", rec.Code)
	}
}
