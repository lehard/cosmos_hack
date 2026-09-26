package access

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
	"ant/internal/domain/kernel"
)

// Управление политикой (эпик 26; AD-11, AD-15, AD-39, FR-78, FR-85,
// FR-145): права меняются только записями policy.* журнала; выдача, которая
// расширяет права, — по маршруту подписей документа «Выдача ролей,
// полномочий, клейм» со второй подписью независимой стороны; никто не
// выдаёт права сам себе. Отзыв — защитное действие одной подписью и доходит
// до всех копий api (проекция по LISTEN/NOTIFY + проверка policy_seq в
// journal.Append).

// WithGrantDocuments подключает документы выдачи прав (маршрут подписей documents).
func WithGrantDocuments(g GrantDocuments) Option { return func(s *Service) { s.grants = g } }

// livePolicy — управление политикой подключено (журнал, проекция, часы).
func (s *Service) livePolicy() bool { return s.decisions != nil && s.policy != nil && s.now != nil }

// scopePattern — путь области (схемы policy.*).
var scopePattern = regexp.MustCompile(`^[a-z0-9_-]+(/[a-z0-9_-]+)*$`)

// GrantAssessment — чья подпись нужна для выдачи (access.grant.assess):
// сфера, вторая подпись независимой стороны и этапы документа выдачи
// (RequiredApprovals) — для кнопки «Запросить решение» (FR-146).
type GrantAssessment struct {
	Domain          string   `json:"domain" enum:"ordinary,qc,production,admin" doc:"Сфера выдачи (AD-11): обычная, ОТК, производство, администраторы и аудит."`
	SecondAuthority string   `json:"second_authority,omitempty" doc:"Полномочие второй подписи; пусто — одной подписью администратора безопасности."`
	SecondSigner    string   `json:"second_signer" doc:"Кто ставит вторую подпись (по-русски)."`
	SelfGrant       bool     `json:"self_grant" doc:"Выдача себе: изменение расширяет права инициатора (по эффективным правам)."`
	AdminExpansion  bool     `json:"admin_expansion" doc:"Расширяются права администратора или аудита — привилегированная выдача, кто бы её ни делал."`
	Expands         []string `json:"expands" doc:"Права, которые добавляет выдача: action:шаблон@область, authority:id@область, stamp:вид@область."`
	Reason          string   `json:"reason"`
	// Документ выдачи для «Запросить решение».
	Template  string               `json:"template,omitempty" doc:"Шаблон документа «Выдача ролей, полномочий, клейм» (шаблон@версия)."`
	Decision  string               `json:"decision" doc:"Решение документа (grant:‹вид›:‹что›:‹кому›:‹область›) — поле decision запроса решения."`
	Stages    []GrantApprovalStage `json:"stages" doc:"Этапы подписей документа выдачи (RequiredApprovals, AD-43)."`
	Subject   platform.DrillRef    `json:"subject" doc:"Объект документа — сотрудник."`
	PolicySeq int64                `json:"policy_seq"`
}

// GrantApprovalStage — этап документа выдачи и кто может его подписать.
type GrantApprovalStage struct {
	Stage       int      `json:"stage"`
	Title       string   `json:"title"`
	AuthorityID string   `json:"authority_id"`
	BySource    bool     `json:"by_source" doc:"Этап закрывается самим запросом (инициатор)."`
	Candidates  []string `json:"candidates" doc:"Кто может подписать (не инициатор и не получатель)."`
}

// GrantQuery — что оценить (access.grant.assess).
type GrantQuery struct {
	PersonID       string
	Kind           string
	SubjectID      string
	InspectionKind string
	Scope          string
}

// change — изменение политики из формы выдачи с проверкой по действующей политике.
func (s *Service) change(pol accessdom.Policy, q GrantQuery, from time.Time, until *time.Time, orderRef string) (accessdom.PolicyChange, error) {
	if !personPattern.MatchString(q.PersonID) {
		return accessdom.PolicyChange{}, platform.Fail(errcodes.ApiValidationFailed, "field", "person_id", "reason", "псевдоним сотрудника")
	}
	if _, ok := pol.Person(q.PersonID); !ok {
		return accessdom.PolicyChange{}, platform.Fail(errcodes.ApiNotFound, "object", "сотрудник", "id", q.PersonID)
	}
	scope := strings.Trim(q.Scope, "/")
	if scope == "" {
		scope = pol.Root
	}
	if !scopePattern.MatchString(scope) {
		return accessdom.PolicyChange{}, platform.Fail(errcodes.ApiValidationFailed, "field", "scope", "reason", "путь области: здание/цех/участок/рабочее место")
	}
	c := accessdom.PolicyChange{Kind: q.Kind, PersonID: q.PersonID, SubjectID: q.SubjectID, InspectionKind: q.InspectionKind, Scope: scope,
		OrderRef: orderRef, ValidFrom: from}
	if until != nil {
		c.ValidUntil = *until
		if !c.ValidUntil.After(from) {
			return c, platform.Fail(errcodes.ApiValidationFailed, "field", "valid_until", "reason", "позже valid_from")
		}
	}
	switch q.Kind {
	case accessdom.ChangeRole:
		if _, ok := pol.Role(q.SubjectID); !ok {
			return c, platform.Fail(errcodes.ApiNotFound, "object", "роль", "id", q.SubjectID)
		}
	case accessdom.ChangeAuthority:
		if len(pol.Catalog.Authorities) > 0 {
			if _, ok := pol.Catalog.Authority(q.SubjectID); !ok {
				return c, platform.Fail(errcodes.ApiNotFound, "object", "полномочие", "id", q.SubjectID)
			}
		}
	case accessdom.ChangeStamp:
		if q.InspectionKind == "" || (len(pol.Catalog.StampKinds) > 0 && !slices.Contains(pol.Catalog.StampKinds, q.InspectionKind)) {
			return c, platform.Fail(errcodes.ApiValidationFailed, "field", "inspection_kind", "reason", "вид контроля из stamp_kinds политики")
		}
	default:
		return c, platform.Fail(errcodes.ApiValidationFailed, "field", "kind", "reason", "role | authority | stamp")
	}
	if q.SubjectID == "" {
		return c, platform.Fail(errcodes.ApiValidationFailed, "field", "subject", "reason", "роль, полномочие или id клейма")
	}
	return c, nil
}

// AssessGrant — чья подпись нужна для выдачи (access.grant.assess, AD-11,
// AD-43): оценка по эффективным правам и этапы документа выдачи.
func (s *Service) AssessGrant(ctx context.Context, q GrantQuery) (GrantAssessment, error) {
	if !s.livePolicy() {
		return s.Queries.AssessGrant(ctx, q)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return GrantAssessment{}, err
	}
	now, err := s.now(ctx)
	if err != nil {
		return GrantAssessment{}, err
	}
	c, err := s.change(pol, q, now, nil, "")
	if err != nil {
		return GrantAssessment{}, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	a := accessdom.Assess(pol, c, actor, now)
	out := GrantAssessment{Domain: a.Domain, SecondAuthority: a.SecondAuthority, SecondSigner: accessdom.DomainTitle(a.Domain),
		SelfGrant: a.SelfGrant, AdminExpansion: a.AdminExpansion, Expands: []string{}, Reason: a.Reason,
		Template: pol.Catalog.GrantTemplate, Decision: accessdom.FormatGrantDecision(c), Stages: []GrantApprovalStage{},
		Subject: platform.DrillRef{Entity: "person", ID: c.PersonID}, PolicySeq: pol.Seq}
	for _, r := range a.Expands {
		out.Expands = append(out.Expands, r.Kind+":"+r.Value+"@"+r.Scope)
	}
	for _, st := range accessdom.RequiredApprovals(pol.Catalog.GrantRoute, accessdom.ApprovalContext{Grant: &c, GrantDomain: a.Domain,
		Initiator: actor, At: now}, pol) {
		cand := st.Candidates
		if cand == nil {
			cand = []string{}
		}
		out.Stages = append(out.Stages, GrantApprovalStage{Stage: st.Stage, Title: st.Title, AuthorityID: st.AuthorityID, BySource: st.BySource, Candidates: cand})
	}
	return out, nil
}

// GrantPolicy — выдать роль, полномочие или клеймо (access.policy.grant):
// обычная выдача — одной подписью; расширение прав, требующее второй
// подписи (сфера ОТК, производства, администраторов и аудита; выдача себе;
// расширение прав администратора), — только по закрытому маршруту документа
// выдачи, где засчитана подпись независимой стороны с полномочием второй
// подписи на момент её подписи (AD-11, AD-43).
func (s *Service) GrantPolicy(ctx context.Context, in GrantPolicy) (platform.Receipt, error) {
	if !s.livePolicy() {
		return s.Commands.GrantPolicy(ctx, in)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	subject := in.RoleID
	switch in.Kind {
	case accessdom.ChangeAuthority:
		subject = in.AuthorityID
	case accessdom.ChangeStamp:
		subject = in.StampID
		if strings.TrimSpace(in.OrderRef) == "" {
			return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "order_ref", "reason", "клеймо выдаётся по приказу (FR-145)")
		}
	}
	from := in.ValidFrom
	if from.IsZero() {
		from = now
	}
	c, err := s.change(pol, GrantQuery{PersonID: in.PersonID, Kind: in.Kind, SubjectID: subject, InspectionKind: in.InspectionKind, Scope: in.Scope},
		from, in.ValidUntil, in.OrderRef)
	if err != nil {
		return platform.Receipt{}, err
	}
	if c.Kind == accessdom.ChangeStamp {
		// FR-145: одно клеймо на вид контроля.
		if st, ok := pol.StampFor(c.PersonID, c.InspectionKind, "", from); ok {
			return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "inspection_kind",
				"reason", "у сотрудника уже есть клеймо "+st.StampID+" по этому виду контроля — сначала отзовите его")
		}
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	a := accessdom.Assess(pol, c, actor, now)
	second := ""
	if a.SecondAuthority != "" {
		if second, err = s.secondSignature(ctx, pol, a, c, in.DocumentID); err != nil {
			return platform.Receipt{}, err
		}
	}
	rec := grantRecord(c, in, second)
	rc, err := s.decisions.Write(ctx, Batch{Records: []Record{rec}, Meta: in.CommandMeta(), Actor: actor, OccurredAt: now})
	s.refresh(ctx)
	return rc, err
}

// secondSignature — гард второй подписи: документ выдачи закрыт, выдаёт
// именно это изменение, и в нём засчитана подпись независимой стороны с
// полномочием второй подписи (CheckApprovals). Возвращает, кто её поставил.
func (s *Service) secondSignature(ctx context.Context, pol accessdom.Policy, a accessdom.Assessment, c accessdom.PolicyChange, documentID string) (string, error) {
	need := func(detail string) error {
		err := accessdom.CheckApprovals(pol, a, c, nil)
		pe := platform.Fail(refusalCode(err), "who", accessdom.DomainTitle(a.Domain)+" (полномочие "+a.SecondAuthority+")")
		pe.Detail = a.Reason + ". " + detail
		pe.AllowedActions = slices.Clone(RequestDecisionActions)
		return pe
	}
	if documentID == "" {
		return "", need("Запросите решение: документ «Выдача ролей, полномочий, клейм» с маршрутом подписей, затем выполните выдачу с его document_id.")
	}
	if s.grants == nil {
		return "", need("Маршрут подписей документов не подключён — выдача со второй подписью невозможна.")
	}
	doc, err := s.grants.GrantDocument(ctx, documentID)
	if err != nil {
		return "", err
	}
	if !doc.Closed {
		return "", need("Маршрут документа " + documentID + " ещё не закрыт.")
	}
	if doc.Decision != accessdom.FormatGrantDecision(c) {
		return "", platform.Fail(errcodes.ApiValidationFailed, "field", "document_id", "reason", "документ выдаёт другое: "+doc.Decision)
	}
	if err := accessdom.CheckApprovals(pol, a, c, doc.Approvals); err != nil {
		return "", err
	}
	for _, ap := range doc.Approvals {
		if !slices.Contains(a.Excluded, ap.PersonID) && pol.HasAuthority(ap.PersonID, a.SecondAuthority, c.Scope, ap.At) {
			return ap.PersonID, nil
		}
	}
	return "", need("")
}

// refusalCode — код отказа доменного гарда.
func refusalCode(err error) errcodes.Code {
	if r, ok := err.(*kernel.Refusal); ok {
		return r.Code
	}
	return errcodes.AccessSignatureRequired
}

// grantRecord — запись policy.* выдачи в поток области (catalog streams: `policy:‹область›`).
func grantRecord(c accessdom.PolicyChange, in GrantPolicy, second string) Record {
	stream := accessdom.PolicyStream(c.Scope)
	var doc *ev.ObjectID
	if in.DocumentID != "" {
		d := ev.ObjectID(in.DocumentID)
		doc = &d
	}
	var by *ev.PersonRef
	if second != "" {
		b := ev.PersonRef(second)
		by = &b
	}
	var until *ev.Timestamp
	if !c.ValidUntil.IsZero() {
		u := ev.Timestamp(c.ValidUntil)
		until = &u
	}
	switch c.Kind {
	case accessdom.ChangeAuthority:
		d := ev.PolicyAuthorityGrantedV1{PersonID: ev.PersonRef(c.PersonID), AuthorityID: ev.ObjectID(c.SubjectID), Scope: c.Scope,
			ValidFrom: ev.Timestamp(c.ValidFrom), ValidUntil: until, DocumentID: doc, SecondSignatureBy: by}
		if c.OrderRef != "" {
			o := c.OrderRef
			d.OrderRef = &o
		}
		return Record{Type: catalog.PolicyAuthorityGranted, Stream: stream, Data: d}
	case accessdom.ChangeStamp:
		return Record{Type: catalog.PolicyStampIssued, Stream: stream, Data: ev.PolicyStampIssuedV1{PersonID: ev.PersonRef(c.PersonID),
			StampID: ev.ObjectID(c.SubjectID), InspectionKind: ev.Code(c.InspectionKind), Scope: c.Scope, OrderRef: c.OrderRef,
			ValidFrom: ev.Timestamp(c.ValidFrom), ValidUntil: until, DocumentID: doc, SecondSignatureBy: by}}
	}
	return Record{Type: catalog.PolicyRoleAssigned, Stream: stream, Data: ev.PolicyRoleAssignedV1{PersonID: ev.PersonRef(c.PersonID),
		RoleID: ev.ObjectID(c.SubjectID), Scope: c.Scope, ValidFrom: ev.Timestamp(c.ValidFrom), ValidUntil: until, DocumentID: doc, SecondSignatureBy: by}}
}

// RevokePolicy — отозвать роль, полномочие или клеймо (access.policy.revoke):
// защитное действие одной подписью (AD-27). Запись — в поток области
// отзываемого (поэтому она попадает в проверку policy_seq команд субъекта,
// AD-39); без области — отзыв во всех областях, где право действует.
func (s *Service) RevokePolicy(ctx context.Context, in RevokePolicy) (platform.Receipt, error) {
	if !s.livePolicy() {
		return s.Commands.RevokePolicy(ctx, in)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if strings.TrimSpace(in.Reason.Text) == "" {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "reason", "reason", "основание отзыва")
	}
	at := in.EffectiveFrom
	if at.IsZero() {
		at = now
	}
	reason := ev.Reason{Text: ev.Text(in.Reason.Text)}
	if in.Reason.Code != "" {
		code := ev.Code(in.Reason.Code)
		reason.Code = &code
	}
	scope := strings.Trim(in.Scope, "/")
	var recs []Record
	seen := map[string]bool{}
	switch in.Kind {
	case accessdom.ChangeRole:
		for _, a := range pol.AssignmentsOf(in.PersonID, at) {
			if a.RoleID == in.SubjectID && (scope == "" || a.Scope == scope) && !seen[a.Scope] {
				seen[a.Scope] = true
				recs = append(recs, Record{Type: catalog.PolicyRoleUnassigned, Stream: accessdom.PolicyStream(a.Scope), Data: ev.PolicyRoleUnassignedV1{
					PersonID: ev.PersonRef(in.PersonID), RoleID: ev.ObjectID(in.SubjectID), Scope: a.Scope, EffectiveFrom: ev.Timestamp(at), Reason: &reason}})
			}
		}
	case accessdom.ChangeAuthority:
		for _, a := range pol.AuthoritiesOf(in.PersonID, at) {
			if a.AuthorityID == in.SubjectID && (scope == "" || a.Scope == scope) && !seen[a.Scope] {
				seen[a.Scope] = true
				recs = append(recs, Record{Type: catalog.PolicyAuthorityRevoked, Stream: accessdom.PolicyStream(a.Scope), Data: ev.PolicyAuthorityRevokedV1{
					PersonID: ev.PersonRef(in.PersonID), AuthorityID: ev.ObjectID(in.SubjectID), Scope: a.Scope, EffectiveFrom: ev.Timestamp(at), Reason: &reason}})
			}
		}
	case accessdom.ChangeStamp:
		for _, st := range pol.StampsOf(in.PersonID, at) {
			if st.StampID == in.SubjectID && !seen[st.Scope] {
				seen[st.Scope] = true
				recs = append(recs, Record{Type: catalog.PolicyStampRevoked, Stream: accessdom.PolicyStream(st.Scope), Data: ev.PolicyStampRevokedV1{
					PersonID: ev.PersonRef(in.PersonID), StampID: ev.ObjectID(in.SubjectID), EffectiveFrom: ev.Timestamp(at), Reason: reason}})
			}
		}
	default:
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "kind", "reason", "role | authority | stamp")
	}
	if len(recs) == 0 {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", in.Kind, "id", in.SubjectID+" у "+in.PersonID)
	}
	rc, err := s.decisions.Write(ctx, Batch{Records: recs, Meta: in.CommandMeta(), Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now})
	s.refresh(ctx)
	return rc, err
}

// digestPattern — отпечаток ключа хранителя (defs.digest).
var digestPattern = regexp.MustCompile(`^streebog256:[0-9a-f]{64}$`)

// SetAuditParameters — параметры аудита (access.audit.set_parameters,
// policy.audit.parameters_set): данные журнала, принадлежат Аудитору ИБ
// (право — роль security_auditor в политике; AD-8, AD-15). Хранитель и
// проверка контрольных точек берут интервал и предельный разрыв отсюда.
func (s *Service) SetAuditParameters(ctx context.Context, in SetAuditParameters) (platform.Receipt, error) {
	if !s.livePolicy() {
		return s.Commands.SetAuditParameters(ctx, in)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if in.CheckpointMaxGapS < in.CheckpointIntervalS {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "checkpoint_max_gap_s", "reason", "не меньше интервала контрольных точек")
	}
	fp := in.KeeperKeyFingerprint
	if fp == "" {
		fp = pol.Audit.KeeperKeyFingerprint
	}
	if !digestPattern.MatchString(fp) {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "keeper_key_fingerprint", "reason", "streebog256:‹64 hex›")
	}
	d := ev.PolicyAuditParametersSetV1{CheckpointIntervalS: in.CheckpointIntervalS, CheckpointMaxGapS: in.CheckpointMaxGapS, KeeperKeyFingerprint: ev.Digest(fp)}
	for _, t := range in.CriticalTypes {
		if _, ok := catalog.Lookup(catalog.Type(t)); !ok {
			return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "critical_types", "reason", "тип не из каталога: "+t)
		}
		d.CriticalTypes = append(d.CriticalTypes, ev.EventType(t))
	}
	for _, x := range in.SecurityBusSubscribers {
		d.SecurityBusSubscribers = append(d.SecurityBusSubscribers, ev.Code(x))
	}
	rc, err := s.decisions.Write(ctx, Batch{Records: []Record{{Type: catalog.PolicyAuditParametersSet, Stream: accessdom.PolicyStream(pol.Root), Data: d}},
		Meta: in.CommandMeta(), Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now})
	s.refresh(ctx)
	return rc, err
}

// Audit — действующие параметры аудита (access.audit.read).
func (s *Service) Audit(ctx context.Context, m platform.Moment) (AccessAuditParameters, error) {
	if s.policy == nil {
		return s.Queries.Audit(ctx, m)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return AccessAuditParameters{}, err
	}
	a := pol.Audit
	out := AccessAuditParameters{CriticalTypes: slices.Clone(a.CriticalTypes), CheckpointIntervalS: a.CheckpointIntervalS, CheckpointMaxGapS: a.CheckpointMaxGapS,
		KeeperKeyFingerprint: a.KeeperKeyFingerprint, SecurityBusSubscribers: slices.Clone(a.SecurityBusSubscribers), SetBy: s.personOf(pol, a.SetBy), PolicySeq: pol.Seq}
	if out.CriticalTypes == nil {
		out.CriticalTypes = []string{}
	}
	if out.SecurityBusSubscribers == nil {
		out.SecurityBusSubscribers = []string{}
	}
	if !a.SetAt.IsZero() {
		t := a.SetAt
		out.SetAt = &t
	}
	return out, nil
}

// personOf — псевдоним сотрудника по подписанту записи (`‹псевдоним›@версия` в нижнем регистре).
func (s *Service) personOf(pol accessdom.Policy, actor string) string {
	id, _, _ := strings.Cut(actor, "@")
	for _, x := range pol.Persons {
		if strings.EqualFold(x.ID, id) {
			return x.ID
		}
	}
	return id
}

// Stamps — цифровые клейма (access.stamp.list, FR-145): действующие, истёкшие, отозванные.
func (s *Service) Stamps(ctx context.Context, personID string, m platform.Moment) (AccessStampList, error) {
	if s.policy == nil {
		return s.Queries.Stamps(ctx, personID, m)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return AccessStampList{}, err
	}
	at := s.at(ctx, m)
	out := AccessStampList{Items: []AccessStamp{}}
	for _, st := range pol.Stamps {
		if personID != "" && st.PersonID != personID {
			continue
		}
		v := AccessStamp{StampID: st.StampID, PersonID: st.PersonID, InspectionKind: st.Kind, Scope: st.Scope, OrderRef: st.OrderRef,
			ValidFrom: st.ValidFrom, ValidUntil: until(st.ValidUntil), Status: "active"}
		switch {
		case st.Revoked:
			v.Status = "revoked"
		case !accessdom.ValidAt(at, st.ValidFrom, st.ValidUntil):
			v.Status = "expired"
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}

// grantKinds — тип записи → вид и действие строки истории выдачи прав.
var grantKinds = map[catalog.Type][2]string{
	catalog.PolicyRoleAssigned:       {"role", "granted"},
	catalog.PolicyRoleUnassigned:     {"role", "revoked"},
	catalog.PolicyAuthorityGranted:   {"authority", "granted"},
	catalog.PolicyAuthorityRevoked:   {"authority", "revoked"},
	catalog.PolicyStampIssued:        {"stamp", "granted"},
	catalog.PolicyStampRevoked:       {"stamp", "revoked"},
	catalog.PolicyAuditParametersSet: {"audit", "set"},
	catalog.AccessAccountActivated:   {"account", "granted"},
}

// Grants — история выдачи и отзыва прав (access.grant.list; журнал выдачи
// прав у Аудитора ИБ, AD-15): кто, кому, что, в какой области, со второй
// подписью и документом выдачи. Новые записи — первыми; курсор — seq.
func (s *Service) Grants(ctx context.Context, personID string, m platform.Moment, pg platform.Page) (AccessGrantHistory, error) {
	src, ok := s.policy.(interface {
		Records(ctx context.Context) ([]accessdom.Record, error)
	})
	if !ok {
		return s.Queries.Grants(ctx, personID, m, pg)
	}
	recs, err := src.Records(ctx)
	if err != nil {
		return AccessGrantHistory{}, err
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return AccessGrantHistory{}, err
	}
	before := int64(0)
	if pg.Cursor != "" {
		if before, err = strconv.ParseInt(pg.Cursor, 10, 64); err != nil {
			return AccessGrantHistory{}, platform.Fail(errcodes.ApiValidationFailed, "field", "cursor", "reason", "курсор — seq")
		}
	}
	limit := pg.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	out := AccessGrantHistory{Items: []AccessGrantEntry{}}
	for i := len(recs) - 1; i >= 0; i-- {
		r := recs[i]
		if before > 0 && r.Seq >= before {
			continue
		}
		if m.AsOf != nil && r.OccurredAt.After(*m.AsOf) {
			continue
		}
		e, ok := grantEntry(r)
		if !ok || (personID != "" && e.PersonID != personID) {
			continue
		}
		e.By = s.personOf(pol, r.Actor)
		if r.Genesis {
			e.By = "genesis"
		}
		if len(out.Items) == limit {
			out.NextCursor = strconv.FormatInt(out.Items[len(out.Items)-1].Seq, 10)
			break
		}
		out.Items = append(out.Items, e)
	}
	return out, nil
}

// grantEntry — строка истории по записи политики.
func grantEntry(r accessdom.Record) (AccessGrantEntry, bool) {
	k, ok := grantKinds[catalog.Type(r.Type)]
	if !ok {
		return AccessGrantEntry{}, false
	}
	var d struct {
		PersonID          string `json:"person_id"`
		RoleID            string `json:"role_id"`
		AuthorityID       string `json:"authority_id"`
		StampID           string `json:"stamp_id"`
		Login             string `json:"login"`
		Scope             string `json:"scope"`
		DocumentID        string `json:"document_id"`
		SecondSignatureBy string `json:"second_signature_by"`
	}
	if err := jsonUnmarshal(r.Data, &d); err != nil {
		return AccessGrantEntry{}, false
	}
	subject := d.RoleID + d.AuthorityID + d.StampID + d.Login
	if k[0] == "audit" {
		subject = "policy.audit"
	}
	return AccessGrantEntry{EventID: r.EventID, Seq: r.Seq, Kind: k[0], Action: k[1], PersonID: d.PersonID, SubjectID: subject, Scope: d.Scope,
		At: r.OccurredAt, SecondSignatureBy: d.SecondSignatureBy, DocumentID: d.DocumentID}, true
}

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
