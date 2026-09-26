package process_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"ant/internal/application/platform"
	app "ant/internal/application/process"
	"ant/internal/contracts/errcodes"
	dp "ant/internal/domain/process"
)

// UI-11: процессов может быть несколько — список процессов, параметр
// процесса у живой карты и списка версий; без параметра — основной процесс,
// даже если другой процесс введён в действие позже.
func TestProcessesAndProcessParam(t *testing.T) {
	w := newLiveWorld(t)
	ctx := context.Background()
	w.register("ent01:FL-0001", 0)
	w.sync()

	// Второй процесс — копия фланца с другим главным bpmn:process, введённая позже основного.
	xml := strings.ReplaceAll(string(flangeXML(t)), "Process_Flange", "Process_Copy")
	xml = strings.Replace(xml, `name="Фланец люка гермокорпуса в сборе"`, `name="Копия маршрута"`, 1)
	eff := t0.Add(24 * time.Hour)
	copyV := app.VersionRecord{ID: "copy-1", Label: "v1", Status: dp.StatusActive, Hash: dp.VersionHash([]byte(xml)), XML: []byte(xml),
		CreatedAt: eff, EffectiveFrom: &eff, Genesis: true}
	if err := w.store.Save(ctx, copyV); err != nil {
		t.Fatal(err)
	}

	pl, err := w.svc.Processes(ctx, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if len(pl.Items) != 2 {
		t.Fatalf("процессов: %+v", pl.Items)
	}
	f, c := pl.Items[0], pl.Items[1]
	if f.ProcessID != "Process_Flange" || !f.IsDefault || f.Name != "Фланец люка гермокорпуса в сборе" || f.ActiveVersion == nil ||
		f.ActiveVersion.VersionID != app.SeedVersionID || f.Versions != 1 || f.Status != "active" || f.ItemsInWork != 1 {
		t.Fatalf("основной процесс: %+v", f)
	}
	if c.ProcessID != "Process_Copy" || c.IsDefault || c.Name != "Копия маршрута" || c.ActiveVersion.VersionID != "copy-1" || c.ItemsInWork != 0 {
		t.Fatalf("второй процесс: %+v", c)
	}

	// Без параметра — основной процесс и его действующая версия.
	lm, err := w.svc.LiveMap(ctx, app.LiveMapQuery{}, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if lm.ProcessID != "Process_Flange" || lm.ProcessVersion.ProcessVersionID != app.SeedVersionID || len(lm.Items) != 1 || len(lm.Versions) != 1 {
		t.Fatalf("карта по умолчанию: %s %+v %d изделий", lm.ProcessID, lm.ProcessVersion, len(lm.Items))
	}
	lc, err := w.svc.LiveMap(ctx, app.LiveMapQuery{ProcessID: "Process_Copy"}, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if lc.ProcessID != "Process_Copy" || lc.ProcessName != "Копия маршрута" || lc.ProcessVersion.ProcessVersionID != "copy-1" || len(lc.Items) != 0 {
		t.Fatalf("карта второго процесса: %s %+v %d изделий", lc.ProcessID, lc.ProcessVersion, len(lc.Items))
	}
	// Версия чужого процесса — «не найдено».
	_, err = w.svc.LiveMap(ctx, app.LiveMapQuery{ProcessID: "Process_Flange", ProcessVersionID: "copy-1"}, platform.Moment{})
	if pe, ok := platform.AsError(err); !ok || pe.Code != errcodes.ApiNotFound {
		t.Fatalf("версия чужого процесса: %v", err)
	}

	// Список версий по процессу.
	vl, err := w.svc.Versions(ctx, "", platform.Moment{})
	if err != nil || len(vl.Items) != 1 || vl.Items[0].VersionID != app.SeedVersionID || vl.Items[0].ProcessID != "Process_Flange" {
		t.Fatalf("версии основного: %+v %v", vl, err)
	}
	vc, err := w.svc.Versions(ctx, "Process_Copy", platform.Moment{})
	if err != nil || len(vc.Items) != 1 || vc.Items[0].VersionID != "copy-1" {
		t.Fatalf("версии второго: %+v %v", vc, err)
	}
	if _, err := w.svc.Versions(ctx, "Process_None", platform.Moment{}); err == nil {
		t.Fatal("версии неизвестного процесса без отказа")
	}
	// Разница без «с чем» — с действующей версией своего процесса.
	d, err := w.svc.Diff(ctx, "copy-1", "", platform.Moment{})
	if err != nil || d.AgainstID != "copy-1" {
		t.Fatalf("разница второго процесса: %+v %v", d, err)
	}
}
