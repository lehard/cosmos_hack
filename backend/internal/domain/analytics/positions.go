package analytics

import "time"

// Position — где изделие сейчас по процессу (process State.Positions,
// эпик 17): шаг и вид — очередь (RowQueue) или работа (RowInProgress) — с
// момента Since.
type Position struct {
	Step  string
	Kind  string
	Since time.Time
}

// ApplyPositions — «очередь» и «в работе» изделия сейчас — по его положению
// в процессе (стык эпиков 17 и 25, эпик 16; FR-2, FR-5): открытые (без Until)
// интервалы queue и in_progress, выведенные из фактов, заменяются положением
// токенов. Строка из фактов с тем же видом и шагом остаётся (у неё полнее
// измерения и источники раскрытия); закрытые интервалы — история — не
// трогаются. Чистая функция (AD-4).
func ApplyPositions(itemID string, rows []Row, ps []Position) []Row {
	type key struct{ kind, step string }
	want := map[key]bool{}
	for _, p := range ps {
		if p.Step != "" && (p.Kind == RowQueue || p.Kind == RowInProgress) {
			want[key{p.Kind, p.Step}] = true
		}
	}
	base := Dims{Label: label(itemID)}
	have := map[key]bool{}
	out := make([]Row, 0, len(rows)+len(ps))
	for i, r := range rows {
		if i == 0 {
			base.Run, base.ItemType, base.Label = r.Dims.Run, r.Dims.ItemType, r.Dims.Label
		}
		if (r.Metric == RowQueue || r.Metric == RowInProgress) && r.Interval && r.Until == nil {
			k := key{r.Metric, r.Dims.Step}
			if !want[k] || have[k] {
				continue
			}
			have[k] = true
		}
		out = append(out, r)
	}
	for _, p := range ps {
		k := key{p.Kind, p.Step}
		if !want[k] || have[k] {
			continue
		}
		have[k] = true
		d := base
		d.Step = p.Step
		if p.Kind == RowQueue {
			d.Meaning, d.DurationOrigin = MeaningOther, OriginSystem
		}
		out = append(out, Row{Metric: p.Kind, At: p.Since, Interval: true, Value: 1, Unit: UnitPcs, Dims: d, Sources: []string{}, Kinds: []string{}})
	}
	SortRows(out)
	return out
}
