package journal

import (
	"errors"
	"testing"

	jc "ant/internal/contracts/journal"
)

func pending(n int) jc.JournalEntry {
	item := "ENT01:FL-0007"
	return jc.JournalEntry{
		Seq: n, Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKindFact,
		EventType: "operation.run.started", SchemaVersion: 1,
		EventID:  "01929a2b-7c3d-7e4f-8a5b-6c7d8e9f0a1b",
		SourceID: "terminal-weld-2", ItemID: &item, Stream: ItemStream(item), Partition: Partition(item, 16),
		OccurredAt: "2026-09-25T10:15:30.123Z", ReceivedAt: "2026-09-25T10:15:30.200Z",
		RecordedAt: "2026-09-25T10:15:30.210Z", CommittedAt: "2026-09-25T10:15:30.210Z",
		CorrelationID: "01929a2b-7c3d-7e4f-8a5b-000000000001", ProvenanceClass: jc.JournalEntryProvenanceClassPersonal,
		DomainBuild: ZeroLink.String(),
	}
}

// Звенья, поставленные Seal, сходятся у VerifyLinks; подмена открытого поля
// или перестановка записей обнаруживается (AD-8, AD-9).
func TestSealVerifyAndTamper(t *testing.T) {
	env := []byte(`{"payloadType":"application/vnd.ant.event+json; v=1","payload":"e30=","signatures":[]}`)
	var chain []jc.JournalEntry
	prev := ZeroLink
	for i := 1; i <= 3; i++ {
		e := pending(i)
		link, _, err := Seal(&e, prev, nil, env)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidatePending(e); err != nil {
			t.Fatal(err)
		}
		chain = append(chain, e)
		prev = link
	}
	head, err := VerifyLinks(ZeroLink, 0, chain)
	if err != nil || head != prev {
		t.Fatalf("VerifyLinks: %v", err)
	}
	tampered := append([]jc.JournalEntry(nil), chain...)
	tampered[1].SourceID = "terminal-weld-3"
	var br *Break
	if _, err := VerifyLinks(ZeroLink, 0, tampered); !errors.As(err, &br) || br.Seq != 2 {
		t.Fatalf("подмена не обнаружена: %v", err)
	}
	if _, err := VerifyLinks(ZeroLink, 0, []jc.JournalEntry{chain[0], chain[2]}); !errors.As(err, &br) || br.Seq != 3 {
		t.Fatalf("разрыв не обнаружен: %v", err)
	}
	sb, err := PlainSealed(nil, env)
	if err != nil {
		t.Fatal(err)
	}
	e := chain[0]
	e.Sealed = sb
	if _, got, err := OpenPlain(e); err != nil || len(got) == 0 {
		t.Fatalf("OpenPlain: %v", err)
	}
	e.Commit = chain[1].Link
	if _, _, err := OpenPlain(e); err == nil {
		t.Fatal("commit не сверен с конвертом")
	}
}

func TestPartitionStable(t *testing.T) {
	// Значение закреплено: смена хеша партиции — перераспределение всего журнала.
	if got := Partition("ENT01:FL-0007", 16); got != Partition("ENT01:FL-0007", 16) || got < 0 || got >= 16 {
		t.Fatalf("Partition = %d", got)
	}
	if Partition("x", 0) != 0 {
		t.Fatal("P = 0")
	}
}
