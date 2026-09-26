package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Client — обычный клиент API ant (AD-26): вход демо-персоной через
// IdentityProvider, операции — по operationId из спецификации /api/v1/openapi.json.
type Client struct {
	Base string
	HTTP *http.Client

	mu      sync.Mutex
	ops     map[string]Operation
	cookies map[string]string
}

// Operation — маршрут и x-ant-action операции.
type Operation struct {
	Method         string
	Path           string
	Emits          []string
	SignatureLevel int
	Critical       bool
}

// Login — сеанс демо-персоны (POST /api/v1/auth/session, cookie ant_session).
func (c *Client) Login(ctx context.Context, persona string) (string, error) {
	c.mu.Lock()
	if v, ok := c.cookies[persona]; ok {
		c.mu.Unlock()
		return v, nil
	}
	c.mu.Unlock()
	b, _ := json.Marshal(map[string]string{"persona_id": persona})
	rq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/api/v1/auth/session", bytes.NewReader(b))
	rq.Header.Set("Content-Type", "application/json")
	rs, err := c.HTTP.Do(rq)
	if err != nil {
		return "", err
	}
	defer func() { _ = rs.Body.Close() }()
	_, _ = io.Copy(io.Discard, rs.Body)
	if rs.StatusCode/100 != 2 {
		return "", fmt.Errorf("вход персоной %s: HTTP %d", persona, rs.StatusCode)
	}
	for _, ck := range rs.Cookies() {
		if ck.Name == "ant_session" {
			c.mu.Lock()
			if c.cookies == nil {
				c.cookies = map[string]string{}
			}
			c.cookies[persona] = ck.Value
			c.mu.Unlock()
			return ck.Value, nil
		}
	}
	return "", errors.New("вход персоной " + persona + ": нет cookie сеанса")
}

// Op — операция по operationId (спецификация читается один раз).
func (c *Client) Op(ctx context.Context, id string) (Operation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ops == nil {
		rq, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+"/api/v1/openapi.json", nil)
		rs, err := c.HTTP.Do(rq)
		if err != nil {
			return Operation{}, err
		}
		defer func() { _ = rs.Body.Close() }()
		var spec struct {
			Paths map[string]map[string]struct {
				OperationID string `json:"operationId"`
				Action      struct {
					Emits          []string `json:"emits"`
					SignatureLevel int      `json:"signature_level"`
					Critical       bool     `json:"critical"`
				} `json:"x-ant-action"`
			} `json:"paths"`
		}
		if err := json.NewDecoder(rs.Body).Decode(&spec); err != nil {
			return Operation{}, fmt.Errorf("спецификация API: %w", err)
		}
		c.ops = map[string]Operation{}
		for p, ms := range spec.Paths {
			for m, o := range ms {
				c.ops[o.OperationID] = Operation{Method: strings.ToUpper(m), Path: p, Emits: o.Action.Emits,
					SignatureLevel: o.Action.SignatureLevel, Critical: o.Action.Critical}
			}
		}
	}
	o, ok := c.ops[id]
	if !ok {
		return o, fmt.Errorf("операции %s в API нет", id)
	}
	return o, nil
}

// Do — вызов операции от имени персоны: параметры пути подставляются в
// маршрут, остальные — в строку запроса.
func (c *Client) Do(ctx context.Context, persona, opID string, params map[string]string, body any) (int, []byte, error) {
	o, err := c.Op(ctx, opID)
	if err != nil {
		return 0, nil, err
	}
	cookie, err := c.Login(ctx, persona)
	if err != nil {
		return 0, nil, err
	}
	path, q := o.Path, url.Values{}
	for k, v := range params {
		if strings.Contains(path, "{"+k+"}") {
			path = strings.ReplaceAll(path, "{"+k+"}", url.PathEscape(v))
		} else {
			q.Set(k, v)
		}
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var rd io.Reader
	if body != nil && o.Method != http.MethodGet {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	rq, _ := http.NewRequestWithContext(ctx, o.Method, c.Base+path, rd)
	rq.Header.Set("Content-Type", "application/json")
	rq.AddCookie(&http.Cookie{Name: "ant_session", Value: cookie})
	rs, err := c.HTTP.Do(rq)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = rs.Body.Close() }()
	b, err := io.ReadAll(rs.Body)
	return rs.StatusCode, b, err
}

// Read — чтение операцией в JSON (числа — json.Number).
func (c *Client) Read(ctx context.Context, persona, opID string, params map[string]string) (map[string]any, error) {
	st, b, err := c.Do(ctx, persona, opID, params, nil)
	if err != nil {
		return nil, err
	}
	if st != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", opID, st)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	return m, dec.Decode(&m)
}
