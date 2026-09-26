package documents

import (
	"context"
	"html"
	"net/url"
	"slices"
	"strconv"
	"strings"

	app "ant/internal/application/documents"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/documents"
	"ant/internal/infrastructure/fixtures/loader"
)

// Adapter — реализация fixtures ведущих портов модуля documents (AD-36):
// реестр и документы объекта, документ с маршрутом подписей, каноническая
// отрисовка и печатная форма — из мира заготовок (генератор world строит их
// той же отрисовкой domain/documents, что и live); команды подписи, отказа,
// печати и заверения мир не меняют, кроме шага ожидания именно этого решения.
// Запросы решения и карточки редких подписантов — из тех же документов по
// правилу live: ближайший незакрытый этап вправе подписать пользователь
// сеанса (app.RequestFromView, app.CardFromView).
type Adapter struct {
	app.Unimplemented
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Операции чтения мира заготовок.
const (
	opList   = "documents.document.list"
	opRead   = "documents.document.read"
	opRender = "documents.document.render"
)

func respond[T any](ctx context.Context, op string, params map[string]string, m *platform.Moment) (T, error) {
	var out T
	rt, err := loader.Default()
	if err != nil {
		return out, err
	}
	err = rt.Respond(ctx, op, params, m, &out)
	return out, err
}

func docParams(id string, version int) map[string]string {
	p := map[string]string{"document_id": id}
	if version > 0 {
		p["version"] = strconv.Itoa(version)
	}
	return p
}

// all — все документы мира на шаге курсора (или на момент as_of) с
// наложением сессии: строки документов с фактами пересчитаны, документы
// «Запросить решение» — сверху.
func all(ctx context.Context, m platform.Moment) (app.DocumentList, error) {
	l, err := respond[app.DocumentList](ctx, opList, nil, &m)
	if err != nil {
		return l, err
	}
	created, items := requested(ctx, &m)
	for i, d := range l.Items {
		if slices.ContainsFunc(created, func(v app.DocumentView) bool { return v.DocumentID == d.DocumentID }) {
			continue
		}
		if fs := facts(ctx, &m, d.DocumentID); len(fs) > 0 {
			if v, err := respond[app.DocumentView](ctx, opRead, docParams(d.DocumentID, 0), &m); err == nil {
				l.Items[i] = summaryOf(d, overlay(v, fs))
			}
		}
	}
	var mine []app.DocumentSummary
	for _, v := range created {
		v = overlay(v, facts(ctx, &m, v.DocumentID))
		row := summaryOfView(v, items[v.DocumentID])
		if i := slices.IndexFunc(l.Items, func(d app.DocumentSummary) bool { return d.DocumentID == v.DocumentID }); i >= 0 {
			row.ItemIDs = l.Items[i].ItemIDs
			l.Items = slices.Delete(l.Items, i, i+1)
		}
		mine = append(mine, row)
	}
	slices.Reverse(mine)
	l.Items = append(mine, l.Items...)
	return l, nil
}

// Documents — документы объекта (documents.document.list, FR-65).
func (Adapter) Documents(ctx context.Context, subject platform.DrillRef, m platform.Moment, p platform.Page) (app.DocumentList, error) {
	return Adapter{}.Registry(ctx, app.DocumentFilter{Subject: subject}, m, p)
}

// Registry — реестр документов с отбором (раздел «Документы» столов).
func (Adapter) Registry(ctx context.Context, f app.DocumentFilter, m platform.Moment, p platform.Page) (app.DocumentList, error) {
	l, err := all(ctx, m)
	if err != nil {
		return l, err
	}
	return app.PageDocuments(app.FilterDocuments(l, f), p), nil
}

// Document — документ с маршрутом подписей (documents.document.read): ответ
// мира или документ «Запросить решение» сессии, поверх — подписи и отказы сессии.
func (Adapter) Document(ctx context.Context, documentID string, version int, m platform.Moment) (app.DocumentView, error) {
	created, _ := requested(ctx, &m)
	for _, v := range created {
		if v.DocumentID == documentID && (version == 0 || version == v.Version) {
			return overlay(v, facts(ctx, &m, documentID)), nil
		}
	}
	v, err := respond[app.DocumentView](ctx, opRead, docParams(documentID, version), &m)
	if err != nil || (version != 0 && version != v.Version) {
		return v, err
	}
	return overlay(v, facts(ctx, &m, documentID)), nil
}

// Render — каноническая отрисовка (documents.document.render); у документа
// «Запросить решение» сессии — простая отрисовка полей сводки.
func (Adapter) Render(ctx context.Context, documentID string, version int) (app.DocumentRendering, error) {
	created, _ := requested(ctx, nil)
	for _, v := range created {
		if v.DocumentID == documentID {
			var b strings.Builder
			b.WriteString("<article><h1>" + html.EscapeString(v.Title) + "</h1><dl>")
			for _, f := range v.SummaryFields {
				b.WriteString("<dt>" + html.EscapeString(f.Label) + "</dt><dd>" + html.EscapeString(f.Value) + "</dd>")
			}
			b.WriteString("</dl></article>")
			return app.DocumentRendering{DocumentID: v.DocumentID, Version: v.Version, RenderingHash: v.RenderingHash, HTML: b.String()}, nil
		}
	}
	return respond[app.DocumentRendering](ctx, opRender, docParams(documentID, version), nil)
}

// DecisionRequests — запросы решения, ждущие подписи пользователя сеанса
// (documents.request.list, FR-136): документы реестра с этапом, который ждёт
// подписи, где пользователь — среди тех, кто вправе подписать.
func (Adapter) DecisionRequests(ctx context.Context, m platform.Moment) (app.DecisionRequestList, error) {
	out := app.DecisionRequestList{Items: []app.DecisionRequest{}}
	l, err := all(ctx, m)
	if err != nil {
		return out, err
	}
	person := platform.PrincipalFrom(ctx).PersonID
	for _, d := range l.Items {
		if d.Awaiting == nil || (person != "" && !slices.Contains(d.Awaiting.Candidates, person)) {
			continue
		}
		v, err := Adapter{}.Document(ctx, d.DocumentID, 0, m)
		if err != nil {
			return out, err
		}
		if rq, ok := app.RequestFromView(v, d.ItemIDs, person); ok {
			out.Items = append(out.Items, rq)
		}
	}
	return out, nil
}

// DecisionCard — карточка «требуется ваше решение» (documents.decision_card.read, FR-136).
func (Adapter) DecisionCard(ctx context.Context, documentID string, m platform.Moment) (app.DecisionCard, error) {
	v, err := Adapter{}.Document(ctx, documentID, 0, m)
	if err != nil {
		return app.DecisionCard{}, err
	}
	var items []string
	if l, err := all(ctx, m); err == nil {
		for _, d := range l.Items {
			if d.DocumentID == documentID {
				items = d.ItemIDs
			}
		}
	}
	card, ok := app.CardFromView(v, items, platform.PrincipalFrom(ctx).PersonID)
	if !ok {
		return app.DecisionCard{}, platform.Fail(errcodes.ApiNotFound, "document_id", documentID, "reason", "версия не ждёт подписей")
	}
	return card, nil
}

// PrintView — печатная форма (documents.paper.print_view, FR-139): отрисовка
// заготовки в печатной рамке с QR и датой печати — часы шага курсора.
func (Adapter) PrintView(ctx context.Context, documentID string, version int) (app.PrintView, error) {
	r, err := Adapter{}.Render(ctx, documentID, version)
	if err != nil {
		return app.PrintView{}, err
	}
	d, err := Adapter{}.Document(ctx, documentID, version, platform.Moment{})
	if err != nil {
		return app.PrintView{}, err
	}
	rt, err := loader.Default()
	if err != nil {
		return app.PrintView{}, err
	}
	now, err := rt.Clock(ctx)
	if err != nil {
		return app.PrintView{}, err
	}
	qr := dom.QR(d.DocumentID, d.DocDigest)
	svg, err := app.QRSVG(qr)
	if err != nil {
		return app.PrintView{}, err
	}
	return app.PrintView{DocumentID: d.DocumentID, Version: r.Version, DocDigest: d.DocDigest, RenderingHash: r.RenderingHash, QR: qr, QRSVG: svg,
		PrintedAt: now, HTML: app.PrintFrame(r.HTML, qr, svg, dom.FormatTime(now))}, nil
}

// ── команды: квитанция и факт сессии поверх мира (FR-129, loader.Runtime.Record) ──

// record — команда над документом: квитанция и факт сессии с телом команды.
func record(ctx context.Context, op, id string, meta platform.CommandMeta, body any) (platform.Receipt, error) {
	rt, err := loader.Default()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Record(ctx, op, loader.ObjectRef{Kind: kindDocument, ID: id}, meta, body,
		loader.Change{Entity: kindDocument, ID: "global"}, loader.Change{Entity: string(platform.EntityNotification), ID: "global"})
}

// exists — документ есть (в мире или в сессии): иначе 404, а не пустая квитанция.
func exists(ctx context.Context, id string) error {
	_, err := Adapter{}.Document(ctx, id, 0, platform.Moment{})
	return err
}

// Sign — подпись документа (documents.document.sign): этап маршрута закрыт в сессии.
func (Adapter) Sign(ctx context.Context, documentID string, in app.SignDocument) (platform.Receipt, error) {
	if err := exists(ctx, documentID); err != nil {
		return platform.Receipt{}, err
	}
	return record(ctx, opSign, documentID, in.CommandMeta(), in)
}

// RecordSignature — подпись этапа (documents.signature.record).
func (Adapter) RecordSignature(ctx context.Context, documentID string, in app.RecordSignature) (platform.Receipt, error) {
	if err := exists(ctx, documentID); err != nil {
		return platform.Receipt{}, err
	}
	return record(ctx, opRecord, documentID, in.CommandMeta(), in)
}

// Decline — вернуть с замечанием (documents.signature.decline): версия «возвращена».
func (Adapter) Decline(ctx context.Context, documentID string, in app.DeclineSignature) (platform.Receipt, error) {
	if err := exists(ctx, documentID); err != nil {
		return platform.Receipt{}, err
	}
	return record(ctx, opDecline, documentID, in.CommandMeta(), in)
}

// AttestPaper — заверение бумажной подписи (documents.paper.attest): подпись
// ручкой засчитана этапу с заверителем, номером оригинала и адресом скана.
func (Adapter) AttestPaper(ctx context.Context, documentID string, in app.AttestPaper) (platform.Receipt, error) {
	if err := exists(ctx, documentID); err != nil {
		return platform.Receipt{}, err
	}
	return record(ctx, opAttest, documentID, in.CommandMeta(), in)
}

// SetPaperStatus — статус бумажного экземпляра (documents.paper.status_set).
func (Adapter) SetPaperStatus(ctx context.Context, documentID string, in app.SetPaperStatus) (platform.Receipt, error) {
	if err := exists(ctx, documentID); err != nil {
		return platform.Receipt{}, err
	}
	return record(ctx, opPaper, documentID, in.CommandMeta(), in)
}

// RequestVersion — «Запросить решение» / новая версия документа
// (documents.version.request, FR-146): документ сессии с маршрутом к тем, у
// кого есть полномочие на действие; он виден в реестре и у подписантов в
// «Требуется ваше решение» до сброса прогона.
func (Adapter) RequestVersion(ctx context.Context, in app.RequestVersion) (app.RequestAccepted, error) {
	rt, err := loader.Default()
	if err != nil {
		return app.RequestAccepted{}, err
	}
	now, err := rt.Clock(ctx)
	if err != nil {
		return app.RequestAccepted{}, err
	}
	v, items, err := newRequest(ctx, in, now)
	if err != nil {
		return app.RequestAccepted{}, err
	}
	rc, err := record(ctx, opRequest, v.DocumentID, in.CommandMeta(), requestedDoc{View: v, Items: items})
	if err != nil {
		return app.RequestAccepted{}, err
	}
	return app.RequestAccepted{CommandID: rc.CommandID, Seq: rc.Seq, EventIDs: rc.EventIDs, Replayed: rc.Replayed, DocumentID: v.DocumentID, Version: v.Version, DocDigest: v.DocDigest}, nil
}

// Print — бумажный экземпляр с QR (documents.paper.print): квитанция и адрес печатной формы.
func (Adapter) Print(ctx context.Context, documentID string, in app.PrintPaper) (app.PrintAccepted, error) {
	d, err := Adapter{}.Document(ctx, documentID, in.Version, platform.Moment{})
	if err != nil {
		return app.PrintAccepted{}, err
	}
	r, err := record(ctx, opPrint, documentID, in.CommandMeta(), in)
	if err != nil {
		return app.PrintAccepted{}, err
	}
	return app.PrintAccepted{CommandID: r.CommandID, Seq: r.Seq, EventIDs: r.EventIDs, Replayed: r.Replayed, Version: d.Version, DocDigest: d.DocDigest,
		QR: dom.QR(d.DocumentID, d.DocDigest), PrintURL: "/api/v1/documents/" + url.PathEscape(documentID) + "/print?version=" + strconv.Itoa(d.Version)}, nil
}
