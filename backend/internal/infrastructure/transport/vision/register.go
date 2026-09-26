package vision

import (
	"context"

	"ant/internal/application/platform"
	app "ant/internal/application/vision"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "vision"

// Register объявляет операции модуля vision: анализаторы VisionQC и
// OperatorVision, паспорта допуска и отчёты проверки, допуск, возврат после
// отката и вывод из действия (FR-97…FR-101, FR-126; AD-29). Экраны — эпики 33, 40.
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	emits := func(t ...catalog.Type) []catalog.Type { return t }

	httpapi.Read(api, httpapi.Get("/analyzers", "Анализаторы",
		"FR-97, FR-126: анализаторы визуального контроля и контроля действий оператора — версии, стадия допуска, уровень доверия паспорта (AD-29)."),
		platform.Action{ID: "vision.analyzer.list", Owner: owner, Subject: "analyzer_passport"},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }, m platform.Moment) (app.AnalyzerList, error) {
			return q.Analyzers(ctx, m)
		})

	httpapi.Read(api, httpapi.Get("/analyzer-passports/{passport_id}", "Паспорт допуска",
		"FR-98: паспорт допуска карты контроля — стадия, уровень доверия и допустимые автоматические действия, приостановка (откат) и её триггер."),
		platform.Action{ID: "vision.passport.read", Owner: owner, Subject: "analyzer_passport"},
		func(ctx context.Context, in *struct {
			PassportID string `path:"passport_id" maxLength:"128" doc:"Паспорт допуска."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.AnalyzerPassport, error) {
			return q.Passport(ctx, in.PassportID, m)
		})

	httpapi.Read(api, httpapi.Get("/analyzer-passports/{passport_id}/checks", "Отчёты проверки анализатора",
		"FR-99, FR-100: экзамен, эталонный набор, теневое сравнение, контроль дрейфа — доли пропусков, ложных тревог и расхождений с людьми в б. п."),
		platform.Action{ID: "vision.check.list", Owner: owner, Subject: "analyzer_passport"},
		func(ctx context.Context, in *struct {
			PassportID string `path:"passport_id" maxLength:"128" doc:"Паспорт допуска."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.AnalyzerCheckList, error) {
			return q.Checks(ctx, in.PassportID, m, in.Page())
		})

	httpapi.Do(api, httpapi.Post("/analyzer-passports", "Допустить версию анализатора",
		"FR-98: допуск по закрытому маршруту протокола допуска (начальник ОТК + технолог + метролог) — разрешающее действие, изменение контроля (AD-28)."),
		platform.Action{ID: "vision.passport.admit", Class: platform.ClassPermissive, Critical: true, CAGroup: "control_change", Owner: owner,
			Subject: "analyzer_passport", Emits: emits(catalog.AnalyzerPassportAdmitted), SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.AdmitPassport }) (platform.Receipt, error) {
			return c.AdmitPassport(ctx, in.Body)
		})

	type passportCmd[B any] struct {
		PassportID string `path:"passport_id" maxLength:"128"`
		Body       B
	}
	httpapi.Do(api, httpapi.Post("/analyzer-passports/{passport_id}/reinstate", "Вернуть анализатор после отката",
		"FR-101, AD-29: возврат после приостановки — разрешающее действие начальника ОТК (analyzer.reinstate_requires_head_of_qc)."),
		platform.Action{ID: "vision.passport.reinstate", Class: platform.ClassPermissive, Critical: true, CAGroup: "control_change", Owner: owner,
			Subject: "analyzer_passport", Emits: emits(catalog.AnalyzerPassportReinstated), SignatureLevel: 2},
		func(ctx context.Context, in *passportCmd[app.ReinstatePassport]) (platform.Receipt, error) {
			return c.ReinstatePassport(ctx, in.PassportID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/analyzer-passports/{passport_id}/retire", "Вывести паспорт из действия",
		"Паспорт выводится; старые наблюдения сохраняют «проанализировано версией …»."),
		platform.Action{ID: "vision.passport.retire", Class: platform.ClassRecord, Critical: true, CAGroup: "control_change", Owner: owner,
			Subject: "analyzer_passport", Emits: emits(catalog.AnalyzerPassportRetired), SignatureLevel: 2},
		func(ctx context.Context, in *passportCmd[app.RetirePassport]) (platform.Receipt, error) {
			return c.RetirePassport(ctx, in.PassportID, in.Body)
		})
}
