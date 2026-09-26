package mes

import (
	"encoding/json"
	"slices"
	"strconv"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// RuleHold — правило реакции «блок передаётся в MES» (AD-3): слот —
// (правило, `erp_message:‹бизнес-ключ›`, пусто).
const RuleHold = "mes.hold"

// Hold — блокировка субъекта в MES глазами журнала: изделие или партия,
// заблокирован ли, номер цикла блок → снятие и записи сдерживания, которые
// держат блок (снятие — когда их не осталось, AD-27).
type Hold struct {
	Subject string   `json:"subject"`
	Lot     bool     `json:"lot"`
	Held    bool     `json:"held"`
	Cycle   int      `json:"cycle"`
	Active  []string `json:"active,omitempty"`
}

// HoldDraft — запрос в MES до записи реакции.
type HoldDraft struct {
	Key   string
	Data  ev.MesHoldRequestedV1
	Cause kernel.Record
}

// Triggers — записи, по которым модуль mes решает о блоке в MES.
var Triggers = []catalog.Type{catalog.DecisionContainmentApplied, catalog.DecisionContainmentSet, catalog.DecisionContainmentReleased,
	catalog.DecisionItemIsolated, catalog.GenealogyContainmentPropagated, catalog.DecisionLotResolved}

// HoldSubject — субъект блока по записи-триггеру: партия (lot = true) или
// изделие; пусто — запись блок в MES не меняет.
func HoldSubject(r kernel.Record) (subject string, lot bool, err error) {
	switch r.Type {
	case catalog.DecisionContainmentApplied:
		var d ev.DecisionContainmentAppliedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return "", false, err
		}
		if d.Level == ev.AxisContainmentLotHold && d.LotID != nil {
			return string(*d.LotID), true, nil
		}
		return r.ItemID, false, nil
	case catalog.DecisionLotResolved:
		var d ev.DecisionLotResolvedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return "", false, err
		}
		return string(d.LotID), true, nil
	case catalog.DecisionContainmentSet, catalog.DecisionContainmentReleased, catalog.DecisionItemIsolated, catalog.GenealogyContainmentPropagated:
		return r.ItemID, false, nil
	}
	return "", false, nil
}

func blocking(l ev.AxisContainment) bool {
	return l == ev.AxisContainmentItemHold || l == ev.AxisContainmentLotHold
}

// PlanHold — что сообщить MES по записи-триггеру (чистая функция, AD-4):
//   - блок изделия или партии (сдерживание item_hold / lot_hold правилом или
//     человеком, изоляция изделия, распространение по генеалогии) — «заблокировать»,
//     если субъект ещё не заблокирован;
//   - снятие сдерживания человеком (все записи, державшие блок, сняты), понижение
//     уровня решением человека, решение по партии (кроме «не годна») —
//     «снять блок». Снятие основания правилом блок не снимает (AD-27).
func PlanHold(h Hold, r kernel.Record) (Hold, []HoldDraft, error) {
	block := func() (Hold, []HoldDraft, error) {
		if !slices.Contains(h.Active, r.EventID) {
			h.Active = append(h.Active, r.EventID)
		}
		if h.Held {
			return h, nil, nil
		}
		h.Held = true
		h.Cycle++
		return h, []HoldDraft{h.draft("hold", true, r)}, nil
	}
	release := func() (Hold, []HoldDraft, error) {
		h.Active = nil
		if !h.Held {
			return h, nil, nil
		}
		h.Held = false
		return h, []HoldDraft{h.draft("release", false, r)}, nil
	}
	switch r.Type {
	case catalog.DecisionContainmentApplied:
		var d ev.DecisionContainmentAppliedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return h, nil, err
		}
		if blocking(d.Level) {
			return block()
		}
	case catalog.DecisionContainmentSet:
		var d ev.DecisionContainmentSetV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return h, nil, err
		}
		if blocking(d.Level) {
			return block()
		}
		return release()
	case catalog.DecisionItemIsolated:
		return block()
	case catalog.GenealogyContainmentPropagated:
		var d ev.GenealogyContainmentPropagatedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return h, nil, err
		}
		if blocking(d.Level) && (d.Released == nil || !*d.Released) {
			return block()
		}
	case catalog.DecisionContainmentReleased:
		var d ev.DecisionContainmentReleasedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return h, nil, err
		}
		h.Active = slices.DeleteFunc(h.Active, func(id string) bool { return slices.Contains(d.ReleasedEventIds, ev.UUID(id)) })
		if len(h.Active) == 0 {
			return release()
		}
	case catalog.DecisionLotResolved:
		var d ev.DecisionLotResolvedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return h, nil, err
		}
		if d.Resolution != ev.DecisionLotResolvedV1ResolutionReject {
			return release()
		}
	}
	return h, nil, nil
}

func (h Hold) draft(action string, hold bool, cause kernel.Record) HoldDraft {
	key := h.Subject + "/" + action + "/" + strconv.Itoa(h.Cycle)
	d := ev.MesHoldRequestedV1{BusinessKey: key, Hold: hold}
	if h.Lot {
		x := ev.ObjectID(h.Subject)
		d.LotID = &x
	} else {
		x := ev.ItemID(h.Subject)
		d.ItemID = &x
	}
	return HoldDraft{Key: key, Data: d, Cause: cause}
}

// Stream — поток запроса в MES (каталог: поток erp_message, AD-39).
func Stream(key string) string { return "erp_message:" + key }

// HoldReaction — реакция mes.hold.requested (AD-3): слот (mes.hold,
// erp_message:‹ключ›), причина — запись-триггер; версия 1 (содержимое по
// ключу не меняется: другой блок — другой цикл).
func HoldReaction(d HoldDraft) (kernel.Reaction, error) {
	rx, err := kernel.NewReaction(Module, catalog.MesHoldRequested, kernel.Slot{RuleID: RuleHold, Subject: Stream(d.Key)}, d.Data, d.Cause)
	if err != nil {
		return rx, err
	}
	rx.AutomationMode = 2 // защитная реакция: автоматика изолирует, но не списывает (AD-27)
	return rx, nil
}

// MessageID — номер сообщения для MES (BODID): UUIDv5 от бизнес-ключа;
// повтор и переотправка — с тем же номером (AD-7).
func MessageID(key string) string {
	return kernel.UUIDv5(constants.NsAnt, "mes.hold\x1f"+key)
}

// ResponseID — id записи ответа MES (факт роли outbox) на запрос и попытку-поколение.
func ResponseID(requestEventID string, generation int) string {
	return kernel.UUIDv5(constants.NsAnt, "mes.hold.responded\x1f"+requestEventID+"\x1f"+strconv.Itoa(generation))
}
