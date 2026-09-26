package keeper

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	app "ant/internal/application/security"
)

// Client — mTLS-клиент хранителя (contracts/internal/keeper.openapi.yaml):
// реализация порта application/security.Keeper для ant и верификатора.
type Client struct {
	Base string
	HTTP *http.Client
}

var _ app.Keeper = (*Client)(nil)

// New — клиент хранителя по адресу base (https://keeper:8444) с tls (mtls.Client).
func New(base string, tlsCfg *tls.Config) *Client {
	return &Client{Base: strings.TrimRight(base, "/"), HTTP: &http.Client{Timeout: 30 * time.Second,
		Transport: &http.Transport{TLSClientConfig: tlsCfg, MaxIdleConns: 4, IdleConnTimeout: time.Minute}}}
}

type problem struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Title  string `json:"title"`
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	rq, err := http.NewRequestWithContext(ctx, method, c.Base+path, rd)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		rq.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(rq)
	if err != nil {
		return nil, 0, fmt.Errorf("хранитель недоступен: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	return b, resp.StatusCode, err
}

func fail(code int, b []byte) error {
	var p problem
	_ = json.Unmarshal(b, &p)
	switch code {
	case http.StatusNotFound:
		return app.ErrNotFound
	case http.StatusConflict:
		return &app.RejectedError{Code: p.Code, Detail: p.Detail}
	}
	return fmt.Errorf("хранитель: %d %s %s", code, p.Code, p.Detail)
}

// SubmitHeads — POST /v1/heads.
func (c *Client) SubmitHeads(ctx context.Context, s app.HeadsSubmission) (app.Checkpoint, error) {
	body, _ := json.Marshal(s)
	b, code, err := c.do(ctx, http.MethodPost, "/v1/heads", body)
	if err != nil {
		return app.Checkpoint{}, err
	}
	if code != http.StatusOK {
		return app.Checkpoint{}, fail(code, b)
	}
	return app.ParseCheckpoint(b)
}

// LatestCheckpoint — GET /v1/checkpoints/latest.
func (c *Client) LatestCheckpoint(ctx context.Context) (app.Checkpoint, error) {
	b, code, err := c.do(ctx, http.MethodGet, "/v1/checkpoints/latest", nil)
	if err != nil {
		return app.Checkpoint{}, err
	}
	if code != http.StatusOK {
		return app.Checkpoint{}, fail(code, b)
	}
	return app.ParseCheckpoint(b)
}

// Checkpoints — GET /v1/checkpoints?after_no=&limit=.
func (c *Client) Checkpoints(ctx context.Context, afterNo int64, limit int) ([]app.Checkpoint, error) {
	b, code, err := c.do(ctx, http.MethodGet, "/v1/checkpoints?after_no="+strconv.FormatInt(afterNo, 10)+"&limit="+strconv.Itoa(limit), nil)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fail(code, b)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := make([]app.Checkpoint, 0, len(raw))
	for _, r := range raw {
		cp, err := app.ParseCheckpoint(r)
		if err != nil {
			return nil, err
		}
		out = append(out, cp)
	}
	return out, nil
}

// SubmitReport — POST /v1/verifier-reports.
func (c *Client) SubmitReport(ctx context.Context, env []byte) error {
	b, code, err := c.do(ctx, http.MethodPost, "/v1/verifier-reports", env)
	if err != nil {
		return err
	}
	if code != http.StatusCreated && code != http.StatusOK {
		return fail(code, b)
	}
	return nil
}

func (c *Client) report(ctx context.Context, path string) (app.Report, error) {
	b, code, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return app.Report{}, err
	}
	if code != http.StatusOK {
		return app.Report{}, fail(code, b)
	}
	return app.ParseReport(b)
}

// LatestReport — GET /v1/verifier-reports/latest.
func (c *Client) LatestReport(ctx context.Context) (app.Report, error) {
	return c.report(ctx, "/v1/verifier-reports/latest")
}

// Report — GET /v1/verifier-reports/{digest}.
func (c *Client) Report(ctx context.Context, digest string) (app.Report, error) {
	return c.report(ctx, "/v1/verifier-reports/"+url.PathEscape(digest))
}

// Reports — GET /v1/verifier-reports?limit=.
func (c *Client) Reports(ctx context.Context, limit int) ([]app.Report, error) {
	b, code, err := c.do(ctx, http.MethodGet, "/v1/verifier-reports?limit="+strconv.Itoa(limit), nil)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fail(code, b)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	var out []app.Report
	for _, r := range raw {
		rp, err := app.ParseReport(r)
		if err != nil {
			return nil, err
		}
		out = append(out, rp)
	}
	return out, nil
}

// Status — GET /v1/status.
func (c *Client) Status(ctx context.Context) (app.KeeperStatus, error) {
	b, code, err := c.do(ctx, http.MethodGet, "/v1/status", nil)
	if err != nil {
		return app.KeeperStatus{}, err
	}
	if code != http.StatusOK {
		return app.KeeperStatus{}, fail(code, b)
	}
	var s app.KeeperStatus
	return s, json.Unmarshal(b, &s)
}

// Links — GET /v1/links?chain=&after_seq=&limit=.
func (c *Client) Links(ctx context.Context, chain string, afterSeq int64, limit int) ([]app.Link, error) {
	b, code, err := c.do(ctx, http.MethodGet, "/v1/links?chain="+url.QueryEscape(chain)+"&after_seq="+strconv.FormatInt(afterSeq, 10)+"&limit="+strconv.Itoa(limit), nil)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fail(code, b)
	}
	var out []app.Link
	return out, json.Unmarshal(b, &out)
}
