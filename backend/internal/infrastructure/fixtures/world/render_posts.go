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

// postEv — событие поста мира заготовок.
type postEv struct {
	at                    time.Time
	typ, kind             string
	person, shift, reason string
	order                 int
}

// zoneOfWorkplace — зона СКУД поста: зона цеха (access_zone_id: WS-WC → Z-WC).
func zoneOfWorkplace(wp string) string {
	ws := workplaceWorkshop[wp]
	if ws == "" {
		return ""
	}
	return "Z-" + strings.TrimPrefix(ws, "WS-")
}

// lunchStory — эпик 37 (FR-84): сварщик W21 в первую смену выходит из зоны
// цеха на обед, оставив ключ на посту: «ключ вставлен, владельца нет в зоне» —
// тревога администратору, допуск снят (zone_exit); вернулся — допуск снова.
func lunchStory(p PersonRef, shift string) bool { return p.Person == "W21" && shift == "SHIFT-1" }

// postEvents — события поста по расписанию смен, по возрастанию времени: проход
// СКУД в зону цеха, назначение, ключ и допуск в начале смены, завершение, ключ
// и выход из зоны — в конце (эпик 37: присутствие видно в истории поста).
func (c *Ctx) postEvents(wp string) []postEv {
	loc := c.M.clk.loc
	zone := zoneOfWorkplace(wp)
	var evs []postEv
	t0 := c.M.Steps[0].In(loc)
	for day := time.Date(t0.Year(), t0.Month(), t0.Day()-1, 0, 0, 0, 0, loc); !day.After(c.T); day = day.AddDate(0, 0, 1) {
		for _, s := range postShifts {
			start, end := day.Add(s.start), day.Add(s.end)
			for _, p := range c.M.Spec.People {
				if p.Workplace != wp || shiftOf(p) != s.id {
					continue
				}
				zin := string(catalog.AccessZonePassed)
				evs = append(evs,
					postEv{start.Add(-20 * time.Minute), zin, "zone_in", p.Person, s.id, zone, 0},
					postEv{start.Add(-15 * time.Minute), string(catalog.AccessAssignmentSet), "assigned", p.Person, s.id, "", 1},
					postEv{start.Add(5 * time.Minute), string(catalog.AccessTokenPresenceChanged), "token_in", p.Person, s.id, "", 2},
					postEv{start.Add(5 * time.Minute), string(catalog.AccessWorkplaceAdmitted), "admitted", p.Person, s.id, "", 3})
				if lunchStory(p, s.id) {
					out, back := day.Add(12*time.Hour), day.Add(12*time.Hour+35*time.Minute)
					evs = append(evs,
						postEv{out, zin, "zone_out", p.Person, s.id, zone, 4},
						postEv{out, string(catalog.SecurityPresenceDeviation), "presence_deviation", p.Person, s.id, "token_without_presence", 5},
						postEv{out, string(catalog.AccessWorkplaceRevoked), "revoked", p.Person, s.id, "zone_exit", 6},
						postEv{back, zin, "zone_in", p.Person, s.id, zone, 7},
						postEv{back.Add(time.Minute), string(catalog.AccessWorkplaceAdmitted), "admitted", p.Person, s.id, "", 8})
				}
				evs = append(evs,
					postEv{end.Add(-5 * time.Minute), string(catalog.AccessWorkplaceReleased), "released", p.Person, s.id, "", 9},
					postEv{end.Add(-5 * time.Minute), string(catalog.AccessTokenPresenceChanged), "token_out", p.Person, s.id, "", 10},
					postEv{end.Add(5 * time.Minute), zin, "zone_out", p.Person, s.id, zone, 11})
			}
		}
	}
	slices.SortStableFunc(evs, func(a, b postEv) int {
		if x := a.at.Compare(b.at); x != 0 {
			return x
		}
		if x := strings.Compare(a.person, b.person); x != 0 {
			return x
		}
		return a.order - b.order
	})
	return evs
}

// postPresence — присутствие сотрудника на посту к часам шага по событиям
// поста (FR-6): в зоне и ключ вставлен — на месте; в зоне без ключа — ключ не
// вставлен; ключ есть, владельца нет в зоне — владельца нет; иначе — нет на месте.
func (c *Ctx) postPresence(wp, person string) string {
	known, in, token := false, false, false
	for _, e := range c.postEvents(wp) {
		if e.at.After(c.T) || e.person != person {
			continue
		}
		switch e.kind {
		case "zone_in", "zone_out":
			known, in = true, e.kind == "zone_in"
		case "token_in", "token_out":
			token = e.kind == "token_in"
		}
	}
	switch {
	case !known:
		return "unknown"
	case in && token:
		return "present"
	case in:
		return "key_missing"
	case token:
		return "owner_absent"
	}
	return "absent"
}

// postHistory — события поста за сутки до часов шага по расписанию смен, новые сверху.
func (c *Ctx) postHistory(wp string) accessapp.WorkplaceHistory {
	from := c.T.Add(-historyWindow)
	h := accessapp.WorkplaceHistory{WorkplaceID: wp, Items: []accessapp.WorkplaceEvent{}}
	// seq — порядковый номер события поста в мире заготовок (журнала у заготовок нет).
	for i, e := range c.postEvents(wp) {
		if e.at.After(c.T) || e.at.Before(from) {
			continue
		}
		h.Items = append(h.Items, accessapp.WorkplaceEvent{Seq: int64(i + 1), At: e.at.UTC(), EventType: e.typ, Kind: e.kind, PersonID: e.person,
			PersonDisplay: c.M.personName(e.person), ShiftID: e.shift, Reason: e.reason})
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
