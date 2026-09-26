package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	app "ant/internal/application/ingest"
	"ant/internal/application/ingest/inmem"
	appvision "ant/internal/application/vision"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/statuses"
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality"
	ovstand "ant/internal/infrastructure/integration/vision/operatorvision/stand"
	vqcstand "ant/internal/infrastructure/integration/vision/visionqc/stand"
	storequality "ant/internal/infrastructure/storage/quality"
	storevision "ant/internal/infrastructure/storage/vision"
)

// visionPath — stand-ы VisionQC и OperatorVision → локальный вход edge-агента
// → адаптеры → буфер → приём ядра (журнал в памяти). Возвращает ядро и записи
// журнала в представлении домена.
func visionPath(t *testing.T, seedPassports bool) (*inmem.Core, []kernel.Record) {
	t.Helper()
	ctx := context.Background()
	cfg := app.DefaultConfig()
	cfg.Profile = "demo"
	core := inmem.NewCore(cfg, nil, nil)
	if seedPassports {
		seed, err := storevision.PassportsSeed()
		if err != nil {
			t.Fatal(err)
		}
		if n, err := appvision.SeedPassports(ctx, core.Journal, seed, appvision.SeedConfig{Profile: "demo", DomainBuild: "test"}); err != nil || n == 0 {
			t.Fatalf("затравка паспортов: %d %v", n, err)
		}
	}
	coreSrv := httptest.NewServer(coreHandler(core.Service))
	defer coreSrv.Close()
	a, err := NewAgent(Config{SourceID: "edge-vision-1", CoreURL: coreSrv.URL, StateDir: t.TempDir(), SourceKind: "camera", Reliability: "high"},
		Unsigned{Ref: "device-edge-vision-1@1"}, coreSrv.Client(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	routes, err := VisionRoutes(coreSrv.URL, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	local := httptest.NewServer(LocalHandler(a, NewExtractor(), routes...))
	defer local.Close()

	now := time.Now().UTC().Truncate(time.Second)
	kt3 := vqcstand.New(vqcstand.Options{Name: "visionqc-kt3", EdgeURL: local.URL})
	kt3.Client, kt3.Now = local.Client(), func() time.Time { now = now.Add(time.Minute); return now }
	for range vqcstand.MainStory {
		if err := kt3.Tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	ov := ovstand.New(ovstand.Options{Name: "operatorvision-asm", EdgeURL: local.URL})
	ov.Client, ov.Now = local.Client(), kt3.Now
	if err := ov.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Drain(ctx); err != nil {
		t.Fatal(err)
	}
	codec := &engineapp.Codec{Store: core.Journal, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", Partitions: 1}
	var recs []kernel.Record
	for _, s := range core.Journal.Main() {
		d, err := codec.Decode(ctx, s.Entry)
		if err != nil {
			t.Fatalf("seq %d: %v", s.Entry.Seq, err)
		}
		recs = append(recs, d.Record)
	}
	return core, recs
}

// Эпик 33: stand → edge → приём → наблюдение с вектором версий (AD-29):
// все семь наблюдений главной истории КТ-3 и гипотеза OperatorVision приняты,
// привязаны к изделиям, у каждого — восемь составляющих вектора версий.
func TestStandEdgeIngestVersions(t *testing.T) {
	core, recs := visionPath(t, false)
	st, _ := core.Service.Stats(context.Background())
	if st.Quarantined != 0 || st.Accepted != int64(len(vqcstand.MainStory)+1) {
		t.Fatalf("приём: %+v", st)
	}
	n := 0
	for _, r := range recs {
		switch r.Type {
		case catalog.InspectionResultRecorded:
			n++
			var d struct {
				Versions map[string]string `json:"versions"`
				Stages   []map[string]any  `json:"stages"`
			}
			_ = json.Unmarshal(r.Data, &d)
			if len(d.Versions) != 8 || d.Versions["recipe_ref"] != "kt3-weld@1" || r.SourceKind != "camera" || r.ItemID == "" || r.Provenance != "device" {
				t.Fatalf("наблюдение %s: %+v %+v", r.EventID, r, d.Versions)
			}
			for k, v := range d.Versions {
				if v == "" || v == "unknown" {
					t.Fatalf("составляющая %s: %q", k, v)
				}
			}
		case catalog.OperatorActionObserved:
			if r.ItemID != "ENT01:F-231" || !json.Valid(r.Data) {
				t.Fatalf("гипотеза: %+v", r)
			}
		}
	}
	if n != len(vqcstand.MainStory) {
		t.Fatalf("наблюдений %d", n)
	}
}

// fold — свёртка quality по записям изделия (как шаг движка, без поздних модулей).
func fold(env quality.Env, in []kernel.Record) quality.State {
	var s quality.State
	for _, r := range in {
		s = quality.Reduce(s, r, env, quality.Upstream{})
		for _, it := range quality.React(s, env, quality.Upstream{}).Intents {
			if it.Target == quality.Module {
				s = quality.Apply(s, it)
			}
		}
	}
	return s
}

func items(recs []kernel.Record) map[string][]kernel.Record {
	out := map[string][]kernel.Record{}
	for _, r := range recs {
		if r.ItemID != "" {
			out[r.ItemID] = append(out[r.ItemID], r)
		}
	}
	for k := range out {
		slices.SortStableFunc(out[k], func(a, b kernel.Record) int {
			if kernel.Less(a, b) {
				return -1
			}
			return 0
		})
	}
	return out
}

// Эпик 33: паспорт уровня 3 из затравки даёт блок по карте реакций качества
// (прожог КТ-3 → item_hold), без паспорта — только запись (уровень 0);
// испорченный кадр 0,3 и блик — «оценка невозможна», а не «годно».
func TestPassportLevel3BlocksAndBadFrameUnable(t *testing.T) {
	base, err := storequality.SeedEnv("normative-seed-v1")
	if err != nil {
		t.Fatal(err)
	}
	_, recs := visionPath(t, true)
	var passports []kernel.Record
	for _, r := range recs {
		if r.Type == catalog.AnalyzerPassportAdmitted {
			passports = append(passports, r)
		}
	}
	env := base
	env.Passports = quality.PassportsFrom(passports)
	byItem := items(recs)

	// Главная история: F-231 годный, F-232 поры, F-233 испорченный кадр,
	// F-234 годный, F-235 блик, F-236 прожог, F-237 подрез.
	burn := fold(env, byItem["ENT01:F-236"])
	if len(burn.Observations) != 1 || burn.Observations[0].TrustLevel != 3 || burn.Observations[0].PassportID != "AP-KT3-WELD-1" {
		t.Fatalf("прожог: уровень доверия %+v", burn.Observations)
	}
	if len(burn.Signals) != 1 || burn.Signals[0].Assessment.Containment != statuses.ContainmentItemHold || !burn.Signals[0].Raised || burn.Signals[0].Assessment.Suppressed {
		t.Fatalf("прожог при паспорте уровня 3 — не блок: %+v", burn.Signals)
	}
	under := fold(env, byItem["ENT01:F-237"])
	if len(under.Signals) != 1 || under.Signals[0].Assessment.Containment != statuses.ContainmentItemHold {
		t.Fatalf("подрез: %+v", under.Signals)
	}
	noPass := fold(base, byItem["ENT01:F-236"])
	if noPass.Observations[0].TrustLevel != 0 || noPass.Observations[0].TrustNote != "no_qualified_analyzer" ||
		len(noPass.Signals) != 1 || noPass.Signals[0].Raised || !noPass.Signals[0].Assessment.Suppressed {
		t.Fatalf("без паспорта анализатор блокирует: %+v %+v", noPass.Observations, noPass.Signals)
	}

	bad := fold(env, byItem["ENT01:F-233"])
	if o := bad.Observations[0]; o.Reported != quality.OutcomeNoDefect || o.Outcome != quality.OutcomeUnable || o.UnableReason != "poor_image" || *o.QualityBP != 3000 {
		t.Fatalf("испорченный кадр: %+v", o)
	}
	if bad.Axis != statuses.QualityUnableToAssess {
		t.Fatalf("ось после кадра 0,3: %s", bad.Axis)
	}
	glare := fold(env, byItem["ENT01:F-235"])
	if o := glare.Observations[0]; o.Outcome != quality.OutcomeUnable || o.UnableReason != "poor_image" || glare.Axis != statuses.QualityUnableToAssess {
		t.Fatalf("блик: %+v, ось %s", o, glare.Axis)
	}
	ok := fold(env, byItem["ENT01:F-231"])
	if ok.Axis == statuses.QualityConforming {
		t.Fatal("«годно» ставит только человек")
	}
}
