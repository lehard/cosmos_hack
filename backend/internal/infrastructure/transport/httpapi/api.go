package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"ant/internal/application/access"
	"ant/internal/application/platform"
)

func init() {
	// Списки в ответах — массивы, а не «массив или null»: обработчики отдают
	// пустые списки (клиент фронтенда получает T[], а не T[] | null).
	huma.DefaultArrayNullable = false
}

// Prefix — префикс всех операций API; адрес SSE-канала — из asyncapi.yaml
// (/api/v1/stream).
const Prefix = "/api/v1"

// Config — параметры API.
type Config struct {
	// Version — версия приложения (в info.version спецификации не пишется:
	// спецификация должна быть детерминированной, AD-20).
	Version string
	// Mode — режим ведущих портов по умолчанию (AD-36).
	Mode platform.Mode
	// ModuleModes — переопределение режима по модулю (вертикальные срезы, AD-36).
	ModuleModes map[string]platform.Mode
	// Gate — общий декоратор приложения (права, место сеанса, гарды, допустимые действия).
	Gate *access.Gate
	// Identity — порт определения субъекта по сеансу (барьер 1, AD-15).
	Identity access.IdentityProvider
	// Signatures — проверка подписи команд уровня ≥ 1 (модуль signing, Д-59);
	// nil — подпись не проверяется (выгрузка OpenAPI, тесты).
	Signatures platform.SignatureChecker
}

// API — Huma API ant с каталогом зарегистрированных операций.
type API struct {
	huma    huma.API
	cfg     Config
	actions []platform.Action
}

// New создаёт API на mux. Документация (/docs) выключена: закрытый контур,
// внешних ресурсов нет (NFR-SEC-1); спецификация доступна по /api/v1/openapi.yaml.
func New(mux *http.ServeMux, cfg Config) *API {
	hc := huma.DefaultConfig("ant API", "1")
	hc.Info.Description = "HTTP API системы ant — доверенная система контроля качества. " +
		"Сгенерировано из операций Huma (backend/internal/infrastructure/transport); руками не править (AD-20). " +
		"Каждая операция несёт x-ant-action: id (он же operationId, ключ прав Casbin и @casl), класс (AD-27), " +
		"критичность (AD-28), модуль-владелец (AD-40), гарды (AD-39), эмитируемые типы записей журнала. " +
		"Чтение принимает axis и as_of (AD-21, AD-22); команды — command_id, basis_seq, policy_seq (AD-7, AD-39); " +
		"режим fixtures | live — в заголовке ответа Ant-Backend (AD-36); ошибки — RFC 9457 problem+json с кодом из contracts/errors.yaml."
	hc.Servers = []*huma.Server{{URL: "/"}}
	hc.OpenAPIPath = Prefix + "/openapi"
	hc.DocsPath = ""
	hc.SchemasPath = ""
	hc.CreateHooks = nil
	h := humago.New(mux, hc)
	a := &API{huma: h, cfg: cfg}
	h.UseMiddleware(a.identify)
	h.UseMiddleware(a.captureSigned)
	registerComponents(h)
	return a
}

// Huma возвращает нижележащий huma.API (для особых операций: SSE, загрузка файлов).
func (a *API) Huma() huma.API { return a.huma }

// Actions — каталог зарегистрированных операций, отсортированный по id
// (для Gate: допустимые действия, плоский список прав фронтенда).
func (a *API) Actions() []platform.Action {
	out := slices.Clone(a.actions)
	slices.SortFunc(out, func(x, y platform.Action) int { return strings.Compare(x.ID, y.ID) })
	return out
}

// ModeFor — режим ведущих портов модуля (AD-36).
func (a *API) ModeFor(module string) platform.Mode {
	if m, ok := a.cfg.ModuleModes[module]; ok {
		return m
	}
	if a.cfg.Mode == "" {
		return platform.ModeLive
	}
	return a.cfg.Mode
}

// OpenAPIYAML — спецификация для contracts/openapi.yaml (OpenAPI 3.1).
func (a *API) OpenAPIYAML() ([]byte, error) {
	o := a.huma.OpenAPI()
	// Теги — по модулям, отсортированы (детерминизм выгрузки).
	seen := map[string]bool{}
	var tags []*huma.Tag
	for _, act := range a.Actions() {
		if !seen[act.Owner] {
			seen[act.Owner] = true
			tags = append(tags, &huma.Tag{Name: act.Owner, Description: moduleTitle(act.Owner)})
		}
	}
	slices.SortFunc(tags, func(x, y *huma.Tag) int { return strings.Compare(x.Name, y.Name) })
	o.Tags = tags
	return o.YAML()
}

// identify — барьер 1 (AD-15): субъект по cookie сеанса или заголовку демо-персоны
// через порт IdentityProvider; нет сеанса — анонимный (права решает Gate).
func (a *API) identify(ctx huma.Context, next func(huma.Context)) {
	if a.cfg.Identity == nil {
		next(ctx)
		return
	}
	cred := access.Credentials{}
	if c, err := huma.ReadCookie(ctx, SessionCookie); err == nil {
		cred.SessionToken = c.Value
	}
	cred.DemoPersona = ctx.Header(DemoPersonaHeader)
	p, err := a.cfg.Identity.Identify(ctx.Context(), cred)
	if err != nil {
		p = platform.Principal{}
	}
	next(huma.WithContext(ctx, platform.WithPrincipal(ctx.Context(), p)))
}

// SessionCookie — cookie сеанса веба (scs, эпик 08).
const SessionCookie = "ant_session"

// DemoPersonaHeader — заголовок выбора демо-персоны (только профили fixtures и
// demo, демо-трек эпика 08); в prod IdentityProvider его не принимает.
const DemoPersonaHeader = "Ant-Demo-Persona"

// objecter — вход операции, знающий свой объект (для прав по объекту, AD-15).
type objecter interface{ Object() platform.ObjectRef }

// momenter — вход чтения с моментом.
type momenter interface {
	Moment() (platform.Moment, error)
}

// metaSetter — выход с заголовками Ant-Backend и Ant-Seq.
type metaSetter interface{ setMode(platform.Mode) }

// Register объявляет операцию: маршрут route, описание x-ant-action act и
// обработчик h, вызывающий ведущий порт модуля. Общий декоратор (Gate) и
// перевод ошибок в problem+json — здесь, одинаково для всех операций.
// Паникует, если описание операции нарушает контракт (make generate краснеет).
func Register[I, O any](a *API, route Route, act platform.Action, h func(ctx context.Context, in *I) (*O, error)) {
	if err := act.Validate(); err != nil {
		panic(err)
	}
	if route.Method == http.MethodGet && act.IsCommand() {
		panic(fmt.Errorf("x-ant-action %s: команда не может быть GET", act.ID))
	}
	if route.Method != http.MethodGet && !act.IsCommand() && !route.ReadByPost {
		panic(fmt.Errorf("x-ant-action %s: чтение — только GET", act.ID))
	}
	if act.IsCommand() && route.Method != http.MethodDelete && !route.NoCommandMeta {
		if !hasCommandMeta[I]() {
			panic(fmt.Errorf("x-ant-action %s: у команды нет command_id/basis_seq/policy_seq (platform.CommandHeader)", act.ID))
		}
	}
	for _, x := range a.actions {
		if x.ID == act.ID {
			panic(fmt.Errorf("x-ant-action %s: операция объявлена дважды", act.ID))
		}
	}
	a.actions = append(a.actions, act)

	op := huma.Operation{
		OperationID:   act.ID,
		Method:        route.Method,
		Path:          Prefix + route.Path,
		Summary:       route.Summary,
		Description:   route.Description,
		Tags:          []string{act.Owner},
		DefaultStatus: route.Status,
		Extensions:    map[string]any{"x-ant-action": actionExtension(act)},
	}
	if route.ReadByPost {
		op.Extensions["x-ant-read-by-post"] = true
	}
	mode := a.ModeFor(act.Owner)
	huma.Register(a.huma, op, func(ctx context.Context, in *I) (*O, error) {
		ctx, err := a.before(ctx, act, in)
		if err != nil {
			return nil, err
		}
		out, err := h(ctx, in)
		if err != nil {
			return nil, problemFrom(err, act.ID)
		}
		if ms, ok := any(out).(metaSetter); ok {
			ms.setMode(mode)
		}
		return out, nil
	})
}

// before — общий декоратор до вызова порта: момент чтения, запрет команд в
// воспроизведении, права и гарды через Gate (AD-15, AD-21, AD-36, AD-39).
// Возвращает контекст команды с проверками политики субъекта: journal.Append
// отвергнет запись по устаревшей политике (journal.stale_policy, AD-39).
func (a *API) before(ctx context.Context, act platform.Action, in any) (context.Context, error) {
	var obj platform.ObjectRef
	if o, ok := in.(objecter); ok {
		obj = o.Object()
	}
	var meta *platform.CommandMeta
	if m, ok := commandMeta(in); ok {
		meta = &m
	}
	if m, ok := in.(momenter); ok {
		mo, err := m.Moment()
		if err != nil {
			return ctx, problemFrom(platformValidation("query", err.Error()), act.ID)
		}
		if mo.IsReplay() && act.IsCommand() {
			return ctx, problemFrom(platformFail("api.replay_read_only"), act.ID)
		}
	}
	if a.cfg.Gate == nil {
		return a.checkSignature(ctx, act, meta)
	}
	ctx, err := a.cfg.Gate.Admit(ctx, platform.PrincipalFrom(ctx), act, obj, meta)
	if err != nil {
		return ctx, problemFrom(err, act.ID)
	}
	return a.checkSignature(ctx, act, meta)
}

func actionExtension(act platform.Action) map[string]any {
	m := map[string]any{
		"id":       act.ID,
		"class":    string(act.Class),
		"critical": act.Critical,
		"owner":    act.Owner,
	}
	if act.CAGroup != "" {
		m["ca_group"] = act.CAGroup
	}
	if act.Subject != "" {
		m["subject"] = act.Subject
	}
	if len(act.Guards) > 0 {
		m["guards"] = slices.Clone(act.Guards)
	}
	if len(act.Emits) > 0 {
		emits := make([]string, len(act.Emits))
		for i, t := range act.Emits {
			emits[i] = string(t)
		}
		m["emits"] = emits
	}
	if act.SignatureLevel > 0 {
		m["signature_level"] = act.SignatureLevel
	}
	return m
}

// Route — HTTP-часть операции.
type Route struct {
	Method      string
	Path        string // от Prefix: "/items/{item_id}/passport"
	Summary     string
	Description string
	// Status — код успешного ответа (по умолчанию 200 или 204 без тела).
	Status int
	// ReadByPost — чтение с телом запроса (сложный фильтр); класс read, метод POST.
	ReadByPost bool
	// NoCommandMeta — команда без метаданных AD-39 (вход, выход, служебные операции сеанса).
	NoCommandMeta bool
}

// Get, Post, Put, Delete — сокращения маршрута.
func Get(path, summary, description string) Route {
	return Route{Method: http.MethodGet, Path: path, Summary: summary, Description: description}
}

// Post — команда (или чтение с телом при ReadByPost).
func Post(path, summary, description string) Route {
	return Route{Method: http.MethodPost, Path: path, Summary: summary, Description: description}
}

// Delete — команда удаления (выход из сеанса и т. п.).
func Delete(path, summary, description string) Route {
	return Route{Method: http.MethodDelete, Path: path, Summary: summary, Description: description}
}
