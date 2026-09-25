// Пакет logging — структурные JSON-логи (slog) для всех процессов ant.
//
// Слой: infrastructure/observability — технический механизм, доступный всем
// зонам инфраструктуры и точкам входа (AD-1).
// Связи: создаётся в cmd/*, передаётся адаптерам как *slog.Logger.
//
// Соглашение (спайн, «Логи»): JSON; поля event_id, correlation_id, item_id,
// run_id, module; без содержимого записей журнала и без секретов.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

// Имена стандартных полей логов (соглашения спайна).
const (
	KeyService       = "service"
	KeyRole          = "role"
	KeyModule        = "module"
	KeyEventID       = "event_id"
	KeyCorrelationID = "correlation_id"
	KeyItemID        = "item_id"
	KeyRunID         = "run_id"
)

// New возвращает JSON-логгер с уровнем level (debug | info | warn | error).
func New(w io.Writer, level string, service string) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: ParseLevel(level)})
	return slog.New(h).With(KeyService, service)
}

// ParseLevel разбирает уровень; неизвестное значение даёт info.
func ParseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
