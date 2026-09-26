package stand

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations — миграции goose схемы stand_onec (состояние stand-а 1С, AD-18):
// роль migrate применяет их вместе с миграциями модулей (cmd/ant/migrate.go).
//
//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir — каталог миграций внутри Migrations.
const MigrationsDir = "migrations"

// Postgres — состояние stand-а в схеме stand_onec: одна таблица объектов
// (сообщение, документ, этап) — вид, id, тело JSON.
type Postgres struct {
	Pool *pgxpool.Pool
}

// Load — всё состояние (stand-у его хватает в памяти: сотни сообщений).
func (p Postgres) Load(ctx context.Context) (Snapshot, error) {
	var s Snapshot
	rows, err := p.Pool.Query(ctx, `SELECT kind, body FROM stand_onec.objects ORDER BY created, id`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var body []byte
		if err := rows.Scan(&kind, &body); err != nil {
			return s, err
		}
		switch kind {
		case "message":
			var m Message
			if err := json.Unmarshal(body, &m); err != nil {
				return s, fmt.Errorf("stand_onec: сообщение: %w", err)
			}
			s.Messages = append(s.Messages, m)
		case "document":
			var d Document
			if err := json.Unmarshal(body, &d); err != nil {
				return s, fmt.Errorf("stand_onec: документ: %w", err)
			}
			s.Documents = append(s.Documents, d)
		case "stage":
			var r Row
			if err := json.Unmarshal(body, &r); err != nil {
				return s, fmt.Errorf("stand_onec: этап: %w", err)
			}
			s.Stages = append(s.Stages, r)
		}
	}
	return s, rows.Err()
}

func (p Postgres) save(ctx context.Context, kind, id string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = p.Pool.Exec(ctx, `INSERT INTO stand_onec.objects (kind, id, body, created, updated) VALUES ($1, $2, $3, $4, $4)
ON CONFLICT (kind, id) DO UPDATE SET body = EXCLUDED.body, updated = EXCLUDED.updated`, kind, id, b, time.Now().UTC())
	return err
}

// SaveMessage — сохранить сообщение.
func (p Postgres) SaveMessage(ctx context.Context, m Message) error {
	return p.save(ctx, "message", m.MessageID, m)
}

// SaveDocument — сохранить документ.
func (p Postgres) SaveDocument(ctx context.Context, d Document) error {
	return p.save(ctx, "document", d.RefKey, d)
}

// SaveStage — сохранить этап производства.
func (p Postgres) SaveStage(ctx context.Context, id string, r Row) error {
	return p.save(ctx, "stage", id, r)
}
