package documents

import (
	"context"

	app "ant/internal/application/documents"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

const owner = "documents"

// Register объявляет операции модуля documents: документы объекта, документ для
// подписи и его отрисовка, карточка редкого подписанта, «Запросить решение»,
// подпись, бумага с заверением, аннулирование (FR-65, FR-66, FR-136, FR-139,
// FR-146; AD-12, AD-13, AD-43).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	httpapi.Read(api, httpapi.Get("/documents", "Документы объекта",
		"FR-65: документы изделия, несоответствия, партии — «документов собрано из истории»; статус маршрута и бумажного экземпляра."),
		platform.Action{ID: "documents.document.list", Owner: owner, Subject: "document"},
		func(ctx context.Context, in *struct {
			Subject platform.EntityKind `query:"subject" required:"true" doc:"Вид объекта."`
			ID      string              `query:"id" required:"true" maxLength:"128" doc:"Идентификатор объекта."`
			httpapi.MomentQuery
			httpapi.PageQuery
		}, m platform.Moment) (app.DocumentList, error) {
			return q.Documents(ctx, platform.DrillRef{Entity: in.Subject, ID: in.ID}, m, in.Page())
		})

	type docIn struct {
		DocumentID string `path:"document_id" maxLength:"128" doc:"Документ."`
		Version    int    `query:"version" minimum:"0" doc:"Версия; 0 — последняя."`
		httpapi.MomentQuery
	}
	httpapi.Read(api, httpapi.Get("/documents/{document_id}", "Документ для подписи",
		"AD-12, AD-43: канонический content, отпечаток doc_digest, rendering_hash, поля сводки уровня 2, замороженный набор обязательных подписей "+
			"с прогрессом и статусом проверки каждой подписи. Агент токена пересчитывает отпечаток сам (AD-14)."),
		platform.Action{ID: "documents.document.read", Owner: owner, Subject: "document"},
		func(ctx context.Context, in *docIn, m platform.Moment) (app.DocumentView, error) {
			return q.Document(ctx, in.DocumentID, in.Version, m)
		})

	httpapi.Read(api, httpapi.Get("/documents/{document_id}/rendering", "Каноническая отрисовка документа",
		"AD-12: HTML по шаблону@версия; серверного PDF нет, печатная рамка (QR, дата печати) в отрисовку не входит."),
		platform.Action{ID: "documents.document.render", Owner: owner, Subject: "document"},
		func(ctx context.Context, in *struct {
			DocumentID string `path:"document_id" maxLength:"128"`
			Version    int    `query:"version" minimum:"0" doc:"Версия; 0 — последняя."`
		}, _ platform.Moment) (app.DocumentRendering, error) {
			return q.Render(ctx, in.DocumentID, in.Version)
		})

	httpapi.Read(api, httpapi.Get("/decision-cards/{document_id}", "Карточка «требуется ваше решение»",
		"FR-136: для редких подписантов (представитель заказчика, согласующий): что решается, почему вы, основания, кто ещё подписывает, что будет после подписи."),
		platform.Action{ID: "documents.decision_card.read", Owner: owner, Subject: "document"},
		func(ctx context.Context, in *struct {
			DocumentID string `path:"document_id" maxLength:"128"`
			httpapi.MomentQuery
		}, m platform.Moment) (app.DecisionCard, error) {
			return q.DecisionCard(ctx, in.DocumentID, m)
		})

	// ── команды ──
	httpapi.Do(api, httpapi.Post("/documents/requests", "Запросить решение",
		"FR-146, FR-136: вместо недоступного действия — документ с маршрутом подписей по шаблону; обязательные подписи вычисляются один раз (AD-43)."),
		platform.Action{ID: "documents.document.request", Class: platform.ClassRecord, Owner: owner, Subject: "document",
			Emits: []catalog.Type{catalog.DocumentVersionRequested}},
		func(ctx context.Context, in *struct{ Body app.RequestDecision }) (platform.Receipt, error) {
			return c.RequestDecision(ctx, in.Body)
		})

	type docCmd[B any] struct {
		DocumentID string `path:"document_id" maxLength:"128"`
		Body       B
	}
	httpapi.Do(api, httpapi.Post("/documents/{document_id}/signatures", "Подписать документ",
		"FR-66, AD-13, AD-14: подпись уровня 2 через агент токена (пакет DSSE в signature) над текущим отпечатком; этап, полномочие, клеймо и разделение обязанностей проверяются по маршруту. "+
			"«Маршрут закрыт» — только реакция document.route.closed (AD-43)."),
		platform.Action{ID: "documents.document.sign", Class: platform.ClassRecord, Owner: owner, Subject: "document",
			Guards: []string{"access.signature_required", "signing.document_changed", "access.separation_of_duties", "access.no_stamp"},
			Emits:  []catalog.Type{catalog.DocumentSignatureRecorded}, SignatureLevel: 2},
		func(ctx context.Context, in *docCmd[app.SignDocument]) (platform.Receipt, error) {
			return c.Sign(ctx, in.DocumentID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/documents/{document_id}/paper-signatures", "Заверить бумажную подпись",
		"AD-43, FR-139: скан распечатки с QR загружен (materials.material.upload); заверитель подписывает уровнем 2, заверитель ≠ подписант; "+
			"полномочие и клеймо проверяются по подписанту на seq заверения. В записи — подписант, заверитель, учётный номер оригинала."),
		platform.Action{ID: "documents.paper.attest", Class: platform.ClassRecord, Owner: owner, Subject: "document",
			Guards: []string{"signing.attester_is_signer", "signing.paper_forbidden", "signing.qr_mismatch"},
			Emits:  []catalog.Type{catalog.DocumentSignatureRecorded}, SignatureLevel: 2},
		func(ctx context.Context, in *docCmd[app.AttestPaper]) (platform.Receipt, error) {
			return c.AttestPaper(ctx, in.DocumentID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/documents/{document_id}/paper-status", "Статус бумажного экземпляра",
		"AD-12: напечатан / подписан / уничтожен — событием; журнал не трогается."),
		platform.Action{ID: "documents.paper.status_set", Class: platform.ClassRecord, Owner: owner, Subject: "document",
			Emits: []catalog.Type{catalog.DocumentPaperStatusChanged}},
		func(ctx context.Context, in *docCmd[app.SetPaperStatus]) (platform.Receipt, error) {
			return c.SetPaperStatus(ctx, in.DocumentID, in.Body)
		})

	httpapi.Do(api, httpapi.Post("/documents/{document_id}/annul", "Аннулировать версию документа",
		"AD-12: документ не редактируется; аннулирование — новая запись с основанием (критическое действие, защищённые данные)."),
		platform.Action{ID: "documents.version.annul", Class: platform.ClassRecord, Critical: true, CAGroup: "protected_data", Owner: owner, Subject: "document",
			Emits: []catalog.Type{catalog.DocumentVersionAnnulled}, SignatureLevel: 2},
		func(ctx context.Context, in *docCmd[app.AnnulVersion]) (platform.Receipt, error) {
			return c.AnnulVersion(ctx, in.DocumentID, in.Body)
		})
}
