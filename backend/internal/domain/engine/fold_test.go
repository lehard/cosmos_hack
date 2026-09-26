package engine

import (
	"testing"
	"time"

	"ant/internal/domain/kernel"
)

// Рамка движка: свёртка пустых модулей детерминирована и не зависит от
// порядка подачи входа (AD-4, AD-5).
func TestFoldFrame(t *testing.T) {
	t0 := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	in := []kernel.Record{
		{Seq: 2, EventID: "b", Type: "operation.run.started", OccurredAt: t0.Add(time.Minute)},
		{Seq: 1, EventID: "a", Type: "item.item.registered", OccurredAt: t0},
	}
	s1, r1 := Fold(Bundle{}, in)
	s2, r2 := Fold(Bundle{}, []kernel.Record{in[1], in[0]})
	if s1.BasisSeq != 2 || s2.BasisSeq != 2 || len(r1) != len(r2) {
		t.Fatalf("свёртка не детерминирована: %v %v", s1, s2)
	}
}
