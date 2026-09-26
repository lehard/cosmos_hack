package access

import (
	"context"
	"errors"
	"time"
	"uuid"

	app "ant/internal/application/access"
	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
	"ant/internal/domain/kernel"
)

// PresenceWriter — порт application/access.PresenceWriter над журналом (эпик
// 37): записи собирает кодек движка (происхождение server_attested — сервер
// заверяет данные адаптера СКУД и свои выводы). Идентификаторы однозначны от
// основания (UUIDv5): повтор опроса СКУД и повтор проверки присутствия не
// пишут запись дважды — журнал отвечает journal.duplicate, это не ошибка.
//
// security.presence.deviation и security.admission.denied — типы модуля
// security (AD-40): этот адаптер пишет их от его имени, как SecurityBus.
type PresenceWriter struct {
	Journal appjournal.JournalStore
	Codec   *engineapp.Codec
}

var _ app.PresenceWriter = PresenceWriter{}

// RulePresence — правило реакций присутствия (снятие допуска автоматически).
const RulePresence = "access.presence"

// ZonePassed — access.zone.passed в поток сотрудника (FR-82).
func (w PresenceWriter) ZonePassed(ctx context.Context, p app.ZonePassIn) (bool, error) {
	d := ev.AccessZonePassedV1{PersonID: ev.PersonRef(p.PersonID), ZoneID: ev.ObjectID(p.ZoneID), Direction: ev.AccessZonePassedV1DirectionExit}
	if p.Enter {
		d.Direction = ev.AccessZonePassedV1DirectionEnter
	}
	if p.ReaderID != "" {
		r := ev.ObjectID(p.ReaderID)
		d.ReaderID = &r
	}
	id := kernel.UUIDv5(constants.NsAnt, "skud:"+p.SourceEventID)
	return w.append(ctx, engineapp.Out{EventID: id, Type: catalog.AccessZonePassed, Kind: catalog.KindFact, Stream: "person:" + p.PersonID,
		OccurredAt: p.At.UTC().Truncate(time.Millisecond), Data: d})
}

// Revoked — access.workplace.revoked (реакция правила access.presence).
func (w PresenceWriter) Revoked(ctx context.Context, s accessdom.WorkplaceSession, cause, causeEventID string, at time.Time) error {
	id := kernel.UUIDv5(constants.NsAnt, "access.workplace.revoked|"+s.SessionID+"|"+cause)
	causes := []string{}
	if causeEventID != "" {
		causes = append(causes, causeEventID)
	}
	meta := &engineapp.ReactionMeta{RuleID: RulePresence, AutomationMode: 1, Version: 1, Causes: causes,
		Slot: engineapp.SlotMeta{RuleID: RulePresence, Subject: "workplace:" + s.WorkplaceID, TriggerKey: s.SessionID}}
	_, err := w.append(ctx, engineapp.Out{EventID: id, Type: catalog.AccessWorkplaceRevoked, Kind: catalog.KindReaction, Stream: "workplace:" + s.WorkplaceID,
		OccurredAt: at, Causation: causeEventID, Reaction: meta, Data: ev.AccessWorkplaceRevokedV1{WorkplaceID: ev.ObjectID(s.WorkplaceID),
			WorkplaceSessionID: ev.UUID(s.SessionID), Cause: ev.AccessWorkplaceRevokedV1Cause(cause)}})
	return err
}

// Deviation — security.presence.deviation в поток рабочего места (FR-84).
func (w PresenceWriter) Deviation(ctx context.Context, d accessdom.Deviation, at time.Time) error {
	id := kernel.UUIDv5(constants.NsAnt, "security.presence.deviation|"+d.Key)
	_, err := w.append(ctx, engineapp.Out{EventID: id, Type: catalog.SecurityPresenceDeviation, Kind: catalog.KindService, Stream: "workplace:" + d.WorkplaceID,
		OccurredAt: at, Causation: d.Cause, Data: ev.SecurityPresenceDeviationV1{Deviation: ev.SecurityPresenceDeviationV1Deviation(d.Kind),
			PersonID: ev.PersonRef(d.PersonID), WorkplaceID: ev.ObjectID(d.WorkplaceID)}})
	return err
}

// AdmissionDenied — security.admission.denied: отказ в допуске (FR-83, AD-24).
func (w PresenceWriter) AdmissionDenied(ctx context.Context, workplaceID, personID string, failed []string, at time.Time) error {
	d := ev.SecurityAdmissionDeniedV1{PersonID: ev.PersonRef(personID), WorkplaceID: ev.ObjectID(workplaceID)}
	for _, c := range failed {
		d.FailedChecks = append(d.FailedChecks, ev.SecurityAdmissionDeniedV1FailedChecksElem(c))
	}
	_, err := w.append(ctx, engineapp.Out{EventID: uuid.NewV7().String(), Type: catalog.SecurityAdmissionDenied, Kind: catalog.KindService,
		Stream: "workplace:" + workplaceID, OccurredAt: at, Data: d})
	return err
}

// append — запись одной пачкой; повтор event_id — false без ошибки.
func (w PresenceWriter) append(ctx context.Context, o engineapp.Out) (bool, error) {
	p, err := w.Codec.Encode(ctx, o)
	if err != nil {
		return false, err
	}
	_, err = w.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}})
	if errors.Is(err, appjournal.ErrDuplicate) {
		return false, nil
	}
	return err == nil, err
}
