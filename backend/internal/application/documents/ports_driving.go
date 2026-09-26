package documents

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля documents (AD-36).
type Queries interface {
	// Documents — документы объекта (documents.document.list, FR-65).
	Documents(ctx context.Context, subject platform.DrillRef, m platform.Moment, p platform.Page) (DocumentList, error)
	// Document — документ для подписи (documents.document.read, AD-12, AD-43).
	Document(ctx context.Context, documentID string, version int, m platform.Moment) (DocumentView, error)
	// Render — каноническая отрисовка HTML (documents.document.render, AD-12).
	Render(ctx context.Context, documentID string, version int) (DocumentRendering, error)
	// DecisionCard — карточка «требуется ваше решение» (documents.decision_card.read, FR-136).
	DecisionCard(ctx context.Context, documentID string, m platform.Moment) (DecisionCard, error)
}

// Commands — ведущий порт команд модуля documents (AD-39).
type Commands interface {
	RequestDecision(ctx context.Context, in RequestDecision) (platform.Receipt, error)
	Sign(ctx context.Context, documentID string, in SignDocument) (platform.Receipt, error)
	AttestPaper(ctx context.Context, documentID string, in AttestPaper) (platform.Receipt, error)
	SetPaperStatus(ctx context.Context, documentID string, in SetPaperStatus) (platform.Receipt, error)
	AnnulVersion(ctx context.Context, documentID string, in AnnulVersion) (platform.Receipt, error)
}

// Unimplemented — заглушка портов documents: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Documents(context.Context, platform.DrillRef, platform.Moment, platform.Page) (DocumentList, error) {
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
