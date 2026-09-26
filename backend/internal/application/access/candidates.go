package access

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	accessdom "ant/internal/domain/access"
)

// Кандидаты на пост в смене (FR-80, FR-81, PRD §11.18; интерфейс 6 — «Смена →
// Назначить на пост» у мастера и начальника цеха): кого можно назначить на
// пост без права читать всех сотрудников, роли и квалификации предприятия
// (access.person.list, access.role.list, access.qualification.list — у
// администратора и Аудитора ИБ). Условия — те же, что проверяет гард
// access.assignment.set (accessdom.GuardAssignment): роль в области поста,
// у исполнителя — действующая квалификация на дату смены, у контролёра —
// документ согласования начальника ОТК.

// InspectorPost — пост контролёра ОТК (пост ОТК цеха, окончательный
// контроль): на него назначают контролёра, на остальные — исполнителя.
func InspectorPost(workplaceID string) bool {
	return strings.HasPrefix(workplaceID, "WP-QC-") || strings.HasPrefix(workplaceID, "WP-FINAL-")
}

// ShiftDate — дата смены для проверки квалификации: смена справочника
// `‹шаблон›@ГГГГ-ММ-ДД` — полдень этой даты по UTC, иначе — момент чтения.
func ShiftDate(shiftID string, at time.Time) time.Time {
	if _, d, ok := strings.Cut(shiftID, "@"); ok {
		if t, err := time.Parse(time.DateOnly, d); err == nil {
			return t.Add(12 * time.Hour)
		}
	}
	return at
}

// QualificationVerdict — вердикт квалификации исполнителя на дату at: нет
// квалификации в области поста (found = false), истекла, истекает в 30 дней,
// действует — и почему словами.
func QualificationVerdict(found bool, qualificationID string, validUntil *time.Time, expired bool, at time.Time) (verdict, why string) {
	date := at.Format("02.01.2006")
	switch {
	case !found:
		return "missing", "Нет квалификации в области поста — назначение не пройдёт (FR-80)"
	case expired:
		return "expired", fmt.Sprintf("Квалификация %s истекла %s — на дату смены %s не действует", qualificationID, validUntil.Format("02.01.2006"), date)
	case validUntil != nil && validUntil.Sub(at) < expiringWithin:
		return "expiring", fmt.Sprintf("Квалификация %s действует до %s — истекает меньше чем через 30 дней", qualificationID, validUntil.Format("02.01.2006"))
	case validUntil != nil:
		return "ok", fmt.Sprintf("Квалификация %s действует до %s", qualificationID, validUntil.Format("02.01.2006"))
	}
	return "ok", "Квалификация " + qualificationID + " действует без срока"
}

// InspectorWhy — почему контролёр назначается иначе, чем исполнитель.
const InspectorWhy = "Контролёр ОТК: квалификация не проверяется, назначение — по документу «запрос мастера → согласование начальника ОТК» (PRD §11.18)"

// SortCandidates — порядок кандидатов: допустимые, свободные в смене, по вердикту и имени.
func SortCandidates(items []WorkplaceCandidate) {
	rank := map[string]int{"ok": 0, "expiring": 1, "expired": 2, "missing": 3}
	slices.SortStableFunc(items, func(a, b WorkplaceCandidate) int {
		if a.Allowed != b.Allowed {
			if a.Allowed {
				return -1
			}
			return 1
		}
		if (a.AssignedElsewhere == "") != (b.AssignedElsewhere == "") {
			if a.AssignedElsewhere == "" {
				return -1
			}
			return 1
		}
		return cmp.Or(rank[a.QualificationVerdict]-rank[b.QualificationVerdict], strings.Compare(a.Display, b.Display))
	})
}

// Candidates — кандидаты на пост в смене (access.candidate.list): live — по
// проекции политики; в режиме fixtures — у заготовок.
func (s *Service) Candidates(ctx context.Context, workplaceID, shiftID string, m platform.Moment) (WorkplaceCandidateList, error) {
	if !s.live || s.policy == nil || s.dir == nil {
		return s.Queries.Candidates(ctx, workplaceID, shiftID, m)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return WorkplaceCandidateList{}, err
	}
	wp, ok := s.dir.Workplace(workplaceID)
	if !ok {
		return WorkplaceCandidateList{}, platform.Fail(errcodes.ApiNotFound, "object", "пост", "id", workplaceID)
	}
	at := ShiftDate(shiftID, s.at(ctx, m))
	out := WorkplaceCandidateList{WorkplaceID: wp.ID, ShiftID: shiftID, Items: []WorkplaceCandidate{}, BasisSeq: pol.Seq}
	role := accessdom.AssigneePerformer
	if InspectorPost(wp.ID) {
		role = accessdom.AssigneeInspector
	}
	posts := pol.PostsIn(shiftID)
	for _, p := range pol.Persons {
		if !pol.HasRole(p.ID, role, wp.Scope, at) {
			continue
		}
		c := WorkplaceCandidate{PersonID: p.ID, Display: cmp.Or(p.Name, p.ID), Role: role}
		if role == accessdom.AssigneeInspector {
			c.QualificationVerdict, c.Why, c.Allowed, c.NeedsApproval = "ok", InspectorWhy, true, true
		} else {
			q, found := pol.QualifiedAt(p.ID, wp.Scope, at)
			expired := false
			if !found {
				// Квалификация в области есть, но не действует на дату — «истекла».
				for _, x := range pol.Qualifications {
					if x.PersonID == p.ID && !x.Revoked && (x.Scope == "" || accessdom.ScopeCovers(x.Scope, wp.Scope)) {
						q, found, expired = x, true, true
					}
				}
			}
			c.QualificationID, c.ValidUntil = q.QualificationID, until(q.ValidUntil)
			c.QualificationVerdict, c.Why = QualificationVerdict(found, q.QualificationID, c.ValidUntil, expired, at)
			c.Allowed = found && !expired
		}
		for _, a := range posts {
			if a.PersonID != p.ID {
				continue
			}
			if a.WorkplaceID == wp.ID {
				c.AssignedHere = true
			} else {
				c.AssignedElsewhere = a.WorkplaceID
			}
		}
		out.Items = append(out.Items, c)
	}
	SortCandidates(out.Items)
	return out, nil
}
