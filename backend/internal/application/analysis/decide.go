package analysis

import (
	"context"
	"strings"
	"uuid"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/analysis"
)

// Команды модуля analysis в режиме live (AD-39): гард — чистая доменная
// функция над состоянием инцидента на basis_seq (проекция — кэш её входа),
// затем решение человека одной записью журнала с проверкой потока инцидента.
// Классы (AD-27): сузить и исключить — разрешающее; расширить — защитное;
// причина — необратимое; остальное — запись.

// decide — записать решение над инцидентом (поток incident:‹id›, гард проверил его).
func (s *Service) decide(ctx context.Context, t catalog.Type, incidentID string, meta platform.CommandMeta, level int, data any) (platform.Receipt, error) {
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	stream := "incident:" + incidentID
	return s.cfg.Decisions.Write(ctx, Decision{Type: t, Stream: stream, Data: data, Meta: meta, Actor: platform.PrincipalFrom(ctx).PersonID,
		OccurredAt: now, GuardStreams: []string{stream}, SignatureLevel: level})
}

func refusal(err error) error {
	if err == nil {
		return nil
	}
	if pe, ok := platform.AsError(err); ok {
		return pe
	}
	return err
}

func reasonOf(r Reason) ev.Reason {
	x := ev.Reason{Text: ev.Text(r.Text)}
	if r.Code != nil && *r.Code != "" {
		c := ev.Code(*r.Code)
		x.Code = &c
	}
	return x
}

// ncIncident — инцидент несоответствия: из решений людей по нему, иначе из
// статуса его изделия в инцидентах (адресованные записи стадии).
func (s *Service) ncIncident(ctx context.Context, n dom.NCRecord) (dom.IncidentRecord, error) {
	ids := append([]string{}, n.IncidentIDs...)
	iv, err := s.item(ctx, n.ItemID)
	if err != nil {
		return dom.IncidentRecord{}, err
	}
	for _, m := range iv.State.Incidents {
		ids = append(ids, m.IncidentID)
	}
	for _, id := range ids {
		v, err := s.incident(ctx, id)
		if err == nil && !v.Closed {
			return v, nil
		}
	}
	for _, id := range ids {
		if v, err := s.incident(ctx, id); err == nil {
			return v, nil
		}
	}
	e := platform.Fail(errcodes.ApiValidationFailed, "field", "nc_id", "reason", "несоответствие не отнесено к инциденту")
	e.Detail = "Несоответствие " + n.NCID + " не отнесено ни к одному инциденту"
	return dom.IncidentRecord{}, e
}

// RecordHypothesis — гипотеза причины человеком (incident.hypothesis.recorded, FR-59).
func (s *Service) RecordHypothesis(ctx context.Context, ncID string, in RecordHypothesis) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RecordHypothesis(ctx, ncID, in)
	}
	if _, err := s.nc(ctx, ncID); err != nil {
		return platform.Receipt{}, err
	}
	v, err := s.incident(ctx, in.IncidentID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := refusal(dom.GuardOpen(v)); err != nil {
		return platform.Receipt{}, err
	}
	id := "HYP-" + ncID + "-" + shortID(in.CommandID)
	data := map[string]any{"incident_id": in.IncidentID, "nc_ids": []string{ncID}, "branch": in.Branch, "category": in.Category,
		"statement": in.Statement, "hypothesis_id": id, "verdict": "proposed"}
	if len(in.SupportingEventIDs) > 0 {
		data["supporting_event_ids"] = in.SupportingEventIDs
	}
	return s.decide(ctx, catalog.IncidentHypothesisRecorded, in.IncidentID, in.CommandMeta(), 0, data)
}

// RejectHypothesis — гипотеза отклонена с основанием (incident.hypothesis.recorded,
// verdict = rejected): вывод системы не переписывается.
func (s *Service) RejectHypothesis(ctx context.Context, ncID string, in RejectHypothesis) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RejectHypothesis(ctx, ncID, in)
	}
	n, err := s.nc(ctx, ncID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if strings.TrimSpace(in.Reason.Text) == "" {
		return platform.Receipt{}, refusal(platform.Fail(errcodes.IncidentBasisRequired))
	}
	v, err := s.ncIncident(ctx, n)
	if err != nil {
		return platform.Receipt{}, err
	}
	category, branch, statement, err := s.hypothesisOf(ctx, n, in.HypothesisID)
	if err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"incident_id": v.IncidentID, "nc_ids": []string{ncID}, "branch": branch, "category": category,
		"statement": "Отклонена: " + statement, "hypothesis_id": in.HypothesisID, "verdict": "rejected", "reason": reasonOf(in.Reason)}
	return s.decide(ctx, catalog.IncidentHypothesisRecorded, v.IncidentID, in.CommandMeta(), 0, data)
}

// hypothesisOf — категория, ветка и формулировка гипотезы по id (вывод системы или запись человека).
func (s *Service) hypothesisOf(ctx context.Context, n dom.NCRecord, id string) (category, branch, statement string, err error) {
	for _, h := range n.Hypotheses {
		if h.HypothesisID == id {
			return h.Category, h.Branch, h.Statement, nil
		}
	}
	a, _, err := s.analyze(ctx, n)
	if err != nil {
		return "", "", "", err
	}
	for _, h := range a.Hypotheses {
		if h.ID == id {
			return h.Category, "why_made", h.Statement, nil
		}
	}
	return "", "", "", notFound("Гипотеза", id)
}

// RequestMeasurement — запросить измерение (incident.measurement.requested):
// задачу исполнителю ставит notifications по этой записи.
func (s *Service) RequestMeasurement(ctx context.Context, ncID string, in RequestMeasurement) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RequestMeasurement(ctx, ncID, in)
	}
	n, err := s.nc(ctx, ncID)
	if err != nil {
		return platform.Receipt{}, err
	}
	v, err := s.ncIncident(ctx, n)
	if err != nil {
		return platform.Receipt{}, err
	}
	if _, _, _, err := s.hypothesisOf(ctx, n, in.HypothesisID); err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"incident_id": v.IncidentID, "nc_ids": []string{ncID}, "hypothesis_id": in.HypothesisID, "what": in.What}
	if in.AssigneeID != "" {
		data["assignee_id"] = in.AssigneeID
	}
	return s.decide(ctx, catalog.IncidentMeasurementRequested, v.IncidentID, in.CommandMeta(), 0, data)
}

// ConcludeCause — вывод о причине (incident.cause.concluded, FR-59): только
// человек; ошибка исполнителя — только после письменного объяснения.
func (s *Service) ConcludeCause(ctx context.Context, incidentID string, in ConcludeCause) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ConcludeCause(ctx, incidentID, in)
	}
	v, err := s.incident(ctx, incidentID)
	if err != nil {
		return platform.Receipt{}, err
	}
	category := in.Category
	if category == "" && in.HypothesisID != "" && len(in.NCIDs) > 0 {
		if n, err := s.nc(ctx, in.NCIDs[0]); err == nil {
			category, _, _, _ = s.hypothesisOf(ctx, n, in.HypothesisID)
		}
	}
	if in.Conclusion == "not_established" {
		category = dom.CatNotEstablished
	}
	if err := refusal(dom.GuardConclude(v, in.Conclusion, category, in.Verification)); err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"incident_id": incidentID, "nc_ids": in.NCIDs, "conclusion": in.Conclusion, "verification": in.Verification,
		"reason": reasonOf(in.Reason)}
	if category != "" {
		data["category"] = category
	}
	return s.decide(ctx, catalog.IncidentCauseConcluded, incidentID, in.CommandMeta(), 2, data)
}

// NarrowScope — сузить область (incident.scope.narrowed, FR-61): только
// человек и только с основанием; стадия запишет новую версию.
func (s *Service) NarrowScope(ctx context.Context, incidentID string, in ChangeScope) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.NarrowScope(ctx, incidentID, in)
	}
	v, err := s.incident(ctx, incidentID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := refusal(dom.GuardNarrow(v, in.ItemIDs, in.EvidenceEventIDs, in.Reason.Text)); err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"incident_id": incidentID, "item_ids": in.ItemIDs, "evidence_event_ids": in.EvidenceEventIDs,
		"release_containment": in.ReleaseContainment, "reason": reasonOf(in.Reason)}
	return s.decide(ctx, catalog.IncidentScopeNarrowed, incidentID, in.CommandMeta(), 2, data)
}

// ExpandScope — расширить область человеком (incident.scope.expanded, защитное).
func (s *Service) ExpandScope(ctx context.Context, incidentID string, in ChangeScope) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ExpandScope(ctx, incidentID, in)
	}
	v, err := s.incident(ctx, incidentID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := refusal(dom.GuardExpand(v, in.ItemIDs, in.Reason.Text)); err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"incident_id": incidentID, "item_ids": in.ItemIDs, "reason": reasonOf(in.Reason)}
	return s.decide(ctx, catalog.IncidentScopeExpanded, incidentID, in.CommandMeta(), 0, data)
}

// AssessItem — оценка изделия в инциденте (incident.item.assessed, FR-62).
func (s *Service) AssessItem(ctx context.Context, incidentID string, in AssessItem) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.AssessItem(ctx, incidentID, in)
	}
	v, err := s.incident(ctx, incidentID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := refusal(dom.GuardAssess(v, in.ItemID, in.EvidenceEventIDs)); err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"incident_id": incidentID, "item_id": in.ItemID, "assessment": in.Assessment, "evidence_event_ids": in.EvidenceEventIDs}
	return s.decide(ctx, catalog.IncidentItemAssessed, incidentID, in.CommandMeta(), 2, data)
}

// ScopeAnalysis — решение о глубине разбора (incident.analysis.scoped).
func (s *Service) ScopeAnalysis(ctx context.Context, incidentID string, in ScopeAnalysis) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ScopeAnalysis(ctx, incidentID, in)
	}
	v, err := s.incident(ctx, incidentID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := refusal(dom.GuardOpen(v)); err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"incident_id": incidentID, "full_analysis": in.FullAnalysis, "reason": reasonOf(in.Reason)}
	return s.decide(ctx, catalog.IncidentAnalysisScoped, incidentID, in.CommandMeta(), 0, data)
}

// CloseIncident — закрыть инцидент (incident.incident.closed): итог — исходный
// размер области, подтверждено, исключено.
func (s *Service) CloseIncident(ctx context.Context, incidentID string, in CloseIncident) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.CloseIncident(ctx, incidentID, in)
	}
	v, err := s.incident(ctx, incidentID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := refusal(dom.GuardOpen(v)); err != nil {
		return platform.Receipt{}, err
	}
	confirmed, excluded := 0, 0
	for _, m := range v.Members {
		switch m.Status {
		case dom.StatusConfirmed:
			confirmed++
		case dom.StatusExcluded:
			excluded++
		}
	}
	data := map[string]any{"incident_id": incidentID, "initial_size": v.InitialSize, "confirmed": confirmed, "excluded": excluded}
	if in.Summary != "" {
		data["summary"] = in.Summary
	}
	return s.decide(ctx, catalog.IncidentIncidentClosed, incidentID, in.CommandMeta(), 0, data)
}

// AssignAction — назначить меру (incident.action.assigned, FR-64).
func (s *Service) AssignAction(ctx context.Context, incidentID string, in AssignAction) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.AssignAction(ctx, incidentID, in)
	}
	if _, err := s.incident(ctx, incidentID); err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"incident_id": incidentID, "action_id": "ACT-" + shortID(in.CommandID), "action_type": in.ActionType,
		"direction": in.Direction, "owner_id": in.OwnerID, "effectiveness_plan": in.EffectivenessPlan}
	if in.DueAt != "" {
		data["due_at"] = in.DueAt
	}
	return s.decide(ctx, catalog.IncidentActionAssigned, incidentID, in.CommandMeta(), 0, data)
}

// ImplementAction — мера выполнена (incident.action.implemented).
func (s *Service) ImplementAction(ctx context.Context, incidentID, actionID string, in ImplementAction) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ImplementAction(ctx, incidentID, actionID, in)
	}
	if err := s.actionExists(ctx, incidentID, actionID); err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"incident_id": incidentID, "action_id": actionID}
	if in.Note != "" {
		data["note"] = in.Note
	}
	return s.decide(ctx, catalog.IncidentActionImplemented, incidentID, in.CommandMeta(), 0, data)
}

// EvaluateAction — результативность меры (incident.action.evaluated).
func (s *Service) EvaluateAction(ctx context.Context, incidentID, actionID string, in EvaluateAction) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.EvaluateAction(ctx, incidentID, actionID, in)
	}
	if err := s.actionExists(ctx, incidentID, actionID); err != nil {
		return platform.Receipt{}, err
	}
	data := map[string]any{"incident_id": incidentID, "action_id": actionID, "result": in.Result}
	if in.Evidence != "" {
		data["evidence"] = in.Evidence
	}
	return s.decide(ctx, catalog.IncidentActionEvaluated, incidentID, in.CommandMeta(), 0, data)
}

func (s *Service) actionExists(ctx context.Context, incidentID, actionID string) error {
	v, err := s.incident(ctx, incidentID)
	if err != nil {
		return err
	}
	for _, a := range v.Actions {
		if a.ActionID == actionID {
			return nil
		}
	}
	return notFound("Мера", actionID)
}

// shortID — короткий суффикс идентификатора объекта из command_id (или новый).
func shortID(commandID string) string {
	id := strings.ReplaceAll(strings.ToLower(commandID), "-", "")
	if len(id) < 12 {
		id = strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	}
	return strings.ToUpper(id[len(id)-12:])
}
