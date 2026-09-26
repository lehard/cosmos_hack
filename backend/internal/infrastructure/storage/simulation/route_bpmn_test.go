package simulation

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	processapp "ant/internal/application/process"
	qualityapp "ant/internal/application/quality"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality"
	sim "ant/internal/domain/simulation"
	dvision "ant/internal/domain/vision"
	storevision "ant/internal/infrastructure/storage/vision"
)

// Генератор против процесса фланца (normative/process/flange-process.bpmn):
// каждый шаг изделия, который строит генератор, проходим токеном BPMN — ни
// одного факта или решения «вне маршрута» (FR-44), ни одной точки
// предъявления без изделия на ней. Свёртка — та же композиция движка
// (domain/engine.Fold) с нормативным слоем от process.Bundles, что у воркера;
// решения людей — записями тех типов, что пишут их операции API.

const flangeBPMN = "normative/process/flange-process.bpmn"

// routeRecord — запись входа изделия: факт источника или решение человека.
type routeRecord struct {
	at    time.Time
	order int
	label string
	rec   kernel.Record
}

// decisionRecord — запись, которую пишет операция решения (только то, что
// читает исполнитель BPMN); ok=false — операция в процесс не входит.
func decisionRecord(a sim.Action, ids *sim.IDMap, hash string) (catalog.Type, map[string]any, bool) {
	body, _ := ids.ExpandMap(a.Body, false)
	params, _ := ids.ExpandMap(a.Params, false)
	get := func(m map[string]any, k string) any {
		if m == nil {
			return nil
		}
		return m[k]
	}
	switch a.Operation {
	case "item.item.register":
		d := map[string]any{"item_id": ids.Items[a.Item], "item_type_id": get(body, "item_type_id"), "item_revision": get(body, "item_revision"),
			"process_version_hash": hash, "normative_rev": "flange-1", "lot_ids": get(body, "lot_ids")}
		if v := get(body, "entry_step_key"); v != nil {
			d["entry_step_key"] = v
		}
		return catalog.ItemItemRegistered, d, true
	case "process.operation.start":
		d := map[string]any{"operation_run_id": get(body, "operation_run_id"), "operation_code": get(body, "operation_code"),
			"step_key": get(body, "step_key"), "operator_id": a.Actor}
		if v := get(body, "equipment_id"); v != nil {
			d["equipment_id"] = v
		}
		return catalog.OperationRunStarted, d, true
	case "process.operation.finish":
		return catalog.OperationRunFinished, map[string]any{"operation_run_id": get(params, "run_id"), "completion": get(body, "completion")}, true
	case "process.operation.pause":
		return catalog.OperationRunPaused, map[string]any{"operation_run_id": get(params, "run_id")}, true
	case "process.operation.resume":
		return catalog.OperationRunResumed, map[string]any{"operation_run_id": get(params, "run_id")}, true
	case "process.movement.send":
		return catalog.OperationMovementSent, map[string]any{"step_key": get(body, "step_key"), "to_location_id": get(body, "to_location_id")}, true
	case "process.movement.receive":
		return catalog.OperationMovementReceived, map[string]any{"step_key": get(body, "step_key"), "to_location_id": get(body, "to_location_id"),
			"destination_kind": get(body, "destination_kind"), "inspection_on_receipt": get(body, "inspection_on_receipt"), "received_by": a.Actor}, true
	case "item.presentation.record":
		return catalog.ItemPresentationRecorded, map[string]any{"step_key": get(body, "step_key"), "presentation_no": get(body, "presentation_no"),
			"presented_to": get(body, "presented_to")}, true
	case "nonconformity.presentation.resolve":
		return catalog.DecisionPresentationResolved, map[string]any{"step_key": get(body, "step_key"), "closing_point": get(body, "closing_point"),
			"resolution": get(body, "resolution"), "presentation_no": get(body, "presentation_no"), "method_event_ids": []string{}}, true
	case "nonconformity.nonconformity.confirm":
		return catalog.DecisionNonconformityConfirmed, map[string]any{"nc_id": "NC", "signal_ids": get(body, "signal_ids"),
			"severity": get(body, "severity"), "reason": map[string]string{"code": "x", "text": "x"}}, true
	case "nonconformity.disposition.set":
		return catalog.DecisionDispositionSet, map[string]any{"nc_id": "NC", "disposition": get(body, "disposition"),
			"concession_id": get(body, "concession_id"), "reason": map[string]string{"code": "x", "text": "x"}}, true
	case "nonconformity.item.isolate":
		return catalog.DecisionItemIsolated, map[string]any{}, true
	}
	return "", nil, false
}

// itemRoutes — записи входа каждого изделия прогона в порядке доставки:
// события источников (без потерянных и не по контракту) и решения людей
// (кроме шагов, которые ждут отказа).
func itemRoutes(t *testing.T, p *sim.Plan, hash string) map[string][]routeRecord {
	t.Helper()
	out := map[string][]routeRecord{}
	for _, e := range p.Emissions {
		if e.Item == "" || !e.Contract || e.Quarantine || e.Delivery == sim.DeliveryDuplicate || e.Delivery == sim.DeliveryConflict {
			continue
		}
		var env struct {
			Type string          `json:"event_type"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(e.Event, &env); err != nil {
			t.Fatal(err)
		}
		out[e.Item] = append(out[e.Item], routeRecord{at: e.DeliverAt, order: e.Order, label: e.Label,
			rec: kernel.Record{EventID: e.EventID, Type: catalog.Type(env.Type), Provenance: "device", OccurredAt: e.OccurredAt, Data: env.Data}})
	}
	// Групповое решение по несоответствию (NC-G1, S05) — без изделия в шаге:
	// модуль nonconformity пишет его в поток каждого изделия группы. Состав
	// группы здесь — изделия, которые комиссия подтвердила в области
	// инцидента того же сценария до решения (analysis.item.assess confirmed);
	// живой прогон берёт его из несоответствия окна спецпроцесса.
	confirmed := map[string][]string{}
	for _, a := range p.Actions {
		if a.Kind != sim.ActionDecision || a.Operation != "analysis.item.assess" || a.Item == "" || a.Body["assessment"] != "confirmed" {
			continue
		}
		if !slices.Contains(confirmed[a.Scenario], a.Item) {
			confirmed[a.Scenario] = append(confirmed[a.Scenario], a.Item)
		}
	}
	for _, a := range p.Actions {
		if a.Kind != sim.ActionDecision || a.Refusal != "" {
			continue
		}
		items := []string{a.Item}
		if a.Item == "" {
			if a.Operation != "nonconformity.disposition.set" {
				continue
			}
			items = confirmed[a.Scenario]
		}
		typ, data, ok := decisionRecord(a, p.IDs, hash)
		if !ok {
			continue
		}
		b, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			// решения идут после событий того же момента: раннер доставляет
			// наступившие события до шагов людей
			out[item] = append(out[item], routeRecord{at: a.At, order: 1 << 30, label: a.Label,
				rec: kernel.Record{EventID: p.IDs.CommandID(a.Seq) + "/" + item, Type: typ, Provenance: "personal", OccurredAt: a.At, Data: b}})
		}
	}
	for item, rs := range out {
		slices.SortStableFunc(rs, func(x, y routeRecord) int {
			if c := x.at.Compare(y.at); c != 0 {
				return c
			}
			return cmp.Compare(x.order, y.order)
		})
		itemID := "ent01:" + item
		for i := range rs {
			info, _ := catalog.Lookup(rs[i].rec.Type)
			r := &rs[i].rec
			r.Seq, r.Kind, r.ItemID, r.Stream = int64(i+1), info.Kind, itemID, "item:"+itemID
			r.ReceivedAt, r.RecordedAt = rs[i].at, rs[i].at
		}
		out[item] = rs
	}
	return out
}

// knownGaps — изделия, чей путь в карточках процессной сессии не проходим
// токеном по причинам вне данных сценариев (логика модулей, отчёт «сведение
// табло»). Изделие здесь обязано давать отказ: когда модуль починят, тест
// покраснеет — строку нужно убрать.
var knownGaps = map[string]map[string]string{
	"MS-1": {
		// Ф-015 уже в сборке, когда приходит групповое решение «переделка»
		// (NC-G1, S05): подпроцесс брака сборочного участка (АН) возвращает
		// токен в сборку, а карточка ведёт изделие обратно на сварку
		// (перемещение СИЦ → СЦ, переварка SV-015-2, повторная ЗТ-3). Пути из
		// сборочной дорожки на сварку BPMN не описывает — вопрос процессной
		// сессии (эпик 39); модуль его не выдумывает.
		"F-015": "переварка изделия, ушедшего со сварки: пути назад в процессе нет — вопрос процессной сессии",
	},
}

// seedPassports — паспорта допуска анализаторов затравки demo (роль migrate):
// без них результат камеры — уровень доверия 0 и в полноту не идёт (AD-29).
func seedPassports(t *testing.T) []quality.Passport {
	t.Helper()
	seed, err := storevision.PassportsSeed()
	if err != nil {
		t.Fatal(err)
	}
	at, err := time.Parse(time.RFC3339Nano, seed.AdmittedAt)
	if err != nil {
		t.Fatal(err)
	}
	var recs []kernel.Record
	for i, ps := range seed.Passports {
		v := ps.Versions
		vers := dvision.Versions{ItemRevision: v.ItemRevision, RecipeRef: v.RecipeRef, CameraConfig: v.CameraConfig, Calibration: v.Calibration,
			AnalyzerVersion: v.AnalyzerVersion, ThresholdProfile: v.ThresholdProfile, ContractVersion: v.ContractVersion, AppVersion: v.AppVersion}
		d := ev.AnalyzerPassportAdmittedV1{PassportID: ev.ObjectID(ps.PassportID), Stage: ev.AnalyzerPassportAdmittedV1Stage(ps.Stage),
			TrustLevel: ps.TrustLevel, RecipeRef: ps.RecipeRef, Versions: vers.Contract(), DocumentID: ev.ObjectID(ps.DocumentID)}
		b, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		recs = append(recs, kernel.Record{Seq: int64(i + 1), EventID: ps.PassportID, Type: catalog.AnalyzerPassportAdmitted,
			Kind: catalog.KindDecision, OccurredAt: at, ReceivedAt: at, RecordedAt: at, Data: b})
	}
	return quality.PassportsFrom(recs)
}

// TestRouteWalksFlangeBPMN — FR-44, AD-17: маршрут генератора по каждому
// изделию каждого прогона проходим токеном процесса фланца.
func TestRouteWalksFlangeBPMN(t *testing.T) {
	ctx := context.Background()
	xml, err := os.ReadFile(filepath.Join(repo, flangeBPMN))
	if err != nil {
		t.Fatal(err)
	}
	store := &processapp.MemVersions{}
	seed, err := processapp.EnsureSeed(ctx, store, xml, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	bundles := &processapp.Bundles{Store: store}
	// часть quality нормативного слоя (точки контроля, классификатор) —
	// полнота контроля на закрывающих точках, как у гарда nonconformity
	qenv, err := qualityapp.EnvFromFS(os.DirFS(repo), "flange-1")
	if err != nil {
		t.Fatal(err)
	}
	qenv.Passports = seedPassports(t)
	f := NewFiles(filepath.Join(repo, "scenarios"))
	runs, err := f.Runs()
	if err != nil {
		t.Fatal(err)
	}
	for _, run := range runs {
		t.Run(run, func(t *testing.T) {
			b, err := f.Bundle(ctx, run)
			if err != nil {
				t.Fatal(err)
			}
			p, err := sim.Generate(b, sim.Params{RunID: GoldenRunID(run, b.Run.Seed)})
			if err != nil {
				t.Fatal(err)
			}
			for _, it := range b.Run.Items {
				p.IDs.Items[it.ID] = "ent01:" + it.ID
			}
			routes := itemRoutes(t, p, seed.Hash)
			for _, it := range b.Run.Items {
				rs := routes[it.ID]
				labels := map[string]string{}
				in := make([]kernel.Record, 0, len(rs))
				for _, r := range rs {
					labels[r.rec.EventID] = r.label
					in = append(in, r.rec)
				}
				bundle, _, err := bundles.Bundle(ctx, "ent01:"+it.ID, in)
				if err != nil {
					t.Fatal(err)
				}
				bundle.Quality = qenv
				s, _ := engine.Fold(bundle, in)
				var bad []string
				// Полнота контроля на точке (FR-35, гард nonconformity): к
				// моменту решения «принять» у точки нет «нет данных» по плану.
				for i, r := range in {
					if r.Type != catalog.DecisionPresentationResolved {
						continue
					}
					var d struct {
						StepKey    string `json:"step_key"`
						Resolution string `json:"resolution"`
					}
					_ = json.Unmarshal(r.Data, &d)
					if d.Resolution != "accept" {
						continue
					}
					before, _ := engine.Fold(bundle, in[:i])
					for _, b := range quality.PresentationBlockers(before.Quality, bundle.Quality, d.StepKey) {
						// открытые сигналы закрывают решения по ссылкам {ref:SIG-…},
						// известным только живому прогону, — здесь не проверяются
						if b.Code == "open_signal" {
							continue
						}
						bad = append(bad, fmt.Sprintf("%s [quality]: %s на %s", labels[r.EventID], b.Code, b.StepKey))
					}
				}
				for _, r := range s.Process.Refusals {
					bad = append(bad, fmt.Sprintf("%s [%s]: %s — %s", labels[r.EventID], r.Kind, r.Code, r.Detail))
				}
				for k, e := range s.Process.Errors {
					bad = append(bad, "ошибка "+k+": "+e)
				}
				gap, known := knownGaps[run][it.ID]
				switch {
				case known && len(bad) == 0:
					t.Errorf("%s: известный разрыв «%s» больше не воспроизводится — убрать из knownGaps", it.ID, gap)
				case known:
					t.Logf("%s — известный разрыв (%s):\n  %s", it.ID, gap, strings.Join(bad, "\n  "))
				case len(bad) > 0:
					t.Errorf("%s (до %s): шаги не проходимы токеном BPMN:\n  %s", it.ID, it.Until, strings.Join(bad, "\n  "))
				}
				if it.Until == "release" && !slices.Contains(it.Skip, "release") && !s.Process.Completed {
					t.Errorf("%s: маршрут до выпуска, а изделие на %v", it.ID, s.Process.Steps())
				}
				// S01-11: «принято в работу» уходит по каждой ветке раздачи
				// (заготовка, патрубок, покупные на сборку) — регистрация до
				// параллельной раздачи, приёмы веток — шагами генератора.
				if it.Until == "release" && !slices.Contains(it.Skip, "release") {
					n := 0
					for _, th := range s.Process.Thrown {
						if th.ErpAction == "accept_into_work" {
							n++
						}
					}
					if n != 3 {
						t.Errorf("%s: «принято в работу» %d раз, ожидалось 3 (заготовка, патрубок, покупные)", it.ID, n)
					}
				}
			}
		})
	}
}
