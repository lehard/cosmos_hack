package galaktika

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	app "ant/internal/application/erp"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	dcad "ant/internal/domain/cad"
	dom "ant/internal/domain/erp"
	"ant/internal/domain/kernel"
	"ant/internal/infrastructure/integration/schemacheck"
)

// Схемы контракта gal.qc.v1 во встроенной копии contracts/.
const (
	contractDir = "integrations/erp/galaktika/gal.qc.v1/"
	// ContractVersion — версия контракта, которую знает адаптер.
	ContractVersion = "gal.qc.v1"
	schemaExchange  = contractDir + "exchange.schema.json"
	schemaAck       = contractDir + "ack.schema.json"
	schemaAbout     = contractDir + "about.schema.json"
	schemaInbox     = contractDir + "inbox.schema.json"
)

const timeLayout = "2006-01-02T15:04:05.000Z"

// PostingKind — вид учётного действия в пакете Posting (наш язык → gal.qc.v1).
func PostingKind(a dom.Action) string {
	switch a {
	case dom.AcceptIntoWork:
		return "accept_to_work"
	case dom.WarehouseTransfer:
		return "internal_move"
	case dom.ScrapRework:
		return "defect_rework"
	case dom.ScrapWriteoff:
		return "defect_writeoff"
	case dom.ScrapReprocess:
		return "defect_reprocess"
	case dom.ReturnToSupplier:
		return "return_to_supplier"
	case dom.ReturnFromDefect:
		return "return_from_defect"
	case dom.Release:
		return "release"
	}
	return ""
}

// Verdict — итог контроля gal.qc.v1 по решению на закрывающей точке.
func Verdict(resolution string) string {
	switch resolution {
	case "accept":
		return "accepted"
	case "accept_with_concession":
		return "accepted_with_concession"
	case "accept_partially":
		return "accepted_partially"
	case "reject":
		return "rejected"
	}
	return "insufficient_data"
}

// Encode — пакет gal.qc.v1 по исходящему сообщению порта учёта: тот же
// внутренний сигнал, что уходит в 1С (FR-92). resource — ресурс REST-фасада.
func Encode(node, peer, enterprise string, m app.Outgoing) (resource string, e Exchange) {
	r := m.Request
	at := m.OccurredAt.UTC()
	if m.OccurredAt.IsZero() {
		at = time.Unix(0, 0).UTC()
	}
	e = Exchange{MessageID: m.MessageID, CreatedAt: at.Format(timeLayout), From: node, To: peer, Contract: ContractVersion}
	src := Source{BusinessKey: r.BusinessKey, MessageVersion: m.Version, Enterprise: enterprise, CorrectsMessageID: m.CorrectsMessageID}
	if r.ClosingPoint != nil {
		src.ClosingPoint = *r.ClosingPoint
	}
	if r.StepKey != nil {
		src.StepKey = string(*r.StepKey)
	}
	for _, id := range r.BasisEventIds {
		src.Basis = append(src.Basis, Ref{Ref: string(id)})
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
	var order *ProductionOrder
	if r.OrderID != nil {
		order = &ProductionOrder{OrderID: string(*r.OrderID)}
	}
	var ncs []Ref
	for _, n := range r.NcIds {
		ncs = append(ncs, Ref{Ref: string(n)})
	}
	concession := ""
	if r.ConcessionID != nil {
		concession = string(*r.ConcessionID)
	}
	if m.Action() == dom.InspectionResult {
		q := &QualityLotResult{Source: src, Item: item, Lot: lot, ProductionOrder: order, Nonconformities: ncs, Concession: concession}
		if r.PresentationNo != nil {
			q.Presentation = *r.PresentationNo
		}
		res := ""
		if r.Resolution != nil {
			res = string(*r.Resolution)
		}
		q.Verdict = Verdict(res)
		e.QualityLotResult = q
		return "/quality/lot-results", e
	}
	p := &Posting{Kind: PostingKind(m.Action()), Source: src, Item: item, Lot: lot, ProductionOrder: order, Nonconformities: ncs,
		Concession: concession, AfterRework: r.AfterRework}
	if r.FromWarehouseID != nil {
		p.WarehouseFrom = &Warehouse{Code: string(*r.FromWarehouseID)}
	}
	if r.ToWarehouseID != nil {
		p.WarehouseTo = &Warehouse{Code: string(*r.ToWarehouseID)}
	}
	if r.ClaimBasis != nil {
		p.ClaimBasis = *r.ClaimBasis
	}
	e.Posting = p
	return "/production/postings", e
}

// Check — пакет по контракту: JSON-форма проходит схему exchange, внутри
// ровно одно сообщение. Одна проверка для обоих транспортов (FR-111).
func Check(e Exchange) error {
	if e.Kind() == "" {
		return errors.New("в пакете должно быть ровно одно сообщение")
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if err := schemacheck.Validate(schemaExchange, b); err != nil {
		f, d := schemacheck.Violation(err)
		return fmt.Errorf("%s: %s", f, d)
	}
	return nil
}

// MarshalXML — XML-пакет каталога обмена с объявлением XML.
func MarshalXML(v any) ([]byte, error) {
	b, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(b, '\n')...), nil
}

// ParseExchangeXML — XML-пакет → модель; лишние элементы не мешают разбору.
func ParseExchangeXML(b []byte) (Exchange, error) {
	var e Exchange
	if err := xml.NewDecoder(bytes.NewReader(b)).Decode(&e); err != nil {
		return e, err
	}
	if e.XMLName.Space != Namespace {
		return e, fmt.Errorf("пространство имён %q, ожидалось %q", e.XMLName.Space, Namespace)
	}
	return e, nil
}

// Ack из XML или JSON с проверкой схемой квитанции.
func parseAck(b []byte, isXML bool) (Ack, error) {
	var a Ack
	var err error
	if isXML {
		err = xml.Unmarshal(b, &a)
	} else {
		err = json.Unmarshal(b, &a)
	}
	if err != nil {
		return a, err
	}
	j, _ := json.Marshal(a)
	if err := schemacheck.Validate(schemaAck, j); err != nil {
		f, d := schemacheck.Violation(err)
		return a, fmt.Errorf("квитанция не по контракту: %s: %s", f, d)
	}
	return a, nil
}

// response — ответ порта учёта по квитанции.
func response(a Ack, httpStatus int) app.Response {
	if a.Status == "ok" {
		out := app.Response{Outcome: ev.ErpPostingRespondedV1OutcomeAccepted, Receipt: a.Document, DocumentRef: a.NRecCreated, HTTPStatus: httpStatus}
		if out.Receipt == "" {
			out.Receipt = a.NRecCreated
		}
		if a.Duplicate || httpStatus == 200 {
			out.Outcome = ev.ErpPostingRespondedV1OutcomeDuplicate
		}
		return out
	}
	return app.Response{Outcome: ev.ErpPostingRespondedV1OutcomeRejected, Code: a.Code, Message: a.Text, ErrorCode: string(ErrorCode(a.Code)), HTTPStatus: httpStatus}
}

// ErrorCode — наш код ошибки (contracts/errors.yaml) по коду Галактики.
func ErrorCode(code string) errcodes.Code {
	switch code {
	case "CONTRACT_NOT_FOUND":
		return errcodes.ErpContractNotFound
	case "LOT_NOT_FOUND", "ITEM_NOT_MAPPED", "ITEM_NOT_FOUND", "WAREHOUSE_NOT_FOUND", "ORDER_NOT_FOUND":
		return errcodes.ErpIdMappingMissing
	}
	return errcodes.ErpDataError
}

// contractCode — код GalAck, означающий несовместимость контракта, а не ошибку данных.
func contractCode(code string) bool { return code == "CONTRACT_VERSION" || code == "SCHEMA_VIOLATION" }

// Входящие пакеты → факты порта учёта (AD-18: через обычный приём).

// ExternalID — внешний ID записи Галактики `galaktika:‹база›:‹таблица›:‹NRec›` (FR-95).
func ExternalID(db, table, nrec string) string {
	return "galaktika:" + db + ":" + table + ":" + nrec
}

// OrderID — наш ID задания по номеру Галактики («СЗ-000812» → ORD-SZ-000812):
// префикс системы в номере не даёт совпасть с заданиями 1С.
func OrderID(number string) string { return "ORD-" + dcad.Latin(number) }

// LotID — наш ID партии по номеру партии (правило 1С: «П-2026-0915» → LOT-P-2026-0915).
func LotID(number string) string {
	l := dcad.Latin(number)
	if strings.HasPrefix(l, "LOT-") {
		return l
	}
	return "LOT-" + l
}

func uid(kind, key string) string { return kernel.UUIDv5(constants.NsAnt, kind+"\x1f"+key) }

func mapping(kind ev.ReferenceExternalIDMappedV1ObjectKind, ext, id string) app.Inbound {
	return app.Inbound{Type: catalog.ReferenceExternalIdMapped, EventID: uid("galaktika.map", string(kind)+":"+ext+":"+id),
		Data: ev.ReferenceExternalIDMappedV1{System: ev.ReferenceExternalIDMappedV1SystemGalaktika, ObjectKind: kind, ExternalID: ext, InternalID: ev.ObjectID(id)}}
}

func or(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// Inbound — факты порта учёта из входящего пакета: задание, партия, каталог
// МЦ и соответствия внешних ID. event_id — от номера пакета: повторное
// чтение того же пакета не даёт новых записей (AD-7). db — база Галактики.
func Inbound(db string, e Exchange) []app.Inbound {
	at, err := time.Parse(time.RFC3339Nano, e.CreatedAt)
	if err != nil {
		at = time.Time{}
	}
	at = at.UTC()
	var out []app.Inbound
	itemType := func(it Item) string { return dcad.ItemTypeID(it.Designation) }
	itemMap := func(it Item) {
		if t := itemType(it); t != "" && it.NRec != "" {
			out = append(out, mapping(ev.ReferenceExternalIDMappedV1ObjectKindItemType, ExternalID(db, or(it.Table, "KatMC"), it.NRec), t))
		}
	}
	switch {
	case e.ProductionTask != nil:
		p := e.ProductionTask
		t := itemType(p.Item)
		if t == "" {
			return nil
		}
		oid := OrderID(p.Number)
		d := ev.ErpOrderReceivedV1{OrderID: ev.ObjectID(oid), ExternalSystem: ev.ErpOrderReceivedV1ExternalSystemGalaktika, ExternalNumber: p.Number,
			ItemTypeID: ev.ObjectID(t), Quantity: max(p.Quantity, 1)}
		if p.DueDate != "" {
			x := ev.Date(p.DueDate)
			d.DueDate = &x
		}
		if p.KdRevision != "" {
			rev := p.KdRevision
			d.ItemRevision = &rev
		}
		out = append(out, app.Inbound{Type: catalog.ErpOrderReceived, EventID: uid("galaktika.order", e.MessageID), OccurredAt: at, Data: d},
			mapping(ev.ReferenceExternalIDMappedV1ObjectKindOrder, ExternalID(db, or(p.Table, "MnPlan"), p.NRec), oid))
		itemMap(p.Item)
	case e.LotReceived != nil:
		l := e.LotReceived
		t := itemType(l.Item)
		sup := dcad.Latin(l.Supplier.Code)
		if t == "" || sup == "" || l.Lot.Number == "" {
			return nil
		}
		lot := LotID(l.Lot.Number)
		d := ev.ErpLotReceivedV1{LotID: ev.ObjectID(lot), ExternalSystem: ev.ErpLotReceivedV1ExternalSystemGalaktika, ExternalNumber: l.Number,
			SupplierID: ev.ObjectID(sup), ItemTypeID: ev.ObjectID(t), Quantity: max(l.Lot.Quantity, 1)}
		if l.Lot.HeatNo != "" {
			h := l.Lot.HeatNo
			d.HeatNo = &h
		}
		if l.Lot.CertificateNo != "" {
			c := l.Lot.CertificateNo
			d.CertificateNo = &c
		}
		if l.Lot.ExpiryDate != "" {
			x := ev.Date(l.Lot.ExpiryDate)
			d.ExpiryDate = &x
		}
		out = append(out, app.Inbound{Type: catalog.ErpLotReceived, EventID: uid("galaktika.lot", e.MessageID), OccurredAt: at, Data: d},
			mapping(ev.ReferenceExternalIDMappedV1ObjectKindLot, ExternalID(db, or(l.Table, "KatSopr"), l.NRec)+"#"+l.Lot.Number, lot))
		if l.Supplier.NRec != "" {
			out = append(out, mapping(ev.ReferenceExternalIDMappedV1ObjectKindSupplier, ExternalID(db, "KatOrg", l.Supplier.NRec), sup))
		}
		itemMap(l.Item)
	case e.ItemCatalog != nil:
		var entries []ev.NomenclatureEntry
		var maps []app.Inbound
		for _, it := range e.ItemCatalog.Items {
			if it.Deleted || it.NRec == "" {
				continue
			}
			ext := ExternalID(db, or(it.Table, "KatMC"), it.NRec)
			x := ev.NomenclatureEntry{ExternalID: ext, Designation: it.Designation, Name: or(it.Name, it.Designation)}
			if t := itemType(it); t != "" {
				id := ev.ObjectID(t)
				x.ItemTypeID = &id
				maps = append(maps, mapping(ev.ReferenceExternalIDMappedV1ObjectKindItemType, ext, t))
			}
			entries = append(entries, x)
		}
		if len(entries) == 0 {
			return nil
		}
		out = append(out, app.Inbound{Type: catalog.ErpNomenclatureSynced, EventID: uid("galaktika.nomenclature", e.MessageID), OccurredAt: at,
			Data: ev.ErpNomenclatureSyncedV1{ExternalSystem: ev.ErpNomenclatureSyncedV1ExternalSystemGalaktika, Entries: entries}})
		out = append(out, maps...)
	}
	for i := range out {
		if out[i].OccurredAt.IsZero() {
			out[i].OccurredAt = at
		}
	}
	return slices.Clip(out)
}
