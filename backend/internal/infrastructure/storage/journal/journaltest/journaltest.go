// Пакет journaltest — опоры интеграционных тестов журнала и его потребителей
// на своей БД агента (make dev-db): отдельная база на тест с ролями и
// миграциями, как после ant migrate, пулы ролей, тестовые записи.
// Используется только из тестов.
package journaltest

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	app "ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	store "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/migrator"
)

// DB — тестовая база: создаётся NewDB, удаляется в конце теста.
// Без ANT_DB_HOST тесты пропускаются.
type DB struct {
	// Admin — суперпользователь (роль migrate).
	Admin *pgxpool.Config
	Name  string
}

func adminConfig(t *testing.T, db string) *pgxpool.Config {
	t.Helper()
	host := os.Getenv("ANT_DB_HOST")
	if host == "" {
		t.Skip("нет ANT_DB_HOST — интеграционные тесты журнала пропущены (make dev-db)")
	}
	pw := ""
	if f := os.Getenv("ANT_DB_PASSWORD_FILE"); f != "" {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		pw = strings.TrimSpace(string(raw))
	}
	cfg, err := pgxpool.ParseConfig(fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable",
		host, cmpOr(os.Getenv("ANT_DB_PORT"), "5432"), os.Getenv("ANT_DB_USER"), db))
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Password = pw
	cfg.MaxConns = 8
	return cfg
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

var dbCounter atomic.Int64

// NewDB — чистая база с ролями и миграциями журнала.
func NewDB(t *testing.T) *DB {
	t.Helper()
	ctx := context.Background()
	base := adminConfig(t, os.Getenv("ANT_DB_NAME"))
	var rnd [4]byte
	_, _ = rand.Read(rnd[:])
	name := fmt.Sprintf("%s_t%d_%s", os.Getenv("ANT_DB_NAME"), dbCounter.Add(1), hex.EncodeToString(rnd[:]))
	conn, err := pgx.ConnectConfig(ctx, base.ConnConfig)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	_ = conn.Close(ctx)
	t.Cleanup(func() {
		c, err := pgx.ConnectConfig(context.Background(), base.ConnConfig)
		if err == nil {
			_, _ = c.Exec(context.Background(), "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
			_ = c.Close(context.Background())
		}
	})
	admin := adminConfig(t, name)
	ac, err := pgx.ConnectConfig(ctx, admin.ConnConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ac.Close(ctx) }()
	if err := migrator.EnsureRoles(ctx, ac); err != nil {
		t.Fatal(err)
	}
	applied, err := migrator.Up(ctx, admin.ConnConfig, nil, migrator.Set{Module: "journal", FS: store.Migrations, Dir: store.MigrationsDir})
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) == 0 {
		t.Fatal("миграции журнала не применились")
	}
	return &DB{Admin: admin, Name: name}
}

// AppPool — пул роли приложения (SET ROLE ant_app), как у роли ant.
func (d *DB) AppPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	p, err := store.NewAppPool(context.Background(), d.Admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func (d *DB) AdminConn(t *testing.T) *pgx.Conn {
	t.Helper()
	c, err := pgx.ConnectConfig(context.Background(), d.Admin.ConnConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close(context.Background()) })
	return c
}

// SysClock — системные часы InfraClock.
type SysClock struct{}

// Now — реальное время.
func (SysClock) Now() time.Time { return time.Now().UTC() }

var eventN atomic.Int64

// uuid7ish — уникальный UUID вида v7 для тестовых записей.
func uuid7ish() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// Envelope — конверт DSSE с payload события и data.
func Envelope(eventType string, data any) []byte {
	d, _ := json.Marshal(data)
	payload, _ := json.Marshal(map[string]any{"event_type": eventType, "data": json.RawMessage(d), "n": eventN.Add(1)})
	env, _ := json.Marshal(map[string]any{
		"payloadType": "application/vnd.ant.event+json; v=1",
		"payload":     base64.StdEncoding.EncodeToString(payload),
		"signatures":  []map[string]string{{"keyid": "test@1", "sig": "AA=="}},
	})
	return env
}

// Entry — запись типа из каталога в потоке stream (item — изделие или пусто).
func Entry(eventType string, kind jc.JournalEntryEntryKind, stream string, item string, occurred time.Time) app.Pending {
	e := jc.JournalEntry{
		EntryKind: kind, EventType: eventType, SchemaVersion: 1, EventID: uuid7ish(),
		SourceID: "test-source", Stream: stream, Partition: 0,
		OccurredAt: dj.FormatTime(occurred), ReceivedAt: dj.FormatTime(occurred),
		CorrelationID: "corr-1", ProvenanceClass: jc.JournalEntryProvenanceClassDevice,
		DomainBuild: dj.ZeroLink.String(),
	}
	if item != "" {
		it := item
		e.ItemID = &it
		e.Partition = dj.Partition(item, 4)
	}
	return app.Pending{Entry: e, Envelope: Envelope(eventType, map[string]any{"v": 1})}
}

// Fact — факт контроля изделия (inspection.result.recorded, триггер свёртки).
func Fact(item string) app.Pending {
	return Entry("inspection.result.recorded", jc.JournalEntryEntryKindFact, dj.ItemStream(item), item, time.Now())
}

// ReadAll — вся цепочка по порядку.
func ReadAll(t *testing.T, s *store.Store, chain string) []jc.JournalEntry {
	t.Helper()
	var out []jc.JournalEntry
	var after int64
	for {
		part, err := s.Read(context.Background(), app.ReadQuery{Chain: chain, AfterSeq: after, Limit: 500})
		if err != nil {
			t.Fatal(err)
		}
		if len(part) == 0 {
			return out
		}
		out = append(out, part...)
		after = int64(part[len(part)-1].Seq)
	}
}

// MigrateAgain — повтор роли migrate на той же базе.
func MigrateAgain(d *DB) ([]migrator.Applied, error) {
	ctx := context.Background()
	c, err := pgx.ConnectConfig(ctx, d.Admin.ConnConfig)
	if err != nil {
		return nil, err
	}
	defer func() { _ = c.Close(ctx) }()
	if err := migrator.EnsureRoles(ctx, c); err != nil {
		return nil, err
	}
	return migrator.Up(ctx, d.Admin.ConnConfig, nil, migrator.Set{Module: "journal", FS: store.Migrations, Dir: store.MigrationsDir})
}
