package main

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	app "ant/internal/application/security"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	"ant/internal/infrastructure/security/hybrid"
)

// chain — n записей основной цепочки с настоящими звеньями (AD-44).
func chain(t *testing.T, n int, tweak int) []app.Link {
	t.Helper()
	prev := dj.ZeroLink
	var out []app.Link
	for i := 1; i <= n; i++ {
		e := jc.JournalEntry{Seq: i, Chain: "main", EntryKind: "fact", EventType: "inspection.result.recorded", SchemaVersion: 1,
			EventID: "0190a8c4-1111-7000-8000-00000000000" + string(rune('0'+i)), SourceID: "s", Stream: "item:I", CorrelationID: "c",
			OccurredAt: "2026-09-26T10:00:00.000Z", ReceivedAt: "2026-09-26T10:00:00.000Z", RecordedAt: "2026-09-26T10:00:00.000Z",
			CommittedAt: "2026-09-26T10:00:00.000Z", ProvenanceClass: "device", DomainBuild: dj.ZeroLink.String()}
		env := []byte(`{"payload":"e30=","payloadType":"application/vnd.ant.event+json; v=1","signatures":[]}`)
		if i == tweak {
			env = []byte(`{"payload":"e30K","payloadType":"application/vnd.ant.event+json; v=1","signatures":[]}`)
		}
		link, open, err := dj.Seal(&e, prev, []byte("0123456789abcdef"), env)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, app.Link{Chain: "main", Seq: int64(i), Commit: e.Commit, OpenFieldsHash: dj.H(open).String(), Link: link.String()})
		prev = link
	}
	return out
}

func sub(links []app.Link, from int) app.HeadsSubmission {
	l := links[len(links)-1]
	return app.HeadsSubmission{Heads: []app.Head{{Chain: "main", Seq: l.Seq, Link: l.Link}, {Chain: "ca", Seq: 0, Link: dj.ZeroLink.String()}},
		Links: links[from:], CommittedAtFrom: "2026-09-26T10:00:00.000Z", CommittedAtTo: "2026-09-26T10:00:00.000Z"}
}

func newStore(t *testing.T) (*Store, hybrid.Anchors, string) {
	t.Helper()
	dir := t.TempDir()
	vdir := filepath.Join(dir, "verifier")
	if err := initTrust(dir, vdir, filepath.Join(dir, "ant"), "", filepath.Join(dir, "kek", "kek"), []string{"keeper"}); err != nil {
		t.Fatal(err)
	}
	signer, err := hybrid.Load(filepath.Join(dir, "keys"), "keeper")
	if err != nil {
		t.Fatal(err)
	}
	a, _, err := hybrid.LoadAnchors(filepath.Join(dir, AnchorsFile))
	if err != nil {
		t.Fatal(err)
	}
	st, err := OpenStore(dir, signer, a, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return st, a, vdir
}

// AD-8: головы принимаются, только если звенья сходятся; вилка и откат —
// отказ и тревога; точка подписана hybrid и продолжает предыдущую.
func TestKeeperAcceptsOnlyContinuation(t *testing.T) {
	st, anchors, vdir := newStore(t)
	links := chain(t, 5, 0)
	cp1, err := st.Submit(sub(links[:3], 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := anchors.Verify(cp1.Envelope, "checkpoint", "keeper"); err != nil {
		t.Fatalf("подпись точки: %v", err)
	}
	if s, l := cp1.Head("main"); s != 3 || l != links[2].Link || cp1.Payload.GenesisDigest == nil {
		t.Fatalf("точка 1: %+v", cp1.Payload)
	}
	cp2, err := st.Submit(sub(links, 3))
	if err != nil || cp2.Payload.CheckpointNo != 2 || *cp2.Payload.PreviousCheckpointDigest != cp1.Digest {
		t.Fatalf("точка 2: %+v %v", cp2.Payload, err)
	}
	if got := st.Links("main", 2, 10); len(got) != 3 || got[0].Seq != 3 {
		t.Fatalf("звенья у хранителя: %+v", got)
	}
	// Переписанная цепочка (другая запись 4 и пересчитанные звенья) — вилка.
	forged := chain(t, 6, 4)
	if _, err := st.Submit(sub(forged, 5)); !errors.Is(err, ErrFork) {
		t.Fatalf("вилка: %v", err)
	}
	if _, err := st.Submit(sub(links[:4], 4)); !errors.Is(err, ErrRollback) {
		t.Fatalf("откат: %v", err)
	}
	s := st.Status()
	if len(s.Alarms) != 2 || s.Alarms[0].Alert != "fork_attempt" || s.Alarms[1].Alert != "rollback_attempt" {
		t.Fatalf("тревоги: %+v", s.Alarms)
	}
	// Повтор без новых записей — сигнал жизни, новой точки нет.
	if cp, err := st.Submit(sub(links, 5)); err != nil || cp.Payload.CheckpointNo != 2 {
		t.Fatalf("сигнал жизни: %v", err)
	}
	// Молчание дольше 2N — тревога хранителя.
	st.now = func() time.Time { return time.Now().Add(time.Minute) }
	st.Watch()
	st.Watch()
	if a := st.Status().Alarms; len(a) != 3 || a[2].Alert != "heads_silent" {
		t.Fatalf("молчание: %+v", a)
	}
	// Отчёт верификатора: принимается только с подписью ключа верификатора.
	vs, err := hybrid.Load(filepath.Join(vdir, "keys"), "verifier")
	if err != nil {
		t.Fatal(err)
	}
	env, _ := vs.Sign("verifier-report", []byte(`{"format_version":1,"verdict":"intact"}`))
	if _, err := st.SubmitReport(env); err != nil {
		t.Fatalf("отчёт: %v", err)
	}
	ks, _ := hybrid.Load(filepath.Join(filepath.Dir(vdir), "keys"), "keeper")
	bad, _ := ks.Sign("verifier-report", []byte(`{"format_version":1,"verdict":"intact"}`))
	if _, err := st.SubmitReport(bad); !errors.Is(err, ErrInvalid) {
		t.Fatalf("чужой отчёт принят: %v", err)
	}
	// Состояние переживает перезапуск.
	re, err := OpenStore(st.dir, st.signer, anchors, 10*time.Second)
	if err != nil || len(re.checkpoints) != 2 || len(re.reports) != 1 || len(re.alarms) != 3 || len(re.links["main"]) != 5 {
		t.Fatalf("перезапуск: %v", err)
	}
}
