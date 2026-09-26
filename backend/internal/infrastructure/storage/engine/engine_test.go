package engine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	storage "ant/internal/infrastructure/storage/engine"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// pool — чистая база на своей БД агента (make dev-db) с миграциями журнала и
// движка, пул роли приложения ant_app (как у роли ant); без БД тест пропускается.
func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return journaltest.NewDB(t).AppPool(t)
}

// Эффекты в транзакции, журнал изменений и LISTEN/NOTIFY (AD-6, AD-45).
func TestEffectsAndNotify(t *testing.T) {
	p := pool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	st := &storage.Store{Pool: p}
	defer st.Close()

	waited := make(chan error, 1)
	// Первый Wait открывает LISTEN; сигнал до него не теряется благодаря After.
	listenReady := make(chan struct{})
	wctx, wcancel := context.WithCancel(ctx)
	defer wcancel()
	go func() {
		close(listenReady)
		waited <- st.Wait(wctx)
	}()
	<-listenReady

	tx, err := p.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	recv := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	for _, e := range []any{
		engineapp.ProjectionPut{Name: "engine.item_state", Key: "ENT01:I-1", ItemID: "ENT01:I-1", Value: json.RawMessage(`{"basis_seq":1}`)},
		engineapp.ContributionsReplace{ItemID: "ENT01:I-1", Rows: []engineapp.Contribution{{ItemID: "ENT01:I-1", Metric: "defects", Slice: "wc", Value: 1}}},
		engineapp.Notify{Changes: []engineapp.Change{{Entity: platform.EntityItem, ID: "ENT01:I-1", Seq: 7, ReceivedAt: recv}}},
	} {
		if ok, err := storage.ApplyEffect(ctx, tx, e.(interface{ EffectKind() string })); !ok || err != nil {
			t.Fatalf("эффект %T: %v %v", e, ok, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	// Wait мог открыть LISTEN уже после фиксации: тогда сигнал пропущен, и
	// догоняем по seq — ровно так работает публикатор.
	select {
	case err := <-waited:
		if err != nil {
			t.Fatalf("Wait: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		// Ждущий один: снимаем первое ожидание, соединение LISTEN остаётся.
		wcancel()
		<-waited
	}
	changes, err := st.After(ctx, engineapp.ChangeQuery{Limit: 10})
	if err != nil || len(changes) != 1 || changes[0].Pos != 1 || changes[0].Seq != 7 || changes[0].Entity != platform.EntityItem || !changes[0].ReceivedAt.Equal(recv) {
		t.Fatalf("журнал изменений: %+v %v", changes, err)
	}
	v, ok, err := st.Get(ctx, "engine.item_state", "ENT01:I-1")
	if err != nil || !ok || !strings.Contains(string(v), `"basis_seq": 1`) && !strings.Contains(string(v), `"basis_seq":1`) {
		t.Fatalf("проекция: %s %v %v", v, ok, err)
	}

	// Сигнал после установленного LISTEN доходит.
	got := make(chan error, 1)
	go func() { got <- st.Wait(ctx) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := p.Exec(ctx, fmt.Sprintf("SELECT pg_notify('%s', '8')", storage.Channel)); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-got:
			if err != nil {
				t.Fatal(err)
			}
			return
		case <-time.After(100 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			t.Fatal("NOTIFY не дошёл")
		}
	}
}
