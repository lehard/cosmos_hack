package analytics

import (
	"testing"
	"time"
)

// Эпик 16, стык 17 и 25: «очередь» и «в работе» сейчас — по положению
// изделия в процессе; история (закрытые интервалы) — из фактов.
func TestApplyPositions(t *testing.T) {
	t0 := time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)
	closed := t0.Add(time.Hour)
	rows := []Row{
		{Metric: RowQueue, At: t0, Interval: true, Until: &closed, Value: 1, Dims: Dims{Run: "r", Label: "F-1", Step: "machining.cnc"}},
		{Metric: RowQueue, At: t0.Add(2 * time.Hour), Interval: true, Value: 1, Dims: Dims{Run: "r", Label: "F-1", Step: "welding.weld"}},
		{Metric: RowInProgress, At: t0.Add(3 * time.Hour), Interval: true, Value: 1, Dims: Dims{Run: "r", Label: "F-1", Step: "assembly.fit", Station: "ST-A"}},
	}
	ps := []Position{{Step: "assembly.fit", Kind: RowInProgress, Since: t0.Add(3 * time.Hour)}, {Step: "zt3", Kind: RowQueue, Since: t0.Add(4 * time.Hour)}}
	out := ApplyPositions("ENT01:F-1", rows, ps)
	at := t0.Add(5 * time.Hour)
	var queue, work []string
	for _, r := range out {
		if !r.ActiveAt(at) {
			continue
		}
		switch r.Metric {
		case RowQueue:
			queue = append(queue, r.Dims.Step)
		case RowInProgress:
			work = append(work, r.Dims.Step+"@"+r.Dims.Station)
		}
	}
	if len(queue) != 1 || queue[0] != "zt3" || len(work) != 1 || work[0] != "assembly.fit@ST-A" {
		t.Fatalf("очередь %v, в работе %v", queue, work)
	}
	hist := 0
	for _, r := range out {
		if r.Metric == RowQueue && r.Until != nil && r.Until.Equal(closed) {
			hist++
		}
	}
	if hist != 1 {
		t.Fatal("закрытый интервал истории потерян")
	}
	if got := ApplyPositions("ENT01:F-1", rows, nil); len(got) != 1 {
		t.Fatalf("изделие вне процесса — только история: %d строк", len(got))
	}
}
