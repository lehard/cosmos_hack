package security

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/contracts/procs"
	dj "ant/internal/domain/journal"
)

// Read журнала в памяти: цепочка, тип, после seq, с конца.
func (m *memJournal) Read(_ context.Context, q appjournal.ReadQuery) ([]jc.JournalEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []jc.JournalEntry
	for _, e := range m.es {
		chain := string(e.Chain)
		if chain == "" {
			chain = "main"
		}
		if (q.Chain != "" && q.Chain != chain) || (q.Chain == "" && chain != "main") || (q.EventType != "" && q.EventType != e.EventType) {
			continue
		}
		if !q.Backward && int64(e.Seq) <= q.AfterSeq {
			continue
		}
		out = append(out, e)
	}
	if q.Backward {
		slices.Reverse(out)
	}
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

// fakeKeeper — хранитель в памяти.
type fakeKeeper struct {
	Keeper
	subs   []HeadsSubmission
	report Report
	status KeeperStatus
}

func (k *fakeKeeper) LatestCheckpoint(context.Context) (Checkpoint, error) {
	return Checkpoint{}, ErrNotFound
}
func (k *fakeKeeper) SubmitHeads(_ context.Context, s HeadsSubmission) (Checkpoint, error) {
	k.subs = append(k.subs, s)
	return Checkpoint{}, nil
}
func (k *fakeKeeper) LatestReport(context.Context) (Report, error) { return k.report, nil }
func (k *fakeKeeper) Status(context.Context) (KeeperStatus, error) { return k.status, nil }

// AD-8: головы и звенья обеих цепочек уходят хранителю; AD-46: ant забирает
// отчёт и журналирует security.integrity.checked и нарушения с местом и
// типом один раз; тревоги хранителя — security.keeper.alert один раз.
func TestHeadsAndIntegrityPoller(t *testing.T) {
	ctx := context.Background()
	j := &memJournal{}
	enc := Encoder{DomainBuild: dj.ZeroLink.String()}
	emit := Emitter{Journal: j, Enc: enc}
	for range 3 {
		if _, err := emit.Emit(ctx, Out{Type: catalog.SecurityAuthFailed, OccurredAt: time.Now(), Data: map[string]any{}}); err != nil {
			t.Fatal(err)
		}
	}
	for i := range j.es {
		j.es[i].Commit, j.es[i].Link, j.es[i].CommittedAt = dj.ZeroLink.String(), dj.ZeroLink.String(), dj.FormatTime(time.Now())
	}
	k := &fakeKeeper{}
	if err := (&HeadsSender{Journal: j, Keeper: k}).Once(ctx); err != nil {
		t.Fatal(err)
	}
	if len(k.subs) != 1 || len(k.subs[0].Links) != 3 || k.subs[0].Heads[0].Seq != 3 || k.subs[0].Heads[1].Chain != "ca" {
		t.Fatalf("головы: %+v", k.subs)
	}
	seq := 42
	ca := "CA-3"
	k.report = Report{Digest: dj.ZeroLink.String(), Payload: procs.VerifierReportV1{
		Verdict: "violated", GeneratedAt: dj.FormatTime(time.Now()), Range: procs.VerifierReportV1Range{MainToSeq: 99},
		Checks: []procs.VerifierReportV1ChecksElem{{Check: "projections", Status: "rejected", Findings: []procs.VerifierReportV1ChecksElemFindingsElem{
			{Code: "projection_mismatch", Detail: "проекция расходится с журналом: изделие X в журнале «заблокировано» (CA-3), в проекции «разрешено»", Seq: &seq, CaRef: &ca}}}}}}
	k.status = KeeperStatus{Alarms: []KeeperAlarm{{No: 1, Alert: "fork_attempt", Chain: "main", Seq: 7, Detail: "цепочку переписали", At: time.Now().Format(time.RFC3339Nano)}}}
	p := &IntegrityPoller{Journal: j, Keeper: k, Emit: emit}
	for range 2 {
		if err := p.Once(ctx); err != nil {
			t.Fatal(err)
		}
	}
	count := map[string]int{}
	var violated map[string]any
	for _, e := range j.es {
		count[e.EventType]++
		if e.EventType == string(catalog.SecurityIntegrityViolated) {
			ev, _, _ := ParseEnvelope(j.env[e.EventID])
			_ = json.Unmarshal(ev.Data, &violated)
		}
	}
	if count[string(catalog.SecurityIntegrityChecked)] != 1 || count[string(catalog.SecurityIntegrityViolated)] != 1 || count[string(catalog.SecurityKeeperAlert)] != 1 {
		t.Fatalf("записи: %v", count)
	}
	if violated["violation"] != "projection_mismatch" || violated["ca_ref"] != "CA-3" || violated["seq"] != float64(42) {
		t.Fatalf("нарушение: %v", violated)
	}
	st, err := NewService(WithJournal(j), WithInterval(time.Minute)).Integrity(ctx)
	if err != nil || st.Status != "violated" || !st.ServerSide {
		t.Fatalf("индикатор: %+v %v", st, err)
	}
}
