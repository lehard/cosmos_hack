package erp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
)

// Шлюз входящих (AD-7, AD-18, FR-31): повторный опрос учётной системы не
// подаёт уже поданное, повторная доставка в одном чтении — один факт; и то
// и другое — через обёртку хранилища, как в cmd/ant (встроенный OutboxStore).

type pullLedger struct {
	Ledger
	facts []Inbound
}

func (l pullLedger) Info() LedgerInfo                        { return LedgerInfo{System: "onec"} }
func (l pullLedger) Pull(context.Context) ([]Inbound, error) { return l.facts, nil }

type seenStore struct {
	OutboxStore
	next int64
	seen map[string]int64
}

func (s *seenStore) NextSourceSeq(_ context.Context, _ string, n int) (int64, error) {
	first := s.next + 1
	s.next += int64(n)
	return first, nil
}

func (s *seenStore) Seen(_ context.Context, _ string, ids []string) (map[string]int64, error) {
	out := map[string]int64{}
	for _, id := range ids {
		if q, ok := s.seen[id]; ok {
			out[id] = q
		}
	}
	return out, nil
}

func (s *seenStore) MarkSeen(_ context.Context, _ string, seqs map[string]int64) error {
	for id, q := range seqs {
		s.seen[id] = q
	}
	return nil
}

// wrapped — обёртка хранилища, как switchedStore/channelWatch в cmd/ant.
type wrapped struct{ OutboxStore }

type recIntake struct{ ids []string }

func (r *recIntake) Submit(_ context.Context, _ string, envs [][]byte) (IntakeResult, error) {
	for _, raw := range envs {
		var env struct{ Payload string }
		_ = json.Unmarshal(raw, &env)
		b, _ := base64.StdEncoding.DecodeString(env.Payload)
		var e struct {
			EventID string `json:"event_id"`
		}
		_ = json.Unmarshal(b, &e)
		r.ids = append(r.ids, e.EventID)
	}
	return IntakeResult{Accepted: len(envs)}, nil
}

func TestGatewayPullNoRepeatThroughWrapper(t *testing.T) {
	at := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	a := Inbound{Type: catalog.ErpOrderReceived, EventID: "11111111-1111-5111-8111-111111111111", OccurredAt: at, Data: map[string]any{"order_id": "ORD-0917"}}
	b := Inbound{Type: catalog.ReferenceExternalIdMapped, EventID: "22222222-2222-5222-8222-222222222222", Data: map[string]any{"system": "onec"}}
	in := &recIntake{}
	st := &seenStore{seen: map[string]int64{}}
	// Сбой «дубль» stand-а: каждая строка OData приходит дважды.
	o := &Outbox{Store: wrapped{st}, Ledger: pullLedger{facts: []Inbound{a, a, b, b}}, Intake: in}
	n, err := o.PullOnce(context.Background())
	if err != nil || n != 2 || len(in.ids) != 2 {
		t.Fatalf("первый опрос: подано %d (%v), %v", n, in.ids, err)
	}
	n, err = o.PullOnce(context.Background())
	if err != nil || n != 0 || len(in.ids) != 2 {
		t.Fatalf("повторный опрос подал заново: %d (%v), %v", n, in.ids, err)
	}
}
