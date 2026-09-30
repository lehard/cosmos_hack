package skud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	accessapp "ant/internal/application/access"
)

// Протокол skud.v1 — REST-журнал проходов СКУД, который опрашивает адаптер
// (типичная схема промышленных СКУД: события с возрастающим номером и
// выборка «после номера»). Внешний формат не выходит за адаптер: наружу —
// accessapp.ZonePassIn.

// Contract — версия протокола, с которой работает адаптер.
const Contract = "skud.v1"

// Направления прохода в протоколе.
const (
	DirIn  = "in"
	DirOut = "out"
)

// ZoneInfo — зона доступа в ответе about.
type ZoneInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// About — ответная сторона: система, версия протокола, зоны.
type About struct {
	System   string     `json:"system"`
	Contract string     `json:"contract"`
	Zones    []ZoneInfo `json:"zones"`
}

// Event — проход через турникет.
type Event struct {
	// Seq — номер события в журнале СКУД (возрастает); ID — однозначный id события.
	Seq       int64     `json:"seq"`
	ID        string    `json:"id"`
	Time      time.Time `json:"time"`
	Holder    string    `json:"holder"`
	Zone      string    `json:"zone"`
	Reader    string    `json:"reader,omitempty"`
	Direction string    `json:"direction"`
}

// EventPage — выборка событий после номера и номер последнего события журнала.
type EventPage struct {
	Events  []Event `json:"events"`
	LastSeq int64   `json:"last_seq"`
}

// PassRequest — заказ прохода у stand-а (эмуляция турникета).
type PassRequest struct {
	Holder    string `json:"holder"`
	Zone      string `json:"zone"`
	Direction string `json:"direction"`
}

// Client — клиент протокола skud.v1 (адрес — конфигурация access.skud.base_url).
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient — клиент с таймаутом запроса.
func NewClient(base string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{BaseURL: strings.TrimRight(base, "/"), HTTP: &http.Client{Timeout: timeout}}
}

// About — сверка ответной стороны: версия протокола должна совпасть.
func (c *Client) About(ctx context.Context) (About, error) {
	var a About
	if err := c.get(ctx, "/about", &a); err != nil {
		return a, err
	}
	if a.Contract != Contract {
		return a, fmt.Errorf("СКУД: протокол %q, ожидается %s", a.Contract, Contract)
	}
	return a, nil
}

// Events — события после номера after (в порядке номеров) и номер последнего события.
func (c *Client) Events(ctx context.Context, after int64) ([]accessapp.SKUDEvent, int64, error) {
	var p EventPage
	if err := c.get(ctx, "/events?after="+strconv.FormatInt(after, 10)+"&limit=500", &p); err != nil {
		return nil, 0, err
	}
	out := make([]accessapp.SKUDEvent, 0, len(p.Events))
	for _, e := range p.Events {
		out = append(out, accessapp.SKUDEvent{Seq: e.Seq, Pass: accessapp.ZonePassIn{SourceEventID: e.ID, PersonID: e.Holder, ZoneID: e.Zone,
			ReaderID: e.Reader, Enter: e.Direction == DirIn, At: e.Time}})
	}
	return out, p.LastSeq, nil
}

// Endpoint — адрес ответной стороны.
func (c *Client) Endpoint() string { return c.BaseURL }

func (c *Client) get(ctx context.Context, path string, v any) error {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return err
	}
	rq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(rq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("СКУД: %s %d: %s", path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(v)
}
