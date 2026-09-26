package documents

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля documents (AD-36).
type Queries interface {
	// Documents — документы объекта (documents.document.list, FR-65).
	Documents(ctx context.Context, subject platform.DrillRef, m platform.Moment, p platform.Page) (DocumentList, error)
	// Registry — реестр документов: все документы с отбором по объекту,
	// изделию, процессу, виду, состоянию и поиску (documents.document.list
	// без объекта или с фильтрами; раздел «Документы» столов).
	Registry(ctx context.Context, f DocumentFilter, m platform.Moment, p platform.Page) (DocumentList, error)
	// Document — документ для подписи (documents.document.read, AD-12, AD-43).
	Document(ctx context.Context, documentID string, version int, m platform.Moment) (DocumentView, error)
	// Render — каноническая отрисовка HTML (documents.document.render, AD-12).
	Render(ctx context.Context, documentID string, version int) (DocumentRendering, error)
	// DecisionCard — карточка «требуется ваше решение» (documents.decision_card.read, FR-136).
	DecisionCard(ctx context.Context, documentID string, m platform.Moment) (DecisionCard, error)
	// DecisionRequests — запросы решения, ждущие подписи текущего
	// пользователя (documents.request.list, FR-136, форма эпика 11).
	DecisionRequests(ctx context.Context, m platform.Moment) (DecisionRequestList, error)
	// PrintView — печатная форма версии с рамкой и QR (documents.paper.print_view, FR-139).
	PrintView(ctx context.Context, documentID string, version int) (PrintView, error)
}

// Commands — ведущий порт команд модуля documents (AD-39).
type Commands interface {
	RequestDecision(ctx context.Context, in RequestDecision) (platform.Receipt, error)
	Sign(ctx context.Context, documentID string, in SignDocument) (platform.Receipt, error)
	AttestPaper(ctx context.Context, documentID string, in AttestPaper) (platform.Receipt, error)
	SetPaperStatus(ctx context.Context, documentID string, in SetPaperStatus) (platform.Receipt, error)
	AnnulVersion(ctx context.Context, documentID string, in AnnulVersion) (platform.Receipt, error)
	// RequestVersion — «Запросить решение» / новая версия (documents.version.request).
	RequestVersion(ctx context.Context, in RequestVersion) (RequestAccepted, error)
	// RecordSignature — подпись этапа (documents.signature.record).
	RecordSignature(ctx context.Context, documentID string, in RecordSignature) (platform.Receipt, error)
	// Decline — не согласовать, вернуть с замечанием (documents.signature.decline).
	Decline(ctx context.Context, documentID string, in DeclineSignature) (platform.Receipt, error)
	// Print — напечатать бумажный экземпляр с QR (documents.paper.print).
	Print(ctx context.Context, documentID string, in PrintPaper) (PrintAccepted, error)
}

// Unimplemented — заглушка портов documents: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Documents(context.Context, platform.DrillRef, platform.Moment, platform.Page) (DocumentList, error) {
	return DocumentList{}, ni("documents.document.list")
}
func (Unimplemented) Registry(context.Context, DocumentFilter, platform.Moment, platform.Page) (DocumentList, error) {
	return DocumentList{}, ni("documents.document.list")
}
func (Unimplemented) Document(context.Context, string, int, platform.Moment) (DocumentView, error) {
	return DocumentView{}, ni("documents.document.read")
}
func (Unimplemented) Render(context.Context, string, int) (DocumentRendering, error) {
	return DocumentRendering{}, ni("documents.document.render")
}
func (Unimplemented) DecisionCard(context.Context, string, platform.Moment) (DecisionCard, error) {
	return DecisionCard{}, ni("documents.decision_card.read")
}
func (Unimplemented) DecisionRequests(context.Context, platform.Moment) (DecisionRequestList, error) {
	return DecisionRequestList{}, ni("documents.request.list")
}
func (Unimplemented) PrintView(context.Context, string, int) (PrintView, error) {
	return PrintView{}, ni("documents.paper.print_view")
}
func (Unimplemented) RequestVersion(context.Context, RequestVersion) (RequestAccepted, error) {
	return RequestAccepted{}, ni("documents.version.request")
}
func (Unimplemented) RecordSignature(context.Context, string, RecordSignature) (platform.Receipt, error) {
	return platform.Receipt{}, ni("documents.signature.record")
}
func (Unimplemented) Decline(context.Context, string, DeclineSignature) (platform.Receipt, error) {
	return platform.Receipt{}, ni("documents.signature.decline")
}
func (Unimplemented) Print(context.Context, string, PrintPaper) (PrintAccepted, error) {
	return PrintAccepted{}, ni("documents.paper.print")
}
func (Unimplemented) RequestDecision(context.Context, RequestDecision) (platform.Receipt, error) {
	return platform.Receipt{}, ni("documents.document.request")
}
func (Unimplemented) Sign(context.Context, string, SignDocument) (platform.Receipt, error) {
	return platform.Receipt{}, ni("documents.document.sign")
}
func (Unimplemented) AttestPaper(context.Context, string, AttestPaper) (platform.Receipt, error) {
	return platform.Receipt{}, ni("documents.paper.attest")
}
func (Unimplemented) SetPaperStatus(context.Context, string, SetPaperStatus) (platform.Receipt, error) {
	return platform.Receipt{}, ni("documents.paper.status_set")
}
func (Unimplemented) AnnulVersion(context.Context, string, AnnulVersion) (platform.Receipt, error) {
	return platform.Receipt{}, ni("documents.version.annul")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
