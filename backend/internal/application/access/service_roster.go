package access

import (
	"context"
	"slices"
	"strings"
	"time"

	itemapp "ant/internal/application/item"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
	"ant/internal/domain/kernel"
)

// Посты, назначения, квалификации и факты исполнителя (FR-6, FR-80, FR-81,
// FR-137, PRD §11.18; эпик 26 по просьбе дирижёра после эпика 13): живые
// операции над той же проекцией политики. СКУД, ключ и допуск к рабочему
// месту — эпик 37: присутствие без этих данных — «неизвестно», не «на месте».

// ControllerAssignmentTemplate — шаблон документа «Назначение контролёра на
// пост»: запрос мастера → согласование начальника ОТК (PRD §11.18).
const ControllerAssignmentTemplate = "controller-assignment"

// AuthControllerApproval — полномочие согласования назначения контролёра.
const AuthControllerApproval = "controller_assignment_approval"

// expiringWithin — квалификация «истекает»: до конца срока меньше этого.
const expiringWithin = 30 * 24 * time.Hour

// WithLiveRoster — модуль access в режиме live (AD-36): панель «Посты» из
// журнала и факты исполнителя в журнал через writer (запись фактов и
// решений ядра). В режиме fixtures панель и факты — у заготовок.
func WithLiveRoster(facts itemapp.Writer) Option {
	return func(s *Service) { s.live, s.facts = true, facts }
}

// Qualifications — квалификации (access.qualification.list, FR-80): действует,
// истекает (меньше 30 дней), истекла, отозвана — на момент чтения.
func (s *Service) Qualifications(ctx context.Context, personID string, m platform.Moment) (AccessQualificationList, error) {
	if s.policy == nil {
		return s.Queries.Qualifications(ctx, personID, m)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return AccessQualificationList{}, err
	}
	at := s.at(ctx, m)
	out := AccessQualificationList{Items: []AccessQualification{}}
	for _, q := range pol.Qualifications {
		if personID != "" && q.PersonID != personID {
			continue
		}
		v := AccessQualification{PersonID: q.PersonID, QualificationID: q.QualificationID, Scope: q.Scope, CertificateRef: q.CertificateRef,
			ValidFrom: q.ValidFrom, ValidUntil: until(q.ValidUntil), Status: "valid"}
		switch {
		case q.Revoked:
			v.Status = "revoked"
		case !accessdom.ValidAt(at, q.ValidFrom, q.ValidUntil):
			v.Status = "expired"
		case !q.ValidUntil.IsZero() && !at.IsZero() && q.ValidUntil.Sub(at) < expiringWithin:
			v.Status = "expiring"
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}

// Assignments — назначения на посты в смене (access.assignment.list, FR-81):
// смена пуста — последние назначения каждого поста; цех — location id WS-….
func (s *Service) Assignments(ctx context.Context, shiftID, workshop string, m platform.Moment) (AccessAssignmentList, error) {
	if s.policy == nil || s.dir == nil {
		return s.Queries.Assignments(ctx, shiftID, workshop, m)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return AccessAssignmentList{}, err
	}
	at := s.at(ctx, m)
	out := AccessAssignmentList{Items: []AccessAssignment{}, BasisSeq: pol.Seq}
	for _, a := range pol.PostsIn(shiftID) {
		wp, _ := s.dir.Workplace(a.WorkplaceID)
		if workshop != "" && wp.Workshop != workshop {
			continue
		}
		v := AccessAssignment{WorkplaceID: a.WorkplaceID, ShiftID: a.ShiftID, PersonID: a.PersonID, AssigneeRole: a.AssigneeRole,
			ApprovalDocumentID: a.ApprovalDocumentID, QualificationOK: true}
		if a.AssigneeRole == accessdom.AssigneePerformer {
			_, v.QualificationOK = pol.QualifiedAt(a.PersonID, wp.Scope, at)
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}

// Workplaces — панель «Посты» (access.workplace.list, FR-6, FR-81): пост —
// кто назначен — присутствие. Только в режиме live; присутствие без СКУД и
// ключа (эпик 37) — «неизвестно»; текущее изделие поста — у модуля process.
func (s *Service) Workplaces(ctx context.Context, workshop string, m platform.Moment) (PostList, error) {
	if !s.live || s.policy == nil || s.dir == nil || len(s.dir.Workplaces) == 0 {
		return s.Queries.Workplaces(ctx, workshop, m)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return PostList{}, err
	}
	posts := pol.PostsIn("")
	out := PostList{Items: []PostRow{}}
	for _, wp := range s.dir.Workplaces {
		if workshop != "" && wp.Workshop != workshop {
			continue
		}
		row := PostRow{WorkplaceID: wp.ID, Station: wp.Name, Workshop: wp.Workshop, Presence: "not_assigned"}
		var pick *accessdom.PostAssignment
		for i := range posts {
			a := &posts[i]
			if a.WorkplaceID == wp.ID && (pick == nil || (pick.AssigneeRole != accessdom.AssigneePerformer && a.AssigneeRole == accessdom.AssigneePerformer)) {
				pick = a
			}
		}
		if pick != nil {
			name := pick.PersonID
			if x, ok := pol.Person(pick.PersonID); ok && x.Name != "" {
				name = x.Name
			}
			row.Assigned, row.Presence = &PostPerson{PersonID: pick.PersonID, Display: name}, "unknown"
		}
		out.Items = append(out.Items, row)
	}
	slices.SortFunc(out.Items, func(a, b PostRow) int { return strings.Compare(a.WorkplaceID, b.WorkplaceID) })
	return out, nil
}

// SetAssignment — назначить на пост в смене (access.assignment.set, FR-81,
// PRD §11.18): гард GuardAssignment; контролёр — по закрытому документу
// «Назначение контролёра на пост» с подписью начальника ОТК.
func (s *Service) SetAssignment(ctx context.Context, in SetAssignment) (platform.Receipt, error) {
	if s.policy == nil || s.decisions == nil || s.now == nil || s.dir == nil {
		return s.Commands.SetAssignment(ctx, in)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	wp, ok := s.dir.Workplace(in.WorkplaceID)
	if !ok {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "пост", "id", in.WorkplaceID)
	}
	if strings.TrimSpace(in.ShiftID) == "" {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "shift_id", "reason", "смена")
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	rq := accessdom.AssignmentRequest{WorkplaceID: wp.ID, WorkplaceScope: wp.Scope, ShiftID: in.ShiftID, PersonID: in.PersonID,
		AssigneeRole: in.AssigneeRole, At: now}
	if in.AssigneeRole == accessdom.AssigneeInspector && in.ApprovalDocumentID != "" {
		if rq.ApprovalClosed, err = s.controllerApproved(ctx, pol, in, actor); err != nil {
			return platform.Receipt{}, err
		}
	}
	if err := accessdom.GuardAssignment(pol, rq); err != nil {
		return platform.Receipt{}, err
	}
	d := ev.AccessAssignmentSetV1{PersonID: ev.PersonRef(in.PersonID), WorkplaceID: ev.ObjectID(wp.ID), ShiftID: ev.ObjectID(in.ShiftID),
		AssigneeRole: ev.AccessAssignmentSetV1AssigneeRole(in.AssigneeRole)}
	if in.ApprovalDocumentID != "" {
		doc := ev.ObjectID(in.ApprovalDocumentID)
		d.ApprovalDocumentID = &doc
	}
	rc, err := s.decisions.Write(ctx, Batch{Records: []Record{{Type: catalog.AccessAssignmentSet, Stream: "workplace:" + wp.ID, Data: d}},
		Meta: in.CommandMeta(), Actor: actor, OccurredAt: now})
	s.refresh(ctx)
	return rc, err
}

// controllerApproved — документ согласования контролёра закрыт, согласует
// именно это назначение (пост, кто, смена) и подписан не автором запроса, а
// сотрудником с полномочием согласования на момент подписи (PRD §11.18).
func (s *Service) controllerApproved(ctx context.Context, pol accessdom.Policy, in SetAssignment, actor string) (bool, error) {
	if s.grants == nil {
		return false, nil
	}
	doc, err := s.grants.GrantDocument(ctx, in.ApprovalDocumentID)
	if err != nil {
		return false, err
	}
	tpl, _, _ := strings.Cut(doc.Template, "@")
	if !doc.Closed || tpl != ControllerAssignmentTemplate {
		return false, nil
	}
	if doc.Decision != accessdom.ControllerDecision(in.PersonID, in.ShiftID) || (doc.Subject != "" && doc.Subject != in.WorkplaceID) {
		return false, platform.Fail(errcodes.ApiValidationFailed, "field", "approval_document_id", "reason", "документ согласует другое назначение: "+doc.Decision)
	}
	for _, ap := range doc.Approvals {
		if ap.PersonID != in.PersonID && pol.HasAuthority(ap.PersonID, AuthControllerApproval, "", ap.At) {
			return true, nil
		}
	}
	return false, nil
}

// ClearAssignment — снять с поста (access.assignment.clear).
func (s *Service) ClearAssignment(ctx context.Context, in ClearAssignment) (platform.Receipt, error) {
	if s.policy == nil || s.decisions == nil || s.now == nil {
		return s.Commands.ClearAssignment(ctx, in)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if !slices.ContainsFunc(pol.Posts, func(a accessdom.PostAssignment) bool {
		return a.WorkplaceID == in.WorkplaceID && a.ShiftID == in.ShiftID && a.PersonID == in.PersonID
	}) {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "назначение", "id", in.PersonID+" · "+in.WorkplaceID+" · "+in.ShiftID)
	}
	d := ev.AccessAssignmentClearedV1{PersonID: ev.PersonRef(in.PersonID), WorkplaceID: ev.ObjectID(in.WorkplaceID), ShiftID: ev.ObjectID(in.ShiftID)}
	if in.Reason != nil && in.Reason.Text != "" {
		r := ev.Reason{Text: ev.Text(in.Reason.Text)}
		if in.Reason.Code != "" {
			c := ev.Code(in.Reason.Code)
			r.Code = &c
		}
		d.Reason = &r
	}
	rc, err := s.decisions.Write(ctx, Batch{Records: []Record{{Type: catalog.AccessAssignmentCleared, Stream: "workplace:" + in.WorkplaceID, Data: d}},
		Meta: in.CommandMeta(), Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now})
	s.refresh(ctx)
	return rc, err
}

// ReportDeviation — исполнитель сообщает об отклонении (operator.deviation.reported, FR-137).
func (s *Service) ReportDeviation(ctx context.Context, workplaceID string, in ReportDeviation) (platform.Receipt, error) {
	if !s.liveFacts() {
		return s.Commands.ReportDeviation(ctx, workplaceID, in)
	}
	if strings.TrimSpace(in.Description) == "" {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "description", "reason", "что произошло")
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	d := ev.OperatorDeviationReportedV1{OperatorID: ev.PersonRef(actor), Description: in.Description, WorkplaceID: oid(workplaceID),
		OperationRunID: oid(in.OperationRunID), ZoneID: oid(in.ZoneID)}
	return s.fact(ctx, workplaceID, in.ItemID, catalog.OperatorDeviationReported, d, in.CommandMeta())
}

// RequestInspection — исполнитель запрашивает контроль (operator.inspection.requested, FR-137).
func (s *Service) RequestInspection(ctx context.Context, workplaceID string, in RequestInspection) (platform.Receipt, error) {
	if !s.liveFacts() {
		return s.Commands.RequestInspection(ctx, workplaceID, in)
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	d := ev.OperatorInspectionRequestedV1{OperatorID: ev.PersonRef(actor), StepKey: ev.StepKey(in.StepKey), WorkplaceID: oid(workplaceID),
		OperationRunID: oid(in.OperationRunID)}
	return s.fact(ctx, workplaceID, in.ItemID, catalog.OperatorInspectionRequested, d, in.CommandMeta())
}

// ConfirmStep — исполнитель подтверждает шаг ТП (operator.step.confirmed, FR-137).
func (s *Service) ConfirmStep(ctx context.Context, workplaceID string, in ConfirmStep) (platform.Receipt, error) {
	if !s.liveFacts() {
		return s.Commands.ConfirmStep(ctx, workplaceID, in)
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	d := ev.OperatorStepConfirmedV1{OperatorID: ev.PersonRef(actor), StepKey: ev.StepKey(in.StepKey), WorkplaceID: oid(workplaceID),
		OperationRunID: oid(in.OperationRunID)}
	if in.TPStep != "" {
		t := in.TPStep
		d.TpStep = &t
	}
	return s.fact(ctx, workplaceID, in.ItemID, catalog.OperatorStepConfirmed, d, in.CommandMeta())
}

func (s *Service) liveFacts() bool {
	return s.live && s.facts != nil && s.policy != nil && s.now != nil
}

// fact — факт исполнителя (ручной ввод, FR-140) в поток изделия, а без
// изделия — в поток поста. Только со своего рабочего места (FR-137): место
// операции проверил декоратор (роль в области поста); если исполнитель
// назначен на посты, пост команды — один из них.
func (s *Service) fact(ctx context.Context, workplaceID, itemID string, t catalog.Type, data any, meta platform.CommandMeta) (platform.Receipt, error) {
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	var mine []string
	for _, a := range pol.PostsIn("") {
		if a.PersonID == actor {
			mine = append(mine, a.WorkplaceID)
		}
	}
	if len(mine) > 0 && !slices.Contains(mine, workplaceID) {
		return platform.Receipt{}, platform.Fail(errcodes.AccessWrongWorkplace, "workplace", strings.Join(mine, ", "))
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	stream := "workplace:" + workplaceID
	if itemID != "" {
		stream = "item:" + itemID
	}
	meta.WorkplaceID = workplaceID
	return s.facts.Write(ctx, kernel.Module("access"), []itemapp.Record{{Type: t, Stream: stream, ItemID: itemID, Data: data, Meta: meta,
		Actor: actor, OccurredAt: now, SignatureLevel: 1}})
}

func oid(s string) *ev.ObjectID {
	if s == "" {
		return nil
	}
	v := ev.ObjectID(s)
	return &v
}
