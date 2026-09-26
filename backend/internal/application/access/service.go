package access

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	itemapp "ant/internal/application/item"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	accessdom "ant/internal/domain/access"
)

// Service — реализация live ведущих портов модуля access (AD-36).
//
// Живые в обоих режимах: вход (демо-персоной или по логину и паролю), сеанс,
// стол роли, демо-персоны, заявка на регистрацию; при подключённом журнале —
// сотрудники, роли, заведение сотрудника и активация учётной записи с
// назначением роли (FR-78, FR-128). Остальные операции (посты, клейма,
// квалификации, допуск) — у запасной реализации: в режиме fixtures это адаптер
// заготовок, в live — Unimplemented (501) до эпиков 26, 37. Права решает
// общий декоратор (Gate), не Service.
type Service struct {
	// Queries, Commands — запасная реализация операций, которых Service сам не ведёт.
	Queries
	Commands

	idp       IdentityProvider
	dir       *Directory
	policy    PolicySource
	creds     CredentialStore
	hasher    PasswordHasher
	decisions DecisionWriter
	// grants — документы выдачи прав (маршрут подписей documents, эпик 26).
	grants GrantDocuments
	// live — модуль access в режиме live (панель «Посты», факты исполнителя);
	// facts — запись фактов исполнителя в журнал.
	live  bool
	facts itemapp.Writer
	// wplog — история поста из журнала (access.workplace.history, UI-16).
	wplog WorkplaceLog
	// presence, pw, shifts — присутствие по СКУД и ключу, запись фактов СКУД
	// и отклонений, график смен (эпик 37).
	presence PresenceSource
	pw       PresenceWriter
	shifts   ShiftSchedule
	// now — доменное «сейчас» (DomainClock, AD-37).
	now func(ctx context.Context) (time.Time, error)
}

// Option — настройка Service.
type Option func(*Service)

// WithIdentity подключает порт входа (барьер 1, AD-15).
func WithIdentity(idp IdentityProvider) Option { return func(s *Service) { s.idp = idp } }

// WithDirectory подключает каталог затравки: демо-персоны и столы ролей.
func WithDirectory(d *Directory) Option { return func(s *Service) { s.dir = d } }

// WithPolicy подключает проекцию политики (сотрудники, роли, назначения).
func WithPolicy(p PolicySource) Option { return func(s *Service) { s.policy = p } }

// WithAccounts подключает учётные данные и хеш паролей (FR-128).
func WithAccounts(c CredentialStore, h PasswordHasher) Option {
	return func(s *Service) { s.creds, s.hasher = c, h }
}

// WithDecisions подключает запись решений в журнал и доменные часы.
func WithDecisions(w DecisionWriter, now func(ctx context.Context) (time.Time, error)) Option {
	return func(s *Service) { s.decisions, s.now = w, now }
}

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
	return s.sessionOf(ctx, p), nil
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
	d, ok := s.deskFor(ctx, p.Role)
	if !ok {
		return Desk{}, platform.Fail(errcodes.ApiNotFound, "object", "стол роли", "id", p.Role)
	}
	d.Role = p.Role
	return d, nil
}

// deskFor — стол роли по наследованию действующей политики (роль, заведённая
// записью политики, получает стол своей базовой роли без правки кода, FR-78).
func (s *Service) deskFor(ctx context.Context, role string) (Desk, bool) {
	if s.policy != nil {
		if pol, err := s.policy.Policy(ctx); err == nil {
			for _, r := range pol.Hierarchy().Closure(role) {
				if d, ok := s.dir.Desks[r]; ok {
					return d, true
				}
			}
		}
	}
	return s.dir.DeskFor(role)
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
	return s.sessionOf(ctx, p), token, nil
}

// CloseSession — выход (access.session.delete).
func (s *Service) CloseSession(ctx context.Context, token string) error {
	if s.idp == nil {
		return platform.NotImplemented("access.session.delete")
	}
	return s.idp.Close(ctx, token)
}

// sessionOf — представление сеанса субъекта: роль с названием и наследованием из политики.
func (s *Service) sessionOf(ctx context.Context, p platform.Principal) Session {
	role, ok := s.dir.Role(p.Role)
	if s.policy != nil {
		if pol, err := s.policy.Policy(ctx); err == nil {
			if _, has := pol.Role(p.Role); has {
				role, ok = RoleRefOf(pol, p.Role), true
			}
		}
	}
	if !ok {
		role = RoleRef{ID: p.Role, Title: p.Role}
	}
	out := Session{
		User:      SessionUser{ID: p.PersonID, Name: p.Name},
		Role:      role,
		Scope:     p.Scope,
		PolicySeq: p.PolicySeq,
		Demo:      p.Demo,
	}
	// Режим заготовок (присутствия по СКУД нет): пост и смена сеанса — по
	// назначениям мира заготовок (сеанс демо-персоны запасной реализации), чтобы
	// терминал исполнителя и столы работали с «своим» постом (FR-137).
	if s.presence == nil {
		if fx, err := s.Queries.Session(platform.WithPrincipal(ctx, p)); err == nil && fx.User.ID == p.PersonID {
			out.Workplace, out.Shift = fx.Workplace, fx.Shift
		}
	}
	// Эпик 37: рабочее место сеанса — открытый допуск сотрудника (барьер 2).
	if pr, ok, err := s.presenceNow(ctx); err == nil && ok {
		if ss, ok := pr.SessionOf(p.PersonID); ok {
			title := ss.WorkplaceID
			if wp, ok := s.dir.Workplace(ss.WorkplaceID); ok && wp.Name != "" {
				title = wp.Name
			}
			out.Workplace = &SessionWorkplace{ID: ss.WorkplaceID, Title: title}
		}
	}
	return out
}

// loginPattern — логин учётной записи (схема access.account.activated).
var loginPattern = regexp.MustCompile(`^[a-z0-9._-]{3,64}$`)

// MinPasswordLength — наименьшая длина пароля.
const MinPasswordLength = 8

// RequestAccount — заявка на регистрацию (access.account.request, FR-128):
// учётные данные сохраняются в схеме access со статусом pending; в журнал
// заявка не пишется — решение принимает администратор (ActivateAccount).
// Псевдоним сотрудника — `U-‹логин›` (кейс §4.6: условный идентификатор).
func (s *Service) RequestAccount(ctx context.Context, rq AccountRequest) (AccountRequestResult, error) {
	if s.creds == nil || s.hasher == nil {
		return s.Commands.RequestAccount(ctx, rq)
	}
	login := strings.TrimSpace(rq.Login)
	if !loginPattern.MatchString(login) {
		return AccountRequestResult{}, platform.Fail(errcodes.ApiValidationFailed, "field", "login", "reason", "латиница в нижнем регистре, цифры, «.», «_», «-»; 3–64 знака")
	}
	if len([]rune(rq.Password)) < MinPasswordLength {
		return AccountRequestResult{}, platform.Fail(errcodes.ApiValidationFailed, "field", "password", "reason", "не короче 8 знаков")
	}
	hash, err := s.hasher.Hash(rq.Password)
	if err != nil {
		return AccountRequestResult{}, err
	}
	personID := "U-" + login
	c := Credential{Login: login, PersonID: personID, Hash: hash, Status: AccountPending, DisplayName: strings.TrimSpace(rq.DisplayName)}
	if err := s.creds.Create(ctx, c); err != nil {
		if errors.Is(err, ErrLoginTaken) {
			return AccountRequestResult{}, platform.Fail(errcodes.ApiValidationFailed, "field", "login", "reason", "логин занят")
		}
		return AccountRequestResult{}, err
	}
	return AccountRequestResult{Login: login, PersonID: personID, Status: AccountPending}, nil
}

// liveAdmin — администрирование учётных записей подключено (журнал, политика, учётные данные).
func (s *Service) liveAdmin() bool {
	return s.decisions != nil && s.policy != nil && s.creds != nil && s.hasher != nil && s.now != nil
}

// RegisterPerson — завести сотрудника с псевдонимом (access.person.registered, FR-78).
func (s *Service) RegisterPerson(ctx context.Context, in RegisterPerson) (platform.Receipt, error) {
	if !s.liveAdmin() {
		return s.Commands.RegisterPerson(ctx, in)
	}
	if !personPattern.MatchString(in.PersonID) {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "person_id", "reason", "псевдоним: латиница, цифры, «.», «_», «:», «-»")
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	d := ev.AccessPersonRegisteredV1{PersonID: ev.PersonRef(in.PersonID), DisplayName: in.DisplayName}
	if in.OrgUnit != "" {
		d.OrgUnit = &in.OrgUnit
	}
	rc, err := s.decisions.Write(ctx, Batch{Records: []Record{{Type: catalog.AccessPersonRegistered, Stream: "person:" + in.PersonID, Data: d}},
		Meta: in.CommandMeta(), Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now})
	s.refresh(ctx)
	return rc, err
}

// personPattern — псевдоним сотрудника (defs.person_ref).
var personPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)

// ActivateAccount — активировать учётную запись с начальной ролью (FR-128):
// заявку на регистрацию с этим логином или новую учётную запись с начальным
// паролем от администратора. Одной пачкой журнала: сотрудник заведён (если
// его ещё нет), учётная запись активирована, начальная роль назначена в
// области (обычная роль — одной подписью, AD-15). Выдать роль себе нельзя —
// только со второй подписью независимой стороны (эпик 26).
func (s *Service) ActivateAccount(ctx context.Context, personID string, in ActivateAccount) (platform.Receipt, error) {
	if !s.liveAdmin() {
		return s.Commands.ActivateAccount(ctx, personID, in)
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	login := strings.TrimSpace(in.Login)
	if !loginPattern.MatchString(login) {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "login", "reason", "латиница в нижнем регистре, цифры, «.», «_», «-»; 3–64 знака")
	}
	if !personPattern.MatchString(personID) {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "person_id", "reason", "псевдоним сотрудника")
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	scope := in.Scope
	if scope == "" {
		scope = pol.Root
	}
	if in.InitialRoleID != "" {
		if _, ok := pol.Role(in.InitialRoleID); !ok {
			return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "роль", "id", in.InitialRoleID)
		}
		// Начальная роль — обычной выдачей одной подписью; выдача себе и
		// привилегированная роль (администратор, аудит) — только документом
		// выдачи со второй подписью независимой стороны (эпик 26, AD-11).
		if now, err := s.now(ctx); err == nil {
			withPerson := pol
			if _, ok := pol.Person(personID); !ok {
				withPerson = pol.Clone()
				withPerson.Persons = append(withPerson.Persons, accessdom.Person{ID: personID, Name: personID})
			}
			c := accessdom.PolicyChange{Kind: accessdom.ChangeRole, PersonID: personID, SubjectID: in.InitialRoleID, Scope: scope, ValidFrom: now}
			if a := accessdom.Assess(withPerson, c, actor, now); a.SecondAuthority != "" {
				code := errcodes.AccessSignatureRequired
				if a.SelfGrant {
					code = errcodes.AccessSelfGrant
				}
				pe := platform.Fail(code, "who", accessdom.DomainTitle(a.Domain)+" (полномочие "+a.SecondAuthority+")")
				pe.Detail = a.Reason + ". Активируйте учётную запись с обычной ролью, а эту роль выдайте документом «Выдача ролей, полномочий, клейм» (access.policy.grant)."
				pe.AllowedActions = slices.Clone(RequestDecisionActions)
				return platform.Receipt{}, pe
			}
		}
	}
	cred, found, err := s.creds.Get(ctx, login)
	if err != nil {
		return platform.Receipt{}, err
	}
	name := personID
	switch {
	case found && cred.PersonID != personID:
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "login", "reason", "логин принадлежит другому сотруднику ("+cred.PersonID+")")
	case found:
		if cred.DisplayName != "" {
			name = cred.DisplayName
		}
	case in.Password == "":
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "password", "reason", "нет заявки с этим логином — задайте начальный пароль")
	case len([]rune(in.Password)) < MinPasswordLength:
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "password", "reason", "не короче 8 знаков")
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	var recs []Record
	if _, ok := pol.Person(personID); !ok {
		recs = append(recs, Record{Type: catalog.AccessPersonRegistered, Stream: "person:" + personID,
			Data: ev.AccessPersonRegisteredV1{PersonID: ev.PersonRef(personID), DisplayName: name}})
	}
	act := ev.AccessAccountActivatedV1{PersonID: ev.PersonRef(personID), Login: login}
	if in.InitialRoleID != "" {
		r := ev.ObjectID(in.InitialRoleID)
		act.InitialRoleID = &r
	}
	recs = append(recs, Record{Type: catalog.AccessAccountActivated, Stream: "person:" + personID, Data: act})
	if in.InitialRoleID != "" {
		recs = append(recs, Record{Type: catalog.PolicyRoleAssigned, Stream: accessdom.PolicyStream(scope),
			Data: ev.PolicyRoleAssignedV1{PersonID: ev.PersonRef(personID), RoleID: ev.ObjectID(in.InitialRoleID), Scope: scope, ValidFrom: ev.Timestamp(now)}})
	}
	// Учётные данные — до записи в журнал: сбой записи оставит неактивную
	// заявку или учётную запись без сотрудника в политике — вход не пройдёт.
	if !found {
		hash, err := s.hasher.Hash(in.Password)
		if err != nil {
			return platform.Receipt{}, err
		}
		if err := s.creds.Create(ctx, Credential{Login: login, PersonID: personID, Hash: hash, Status: AccountPending, DisplayName: name}); err != nil {
			if errors.Is(err, ErrLoginTaken) {
				return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "login", "reason", "логин занят")
			}
			return platform.Receipt{}, err
		}
	}
	rc, err := s.decisions.Write(ctx, Batch{Records: recs, Meta: in.CommandMeta(), Actor: actor, OccurredAt: now})
	if err != nil {
		return rc, err
	}
	if err := s.creds.Activate(ctx, login, personID); err != nil {
		return rc, err
	}
	s.refresh(ctx)
	return rc, nil
}

// refresh — проекция политики видит свою запись сразу.
func (s *Service) refresh(ctx context.Context) {
	if r, ok := s.policy.(interface{ Refresh(context.Context) error }); ok {
		_ = r.Refresh(ctx)
	}
}

// Persons — сотрудники с ролями и учётными записями (access.person.list, FR-78):
// политика и заявки на регистрацию (pending).
func (s *Service) Persons(ctx context.Context, m platform.Moment, pg platform.Page) (AccessPersonList, error) {
	if s.policy == nil || s.creds == nil {
		return s.Queries.Persons(ctx, m, pg)
	}
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return AccessPersonList{}, err
	}
	creds, err := s.creds.List(ctx)
	if err != nil {
		return AccessPersonList{}, err
	}
	at := s.at(ctx, m)
	out := AccessPersonList{Items: []AccessPerson{}}
	seen := map[string]bool{}
	for _, x := range pol.Persons {
		seen[x.ID] = true
		out.Items = append(out.Items, personView(pol, x, creds, at))
	}
	for _, c := range creds {
		if !seen[c.PersonID] {
			seen[c.PersonID] = true
			out.Items = append(out.Items, AccessPerson{PersonID: c.PersonID, DisplayName: c.DisplayName, AccountStatus: statusOf(c), Login: c.Login,
				Roles: []AccessRoleGrant{}, PolicySeq: pol.Seq})
		}
	}
	return out, nil
}

// Person — сотрудник (access.person.read).
func (s *Service) Person(ctx context.Context, personID string, m platform.Moment) (AccessPerson, error) {
	if s.policy == nil || s.creds == nil {
		return s.Queries.Person(ctx, personID, m)
	}
	list, err := s.Persons(ctx, m, platform.Page{})
	if err != nil {
		return AccessPerson{}, err
	}
	for _, x := range list.Items {
		if x.PersonID == personID {
			return x, nil
		}
	}
	return AccessPerson{}, platform.Fail(errcodes.ApiNotFound, "object", "сотрудник", "id", personID)
}

// Roles — роли и полномочия действующей политики (access.role.list, AD-15).
func (s *Service) Roles(ctx context.Context, m platform.Moment) (AccessRoleList, error) {
	if s.policy == nil {
		return s.Queries.Roles(ctx, m)
	}
	fb, fbErr := s.Queries.Roles(ctx, m)
	pol, err := s.policy.Policy(ctx)
	if err != nil {
		return AccessRoleList{}, err
	}
	out := AccessRoleList{Items: []AccessRole{}, Authorities: []AccessAuthority{}, StampKinds: []string{}, PolicySeq: pol.Seq}
	// Полномочия и виды клейм — из справочной части политики (нормативный слой);
	// нет её — из запасной реализации.
	for _, a := range pol.Catalog.Authorities {
		out.Authorities = append(out.Authorities, AccessAuthority{ID: a.ID, Title: a.Title})
	}
	out.StampKinds = append(out.StampKinds, pol.Catalog.StampKinds...)
	if fbErr == nil && len(out.Authorities) == 0 {
		out.Authorities, out.StampKinds = fb.Authorities, fb.StampKinds
	}
	for _, r := range pol.Roles {
		inh := slices.Clone(r.Inherits)
		if inh == nil {
			inh = []string{}
		}
		out.Items = append(out.Items, AccessRole{ID: r.ID, Title: r.Title, CaseRole: r.CaseRole, Inherits: inh, Actions: slices.Clone(r.Actions)})
	}
	return out, nil
}

// at — доменный момент чтения: as_of или «сейчас».
func (s *Service) at(ctx context.Context, m platform.Moment) time.Time {
	if m.AsOf != nil {
		return *m.AsOf
	}
	if s.now != nil {
		if t, err := s.now(ctx); err == nil {
			return t
		}
	}
	return time.Time{}
}

func personView(pol accessdom.Policy, x accessdom.Person, creds []Credential, at time.Time) AccessPerson {
	v := AccessPerson{PersonID: x.ID, DisplayName: x.Name, OrgUnit: x.OrgUnit, AccountStatus: AccountNone, Login: x.Login,
		Roles: []AccessRoleGrant{}, PolicySeq: pol.Seq}
	if x.Active {
		v.AccountStatus = AccountActive
	}
	for _, c := range creds {
		if c.PersonID == x.ID {
			v.Login, v.AccountStatus = c.Login, statusOf(c)
		}
	}
	for _, a := range pol.AssignmentsOf(x.ID, at) {
		g := AccessRoleGrant{RoleID: a.RoleID, Scope: a.Scope, ValidFrom: a.ValidFrom}
		if !a.ValidUntil.IsZero() {
			u := a.ValidUntil
			g.ValidUntil = &u
		}
		v.Roles = append(v.Roles, g)
	}
	return v
}

func statusOf(c Credential) string {
	switch c.Status {
	case AccountActive, AccountPending, AccountBlocked:
		return c.Status
	}
	return AccountNone
}
