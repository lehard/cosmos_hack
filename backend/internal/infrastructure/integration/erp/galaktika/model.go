package galaktika

import "encoding/xml"

// Модель пакета gal.qc.v1 (contracts/integrations/erp/galaktika): одни и те
// же типы кодируются XML-пакетом каталога обмена (теги xml, имена — как в
// gal.qc.v1.xsd) и JSON-формой REST-фасада (теги json, схема
// gal.qc.v1/exchange.schema.json). Типы рукописные; их соответствие схеме и
// XSD проверяет контрактный тест на эталонах.

// Namespace — пространство имён XML-пакета.
const Namespace = "urn:ant-qc:galaktika:v1"

// Exchange — пакет `GalExchange`: заголовок и ровно одно сообщение.
type Exchange struct {
	XMLName   xml.Name `xml:"urn:ant-qc:galaktika:v1 GalExchange" json:"-"`
	MessageID string   `xml:"messageId,attr" json:"message_id"`
	CreatedAt string   `xml:"createdAt,attr" json:"created_at"`
	From      string   `xml:"from,attr" json:"from"`
	To        string   `xml:"to,attr" json:"to"`
	Contract  string   `xml:"contract,attr" json:"contract"`

	QualityLotResult *QualityLotResult `xml:"QualityLotResult,omitempty" json:"quality_lot_result,omitempty"`
	Posting          *Posting          `xml:"Posting,omitempty" json:"posting,omitempty"`
	ProductionTask   *ProductionTask   `xml:"ProductionTask,omitempty" json:"production_task,omitempty"`
	LotReceived      *LotReceived      `xml:"LotReceived,omitempty" json:"lot_received,omitempty"`
	ItemCatalog      *ItemCatalog      `xml:"ItemCatalog,omitempty" json:"item_catalog,omitempty"`
}

// Kind — какое сообщение внутри пакета (пусто — ни одного или больше одного).
func (e Exchange) Kind() string {
	var kinds []string
	if e.QualityLotResult != nil {
		kinds = append(kinds, "QualityLotResult")
	}
	if e.Posting != nil {
		kinds = append(kinds, "Posting")
	}
	if e.ProductionTask != nil {
		kinds = append(kinds, "ProductionTask")
	}
	if e.LotReceived != nil {
		kinds = append(kinds, "LotReceived")
	}
	if e.ItemCatalog != nil {
		kinds = append(kinds, "ItemCatalog")
	}
	if len(kinds) != 1 {
		return ""
	}
	return kinds[0]
}

// Source — бизнес-ключ исходящего сообщения ant и основание (AD-7).
type Source struct {
	BusinessKey       string `xml:"businessKey,attr" json:"business_key"`
	MessageVersion    int    `xml:"messageVersion,attr" json:"message_version"`
	Enterprise        string `xml:"enterprise,attr" json:"enterprise"`
	ClosingPoint      string `xml:"closingPoint,attr,omitempty" json:"closing_point,omitempty"`
	StepKey           string `xml:"stepKey,attr,omitempty" json:"step_key,omitempty"`
	CorrectsMessageID string `xml:"correctsMessageId,attr,omitempty" json:"corrects_message_id,omitempty"`
	Basis             []Ref  `xml:"Basis,omitempty" json:"basis,omitempty"`
}

// Ref — ссылка на запись ant (несоответствие, запись-основание).
type Ref struct {
	Ref string `xml:"ref,attr" json:"ref"`
}

// Item — изделие или запись каталога МЦ.
type Item struct {
	ItemID      string `xml:"itemId,attr,omitempty" json:"item_id,omitempty"`
	SerialNo    string `xml:"serialNo,attr,omitempty" json:"serial_no,omitempty"`
	Designation string `xml:"designation,attr,omitempty" json:"designation,omitempty"`
	Name        string `xml:"name,attr,omitempty" json:"name,omitempty"`
	NRec        string `xml:"nrec,attr,omitempty" json:"nrec,omitempty"`
	Table       string `xml:"table,attr,omitempty" json:"table,omitempty"`
	Deleted     bool   `xml:"deleted,attr,omitempty" json:"deleted,omitempty"`
}

// Lot — партия.
type Lot struct {
	LotID         string `xml:"lotId,attr,omitempty" json:"lot_id,omitempty"`
	Number        string `xml:"number,attr,omitempty" json:"number,omitempty"`
	Quantity      int    `xml:"quantity,attr,omitempty" json:"quantity,omitempty"`
	HeatNo        string `xml:"heatNo,attr,omitempty" json:"heat_no,omitempty"`
	CertificateNo string `xml:"certificateNo,attr,omitempty" json:"certificate_no,omitempty"`
	ExpiryDate    string `xml:"expiryDate,attr,omitempty" json:"expiry_date,omitempty"`
}

// Warehouse — склад по коду.
type Warehouse struct {
	Code string `xml:"code,attr" json:"code"`
	NRec string `xml:"nrec,attr,omitempty" json:"nrec,omitempty"`
}

// ProductionOrder — задание.
type ProductionOrder struct {
	OrderID    string `xml:"orderId,attr,omitempty" json:"order_id,omitempty"`
	NRec       string `xml:"nrec,attr,omitempty" json:"nrec,omitempty"`
	RouteSheet string `xml:"routeSheet,attr,omitempty" json:"route_sheet,omitempty"`
}

// QualityLotResult — результат контроля (ant → «Управление качеством продукции»).
type QualityLotResult struct {
	Presentation    int              `xml:"presentation,attr,omitempty" json:"presentation,omitempty"`
	Verdict         string           `xml:"verdict,attr" json:"verdict"`
	Concession      string           `xml:"concession,attr,omitempty" json:"concession,omitempty"`
	Source          Source           `xml:"Source" json:"source"`
	Item            *Item            `xml:"Item,omitempty" json:"item,omitempty"`
	Lot             *Lot             `xml:"Lot,omitempty" json:"lot,omitempty"`
	ProductionOrder *ProductionOrder `xml:"ProductionOrder,omitempty" json:"production_order,omitempty"`
	Nonconformities []Ref            `xml:"Nonconformity,omitempty" json:"nonconformities,omitempty"`
}

// Posting — учётное действие (ant → Галактика).
type Posting struct {
	Kind            string           `xml:"kind,attr" json:"kind"`
	AfterRework     *bool            `xml:"afterRework,attr,omitempty" json:"after_rework,omitempty"`
	Concession      string           `xml:"concession,attr,omitempty" json:"concession,omitempty"`
	Source          Source           `xml:"Source" json:"source"`
	Item            *Item            `xml:"Item,omitempty" json:"item,omitempty"`
	Lot             *Lot             `xml:"Lot,omitempty" json:"lot,omitempty"`
	WarehouseFrom   *Warehouse       `xml:"WarehouseFrom,omitempty" json:"warehouse_from,omitempty"`
	WarehouseTo     *Warehouse       `xml:"WarehouseTo,omitempty" json:"warehouse_to,omitempty"`
	ProductionOrder *ProductionOrder `xml:"ProductionOrder,omitempty" json:"production_order,omitempty"`
	ClaimBasis      string           `xml:"ClaimBasis,omitempty" json:"claim_basis,omitempty"`
	Nonconformities []Ref            `xml:"Nonconformity,omitempty" json:"nonconformities,omitempty"`
}

// ProductionTask — сменное задание / маршрутный лист (Галактика → ant).
type ProductionTask struct {
	NRec       string `xml:"nrec,attr" json:"nrec"`
	Table      string `xml:"table,attr,omitempty" json:"table,omitempty"`
	Number     string `xml:"number,attr" json:"number"`
	Date       string `xml:"date,attr" json:"date"`
	DueDate    string `xml:"dueDate,attr,omitempty" json:"due_date,omitempty"`
	Quantity   int    `xml:"quantity,attr" json:"quantity"`
	RouteSheet string `xml:"routeSheet,attr,omitempty" json:"route_sheet,omitempty"`
	KdRevision string `xml:"kdRevision,attr,omitempty" json:"kd_revision,omitempty"`
	Item       Item   `xml:"Item" json:"item"`
}

// Supplier — поставщик.
type Supplier struct {
	NRec string `xml:"nrec,attr,omitempty" json:"nrec,omitempty"`
	Code string `xml:"code,attr" json:"code"`
	Name string `xml:"name,attr,omitempty" json:"name,omitempty"`
}

// LotReceived — приходная накладная с партией (Галактика → ant).
type LotReceived struct {
	NRec     string   `xml:"nrec,attr" json:"nrec"`
	Table    string   `xml:"table,attr,omitempty" json:"table,omitempty"`
	Number   string   `xml:"number,attr" json:"number"`
	Date     string   `xml:"date,attr" json:"date"`
	Supplier Supplier `xml:"Supplier" json:"supplier"`
	Item     Item     `xml:"Item" json:"item"`
	Lot      Lot      `xml:"Lot" json:"lot"`
}

// ItemCatalog — записи каталога МЦ (Галактика → ant).
type ItemCatalog struct {
	Items []Item `xml:"Item" json:"items"`
}

// Ack — квитанция `GalAck`.
type Ack struct {
	XMLName     xml.Name `xml:"urn:ant-qc:galaktika:v1 GalAck" json:"-"`
	MessageID   string   `xml:"messageId,attr" json:"message_id"`
	Status      string   `xml:"status,attr" json:"status"`
	Duplicate   bool     `xml:"duplicate,attr,omitempty" json:"duplicate,omitempty"`
	NRecCreated string   `xml:"nrecCreated,attr,omitempty" json:"nrec_created,omitempty"`
	Document    string   `xml:"document,attr,omitempty" json:"document,omitempty"`
	Code        string   `xml:"code,attr,omitempty" json:"code,omitempty"`
	Text        string   `xml:"text,attr,omitempty" json:"text,omitempty"`
	ReceivedAt  string   `xml:"receivedAt,attr,omitempty" json:"received_at,omitempty"`
}

// About — описание ответной стороны `GalAbout`.
type About struct {
	XMLName   xml.Name `xml:"urn:ant-qc:galaktika:v1 GalAbout" json:"-"`
	Node      string   `xml:"node,attr" json:"node"`
	Database  string   `xml:"database,attr" json:"database"`
	Contract  string   `xml:"contract,attr" json:"contract"`
	Platform  string   `xml:"platform,attr,omitempty" json:"platform,omitempty"`
	Supported []string `xml:"Supported" json:"supported"`
}

// Inbox — входящие пакеты у REST-фасада.
type Inbox struct {
	Packets []Exchange `json:"packets"`
}
