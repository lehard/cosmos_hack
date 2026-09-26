package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	app "ant/internal/application/analysis"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	_ "ant/internal/infrastructure/fixtures/world" // встроенный мир заготовок (loader.Builtin)
)

func code(err error) errcodes.Code {
	var pe *platform.Error
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

// recordWithText — event_id записи с текстом из обстоятельств несоответствия.
func recordWithText(t *testing.T, c app.Circumstances) string {
	t.Helper()
	b, _ := json.Marshal(c)
	var g any
	_ = json.Unmarshal(b, &g)
	var id string
	var walk func(any)
	walk = func(x any) {
		switch v := x.(type) {
		case map[string]any:
			if s, ok := v["event_id"].(string); ok && id == "" && v["event_type"] != nil && v["text"] != nil {
				id = s
			}
			for _, y := range v {
				walk(y)
			}
		case []any:
			for _, y := range v {
				walk(y)
			}
		}
	}
	walk(g)
	if id == "" {
		t.Fatal("в обстоятельствах НС-01 нет записи с текстом")
	}
	return id
}

// TestTechnologistOverlay — сужение области риска: гарды live (основание
// обязательно, только изделия текущей области), новая ступень от текущей
// области с исключённым изделием, доказательствами словами и именем автора;
// размер инцидента; назначенная мера — в списке мер «назначена».
func TestTechnologistOverlay(t *testing.T) {
	ctx := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "TEC-01", Role: "technologist"})
	a := New()
	rs, err := a.RiskScope(ctx, "RS-01", platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	cur := inScope(rs.Items)
	if len(cur) < 2 || !slices.ContainsFunc(cur, func(x app.ScopeItem) bool { return x.ItemID == "ENT01:F-015" }) {
		t.Fatalf("текущая область RS-01: %+v", cur)
	}
	circ, err := a.Circumstances(ctx, "NC-01", platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	ev := recordWithText(t, circ)
	hdr := func(id string) platform.CommandHeader { return platform.CommandHeader{CommandID: id} }
	// Без доказательств — 422, как у live.
	_, err = a.NarrowScope(ctx, "RS-01", app.ChangeScope{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000b000"), ItemIDs: []string{"ENT01:F-015"},
		EvidenceEventIDs: []string{}, Reason: app.Reason{Text: "рентген чистый"}})
	if code(err) != errcodes.IncidentBasisRequired {
		t.Fatalf("сужение без доказательств: %v", err)
	}
	// Изделие вне текущей области — 422.
	var out string
	for _, x := range rs.Items {
		if x.Known == "excluded" {
			out = x.ItemID
		}
	}
	_, err = a.NarrowScope(ctx, "RS-01", app.ChangeScope{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000b00a"), ItemIDs: []string{out},
		EvidenceEventIDs: []string{ev}, Reason: app.Reason{Text: "x"}})
	if code(err) != errcodes.IncidentItemNotInScope {
		t.Fatalf("сужение изделия вне области: %v", err)
	}
	if _, err := a.NarrowScope(ctx, "RS-01", app.ChangeScope{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000b001"), ItemIDs: []string{"ENT01:F-015"},
		EvidenceEventIDs: []string{ev}, Reason: app.Reason{Text: "рентген чистый"}}); err != nil {
		t.Fatal(err)
	}
	after, _ := a.RiskScope(ctx, "RS-01", platform.Moment{})
	last := after.Versions[len(after.Versions)-1]
	if len(after.Versions) != len(rs.Versions)+1 || last.Change != "narrowed" || last.Size != len(cur)-1 ||
		!slices.Equal(last.ItemsRemoved, []string{"ENT01:F-015"}) || len(inScope(after.Items)) != len(cur)-1 {
		t.Fatalf("ступень сужения: %+v", last)
	}
	if len(last.Evidence) != 1 || last.Evidence[0].EventID != ev || last.Evidence[0].Text == nil || *last.Evidence[0].Text == "" {
		t.Fatalf("доказательства ступени: %+v", last.Evidence)
	}
	if last.AuthorName == nil || *last.AuthorName == "" || last.Author == nil || *last.Author != "TEC-01" {
		t.Fatalf("автор ступени: %+v %+v", last.Author, last.AuthorName)
	}
	l, err := a.Incidents(ctx, platform.Moment{}, platform.Page{})
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range l.Items {
		if x.IncidentID == "RS-01" && (x.Size != len(cur)-1 || x.ScopeVersion != len(after.Versions) || x.Counts.Excluded != len(after.Items)-len(cur)+1) {
			t.Fatalf("инцидент: размер %d версия %d счётчики %+v, в области %d", x.Size, x.ScopeVersion, x.Counts, len(cur)-1)
		}
	}
	// Расширение возвращает исключённое изделие; всё уже в области — 422.
	if _, err := a.ExpandScope(ctx, "RS-01", app.ChangeScope{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000b003"), ItemIDs: []string{"ENT01:F-015"},
		Reason: app.Reason{Text: "новые данные"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ExpandScope(ctx, "RS-01", app.ChangeScope{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000b004"), ItemIDs: []string{"ENT01:F-015"},
		Reason: app.Reason{Text: "ещё раз"}}); code(err) != errcodes.ApiValidationFailed {
		t.Fatalf("расширение изделием из области: %v", err)
	}
	if rs2, _ := a.RiskScope(ctx, "RS-01", platform.Moment{}); len(inScope(rs2.Items)) != len(cur) || !slices.Equal(rs2.Versions[len(rs2.Versions)-1].ItemsAdded, []string{"ENT01:F-015"}) {
		t.Fatalf("расширение: %+v", rs2.Versions[len(rs2.Versions)-1])
	}
	before, _ := a.CorrectiveActions(ctx, platform.Moment{})
	if _, err := a.AssignAction(ctx, "RS-01", app.AssignAction{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000b002"), ActionType: "corrective_action",
		Direction: "prevent_occurrence", OwnerID: "FOR-WC", Title: "Заменить кабель ИС-2", EffectivenessPlan: app.EffectivenessPlanInput{Metric: "отклонения тока", WindowDays: 14}}); err != nil {
		t.Fatal(err)
	}
	ca, _ := a.CorrectiveActions(ctx, platform.Moment{})
	if len(ca.Items) != len(before.Items)+1 || ca.Items[0].Status != "assigned" || ca.Items[0].Plan.WindowDays != 14 || ca.Items[0].IncidentID != "RS-01" {
		t.Fatalf("мера: %+v", ca.Items[0])
	}
}
