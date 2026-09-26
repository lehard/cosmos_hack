package onec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	app "ant/internal/application/erp"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
)

// Config — канал обмена с 1С (конфигурация erp.onec, AD-18: включённые
// системы и адреса — конфигурация).
type Config struct {
	// BaseURL — адрес публикации базы: `http(s)://‹хост›/‹база›` (для stand-а —
	// `http://‹stands›/stand/1c/erp`). OData — `‹BaseURL›/odata/standard.odata/`,
	// сервис qc — `‹BaseURL›/hs/qc/v1/`.
	BaseURL string
	// User, Password — HTTP Basic пользователя с правом удалённого доступа (пароль — файлом).
	User, Password string
	// Enterprise — код предприятия в источнике сообщения.
	Enterprise string
	// Timeout — предел ответа 1С; дольше — транспортная ошибка (повтор).
	Timeout time.Duration
	// Stand — на месте 1С работает stand (эмулятор кейса).
	Stand bool
	// HTTP — клиент (nil — свой с Timeout).
	HTTP *http.Client
}

// Client — адаптер порта учёта для 1С (application/erp.Ledger).
type Client struct {
	cfg      Config
	http     *http.Client
	manifest Manifest
}

var _ app.Ledger = (*Client)(nil)

// New создаёт адаптер; манифест и схемы — из встроенного контракта.
func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("onec: не задан адрес 1С (erp.onec.base_url)")
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.Enterprise == "" {
		cfg.Enterprise = "ENT01"
	}
	m, err := LoadManifest()
	if err != nil {
		return nil, fmt.Errorf("onec: манифест метаданных: %w", err)
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{cfg: cfg, http: hc, manifest: m}, nil
}

// Info — канал обмена.
func (c *Client) Info() app.LedgerInfo {
	return app.LedgerInfo{System: "onec", Endpoint: c.cfg.BaseURL, Stand: c.cfg.Stand, ContractVersion: ContractVersion}
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, hdr map[string]string) (int, []byte, error) {
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
	for k, v := range hdr {
		req.Header.Set(k, v)
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

// Check — сверка ответной стороны (AD-18): `$metadata` против манифеста
// контракта и версия HTTP-сервиса qc.
func (c *Client) Check(ctx context.Context) (app.Checked, error) {
	code, b, err := c.do(ctx, http.MethodGet, c.manifest.OData.Path+"$metadata", nil, map[string]string{"Accept": "application/xml"})
	if err != nil {
		return app.Checked{}, err
	}
	if err := statusErr(code, b); err != nil {
		return app.Checked{}, err
	}
	diff, err := CompareMetadata(c.manifest, b)
	if err != nil {
		return app.Checked{}, &app.ContractError{Detail: err.Error()}
	}
	if len(diff) > 0 {
		return app.Checked{}, &app.ContractError{Detail: "метаданные 1С не совпали с манифестом " + c.manifest.Manifest + ": " + strings.Join(diff, "; ")}
	}
	code, b, err = c.do(ctx, http.MethodGet, c.manifest.QC.Path+"about", nil, nil)
	if err != nil {
		return app.Checked{}, err
	}
	if err := statusErr(code, b); err != nil {
		return app.Checked{}, err
	}
	if err := Validate(schemaAbout, b); err != nil {
		return app.Checked{}, &app.ContractError{Detail: "ответ сервиса qc/about не по контракту: " + err.Error()}
	}
	var a About
	_ = json.Unmarshal(b, &a)
	if !slices.Contains(a.Supported, ContractVersion) {
		return app.Checked{}, &app.ContractError{Detail: fmt.Sprintf("расширение 1С поддерживает %v, адаптер — %s", a.Supported, ContractVersion)}
	}
	return app.Checked{ContractVersion: a.Contract, Detail: fmt.Sprintf("$metadata совпал с манифестом (%d сущностей); сервис qc: %s", len(c.manifest.Entities), a.Contract)}, nil
}

// statusErr — ошибки чтения: 5xx — транспорт, 401/403 и прочие 4xx — контракт.
func statusErr(code int, body []byte) error {
	switch {
	case code >= 200 && code < 300:
		return nil
	case code >= 500 || code == http.StatusRequestTimeout || code == http.StatusTooManyRequests:
		return &app.TransportError{HTTPStatus: code, Err: errors.New(snippet(body))}
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		return &app.ContractError{Detail: fmt.Sprintf("отказ доступа 1С (HTTP %d)", code)}
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

// Post — отправить сообщение qc.v1 с ключом идемпотентности X-Message-Id.
// Сообщение сначала проверяется схемой контракта: не по схеме — в 1С ничего
// не уходит (ContractError{Local}), FR-111.
func (c *Client) Post(ctx context.Context, m app.Outgoing) (app.Response, error) {
	path, schema, body := Encode(c.cfg.Enterprise, m)
	raw, err := json.Marshal(body)
	if err != nil {
		return app.Response{}, err
	}
	if err := Validate(schema, raw); err != nil {
		return app.Response{}, &app.ContractError{Local: true, Detail: "сообщение не прошло схему " + schema + " до отправки: " + err.Error()}
	}
	code, b, err := c.do(ctx, http.MethodPost, path, raw, map[string]string{"X-Message-Id": m.MessageID, "X-Contract-Version": ContractVersion})
	if err != nil {
		return app.Response{}, err
	}
	return classify(code, b)
}

// classify — класс ответа сервиса qc (docs/integrations/1c.md, §8).
func classify(code int, b []byte) (app.Response, error) {
	switch code {
	case http.StatusAccepted, http.StatusOK, http.StatusCreated:
		var r Receipt
		if err := json.Unmarshal(b, &r); err != nil || Validate(schemaReceipt, b) != nil {
			return app.Response{}, &app.ContractError{Detail: fmt.Sprintf("квитанция 1С не по контракту (HTTP %d): %s", code, snippet(b))}
		}
		out := app.Response{Outcome: ev.ErpPostingRespondedV1OutcomeAccepted, Receipt: r.Receipt, DocumentRef: r.DocumentRefKey, HTTPStatus: code}
		if code == http.StatusOK {
			// Повтор с тем же X-Message-Id: та же квитанция, второй документ не создан (AD-7).
			out.Outcome = ev.ErpPostingRespondedV1OutcomeDuplicate
		}
		return out, nil
	case http.StatusUnprocessableEntity:
		var e Error
		_ = json.Unmarshal(b, &e)
		return app.Response{Outcome: ev.ErpPostingRespondedV1OutcomeRejected, Code: e.Code, Message: e.Message, Field: e.Field,
			ErrorCode: string(errorCode(e.Code)), HTTPStatus: code}, nil
	case http.StatusBadRequest:
		var e Error
		_ = json.Unmarshal(b, &e)
		if e.Code == "CONTRACT_VERSION" || e.Code == "SCHEMA_VIOLATION" {
			return app.Response{}, &app.ContractError{Detail: fmt.Sprintf("1С отвергла сообщение по контракту: %s %s", e.Code, e.Message)}
		}
		return app.Response{Outcome: ev.ErpPostingRespondedV1OutcomeRejected, Code: e.Code, Message: e.Message, Field: e.Field,
			ErrorCode: string(errcodes.ErpDataError), HTTPStatus: code}, nil
	case http.StatusConflict:
		// Конфликт версии данных 1С — ограниченный повтор тем же номером.
		return app.Response{}, &app.TransportError{HTTPStatus: code, Err: errors.New("конфликт версии данных 1С: " + snippet(b))}
	}
	return app.Response{}, statusErr(code, b)
}

// errorCode — наш код ошибки (contracts/errors.yaml) по коду сервиса qc.
func errorCode(qc string) errcodes.Code {
	switch qc {
	case "CONTRACT_NOT_FOUND":
		return errcodes.ErpContractNotFound
	case "NOMENCLATURE_NOT_FOUND", "SERIES_NOT_FOUND", "WAREHOUSE_NOT_FOUND", "ITEM_NOT_FOUND":
		return errcodes.ErpIdMappingMissing
	}
	return errcodes.ErpDataError
}
