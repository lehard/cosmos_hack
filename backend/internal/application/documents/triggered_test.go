package documents_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	dom "ant/internal/domain/documents"
	"ant/internal/domain/kernel"
)

// TestTriggeredDocuments — эпик 44: документы каталога по событию-триггеру
// из нормативного слоя репозитория. Запись вмешательства — журнал изделия
// (открытие и закрытие — две версии, строка на запись); решение «списать» —
// акт о браке, «ремонт» — разрешение на отклонение, «переделка» — ни того ни
// другого; предъявление ВП — извещение, предъявление ОТК — только журнал.
func TestTriggeredDocuments(t *testing.T) {
	env := repoEnv(t)
	const item = "ENT01:F-015"
	t0 := time.Date(2026, 9, 23, 12, 50, 0, 0, time.UTC)
	n := 0
	rec := func(tp catalog.Type, kind catalog.Kind, data string) kernel.Record {
		n++
		return kernel.Record{Seq: int64(n), EventID: "ev-" + string(rune('a'+n)), Type: tp, Kind: kind, ItemID: item, Stream: "item:" + item,
			OccurredAt: t0.Add(time.Duration(n) * time.Minute), Actor: "FOR-AC@1", Data: json.RawMessage(data)}
	}
	s := dom.State{}
	for _, r := range []kernel.Record{
		rec(catalog.ItemInterventionOpened, catalog.KindDecision, `{"intervention_id":"INT-1","zone_ids":["S-1"],"purpose":"Возврат в сварку"}`),
		rec(catalog.ItemInterventionClosed, catalog.KindDecision, `{"intervention_id":"INT-1","recheck_event_ids":[],"retest_required":false}`),
		rec(catalog.DecisionDispositionSet, catalog.KindDecision, `{"nc_id":"NC-06","disposition":"scrap","reason":"Пористость нового шва"}`),
		rec(catalog.DecisionDispositionSet, catalog.KindDecision, `{"nc_id":"NC-07","disposition":"repair","reason":"Подварка"}`),
		rec(catalog.DecisionDispositionSet, catalog.KindDecision, `{"nc_id":"NC-08","disposition":"rework","reason":"Переварка"}`),
		rec(catalog.ItemPresentationRecorded, catalog.KindFact, `{"step_key":"final.zt6_acceptance","presentation_no":1,"presented_to":"customer_representative","presented_by":"FOR-AC"}`),
		rec(catalog.ItemPresentationRecorded, catalog.KindFact, `{"step_key":"welding.zt3_acceptance","presentation_no":1,"presented_to":"qc","presented_by":"W21"}`),
	} {
		s = dom.Reduce(s, r, env, dom.Upstream{})
	}
	byTemplate := map[string][]dom.Doc{}
	for _, d := range s.Docs {
		id, _, _ := strings.Cut(d.TemplateRef, "@")
		byTemplate[id] = append(byTemplate[id], d)
	}
	iv := byTemplate["intervention-record"]
	if len(iv) != 1 || iv[0].ID != "INTERVENTION-RECORD-"+item || len(iv[0].Versions) != 2 {
		t.Fatalf("запись вмешательства: %+v", iv)
	}
	cur := iv[0].Current()
	if !strings.Contains(string(cur.Content), "Возврат в сварку") && !strings.Contains(string(cur.Content), "INT-1") {
		t.Errorf("содержимое вмешательства: %s", cur.Content)
	}
	if !strings.Contains(string(cur.Content), `"item.intervention.closed"`) {
		t.Errorf("строки журнала: %s", cur.Content)
	}
	if len(cur.SourceSigners) != 1 || cur.SourceSigners[0].Person != "FOR-AC" {
		t.Errorf("этап «мастер — вскрытие» закрывает автор записи: %+v", cur.SourceSigners)
	}
	if len(byTemplate["scrap-act"]) != 1 || len(byTemplate["concession"]) != 1 {
		t.Errorf("акт о браке %d (ожидался 1), разрешение на отклонение %d (ожидалось 1)", len(byTemplate["scrap-act"]), len(byTemplate["concession"]))
	}
	act := byTemplate["scrap-act"][0].Current()
	if len(act.Stages) != 3 || !strings.Contains(string(act.Content), "Пористость нового шва") {
		t.Errorf("акт о браке: этапов %d, %s", len(act.Stages), act.Content)
	}
	if len(byTemplate["customer-presentation-notice"]) != 1 {
		t.Errorf("извещение ВП — только по предъявлению представителю заказчика: %d", len(byTemplate["customer-presentation-notice"]))
	}
	if pl := byTemplate["presentation-log"]; len(pl) != 1 || len(pl[0].Versions) != 2 {
		t.Errorf("журнал предъявления ОТК: одна запись изделия, две версии: %+v", pl)
	}
	for _, id := range []string{"acceptance-certificate", "passport", "nc-label"} {
		if len(byTemplate[id]) != 0 {
			t.Errorf("%s без триггера оформлен", id)
		}
	}
}
