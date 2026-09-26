package access

import "time"

// RoleRef — роль из политики (normative/policy).
type RoleRef struct {
	ID       string   `json:"id" pattern:"^[a-z][a-z0-9_]*$" doc:"id роли из политики (normative/policy)."`
	Title    string   `json:"title" doc:"Название роли из политики."`
	Inherits []string `json:"inherits,omitempty" doc:"Базовые роли (руководящие подписанты — наследники)."`
}

// DemoPersona — демо-персона экрана входа: псевдоним из стартовой политики с ролью и областью.
type DemoPersona struct {
	ID    string  `json:"id" doc:"Псевдоним сотрудника (кейс §4.6)."`
	Name  string  `json:"name" doc:"Отображаемое имя (условное)."`
	Role  RoleRef `json:"role"`
	Scope string  `json:"scope,omitempty" doc:"Область действия роли (здание → цех → участок → рабочее место)."`
}

// DemoPersonaList — список демо-персон.
type DemoPersonaList struct {
	Items []DemoPersona `json:"items"`
}

// SessionCreate — вход: демо-персоной (persona_id) или по логину; пароль пока
// необязателен (демо-трек), после эпика 08 — обязателен для входа по логину.
type SessionCreate struct {
	PersonaID string `json:"persona_id,omitempty" maxLength:"64" doc:"Демо-персона (только профили fixtures и demo)."`
	Login     string `json:"login,omitempty" maxLength:"128" doc:"Логин."`
	Password  string `json:"password,omitempty" maxLength:"256" doc:"Пароль (argon2id на сервере, эпик 08)."`
}

// SessionUser — пользователь сеанса.
type SessionUser struct {
	ID   string `json:"id" doc:"Псевдоним сотрудника."`
	Name string `json:"name" doc:"Отображаемое имя."`
}

// SessionShift — смена сеанса.
type SessionShift struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`
}

// SessionWorkplace — рабочее место сеанса (допуск — барьер 2).
type SessionWorkplace struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Session — текущий сеанс: пользователь, активная роль, область, смена,
// рабочее место, версия политики (FR-128, AD-15).
type Session struct {
	User      SessionUser       `json:"user"`
	Role      RoleRef           `json:"role"`
	Scope     string            `json:"scope,omitempty" doc:"Область роли."`
	Shift     *SessionShift     `json:"shift,omitempty"`
	Workplace *SessionWorkplace `json:"workplace,omitempty"`
	PolicySeq int64             `json:"policy_seq" minimum:"0" doc:"Версия политики, по которой вычислены права (AD-39)."`
	Demo      bool              `json:"demo" doc:"Вход демо-персоной без пароля."`
}

// Density — плотность стола: мастеру и исполнителю крупно, технологу — плотный инженерный вид.
type Density string

// Values — значения плотности.
func (Density) Values() []string { return []string{"large", "comfortable", "compact"} }

// Describe — описание для спецификации.
func (Density) Describe() string {
	return "Плотность — мастеру и исполнителю крупно (large), технологу — плотный инженерный вид (compact)."
}

// DeskLayout — раскладка вкладки стола.
type DeskLayout string

// Values — раскладки.
func (DeskLayout) Values() []string {
	return []string{"single", "main-side", "queue-main-side", "overview"}
}

// Describe — описание для спецификации.
func (DeskLayout) Describe() string {
	return "Раскладка вкладки: single — main; main-side — main + right; queue-main-side — left + main + right; overview — top + main + right + bottom."
}

// DeskArea — область раскладки.
type DeskArea string

// Values — области.
func (DeskArea) Values() []string { return []string{"top", "left", "main", "right", "bottom"} }

// DeskSlot — слот стола: область, id виджета из frontend/src/widgets/registry.ts, плотность и срез.
type DeskSlot struct {
	ID      string         `json:"id" pattern:"^[a-z][a-z0-9-]*$"`
	Area    DeskArea       `json:"area"`
	Widget  string         `json:"widget" pattern:"^[a-z][a-z0-9-]*$" doc:"id виджета из frontend/src/widgets/registry.ts."`
	Density Density        `json:"density,omitempty"`
	Slice   map[string]any `json:"slice,omitempty" doc:"Параметры среза — «одна правда, разные взгляды»."`
}

// DeskTab — вкладка стола.
type DeskTab struct {
	ID       string     `json:"id" pattern:"^[a-z][a-z0-9-]*$"`
	TitleKey string     `json:"title_key" doc:"Ключ текста интерфейса (ru.json)."`
	Layout   DeskLayout `json:"layout"`
	Density  Density    `json:"density,omitempty"`
	Slots    []DeskSlot `json:"slots"`
}

// Desk — стол роли: normative/desks/‹роль›.yaml (схема normative/desks/desk.schema.json, AD-21).
type Desk struct {
	Version  int       `json:"version" enum:"1" doc:"Версия формата стола."`
	Role     string    `json:"role" doc:"id роли из политики."`
	TitleKey string    `json:"title_key" doc:"Ключ текста названия стола."`
	Density  Density   `json:"density"`
	HelpKey  string    `json:"help_key,omitempty" doc:"Раздел встроенной справки роли."`
	Tabs     []DeskTab `json:"tabs" minItems:"1"`
}

// PostPerson — назначенный на пост сотрудник (псевдоним, «Соглашения/Идентификаторы»).
type PostPerson struct {
	PersonID string `json:"person_id"`
	Display  string `json:"display"`
}

// PostItem — текущее изделие на посту.
type PostItem struct {
	ItemID string `json:"item_id"`
	Label  string `json:"label"`
}

// PostRow — строка панели «Посты» (PRD §3a, FR-6, FR-81): участок — кто
// назначен — на месте ли (СКУД, ключ вставлен) — текущее изделие. Данных нет —
// «unknown», а не «на месте».
type PostRow struct {
	WorkplaceID string      `json:"workplace_id"`
	Station     string      `json:"station" doc:"Участок (пост) — подпись."`
	Workshop    string      `json:"workshop,omitempty" doc:"Цех (FR-130)."`
	Assigned    *PostPerson `json:"assigned,omitempty" doc:"Нет — никто не назначен."`
	Presence    string      `json:"presence" enum:"present,key_missing,owner_absent,absent,not_assigned,unknown" doc:"На месте; по СКУД на месте, но ключ не вставлен; ключ вставлен, а владельца нет в зоне; нет ни в зоне, ни ключа; никто не назначен; неизвестно."`
	CurrentItem *PostItem   `json:"current_item,omitempty" doc:"Нет — на посту нет изделия."`
}

// PostList — посты в области.
type PostList struct {
	Items []PostRow `json:"items"`
}
