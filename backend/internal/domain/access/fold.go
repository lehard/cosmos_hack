package access

import (
	"encoding/json"
	"fmt"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
)

// Record — запись журнала для свёртки политики: seq, тип, data в текущей
// версии схемы (повышенные при чтении, AD-20), время происшествия.
type Record struct {
	Seq        int64
	Type       string
	Data       json.RawMessage
	OccurredAt time.Time
	// Genesis — запись генезиса (provenance_class genesis, AD-33).
	Genesis bool
	// EventID, Actor (key_id@версия первого подписанта), CausationID — для
	// истории выдачи прав у Аудитора ИБ; свёртка их не использует.
	EventID     string
	Actor       string
	CausationID string
}

// Types — типы записей, которые меняют политику и учётные записи (проекция
// политики читает из журнала только их).
var Types = []catalog.Type{
	catalog.PolicyRoleDefined, catalog.PolicyRoleAssigned, catalog.PolicyRoleUnassigned,
	catalog.PolicyAuthorityGranted, catalog.PolicyAuthorityRevoked,
	catalog.PolicyStampIssued, catalog.PolicyStampRevoked, catalog.PolicyAuditParametersSet,
	catalog.AccessPersonRegistered, catalog.AccessAccountActivated,
	catalog.AccessQualificationGranted, catalog.AccessQualificationRevoked,
	catalog.AccessAssignmentSet, catalog.AccessAssignmentCleared,
}

// Apply — свёртка политики (AD-15: источник правды — журнал): применяет
// запись к политике и сдвигает Seq. Записи других типов не меняют политику.
// Отзыв и снятие закрывают срок действующих назначений моментом effective_from
// (история назначений сохраняется — верификатор проверяет право на момент подписи).
func (p *Policy) Apply(r Record) error {
	if r.Seq > p.Seq {
		p.Seq = r.Seq
	}
	if ok, err := p.applyRoster(r); ok {
		return err
	}
	switch catalog.Type(r.Type) {
	case catalog.PolicyRoleDefined:
		var d ev.PolicyRoleDefinedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		role := Role{ID: string(d.RoleID), Title: d.Title, Actions: append([]string(nil), d.Actions...)}
		for _, b := range d.Inherits {
			role.Inherits = append(role.Inherits, string(b))
		}
		if old, ok := p.Role(role.ID); ok {
			role.CaseRole = old.CaseRole
		}
		p.upsertRole(role)
	case catalog.PolicyRoleAssigned:
		var d ev.PolicyRoleAssignedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		a := Assignment{PersonID: string(d.PersonID), RoleID: string(d.RoleID), Scope: d.Scope, ValidFrom: d.ValidFrom.Time()}
		if d.ValidUntil != nil {
			a.ValidUntil = d.ValidUntil.Time()
		}
		p.Assignments = append(p.Assignments, a)
	case catalog.PolicyRoleUnassigned:
		var d ev.PolicyRoleUnassignedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		at := d.EffectiveFrom.Time()
		for i, a := range p.Assignments {
			if a.PersonID == string(d.PersonID) && a.RoleID == string(d.RoleID) && a.Scope == d.Scope && closes(a.ValidUntil, at) {
				p.Assignments[i].ValidUntil = at
			}
		}
	case catalog.PolicyAuthorityGranted:
		var d ev.PolicyAuthorityGrantedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		a := Authority{PersonID: string(d.PersonID), AuthorityID: string(d.AuthorityID), Scope: d.Scope, ValidFrom: d.ValidFrom.Time()}
		if d.ValidUntil != nil {
			a.ValidUntil = d.ValidUntil.Time()
		}
		p.Authorities = append(p.Authorities, a)
	case catalog.PolicyAuthorityRevoked:
		var d ev.PolicyAuthorityRevokedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		at := d.EffectiveFrom.Time()
		for i, a := range p.Authorities {
			if a.PersonID == string(d.PersonID) && a.AuthorityID == string(d.AuthorityID) && a.Scope == d.Scope && closes(a.ValidUntil, at) {
				p.Authorities[i].ValidUntil = at
			}
		}
	case catalog.PolicyStampIssued:
		var d ev.PolicyStampIssuedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		s := Stamp{StampID: string(d.StampID), PersonID: string(d.PersonID), Kind: string(d.InspectionKind), Scope: d.Scope,
			OrderRef: d.OrderRef, ValidFrom: d.ValidFrom.Time()}
		if d.ValidUntil != nil {
			s.ValidUntil = d.ValidUntil.Time()
		}
		p.Stamps = append(p.Stamps, s)
	case catalog.PolicyStampRevoked:
		var d ev.PolicyStampRevokedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		at := d.EffectiveFrom.Time()
		for i, s := range p.Stamps {
			if s.StampID == string(d.StampID) && s.PersonID == string(d.PersonID) && closes(s.ValidUntil, at) {
				p.Stamps[i].ValidUntil = at
				p.Stamps[i].Revoked = true
			}
		}
	case catalog.PolicyAuditParametersSet:
		// Параметры аудита — данные журнала, а не конфигурация (AD-8, AD-15).
		var d ev.PolicyAuditParametersSetV1
		if err := decode(r, &d); err != nil {
			return err
		}
		a := AuditParameters{CheckpointIntervalS: d.CheckpointIntervalS, CheckpointMaxGapS: d.CheckpointMaxGapS,
			KeeperKeyFingerprint: string(d.KeeperKeyFingerprint), Seq: r.Seq, EventID: r.EventID, SetBy: r.Actor, SetAt: r.OccurredAt}
		for _, t := range d.CriticalTypes {
			a.CriticalTypes = append(a.CriticalTypes, string(t))
		}
		for _, x := range d.SecurityBusSubscribers {
			a.SecurityBusSubscribers = append(a.SecurityBusSubscribers, string(x))
		}
		p.Audit = a
	case catalog.AccessPersonRegistered:
		var d ev.AccessPersonRegisteredV1
		if err := decode(r, &d); err != nil {
			return err
		}
		x := Person{ID: string(d.PersonID), Name: d.DisplayName}
		if d.OrgUnit != nil {
			x.OrgUnit = *d.OrgUnit
		}
		if old, ok := p.Person(x.ID); ok {
			x.Login, x.Active = old.Login, old.Active
		}
		p.upsertPerson(x)
	case catalog.AccessAccountActivated:
		var d ev.AccessAccountActivatedV1
		if err := decode(r, &d); err != nil {
			return err
		}
		x, ok := p.Person(string(d.PersonID))
		if !ok {
			x = Person{ID: string(d.PersonID), Name: string(d.PersonID)}
		}
		// Логин однозначен: прежний владелец логина его теряет.
		for i := range p.Persons {
			if p.Persons[i].Login == d.Login && p.Persons[i].ID != x.ID {
				p.Persons[i].Login, p.Persons[i].Active = "", false
			}
		}
		x.Login, x.Active = d.Login, true
		p.upsertPerson(x)
	}
	return nil
}

// PolicyAt — политика на позиции журнала seq (AD-9, FR-85: «у подписанта было
// право на момент подписи»): base (затравка или политика генезиса) плюс записи
// recs с seq ≤ seq по возрастанию seq. Записи после seq не применяются.
func PolicyAt(base Policy, recs []Record, seq int64) (Policy, error) {
	p := base.Clone()
	for _, r := range recs {
		if r.Seq > seq {
			break
		}
		if err := p.Apply(r); err != nil {
			return p, err
		}
	}
	return p, nil
}

// closes — срок until ещё открыт в момент at (бессрочно или позже at).
func closes(until, at time.Time) bool { return until.IsZero() || until.After(at) }

func (p *Policy) upsertRole(r Role) {
	for i := range p.Roles {
		if p.Roles[i].ID == r.ID {
			p.Roles[i] = r
			return
		}
	}
	p.Roles = append(p.Roles, r)
}

func (p *Policy) upsertPerson(x Person) {
	for i := range p.Persons {
		if p.Persons[i].ID == x.ID {
			p.Persons[i] = x
			return
		}
	}
	p.Persons = append(p.Persons, x)
}

func decode(r Record, v any) error {
	if err := json.Unmarshal(r.Data, v); err != nil {
		return fmt.Errorf("политика: запись seq %d %s: %w", r.Seq, r.Type, err)
	}
	return nil
}
