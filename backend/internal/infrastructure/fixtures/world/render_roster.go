package world

import (
	"slices"
	"strings"
	"time"

	accessapp "ant/internal/application/access"
	refapp "ant/internal/application/reference"
	accessdom "ant/internal/domain/access"
	"ant/internal/infrastructure/fixtures/loader"
)

// Смены, план смены и кандидаты на пост в мире заготовок (интерфейс 6, стол
// мастера и начальника цеха; FR-80, FR-81, PRD §11.18):
//   - reference.shift.list — смены шаблонов normative/reference/flange/shifts.yaml
//     по рабочим дням в окне live (от полусуток до момента до полутора суток
//     после), id — как у live: `‹шаблон›@ГГГГ-ММ-ДД`;
//   - access.assignment.list — план смен этого окна: люди мира на своих постах
//     (те же, кто «на месте» на панели «Посты»); допуск — у идущей смены по
//     присутствию на посту;
//   - access.candidate.list — кандидаты на пост по политике затравки: роль в
//     области поста и квалификация на дату (как гард access.assignment.set).

// shiftWindowBefore, shiftWindowAfter — окно графика смен (как у live reference.shift.list).
const (
	shiftWindowBefore = 12 * time.Hour
	shiftWindowAfter  = 36 * time.Hour
)

// shiftInstance — смена шаблона в конкретный день.
type shiftInstance struct {
	id, pattern, name string
	from, to          time.Time
	locations         []string
}

// shiftsBetween — смены шаблонов по рабочим дням (без субботы и воскресенья),
// пересекающие окно [from, to).
func (m *Model) shiftsBetween(from, to time.Time) []shiftInstance {
	loc := m.clk.loc
	f := from.In(loc)
	var out []shiftInstance
	for day := time.Date(f.Year(), f.Month(), f.Day()-1, 0, 0, 0, 0, loc); day.Before(to); day = day.AddDate(0, 0, 1) {
		if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		for _, p := range m.shifts {
			s := day.Add(time.Duration(p.starts) * time.Minute)
			e := day.Add(time.Duration(p.ends) * time.Minute)
			if !e.After(s) {
				e = e.Add(24 * time.Hour)
			}
			if s.Before(to) && e.After(from) {
				out = append(out, shiftInstance{id: p.id + "@" + day.Format(time.DateOnly), pattern: p.id, name: p.name, from: s.UTC(), to: e.UTC(), locations: p.locations})
			}
		}
	}
	slices.SortStableFunc(out, func(a, b shiftInstance) int { return a.from.Compare(b.from) })
	return out
}

// scopeCovers — область granted включает место target (путь или его префикс).
func scopeCovers(granted, target string) bool {
	return granted == "" || granted == target || strings.HasPrefix(target, granted+"/")
}

// qualificationAt — квалификация сотрудника в области места на момент at:
// found — есть в области, expired — есть, но на дату не действует.
func (m *Model) qualificationAt(person, scope string, at time.Time) (q PolicyQualification, until *time.Time, found, expired bool) {
	if m.policy == nil {
		return q, nil, false, false
	}
	for _, x := range m.policy.Grants.Qualifications {
		if x.Person != person || !scopeCovers(x.Scope, scope) {
			continue
		}
		var u *time.Time
		if t, err := time.Parse(time.RFC3339, x.ValidUntil); err == nil {
			u = &t
		}
		ok := u == nil || u.After(at)
		if ok || !found {
			q, until, found, expired = x, u, true, !ok
		}
		if ok {
			return
		}
	}
	return
}

// postCandidates — кандидаты на пост на дату at по политике затравки (без отметок
// «уже назначен» — их ставит адаптер по плану смены с наложением сессии).
func (m *Model) postCandidates(wp string, at time.Time) accessapp.WorkplaceCandidateList {
	out := accessapp.WorkplaceCandidateList{WorkplaceID: wp, Items: []accessapp.WorkplaceCandidate{}, BasisSeq: 1}
	if m.policy == nil {
		return out
	}
	h := make(accessdom.Roles, len(m.policy.Roles))
	for _, r := range m.policy.Roles {
		h[r.ID] = r.Inherits
	}
	role := accessdom.AssigneePerformer
	if accessapp.InspectorPost(wp) {
		role = accessdom.AssigneeInspector
	}
	scope := workplaceScope[wp]
	for _, p := range m.policy.Persons {
		has := false
		for _, r := range p.Roles {
			has = has || (scopeCovers(r.Scope, scope) && slices.Contains(h.Closure(r.Role), role))
		}
		if !has {
			continue
		}
		c := accessapp.WorkplaceCandidate{PersonID: p.ID, Display: p.Name, Role: role}
		if role == accessdom.AssigneeInspector {
			c.QualificationVerdict, c.Why, c.Allowed, c.NeedsApproval = "ok", accessapp.InspectorWhy, true, true
		} else {
			q, until, found, expired := m.qualificationAt(p.ID, scope, at)
			c.QualificationID, c.ValidUntil = q.Qualification, until
			c.QualificationVerdict, c.Why = accessapp.QualificationVerdict(found, q.Qualification, until, expired, at)
			c.Allowed = found && !expired
		}
		out.Items = append(out.Items, c)
	}
	accessapp.SortCandidates(out.Items)
	return out
}

// controllerApproval — документы согласования контролёров мира (render_documents.go):
// пост → документ и день, с которого он действует.
var controllerApproval = map[string]struct {
	doc string
	day int
}{"WP-QC-WC": {"DOC-CA-WP-QC-WC-0921", 21}, "WP-QC-AC": {"DOC-CA-WP-QC-AC-0924", 24}}

// renderRoster — смены, план смен и кандидаты на посты на часах шага.
func renderRoster(c *Ctx) []loader.Response {
	shifts := c.M.shiftsBetween(c.T.Add(-shiftWindowBefore), c.T.Add(shiftWindowAfter))
	list := refapp.RefShiftList{Items: []refapp.RefShift{}}
	plan := accessapp.AccessAssignmentList{Items: []accessapp.AccessAssignment{}, BasisSeq: c.Seq()}
	for _, s := range shifts {
		for _, l := range s.locations {
			list.Items = append(list.Items, refapp.RefShift{ShiftID: s.id, LocationID: l, Name: s.name, StartsAt: s.from, EndsAt: s.to})
		}
		running := !s.from.After(c.T) && s.to.After(c.T)
		for _, p := range c.M.Spec.People {
			if p.Workplace == "" || shiftOf(p) != s.pattern || !slices.Contains(s.locations, workplaceWorkshop[p.Workplace]) {
				continue
			}
			a := accessapp.AccessAssignment{WorkplaceID: p.Workplace, ShiftID: s.id, PersonID: p.Person, AssigneeRole: c.M.assigneeRole(p.Person), QualificationOK: true}
			if a.AssigneeRole == accessdom.AssigneePerformer {
				_, _, found, expired := c.M.qualificationAt(p.Person, workplaceScope[p.Workplace], s.from)
				a.QualificationOK = found && !expired
			} else if ap, ok := controllerApproval[p.Workplace]; ok && !s.from.Before(c.M.clk.at(ap.day, 0, 0)) {
				a.ApprovalDocumentID = ap.doc
			}
			a.Admitted = running && c.postPresence(p.Workplace, p.Person) == "present"
			plan.Items = append(plan.Items, a)
		}
	}
	out := []loader.Response{resp("reference.shift.list", list), resp("access.assignment.list", plan)}
	for _, wp := range sortedKeys(workplaceTitle) {
		out = append(out, resp("access.candidate.list", c.M.postCandidates(wp, c.T), "workplace_id", wp))
	}
	return out
}
