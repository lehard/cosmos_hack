package verify

import (
	"context"
	"testing"
	"time"

	"ant/internal/application/access"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/contracts/procs"
	accessdom "ant/internal/domain/access"
	dj "ant/internal/domain/journal"
	signing "ant/internal/domain/signing"
)

// Верификатор подтверждает право каждого подписанта на момент подписи (FR-85,
// эпик 26): выдача себе без второй подписи Аудитора ИБ, записанная в обход
// api, и выдача без права — отвергаются; законная выдача — цела.
func TestAuthorityCheck(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	pol := accessdom.FromSeed(accessdom.Seed{Root: "ent01",
		Roles: []accessdom.Role{
			{ID: "technologist", Actions: []string{"analysis.cause.conclude"}},
			{ID: "administrator", Actions: []string{"access.policy.grant", "access.policy.revoke"}},
			{ID: "security_auditor", Actions: []string{"access.audit.set_parameters"}},
		},
		Persons: []accessdom.SeedPerson{
			{ID: "ADM-01", Roles: []accessdom.SeedGrant{{Role: "administrator", Scope: "ent01"}}},
			{ID: "AUD-01", Roles: []accessdom.SeedGrant{{Role: "security_auditor", Scope: "ent01"}}},
			{ID: "TEC-01", Roles: []accessdom.SeedGrant{{Role: "technologist", Scope: "ent01"}}},
		},
		Authorities: []accessdom.Authority{{PersonID: "AUD-01", AuthorityID: signing.AuthSecondAudit, Scope: "ent01"}},
		Catalog:     accessdom.Catalog{Roles: map[string]accessdom.RoleTraits{"administrator": {Domain: accessdom.DomainAdmin}}},
	})
	run1 := func(write func(w access.JournalDecisions)) *check {
		j := enginemem.New(nil)
		w := access.JournalDecisions{Journal: j, DomainBuild: dj.ZeroLink.String()}
		write(w)
		codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: 1}
		v := &run{in: Input{Codec: codec, Policy: &pol}, checks: map[procs.VerifierReportV1ChecksElemCheck]*check{"authority": {name: "authority"}}}
		if err := v.authority(ctx); err != nil {
			t.Fatal(err)
		}
		return v.checks["authority"]
	}
	grant := func(w access.JournalDecisions, actor, person, second string) {
		d := ev.PolicyRoleAssignedV1{PersonID: ev.PersonRef(person), RoleID: "technologist", Scope: "ent01", ValidFrom: ev.Timestamp(at)}
		if second != "" {
			s := ev.PersonRef(second)
			d.SecondSignatureBy = &s
		}
		if _, err := w.Write(ctx, access.Batch{Actor: actor, OccurredAt: at, Records: []access.Record{{Type: catalog.PolicyRoleAssigned, Stream: "policy:ent01", Data: d}}}); err != nil {
			t.Fatal(err)
		}
	}
	if c := run1(func(w access.JournalDecisions) { grant(w, "ADM-01", "ADM-01", "") }); !c.rejected || c.findings[0].Code != "authority.second_signature_missing" {
		t.Fatalf("выдача себе без второй подписи: %+v", c.findings)
	}
	if c := run1(func(w access.JournalDecisions) { grant(w, "ADM-01", "ADM-01", "ADM-01") }); !c.rejected || c.findings[0].Code != "authority.second_signature_invalid" {
		t.Fatalf("вторая подпись самого себя: %+v", c.findings)
	}
	if c := run1(func(w access.JournalDecisions) { grant(w, "TEC-01", "TEC-01", "") }); !c.rejected || c.findings[0].Code != "authority.no_right" {
		t.Fatalf("выдача без права: %+v", c.findings)
	}
	if c := run1(func(w access.JournalDecisions) { grant(w, "ADM-01", "ADM-01", "AUD-01") }); c.rejected || c.unverif || c.checked != 1 {
		t.Fatalf("законная выдача: %+v", c)
	}
}
