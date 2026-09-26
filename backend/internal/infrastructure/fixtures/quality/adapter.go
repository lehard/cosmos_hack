package quality

import (
	"context"

	"ant/internal/application/platform"
	app "ant/internal/application/quality"
)

// Adapter — реализация fixtures ведущих портов модуля quality (AD-36):
// сигналы, результаты контроля, полнота, дефекты, пропуски брака и карта
// реакций — из мира заготовок. Команд у модуля нет.
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

func itemP(id string) map[string]string { return map[string]string{"item_id": id} }

// Signals — сигналы о признаке дефекта (quality.signal.list) с фильтром.
func (Adapter) Signals(ctx context.Context, f app.SignalFilter, m platform.Moment, _ platform.Page) (app.QualitySignalList, error) {
	v, err := respond[app.QualitySignalList](ctx, "quality.signal.list", map[string]string{"item_id": f.ItemID, "state": f.State}, &m)
	if err != nil {
		return v, err
	}
	out := v.Items[:0]
	for _, x := range v.Items {
		if (f.ItemID == "" || x.ItemID == f.ItemID) && (f.State == "" || x.State == f.State) {
			out = append(out, x)
		}
	}
	v.Items = out
	return v, nil
}

// Signal — сигнал с исходным наблюдением (quality.signal.read, FR-38).
func (Adapter) Signal(ctx context.Context, signalID string, m platform.Moment) (app.QualitySignal, error) {
	return respond[app.QualitySignal](ctx, "quality.signal.read", map[string]string{"signal_id": signalID}, &m)
}

// Inspections — результаты контроля изделия (quality.inspection.list, FR-36).
func (Adapter) Inspections(ctx context.Context, itemID string, m platform.Moment, _ platform.Page) (app.InspectionResultList, error) {
	return respond[app.InspectionResultList](ctx, "quality.inspection.list", itemP(itemID), &m)
}

// Coverage — полнота контроля изделия (quality.coverage.read, FR-35).
func (Adapter) Coverage(ctx context.Context, itemID string, m platform.Moment) (app.InspectionCoverage, error) {
	return respond[app.InspectionCoverage](ctx, "quality.coverage.read", itemP(itemID), &m)
}

// Defects — дефекты (quality.defect.list, FR-37): с фильтром по изделию
// счётчики пересчитываются по оставшимся строкам.
func (Adapter) Defects(ctx context.Context, itemID string, m platform.Moment, _ platform.Page) (app.QualityDefectList, error) {
	v, err := respond[app.QualityDefectList](ctx, "quality.defect.list", itemP(itemID), &m)
	if err != nil || itemID == "" {
		return v, err
	}
	out := v.Items[:0]
	for _, x := range v.Items {
		if x.ItemID == itemID {
			out = append(out, x)
		}
	}
	v.Items = out
	v.DefectCount = len(out)
	v.ItemsWithDefect = 0
	if len(out) > 0 {
		v.ItemsWithDefect = 1
	}
	return v, nil
}

// Escapes — пропуски брака (quality.escape.list).
func (Adapter) Escapes(ctx context.Context, m platform.Moment, _ platform.Page) (app.QualityEscapeList, error) {
	return respond[app.QualityEscapeList](ctx, "quality.escape.list", nil, &m)
}

// ReactionMap — действующая карта реакций (quality.reaction_map.read, FR-48).
func (Adapter) ReactionMap(ctx context.Context, m platform.Moment) (app.ReactionMap, error) {
	return respond[app.ReactionMap](ctx, "quality.reaction_map.read", nil, &m)
}
