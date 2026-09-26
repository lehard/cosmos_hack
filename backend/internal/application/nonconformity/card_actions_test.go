package nonconformity_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/nonconformity"
	"ant/internal/application/nonconformity/nctest"
	"ant/internal/application/platform"
)

// actionOf — решение карточки по операции и варианту.
func actionOf(t *testing.T, as []app.NCDecisionAction, op, disposition string) app.NCDecisionAction {
	t.Helper()
	i := slices.IndexFunc(as, func(a app.NCDecisionAction) bool {
		return a.Operation == op && (disposition == "" || a.Disposition != nil && *a.Disposition == disposition)
	})
	if i < 0 {
		t.Fatalf("нет решения %s %s: %+v", op, disposition, as)
	}
	return as[i]
}

// Интерфейс 7: решения карточки с доступностью (гарды команд) и
// последствиями — деловые отдельно от технических; после решения по
// изделию — кому передано исполнение и в каком оно состоянии.
func TestCardActionsAndHandoff(t *testing.T) {
	w := nctest.NewWorld(t)
	svc := w.Service(app.DemoRoutes{})
	item := "FL:0021"
	ncID, c := signalNC(t, w, svc, item)
	confirm := actionOf(t, c.ToDecide.Actions, "nonconformity.nonconformity.confirm", "")
	if !confirm.Allowed || len(confirm.Consequences) == 0 || len(confirm.TechnicalConsequences) == 0 ||
		!strings.HasPrefix(confirm.TechnicalConsequences[0], "Статус несоответствия") {
		t.Fatalf("подтвердить: %+v", confirm)
	}
	for _, line := range confirm.Consequences {
		if strings.HasPrefix(line, "1С:") || strings.HasPrefix(line, "История:") {
			t.Fatalf("техническая строка среди деловых: %q", line)
		}
	}
	if c.Handoff != nil {
		t.Fatalf("до решения исполнения нет: %+v", c.Handoff)
	}
	ctx := nctest.As("qc-1", "quality_inspector")
	if _, err := svc.Confirm(ctx, ncID, app.ConfirmNonconformity{CommandHeader: hdr(c.BasisSeq), SignalIDs: []string{c.Evidence.Signals[0].SignalID},
		Severity: "major", Reason: reason("Прожог подтверждён")}); err != nil {
		t.Fatal(err)
	}
	w.Settle()
	chief := nctest.As("chief-1", "technologist")
	c, _ = svc.Card(chief, ncID, platform.Moment{})
	rework := actionOf(t, c.ToDecide.Actions, "nonconformity.disposition.set", "rework")
	tech := strings.Join(rework.TechnicalConsequences, " | ")
	if !rework.Allowed || !strings.Contains(rework.Consequences[0], "Демо: подписи маршрута") ||
		!strings.Contains(strings.Join(rework.Consequences, " | "), "переделка на "+app.RunLabel("RUN-"+item)) ||
		!strings.Contains(tech, "1С: перевод в брак (переделка)") || !strings.Contains(tech, "История:") {
		t.Fatalf("переделка: %+v", rework)
	}
	asIs := actionOf(t, c.ToDecide.Actions, "nonconformity.disposition.set", "use_as_is")
	if asIs.Allowed || asIs.WhyAvailable == "" || !strings.Contains(asIs.Consequences[0], "разрешение на отклонение") {
		t.Fatalf("«как есть» без разрешения: %+v", asIs)
	}
	if _, err := svc.SetDisposition(chief, ncID, app.SetDisposition{CommandHeader: hdr(c.BasisSeq), Disposition: "rework", Reason: reason("переварить")}); err != nil {
		t.Fatal(err)
	}
	w.Settle()
	c, _ = svc.Card(ctx, ncID, platform.Moment{})
	h := c.Handoff
	if h == nil || h.RoleID != "site_foreman" || h.RoleLabel != "мастеру участка" || h.Status != "waiting" || h.StatusLabel != "ожидает исполнения" ||
		!strings.Contains(h.TaskTitle, "Переделка на ") || h.Since.IsZero() {
		t.Fatalf("передано на исполнение: %+v", h)
	}
	// Переделка началась — «исполняется», с исполнителем.
	w.Add(nctest.Run(item, w.Now.Add(time.Minute), "RUN-"+item+"-R", "op-8"))
	w.Settle()
	c, _ = svc.Card(ctx, ncID, platform.Moment{})
	if h := c.Handoff; h == nil || h.Status != "in_progress" || h.Person == nil || *h.Person != "op-8" {
		t.Fatalf("исполняется: %+v", c.Handoff)
	}
}
