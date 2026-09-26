package access_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ant/internal/application/access"
	itemapp "ant/internal/application/item"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/crypto"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
	"ant/internal/domain/kernel"
)

// Эпик 37: приём СКУД, панель «Посты» вживую, допуск к рабочему месту и его
// снятие, отказ с кодом, отклонения присутствия, отзыв квалификации.

// memPresence — присутствие в памяти: проекция (PresenceSource), запись
// фактов СКУД и отклонений (PresenceWriter) и запись фактов допуска (itemapp.Writer).
type memPresence struct {
	pr      accessdom.Presence
	passes  []accessdom.Record
	wpRecs  []accessdom.Record
	seq     int64
	seen    map[string]bool
	devs    []accessdom.Deviation
	revoked []string
	denied  [][]string
	facts   []itemapp.Record
}

func newMemPresence() *memPresence {
	return &memPresence{pr: accessdom.NewPresence(), seq: 1000, seen: map[string]bool{}}
}

func (m *memPresence) Presence(context.Context) (accessdom.Presence, error) { return m.pr.Clone(), nil }
func (m *memPresence) Passes(context.Context) ([]accessdom.Record, error) {
	return m.passes, nil
}

func (m *memPresence) apply(t catalog.Type, data any, id string) {
	m.seq++
	b, _ := jsonMarshal(data)
	r := accessdom.Record{Seq: m.seq, Type: string(t), Data: b, OccurredAt: now26, EventID: id}
	_ = m.pr.Apply(r)
	if t == catalog.AccessZonePassed {
		m.passes = append(m.passes, r)
	} else {
		m.wpRecs = append(m.wpRecs, r)
	}
}

// Stream — записи потока поста (WorkplaceLog): всё, кроме проходов, у поста.
func (m *memPresence) Stream(_ context.Context, wp string, _ platform.Moment) ([]accessdom.Record, error) {
	var out []accessdom.Record
	for _, r := range m.wpRecs {
		var d struct {
			WorkplaceID string `json:"workplace_id"`
		}
		if json.Unmarshal(r.Data, &d) == nil && d.WorkplaceID == wp {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memPresence) ZonePassed(_ context.Context, p access.ZonePassIn) (bool, error) {
	if m.seen[p.SourceEventID] {
		return false, nil
	}
	m.seen[p.SourceEventID] = true
	dir := ev.AccessZonePassedV1DirectionExit
	if p.Enter {
		dir = ev.AccessZonePassedV1DirectionEnter
	}
	m.apply(catalog.AccessZonePassed, ev.AccessZonePassedV1{PersonID: ev.PersonRef(p.PersonID), ZoneID: ev.ObjectID(p.ZoneID), Direction: dir}, "skud-"+p.SourceEventID)
	return true, nil
}

func (m *memPresence) Revoked(_ context.Context, s accessdom.WorkplaceSession, cause, _ string, _ time.Time) error {
	m.revoked = append(m.revoked, s.WorkplaceID+":"+cause)
	m.apply(catalog.AccessWorkplaceRevoked, ev.AccessWorkplaceRevokedV1{WorkplaceID: ev.ObjectID(s.WorkplaceID), WorkplaceSessionID: ev.UUID(s.SessionID),
		Cause: ev.AccessWorkplaceRevokedV1Cause(cause)}, "")
	return nil
}

func (m *memPresence) Deviation(_ context.Context, d accessdom.Deviation, _ time.Time) error {
	if !m.seen[d.Key] {
		m.seen[d.Key] = true
		m.devs = append(m.devs, d)
	}
	return nil
}

func (m *memPresence) AdmissionDenied(_ context.Context, _, _ string, failed []string, _ time.Time) error {
	m.denied = append(m.denied, failed)
	return nil
}

// Write — факты допуска (itemapp.Writer): в память и в свёртку присутствия.
func (m *memPresence) Write(_ context.Context, _ kernel.Module, recs []itemapp.Record) (platform.Receipt, error) {
	for _, r := range recs {
		m.facts = append(m.facts, r)
		m.apply(r.Type, r.Data, "")
	}
	return platform.Receipt{Seq: m.seq, EventIDs: []string{"e"}}, nil
}

type shifts37 struct{}

func (shifts37) ShiftsAt(context.Context, time.Time) ([]access.ShiftWindow, error) {
	return []access.ShiftWindow{{ID: "S1", Start: now26.Add(-time.Hour), End: now26.Add(7 * time.Hour)}}, nil
}

func dir37() *access.Directory {
	d := rosterDir()
	for i := range d.Workplaces {
		d.Workplaces[i].Zone = map[string]string{"WS-WC": "Z-WC", "WS-MC": "Z-MC"}[d.Workplaces[i].Workshop]
	}
	return d
}

func TestAdmissionAndPresence(t *testing.T) {
	ctx := context.Background()
	pol := rosterPolicy()
	w := &applyWriter{pol: &pol, n: 10}
	m := newMemPresence()
	s := access.NewService(access.WithPolicy(ptrPolicy{&pol}), access.WithDecisions(w, clock26), access.WithDirectory(dir37()),
		access.WithLiveRoster(m), access.WithPresence(m, m, shifts37{}), access.WithWorkplaceLog(m))
	if _, err := s.SetAssignment(as("FOR-WC"), access.SetAssignment{PersonID: "W21", WorkplaceID: "WP-WELD-1", ShiftID: "S1", AssigneeRole: "performer"}); err != nil {
		t.Fatal(err)
	}
	presence := func() string {
		t.Helper()
		posts, err := s.Workplaces(ctx, "WS-WC", platform.Moment{})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range posts.Items {
			if r.WorkplaceID == "WP-WELD-1" {
				return r.Presence
			}
		}
		return ""
	}
	if got := presence(); got != "unknown" {
		t.Fatalf("без СКУД: %s", got)
	}
	admit := access.AdmitWorkplace{KeyRef: "w21@1", PinVerified: true}
	// Не в зоне — отказ с кодом, отказ зафиксирован.
	if _, err := s.AdmitWorkplace(as("W21"), "WP-WELD-1", admit); code26(err) != errcodes.AccessNotInZone {
		t.Fatalf("не в зоне: %v", err)
	}
	if len(m.denied) != 1 || m.denied[0][0] != accessdom.CheckNotInZone {
		t.Fatalf("отказ не зафиксирован: %v", m.denied)
	}
	// Проход СКУД — панель видит «в зоне, ключа нет»; повтор опроса не пишется.
	in := access.ZonePassIn{SourceEventID: "1", PersonID: "W21", ZoneID: "Z-WC", Enter: true, At: now26}
	if n, err := s.IngestZonePasses(ctx, []access.ZonePassIn{in, in}); err != nil || n != 1 {
		t.Fatalf("приём: %d %v", n, err)
	}
	if got := presence(); got != "key_missing" {
		t.Fatalf("в зоне без ключа: %s", got)
	}
	// Уже прошло больше 15 мин смены, ключа нет — отклонение «по графику должен быть» мастеру.
	if len(m.devs) != 1 || m.devs[0].Kind != accessdom.DeviationScheduledWithoutKey {
		t.Fatalf("по графику: %+v", m.devs)
	}
	// Без PIN — отказ.
	if _, err := s.AdmitWorkplace(as("W21"), "WP-WELD-1", access.AdmitWorkplace{KeyRef: "w21@1"}); code26(err) != errcodes.AccessAdmissionDenied {
		t.Fatalf("без PIN: %v", err)
	}
	// Чужим ключом — ключа нет.
	if _, err := s.AdmitWorkplace(as("W21"), "WP-WELD-1", access.AdmitWorkplace{KeyRef: "w22@1", PinVerified: true}); code26(err) != errcodes.AccessAdmissionDenied {
		t.Fatalf("чужой ключ: %v", err)
	}
	// Допуск: запись допуска и ключа, присутствие — на месте.
	if _, err := s.AdmitWorkplace(as("W21"), "WP-WELD-1", admit); err != nil {
		t.Fatalf("допуск: %v", err)
	}
	if len(m.facts) != 2 || m.facts[0].Type != catalog.AccessWorkplaceAdmitted || m.facts[1].Type != catalog.AccessTokenPresenceChanged {
		t.Fatalf("записи допуска: %+v", m.facts)
	}
	if got := presence(); got != "present" {
		t.Fatalf("на месте: %s", got)
	}
	list, _ := s.Assignments(ctx, "S1", "", platform.Moment{})
	if len(list.Items) != 1 || !list.Items[0].Admitted {
		t.Fatalf("назначение с допуском: %+v", list)
	}
	sess, err := s.Session(as("W21"))
	if err != nil || sess.Workplace == nil || sess.Workplace.ID != "WP-WELD-1" {
		t.Fatalf("рабочее место сеанса: %+v %v", sess, err)
	}
	// Вышел из зоны с ключом — тревога администратору и снятие допуска.
	if _, err := s.IngestZonePasses(ctx, []access.ZonePassIn{{SourceEventID: "2", PersonID: "W21", ZoneID: "Z-WC", At: now26}}); err != nil {
		t.Fatal(err)
	}
	if got := presence(); got != "owner_absent" {
		t.Fatalf("владельца нет: %s", got)
	}
	if len(m.devs) != 2 || m.devs[1].Kind != accessdom.DeviationTokenWithoutPresence || len(m.revoked) != 1 || m.revoked[0] != "WP-WELD-1:zone_exit" {
		t.Fatalf("отклонение и снятие: %+v %v", m.devs, m.revoked)
	}
	// История поста — проходы СКУД назначенного на пост.
	h, err := s.WorkplaceHistory(ctx, "WP-WELD-1", platform.Moment{}, platform.Page{})
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]int{}
	for _, e := range h.Items {
		kinds[e.Kind]++
	}
	if kinds["zone_in"] != 1 || kinds["zone_out"] != 1 || kinds["admitted"] != 1 || kinds["token_in"] != 1 || kinds["revoked"] != 1 {
		t.Fatalf("история поста: %v", kinds)
	}
	if h.Items[0].Kind == "zone_in" {
		t.Fatalf("новые сверху: %+v", h.Items[0])
	}
}

func TestReleaseAndQualificationRevoke(t *testing.T) {
	ctx := context.Background()
	pol := rosterPolicy()
	w := &applyWriter{pol: &pol, n: 10}
	m := newMemPresence()
	s := access.NewService(access.WithPolicy(ptrPolicy{&pol}), access.WithDecisions(w, clock26), access.WithDirectory(dir37()),
		access.WithLiveRoster(m), access.WithPresence(m, m, nil))
	if _, err := s.SetAssignment(as("FOR-WC"), access.SetAssignment{PersonID: "W21", WorkplaceID: "WP-WELD-1", ShiftID: "S1", AssigneeRole: "performer"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.IngestZonePasses(ctx, []access.ZonePassIn{{SourceEventID: "1", PersonID: "W21", ZoneID: "Z-WC", Enter: true, At: now26}}); err != nil {
		t.Fatal(err)
	}
	// Подписанный пакет команды — ключ вставлен и открыт PIN-ом.
	in := access.AdmitWorkplace{}
	in.CommandHeader.Signature = &crypto.DsseEnvelope{PayloadType: "application/vnd.ant.event+json; v=1", Payload: "e30=",
		Signatures: []crypto.DsseEnvelopeSignaturesElem{{Keyid: "w21@1", Sig: "c2ln"}}}
	if _, err := s.AdmitWorkplace(as("W21"), "WP-WELD-1", in); err != nil {
		t.Fatalf("допуск по подписанному пакету: %v", err)
	}
	if _, err := s.ReleaseWorkplace(as("W22"), "WP-WELD-1", access.ReleaseWorkplace{}); code26(err) != errcodes.AccessWrongWorkplace {
		t.Fatalf("чужой сеанс: %v", err)
	}
	if _, err := s.ReleaseWorkplace(as("W21"), "WP-WELD-1", access.ReleaseWorkplace{}); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.pr.Session("WP-WELD-1"); ok {
		t.Fatal("сеанс закрыт")
	}
	if _, ok := m.pr.Token("WP-WELD-1"); ok {
		t.Fatal("ключ извлечён")
	}
	// Снова допуск, затем отзыв аттестации: допуск снят, назначить нельзя.
	if _, err := s.AdmitWorkplace(as("W21"), "WP-WELD-1", access.AdmitWorkplace{KeyRef: "W21@1", PinVerified: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RevokeQualification(as("ADM-01"), "W21", access.RevokeQualification{QualificationID: "welder", Reason: access.AccessReason{Text: "истекла аттестация НАКС"}}); err != nil {
		t.Fatal(err)
	}
	if len(m.revoked) != 1 || m.revoked[0] != "WP-WELD-1:qualification_revoked" {
		t.Fatalf("снятие по отзыву квалификации: %v", m.revoked)
	}
	if _, err := s.SetAssignment(as("FOR-WC"), access.SetAssignment{PersonID: "W21", WorkplaceID: "WP-WELD-1", ShiftID: "S2", AssigneeRole: "performer"}); code26(err) != errcodes.AccessNotQualified {
		t.Fatalf("назначение после отзыва: %v", err)
	}
	// Выдать квалификацию снова — назначение проходит.
	until := now26.Add(24 * time.Hour)
	if _, err := s.GrantQualification(as("ADM-01"), "W21", access.GrantQualification{QualificationID: "welder", Scope: "ent01/b1/wc", ValidUntil: &until}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetAssignment(as("FOR-WC"), access.SetAssignment{PersonID: "W21", WorkplaceID: "WP-WELD-1", ShiftID: "S2", AssigneeRole: "performer"}); err != nil {
		t.Fatalf("после новой аттестации: %v", err)
	}
}

func TestStampRegistry(t *testing.T) {
	pol := rosterPolicy()
	pol.Stamps = append(pol.Stamps, accessdom.Stamp{StampID: "OTK-01-SV", PersonID: "INS-01", Kind: "weld", Scope: "ent01/b1/wc"})
	r := access.StampRegistry{Policy: ptrPolicy{&pol}}
	if id, ok, err := r.ValidStamp(context.Background(), "INS-01", "weld", now26); err != nil || !ok || id != "OTK-01-SV" {
		t.Fatalf("клеймо: %s %v %v", id, ok, err)
	}
	if _, ok, _ := r.ValidStamp(context.Background(), "INS-01", "final", now26); ok {
		t.Fatal("нет клейма окончательного контроля")
	}
}
