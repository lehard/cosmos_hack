package access

import (
	"context"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// Service — реализация live ведущих портов модуля access (AD-36).
//
// Демо-трек эпика 08 (заметка эпика): вход демо-персоной без пароля, сеанс,
// стол роли и демо-персоны — живые, над портом IdentityProvider и каталогом
// политики (Directory, normative/policy + normative/desks). Остальные операции
// (посты, администрирование доступа) — у запасной реализации: в режиме
// fixtures это адаптер заготовок, в live — Unimplemented (501) до эпиков 08,
// 26, 37. Права решает общий декоратор (Gate), не Service.
type Service struct {
	// Queries, Commands — запасная реализация операций, которых Service сам не ведёт.
	Queries
	Commands

	idp IdentityProvider
	dir *Directory
}

// Option — настройка Service.
type Option func(*Service)

// WithIdentity подключает порт входа (барьер 1, AD-15).
func WithIdentity(idp IdentityProvider) Option { return func(s *Service) { s.idp = idp } }

// WithDirectory подключает каталог политики: роли, демо-персоны, столы ролей.
func WithDirectory(d *Directory) Option { return func(s *Service) { s.dir = d } }

// WithFallback задаёт реализацию операций, которых Service сам не ведёт
// (посты, администрирование): адаптер заготовок в режиме fixtures.
func WithFallback(q Queries, c Commands) Option {
	return func(s *Service) { s.Queries, s.Commands = q, c }
}

// NewService создаёт реализацию live. Без WithIdentity и WithDirectory
// операции входа и стола отвечают 501 (выгрузка OpenAPI, тесты).
func NewService(opts ...Option) *Service {
	s := &Service{Queries: Unimplemented{}, Commands: Unimplemented{}}
	for _, o := range opts {
		o(s)
	}
	return s
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// Personas — демо-персоны экрана входа (access.persona.list, FR-128); вне
// профилей fixtures и demo порт входа отвечает api.not_found.
func (s *Service) Personas(ctx context.Context) (DemoPersonaList, error) {
	if s.idp == nil {
		return DemoPersonaList{}, platform.NotImplemented("access.persona.list")
	}
	items, err := s.idp.DemoPersonas(ctx)
	if err != nil {
		return DemoPersonaList{}, err
	}
	if items == nil {
		items = []DemoPersona{}
	}
	return DemoPersonaList{Items: items}, nil
}

// Session — сеанс субъекта запроса (access.session.read, FR-128): субъекта
// определил IdentityProvider по cookie сеанса; нет сеанса — 401.
func (s *Service) Session(ctx context.Context) (Session, error) {
	if s.dir == nil {
		return Session{}, platform.NotImplemented("access.session.read")
	}
	p := platform.PrincipalFrom(ctx)
	if p.Anonymous() {
		return Session{}, platform.Fail(errcodes.AccessUnauthenticated)
	}
	return s.sessionOf(p), nil
}

// Desks — стол активной роли субъекта (access.desk.read, AD-21): файл
// normative/desks/‹роль›.yaml; у наследника без своего файла — стол ближайшей
// базовой роли. Поле role — активная роль субъекта.
func (s *Service) Desks(ctx context.Context) (Desk, error) {
	if s.dir == nil {
		return Desk{}, platform.NotImplemented("access.desk.read")
	}
	p := platform.PrincipalFrom(ctx)
	if p.Anonymous() {
		return Desk{}, platform.Fail(errcodes.AccessUnauthenticated)
	}
	d, ok := s.dir.DeskFor(p.Role)
	if !ok {
		return Desk{}, platform.Fail(errcodes.ApiNotFound, "object", "стол роли", "id", p.Role)
	}
	d.Role = p.Role
	return d, nil
}

// OpenSession — вход (access.session.create, FR-128): субъект и токен сеанса
// для cookie от IdentityProvider.
func (s *Service) OpenSession(ctx context.Context, rq SessionCreate) (Session, string, error) {
	if s.idp == nil || s.dir == nil {
		return Session{}, "", platform.NotImplemented("access.session.create")
	}
	p, token, err := s.idp.Open(ctx, rq)
	if err != nil {
		return Session{}, "", err
	}
	return s.sessionOf(p), token, nil
}

// CloseSession — выход (access.session.delete).
func (s *Service) CloseSession(ctx context.Context, token string) error {
	if s.idp == nil {
		return platform.NotImplemented("access.session.delete")
	}
	return s.idp.Close(ctx, token)
}

// sessionOf — представление сеанса субъекта: роль с названием и наследованием из политики.
func (s *Service) sessionOf(p platform.Principal) Session {
	role, ok := s.dir.Role(p.Role)
	if !ok {
		role = RoleRef{ID: p.Role, Title: p.Role}
	}
	return Session{
		User:      SessionUser{ID: p.PersonID, Name: p.Name},
		Role:      role,
		Scope:     p.Scope,
		PolicySeq: p.PolicySeq,
		Demo:      p.Demo,
	}
}
