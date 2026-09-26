package analysis

import (
	"context"

	app "ant/internal/application/analysis"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// Adapter — реализация fixtures ведущих портов модуля analysis (AD-36):
// разбор обстоятельств, гипотезы, похожие случаи, группы и общие факторы,
// инциденты и области риска — из мира заготовок; команды двигают сценарий,
// если он ждёт именно этого решения (сужение области, причина).
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

func nc(id string) map[string]string       { return map[string]string{"nc_id": id} }
func incident(id string) map[string]string { return map[string]string{"incident_id": id} }

// Circumstances — разбор обстоятельств (analysis.circumstances.read).
func (Adapter) Circumstances(ctx context.Context, ncID string, m platform.Moment) (app.Circumstances, error) {
	return respond[app.Circumstances](ctx, "analysis.circumstances.read", nc(ncID), &m)
}

// Hypotheses — гипотезы по несоответствию (analysis.hypothesis.list).
func (Adapter) Hypotheses(ctx context.Context, ncID string, m platform.Moment) (app.Hypotheses, error) {
	h, err := respond[app.Hypotheses](ctx, "analysis.hypothesis.list", nc(ncID), &m)
	if err != nil {
		return h, err
	}
	return withHypotheses(ctx, h, m), nil
}

// Similar — похожие случаи (analysis.similar.list).
func (Adapter) Similar(ctx context.Context, ncID string, m platform.Moment) (app.SimilarCaseList, error) {
	return respond[app.SimilarCaseList](ctx, "analysis.similar.list", nc(ncID), &m)
}

// Groups — группы несоответствий (analysis.group.list).
func (Adapter) Groups(ctx context.Context, m platform.Moment) (app.NcGroupList, error) {
	return respond[app.NcGroupList](ctx, "analysis.group.list", nil, &m)
}

// CommonFactors — общие факторы группы (analysis.common_factors.read).
func (Adapter) CommonFactors(ctx context.Context, groupKey string, m platform.Moment) (app.CommonFactors, error) {
	return respond[app.CommonFactors](ctx, "analysis.common_factors.read", map[string]string{"group_key": groupKey}, &m)
}

// Incidents — инциденты (analysis.incident.list); пагинации на заготовках нет.
func (Adapter) Incidents(ctx context.Context, m platform.Moment, _ platform.Page) (app.IncidentList, error) {
	l, err := respond[app.IncidentList](ctx, "analysis.incident.list", nil, &m)
	if err != nil {
		return l, err
	}
	return withIncidents(ctx, l, m), nil
}

// RiskScope — область риска инцидента (analysis.risk_scope.read).
func (Adapter) RiskScope(ctx context.Context, incidentID string, m platform.Moment) (app.RiskScope, error) {
	rs, err := respond[app.RiskScope](ctx, "analysis.risk_scope.read", incident(incidentID), &m)
	if err != nil {
		return rs, err
	}
	return withScope(ctx, rs, m), nil
}

// RecordHypothesis — записать гипотезу (analysis.hypothesis.record).
func (Adapter) RecordHypothesis(ctx context.Context, ncID string, in app.RecordHypothesis) (platform.Receipt, error) {
	return recordNC(ctx, "analysis.hypothesis.record", ncID, in.CommandMeta(), in)
}

// ConcludeCause — подтвердить причину (analysis.cause.conclude).
func (Adapter) ConcludeCause(ctx context.Context, incidentID string, in app.ConcludeCause) (platform.Receipt, error) {
	return record(ctx, "analysis.cause.conclude", incidentID, in.CommandMeta(), in)
}

// RejectHypothesis — отклонить гипотезу (analysis.hypothesis.reject).
func (Adapter) RejectHypothesis(ctx context.Context, ncID string, in app.RejectHypothesis) (platform.Receipt, error) {
	return recordNC(ctx, "analysis.hypothesis.reject", ncID, in.CommandMeta(), in)
}

// RequestMeasurement — запросить измерение (analysis.measurement.request).
func (Adapter) RequestMeasurement(ctx context.Context, ncID string, in app.RequestMeasurement) (platform.Receipt, error) {
	return recordNC(ctx, "analysis.measurement.request", ncID, in.CommandMeta(), in)
}

// NarrowScope — сузить область риска (analysis.scope.narrow).
func (Adapter) NarrowScope(ctx context.Context, incidentID string, in app.ChangeScope) (platform.Receipt, error) {
	return record(ctx, "analysis.scope.narrow", incidentID, in.CommandMeta(), scopeChange{Narrow: true, In: in})
}

// ExpandScope — расширить область риска (analysis.scope.expand).
func (Adapter) ExpandScope(ctx context.Context, incidentID string, in app.ChangeScope) (platform.Receipt, error) {
	return record(ctx, "analysis.scope.expand", incidentID, in.CommandMeta(), scopeChange{In: in})
}

// AssessItem — оценить изделие в инциденте (analysis.item.assess).
func (Adapter) AssessItem(ctx context.Context, incidentID string, in app.AssessItem) (platform.Receipt, error) {
	return decide(ctx, "analysis.item.assess", "incident", incidentID, in.CommandMeta())
}

// ScopeAnalysis — назначить разбор (analysis.analysis.scope).
func (Adapter) ScopeAnalysis(ctx context.Context, incidentID string, in app.ScopeAnalysis) (platform.Receipt, error) {
	return decide(ctx, "analysis.analysis.scope", "incident", incidentID, in.CommandMeta())
}

// CloseIncident — закрыть инцидент (analysis.incident.close). Закрыть
// расследование (scope=investigation) на заготовках нельзя, пока в шапке
// инцидента на шаге курсора есть блокеры (close_blockers) — отказ тем же
// кодом, что у live (422 incident.cause_branch_open, incident.effectiveness_unchecked).
func (Adapter) CloseIncident(ctx context.Context, incidentID string, in app.CloseIncident) (platform.Receipt, error) {
	if in.Scope == "investigation" {
		list, err := respond[app.IncidentList](ctx, "analysis.incident.list", nil, &platform.Moment{})
		if err != nil {
			return platform.Receipt{}, err
		}
		for _, x := range list.Items {
			if x.IncidentID == incidentID && len(x.CloseBlockers) > 0 {
				b := x.CloseBlockers[0]
				e := platform.Fail(errcodes.Code(b.Code), "incident_id", incidentID)
				e.Detail = b.Text
				return platform.Receipt{}, e
			}
		}
	}
	return record(ctx, "analysis.incident.close", incidentID, in.CommandMeta(), in)
}

// AssignAction — назначить меру (analysis.action.assign).
func (Adapter) AssignAction(ctx context.Context, incidentID string, in app.AssignAction) (platform.Receipt, error) {
	return record(ctx, "analysis.action.assign", incidentID, in.CommandMeta(), in)
}

// ImplementAction — мера выполнена (analysis.action.implement).
func (Adapter) ImplementAction(ctx context.Context, incidentID, _ string, in app.ImplementAction) (platform.Receipt, error) {
	return decide(ctx, "analysis.action.implement", "incident", incidentID, in.CommandMeta())
}

// EvaluateAction — оценить эффективность меры (analysis.action.evaluate).
func (Adapter) EvaluateAction(ctx context.Context, incidentID, _ string, in app.EvaluateAction) (platform.Receipt, error) {
	return decide(ctx, "analysis.action.evaluate", "incident", incidentID, in.CommandMeta())
}
