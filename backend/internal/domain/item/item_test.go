package item_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
)

var t0 = time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)

const id = "ENT01:F-1"

var env = item.Env{Types: map[string]item.TypeDef{"FL-100.00.000": {
	Zones: []item.ZoneDef{{ID: "W-1.U1", Name: "Шов W-1, участок У1"}, {ID: "S-1", Name: "Канавка уплотнения"}, {ID: "CAV", Name: "Полость"}},
	Links: []item.LinkDef{{ID: "J-1", Kind: "bolted_joint", Zones: []string{"J-1"}, ClosesAccessTo: []string{"S-1", "CAV"}}},
}}}

type seqRec struct {
	n  int
	in []kernel.Record
}

func (s *seqRec) add(typ catalog.Type, at time.Duration, data any) kernel.Record {
	s.n++
	raw, _ := json.Marshal(data)
	info, _ := catalog.Lookup(typ)
	r := kernel.Record{Seq: int64(s.n), EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", s.n), Type: typ, Kind: info.Kind, ItemID: id,
		Stream: "item:" + id, OccurredAt: t0.Add(at), ReceivedAt: t0.Add(at), Data: raw}
	s.in = append(s.in, r)
	return r
}

func (s *seqRec) fold() item.State {
	var st item.State
	in := slices.Clone(s.in)
	slices.SortStableFunc(in, func(a, b kernel.Record) int { return a.OccurredAt.Compare(b.OccurredAt) })
	for _, r := range in {
		st = item.Reduce(st, r, env)
	}
	return st
}

func registered() *seqRec {
	s := &seqRec{}
	s.add(catalog.ItemItemRegistered, 0, map[string]any{"item_id": id, "item_type_id": "FL-100.00.000", "item_revision": "Б",
		"process_version_hash": "streebog256:00", "normative_rev": "r1", "lot_ids": []string{"L-1"}})
	return s
}

func refusal(t *testing.T, err error) errcodes.Code {
	t.Helper()
	var r *kernel.Refusal
	if !errors.As(err, &r) {
		t.Fatalf("ожидали отказ, получили %v", err)
	}
	return r.Code
}

// Паспорт: регистрация, носители, зоны по КД, закрытие доступа связью,
// вмешательство и его закрытие (FR-42, FR-46, FR-21, AD-16).
func TestPassportZonesAndCarriers(t *testing.T) {
	s := registered()
	s.add(catalog.ItemCarrierApplied, time.Minute, map[string]any{"carrier_type": "tag_qr", "value": "Q-1", "is_temporary": true})
	s.add(catalog.ItemCarrierApplied, time.Hour, map[string]any{"carrier_type": "dpm_datamatrix", "value": "DM-1", "is_temporary": false, "replaces_value": "Q-1"})
	insp := s.add(catalog.InspectionResultRecorded, 2*time.Hour, map[string]any{"outcome": "no_defect_indicated", "zone_ids": []string{"S-1", "W-1.U1"}})
	s.add(catalog.ItemAssemblyRecorded, 3*time.Hour, map[string]any{"assembly_item_id": id, "component_lot_id": "L-B", "component_type_id": "FL-100.00.005",
		"quantity": 12, "position": "J-1", "binding_method": "container_cell"})
	st := s.fold()
	if !st.Registered || st.ItemTypeID != "FL-100.00.000" || st.Identification() != item.IdentUnique {
		t.Fatalf("регистрация и идентификация: %+v / %s", st, st.Identification())
	}
	var tag, dm item.Carrier
	for _, c := range st.Carriers {
		switch c.Value {
		case "Q-1":
			tag = c
		case "DM-1":
			dm = c
		}
	}
	if tag.State != item.CarrierReplaced || tag.RemovedAt == nil || dm.State != item.CarrierApplied {
		t.Fatalf("перемаркировка: %+v %+v", tag, dm)
	}
	zones := map[string]item.Zone{}
	for _, z := range st.Zones {
		zones[z.ZoneID] = z
	}
	if zones["S-1"].Status != item.ZoneInspected || !zones["S-1"].Closed || zones["S-1"].ClosedBy != "J-1" || !zones["CAV"].Closed ||
		zones["W-1.U1"].Closed || zones["S-1"].LastInspection != insp.EventID {
		t.Fatalf("зоны: %+v", zones)
	}
	// Выпуск невозможен при открытом вмешательстве; открытие делает зону «устаревшей».
	s.add(catalog.ItemInterventionOpened, 4*time.Hour, map[string]any{"intervention_id": "IV-1", "zone_ids": []string{"S-1"}, "purpose": map[string]any{"text": "замена уплотнения"}})
	st = s.fold()
	if z := st.Zones[1]; z.ZoneID != "S-1" || z.Status != item.ZoneStale || z.Closed || z.OpenIntervention != "IV-1" {
		t.Fatalf("вмешательство: %+v", z)
	}
	if code := refusal(t, item.Guard(st, env, kernel.Command{Action: "item.release.record"})); code != errcodes.NonconformityInterventionOpen {
		t.Fatalf("выпуск при вмешательстве: %s", code)
	}
	if code := refusal(t, item.Guard(st, env, kernel.Command{Action: "item.intervention.close", Payload: item.InterventionCmd{ID: "IV-9"}})); code != errcodes.ItemInterventionNotOpen {
		t.Fatalf("закрыть чужое: %s", code)
	}
	if code := refusal(t, item.Guard(st, env, kernel.Command{Action: "item.intervention.open", Payload: item.InterventionCmd{Zones: []string{"X"}}})); code != errcodes.ItemZoneUnknown {
		t.Fatalf("неизвестная зона: %s", code)
	}
	s.add(catalog.ItemInterventionClosed, 5*time.Hour, map[string]any{"intervention_id": "IV-1", "recheck_event_ids": []string{insp.EventID}})
	st = s.fold()
	if z := st.Zones[1]; z.Status != item.ZoneInspected || !z.Closed || z.OpenIntervention != "" {
		t.Fatalf("закрытие вмешательства: %+v", z)
	}
	if err := item.Guard(st, env, kernel.Command{Action: "item.release.record"}); err != nil {
		t.Fatal(err)
	}
	if code := refusal(t, item.Guard(st, env, kernel.Command{Action: "item.item.register"})); code != errcodes.ItemAlreadyRegistered {
		t.Fatal(code)
	}
}

// Нечитаемый носитель → «идентификация под сомнением» → изоляция до
// повторной идентификации человеком (AD-16); реакция остаётся после
// подтверждения — снятие сомнения не выглядит исчезновением защиты (AD-3).
func TestIdentificationQuestioned(t *testing.T) {
	s := registered()
	s.add(catalog.ItemCarrierApplied, time.Minute, map[string]any{"carrier_type": "dpm_datamatrix", "value": "DM-1", "is_temporary": false})
	bad := s.add(catalog.ItemCarrierVerified, time.Hour, map[string]any{"carrier_type": "dpm_datamatrix", "read_outcome": "unreadable"})
	st := s.fold()
	if !st.Questioned() || st.Identification() != item.IdentUnidentified {
		t.Fatalf("сомнение: %+v", st.Questions)
	}
	out := item.React(st, env)
	if len(out.Reactions) != 1 || out.Reactions[0].Type != catalog.ItemIdentificationQuestioned || out.Reactions[0].Slot.TriggerKey != bad.EventID {
		t.Fatalf("реакция: %+v", out)
	}
	if code := refusal(t, item.Guard(st, env, kernel.Command{Action: "item.assembly.record"})); code != errcodes.ItemIdentificationQuestioned {
		t.Fatalf("изоляция: %s", code)
	}
	questioned := out.Reactions[0].ID(1)
	if err := item.Guard(st, env, kernel.Command{Action: "item.identification.confirm", Payload: item.ConfirmCmd{QuestionedEventID: questioned}}); err != nil {
		t.Fatal(err)
	}
	if code := refusal(t, item.Guard(st, env, kernel.Command{Action: "item.identification.confirm", Payload: item.ConfirmCmd{QuestionedEventID: "00000000-0000-7000-8000-999999999999"}})); code != errcodes.ItemIdentificationNotQuestioned {
		t.Fatal(code)
	}
	s.add(catalog.ItemIdentificationConfirmed, 2*time.Hour, map[string]any{"method": "manual_entry", "questioned_event_id": questioned})
	st = s.fold()
	if st.Questioned() {
		t.Fatalf("после подтверждения: %+v", st.Questions)
	}
	if out := item.React(st, env); len(out.Reactions) != 1 {
		t.Fatalf("реакция истории: %+v", out)
	}
	// Снят последний физический носитель — идентификация утрачена.
	s.add(catalog.ItemCarrierRemoved, 3*time.Hour, map[string]any{"carrier_type": "dpm_datamatrix", "value": "DM-1"})
	if st = s.fold(); !st.Questioned() || st.OpenQuestions()[0].Cause != "carrier_missing" {
		t.Fatalf("носитель снят: %+v", st.Questions)
	}
}

// Паспорт показывает адресованные записи стадии: результат свидетеля садки,
// блок партии, неоднозначную привязку с кандидатами (FR-15, FR-45, FR-34).
func TestStageRecordsInPassport(t *testing.T) {
	s := registered()
	s.add(catalog.GenealogyLinkAdded, time.Minute, map[string]any{"child_item_id": id, "group_id": "CH-7", "relation": "grouped_with", "basis_event_id": "00000000-0000-7000-8000-000000000077"})
	s.add(catalog.GenealogyWitnessPropagated, time.Hour, map[string]any{"group_id": "CH-7", "group_kind": "charge", "witness_item_id": "ENT01:W",
		"inspection_event_id": "00000000-0000-7000-8000-000000000078", "outcome": "no_defect_indicated"})
	s.add(catalog.GenealogyContainmentPropagated, 2*time.Hour, map[string]any{"level": "lot_hold", "source": "lot", "source_event_id": "00000000-0000-7000-8000-000000000079",
		"lot_id": "L-1", "basis": []string{"00000000-0000-7000-8000-000000000079"}})
	amb := s.add(catalog.BindingLinkResolved, 3*time.Hour, map[string]any{"subject_event_id": "00000000-0000-7000-8000-000000000080", "candidates": []string{id, "ENT01:F-2"},
		"binding_basis": "carrier", "binding_reliability": "ambiguous"})
	st := s.fold()
	if len(st.Witnesses) != 1 || st.Witnesses[0].WitnessID != "ENT01:W" || st.HoldLevel() != "lot_hold" || len(st.Links) != 1 {
		t.Fatalf("записи стадии: %+v", st)
	}
	if !st.Questioned() || st.Identification() != item.IdentAmbiguous || len(st.OpenQuestions()[0].Candidates) != 2 {
		t.Fatalf("неоднозначная привязка: %+v", st.Questions)
	}
	// Контролёр привязал событие к другому изделию — сомнение снято, событие не наше.
	s.add(catalog.BindingLinkResolved, 4*time.Hour, map[string]any{"subject_event_id": "00000000-0000-7000-8000-000000000080", "item_id": "ENT01:F-2",
		"previous_item_id": id, "binding_basis": "manual", "binding_reliability": "unique"})
	st = s.fold()
	if st.Questioned() || len(st.BoundEvents()) != 0 || st.Bindings[0].BoundTo != "ENT01:F-2" {
		t.Fatalf("после ручной привязки: %+v / %+v", st.Questions, st.Bindings)
	}
	_ = amb
}
