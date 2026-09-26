package main

import (
	"net/http"

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
	referencefx "ant/internal/infrastructure/fixtures/reference"
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
}

// buildAPI собирает HTTP API: общий декоратор (Gate) над портами прав и входа,
// реализации ведущих портов каждого модуля по режиму fixtures | live (AD-36) и
// регистрация операций всех модулей. Все модули зарегистрированы заранее
// (волна 1): эпики модулей меняют реализации портов, а не этот список.
func buildAPI(mux *http.ServeMux, o apiOptions) *httpapi.API {
	// Волна 1: разрешающие заглушки прав и входа (эпик 08 заменит адаптеры).
	ac := permissive.AccessControl{}
	idp := permissive.Identity{}
	gate := accessapp.NewGate(ac, nil, nil)
	a := httpapi.New(mux, httpapi.Config{Mode: o.mode, ModuleModes: o.moduleModes, Gate: gate, Identity: idp})
	gate.SetCatalog(a.Actions)

	{
		live := o.journal
		if live == nil {
			live = journalapp.NewService()
		}
		q, c := pick[journalapp.Queries, journalapp.Commands](a.ModeFor("journal"), live, journalfx.New())
		journalhttp.Register(a, q, c)
	}
	{
		q, c := pick[crossitemapp.Queries, crossitemapp.Commands](a.ModeFor("crossitem"), crossitemapp.NewService(), crossitemfx.New())
		crossitemhttp.Register(a, q, c)
	}
	{
		q, c := pick[ingestapp.Queries, ingestapp.Commands](a.ModeFor("ingest"), ingestapp.NewService(), ingestfx.New())
		ingesthttp.Register(a, q, c)
	}
	{
		q, c := pick[referenceapp.Queries, referenceapp.Commands](a.ModeFor("reference"), referenceapp.NewService(), referencefx.New())
		referencehttp.Register(a, q, c)
	}
	{
		q, c := pick[processapp.Queries, processapp.Commands](a.ModeFor("process"), processapp.NewService(), processfx.New())
		processhttp.Register(a, q, c)
	}
	{
		q, c := pick[itemapp.Queries, itemapp.Commands](a.ModeFor("item"), itemapp.NewService(), itemfx.New())
		itemhttp.Register(a, q, c)
	}
	{
		q, c := pick[qualityapp.Queries, qualityapp.Commands](a.ModeFor("quality"), qualityapp.NewService(), qualityfx.New())
		qualityhttp.Register(a, q, c)
	}
	{
		q, c := pick[nonconformityapp.Queries, nonconformityapp.Commands](a.ModeFor("nonconformity"), nonconformityapp.NewService(), nonconformityfx.New())
		nonconformityhttp.Register(a, q, c)
	}
	{
		q, c := pick[analysisapp.Queries, analysisapp.Commands](a.ModeFor("analysis"), analysisapp.NewService(), analysisfx.New())
		analysishttp.Register(a, q, c)
	}
	{
		q, c := pick[machinelogsapp.Queries, machinelogsapp.Commands](a.ModeFor("machinelogs"), machinelogsapp.NewService(), machinelogsfx.New())
		machinelogshttp.Register(a, q, c)
	}
	{
		q, c := pick[visionapp.Queries, visionapp.Commands](a.ModeFor("vision"), visionapp.NewService(), visionfx.New())
		visionhttp.Register(a, q, c)
	}
	{
		q, c := pick[documentsapp.Queries, documentsapp.Commands](a.ModeFor("documents"), documentsapp.NewService(), documentsfx.New())
		documentshttp.Register(a, q, c)
	}
	{
		q, c := pick[signingapp.Queries, signingapp.Commands](a.ModeFor("signing"), signingapp.NewService(), signingfx.New())
		signinghttp.Register(a, q, c)
	}
	{
		q, c := pick[accessapp.Queries, accessapp.Commands](a.ModeFor("access"), accessapp.NewService(), accessfx.New())
		accesshttp.Register(a, q, c, gate)
	}
	{
		q, c := pick[securityapp.Queries, securityapp.Commands](a.ModeFor("security"), securityapp.NewService(), securityfx.New())
		securityhttp.Register(a, q, c)
	}
	{
		q, c := pick[materialsapp.Queries, materialsapp.Commands](a.ModeFor("materials"), materialsapp.NewService(), materialsfx.New())
		materialshttp.Register(a, q, c)
	}
	{
		q, c := pick[notificationsapp.Queries, notificationsapp.Commands](a.ModeFor("notifications"), notificationsapp.NewService(), notificationsfx.New())
		notificationshttp.Register(a, q, c)
	}
	{
		q, c := pick[analyticsapp.Queries, analyticsapp.Commands](a.ModeFor("analytics"), analyticsapp.NewService(), analyticsfx.New())
		analyticshttp.Register(a, q, c)
	}
	{
		q, c := pick[erpapp.Queries, erpapp.Commands](a.ModeFor("erp"), erpapp.NewService(), erpfx.New())
		erphttp.Register(a, q, c)
	}
	{
		q, c := pick[mesapp.Queries, mesapp.Commands](a.ModeFor("mes"), mesapp.NewService(), mesfx.New())
		meshttp.Register(a, q, c)
	}
	{
		q, c := pick[cadapp.Queries, cadapp.Commands](a.ModeFor("cad"), cadapp.NewService(), cadfx.New())
		cadhttp.Register(a, q, c)
	}
	{
		q, c := pick[federationapp.Queries, federationapp.Commands](a.ModeFor("federation"), federationapp.NewService(), federationfx.New())
		federationhttp.Register(a, q, c)
	}
	{
		q, c := pick[simulationapp.Queries, simulationapp.Commands](a.ModeFor("simulation"), simulationapp.NewService(), simulationfx.New())
		simulationhttp.Register(a, q, c)
	}
	{
		q, c := pick[opsapp.Queries, opsapp.Commands](a.ModeFor("ops"), opsapp.NewService(), opsfx.New())
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
