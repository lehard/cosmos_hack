package main

import (
	"context"
	"net/http"
	"time"

	accessapp "ant/internal/application/access"
	analysisapp "ant/internal/application/analysis"
	analyticsapp "ant/internal/application/analytics"
	cadapp "ant/internal/application/cad"
	crossitemapp "ant/internal/application/crossitem"
	documentsapp "ant/internal/application/documents"
	erpapp "ant/internal/application/erp"
	federationapp "ant/internal/application/federation"
	ingestapp "ant/internal/application/ingest"
	itemapp "ant/internal/application/item"
	journalapp "ant/internal/application/journal"
	machinelogsapp "ant/internal/application/machinelogs"
	materialsapp "ant/internal/application/materials"
	mesapp "ant/internal/application/mes"
	nonconformityapp "ant/internal/application/nonconformity"
	notificationsapp "ant/internal/application/notifications"
	opsapp "ant/internal/application/ops"
	"ant/internal/application/platform"
	processapp "ant/internal/application/process"
	qualityapp "ant/internal/application/quality"
	referenceapp "ant/internal/application/reference"
	securityapp "ant/internal/application/security"
	signingapp "ant/internal/application/signing"
	simulationapp "ant/internal/application/simulation"
	visionapp "ant/internal/application/vision"
	accessfx "ant/internal/infrastructure/fixtures/access"
	analysisfx "ant/internal/infrastructure/fixtures/analysis"
	analyticsfx "ant/internal/infrastructure/fixtures/analytics"
	cadfx "ant/internal/infrastructure/fixtures/cad"
	crossitemfx "ant/internal/infrastructure/fixtures/crossitem"
	documentsfx "ant/internal/infrastructure/fixtures/documents"
	erpfx "ant/internal/infrastructure/fixtures/erp"
	federationfx "ant/internal/infrastructure/fixtures/federation"
	ingestfx "ant/internal/infrastructure/fixtures/ingest"
	itemfx "ant/internal/infrastructure/fixtures/item"
	journalfx "ant/internal/infrastructure/fixtures/journal"
	machinelogsfx "ant/internal/infrastructure/fixtures/machinelogs"
	materialsfx "ant/internal/infrastructure/fixtures/materials"
	mesfx "ant/internal/infrastructure/fixtures/mes"
	nonconformityfx "ant/internal/infrastructure/fixtures/nonconformity"
	notificationsfx "ant/internal/infrastructure/fixtures/notifications"
	opsfx "ant/internal/infrastructure/fixtures/ops"
	processfx "ant/internal/infrastructure/fixtures/process"
	qualityfx "ant/internal/infrastructure/fixtures/quality"
	securityfx "ant/internal/infrastructure/fixtures/security"
	signingfx "ant/internal/infrastructure/fixtures/signing"
	simulationfx "ant/internal/infrastructure/fixtures/simulation"
	visionfx "ant/internal/infrastructure/fixtures/vision"
	"ant/internal/infrastructure/security/permissive"
	accesshttp "ant/internal/infrastructure/transport/access"
	analysishttp "ant/internal/infrastructure/transport/analysis"
	analyticshttp "ant/internal/infrastructure/transport/analytics"
	cadhttp "ant/internal/infrastructure/transport/cad"
	crossitemhttp "ant/internal/infrastructure/transport/crossitem"
	documentshttp "ant/internal/infrastructure/transport/documents"
	erphttp "ant/internal/infrastructure/transport/erp"
	federationhttp "ant/internal/infrastructure/transport/federation"
	"ant/internal/infrastructure/transport/httpapi"
	ingesthttp "ant/internal/infrastructure/transport/ingest"
	itemhttp "ant/internal/infrastructure/transport/item"
	journalhttp "ant/internal/infrastructure/transport/journal"
	machinelogshttp "ant/internal/infrastructure/transport/machinelogs"
	materialshttp "ant/internal/infrastructure/transport/materials"
	meshttp "ant/internal/infrastructure/transport/mes"
	nonconformityhttp "ant/internal/infrastructure/transport/nonconformity"
	notificationshttp "ant/internal/infrastructure/transport/notifications"
	opshttp "ant/internal/infrastructure/transport/ops"
	processhttp "ant/internal/infrastructure/transport/process"
	qualityhttp "ant/internal/infrastructure/transport/quality"
	referencehttp "ant/internal/infrastructure/transport/reference"
	securityhttp "ant/internal/infrastructure/transport/security"
	signinghttp "ant/internal/infrastructure/transport/signing"
	simulationhttp "ant/internal/infrastructure/transport/simulation"
	visionhttp "ant/internal/infrastructure/transport/vision"
)

// apiOptions — режимы ведущих портов и адаптеры прав для сборки API.
type apiOptions struct {
	mode        platform.Mode
	moduleModes map[string]platform.Mode
	// journal — live-реализация journal над журналом и публикатором SSE
	// (core.go); nil — заглушка 501 (выгрузка OpenAPI, тесты).
	journal *journalapp.Service
	// machinelogs — live-реализация machinelogs над проекциями (engine.go);
	// nil — заглушка 501.
	machinelogs *machinelogsapp.Service
	// analysis — live-реализация analysis над проекциями и журналом
	// (engine.go, эпик 22); nil — заглушка 501.
	analysis *analysisapp.Service
	// nonconformity — live-реализация nonconformity над журналом и свёрткой
	// изделия (nonconformity.go, эпик 21); nil — заглушка 501.
	nonconformity *nonconformityapp.Service
	// erp — live-реализация erp над проекциями erp.*, каналами обмена и
	// журналом (outbox.go, эпик 30); nil — заглушка 501.
	erp *erpapp.Service
	// ingest — live-приём над журналом ядра (ingest.go); nil — заглушка 501.
	ingest *ingestapp.Service
	// access — вход, сеансы и права (эпик 08, identity.go); nil —
	// разрешающая заглушка без сеансов (выгрузка OpenAPI, тесты).
	access *accessBundle
	// quality — живые операции quality над проекциями движка (quality.go); nil — 501.
	quality *qualityapp.Service
	// analytics — live-показатели над строками вклада ядра (analytics.go);
	// nil — без хранилища (операции 501).
	analytics *analyticsapp.Service
	// vision — live-реализация vision над журналом ядра (vision.go, эпик 33); nil — 501.
	vision *visionapp.Service
	// notifications — live-реализация notifications над проекциями сроков,
	// задач и уведомлений (notifications.go, эпик 24); nil — заглушка 501.
	notifications *notificationsapp.Service
	// item, crossitem — живые операции изделия и межизделийной стадии
	// (item.go, эпик 18); nil — заглушка 501.
	item      *itemapp.Service
	crossitem *crossitemapp.Service
	// process — живая карта, версии и команды исполнителя (process.go, эпик 17); nil — 501.
	process *processapp.Service
	// simulation — пульт тестовых сценариев (simulation.go, эпики 32, 16); nil — 501.
	simulation *simulationapp.Service
	// security — журнал CA, шина безопасности, индикатор целостности (security.go, эпик 29); nil — 501.
	security *securityapp.Service
	// reference — справочники из журнала ядра (reference.go, эпик 19); nil — 501.
	reference *referenceapp.Service
	// documents — документы-проекции, маршруты подписей, печать с QR
	// (documents.go, эпик 28); nil — заглушка 501.
	documents *documentsapp.Service
	// ops — состояние компонентов, остановленные изделия, настройки
	// (ops.go, эпик 34); nil — заглушка 501.
	ops *opsapp.Service
	// mes, cad — блоки и задания MES, импорт сборки КОМПАС (mes.go, cad.go,
	// эпик 31); nil — заглушка 501.
	mes *mesapp.Service
	cad *cadapp.Service
	// signing — ключи, профили, акты и проверка подписи команд уровня ≥ 1
	// общим декоратором (signing.go, Д-59); nil — 501 и без проверки подписи.
	signing *signingapp.Service
	// federation — партнёры и выписки паспорта (federation.go, эпик 41); nil — 501.
	federation *federationapp.Service
}

// buildAPI собирает HTTP API: общий декоратор (Gate) над портами прав и входа,
// реализации ведущих портов каждого модуля по режиму fixtures | live (AD-36) и
// регистрация операций всех модулей. Все модули зарегистрированы заранее
// (волна 1): эпики модулей меняют реализации портов, а не этот список.
func buildAPI(mux *http.ServeMux, o apiOptions) *httpapi.API {
	// Эпик 08: субъект — по сеансу (IdentityProvider), права — Casbin над
	// проекцией политики (access_control = casbin); без них — заглушки волны 1.
	var ac accessapp.AccessControl = permissive.AccessControl{}
	var idp accessapp.IdentityProvider = permissive.Identity{}
	if x := o.access; x != nil {
		idp = x.identity
		if x.control != nil {
			ac = x.control
		}
	}
	gate := accessapp.NewGate(ac, nil, nil)
	if x := o.access; x != nil {
		gate.Places, gate.Events = x.places, x.events
		gate.Now = func() time.Time { t, _ := x.now(context.Background()); return t }
		// Эпик 26: политика — для проверки policy_seq команды в journal.Append
		// (AD-39), редких подписантов и объяснения прав своим кодом.
		gate.Policy = x.policy
	}
	hc := httpapi.Config{Mode: o.mode, ModuleModes: o.moduleModes, Gate: gate, Identity: idp}
	if o.signing != nil {
		// Д-59: подпись команд уровня ≥ 1 — signing.CheckCommand по ExpectRequest.
		hc.Signatures = o.signing
	}
	a := httpapi.New(mux, hc)
	gate.SetCatalog(a.Actions)

	{
		live := o.journal
		if live == nil {
			live = journalapp.NewService()
		}
		var fx any = journalfx.New()
		if o.journal != nil {
			// Заготовки + живое ядро: SSE присылает и смену шага курсора, и
			// изменения движка по фактам живого приёма (hybrid.go).
			fx = withLiveStream(journalfx.New(), o.journal)
		}
		q, c := pick[journalapp.Queries, journalapp.Commands](a.ModeFor("journal"), live, fx)
		journalhttp.Register(a, q, c)
	}
	{
		live := o.crossitem
		if live == nil {
			live = crossitemapp.NewService()
		}
		q, c := pick[crossitemapp.Queries, crossitemapp.Commands](a.ModeFor("crossitem"), live, crossitemfx.New())
		crossitemhttp.Register(a, q, c)
	}
	{
		live := o.ingest
		if live == nil {
			live = ingestapp.NewService()
		}
		q, c := pick[ingestapp.Queries, ingestapp.Commands](a.ModeFor("ingest"), live, ingestfx.New())
		ingesthttp.Register(a, q, c)
	}
	{
		live := o.reference
		if live == nil {
			live = referenceapp.NewService()
		}
		q, c := pick[referenceapp.Queries, referenceapp.Commands](a.ModeFor("reference"), live, referenceFixtures())
		referencehttp.Register(a, q, c)
	}
	{
		live := o.process
		if live == nil {
			live = processapp.NewService()
		}
		q, c := pick[processapp.Queries, processapp.Commands](a.ModeFor("process"), live, processfx.New())
		processhttp.Register(a, q, c)
	}
	{
		live := o.item
		if live == nil {
			live = itemapp.NewService()
		}
		q, c := pick[itemapp.Queries, itemapp.Commands](a.ModeFor("item"), live, itemfx.New())
		itemhttp.Register(a, q, c)
	}
	{
		live := o.quality
		if live == nil {
			live = qualityapp.NewService()
		}
		q, c := pick[qualityapp.Queries, qualityapp.Commands](a.ModeFor("quality"), live, qualityfx.New())
		qualityhttp.Register(a, q, c)
	}
	{
		live := o.nonconformity
		if live == nil {
			live = nonconformityapp.NewService()
		}
		q, c := pick[nonconformityapp.Queries, nonconformityapp.Commands](a.ModeFor("nonconformity"), live, nonconformityfx.New())
		nonconformityhttp.Register(a, q, c)
	}
	{
		live := o.analysis
		if live == nil {
			live = analysisapp.NewService()
		}
		q, c := pick[analysisapp.Queries, analysisapp.Commands](a.ModeFor("analysis"), live, analysisfx.New())
		analysishttp.Register(a, q, c)
	}
	{
		live := o.machinelogs
		if live == nil {
			live = machinelogsapp.NewService()
		}
		q, c := pick[machinelogsapp.Queries, machinelogsapp.Commands](a.ModeFor("machinelogs"), live, machinelogsfx.New())
		machinelogshttp.Register(a, q, c)
	}
	{
		live := o.vision
		if live == nil {
			live = visionapp.NewService()
		}
		q, c := pick[visionapp.Queries, visionapp.Commands](a.ModeFor("vision"), live, visionfx.New())
		visionhttp.Register(a, q, c)
	}
	{
		live := o.documents
		if live == nil {
			live = documentsapp.NewService()
		}
		q, c := pick[documentsapp.Queries, documentsapp.Commands](a.ModeFor("documents"), live, documentsfx.New())
		documentshttp.Register(a, q, c)
	}
	{
		live := o.signing
		if live == nil {
			live = signingapp.NewService()
		}
		q, c := pick[signingapp.Queries, signingapp.Commands](a.ModeFor("signing"), live, signingfx.New())
		signinghttp.Register(a, q, c)
	}
	{
		// Вход, сеанс, стол роли, демо-персоны, заявка на регистрацию, сотрудники,
		// роли и активация учётной записи — живые в обоих режимах (эпик 08);
		// посты, клейма, допуск — заготовки или 501 по режиму.
		fq, fc := pick[accessapp.Queries, accessapp.Commands](a.ModeFor("access"), accessapp.Unimplemented{}, accessfx.New())
		opts := []accessapp.Option{accessapp.WithFallback(fq, fc)}
		if x := o.access; x != nil {
			opts = append(opts, accessapp.WithIdentity(x.identity), accessapp.WithDirectory(x.directory), accessapp.WithPolicy(x.policy),
				accessapp.WithAccounts(x.creds, x.hasher), accessapp.WithDecisions(x.decisions, x.now))
			// Эпик 26: документ выдачи прав и карточки редких подписантов — по
			// живому модулю documents (маршрут подписей, AD-43, FR-136).
			// Посты, назначения и факты исполнителя — живые в режиме live модуля
			// access (в режиме fixtures панель «Посты» — у заготовок).
			if a.ModeFor("access") == platform.ModeLive {
				opts = append(opts, accessapp.WithLiveRoster(x.facts))
				if x.wplog != nil {
					opts = append(opts, accessapp.WithWorkplaceLog(x.wplog))
				}
				// Эпик 37: присутствие по СКУД и ключу, допуск к рабочему месту.
				if x.presence != nil {
					opts = append(opts, accessapp.WithPresence(x.presence, x.pw, x.shifts))
				}
			}
			if o.documents != nil && a.ModeFor("documents") == platform.ModeLive {
				bridge := accessapp.DocumentsBridge{Docs: o.documents}
				opts = append(opts, accessapp.WithGrantDocuments(bridge))
				gate.Cards = bridge
			}
		}
		live := accessapp.NewService(opts...)
		accesshttp.Register(a, live, live, gate)
	}
	{
		live := o.security
		if live == nil {
			live = securityapp.NewService()
		}
		q, c := pick[securityapp.Queries, securityapp.Commands](a.ModeFor("security"), live, securityfx.New())
		securityhttp.Register(a, q, c)
	}
	{
		q, c := pick[materialsapp.Queries, materialsapp.Commands](a.ModeFor("materials"), materialsapp.NewService(), materialsfx.New())
		materialshttp.Register(a, q, c)
	}
	{
		live := o.notifications
		if live == nil {
			live = notificationsapp.NewService()
		}
		q, c := pick[notificationsapp.Queries, notificationsapp.Commands](a.ModeFor("notifications"), live, notificationsfx.New())
		notificationshttp.Register(a, q, c)
	}
	{
		live := o.analytics
		if live == nil {
			live = analyticsapp.NewService()
		}
		q, c := pick[analyticsapp.Queries, analyticsapp.Commands](a.ModeFor("analytics"), live, analyticsfx.New())
		analyticshttp.Register(a, q, c)
	}
	{
		live := o.erp
		if live == nil {
			live = erpapp.NewService()
		}
		q, c := pick[erpapp.Queries, erpapp.Commands](a.ModeFor("erp"), live, erpfx.New())
		erphttp.Register(a, q, c)
	}
	{
		live := o.mes
		if live == nil {
			live = mesapp.NewService()
		}
		q, c := pick[mesapp.Queries, mesapp.Commands](a.ModeFor("mes"), live, mesfx.New())
		meshttp.Register(a, q, c)
	}
	{
		live := o.cad
		if live == nil {
			live = cadapp.NewService()
		}
		q, c := pick[cadapp.Queries, cadapp.Commands](a.ModeFor("cad"), live, cadfx.New())
		cadhttp.Register(a, q, c)
	}
	{
		live := o.federation
		if live == nil {
			live = federationapp.NewService()
		}
		q, c := pick[federationapp.Queries, federationapp.Commands](a.ModeFor("federation"), live, federationfx.New())
		federationhttp.Register(a, q, c)
	}
	{
		var live any = simulationapp.NewService()
		if o.simulation != nil {
			live = o.simulation
		}
		q, c := pick[simulationapp.Queries, simulationapp.Commands](a.ModeFor("simulation"), live, simulationfx.New())
		simulationhttp.Register(a, q, c)
	}
	{
		live := o.ops
		if live == nil {
			live = opsapp.NewService()
		} else {
			// Режимы ведущих портов модулей — из каталога операций API (ops.setting.list).
			live.SetModules(func() []opsapp.ModuleMode { return moduleModesOf(a.Actions(), a.ModeFor) })
		}
		q, c := pick[opsapp.Queries, opsapp.Commands](a.ModeFor("ops"), live, opsfx.New())
		opshttp.Register(a, q, c)
	}
	return a
}

// pick выбирает реализацию ведущих портов по режиму (AD-36): live — сценарии
// приложения, fixtures — заготовки; одна реализация отвечает за оба порта.
func pick[Q, C any](mode platform.Mode, live, fixtures interface{}) (Q, C) {
	impl := live
	if mode == platform.ModeFixtures {
		impl = fixtures
	}
	return impl.(Q), impl.(C)
}

// accessDirectory — каталог политики из пакета доступа (эпик 08); nil, если
// доступ не собран (полномочия сотрудников для гарда точки предъявления, эпик 16).
func (o apiOptions) accessDirectory() *accessapp.Directory {
	if o.access == nil {
		return nil
	}
	return o.access.directory
}

// policyAuthorities — полномочия по проекции политики (эпик 26); нет доступа
// или проекции — false (модули остаются на стартовом каталоге).
func (o apiOptions) policyAuthorities() (accessapp.PolicyAuthorities, bool) {
	if o.access == nil {
		return accessapp.PolicyAuthorities{}, false
	}
	return o.access.authorities()
}
