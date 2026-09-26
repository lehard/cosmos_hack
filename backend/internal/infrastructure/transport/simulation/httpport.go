package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"

	app "ant/internal/application/simulation"
	"ant/internal/infrastructure/transport/httpapi"
)

// HTTPPort — порты Probe и Actor симуляции поверх того же HTTP API ant в
// процессе (AD-26): автосверка читает теми же операциями (operationId), что
// показывают столы, а demo-signer принимает решения теми же командами от
// имени демо-персоны (заголовок Ant-Demo-Persona, профили fixtures и demo) —
// с правами, местом сеанса и гардами общего декоратора. Служебных обходов нет.
type HTTPPort struct {
	handler http.Handler
	routes  map[string]route
	emits   map[string][]string
	// ReadPersona — демо-персона сверки (чтение): по умолчанию администратор.
	ReadPersona string
}

type route struct {
	method, path string
}

// NewHTTPPort — порты над API api и обработчиком h (тот же mux, что слушает
// роль api). Маршруты — из спецификации Huma по operationId.
func NewHTTPPort(api *httpapi.API, h http.Handler) *HTTPPort {
	p := &HTTPPort{handler: h, routes: map[string]route{}, emits: map[string][]string{}, ReadPersona: "ADM-01"}
	for path, item := range api.Huma().OpenAPI().Paths {
		if item.Get != nil {
			p.routes[item.Get.OperationID] = route{http.MethodGet, path}
		}
		if item.Post != nil {
			p.routes[item.Post.OperationID] = route{http.MethodPost, path}
		}
		if item.Delete != nil {
			p.routes[item.Delete.OperationID] = route{http.MethodDelete, path}
		}
	}
	for _, a := range api.Actions() {
		for _, t := range a.Emits {
			p.emits[a.ID] = append(p.emits[a.ID], string(t))
		}
	}
	return p
}

var (
	_ app.Probe = (*HTTPPort)(nil)
	_ app.Actor = (*HTTPPort)(nil)
)

// Read — чтение операцией operationId в пределах прогона (run_id, AD-38).
func (p *HTTPPort) Read(ctx context.Context, operation string, params map[string]string, runID string) (any, error) {
	q := map[string]string{}
	for k, v := range params {
		q[k] = v
	}
	if runID != "" {
		q["run_id"] = runID
	}
	status, body, err := p.do(ctx, p.ReadPersona, operation, http.MethodGet, q, nil)
	if err != nil {
		return nil, err
	}
	switch status {
	case http.StatusOK:
		return decode(body)
	case http.StatusNotImplemented:
		return nil, app.ErrUnavailable
	case http.StatusNotFound:
		return nil, app.ErrNotFound
	}
	return nil, fmt.Errorf("%s: HTTP %d: %s", operation, status, problemCode(body))
}

// Act — команда operationId от имени демо-персоны (demo-signer, AD-26):
// basis_seq — голова журнала прогона, policy_seq — версия политики сеанса,
// если шаг их не задал.
func (p *HTTPPort) Act(ctx context.Context, persona, operation string, params map[string]string, body map[string]any) (app.ActResult, error) {
	if v, ok := body["basis_seq"]; !ok || fmt.Sprint(v) == "0" {
		if head, err := p.Read(ctx, "journal.head.read", nil, ""); err == nil {
			if m, ok := head.(map[string]any); ok && m["seq"] != nil {
				body["basis_seq"] = m["seq"]
			}
		}
	}
	if v, ok := body["policy_seq"]; !ok || fmt.Sprint(v) == "0" {
		if _, b, err := p.do(ctx, persona, "access.session.read", http.MethodGet, nil, nil); err == nil {
			if s, err := decode(b); err == nil {
				if m, ok := s.(map[string]any); ok && m["policy_seq"] != nil {
					body["policy_seq"] = m["policy_seq"]
				}
			}
		}
	}
	status, b, err := p.do(ctx, persona, operation, "", params, body)
	if err != nil {
		return app.ActResult{}, err
	}
	res := app.ActResult{Status: status}
	switch {
	case status == http.StatusNotImplemented:
		return res, app.ErrUnavailable
	case status >= 200 && status < 300:
		var r struct {
			Seq      int64    `json:"seq"`
			EventIDs []string `json:"event_ids"`
		}
		_ = json.Unmarshal(b, &r)
		res.Seq, res.EventIDs = r.Seq, r.EventIDs
	default:
		res.Code = problemCode(b)
	}
	return res, nil
}

// Decided — в журнале прогона после since есть запись, которую эмитит operation
// (решение принято на столе роли, FR-129).
func (p *HTTPPort) Decided(ctx context.Context, runID, operation string, since int64) (bool, int64, error) {
	for _, t := range p.emits[operation] {
		doc, err := p.Read(ctx, "journal.entry.list", map[string]string{"event_type": t, "after_seq": fmt.Sprint(since)}, runID)
		if err != nil {
			return false, 0, err
		}
		m, _ := doc.(map[string]any)
		items, _ := m["items"].([]any)
		for _, it := range items {
			e, _ := it.(map[string]any)
			if n, ok := e["seq"].(json.Number); ok {
				seq, _ := n.Int64()
				if seq > since {
					return true, seq, nil
				}
			}
		}
	}
	return false, 0, nil
}

// do — запрос к обработчику API в процессе.
func (p *HTTPPort) do(ctx context.Context, persona, operation, method string, params map[string]string, body map[string]any) (int, []byte, error) {
	r, ok := p.routes[operation]
	if !ok {
		return 0, nil, fmt.Errorf("операции %s нет в API", operation)
	}
	if method == "" {
		method = r.method
	}
	path := r.path
	q := url.Values{}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		ph := "{" + k + "}"
		if strings.Contains(path, ph) {
			path = strings.ReplaceAll(path, ph, url.PathEscape(params[k]))
			continue
		}
		q.Set(k, params[k])
	}
	if strings.Contains(path, "{") {
		return 0, nil, fmt.Errorf("%s: не хватает параметров пути в %s", operation, path)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var rd *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequestWithContext(ctx, method, path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if persona != "" {
		req.Header.Set(httpapi.DemoPersonaHeader, persona)
	}
	rec := httptest.NewRecorder()
	p.handler.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes(), nil
}

func decode(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// problemCode — код ошибки problem+json (RFC 9457, FR-28).
func problemCode(b []byte) string {
	var pr struct {
		Code  string `json:"code"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal(b, &pr); err != nil {
		return strings.TrimSpace(string(b))
	}
	if pr.Code != "" {
		return pr.Code
	}
	return pr.Title
}
