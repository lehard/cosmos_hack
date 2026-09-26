package access

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Присутствие на посту и допуск к рабочему месту (FR-6, FR-82, FR-83, FR-84;
// AD-15, барьер 2; эпик 37). Свёртка записей СКУД (access.zone.passed),
// ключа (access.token.presence_changed) и сеансов рабочего места
// (access.workplace.admitted | released | revoked) — отдельно от политики:
// проходы по зонам не меняют права и не сдвигают policy_seq.
//
// Допуск — чистая функция CheckAdmission над политикой и присутствием:
// в зоне по СКУД ∧ роль в области места ∧ квалификация на дату ∧ назначение
// на пост в смене ∧ ключ и PIN. Отклонения присутствия — тоже чистые функции:
// «ключ вставлен, владельца нет в зоне» (тревога администратору) и «по
// графику должен быть, ключа нет» (мастеру).

// PresenceTypes — типы записей свёртки присутствия (проекция читает только их).
var PresenceTypes = []catalog.Type{
	catalog.AccessZonePassed, catalog.AccessTokenPresenceChanged,
	catalog.AccessWorkplaceAdmitted, catalog.AccessWorkplaceReleased, catalog.AccessWorkplaceRevoked,
}

// Значения присутствия на посту (PostRow.presence, FR-6): данных СКУД нет —
// «неизвестно», а не «на месте».
const (
	PresencePresent     = "present"
	PresenceKeyMissing  = "key_missing"
	PresenceOwnerAbsent = "owner_absent"
	PresenceAbsent      = "absent"
	PresenceNotAssigned = "not_assigned"
	PresenceUnknown     = "unknown"
)

// Виды отклонения присутствия (security.presence.deviation, FR-84).
const (
	DeviationTokenWithoutPresence = "token_without_presence"
	DeviationScheduledWithoutKey  = "scheduled_without_token"
)

// ZonePass — последний проход сотрудника через точку СКУД зоны.
type ZonePass struct {
	PersonID string
	ZoneID   string
	// In — вошёл (enter); false — вышел (exit).
	In      bool
	At      time.Time
	Seq     int64
	EventID string
}

// TokenIn — ключ, вставленный на рабочем месте.
type TokenIn struct {
	PersonID string
	At       time.Time
	Seq      int64
	EventID  string
}

// WorkplaceSession — открытый сеанс рабочего места (допуск, барьер 2).
type WorkplaceSession struct {
	SessionID   string
	WorkplaceID string
	PersonID    string
	ShiftID     string
	At          time.Time
	Seq         int64
}

// Presence — свёртка присутствия: зоны по СКУД, ключи и сеансы рабочих мест.
type Presence struct {
	// Seq — seq последней применённой записи.
	Seq int64
	// Zones — последний проход сотрудника по каждой зоне: сотрудник → зона → проход.
	Zones map[string]map[string]ZonePass
	// Tokens — вставленные ключи: рабочее место → ключ.
	Tokens map[string]TokenIn
	// Sessions — открытые сеансы: рабочее место → сеанс.
	Sessions map[string]WorkplaceSession
}

// NewPresence — пустая свёртка присутствия.
func NewPresence() Presence {
	return Presence{Zones: map[string]map[string]ZonePass{}, Tokens: map[string]TokenIn{}, Sessions: map[string]WorkplaceSession{}}
}

// Clone — независимая копия (проекция отдаёт копию, свёртка идёт дальше).
func (p Presence) Clone() Presence {
	c := Presence{Seq: p.Seq, Zones: make(map[string]map[string]ZonePass, len(p.Zones)), Tokens: make(map[string]TokenIn, len(p.Tokens)),
		Sessions: make(map[string]WorkplaceSession, len(p.Sessions))}
	for k, v := range p.Zones {
		m := make(map[string]ZonePass, len(v))
		for z, x := range v {
			m[z] = x
		}
		c.Zones[k] = m
	}
	for k, v := range p.Tokens {
		c.Tokens[k] = v
	}
	for k, v := range p.Sessions {
		c.Sessions[k] = v
	}
	return c
}

// Apply — применить запись к свёртке присутствия; записи других типов не меняют её.
func (p *Presence) Apply(r Record) error {
	if p.Zones == nil || p.Tokens == nil || p.Sessions == nil {
		*p = p.Clone()
	}
	if r.Seq > p.Seq {
		p.Seq = r.Seq
	}
	switch catalog.Type(r.Type) {
	case catalog.AccessZonePassed:
		var d ev.AccessZonePassedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		person := string(d.PersonID)
		if p.Zones[person] == nil {
			p.Zones[person] = map[string]ZonePass{}
		}
		p.Zones[person][string(d.ZoneID)] = ZonePass{PersonID: person, ZoneID: string(d.ZoneID), In: d.Direction == ev.AccessZonePassedV1DirectionEnter,
			At: r.OccurredAt, Seq: r.Seq, EventID: r.EventID}
	case catalog.AccessTokenPresenceChanged:
		var d ev.AccessTokenPresenceChangedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		wp := string(d.WorkplaceID)
		switch {
		case d.Present:
			p.Tokens[wp] = TokenIn{PersonID: string(d.PersonID), At: r.OccurredAt, Seq: r.Seq, EventID: r.EventID}
		case p.Tokens[wp].PersonID == string(d.PersonID):
			delete(p.Tokens, wp)
		}
	case catalog.AccessWorkplaceAdmitted:
		var d ev.AccessWorkplaceAdmittedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		s := WorkplaceSession{SessionID: string(d.WorkplaceSessionID), WorkplaceID: string(d.WorkplaceID), PersonID: string(d.PersonID), At: r.OccurredAt, Seq: r.Seq}
		if d.ShiftID != nil {
			s.ShiftID = string(*d.ShiftID)
		}
		p.Sessions[s.WorkplaceID] = s
	case catalog.AccessWorkplaceReleased:
		var d ev.AccessWorkplaceReleasedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		p.closeSession(string(d.WorkplaceID), string(d.WorkplaceSessionID))
	case catalog.AccessWorkplaceRevoked:
		var d ev.AccessWorkplaceRevokedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		p.closeSession(string(d.WorkplaceID), string(d.WorkplaceSessionID))
	}
	return nil
}

func (p *Presence) closeSession(wp, id string) {
	if s, ok := p.Sessions[wp]; ok && (id == "" || s.SessionID == id) {
		delete(p.Sessions, wp)
	}
}

// Known — есть ли у сотрудника хоть один проход по СКУД (иначе присутствие неизвестно).
func (p Presence) Known(personID string) bool { return len(p.Zones[personID]) > 0 }

// InZone — сотрудник в зоне zone по последнему проходу; known=false — данных
// СКУД о сотруднике нет или у места нет зоны.
func (p Presence) InZone(personID, zone string) (in, known bool) {
	if zone == "" || !p.Known(personID) {
		return false, false
	}
	z, ok := p.Zones[personID][zone]
	return ok && z.In, true
}

// ZonesOf — зоны, в которых сотрудник сейчас находится (по последним проходам), по алфавиту.
func (p Presence) ZonesOf(personID string) []string {
	var out []string
	for z, x := range p.Zones[personID] {
		if x.In {
			out = append(out, z)
		}
	}
	slices.Sort(out)
	return out
}

// Token — владелец ключа, вставленного на рабочем месте.
func (p Presence) Token(workplaceID string) (TokenIn, bool) {
	t, ok := p.Tokens[workplaceID]
	return t, ok
}

// Session — открытый сеанс рабочего места.
func (p Presence) Session(workplaceID string) (WorkplaceSession, bool) {
	s, ok := p.Sessions[workplaceID]
	return s, ok
}

// SessionOf — открытый сеанс сотрудника (первое рабочее место по алфавиту).
func (p Presence) SessionOf(personID string) (WorkplaceSession, bool) {
	var ids []string
	for wp, s := range p.Sessions {
		if s.PersonID == personID {
			ids = append(ids, wp)
		}
	}
	if len(ids) == 0 {
		return WorkplaceSession{}, false
	}
	slices.Sort(ids)
	return p.Sessions[ids[0]], true
}

// PostPresence — присутствие назначенного сотрудника на посту (FR-6): в зоне
// поста по СКУД и ключ вставлен — на месте; в зоне без ключа — «ключ не
// вставлен»; ключ вставлен, а владельца нет в зоне — «владельца нет»; ни
// того ни другого — нет на месте; данных СКУД нет — неизвестно.
func (p Presence) PostPresence(workplaceID, zone, personID string) string {
	if personID == "" {
		return PresenceNotAssigned
	}
	in, known := p.InZone(personID, zone)
	if !known {
		return PresenceUnknown
	}
	t, ok := p.Tokens[workplaceID]
	mine := ok && t.PersonID == personID
	switch {
	case in && mine:
		return PresencePresent
	case in:
		return PresenceKeyMissing
	case mine:
		return PresenceOwnerAbsent
	}
	return PresenceAbsent
}

// Проверки допуска (security.admission.denied.failed_checks).
const (
	CheckNotInZone     = "not_in_zone"
	CheckNoRoleInScope = "no_role_in_scope"
	CheckQualification = "qualification_invalid"
	CheckNotAssigned   = "not_assigned"
	CheckNoToken       = "no_token"
	CheckWrongPIN      = "wrong_pin"
)

// CheckTitle — проверка допуска по-русски (текст отказа).
func CheckTitle(c string) string {
	switch c {
	case CheckNotInZone:
		return "нет в зоне по СКУД"
	case CheckNoRoleInScope:
		return "нет роли в области рабочего места"
	case CheckQualification:
		return "нет действующей квалификации на дату"
	case CheckNotAssigned:
		return "нет назначения на пост в смене"
	case CheckNoToken:
		return "ключ не вставлен"
	case CheckWrongPIN:
		return "PIN не подтверждён"
	}
	return c
}

// AdmissionRequest — запрос допуска к рабочему месту (барьер 2).
type AdmissionRequest struct {
	WorkplaceID string
	// Scope — область рабочего места; Zone — зона СКУД, в которой оно.
	Scope string
	Zone  string
	// ShiftID — смена; пусто — последние назначения поста.
	ShiftID  string
	PersonID string
	// At — доменное время запроса (AD-37): роль и квалификация — на него.
	At time.Time
	// Token — ключ сотрудника вставлен; PIN — PIN введён и ключ им открыт.
	Token bool
	PIN   bool
}

// CheckAdmission — допуск к рабочему месту (FR-83, AD-15): невыполненные
// проверки в постоянном порядке (пусто — допуск разрешён) и назначение, на
// котором он основан. Роль назначения — performer или quality_inspector (с
// наследованием) в области рабочего места; квалификация — у исполнителя.
func CheckAdmission(pol Policy, pr Presence, rq AdmissionRequest) (PostAssignment, []string) {
	var failed []string
	if in, _ := pr.InZone(rq.PersonID, rq.Zone); !in {
		failed = append(failed, CheckNotInZone)
	}
	var post PostAssignment
	found := false
	for _, a := range pol.PostsIn(rq.ShiftID) {
		if a.WorkplaceID == rq.WorkplaceID && a.PersonID == rq.PersonID {
			post, found = a, true
		}
	}
	role := AssigneePerformer
	if found {
		role = post.AssigneeRole
	} else {
		failed = append(failed, CheckNotAssigned)
	}
	if !pol.HasRole(rq.PersonID, role, rq.Scope, rq.At) {
		failed = append(failed, CheckNoRoleInScope)
	}
	if role == AssigneePerformer {
		if _, ok := pol.QualifiedAt(rq.PersonID, rq.Scope, rq.At); !ok {
			failed = append(failed, CheckQualification)
		}
	}
	if !rq.Token {
		failed = append(failed, CheckNoToken)
	} else if !rq.PIN {
		failed = append(failed, CheckWrongPIN)
	}
	return post, failed
}

// AdmissionRefusal — отказ в допуске с перечнем невыполненного; одна
// проверка «нет в зоне» — свой код access.not_in_zone.
func AdmissionRefusal(rq AdmissionRequest, failed []string) error {
	if len(failed) == 0 {
		return nil
	}
	if len(failed) == 1 && failed[0] == CheckNotInZone {
		return kernel.Refuse(errcodes.AccessNotInZone, "zone", rq.Zone)
	}
	titles := make([]string, len(failed))
	for i, c := range failed {
		titles[i] = CheckTitle(c)
	}
	return kernel.Refuse(errcodes.AccessAdmissionDenied, "checks", strings.Join(titles, "; "))
}

// Deviation — отклонение присутствия, которое нужно поднять (FR-84).
type Deviation struct {
	Kind        string
	WorkplaceID string
	PersonID    string
	// Key — основание (однозначный ключ): повтор того же отклонения не пишется.
	Key string
	// Cause — event_id записи-причины (выход из зоны); пусто — проверка по графику.
	Cause string
}

// TokenWithoutPresence — «ключ вставлен, владельца нет в зоне» (тревога
// администратору): рабочие места, где вставлен ключ сотрудника, которого по
// СКУД нет в зоне места. zoneOf — зона рабочего места.
func TokenWithoutPresence(pr Presence, zoneOf func(workplaceID string) string) []Deviation {
	var out []Deviation
	for _, wp := range sortedKeys(pr.Tokens) {
		t := pr.Tokens[wp]
		zone := zoneOf(wp)
		in, known := pr.InZone(t.PersonID, zone)
		if !known || in {
			continue
		}
		exit := pr.Zones[t.PersonID][zone]
		cause := exit.EventID
		out = append(out, Deviation{Kind: DeviationTokenWithoutPresence, WorkplaceID: wp, PersonID: t.PersonID,
			Key: DeviationTokenWithoutPresence + "|" + wp + "|" + t.PersonID + "|" + t.EventID + "|" + cause, Cause: cause})
	}
	return out
}

// ScheduledWithoutToken — «по графику должен быть, ключа нет» (мастеру):
// назначенные исполнители смены shiftID (начало — shiftStart), у которых к
// моменту at (не раньше shiftStart + grace) ключ на посту не вставлен.
func ScheduledWithoutToken(pol Policy, pr Presence, shiftID string, shiftStart, at time.Time, grace time.Duration) []Deviation {
	if shiftID == "" || at.Before(shiftStart.Add(grace)) {
		return nil
	}
	var out []Deviation
	for _, a := range pol.PostsIn(shiftID) {
		if a.AssigneeRole != AssigneePerformer {
			continue
		}
		if t, ok := pr.Tokens[a.WorkplaceID]; ok && t.PersonID == a.PersonID {
			continue
		}
		out = append(out, Deviation{Kind: DeviationScheduledWithoutKey, WorkplaceID: a.WorkplaceID, PersonID: a.PersonID,
			Key: DeviationScheduledWithoutKey + "|" + a.WorkplaceID + "|" + a.PersonID + "|" + shiftID + "|" + shiftStart.UTC().Format(TimeLayout)})
	}
	return out
}

// SessionsOf — открытые сеансы сотрудника по рабочим местам (по алфавиту).
func (p Presence) SessionsOf(personID string) []WorkplaceSession {
	var out []WorkplaceSession
	for _, wp := range sortedKeys(p.Sessions) {
		if s := p.Sessions[wp]; s.PersonID == personID {
			out = append(out, s)
		}
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// ZonePassRecord — данные записи прохода (для адаптера СКУД и тестов).
func ZonePassRecord(personID, zoneID, readerID string, enter bool) json.RawMessage {
	d := ev.AccessZonePassedV1{PersonID: ev.PersonRef(personID), ZoneID: ev.ObjectID(zoneID), Direction: ev.AccessZonePassedV1DirectionExit}
	if enter {
		d.Direction = ev.AccessZonePassedV1DirectionEnter
	}
	if readerID != "" {
		r := ev.ObjectID(readerID)
		d.ReaderID = &r
	}
	b, _ := json.Marshal(d)
	return b
}
