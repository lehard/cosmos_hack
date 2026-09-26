package access

import (
	"context"
	"strconv"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля access (AD-36): реализации live
// (Service) и fixtures (infrastructure/fixtures/access).
type Queries interface {
	// Personas — демо-персоны экрана входа (access.persona.list).
	Personas(ctx context.Context) (DemoPersonaList, error)
	// Session — текущий сеанс (access.session.read).
	Session(ctx context.Context) (Session, error)
	// Desks — стол активной роли: normative/desks/‹роль›.yaml; у роли-наследника
	// без своего файла — стол ближайшей базовой роли (access.desk.read, AD-21).
	Desks(ctx context.Context) (Desk, error)
	// Workplaces — посты для панели «Посты» (access.workplace.list, FR-6, FR-81).
	Workplaces(ctx context.Context, workshop string, m platform.Moment) (PostList, error)
	// WorkplaceCard — карточка поста (access.workplace.read, UI-16).
	WorkplaceCard(ctx context.Context, workplaceID string, m platform.Moment) (WorkplaceCard, error)
	// WorkplaceHistory — история поста, новые сверху (access.workplace.history, UI-16).
	WorkplaceHistory(ctx context.Context, workplaceID string, m platform.Moment, p platform.Page) (WorkplaceHistory, error)
	// PersonCard — карточка сотрудника без учётной записи (access.person.card, UI-16).
	PersonCard(ctx context.Context, personID string, m platform.Moment) (PersonCard, error)
	AdminQueries
}

// Commands — ведущий порт команд модуля access.
type Commands interface {
	// OpenSession — вход (access.session.create, FR-128): субъект сеанса и токен для cookie.
	OpenSession(ctx context.Context, rq SessionCreate) (Session, string, error)
	// CloseSession — выход (access.session.delete).
	CloseSession(ctx context.Context, token string) error
	// RequestAccount — заявка на регистрацию (access.account.request, FR-128).
	RequestAccount(ctx context.Context, rq AccountRequest) (AccountRequestResult, error)
	AdminCommands
}

// Unimplemented — заглушка портов access: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации, чтобы новые операции
// контракта не ломали сборку (реализация переопределяет метод).
type Unimplemented struct{}

func (Unimplemented) Personas(context.Context) (DemoPersonaList, error) {
	return DemoPersonaList{}, platform.NotImplemented("access.persona.list")
}

func (Unimplemented) Session(context.Context) (Session, error) {
	return Session{}, platform.NotImplemented("access.session.read")
}

func (Unimplemented) Desks(context.Context) (Desk, error) {
	return Desk{}, platform.NotImplemented("access.desk.read")
}

func (Unimplemented) Workplaces(context.Context, string, platform.Moment) (PostList, error) {
	return PostList{}, platform.NotImplemented("access.workplace.list")
}

func (Unimplemented) WorkplaceCard(context.Context, string, platform.Moment) (WorkplaceCard, error) {
	return WorkplaceCard{}, platform.NotImplemented("access.workplace.read")
}

func (Unimplemented) WorkplaceHistory(context.Context, string, platform.Moment, platform.Page) (WorkplaceHistory, error) {
	return WorkplaceHistory{}, platform.NotImplemented("access.workplace.history")
}

func (Unimplemented) PersonCard(context.Context, string, platform.Moment) (PersonCard, error) {
	return PersonCard{}, platform.NotImplemented("access.person.card")
}

// PageOf — страница списка: курсор — смещение (десятичное), по умолчанию 50.
func PageOf[T any](items []T, p platform.Page) ([]T, string) {
	off, _ := strconv.Atoi(p.Cursor)
	if off < 0 || off > len(items) {
		off = len(items)
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	end := min(off+limit, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[off:end], next
}

func (Unimplemented) OpenSession(context.Context, SessionCreate) (Session, string, error) {
	return Session{}, "", platform.NotImplemented("access.session.create")
}

func (Unimplemented) CloseSession(context.Context, string) error {
	return platform.NotImplemented("access.session.delete")
}

func (Unimplemented) RequestAccount(context.Context, AccountRequest) (AccountRequestResult, error) {
	return AccountRequestResult{}, platform.NotImplemented("access.account.request")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
