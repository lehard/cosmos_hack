package access_test

import (
	"context"
	"testing"
	"time"

	"ant/internal/application/access"
	itemapp "ant/internal/application/item"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
	"ant/internal/domain/kernel"
)

// Посты, назначения и факты исполнителя (FR-81, FR-137, PRD §11.18).

func rosterPolicy() accessdom.Policy {
	return accessdom.FromSeed(accessdom.Seed{Root: "ent01",
		Roles: []accessdom.Role{
			{ID: "performer", Title: "Исполнитель", Actions: []string{"access.operator.report_deviation"}},
			{ID: "quality_inspector", Title: "Контролёр"},
			{ID: "head_of_qc", Title: "Начальник ОТК", Inherits: []string{"quality_inspector"}},
			{ID: "site_foreman", Title: "Мастер", Actions: []string{"access.assignment.set"}},
		},
		Persons: []accessdom.SeedPerson{
			{ID: "W21", Name: "Сварщик W21", Roles: []accessdom.SeedGrant{{Role: "performer", Scope: "ent01/b1/wc"}}},
			{ID: "W22", Name: "Сварщик W22", Roles: []accessdom.SeedGrant{{Role: "performer", Scope: "ent01/b1/wc"}}},
			{ID: "O17", Roles: []accessdom.SeedGrant{{Role: "performer", Scope: "ent01/b1/mc"}}},
			{ID: "INS-01", Roles: []accessdom.SeedGrant{{Role: "quality_inspector", Scope: "ent01/b1"}}},
			{ID: "HQC-01", Roles: []accessdom.SeedGrant{{Role: "head_of_qc", Scope: "ent01"}}},
			{ID: "FOR-WC", Roles: []accessdom.SeedGrant{{Role: "site_foreman", Scope: "ent01/b1/wc"}}},
		},
		Authorities: []accessdom.Authority{{PersonID: "HQC-01", AuthorityID: access.AuthControllerApproval, Scope: "ent01"}},
		Qualifications: []accessdom.Qualification{
			{PersonID: "W21", QualificationID: "welder", Scope: "ent01/b1/wc", ValidUntil: now26.Add(365 * 24 * time.Hour)},
			{PersonID: "W22", QualificationID: "welder", Scope: "ent01/b1/wc", ValidUntil: now26.Add(-24 * time.Hour)},
		},
	})
}

func rosterDir() *access.Directory {
	return &access.Directory{Workplaces: []access.WorkplaceRef{
		{ID: "WP-WELD-1", Name: "Пост сварки 1", Scope: "ent01/b1/wc/weld/wp1", Workshop: "WS-WC"},
		{ID: "WP-QC-WC", Name: "Пост ОТК сварочного цеха", Scope: "ent01/b1/wc/qc", Workshop: "WS-WC"},
		{ID: "WP-CNC-1", Name: "Пост ЧПУ 1", Scope: "ent01/b1/mc/cnc/wp1", Workshop: "WS-MC"},
	}}
}

// applyWriter — запись решений в память с применением к политике (как проекция).
type applyWriter struct {
	pol *accessdom.Policy
	n   int64
}

func (w *applyWriter) Write(_ context.Context, b access.Batch) (platform.Receipt, error) {
	for _, r := range b.Records {
		w.n++
		data, _ := jsonMarshal(r.Data)
		if err := w.pol.Apply(accessdom.Record{Seq: w.n, Type: string(r.Type), Data: data, OccurredAt: b.OccurredAt}); err != nil {
			return platform.Receipt{}, err
		}
	}
	return platform.Receipt{Seq: w.n}, nil
}

type factLog struct{ recs []itemapp.Record }

func (f *factLog) Write(_ context.Context, owner kernel.Module, recs []itemapp.Record) (platform.Receipt, error) {
	f.recs = append(f.recs, recs...)
	return platform.Receipt{Seq: int64(len(f.recs))}, nil
}

func TestAssignmentsGuardsAndPosts(t *testing.T) {
	pol := rosterPolicy()
	w := &applyWriter{pol: &pol, n: 10}
	facts := &factLog{}
	approve := func(decision string, signer string) access.GrantDocument {
		return access.GrantDocument{DocumentID: "DOC-C", Template: "controller-assignment@1", Closed: true, Decision: decision, Subject: "WP-QC-WC",
			Approvals: []accessdom.Approval{{PersonID: signer, Stage: 1, At: now26}}}
	}
	docs := grantDocs{"DOC-C": approve(accessdom.ControllerDecision("INS-01", "S1"), "HQC-01"),
		"DOC-SELF": approve(accessdom.ControllerDecision("INS-01", "S1"), "FOR-WC")}
	docs["DOC-SELF"] = access.GrantDocument{DocumentID: "DOC-SELF", Template: "controller-assignment@1", Closed: true,
		Decision: accessdom.ControllerDecision("INS-01", "S1"), Approvals: []accessdom.Approval{{PersonID: "FOR-WC", At: now26}}}
	s := access.NewService(access.WithPolicy(ptrPolicy{&pol}), access.WithDecisions(w, clock26), access.WithDirectory(rosterDir()),
		access.WithGrantDocuments(docs), access.WithLiveRoster(facts))
	master := as("FOR-WC")
	set := func(person, wp, role, doc string) error {
		_, err := s.SetAssignment(master, access.SetAssignment{PersonID: person, WorkplaceID: wp, ShiftID: "S1", AssigneeRole: role, ApprovalDocumentID: doc})
		return err
	}
	// Исполнитель — только допущенный по квалификации на дату смены.
	if err := set("W22", "WP-WELD-1", "performer", ""); code26(err) != errcodes.AccessNotQualified {
		t.Fatalf("квалификация истекла: %v", err)
	}
	if err := set("O17", "WP-WELD-1", "performer", ""); code26(err) != errcodes.AccessWrongWorkplace {
		t.Fatalf("роль не в области поста: %v", err)
	}
	if err := set("W21", "WP-WELD-1", "performer", ""); err != nil {
		t.Fatalf("исполнитель: %v", err)
	}
	// Контролёр — только по согласованию начальника ОТК (PRD §11.18).
	for doc, want := range map[string]errcodes.Code{"": errcodes.AccessControllerApprovalRequired, "DOC-SELF": errcodes.AccessControllerApprovalRequired} {
		if err := set("INS-01", "WP-QC-WC", "quality_inspector", doc); code26(err) != want {
			t.Fatalf("контролёр, документ %q: %v", doc, err)
		}
	}
	if err := set("INS-01", "WP-QC-WC", "quality_inspector", "DOC-C"); err != nil {
		t.Fatalf("контролёр по согласованию: %v", err)
	}
	list, err := s.Assignments(context.Background(), "S1", "WS-WC", platform.Moment{})
	if err != nil || len(list.Items) != 2 || !list.Items[0].QualificationOK || list.Items[1].ApprovalDocumentID != "DOC-C" {
		t.Fatalf("назначения: %+v %v", list, err)
	}
	posts, err := s.Workplaces(context.Background(), "WS-WC", platform.Moment{})
	if err != nil || len(posts.Items) != 2 || posts.Items[1].Assigned == nil || posts.Items[1].Assigned.PersonID != "W21" || posts.Items[1].Presence != "unknown" ||
		posts.Items[0].Assigned == nil || posts.Items[0].Assigned.PersonID != "INS-01" {
		t.Fatalf("посты: %+v %v", posts, err)
	}
	q, err := s.Qualifications(context.Background(), "", platform.Moment{})
	if err != nil || len(q.Items) != 2 || q.Items[0].Status != "valid" || q.Items[1].Status != "expired" {
		t.Fatalf("квалификации: %+v %v", q, err)
	}
	// Факт исполнителя — только со своего поста.
	if _, err := s.ReportDeviation(as("W21"), "WP-WELD-2", access.ReportDeviation{Description: "подрез"}); code26(err) != errcodes.AccessWrongWorkplace {
		t.Fatalf("чужой пост: %v", err)
	}
	if _, err := s.ReportDeviation(as("W21"), "WP-WELD-1", access.ReportDeviation{Description: "подрез", ItemID: "ENT01:F-1"}); err != nil {
		t.Fatal(err)
	}
	if r := facts.recs[0]; r.Type != catalog.OperatorDeviationReported || r.Stream != "item:ENT01:F-1" || r.Data.(ev.OperatorDeviationReportedV1).OperatorID != "W21" {
		t.Fatalf("факт: %+v", r)
	}
	if _, err := s.ClearAssignment(master, access.ClearAssignment{PersonID: "W21", WorkplaceID: "WP-WELD-1", ShiftID: "S1"}); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.Assignments(context.Background(), "S1", "", platform.Moment{}); len(list.Items) != 1 {
		t.Fatalf("после снятия: %+v", list)
	}
}
