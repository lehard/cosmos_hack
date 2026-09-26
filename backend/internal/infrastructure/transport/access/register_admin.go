package access

import (
	"context"

	app "ant/internal/application/access"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

// registerAdmin — администрирование доступа (PRD §3a «Администратор», «Аудитор
// ИБ»; FR-78…FR-85, FR-145; AD-11, AD-15), назначения и допуск к рабочему месту
// (FR-81, FR-83), действия исполнителя у терминала (FR-137).
func registerAdmin(api *httpapi.API, q app.Queries, c app.Commands) {
	const owner = "access"
	emits := func(t ...catalog.Type) []catalog.Type { return t }
	type personIn struct {
		PersonID string `path:"person_id" maxLength:"64" doc:"Псевдоним сотрудника."`
		httpapi.MomentQuery
	}

	httpapi.Read(api, httpapi.Get("/persons", "Сотрудники", "FR-78: сотрудники с условными идентификаторами, учётные записи, роли в областях."),
		platform.Action{ID: "access.person.list", Owner: owner, Subject: "policy"},
		func(ctx context.Context, in *struct {
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.AccessPersonList, error) {
			return q.Persons(ctx, m, in.Page())
		})
	httpapi.Read(api, httpapi.Get("/persons/{person_id}", "Сотрудник", "FR-78: учётная запись и роли сотрудника."),
		platform.Action{ID: "access.person.read", Owner: owner, Subject: "policy"},
		func(ctx context.Context, in *personIn, m platform.Moment) (app.AccessPerson, error) {
			return q.Person(ctx, in.PersonID, m)
		})
	httpapi.Read(api, httpapi.Get("/roles", "Роли и полномочия", "AD-15: роли, наследование, действия, полномочия и виды клейм действующей политики."),
		platform.Action{ID: "access.role.list", Owner: owner, Subject: "policy"},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }, m platform.Moment) (app.AccessRoleList, error) {
			return q.Roles(ctx, m)
		})
	httpapi.Read(api, httpapi.Get("/grants", "История выдачи прав",
		"AD-15: журнал выдачи и отзыва ролей, полномочий, клейм, квалификаций, параметров аудита — для Аудитора ИБ; со второй подписью и CA."),
		platform.Action{ID: "access.grant.list", Owner: owner, Subject: "policy"},
		func(ctx context.Context, in *struct {
			PersonID string `query:"person_id" maxLength:"64" doc:"Сотрудник; пусто — все."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.AccessGrantHistory, error) {
			return q.Grants(ctx, in.PersonID, m, in.Page())
		})
	httpapi.Read(api, httpapi.Get("/stamps", "Цифровые клейма", "FR-145: клейма контролёров — вид контроля, область, приказ, срок."),
		platform.Action{ID: "access.stamp.list", Owner: owner, Subject: "policy"},
		func(ctx context.Context, in *struct {
			PersonID string `query:"person_id" maxLength:"64"`
			httpapi.MomentQuery
		}, m platform.Moment) (app.AccessStampList, error) {
			return q.Stamps(ctx, in.PersonID, m)
		})
	httpapi.Read(api, httpapi.Get("/assignments", "Назначения на посты", "FR-81: кто назначен на какой пост в смене; допуск и квалификация на дату."),
		platform.Action{ID: "access.assignment.list", Owner: owner, Subject: "workplace"},
		func(ctx context.Context, in *struct {
			ShiftID  string `query:"shift_id" maxLength:"128" doc:"Смена; пусто — текущая."`
			Workshop string `query:"workshop" maxLength:"128"`
			httpapi.MomentQuery
		}, m platform.Moment) (app.AccessAssignmentList, error) {
			return q.Assignments(ctx, in.ShiftID, in.Workshop, m)
		})
	httpapi.Read(api, httpapi.Get("/qualifications", "Квалификации", "FR-80: квалификации и аттестации со сроками; проверяются на дату операции."),
		platform.Action{ID: "access.qualification.list", Owner: owner, Subject: "policy"},
		func(ctx context.Context, in *struct {
			PersonID string `query:"person_id" maxLength:"64"`
			httpapi.MomentQuery
		}, m platform.Moment) (app.AccessQualificationList, error) {
			return q.Qualifications(ctx, in.PersonID, m)
		})
	httpapi.Read(api, httpapi.Get("/audit/parameters", "Параметры аудита", "AD-8, AD-15: интервал и предельный разрыв контрольных точек, критические типы, ключ хранителя, подписчики шины."),
		platform.Action{ID: "access.audit.read", Owner: owner, Subject: "policy"},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }, m platform.Moment) (app.AccessAuditParameters, error) {
			return q.Audit(ctx, m)
		})

	// ── команды ──
	httpapi.Do(api, httpapi.Post("/persons", "Завести сотрудника", "FR-78: сотрудник с псевдонимом; соответствие человеку хранится отдельно."),
		platform.Action{ID: "access.person.register", Class: platform.ClassRecord, Owner: owner, Subject: "policy", Emits: emits(catalog.AccessPersonRegistered)},
		func(ctx context.Context, in *struct{ Body app.RegisterPerson }) (platform.Receipt, error) {
			return c.RegisterPerson(ctx, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/persons/{person_id}/account", "Активировать учётную запись", "FR-128: заявка на регистрацию активируется администратором с начальной ролью."),
		platform.Action{ID: "access.account.activate", Class: platform.ClassPermissive, Critical: true, CAGroup: "authority", Owner: owner, Subject: "policy",
			Emits: emits(catalog.AccessAccountActivated), SignatureLevel: 2},
		func(ctx context.Context, in *struct {
			PersonID string `path:"person_id" maxLength:"64"`
			Body     app.ActivateAccount
		}) (platform.Receipt, error) {
			return c.ActivateAccount(ctx, in.PersonID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/grants", "Выдать роль, полномочие или клеймо",
		"AD-11, AD-15, FR-145: выдача документом с маршрутом; полномочия ОТК и клейма — вторая подпись начальника ОТК, производства — руководителя производства, "+
			"администраторов и аудита — Аудитора ИБ; «выдача себе» — только со второй подписью."),
		platform.Action{ID: "access.policy.grant", Class: platform.ClassPermissive, Critical: true, CAGroup: "authority", Owner: owner, Subject: "policy",
			Guards: []string{"self_grant", "second_signature"},
			Emits:  emits(catalog.PolicyRoleAssigned, catalog.PolicyAuthorityGranted, catalog.PolicyStampIssued), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.GrantPolicy }) (platform.Receipt, error) {
			return c.GrantPolicy(ctx, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/grants/revocations", "Отозвать роль, полномочие или клеймо", "AD-15: отзыв доходит до всех копий api; команда по устаревшей политике — 409 journal.stale_policy."),
		platform.Action{ID: "access.policy.revoke", Class: platform.ClassProtective, Critical: true, CAGroup: "authority", Owner: owner, Subject: "policy",
			Emits: emits(catalog.PolicyRoleUnassigned, catalog.PolicyAuthorityRevoked, catalog.PolicyStampRevoked), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.RevokePolicy }) (platform.Receipt, error) {
			return c.RevokePolicy(ctx, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/assignments", "Назначить на пост",
		"FR-81, PRD §11.18: исполнителей назначает мастер (только допущенных по квалификации); контролёра — по документу «запрос мастера → согласование начальника ОТК»."),
		platform.Action{ID: "access.assignment.set", Class: platform.ClassRecord, Owner: owner, Subject: "workplace",
			Guards: []string{"qualification", "controller_approval"}, Emits: emits(catalog.AccessAssignmentSet), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.SetAssignment }) (platform.Receipt, error) {
			return c.SetAssignment(ctx, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/assignments/clear", "Снять с поста", "FR-81."),
		platform.Action{ID: "access.assignment.clear", Class: platform.ClassRecord, Owner: owner, Subject: "workplace", Emits: emits(catalog.AccessAssignmentCleared)},
		func(ctx context.Context, in *struct{ Body app.ClearAssignment }) (platform.Receipt, error) {
			return c.ClearAssignment(ctx, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/persons/{person_id}/qualifications", "Выдать квалификацию", "FR-80: квалификация с областью и сроком."),
		platform.Action{ID: "access.qualification.grant", Class: platform.ClassPermissive, Critical: true, CAGroup: "authority", Owner: owner, Subject: "policy",
			Emits: emits(catalog.AccessQualificationGranted), SignatureLevel: 2},
		func(ctx context.Context, in *struct {
			PersonID string `path:"person_id" maxLength:"64"`
			Body     app.GrantQualification
		}) (platform.Receipt, error) {
			return c.GrantQualification(ctx, in.PersonID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/persons/{person_id}/qualifications/revocations", "Отозвать квалификацию", "FR-80: отзыв снимает допуск к рабочему месту (access.workplace.revoked)."),
		platform.Action{ID: "access.qualification.revoke", Class: platform.ClassProtective, Critical: true, CAGroup: "authority", Owner: owner, Subject: "policy",
			Emits: emits(catalog.AccessQualificationRevoked), SignatureLevel: 2},
		func(ctx context.Context, in *struct {
			PersonID string `path:"person_id" maxLength:"64"`
			Body     app.RevokeQualification
		}) (platform.Receipt, error) {
			return c.RevokeQualification(ctx, in.PersonID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/audit/parameters", "Установить параметры аудита",
		"AD-8, AD-15: только Аудитор ИБ; хранитель берёт интервал и предельный разрыв контрольных точек из этой записи."),
		platform.Action{ID: "access.audit.set_parameters", Class: platform.ClassRecord, Critical: true, CAGroup: "admin_security", Owner: owner, Subject: "policy",
			Emits: emits(catalog.PolicyAuditParametersSet), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.SetAuditParameters }) (platform.Receipt, error) {
			return c.SetAuditParameters(ctx, in.Body)
		})

	type workplaceCmd[B any] struct {
		WorkplaceID string `path:"workplace_id" maxLength:"128" doc:"Рабочее место."`
		Body        B
	}
	httpapi.Do(api, httpapi.Post("/workplaces/{workplace_id}/admission", "Допуск к рабочему месту",
		"Барьер 2 (AD-15, FR-83): СКУД в зоне ∧ роль в области места ∧ квалификация на дату ∧ назначение в смене ∧ токен и PIN."),
		platform.Action{ID: "access.workplace.admit", Class: platform.ClassPermissive, Owner: owner, Subject: "workplace",
			Guards: []string{"zone", "role_scope", "qualification", "assignment", "token"}, Emits: emits(catalog.AccessWorkplaceAdmitted), SignatureLevel: 2},
		func(ctx context.Context, in *workplaceCmd[app.AdmitWorkplace]) (platform.Receipt, error) {
			return c.AdmitWorkplace(ctx, in.WorkplaceID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/workplaces/{workplace_id}/release", "Снять допуск к рабочему месту", "FR-83."),
		platform.Action{ID: "access.workplace.release", Class: platform.ClassRecord, Owner: owner, Subject: "workplace", Emits: emits(catalog.AccessWorkplaceReleased)},
		func(ctx context.Context, in *workplaceCmd[app.ReleaseWorkplace]) (platform.Receipt, error) {
			return c.ReleaseWorkplace(ctx, in.WorkplaceID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/workplaces/{workplace_id}/steps/confirm", "Подтвердить шаг", "FR-137: исполнитель подтверждает шаг ТП у рабочего места (уровень подписи 1)."),
		platform.Action{ID: "access.operator.confirm_step", Class: platform.ClassRecord, Owner: owner, Subject: "workplace", Emits: emits(catalog.OperatorStepConfirmed), SignatureLevel: 1},
		func(ctx context.Context, in *workplaceCmd[app.ConfirmStep]) (platform.Receipt, error) {
			return c.ConfirmStep(ctx, in.WorkplaceID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/workplaces/{workplace_id}/deviations", "Сообщить об отклонении", "FR-137: исполнитель сообщает об отклонении или подозрении на дефект (уровень подписи 1)."),
		platform.Action{ID: "access.operator.report_deviation", Class: platform.ClassRecord, Owner: owner, Subject: "workplace", Emits: emits(catalog.OperatorDeviationReported), SignatureLevel: 1},
		func(ctx context.Context, in *workplaceCmd[app.ReportDeviation]) (platform.Receipt, error) {
			return c.ReportDeviation(ctx, in.WorkplaceID, in.Body)
		})
	httpapi.Do(api, httpapi.Post("/workplaces/{workplace_id}/inspection-requests", "Запросить контроль", "FR-137: исполнитель запрашивает контроль на шаге (уровень подписи 1)."),
		platform.Action{ID: "access.operator.request_inspection", Class: platform.ClassRecord, Owner: owner, Subject: "workplace", Emits: emits(catalog.OperatorInspectionRequested), SignatureLevel: 1},
		func(ctx context.Context, in *workplaceCmd[app.RequestInspection]) (platform.Receipt, error) {
			return c.RequestInspection(ctx, in.WorkplaceID, in.Body)
		})
}
