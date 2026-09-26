package ops

import (
	"context"
	"strings"

	dom "ant/internal/domain/ops"
)

// Показатели ops в /metrics (FR-113): операционные, не проекции (AD-7);
// снимаются перед выдачей /metrics из того же, что видит стол администратора.
const (
	MetricStoppedItems   = "ant_ops_stopped_items"
	MetricConsumerLag    = "ant_ops_consumer_lag_seq"
	MetricQueuePending   = "ant_ops_queue_pending"
	MetricOutboxQuar     = "ant_ops_outbox_quarantined"
	MetricIntegrationBad = "ant_ops_integration_degraded"
	MetricComponentUp    = "ant_ops_component_up"
	MetricQuarantineOpen = "ant_ops_quarantine_open"
)

// Collect обновляет показатели ops в телеметрии (вызывается перед выдачей
// /metrics). Без телеметрии или вне live — ничего.
func (s *Service) Collect(ctx context.Context) error {
	t := s.cfg.Telemetry
	if !s.live || t == nil {
		return nil
	}
	h, err := s.Health(ctx)
	if err != nil {
		return err
	}
	t.Gauge(MetricStoppedItems, int64(h.StoppedItems))
	t.Gauge(MetricQuarantineOpen, int64(h.QuarantineOpen))
	for _, q := range h.Queues {
		consumer, part, _ := strings.Cut(q.Name, "/")
		if q.Scope != "partition" {
			part = "global"
		}
		t.Gauge(MetricConsumerLag, q.LagSeq, "consumer", consumer, "partition", part)
		t.Gauge(MetricQueuePending, q.Pending, "consumer", consumer, "partition", part)
		if q.Quarantined != nil {
			t.Gauge(MetricOutboxQuar, *q.Quarantined, "queue", q.Name)
		}
	}
	for _, it := range h.Integrations {
		v := int64(0)
		if it.State == dom.IntegrationDegraded {
			v = 1
		}
		t.Gauge(MetricIntegrationBad, v, "system", it.System)
	}
	for _, c := range h.Components {
		v := int64(0)
		if c.State == dom.StateOK {
			v = 1
		}
		t.Gauge(MetricComponentUp, v, "component", c.Component)
	}
	return nil
}
