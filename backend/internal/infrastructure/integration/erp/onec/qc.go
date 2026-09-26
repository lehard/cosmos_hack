package onec

import (
	"strings"
	"time"

	app "ant/internal/application/erp"
	dom "ant/internal/domain/erp"
)

// Сообщения HTTP-сервиса qc.v1 (contracts/integrations/erp/1c/qc.v1). Типы —
// рукописные по схемам; соответствие схемам проверяет контрактный тест на
// эталонных сообщениях и проверка каждого сообщения перед отправкой.

// Source — источник сообщения и бизнес-ключ.
type Source struct {
	System         string   `json:"system"`
	Enterprise     string   `json:"enterprise"`
	BusinessKey    string   `json:"business_key"`
	MessageVersion int      `json:"message_version"`
	ClosingPoint   string   `json:"closing_point,omitempty"`
	StepKey        string   `json:"step_key,omitempty"`
	BasisEventIDs  []string `json:"basis_event_ids,omitempty"`
}

// Item — изделие.
type Item struct {
	ItemID             string `json:"item_id"`
	Designation        string `json:"designation,omitempty"`
	SerialNo           string `json:"serial_no,omitempty"`
	NomenclatureRefKey string `json:"nomenclature_ref_key,omitempty"`
	SeriesRefKey       string `json:"series_ref_key,omitempty"`
}

// Lot — партия.
type Lot struct {
	LotID        string `json:"lot_id"`
	Quantity     int    `json:"quantity,omitempty"`
	SeriesRefKey string `json:"series_ref_key,omitempty"`
}

// Warehouse — склад по коду.
type Warehouse struct {
	Code   string `json:"code"`
	RefKey string `json:"ref_key,omitempty"`
}

// Document — документ-основание ant.
type Document struct {
	Kind   string `json:"kind"`
	Number string `json:"number,omitempty"`
	Digest string `json:"digest,omitempty"`
}

// Posting — тело POST /hs/qc/v1/postings.
type Posting struct {
	MessageID         string     `json:"message_id"`
	Contract          string     `json:"contract"`
	Kind              string     `json:"kind"`
	DefectKind        string     `json:"defect_kind,omitempty"`
	OccurredAt        string     `json:"occurred_at"`
	Source            Source     `json:"source"`
	Item              *Item      `json:"item,omitempty"`
	Lot               *Lot       `json:"lot,omitempty"`
	WarehouseFrom     *Warehouse `json:"warehouse_from,omitempty"`
	WarehouseTo       *Warehouse `json:"warehouse_to,omitempty"`
	OrderID           string     `json:"order_id,omitempty"`
	ConcessionNumber  string     `json:"concession_number,omitempty"`
	AfterRework       *bool      `json:"after_rework,omitempty"`
	ClaimBasis        string     `json:"claim_basis,omitempty"`
	Nonconformities   []string   `json:"nonconformities,omitempty"`
	Document          *Document  `json:"document,omitempty"`
	CorrectsMessageID string     `json:"corrects_message_id,omitempty"`
}

// InspectionResult — тело POST /hs/qc/v1/inspection-results.
type InspectionResult struct {
	MessageID         string   `json:"message_id"`
	Contract          string   `json:"contract"`
	OccurredAt        string   `json:"occurred_at"`
	Source            Source   `json:"source"`
	Item              *Item    `json:"item,omitempty"`
	Lot               *Lot     `json:"lot,omitempty"`
	ClosingPoint      string   `json:"closing_point"`
	PresentationNo    int      `json:"presentation_no,omitempty"`
	Result            string   `json:"result"`
	ConcessionNumber  string   `json:"concession_number,omitempty"`
	Nonconformities   []string `json:"nonconformities,omitempty"`
	CorrectsMessageID string   `json:"corrects_message_id,omitempty"`
}

// Receipt — квитанция qc.v1.
type Receipt struct {
	MessageID      string `json:"message_id"`
	Status         string `json:"status"`
	Receipt        string `json:"receipt"`
	DocumentRefKey string `json:"document_ref_key"`
	DocumentType   string `json:"document_type,omitempty"`
	DocumentNumber string `json:"document_number,omitempty"`
	ReceivedAt     string `json:"received_at,omitempty"`
}

// Error — ошибка qc.v1.
type Error struct {
	MessageID string `json:"message_id,omitempty"`
	Status    string `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Field     string `json:"field,omitempty"`
}

// About — описание сервиса qc.
type About struct {
	Service       string   `json:"service"`
	Contract      string   `json:"contract"`
	Supported     []string `json:"supported"`
	Configuration string   `json:"configuration,omitempty"`
}

// Kind — вид qc.v1 для учётного действия (словарь — contracts/integrations/erp/1c/README.md).
func Kind(a dom.Action) (kind, defect string) {
	switch a {
	case dom.AcceptIntoWork:
		return "accepted_to_work", ""
	case dom.WarehouseTransfer:
		return "warehouse_transfer", ""
	case dom.ScrapRework:
		return "moved_to_defect", "rework"
	case dom.ScrapWriteoff:
		return "moved_to_defect", "scrap"
	case dom.ScrapReprocess:
		return "moved_to_defect", "reprocessing"
	case dom.ReturnToSupplier:
		return "returned_to_supplier", ""
	case dom.ReturnFromDefect:
		return "returned_from_defect", ""
	case dom.Release:
		return "released", ""
	}
	return "", ""
}

// Result — итог контроля qc.v1 по решению на закрывающей точке.
func Result(resolution string) string {
	switch resolution {
	case "accept":
		return "conforming"
	case "accept_with_concession":
		return "conforming_with_concession"
	case "accept_partially":
		return "partially_conforming"
	case "reject":
		return "nonconforming"
	}
	return "insufficient_data"
}

const timeLayout = "2006-01-02T15:04:05.000Z"

// Encode — сообщение qc.v1 по исходящему сообщению порта учёта: путь
// ресурса, схема контракта и тело.
func Encode(enterprise string, m app.Outgoing) (path, schema string, body any) {
	r := m.Request
	src := Source{System: "ant", Enterprise: enterprise, BusinessKey: r.BusinessKey, MessageVersion: m.Version}
	if r.ClosingPoint != nil {
		src.ClosingPoint = *r.ClosingPoint
	}
	if r.StepKey != nil {
		src.StepKey = string(*r.StepKey)
	}
	for _, id := range r.BasisEventIds {
		src.BasisEventIDs = append(src.BasisEventIDs, string(id))
	}
	var item *Item
	if r.ItemID != nil {
		id := string(*r.ItemID)
		_, serial, _ := strings.Cut(id, ":")
		item = &Item{ItemID: id, SerialNo: serial}
	}
	var lot *Lot
	if r.LotID != nil {
		lot = &Lot{LotID: string(*r.LotID)}
		if r.Quantity != nil {
			lot.Quantity = *r.Quantity
		}
	}
	var ncs []string
	for _, n := range r.NcIds {
		ncs = append(ncs, string(n))
	}
	at := m.OccurredAt.UTC().Format(timeLayout)
	if m.OccurredAt.IsZero() {
		at = time.Unix(0, 0).UTC().Format(timeLayout)
	}
	if m.Action() == dom.InspectionResult {
		ir := InspectionResult{MessageID: m.MessageID, Contract: ContractVersion, OccurredAt: at, Source: src, Item: item, Lot: lot,
			Nonconformities: ncs, CorrectsMessageID: m.CorrectsMessageID}
		if r.ClosingPoint != nil {
			ir.ClosingPoint = *r.ClosingPoint
		}
		if r.PresentationNo != nil {
			ir.PresentationNo = *r.PresentationNo
		}
		res := ""
		if r.Resolution != nil {
			res = string(*r.Resolution)
		}
		ir.Result = Result(res)
		if r.ConcessionID != nil {
			ir.ConcessionNumber = string(*r.ConcessionID)
		}
		return "/hs/qc/v1/inspection-results", schemaInspect, ir
	}
	kind, defect := Kind(m.Action())
	p := Posting{MessageID: m.MessageID, Contract: ContractVersion, Kind: kind, DefectKind: defect, OccurredAt: at, Source: src,
		Item: item, Lot: lot, AfterRework: r.AfterRework, Nonconformities: ncs, CorrectsMessageID: m.CorrectsMessageID}
	if r.FromWarehouseID != nil {
		p.WarehouseFrom = &Warehouse{Code: string(*r.FromWarehouseID)}
	}
	if r.ToWarehouseID != nil {
		p.WarehouseTo = &Warehouse{Code: string(*r.ToWarehouseID)}
	}
	if r.OrderID != nil {
		p.OrderID = string(*r.OrderID)
	}
	if r.ConcessionID != nil {
		p.ConcessionNumber = string(*r.ConcessionID)
	}
	if r.ClaimBasis != nil {
		p.ClaimBasis = *r.ClaimBasis
	}
	return "/hs/qc/v1/postings", schemaPosting, p
}
