package world

import (
	"context"
	"strings"
	"testing"
	"time"

	analysisapp "ant/internal/application/analysis"
	itemapp "ant/internal/application/item"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

// TestRuntime — мир заготовок через рантайм загрузчика: шаг курсора, as_of,
// префикс прогона, остановка на решении человека и продолжение (FR-129, AD-36, AD-38).
func TestRuntime(t *testing.T) {
	ctx := context.Background()
	lib, err := testLibrary()
	if err != nil {
		t.Fatal(err)
	}
	rt := loader.New(lib, loader.NewMemoryCursor())
	st, sc, err := rt.State(ctx)
	if err != nil || st.Step != sc.Manifest.InitialStep || !st.Paused {
		t.Fatalf("начальное положение: %+v %v", st, err)
	}
	// Разгар (Ср 13:45): область RS-01 — 6.
	var rs analysisapp.RiskScope
	if err := rt.Respond(ctx, "analysis.risk_scope.read", map[string]string{"incident_id": "RS-01"}, nil, &rs); err != nil {
		t.Fatal(err)
	}
	if got := rs.Versions[len(rs.Versions)-1].Size; got != 6 {
		t.Errorf("область на разгаре: %d", got)
	}
	// as_of в прошлом — шаг с часами ≤ as_of: Ср 11:30 → версия 1 (34).
	at := time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC)
	if err := rt.Respond(ctx, "analysis.risk_scope.read", map[string]string{"incident_id": "RS-01"}, &platform.Moment{AsOf: &at}, &rs); err != nil {
		t.Fatal(err)
	}
	if n := len(rs.Versions); n != 1 || rs.Versions[0].Size != 34 {
		t.Errorf("as_of Ср 11:30: версий %d", n)
	}
	// Нет ответа для параметров — 404; операция без заготовок — 501.
	var p itemapp.ItemPassport
	if err := rt.Respond(ctx, "item.passport.read", map[string]string{"item_id": "ENT01:F-999"}, nil, &p); err == nil || !strings.Contains(err.Error(), "api.not_found") {
		t.Errorf("нет изделия: %v", err)
	}
	// Прогон: префикс прогона в ID и снятие его во входе.
	if _, err := rt.Start(ctx, "flange-bad-day", "fx-test", "interactive", 0, 1000, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := rt.Respond(ctx, "item.passport.read", map[string]string{"item_id": "ENT01:fx-test/F-201"}, nil, &p); err != nil {
		t.Fatal(err)
	}
	if p.ItemID != "ENT01:fx-test/F-201" || p.OrderID != "fx-test/ORD-0911" {
		t.Errorf("префикс прогона: %s %s", p.ItemID, p.OrderID)
	}
	// Шаг ожидания решения: команда другого действия мир не двигает, нужная — двигает.
	wait := -1
	for i := 0; i < sc.Steps(); i++ {
		if sc.Header(i).Wait != nil {
			wait = i
			break
		}
	}
	if _, err := rt.Seek(ctx, wait); err != nil {
		t.Fatal(err)
	}
	w := sc.Header(wait).Wait
	if _, err := rt.Decide(ctx, "nonconformity.signal.reject", loader.ObjectRef{Kind: w.Object.Kind, ID: w.Object.ID}, platform.CommandMeta{}); err != nil {
		t.Fatal(err)
	}
	if st, _, _ := rt.State(ctx); st.Step != wait {
		t.Fatalf("чужое решение сдвинуло курсор: %d", st.Step)
	}
	// Интерактивный прогон на шаге ожидания стоит, сколько бы ни прошло времени.
	now := time.Now()
	_, _ = rt.Resume(ctx)
	_ = rt.Tick(ctx, now)
	_ = rt.Tick(ctx, now.Add(time.Hour))
	if st, _, _ := rt.State(ctx); st.Step != wait {
		t.Fatalf("прогон ушёл с шага ожидания: %d", st.Step)
	}
	rec, err := rt.Decide(ctx, w.Action, loader.ObjectRef{Kind: w.Object.Kind, ID: "fx-test/" + w.Object.ID}, platform.CommandMeta{CommandID: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	if st, _, _ := rt.State(ctx); st.Step != wait+1 || rec.Seq != loader.StepSeq(wait+1) {
		t.Fatalf("решение не продвинуло сценарий: шаг %d, seq %d", st.Step, rec.Seq)
	}
	// Идущий прогон ×1000: за минуту реального времени часы уходят вперёд до следующего ожидания.
	_ = rt.Tick(ctx, now.Add(2*time.Hour))
	_ = rt.Tick(ctx, now.Add(2*time.Hour+time.Minute))
	st, _, _ = rt.State(ctx)
	if st.Step <= wait+1 || sc.Header(st.Step).Wait == nil {
		t.Errorf("прогон не дошёл до следующего ожидания: шаг %d", st.Step)
	}
}
