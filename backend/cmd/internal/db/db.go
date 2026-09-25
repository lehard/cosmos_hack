// Пакет db — пул подключений PostgreSQL (pgx) для точек входа ant.
//
// Слой: сборка зависимостей (cmd/*). Модули получают пул через свои адаптеры
// зоны storage; сам пакет ничего о модулях не знает.
// Связи: config.DB → *pgxpool.Pool.
//
// Пароль читается только из файла (AD-25: секретов в ANT_* нет) и не
// попадает в логи.
package db

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ant/cmd/internal/config"
)

// Open создаёт пул; соединения устанавливаются лениво, при первом запросе.
func Open(ctx context.Context, c config.DB, appName string) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, err
	}
	cc := pc.ConnConfig
	cc.Host = c.Host
	cc.Port = uint16(c.Port)
	cc.Database = c.Name
	cc.User = c.User
	if c.ConnectTimeout > 0 {
		cc.ConnectTimeout = c.ConnectTimeout
	}
	cc.RuntimeParams["application_name"] = appName
	if c.PasswordFile != "" {
		raw, err := os.ReadFile(c.PasswordFile)
		if err != nil {
			return nil, fmt.Errorf("пароль БД: %w", err)
		}
		cc.Password = strings.TrimSpace(string(raw))
	}
	switch c.SSLMode {
	case "", "disable":
		cc.TLSConfig = nil
		cc.Fallbacks = nil
	default:
		return nil, fmt.Errorf("db.sslmode=%q пока не поддержан (TLS к БД — эпик эксплуатации)", c.SSLMode)
	}
	if c.MaxConns > 0 {
		pc.MaxConns = int32(c.MaxConns)
	}
	return pgxpool.NewWithConfig(ctx, pc)
}

// WaitReady ждёт, пока БД ответит на ping, но не дольше timeout.
func WaitReady(ctx context.Context, p *pgxpool.Pool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var last error
	for {
		pctx, pcancel := context.WithTimeout(ctx, 2*time.Second)
		last = p.Ping(pctx)
		pcancel()
		if last == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("БД недоступна: %w", last)
		case <-time.After(500 * time.Millisecond):
		}
	}
}
