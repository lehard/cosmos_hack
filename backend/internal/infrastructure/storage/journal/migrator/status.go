package migrator

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Querier — подключение для чтения таблиц версий (пул или соединение pgx).
type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// SetStatus — состояние миграций модуля для самопроверки после старта
// (FR-109, эпик 34): версия схемы в базе, последняя версия сборки и
// неприменённые версии сборки.
type SetStatus struct {
	Module  string
	Current int64
	Target  int64
	Pending []int64
	// Err — состояние не прочитано (нет прав на ant_migrations и т. п.).
	Err error
}

// Status читает таблицы версий goose (ant_migrations.goose_‹модуль›) без
// блокировок и без изменений схемы и сравнивает их с миграциями сборки.
// Таблицы нет — все миграции модуля не применены. Подключение должно видеть
// схему ant_migrations (владелец — ant_owner; в демо — суперпользователь).
func Status(ctx context.Context, q Querier, sets ...Set) []SetStatus {
	out := make([]SetStatus, 0, len(sets))
	for _, s := range sets {
		st := SetStatus{Module: s.Module}
		st.Err = status(ctx, q, s, &st)
		out = append(out, st)
	}
	return out
}

func status(ctx context.Context, q Querier, s Set, st *SetStatus) error {
	if !reModule.MatchString(s.Module) {
		return fmt.Errorf("имя модуля %q", s.Module)
	}
	want, err := versions(s)
	if err != nil {
		return err
	}
	if len(want) > 0 {
		st.Target = want[len(want)-1]
	}
	table := VersionSchema + ".goose_" + s.Module
	var exists bool
	if err := q.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists); err != nil {
		return err
	}
	applied := map[int64]bool{}
	if exists {
		rows, err := q.Query(ctx, "SELECT version_id FROM "+pgx.Identifier{VersionSchema, "goose_" + s.Module}.Sanitize()+" WHERE is_applied AND version_id > 0")
		if err != nil {
			return err
		}
		vs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil {
			return err
		}
		for _, v := range vs {
			applied[v] = true
			st.Current = max(st.Current, v)
		}
	}
	for _, v := range want {
		if !applied[v] {
			st.Pending = append(st.Pending, v)
		}
	}
	return nil
}

// versions — версии миграций сборки (метки времени из имён *.sql), по возрастанию.
func versions(s Set) ([]int64, error) {
	fsys, err := fs.Sub(s.FS, s.Dir)
	if err != nil {
		return nil, err
	}
	ents, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var out []int64
	for _, e := range ents {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".sql") {
			continue
		}
		head, _, ok := strings.Cut(name, "_")
		if !ok {
			continue
		}
		v, err := strconv.ParseInt(head, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	slices.Sort(out)
	return out, nil
}
