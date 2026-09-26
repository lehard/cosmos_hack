package access

import (
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Квалификации и назначения на посты (FR-80, FR-81, PRD §11.18; эпик 26):
// данные журнала access.qualification.* и access.assignment.* в той же
// свёртке, что и политика. Гард назначения — чистая функция: исполнителя
// назначает мастер, только допущенного по квалификации на дату смены;
// контролёра — только по закрытому маршруту документа «запрос мастера →
// согласование начальника ОТК» (независимость ОТК).

// Роли назначения на пост (access.assignment.set.assignee_role).
const (
	AssigneePerformer = "performer"
	AssigneeInspector = "quality_inspector"
)

// Qualification — квалификация или аттестация со сроком и областью (FR-80).
type Qualification struct {
	PersonID        string
	QualificationID string
	Scope           string
	CertificateRef  string
	ValidFrom       time.Time
	ValidUntil      time.Time
	Revoked         bool
}

// PostAssignment — назначение на пост в смене (FR-81).
type PostAssignment struct {
	WorkplaceID        string
	ShiftID            string
	PersonID           string
	AssigneeRole       string
	ApprovalDocumentID string
	// Seq, At — запись назначения.
	Seq int64
	At  time.Time
}

// applyRoster — свёртка квалификаций и назначений; false — тип не из этой части.
func (p *Policy) applyRoster(r Record) (bool, error) {
	switch catalog.Type(r.Type) {
	case catalog.AccessQualificationGranted:
		var d ev.AccessQualificationGrantedV1
		if err := decode(r, &d); err != nil {
			return true, err
		}
		q := Qualification{PersonID: string(d.PersonID), QualificationID: string(d.QualificationID), ValidFrom: d.ValidFrom.Time()}
		if d.Scope != nil {
			q.Scope = *d.Scope
		}
		if d.CertificateRef != nil {
			q.CertificateRef = *d.CertificateRef
		}
		if d.ValidUntil != nil {
			q.ValidUntil = d.ValidUntil.Time()
		}
		p.Qualifications = append(p.Qualifications, q)
	case catalog.AccessQualificationRevoked:
		var d ev.AccessQualificationRevokedV1
		if err := decode(r, &d); err != nil {
			return true, err
		}
		at := d.EffectiveFrom.Time()
		for i, q := range p.Qualifications {
			if q.PersonID == string(d.PersonID) && q.QualificationID == string(d.QualificationID) && closes(q.ValidUntil, at) {
				p.Qualifications[i].ValidUntil, p.Qualifications[i].Revoked = at, true
			}
		}
	case catalog.AccessAssignmentSet:
		var d ev.AccessAssignmentSetV1
		if err := decode(r, &d); err != nil {
			return true, err
		}
		a := PostAssignment{WorkplaceID: string(d.WorkplaceID), ShiftID: string(d.ShiftID), PersonID: string(d.PersonID),
			AssigneeRole: string(d.AssigneeRole), Seq: r.Seq, At: r.OccurredAt}
		if d.ApprovalDocumentID != nil {
			a.ApprovalDocumentID = string(*d.ApprovalDocumentID)
		}
		p.Posts = slices.DeleteFunc(p.Posts, func(x PostAssignment) bool {
			return x.WorkplaceID == a.WorkplaceID && x.ShiftID == a.ShiftID && x.PersonID == a.PersonID
		})
		p.Posts = append(p.Posts, a)
	case catalog.AccessAssignmentCleared:
		var d ev.AccessAssignmentClearedV1
		if err := decode(r, &d); err != nil {
			return true, err
		}
		p.Posts = slices.DeleteFunc(p.Posts, func(x PostAssignment) bool {
			return x.WorkplaceID == string(d.WorkplaceID) && x.ShiftID == string(d.ShiftID) && x.PersonID == string(d.PersonID)
		})
	default:
		return false, nil
	}
	return true, nil
}

// QualificationsOf — квалификации сотрудника (все, с историей).
func (p Policy) QualificationsOf(personID string) []Qualification {
	var out []Qualification
	for _, q := range p.Qualifications {
		if q.PersonID == personID {
			out = append(out, q)
		}
	}
	return out
}

// QualifiedAt — действующая в момент at квалификация сотрудника, область
// которой включает место scope (FR-17, FR-80: проверяется на дату).
func (p Policy) QualifiedAt(personID, scope string, at time.Time) (Qualification, bool) {
	for _, q := range p.Qualifications {
		if q.PersonID == personID && ValidAt(at, q.ValidFrom, q.ValidUntil) && (q.Scope == "" || scope == "" || ScopeCovers(q.Scope, scope)) {
			return q, true
		}
	}
	return Qualification{}, false
}

// PostsIn — назначения смены shiftID (пусто — последние назначения каждого
// поста по всем сменам) в порядке записи.
func (p Policy) PostsIn(shiftID string) []PostAssignment {
	if shiftID != "" {
		var out []PostAssignment
		for _, a := range p.Posts {
			if a.ShiftID == shiftID {
				out = append(out, a)
			}
		}
		return out
	}
	last := map[string]int64{}
	for _, a := range p.Posts {
		if a.Seq > last[a.WorkplaceID] {
			last[a.WorkplaceID] = a.Seq
		}
	}
	shiftOf := map[string]string{}
	for _, a := range p.Posts {
		if a.Seq == last[a.WorkplaceID] {
			shiftOf[a.WorkplaceID] = a.ShiftID
		}
	}
	var out []PostAssignment
	for _, a := range p.Posts {
		if a.ShiftID == shiftOf[a.WorkplaceID] {
			out = append(out, a)
		}
	}
	return out
}

// AssignmentRequest — назначение на пост для гарда.
type AssignmentRequest struct {
	WorkplaceID string
	// WorkplaceScope — область поста (справочник мест).
	WorkplaceScope string
	ShiftID        string
	PersonID       string
	AssigneeRole   string
	// At — дата смены (начало или «сейчас», AD-37): квалификация — на неё.
	At time.Time
	// ApprovalClosed — для контролёра: документ согласования закрыт, выдаёт
	// именно это назначение и подписан начальником ОТК (проверяет api).
	ApprovalClosed bool
}

// GuardAssignment — гард назначения на пост (FR-81, PRD §11.18): у сотрудника
// роль вида назначения (с наследованием) в области поста; исполнитель —
// действующая квалификация на дату смены в области поста; контролёр —
// только по закрытому маршруту согласования начальника ОТК.
func GuardAssignment(p Policy, rq AssignmentRequest) error {
	if _, ok := p.Person(rq.PersonID); !ok {
		return kernel.Refuse(errcodes.ApiNotFound, "object", "сотрудник", "id", rq.PersonID)
	}
	if !p.HasRole(rq.PersonID, rq.AssigneeRole, rq.WorkplaceScope, rq.At) {
		return kernel.Refuse(errcodes.AccessWrongWorkplace, "workplace", rq.WorkplaceID+" ("+rq.WorkplaceScope+")")
	}
	switch rq.AssigneeRole {
	case AssigneeInspector:
		if !rq.ApprovalClosed {
			return kernel.Refuse(errcodes.AccessControllerApprovalRequired, "workplace", rq.WorkplaceID)
		}
	default:
		if _, ok := p.QualifiedAt(rq.PersonID, rq.WorkplaceScope, rq.At); !ok {
			return kernel.Refuse(errcodes.AccessNotQualified, "person", rq.PersonID, "workplace", rq.WorkplaceID, "date", rq.At.UTC().Format(time.DateOnly))
		}
	}
	return nil
}

// ControllerDecision — решение документа согласования контролёра:
// `quality_inspector:‹кто›@‹смена›` (объект документа — пост).
func ControllerDecision(personID, shiftID string) string {
	return AssigneeInspector + ":" + personID + "@" + shiftID
}
