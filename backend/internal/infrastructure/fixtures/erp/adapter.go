package erp

import (
	"context"

	app "ant/internal/application/erp"
	"ant/internal/application/platform"
)

// Adapter — реализация fixtures ведущих портов модуля erp (AD-36): задания,
// исходящие учётные сообщения 1С с попытками и квитанциями, каналы обмена —
// из мира заготовок. Наружу ничего не отправляется.
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Orders — задания 1С (erp.order.list).
func (Adapter) Orders(ctx context.Context, m platform.Moment, _ platform.Page) (app.ErpOrderList, error) {
	return respond[app.ErpOrderList](ctx, "erp.order.list", nil, &m)
}

// Messages — исходящие сообщения (erp.message.list) с фильтром по изделию, состоянию, системе.
func (Adapter) Messages(ctx context.Context, f app.MessageFilter, m platform.Moment, _ platform.Page) (app.ErpMessageList, error) {
	v, err := respond[app.ErpMessageList](ctx, "erp.message.list",
		map[string]string{"item_id": f.ItemID, "status": f.Status, "system": f.System}, &m)
	if err != nil {
		return v, err
	}
	out := v.Items[:0]
	for _, x := range v.Items {
		if f.ItemID != "" && (x.ItemID == nil || *x.ItemID != f.ItemID) {
			continue
		}
		if (f.Status != "" && x.Status != f.Status) || (f.System != "" && x.ExternalSystem != f.System) {
			continue
		}
		out = append(out, x)
	}
	v.Items = out
	return v, nil
}

// Message — сообщение по бизнес-ключу (erp.message.read).
func (Adapter) Message(ctx context.Context, businessKey string, m platform.Moment) (app.ErpMessage, error) {
	return respond[app.ErpMessage](ctx, "erp.message.read", map[string]string{"business_key": businessKey}, &m)
}

// Channels — каналы обмена (erp.channel.list).
func (Adapter) Channels(ctx context.Context) (app.ErpChannelList, error) {
	return respond[app.ErpChannelList](ctx, "erp.channel.list", nil, nil)
}

// ResendPosting — повторить отправку (erp.posting.resend).
func (Adapter) ResendPosting(ctx context.Context, businessKey string, in app.ResendPosting) (platform.Receipt, error) {
	return decide(ctx, "erp.posting.resend", "erp_message", businessKey, in.CommandMeta())
}

// CompensatePosting — решение о компенсации (erp.posting.compensate).
func (Adapter) CompensatePosting(ctx context.Context, businessKey string, in app.CompensatePosting) (platform.Receipt, error) {
	return decide(ctx, "erp.posting.compensate", "erp_message", businessKey, in.CommandMeta())
}
