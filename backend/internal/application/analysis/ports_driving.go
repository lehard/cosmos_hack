package analysis

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля analysis (AD-36).
type Queries interface {
	// Circumstances — разбор обстоятельств по несоответствию (analysis.circumstances.read, FR-58, FR-153).
	Circumstances(ctx context.Context, ncID string, m platform.Moment) (Circumstances, error)
	// Hypotheses — гипотезы по несоответствию с похожими случаями (analysis.hypothesis.list, FR-59, FR-60).
	Hypotheses(ctx context.Context, ncID string, m platform.Moment) (Hypotheses, error)
	// Similar — похожие случаи (analysis.similar.list, FR-60).
	Similar(ctx context.Context, ncID string, m platform.Moment) (SimilarCaseList, error)
	// Groups — группы несоответствий (analysis.group.list).
	Groups(ctx context.Context, m platform.Moment) (NcGroupList, error)
	// CommonFactors — общие факторы группы (analysis.common_factors.read, FR-135).
	CommonFactors(ctx context.Context, groupKey string, m platform.Moment) (CommonFactors, error)
	// Incidents — инциденты (analysis.incident.list).
	Incidents(ctx context.Context, m platform.Moment, p platform.Page) (IncidentList, error)
	// RiskScope — область риска с версиями (analysis.risk_scope.read, FR-61, FR-62).
	RiskScope(ctx context.Context, incidentID string, m platform.Moment) (RiskScope, error)
}

// Commands — ведущий порт команд модуля analysis (AD-39).
type Commands interface {
	RecordHypothesis(ctx context.Context, ncID string, in RecordHypothesis) (platform.Receipt, error)
	ConcludeCause(ctx context.Context, incidentID string, in ConcludeCause) (platform.Receipt, error)
	RejectHypothesis(ctx context.Context, ncID string, in RejectHypothesis) (platform.Receipt, error)
	RequestMeasurement(ctx context.Context, ncID string, in RequestMeasurement) (platform.Receipt, error)
	NarrowScope(ctx context.Context, incidentID string, in ChangeScope) (platform.Receipt, error)
	ExpandScope(ctx context.Context, incidentID string, in ChangeScope) (platform.Receipt, error)
	AssessItem(ctx context.Context, incidentID string, in AssessItem) (platform.Receipt, error)
	ScopeAnalysis(ctx context.Context, incidentID string, in ScopeAnalysis) (platform.Receipt, error)
	CloseIncident(ctx context.Context, incidentID string, in CloseIncident) (platform.Receipt, error)
	AssignAction(ctx context.Context, incidentID string, in AssignAction) (platform.Receipt, error)
	ImplementAction(ctx context.Context, incidentID, actionID string, in ImplementAction) (platform.Receipt, error)
	EvaluateAction(ctx context.Context, incidentID, actionID string, in EvaluateAction) (platform.Receipt, error)
}

// Unimplemented — заглушка портов analysis: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Circumstances(context.Context, string, platform.Moment) (Circumstances, error) {
	return Circumstances{}, ni("analysis.circumstances.read")
}
func (Unimplemented) Hypotheses(context.Context, string, platform.Moment) (Hypotheses, error) {
	return Hypotheses{}, ni("analysis.hypothesis.list")
}
func (Unimplemented) Similar(context.Context, string, platform.Moment) (SimilarCaseList, error) {
	return SimilarCaseList{}, ni("analysis.similar.list")
}
func (Unimplemented) Groups(context.Context, platform.Moment) (NcGroupList, error) {
	return NcGroupList{}, ni("analysis.group.list")
}
func (Unimplemented) CommonFactors(context.Context, string, platform.Moment) (CommonFactors, error) {
	return CommonFactors{}, ni("analysis.common_factors.read")
}
func (Unimplemented) Incidents(context.Context, platform.Moment, platform.Page) (IncidentList, error) {
	return IncidentList{}, ni("analysis.incident.list")
}
func (Unimplemented) RiskScope(context.Context, string, platform.Moment) (RiskScope, error) {
	return RiskScope{}, ni("analysis.risk_scope.read")
}
func (Unimplemented) RecordHypothesis(context.Context, string, RecordHypothesis) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.hypothesis.record")
}
func (Unimplemented) ConcludeCause(context.Context, string, ConcludeCause) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.cause.conclude")
}
func (Unimplemented) RejectHypothesis(context.Context, string, RejectHypothesis) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.hypothesis.reject")
}
func (Unimplemented) RequestMeasurement(context.Context, string, RequestMeasurement) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.measurement.request")
}
func (Unimplemented) NarrowScope(context.Context, string, ChangeScope) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.scope.narrow")
}
func (Unimplemented) ExpandScope(context.Context, string, ChangeScope) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.scope.expand")
}
func (Unimplemented) AssessItem(context.Context, string, AssessItem) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.item.assess")
}
func (Unimplemented) ScopeAnalysis(context.Context, string, ScopeAnalysis) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.analysis.scope")
}
func (Unimplemented) CloseIncident(context.Context, string, CloseIncident) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.incident.close")
}
func (Unimplemented) AssignAction(context.Context, string, AssignAction) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.action.assign")
}
func (Unimplemented) ImplementAction(context.Context, string, string, ImplementAction) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.action.implement")
}
func (Unimplemented) EvaluateAction(context.Context, string, string, EvaluateAction) (platform.Receipt, error) {
	return platform.Receipt{}, ni("analysis.action.evaluate")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
