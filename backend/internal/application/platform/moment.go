package platform

import (
	"fmt"
	"time"
)

// Axis — ось момента чтения (AD-21, AD-22, AD-37).
type Axis string

const (
	// AxisOccurred — «как было»: события с occurred_at ≤ T по всему известному (по умолчанию).
	AxisOccurred Axis = "occurred"
	// AxisRecorded — «что мы знали»: префикс журнала по seq с recorded_at ≤ T.
	AxisRecorded Axis = "recorded"
)

// Moment — момент, на который строится ответ чтения (AD-22). AsOf пуст —
// «сейчас»; задан — воспроизведение: команды выключены правилом прав (AD-21).
type Moment struct {
	Axis Axis
	AsOf *time.Time
	// RunID — прогон сценария, в пределах которого читаются данные (AD-38).
	RunID string
}

// IsReplay — чтение на момент в прошлом (воспроизведение).
func (m Moment) IsReplay() bool { return m.AsOf != nil }

// ParseMoment разбирает параметры запроса axis, as_of (RFC 3339) и run_id.
func ParseMoment(axis, asOf, runID string) (Moment, error) {
	m := Moment{Axis: AxisOccurred, RunID: runID}
	switch Axis(axis) {
	case "", AxisOccurred:
	case AxisRecorded:
		m.Axis = AxisRecorded
	default:
		return m, fmt.Errorf("axis = %q, допустимы occurred | recorded", axis)
	}
	if asOf != "" {
		t, err := time.Parse(time.RFC3339Nano, asOf)
		if err != nil {
			return m, fmt.Errorf("as_of: %w", err)
		}
		t = t.UTC()
		m.AsOf = &t
	}
	return m, nil
}

// Page — страница списка: курсор и размер (списки очередей, журналов, лент).
type Page struct {
	// Cursor — непрозрачный курсор следующей страницы; пусто — первая.
	Cursor string
	// Limit — размер страницы (1…500).
	Limit int
}
