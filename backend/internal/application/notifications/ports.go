package notifications

import (
	"context"
	"encoding/json"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	dom "ant/internal/domain/notifications"
)

// Ведомые порты модуля notifications.

// Projections — чтение глобальных проекций модуля (AD-45: пишет только
// notifications эффектами в транзакции Append; читают операции API и
// планировщик). Адаптер — infrastructure/storage/notifications.
type Projections interface {
	// Get — значение проекции name по ключу.
	Get(ctx context.Context, name, key string) (json.RawMessage, bool, error)
	// All — все строки проекции name.
	All(ctx context.Context, name string) (map[string]json.RawMessage, error)
	// OpenObligations — действующие сроки проекции сроков (планировщик, AD-4).
	OpenObligations(ctx context.Context) ([]dom.ObligationRecord, error)
}

// Decision — решение человека к записи (AD-2, AD-39): тип модуля
// notifications, поток объекта, данные и метаданные команды.
type Decision struct {
	Type   catalog.Type
	Stream string
	// ItemID — изделие, если решение в потоке изделия.
	ItemID string
	RunID  string
	Data   any
	Meta   platform.CommandMeta
	// Actor — псевдоним автора (Principal.PersonID).
	Actor string
	// OccurredAt — доменное «сейчас» при приёме команды (AD-37).
	OccurredAt time.Time
	// GuardStreams — потоки, которые проверил гард (AD-39).
	GuardStreams []string
}

// DecisionWriter — ведомый порт записи решений (AD-44: в журнал пишет только
// journal.Append): одна команда — одна пачка с проверками AD-39.
type DecisionWriter interface {
	Write(ctx context.Context, d Decision) (platform.Receipt, error)
}
