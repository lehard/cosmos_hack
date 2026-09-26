package engine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	storage "ant/internal/infrastructure/storage/engine"
)

// pool — своя БД агента (make dev-db, .dev/db.env); без неё тест пропускается.
func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	host := os.Getenv("ANT_DB_HOST")
	if host == "" {
		t.Skip("нет ANT_DB_HOST (make dev-db) — интеграционный тест Postgres пропущен")
	}
	pw, err := os.ReadFile(os.Getenv("ANT_DB_PASSWORD_FILE"))
	if err != nil {
		t.Skip("нет пароля dev-db: ", err)
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword(os.Getenv("ANT_DB_USER"), strings.TrimSpace(string(pw))),
		Host: host + ":" + os.Getenv("ANT_DB_PORT"), Path: os.Getenv("ANT_DB_NAME"), RawQuery: "sslmode=disable"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	// Миграция схемы engine (часть Up из файла goose).
	up, err := storage.Migrations.ReadFile("migrations/20260926090000_engine.sql")
	if err != nil {
		t.Fatal(err)
	}
	body := strings.SplitN(strings.SplitN(string(up), "-- +goose Down", 2)[0], "-- +goose Up", 2)[1]
	if _, err := p.Exec(ctx, "DROP SCHEMA IF EXISTS engine CASCADE"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Exec(ctx, body); err != nil {
		t.Fatalf("миграция: %v", err)
	}
	return p
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
	go func() {
		close(listenReady)
		waited <- st.Wait(ctx)
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
	}
	changes, err := st.After(ctx, 0, 10)
	if err != nil || len(changes) != 1 || changes[0].Seq != 7 || changes[0].Entity != platform.EntityItem || !changes[0].ReceivedAt.Equal(recv) {
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
