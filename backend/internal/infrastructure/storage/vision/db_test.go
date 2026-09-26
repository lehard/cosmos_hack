package vision_test

import (
	"context"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	"ant/internal/application/platform"
	appvision "ant/internal/application/vision"
	dj "ant/internal/domain/journal"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
	storage "ant/internal/infrastructure/storage/vision"
)

// Затравка паспортов и живые операции vision на своей БД (make dev-db):
// запись генезисом через journal.Append, повтор migrate — дубль (ErrDuplicate
// журнала), чтение реестра, допуск новой версии с проверкой AD-39 потока.
func TestSeedAndLiveOnDB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	p := journaltest.NewDB(t).AppPool(t)
	j := journalstore.NewStore(p, journaltest.SysClock{})
	seed, err := storage.PassportsSeed()
	if err != nil {
		t.Fatal(err)
	}
	cfg := appvision.SeedConfig{Profile: "demo", DomainBuild: dj.ZeroLink.String(), Partitions: 2}
	n, err := appvision.SeedPassports(ctx, j, seed, cfg)
	if err != nil || n != len(seed.Passports) {
		t.Fatalf("затравка: %d %v", n, err)
	}
	if n, err := appvision.SeedPassports(ctx, j, seed, cfg); err != nil || n != 0 {
		t.Fatalf("повтор затравки: %d %v", n, err)
	}
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: 2}
	svc := appvision.NewService(appvision.WithDeps(appvision.Deps{Journal: j, Codec: codec, Routes: appvision.DemoRoutes{}}),
		appvision.WithConfig(appvision.Config{DomainBuild: dj.ZeroLink.String(), Partitions: 2}))
	pp, err := svc.Passport(ctx, "AP-KT3-WELD-1", platform.Moment{})
	if err != nil || pp.TrustLevel != 3 || pp.Status != "active" || *pp.Provenance != "genesis" {
		t.Fatalf("%+v %v", pp, err)
	}
	in := appvision.AdmitPassport{PassportID: "AP-KT3-WELD-2", AnalyzerID: "vqc-weld", Stage: "shadow", TrustLevel: 0, RecipeRef: "kt3-weld@2",
		Versions: map[string]string{"recipe_ref": "kt3-weld@2", "analyzer_version": "vqc-weld 2.4.0", "contract_version": "1.0"}, DocumentID: "DOC-2"}
	in.CommandID = "0190a000-0000-7000-8000-00000000bb01"
	if r, err := svc.AdmitPassport(ctx, in); err != nil || r.Seq == 0 {
		t.Fatalf("допуск: %+v %v", r, err)
	}
	l, err := svc.Analyzers(ctx, platform.Moment{})
	if err != nil || len(l.Items) != len(seed.Passports) {
		t.Fatalf("%+v %v", l, err)
	}
}
