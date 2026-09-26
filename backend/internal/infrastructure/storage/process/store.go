package process

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	app "ant/internal/application/process"
	dp "ant/internal/domain/process"
)

// Versions — хранилище версий процесса в Postgres (схема process), ведомый
// порт application/process.VersionStore. Байты XML хранятся как загружены
// (bytea, без канонизации — хеш версии над ними, AD-17).
type Versions struct {
	Pool *pgxpool.Pool
}

var _ app.VersionStore = (*Versions)(nil)

const columns = `version_id, label, status, base_version_id, author, hash, xml, created_at, effective_from, genesis, signatures, route_closed_event_id, approval_document_id`

func scan(row pgx.Row) (app.VersionRecord, error) {
	var v app.VersionRecord
	var sig []byte
	var eff *time.Time
	err := row.Scan(&v.ID, &v.Label, &v.Status, &v.BaseVersionID, &v.Author, &v.Hash, &v.XML, &v.CreatedAt, &eff, &v.Genesis, &sig, &v.RouteClosedEventID, &v.ApprovalDocumentID)
	if err != nil {
		return v, err
	}
	v.CreatedAt = v.CreatedAt.UTC()
	if eff != nil {
		t := eff.UTC()
		v.EffectiveFrom = &t
	}
	if len(sig) > 0 {
		var ss []dp.Signature
		if err := json.Unmarshal(sig, &ss); err != nil {
			return v, err
		}
		v.Signatures = ss
	}
	return v, nil
}

func (s *Versions) one(ctx context.Context, where string, arg string) (app.VersionRecord, bool, error) {
	v, err := scan(s.Pool.QueryRow(ctx, `SELECT `+columns+` FROM process.versions WHERE `+where+` ORDER BY created_at, version_id LIMIT 1`, arg))
	if errors.Is(err, pgx.ErrNoRows) {
		return v, false, nil
	}
	return v, err == nil, err
}

// ByHash — версия по хешу, закреплённому за изделием.
func (s *Versions) ByHash(ctx context.Context, hash string) (app.VersionRecord, bool, error) {
	return s.one(ctx, "hash = $1", hash)
}

// ByID — версия по id.
func (s *Versions) ByID(ctx context.Context, id string) (app.VersionRecord, bool, error) {
	return s.one(ctx, "version_id = $1", id)
}

// List — все версии по времени создания.
func (s *Versions) List(ctx context.Context) ([]app.VersionRecord, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+columns+` FROM process.versions ORDER BY created_at, version_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []app.VersionRecord
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Save — новая версия; версия с тем же id не перезаписывается (версии неизменны).
func (s *Versions) Save(ctx context.Context, v app.VersionRecord) error {
	sig, err := json.Marshal(v.Signatures)
	if err != nil {
		return err
	}
	if v.Signatures == nil {
		sig = []byte("[]")
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO process.versions (`+columns+`)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) ON CONFLICT (version_id) DO NOTHING`,
		v.ID, v.Label, v.Status, v.BaseVersionID, v.Author, v.Hash, v.XML, v.CreatedAt, v.EffectiveFrom, v.Genesis, sig, v.RouteClosedEventID, v.ApprovalDocumentID)
	return err
}

// Update — жизненный цикл версии (FR-22): статус, вступление в силу, подписи
// кворума, закрытие маршрута, лист утверждения; xml и hash не меняются.
func (s *Versions) Update(ctx context.Context, v app.VersionRecord) error {
	sig, err := json.Marshal(v.Signatures)
	if err != nil {
		return err
	}
	if v.Signatures == nil {
		sig = []byte("[]")
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE process.versions SET status = $2, effective_from = $3, signatures = $4, route_closed_event_id = $5,
approval_document_id = $6 WHERE version_id = $1`, v.ID, v.Status, v.EffectiveFrom, sig, v.RouteClosedEventID, v.ApprovalDocumentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("process: нет версии " + v.ID)
	}
	return nil
}
