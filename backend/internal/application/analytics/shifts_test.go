package analytics_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"ant/internal/application/analytics"
	"ant/internal/application/platform"
)

// Смены графика справочника (FR-81, эпик 19): одна смена 08:00–16:30 МСК.
type oneShift struct{}

func (oneShift) Schedule(context.Context, string) (analytics.ShiftLookup, error) {
	from := time.Date(2026, 9, 26, 5, 0, 0, 0, time.UTC)
	to := from.Add(8*time.Hour + 30*time.Minute)
	return func(t time.Time) (analytics.ShiftSpan, bool) {
		if t.Before(from) {
			return analytics.ShiftSpan{}, false
		}
		return analytics.ShiftSpan{ID: "SHIFT-1@2026-09-26", Label: "Первая смена 26.09", From: from, To: to, Active: t.Before(to)}, true
	}, nil
}

// Период «смена» — от начала смены графика; длительность операций — со срезом «смена».
func TestShiftPeriodAndSliceFromSchedule(t *testing.T) {
	p := world(t)
	p.svc = analytics.NewService(analytics.WithStore(analytics.MemStore{Src: p.j, Items: func() []string { return p.items }}),
		analytics.WithClock(p.clock), analytics.WithShifts(oneShift{}))
	ov := p.overview(platform.Moment{})
	if want := time.Date(2026, 9, 26, 5, 0, 0, 0, time.UTC); !ov.Period.From.Equal(want) {
		t.Fatalf("начало периода «смена»: %s, ждали %s", ov.Period.From, want)
	}
	d := row(ov, "operation_duration")
	if !slices.ContainsFunc(d.Slices, func(s analytics.MetricSlice) bool { return s.Dimension == "shift" && s.Label == "Первая смена 26.09" }) {
		t.Fatalf("нет среза «смена»: %+v", d.Slices)
	}
}
