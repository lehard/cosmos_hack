package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ant/cmd/token-agent/internal/agent"
	"ant/internal/contracts/procs"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// fakePrompt — окно агента в тесте: запоминает показанное.
type fakePrompt struct {
	ok     bool
	pin    string
	shown  []agent.Field
	called int
}

func (f *fakePrompt) Confirm(_ string, fields []agent.Field, _ bool) (string, bool, error) {
	f.called++
	f.shown = fields
	return f.pin, f.ok, nil
}

func frame(v any) []byte {
	var b bytes.Buffer
	_ = writeMsg(&b, v)
	return b.Bytes()
}

func setup(t *testing.T) (*session, *fakePrompt) {
	t.Helper()
	k, _ := profiles.FromSecret("ins-01-ta@1", dom.ProfileGost, bytes.Repeat([]byte{7}, 32))
	f, _ := json.Marshal(agent.KeyFile{KeyRef: k.Ref, Profile: k.Profile, SecretHex: hex.EncodeToString(k.Secret())})
	b, err := agent.NewBundle([][]byte{f}, "")
	if err != nil {
		t.Fatal(err)
	}
	kdf := agent.KDF{Name: "argon2id", Time: 1, MemoryKiB: 64, Threads: 1}
	s, err := agent.Seal(b, "1234", agent.SealOptions{KeyStorage: dom.StorageHardwareToken, KDF: &kdf}, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfg := Config{TokenDir: filepath.Join(dir, "token"), StateDir: filepath.Join(dir, "state"), WorkplaceID: "WP-QC-1"}
	_ = os.MkdirAll(cfg.TokenDir, 0o700)
	raw, _ := json.Marshal(s)
	if err := os.WriteFile(cfg.sealedPath(), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	fp := &fakePrompt{ok: true, pin: "1234"}
	return &session{cfg: cfg, prompt: fp, now: func() time.Time { return time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC) }}, fp
}

func request(typ procs.TokenAgentRequestV1Type, blk *procs.SignBlock) []byte {
	b, _ := json.Marshal(procs.TokenAgentRequestV1{ProtocolVersion: 1, Type: typ, RequestID: "0192f1a0-0000-7000-8000-00000000000a",
		Origin: "http://127.0.0.1:8480", Sign: blk})
	return b
}

func block(level int, eventType string) *procs.SignBlock {
	body := `{"basis_seq":41,"command_id":"0192f1a0-0000-7000-8000-000000000001","policy_seq":3,"reason":{"text":"Годен"}}`
	item := "ITEM-017"
	return &procs.SignBlock{Level: level, PayloadType: dom.PayloadType(dom.ClassEvent, 1), PayloadB64: base64.StdEncoding.EncodeToString([]byte(body)),
		EventType: &eventType, CommandRequest: &procs.SignBlockCommandRequest{Operation: "quality.inspection.record", ItemID: &item}}
}

// Протокол агента: привет и состояние; решение уровня 2 — окно агента и PIN;
// «годен» уровнем 1 — отказ без окна; отказ в окне; извлечение токена
// сбрасывает PIN.
func TestNativeHost(t *testing.T) {
	s, fp := setup(t)
	r := s.handle(request(procs.TokenAgentRequestV1TypeHello, nil), "chrome-extension://x/")
	if r.Hello == nil || len(r.Hello.Keys) != 1 || r.Hello.Keys[0].PersonID != "INS-01" {
		t.Fatalf("hello: %+v", r)
	}
	r = s.handle(request(procs.TokenAgentRequestV1TypeStatus, nil), "")
	if r.Status == nil || !r.Status.TokenPresent || r.Status.PinUnlocked || *r.Status.KeyStorage != "hardware_token" {
		t.Fatalf("status: %+v", r.Status)
	}
	r = s.handle(request(procs.TokenAgentRequestV1TypeSign, block(1, "inspection.result.recorded")), "")
	if r.Type != procs.TokenAgentResponseV1TypeError || r.Error.Code != agent.CodeLevel || fp.called != 0 {
		t.Fatalf("«годен» уровнем 1: %+v", r)
	}
	r = s.handle(request(procs.TokenAgentRequestV1TypeSign, block(2, "inspection.result.recorded")), "")
	if r.Type != procs.TokenAgentResponseV1TypeSigned || len(r.Signed) != 1 || fp.called != 1 || len(fp.shown) < 3 {
		t.Fatalf("уровень 2: %+v", r)
	}
	if r.Signed[0].LocalJournalSeq != 1 || *r.Signed[0].KeyStorage != "hardware_token" {
		t.Fatalf("журнал и класс: %+v", r.Signed[0])
	}
	r = s.handle(request(procs.TokenAgentRequestV1TypeStatus, nil), "")
	if !r.Status.PinUnlocked {
		t.Fatal("PIN не запомнен на сеанс")
	}
	fp.ok = false
	r = s.handle(request(procs.TokenAgentRequestV1TypeSign, block(2, "inspection.result.recorded")), "")
	if r.Error == nil || r.Error.Code != agent.CodeCancelled {
		t.Fatalf("отказ в окне: %+v", r)
	}
	_ = os.Remove(s.cfg.sealedPath())
	r = s.handle(request(procs.TokenAgentRequestV1TypeStatus, nil), "")
	if r.Status.TokenPresent || r.Status.PinUnlocked || s.dk != nil {
		t.Fatal("извлечение токена не сбросило PIN")
	}
	// Кадр Native Messaging: длина + JSON.
	var out bytes.Buffer
	in := bytes.NewReader(frame(json.RawMessage(request(procs.TokenAgentRequestV1TypeHello, nil))))
	s2, _ := setup(t)
	if code := func() int {
		raw, err := readMsg(in)
		if err != nil {
			return 1
		}
		return map[bool]int{true: 0, false: 1}[writeMsg(&out, s2.handle(raw, "")) == nil]
	}(); code != 0 || out.Len() < 8 {
		t.Fatal("кадр Native Messaging")
	}
}
