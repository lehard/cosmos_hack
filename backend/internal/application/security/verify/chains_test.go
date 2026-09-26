package verify

import (
	"errors"
	"strings"
	"testing"
	"time"

	"ant/internal/contracts/procs"
)

// AD-9, AD-7: номер источника есть в журнале или в карантине; разрыв,
// объявленный потерей, и разрыв в окне ожидания — оговорка; разрыв, не
// объявленный после окна, — нарушение.
func TestSourceSeqGaps(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	mk := func(src string, n int64, at time.Time, run string) rec {
		return rec{chain: "main", sourceID: src, sourceSeq: n, committedAt: at, runID: run}
	}
	v := &run{in: Input{MaxGap: 5 * time.Minute}, checks: map[procs.VerifierReportV1ChecksElemCheck]*check{"source_seq": {name: "source_seq"}},
		idx: map[string][]rec{"main": {
			mk("old", 1, now.Add(-time.Hour), ""), mk("old", 4, now.Add(-time.Hour), ""),
			mk("declared", 1, now.Add(-time.Hour), ""), mk("declared", 3, now.Add(-time.Hour), ""),
			mk("fresh", 1, now.Add(-time.Minute), ""), mk("fresh", 3, now.Add(-time.Minute), ""),
			mk("quar", 1, now.Add(-time.Hour), ""), mk("quar", 3, now.Add(-time.Hour), ""),
			mk("tail", 1, now, ""),
		}},
		quar:   map[string][]int64{"quar": {2}},
		losses: map[string][][2]int64{"declared": {{2, 2}}},
	}
	v.sourceSeq()
	c := v.checks["source_seq"]
	var rejected, pending []string
	for _, f := range c.findings {
		switch f.Code {
		case "source_seq_gap":
			rejected = append(rejected, f.Detail)
		default:
			pending = append(pending, f.Detail)
		}
	}
	if len(rejected) != 1 || !strings.Contains(rejected[0], "источник old: номеров 2…3") {
		t.Fatalf("нарушения: %v", rejected)
	}
	if len(pending) != 2 || !c.rejected {
		t.Fatalf("оговорки: %v", pending)
	}
}

// AD-33, эпик 05: блок генезиса, проверенный по якорю, снимает оговорку
// «генезиса нет»; ошибка проверки и чужой отпечаток у хранителя — нарушение.
func TestGenesisCheck(t *testing.T) {
	mk := func(g *GenesisResult) *run {
		v := &run{in: Input{Genesis: g}, checks: map[procs.VerifierReportV1ChecksElemCheck]*check{}}
		for _, n := range []procs.VerifierReportV1ChecksElemCheck{"genesis", "signing_moment", "authority", "bpmn_quorum", "rendering", "build"} {
			v.checks[n] = &check{name: n}
		}
		v.genesis()
		v.pending()
		return v
	}
	if c := mk(nil).checks["genesis"]; !c.unverif || c.rejected {
		t.Fatalf("без генезиса — оговорка: %+v", c)
	}
	if c := mk(&GenesisResult{Records: 40, Digest: "d"}).checks["genesis"]; c.unverif || c.rejected || c.checked != 40 {
		t.Fatalf("генезис цел: %+v", c)
	}
	if c := mk(&GenesisResult{Err: errors.New("чужой якорь")}).checks["genesis"]; !c.rejected {
		t.Fatalf("ошибка генезиса — нарушение: %+v", c)
	}
}
