package erp

import (
	"encoding/json"
	"fmt"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/erp"
	"ant/internal/domain/kernel"
)

// Проекции модуля erp (AD-45, писатель — erp; роль projector): чистые
// свёртки журнала по ключу, пересобираются `ant rebuild`. Операции чтения
// erp.* опираются только на них и на состояние каналов (AD-36).
const (
	// ProjectionMessage — исходящее сообщение по бизнес-ключу (dom.View):
	// очередь отправки — проекция журнала (AD-18).
	ProjectionMessage = "erp.message"
	// ProjectionMessageIndex — индекс сообщений системы для списка (ключ — система).
	ProjectionMessageIndex = "erp.message_index"
	// ProjectionItemAccounting — ось «учёт в 1С» изделия (AD-30), ключ — item_id.
	ProjectionItemAccounting = "erp.item_accounting"
	// ProjectionOrder — производственное задание учётной системы, ключ — order_id.
	ProjectionOrder = "erp.order"
	// ProjectionOrderIndex — список заданий (ключ — all).
	ProjectionOrderIndex = "erp.order_index"
)

// Order — задание учётной системы и сколько изделий по нему запущено.
type Order struct {
	OrderID        string    `json:"order_id"`
	ExternalSystem string    `json:"external_system"`
	ExternalNumber string    `json:"external_number"`
	ItemTypeID     string    `json:"item_type_id"`
	ItemRevision   string    `json:"item_revision,omitempty"`
	Quantity       int       `json:"quantity"`
	DueDate        string    `json:"due_date,omitempty"`
	ReceivedAt     time.Time `json:"received_at"`
	Items          []string  `json:"items"`
	Seq            int64     `json:"seq"`
}

// Projections — проекции модуля erp для реестра движка.
func Projections() []engineapp.GlobalProjection {
	return []engineapp.GlobalProjection{
		{Name: ProjectionMessage, Writer: dom.Module, Keys: messageKeys, Step: messageStep,
			Entity: func(k string) (platform.EntityKind, string, bool) { return platform.EntityErpMessage, k, true }},
		{Name: ProjectionMessageIndex, Writer: dom.Module, Keys: indexKeys, Step: indexStep},
		{Name: ProjectionItemAccounting, Writer: dom.Module, Keys: itemKeys, Step: itemStep,
			Entity: func(k string) (platform.EntityKind, string, bool) { return platform.EntityItem, k, true }},
		{Name: ProjectionOrder, Writer: dom.Module, Keys: orderKeys, Step: orderStep},
		{Name: ProjectionOrderIndex, Writer: dom.Module, Keys: orderIndexKeys, Step: orderIndexStep},
	}
}

// MustRegister регистрирует проекции erp в реестре движка (cmd/ant, engineRegistry).
func MustRegister(r *engineapp.Registry) {
	for _, p := range Projections() {
		if err := r.AddGlobal(p); err != nil {
			panic(err)
		}
	}
}

func messageKeys(r kernel.Record) []string {
	k, err := dom.Key(r)
	if err != nil || k == "" {
		return nil
	}
	return []string{k}
}

func messageStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var v dom.View
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &v); err != nil {
			return nil, err
		}
	}
	v, err := v.Apply(r)
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// indexKeys — индекс сообщений для списка: одна строка "all" (последние
// dom.IndexLimit сообщений в порядке формирования).
func indexKeys(r kernel.Record) []string {
	if !dom.Is(r.Type, dom.Exchange) {
		return nil
	}
	return []string{"all"}
}

func indexStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var x dom.Index
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &x); err != nil {
			return nil, err
		}
	}
	key, err := dom.Key(r)
	if err != nil || key == "" {
		return prev, err
	}
	e := dom.IndexEntry{Key: key}
	for _, y := range x.Entries {
		if y.Key == key {
			e = y
			break
		}
	}
	var d struct {
		RequestEventID string `json:"request_event_id"`
		ItemID         string `json:"item_id"`
		LotID          string `json:"lot_id"`
		Outcome        string `json:"outcome"`
		Decision       string `json:"decision"`
	}
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil, err
	}
	current := d.RequestEventID == e.Request
	switch r.Type {
	case catalog.ErpPostingRequested:
		e.ItemID, e.LotID, e.Request, e.Status = d.ItemID, d.LotID, r.EventID, dom.StatusQueued
		if e.FirstSeq == 0 {
			e.FirstSeq = r.Seq
		}
	case catalog.ErpPostingResponded:
		if current {
			e.Status = dom.StatusAcknowledged
			if d.Outcome == string(ev.ErpPostingRespondedV1OutcomeRejected) {
				e.Status = dom.StatusRejected
			}
		}
	case catalog.ErpPostingQuarantined:
		if current {
			e.Status = dom.StatusQuarantined
		}
	case catalog.ErpPostingResendRequested:
		e.Status = dom.StatusQueued
	case catalog.ErpPostingCompensationDecided:
		e.Status = dom.StatusAcknowledged
		if d.Decision == string(ev.ErpPostingCompensationDecidedV1DecisionSendCorrection) {
			e.Status = dom.StatusQueued
		}
	}
	return json.Marshal(x.Put(e))
}

func itemKeys(r kernel.Record) []string {
	it, err := dom.ItemOf(r)
	if err != nil || it == "" {
		return nil
	}
	return []string{it}
}

func itemStep(key string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	a := dom.ItemAccounting{ItemID: key}
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &a); err != nil {
			return nil, err
		}
	}
	a, err := a.Apply(r)
	if err != nil {
		return nil, err
	}
	return json.Marshal(a)
}

func orderKeys(r kernel.Record) []string {
	switch r.Type {
	case catalog.ErpOrderReceived, catalog.ItemItemRegistered:
	default:
		return nil
	}
	var d struct {
		OrderID string `json:"order_id"`
	}
	if json.Unmarshal(r.Data, &d) != nil || d.OrderID == "" {
		return nil
	}
	return []string{d.OrderID}
}

func orderStep(key string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	o := Order{OrderID: key, Items: []string{}}
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &o); err != nil {
			return nil, err
		}
	}
	switch r.Type {
	case catalog.ErpOrderReceived:
		var d ev.ErpOrderReceivedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, err
		}
		o.ExternalSystem, o.ExternalNumber, o.ItemTypeID, o.Quantity = string(d.ExternalSystem), d.ExternalNumber, string(d.ItemTypeID), d.Quantity
		if d.ItemRevision != nil {
			o.ItemRevision = *d.ItemRevision
		}
		if d.DueDate != nil {
			o.DueDate = string(*d.DueDate)
		}
		if o.Seq == 0 {
			o.ReceivedAt, o.Seq = r.OccurredAt, r.Seq
		}
	case catalog.ItemItemRegistered:
		var d ev.ItemItemRegisteredV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, err
		}
		for _, x := range o.Items {
			if x == string(d.ItemID) {
				return json.Marshal(o)
			}
		}
		o.Items = append(o.Items, string(d.ItemID))
	}
	return json.Marshal(o)
}

func orderIndexKeys(r kernel.Record) []string {
	if r.Type != catalog.ErpOrderReceived {
		return nil
	}
	return []string{"all"}
}

func orderIndexStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var ids []string
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &ids); err != nil {
			return nil, err
		}
	}
	var d ev.ErpOrderReceivedV1
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil, fmt.Errorf("erp.order_index: %w", err)
	}
	for _, x := range ids {
		if x == string(d.OrderID) {
			return json.Marshal(ids)
		}
	}
	return json.Marshal(append(ids, string(d.OrderID)))
}
