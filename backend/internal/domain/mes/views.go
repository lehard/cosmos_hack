package mes

import (
	"encoding/json"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Block — блокировка в MES по бизнес-ключу глазами журнала (проекция mes.block).
type Block struct {
	Key            string    `json:"business_key"`
	ItemID         string    `json:"item_id,omitempty"`
	LotID          string    `json:"lot_id,omitempty"`
	Hold           bool      `json:"hold"`
	RequestEventID string    `json:"request_event_id"`
	RequestedAt    time.Time `json:"requested_at"`
	// Outcome — квитанция MES (пусто — ждём), ErrorCode — код ошибки.
	Outcome     string     `json:"outcome,omitempty"`
	ErrorCode   string     `json:"error_code,omitempty"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
	// Attempts — сколько ответов записано (поколение отправки).
	Attempts int   `json:"attempts"`
	Seq      int64 `json:"seq"`
}

// Pending — блок ещё не подтверждён MES (нет квитанции или была ошибка данных).
func (b Block) Pending() bool { return b.Outcome == "" }

// BlockKey — бизнес-ключ записи обмена с MES (пусто — не запись обмена).
func BlockKey(r kernel.Record) string {
	if r.Type != catalog.MesHoldRequested && r.Type != catalog.MesHoldResponded {
		return ""
	}
	var d struct {
		BusinessKey string `json:"business_key"`
	}
	if json.Unmarshal(r.Data, &d) != nil {
		return ""
	}
	return d.BusinessKey
}

// Apply — шаг проекции блока (чистая функция, AD-45).
func (b Block) Apply(r kernel.Record) (Block, error) {
	b.Seq = r.Seq
	switch r.Type {
	case catalog.MesHoldRequested:
		var d ev.MesHoldRequestedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return b, err
		}
		b.Key, b.Hold, b.RequestEventID, b.RequestedAt = d.BusinessKey, d.Hold, r.EventID, r.OccurredAt
		if d.ItemID != nil {
			b.ItemID = string(*d.ItemID)
		}
		if d.LotID != nil {
			b.LotID = string(*d.LotID)
		}
	case catalog.MesHoldResponded:
		var d ev.MesHoldRespondedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return b, err
		}
		b.Attempts++
		if string(d.RequestEventID) != b.RequestEventID {
			return b, nil
		}
		at := r.OccurredAt
		b.RespondedAt = &at
		b.Outcome, b.ErrorCode = string(d.Outcome), ""
		if d.ErrorCode != nil {
			b.ErrorCode = *d.ErrorCode
		}
	}
	return b, nil
}

// JobView — задание MES (проекция mes.job).
type JobView struct {
	Data       ev.MesJobReceivedV1 `json:"data"`
	ReceivedAt time.Time           `json:"received_at"`
	Seq        int64               `json:"seq"`
}
