package itemtest

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	crossitemapp "ant/internal/application/crossitem"
	itemapp "ant/internal/application/item"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
)

type jcEntry = jc.JournalEntry

// Scenario — сквозной путь эпика 18 на живых операциях item и crossitem:
//
//  1. партия L-17 (плавка ПЛ-3) из 1С, регистрация кладовщиком, приёмка,
//     выдача на компонент C-1; сборка F-1 ← U-1 ← C-1;
//  2. садка CH-1 (A, B, C, свидетель W): результат свидетеля — в паспортах A, B, C;
//  3. партия L-17 отвергнута после выдачи: блок партии доходит до собранных F-1, U-1;
//  4. событие без изделия с носителем ячейки тары, где лежат X и Y: кандидаты,
//     «идентификация под сомнением» у обоих, ручная привязка к X снимает
//     сомнение у Y; событие видно в паспорте X;
//  5. генеалогия вверх и вниз, «партия / плавка → все изделия», поиск по
//     носителю, журнал изменений паспорта; гарды: цикл сборки, чужой носитель.
func Scenario(t *testing.T, w *World) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	ctx = Principal(ctx, "qc-7")
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	code := func(err error) errcodes.Code {
		t.Helper()
		var pe *platform.Error
		if !errors.As(err, &pe) {
			t.Fatalf("ожидали отказ с кодом, получили %v", err)
		}
		return pe.Code
	}
	reg := func(local string, lots ...string) string {
		t.Helper()
		rc, err := w.Item.Register(ctx, itemapp.RegisterItem{CommandHeader: w.Cmd(), LocalID: local, ItemTypeID: "FL-100.00.000", ItemRevision: "Б", LotIDs: lots})
		must(err)
		if len(rc.EventIDs) != 1 {
			t.Fatalf("регистрация %s: %+v", local, rc)
		}
		return "ENT01:" + local
	}

	// ── 1. партия и сборка ──
	w.At(0)
	_, err := w.Fact(ctx, catalog.ErpLotReceived, "", "lot:L-17", "", w.At(1), map[string]any{"lot_id": "L-17", "external_system": "1c",
		"external_number": "П-000123", "supplier_id": "SUP-1", "item_type_id": "FL-100.01.001", "heat_no": "ПЛ-3", "quantity": 10})
	must(err)
	must(w.Settle(ctx))
	_, err = w.Cross.RegisterLot(ctx, "L-17", crossitemapp.RegisterLot{CommandHeader: w.Cmd(), ActualQuantity: 10, PackagingOK: true, CertificatePresent: true})
	must(err)
	must(w.Settle(ctx))
	if _, err := w.Cross.IssueLot(ctx, "L-17", crossitemapp.IssueLot{CommandHeader: w.Cmd(), Quantity: 1}); code(err) != errcodes.ItemLotNotAccepted {
		t.Fatal("выдача непринятой партии")
	}
	_, err = w.Fact(ctx, catalog.DecisionLotResolved, "", "lot:L-17", "", w.At(5), map[string]any{"lot_id": "L-17", "resolution": "accept", "method_event_ids": []string{}})
	must(err)
	w.At(10)
	c1, u1, f1 := reg("C-1"), reg("U-1"), reg("F-1")
	must(w.Settle(ctx))
	_, err = w.Cross.IssueLot(ctx, "L-17", crossitemapp.IssueLot{CommandHeader: w.Cmd(), Quantity: 1, ItemIDs: []string{c1}})
	must(err)
	w.At(20)
	_, err = w.Item.RecordAssembly(ctx, u1, itemapp.RecordAssembly{CommandHeader: w.Cmd(), ComponentItemID: c1, ComponentTypeID: "FL-100.01.001", BindingMethod: "dpm_datamatrix"})
	must(err)
	must(w.Settle(ctx))
	_, err = w.Item.RecordAssembly(ctx, f1, itemapp.RecordAssembly{CommandHeader: w.Cmd(), ComponentItemID: u1, ComponentTypeID: "FL-100.01.000", BindingMethod: "dpm_datamatrix"})
	must(err)
	must(w.Settle(ctx))
	if _, err := w.Item.RecordAssembly(ctx, c1, itemapp.RecordAssembly{CommandHeader: w.Cmd(), ComponentItemID: f1, ComponentTypeID: "FL-100.00.000", BindingMethod: "manual_entry"}); code(err) != errcodes.ItemAssemblyCycle {
		t.Fatal("цикл сборки не отклонён")
	}

	// ── 2. садка и образец-свидетель ──
	a, b, c, wit := reg("A"), reg("B"), reg("C"), reg("W")
	must(w.Settle(ctx))
	w.At(30)
	_, err = w.Cross.FormGroup(ctx, crossitemapp.FormGroup{CommandHeader: w.Cmd(), GroupID: "CH-1", Kind: "charge", ItemIDs: []string{a, b, c}, WitnessItemID: wit})
	must(err)
	must(w.Settle(ctx))
	insp, err := w.Fact(ctx, catalog.InspectionResultRecorded, wit, "", "", w.At(90), map[string]any{"outcome": "no_defect_indicated", "phase": "test",
		"method": "laboratory", "processing_state": "complete", "conclusion_ref": "ПИ-7"})
	must(err)
	must(w.Settle(ctx))
	for _, id := range []string{a, b, c} {
		p, err := w.Item.Passport(ctx, id, platform.Moment{})
		must(err)
		if len(p.Witnesses) != 1 || p.Witnesses[0].InspectionEventID != insp || p.Witnesses[0].WitnessItemID != wit || p.Witnesses[0].Outcome != "no_defect_indicated" {
			t.Fatalf("%s: результат свидетеля в паспорте: %+v", id, p.Witnesses)
		}
		if !slices.ContainsFunc(p.Entries, func(e itemapp.PassportEntry) bool { return e.EventType == string(catalog.GenealogyWitnessPropagated) }) {
			t.Fatalf("%s: запись свидетеля в ленте паспорта", id)
		}
	}
	gl, err := w.Cross.Groups(ctx, platform.Moment{})
	must(err)
	if len(gl.Items) != 1 || gl.Items[0].WitnessItemID != wit || len(gl.Items[0].ItemIDs) != 4 {
		t.Fatalf("садки: %+v", gl)
	}

	// ── 3. блок партии доходит до собранных изделий ──
	rej, err := w.Fact(ctx, catalog.DecisionLotResolved, "", "lot:L-17", "", w.At(120), map[string]any{"lot_id": "L-17", "resolution": "reject",
		"method_event_ids": []string{}, "reason": map[string]any{"text": "Повторный входной контроль: трещины"}})
	must(err)
	must(w.Settle(ctx))
	for _, id := range []string{c1, u1, f1} {
		p, err := w.Item.Passport(ctx, id, platform.Moment{})
		must(err)
		if len(p.Holds) != 1 || p.Holds[0].Level != "lot_hold" || p.Holds[0].LotID != "L-17" || p.Holds[0].SourceEventID != rej {
			t.Fatalf("%s: блок партии в паспорте: %+v", id, p.Holds)
		}
		if p.Status.Containment != "lot_hold" || p.Status.Summary != "hold" {
			t.Fatalf("%s: статус %+v", id, p.Status)
		}
	}
	card, err := w.Cross.Lot(ctx, "L-17", platform.Moment{})
	must(err)
	if card.Status != "rejected" || card.HeatNo != "ПЛ-3" || len(card.Items) != 3 || !card.Items[2].Assembled {
		t.Fatalf("карточка партии: %+v", card)
	}
	tr, err := w.Cross.Trace(ctx, "", "ПЛ-3", "", platform.Moment{})
	must(err)
	if len(tr.Items) != 3 || tr.Items[0].ItemID != c1 || tr.Items[2].ItemID != f1 || !tr.Items[2].Assembled {
		t.Fatalf("плавка → изделия: %+v", tr)
	}
	list, err := w.Item.List(ctx, itemapp.ItemFilter{LotID: "L-17"}, platform.Moment{}, platform.Page{})
	must(err)
	if len(list.Items) != 3 {
		t.Fatalf("изделия партии: %+v", list)
	}

	// ── 4. неоднозначное событие получает кандидатов ──
	x, y := reg("X"), reg("Y")
	must(w.Settle(ctx))
	w.At(130)
	for _, id := range []string{x, y} {
		_, err := w.Item.ApplyCarrier(ctx, id, itemapp.ApplyCarrier{CommandHeader: w.Cmd(), CarrierType: "container_cell", Value: "BOX-1/3", IsTemporary: true})
		must(err)
	}
	_, err = w.Item.ApplyCarrier(ctx, x, itemapp.ApplyCarrier{CommandHeader: w.Cmd(), CarrierType: "tag_qr", Value: "Q-X", IsTemporary: true})
	must(err)
	must(w.Settle(ctx))
	if _, err := w.Item.ApplyCarrier(ctx, y, itemapp.ApplyCarrier{CommandHeader: w.Cmd(), CarrierType: "tag_qr", Value: "Q-X"}); code(err) != errcodes.ItemCarrierInUse {
		t.Fatal("чужой носитель не отклонён")
	}
	amb, err := w.Fact(ctx, catalog.InspectionResultRecorded, "", "global", "container_cell:BOX-1/3", w.At(140), map[string]any{"outcome": "defect_indicated",
		"phase": "after_operation", "method": "visual", "processing_state": "complete"})
	must(err)
	must(w.Settle(ctx))
	for _, id := range []string{x, y} {
		p, err := w.Item.Passport(ctx, id, platform.Moment{})
		must(err)
		if p.Identification != "ambiguous" || len(p.Questions) != 1 || !p.Questions[0].Open || len(p.Questions[0].Candidates) != 2 || p.Status.Position != "isolated" {
			t.Fatalf("%s: неоднозначная привязка: %s %+v %+v", id, p.Identification, p.Questions, p.Status)
		}
		if slices.ContainsFunc(p.Entries, func(e itemapp.PassportEntry) bool { return e.EventID == amb }) {
			t.Fatalf("%s: неоднозначное событие не угадывается в паспорт", id)
		}
	}
	ub, err := w.Cross.Unbound(ctx, platform.Moment{})
	must(err)
	if len(ub.Items) != 1 || ub.Items[0].EventID != amb || len(ub.Items[0].Candidates) != 2 || ub.Items[0].BoundTo != "" {
		t.Fatalf("события без изделия: %+v", ub)
	}
	if _, err := w.Item.RecordAssembly(ctx, x, itemapp.RecordAssembly{CommandHeader: w.Cmd(), ComponentLotID: "L-9", ComponentTypeID: "FL-100.00.005", BindingMethod: "container_cell"}); code(err) != errcodes.ItemIdentificationQuestioned {
		t.Fatal("изоляция при сомнении в идентификации")
	}
	w.At(150)
	_, err = w.Cross.AssignBinding(ctx, crossitemapp.AssignBinding{CommandHeader: w.Cmd(), SubjectEventID: amb, ItemID: x, Method: "select_expected",
		Reason: crossitemapp.BindingReason{Text: "по маршрутной карте"}})
	must(err)
	must(w.Settle(ctx))
	px, err := w.Item.Passport(ctx, x, platform.Moment{})
	must(err)
	py, err := w.Item.Passport(ctx, y, platform.Moment{})
	must(err)
	if !slices.ContainsFunc(px.Entries, func(e itemapp.PassportEntry) bool {
		return e.EventID == amb && e.Bound && e.BindingBasis == "manual" && e.EventType == string(catalog.InspectionResultRecorded)
	}) {
		t.Fatalf("X: привязанное событие в паспорте: %+v", px.Entries)
	}
	if py.Questions[0].Open || py.Status.Position == "isolated" || px.Questions[0].Open || px.Status.Position == "isolated" {
		t.Fatalf("сомнение снято ручной привязкой: X %+v, Y %+v", px.Questions, py.Questions)
	}
	// Нечитаемый носитель — снова под сомнением; снимает человек с подписью (AD-16).
	_, err = w.Fact(ctx, catalog.ItemCarrierVerified, x, "", "", w.At(155), map[string]any{"carrier_type": "tag_qr", "read_outcome": "unreadable"})
	must(err)
	must(w.Settle(ctx))
	px, err = w.Item.Passport(ctx, x, platform.Moment{})
	must(err)
	open := slices.IndexFunc(px.Questions, func(q itemapp.IdentificationQuestion) bool { return q.Open })
	if open < 0 || px.Questions[open].Cause != "carrier_unreadable" || px.Status.Position != "isolated" {
		t.Fatalf("X: нечитаемый носитель: %+v", px.Questions)
	}
	if _, err := w.Item.ConfirmIdentification(ctx, x, itemapp.ConfirmIdentification{CommandHeader: w.Cmd(), Method: "tag_qr", QuestionedEventID: "00000000-0000-7000-8000-000000000000"}); code(err) != errcodes.ItemIdentificationNotQuestioned {
		t.Fatal("подтверждение чужой записи")
	}
	w.At(160)
	_, err = w.Item.ConfirmIdentification(ctx, x, itemapp.ConfirmIdentification{CommandHeader: w.Cmd(), Method: "manual_entry", QuestionedEventID: px.Questions[open].QuestionedID})
	must(err)
	must(w.Settle(ctx))
	if px, err = w.Item.Passport(ctx, x, platform.Moment{}); err != nil || px.Status.Position == "isolated" {
		t.Fatalf("X после подтверждения: %+v %v", px.Status, err)
	}

	// ── 5. генеалогия, поиск, журнал изменений ──
	gen, err := w.Item.Genealogy(ctx, u1, platform.Moment{})
	must(err)
	if !slices.Equal(gen.Up, []string{f1}) || !slices.Equal(gen.Down, []string{c1}) {
		t.Fatalf("генеалогия U-1: %+v", gen)
	}
	gen, err = w.Item.Genealogy(ctx, c1, platform.Moment{})
	must(err)
	if !slices.Equal(gen.Up, []string{u1, f1}) || !slices.Contains(gen.Lots, "L-17") {
		t.Fatalf("генеалогия C-1: %+v", gen)
	}
	lk, err := w.Item.Lookup(ctx, "ant:carrier:tag_qr:Q-X", platform.Moment{})
	must(err)
	if lk.ItemID != x {
		t.Fatalf("поиск по носителю: %+v", lk)
	}
	lk, err = w.Item.Lookup(ctx, "BOX-1/3", platform.Moment{})
	must(err)
	if len(lk.Candidates) != 2 {
		t.Fatalf("поиск по ячейке тары: %+v", lk)
	}
	if lk, err = w.Item.Lookup(ctx, "F-1", platform.Moment{}); err != nil || lk.ItemID != f1 {
		t.Fatalf("поиск по номеру: %+v %v", lk, err)
	}
	h, err := w.Item.History(ctx, c1, platform.Moment{}, platform.Page{})
	must(err)
	fields := map[string]bool{}
	for _, r := range h.Items {
		fields[r.Field] = true
	}
	for _, f := range []string{"registration", "status.containment", "status.summary"} {
		if !fields[f] {
			t.Fatalf("журнал изменений паспорта C-1: нет %s: %+v", f, h.Items)
		}
	}
	// Паспорт на момент до блока партии (AD-22): блока ещё нет.
	before := T0.Add(110 * time.Minute)
	p, err := w.Item.Passport(ctx, f1, platform.Moment{AsOf: &before})
	must(err)
	if len(p.Holds) != 0 || p.Status.Containment != "none" {
		t.Fatalf("паспорт на момент: %+v", p.Holds)
	}
}
