package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"uuid"

	dom "ant/internal/domain/ingest"
)

// Config — настройки edge-агента.
type Config struct {
	// SourceID — source_id устройства (AD-7: source_seq ведётся на source_id устройства).
	SourceID string
	// CoreURL — адрес ядра ant (роль api), например http://ant:8080.
	CoreURL string
	// StateDir — каталог состояния: номер source_seq и буфер исходных конвертов.
	StateDir string
	// BatchSize — сколько сообщений в пачке досылки.
	BatchSize int
	// FlushInterval — как часто отправлять накопленное при исправной связи.
	FlushInterval time.Duration
	// RetryMin, RetryMax — пауза между попытками при недоступности ядра (удваивается).
	RetryMin, RetryMax time.Duration
	// SourceKind, Reliability — значения по умолчанию для событий без них (FR-140).
	SourceKind, Reliability string
}

// Agent — edge-агент (кейс §3.2, FR-39, AD-7): ключ устройства, source_seq,
// буфер исходных подписанных конвертов на диске и досылка пачками после
// недоступности ядра с исходным occurred_at. Потерь нет — сообщение удаляется
// из буфера только после ответа ядра; дублей в учёте нет — повтор после
// потерянного ответа ядро подтверждает как дубль (FR-31).
type Agent struct {
	cfg    Config
	signer Signer
	client *http.Client
	now    func() time.Time
	log    *slog.Logger

	mu      sync.Mutex
	next    int64
	wake    chan struct{}
	lastErr string
	lastOK  time.Time
	sent    int64
}

// NewAgent открывает состояние в cfg.StateDir: номер следующего source_seq —
// наибольшее из сохранённого и номеров в буфере (после сбоя между записью в
// буфер и сохранением номера номер не повторится).
func NewAgent(cfg Config, signer Signer, client *http.Client, now func() time.Time, log *slog.Logger) (*Agent, error) {
	if cfg.SourceID == "" {
		return nil, errors.New("не задан source_id устройства")
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}
	if cfg.RetryMin <= 0 {
		cfg.RetryMin = 500 * time.Millisecond
	}
	if cfg.RetryMax < cfg.RetryMin {
		cfg.RetryMax = 30 * time.Second
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if now == nil {
		now = time.Now
	}
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	if err := os.MkdirAll(filepath.Join(cfg.StateDir, "buffer"), 0o700); err != nil {
		return nil, err
	}
	a := &Agent{cfg: cfg, signer: signer, client: client, now: now, log: log, next: 1, wake: make(chan struct{}, 1)}
	if b, err := os.ReadFile(a.seqPath()); err == nil {
		if n, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64); err == nil && n > a.next {
			a.next = n
		}
	}
	seqs, err := a.buffered()
	if err != nil {
		return nil, err
	}
	if len(seqs) > 0 && seqs[len(seqs)-1]+1 > a.next {
		a.next = seqs[len(seqs)-1] + 1
	}
	return a, nil
}

func (a *Agent) seqPath() string { return filepath.Join(a.cfg.StateDir, "next_seq") }

func (a *Agent) bufPath(seq int64) string {
	return filepath.Join(a.cfg.StateDir, "buffer", fmt.Sprintf("%020d.msg", seq))
}

// writeFile — запись с fsync и атомарной заменой: буфер переживает сбой питания.
func writeFile(path string, b []byte) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if d, err := os.Open(filepath.Dir(path)); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// buffered — номера сообщений в буфере по возрастанию.
func (a *Agent) buffered() ([]int64, error) {
	ents, err := os.ReadDir(filepath.Join(a.cfg.StateDir, "buffer"))
	if err != nil {
		return nil, err
	}
	var out []int64
	for _, e := range ents {
		name, ok := strings.CutSuffix(e.Name(), ".msg")
		if !ok {
			continue
		}
		if n, err := strconv.ParseInt(name, 10, 64); err == nil {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out, nil
}

// Enqueued — принятое агентом событие.
type Enqueued struct {
	SourceSeq int64  `json:"source_seq"`
	EventID   string `json:"event_id"`
}

// Enqueue дополняет событие источника полями устройства, подписывает ключом
// устройства и кладёт исходный конверт в буфер. source_id и source_seq ставит
// агент; event_id, occurred_at, correlation_id источника сохраняются (AD-38:
// event_id сценария агент не переназначает), пустые — заполняются.
func (a *Agent) Enqueue(ev map[string]any) (Enqueued, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	seq := a.next
	id, _ := ev["event_id"].(string)
	if id == "" {
		id = uuid.NewV7().String()
		ev["event_id"] = id
	}
	ev["source_id"] = a.cfg.SourceID
	ev["source_seq"] = seq
	if _, ok := ev["occurred_at"]; !ok {
		ev["occurred_at"] = a.now().UTC().Format("2006-01-02T15:04:05.000Z")
	}
	if _, ok := ev["correlation_id"]; !ok {
		ev["correlation_id"] = id
	}
	if _, ok := ev["causation_id"]; !ok {
		ev["causation_id"] = nil
	}
	if _, ok := ev["schema_version"]; !ok {
		ev["schema_version"] = 1
	}
	if _, ok := ev["source_kind"]; !ok && a.cfg.SourceKind != "" {
		ev["source_kind"] = a.cfg.SourceKind
	}
	if _, ok := ev["reliability"]; !ok && a.cfg.Reliability != "" {
		ev["reliability"] = a.cfg.Reliability
	}
	ev["integrity"] = map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{a.signer.KeyRef()}}
	raw, err := json.Marshal(ev)
	if err != nil {
		return Enqueued{}, err
	}
	canon, err := dom.Canonicalize(raw)
	if err != nil {
		return Enqueued{}, err
	}
	msg, err := a.signer.Sign(canon)
	if err != nil {
		return Enqueued{}, err
	}
	if err := writeFile(a.bufPath(seq), msg); err != nil {
		return Enqueued{}, err
	}
	if err := writeFile(a.seqPath(), []byte(strconv.FormatInt(seq+1, 10))); err != nil {
		return Enqueued{}, err
	}
	a.next = seq + 1
	select {
	case a.wake <- struct{}{}:
	default:
	}
	return Enqueued{SourceSeq: seq, EventID: id}, nil
}

// batchResult — ответ операции ingest.batch.submit (application/ingest.IngestResult).
type batchResult struct {
	Items []struct {
		Index  int     `json:"index"`
		Status string  `json:"status"`
		Code   *string `json:"code"`
	} `json:"items"`
}

// ErrCoreUnavailable — ядро недоступно или ответило ошибкой: сообщения
// остаются в буфере до следующей попытки.
var ErrCoreUnavailable = errors.New("ядро недоступно")

// Flush отправляет одну пачку из буфера (самые ранние номера — первыми) и
// удаляет из буфера всё, на что ядро дало окончательный ответ (принято, дубль,
// карантин, конфликт — сообщение уже у ядра). Возвращает число отправленных.
func (a *Agent) Flush(ctx context.Context) (int, error) {
	seqs, err := a.buffered()
	if err != nil || len(seqs) == 0 {
		return 0, err
	}
	seqs = seqs[:min(len(seqs), a.cfg.BatchSize)]
	msgs := make([]json.RawMessage, 0, len(seqs))
	for _, s := range seqs {
		b, err := os.ReadFile(a.bufPath(s))
		if err != nil {
			return 0, err
		}
		msgs = append(msgs, b)
	}
	body, err := json.Marshal(map[string]any{"source_id": a.cfg.SourceID,
		"sent_at": a.now().UTC().Format("2006-01-02T15:04:05.000Z"), "envelopes": msgs})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.cfg.CoreURL, "/")+"/api/v1/ingest/batches", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrCoreUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	rb, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if resp.StatusCode == http.StatusRequestEntityTooLarge && a.cfg.BatchSize > 1 {
		a.cfg.BatchSize /= 2
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: HTTP %d: %s", ErrCoreUnavailable, resp.StatusCode, strings.TrimSpace(string(rb)))
	}
	var br batchResult
	if err := json.Unmarshal(rb, &br); err != nil || len(br.Items) != len(seqs) {
		return 0, fmt.Errorf("%w: ответ на пачку не разобран: %v", ErrCoreUnavailable, err)
	}
	for _, r := range br.Items {
		if r.Index < 0 || r.Index >= len(seqs) {
			continue
		}
		switch r.Status {
		case "accepted", "duplicate", "quarantined", "rejected":
			// Окончательный ответ: сообщение у ядра (принято, повтор или в карантине).
			if err := os.Remove(a.bufPath(seqs[r.Index])); err != nil && !errors.Is(err, os.ErrNotExist) {
				return r.Index, err
			}
			if r.Status != "accepted" && r.Status != "duplicate" {
				code := ""
				if r.Code != nil {
					code = *r.Code
				}
				a.log.Warn("ядро не приняло сообщение", "source_seq", seqs[r.Index], "status", r.Status, "code", code)
			}
		}
	}
	a.mu.Lock()
	a.sent += int64(len(seqs))
	a.lastOK, a.lastErr = a.now(), ""
	a.mu.Unlock()
	return len(seqs), nil
}

// Drain досылает весь буфер пачками; первая ошибка прерывает.
func (a *Agent) Drain(ctx context.Context) (int, error) {
	total := 0
	for {
		n, err := a.Flush(ctx)
		total += n
		if err != nil || n == 0 {
			return total, err
		}
	}
}

// Run — цикл досылки: при исправной связи — раз в FlushInterval или сразу по
// новому событию; при недоступности ядра — повтор с удваивающейся паузой.
func (a *Agent) Run(ctx context.Context) error {
	wait := a.cfg.RetryMin
	for {
		_, err := a.Drain(ctx)
		d := a.cfg.FlushInterval
		if err != nil {
			a.mu.Lock()
			a.lastErr = err.Error()
			a.mu.Unlock()
			a.log.Warn("досылка отложена", "err", err, "retry_in", wait)
			d, wait = wait, min(wait*2, a.cfg.RetryMax)
		} else {
			wait = a.cfg.RetryMin
		}
		t := time.NewTimer(d)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil
		case <-a.wake:
			t.Stop()
			if err != nil {
				// при недоступности ждём паузу целиком, а не каждое новое событие
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(d):
				}
			}
		case <-t.C:
		}
	}
}

// Status — состояние агента.
type Status struct {
	SourceID  string    `json:"source_id"`
	NextSeq   int64     `json:"next_seq"`
	Buffered  int       `json:"buffered"`
	Sent      int64     `json:"sent"`
	LastOK    time.Time `json:"last_ok,omitzero"`
	LastError string    `json:"last_error,omitempty"`
	KeyRef    string    `json:"key_ref"`
	Signed    bool      `json:"signed"`
}

// Status — состояние агента.
func (a *Agent) Status() Status {
	seqs, _ := a.buffered()
	a.mu.Lock()
	defer a.mu.Unlock()
	return Status{SourceID: a.cfg.SourceID, NextSeq: a.next, Buffered: len(seqs), Sent: a.sent, LastOK: a.lastOK,
		LastError: a.lastErr, KeyRef: a.signer.KeyRef(), Signed: a.signer.Signs()}
}
