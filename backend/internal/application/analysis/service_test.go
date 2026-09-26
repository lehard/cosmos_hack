package analysis_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	appanalysis "ant/internal/application/analysis"
	"ant/internal/application/analysis/analysistest"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
)

// checked — журнал в памяти с проверкой конкурентности команд AD-39 по потоку
// (как journal.Append на Postgres; enginemem проверок не делает).
type checked struct{ *enginemem.Journal }

func (c checked) Append(ctx context.Context, rq appjournal.AppendRequest) (appjournal.AppendResult, error) {
	for _, ch := range rq.Checks {
		for _, e := range c.Entries() {
			info, _ := catalog.Lookup(catalog.Type(e.EventType))
			if ch.Stream != "" && e.Stream == ch.Stream && int64(e.Seq) > ch.BasisSeq && info.GuardRelevant {
				return appjournal.AppendResult{}, appjournal.Reject(appjournal.ErrStaleState, ch.BasisSeq, "stream", ch.Stream)
			}
		}
	}
	return c.Journal.Append(ctx, rq)
}

func memWorld(t *testing.T) (*analysistest.World, *enginemem.Journal) {
	t.Helper()
	j := enginemem.New(nil)
	store := checked{j}
	codec := &engineapp.Codec{Store: store, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}
	return analysistest.New(store, codec, "mem"), j
}

// «Станок сломался» на журнале в памяти: область 34 → 13 → 6 с основаниями,
// правило только расширяет, разбор на трёх дорожках, причина — решением человека.
func TestScenarioOnFakes(t *testing.T) {
	w, j := memWorld(t)
	analysistest.Scenario(t, w, j)
}

// Без зависимостей живая реализация отвечает 501 (заглушка волны 1).
func TestServiceWithoutDepsIsNotImplemented(t *testing.T) {
	_, err := appanalysis.NewService().RiskScope(context.Background(), "RS-1", platform.Moment{})
	if pe, ok := platform.AsError(err); !ok || pe.Code != "api.not_implemented" {
		t.Fatalf("%v", err)
	}
}

// Свёртка изделия движком (domain/engine.Fold, композиция AD-40): воркер
// пишет версию вывода разбора incident.hypothesis.computed и проекцию
// analysis.circumstances; повторная свёртка без новых данных версию не
// меняет (AD-3).
func TestWorkerWritesHypothesisVersion(t *testing.T) {
	w, j := memWorld(t)
	ctx := context.Background()
	if err := w.Story(ctx); err != nil {
		t.Fatal(err)
	}
	j.SetEpoch(appjournal.PartitionLease(0), 1)
	part := engineapp.Partition{Number: 0, Epoch: 1}
	feed := &enginemem.Feed{J: j, Parts: []engineapp.Partition{part}}
	worker := engineapp.NewWorker(engineapp.WorkerConfig{Feed: feed, Codec: w.Codec, Projections: w.Registry, Refresh: 50 * time.Millisecond, Backoff: 10 * time.Millisecond})
	step := func() {
		cctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		works, err := feed.Next(cctx, part)
		if err != nil {
			return
		}
		if err := worker.Process(ctx, part, works); err != nil {
			t.Fatal(err)
		}
	}
	step()
	var versions int
	for _, e := range j.Entries() {
		if e.EventType == string(catalog.IncidentHypothesisComputed) {
			versions++
			if e.ItemID == nil || *e.ItemID != "ENT01:F-017" || e.Stream != "item:ENT01:F-017" {
				t.Fatalf("реакция не в потоке изделия: %+v", e)
			}
		}
	}
	if versions != 1 {
		t.Fatalf("версий вывода разбора %d, ждали 1", versions)
	}
	raw, ok, _ := j.Get(ctx, appanalysis.ProjectionCircumstances, "ENT01:F-017")
	var iv appanalysis.ItemView
	if !ok || json.Unmarshal(raw, &iv) != nil || len(iv.State.Cases) != 1 || iv.Equipment == nil {
		t.Fatalf("проекция изделия: %s", raw)
	}
	step()
	n := 0
	for _, e := range j.Entries() {
		if e.EventType == string(catalog.IncidentHypothesisComputed) {
			n++
		}
	}
	if n != versions {
		t.Fatalf("пересвёртка без новых данных дала новую версию: %d", n)
	}
}
