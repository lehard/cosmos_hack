package signing

import (
	"context"

	"ant/internal/application/platform"
	app "ant/internal/application/signing"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "signing"

// Register объявляет операции модуля signing: реестр ключей и актов,
// криптопрофили, регистрация и отзыв ключа (FR-67…FR-70, FR-76, FR-79;
// AD-10, AD-11, AD-32).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	httpapi.Read(api, httpapi.Get("/keys", "Реестр ключей",
		"FR-79, AD-11: ключи людей, устройств, движка, шлюзов, хранителя, верификатора и корни партнёров — с профилем, отпечатком, статусом и актами. "+
			"Закрытых ключей людей и устройств на сервере нет."),
		platform.Action{ID: "signing.key.list", Owner: owner},
		func(ctx context.Context, in *struct {
			SubjectKind string `query:"subject_kind" enum:"person,device,engine,gateway,enterprise_gateway,keeper,verifier,demo_persona,partner_root" doc:"Вид субъекта."`
			SubjectID   string `query:"subject_id" maxLength:"128" doc:"Субъект."`
			Status      string `query:"status" enum:"active,revoked,expired,unavailable" doc:"Статус."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.KeyList, error) {
			return q.Keys(ctx, in.SubjectKind, in.SubjectID, in.Status, m, in.Page())
		})

	httpapi.Read(api, httpapi.Get("/keys/{key_ref}", "Ключ и его акты",
		"AD-11: акт регистрации с подписями (владение, подтверждение субъекта, вторая подпись независимой стороны), история ротаций, отзыв и «скомпрометирован с»."),
		platform.Action{ID: "signing.key.read", Owner: owner},
		func(ctx context.Context, in *struct {
			KeyRef string `path:"key_ref" maxLength:"128" doc:"key_id@версия."`
			httpapi.MomentQuery
		}, m platform.Moment) (app.KeyDetails, error) {
			return q.Key(ctx, in.KeyRef, m)
		})

	httpapi.Read(api, httpapi.Get("/crypto-profiles", "Криптопрофили",
		"AD-32, кейс §6.3: gost, pq (демонстрационный), hybrid; объект → обязательный профиль; с какой записи действует."),
		platform.Action{ID: "signing.profile.list", Owner: owner},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }, m platform.Moment) (app.CryptoProfileList, error) {
			return q.Profiles(ctx, m)
		})

	httpapi.Do(api, httpapi.Post("/keys", "Зарегистрировать ключ",
		"AD-11, FR-70, FR-79: акт регистрации с доказательством владения и подтверждением субъекта; вторая подпись — от независимой стороны "+
			"(ОТК — начальник ОТК, производство — руководитель производства, администраторы и аудит — Аудитор ИБ). Разрешающее критическое действие."),
		platform.Action{ID: "signing.key.register", Class: platform.ClassPermissive, Critical: true, CAGroup: "admin_security", Owner: owner,
			Guards: []string{"access.self_grant", "access.separation_of_duties"},
			Emits:  []catalog.Type{catalog.KeyRegistrationRecorded}, SignatureLevel: 2},
		func(ctx context.Context, in *struct{ Body app.RegisterKey }) (platform.Receipt, error) {
			return c.RegisterKey(ctx, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/keys/{key_ref}/revocation", "Отозвать ключ",
		"AD-11: отзыв с «скомпрометирован с X» (X может быть раньше даты отзыва) — защитная реакция: изделиям с решениями, подписанными этим ключом после X, "+
			"— сдерживание «подпись под сомнением» и задачи на переподписание."),
		platform.Action{ID: "signing.key.revoke", Class: platform.ClassProtective, Critical: true, CAGroup: "admin_security", Owner: owner,
			Emits: []catalog.Type{catalog.KeyRevocationRecorded}, SignatureLevel: 2},
		func(ctx context.Context, in *struct {
			KeyRef string `path:"key_ref" maxLength:"128"`
			Body   app.RevokeKey
		}) (platform.Receipt, error) {
			return c.RevokeKey(ctx, in.KeyRef, in.Body)
		})
}
