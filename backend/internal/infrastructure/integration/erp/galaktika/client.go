package galaktika

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	app "ant/internal/application/erp"
	"ant/internal/infrastructure/integration/schemacheck"
)

// Транспорты канала Галактики (ключ конфигурации erp.galaktika.transport).
const (
	// TransportDir — XML-пакеты в каталоге обмена (основной путь для 9.x).
	TransportDir = "exchange-dir"
	// TransportREST — JSON-форма пакета у REST-фасада «под Галактика ESB».
	TransportREST = "rest-facade"
)

// Config — канал обмена с Галактикой (конфигурация erp.galaktika; включённые
// системы и адреса — конфигурация, AD-18).
type Config struct {
	// Transport — exchange-dir | rest-facade.
	Transport string
	// BaseURL — адрес REST-фасада (`http(s)://‹хост›/galaktika/esb/v1`).
	BaseURL string
	// Dir — корень каталога обмена: out/ (ant → Галактика), ack/ (квитанции
	// на out), in/ (Галактика → ant), in-ack/ (наши квитанции на in), about.xml.
	Dir string
	// Node — наш узел обмена (ant), Peer — узел Галактики (пусто — из about).
	Node, Peer string
	// Enterprise — код предприятия в источнике сообщения.
	Enterprise string
	// Database — база Галактики для внешних ID (пусто — из about).
	Database string
	// User, Password — HTTP Basic фасада (пароль — файлом).
	User, Password string
	// Timeout — предел ответа фасада; дольше — транспортная ошибка (повтор).
	Timeout time.Duration
	// Stand — на месте Галактики работает stand (эпик 43).
	Stand bool
	// HTTP — клиент (nil — свой с Timeout).
	HTTP *http.Client
}

// Client — адаптер порта учёта для Галактики (application/erp.Ledger).
type Client struct {
	cfg  Config
	http *http.Client

	mu    sync.Mutex
	about *About
}

var _ app.Ledger = (*Client)(nil)

// New создаёт адаптер.
func New(cfg Config) (*Client, error) {
	switch cfg.Transport {
	case "":
		cfg.Transport = TransportREST
		if cfg.BaseURL == "" && cfg.Dir != "" {
			cfg.Transport = TransportDir
		}
	case TransportREST, TransportDir:
	default:
		return nil, fmt.Errorf("galaktika: неизвестный транспорт %q (exchange-dir | rest-facade)", cfg.Transport)
	}
	if cfg.Transport == TransportREST && cfg.BaseURL == "" {
		return nil, errors.New("galaktika: не задан адрес REST-фасада (erp.galaktika.base_url)")
	}
	if cfg.Transport == TransportDir && cfg.Dir == "" {
		return nil, errors.New("galaktika: не задан каталог обмена (erp.galaktika.dir)")
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.Node == "" {
		cfg.Node = "ant"
	}
	if cfg.Enterprise == "" {
		cfg.Enterprise = "ENT01"
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{cfg: cfg, http: hc}, nil
}

// Info — канал обмена.
func (c *Client) Info() app.LedgerInfo {
	ep := c.cfg.BaseURL
	if c.cfg.Transport == TransportDir {
		ep = "file://" + c.cfg.Dir
	}
	return app.LedgerInfo{System: "galaktika", Endpoint: ep, Stand: c.cfg.Stand, ContractVersion: ContractVersion}
}

// peer — узел и база Галактики: из конфигурации или из about.
func (c *Client) peer() (node, db string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	node, db = c.cfg.Peer, c.cfg.Database
	if c.about != nil {
		node, db = or(node, c.about.Node), or(db, c.about.Database)
	}
	return or(node, "GAL-ERP"), or(db, "main")
}

// Check — сверка ответной стороны (AD-18): версия контракта обработчика
// Галактики (`about`); нашей версии нет среди поддерживаемых — ContractError.
func (c *Client) Check(ctx context.Context) (app.Checked, error) {
	var (
		raw   []byte
		isXML bool
	)
	if c.cfg.Transport == TransportDir {
		b, err := os.ReadFile(filepath.Join(c.cfg.Dir, "about.xml"))
		if errors.Is(err, os.ErrNotExist) {
			return app.Checked{}, &app.ContractError{Detail: "в каталоге обмена нет about.xml — обработчик Галактики не объявил версию контракта"}
		}
		if err != nil {
			return app.Checked{}, &app.TransportError{Err: err}
		}
		raw, isXML = b, true
	} else {
		code, b, err := c.do(ctx, http.MethodGet, "/about", nil, nil)
		if err != nil {
			return app.Checked{}, err
		}
		if err := statusErr(code, b); err != nil {
			return app.Checked{}, err
		}
		raw = b
	}
	var a About
	var err error
	if isXML {
		err = xml.Unmarshal(raw, &a)
	} else {
		err = json.Unmarshal(raw, &a)
	}
	if err != nil {
		return app.Checked{}, &app.ContractError{Detail: "описание ответной стороны не разобрано: " + err.Error()}
	}
	j, _ := json.Marshal(a)
	if err := schemacheck.Validate(schemaAbout, j); err != nil {
		f, d := schemacheck.Violation(err)
		return app.Checked{}, &app.ContractError{Detail: "описание ответной стороны не по контракту: " + f + ": " + d}
	}
	if !slices.Contains(a.Supported, ContractVersion) {
		return app.Checked{}, &app.ContractError{Detail: fmt.Sprintf("обработчик Галактики поддерживает %v, адаптер — %s", a.Supported, ContractVersion)}
	}
	c.mu.Lock()
	c.about = &a
	c.mu.Unlock()
	return app.Checked{ContractVersion: a.Contract, Detail: fmt.Sprintf("Галактика %s (база %s, %s): контракт %s", a.Node, a.Database, c.cfg.Transport, a.Contract)}, nil
}

// Post — отправить пакет gal.qc.v1 с номером пакета = X-Message-Id. Пакет
// сначала проверяется схемой контракта: не по схеме — в Галактику ничего не
// уходит (ContractError{Local}, FR-111).
func (c *Client) Post(ctx context.Context, m app.Outgoing) (app.Response, error) {
	peer, _ := c.peer()
	resource, e := Encode(c.cfg.Node, peer, c.cfg.Enterprise, m)
	if err := Check(e); err != nil {
		return app.Response{}, &app.ContractError{Local: true, Detail: "пакет не прошёл схему gal.qc.v1 до отправки: " + err.Error()}
	}
	if c.cfg.Transport == TransportDir {
		return c.postDir(e)
	}
	body, err := json.Marshal(e)
	if err != nil {
		return app.Response{}, err
	}
	code, b, err := c.do(ctx, http.MethodPost, resource, body, map[string]string{"X-Message-Id": m.MessageID, "X-Contract-Version": ContractVersion})
	if err != nil {
		return app.Response{}, err
	}
	return classify(code, b)
}

// classify — класс ответа REST-фасада (docs/integrations/galaktika.md, §8).
func classify(code int, b []byte) (app.Response, error) {
	switch code {
	case http.StatusAccepted, http.StatusCreated, http.StatusOK:
		a, err := parseAck(b, false)
		if err != nil || a.Status != "ok" {
			return app.Response{}, &app.ContractError{Detail: fmt.Sprintf("квитанция Галактики не по контракту (HTTP %d): %s", code, snippet(b))}
		}
		return response(a, code), nil
	case http.StatusUnprocessableEntity, http.StatusBadRequest:
		a, err := parseAck(b, false)
		if err != nil {
			return app.Response{}, &app.ContractError{Detail: fmt.Sprintf("ошибка Галактики не по контракту (HTTP %d): %s", code, snippet(b))}
		}
		if contractCode(a.Code) {
			return app.Response{}, &app.ContractError{Detail: fmt.Sprintf("Галактика отвергла пакет по контракту: %s %s", a.Code, a.Text)}
		}
		return response(a, code), nil
	case http.StatusConflict:
		return app.Response{}, &app.TransportError{HTTPStatus: code, Err: errors.New("конфликт версии данных Галактики: " + snippet(b))}
	}
	return app.Response{}, statusErr(code, b)
}

// postDir — пакет в каталог обмена и квитанция, если Галактика её уже
// положила. Нет квитанции — транспортная ошибка: повтор с тем же номером
// пакета по расписанию, затем карантин (файл пакета не переписывается).
func (c *Client) postDir(e Exchange) (app.Response, error) {
	out := filepath.Join(c.cfg.Dir, "out")
	name := safeName(e.MessageID) + ".xml"
	if _, err := os.Stat(filepath.Join(out, name)); errors.Is(err, os.ErrNotExist) {
		b, err := MarshalXML(e)
		if err != nil {
			return app.Response{}, err
		}
		if err := writeAtomic(out, name, b); err != nil {
			return app.Response{}, &app.TransportError{Err: err}
		}
	} else if err != nil {
		return app.Response{}, &app.TransportError{Err: err}
	}
	b, err := os.ReadFile(filepath.Join(c.cfg.Dir, "ack", name))
	if errors.Is(err, os.ErrNotExist) {
		return app.Response{}, &app.TransportError{Err: errors.New("квитанция GalAck на пакет " + e.MessageID + " ещё не получена")}
	}
	if err != nil {
		return app.Response{}, &app.TransportError{Err: err}
	}
	a, err := parseAck(b, true)
	if err != nil {
		return app.Response{}, &app.ContractError{Detail: err.Error()}
	}
	if a.Status != "ok" && contractCode(a.Code) {
		return app.Response{}, &app.ContractError{Detail: fmt.Sprintf("Галактика отвергла пакет по контракту: %s %s", a.Code, a.Text)}
	}
	return response(a, 0), nil
}

// Pull — входящие пакеты Галактики → факты порта учёта (задания, партии,
// каталог МЦ, соответствия внешних ID). Пакет не по контракту не даёт фактов
// (в каталоге обмена — квитанция с ошибкой SCHEMA_VIOLATION).
func (c *Client) Pull(ctx context.Context) ([]app.Inbound, error) {
	_, db := c.peer()
	var packets []Exchange
	if c.cfg.Transport == TransportDir {
		ps, err := c.pullDir()
		if err != nil {
			return nil, err
		}
		packets = ps
	} else {
		code, b, err := c.do(ctx, http.MethodGet, "/exchange/outbox", nil, nil)
		if err != nil {
			return nil, err
		}
		if err := statusErr(code, b); err != nil {
			return nil, err
		}
		if err := schemacheck.Validate(schemaInbox, b); err != nil {
			f, d := schemacheck.Violation(err)
			return nil, &app.ContractError{Detail: "входящие пакеты фасада не по контракту: " + f + ": " + d}
		}
		var in Inbox
		if err := json.Unmarshal(b, &in); err != nil {
			return nil, &app.ContractError{Detail: "входящие пакеты фасада не разобраны: " + err.Error()}
		}
		packets = in.Packets
	}
	var out []app.Inbound
	for _, p := range packets {
		out = append(out, Inbound(db, p)...)
	}
	return out, nil
}

func (c *Client) pullDir() ([]Exchange, error) {
	in := filepath.Join(c.cfg.Dir, "in")
	ents, err := os.ReadDir(in)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &app.TransportError{Err: err}
	}
	var out []Exchange
	for _, en := range ents {
		if en.IsDir() || !strings.HasSuffix(en.Name(), ".xml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(in, en.Name()))
		if err != nil {
			return nil, &app.TransportError{Err: err}
		}
		id := strings.TrimSuffix(en.Name(), ".xml")
		e, err := ParseExchangeXML(b)
		if err == nil {
			id = or(e.MessageID, id)
			err = Check(e)
		}
		ack := Ack{MessageID: id, Status: "ok"}
		if err != nil {
			ack = Ack{MessageID: id, Status: "error", Code: "SCHEMA_VIOLATION", Text: trim(err.Error(), 1000)}
		} else {
			out = append(out, e)
		}
		// Квитанция на входящий пакет: повторное чтение даёт те же event_id,
		// поэтому квитанция «принято» не приводит к двойному учёту (AD-7).
		name := safeName(ack.MessageID) + ".xml"
		if _, serr := os.Stat(filepath.Join(c.cfg.Dir, "in-ack", name)); errors.Is(serr, os.ErrNotExist) {
			ab, merr := MarshalXML(ack)
			if merr == nil {
				_ = writeAtomic(filepath.Join(c.cfg.Dir, "in-ack"), name, ab)
			}
		}
	}
	return out, nil
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

// statusErr — ошибки чтения: 5xx, 408, 429 — транспорт; 401/403 и прочие 4xx — контракт.
func statusErr(code int, body []byte) error {
	switch {
	case code >= 200 && code < 300:
		return nil
	case code >= 500 || code == http.StatusRequestTimeout || code == http.StatusTooManyRequests:
		return &app.TransportError{HTTPStatus: code, Err: errors.New(snippet(body))}
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		return &app.ContractError{Detail: fmt.Sprintf("отказ доступа Галактики (HTTP %d)", code)}
	}
	return &app.ContractError{Detail: fmt.Sprintf("HTTP %d: %s", code, snippet(body))}
}

func snippet(b []byte) string { return trim(strings.TrimSpace(string(b)), 300) }

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// safeName — имя файла пакета по его номеру (ASCII без разделителей пути).
func safeName(id string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, id)
}

// writeAtomic — файл целиком или никак: временный файл и переименование
// (Галактика не прочтёт недописанный пакет).
func writeAtomic(dir, name string, b []byte) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+name+".*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, name))
}
