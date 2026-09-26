package documents

import (
	"context"
	"net/url"
	"strconv"

	app "ant/internal/application/documents"
	"ant/internal/application/platform"
	dom "ant/internal/domain/documents"
	"ant/internal/infrastructure/fixtures/loader"
)

// Adapter — реализация fixtures ведущих портов модуля documents (AD-36):
// реестр и документы объекта, документ с маршрутом подписей, каноническая
// отрисовка и печатная форма — из мира заготовок (генератор world строит их
// той же отрисовкой domain/documents, что и live); команды подписи, отказа,
// печати и заверения мир не меняют, кроме шага ожидания именно этого решения.
// Карточки редких подписантов — пока 501 (app.Unimplemented).
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

// all — все документы мира на шаге курсора (или на момент as_of).
func all(ctx context.Context, m platform.Moment) (app.DocumentList, error) {
	return respond[app.DocumentList](ctx, opList, nil, &m)
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

// Document — документ с маршрутом подписей (documents.document.read).
func (Adapter) Document(ctx context.Context, documentID string, version int, m platform.Moment) (app.DocumentView, error) {
	return respond[app.DocumentView](ctx, opRead, docParams(documentID, version), &m)
}

// Render — каноническая отрисовка (documents.document.render).
func (Adapter) Render(ctx context.Context, documentID string, version int) (app.DocumentRendering, error) {
	return respond[app.DocumentRendering](ctx, opRender, docParams(documentID, version), nil)
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

// ── команды: квитанция без изменения мира (FR-129) ──

func decide(ctx context.Context, op, id string, meta platform.CommandMeta) (platform.Receipt, error) {
	rt, err := loader.Default()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Decide(ctx, op, loader.ObjectRef{Kind: "document", ID: id}, meta)
}

// Sign — подпись документа (documents.document.sign).
func (Adapter) Sign(ctx context.Context, documentID string, in app.SignDocument) (platform.Receipt, error) {
	return decide(ctx, "documents.document.sign", documentID, in.CommandMeta())
}

// RecordSignature — подпись этапа (documents.signature.record).
func (Adapter) RecordSignature(ctx context.Context, documentID string, in app.RecordSignature) (platform.Receipt, error) {
	return decide(ctx, "documents.signature.record", documentID, in.CommandMeta())
}

// Decline — вернуть с замечанием (documents.signature.decline).
func (Adapter) Decline(ctx context.Context, documentID string, in app.DeclineSignature) (platform.Receipt, error) {
	return decide(ctx, "documents.signature.decline", documentID, in.CommandMeta())
}

// AttestPaper — заверение бумажной подписи (documents.paper.attest).
func (Adapter) AttestPaper(ctx context.Context, documentID string, in app.AttestPaper) (platform.Receipt, error) {
	return decide(ctx, "documents.paper.attest", documentID, in.CommandMeta())
}

// SetPaperStatus — статус бумажного экземпляра (documents.paper.status_set).
func (Adapter) SetPaperStatus(ctx context.Context, documentID string, in app.SetPaperStatus) (platform.Receipt, error) {
	return decide(ctx, "documents.paper.status_set", documentID, in.CommandMeta())
}

// Print — бумажный экземпляр с QR (documents.paper.print): квитанция и адрес печатной формы.
func (Adapter) Print(ctx context.Context, documentID string, in app.PrintPaper) (app.PrintAccepted, error) {
	d, err := Adapter{}.Document(ctx, documentID, in.Version, platform.Moment{})
	if err != nil {
		return app.PrintAccepted{}, err
	}
	r, err := decide(ctx, "documents.paper.print", documentID, in.CommandMeta())
	if err != nil {
		return app.PrintAccepted{}, err
	}
	return app.PrintAccepted{CommandID: r.CommandID, Seq: r.Seq, EventIDs: r.EventIDs, Replayed: r.Replayed, Version: d.Version, DocDigest: d.DocDigest,
		QR: dom.QR(d.DocumentID, d.DocDigest), PrintURL: "/api/v1/documents/" + url.PathEscape(documentID) + "/print?version=" + strconv.Itoa(d.Version)}, nil
}
