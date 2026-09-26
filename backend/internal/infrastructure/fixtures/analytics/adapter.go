package analytics

import (
	"context"
	"time"

	app "ant/internal/application/analytics"
	"ant/internal/application/platform"
)

// Adapter — реализация fixtures ведущих портов модуля analytics (AD-36):
// плитки, счётчики узлов, показатели кейса, раскрытие, контрольные карты — из
// мира заготовок на шаге курсора. Команд у модуля нет.
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// period — параметры периода: period, from, to (RFC 3339 UTC).
func period(p app.PeriodQuery, kv ...string) map[string]string {
	out := map[string]string{"period": p.Kind, "from": ts(p.From), "to": ts(p.To)}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i]] = kv[i+1]
	}
	return out
}

func ts(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// Tiles — плитки показателей (analytics.tile.list).
func (Adapter) Tiles(ctx context.Context, p app.PeriodQuery, m platform.Moment) (app.MetricTileList, error) {
	return respond[app.MetricTileList](ctx, "analytics.tile.list", period(p), &m)
}

// NodeCounters — счётчики узлов живой карты (analytics.node_counters.read).
func (Adapter) NodeCounters(ctx context.Context, processVersionID string, p app.PeriodQuery, m platform.Moment) (app.NodeCounterSet, error) {
	return respond[app.NodeCounterSet](ctx, "analytics.node_counters.read", period(p, "process_version_id", processVersionID), &m)
}

// Overview — показатели кейса (analytics.overview.read).
func (Adapter) Overview(ctx context.Context, p app.PeriodQuery, m platform.Moment) (app.AnalyticsOverview, error) {
	return respond[app.AnalyticsOverview](ctx, "analytics.overview.read", period(p), &m)
}

// Drilldown — раскрытие показателя до записей (analytics.metric.drilldown).
func (Adapter) Drilldown(ctx context.Context, metricID, sliceKey string, p app.PeriodQuery, m platform.Moment, _ platform.Page) (app.MetricDrilldown, error) {
	return respond[app.MetricDrilldown](ctx, "analytics.metric.drilldown", period(p, "metric_id", metricID, "slice", sliceKey), &m)
}

// ControlChart — контрольная карта узла (analytics.control_chart.read).
func (Adapter) ControlChart(ctx context.Context, stepKey, metricID string, p app.PeriodQuery, m platform.Moment) (app.ControlChart, error) {
	return respond[app.ControlChart](ctx, "analytics.control_chart.read", period(p, "step_key", stepKey, "metric_id", metricID), &m)
}
