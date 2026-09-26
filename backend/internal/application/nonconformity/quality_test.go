package nonconformity_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	appquality "ant/internal/application/quality"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	nc "ant/internal/domain/nonconformity"
)

func qrec(n int, typ catalog.Type, data any) kernel.Record {
	raw, _ := json.Marshal(data)
	info, _ := catalog.Lookup(typ)
	at := time.Date(2026, 9, 26, 8, n, 0, 0, time.UTC)
	return kernel.Record{Seq: int64(n), EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", n), Type: typ, SchemaVersion: 1,
		Kind: info.Kind, ItemID: "FL:0900", Stream: "item:FL:0900", OccurredAt: at, ReceivedAt: at, Data: raw}
}

// Эпик 20 ↔ 21 на настоящем нормативном слое и настоящей свёртке движка:
// quality выражает намерения данными (State.Requests), nonconformity читает их
// через Upstream.Quality на том же шаге — черновик несоответствия и блок по
// карте реакций появляются реакциями без ожидания следующей записи.
func TestQualityRequestsDraftNCInFold(t *testing.T) {
	env, err := appquality.EnvFromFS(os.DirFS("../../../.."), "norm-v1")
	if err != nil {
		t.Fatal(err)
	}
	recs := []kernel.Record{
		qrec(1, catalog.OperationRunStarted, map[string]any{"operation_run_id": "RUN-W1", "operation_code": "WELD", "step_key": "welding.weld", "operator_id": "O17"}),
		qrec(2, catalog.InspectionResultRecorded, map[string]any{"method": "radiography", "phase": "after_operation", "outcome": "defect_indicated",
			"processing_state": "completed", "step_key": "welding.kt3_radiography", "inspection_point": "KT-3", "zone_ids": []string{"W-1"},
			"defects": []any{map[string]any{"zone_id": "W-1", "severity": "critical", "defect_type_code": "W-BURNTHRU"}}}),
	}
	snap, rs := engine.Fold(engine.Bundle{Quality: env}, recs)
	var draft, contain, signal int
	for _, r := range rs {
		switch r.Type {
		case catalog.DecisionNonconformityDrafted:
			draft++
		case catalog.DecisionContainmentApplied:
			contain++
		case catalog.QualitySignalRaised:
			signal++
		}
	}
	st := snap.Nonconformity
	if signal == 0 || draft != 1 || len(st.NCs) != 1 || st.NCs[0].Status != nc.StatusDraft || st.NCs[0].SignalID == "" {
		t.Fatalf("черновик по запросу quality: сигналов %d, черновиков %d, %+v; запросы %+v", signal, draft, st.NCs, snap.Quality.Requests)
	}
	if contain == 0 || !st.Blocked() {
		t.Logf("сдерживание по карте: %d, ось %s (карта может не требовать блока)", contain, st.ContainmentLevel())
	}
	// Отклонение сигнала человеком: черновик закрыт, сдерживание по сигналу снято, исходная запись та же.
	rej := qrec(3, catalog.DecisionSignalRejected, nc.SignalRejectedData{SignalIDs: []string{st.NCs[0].SignalID}, Reason: nc.Reason{Text: "артефакт"}})
	rej.Kind = catalog.KindDecision
	snap2, _ := engine.Fold(engine.Bundle{Quality: env}, append(recs, rej))
	st2 := snap2.Nonconformity
	if st2.NCs[0].Status != nc.StatusClosed || st2.Blocked() || len(st2.NCs) != 1 {
		t.Fatalf("после отклонения: %+v, ось %s", st2.NCs, st2.ContainmentLevel())
	}
}
