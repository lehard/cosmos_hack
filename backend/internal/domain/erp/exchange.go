package erp

import (
	"encoding/json"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Status — состояние исходящего сообщения (словарь erp_message_status, не ось).
type Status string

// Состояния сообщения по записям журнала.
const (
	// StatusQueued — в очереди отправки (сформировано, переотправлено, решено исправить).
	StatusQueued Status = "queued"
	// StatusAcknowledged — учётная система подтвердила приём (квитанция).
	StatusAcknowledged Status = "acknowledged"
	// StatusRejected — ошибка данных учётной системы (дальше — карантин).
	StatusRejected Status = "rejected"
	// StatusQuarantined — карантин исходящих: ждёт решения человека (FR-96).
	StatusQuarantined Status = "quarantined"
)

// Attempt — ответ учётной системы или карантин по сообщению (журнал обмена).
type Attempt struct {
	At           time.Time `json:"at"`
	Outcome      string    `json:"outcome"`
	Receipt      string    `json:"receipt,omitempty"`
	DocumentRef  string    `json:"document_ref,omitempty"`
	ErrorCode    string    `json:"error_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	HTTPStatus   int       `json:"http_status,omitempty"`
	Seq          int64     `json:"seq"`
}

// View — исходящее сообщение по бизнес-ключу глазами журнала (проекция
// erp.message, писатель erp): версии, ответы, карантин, решения людей.
// Очередь отправки — проекция журнала (AD-18).
type View struct {
	Key       string `json:"business_key"`
	System    string `json:"external_system"`
	Action    Action `json:"action"`
	ItemID    string `json:"item_id,omitempty"`
	LotID     string `json:"lot_id,omitempty"`
	Version   int    `json:"message_version"`
	MessageID string `json:"message_id"`
	// RunID — прогон сценария запроса (AD-38): ответы пишутся в тот же прогон.
	RunID          string    `json:"run_id,omitempty"`
	RequestEventID string    `json:"request_event_id"`
	RequestedAt    time.Time `json:"requested_at"`
	FirstSeq       int64     `json:"first_seq"`
	Status         Status    `json:"status"`
	// Accounting — ось «учёт в 1С» после квитанции (AD-30).
	Accounting   string `json:"accounting_state,omitempty"`
	AfterRework  bool   `json:"after_rework"`
	AckedVersion int    `json:"acked_version,omitempty"`
	// AckedRequest — запрос подтверждённой версии (для решения об исправлении).
	AckedRequest string `json:"acked_request,omitempty"`
	// Generation — число ручных переотправок и решений об исправлении.
	Generation int `json:"generation"`
	// Correction — отправляется исправление (сторно прежнего + новое).
	Correction    bool      `json:"correction,omitempty"`
	LastErrorCode string    `json:"last_error_code,omitempty"`
	Attempts      []Attempt `json:"attempts"`
	// Request — data последней версии (для очереди и журнала обмена).
	Request ev.ErpPostingRequestedV1 `json:"request"`
	// BasisSeq — seq последней записи сообщения.
	BasisSeq int64 `json:"basis_seq"`
}

// Key — бизнес-ключ записи обмена своего семейства (пусто — не запись обмена).
func Key(r kernel.Record) (string, error) {
	if !Is(r.Type, Exchange) {
		return "", nil
	}
	var d struct {
		BusinessKey string `json:"business_key"`
	}
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return "", err
	}
	return d.BusinessKey, nil
}

// Apply — шаг проекции сообщения (чистая функция, AD-45).
func (v View) Apply(r kernel.Record) (View, error) {
	v.BasisSeq = r.Seq
	switch r.Type {
	case catalog.ErpPostingRequested:
		var d ev.ErpPostingRequestedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return v, err
		}
		if d.MessageVersion < v.Version {
			return v, nil
		}
		if v.FirstSeq == 0 {
			v.FirstSeq, v.RequestedAt = r.Seq, r.OccurredAt
		}
		v.Key, v.System, v.Action, v.Version = d.BusinessKey, string(d.ExternalSystem), Action(d.Action), d.MessageVersion
		v.RequestEventID, v.Request, v.RunID = r.EventID, d, r.RunID
		if d.MessageID != nil {
			v.MessageID = string(*d.MessageID)
		} else {
			v.MessageID = MessageID(d.BusinessKey, d.MessageVersion)
		}
		if d.ItemID != nil {
			v.ItemID = string(*d.ItemID)
		}
		if d.LotID != nil {
			v.LotID = string(*d.LotID)
		}
		v.AfterRework = d.AfterRework != nil && *d.AfterRework
		v.Status, v.Correction = StatusQueued, false
		if v.Attempts == nil {
			v.Attempts = []Attempt{}
		}
	case catalog.ErpPostingResponded:
		var d ev.ErpPostingRespondedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return v, err
		}
		a := Attempt{At: r.OccurredAt, Outcome: string(d.Outcome), Seq: r.Seq}
		a.Receipt, a.DocumentRef = deref(d.Receipt), deref(d.ExternalDocumentRef)
		a.ErrorCode, a.ErrorMessage = deref(d.ErrorCode), deref(d.ErrorMessage)
		if d.HTTPStatus != nil {
			a.HTTPStatus = *d.HTTPStatus
		}
		v.Attempts = append(v.Attempts, a)
		if string(d.RequestEventID) != v.RequestEventID {
			return v, nil // ответ на прежнюю версию — только в журнал обмена
		}
		switch d.Outcome {
		case ev.ErpPostingRespondedV1OutcomeAccepted, ev.ErpPostingRespondedV1OutcomeDuplicate:
			v.Status, v.AckedVersion, v.AckedRequest, v.LastErrorCode, v.Correction = StatusAcknowledged, v.Version, v.RequestEventID, "", false
			if d.ResultingStatus != nil {
				v.Accounting = string(*d.ResultingStatus)
			} else if ax, ok := Axis(v.Action); ok {
				v.Accounting = string(ax)
			}
		default:
			v.Status, v.LastErrorCode = StatusRejected, a.ErrorCode
		}
	case catalog.ErpPostingQuarantined:
		var d ev.ErpPostingQuarantinedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return v, err
		}
		a := Attempt{At: r.OccurredAt, Outcome: "quarantined", ErrorCode: deref(d.LastErrorCode), ErrorMessage: deref(d.ErrorMessage), Seq: r.Seq}
		v.Attempts = append(v.Attempts, a)
		if string(d.RequestEventID) == v.RequestEventID {
			v.Status, v.LastErrorCode = StatusQuarantined, a.ErrorCode
		}
	case catalog.ErpPostingResendRequested:
		v.Generation++
		v.Status, v.LastErrorCode = StatusQueued, ""
	case catalog.ErpPostingCompensationDecided:
		var d ev.ErpPostingCompensationDecidedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return v, err
		}
		v.Generation++
		if d.Decision == ev.ErpPostingCompensationDecidedV1DecisionSendCorrection {
			v.Status, v.Correction, v.LastErrorCode = StatusQueued, true, ""
		} else {
			// «Оставить как отправлено»: в учёте остаётся подтверждённая версия.
			v.Status, v.LastErrorCode = StatusAcknowledged, ""
		}
	}
	return v, nil
}

// NeedsDecision — новая версия уже подтверждённого сообщения: исправление
// (сторно + новое) — только по решению человека (AD-7).
func (v View) NeedsDecision() bool {
	return v.AckedVersion > 0 && v.Version > v.AckedVersion && !v.Correction && v.Status == StatusQueued
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ItemAccounting — ось «учёт в 1С» изделия (AD-30, владелец erp): меняется
// только квитанцией учётной системы; где изделие числится в учёте — склад
// подтверждённого сообщения (FR-130).
type ItemAccounting struct {
	ItemID string `json:"item_id"`
	// State — значение оси (not_sent — сообщений ещё не подтверждено).
	State     string    `json:"state"`
	Action    Action    `json:"action,omitempty"`
	Key       string    `json:"business_key,omitempty"`
	Warehouse string    `json:"warehouse,omitempty"`
	At        time.Time `json:"at,omitzero"`
	Seq       int64     `json:"seq,omitempty"`
	// Keys — бизнес-ключи сообщений изделия в порядке формирования.
	Keys []string `json:"keys"`
	// Pending — сформированные, но ещё не подтверждённые запросы: id → действие и склад.
	Pending []PendingRequest `json:"pending,omitempty"`
}

// PendingRequest — запрос, ждущий квитанции.
type PendingRequest struct {
	EventID   string `json:"event_id"`
	Action    Action `json:"action"`
	Key       string `json:"business_key"`
	Warehouse string `json:"warehouse,omitempty"`
}

// ItemOf — изделие записи обмена (пусто — не изделие).
func ItemOf(r kernel.Record) (string, error) {
	switch r.Type {
	case catalog.ErpPostingRequested, catalog.ErpPostingResponded:
	default:
		return "", nil
	}
	var d struct {
		ItemID string `json:"item_id"`
	}
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return "", err
	}
	return d.ItemID, nil
}

// Apply — шаг проекции оси «учёт в 1С» изделия.
func (a ItemAccounting) Apply(r kernel.Record) (ItemAccounting, error) {
	if a.State == "" {
		a.State = string(ev.AxisErpAccountingNotSent)
	}
	if a.Keys == nil {
		a.Keys = []string{}
	}
	switch r.Type {
	case catalog.ErpPostingRequested:
		var d ev.ErpPostingRequestedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return a, err
		}
		if d.ItemID != nil {
			a.ItemID = string(*d.ItemID)
		}
		found := false
		for _, k := range a.Keys {
			found = found || k == d.BusinessKey
		}
		if !found {
			a.Keys = append(a.Keys, d.BusinessKey)
		}
		p := PendingRequest{EventID: r.EventID, Action: Action(d.Action), Key: d.BusinessKey}
		if d.ToWarehouseID != nil {
			p.Warehouse = string(*d.ToWarehouseID)
		}
		a.Pending = append(a.Pending, p)
	case catalog.ErpPostingResponded:
		var d ev.ErpPostingRespondedV1
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return a, err
		}
		if d.Outcome == ev.ErpPostingRespondedV1OutcomeRejected {
			return a, nil
		}
		for i, p := range a.Pending {
			if p.EventID != string(d.RequestEventID) {
				continue
			}
			a.Pending = append(a.Pending[:i:i], a.Pending[i+1:]...)
			ax, ok := Axis(p.Action)
			if d.ResultingStatus != nil {
				ax, ok = *d.ResultingStatus, true
			}
			if !ok {
				return a, nil // результат контроля оси не меняет
			}
			a.State, a.Action, a.Key, a.At, a.Seq = string(ax), p.Action, p.Key, r.OccurredAt, r.Seq
			if p.Warehouse != "" {
				a.Warehouse = p.Warehouse
			}
			return a, nil
		}
	}
	return a, nil
}

// IndexEntry — строка индекса сообщений для списка (erp.message.list).
type IndexEntry struct {
	Key      string `json:"k"`
	ItemID   string `json:"i,omitempty"`
	LotID    string `json:"l,omitempty"`
	Status   Status `json:"s"`
	FirstSeq int64  `json:"q"`
	// Request — текущий запрос (erp.posting.requested) сообщения.
	Request string `json:"r,omitempty"`
}

// IndexLimit — сколько последних сообщений держит индекс списка; остальные
// читаются по бизнес-ключу (erp.message.read).
const IndexLimit = 2000

// Index — индекс сообщений системы в порядке формирования.
type Index struct {
	Entries []IndexEntry `json:"entries"`
}

// Put — строка сообщения в индексе (новая — в конец; сверх предела старые уходят).
func (x Index) Put(e IndexEntry) Index {
	for i := range x.Entries {
		if x.Entries[i].Key == e.Key {
			x.Entries[i] = e
			return x
		}
	}
	x.Entries = append(x.Entries, e)
	if len(x.Entries) > IndexLimit {
		x.Entries = x.Entries[len(x.Entries)-IndexLimit:]
	}
	return x
}
