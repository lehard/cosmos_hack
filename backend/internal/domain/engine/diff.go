package engine

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
	"ant/internal/domain/notifications"
)

// Recorded — версия слота реакции, уже записанная в журнал (AD-3). Воркер
// восстанавливает её из записи журнала и конверта; свёртка её не читает —
// только сравнение (AD-5).
type Recorded struct {
	// EventID — reaction_id этой версии = UUIDv5(NS_ANT, слот ‖ версия).
	EventID string
	// Seq — позиция записи в журнале.
	Seq            int64
	Module         kernel.Module
	Type           catalog.Type
	Slot           kernel.Slot
	Version        int
	RuleRev        string
	AutomationMode int
	Causes         []string
	OccurredAt     time.Time
	// Data — канонический JSON поля data.
	Data json.RawMessage
}

// Change — что воркер делает со слотом при пересвёртке (AD-3, AD-5).
type Change string

// Виды изменений слота.
const (
	// ChangeNew — первая версия слота.
	ChangeNew Change = "new"
	// ChangeRevised — следующая версия слота: вывод изменился
	// («пересмотрен из-за записи ‹id›», FR-32) или вернулся после отзыва.
	ChangeRevised Change = "revised"
	// ChangeWithdrawn — реакция исчезла; задача, уведомление или срок
	// снимаются следующей версией слота с типом отзыва.
	ChangeWithdrawn Change = "withdrawn"
)

// Planned — версия слота, которую воркер допишет в журнал одной пачкой с
// basis_seq (AD-3, AD-5). Прежние записи не трогаются.
type Planned struct {
	Reaction kernel.Reaction
	Change   Change
	// Version — версия слота: 1 — первая, пересмотр и отзыв — следующая.
	Version int
	// EventID — reaction_id = UUIDv5(NS_ANT, слот ‖ версия).
	EventID string
	// Supersedes — event_id заменяемой версии; пусто у первой.
	Supersedes string
	// RevisedDueTo — запись, из-за которой вывод пересмотрен (FR-32); пусто у первой версии.
	RevisedDueTo string
}

// Trigger — запись, на которую идёт пересвёртка (AD-5): новая запись потока
// изделия. Её event_id — «пересмотрен из-за записи ‹id›».
type Trigger struct {
	EventID    string
	OccurredAt time.Time
}

// Правила движка — слоты задач, которые ставит движок по AD-3 и AD-5. Эмитент
// типа задачи — notifications (AD-40): движок строит запись функцией
// notifications-типа, своих типов записей не имеет.
const (
	// RuleProtectionBasisChanged — «основание защиты изменилось — пересмотрите» (AD-3).
	RuleProtectionBasisChanged = "engine.protection_basis_changed"
	// RuleDecisionBeforeNewData — «решение принято до новых данных — пересмотрите» (AD-5, FR-32).
	RuleDecisionBeforeNewData = "engine.decision_before_new_data"
	// ReviewerRole — роль, которой ставятся задачи пересмотра защиты:
	// начальник ОТК — уполномоченный на снятие сдерживания (AD-11, AD-27).
	ReviewerRole = "head_of_qc"
)

// withdrawals — типы реакций, которые снимаются при исчезновении, и тип
// отзыва (AD-3): задача → «задача снята», уведомление → «уведомление
// отозвано», срок → «срок снят». Защитные реакции не снимаются никогда.
var withdrawals = map[catalog.Type]catalog.Type{
	catalog.TaskTaskCreated:      catalog.TaskTaskWithdrawn,
	catalog.TaskNotificationSent: catalog.TaskNotificationWithdrawn,
	catalog.ObligationDueSet:     catalog.ObligationDueCleared,
}

// isWithdrawal — тип записи-отзыва: слот, чья последняя версия — отзыв,
// считается пустым.
func isWithdrawal(t catalog.Type) bool {
	switch t {
	case catalog.TaskTaskWithdrawn, catalog.TaskNotificationWithdrawn, catalog.ObligationDueCleared:
		return true
	}
	return false
}

// Protective — реакция защитная (блок, изоляция, расширение области риска):
// при исчезновении не снимается (AD-3, AD-27).
func Protective(t catalog.Type) bool {
	info, ok := catalog.Lookup(t)
	return ok && info.ActionClass == catalog.ClassProtective
}

// Latest — последняя версия каждого слота из записанных версий (AD-3).
func Latest(recorded []Recorded) map[string]Recorded {
	out := make(map[string]Recorded, len(recorded))
	for _, r := range recorded {
		k := r.Slot.Key()
		if prev, ok := out[k]; !ok || r.Version > prev.Version {
			out[k] = r
		}
	}
	return out
}

// Diff сравнивает реакции, вычисленные пересвёрткой, с записанными
// версиями слотов (AD-3, AD-5) и возвращает, что дописать:
//   - новый слот — версия 1;
//   - изменённый вывод — следующая версия с supersedes и «пересмотрен из-за
//     записи ‹trigger›»;
//   - исчезнувшие задача, уведомление, срок — следующая версия с типом отзыва;
//   - исчезнувшая защитная реакция не снимается: движок ставит задачу
//     «основание защиты изменилось — пересмотрите» (исчезнет сама, если
//     защита вернётся);
//   - прочие исчезнувшие выводы остаются в журнале как есть.
//
// Результат упорядочен по ключу слота; функция чистая (AD-4).
func Diff(computed []kernel.Reaction, recorded []Recorded, trigger Trigger) ([]Planned, error) {
	latest := Latest(recorded)
	want := make(map[string]kernel.Reaction, len(computed))
	for _, r := range computed {
		k := r.Slot.Key()
		if prev, dup := want[k]; dup && prev.Type != r.Type {
			return nil, fmt.Errorf("engine: слот %q вычислен дважды с разными типами (%s, %s) — reaction_id совпадёт (AD-3)", k, prev.Type, r.Type)
		}
		want[k] = r
	}
	// Защиты, у которых исчезло основание (AD-3).
	for _, k := range slices.Sorted(maps.Keys(latest)) {
		l := latest[k]
		if _, still := want[k]; still || isWithdrawal(l.Type) || !Protective(l.Type) {
			continue
		}
		task, err := protectionBasisTask(l)
		if err != nil {
			return nil, err
		}
		if _, taken := want[task.Slot.Key()]; !taken {
			want[task.Slot.Key()] = task
		}
	}

	keys := slices.Concat(slices.Sorted(maps.Keys(want)), slices.Sorted(maps.Keys(latest)))
	slices.Sort(keys)
	keys = slices.Compact(keys)

	var plan []Planned
	for _, k := range keys {
		c, isComputed := want[k]
		l, isRecorded := latest[k]
		if isComputed && isRecorded && isWithdrawal(l.Type) && c.Type == l.Type {
			// Вывод правила сам имеет тип отзыва (срок снят исполнением —
			// obligation.due.cleared): тот же вывод — не пересмотр. Иначе
			// каждая пересвёртка дописывала бы новую версию, а верификатор
			// видел бы её «следующей из журнала, но не записанной» (AD-3, AD-9).
			fc, err := Fingerprint(c)
			if err != nil {
				return nil, err
			}
			fl, err := l.Fingerprint()
			if err != nil {
				return nil, err
			}
			if fc == fl {
				continue
			}
		}
		switch {
		case isComputed && (!isRecorded || isWithdrawal(l.Type)):
			p := Planned{Reaction: c, Change: ChangeNew, Version: 1}
			if isRecorded {
				p.Change, p.Version, p.Supersedes, p.RevisedDueTo = ChangeRevised, l.Version+1, l.EventID, trigger.EventID
			}
			p.EventID = c.ID(p.Version)
			plan = append(plan, p)
		case isComputed:
			fc, err := Fingerprint(c)
			if err != nil {
				return nil, err
			}
			fl, err := l.Fingerprint()
			if err != nil {
				return nil, err
			}
			if fc == fl {
				continue
			}
			plan = append(plan, Planned{Reaction: c, Change: ChangeRevised, Version: l.Version + 1,
				EventID: c.ID(l.Version + 1), Supersedes: l.EventID, RevisedDueTo: trigger.EventID})
		case isRecorded && !isWithdrawal(l.Type):
			wt, ok := withdrawals[l.Type]
			if !ok {
				// Защитная — не снимается (задача пересмотра уже в want);
				// прочий вывод остаётся в журнале, пока правило не вычислит новый.
				continue
			}
			w, err := withdrawal(l, wt, trigger)
			if err != nil {
				return nil, err
			}
			plan = append(plan, Planned{Reaction: w, Change: ChangeWithdrawn, Version: l.Version + 1,
				EventID: w.ID(l.Version + 1), Supersedes: l.EventID, RevisedDueTo: trigger.EventID})
		}
	}
	return plan, nil
}

// withdrawal — реакция-отзыв для исчезнувшей задачи, уведомления или срока (AD-3).
func withdrawal(l Recorded, wt catalog.Type, trigger Trigger) (kernel.Reaction, error) {
	var ids struct {
		TaskID         string `json:"task_id"`
		NotificationID string `json:"notification_id"`
		ObligationID   string `json:"obligation_id"`
	}
	if len(l.Data) > 0 {
		if err := json.Unmarshal(l.Data, &ids); err != nil {
			return kernel.Reaction{}, fmt.Errorf("engine: данные слота %s: %w", l.EventID, err)
		}
	}
	var data any
	switch wt {
	case catalog.TaskTaskWithdrawn:
		data = ev.TaskTaskWithdrawnV1{TaskID: ev.ObjectID(ids.TaskID),
			Reason: &ev.Reason{Text: "Основание задачи исчезло при пересвёртке изделия"}}
	case catalog.TaskNotificationWithdrawn:
		data = ev.TaskNotificationWithdrawnV1{NotificationID: ev.ObjectID(ids.NotificationID)}
	case catalog.ObligationDueCleared:
		data = ev.ObligationDueClearedV1{ObligationID: ev.ObjectID(ids.ObligationID), Cause: ev.ObligationDueClearedV1CauseWithdrawn}
	}
	cause := kernel.Record{EventID: trigger.EventID, OccurredAt: trigger.OccurredAt}
	r, err := kernel.NewReaction(l.Module, wt, l.Slot, data, cause)
	if err != nil {
		return kernel.Reaction{}, err
	}
	r.RuleRev, r.AutomationMode = l.RuleRev, l.AutomationMode
	return r, nil
}

// protectionBasisTask — задача «основание защиты изменилось — пересмотрите»
// уполномоченному (AD-3). Слот — от слота защиты, причина — записанная
// защита: пока основание не вернулось, задача вычисляется одинаково и не
// пересматривается.
func protectionBasisTask(l Recorded) (kernel.Reaction, error) {
	slot := kernel.Slot{RuleID: RuleProtectionBasisChanged, Subject: l.Slot.Subject, TriggerKey: l.Slot.RuleID + "/" + l.Slot.TriggerKey}
	info, _ := catalog.Lookup(l.Type)
	return notifications.ReviewTask(slot, ev.TaskTaskCreatedV1KindProtectionBasisChanged,
		"Основание защиты изменилось — пересмотрите: "+info.Title, ReviewerRole, "",
		kernel.Record{EventID: l.EventID, OccurredAt: l.OccurredAt})
}
