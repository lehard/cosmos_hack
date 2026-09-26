package cad_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"ant/internal/domain/cad"
)

// flange — условная сборка ФЛ-100.00.000 СБ (PRD §11.15) на нашем языке.
func flange() cad.Assembly {
	root := "FL-100.00.000"
	return cad.Assembly{Designation: "ФЛ-100.00.000 СБ", AssemblyID: root, Name: "Фланец люка гермокорпуса в сборе", Version: "Б", Letter: "О1",
		GeometryNote: "Геометрия отсутствует", Digest: "streebog256:" + strings.Repeat("a", 64),
		Components: []cad.Component{
			{Position: 1, Designation: "ФЛ-100.01.001", Name: "Фланец", Kind: cad.KindDetail, Qty: 1, MakeOrBuy: "make", Parent: root},
			{Position: 2, Designation: "ФЛ-100.01.002", Name: "Патрубок (кольцо)", Kind: cad.KindDetail, Qty: 1, MakeOrBuy: "buy", Parent: root},
			{Position: 3, Designation: "ФЛ-100.02.000", Name: "Крышка люка", Kind: cad.KindSubassembly, Qty: 1, MakeOrBuy: "buy", Parent: root},
			{Position: 4, Designation: "ФЛ-100.00.004", Name: "Уплотнение кольцевое", Kind: cad.KindStandardPart, Qty: 1, ShelfLifeTracked: true, Parent: root},
			{Position: 5, Designation: "Болт М6×20 [П]", Name: "Болт крепления крышки", Kind: cad.KindFastener, Qty: 12, LotTracked: true, Parent: root},
			{Position: 6, Designation: "Шайба 6 [П]", Name: "Шайба", Kind: cad.KindFastener, Qty: 12, LotTracked: true, Parent: root},
			{Position: 7, Designation: "КВД-6 [ПП]", Name: "Клапан выравнивания давления", Kind: cad.KindPurchasedEquipment, Qty: 1, SerialTracked: true, Parent: "ФЛ-100.02.000"},
		},
		Links: []cad.Link{
			{ID: "W-1", Type: cad.LinkWeld, From: "ФЛ-100.01.001", To: "ФЛ-100.01.002"},
			{ID: "J-1", Type: cad.LinkBoltedJoint, From: "ФЛ-100.02.000", To: "ФЛ-100.01.001", Fasteners: []string{"Болт М6×20 [П]", "Шайба 6 [П]"}, Qty: 12, TighteningSequence: "крест-накрест"},
			{ID: "S-1", Type: cad.LinkSeal, Between: []string{"ФЛ-100.02.000", "ФЛ-100.01.001"}, Part: "ФЛ-100.00.004"},
			{ID: "J-2", Type: cad.LinkThreadedJoint, From: "КВД-6 [ПП]", To: "ФЛ-100.02.000", TorqueNm: 12, TolerancePct: 10, Locking: "контровка проволокой"},
		},
	}
}

// Env как из BPMN фланца: сварка — лимит 3 на участок шва W-1, установка
// крышки закрывает доступ к S-1 и CAV, установка уплотнения работает с S-1.
func env() cad.Env {
	return cad.Env{ReworkLimits: map[string]cad.Limit{"W-1": {Value: 3, StepKey: "welding.weld"}},
		ClosingSteps: map[string]string{"S-1": "assembly.cover_install", "CAV": "assembly.cover_install"},
		ZoneSteps:    map[string][]string{"S-1": {"assembly.seal_install", "assembly.cover_install"}}}
}

func TestItemTypeID(t *testing.T) {
	cases := [][2]string{
		{"ФЛ-100.00.000 СБ", "FL-100.00.000"}, {"ФЛ-100.01.001", "FL-100.01.001"}, {"Болт М6×20 [П]", "BOLT-M6-20"},
		{"Шайба 6 [П]", "SHAYBA-6"}, {"КВД-6 [ПП]", "KVD-6"}, {"  ", ""},
	}
	for _, c := range cases {
		if got := cad.ItemTypeID(c[0]); got != c[1] {
			t.Errorf("ItemTypeID(%q) = %q, ожидалось %q", c[0], got, c[1])
		}
	}
}

// Импорт даёт дерево компонентов с количествами, лимит ремонтов шва W-1 и
// «закрывает доступ к зоне» у болтового соединения J-1 (критерий эпика 31).
func TestTranslateFlange(t *testing.T) {
	r, ps := cad.Translate(flange(), env(), nil)
	if len(ps) > 0 {
		t.Fatalf("проблемы: %v", ps)
	}
	if r.AssemblyItemTypeID != "FL-100.00.000" || len(r.Nodes) != 8 {
		t.Fatalf("корень и дерево: %s, %d узлов", r.AssemblyItemTypeID, len(r.Nodes))
	}
	bolt, _ := r.Node("BOLT-M6-20")
	valve, _ := r.Node("KVD-6")
	seal, _ := r.Node("FL-100.00.004")
	flangeN, _ := r.Node("FL-100.01.001")
	if bolt.Qty != 12 || bolt.Total != 12 || bolt.Tracking != cad.TrackLot || bolt.Parent != "FL-100.00.000" {
		t.Errorf("болт: %+v", bolt)
	}
	if valve.Parent != "FL-100.02.000" || valve.Depth != 2 || valve.Tracking != cad.TrackSerial {
		t.Errorf("клапан — в крышке, по номеру: %+v", valve)
	}
	if seal.Tracking != cad.TrackLot || flangeN.Tracking != cad.TrackSerial {
		t.Errorf("учёт: уплотнение %s, фланец %s", seal.Tracking, flangeN.Tracking)
	}
	if got := r.ZoneIDs(); !slices.Equal(got, []string{"J-1", "J-2", "S-1", "W-1"}) {
		t.Errorf("зоны: %v", got)
	}
	w, ok := r.Constraint("W-1", cad.ConstraintReworkLimit)
	if !ok || w.Limit == nil || *w.Limit != 3 || w.StepKey != "welding.weld" || w.ZoneID != "W-1" {
		t.Fatalf("лимит ремонтов шва W-1: %+v", w)
	}
	j, ok := r.Constraint("J-1", cad.ConstraintClosesAccess)
	if !ok || !slices.Equal(j.Closes, []string{"S-1"}) || j.StepKey != "assembly.cover_install" {
		t.Fatalf("J-1 закрывает доступ к S-1: %+v", j)
	}
	s, ok := r.Constraint("S-1", cad.ConstraintInstallOrder)
	if !ok || s.First != "FL-100.00.004" || s.Then != "FL-100.02.000" || s.StepKey != "assembly.seal_install" {
		t.Fatalf("S-1 до крышки: %+v", s)
	}
	if f, ok := r.Constraint("J-1", cad.ConstraintFastening); !ok || f.Qty != 12 || !strings.Contains(f.Note, "крест-накрест") {
		t.Errorf("крепёж J-1: %+v", f)
	}
	if q, ok := r.Constraint("J-2", cad.ConstraintTorque); !ok || *q.Value != 12 || *q.Tolerance != 10 {
		t.Errorf("момент J-2: %+v", q)
	}
	for _, z := range r.Zones {
		want := "FL-100.00.000"
		if z.ID == "J-2" {
			want = "FL-100.02.000"
		}
		if z.ItemTypeID != want {
			t.Errorf("зона %s лежит на %s, ожидалось %s", z.ID, z.ItemTypeID, want)
		}
	}
	if len(r.Mappings) != 8 || r.Mappings[0] != (cad.Mapping{ExternalID: "ФЛ-100.00.000 СБ", InternalID: "FL-100.00.000"}) {
		t.Errorf("соответствия: %v", r.Mappings)
	}
	if len(r.Discrepancies) != 1 || r.Discrepancies[0].Reason != cad.ReasonNotChecked {
		t.Errorf("без номенклатуры — «сверка не выполнялась»: %v", r.Discrepancies)
	}

	d := cad.Fact(flange(), r)
	if d.GeometryPresent || d.GeometryNote == nil || len(d.Components) != 7 || len(d.Links) != 4 || len(d.Constraints) != 5 {
		t.Fatalf("факт: геометрия %v, позиций %d, связей %d, ограничений %d", d.GeometryPresent, len(d.Components), len(d.Links), len(d.Constraints))
	}
	if d.Links[3].Kind != "other" || *d.Links[3].LinkType != "threaded_joint" || !slices.Equal(d.Links[1].Components, []string{"3", "1", "5", "6"}) {
		t.Errorf("связи в факте: %+v %+v", d.Links[3], d.Links[1])
	}
	b, _ := json.Marshal(d)
	if !strings.Contains(string(b), `"closes_zone_ids":["S-1"]`) || !strings.Contains(string(b), `"limit":3`) {
		t.Errorf("факт без ограничений: %s", b)
	}
}

// Лимит из файла сильнее ТП; без ТП и файла — лимит не задан явно.
func TestReworkLimitSource(t *testing.T) {
	a := flange()
	a.Links[0].ReworkLimit = 2
	r, _ := cad.Translate(a, env(), nil)
	if w, _ := r.Constraint("W-1", cad.ConstraintReworkLimit); *w.Limit != 2 || w.LimitSource != "файл сборки" {
		t.Errorf("лимит из файла: %+v", w)
	}
	r, _ = cad.Translate(flange(), cad.Env{}, nil)
	if w, _ := r.Constraint("W-1", cad.ConstraintReworkLimit); w.Limit != nil || !strings.Contains(w.Note, "задаёт технолог") {
		t.Errorf("лимит не задан: %+v", w)
	}
}

// Сверка с номенклатурой 1С: найдено — соответствие, нет — отчёт, без автосоздания.
func TestDiscrepancies(t *testing.T) {
	nom := &cad.Nomenclature{System: "onec", ItemTypes: map[string]bool{"FL-100.00.000": true, "FL-100.01.001": true, "FL-100.01.002": true,
		"FL-100.02.000": true, "FL-100.00.004": true, "KVD-6": true}}
	r, _ := cad.Translate(flange(), env(), nom)
	var got []string
	for _, d := range r.Discrepancies {
		got = append(got, d.ItemTypeID+":"+d.Reason)
	}
	if !slices.Equal(got, []string{"BOLT-M6-20:not_in_erp", "SHAYBA-6:not_in_erp"}) {
		t.Errorf("расхождения: %v", got)
	}
}

// Структурные ошибки — отказ импорта с полем: связь на несуществующую
// позицию, пустое обозначение, нет версии КД, цикл в дереве.
func TestValidate(t *testing.T) {
	a := flange()
	a.Links[1].From = "ФЛ-999"
	a.Components[1].Designation = " "
	a.Version = ""
	ps := cad.Validate(a)
	var fields []string
	for _, p := range ps {
		fields = append(fields, p.Field)
	}
	for _, want := range []string{"/assembly/version", "/components/1/designation", "/links/1/from"} {
		if !slices.Contains(fields, want) {
			t.Errorf("нет проблемы %s: %v", want, ps)
		}
	}
	b := flange()
	b.Components[2].Parent = "КВД-6 [ПП]"
	if ps := cad.Validate(b); len(ps) == 0 || !strings.Contains(ps[0].Detail, "цикл") {
		t.Errorf("цикл в дереве: %v", ps)
	}
	if _, ps := cad.Translate(a, env(), nil); len(ps) == 0 {
		t.Error("перевод с проблемами должен отказать")
	}
}

func TestIDs(t *testing.T) {
	d := cad.Digest([]byte("{}"))
	if !strings.HasPrefix(d, "streebog256:") || len(d) != 12+64 {
		t.Fatalf("отпечаток: %s", d)
	}
	first, again, other := cad.ImportEventID(d), cad.ImportEventID(cad.Digest([]byte("{}"))), cad.ImportEventID(cad.Digest([]byte("[]")))
	if first != again || first == other {
		t.Error("id импорта детерминирован и зависит от файла")
	}
}
