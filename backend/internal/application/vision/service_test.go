package vision_test

import (
	"context"
	"os"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	"ant/internal/application/ingest/inmem"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	app "ant/internal/application/vision"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality"
	"go.yaml.in/yaml/v3"
)

var repo = os.DirFS("../../../..")

func core(t *testing.T) (*inmem.Journal, *engineapp.Codec) {
	j := inmem.NewJournal(func() time.Time { return time.Date(2026, 9, 21, 9, 0, 0, 0, time.UTC) })
	return j, &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", Partitions: 4}
}

func seed(t *testing.T, j *inmem.Journal, profile string) int {
	t.Helper()
	s, err := app.LoadPassportsSeed(repo)
	if err != nil {
		t.Fatal(err)
	}
	n, err := app.SeedPassports(context.Background(), j, s, app.SeedConfig{Profile: profile, DomainBuild: "test", Partitions: 4})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func code(t *testing.T, err error, c errcodes.Code) {
	t.Helper()
	pe, ok := platform.AsError(err)
	if !ok || pe.Code != c {
		t.Fatalf("ждали %s, получили %v", c, err)
	}
}

// AD-29, AD-33: затравка паспортов пишется в demo и fixtures, в prod — никогда;
// повторный запуск — дубль, а не вторая запись; quality читает её как паспорта
// уровня 3 для карт контроля процесса.
func TestSeedPassports(t *testing.T) {
	j, codec := core(t)
	if n := seed(t, j, "prod"); n != 0 {
		t.Fatalf("prod: %d", n)
	}
	n := seed(t, j, "demo")
	if n < 7 {
		t.Fatalf("demo: %d", n)
	}
	if again := seed(t, j, "fixtures"); again != 0 || len(j.Main()) != n {
		t.Fatalf("повтор: %d, записей %d", again, len(j.Main()))
	}
	var recs []kernel.Record
	for _, s := range j.Main() {
		d, err := codec.Decode(context.Background(), s.Entry)
		if err != nil {
			t.Fatal(err)
		}
		if d.Record.Type != catalog.AnalyzerPassportAdmitted || d.Record.Provenance != "genesis" {
			t.Fatalf("%+v", d.Record)
		}
		recs = append(recs, d.Record)
	}
	found := false
	for _, p := range quality.PassportsFrom(recs) {
		if p.RecipeRef == "kt3-weld@1" {
			found = p.TrustLevel == 3 && p.Stage == "active" && p.AnalyzerVersion == "vqc-weld 2.3.1"
		}
	}
	if !found {
		t.Fatal("quality не видит паспорт kt3-weld@1 уровня 3")
	}
}

// Живые операции vision: анализаторы, паспорт, проверки, допуск, возврат, вывод.
func TestLiveOperations(t *testing.T) {
	ctx := context.Background()
	j, codec := core(t)
	seed(t, j, "demo")
	svc := app.NewService(app.WithDeps(app.Deps{Journal: j, Codec: codec, Routes: app.DemoRoutes{}}), app.WithConfig(app.Config{DomainBuild: "test", Partitions: 4}))

	list, err := svc.Analyzers(ctx, platform.Moment{})
	if err != nil || len(list.Items) != 7 {
		t.Fatalf("%+v %v", list, err)
	}
	for _, a := range list.Items {
		if a.AnalyzerID == "vqc-weld" && (*a.TrustLevel != 3 || a.Status != "active" || *a.Provenance != "genesis" || a.Versions["calibration"] == "") {
			t.Fatalf("%+v", a)
		}
	}
	p, err := svc.Passport(ctx, "AP-KT3-WELD-1", platform.Moment{})
	if err != nil || p.Status != "active" || p.RecipeRef != "kt3-weld@1" || len(p.AllowedAutoActions) != 6 || p.BasisSeq == 0 {
		t.Fatalf("%+v %v", p, err)
	}
	_, err = svc.Passport(ctx, "AP-404", platform.Moment{})
	code(t, err, errcodes.ApiNotFound)
	if c, err := svc.Checks(ctx, "AP-KT3-WELD-1", platform.Moment{}, platform.Page{}); err != nil || len(c.Items) != 0 {
		t.Fatalf("%+v %v", c, err)
	}
	past := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if l, _ := svc.Analyzers(ctx, platform.Moment{Axis: platform.AxisOccurred, AsOf: &past}); len(l.Items) != 0 {
		t.Fatal("на момент до затравки паспортов нет")
	}

	// Допуск новой версии: протокол не подписан — отказ; демо-заглушка — запись.
	in := app.AdmitPassport{PassportID: "AP-KT3-WELD-2", AnalyzerID: "vqc-weld", Stage: "pilot", TrustLevel: 2, RecipeRef: "kt3-weld@2",
		Versions:   map[string]string{"recipe_ref": "kt3-weld@2", "analyzer_version": "vqc-weld 2.4.0", "contract_version": "1.0"},
		DocumentID: "DOC-AP-2", PreviousPassportID: "AP-KT3-WELD-1"}
	strict := app.NewService(app.WithDeps(app.Deps{Journal: j, Codec: codec}))
	_, err = strict.AdmitPassport(ctx, in)
	code(t, err, errcodes.AnalyzerAdmissionRouteOpen)
	in.CommandID = "0190a000-0000-7000-8000-00000000aa01"
	r, err := svc.AdmitPassport(ctx, in)
	if err != nil || r.Seq == 0 {
		t.Fatalf("%+v %v", r, err)
	}
	if r2, err := svc.AdmitPassport(ctx, in); err != nil || !r2.Replayed || r2.Seq != r.Seq {
		t.Fatalf("повтор команды: %+v %v", r2, err)
	}
	p2, _ := svc.Passport(ctx, "AP-KT3-WELD-2", platform.Moment{})
	if p2.Provenance == nil || *p2.Provenance != "personal" || p2.Versions["calibration"] != "unknown" || *p2.PreviousPassportID != "AP-KT3-WELD-1" {
		t.Fatalf("%+v", p2)
	}

	// Откат (реакция проектора, эпик 40) → возврат только начальником ОТК.
	sus := kernel.UUIDv5("4b82fbf1-fc9e-5a06-91c6-8c100ebac4ef", "test-suspend")
	pend, err := codec.Encode(ctx, engineapp.Out{EventID: sus, Type: catalog.AnalyzerPassportSuspended, Kind: catalog.KindReaction,
		Stream: "analyzer_passport:AP-KT3-WELD-1", OccurredAt: time.Date(2026, 9, 21, 8, 0, 0, 0, time.UTC),
		Data: map[string]any{"passport_id": "AP-KT3-WELD-1", "trigger": "drift", "fallback": "manual_control", "basis": []string{}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(ctx, appjournalRequest(pend)); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.Passport(ctx, "AP-KT3-WELD-1", platform.Moment{})
	if p.Status != "suspended" || p.Suspension == nil || p.Suspension.Trigger != "drift" || len(p.AllowedAutoActions) != 1 {
		t.Fatalf("приостановка: %+v", p)
	}
	re := app.ReinstatePassport{SuspensionEventID: sus, Reason: app.AnalyzerReason{Text: "эталонный набор пройден"}}
	inspector := platform.WithPrincipal(ctx, platform.Principal{PersonID: "INS-02", Role: "quality_inspector"})
	_, err = svc.ReinstatePassport(inspector, "AP-KT3-WELD-1", re)
	code(t, err, errcodes.AnalyzerReinstateRequiresHeadOfQc)
	head := platform.WithPrincipal(ctx, platform.Principal{PersonID: "HQC-01", Role: "head_of_qc"})
	if _, err := svc.ReinstatePassport(head, "AP-KT3-WELD-1", re); err != nil {
		t.Fatal(err)
	}
	_, err = svc.ReinstatePassport(head, "AP-KT3-WELD-1", re)
	code(t, err, errcodes.AnalyzerInvalidTransition)

	rt := app.RetirePassport{Reason: app.AnalyzerReason{Text: "заменён AP-KT3-WELD-2"}}
	if _, err := svc.RetirePassport(head, "AP-KT3-WELD-1", rt); err != nil {
		t.Fatal(err)
	}
	_, err = svc.RetirePassport(head, "AP-KT3-WELD-1", rt)
	code(t, err, errcodes.AnalyzerInvalidTransition)
	list, _ = svc.Analyzers(ctx, platform.Moment{})
	for _, a := range list.Items {
		if a.AnalyzerID == "vqc-weld" && *a.PassportID != "AP-KT3-WELD-2" {
			t.Fatalf("действующий паспорт анализатора: %+v", a)
		}
	}
}

// Таблица допустимых действий домена совпадает по уровням с
// contracts/analyzer-trust-levels.yaml.
func TestTrustTableMatchesContract(t *testing.T) {
	b, err := os.ReadFile("../../../../contracts/analyzer-trust-levels.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Levels []struct {
			Level int `yaml:"level"`
		} `yaml:"levels"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Levels) != 5 {
		t.Fatalf("уровней в контракте: %d", len(doc.Levels))
	}
	for i, l := range doc.Levels {
		if l.Level != i {
			t.Fatalf("уровень %d на месте %d", l.Level, i)
		}
	}
}

func appjournalRequest(p appjournal.Pending) appjournal.AppendRequest {
	return appjournal.AppendRequest{Batch: []appjournal.Pending{p}}
}
