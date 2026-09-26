package quality

import (
	"context"

	"ant/internal/application/platform"
)

// SignalFilter — фильтр списка сигналов.
type SignalFilter struct {
	ItemID string
	State  string
}

// Queries — ведущий порт чтения модуля quality (AD-36).
type Queries interface {
	// Signals — сигналы о признаке дефекта (quality.signal.list).
	Signals(ctx context.Context, f SignalFilter, m platform.Moment, p platform.Page) (QualitySignalList, error)
	// Signal — исходный сигнал с наблюдением (quality.signal.read, FR-38).
	Signal(ctx context.Context, signalID string, m platform.Moment) (QualitySignal, error)
	// Inspections — результаты контроля изделия (quality.inspection.list, FR-36).
	Inspections(ctx context.Context, itemID string, m platform.Moment, p platform.Page) (InspectionResultList, error)
	// Coverage — полнота контроля изделия (quality.coverage.read, FR-35).
	Coverage(ctx context.Context, itemID string, m platform.Moment) (InspectionCoverage, error)
	// Defects — дефекты (quality.defect.list, FR-37).
	Defects(ctx context.Context, itemID string, m platform.Moment, p platform.Page) (QualityDefectList, error)
	// Escapes — пропуски брака (quality.escape.list).
	Escapes(ctx context.Context, m platform.Moment, p platform.Page) (QualityEscapeList, error)
	// ReactionMap — действующая карта реакций (quality.reaction_map.read, FR-48).
	ReactionMap(ctx context.Context, m platform.Moment) (ReactionMap, error)
}

// Commands — ведущий порт команд модуля quality. Команд нет: у семейств
// quality и inspection нет записей-решений — результаты контроля приходят
// фактами через приём (ingest), подтверждение контролёра — решение
// nonconformity (AD-30).
type Commands interface{}

// Unimplemented — заглушка портов quality: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Signals(context.Context, SignalFilter, platform.Moment, platform.Page) (QualitySignalList, error) {
	return QualitySignalList{}, ni("quality.signal.list")
}
func (Unimplemented) Signal(context.Context, string, platform.Moment) (QualitySignal, error) {
	return QualitySignal{}, ni("quality.signal.read")
}
func (Unimplemented) Inspections(context.Context, string, platform.Moment, platform.Page) (InspectionResultList, error) {
	return InspectionResultList{}, ni("quality.inspection.list")
}
func (Unimplemented) Coverage(context.Context, string, platform.Moment) (InspectionCoverage, error) {
	return InspectionCoverage{}, ni("quality.coverage.read")
}
func (Unimplemented) Defects(context.Context, string, platform.Moment, platform.Page) (QualityDefectList, error) {
	return QualityDefectList{}, ni("quality.defect.list")
}
func (Unimplemented) Escapes(context.Context, platform.Moment, platform.Page) (QualityEscapeList, error) {
	return QualityEscapeList{}, ni("quality.escape.list")
}
func (Unimplemented) ReactionMap(context.Context, platform.Moment) (ReactionMap, error) {
	return ReactionMap{}, ni("quality.reaction_map.read")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
