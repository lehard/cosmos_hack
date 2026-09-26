package erp

import (
	"encoding/json"
	"strconv"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// RulePosting — правило реакции «учётное сообщение сформировано» (AD-3):
// слот — (правило, `erp_message:‹бизнес-ключ›`, пусто); версия слота —
// версия содержимого сообщения.
const RulePosting = "erp.posting"

// Sent — последняя сформированная версия сообщения по бизнес-ключу: её id и
// отпечаток содержимого (AD-7).
type Sent struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
	EventID string `json:"event_id"`
	Content string `json:"content"`
}

// Content — отпечаток учётного содержимого сообщения: действие, субъект,
// склады, разрешение, признаки и основание претензии. Шаг процесса и
// записи-основания в отпечаток не входят: новая версия реакции процесса с тем
// же учётным содержимым ничего не даёт (AD-7).
func Content(d ev.ErpPostingRequestedV1) string {
	c := d
	c.BusinessKey, c.MessageVersion, c.MessageID, c.BasisEventIds, c.StepKey = "", 0, nil, nil, nil
	b, _ := json.Marshal(c)
	return string(b)
}

// Version — что делать с черновиком при прежней версии prev (nil — сообщения
// ещё не было): новая версия (emit) или ничего. Новая версия с другим
// содержимым после отправки уходит наружу только по решению человека —
// это решает очередь исходящих (роль outbox), а не эта функция.
func Version(prev *Sent, d Draft) (Sent, bool) {
	c := Content(d.Data)
	if prev == nil {
		return Sent{Key: d.Key, Version: 1, Content: c}, true
	}
	if prev.Content == c {
		return *prev, false
	}
	return Sent{Key: d.Key, Version: prev.Version + 1, Content: c}, true
}

// Reaction — реакция «учётное сообщение сформировано» версии s.Version
// (AD-3): слот (erp.posting, erp_message:‹ключ›), причина — запись-триггер;
// data — черновик с бизнес-ключом, версией и номером сообщения.
func Reaction(d Draft, s Sent) (kernel.Reaction, error) {
	data := d.Data
	data.BusinessKey, data.MessageVersion = d.Key, s.Version
	mid := ev.UUID(MessageID(d.Key, s.Version))
	data.MessageID = &mid
	rx, err := kernel.NewReaction(Module, catalog.ErpPostingRequested,
		kernel.Slot{RuleID: RulePosting, Subject: Stream(d.Key)}, data, d.Cause)
	if err != nil {
		return rx, err
	}
	rx.AutomationMode = 1
	return rx, nil
}

// ResponseID — id записи ответа учётной системы (факт роли outbox): UUIDv5
// от запроса и поколения отправки (0 — автоматическая, n — n-я ручная
// переотправка). Повторная запись того же ответа невозможна (AD-7).
func ResponseID(requestEventID string, generation int) string {
	return kernel.UUIDv5(constants.NsAnt, "erp.posting.responded\x1f"+requestEventID+"\x1f"+strconv.Itoa(generation))
}

// QuarantineID — id записи «сообщение в карантине» того же поколения отправки.
func QuarantineID(requestEventID string, generation int) string {
	return kernel.UUIDv5(constants.NsAnt, "erp.posting.quarantined\x1f"+requestEventID+"\x1f"+strconv.Itoa(generation))
}
