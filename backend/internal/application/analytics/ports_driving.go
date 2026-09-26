package analytics

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля analytics (AD-36, AD-45).
type Queries interface {
	// Tiles — плитки показателей стола руководителя (analytics.tile.list).
	Tiles(ctx context.Context, p PeriodQuery, m platform.Moment) (MetricTileList, error)
	// NodeCounters — счётчики узлов, ограничение линии, аномалии (analytics.node_counters.read, FR-2, FR-3, FR-5).
	NodeCounters(ctx context.Context, processVersionID string, p PeriodQuery, m platform.Moment) (NodeCounterSet, error)
	// Overview — полный набор показателей (analytics.overview.read, FR-86…FR-89).
	Overview(ctx context.Context, p PeriodQuery, m platform.Moment) (AnalyticsOverview, error)
	// Drilldown — раскрытие показателя до вкладов и записей (analytics.metric.drilldown, AD-45).
	Drilldown(ctx context.Context, metricID, sliceKey string, p PeriodQuery, m platform.Moment, pg platform.Page) (MetricDrilldown, error)
	// ControlChart — контрольная карта узла (analytics.control_chart.read).
	ControlChart(ctx context.Context, stepKey, metricID string, p PeriodQuery, m platform.Moment) (ControlChart, error)
}

// Commands — ведущий порт команд модуля analytics: своих типов записей модуль
// не эмитит (каталог), команд нет.
type Commands interface{}

// Unimplemented — заглушка портов analytics: каждая операция отвечает 501.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Tiles(context.Context, PeriodQuery, platform.Moment) (MetricTileList, error) {
	return MetricTileList{}, ni("analytics.tile.list")
}
func (Unimplemented) NodeCounters(context.Context, string, PeriodQuery, platform.Moment) (NodeCounterSet, error) {
	return NodeCounterSet{}, ni("analytics.node_counters.read")
}
func (Unimplemented) Overview(context.Context, PeriodQuery, platform.Moment) (AnalyticsOverview, error) {
	return AnalyticsOverview{}, ni("analytics.overview.read")
}
func (Unimplemented) Drilldown(context.Context, string, string, PeriodQuery, platform.Moment, platform.Page) (MetricDrilldown, error) {
	return MetricDrilldown{}, ni("analytics.metric.drilldown")
}
func (Unimplemented) ControlChart(context.Context, string, string, PeriodQuery, platform.Moment) (ControlChart, error) {
	return ControlChart{}, ni("analytics.control_chart.read")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
