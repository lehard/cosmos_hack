package access

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	app "ant/internal/application/access"
)

// Credentials — учётные данные входа в схеме access (порт
// application/access.CredentialStore): хеш пароля argon2id, состояние,
// счётчик неудач и блокировка (FR-128, AD-15).
type Credentials struct {
	pool *pgxpool.Pool
}

// NewCredentials — хранилище на пуле роли приложения (ant_app).
func NewCredentials(pool *pgxpool.Pool) *Credentials { return &Credentials{pool: pool} }

var _ app.CredentialStore = (*Credentials)(nil)

const credCols = `login, person_id, hash, status, display_name, failures, locked_until, created_at`

func scanCred(row pgx.Row) (app.Credential, error) {
	var c app.Credential
	var locked *time.Time
	if err := row.Scan(&c.Login, &c.PersonID, &c.Hash, &c.Status, &c.DisplayName, &c.Failures, &locked, &c.CreatedAt); err != nil {
		return app.Credential{}, err
	}
	if locked != nil {
		c.LockedUntil = locked.UTC()
	}
	c.CreatedAt = c.CreatedAt.UTC()
	return c, nil
}

// Get — учётные данные по логину.
func (s *Credentials) Get(ctx context.Context, login string) (app.Credential, bool, error) {
	c, err := scanCred(s.pool.QueryRow(ctx, `SELECT `+credCols+` FROM access.credentials WHERE login = $1`, login))
	if errors.Is(err, pgx.ErrNoRows) {
		return app.Credential{}, false, nil
	}
	return c, err == nil, err
}

// Create — новая заявка или учётная запись; логин занят — ErrLoginTaken.
func (s *Credentials) Create(ctx context.Context, c app.Credential) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO access.credentials (login, person_id, hash, status, display_name) VALUES ($1, $2, $3, $4, $5)`,
		c.Login, c.PersonID, c.Hash, c.Status, c.DisplayName)
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		return app.ErrLoginTaken
	}
	return err
}

// Activate — учётная запись действует и привязана к сотруднику.
func (s *Credentials) Activate(ctx context.Context, login, personID string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE access.credentials SET status = 'active', person_id = $2, failures = 0, locked_until = NULL, updated_at = now() WHERE login = $1`,
		login, personID)
	if err == nil && tag.RowsAffected() == 0 {
		return errors.New("access: нет учётных данных " + login)
	}
	return err
}

// Failed — неудачная попытка: счётчик +1; достиг max — блокировка до now+lockFor
// и счётчик с нуля (после блокировки — снова max попыток).
func (s *Credentials) Failed(ctx context.Context, login string, now time.Time, max int, lockFor time.Duration) (app.Credential, error) {
	return scanCred(s.pool.QueryRow(ctx, `
UPDATE access.credentials SET
    failures     = CASE WHEN failures + 1 >= $2 THEN 0 ELSE failures + 1 END,
    locked_until = CASE WHEN failures + 1 >= $2 THEN $3::timestamptz ELSE locked_until END,
    updated_at   = now()
WHERE login = $1
RETURNING `+credCols, login, max, now.Add(lockFor)))
}

// Succeeded — сброс счётчика неудач.
func (s *Credentials) Succeeded(ctx context.Context, login string) error {
	_, err := s.pool.Exec(ctx, `UPDATE access.credentials SET failures = 0, locked_until = NULL, updated_at = now() WHERE login = $1 AND (failures <> 0 OR locked_until IS NOT NULL)`, login)
	return err
}

// List — все учётные данные по логину.
func (s *Credentials) List(ctx context.Context) ([]app.Credential, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+credCols+` FROM access.credentials ORDER BY login`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []app.Credential
	for rows.Next() {
		c, err := scanCred(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
