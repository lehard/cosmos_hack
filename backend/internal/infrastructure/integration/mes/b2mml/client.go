package b2mml

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	app "ant/internal/application/mes"
	dom "ant/internal/domain/mes"
	"ant/internal/infrastructure/integration/schemacheck"
)

// Контракт mes.isa95.v1 во встроенной копии contracts/.
const (
	// ContractVersion — версия контракта ant поверх B2MML (versionID).
	ContractVersion = "mes.isa95.v1"
	// ReleaseID — релиз B2MML.
	ReleaseID   = "7.01"
	schemaDir   = "integrations/mes/b2mml/"
	schemaAbout = "integrations/mes/binding/about.schema.json"
	schemaInbox = "integrations/mes/binding/outbox.schema.json"
)

// Schema — путь схемы сообщения B2MML по имени (ProcessOperationsSchedule…).
func Schema(message string) string { return schemaDir + message + ".schema.json" }

const timeLayout = "2006-01-02T15:04:05.000Z"

// Config — канал MES (конфигурация mes.b2mml; включённые системы и адреса —
// конфигурация, AD-18).
type Config struct {
	// BaseURL — адрес HTTP-привязки B2MML (`http(s)://‹хост›/b2mml`).
	BaseURL string
	// LogicalID — наш логический ID отправителя (ant).
	LogicalID string
	// User, Password — HTTP Basic (пароль — файлом).
	User, Password string
	Timeout        time.Duration
	// Stand — на месте MES работает stand (эпик 43).
	Stand bool
	HTTP  *http.Client
}

// Client — адаптер канала MES (application/mes.Channel).
type Client struct {
	cfg  Config
	http *http.Client
}

var _ app.Channel = (*Client)(nil)

// New создаёт адаптер.
func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("b2mml: не задан адрес канала MES (mes.b2mml.base_url)")
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.LogicalID == "" {
		cfg.LogicalID = "ant"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{cfg: cfg, http: hc}, nil
}

// Info — канал обмена.
func (c *Client) Info() app.ChannelInfo {
	return app.ChannelInfo{System: "b2mml", Endpoint: c.cfg.BaseURL, ContractVersion: ContractVersion, Stand: c.cfg.Stand}
}

// Check — сверка ответной стороны (AD-18): релиз B2MML и версия mes.isa95.
func (c *Client) Check(ctx context.Context) (string, error) {
	code, b, err := c.do(ctx, http.MethodGet, "/about", nil)
	if err != nil {
		return "", err
	}
	if err := statusErr(code, b); err != nil {
		return "", err
	}
	if err := schemacheck.Validate(schemaAbout, b); err != nil {
		f, d := schemacheck.Violation(err)
		return "", &app.ContractError{Detail: "описание канала MES не по контракту: " + f + ": " + d}
	}
	var a About
	_ = json.Unmarshal(b, &a)
	if !slices.Contains(a.Supported, ContractVersion) {
		return "", &app.ContractError{Detail: fmt.Sprintf("MES %s принимает %v, адаптер — %s", a.LogicalID, a.Supported, ContractVersion)}
	}
	return fmt.Sprintf("MES %s (%s): B2MML %s, контракт %s", a.LogicalID, a.Product, a.ReleaseID, ContractVersion), nil
}

// Encode — блок или снятие в MES: экземпляр — SyncMaterialSubLot, партия —
// SyncMaterialLot. Disposition = Restricted — блок (заблокированное изделие не
// получает следующую операцию), UnRestricted — снятие; ConfirmationCode =
// Always — блок должен быть подтверждён. Наш ID — в Description (для мастера: «Главный: ‹ID›»).
func Encode(logicalID string, m app.HoldMessage) (message string, body any) {
	at := m.OccurredAt.UTC()
	if m.OccurredAt.IsZero() {
		at = time.Unix(0, 0).UTC()
	}
	var s SyncBody
	s.ReleaseID, s.VersionID = ReleaseID, ContractVersion
	s.ApplicationArea = ApplicationArea{Sender: Sender{LogicalID: logicalID, ConfirmationCode: "Always"}, CreationDateTime: at.Format(timeLayout), BODID: m.MessageID}
	s.DataArea.Sync.ActionCriteria.ActionExpression.ActionCode = "Change"
	subject := or(m.ItemID, m.LotID)
	l := Lot{ID: or(m.ExternalID, subject), Disposition: "UnRestricted", Status: "QC-RELEASED",
		Description: strings.TrimSpace(m.Reason + "; Главный: " + subject + "; ключ " + m.Key)}
	if m.Hold {
		l.Disposition, l.Status, l.StorageLocation = "Restricted", "QC-HOLD", "Изолятор ОТК"
	}
	if m.LotID != "" {
		s.DataArea.MaterialLot = []Lot{l}
		return "SyncMaterialLot", SyncMaterialLot{SyncMaterialLot: s}
	}
	s.DataArea.MaterialSubLot = []Lot{l}
	return "SyncMaterialSubLot", SyncMaterialSubLot{SyncMaterialSubLot: s}
}

// Post — блок или снятие с BODID = номер сообщения (AD-7). Сообщение сначала
// проверяется схемой: не по схеме — в MES ничего не уходит (FR-111).
func (c *Client) Post(ctx context.Context, m app.HoldMessage) (app.Response, error) {
	msg, body := Encode(c.cfg.LogicalID, m)
	raw, err := json.Marshal(body)
	if err != nil {
		return app.Response{}, err
	}
	if err := schemacheck.Validate(Schema(msg), raw); err != nil {
		f, d := schemacheck.Violation(err)
		return app.Response{}, &app.ContractError{Local: true, Detail: msg + " не прошло схему до отправки: " + f + ": " + d}
	}
	code, b, err := c.do(ctx, http.MethodPost, "/"+msg, raw)
	if err != nil {
		return app.Response{}, err
	}
	switch code {
	case http.StatusOK, http.StatusAccepted, http.StatusUnprocessableEntity, http.StatusBadRequest:
		return Confirm(m.MessageID, b)
	}
	return app.Response{}, statusErr(code, b)
}

// Confirm — ConfirmBOD на сообщение bodid: успех (повтор — duplicate),
// ErrorType = Data — ошибка данных, Contract — несовместимость, Transport —
// повтор тем же BODID.
func Confirm(bodid string, b []byte) (app.Response, error) {
	if err := schemacheck.Validate(Schema("ConfirmBOD"), b); err != nil {
		f, d := schemacheck.Violation(err)
		return app.Response{}, &app.ContractError{Detail: "ConfirmBOD не по контракту: " + f + ": " + d}
	}
	var cb ConfirmBOD
	if err := json.Unmarshal(b, &cb); err != nil {
		return app.Response{}, &app.ContractError{Detail: "ConfirmBOD не разобран: " + err.Error()}
	}
	for _, x := range cb.ConfirmBOD.DataArea.BOD {
		if x.OriginalApplicationArea.BODID != bodid {
			continue
		}
		if x.BODFailureMessage != nil && len(x.BODFailureMessage.ErrorMessage) > 0 {
			e := x.BODFailureMessage.ErrorMessage[0]
			switch e.ErrorType {
			case "Contract", "Security":
				return app.Response{}, &app.ContractError{Detail: "MES отвергла сообщение: " + e.ErrorCode + " " + e.ErrorDescription}
			case "Transport":
				return app.Response{}, &app.TransportError{Err: errors.New(e.ErrorCode + " " + e.ErrorDescription)}
			}
			return app.Response{Outcome: "rejected", Code: e.ErrorCode, Message: e.ErrorDescription}, nil
		}
		if x.BODSuccessMessage != nil && x.BODSuccessMessage.Duplicate {
			return app.Response{Outcome: "duplicate"}, nil
		}
		return app.Response{Outcome: "accepted"}, nil
	}
	return app.Response{}, &app.TransportError{Err: errors.New("в ConfirmBOD нет подтверждения BODID " + bodid)}
}

// Pull — сообщения MES с канала: задания и события операций на нашем языке;
// сообщение не по схеме — в Rejected с причиной.
func (c *Client) Pull(ctx context.Context) (app.Inbound, error) {
	var in app.Inbound
	code, b, err := c.do(ctx, http.MethodGet, "/outbox", nil)
	if err != nil {
		return in, err
	}
	if err := statusErr(code, b); err != nil {
		return in, err
	}
	if err := schemacheck.Validate(schemaInbox, b); err != nil {
		f, d := schemacheck.Violation(err)
		return in, &app.ContractError{Detail: "канал MES не по контракту: " + f + ": " + d}
	}
	var box struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(b, &box); err != nil {
		return in, &app.ContractError{Detail: err.Error()}
	}
	for i, raw := range box.Messages {
		jobs, events, err := Decode(raw)
		if err != nil {
			in.Rejected = append(in.Rejected, dom.Deferred{MessageID: "#" + strconv.Itoa(i), Reason: err.Error()})
			continue
		}
		in.Jobs = append(in.Jobs, jobs...)
		in.Events = append(in.Events, events...)
	}
	return in, nil
}

// Decode — входящее сообщение B2MML-JSON → задания или события на нашем языке.
// Сообщение проверяется схемой своего вида; лишние элементы стандарта
// (вне подмножества) отбрасываются до проверки — разбор терпимый.
func Decode(raw []byte) ([]dom.Job, []dom.OperationEvent, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil || len(root) != 1 {
		return nil, nil, errors.New("сообщение B2MML-JSON — объект с одним корнем")
	}
	switch {
	case root["ProcessOperationsSchedule"] != nil:
		var m ProcessOperationsSchedule
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, nil, err
		}
		if err := validate("ProcessOperationsSchedule", m); err != nil {
			return nil, nil, err
		}
		p := m.ProcessOperationsSchedule
		var jobs []dom.Job
		for _, s := range p.DataArea.OperationsSchedule {
			for _, r := range s.OperationsRequest {
				for _, sr := range r.SegmentRequirement {
					j := dom.Job{MessageID: p.ApplicationArea.BODID, RequestID: r.ID, SegmentID: sr.ID, OperationCode: sr.ProcessSegmentID,
						PlannedStart: parseTime(sr.EarliestStartTime)}
					if len(sr.EquipmentRequirement) > 0 {
						j.Station = sr.EquipmentRequirement[0].EquipmentID
					}
					for _, mr := range sr.MaterialRequirement {
						if mr.MaterialUse == "" || mr.MaterialUse == "Produced" {
							j.Material = mr.MaterialDefinitionID
							if mr.Quantity != nil {
								j.Quantity, _ = strconv.Atoi(mr.Quantity.QuantityString)
							}
							break
						}
					}
					jobs = append(jobs, j)
				}
			}
		}
		return jobs, nil, nil
	case root["NotifyOperationsEvent"] != nil:
		var m NotifyOperationsEvent
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, nil, err
		}
		if err := validate("NotifyOperationsEvent", m); err != nil {
			return nil, nil, err
		}
		n := m.NotifyOperationsEvent
		var evs []dom.OperationEvent
		for _, e := range n.DataArea.OperationsEvent {
			evs = append(evs, dom.OperationEvent{MessageID: n.ApplicationArea.BODID, EventID: e.ID, Category: strings.ToLower(e.Category),
				RunRef: e.SegmentResponseID, OperationCode: e.ProcessSegmentID, SubLot: e.MaterialSubLotID, Equipment: e.EquipmentID,
				Personnel: e.PersonnelID, At: parseTime(e.EffectiveTimestamp)})
		}
		return nil, evs, nil
	}
	name := ""
	for k := range root {
		name = k
	}
	return nil, nil, fmt.Errorf("сообщение %s вне подмножества mes.isa95.v1", name)
}

// validate — сообщение подмножества (после терпимого разбора) по его схеме.
func validate(message string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := schemacheck.Validate(Schema(message), b); err != nil {
		f, d := schemacheck.Violation(err)
		return fmt.Errorf("%s не по схеме: %s: %s", message, f, d)
	}
	return nil
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, rd)
	if err != nil {
		return 0, nil, err
	}
	if c.cfg.User != "" {
		req.SetBasicAuth(c.cfg.User, c.cfg.Password)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, &app.TransportError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return resp.StatusCode, nil, &app.TransportError{HTTPStatus: resp.StatusCode, Err: err}
	}
	return resp.StatusCode, b, nil
}

// statusErr — 5xx, 408, 429 — транспорт; 401/403 и прочие 4xx — контракт.
func statusErr(code int, body []byte) error {
	switch {
	case code >= 200 && code < 300:
		return nil
	case code >= 500 || code == http.StatusRequestTimeout || code == http.StatusTooManyRequests:
		return &app.TransportError{HTTPStatus: code, Err: errors.New(snippet(body))}
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		return &app.ContractError{Detail: fmt.Sprintf("отказ доступа MES (HTTP %d)", code)}
	}
	return &app.ContractError{Detail: fmt.Sprintf("HTTP %d: %s", code, snippet(body))}
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 300 {
		s = s[:300] + "…"
	}
	return s
}
