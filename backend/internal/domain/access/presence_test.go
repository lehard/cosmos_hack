package access

import (
	"errors"
	"slices"
	"strconv"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Эпик 37: присутствие по СКУД и ключу, допуск к рабочему месту, отклонения
// присутствия, квалификация на дату и клеймо по виду контроля.

const (
	wpWeld   = "WP-WELD-2"
	wpScope  = "ent01/b1/wc/weld/wp2"
	zoneWeld = "Z-WC"
)

var at37 = time.Date(2026, 9, 26, 9, 0, 0, 0, time.UTC)

// policy37 — сварщик W21 (исполнитель в сварочном цехе) с аттестацией до
// 1 октября; W22 — с истёкшей аттестацией; оба назначены на пост в SHIFT-1.
func policy37(t *testing.T) Policy {
	t.Helper()
	p := FromSeed(Seed{Root: "ent01",
		Roles: []Role{{ID: "performer", Actions: []string{"access.workplace.admit"}}, {ID: "quality_inspector"}},
		Persons: []SeedPerson{
			{ID: "W21", Roles: []SeedGrant{{Role: "performer", Scope: "ent01/b1/wc"}}},
			{ID: "W22", Roles: []SeedGrant{{Role: "performer", Scope: "ent01/b1/wc"}}},
			{ID: "O17", Roles: []SeedGrant{{Role: "performer", Scope: "ent01/b1/mc"}}},
			{ID: "INS-01", Roles: []SeedGrant{{Role: "quality_inspector", Scope: "ent01/b1"}}},
		},
		Qualifications: []Qualification{
			{PersonID: "W21", QualificationID: "welder_argon_arc_amg6", Scope: "ent01/b1/wc", ValidUntil: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
			{PersonID: "W22", QualificationID: "welder_argon_arc_amg6", Scope: "ent01/b1/wc", ValidUntil: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
		},
		Stamps: []Stamp{{StampID: "OTK-01-SV", PersonID: "INS-01", Kind: "weld", Scope: "ent01/b1/wc"}},
	})
	for i, x := range []string{"W21", "W22"} {
		wp := wpWeld
		if x == "W22" {
			wp = "WP-WELD-1"
		}
		if err := p.Apply(rec(t, int64(10+i), catalog.AccessAssignmentSet, ev.AccessAssignmentSetV1{PersonID: ev.PersonRef(x), WorkplaceID: ev.ObjectID(wp),
			ShiftID: "SHIFT-1", AssigneeRole: "performer"})); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func pass(t *testing.T, pr *Presence, seq int64, person, zone string, enter bool) {
	t.Helper()
	r := rec(t, seq, catalog.AccessZonePassed, nil)
	r.Data, r.EventID, r.OccurredAt = ZonePassRecord(person, zone, "R-1", enter), "pass-"+strconv.FormatInt(seq, 10), at37
	if err := pr.Apply(r); err != nil {
		t.Fatal(err)
	}
}

func token(t *testing.T, pr *Presence, seq int64, person, wp string, in bool) {
	t.Helper()
	r := rec(t, seq, catalog.AccessTokenPresenceChanged, ev.AccessTokenPresenceChangedV1{PersonID: ev.PersonRef(person), WorkplaceID: ev.ObjectID(wp), Present: in})
	r.EventID = "tok-" + person
	if err := pr.Apply(r); err != nil {
		t.Fatal(err)
	}
}

func zoneOf(string) string { return zoneWeld }

func TestPostPresence(t *testing.T) {
	pr := NewPresence()
	if got := pr.PostPresence(wpWeld, zoneWeld, "W21"); got != PresenceUnknown {
		t.Fatalf("без данных СКУД — неизвестно, а не %s", got)
	}
	if got := pr.PostPresence(wpWeld, zoneWeld, ""); got != PresenceNotAssigned {
		t.Fatal(got)
	}
	pass(t, &pr, 1, "W21", zoneWeld, true)
	if got := pr.PostPresence(wpWeld, zoneWeld, "W21"); got != PresenceKeyMissing {
		t.Fatalf("в зоне без ключа: %s", got)
	}
	token(t, &pr, 2, "W21", wpWeld, true)
	if got := pr.PostPresence(wpWeld, zoneWeld, "W21"); got != PresencePresent {
		t.Fatalf("в зоне с ключом: %s", got)
	}
	pass(t, &pr, 3, "W21", zoneWeld, false)
	if got := pr.PostPresence(wpWeld, zoneWeld, "W21"); got != PresenceOwnerAbsent {
		t.Fatalf("ключ вставлен, владельца нет: %s", got)
	}
	token(t, &pr, 4, "W21", wpWeld, false)
	if got := pr.PostPresence(wpWeld, zoneWeld, "W21"); got != PresenceAbsent {
		t.Fatalf("нет ни в зоне, ни ключа: %s", got)
	}
	if pr.Seq != 4 {
		t.Fatalf("seq %d", pr.Seq)
	}
}

func TestCheckAdmission(t *testing.T) {
	pol := policy37(t)
	pr := NewPresence()
	rq := AdmissionRequest{WorkplaceID: wpWeld, Scope: wpScope, Zone: zoneWeld, PersonID: "W21", At: at37, Token: true, PIN: true}
	// Не в зоне по СКУД — отказ с кодом access.not_in_zone.
	_, failed := CheckAdmission(pol, pr, rq)
	if !slices.Equal(failed, []string{CheckNotInZone}) {
		t.Fatalf("проверки: %v", failed)
	}
	var r *kernel.Refusal
	if err := AdmissionRefusal(rq, failed); !errors.As(err, &r) || r.Code != errcodes.AccessNotInZone {
		t.Fatalf("отказ: %v", err)
	}
	pass(t, &pr, 1, "W21", zoneWeld, true)
	post, failed := CheckAdmission(pol, pr, rq)
	if len(failed) != 0 || post.ShiftID != "SHIFT-1" {
		t.Fatalf("допуск: %v %+v", failed, post)
	}
	// Без ключа и без PIN.
	rq2 := rq
	rq2.Token, rq2.PIN = false, false
	if _, failed := CheckAdmission(pol, pr, rq2); !slices.Equal(failed, []string{CheckNoToken}) {
		t.Fatal(failed)
	}
	rq2.Token = true
	if _, failed := CheckAdmission(pol, pr, rq2); !slices.Equal(failed, []string{CheckWrongPIN}) {
		t.Fatal(failed)
	}
	// Чужой пост: не назначен, и нет квалификации в области механического цеха.
	o := AdmissionRequest{WorkplaceID: "WP-CNC-1", Scope: "ent01/b1/mc/cnc/wp1", Zone: "Z-MC", PersonID: "W21", At: at37, Token: true, PIN: true}
	pass(t, &pr, 2, "W21", "Z-MC", true)
	if _, failed := CheckAdmission(pol, pr, o); !slices.Equal(failed, []string{CheckNotAssigned, CheckNoRoleInScope, CheckQualification}) {
		t.Fatalf("чужой пост: %v", failed)
	}
	if err := AdmissionRefusal(o, []string{CheckNotAssigned, CheckQualification}); !errors.As(err, &r) || r.Code != errcodes.AccessAdmissionDenied {
		t.Fatalf("отказ: %v", err)
	}
	// Истёкшая аттестация W22 — отказ в допуске.
	pass(t, &pr, 3, "W22", zoneWeld, true)
	w22 := AdmissionRequest{WorkplaceID: "WP-WELD-1", Scope: "ent01/b1/wc/weld/wp1", Zone: zoneWeld, PersonID: "W22", At: at37, Token: true, PIN: true}
	if _, failed := CheckAdmission(pol, pr, w22); !slices.Equal(failed, []string{CheckQualification}) {
		t.Fatalf("истёкшая аттестация: %v", failed)
	}
}

// Мастер не может назначить сварщика с истёкшей аттестацией (FR-81; «Что ожидаем в итоге»).
func TestAssignExpiredWelder(t *testing.T) {
	pol := policy37(t)
	rq := AssignmentRequest{WorkplaceID: "WP-WELD-1", WorkplaceScope: "ent01/b1/wc/weld/wp1", ShiftID: "SHIFT-2", PersonID: "W22", AssigneeRole: AssigneePerformer, At: at37}
	var r *kernel.Refusal
	if err := GuardAssignment(pol, rq); !errors.As(err, &r) || r.Code != errcodes.AccessNotQualified {
		t.Fatalf("назначение с истёкшей аттестацией: %v", err)
	}
	rq.PersonID = "W21"
	if err := GuardAssignment(pol, rq); err != nil {
		t.Fatal(err)
	}
	// Отзыв аттестации — с этого момента назначить нельзя.
	if err := pol.Apply(rec(t, 20, catalog.AccessQualificationRevoked, ev.AccessQualificationRevokedV1{PersonID: "W21", QualificationID: "welder_argon_arc_amg6",
		EffectiveFrom: ev.Timestamp(at37.Add(-time.Minute))})); err != nil {
		t.Fatal(err)
	}
	if err := GuardAssignment(pol, rq); !errors.As(err, &r) || r.Code != errcodes.AccessNotQualified {
		t.Fatalf("после отзыва: %v", err)
	}
}

// Клеймо по виду контроля: действующее — есть; отозванное и чужой вид — нет (FR-145).
func TestStampByKind(t *testing.T) {
	pol := policy37(t)
	if _, ok := pol.StampFor("INS-01", "weld", "", at37); !ok {
		t.Fatal("клеймо сварки действует")
	}
	if _, ok := pol.StampFor("INS-01", "final", "", at37); ok {
		t.Fatal("клейма окончательного контроля нет")
	}
	if err := pol.Apply(rec(t, 30, catalog.PolicyStampRevoked, ev.PolicyStampRevokedV1{PersonID: "INS-01", StampID: "OTK-01-SV",
		EffectiveFrom: ev.Timestamp(at37.Add(-time.Minute)), Reason: ev.Reason{Text: "утеря"}})); err != nil {
		t.Fatal(err)
	}
	if _, ok := pol.StampFor("INS-01", "weld", "", at37); ok {
		t.Fatal("отозванное клеймо не действует")
	}
}

func TestDeviations(t *testing.T) {
	pol := policy37(t)
	pr := NewPresence()
	pass(t, &pr, 1, "W21", zoneWeld, true)
	token(t, &pr, 2, "W21", wpWeld, true)
	if d := TokenWithoutPresence(pr, zoneOf); len(d) != 0 {
		t.Fatalf("в зоне с ключом — отклонения нет: %v", d)
	}
	pass(t, &pr, 3, "W21", zoneWeld, false)
	d := TokenWithoutPresence(pr, zoneOf)
	if len(d) != 1 || d[0].Kind != DeviationTokenWithoutPresence || d[0].WorkplaceID != wpWeld || d[0].Cause == "" {
		t.Fatalf("ключ вставлен, владельца нет: %+v", d)
	}
	// Повторная проверка даёт тот же ключ основания (журнал не запишет дважды).
	if d2 := TokenWithoutPresence(pr, zoneOf); d2[0].Key != d[0].Key {
		t.Fatal("ключ основания не однозначен")
	}
	// По графику: W22 назначен в SHIFT-1, ключа нет; до конца льготы — рано.
	start := at37.Add(-time.Hour)
	if got := ScheduledWithoutToken(pol, pr, "SHIFT-1", start, start.Add(5*time.Minute), 15*time.Minute); got != nil {
		t.Fatalf("рано: %v", got)
	}
	got := ScheduledWithoutToken(pol, pr, "SHIFT-1", start, at37, 15*time.Minute)
	if len(got) != 1 || got[0].PersonID != "W22" || got[0].Kind != DeviationScheduledWithoutKey {
		t.Fatalf("по графику должен быть: %+v", got)
	}
}

func TestPresenceSessions(t *testing.T) {
	pr := NewPresence()
	adm := rec(t, 1, catalog.AccessWorkplaceAdmitted, ev.AccessWorkplaceAdmittedV1{WorkplaceID: wpWeld, WorkplaceSessionID: "0190a1b2-0000-7000-8000-000000000001", PersonID: "W21"})
	if err := pr.Apply(adm); err != nil {
		t.Fatal(err)
	}
	if s, ok := pr.SessionOf("W21"); !ok || s.WorkplaceID != wpWeld {
		t.Fatal("сеанс открыт")
	}
	c := pr.Clone()
	rel := rec(t, 2, catalog.AccessWorkplaceRevoked, ev.AccessWorkplaceRevokedV1{WorkplaceID: wpWeld, WorkplaceSessionID: "0190a1b2-0000-7000-8000-000000000001", Cause: "zone_exit"})
	if err := pr.Apply(rel); err != nil {
		t.Fatal(err)
	}
	if _, ok := pr.Session(wpWeld); ok {
		t.Fatal("сеанс снят")
	}
	if _, ok := c.Session(wpWeld); !ok {
		t.Fatal("копия не зависит от свёртки")
	}
}
