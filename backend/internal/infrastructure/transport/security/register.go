package security

import (
	"context"

	"ant/internal/application/platform"
	app "ant/internal/application/security"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "security"

// Register объявляет операции модуля security: индикатор целостности, журнал
// критических действий, шина безопасности, отчёты верификатора (FR-72…FR-77,
// FR-118, FR-146; AD-24, AD-28, AD-46). Только чтение: записей CA через API не
// создают (AD-28).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	_ = c
	httpapi.Register(api, httpapi.Get("/integrity", "Состояние целостности журнала «по данным сервера»",
		"AD-46: ant забирает последний подписанный отчёт верификатора у хранителя и журналирует security.integrity.checked. "+
			"Индикатор на столах помечен «по данным сервера» и желтеет сам, если свежего отчёта нет дольше двух интервалов."),
		platform.Action{ID: "security.integrity.read", Class: platform.ClassRead, Owner: owner, Subject: "integrity"},
		func(ctx context.Context, _ *struct{}) (*httpapi.Out[app.IntegrityStatus], error) {
			v, err := q.Integrity(ctx)
			return httpapi.OK(v), err
		})

	httpapi.Read(api, httpapi.Get("/critical-actions", "Журнал критических действий",
		"AD-28, FR-77: CA-‹n› — действие, объект, было → стало, кто, полномочие и клеймо с ревизией политики, основание, ссылка на подписанную запись "+
			"основного журнала; отмена — только новой записью. Стол Аудитора ИБ."),
		platform.Action{ID: "security.critical_action.list", Owner: owner, Subject: "integrity"},
		func(ctx context.Context, in *struct {
			Group   string              `query:"group" enum:"product_decision,nc_decision,cause,risk_scope,control_change,authority,protected_data,admin_security" doc:"Группа CA."`
			ActorID string              `query:"actor_id" maxLength:"64" doc:"Кто."`
			Subject platform.EntityKind `query:"subject" doc:"Вид объекта."`
			ID      string              `query:"id" maxLength:"128" doc:"Объект."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.CriticalActionList, error) {
			f := app.CriticalActionFilter{Group: in.Group, ActorID: in.ActorID}
			if in.Subject != "" {
				f.Object = &platform.DrillRef{Entity: in.Subject, ID: in.ID}
			}
			return q.CriticalActions(ctx, f, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/critical-actions/{ca_ref}", "Критическое действие",
		"AD-28: запись CA-‹n› и её связь с подписанной записью основного журнала (event_id, commit); верификатор сверяет их один к одному."),
		platform.Action{ID: "security.critical_action.read", Owner: owner, Subject: "integrity"},
		func(ctx context.Context, in *struct {
			CARef string `path:"ca_ref" pattern:"^CA-[1-9][0-9]*$" doc:"CA-‹n›."`
		}, _ platform.Moment) (app.CriticalAction, error) {
			return q.CriticalAction(ctx, in.CARef)
		})

	httpapi.Read(api, httpapi.Get("/security-events", "Шина безопасности",
		"AD-24, FR-118: ошибки аутентификации, недействительные подписи, нарушения целостности, отказы в доступе и допуске, конфликты идемпотентности, "+
			"выдача привилегий, отзыв ключей, тревоги хранителя и агента."),
		platform.Action{ID: "security.event.list", Owner: owner, Subject: "integrity"},
		func(ctx context.Context, in *struct {
			EventType string `query:"event_type" maxLength:"128" doc:"Тип записи семейства security."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.SecurityEventList, error) {
			return q.Events(ctx, in.EventType, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/verifier-reports", "Отчёты верификатора",
		"AD-9, AD-46: подписанные отчёты независимого верификатора, полученные у хранителя; вердикт «цело» / «цело с оговорками» / «нарушено»."),
		platform.Action{ID: "security.verifier_report.list", Owner: owner, Subject: "integrity"},
		func(ctx context.Context, in *struct{ httpapi.PageQuery }, _ platform.Moment) (app.VerifierReportList, error) {
			return q.VerifierReports(ctx, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/verifier-reports/{report_digest}", "Отчёт верификатора",
		"AD-9: проверки со статусами «цело» / «отвергнуто» / «не проверяемо», классы подписей раздельно, реестр бумажных решений."),
		platform.Action{ID: "security.verifier_report.read", Owner: owner, Subject: "integrity"},
		func(ctx context.Context, in *struct {
			ReportDigest string `path:"report_digest" maxLength:"128" doc:"Отпечаток отчёта."`
		}, _ platform.Moment) (app.VerifierReport, error) {
			return q.VerifierReport(ctx, in.ReportDigest)
		})
}
