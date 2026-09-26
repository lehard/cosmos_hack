package federation

import (
	"context"

	app "ant/internal/application/federation"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "federation"

// Register объявляет операции модуля federation: партнёры, выписки паспорта,
// регистрация партнёра, отправка выписки (FR-131…FR-134; AD-19).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	httpapi.Read(api, httpapi.Get("/partners", "Предприятия-партнёры",
		"AD-19: партнёр — такая же копия ant со своим кодом предприятия; доверие ключам — от нашего акта регистрации партнёра и его корней."),
		platform.Action{ID: "federation.partner.list", Owner: owner, Subject: "lot"},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }, m platform.Moment) (app.PartnerList, error) {
			return q.Partners(ctx, m)
		})

	httpapi.Read(api, httpapi.Get("/passport-extracts", "Выписки паспорта",
		"FR-131, FR-132: входящие (через приём, source_id = partner:‹код›) и исходящие выписки; происхождение подтверждено / только сервером отправителя / не подтверждено."),
		platform.Action{ID: "federation.extract.list", Owner: owner, Subject: "lot"},
		func(ctx context.Context, in *struct {
			Direction   string `query:"direction" enum:"incoming,outgoing" doc:"Направление."`
			PartnerCode string `query:"partner_code" maxLength:"16" doc:"Партнёр."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.PassportExtractList, error) {
			return q.Extracts(ctx, in.Direction, in.PartnerCode, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/passport-extracts/{extract_digest}", "Выписка паспорта",
		"FR-133, AD-19: содержимое, подписи класса partner с проверкой цепочкой к корням партнёра, контрольная точка хранителя отправителя."),
		platform.Action{ID: "federation.extract.read", Owner: owner, Subject: "lot"},
		func(ctx context.Context, in *struct {
			ExtractDigest string `path:"extract_digest" maxLength:"160" doc:"Отпечаток выписки."`
		}, _ platform.Moment) (app.PassportExtractView, error) {
			return q.Extract(ctx, in.ExtractDigest)
		})

	httpapi.Do(api, httpapi.Post("/partners", "Зарегистрировать партнёра",
		"AD-19, AD-11: акт регистрации партнёра и его корней — администратор безопасности + вторая подпись начальника ОТК; разрешающее критическое действие."),
		platform.Action{ID: "federation.partner.register", Class: platform.ClassPermissive, Critical: true, CAGroup: "admin_security", Owner: owner, Subject: "lot",
			Emits: []catalog.Type{catalog.FederationPartnerRegistered}, SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.RegisterPartner }) (platform.Receipt, error) {
			return c.RegisterPartner(ctx, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/passport-extracts", "Отправить выписку паспорта партнёру",
		"FR-131, AD-19: исходящая выписка — документ с маршрутом «контролёр ОТК (2) + ключ шлюза предприятия»; отправляет роль outbox с досылкой, квитанция — событием."),
		platform.Action{ID: "federation.extract.send", Class: platform.ClassRecord, Owner: owner, Subject: "lot",
			Emits: []catalog.Type{catalog.FederationMessageSent}},
		func(ctx context.Context, in *struct{ Body app.SendExtract }) (platform.Receipt, error) {
			return c.SendExtract(ctx, in.Body)
		})

	registerReceive(api, c)
}

// registerReceive — порт межзаводского обмена: приём выписки партнёра (эпик 41).
func registerReceive(api *httpapi.API, c app.Commands) {
	httpapi.Do(api, httpapi.Post("/passport-extracts/incoming", "Принять выписку паспорта партнёра",
		"FR-132, AD-19: получатель сам проверяет подписи выписки цепочкой к корням партнёра из нашего акта регистрации; изменённая — 422 federation.extract_tampered; непроверяемая — принята с пометкой «происхождение не подтверждено»; принятая — корень генеалогии партии."),
		platform.Action{ID: "federation.extract.receive", Class: platform.ClassRecord, Owner: owner, Subject: "lot",
			Emits: []catalog.Type{catalog.FederationExtractReceived}},
		func(ctx context.Context, in *struct{ Body app.ReceiveExtract }) (platform.Receipt, error) {
			return c.ReceiveExtract(ctx, in.Body)
		})
}
