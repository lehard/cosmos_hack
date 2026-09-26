package access

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
	signing "ant/internal/domain/signing"
)

// Эпик 26: вторая подпись независимой стороны, выдача себе по эффективным
// правам, единая функция обязательных подписей, политика на seq.

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

// testPolicy — срез стартовой политики (normative/policy/policy.v1.yaml).
func testPolicy() Policy {
	return FromSeed(Seed{
		Root: "ent01",
		Roles: []Role{
			{ID: "employee", Actions: []string{"access.session.read", "notifications.*.read"}},
			{ID: "staff", Inherits: []string{"employee"}, Actions: []string{"quality.*.read", "item.*.read"}},
			{ID: "quality_inspector", Title: "Контролёр качества", Inherits: []string{"staff"}, Actions: []string{"nonconformity.signal.reject", "documents.document.sign"}},
			{ID: "head_of_qc", Title: "Начальник ОТК", Inherits: []string{"quality_inspector"}, Actions: []string{"vision.passport.reinstate"}},
			{ID: "technologist", Title: "Технолог", Inherits: []string{"staff"}, Actions: []string{"analysis.cause.conclude"}},
			{ID: "production_manager", Title: "Руководитель производства", Inherits: []string{"staff"}, Actions: []string{"process.version.activate"}},
			{ID: "administrator", Title: "Администратор", Inherits: []string{"staff"}, Actions: []string{"access.policy.grant", "access.policy.revoke"}},
			{ID: "security_auditor", Title: "Аудитор ИБ", Inherits: []string{"staff"}, Actions: []string{"security.*.read", "access.audit.set_parameters"}},
			{ID: "customer_representative", Title: "Представитель заказчика", Inherits: []string{"employee"}, Actions: []string{"documents.decision_card.read", "documents.signature.*"}},
			{ID: "approver", Title: "Согласующий", Inherits: []string{"employee"}, Actions: []string{"documents.decision_card.read"}},
			{ID: "metrologist", Title: "Метролог", Inherits: []string{"approver", "staff"}, Actions: []string{"reference.equipment.verify"}},
			{ID: "design_authority", Title: "Держатель КД", Inherits: []string{"approver"}},
		},
		Persons: []SeedPerson{
			{ID: "INS-01", Roles: []SeedGrant{{Role: "quality_inspector", Scope: "ent01/b1"}}},
			{ID: "HQC-01", Roles: []SeedGrant{{Role: "head_of_qc", Scope: "ent01"}}},
			{ID: "TEC-01", Roles: []SeedGrant{{Role: "technologist", Scope: "ent01"}}},
			{ID: "PM-01", Roles: []SeedGrant{{Role: "production_manager", Scope: "ent01"}}},
			{ID: "ADM-01", Roles: []SeedGrant{{Role: "administrator", Scope: "ent01"}}},
			{ID: "ADM-02", Roles: []SeedGrant{{Role: "administrator", Scope: "ent01"}}},
			{ID: "AUD-01", Roles: []SeedGrant{{Role: "security_auditor", Scope: "ent01"}}},
			{ID: "CR-71", Roles: []SeedGrant{{Role: "customer_representative", Scope: "ent01"}}},
			{ID: "DA-81", Roles: []SeedGrant{{Role: "design_authority", Scope: "ent01"}}},
			{ID: "MET-82", Roles: []SeedGrant{{Role: "metrologist", Scope: "ent01"}}},
			{ID: "W21", Roles: []SeedGrant{}},
		},
		Authorities: []Authority{
			{PersonID: "HQC-01", AuthorityID: signing.AuthSecondQC, Scope: "ent01"},
			{PersonID: "HQC-01", AuthorityID: "concession_approval", Scope: "ent01"},
			{PersonID: "TEC-01", AuthorityID: "concession_approval", Scope: "ent01"},
			{PersonID: "PM-01", AuthorityID: signing.AuthSecondProduction, Scope: "ent01"},
			{PersonID: "AUD-01", AuthorityID: signing.AuthSecondAudit, Scope: "ent01"},
			{PersonID: "CR-71", AuthorityID: "customer_acceptance", Scope: "ent01"},
			{PersonID: "CR-71", AuthorityID: "concession_approval", Scope: "ent01"},
			{PersonID: "DA-81", AuthorityID: "concession_approval", Scope: "ent01"},
		},
		Stamps: []Stamp{{StampID: "OTK-01-SV", PersonID: "INS-01", Kind: "weld", Scope: "ent01/b1/wc"}},
		Catalog: Catalog{
			Authorities: []AuthorityDef{{ID: "qc_acceptance", Domain: DomainQC}, {ID: "scrap_approval", Domain: DomainProduction},
				{ID: "concession_approval", Domain: DomainQC}, {ID: signing.AuthSecondAudit, Domain: DomainAdmin}},
			Roles: map[string]RoleTraits{"quality_inspector": {Domain: DomainQC}, "administrator": {Domain: DomainAdmin},
				"security_auditor": {Domain: DomainAdmin}, "customer_representative": {CardOnly: true}, "approver": {CardOnly: true},
				"metrologist": {}, "design_authority": {CardOnly: true}},
			StampKinds: []string{"weld", "final"},
			GrantRoute: grantRoute(),
		},
	})
}

func grantRoute() []RouteStage {
	return []RouteStage{
		{Stage: 1, Title: "Администратор безопасности", Role: "administrator", AuthorityID: "administrator", Quorum: QuorumOne, SignatureLevel: 2, BySource: true},
		{Stage: 2, Title: "Начальник ОТК", AuthorityID: signing.AuthSecondQC, Quorum: QuorumOne, SignatureLevel: 2, When: &StageCondition{GrantDomains: []string{DomainQC}}},
		{Stage: 3, Title: "Руководитель производства", AuthorityID: signing.AuthSecondProduction, Quorum: QuorumOne, SignatureLevel: 2, When: &StageCondition{GrantDomains: []string{DomainProduction}}},
		{Stage: 4, Title: "Аудитор ИБ", AuthorityID: signing.AuthSecondAudit, Quorum: QuorumOne, SignatureLevel: 2, When: &StageCondition{GrantDomains: []string{DomainAdmin}}},
	}
}

func role(person, r, scope string) PolicyChange {
	return PolicyChange{Kind: ChangeRole, PersonID: person, SubjectID: r, Scope: scope, ValidFrom: t0}
}

func TestAssessSecondSignature(t *testing.T) {
	p := testPolicy()
	for _, c := range []struct {
		name      string
		change    PolicyChange
		initiator string
		domain    string
		self      bool
	}{
		{"обычная роль другому — одной подписью", role("W21", "technologist", "ent01/b1"), "ADM-01", DomainOrdinary, false},
		{"администратор выдаёт себе роль технолога — Аудитор ИБ", role("ADM-01", "technologist", "ent01"), "ADM-01", DomainAdmin, true},
		{"роль администратора — привилегия, Аудитор ИБ", role("W21", "administrator", "ent01"), "ADM-01", DomainAdmin, false},
		{"расширение прав другого администратора — Аудитор ИБ, кто бы ни выдавал", role("ADM-02", "technologist", "ent01"), "ADM-01", DomainAdmin, false},
		{"полномочие ОТК — начальник ОТК", PolicyChange{Kind: ChangeAuthority, PersonID: "INS-01", SubjectID: "qc_acceptance", Scope: "ent01/b1", ValidFrom: t0}, "ADM-01", DomainQC, false},
		{"полномочие производства — руководитель производства", PolicyChange{Kind: ChangeAuthority, PersonID: "TEC-01", SubjectID: "scrap_approval", Scope: "ent01", ValidFrom: t0}, "ADM-01", DomainProduction, false},
		{"клеймо — начальник ОТК", PolicyChange{Kind: ChangeStamp, PersonID: "INS-01", SubjectID: "OTK-01-OK", InspectionKind: "final", Scope: "ent01/b1", ValidFrom: t0}, "ADM-01", DomainQC, false},
		{"роль, которая уже покрыта, — права не расширяются", role("ADM-01", "staff", "ent01"), "ADM-01", DomainOrdinary, false},
		{"отзыв — защитное, одной подписью", PolicyChange{Kind: ChangeRole, Revoke: true, PersonID: "ADM-01", SubjectID: "administrator", Scope: "ent01", ValidFrom: t0}, "ADM-01", DomainOrdinary, false},
	} {
		a := Assess(p, c.change, c.initiator, t0)
		if a.Domain != c.domain || a.SelfGrant != c.self || (a.SecondAuthority == "") != (c.domain == DomainOrdinary) {
			t.Errorf("%s: %+v", c.name, a)
		}
	}
}

func TestCheckApprovalsIndependentParty(t *testing.T) {
	p := testPolicy()
	c := role("ADM-01", "technologist", "ent01")
	a := Assess(p, c, "ADM-01", t0)
	code := func(err error) errcodes.Code {
		var r *kernel.Refusal
		if errors.As(err, &r) {
			return r.Code
		}
		return ""
	}
	// Администратор не может расширить себе права без Аудитора ИБ.
	if got := code(CheckApprovals(p, a, c, nil)); got != errcodes.AccessSelfGrant {
		t.Fatalf("без второй подписи: %v", got)
	}
	// Своя «вторая подпись» и подпись другого администратора не засчитываются.
	if got := code(CheckApprovals(p, a, c, []Approval{{PersonID: "ADM-01", At: t0}, {PersonID: "ADM-02", At: t0}, {PersonID: "HQC-01", At: t0}})); got != errcodes.AccessSelfGrant {
		t.Fatalf("не та сторона: %v", got)
	}
	if err := CheckApprovals(p, a, c, []Approval{{PersonID: "AUD-01", At: t0}}); err != nil {
		t.Fatalf("Аудитор ИБ: %v", err)
	}
	// Полномочие второй подписи — на момент подписи: после отзыва не засчитывается.
	revoked := p.With(PolicyChange{Kind: ChangeAuthority, Revoke: true, PersonID: "AUD-01", SubjectID: signing.AuthSecondAudit, Scope: "ent01", ValidFrom: t0.Add(time.Hour)})
	if err := CheckApprovals(revoked, a, c, []Approval{{PersonID: "AUD-01", At: t0.Add(2 * time.Hour)}}); code(err) != errcodes.AccessSelfGrant {
		t.Fatalf("после отзыва полномочия: %v", err)
	}
	if err := CheckApprovals(revoked, a, c, []Approval{{PersonID: "AUD-01", At: t0.Add(30 * time.Minute)}}); err != nil {
		t.Fatalf("до отзыва полномочия: %v", err)
	}
	// Не себе — код «нужна подпись».
	b := Assess(p, role("W21", "administrator", "ent01"), "ADM-01", t0)
	if got := code(CheckApprovals(p, b, role("W21", "administrator", "ent01"), nil)); got != errcodes.AccessSignatureRequired {
		t.Fatalf("привилегия другому: %v", got)
	}
}

func TestRequiredApprovalsGrantRoute(t *testing.T) {
	p := testPolicy()
	// Обычная выдача — только этап инициатора.
	st := RequiredApprovals(grantRoute(), ApprovalContext{Decision: FormatGrantDecision(role("W21", "technologist", "ent01/b1")), Initiator: "ADM-01", At: t0}, p)
	if len(st) != 1 || !st[0].BySource {
		t.Fatalf("обычная выдача: %+v", st)
	}
	// Выдача себе администратором — этап Аудитора ИБ; кандидаты — не инициатор.
	st = RequiredApprovals(grantRoute(), ApprovalContext{Decision: FormatGrantDecision(role("ADM-01", "technologist", "ent01")), Initiator: "ADM-01", At: t0}, p)
	if len(st) != 2 || st[1].Stage != 2 || st[1].AuthorityID != signing.AuthSecondAudit || !slices.Equal(st[1].Candidates, []string{"AUD-01"}) {
		t.Fatalf("выдача себе: %+v", st)
	}
	// Клеймо — начальник ОТК.
	stamp := PolicyChange{Kind: ChangeStamp, PersonID: "INS-01", SubjectID: "OTK-01-OK", InspectionKind: "final", Scope: "ent01/b1"}
	st = RequiredApprovals(grantRoute(), ApprovalContext{Grant: &stamp, Initiator: "ADM-01", At: t0}, p)
	if len(st) != 2 || !slices.Equal(st[1].Candidates, []string{"HQC-01"}) {
		t.Fatalf("клеймо: %+v", st)
	}
	// Кандидаты не входят в канонический JSON (замороженный набор не зависит от состава людей).
	if b, _ := json.Marshal(st[1]); strings.Contains(string(b), "candidates") || !strings.Contains(string(b), `"required_count":1`) {
		t.Fatalf("JSON этапа: %s", b)
	}
}

// Совместимость с заготовкой эпика 28: без политики — те же этапы по условиям.
func TestRequiredApprovalsWithoutPolicy(t *testing.T) {
	route := []RouteStage{
		{Stage: 1, Title: "Автор", AuthorityID: "nc_disposition", Quorum: QuorumOne, SignatureLevel: 2},
		{Stage: 2, Title: "Комиссия", AuthorityID: "commission", Quorum: QuorumKOfN, K: 2, SignatureLevel: 2, PaperAllowed: true},
		{Stage: 3, Title: "Только для ремонта", AuthorityID: "x", Quorum: QuorumOne, When: &StageCondition{Decisions: []string{"repair"}}},
	}
	st := RequiredApprovals(route, ApprovalContext{Decision: "scrap"}, Policy{})
	if len(st) != 2 || st[1].Required != 2 || st[1].PaperAllowed || st[0].Candidates != nil {
		t.Fatalf("этапы: %+v", st)
	}
}

// ВП подписывает разрешение на отклонение: этап представителя заказчика по
// маршруту каталога документов; кандидат — CR-71, редкий подписант.
func TestConcessionRouteCustomerRepresentative(t *testing.T) {
	p := testPolicy()
	route := []RouteStage{
		{Stage: 1, Title: "Технолог", Role: "technologist", AuthorityID: "concession_approval", Quorum: QuorumOne, SignatureLevel: 2},
		{Stage: 2, Title: "Начальник ОТК", Role: "head_of_qc", AuthorityID: "concession_approval", Quorum: QuorumOne, SignatureLevel: 2},
		{Stage: 3, Title: "Представитель заказчика", Role: "customer_representative", AuthorityID: "concession_approval", Quorum: QuorumOne,
			SignatureLevel: 2, ExternalParty: "customer_representative", When: &StageCondition{CustomerAcceptance: ptr(true)}},
		{Stage: 4, Title: "Держатель КД", Role: "design_authority", AuthorityID: "concession_approval", Quorum: QuorumOne, SignatureLevel: 2},
	}
	st := RequiredApprovals(route, ApprovalContext{Decision: "use_as_is", CustomerAcceptance: true, Scope: "ent01/b1/wc", At: t0}, p)
	if len(st) != 4 || !slices.Equal(st[2].Candidates, []string{"CR-71"}) || !slices.Equal(st[3].Candidates, []string{"DA-81"}) {
		t.Fatalf("маршрут разрешения на отклонение: %+v", st)
	}
	if !p.CardOnly("CR-71", t0) || !p.CardOnly("DA-81", t0) || p.CardOnly("MET-82", t0) || p.CardOnly("INS-01", t0) || p.CardOnly("W21", t0) {
		t.Fatal("редкие подписанты")
	}
}

func ptr[T any](v T) *T { return &v }

func TestRightsAndFrames(t *testing.T) {
	p := testPolicy()
	if !p.HasAuthority("HQC-01", "concession_approval", "ent01/b1/wc", t0) || p.HasAuthority("INS-01", "concession_approval", "", t0) {
		t.Fatal("полномочие в области")
	}
	if _, ok := p.StampFor("INS-01", "weld", "ent01/b1/wc/weld/wp1", t0); !ok {
		t.Fatal("клеймо в области")
	}
	if _, ok := p.StampFor("INS-01", "weld", "ent01/b1/mc", t0); ok {
		t.Fatal("клеймо вне области")
	}
	if p.PersonDomain("HQC-01", t0) != DomainQC || p.PersonDomain("AUD-01", t0) != DomainAdmin || p.PersonDomain("PM-01", t0) != DomainProduction {
		t.Fatal("сферы сотрудников")
	}
	r := p.EffectiveRights("INS-01", t0)
	if !r.Covers(Right{Kind: RightAction, Value: "quality.signal.read", Scope: "ent01/b1/wc"}) || r.Covers(Right{Kind: RightAction, Value: "quality.signal.read", Scope: "ent01"}) {
		t.Fatal("эффективные права: шаблон и область")
	}
	if got := p.SubjectStreams("INS-01", t0); !slices.Equal(got, []string{"policy:ent01", "policy:ent01/b1", "policy:ent01/b1/wc"}) {
		t.Fatalf("потоки политики субъекта: %v", got)
	}
	c, ok := ParseGrantDecision(FormatGrantDecision(PolicyChange{Kind: ChangeStamp, SubjectID: "OTK-01-OK", InspectionKind: "final", PersonID: "INS-01", Scope: "ent01/b1"}))
	if !ok || c.InspectionKind != "final" || c.SubjectID != "OTK-01-OK" || c.Scope != "ent01/b1" {
		t.Fatalf("решение выдачи: %+v", c)
	}
}

func TestPolicyAtAndAudit(t *testing.T) {
	base := testPolicy()
	rec := func(seq int64, typ catalog.Type, d any) Record {
		b, _ := json.Marshal(d)
		return Record{Seq: seq, Type: string(typ), Data: b, OccurredAt: t0.Add(time.Duration(seq) * time.Minute), Actor: "aud-01@1", EventID: "e"}
	}
	recs := []Record{
		rec(10, catalog.PolicyRoleUnassigned, ev.PolicyRoleUnassignedV1{PersonID: "INS-01", RoleID: "quality_inspector", Scope: "ent01/b1", EffectiveFrom: ev.Timestamp(t0.Add(10 * time.Minute))}),
		rec(20, catalog.PolicyAuditParametersSet, ev.PolicyAuditParametersSetV1{CheckpointIntervalS: 60, CheckpointMaxGapS: 300, KeeperKeyFingerprint: "streebog256:00"}),
	}
	at5, err := PolicyAt(base, recs, 5)
	if err != nil || !at5.HasRole("INS-01", "quality_inspector", "", t0.Add(time.Hour)) {
		t.Fatalf("политика до отзыва: %v", err)
	}
	at20, err := PolicyAt(base, recs, 20)
	if err != nil || at20.HasRole("INS-01", "quality_inspector", "", t0.Add(time.Hour)) || at20.Audit.CheckpointIntervalS != 60 || at20.Audit.Seq != 20 || at20.Seq != 20 {
		t.Fatalf("политика после отзыва и параметры аудита: %v %+v", err, at20.Audit)
	}
}
