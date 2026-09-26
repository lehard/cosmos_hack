package ingest_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/application/ingest/inmem"
	"ant/internal/application/platform"
	"ant/internal/application/signing"
	"ant/internal/contracts/crypto"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/ingest"
)

const repo = "../../../.."

var t0 = time.Date(2026, 9, 25, 10, 16, 0, 0, time.UTC)

func demoCore(t *testing.T) *inmem.Core {
	t.Helper()
	cfg := app.DefaultConfig()
	cfg.Profile = "demo"
	return inmem.NewCore(cfg, inmem.NewClock(t0), nil)
}

// example — пример контракта contracts/events/examples/contract-change.
type example struct {
	Case    string          `json:"case"`
	Expect  map[string]any  `json:"expect"`
	Message json.RawMessage `json:"message"`
}

func examples(t *testing.T) map[string]example {
	t.Helper()
	dir := filepath.Join(repo, "contracts/events/examples/contract-change")
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("примеры: %v", err)
	}
	out := map[string]example{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var ex example
		if err := json.Unmarshal(b, &ex); err != nil {
			t.Fatal(err)
		}
		out[filepath.Base(f)] = ex
	}
	return out
}

// FR-29, кейс §4.7: для каждого из пяти случаев — пример контракта и видимое
// поведение приёма (итог, код, поле, записи журнала и карантин).
func TestFiveContractCases(t *testing.T) {
	ctx := context.Background()
	for name, ex := range examples(t) {
		t.Run(name, func(t *testing.T) {
			c := demoCore(t)
			r, err := c.Service.Ingest(ctx, ex.Message)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s → %s %s %s %s", ex.Case, r.Outcome, r.Code, r.Field, r.Detail)
			want := ex.Expect["outcome"].(string)
			if string(r.Outcome) != want {
				t.Fatalf("итог %s, ждали %s (%s)", r.Outcome, want, r.Detail)
			}
			if code, ok := ex.Expect["code"].(string); ok && string(r.Code) != code {
				t.Fatalf("код %s, ждали %s", r.Code, code)
			}
			if f, ok := ex.Expect["field"].(string); ok && r.Field != f {
				t.Fatalf("поле %s, ждали %s", r.Field, f)
			}
			switch want {
			case "quarantined":
				if c.Journal.Count("ingest.message.quarantined") != 1 || len(c.Journal.Main()) != 1 {
					t.Fatalf("в журнале должен быть только факт карантина: %d записей", len(c.Journal.Main()))
				}
				q, _ := c.Service.QuarantineItem(ctx, r.QuarantineID)
				if q.Code != r.Code || q.Status != app.QuarantineOpen || !strings.HasPrefix(q.MaterialAddress, "streebog256:") {
					t.Fatalf("карантин: %+v", q)
				}
				if r.Status != 422 {
					t.Fatalf("HTTP %d", r.Status)
				}
			case "accepted":
				m := c.Journal.Main()
				if len(m) != 1 || m[0].Entry.EventType == "ingest.message.quarantined" {
					t.Fatalf("журнал: %+v", m)
				}
				// Случай 2: новое необязательное поле сохранено в исходнике.
				if !strings.Contains(payload(t, m[0].Envelope), "spindle_warmup_s") {
					t.Fatal("новое поле не сохранено")
				}
				if r.SignatureNote != app.SignatureNotVerified || r.SignatureVerified {
					t.Fatalf("пометка подписи: %+v", r)
				}
			case "accepted_with_flag":
				if c.Journal.Count("ingest.anomaly.flagged") != 1 || r.Flags[0].RawValue != ex.Expect["stored_as"] {
					t.Fatalf("флаг: %+v", r.Flags)
				}
				if r.Status != 202 {
					t.Fatalf("HTTP %d", r.Status)
				}
			}
			checkEnvelopes(t, c)
		})
	}
}

// payload — содержимое DSSE-конверта записи.
func payload(t *testing.T, env []byte) string {
	t.Helper()
	var d struct {
		Payload string `json:"payload"`
	}
	if err := json.Unmarshal(env, &d); err != nil {
		t.Fatal(err)
	}
	b, err := base64.StdEncoding.DecodeString(d.Payload)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// checkEnvelopes — самопроверка: каждая служебная запись приёма проходит схемы
// контракта (конверт + data своего типа).
func checkEnvelopes(t *testing.T, c *inmem.Core) {
	t.Helper()
	all := append(c.Journal.Main(), c.Journal.CA()...)
	for _, s := range all {
		if !strings.HasPrefix(s.Entry.EventType, "ingest.") && !strings.HasPrefix(s.Entry.EventType, "security.") {
			continue
		}
		d, err := app.ValidateEnvelope([]byte(payload(t, s.Envelope)))
		if err != nil || d.Outcome != dom.OutcomeAccepted {
			t.Fatalf("%s: конверт не по схеме: %+v %v\n%s", s.Entry.EventType, d, err, payload(t, s.Envelope))
		}
	}
}

func event(id, src string, seq int64, occurred time.Time, data string) []byte {
	return fmt.Appendf(nil, `{"event_id":"%s","event_type":"operation.run.started","schema_version":1,"source_id":"%s","source_seq":%d,
"source_kind":"machine","reliability":"high","occurred_at":"%s","correlation_id":"01929a2b-7c3d-7e4f-8a5b-000000000001",
"causation_id":null,"item_id":"ENT01:FL-0007","integrity":{"format_version":1,"crypto_profile":"gost","signers":["device-%s@1"]},
"data":{"operation_run_id":"run-W2-FL-0007-1","operation_code":"030","step_key":"welding.weld","operator_id":"O17",%s}}`,
		id, src, seq, occurred.UTC().Format("2006-01-02T15:04:05.000Z"), src, data)
}

func uid(n int) string { return fmt.Sprintf("01929a2b-7c3d-7e4f-8a5b-%012d", n) }

// FR-31, кейс §4.5: повтор не меняет ни одного показателя; тот же ключ с
// другим содержимым — конфликт: карантин, событие security и запись CA.
func TestRepeatAndConflict(t *testing.T) {
	ctx := context.Background()
	c := demoCore(t)
	ev := event(uid(1), "edge-weld-1", 1, t0.Add(-time.Minute), `"station_id":"weld-2"`)
	first, err := c.Service.Ingest(ctx, ev)
	if err != nil || first.Outcome != app.OutcomeAccepted {
		t.Fatalf("%+v %v", first, err)
	}
	before, _ := c.Service.Stats(ctx)
	n := len(c.Journal.Main())
	for range 3 {
		r, err := c.Service.Ingest(ctx, ev)
		if err != nil || r.Outcome != app.OutcomeDuplicate || r.Seq != first.Seq || !r.Replayed {
			t.Fatalf("повтор: %+v %v", r, err)
		}
	}
	after, _ := c.Service.Stats(ctx)
	if len(c.Journal.Main()) != n || after.Accepted != before.Accepted || after.Duplicates != 3 || after.QuarantineOpen != 0 {
		t.Fatalf("повтор изменил показатели: журнал %d→%d, %+v", n, len(c.Journal.Main()), after)
	}
	if after.Sources[0].CompletenessBP != 10000 || after.Sources[0].HighWater != 1 {
		t.Fatalf("учёт номеров: %+v", after.Sources)
	}
	// Тот же ключ, другое содержимое.
	bad := event(uid(1), "edge-weld-1", 1, t0.Add(-time.Minute), `"station_id":"weld-3"`)
	r, err := c.Service.Ingest(ctx, bad)
	if err != nil || r.Outcome != app.OutcomeConflict || r.Code != errcodes.IngestDuplicateConflict || r.Status != 409 {
		t.Fatalf("конфликт: %+v %v", r, err)
	}
	if c.Journal.Count("security.idempotency.conflict") != 1 || len(c.Journal.CA()) != 1 || c.Journal.Count("ingest.message.quarantined") != 1 {
		t.Fatalf("конфликт: security %d, CA %d", c.Journal.Count("security.idempotency.conflict"), len(c.Journal.CA()))
	}
	if !strings.Contains(r.Detail, "CA-1") {
		t.Fatalf("нет ссылки на CA: %s", r.Detail)
	}
	// Исходное не перезаписано: повтор первого — прежний ответ.
	if r2, _ := c.Service.Ingest(ctx, ev); r2.Outcome != app.OutcomeDuplicate || r2.Seq != first.Seq {
		t.Fatalf("исходное: %+v", r2)
	}
	// Повтор конфликтного — тот же ответ, без новых записей.
	n = len(c.Journal.Main())
	if r3, _ := c.Service.Ingest(ctx, bad); r3.Outcome != app.OutcomeConflict || !r3.Replayed || len(c.Journal.Main()) != n {
		t.Fatalf("повтор конфликта: %+v", r3)
	}
	checkEnvelopes(t, c)
}

// fakeVerifier — проверка «подписи» для тестов: sig = base64("ok:"+keyid).
type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, env []byte) (signing.Verified, error) {
	var d struct {
		PayloadType string `json:"payloadType"`
		Payload     string `json:"payload"`
		Signatures  []struct {
			KeyID string `json:"keyid"`
			Sig   string `json:"sig"`
		} `json:"signatures"`
	}
	if err := json.Unmarshal(env, &d); err != nil {
		return signing.Verified{}, err
	}
	p, _ := base64.StdEncoding.DecodeString(d.Payload)
	v := signing.Verified{PayloadType: d.PayloadType, Payload: p}
	for _, s := range d.Signatures {
		if s.Sig != base64.StdEncoding.EncodeToString([]byte("ok:"+s.KeyID)) {
			return signing.Verified{}, fmt.Errorf("подпись %s не сходится", s.KeyID)
		}
		v.Signers = append(v.Signers, signing.SignerRef{KeyRef: s.KeyID, Profile: "gost"})
	}
	return v, nil
}

func sign(ev []byte, keyid string, good bool) []byte {
	c, _ := dom.Canonicalize(ev)
	sig := "ok:" + keyid
	if !good {
		sig = "bad"
	}
	b, _ := json.Marshal(map[string]any{"payloadType": app.PayloadTypeEvent, "payload": base64.StdEncoding.EncodeToString(c),
		"signatures": []map[string]string{{"keyid": keyid, "sig": base64.StdEncoding.EncodeToString([]byte(sig))}}})
	return b
}

// FR-26: приём только от зарегистрированных источников; неподписанное,
// неизвестный и отозванный ключ — отказ с кодом и событие безопасности.
func TestSignatureRequired(t *testing.T) {
	ctx := context.Background()
	cfg := app.DefaultConfig()
	cfg.Profile = "prod"
	c := inmem.NewCore(cfg, inmem.NewClock(t0), func(d *app.Deps) { d.Verifier = fakeVerifier{} })
	c.Service = app.NewService(app.WithConfig(cfg), app.WithDeps(app.Deps{Journal: c.Journal, Registry: c.Registry,
		Quarantine: c.Quarantine, Materials: c.Materials, Verifier: fakeVerifier{}, Keys: c.Keys,
		DomainClock: c.Clock, InfraClock: c.Clock.Infra(), Telemetry: c.Telemetry}))
	c.Keys.Register(app.KeyInfo{KeyRef: "device-edge-weld-1@1", SourceID: "edge-weld-1", Provenance: "device"})
	c.Keys.Register(app.KeyInfo{KeyRef: "device-edge-weld-9@1", SourceID: "edge-weld-9", Provenance: "device"})
	ev := func(n int, src string) []byte {
		return event(uid(n), src, int64(n), t0.Add(-time.Second), `"station_id":"weld-2"`)
	}
	cases := []struct {
		name string
		raw  []byte
		code errcodes.Code
		fail string
	}{
		{"неподписанное", ev(1, "edge-weld-1"), errcodes.IngestSignatureInvalid, "bad_signature"},
		{"неизвестный источник", sign(ev(2, "edge-weld-5"), "device-edge-weld-5@1", true), errcodes.IngestUnknownSource, "unknown_key"},
		{"чужой ключ", sign(ev(3, "edge-weld-1"), "device-edge-weld-9@1", true), errcodes.IngestSignatureInvalid, "foreign_key"},
		{"подпись не сходится", sign(ev(4, "edge-weld-1"), "device-edge-weld-1@1", false), errcodes.IngestSignatureInvalid, "bad_signature"},
	}
	for i, tc := range cases {
		r, err := c.Service.Ingest(ctx, tc.raw)
		if err != nil || r.Outcome != app.OutcomeQuarantined || r.Code != tc.code {
			t.Fatalf("%s: %+v %v", tc.name, r, err)
		}
		if got := c.Journal.Count("security.signature.invalid"); got != i+1 {
			t.Fatalf("%s: событий безопасности %d", tc.name, got)
		}
		if !strings.Contains(payload(t, c.Journal.Main()[len(c.Journal.Main())-1].Envelope), tc.fail) {
			t.Fatalf("%s: нет failure=%s", tc.name, tc.fail)
		}
	}
	ok := sign(ev(5, "edge-weld-1"), "device-edge-weld-1@1", true)
	if r, _ := c.Service.Ingest(ctx, ok); r.Outcome != app.OutcomeAccepted || !r.SignatureVerified || r.SignatureNote != "" {
		t.Fatalf("подписанное: %+v", r)
	}
	// FR-26, FR-70: отозванный ключ — отказ с кодом и шина безопасности.
	c.Keys.Revoke("device-edge-weld-1@1")
	r, _ := c.Service.Ingest(ctx, sign(ev(6, "edge-weld-1"), "device-edge-weld-1@1", true))
	if r.Code != errcodes.IngestRevokedKey || r.Status != 403 || c.Journal.Count("security.signature.invalid") != 5 {
		t.Fatalf("отозванный: %+v", r)
	}
	checkEnvelopes(t, c)
}

// FR-39, AD-7: недоступность и восстановление — пачка досылки закрывает
// разрыв без потерь и дублей; незакрытый за окно разрыв даёт сигнал потери.
func TestSequenceGapLossAndRecovery(t *testing.T) {
	ctx := context.Background()
	c := demoCore(t)
	send := func(ns ...int) {
		var msgs []json.RawMessage
		for _, n := range ns {
			msgs = append(msgs, event(uid(n), "edge-weld-1", int64(n), t0.Add(time.Duration(n)*time.Second-time.Hour), `"station_id":"weld-2"`))
		}
		b, _ := json.Marshal(map[string]any{"source_id": "edge-weld-1", "sent_at": c.Clock.At().Format("2006-01-02T15:04:05.000Z"), "messages": msgs})
		br, err := c.Service.IngestBatch(ctx, b)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range br.Results {
			if r.Outcome != app.OutcomeAccepted && r.Outcome != app.OutcomeDuplicate {
				t.Fatalf("%+v", r)
			}
		}
	}
	send(1, 2, 6)
	st, _ := c.Service.Stats(ctx)
	if st.Sources[0].Missing != 3 || st.Sources[0].CompletenessBP != 5000 {
		t.Fatalf("разрыв: %+v", st.Sources)
	}
	// Досылка в окне отменяет сигнал.
	c.Clock.Advance(30 * time.Second)
	send(3, 4, 4, 2)
	if l, _ := c.Service.CheckLosses(ctx); len(l) != 0 {
		t.Fatalf("рано: %+v", l)
	}
	c.Clock.Advance(2 * time.Minute)
	l, err := c.Service.CheckLosses(ctx)
	if err != nil || len(l) != 1 || l[0].From != 5 || l[0].To != 5 {
		t.Fatalf("потеря: %+v %v", l, err)
	}
	if l2, _ := c.Service.CheckLosses(ctx); len(l2) != 0 {
		t.Fatal("повторный сигнал")
	}
	st, _ = c.Service.Stats(ctx)
	if st.Accepted != 5 || st.Duplicates != 2 || c.Journal.Count("operation.run.started") != 5 {
		t.Fatalf("без потерь и дублей: %+v", st)
	}
	checkEnvelopes(t, c)
}

// FR-33, AD-5: флаги часов и последовательности.
func TestClockAndSequenceFlags(t *testing.T) {
	ctx := context.Background()
	c := demoCore(t)
	// Событие из будущего.
	r, _ := c.Service.Ingest(ctx, event(uid(1), "cam-1", 1, t0.Add(5*time.Minute), `"station_id":"weld-2"`))
	if len(r.Flags) != 1 || r.Flags[0].Flag != "future_timestamp" {
		t.Fatalf("из будущего: %+v", r)
	}
	// Часы источника отстают на 10 минут (sent_at пачки).
	b, _ := json.Marshal(map[string]any{"source_id": "cam-1", "sent_at": t0.Add(-10 * time.Minute).Format("2006-01-02T15:04:05.000Z"),
		"messages": []json.RawMessage{event(uid(2), "cam-1", 2, t0.Add(-10*time.Minute), `"station_id":"weld-2"`)}})
	br, _ := c.Service.IngestBatch(ctx, b)
	if f := br.Results[0].Flags; len(f) != 1 || f[0].Flag != "clock_skew" || f[0].SkewMS != -600000 {
		t.Fatalf("сдвиг часов: %+v", br.Results[0])
	}
	// Номер 2 у другого события — нарушение последовательности.
	r, _ = c.Service.Ingest(ctx, event(uid(3), "cam-1", 2, t0, `"station_id":"weld-2"`))
	if r.Outcome != app.OutcomeAccepted || len(r.Flags) != 1 || r.Flags[0].Flag != "sequence_violation" {
		t.Fatalf("последовательность: %+v", r)
	}
	if c.Journal.Count("ingest.anomaly.flagged") != 3 {
		t.Fatal("флаги не записаны")
	}
	checkEnvelopes(t, c)
}

// FR-141, FR-140: ручной ввод и импорт CSV — те же события, с пометкой источника.
func TestManualAndImport(t *testing.T) {
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "O17"})
	c := demoCore(t)
	occ := t0.Add(-time.Hour)
	r, err := c.Service.SubmitManual(ctx, app.Cmd[app.ManualInput]{Meta: platform.CommandMeta{CommandID: uid(7)},
		Body: app.ManualInput{SourceID: "terminal-weld-2", EventType: "operation.run.started", OccurredAt: &occ, ItemID: "ENT01:FL-0007",
			Data: map[string]any{"operation_run_id": "run-W2-FL-0007-1", "operation_code": "030", "step_key": "welding.weld", "operator_id": "O17"}}})
	if err != nil || r.Outcome != app.OutcomeAccepted || r.SignatureNote != app.ManualNote {
		t.Fatalf("ручной ввод: %+v %v", r, err)
	}
	m := c.Journal.Main()
	if e := m[len(m)-1].Entry; e.ProvenanceClass != "personal" || !strings.Contains(payload(t, m[len(m)-1].Envelope), `"source_kind":"manual_entry"`) {
		t.Fatalf("пометка ручного ввода: %+v", e)
	}
	// Повтор формы с тем же command_id не удваивает факт.
	if r2, _ := c.Service.SubmitManual(ctx, app.Cmd[app.ManualInput]{Meta: platform.CommandMeta{CommandID: uid(7)},
		Body: app.ManualInput{SourceID: "terminal-weld-2", EventType: "operation.run.started", OccurredAt: &occ, ItemID: "ENT01:FL-0007",
			Data: map[string]any{"operation_run_id": "run-W2-FL-0007-1", "operation_code": "030", "step_key": "welding.weld", "operator_id": "O17"}}}); r2.Outcome != app.OutcomeDuplicate {
		t.Fatalf("повтор формы: %+v", r2)
	}
	csv := "row_id;event_type;occurred_at;item_id;data.operation_run_id;data.operation_code;data.step_key;data.station_id;data.operator_id\n" +
		"W-001;operation.run.started;2026-09-25 09:00:00;ENT01:FL-0008;run-W2-FL-0008-1;030;welding.weld;weld-2;O17\n" +
		"W-002;operation.run.started;2026-09-25 09:30;ENT01:FL-0009;run-W2-FL-0009-1;030;welding.weld;weld-2;O17\n" +
		"W-003;operation.run.started;вчера;ENT01:FL-0010;run-W2-FL-0010-1;030;welding.weld;weld-2;O17\n" +
		"W-004;operation.run.started;2026-09-25 10:00:00;ENT01:FL-0011;;030;welding.weld;weld-2;O17\n"
	in := app.Cmd[app.ImportInput]{Meta: platform.CommandMeta{CommandID: uid(8)},
		Body: app.ImportInput{SourceID: "import:weld-journal", FileName: "журнал сварки.csv", Content: []byte(csv), Defaults: map[string]string{"utc_offset": "+03:00"}}}
	ir, err := c.Service.ImportCSV(ctx, in)
	if err != nil || ir.Total != 4 || ir.Accepted != 2 || ir.Rejected != 2 {
		t.Fatalf("импорт: %+v %v", ir, err)
	}
	if ir.Rows[3].Result.Code != errcodes.IngestMissingRequiredField {
		t.Fatalf("строка без поля: %+v", ir.Rows[3].Result)
	}
	var imported string
	for _, s := range c.Journal.Main() {
		if s.Entry.SourceID == "import:weld-journal" && s.Entry.EventType == "operation.run.started" {
			imported = payload(t, s.Envelope)
			if s.Entry.OccurredAt != "2026-09-25T06:00:00.000Z" && s.Entry.OccurredAt != "2026-09-25T06:30:00.000Z" {
				t.Fatalf("время строки: %s", s.Entry.OccurredAt)
			}
		}
	}
	if !strings.Contains(imported, `"source_kind":"import"`) {
		t.Fatalf("пометка импорта: %s", imported)
	}
	// Повторный импорт того же файла (другая команда) — дубли, а не второй учёт.
	in.Meta.CommandID = uid(9)
	ir2, _ := c.Service.ImportCSV(ctx, in)
	if ir2.Duplicate != 2 || ir2.Accepted != 0 || c.Journal.Count("operation.run.started") != 3 {
		t.Fatalf("повторный импорт: %+v", ir2)
	}
	if c.Journal.Count("ingest.import.completed") != 2 {
		t.Fatal("итог импорта")
	}
	checkEnvelopes(t, c)
}

// FR-30: администратор переобрабатывает и закрывает записи карантина.
func TestReprocess(t *testing.T) {
	ctx := context.Background()
	c := demoCore(t)
	ex := examples(t)["01-unknown-version.json"]
	r, _ := c.Service.Ingest(ctx, ex.Message)
	rr, err := c.Service.ReprocessCommand(ctx, app.Cmd[app.ReprocessInput]{Meta: platform.CommandMeta{CommandID: uid(20), Reason: "повышатель v3 ещё не выпущен"},
		Body: app.ReprocessInput{QuarantineID: r.QuarantineID}})
	if err != nil || rr.Outcome != "still_invalid" {
		t.Fatalf("переобработка: %+v %v", rr, err)
	}
	rr, err = c.Service.ReprocessCommand(ctx, app.Cmd[app.ReprocessInput]{Meta: platform.CommandMeta{CommandID: uid(21), Reason: "источник пришлёт заново"},
		Body: app.ReprocessInput{QuarantineID: r.QuarantineID, Discard: true}})
	if err != nil || rr.Outcome != "discarded" {
		t.Fatalf("отброшено: %+v %v", rr, err)
	}
	if q, _ := c.Service.QuarantineItem(ctx, r.QuarantineID); q.Status != app.QuarantineDiscarded {
		t.Fatalf("%+v", q)
	}
	if c.Journal.Count("ingest.message.reprocessed") != 2 {
		t.Fatal("решения переобработки")
	}
	list, _ := c.Service.QuarantineRecords(ctx, app.QuarantineQuery{Status: app.QuarantineOpen})
	if len(list) != 0 {
		t.Fatalf("%+v", list)
	}
	checkEnvelopes(t, c)
}

// FR-41: метрики приёма в /metrics.
func TestMetrics(t *testing.T) {
	ctx := context.Background()
	c := demoCore(t)
	ev := event(uid(1), "edge-weld-1", 1, t0.Add(-time.Minute), `"station_id":"weld-2"`)
	_, _ = c.Service.Ingest(ctx, ev)
	_, _ = c.Service.Ingest(ctx, ev)
	_, _ = c.Service.Ingest(ctx, []byte(`{"event_type":"x"`))
	if c.Telemetry.Sum(app.MetricMessages, "accepted") != 1 || c.Telemetry.Sum(app.MetricMessages, "duplicate") != 1 ||
		c.Telemetry.Sum(app.MetricMessages, "quarantined") != 1 {
		t.Fatalf("%+v", c.Telemetry.Counters)
	}
	keys := []string{}
	for k := range c.Telemetry.Gauges {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if !slices.ContainsFunc(keys, func(k string) bool { return strings.HasPrefix(k, app.MetricQuarantineOpen) }) ||
		!slices.ContainsFunc(keys, func(k string) bool { return strings.HasPrefix(k, app.MetricCompleteness) }) {
		t.Fatalf("%v", keys)
	}
	if c.Telemetry.Observed[app.MetricLatency+"{}"] != 3 {
		t.Fatalf("%v", c.Telemetry.Observed)
	}
}

// Операции API (формы views.go): пачка с sent_at, импорт, источники, метрики.
func TestAPIForms(t *testing.T) {
	ctx := context.Background()
	c := demoCore(t)
	env := func(ev []byte) crypto.DsseEnvelope {
		cn, _ := dom.Canonicalize(ev)
		return crypto.DsseEnvelope{PayloadType: app.PayloadTypeEvent, Payload: base64.StdEncoding.EncodeToString(cn)}
	}
	sent := t0.Add(-10 * time.Minute)
	res, err := c.Service.SubmitBatch(ctx, app.IngestBatch{SourceID: "edge-weld-1", SentAt: &sent, Envelopes: []crypto.DsseEnvelope{
		env(event(uid(1), "edge-weld-1", 1, t0.Add(-11*time.Minute), `"station_id":"weld-2"`)),
		env(event(uid(1), "edge-weld-1", 1, t0.Add(-11*time.Minute), `"station_id":"weld-2"`)),
		env(event(uid(3), "edge-weld-1", 3, t0.Add(-11*time.Minute), `"station_id":"weld-2"`)),
	}})
	if err != nil || res.Accepted != 2 || res.Duplicates != 1 || res.Items[0].SignatureChecked || !slices.Contains(res.Items[0].Flags, "clock_skew") {
		t.Fatalf("пачка: %+v %v", res, err)
	}
	src, _ := c.Service.Sources(ctx, platform.Page{})
	if len(src.Items) != 1 || src.Items[0].GapCount != 1 || *src.Items[0].ClockSkewMs != -600000 || *src.Items[0].LastSeq != 3 {
		t.Fatalf("источники: %+v", src.Items)
	}
	csv := "row_id,occurred_at,item_id,data.operation_run_id,data.operation_code,data.step_key,data.operator_id\n" +
		"1,2026-09-25T09:00:00.000Z,ENT01:FL-0020,run-1,030,welding.weld,O17\n2,2026-09-25T09:10:00.000Z,ENT01:FL-0021,run-2,030,welding.weld,\n"
	in := app.ImportFile{Format: "csv", FileName: "weld.csv", Mapping: "operation.run.started", Content: base64.StdEncoding.EncodeToString([]byte(csv)), DryRun: true}
	ir, err := c.Service.Import(ctx, in)
	if err != nil || ir.RowsAccepted != 1 || ir.RowsRejected != 1 || c.Journal.Count("ingest.import.completed") != 0 {
		t.Fatalf("проверка импорта: %+v %v", ir, err)
	}
	in.DryRun = false
	if ir, err = c.Service.Import(ctx, in); err != nil || ir.RowsAccepted != 1 || ir.SourceID != "import:operation.run.started" || c.Journal.Count("ingest.import.completed") != 1 {
		t.Fatalf("импорт: %+v %v", ir, err)
	}
	if _, err := c.Service.Import(ctx, app.ImportFile{Format: "xlsx"}); err == nil {
		t.Fatal("xlsx")
	}
	m, err := c.Service.Metrics(ctx)
	if err != nil || m.Accepted != 3 || m.Duplicates != 1 || m.CompletenessBP == 10000 {
		t.Fatalf("метрики: %+v %v", m, err)
	}
	// Один отказ по ingest.event.submit — problem+json с кодом (ошибка порта).
	bad := env([]byte(strings.Replace(string(event(uid(9), "cam-2", 1, t0, `"station_id":"weld-2"`)), `"schema_version":1`, `"schema_version":5`, 1)))
	_, err = c.Service.SubmitEvent(ctx, app.ManualEvent{Envelope: bad})
	if e, ok := platform.AsError(err); !ok || e.Code != errcodes.IngestUnknownSchemaVersion || e.Params["quarantine_id"] == "" {
		t.Fatalf("отказ: %v", err)
	}
	checkEnvelopes(t, c)
}

// gate — выключенные источники (эпик 48, AD-28, AD-47).
type gate map[string]string

func (g gate) SourceBlocked(_ context.Context, id string) (bool, string, error) {
	why, ok := g[id]
	return ok, why, nil
}

// Эпик 48 (хвост 34): источник, отключённый администратором, или выключенная
// интеграция — приём отвергает с кодом ingest.source_disabled, сообщение в
// карантине; прочие источники принимаются.
func TestSourceDisabled(t *testing.T) {
	ctx := context.Background()
	cfg := app.DefaultConfig()
	cfg.Profile = "demo"
	c := inmem.NewCore(cfg, inmem.NewClock(t0), func(d *app.Deps) { d.Gate = gate{"erp.onec": "интеграция onec выключена"} })
	r, err := c.Service.Ingest(ctx, event(uid(1), "erp.onec", 1, t0.Add(-time.Second), `"station_id":"weld-2"`))
	if err != nil || r.Outcome != app.OutcomeQuarantined || r.Code != errcodes.IngestSourceDisabled || r.Status != 403 {
		t.Fatalf("выключенный источник: %+v %v", r, err)
	}
	r, err = c.Service.Ingest(ctx, event(uid(2), "edge-weld-1", 1, t0.Add(-time.Second), `"station_id":"weld-2"`))
	if err != nil || r.Outcome != app.OutcomeAccepted {
		t.Fatalf("включённый источник: %+v %v", r, err)
	}
}
