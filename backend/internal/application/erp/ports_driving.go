package erp

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля erp (AD-36).
type Queries interface {
	// Orders — задания учётных систем (erp.order.list).
	Orders(ctx context.Context, m platform.Moment, p platform.Page) (ErpOrderList, error)
	// Messages — исходящие учётные сообщения (erp.message.list).
	Messages(ctx context.Context, f MessageFilter, m platform.Moment, p platform.Page) (ErpMessageList, error)
	// Message — сообщение по бизнес-ключу (erp.message.read).
	Message(ctx context.Context, businessKey string, m platform.Moment) (ErpMessage, error)
	// Channels — состояние каналов обмена (erp.channel.list).
	Channels(ctx context.Context) (ErpChannelList, error)
}

// Commands — ведущий порт команд модуля erp.
type Commands interface {
	ResendPosting(ctx context.Context, businessKey string, in ResendPosting) (platform.Receipt, error)
	CompensatePosting(ctx context.Context, businessKey string, in CompensatePosting) (platform.Receipt, error)
}

// Unimplemented — заглушка портов erp: каждая операция отвечает 501.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Orders(context.Context, platform.Moment, platform.Page) (ErpOrderList, error) {
	return ErpOrderList{}, ni("erp.order.list")
}
func (Unimplemented) Messages(context.Context, MessageFilter, platform.Moment, platform.Page) (ErpMessageList, error) {
	return ErpMessageList{}, ni("erp.message.list")
}
func (Unimplemented) Message(context.Context, string, platform.Moment) (ErpMessage, error) {
	return ErpMessage{}, ni("erp.message.read")
}
func (Unimplemented) Channels(context.Context) (ErpChannelList, error) {
	return ErpChannelList{}, ni("erp.channel.list")
}
func (Unimplemented) ResendPosting(context.Context, string, ResendPosting) (platform.Receipt, error) {
	return platform.Receipt{}, ni("erp.posting.resend")
}
func (Unimplemented) CompensatePosting(context.Context, string, CompensatePosting) (platform.Receipt, error) {
	return platform.Receipt{}, ni("erp.posting.compensate")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
