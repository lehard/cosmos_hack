package events

// Тесты JSON-методов Timestamp (timestamp_gen.go): соглашение спайна «Время» —
// RFC 3339 UTC, ровно три знака миллисекунд; круговое преобразование
// сгенерированного типа с полями времени (обязательным и необязательным).

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTimestampRoundTrip(t *testing.T) {
	cases := []string{
		"2026-09-26T08:15:30.000Z",
		"2026-09-26T08:15:30.123Z",
		"1999-12-31T23:59:59.999Z",
	}
	for _, s := range cases {
		var ts Timestamp
		if err := json.Unmarshal([]byte(`"`+s+`"`), &ts); err != nil {
			t.Fatalf("%s: разбор: %v", s, err)
		}
		b, err := json.Marshal(ts)
		if err != nil {
			t.Fatalf("%s: запись: %v", s, err)
		}
		if got := string(b); got != `"`+s+`"` {
			t.Fatalf("круговое преобразование: %s → %s", s, got)
		}
	}
}

func TestTimestampMarshalNormalizes(t *testing.T) {
	msk := time.FixedZone("MSK", 3*3600)
	ts := Timestamp(time.Date(2026, 9, 26, 11, 15, 30, 123_456_789, msk))
	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), `"2026-09-26T08:15:30.123Z"`; got != want {
		t.Fatalf("UTC и три знака: %s, ожидалось %s", got, want)
	}
	if got := NewTimestamp(time.Time(ts)).Time(); !got.Equal(time.Date(2026, 9, 26, 8, 15, 30, 123_000_000, time.UTC)) || got.Location() != time.UTC {
		t.Fatalf("NewTimestamp: %v", got)
	}
}

func TestTimestampUnmarshalStrict(t *testing.T) {
	bad := []string{
		`"2026-09-26T08:15:30Z"`,          // без миллисекунд
		`"2026-09-26T08:15:30.12Z"`,       // два знака
		`"2026-09-26T08:15:30.1234Z"`,     // четыре знака
		`"2026-09-26T11:15:30.123+03:00"`, // не UTC
		`"2026-09-26 08:15:30.123Z"`,      // пробел вместо T
		`1695716130123`,                   // число
	}
	for _, s := range bad {
		var ts Timestamp
		if err := json.Unmarshal([]byte(s), &ts); err == nil {
			t.Fatalf("%s: ожидалась ошибка, получено %v", s, ts)
		}
	}
	ts := NewTimestamp(time.Date(2026, 1, 2, 3, 4, 5, 6_000_000, time.UTC))
	if err := json.Unmarshal([]byte(`null`), &ts); err != nil || ts.String() != "2026-01-02T03:04:05.006Z" {
		t.Fatalf("null не должен менять значение: %v %v", ts, err)
	}
}

// Сгенерированный тип с обязательными и необязательными полями времени
// разбирается и записывается без собственных структур модуля.
func TestTimestampInGeneratedType(t *testing.T) {
	in := `{"equipment_id":"WS-01","parameters":[],"window_end":"2026-09-26T08:20:00.000Z","window_start":"2026-09-26T08:15:30.250Z"}`
	var v EquipmentCycleSummarizedV1
	if err := json.Unmarshal([]byte(in), &v); err != nil {
		t.Fatal(err)
	}
	if got := v.WindowStart.Time(); !got.Equal(time.Date(2026, 9, 26, 8, 15, 30, 250_000_000, time.UTC)) {
		t.Fatalf("window_start: %v", got)
	}
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != in {
		t.Fatalf("круговое преобразование:\n%s\n%s", in, out)
	}

	started := NewTimestamp(time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC))
	run := OperationRunStartedV1{OperationCode: "welding", OperationRunID: "RUN-1", OperationStartedAt: &started}
	b, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"operation_started_at":"2026-09-26T08:00:00.000Z"`) {
		t.Fatalf("необязательное поле времени: %s", b)
	}
	var back OperationRunStartedV1
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.OperationStartedAt == nil || !back.OperationStartedAt.Time().Equal(started.Time()) {
		t.Fatalf("обратный разбор: %+v", back.OperationStartedAt)
	}
}
