package world

import (
	"fmt"
	"sort"
	"strings"
	"time"

	accessapp "ant/internal/application/access"
	"ant/internal/application/platform"
	securityapp "ant/internal/application/security"
	"ant/internal/infrastructure/fixtures/loader"
)

// Рендеры ответов: снимок мира на шаг n → тела операций contracts/openapi.yaml
// типами application/‹модуль› (тот же тип, что отдаёт live). Параметры ответа —
// HTTP-параметры операции (path + query) без момента и страницы; адаптеры
// infrastructure/fixtures/‹модуль› ищут ответ по тем же ключам.

// Ctx — снимок для рендера: модель, шаг, часы шага.
type Ctx struct {
	M      *Model
	N      int
	T      time.Time
	states map[*Item]ItemState
	seqIt  map[*Item]int64
	seqEnt map[string]int64
}

func (m *Model) ctx(n int) *Ctx {
	c := &Ctx{M: m, N: n, T: m.Steps[n], states: map[*Item]ItemState{}}
	return c
}

// S — состояние изделия на часах шага (кэш).
func (c *Ctx) S(it *Item) ItemState {
	if st, ok := c.states[it]; ok {
		return st
	}
	st := it.State(c.T)
	c.states[it] = st
	return st
}

// Seq — basis_seq ответов шага (AD-39): для сводок, карт и списков.
func (c *Ctx) Seq() int64 { return loader.StepSeq(c.N) }

func (c *Ctx) indexSeq() {
	if c.seqIt != nil {
		return
	}
	c.seqIt, c.seqEnt = map[*Item]int64{}, map[string]int64{}
	for _, e := range c.M.Events {
		if e.Step > c.N {
			continue
		}
		if e.Item != nil && e.Seq > c.seqIt[e.Item] {
			c.seqIt[e.Item] = e.Seq
		}
		if e.Entity.ID != "" && e.Seq > c.seqEnt[e.Entity.ID] {
			c.seqEnt[e.Entity.ID] = e.Seq
		}
	}
}

// ItemSeq — версия потока изделия: seq последней видимой записи изделия (AD-39).
func (c *Ctx) ItemSeq(it *Item) int64 {
	c.indexSeq()
	if s := c.seqIt[it]; s > 0 {
		return s
	}
	return 1
}

// EntitySeq — версия потока объекта вне изделия (несоответствие, инцидент, сообщение 1С).
func (c *Ctx) EntitySeq(id string, items ...*Item) int64 {
	c.indexSeq()
	s := c.seqEnt[id]
	for _, it := range items {
		if x := c.seqIt[it]; x > s {
			s = x
		}
	}
	if s == 0 {
		return 1
	}
	return s
}

// Existing — изделия, запущенные к часам шага.
func (c *Ctx) Existing() []*Item {
	var out []*Item
	for _, it := range c.M.Items {
		if c.S(it).Exists {
			out = append(out, it)
		}
	}
	return out
}

// Visible — записи журнала, известные к шагу (порядок знания, AD-37).
func (c *Ctx) Visible() []*Event {
	var out []*Event
	for _, e := range c.M.Events {
		if e.Step <= c.N {
			out = append(out, e)
		}
	}
	return out
}

func resp(op string, body any, kv ...string) loader.Response {
	r := loader.Response{Op: op, Body: body}
	if len(kv) > 1 {
		r.Params = map[string]string{}
		for i := 0; i+1 < len(kv); i += 2 {
			r.Params[kv[i]] = kv[i+1]
		}
	}
	return r
}

func ptr[T any](v T) *T { return &v }

func tptr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}

// renderers — рендеры всех модулей по шагу.
var renderers = []func(c *Ctx) []loader.Response{
	renderSecurity, renderWorkplaces,
	renderItems, renderProcess, renderQuality, renderNonconformity, renderAnalysis, renderSuggestions,
	renderAnalytics, renderNotifications, renderERP, renderJournal, renderMachinelogs,
	renderVision, renderIngest, renderOps, renderSimulation, renderMaterials,
}

// Render — все ответы шага n.
func (m *Model) Render(n int) []loader.Response {
	c := m.ctx(n)
	var out []loader.Response
	for _, r := range renderers {
		out = append(out, r(c)...)
	}
	return out
}

// Changes — изменения сущностей шага n (сообщения SSE): изделия и объекты его записей.
func (m *Model) Changes(n int) []loader.Change {
	seen := map[string]bool{}
	var out []loader.Change
	add := func(kind, id string) {
		k := kind + "/" + id
		if !seen[k] {
			seen[k] = true
			out = append(out, loader.Change{Entity: kind, ID: id})
		}
	}
	for _, e := range m.Events {
		if e.Step != n {
			continue
		}
		if e.Item != nil {
			add(string(platform.EntityItem), FullID(e.Item.ID))
		}
		if e.Entity.Entity != "" {
			add(e.Entity.Entity, e.Entity.ID)
		}
		if strings.HasPrefix(e.Type, "task.") || strings.HasPrefix(e.Type, "decision.containment") {
			add(string(platform.EntityTask), "global")
		}
	}
	add(string(platform.EntityNotification), "global")
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Entity != out[j].Entity {
			return out[i].Entity < out[j].Entity
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// ─────────────────────────────── общие ───────────────────────────────

// Common — ответы, общие для сценариев: демо-персоны, сеансы, столы ролей (access).
func Common(pol *Policy, people []PersonRef) ([]loader.Response, error) {
	var out []loader.Response
	personas := accessapp.DemoPersonaList{Items: []accessapp.DemoPersona{}}
	roles := map[string]bool{}
	for _, pr := range people {
		p, ok := pol.Person(pr.Person)
		if !ok || len(p.Roles) == 0 {
			return nil, fmt.Errorf("демо-персона %s: нет в normative/policy", pr.Person)
		}
		role := roleRef(pol, p.Roles[0].Role)
		personas.Items = append(personas.Items, accessapp.DemoPersona{ID: p.ID, Name: p.Name, Role: role, Scope: p.Roles[0].Scope})
		s := accessapp.Session{User: accessapp.SessionUser{ID: p.ID, Name: p.Name}, Role: role, Scope: p.Roles[0].Scope, Demo: true}
		if pr.Shift != "" {
			s.Shift = &accessapp.SessionShift{ID: pr.Shift, Title: map[string]string{"SHIFT-1": "Первая смена 08:00–16:30", "SHIFT-2": "Вторая смена 16:30–01:00"}[pr.Shift]}
		}
		if pr.Workplace != "" {
			s.Workplace = &accessapp.SessionWorkplace{ID: pr.Workplace, Title: workplaceTitle[pr.Workplace]}
		}
		out = append(out, resp("access.session.read", s, "persona", p.ID))
		roles[role.ID] = true
		for _, r := range p.Roles {
			roles[r.Role] = true
		}
	}
	out = append([]loader.Response{resp("access.persona.list", personas)}, out...)
	for _, r := range pol.Roles {
		roles[r.ID] = true
	}
	for _, id := range sortedKeys(roles) {
		d, ok := pol.DeskFor(id)
		if !ok {
			continue
		}
		body := map[string]any{}
		for k, v := range d {
			body[k] = v
		}
		body["role"] = id
		out = append(out, resp("access.desk.read", body, "role", id))
	}
	return out, nil
}

func roleRef(pol *Policy, id string) accessapp.RoleRef {
	r, _ := pol.Role(id)
	return accessapp.RoleRef{ID: id, Title: r.Title, Inherits: r.Inherits}
}

var workplaceTitle = map[string]string{
	"WP-VK-1": "Пост входного контроля 1", "WP-CNC-1": "Пост ЧПУ 1", "WP-CMM-1": "Пост КИМ 1", "WP-QC-MC": "Пост ОТК механического цеха",
	"WP-WELD-1": "Пост сварки 1 (источник ИС-1)", "WP-WELD-2": "Пост сварки 2 (источник ИС-2)", "WP-QC-WC": "Пост ОТК сварочного цеха",
	"WP-ASM-1": "Пост сборки 1", "WP-LEAK-1": "Стенд герметичности", "WP-QC-AC": "Пост ОТК сборочно-испытательного цеха", "WP-FINAL-1": "Пост окончательного контроля",
}

// workshopName — имя цеха справочника мест (normative/reference/flange/locations.yaml).
var workshopName = map[string]string{
	"WS-SK": "Склад и входной контроль", "WS-MC": "Механический цех", "WS-WC": "Сварочный цех", "WS-AC": "Сборочно-испытательный цех", "WS-QA": "ОТК и выпуск",
}

var workplaceWorkshop = map[string]string{
	"WP-VK-1": "WS-SK", "WP-CNC-1": "WS-MC", "WP-CMM-1": "WS-MC", "WP-QC-MC": "WS-MC", "WP-WELD-1": "WS-WC", "WP-WELD-2": "WS-WC",
	"WP-QC-WC": "WS-WC", "WP-ASM-1": "WS-AC", "WP-LEAK-1": "WS-AC", "WP-QC-AC": "WS-AC", "WP-FINAL-1": "WS-QA",
}

// renderSecurity — индикатор целостности «по данным сервера» (AD-46, S09).
func renderSecurity(c *Ctx) []loader.Response {
	iv := c.M.Spec.Integrity.IntervalMinutes
	if iv == 0 {
		iv = 15
	}
	checked := c.T.Truncate(time.Duration(iv) * time.Minute)
	st := securityapp.IntegrityStatus{Status: "ok", CheckedAt: tptr(checked), IntervalSeconds: iv * 60, ServerSide: true}
	for _, v := range c.M.Spec.Integrity.Violations {
		if !v.At.Time().After(c.T) {
			st.Status = "violated"
			st.CheckedAt = tptr(v.At.Time())
		}
	}
	st.ReportRef = Digest([]byte(fmt.Sprintf("verifier-report/%s/%s", st.Status, st.CheckedAt.Format(time.RFC3339))))
	return []loader.Response{resp("security.integrity.read", st)}
}

// renderWorkplaces — панель «Посты» (FR-6): кто назначен, на месте ли, текущее изделие.
func renderWorkplaces(c *Ctx) []loader.Response {
	h := c.T.In(c.M.clk.loc)
	dayShift := h.Hour() >= 8 && (h.Hour() < 16 || (h.Hour() == 16 && h.Minute() < 30))
	assigned := map[string]string{}
	for _, p := range c.M.Spec.People {
		if p.Workplace == "" {
			continue
		}
		if (p.Shift == "SHIFT-1") == dayShift || p.Shift == "" {
			if _, ok := assigned[p.Workplace]; !ok || p.Person == "W21" || p.Person == "W22" {
				assigned[p.Workplace] = p.Person
			}
		}
	}
	current := map[string]*Item{}
	for _, it := range c.Existing() {
		st := c.S(it)
		if st.Position == "in_progress" && workplaceTitle[st.Location] != "" {
			current[st.Location] = it
		}
	}
	list := accessapp.PostList{Items: []accessapp.PostRow{}}
	for _, wp := range sortedKeys(workplaceTitle) {
		row := accessapp.PostRow{WorkplaceID: wp, Station: workplaceTitle[wp], Workshop: workplaceWorkshop[wp], WorkshopName: workshopName[workplaceWorkshop[wp]], Presence: "not_assigned"}
		if p, ok := assigned[wp]; ok {
			row.Assigned = &accessapp.PostPerson{PersonID: p, Display: c.M.personName(p)}
			row.Presence = "present"
		}
		if it := current[wp]; it != nil {
			row.CurrentItem = &accessapp.PostItem{ItemID: FullID(it.ID), Label: it.Label}
		}
		list.Items = append(list.Items, row)
	}
	out := []loader.Response{resp("access.workplace.list", list)}
	return append(out, renderPostCards(c, list, dayShift)...)
}
