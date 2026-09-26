package kompas_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/cad"
	appingest "ant/internal/application/ingest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/schemas"
	dom "ant/internal/domain/cad"
	domingest "ant/internal/domain/ingest"
	"ant/internal/domain/kernel"
	"ant/internal/infrastructure/integration/cad/kompas"
	"ant/internal/infrastructure/integration/schemacheck"
)

// Контрактные тесты импорта КОМПАС (AD-18, AD-20, AD-35): эталон
// contracts/integrations/cad/examples/kompas-assembly-flange.json (файл
// процессной сессии, PRD §11.15) — по схеме контракта; импорт через
// application/cad даёт дерево компонентов, зоны и ограничения нормативного
// слоя (лимит ремонтов шва W-1 по ТП, «закрывает доступ к зоне» у J-1,
// порядок установки S-1), соответствия обозначений КД нашим типам; факты
// проходят схемы приёма; тот же файл — дубль; ошибки формата — с полем.

const example = "contracts/integrations/cad/examples/kompas-assembly-flange.json"

func flangeFile(t *testing.T) []byte {
	t.Helper()
	b, err := schemas.FS.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// repo — корень репозитория (нормативный слой читается как есть).
func repo(t *testing.T) string {
	t.Helper()
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "../../../../../..")
}

func flangeEnv(t *testing.T) dom.Env {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo(t), "normative/process/flange-process.bpmn"))
	if err != nil {
		t.Fatal(err)
	}
	env, err := app.EnvFromBPMN(b)
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func TestExampleMatchesSchema(t *testing.T) {
	if err := schemacheck.Validate(kompas.SchemaPath, flangeFile(t)); err != nil {
		t.Fatalf("эталон сборки не по схеме: %v", err)
	}
}

func TestParseFlange(t *testing.T) {
	a, err := kompas.Parse(flangeFile(t))
	if err != nil {
		t.Fatal(err)
	}
	if a.Designation != "ФЛ-100.00.000 СБ" || a.Version != "Б" || a.Letter != "О1" || a.GeometryPresent || a.GeometryNote == "" {
		t.Fatalf("сборка: %+v", a)
	}
	if len(a.Components) != 7 || len(a.Links) != 4 || len(a.Characteristics) != 4 {
		t.Fatalf("позиций %d, связей %d, характеристик %d", len(a.Components), len(a.Links), len(a.Characteristics))
	}
	if j2 := a.Links[3]; j2.TorqueNm != 12 || j2.TolerancePct != 10 {
		t.Errorf("момент J-2 из 12.0: %+v", j2)
	}
	if !strings.HasPrefix(a.Digest, "streebog256:") || a.Source.ExportedAt != time.Date(2026, 9, 21, 5, 0, 0, 0, time.UTC) {
		t.Errorf("отпечаток и время выгрузки: %s %v", a.Digest, a.Source.ExportedAt)
	}
}

// Нормативный слой фланца: лимит 3 на участки шва W-1 (шаг сварки),
// установка крышки закрывает доступ к S-1 и CAV.
func TestEnvFromBPMN(t *testing.T) {
	env := flangeEnv(t)
	if l := env.ReworkLimits["W-1"]; l.Value != 3 || l.StepKey != "welding.weld" {
		t.Errorf("лимит W-1: %+v", l)
	}
	if env.ClosingSteps["S-1"] != "assembly.cover_install" || env.ClosingSteps["CAV"] != "assembly.cover_install" {
		t.Errorf("закрывающие шаги: %v", env.ClosingSteps)
	}
}

// intake — приём в памяти: конверт проверяется схемами приёма
// (ingest.ValidateEnvelope), повтор event_id — дубль (AD-7).
type intake struct {
	seen   map[string]int64
	events []map[string]any
	seq    int64
}

func (i *intake) Submit(_ context.Context, source string, envs [][]byte) ([]app.IntakeOutcome, error) {
	var out []app.IntakeOutcome
	for _, raw := range envs {
		var d struct {
			Payload    string `json:"payload"`
			Signatures []any  `json:"signatures"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, err
		}
		p, err := base64.StdEncoding.DecodeString(d.Payload)
		if err != nil {
			return nil, err
		}
		dec, err := appingest.ValidateEnvelope(p)
		if err != nil {
			return nil, err
		}
		var e map[string]any
		_ = json.Unmarshal(p, &e)
		id, _ := e["event_id"].(string)
		if dec.Outcome != domingest.OutcomeAccepted {
			out = append(out, app.IntakeOutcome{EventID: id, Outcome: "quarantined", Code: string(dec.Code), Field: dec.Field, Detail: dec.Detail})
			continue
		}
		if e["source_id"] != source || e["source_kind"] != "import" {
			return nil, errors.New("источник импорта не cad.kompas/import")
		}
		if s, ok := i.seen[id]; ok {
			out = append(out, app.IntakeOutcome{EventID: id, Outcome: "duplicate", Seq: s})
			continue
		}
		i.seq++
		i.seen[id] = i.seq
		i.events = append(i.events, e)
		out = append(out, app.IntakeOutcome{EventID: id, Outcome: "accepted", Seq: i.seq})
	}
	return out, nil
}

// store — проекции cad.* в памяти: записи журнала проходят шаги проекций.
type store map[string]json.RawMessage

func (s store) Get(_ context.Context, name, key string) (json.RawMessage, bool, error) {
	v, ok := s[name+"/"+key]
	return v, ok, nil
}

func (s store) apply(t *testing.T, r kernel.Record) {
	t.Helper()
	for _, p := range app.Projections() {
		for _, k := range p.Keys(r) {
			v, err := p.Step(k, s[p.Name+"/"+k], r)
			if err != nil {
				t.Fatal(err)
			}
			s[p.Name+"/"+k] = v
		}
	}
}

func importFile(t *testing.T, svc *app.Service, file []byte) (platform.Receipt, error) {
	t.Helper()
	return svc.ImportAssembly(context.Background(), app.ImportAssembly{CommandHeader: platform.CommandHeader{CommandID: "019a0000-0000-7000-8000-000000000031"},
		FileName: "ФЛ-100.00.000СБ.json", Content: base64.StdEncoding.EncodeToString(file)})
}

// Импорт сборки ФЛ-100.00.000 СБ: дерево компонентов, лимит ремонтов шва и
// «закрывает доступ к зоне» у J-1 (критерий эпика 31).
func TestImportFlange(t *testing.T) {
	in := &intake{seen: map[string]int64{}}
	st := store{}
	svc := app.NewLive(app.Config{Reader: kompas.Reader{}, Intake: in, Projections: st, Env: flangeEnv(t)})
	rc, err := importFile(t, svc, flangeFile(t))
	if err != nil {
		t.Fatal(err)
	}
	if rc.Replayed || rc.Seq != 1 || len(rc.EventIDs) != 9 || len(in.events) != 9 {
		t.Fatalf("квитанция: %+v, фактов %d", rc, len(in.events))
	}
	// Соответствия обозначений КД нашим типам — система kompas (FR-95).
	var maps []string
	for _, e := range in.events[1:] {
		d := e["data"].(map[string]any)
		if e["event_type"] != string(catalog.ReferenceExternalIdMapped) || d["system"] != "kompas" || d["object_kind"] != "item_type" {
			t.Fatalf("соответствие: %v", e)
		}
		maps = append(maps, d["external_id"].(string)+"→"+d["internal_id"].(string))
	}
	for _, want := range []string{"ФЛ-100.00.000 СБ→FL-100.00.000", "Болт М6×20 [П]→BOLT-M6-20", "КВД-6 [ПП]→KVD-6"} {
		if !slices.Contains(maps, want) {
			t.Errorf("нет соответствия %s: %v", want, maps)
		}
	}
	// Журнал → проекция → операция cad.assembly.list.
	e := in.events[0]
	data, _ := json.Marshal(e["data"])
	st.apply(t, kernel.Record{Seq: 1, EventID: e["event_id"].(string), Type: catalog.CadAssemblyImported, Data: data, OccurredAt: time.Now().UTC()})
	list, err := svc.Assemblies(context.Background(), platform.Moment{})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("список сборок: %+v %v", list, err)
	}
	a := list.Items[0]
	if a.AssemblyItemTypeID != "FL-100.00.000" || a.GeometryPresent || a.GeometryNote == "" || a.Revision != "Б" || a.LifecycleLetter != "О1" {
		t.Fatalf("сборка: %+v", a)
	}
	tree := map[string]app.CadComponent{}
	for _, c := range a.Components {
		tree[c.ItemTypeID] = c
	}
	if b := tree["BOLT-M6-20"]; b.Quantity != 12 || b.TotalQuantity != 12 || b.Tracking != "lot" || b.ParentItemTypeID != "FL-100.00.000" {
		t.Errorf("болты в дереве: %+v", b)
	}
	if v := tree["KVD-6"]; v.ParentItemTypeID != "FL-100.02.000" || v.Tracking != "serial" {
		t.Errorf("клапан в крышке: %+v", v)
	}
	if s := tree["FL-100.00.004"]; s.Tracking != "lot" || !s.ShelfLifeTracked {
		t.Errorf("уплотнение со сроком хранения: %+v", s)
	}
	cons := map[string]app.CadConstraint{}
	for _, c := range a.Constraints {
		cons[c.ConstraintID] = c
	}
	w := cons["W-1/rework_limit"]
	if w.Limit == nil || *w.Limit != 3 || w.ZoneID != "W-1" || w.StepKey != "welding.weld" {
		t.Errorf("лимит ремонтов шва W-1: %+v", w)
	}
	j := cons["J-1/closes_access"]
	if !slices.Equal(j.ClosesZoneIDs, []string{"S-1"}) || j.StepKey != "assembly.cover_install" {
		t.Errorf("J-1 «закрывает доступ к зоне»: %+v", j)
	}
	if s := cons["S-1/install_order"]; s.FirstItemTypeID != "FL-100.00.004" || s.ThenItemTypeID != "FL-100.02.000" || s.StepKey != "assembly.seal_install" {
		t.Errorf("S-1 до крышки: %+v", s)
	}
	if q := cons["J-2/torque"]; q.Value == nil || *q.Value != 12 || q.Unit != "Н·м" {
		t.Errorf("момент J-2: %+v", q)
	}
	if len(a.Zones) != 4 || len(a.Discrepancies) != 1 || a.Discrepancies[0].Reason != "not_checked" {
		t.Errorf("зоны %d, расхождения %+v", len(a.Zones), a.Discrepancies)
	}

	// Тот же файл повторно — дубль: прежние записи, новых фактов нет (AD-7).
	rc2, err := importFile(t, svc, flangeFile(t))
	if err != nil || !rc2.Replayed || rc2.Seq != 1 || len(in.events) != 9 {
		t.Fatalf("повтор: %+v %v, фактов %d", rc2, err, len(in.events))
	}
}

// Сверка с номенклатурой 1С: крепёж без соответствия — отчёт о расхождениях.
type nomenclature struct{}

func (nomenclature) ItemTypes(context.Context) (*dom.Nomenclature, error) {
	return &dom.Nomenclature{System: "onec", ItemTypes: map[string]bool{"FL-100.00.000": true, "FL-100.01.001": true, "FL-100.01.002": true,
		"FL-100.02.000": true, "FL-100.00.004": true, "KVD-6": true}}, nil
}

func TestImportDiscrepancies(t *testing.T) {
	in := &intake{seen: map[string]int64{}}
	svc := app.NewLive(app.Config{Reader: kompas.Reader{}, Intake: in, Env: flangeEnv(t), Nomenclature: nomenclature{}})
	if _, err := importFile(t, svc, flangeFile(t)); err != nil {
		t.Fatal(err)
	}
	ds := in.events[0]["data"].(map[string]any)["discrepancies"].([]any)
	if len(ds) != 2 || ds[0].(map[string]any)["item_type_id"] != "BOLT-M6-20" || ds[0].(map[string]any)["reason"] != "not_in_erp" {
		t.Fatalf("расхождения: %v", ds)
	}
}

// Ошибки формата: поле и пояснение (problem+json api.validation_failed).
func TestFormatErrors(t *testing.T) {
	cases := []struct {
		name, field string
		mutate      func(m map[string]any)
	}{
		{"нет geometry — ошибка, а не «геометрии нет»", "/assembly/geometry", func(m map[string]any) { delete(m["assembly"].(map[string]any), "geometry") }},
		{"геометрия не null", "/assembly/geometry", func(m map[string]any) { m["assembly"].(map[string]any)["geometry"] = "model.step" }},
		{"нет версии КД", "/assembly/version", func(m map[string]any) { delete(m["assembly"].(map[string]any), "version") }},
		{"пустое обозначение", "/components/1/designation", func(m map[string]any) { m["components"].([]any)[1].(map[string]any)["designation"] = "" }},
		{"связь на несуществующую позицию", "/links/1/from", func(m map[string]any) { m["links"].([]any)[1].(map[string]any)["from"] = "ФЛ-999" }},
		{"дробный момент", "/links/3/torque_nm", func(m map[string]any) { m["links"].([]any)[3].(map[string]any)["torque_nm"] = 12.5 }},
		{"неизвестная версия формата", "/format_version", func(m map[string]any) { m["format_version"] = "2.0" }},
	}
	for _, c := range cases {
		var m map[string]any
		_ = json.Unmarshal(flangeFile(t), &m)
		c.mutate(m)
		b, _ := json.Marshal(m)
		svc := app.NewLive(app.Config{Reader: kompas.Reader{}, Intake: &intake{seen: map[string]int64{}}})
		_, err := importFile(t, svc, b)
		pe, ok := platform.AsError(err)
		if !ok || pe.Code != "api.validation_failed" || pe.Params["field"] != c.field {
			t.Errorf("%s: ожидалось поле %s, получено %v", c.name, c.field, err)
		}
	}
}
