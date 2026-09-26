package analytics

import (
	"context"
	"time"

	app "ant/internal/application/analytics"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "analytics"

// PeriodParams — параметры периода показателей (FR-3).
type PeriodParams struct {
	Period string `query:"period" enum:"shift,day,week,month,custom" doc:"Период: смена, сутки, неделя, месяц, произвольный (по умолчанию — смена)."`
	From   string `query:"from" format:"date-time" doc:"Начало произвольного периода (RFC 3339 UTC)."`
	To     string `query:"to" format:"date-time" doc:"Конец произвольного периода."`
}

func (p *PeriodParams) query() (app.PeriodQuery, error) {
	out := app.PeriodQuery{Kind: p.Period}
	for _, x := range []struct {
		s   string
		dst **time.Time
	}{{p.From, &out.From}, {p.To, &out.To}} {
		if x.s == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, x.s)
		if err != nil {
			return out, err
		}
		*x.dst = &t
	}
	return out, nil
}

// Register объявляет операции модуля analytics: плитки показателей, счётчики
// узлов живой карты с ограничением линии, полный набор показателей кейса,
// раскрытие до записей, контрольные карты (FR-3, FR-5, FR-86…FR-89; AD-21,
// AD-45). Команд нет: показатели — только проекции.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_ = c
	httpapi.Read(api, httpapi.Get("/metrics/tiles", "Плитки показателей",
		"Стол руководителя: проверено изделий, с подтверждёнными несоответствиями, прохождение контроля с первого раза, дефекты по видам, "+
			"причина установлена / не установлена, время детали в системе, ожидание. «Оценка невозможна» ≠ ноль (NFR-UI-4)."),
		platform.Action{ID: "analytics.tile.list", Owner: owner, Subject: "live_map"},
		func(ctx context.Context, in *struct {
			PeriodParams
			httpapi.MomentQuery
		}, m platform.Moment) (app.MetricTileList, error) {
			p, err := in.query()
			if err != nil {
				return app.MetricTileList{}, err
			}
			return q.Tiles(ctx, p, m)
		})

	httpapi.Read(api, httpapi.Get("/metrics/node-counters", "Счётчики узлов живой карты",
		"FR-2, FR-3, FR-5, AD-21: по step_key — очередь, в работе, прошло, дефекты за период; узел-ограничение линии и аномалии считает сервер; "+
			"узлы без данных источника — «оценка невозможна», не «норма»."),
		platform.Action{ID: "analytics.node_counters.read", Owner: owner, Subject: "live_map"},
		func(ctx context.Context, in *struct {
			ProcessVersionID string `query:"process_version_id" maxLength:"128" doc:"Версия процесса; пусто — действующая."`
			PeriodParams
			httpapi.MomentQuery
		}, m platform.Moment) (app.NodeCounterSet, error) {
			p, err := in.query()
			if err != nil {
				return app.NodeCounterSet{}, err
			}
			return q.NodeCounters(ctx, in.ProcessVersionID, p, m)
		})

	httpapi.Read(api, httpapi.Get("/analytics", "Аналитика: полный набор показателей",
		"FR-86…FR-89, кейс §2.4, §5.2: раздельный учёт (входной брак отличим от производственных ошибок), дефекты и изделия с дефектами раздельно, "+
			"сравнение сопоставимых работ, происхождение времени (передано источником / вычислено системой)."),
		platform.Action{ID: "analytics.overview.read", Owner: owner, Subject: "live_map"},
		func(ctx context.Context, in *struct {
			PeriodParams
			httpapi.MomentQuery
		}, m platform.Moment) (app.AnalyticsOverview, error) {
			p, err := in.query()
			if err != nil {
				return app.AnalyticsOverview{}, err
			}
			return q.Overview(ctx, p, m)
		})

	httpapi.Read(api, httpapi.Get("/analytics/metrics/{metric_id}/contributions", "Раскрытие показателя",
		"AD-45, соглашение «Показатели»: число раскрывается до строк вклада изделий и id исходных записей журнала."),
		platform.Action{ID: "analytics.metric.drilldown", Owner: owner, Subject: "live_map"},
		func(ctx context.Context, in *struct {
			MetricID string `path:"metric_id" maxLength:"64" doc:"Показатель."`
			Slice    string `query:"slice" maxLength:"128" doc:"Ключ среза; пусто — итог."`
			PeriodParams
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.MetricDrilldown, error) {
			p, err := in.query()
			if err != nil {
				return app.MetricDrilldown{}, err
			}
			return q.Drilldown(ctx, in.MetricID, in.Slice, p, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/analytics/control-charts/{step_key}", "Контрольная карта узла",
		"FR-5: доля дефектов или характеристика узла во времени с центральной линией и границами; выход за границы — аномалия узла."),
		platform.Action{ID: "analytics.control_chart.read", Owner: owner, Subject: "live_map"},
		func(ctx context.Context, in *struct {
			StepKey  string `path:"step_key" maxLength:"128" doc:"Шаг процесса."`
			MetricID string `query:"metric_id" maxLength:"64" doc:"Показатель; пусто — доля дефектов."`
			PeriodParams
			httpapi.MomentQuery
		}, m platform.Moment) (app.ControlChart, error) {
			p, err := in.query()
			if err != nil {
				return app.ControlChart{}, err
			}
			return q.ControlChart(ctx, in.StepKey, in.MetricID, p, m)
		})
}
