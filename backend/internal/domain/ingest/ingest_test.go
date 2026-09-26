package ingest

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"ant/internal/contracts/errcodes"
	"ant/internal/contracts/schemas"
)

// AD-10: повторяющиеся ключи, дробные и большие числа — не канонический вид.
func TestCanonicalizeRejects(t *testing.T) {
	for _, in := range []string{
		`{"a":1,"a":2}`,
		`{"a":1.5}`,
		`{"a":1e3}`,
		`{"a":9007199254740992}`,
		`{"a":"\xff"}`,
		`{"a":`,
	} {
		if _, err := Canonicalize([]byte(in)); !errors.Is(err, ErrCanonical) {
			t.Fatalf("%s: ждали ErrCanonical, получили %v", in, err)
		}
	}
	if _, err := Canonicalize([]byte(`{"a":-9007199254740991}`)); err != nil {
		t.Fatal(err)
	}
}

// AD-7: другой integrity (переподпись преемником) — то же содержимое; другой data — другое.
func TestContentFingerprintIgnoresIntegrity(t *testing.T) {
	a, _ := Canonicalize([]byte(`{"event_id":"x","integrity":{"signers":["k@1"]},"data":{"v":1}}`))
	b, _ := Canonicalize([]byte(`{"data":{"v":1},"integrity":{"signers":["k@2"]},"event_id":"x"}`))
	c, _ := Canonicalize([]byte(`{"data":{"v":2},"integrity":{"signers":["k@1"]},"event_id":"x"}`))
	fa, _ := ContentFingerprint(a)
	fb, _ := ContentFingerprint(b)
	fc, _ := ContentFingerprint(c)
	if fa != fb || fa == fc {
		t.Fatalf("отпечатки: %s %s %s", fa, fb, fc)
	}
	if CheckRepeat(nil, fa) != RepeatNew || CheckRepeat(&Seen{Fingerprint: fa}, fb) != RepeatDuplicate ||
		CheckRepeat(&Seen{Fingerprint: fa}, fc) != RepeatConflict {
		t.Fatal("CheckRepeat")
	}
}

// FR-29, кейс §4.7: пять случаев изменения контракта.
func TestClassifyFiveCases(t *testing.T) {
	info := TypeInfo{Known: true, Fact: true, Versions: []int{1, 2}}
	// 1. Неизвестная версия.
	if d, stop := ClassifyVersion("equipment.state.changed", 3, info); !stop || d.Code != errcodes.IngestUnknownSchemaVersion {
		t.Fatalf("случай 1: %+v", d)
	}
	if d, stop := ClassifyVersion("nope.x.y", 1, TypeInfo{}); !stop || d.Code != errcodes.IngestUnknownEventType {
		t.Fatalf("неизвестный тип: %+v", d)
	}
	if d, stop := ClassifyVersion("decision.x.y", 1, TypeInfo{Known: true, Versions: []int{1}}); !stop || d.Code != errcodes.IngestUnknownEventType {
		t.Fatalf("не факт: %+v", d)
	}
	if _, stop := ClassifyVersion("equipment.state.changed", 2, info); stop {
		t.Fatal("версия 2 известна")
	}
	// 2. Новое необязательное поле — нарушений нет.
	if d := Classify("operation.run.started", nil); d.Outcome != OutcomeAccepted {
		t.Fatalf("случай 2: %+v", d)
	}
	// 3. Нет обязательного поля (и случай 5 — несовместимое изменение без новой версии).
	d := Classify("inspection.result.recorded", []Violation{
		{Kind: ViolationEnum, Pointer: "/data/defects/0/severity", Value: "x"},
		{Kind: ViolationRequired, Pointer: "/data", Missing: "outcome"},
	})
	if d.Outcome != OutcomeQuarantined || d.Code != errcodes.IngestMissingRequiredField || d.Field != "/data/outcome" {
		t.Fatalf("случай 3: %+v", d)
	}
	// 4a. Неизвестное значение перечисления — UNKNOWN(значение) с флагом.
	d = Classify("operation.run.paused", []Violation{{Kind: ViolationEnum, Pointer: "/data/pause_reason", Value: "coffee_break"}})
	if d.Outcome != OutcomeAcceptedWithFlag || d.Code != errcodes.IngestUnknownEnumValue || d.Flags[0].StoredAs() != "UNKNOWN(coffee_break)" {
		t.Fatalf("случай 4a: %+v", d)
	}
	// 4b. То же в поле, важном для безопасности, — карантин.
	d = Classify("inspection.result.recorded", []Violation{{Kind: ViolationEnum, Pointer: "/data/outcome", Value: "probably_ok"}})
	if d.Outcome != OutcomeQuarantined || d.Code != errcodes.IngestUnknownEnumValueCritical {
		t.Fatalf("случай 4b: %+v", d)
	}
	d = Classify("inspection.result.recorded", []Violation{{Kind: ViolationEnum, Pointer: "/data/defects/3/severity", Value: "huge"}})
	if d.Code != errcodes.IngestUnknownEnumValueCritical {
		t.Fatalf("массив в поле безопасности: %+v", d)
	}
	// Перечисления конверта важны всегда.
	d = Classify("operation.run.paused", []Violation{{Kind: ViolationEnum, Pointer: "/source_kind", Value: "robot"}})
	if d.Code != errcodes.IngestUnknownEnumValueCritical {
		t.Fatalf("конверт: %+v", d)
	}
	// Прочее — нарушение схемы.
	d = Classify("operation.run.paused", []Violation{{Kind: ViolationOther, Pointer: "/data/x", Message: "не строка"}})
	if d.Code != errcodes.IngestSchemaViolation {
		t.Fatalf("прочее: %+v", d)
	}
}

// Флаги совпадают с перечислением схемы ingest.anomaly.flagged.v1.
func TestFlagsMatchContract(t *testing.T) {
	raw, err := schemas.FS.ReadFile("contracts/events/ingest/ingest.anomaly.flagged.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var s struct {
		Properties struct {
			Flag struct {
				Enum []Flag `json:"enum"`
			} `json:"flag"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(s.Properties.Flag.Enum, Flags) {
		t.Fatalf("контракт %v, код %v", s.Properties.Flag.Enum, Flags)
	}
}

// AD-7, FR-39: разрыв, досылка в окне, нарушение, потеря после окна, полнота.
func TestObserveSeq(t *testing.T) {
	t0 := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	var s SourceState
	var o SeqObservation
	for _, n := range []int64{1, 2} {
		s, o = ObserveSeq(s, n, t0)
		if o.Kind != SeqInOrder {
			t.Fatalf("%d: %v", n, o.Kind)
		}
	}
	s, o = ObserveSeq(s, 6, t0)
	if o.Kind != SeqGapOpened || o.Gap.From != 3 || o.Gap.To != 5 || s.Missing() != 3 {
		t.Fatalf("разрыв: %+v %+v", o, s)
	}
	if s.CompletenessBP() != 5000 {
		t.Fatalf("полнота %d", s.CompletenessBP())
	}
	s, o = ObserveSeq(s, 4, t0.Add(time.Second))
	if o.Kind != SeqGapFilled || len(s.Gaps) != 2 || s.Missing() != 2 {
		t.Fatalf("досылка: %+v %+v", o, s)
	}
	if _, o = ObserveSeq(s, 2, t0); o.Kind != SeqViolation {
		t.Fatalf("повтор номера: %v", o.Kind)
	}
	// В окне ожидания сигнала нет; досылка 3 закрывает часть.
	s, _ = ObserveSeq(s, 3, t0.Add(2*time.Second))
	s2, due := DueLosses(s, t0.Add(30*time.Second), time.Minute)
	if len(due) != 0 {
		t.Fatalf("рано: %+v", due)
	}
	s2, due = DueLosses(s2, t0.Add(2*time.Minute), time.Minute)
	if len(due) != 1 || due[0].From != 5 || due[0].To != 5 {
		t.Fatalf("потеря: %+v", due)
	}
	if _, due = DueLosses(s2, t0.Add(3*time.Minute), time.Minute); len(due) != 0 {
		t.Fatal("повторный сигнал по тому же разрыву")
	}
	// Первое сообщение источника не с 1 — разрыв с начала.
	if _, o = ObserveSeq(SourceState{}, 5, t0); o.Kind != SeqGapOpened || o.Gap.From != 1 {
		t.Fatalf("начало с 5: %+v", o)
	}
}

// FR-33, AD-5: время из будущего и расхождение часов источника.
func TestCheckClock(t *testing.T) {
	now := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	p := DefaultClockPolicy
	if f := CheckClock(now.Add(-time.Hour), now, nil, p); len(f) != 0 {
		t.Fatalf("поздняя пачка — не сдвиг часов: %+v", f)
	}
	f := CheckClock(now.Add(5*time.Minute), now, nil, p)
	if len(f) != 1 || f[0].Flag != FlagFutureTimestamp || f[0].SkewMS != 300000 {
		t.Fatalf("из будущего: %+v", f)
	}
	sent := now.Add(-10 * time.Minute)
	f = CheckClock(now.Add(-11*time.Minute), now, &sent, p)
	if len(f) != 1 || f[0].Flag != FlagClockSkew || f[0].SkewMS != -600000 {
		t.Fatalf("сдвиг: %+v", f)
	}
}

// AD-41, FR-34: простой случай привязки и партиция.
func TestBind(t *testing.T) {
	if b := Bind("ENT01:FL-0007", nil); b.ItemID != "ENT01:FL-0007" || b.Basis != "internal_id" || b.Reliability != "high" {
		t.Fatalf("%+v", b)
	}
	if b := Bind("", &ItemRef{CarrierType: "internal_id", Value: "ENT01:FL-0008", IdentificationLevel: "unique"}); b.ItemID != "ENT01:FL-0008" {
		t.Fatalf("%+v", b)
	}
	b := Bind("", &ItemRef{CarrierType: "dpm_datamatrix", Value: "DM-77", IdentificationLevel: "probable"})
	if !b.NeedsLookup || b.ItemID != "" || b.CarrierRef != "dpm_datamatrix:DM-77" || b.Reliability != "medium" {
		t.Fatalf("%+v", b)
	}
	if b := Bind("", nil); !b.Unbound {
		t.Fatalf("%+v", b)
	}
	p := Partition("ENT01:FL-0007", 16)
	if p < 0 || p >= 16 || p != Partition("ENT01:FL-0007", 16) {
		t.Fatal(p)
	}
}
