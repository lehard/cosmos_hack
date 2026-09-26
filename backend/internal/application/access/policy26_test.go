package access_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"ant/internal/application/access"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
	dj "ant/internal/domain/journal"
	signing "ant/internal/domain/signing"
)

// Эпик 26 на уровне приложения: выдача через документ со второй подписью,
// отзыв, параметры аудита, policy_seq команды в journal.Append, редкие
// подписанты и объяснение прав своим кодом.

var now26 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

func policy26() accessdom.Policy {
	route := []accessdom.RouteStage{
		{Stage: 1, Title: "Администратор безопасности", Role: "administrator", AuthorityID: "administrator", Quorum: "one", SignatureLevel: 2, BySource: true},
		{Stage: 2, Title: "Аудитор ИБ", AuthorityID: signing.AuthSecondAudit, Quorum: "one", SignatureLevel: 2, When: &accessdom.StageCondition{GrantDomains: []string{accessdom.DomainAdmin}}},
	}
	return accessdom.FromSeed(accessdom.Seed{Root: "ent01",
		Roles: []accessdom.Role{
			{ID: "employee", Title: "Сотрудник", Actions: []string{"access.permission.explain"}},
			{ID: "staff", Title: "Работник", Inherits: []string{"employee"}, Actions: []string{"quality.*.read", "item.*.read"}},
			{ID: "technologist", Title: "Технолог", Inherits: []string{"staff"}, Actions: []string{"analysis.cause.conclude"}},
			{ID: "administrator", Title: "Администратор", Inherits: []string{"staff"}, Actions: []string{"access.policy.grant", "access.policy.revoke"}},
			{ID: "security_auditor", Title: "Аудитор ИБ", Inherits: []string{"staff"}, Actions: []string{"access.audit.set_parameters", "documents.signature.*"}},
			{ID: "customer_representative", Title: "Представитель заказчика", Inherits: []string{"employee"},
				Actions: []string{"documents.decision_card.read", "documents.signature.*", "item.passport.read"}},
		},
		Persons: []accessdom.SeedPerson{
			{ID: "ADM-01", Roles: []accessdom.SeedGrant{{Role: "administrator", Scope: "ent01"}}},
			{ID: "AUD-01", Roles: []accessdom.SeedGrant{{Role: "security_auditor", Scope: "ent01"}}},
			{ID: "TEC-01", Roles: []accessdom.SeedGrant{{Role: "technologist", Scope: "ent01"}}},
			{ID: "CR-71", Roles: []accessdom.SeedGrant{{Role: "customer_representative", Scope: "ent01"}}},
			{ID: "W21"},
		},
		Authorities: []accessdom.Authority{{PersonID: "AUD-01", AuthorityID: signing.AuthSecondAudit, Scope: "ent01"}},
		Catalog: accessdom.Catalog{Roles: map[string]accessdom.RoleTraits{"administrator": {Domain: accessdom.DomainAdmin},
			"security_auditor": {Domain: accessdom.DomainAdmin}, "customer_representative": {CardOnly: true}},
			GrantTemplate: "policy-grant@1", GrantRoute: route,
			Audit: accessdom.AuditParameters{CheckpointIntervalS: 30, CheckpointMaxGapS: 120}},
	})
}

// memWriter — запись решений в памяти.
type memWriter struct{ batches []access.Batch }

func (m *memWriter) Write(_ context.Context, b access.Batch) (platform.Receipt, error) {
	m.batches = append(m.batches, b)
	return platform.Receipt{CommandID: b.Meta.CommandID, Seq: int64(len(m.batches))}, nil
}

// grantDocs — документы выдачи.
type grantDocs map[string]access.GrantDocument

func (g grantDocs) GrantDocument(_ context.Context, id string) (access.GrantDocument, error) {
	d, ok := g[id]
	if !ok {
		return d, platform.Fail(errcodes.ApiNotFound, "object", "документ", "id", id)
	}
	return d, nil
}

func code26(err error) errcodes.Code {
	if pe, ok := platform.AsError(err); ok {
		return pe.Code
	}
	return ""
}

func as(person string) context.Context {
	return platform.WithPrincipal(context.Background(), platform.Principal{PersonID: person})
}

func clock26(context.Context) (time.Time, error) { return now26, nil }

// Администратор не может расширить себе права без Аудитора ИБ (AD-11, FR-85).
func TestAdminCannotExpandOwnRightsWithoutSecurityAuditor(t *testing.T) {
	w := &memWriter{}
	self := accessdom.PolicyChange{Kind: accessdom.ChangeRole, PersonID: "ADM-01", SubjectID: "technologist", Scope: "ent01"}
	docs := grantDocs{
		"DOC-SELF": {DocumentID: "DOC-SELF", Closed: true, Decision: accessdom.FormatGrantDecision(self),
			Approvals: []accessdom.Approval{{PersonID: "ADM-01", Stage: 1, At: now26}}},
		"DOC-OPEN": {DocumentID: "DOC-OPEN", Closed: false, Decision: accessdom.FormatGrantDecision(self)},
		"DOC-AUD": {DocumentID: "DOC-AUD", Closed: true, Decision: accessdom.FormatGrantDecision(self),
			Approvals: []accessdom.Approval{{PersonID: "ADM-01", Stage: 1, At: now26}, {PersonID: "AUD-01", Stage: 2, At: now26}}},
		"DOC-OTHER": {DocumentID: "DOC-OTHER", Closed: true, Decision: accessdom.FormatGrantDecision(accessdom.PolicyChange{Kind: "role", PersonID: "ADM-01", SubjectID: "security_auditor", Scope: "ent01"}),
			Approvals: []accessdom.Approval{{PersonID: "AUD-01", Stage: 2, At: now26}}},
	}
	s := access.NewService(access.WithPolicy(access.StaticPolicy(policy26())), access.WithDecisions(w, clock26), access.WithGrantDocuments(docs))
	grant := func(doc string) error {
		_, err := s.GrantPolicy(as("ADM-01"), access.GrantPolicy{PersonID: "ADM-01", Kind: "role", RoleID: "technologist", Scope: "ent01", DocumentID: doc})
		return err
	}
	for doc, want := range map[string]errcodes.Code{"": errcodes.AccessSelfGrant, "DOC-SELF": errcodes.AccessSelfGrant,
		"DOC-OPEN": errcodes.AccessSelfGrant, "DOC-OTHER": errcodes.ApiValidationFailed} {
		if err := grant(doc); code26(err) != want {
			t.Fatalf("документ %q: %v", doc, err)
		}
	}
	if len(w.batches) != 0 {
		t.Fatal("без второй подписи запись не делается")
	}
	// Отказ подсказывает «Запросить решение» и кто должен подписать.
	pe, _ := platform.AsError(grant(""))
	if !slices.Contains(pe.AllowedActions, "documents.version.request") || pe.Params["who"] == "" {
		t.Fatalf("подсказка: %+v", pe)
	}
	// С подписью Аудитора ИБ по закрытому маршруту — выдача записью журнала с document_id и second_signature_by.
	if err := grant("DOC-AUD"); err != nil {
		t.Fatal(err)
	}
	r := w.batches[0].Records[0]
	d := r.Data.(ev.PolicyRoleAssignedV1)
	if r.Type != catalog.PolicyRoleAssigned || r.Stream != "policy:ent01" || d.SecondSignatureBy == nil || *d.SecondSignatureBy != "AUD-01" || *d.DocumentID != "DOC-AUD" {
		t.Fatalf("запись выдачи: %+v %+v", r, d)
	}
	// Обычная роль другому — одной подписью, без документа.
	if _, err := s.GrantPolicy(as("ADM-01"), access.GrantPolicy{PersonID: "W21", Kind: "role", RoleID: "technologist", Scope: "ent01/b1"}); err != nil {
		t.Fatal("обычная выдача:", err)
	}
	if w.batches[1].Records[0].Stream != "policy:ent01/b1" {
		t.Fatal("поток области выдачи")
	}
	// Оценка для «Запросить решение»: этап Аудитора ИБ, кандидат — не инициатор.
	a, err := s.AssessGrant(as("ADM-01"), access.GrantQuery{PersonID: "ADM-01", Kind: "role", SubjectID: "technologist", Scope: "ent01"})
	if err != nil || !a.SelfGrant || a.Domain != "admin" || len(a.Stages) != 2 || !slices.Equal(a.Stages[1].Candidates, []string{"AUD-01"}) || a.Template != "policy-grant@1" {
		t.Fatalf("оценка: %+v %v", a, err)
	}
}

func TestRevokeAndAuditParameters(t *testing.T) {
	w := &memWriter{}
	s := access.NewService(access.WithPolicy(access.StaticPolicy(policy26())), access.WithDecisions(w, clock26))
	if _, err := s.RevokePolicy(as("ADM-01"), access.RevokePolicy{PersonID: "TEC-01", Kind: "role", SubjectID: "technologist", Reason: access.AccessReason{Text: "перевод"}}); err != nil {
		t.Fatal(err)
	}
	if r := w.batches[0].Records[0]; r.Type != catalog.PolicyRoleUnassigned || r.Stream != "policy:ent01" {
		t.Fatalf("отзыв: %+v", r)
	}
	if _, err := s.RevokePolicy(as("ADM-01"), access.RevokePolicy{PersonID: "TEC-01", Kind: "role", SubjectID: "administrator", Reason: access.AccessReason{Text: "x"}}); code26(err) != errcodes.ApiNotFound {
		t.Fatal("отзыв несуществующего", err)
	}
	fp := "streebog256:" + string(make([]byte, 0))
	if _, err := s.SetAuditParameters(as("AUD-01"), access.SetAuditParameters{CheckpointIntervalS: 60, CheckpointMaxGapS: 30, KeeperKeyFingerprint: fp}); code26(err) != errcodes.ApiValidationFailed {
		t.Fatal("разрыв меньше интервала", err)
	}
	fp = "streebog256:0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := s.SetAuditParameters(as("AUD-01"), access.SetAuditParameters{CheckpointIntervalS: 60, CheckpointMaxGapS: 300, KeeperKeyFingerprint: fp,
		CriticalTypes: []string{"decision.containment.set"}}); err != nil {
		t.Fatal(err)
	}
	if r := w.batches[1].Records[0]; r.Type != catalog.PolicyAuditParametersSet || r.Data.(ev.PolicyAuditParametersSetV1).CheckpointIntervalS != 60 {
		t.Fatalf("параметры аудита: %+v", r)
	}
	a, err := s.Audit(context.Background(), platform.Moment{})
	if err != nil || a.CheckpointIntervalS != 30 {
		t.Fatalf("параметры по умолчанию: %+v %v", a, err)
	}
}

// policyAC — вычислитель над политикой (как Casbin: роли с наследованием), версия — Seq политики.
type policyAC struct{ pol *accessdom.Policy }

func (a policyAC) Enforce(_ context.Context, rq access.Request) (access.Decision, error) {
	alias := accessdom.ReadAlias(rq.Action.ID, string(rq.Action.Class))
	for _, as := range a.pol.AssignmentsOf(rq.Principal.PersonID, time.Time{}) {
		for _, rid := range a.pol.Hierarchy().Closure(as.RoleID) {
			r, _ := a.pol.Role(rid)
			for _, p := range r.Actions {
				if (accessdom.ActionMatches(rq.Action.ID, p) || accessdom.ActionMatches(alias, p)) && (rq.Scope == "" || accessdom.ScopeCovers(as.Scope, rq.Scope)) {
					return access.Decision{Allowed: true}, nil
				}
			}
		}
	}
	return access.Decision{}, nil
}

func (a policyAC) PolicySeq(context.Context) (int64, error) { return a.pol.Seq, nil }

type ptrPolicy struct{ pol *accessdom.Policy }

func (p ptrPolicy) Policy(context.Context) (accessdom.Policy, error) { return *p.pol, nil }

// Команда по устаревшей политике — 409 journal.stale_policy (AD-39): решение
// о доступе принято копией api по политике версии S; отзыв, записанный другой
// копией после S в поток политики субъекта, отвергает запись команды в той же
// транзакции journal.Append.
func TestStalePolicyCommandRejected(t *testing.T) {
	ctx := context.Background()
	j := enginemem.New(nil)
	w := access.JournalDecisions{Journal: j, DomainBuild: dj.ZeroLink.String()}
	// Журнал до решения: одна запись политики (seq 1).
	if _, err := w.Write(ctx, access.Batch{Actor: "ADM-01", OccurredAt: now26, Records: []access.Record{{Type: catalog.AccessPersonRegistered, Stream: "person:X-1",
		Data: ev.AccessPersonRegisteredV1{PersonID: "X-1", DisplayName: "X"}}}}); err != nil {
		t.Fatal(err)
	}
	pol := policy26()
	pol.Seq = 1
	g := access.NewGate(policyAC{&pol}, nil, nil)
	g.Policy = ptrPolicy{&pol}
	act := platform.Action{ID: "access.audit.set_parameters", Class: platform.ClassRecord}
	aud := platform.Principal{PersonID: "AUD-01"}
	cctx, err := g.Admit(ctx, aud, act, platform.ObjectRef{}, &platform.CommandMeta{PolicySeq: 1})
	if err != nil {
		t.Fatal(err)
	}
	if cs := appjournal.PolicyChecksFrom(cctx); len(cs) == 0 || cs[0].PolicyStream != "policy:ent01" || cs[0].PolicySeq != 1 {
		t.Fatalf("проверки политики: %+v", cs)
	}
	// Другая копия api отзывает роль Аудитора ИБ (seq 2, поток policy:ent01).
	if _, err := w.Write(ctx, access.Batch{Actor: "ADM-01", OccurredAt: now26, Records: []access.Record{{Type: catalog.PolicyRoleUnassigned, Stream: "policy:ent01",
		Data: ev.PolicyRoleUnassignedV1{PersonID: "AUD-01", RoleID: "security_auditor", Scope: "ent01", EffectiveFrom: ev.Timestamp(now26)}}}}); err != nil {
		t.Fatal(err)
	}
	fp := ev.Digest("streebog256:0000000000000000000000000000000000000000000000000000000000000000")
	cmd := access.Batch{Actor: "AUD-01", OccurredAt: now26, Records: []access.Record{{Type: catalog.PolicyAuditParametersSet, Stream: "policy:ent01",
		Data: ev.PolicyAuditParametersSetV1{CheckpointIntervalS: 60, CheckpointMaxGapS: 300, KeeperKeyFingerprint: fp}}}}
	_, err = w.Write(cctx, cmd)
	pe, ok := platform.AsError(err)
	if !ok || pe.Code != errcodes.JournalStalePolicy {
		t.Fatalf("устаревшая политика: %v", err)
	}
	if info, _ := errcodes.Lookup(pe.Code); info.Status != 409 {
		t.Fatalf("статус %d", info.Status)
	}
	if hs, _ := j.Head(ctx); hs.MainSeq != 2 {
		t.Fatalf("запись по устаревшей политике не должна появиться: голова %d", hs.MainSeq)
	}
	// Клиент видел права до изменения (policy_seq 1), копия api уже знает seq 2 — тоже 409.
	pol.Seq = 2
	pol = pol.With(accessdom.PolicyChange{Kind: "role", Revoke: true, PersonID: "AUD-01", SubjectID: "security_auditor", Scope: "ent01", ValidFrom: now26})
	pol.Seq = 2
	pol.Assignments = append(pol.Assignments, accessdom.Assignment{PersonID: "AUD-01", RoleID: "security_auditor", Scope: "ent01"})
	cctx, err = g.Admit(ctx, aud, act, platform.ObjectRef{}, &platform.CommandMeta{PolicySeq: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(cctx, cmd); code26(err) != errcodes.JournalStalePolicy {
		t.Fatalf("policy_seq клиента: %v", err)
	}
	// По актуальной политике — записывается.
	cctx, _ = g.Admit(ctx, aud, act, platform.ObjectRef{}, &platform.CommandMeta{PolicySeq: 2})
	if _, err := w.Write(cctx, cmd); err != nil {
		t.Fatalf("актуальная политика: %v", err)
	}
}

// cards — карточки CR-71: документ DOC-1 и его изделие.
type cards struct{}

func (cards) Visible(_ context.Context, person string, obj platform.ObjectRef) (bool, error) {
	return person == "CR-71" && (obj.ID == "DOC-1" || obj.ID == "ENT01:F-1"), nil
}

// Редкий подписант видит только свою карточку (FR-136); объяснение прав — своим кодом.
func TestCardOnlyAndExplain(t *testing.T) {
	ctx := context.Background()
	pol := policy26()
	acts := []platform.Action{
		{ID: "documents.decision_card.read", Class: platform.ClassRead, Subject: "document"},
		{ID: "documents.signature.record", Class: platform.ClassRecord, Subject: "document"},
		{ID: "item.passport.read", Class: platform.ClassRead, Subject: "item"},
		{ID: "analysis.cause.conclude", Class: platform.ClassRecord},
		{ID: "documents.version.request", Class: platform.ClassRecord},
	}
	g := access.NewGate(policyAC{&pol}, nil, func() []platform.Action { return acts })
	g.Policy, g.Cards = ptrPolicy{&pol}, cards{}
	cr := platform.Principal{PersonID: "CR-71"}
	doc := func(id string) platform.ObjectRef { return platform.ObjectRef{Kind: "document", ID: id} }
	if err := g.Authorize(ctx, cr, acts[0], doc("DOC-1"), nil); err != nil {
		t.Fatal("своя карточка:", err)
	}
	if err := g.Authorize(ctx, cr, acts[1], doc("DOC-1"), &platform.CommandMeta{}); err != nil {
		t.Fatal("подпись с карточки:", err)
	}
	if err := g.Authorize(ctx, cr, acts[0], doc("DOC-2"), nil); code26(err) != errcodes.AccessForbidden {
		t.Fatal("чужая карточка:", err)
	}
	if err := g.Authorize(ctx, cr, acts[2], platform.ObjectRef{Kind: "item", ID: "ENT01:F-1"}, nil); err != nil {
		t.Fatal("паспорт изделия карточки:", err)
	}
	if err := g.Authorize(ctx, cr, acts[2], platform.ObjectRef{Kind: "item", ID: "ENT01:F-2"}, nil); code26(err) != errcodes.AccessForbidden {
		t.Fatal("чужое изделие:", err)
	}
	// Технолог — не редкий подписант: объект не сужается.
	if err := g.Authorize(ctx, platform.Principal{PersonID: "TEC-01"}, acts[2], platform.ObjectRef{Kind: "item", ID: "ENT01:F-2"}, nil); err != nil {
		t.Fatal("технолог:", err)
	}
	// Объяснение: какая роль разрешает (с наследованием) и кто может.
	ex, err := g.Explain(ctx, platform.Principal{PersonID: "TEC-01"}, "item.passport.read", platform.ObjectRef{})
	if err != nil || !ex.Allowed || ex.GrantedBy == nil || ex.GrantedBy.RoleID != "technologist" || ex.GrantedBy.Via != "staff" {
		t.Fatalf("объяснение разрешения: %+v %v", ex, err)
	}
	ex, err = g.Explain(ctx, cr, "analysis.cause.conclude", platform.ObjectRef{})
	if err != nil || ex.Allowed || ex.Code != string(errcodes.AccessForbidden) || !ex.CardOnly || len(ex.WhoCan) != 1 || ex.WhoCan[0].RoleID != "technologist" ||
		!slices.Equal(ex.WhoCan[0].Persons, []string{"TEC-01"}) {
		t.Fatalf("объяснение отказа: %+v %v", ex, err)
	}
	b, _ := json.Marshal(ex)
	if !json.Valid(b) {
		t.Fatal("json")
	}
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
