package access

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"uuid"

	itemapp "ant/internal/application/item"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/crypto"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
	"ant/internal/domain/kernel"
)

// Допуск к рабочему месту, квалификации и приём СКУД (эпик 37; FR-80,
// FR-82…FR-84, AD-15 барьер 2). Допуск и снятие — записи журнала
// access.workplace.admitted | released с workplace_session_id вместе с
// фактом «ключ вставлен | извлечён»; отказ — событие шины безопасности
// security.admission.denied. Проходы СКУД пишет адаптер СКУД через
// IngestZonePasses; отклонения присутствия — CheckPresence.

// admitLive — допуск ведётся живым (режим live модуля access, журнал, присутствие).
func (s *Service) admitLive() bool {
	return s.liveFacts() && s.presence != nil && s.dir != nil
}

// AdmitWorkplace — допуск к рабочему месту (access.workplace.admit, FR-83):
// в зоне по СКУД ∧ роль в области места ∧ квалификация на дату ∧ назначение
// на пост в смене ∧ ключ и PIN. Ключ и PIN подтверждает подписанный пакет
// команды (ключ открыт PIN-ом, AD-14) или поля key_ref и pin_verified от
// порта подписи. Отказ фиксируется в шине безопасности.
func (s *Service) AdmitWorkplace(ctx context.Context, workplaceID string, in AdmitWorkplace) (platform.Receipt, error) {
	if !s.admitLive() {
		return s.Commands.AdmitWorkplace(ctx, workplaceID, in)
	}
	wp, ok := s.dir.Workplace(workplaceID)
	if !ok {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "пост", "id", workplaceID)
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	pr, _, err := s.presenceNow(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if cur, ok := pr.Session(wp.ID); ok {
		if cur.PersonID == actor {
			return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "workplace_id", "reason", "допуск к этому рабочему месту уже открыт")
		}
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "workplace_id", "reason", "рабочее место занято: открыт допуск "+displayOf(pol, cur.PersonID))
	}
	signed := signedBy(in.CommandMeta().Signature, actor)
	token := signed || in.KeyRef != ""
	if in.KeyRef != "" && !keyOf(in.KeyRef, actor) {
		token = false
	}
	rq := accessdom.AdmissionRequest{WorkplaceID: wp.ID, Scope: wp.Scope, Zone: wp.Zone, ShiftID: in.ShiftID, PersonID: actor, At: now,
		Token: token, PIN: signed || in.PinVerified}
	post, failed := accessdom.CheckAdmission(pol, pr, rq)
	if len(failed) > 0 {
		if s.pw != nil {
			_ = s.pw.AdmissionDenied(ctx, wp.ID, actor, failed, now)
		}
		pe := platform.Fail(errcodes.AccessAdmissionDenied, "checks", checksText(failed))
		if len(failed) == 1 && failed[0] == accessdom.CheckNotInZone {
			pe = platform.Fail(errcodes.AccessNotInZone, "zone", wp.Zone)
		}
		pe.Detail = "Допуск к рабочему месту «" + wp.Name + "» не открыт: " + checksText(failed) + "."
		return platform.Receipt{}, pe
	}
	cmd := strings.ToLower(in.CommandMeta().CommandID)
	if _, err := uuid.Parse(cmd); err != nil || cmd == "" {
		cmd = uuid.NewV7().String()
	}
	meta := in.CommandMeta()
	meta.CommandID, meta.WorkplaceID = cmd, wp.ID
	sessionID := kernel.UUIDv5(cmd, "workplace_session")
	adm := ev.AccessWorkplaceAdmittedV1{WorkplaceID: ev.ObjectID(wp.ID), WorkplaceSessionID: ev.UUID(sessionID), PersonID: ev.PersonRef(actor)}
	if post.ShiftID != "" {
		sh := ev.ObjectID(post.ShiftID)
		adm.ShiftID = &sh
	}
	// Основания допуска — проход СКУД в зону места (и записи из запроса).
	var checks []ev.UUID
	if z, ok := pr.Zones[actor][wp.Zone]; ok && isUUID(z.EventID) {
		checks = append(checks, ev.UUID(z.EventID))
	}
	for _, id := range in.ChecksEventIDs {
		if isUUID(id) && !slices.Contains(checks, ev.UUID(id)) {
			checks = append(checks, ev.UUID(id))
		}
	}
	adm.ChecksEventIds = checks
	stream := "workplace:" + wp.ID
	recs := []itemapp.Record{
		{Type: catalog.AccessWorkplaceAdmitted, Stream: stream, Data: adm, Meta: meta, Actor: actor, OccurredAt: now, SignatureLevel: 2},
	}
	if t, ok := pr.Token(wp.ID); !ok || t.PersonID != actor {
		recs = append(recs, itemapp.Record{Type: catalog.AccessTokenPresenceChanged, Stream: stream, Meta: meta, Actor: actor, OccurredAt: now,
			Data: ev.AccessTokenPresenceChangedV1{PersonID: ev.PersonRef(actor), WorkplaceID: ev.ObjectID(wp.ID), Present: true}})
	}
	rc, err := s.facts.Write(ctx, kernel.Module("access"), recs)
	s.refreshPresence(ctx)
	return rc, err
}

// ReleaseWorkplace — снять допуск (access.workplace.release, FR-83): сеанс
// закрывает его владелец; ключ извлекается вместе со снятием.
func (s *Service) ReleaseWorkplace(ctx context.Context, workplaceID string, in ReleaseWorkplace) (platform.Receipt, error) {
	if !s.admitLive() {
		return s.Commands.ReleaseWorkplace(ctx, workplaceID, in)
	}
	pr, _, err := s.presenceNow(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	cur, ok := pr.Session(workplaceID)
	if !ok || (in.WorkplaceSessionID != "" && !strings.EqualFold(cur.SessionID, in.WorkplaceSessionID)) {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "допуск к рабочему месту", "id", workplaceID)
	}
	if cur.PersonID != actor {
		return platform.Receipt{}, platform.Fail(errcodes.AccessWrongWorkplace, "workplace", workplaceID)
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	meta := in.CommandMeta()
	meta.WorkplaceID = workplaceID
	stream := "workplace:" + workplaceID
	recs := []itemapp.Record{{Type: catalog.AccessWorkplaceReleased, Stream: stream, Meta: meta, Actor: actor, OccurredAt: now, SignatureLevel: 2,
		Data: ev.AccessWorkplaceReleasedV1{WorkplaceID: ev.ObjectID(workplaceID), WorkplaceSessionID: ev.UUID(cur.SessionID)}}}
	if t, ok := pr.Token(workplaceID); ok && t.PersonID == actor {
		recs = append(recs, itemapp.Record{Type: catalog.AccessTokenPresenceChanged, Stream: stream, Meta: meta, Actor: actor, OccurredAt: now,
			Data: ev.AccessTokenPresenceChangedV1{PersonID: ev.PersonRef(actor), WorkplaceID: ev.ObjectID(workplaceID), Present: false}})
	}
	rc, err := s.facts.Write(ctx, kernel.Module("access"), recs)
	s.refreshPresence(ctx)
	return rc, err
}

// GrantQualification — выдать квалификацию или аттестацию с областью и
// сроком (access.qualification.grant, FR-80): запись в поток сотрудника.
func (s *Service) GrantQualification(ctx context.Context, personID string, in GrantQualification) (platform.Receipt, error) {
	if !s.livePolicy() {
		return s.Commands.GrantQualification(ctx, personID, in)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if _, ok := pol.Person(personID); !ok {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "сотрудник", "id", personID)
	}
	if strings.TrimSpace(in.QualificationID) == "" {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "qualification_id", "reason", "квалификация")
	}
	scope := strings.Trim(in.Scope, "/")
	if scope != "" && !scopePattern.MatchString(scope) {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "scope", "reason", "путь области: здание/цех/участок/рабочее место")
	}
	from := in.ValidFrom
	if from.IsZero() {
		from = now
	}
	d := ev.AccessQualificationGrantedV1{PersonID: ev.PersonRef(personID), QualificationID: ev.ObjectID(in.QualificationID), ValidFrom: ev.Timestamp(from)}
	if scope != "" {
		d.Scope = &scope
	}
	if in.CertificateRef != "" {
		c := in.CertificateRef
		d.CertificateRef = &c
	}
	if in.ValidUntil != nil {
		if !in.ValidUntil.After(from) {
			return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "valid_until", "reason", "позже valid_from")
		}
		u := ev.Timestamp(*in.ValidUntil)
		d.ValidUntil = &u
	}
	rc, err := s.decisions.Write(ctx, Batch{Records: []Record{{Type: catalog.AccessQualificationGranted, Stream: "person:" + personID, Data: d}},
		Meta: in.CommandMeta(), Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now})
	s.refresh(ctx)
	return rc, err
}

// RevokeQualification — отозвать квалификацию (access.qualification.revoke,
// FR-80): с момента effective_from мастер не назначит сотрудника на пост, а
// открытый допуск, который держался на этой квалификации, снимается
// (access.workplace.revoked, причина qualification_revoked).
func (s *Service) RevokeQualification(ctx context.Context, personID string, in RevokeQualification) (platform.Receipt, error) {
	if !s.livePolicy() {
		return s.Commands.RevokeQualification(ctx, personID, in)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	at := in.EffectiveFrom
	if at.IsZero() {
		at = now
	}
	if !slices.ContainsFunc(pol.Qualifications, func(q accessdom.Qualification) bool {
		return q.PersonID == personID && q.QualificationID == in.QualificationID && !q.Revoked && (q.ValidUntil.IsZero() || q.ValidUntil.After(at))
	}) {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "квалификация", "id", in.QualificationID+" у "+personID)
	}
	d := ev.AccessQualificationRevokedV1{PersonID: ev.PersonRef(personID), QualificationID: ev.ObjectID(in.QualificationID), EffectiveFrom: ev.Timestamp(at)}
	if strings.TrimSpace(in.Reason.Text) != "" {
		r := ev.Reason{Text: ev.Text(in.Reason.Text)}
		if in.Reason.Code != "" {
			c := ev.Code(in.Reason.Code)
			r.Code = &c
		}
		d.Reason = &r
	}
	rc, err := s.decisions.Write(ctx, Batch{Records: []Record{{Type: catalog.AccessQualificationRevoked, Stream: "person:" + personID, Data: d}},
		Meta: in.CommandMeta(), Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now})
	if err != nil {
		return rc, err
	}
	s.refresh(ctx)
	s.revokeSessions(ctx, personID, "qualification_revoked", first(rc.EventIDs), now, func(pol accessdom.Policy, ss accessdom.WorkplaceSession) bool {
		wp, _ := s.dir.Workplace(ss.WorkplaceID)
		_, ok := pol.QualifiedAt(personID, wp.Scope, now)
		return !ok
	})
	return rc, nil
}

// revokeSessions — снять открытые допуски сотрудника, для которых drop
// вернул true (реакция access.workplace.revoked с причиной cause).
func (s *Service) revokeSessions(ctx context.Context, personID, cause, causeEventID string, at time.Time, drop func(accessdom.Policy, accessdom.WorkplaceSession) bool) {
	if s.pw == nil || s.presence == nil || s.dir == nil {
		return
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return
	}
	pr, _, err := s.presenceNow(ctx)
	if err != nil {
		return
	}
	for _, ss := range pr.SessionsOf(personID) {
		if drop == nil || drop(pol, ss) {
			_ = s.pw.Revoked(ctx, ss, cause, causeEventID, at)
		}
	}
	s.refreshPresence(ctx)
}

// IngestZonePasses — приём проходов СКУД (FR-82): каждый проход — факт
// access.zone.passed (повтор опроса не пишется дважды); после приёма —
// отклонения присутствия (CheckPresence). Вызывает адаптер СКУД.
func (s *Service) IngestZonePasses(ctx context.Context, passes []ZonePassIn) (int, error) {
	if s.pw == nil {
		return 0, errors.New("access: запись фактов СКУД не подключена")
	}
	n := 0
	for _, p := range passes {
		ok, err := s.pw.ZonePassed(ctx, p)
		if err != nil {
			return n, err
		}
		if ok {
			n++
		}
	}
	if n > 0 {
		s.refreshPresence(ctx)
		if err := s.CheckPresence(ctx); err != nil {
			return n, err
		}
	}
	return n, nil
}

// PresenceGrace — через сколько после начала смены отсутствие ключа у
// назначенного исполнителя — отклонение «по графику должен быть, ключа нет».
const PresenceGrace = 15 * time.Minute

// CheckPresence — отклонения присутствия (FR-84): «ключ вставлен, владельца
// нет в зоне» — тревога администратору (шина безопасности), допуск снимается
// (причина zone_exit); «по графику должен быть, ключа нет» — мастеру (запись
// в поток поста, видна на панели «Посты» и в истории поста). Повтор того же
// отклонения не пишется (однозначный ключ основания).
func (s *Service) CheckPresence(ctx context.Context) error {
	if s.pw == nil || s.presence == nil || s.policy == nil || s.now == nil {
		return nil
	}
	pr, _, err := s.presenceNow(ctx)
	if err != nil {
		return err
	}
	now, err := s.now(ctx)
	if err != nil {
		return err
	}
	devs := accessdom.TokenWithoutPresence(pr, s.zoneOf)
	for _, d := range devs {
		if err := s.pw.Deviation(ctx, d, now); err != nil {
			return err
		}
		if ss, ok := pr.Session(d.WorkplaceID); ok && ss.PersonID == d.PersonID {
			if err := s.pw.Revoked(ctx, ss, "zone_exit", d.Cause, now); err != nil {
				return err
			}
		}
	}
	if s.shifts != nil {
		pol, err := s.policy.Policy(ctx)
		if err != nil {
			return err
		}
		shifts, err := s.shifts.ShiftsAt(ctx, now)
		if err != nil {
			return err
		}
		for _, sh := range shifts {
			for _, d := range accessdom.ScheduledWithoutToken(pol, pr, sh.ID, sh.Start, now, PresenceGrace) {
				if err := s.pw.Deviation(ctx, d, now); err != nil {
					return err
				}
			}
		}
	}
	s.refreshPresence(ctx)
	return nil
}

// checksText — невыполненные проверки допуска по-русски.
func checksText(failed []string) string {
	t := make([]string, len(failed))
	for i, c := range failed {
		t[i] = accessdom.CheckTitle(c)
	}
	return strings.Join(t, "; ")
}

// signedBy — пакет подписи команды несёт подпись ключом сотрудника (ключ
// вставлен и открыт PIN-ом, AD-14). Криптографическую проверку пакета делает
// порт подписи операций уровня ≥ 1; здесь — только чей ключ.
func signedBy(raw []byte, personID string) bool {
	if len(raw) == 0 {
		return false
	}
	var env crypto.DsseEnvelope
	if json.Unmarshal(raw, &env) != nil {
		return false
	}
	for _, sg := range env.Signatures {
		if sg.Sig != "" && keyOf(sg.Keyid, personID) {
			return true
		}
	}
	return false
}

// keyOf — ключ key_ref (`‹псевдоним›@‹версия›`) принадлежит сотруднику.
func keyOf(keyRef, personID string) bool {
	id, _, _ := strings.Cut(keyRef, "@")
	return strings.EqualFold(id, personID)
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil && s != ""
}

func first(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}
