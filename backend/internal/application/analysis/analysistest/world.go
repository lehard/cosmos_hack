// Пакет analysistest — опора тестов модуля analysis на фейках (журнал в
// памяти enginemem) и на своей БД (make dev-db): мир главной истории «плохой
// день сварочного участка» — факты, как их записал бы приём (эпик 06), шаг
// межизделийной стадии настоящим StageRunner через точку подключения
// crossitem.Fold → analysis.Stage (AD-42) и пересборка проекций analysis.*
// тем же кодом, что `ant rebuild` (AD-45).
//
// Слой: application (тестовая опора; без драйверов и сети). В сборку ролей не входит.
package analysistest

import (
	"context"
	"strconv"
	"strings"
	"time"

	appanalysis "ant/internal/application/analysis"
	crossitemapp "ant/internal/application/crossitem"
	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/crossitem"
	"ant/internal/domain/kernel"
)

// MSK — московское время сценария.
var MSK = time.FixedZone("MSK", 3*3600)

// At — момент «ДД ЧЧ:ММ» сентября 2026 (московское время).
func At(day, h, m int) time.Time { return time.Date(2026, 9, day, h, m, 0, 0, MSK).UTC() }

// Шаг и программа сварки главной истории.
const (
	StepWeld = "welding.weld"
	Program  = "WPS-12"
)

// Weld — сварка изделия: источник, сварщик, интервал.
type Weld struct {
	Item     string
	Src      string
	Welder   string
	From, To time.Time
}

// W — сварка «ДД ЧЧ:ММ-ЧЧ:ММ» (через полночь — следующий день).
func W(item, src, welder string, day, h1, m1, h2, m2 int) Weld {
	from, to := At(day, h1, m1), At(day, h2, m2)
	if to.Before(from) {
		to = to.Add(24 * time.Hour)
	}
	return Weld{Item: "ENT01:" + item, Src: src, Welder: welder, From: from, To: to}
}

// StoryWelds — сварки главной истории (scenarios/fixtures/flange-bad-day/world.yaml).
func StoryWelds() []Weld {
	return []Weld{
		W("F-001", "IS-2", "W21", 21, 9, 0, 9, 45), W("F-002", "IS-1", "W21", 21, 10, 0, 10, 40),
		W("F-003", "IS-1", "W21", 21, 10, 50, 11, 30), W("F-004", "IS-2", "W21", 21, 11, 40, 12, 20),
		W("F-005", "IS-1", "W21", 21, 12, 30, 13, 10), W("F-006", "IS-2", "W21", 21, 14, 10, 14, 55),
		W("F-007", "IS-1", "W21", 21, 15, 0, 15, 35), W("F-008", "IS-2", "W21", 21, 15, 40, 16, 20),
		W("F-009", "IS-1", "W22", 21, 16, 40, 17, 15), W("F-010", "IS-2", "W22", 21, 17, 30, 18, 10),
		W("F-011", "IS-1", "W22", 21, 18, 20, 18, 55), W("F-012", "IS-2", "W22", 21, 22, 10, 22, 50),
		W("F-013", "IS-1", "W22", 21, 19, 0, 19, 40), W("F-014", "IS-2", "W21", 22, 8, 20, 8, 43),
		W("F-015", "IS-2", "W21", 22, 10, 15, 11, 0), W("F-016", "IS-2", "W21", 22, 9, 20, 10, 5),
		W("F-017", "IS-2", "W21", 23, 10, 40, 11, 0), W("F-018", "IS-1", "W21", 22, 11, 40, 12, 15),
		W("F-019", "IS-2", "W22", 22, 16, 35, 17, 20), W("F-020", "IS-1", "W21", 22, 12, 20, 12, 55),
		W("F-021", "IS-2", "W22", 22, 19, 40, 20, 20), W("F-022", "IS-1", "W21", 22, 13, 0, 13, 35),
		W("F-023", "IS-2", "W21", 23, 8, 30, 9, 15), W("F-024", "IS-1", "W21", 22, 13, 40, 14, 15),
		W("F-025", "IS-2", "W21", 23, 9, 35, 10, 20), W("F-026", "IS-1", "W21", 22, 14, 20, 14, 55),
		W("F-027", "IS-1", "W22", 22, 18, 0, 18, 40), W("F-028", "IS-1", "W22", 22, 18, 50, 19, 25),
		W("F-029", "IS-1", "W22", 22, 21, 30, 22, 10), W("F-030", "IS-1", "W22", 21, 23, 0, 23, 35),
		W("F-031", "IS-1", "W22", 22, 20, 35, 21, 10), W("F-032", "IS-1", "W22", 22, 22, 20, 22, 55),
		W("F-033", "IS-1", "W21", 22, 15, 0, 15, 35), W("F-034", "IS-1", "W21", 22, 15, 40, 16, 15),
		W("F-035", "IS-1", "W21", 22, 11, 5, 11, 35), W("F-036", "IS-1", "W22", 22, 23, 50, 0, 25),
		W("F-221", "IS-2", "W22", 21, 19, 50, 20, 30), W("F-222", "IS-2", "W21", 22, 8, 45, 9, 10),
		W("F-223", "IS-1", "W22", 21, 20, 40, 21, 15), W("F-224", "IS-1", "W22", 22, 23, 5, 23, 40),
	}
}

// World — журнал, кодек и проекции; стадия и пересборка — как у ролей crossitem и rebuild.
type World struct {
	Journal  appjournal.JournalStore
	Codec    *engineapp.Codec
	Registry *engineapp.Registry
	stage crossitem.Stage
	after int64
	n     int
	tag   string
}

// New — мир над журналом; tag различает прогоны в одной БД.
func New(j appjournal.JournalStore, codec *engineapp.Codec, tag string) *World {
	reg := appanalysis.MustRegister(engineapp.NewRegistry())
	return &World{Journal: j, Codec: codec, Registry: reg, tag: tag}
}

// ID — детерминированный UUID записи мира.
func (w *World) ID(kind string) string {
	w.n++
	return kernel.UUIDv5(constants.NsAnt, "analysistest\x1f"+w.tag+"\x1f"+kind+"\x1f"+strconv.Itoa(w.n))
}

// Fact пишет запись как приём (эпик 06): вид из каталога, поток изделия или
// объекта. Возвращает event_id.
func (w *World) Fact(ctx context.Context, t catalog.Type, item, stream string, at time.Time, data any) (string, error) {
	info, _ := catalog.Lookup(t)
	id := w.ID(string(t))
	if stream == "" {
		stream = "item:" + item
	}
	p, err := w.Codec.Encode(ctx, engineapp.Out{EventID: id, Type: t, Kind: info.Kind, Stream: stream, ItemID: item,
		OccurredAt: at, Correlation: id, Data: data})
	if err != nil {
		return "", err
	}
	_, err = w.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}})
	return id, err
}

// WeldRun — выполнение сварки: начало и конец (факты process, publish: stage).
func (w *World) WeldRun(ctx context.Context, x Weld) error {
	run := "RUN-" + strings.TrimPrefix(x.Item, "ENT01:") + "-" + strconv.FormatInt(x.From.Unix(), 36)
	if _, err := w.Fact(ctx, catalog.OperationRunStarted, x.Item, "", x.From, map[string]any{"operation_run_id": run, "step_key": StepWeld,
		"operation_code": "030", "equipment_id": x.Src, "operator_id": x.Welder, "program_ref": Program}); err != nil {
		return err
	}
	_, err := w.Fact(ctx, catalog.OperationRunFinished, x.Item, "", x.To, map[string]any{"operation_run_id": run, "completion": "completed"})
	return err
}

// Story — мир до подтверждения НС-01 (прожог на Ф-017, Ср 11:08): 40 сварок,
// выпуск Ф-001 и Ф-006, кадр КТ-3 с прожогом и решение контролёра.
func (w *World) Story(ctx context.Context) error {
	welds := StoryWelds()
	for i := 1; i < len(welds); i++ {
		for k := i; k > 0 && welds[k].From.Before(welds[k-1].From); k-- {
			welds[k], welds[k-1] = welds[k-1], welds[k]
		}
	}
	for _, x := range welds {
		if err := w.WeldRun(ctx, x); err != nil {
			return err
		}
	}
	for _, rel := range []struct {
		item string
		at   time.Time
	}{{"ENT01:F-006", At(22, 15, 0)}, {"ENT01:F-001", At(23, 10, 30)}} {
		if _, err := w.Fact(ctx, catalog.ItemReleaseRecorded, rel.item, "", rel.at, map[string]any{"received_by": "STK-51",
			"warehouse_id": "WH-FG", "after_rework": false}); err != nil {
			return err
		}
	}
	if _, err := w.Fact(ctx, catalog.InspectionResultRecorded, "ENT01:F-017", "", At(23, 8, 50), map[string]any{"outcome": "no_defect_indicated",
		"phase": "before_operation", "method": "camera", "processing_state": "complete", "zone_ids": []string{"W-1.U2"}}); err != nil {
		return err
	}
	if _, err := w.Fact(ctx, catalog.InspectionResultRecorded, "ENT01:F-017", "", At(23, 11, 6), map[string]any{"outcome": "defect_indicated",
		"phase": "after_operation", "method": "camera", "processing_state": "complete", "zone_ids": []string{"W-1.U2"},
		"defects": []map[string]any{{"zone_id": "W-1.U2", "defect_type_code": "burn_through", "severity": "major"}}}); err != nil {
		return err
	}
	_, err := w.Fact(ctx, catalog.DecisionNonconformityConfirmed, "ENT01:F-017", "", At(23, 11, 8), map[string]any{"nc_id": "NC-01",
		"defect_type_code": "burn_through", "severity": "major", "signal_ids": []string{"SIG-01"}, "reason": map[string]any{"text": "Прожог шва У2"}})
	return err
}

// Settle — межизделийная стадия до неподвижной точки (StageRunner над
// crossitem.Fold, AD-42) и пересборка всех проекций analysis.* из журнала
// (Rebuilder, AD-45).
func (w *World) Settle(ctx context.Context) error {
	runner := &crossitemapp.StageRunner{Codec: w.Codec}
	for range 20 {
		es, err := w.readAfter(ctx, w.after)
		if err != nil {
			return err
		}
		if len(es) == 0 {
			break
		}
		next, rq, err := runner.Apply(ctx, w.stage, es)
		if err != nil {
			return err
		}
		w.after = int64(es[len(es)-1].Seq)
		w.stage = next
		if len(rq.Batch) == 0 {
			continue
		}
		rq.Effects = nil // состояние стадии держит мир; проекция crossitem.stage — забота роли crossitem
		if _, err := w.Journal.Append(ctx, rq); err != nil {
			return err
		}
	}
	_, err := (&engineapp.Rebuilder{Codec: w.Codec, Registry: w.Registry}).RebuildAll(ctx)
	return err
}

// jcEntry — запись журнала.
type jcEntry = jc.JournalEntry

func (w *World) readAfter(ctx context.Context, after int64) ([]jcEntry, error) {
	var out []jcEntry
	for {
		es, err := w.Journal.Read(ctx, appjournal.ReadQuery{AfterSeq: after, Limit: 1000})
		if err != nil {
			return nil, err
		}
		out = append(out, es...)
		if len(es) < 1000 {
			return out, nil
		}
		after = int64(es[len(es)-1].Seq)
	}
}

// Principal — контекст с автором решения (Principal.PersonID).
func Principal(ctx context.Context, person string) context.Context {
	return platform.WithPrincipal(ctx, platform.Principal{PersonID: person, Role: "technologist"})
}

// Clock — доменные часы теста (AD-37).
type Clock struct{ T time.Time }

// Now — доменное «сейчас».
func (c Clock) Now(context.Context) (time.Time, error) { return c.T, nil }

// Service — живая реализация analysis над миром.
func (w *World) Service(store engineapp.ProjectionStore, now time.Time) *appanalysis.Service {
	return appanalysis.NewLive(appanalysis.Config{
		Projections: store,
		Decisions:   appanalysis.JournalDecisions{Journal: w.Journal, DomainBuild: w.Codec.DomainBuild, Partitions: w.Codec.Partitions},
		Equipment:   appanalysis.JournalEquipmentLog{Journal: w.Journal, Codec: w.Codec},
		Clock:       Clock{T: now},
	})
}
