package analysis

import (
	"context"

	app "ant/internal/application/analysis"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "analysis"

// Register объявляет операции модуля analysis: разбор обстоятельств, гипотезы,
// похожие случаи, группы и общие факторы, инциденты и область риска, решения
// технолога (FR-58…FR-64, FR-135, FR-153; AD-29, AD-42). Экран — эпик 12.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	type ncIn struct {
		NCID string `path:"nc_id" maxLength:"128" doc:"Несоответствие."`
		httpapi.MomentQuery
	}
	type incidentIn struct {
		IncidentID string `path:"incident_id" maxLength:"128" doc:"Инцидент."`
		httpapi.MomentQuery
	}

	httpapi.Read(api, httpapi.Get("/nonconformities/{nc_id}/circumstances", "Разбор обстоятельств",
		"FR-58, FR-153: проекция analysis.circumstances — события изделия, исполнителя и оборудования на общей шкале, окно возможного возникновения "+
			"(от последнего подтверждённо нормального состояния до первой находки), ссылки на кадры и записи журнала. Формулировка — «возможные обстоятельства», не «причина»."),
		platform.Action{ID: "analysis.circumstances.read", Owner: owner, Subject: "nonconformity"},
		func(ctx context.Context, in *ncIn, m platform.Moment) (app.Circumstances, error) {
			return q.Circumstances(ctx, in.NCID, m)
		})

	httpapi.Read(api, httpapi.Get("/nonconformities/{nc_id}/hypotheses", "Гипотезы причины и похожие случаи",
		"FR-59, FR-60: версия вывода incident.hypothesis.computed и гипотезы, записанные людьми, с доводами «за» и «против»; нехватка сведений; похожие случаи. Гипотеза ≠ причина."),
		platform.Action{ID: "analysis.hypothesis.list", Owner: owner, Subject: "nonconformity"},
		func(ctx context.Context, in *ncIn, m platform.Moment) (app.Hypotheses, error) {
			return q.Hypotheses(ctx, in.NCID, m)
		})

	httpapi.Read(api, httpapi.Get("/nonconformities/{nc_id}/similar", "Похожие случаи",
		"FR-60: прошлые несоответствия по виду дефекта, операции и оборудованию — причина, мера и её результат."),
		platform.Action{ID: "analysis.similar.list", Owner: owner, Subject: "nonconformity"},
		func(ctx context.Context, in *ncIn, m platform.Moment) (app.SimilarCaseList, error) {
			return q.Similar(ctx, in.NCID, m)
		})

	httpapi.Read(api, httpapi.Get("/analysis/groups", "Группы несоответствий",
		"Стол технолога, «Разбор причин»: несоответствия, сгруппированные по виду дефекта × операции × оборудованию, со статусом расследования."),
		platform.Action{ID: "analysis.group.list", Owner: owner, Subject: "nonconformity"},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }, m platform.Moment) (app.NcGroupList, error) {
			return q.Groups(ctx, m)
		})

	httpapi.Read(api, httpapi.Get("/analysis/groups/{group_key}/common-factors", "Общие факторы группы",
		"FR-135: таблица «сколько из N» — станок, инструмент, оснастка, программа, исполнитель, партия материала — со входом в гипотезу и сужение области."),
		platform.Action{ID: "analysis.common_factors.read", Owner: owner, Subject: "nonconformity"},
		func(ctx context.Context, in *struct {
			GroupKey string `path:"group_key" maxLength:"256" doc:"Ключ группы."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.CommonFactors, error) {
			return q.CommonFactors(ctx, in.GroupKey, m)
		})

	httpapi.Read(api, httpapi.Get("/incidents", "Инциденты", "Инциденты с общим фактором, размером и версией области риска (FR-61)."),
		platform.Action{ID: "analysis.incident.list", Owner: owner, Subject: "incident"},
		func(ctx context.Context, in *struct {
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.IncidentList, error) {
			return q.Incidents(ctx, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/incidents/{incident_id}/risk-scope", "Область риска с версиями",
		"FR-61, FR-62: версии области (вычислена, расширена, сужена) с основаниями и доказательствами, разбивка «в производстве / ушли дальше / собраны / отгружены», "+
			"изделия текущей версии с двумя осями: что известно и что делать."),
		platform.Action{ID: "analysis.risk_scope.read", Owner: owner, Subject: "incident"},
		func(ctx context.Context, in *incidentIn, m platform.Moment) (app.RiskScope, error) {
			return q.RiskScope(ctx, in.IncidentID, m)
		})

	// ── команды ──
	emits := func(t ...catalog.Type) []catalog.Type { return t }

	httpapi.Do(api, httpapi.Post("/nonconformities/{nc_id}/hypotheses", "Записать гипотезу", "FR-59: гипотеза причины человеком (почему возник / почему пропустили)."),
		platform.Action{ID: "analysis.hypothesis.record", Class: platform.ClassRecord, Owner: owner, Subject: "nonconformity", Emits: emits(catalog.IncidentHypothesisRecorded)},
		func(ctx context.Context, in *struct {
			NCID string `path:"nc_id" maxLength:"128"`
			Body app.RecordHypothesis
		}) (platform.Receipt, error) {
			return c.RecordHypothesis(ctx, in.NCID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/nonconformities/{nc_id}/hypotheses/reject", "Отклонить гипотезу",
		"FR-59: гипотеза отклоняется с основанием; вывод системы не переписывается — отклонение записывается решением человека."),
		platform.Action{ID: "analysis.hypothesis.reject", Class: platform.ClassRecord, Owner: owner, Subject: "nonconformity", Emits: emits(catalog.IncidentHypothesisRecorded)},
		func(ctx context.Context, in *struct {
			NCID string `path:"nc_id" maxLength:"128"`
			Body app.RejectHypothesis
		}) (platform.Receipt, error) {
			return c.RejectHypothesis(ctx, in.NCID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/nonconformities/{nc_id}/measurements", "Запросить измерение",
		"Проверка гипотезы измерением: задачу исполнителю ставит notifications по записи запроса (тип записи — предложение контракта, см. docs/codegen.md)."),
		platform.Action{ID: "analysis.measurement.request", Class: platform.ClassRecord, Owner: owner, Subject: "nonconformity"},
		func(ctx context.Context, in *struct {
			NCID string `path:"nc_id" maxLength:"128"`
			Body app.RequestMeasurement
		}) (platform.Receipt, error) {
			return c.RequestMeasurement(ctx, in.NCID, in.Body)
		})

	type incidentCmd[B any] struct {
		IncidentID string `path:"incident_id" maxLength:"128"`
		Body       B
	}

	httpapi.Do(api, httpapi.Post("/incidents/{incident_id}/cause", "Подтвердить причину",
		"FR-59: подтвердить гипотезу как причину или «причина не установлена» — необратимое инженерное решение уполномоченного (AD-27), критическое действие (AD-28)."),
		platform.Action{ID: "analysis.cause.conclude", Class: platform.ClassIrreversible, Critical: true, CAGroup: "cause", Owner: owner, Subject: "incident",
			Emits: emits(catalog.IncidentCauseConcluded), SignatureLevel: 2},
		func(ctx context.Context, in *incidentCmd[app.ConcludeCause]) (platform.Receipt, error) {
			return c.ConcludeCause(ctx, in.IncidentID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/incidents/{incident_id}/scope/narrow", "Сузить область риска",
		"FR-61: исключить изделия из области по основаниям — разрешающее действие (AD-27), только человек; новая версия области."),
		platform.Action{ID: "analysis.scope.narrow", Class: platform.ClassPermissive, Critical: true, CAGroup: "risk_scope", Owner: owner, Subject: "incident",
			Emits: emits(catalog.IncidentScopeNarrowed), SignatureLevel: 2},
		func(ctx context.Context, in *incidentCmd[app.ChangeScope]) (platform.Receipt, error) {
			return c.NarrowScope(ctx, in.IncidentID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/incidents/{incident_id}/scope/expand", "Расширить область риска",
		"FR-61: добавить изделия в область — защитное действие (AD-27); новая версия области."),
		platform.Action{ID: "analysis.scope.expand", Class: platform.ClassProtective, Critical: true, CAGroup: "risk_scope", Owner: owner, Subject: "incident",
			Emits: emits(catalog.IncidentScopeExpanded)},
		func(ctx context.Context, in *incidentCmd[app.ChangeScope]) (platform.Receipt, error) {
			return c.ExpandScope(ctx, in.IncidentID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/incidents/{incident_id}/items/assess", "Оценить изделие в инциденте",
		"FR-62: подтверждено / исключено по доказательствам; исключение — разрешающее действие (AD-27)."),
		platform.Action{ID: "analysis.item.assess", Class: platform.ClassPermissive, Critical: true, CAGroup: "risk_scope", Owner: owner, Subject: "incident",
			Emits: emits(catalog.IncidentItemAssessed), SignatureLevel: 2},
		func(ctx context.Context, in *incidentCmd[app.AssessItem]) (platform.Receipt, error) {
			return c.AssessItem(ctx, in.IncidentID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/incidents/{incident_id}/analysis-scope", "Определить глубину разбора",
		"П-02: полный разбор или упрощённый, с основанием."),
		platform.Action{ID: "analysis.analysis.scope", Class: platform.ClassRecord, Owner: owner, Subject: "incident", Emits: emits(catalog.IncidentAnalysisScoped)},
		func(ctx context.Context, in *incidentCmd[app.ScopeAnalysis]) (platform.Receipt, error) {
			return c.ScopeAnalysis(ctx, in.IncidentID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/incidents/{incident_id}/close", "Закрыть инцидент", "Итог: исходный размер области, подтверждено, исключено."),
		platform.Action{ID: "analysis.incident.close", Class: platform.ClassRecord, Owner: owner, Subject: "incident", Emits: emits(catalog.IncidentIncidentClosed)},
		func(ctx context.Context, in *incidentCmd[app.CloseIncident]) (platform.Receipt, error) {
			return c.CloseIncident(ctx, in.IncidentID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/incidents/{incident_id}/actions", "Назначить корректирующее действие", "FR-64: коррекция, корректирующее или предупреждающее действие с планом проверки эффективности."),
		platform.Action{ID: "analysis.action.assign", Class: platform.ClassRecord, Owner: owner, Subject: "incident", Emits: emits(catalog.IncidentActionAssigned)},
		func(ctx context.Context, in *incidentCmd[app.AssignAction]) (platform.Receipt, error) {
			return c.AssignAction(ctx, in.IncidentID, in.Body)
		})

	type actionCmd[B any] struct {
		IncidentID string `path:"incident_id" maxLength:"128"`
		ActionID   string `path:"action_id" maxLength:"128"`
		Body       B
	}
	httpapi.Do(api, httpapi.Post("/incidents/{incident_id}/actions/{action_id}/implemented", "Отметить действие выполненным", "FR-64."),
		platform.Action{ID: "analysis.action.implement", Class: platform.ClassRecord, Owner: owner, Subject: "incident", Emits: emits(catalog.IncidentActionImplemented)},
		func(ctx context.Context, in *actionCmd[app.ImplementAction]) (platform.Receipt, error) {
			return c.ImplementAction(ctx, in.IncidentID, in.ActionID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/incidents/{incident_id}/actions/{action_id}/evaluation", "Оценить эффективность действия", "FR-64: эффективно / неэффективно по плану проверки."),
		platform.Action{ID: "analysis.action.evaluate", Class: platform.ClassRecord, Owner: owner, Subject: "incident", Emits: emits(catalog.IncidentActionEvaluated)},
		func(ctx context.Context, in *actionCmd[app.EvaluateAction]) (platform.Receipt, error) {
			return c.EvaluateAction(ctx, in.IncidentID, in.ActionID, in.Body)
		})
}
