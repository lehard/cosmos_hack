package engine

import (
	"time"

	"ant/internal/domain/kernel"
)

// Axis — ось момента (AD-22, AD-37).
type Axis string

// Оси момента.
const (
	// AxisOccurred — «как было»: записи с occurred_at ≤ T по всему известному.
	AxisOccurred Axis = "occurred"
	// AxisRecorded — «что мы знали»: префикс журнала по seq с recorded_at ≤ T
	// (recorded_at не убывает по seq, AD-37).
	AxisRecorded Axis = "recorded"
)

// Moment — момент запроса состояния: ось и T. Нулевой T — «сейчас» (весь вход).
type Moment struct {
	Axis Axis
	At   time.Time
	// MaxSeq — дополнительно ограничить префикс по seq (0 — без ограничения):
	// свёртка «на basis_seq» для гардов и верификатора (AD-9, AD-39).
	MaxSeq int64
}

// Prefix — вход изделия на момент (AD-22): для «как было» — occurred_at ≤ T,
// для «что мы знали» — recorded_at ≤ T; MaxSeq отсекает по seq. Порядок входа
// не меняется. Чистая функция: запросы на момент, воспроизведение и прогон
// «по другой версии правил» пользуются той же свёрткой, что и воркер, и ничего
// не пишут.
func Prefix(input []kernel.Record, m Moment) []kernel.Record {
	out := make([]kernel.Record, 0, len(input))
	for _, r := range input {
		if m.MaxSeq > 0 && r.Seq > m.MaxSeq {
			continue
		}
		if !m.At.IsZero() {
			t := r.OccurredAt
			if m.Axis == AxisRecorded {
				t = r.RecordedAt
			}
			if t.After(m.At) {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// FoldAt — свёртка префикса входа на момент (AD-22): состояние и реакции,
// которые система вычислила бы на T. Ничего не пишет.
func FoldAt(b Bundle, input []kernel.Record, m Moment) (Snapshot, []kernel.Reaction) {
	return Fold(b, Prefix(input, m))
}
