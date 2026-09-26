package clock_test

import (
	"context"
	"errors"
	"testing"
	"time"

	app "ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
	store "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/clock"
	jt "ant/internal/infrastructure/storage/journal/journaltest"
)

// Режим часов — из генезиса; в режиме scenario доменное «сейчас» — последний
// тик прогона из журнала (AD-37, AD-38).
func TestJournalDomainClock(t *testing.T) {
	d := jt.NewDB(t)
	ctx := context.Background()
	s := store.NewStore(d.AppPool(t), jt.SysClock{})
	c := clock.NewJournal(s)
	if m, err := c.Mode(ctx); err != nil || m != clock.ModeSystem {
		t.Fatalf("без генезиса: %q %v", m, err)
	}
	mode := jt.Entry("time.clock.mode_set", jc.JournalEntryEntryKindService, "global", "", time.Now())
	mode.Envelope = jt.Envelope("time.clock.mode_set", map[string]string{"mode": "scenario"})
	if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{mode}}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Now(app.WithRun(ctx, "run-1")); !errors.Is(err, clock.ErrNoTick) {
		t.Fatalf("нет тиков: %v", err)
	}
	for i, now := range []string{"2026-01-10T08:00:00.000Z", "2026-01-10T09:30:00.000Z"} {
		for _, run := range []string{"run-1", "run-2"} {
			tick := jt.Entry("time.clock.ticked", jc.JournalEntryEntryKindService, "run:"+run, "", time.Now())
			r := run
			tick.Entry.RunID = &r
			v := now
			if run == "run-2" {
				v = "2027-02-0" + string(rune('1'+i)) + "T00:00:00.000Z"
			}
			tick.Envelope = jt.Envelope("time.clock.ticked", map[string]any{"now": v, "speed": 60})
			if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{tick}}); err != nil {
				t.Fatal(err)
			}
		}
	}
	got, err := c.Now(app.WithRun(ctx, "run-1"))
	if err != nil || !got.Equal(time.Date(2026, 1, 10, 9, 30, 0, 0, time.UTC)) {
		t.Fatalf("run-1: %v %v", got, err)
	}
	got, err = c.Now(app.WithRun(ctx, "run-2"))
	if err != nil || !got.Equal(time.Date(2027, 2, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("run-2: %v %v", got, err)
	}
}
