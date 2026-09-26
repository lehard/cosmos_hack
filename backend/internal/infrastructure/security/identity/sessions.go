package identity

import (
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionTable — таблица сеансов scs в схеме модуля access (миграция —
// infrastructure/storage/access/migrations).
const SessionTable = "access.sessions"

// SessionStore — хранилище сеансов веба scs в Postgres (pgxstore, AD-15):
// сеансы общие для копий api и переживают перезапуск; истёкшие удаляются
// раз в 5 минут.
func SessionStore(pool *pgxpool.Pool) scs.Store {
	return pgxstore.NewWithConfig(pool, pgxstore.Config{TableName: SessionTable, CleanUpInterval: 5 * time.Minute})
}
