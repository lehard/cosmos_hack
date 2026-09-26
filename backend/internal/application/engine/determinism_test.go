package engine_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	"ant/internal/domain/engine/enginetest"
	"ant/internal/domain/kernel"
)

// scenario — детерминированный вход нескольких изделий: поздние события,
// исправления, одинаковое время возникновения, решения (порядок AD-5).
func scenario() map[string][]kernel.Record {
	out := map[string][]kernel.Record{}
	seq := int64(0)
	add := func(item, outcome, corrects string, occurred time.Duration, kind catalog.Kind) {
		seq++
		typ := catalog.InspectionResultRecorded
		if kind == catalog.KindDecision {
			typ = catalog.DecisionNonconformityConfirmed
		}
		out[item] = append(out[item], kernel.Record{
			Seq: seq, EventID: fmt.Sprintf("0190%04d-0000-7000-8000-000000000000", seq), Type: typ, Kind: kind,
			ItemID: item, Stream: "item:" + item, OccurredAt: t0.Add(occurred), ReceivedAt: t0.Add(time.Duration(seq) * time.Second),
			RecordedAt: t0.Add(time.Duration(seq) * time.Second), Corrects: corrects, BasisSeq: seq - 1, Actor: "INS-01",
			Data: json.RawMessage(`{"outcome":"` + outcome + `"}`),
		})
	}
	for i := range 20 {
		item := fmt.Sprintf("ENT01:D-%02d", i)
		add(item, "defect_indicated", "", 10*time.Minute, catalog.KindFact)
		add(item, "no_defect_indicated", "", 10*time.Minute, catalog.KindFact) // то же occurred_at
		if i%3 == 0 {
			add(item, "", "", 20*time.Minute, catalog.KindDecision)
			add(item, "defect_indicated", "", 5*time.Minute, catalog.KindFact) // позднее событие
		}
		if i%4 == 0 {
			add(item, "no_defect_indicated", out[item][0].EventID, 30*time.Minute, catalog.KindFact)
		}
	}
	return out
}

// scenarioHash — хеш итога свёртки всех изделий сценария (порядок подачи
// перемешан по-разному в зависимости от shuffle).
func scenarioHash(t *testing.T, shuffle bool) string {
	t.Helper()
	sc := scenario()
	var lines []string
	for i := range 20 {
		item := fmt.Sprintf("ENT01:D-%02d", i)
		in := sc[item]
		if shuffle {
			rev := make([]kernel.Record, len(in))
			for k := range in {
				rev[len(in)-1-k] = in[k]
			}
			in = rev
		}
		s, rs := enginetest.Fold(engine.Bundle{}, in)
		h, err := engine.StateHash(s, rs)
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, item+"="+h)
	}
	return engine.Hash([]byte(strings.Join(lines, "\n")))
}

const helperEnv = "ANT_DETERMINISM_HELPER"

// TestDeterminismHelper — тело второго процесса: печатает хеш сценария.
func TestDeterminismHelper(t *testing.T) {
	if os.Getenv(helperEnv) == "" {
		t.Skip("вспомогательный процесс теста детерминизма")
	}
	fmt.Println("HASH " + scenarioHash(t, os.Getenv(helperEnv) == "shuffle"))
}

// Тест детерминизма (NFR-DET-1, AD-4): две свёртки одного входа в разных
// процессах (другая раскладка памяти, другой порядок обхода map в рантайме,
// другой порядок подачи) дают один хеш.
func TestDeterminismAcrossProcesses(t *testing.T) {
	local := scenarioHash(t, false)
	for _, mode := range []string{"plain", "shuffle"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestDeterminismHelper$", "-test.count=1")
		cmd.Env = append(os.Environ(), helperEnv+"="+mode, "GODEBUG=randautoseed=1", "GOMAXPROCS=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("второй процесс (%s): %v\n%s", mode, err, out)
		}
		var remote string
		for l := range strings.SplitSeq(string(out), "\n") {
			if h, ok := strings.CutPrefix(l, "HASH "); ok {
				remote = h
			}
		}
		if remote == "" || remote != local {
			t.Fatalf("хеши свёртки разошлись (%s): здесь %s, во втором процессе %q", mode, local, remote)
		}
	}
}
