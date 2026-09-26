package vision

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/contracts/normative"
	"ant/internal/domain/kernel"
)

var t0 = time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC)

type builder struct {
	n   int
	out []kernel.Record
}

func (b *builder) add(t catalog.Type, data any) kernel.Record {
	b.n++
	raw, _ := json.Marshal(data)
	info, _ := catalog.Lookup(t)
	r := kernel.Record{Seq: int64(b.n), EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", b.n), Type: t, SchemaVersion: 1,
		Kind: info.Kind, OccurredAt: t0.Add(time.Duration(b.n) * time.Hour), Data: raw, Provenance: "personal"}
	b.out = append(b.out, r)
	return r
}

func full() Versions {
	return Versions{ItemRevision: "Б", RecipeRef: "kt3-weld@1", CameraConfig: "CAM-KT3/свет-R2", Calibration: "CAL", AnalyzerVersion: "vqc-weld 2.3.1",
		ThresholdProfile: "TP", ContractVersion: "1.0", AppVersion: "gw 1"}
}

func admitted(id string, lvl int, extra map[string]any) map[string]any {
	d := map[string]any{"passport_id": id, "stage": "active", "trust_level": lvl, "recipe_ref": "kt3-weld@1",
		"versions": full().Contract(), "document_id": "DOC-" + id}
	for _, k := range slices.Sorted(maps.Keys(extra)) {
		d[k] = extra[k]
	}
	return d
}

func refusal(t *testing.T, err error, code errcodes.Code) {
	t.Helper()
	var r *kernel.Refusal
	if !errors.As(err, &r) || r.Code != code {
		t.Fatalf("ждали отказ %s, получили %v", code, err)
	}
}

// AD-29: вектор версий — восемь составляющих; несообщённое — явное unknown.
func TestVersionsComplete(t *testing.T) {
	v := Versions{RecipeRef: "kt3-weld@1", AnalyzerVersion: "vqc-weld 2.3.1", ContractVersion: "1.0"}
	c, miss := v.Complete()
	if len(miss) != 5 || c.Calibration != Unknown || c.AppVersion != Unknown || c.RecipeRef != "kt3-weld@1" {
		t.Fatalf("%+v %v", c, miss)
	}
	if m := full().Map(); len(m) != 8 || len(full().Missing()) != 0 {
		t.Fatalf("%v", m)
	}
	if RecipeRef("kt3-weld", "") != "kt3-weld@unknown" || RecipeRef("", "1") != "" {
		t.Fatal("RecipeRef")
	}
}

// FR-101: реестр паспортов — допуск, откат, возврат, вывод; уровень действует,
// только пока паспорт в действии и не в тени.
func TestFoldAndLevels(t *testing.T) {
	var b builder
	b.add(catalog.AnalyzerPassportAdmitted, admitted("AP-1", 3, map[string]any{"analyzer_id": "vqc-weld", "analyzer_kind": "visionqc", "title": "КТ-3"}))
	b.add(catalog.AnalyzerPassportAdmitted, admitted("AP-SH", 4, map[string]any{"stage": "shadow"}))
	s := b.add(catalog.AnalyzerPassportSuspended, map[string]any{"passport_id": "AP-1", "trigger": "drift", "fallback": "manual_control", "basis": []string{}})
	b.add(catalog.AnalyzerCheckRecorded, map[string]any{"passport_id": "AP-1", "check_kind": "drift_monitor", "passed": false, "escape_rate_bp": 300, "versions": full().Contract()})
	reg := Fold(b.out)
	p, _ := reg.Passport("AP-1")
	if p.AnalyzerID != "vqc-weld" || p.Title != "КТ-3" || p.Current().Status != StatusSuspended || p.EffectiveLevel() != 0 {
		t.Fatalf("%+v", p)
	}
	if p.StatusAt(t0.Add(90*time.Minute)) != StatusActive || p.StatusAt(t0) != StatusNotAdmitted {
		t.Fatal("откат действует на будущее, не переписывает прошлое")
	}
	if sh, _ := reg.Passport("AP-SH"); sh.EffectiveLevel() != 0 || sh.AnalyzerID != "vqc-weld" {
		t.Fatalf("тень: %+v", sh)
	}
	if c := reg.ChecksOf("AP-1"); len(c) != 1 || *c[0].EscapeRateBP != 300 || reg.BasisSeq != 4 {
		t.Fatalf("%+v", c)
	}
	b.add(catalog.AnalyzerPassportReinstated, map[string]any{"passport_id": "AP-1", "suspension_event_id": s.EventID, "reason": map[string]any{"text": "эталон пройден"}})
	if p, _ := Fold(b.out).Passport("AP-1"); p.EffectiveLevel() != 3 {
		t.Fatalf("после возврата: %+v", p)
	}
	if got := AllowedAutoActions(3); !slices.Contains(got, "item_hold") || slices.Contains(got, "auto_pass_confident") || !slices.Contains(got, "record_observation") {
		t.Fatalf("уровень 3: %v", got)
	}
	if len(AllowedAutoActions(0)) != 1 || TrustLevels() != 5 {
		t.Fatal("уровень 0 — только запись")
	}
}

// AD-27, FR-101: вернуть анализатор — только начальник ОТК и только из
// действующей приостановки; повторный допуск и вывод — отказ по статусу.
func TestGuards(t *testing.T) {
	var b builder
	b.add(catalog.AnalyzerPassportAdmitted, admitted("AP-1", 3, nil))
	reg := Fold(b.out)
	a := Admission{PassportID: "AP-1", Stage: "active", TrustLevel: 3, RecipeRef: "kt3-weld@1", Versions: full(), DocumentID: "D"}
	refusal(t, GuardAdmit(reg, a), errcodes.AnalyzerInvalidTransition)
	a.PassportID = "AP-2"
	if err := GuardAdmit(reg, a); err != nil {
		t.Fatal(err)
	}
	a2 := a
	a2.Versions.RecipeRef = "kt2@1"
	refusal(t, GuardAdmit(reg, a2), errcodes.ApiValidationFailed)
	a2 = a
	a2.PreviousPassportID = "AP-404"
	refusal(t, GuardAdmit(reg, a2), errcodes.ApiNotFound)
	a2 = a
	a2.Versions.AnalyzerVersion = Unknown
	refusal(t, GuardAdmit(reg, a2), errcodes.ApiValidationFailed)

	_, err := GuardReinstate(reg, "AP-1", "x", RoleHeadOfQC, true)
	refusal(t, err, errcodes.AnalyzerInvalidTransition)
	s := b.add(catalog.AnalyzerPassportSuspended, map[string]any{"passport_id": "AP-1", "trigger": "escape_detected", "fallback": "manual_control", "basis": []string{}})
	reg = Fold(b.out)
	_, err = GuardReinstate(reg, "AP-1", s.EventID, "quality_inspector", false)
	refusal(t, err, errcodes.AnalyzerReinstateRequiresHeadOfQc)
	_, err = GuardReinstate(reg, "AP-1", "other", RoleHeadOfQC, true)
	refusal(t, err, errcodes.ApiValidationFailed)
	if _, err := GuardReinstate(reg, "AP-1", s.EventID, RoleHeadOfQC, true); err != nil {
		t.Fatal(err)
	}
	if _, err := GuardRetire(reg, "AP-1"); err != nil {
		t.Fatal(err)
	}
	b.add(catalog.AnalyzerPassportRetired, map[string]any{"passport_id": "AP-1", "reason": map[string]any{"text": "вывод"}})
	_, err = GuardRetire(Fold(b.out), "AP-1")
	refusal(t, err, errcodes.AnalyzerInvalidTransition)
	_, err = GuardRetire(reg, "AP-404")
	refusal(t, err, errcodes.ApiNotFound)
}

// FR-126: исполнитель гипотезы — по входу на рабочее место; нет сеанса или
// сеансов несколько — неизвестен (система не угадывает человека).
func TestExecutorByWorkplaceEntry(t *testing.T) {
	var b builder
	b.add(catalog.AccessWorkplaceAdmitted, map[string]any{"workplace_id": "WP-ASM-1", "workplace_session_id": "0190a000-0000-7000-8000-000000000001", "person_id": "O21"})
	b.add(catalog.AccessWorkplaceReleased, map[string]any{"workplace_id": "WP-ASM-1", "workplace_session_id": "0190a000-0000-7000-8000-000000000001"})
	b.add(catalog.AccessWorkplaceAdmitted, map[string]any{"workplace_id": "WP-ASM-1", "workplace_session_id": "0190a000-0000-7000-8000-000000000002", "person_id": "O17"})
	ss := Sessions(b.out)
	if s, ok := ExecutorAt(ss, "WP-ASM-1", t0.Add(90*time.Minute)); !ok || s.PersonID != "O21" {
		t.Fatalf("%+v", s)
	}
	if s, ok := ExecutorAt(ss, "WP-ASM-1", t0.Add(4*time.Hour)); !ok || s.PersonID != "O17" {
		t.Fatalf("%+v", s)
	}
	if _, ok := ExecutorAt(ss, "WP-ASM-1", t0.Add(150*time.Minute)); ok {
		t.Fatal("между сеансами исполнитель неизвестен")
	}
	if _, ok := ExecutorAt(ss, "WP-OTHER", t0.Add(4*time.Hour)); ok {
		t.Fatal("чужое рабочее место")
	}
}

// FR-102: иллюстрация по виду дефекта; «оценка невозможна» не иллюстрируется;
// пометка содержит «ИЛЛЮСТРАЦИЯ», источник, лицензию, автора.
func TestChooseIllustration(t *testing.T) {
	cat := normative.IllustrationsCatalog{Datasets: []normative.IllustrationsCatalogDatasetsElem{{ID: "d", Title: "TIG Aluminium 5083",
		URL: "https://example.invalid/d", License: "CC BY-SA 4.0", Author: "Daniel Bacioiu", Mark: "ИЛЛЮСТРАЦИЯ", Note: "не относится к изделию.",
		Classes: []normative.IllustrationsCatalogDatasetsElemClassesElem{{DatasetClass: "good_weld", DefectCodes: []string{}}, {DatasetClass: "burn_through", DefectCodes: []string{"W-BURNTHRU"}}}}}}
	c, ok := ChooseIllustration(cat, "defect_indicated", []string{"W-XXX", "W-BURNTHRU"})
	if !ok || c.Class.DatasetClass != "burn_through" {
		t.Fatalf("%+v", c)
	}
	n := c.SourceNote()
	for _, w := range []string{"ИЛЛЮСТРАЦИЯ", "не относится к изделию", "CC BY-SA 4.0", "Daniel Bacioiu", "https://example.invalid/d", "burn_through"} {
		if !strings.Contains(n, w) {
			t.Fatalf("нет %q в %q", w, n)
		}
	}
	if c, ok := ChooseIllustration(cat, "no_defect_indicated", nil); !ok || c.Class.DatasetClass != "good_weld" {
		t.Fatal("годный шов")
	}
	if _, ok := ChooseIllustration(cat, "unable_to_assess", nil); ok {
		t.Fatal("оценка невозможна иллюстрируется")
	}
	if _, ok := ChooseIllustration(cat, "defect_indicated", []string{"A-FOD"}); ok {
		t.Fatal("вид без класса")
	}
}
