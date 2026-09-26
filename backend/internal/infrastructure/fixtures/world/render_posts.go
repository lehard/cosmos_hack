package world

import (
	"slices"
	"strings"
	"time"

	accessapp "ant/internal/application/access"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/fixtures/loader"
)

// Карточки окон «Пост» и «Сотрудник» раздела «Посты» (UI-16): карточка поста
// с назначениями текущей смены, история поста за сутки и карточка сотрудника.
// Назначения — из people мира (пост и смена), история — по расписанию смен
// (назначение, допуск и токен в начале смены, токен и завершение — в конце).

// workplaceScope — область поста (normative/reference/flange/locations.yaml).
var workplaceScope = map[string]string{
	"WP-VK-1": "ent01/b1/sk/vk/wp1", "WP-CNC-1": "ent01/b1/mc/cnc/wp1", "WP-CMM-1": "ent01/b1/mc/cmm/wp1", "WP-QC-MC": "ent01/b1/mc/qc",
	"WP-WELD-1": "ent01/b1/wc/weld/wp1", "WP-WELD-2": "ent01/b1/wc/weld/wp2", "WP-QC-WC": "ent01/b1/wc/qc",
	"WP-ASM-1": "ent01/b1/ac/asm/wp1", "WP-LEAK-1": "ent01/b1/ac/leak/wp1", "WP-QC-AC": "ent01/b1/ac/qc", "WP-FINAL-1": "ent01/b1/qa/final/wp1",
}

// historyWindow — глубина истории поста в заготовках (размер файлов шагов).
const historyWindow = 24 * time.Hour

// postShift — смена людей мира: SHIFT-1 08:00–16:30, SHIFT-2 16:30–01:00.
type postShift struct {
	id         string
	start, end time.Duration // от начала суток
}

var postShifts = []postShift{{"SHIFT-1", 8 * time.Hour, 16*time.Hour + 30*time.Minute}, {"SHIFT-2", 16*time.Hour + 30*time.Minute, 25 * time.Hour}}

// shiftOf — смена человека мира: без смены — первая.
func shiftOf(p PersonRef) string {
	if p.Shift == "" {
		return "SHIFT-1"
	}
	return p.Shift
}

// assigneeRole — роль назначения: контролёр ОТК или исполнитель.
func (m *Model) assigneeRole(person string) string {
	if m.policy != nil {
		if p, ok := m.policy.Person(person); ok {
			for _, r := range p.Roles {
				if r.Role == "quality_inspector" || r.Role == "head_of_qc" {
					return "quality_inspector"
				}
			}
		}
	}
	return "performer"
}

func renderPostCards(c *Ctx, list accessapp.PostList, dayShift bool) []loader.Response {
	shift := "SHIFT-2"
	if dayShift {
		shift = "SHIFT-1"
	}
	var out []loader.Response
	posts := map[string][]accessapp.PersonPost{}
	for _, row := range list.Items {
		card := accessapp.WorkplaceCard{PostRow: row, Scope: workplaceScope[row.WorkplaceID], Assignments: []accessapp.WorkplaceAssignee{}}
		for _, p := range c.M.Spec.People {
			if p.Workplace != row.WorkplaceID || shiftOf(p) != shift {
				continue
			}
			role := c.M.assigneeRole(p.Person)
			card.ShiftID = shift
			card.Assignments = append(card.Assignments, accessapp.WorkplaceAssignee{PersonID: p.Person, PersonDisplay: c.M.personName(p.Person), ShiftID: shift,
				AssigneeRole: role, QualificationOK: true})
			posts[p.Person] = append(posts[p.Person], accessapp.PersonPost{WorkplaceID: row.WorkplaceID, Station: row.Station, ShiftID: shift, AssigneeRole: role})
		}
		out = append(out, resp("access.workplace.read", card, "workplace_id", row.WorkplaceID))
		out = append(out, resp("access.workplace.history", c.postHistory(row.WorkplaceID), "workplace_id", row.WorkplaceID))
	}
	if c.M.policy != nil {
		for _, p := range c.M.policy.Persons {
			out = append(out, resp("access.person.card", c.personCard(p, posts[p.ID]), "person_id", p.ID))
		}
	}
	return out
}

// postHistory — события поста за сутки до часов шага по расписанию смен, новые сверху.
func (c *Ctx) postHistory(wp string) accessapp.WorkplaceHistory {
	loc := c.M.clk.loc
	from := c.T.Add(-historyWindow)
	h := accessapp.WorkplaceHistory{WorkplaceID: wp, Items: []accessapp.WorkplaceEvent{}}
	type ev struct {
		at            time.Time
		typ, kind     string
		person, shift string
		order         int
	}
	var evs []ev
	t0 := c.M.Steps[0].In(loc)
	for day := time.Date(t0.Year(), t0.Month(), t0.Day()-1, 0, 0, 0, 0, loc); !day.After(c.T); day = day.AddDate(0, 0, 1) {
		for _, s := range postShifts {
			start, end := day.Add(s.start), day.Add(s.end)
			for _, p := range c.M.Spec.People {
				if p.Workplace != wp || shiftOf(p) != s.id {
					continue
				}
				evs = append(evs,
					ev{start.Add(-15 * time.Minute), string(catalog.AccessAssignmentSet), "assigned", p.Person, s.id, 0},
					ev{start.Add(5 * time.Minute), string(catalog.AccessTokenPresenceChanged), "token_in", p.Person, s.id, 1},
					ev{start.Add(5 * time.Minute), string(catalog.AccessWorkplaceAdmitted), "admitted", p.Person, s.id, 2},
					ev{end.Add(-5 * time.Minute), string(catalog.AccessWorkplaceReleased), "released", p.Person, s.id, 3},
					ev{end.Add(-5 * time.Minute), string(catalog.AccessTokenPresenceChanged), "token_out", p.Person, s.id, 4})
			}
		}
	}
	slices.SortStableFunc(evs, func(a, b ev) int {
		if x := a.at.Compare(b.at); x != 0 {
			return x
		}
		if x := strings.Compare(a.person, b.person); x != 0 {
			return x
		}
		return a.order - b.order
	})
	// seq — порядковый номер события поста в мире заготовок (журнала у заготовок нет).
	for i, e := range evs {
		if e.at.After(c.T) || e.at.Before(from) {
			continue
		}
		h.Items = append(h.Items, accessapp.WorkplaceEvent{Seq: int64(i + 1), At: e.at.UTC(), EventType: e.typ, Kind: e.kind, PersonID: e.person,
			PersonDisplay: c.M.personName(e.person), ShiftID: e.shift})
	}
	slices.Reverse(h.Items)
	return h
}

// personCard — карточка сотрудника: роли и квалификации из затравки политики, посты текущей смены.
func (c *Ctx) personCard(p PolicyPerson, posts []accessapp.PersonPost) accessapp.PersonCard {
	since := time.Date(c.M.Steps[0].Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	card := accessapp.PersonCard{PersonID: p.ID, DisplayName: p.Name, Roles: []accessapp.AccessRoleGrant{}, Qualifications: []accessapp.AccessQualification{},
		Posts: []accessapp.PersonPost{}, PolicySeq: 1}
	for _, r := range p.Roles {
		card.Roles = append(card.Roles, accessapp.AccessRoleGrant{RoleID: r.Role, Scope: r.Scope, ValidFrom: since})
	}
	for _, q := range c.M.policy.Grants.Qualifications {
		if q.Person != p.ID {
			continue
		}
		v := accessapp.AccessQualification{PersonID: p.ID, QualificationID: q.Qualification, Scope: q.Scope, CertificateRef: q.CertificateRef, ValidFrom: since, Status: "valid"}
		if u, err := time.Parse(time.RFC3339, q.ValidUntil); err == nil {
			v.ValidUntil = &u
			switch {
			case !u.After(c.T):
				v.Status = "expired"
			case u.Sub(c.T) < 30*24*time.Hour:
				v.Status = "expiring"
			}
		}
		card.Qualifications = append(card.Qualifications, v)
	}
	if posts != nil {
		card.Posts = posts
	}
	return card
}
