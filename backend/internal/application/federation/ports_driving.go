package federation

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля federation (AD-36).
type Queries interface {
	// Partners — партнёры (federation.partner.list).
	Partners(ctx context.Context, m platform.Moment) (PartnerList, error)
	// Extracts — выписки паспорта (federation.extract.list, FR-131).
	Extracts(ctx context.Context, direction, partnerCode string, m platform.Moment, p platform.Page) (PassportExtractList, error)
	// Extract — выписка с подписями (federation.extract.read, FR-133).
	Extract(ctx context.Context, extractDigest string) (PassportExtractView, error)
}

// Commands — ведущий порт команд модуля federation (AD-39).
type Commands interface {
	RegisterPartner(ctx context.Context, in RegisterPartner) (platform.Receipt, error)
	SendExtract(ctx context.Context, in SendExtract) (platform.Receipt, error)
	// ReceiveExtract — принять выписку партнёра (federation.extract.receive, FR-132).
	ReceiveExtract(ctx context.Context, in ReceiveExtract) (platform.Receipt, error)
}

// Unimplemented — заглушка портов federation: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Partners(context.Context, platform.Moment) (PartnerList, error) {
	return PartnerList{}, ni("federation.partner.list")
}
func (Unimplemented) Extracts(context.Context, string, string, platform.Moment, platform.Page) (PassportExtractList, error) {
	return PassportExtractList{}, ni("federation.extract.list")
}
func (Unimplemented) Extract(context.Context, string) (PassportExtractView, error) {
	return PassportExtractView{}, ni("federation.extract.read")
}
func (Unimplemented) RegisterPartner(context.Context, RegisterPartner) (platform.Receipt, error) {
	return platform.Receipt{}, ni("federation.partner.register")
}
func (Unimplemented) SendExtract(context.Context, SendExtract) (platform.Receipt, error) {
	return platform.Receipt{}, ni("federation.extract.send")
}

func (Unimplemented) ReceiveExtract(context.Context, ReceiveExtract) (platform.Receipt, error) {
	return platform.Receipt{}, ni("federation.extract.receive")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
