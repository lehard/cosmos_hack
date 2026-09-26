package access

import (
	"context"
	"slices"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	accessdom "ant/internal/domain/access"
)

// Объяснение прав своим кодом (AD-15: встроенный Explain Casbin ходит во
// внешний ИИ и запрещён; FR-136, FR-146): решение — у того же вычислителя, что
// проверяет команды; объяснение — по политике: какая роль (с наследованием) и
// в какой области разрешает действие, почему не разрешает, кто может, ваши
// полномочия и клейма и что можно сделать вместо («Запросить решение»).

// RequestDecisionActions — операции «Запросить решение» (FR-146): документ с
// маршрутом подписей вместо недоступного действия (documents).
var RequestDecisionActions = []string{"documents.version.request", "documents.document.request"}

// Explanation — «почему вы можете / не можете» (FR-136, FR-146): решение по
// действию над объектом и что можно сделать вместо.
type Explanation struct {
	Action         string   `json:"action" doc:"x-ant-action id."`
	Allowed        bool     `json:"allowed"`
	Code           string   `json:"code,omitempty" doc:"Код отказа из contracts/errors.yaml."`
	Reason         string   `json:"reason,omitempty" doc:"Объяснение по-русски: роль, область, полномочие, клеймо, разделение обязанностей."`
	AllowedActions []string `json:"allowed_actions" doc:"Что можно сделать вместо (например, «Запросить решение»)."`
	// Эпик 26: объяснение своим кодом над политикой.
	Place       string             `json:"place,omitempty" doc:"Место операции (область), по которому принято решение (барьер 3)."`
	GrantedBy   *ExplainRole       `json:"granted_by,omitempty" doc:"Какая ваша роль разрешает действие: назначенная роль, базовая роль с этим действием, область."`
	Roles       []ExplainRole      `json:"roles,omitempty" doc:"Ваши действующие роли в областях."`
	Authorities []ExplainAuthority `json:"authorities,omitempty" doc:"Ваши действующие полномочия — рамки доменных гардов (FR-50, FR-78)."`
	Stamps      []ExplainStamp     `json:"stamps,omitempty" doc:"Ваши действующие цифровые клейма (FR-145)."`
	WhoCan      []ExplainHolder    `json:"who_can,omitempty" doc:"Кто может выполнить действие: роли политики и сотрудники с ними."`
	CardOnly    bool               `json:"card_only,omitempty" doc:"Вы — редкий подписант: доступны только адресованные вам карточки решения (FR-136)."`
	PolicySeq   int64              `json:"policy_seq,omitempty" doc:"Версия политики объяснения (AD-39)."`
}

// ExplainRole — роль в области: назначенная роль и, для granted_by, базовая
// роль, в которой объявлено действие (наследование — одна функция, AD-15).
type ExplainRole struct {
	RoleID     string     `json:"role_id"`
	Title      string     `json:"title"`
	Scope      string     `json:"scope" doc:"Область назначения."`
	Via        string     `json:"via,omitempty" doc:"Базовая роль, в которой объявлено действие."`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
}

// ExplainAuthority — полномочие с областью и сроком.
type ExplainAuthority struct {
	AuthorityID string     `json:"authority_id"`
	Title       string     `json:"title"`
	Scope       string     `json:"scope"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
}

// ExplainStamp — цифровое клеймо.
type ExplainStamp struct {
	StampID        string `json:"stamp_id"`
	InspectionKind string `json:"inspection_kind"`
	Scope          string `json:"scope"`
}

// ExplainHolder — роль, в которой разрешено действие, и сотрудники с ней.
type ExplainHolder struct {
	RoleID  string   `json:"role_id"`
	Title   string   `json:"title"`
	Persons []string `json:"persons" doc:"Сотрудники с этой ролью (псевдонимы, не больше 20)."`
}

// maxHolders — сотрудников в строке «кто может».
const maxHolders = 20

// Explain — объяснение прав по действию над объектом.
func (g *Gate) Explain(ctx context.Context, p platform.Principal, actionID string, obj platform.ObjectRef) (Explanation, error) {
	var act platform.Action
	found := false
	if g.actions != nil {
		for _, a := range g.actions() {
			if a.ID == actionID {
				act, found = a, true
				break
			}
		}
	}
	if !found {
		return Explanation{}, platform.Fail(errcodes.ApiNotFound, "object", "операция", "id", actionID)
	}
	rq := Request{Principal: p, Action: act, Object: obj, Scope: g.Place(p, obj, nil), At: g.now()}
	d, err := g.decide(ctx, rq)
	if err != nil {
		return Explanation{}, err
	}
	ex := Explanation{Action: actionID, Allowed: d.Allowed, Code: string(d.Code), Reason: d.Reason, AllowedActions: d.AllowedActions, Place: rq.Scope}
	if !d.Allowed {
		e := g.refusal(ctx, rq, d)
		ex.Code, ex.Reason = string(e.Code), e.Detail
	}
	if ex.AllowedActions == nil {
		ex.AllowedActions = []string{}
	}
	if g.Policy != nil && !p.Anonymous() {
		pol, err := g.Policy.Policy(ctx)
		if err != nil {
			return Explanation{}, err
		}
		g.explainPolicy(pol, &ex, rq)
	}
	if !ex.Allowed {
		// «Запросить решение» (FR-146): вместо недоступного действия — документ с маршрутом.
		for _, id := range RequestDecisionActions {
			if slices.Contains(ex.AllowedActions, id) {
				continue
			}
			if ra, ok := g.action(id); ok {
				if dd, err := g.ac.Enforce(ctx, Request{Principal: p, Action: ra, At: rq.At}); err == nil && dd.Allowed {
					ex.AllowedActions = append(ex.AllowedActions, id)
				}
			}
		}
	}
	return ex, nil
}

// action — операция каталога по id.
func (g *Gate) action(id string) (platform.Action, bool) {
	if g.actions == nil {
		return platform.Action{}, false
	}
	for _, a := range g.actions() {
		if a.ID == id {
			return a, true
		}
	}
	return platform.Action{}, false
}

// explainPolicy — роли, полномочия, клейма субъекта, какая роль разрешает
// действие и кто может, по действующей политике.
func (g *Gate) explainPolicy(pol accessdom.Policy, ex *Explanation, rq Request) {
	person, at := rq.Principal.PersonID, rq.At
	h := pol.Hierarchy()
	ex.PolicySeq = pol.Seq
	ex.CardOnly = pol.CardOnly(person, at)
	alias := accessdom.ReadAlias(rq.Action.ID, string(rq.Action.Class))
	matches := func(rid string) bool {
		r, ok := pol.Role(rid)
		if !ok {
			return false
		}
		for _, pat := range r.Actions {
			if accessdom.ActionMatches(rq.Action.ID, pat) || (alias != "" && accessdom.ActionMatches(alias, pat)) {
				return true
			}
		}
		return false
	}
	var wrongPlace *ExplainRole
	for _, a := range pol.AssignmentsOf(person, at) {
		er := ExplainRole{RoleID: a.RoleID, Title: roleTitle(pol, a.RoleID), Scope: a.Scope, ValidUntil: until(a.ValidUntil)}
		ex.Roles = append(ex.Roles, er)
		for _, rid := range h.Closure(a.RoleID) {
			if !matches(rid) {
				continue
			}
			x := er
			if rid != a.RoleID {
				x.Via = rid
			}
			if rq.Scope == "" || accessdom.ScopeCovers(a.Scope, rq.Scope) {
				if ex.GrantedBy == nil {
					ex.GrantedBy = &x
				}
			} else if wrongPlace == nil {
				wrongPlace = &x
			}
			break
		}
	}
	for _, a := range pol.AuthoritiesOf(person, at) {
		t := a.AuthorityID
		if d, ok := pol.Catalog.Authority(a.AuthorityID); ok && d.Title != "" {
			t = d.Title
		}
		ex.Authorities = append(ex.Authorities, ExplainAuthority{AuthorityID: a.AuthorityID, Title: t, Scope: a.Scope, ValidUntil: until(a.ValidUntil)})
	}
	for _, s := range pol.StampsOf(person, at) {
		ex.Stamps = append(ex.Stamps, ExplainStamp{StampID: s.StampID, InspectionKind: s.Kind, Scope: s.Scope})
	}
	for _, r := range pol.Roles {
		if !matches(r.ID) {
			continue
		}
		hd := ExplainHolder{RoleID: r.ID, Title: r.Title, Persons: []string{}}
		for _, a := range pol.Assignments {
			if len(hd.Persons) < maxHolders && a.ActiveAt(at) && h.Covers(a.RoleID, r.ID) && !slices.Contains(hd.Persons, a.PersonID) &&
				(rq.Scope == "" || accessdom.ScopeCovers(a.Scope, rq.Scope)) {
				hd.Persons = append(hd.Persons, a.PersonID)
			}
		}
		ex.WhoCan = append(ex.WhoCan, hd)
	}
	if ex.Code != "" && ex.Code != string(errcodes.AccessForbidden) && ex.Code != string(errcodes.AccessWrongWorkplace) {
		return // особый отказ (карточка, гард) — его объяснение уже дано
	}
	switch {
	case ex.Allowed && ex.GrantedBy != nil:
		ex.Reason = "Разрешено ролью «" + ex.GrantedBy.Title + "» в области " + scopeText(ex.GrantedBy.Scope)
		if ex.GrantedBy.Via != "" {
			ex.Reason += " (действие объявлено в базовой роли «" + roleTitle(pol, ex.GrantedBy.Via) + "»)"
		}
	case ex.Allowed:
		ex.Reason = "Разрешено политикой"
	case wrongPlace != nil:
		ex.Reason = "Роль «" + wrongPlace.Title + "» разрешает действие только в области " + scopeText(wrongPlace.Scope) +
			", а место операции — " + scopeText(rq.Scope)
	default:
		var names []string
		for _, r := range ex.Roles {
			names = append(names, "«"+r.Title+"»")
		}
		mine := "у вас нет действующих ролей"
		if len(names) > 0 {
			mine = "ваши роли " + strings.Join(names, ", ") + " его не включают"
		}
		var who []string
		for _, w := range ex.WhoCan {
			who = append(who, "«"+w.Title+"»")
		}
		ex.Reason = "Действие " + rq.Action.ID + " не разрешено: " + mine
		if len(who) > 0 {
			ex.Reason += "; его выполняют роли " + strings.Join(who, ", ")
		}
	}
}

func roleTitle(pol accessdom.Policy, id string) string {
	if r, ok := pol.Role(id); ok && r.Title != "" {
		return r.Title
	}
	return id
}

func scopeText(s string) string {
	if s == "" || s == "*" {
		return "всего предприятия"
	}
	return s
}

func until(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
