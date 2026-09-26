package stand_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ingest "ant/internal/application/ingest"
	app "ant/internal/application/mes"
	dom "ant/internal/domain/mes"
	"ant/internal/infrastructure/integration/ingest/stands"
	"ant/internal/infrastructure/integration/mes/b2mml"
	"ant/internal/infrastructure/integration/mes/b2mml/stand"
)

func setup(t *testing.T) (*stand.Stand, *b2mml.Client, *httptest.Server) {
	t.Helper()
	st := stand.New(stand.Options{})
	srv := httptest.NewServer(stands.NewRegistry(st).Handler())
	t.Cleanup(srv.Close)
	c, err := b2mml.New(b2mml.Config{BaseURL: srv.URL + "/stand/mes/b2mml", Stand: true})
	if err != nil {
		t.Fatal(err)
	}
	return st, c, srv
}

func hold(id, item string, on bool) app.HoldMessage {
	return app.HoldMessage{MessageID: id, Key: item + "/hold", Hold: on, ItemID: item, Reason: "блок ОТК: НС-2026-0040",
		OccurredAt: time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC), Attempt: 1}
}

func fault(kind string, param int64, match string) ingest.Fault {
	return ingest.Fault{Kind: ingest.FaultKind(kind), Param: param, Match: match}
}

// Эпик 43 (FR-93): блок в MES подтверждён ConfirmBOD; заблокированный
// экземпляр не получает следующей операции; после снятия — получает, и
// событие операции приходит в Главный через канал MES.
func TestHoldBlocksNextOperation(t *testing.T) {
	st, c, _ := setup(t)
	ctx := context.Background()
	if d, err := c.Check(ctx); err != nil || !strings.Contains(d, stand.LogicalID) {
		t.Fatalf("сверка: %q %v", d, err)
	}
	h := hold("ca864e07-e236-5855-a32c-ae28ef14d9b0", "ENT01:F-001", true)
	if r, err := c.Post(ctx, h); err != nil || r.Outcome != "accepted" {
		t.Fatalf("блок: %+v %v", r, err)
	}
	if r, err := c.Post(ctx, h); err != nil || r.Outcome != "duplicate" {
		t.Fatalf("повтор блока: %+v %v", r, err)
	}
	if _, err := st.NextOperation("ENT01:F-001", "030"); !errors.Is(err, stand.ErrBlocked) {
		t.Fatalf("заблокированному выдана операция: %v", err)
	}
	if in, err := c.Pull(ctx); err != nil || len(in.Events) != 0 {
		t.Fatalf("событий быть не должно: %+v %v", in, err)
	}
	if r, err := c.Post(ctx, hold("0b6f0e1a-1111-5222-8333-444455556666", "ENT01:F-001", false)); err != nil || r.Outcome != "accepted" {
		t.Fatalf("снятие: %+v %v", r, err)
	}
	op, err := st.NextOperation("ENT01:F-001", "030")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.FinishOperation(op.Run); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Schedule(6); err != nil {
		t.Fatal(err)
	}
	in, err := c.Pull(ctx)
	if err != nil || len(in.Rejected) != 0 || len(in.Events) != 2 || len(in.Jobs) != len(stand.Operations) {
		t.Fatalf("входящие: %+v %v", in, err)
	}
	e := in.Events[0]
	if e.Category != dom.EventStart || e.SubLot != "ENT01:F-001" || e.OperationCode != "030" || e.RunRef != op.Run || in.Events[1].Category != dom.EventEnd {
		t.Fatalf("события: %+v", in.Events)
	}
	// Экземпляр — наш ID: шлюз переводит событие в факт без соответствия.
	if facts, d := dom.EventFacts(dom.Env{Steps: map[string]string{"030": "welding"}}, e); d != nil || len(facts) != 1 {
		t.Fatalf("факт операции: %+v %+v", facts, d)
	}
}

func TestFaults(t *testing.T) {
	st, c, srv := setup(t)
	ctx := context.Background()
	_ = st.Faults().Set(fault("error", 422, "hold:F-002"))
	r, err := c.Post(ctx, hold("11111111-1111-5111-8111-111111111111", "ENT01:F-002", true))
	if err != nil || r.Outcome != "rejected" || r.Code != "SUBLOT_UNKNOWN" {
		t.Fatalf("ошибка данных по образцу: %+v %v", r, err)
	}
	if r, err := c.Post(ctx, hold("22222222-2222-5222-8222-222222222222", "ENT01:F-003", true)); err != nil || r.Outcome != "accepted" {
		t.Fatalf("не по образцу: %+v %v", r, err)
	}
	st.Faults().Clear()

	_ = st.Faults().Set(fault("offline", 0, ""))
	if _, err := c.Post(ctx, hold("33333333-3333-5333-8333-333333333333", "ENT01:F-004", true)); err == nil {
		t.Fatal("недоступность: ожидалась ошибка")
	} else if _, ok := app.AsTransport(err); !ok {
		t.Fatalf("недоступность: %v", err)
	}
	st.Faults().Clear()

	_ = st.Faults().Set(fault("duplicate", 0, ""))
	h := hold("44444444-4444-5444-8444-444444444444", "ENT01:F-005", true)
	if _, err := c.Post(ctx, h); err == nil {
		t.Fatal("дубль: ответ должен потеряться")
	}
	if r, err := c.Post(ctx, h); err != nil || r.Outcome != "duplicate" {
		t.Fatalf("дубль: повтор — Duplicate: %+v %v", r, err)
	}
	st.Faults().Clear()

	_ = st.Faults().Set(fault("corrupt", 0, ""))
	if _, err := c.Check(ctx); err == nil {
		t.Fatal("несовместимая шина: ожидалась ошибка контракта")
	} else if _, ok := app.AsContract(err); !ok {
		t.Fatalf("несовместимая шина: %v", err)
	}
	st.Faults().Clear()

	// Страница и кнопка «выдать операцию» заблокированному — отказ на странице.
	resp, err := http.PostForm(srv.URL+"/stand/mes/ui/operation", map[string][]string{"sublot": {"ENT01:F-005"}, "code": {"040"}})
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("кнопка: %v %v", resp, err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(b), "отказала") || !strings.Contains(string(b), "заблокирован ОТК") {
		t.Fatalf("страница без отказа: %.400s", b)
	}
}
