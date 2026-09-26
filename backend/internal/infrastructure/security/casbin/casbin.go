package casbin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	cb "github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"

	"ant/internal/application/access"
	accessdom "ant/internal/domain/access"
)

// Model — модель Casbin (AD-15): запрос — субъект, домен «место|момент»,
// действие и его второе имя для класса чтения (`‹модуль›.*.read`).
const Model = `[request_definition]
r = sub, dom, act, alias

[policy_definition]
p = role, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.role, r.dom) && (actionMatch(r.act, p.act) || actionMatch(r.alias, p.act))
`

// AccessControl — вычислитель прав над проекцией политики: при смене версии
// политики (policy_seq) Casbin перезагружает политику из снимка своим
// адаптером; решения — только Enforce Casbin.
type AccessControl struct {
	src access.PolicySource

	mu     sync.RWMutex
	e      *cb.Enforcer
	seq    int64
	loaded bool
}

// New — вычислитель над источником политики src.
func New(src access.PolicySource) *AccessControl { return &AccessControl{src: src} }

var _ access.AccessControl = (*AccessControl)(nil)

// Enforce — решение по запросу «кто — что — над чем — где — когда» (барьер 3, FR-85).
func (a *AccessControl) Enforce(ctx context.Context, rq access.Request) (access.Decision, error) {
	e, err := a.enforcer(ctx)
	if err != nil {
		return access.Decision{}, err
	}
	sub := rq.Principal.PersonID
	if sub == "" {
		sub = accessdom.AnonymousSubject
	}
	dom := accessdom.RequestPlace(rq.Scope, rq.At)
	ok, err := e.Enforce(sub, dom, rq.Action.ID, accessdom.ReadAlias(rq.Action.ID, string(rq.Action.Class)))
	if err != nil {
		return access.Decision{}, fmt.Errorf("casbin: %w", err)
	}
	return access.Decision{Allowed: ok}, nil
}

// PolicySeq — версия политики, загруженной в вычислитель (AD-39).
func (a *AccessControl) PolicySeq(ctx context.Context) (int64, error) {
	if _, err := a.enforcer(ctx); err != nil {
		return 0, err
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.seq, nil
}

// enforcer — Casbin для действующей версии политики.
func (a *AccessControl) enforcer(ctx context.Context) (*cb.Enforcer, error) {
	pol, err := a.src.Policy(ctx)
	if err != nil {
		return nil, err
	}
	a.mu.RLock()
	e, fresh := a.e, a.loaded && a.seq == pol.Seq
	a.mu.RUnlock()
	if fresh {
		return e, nil
	}
	e, err = NewEnforcer(pol)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.e, a.seq, a.loaded = e, pol.Seq, true
	a.mu.Unlock()
	return e, nil
}

// NewEnforcer — Casbin с моделью Model, функциями домена и действий и
// политикой pol, загруженной адаптером Adapter.
func NewEnforcer(pol accessdom.Policy) (*cb.Enforcer, error) {
	m, err := model.NewModelFromString(Model)
	if err != nil {
		return nil, err
	}
	e, err := cb.NewEnforcer(m, Adapter{Policy: pol})
	if err != nil {
		return nil, err
	}
	e.AddFunction("actionMatch", actionMatch)
	// Одна функция сопоставления доменов: путь области и срок полномочия (AD-15).
	e.AddNamedDomainMatchingFunc("g", "placeMatch", accessdom.PlaceMatch)
	if err := e.BuildRoleLinks(); err != nil {
		return nil, err
	}
	return e, nil
}

// actionMatch — функция матчера: действие запроса подходит под шаблон роли.
func actionMatch(args ...any) (any, error) {
	if len(args) != 2 {
		return false, errors.New("actionMatch: нужны два аргумента")
	}
	id, _ := args[0].(string)
	pattern, _ := args[1].(string)
	return accessdom.ActionMatches(id, pattern), nil
}
