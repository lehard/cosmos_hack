package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
)

// HeadsSender — передача голов обеих цепочек хранителю (AD-8, FR-72): раз в
// N секунд — головы main и ca и звенья от последней принятой хранителем
// головы. Отказ хранителя (звенья не сходятся — цепочку переписали; откат) —
// тревога хранителя, которую IntegrityPoller пишет в шину безопасности
// (security.keeper.alert). Пустая передача (новых
// записей нет) — сигнал жизни: хранитель поднимает тревогу сам, если голов
// нет дольше 2N секунд.
type HeadsSender struct {
	Journal appjournal.JournalStore
	Keeper  Keeper
	Emit    Emitter
	// MaxLinks — звеньев в одной передаче (догон большого журнала — частями).
	MaxLinks int
	Log      *slog.Logger

	// lastReject — последний отказ хранителя: повтор того же отказа тревогу
	// не дублирует.
	lastReject string
}

// Chains — цепочки журнала в порядке передачи.
var Chains = []string{string(jc.JournalEntryChainMain), string(jc.JournalEntryChainCa)}

// Once — одна передача (и догон частями, если журнал ушёл вперёд).
func (h *HeadsSender) Once(ctx context.Context) error {
	max := h.MaxLinks
	if max <= 0 {
		max = 2000
	}
	for round := 0; round < 1000; round++ {
		last, err := h.Keeper.LatestCheckpoint(ctx)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		sub := HeadsSubmission{}
		var from, to string
		full := false
		for _, ch := range Chains {
			seq, link := last.Head(ch)
			es, err := h.Journal.Read(ctx, appjournal.ReadQuery{Chain: ch, AfterSeq: seq, Limit: max})
			if err != nil {
				return err
			}
			full = full || len(es) == max
			for _, e := range es {
				oh, err := dj.OpenHash(e)
				if err != nil {
					return err
				}
				sub.Links = append(sub.Links, Link{Chain: ch, Seq: int64(e.Seq), Commit: e.Commit, OpenFieldsHash: oh.String(), Link: e.Link})
				seq, link = int64(e.Seq), e.Link
				if from == "" || e.CommittedAt < from {
					from = e.CommittedAt
				}
				if e.CommittedAt > to {
					to = e.CommittedAt
				}
			}
			sub.Heads = append(sub.Heads, Head{Chain: ch, Seq: seq, Link: link})
		}
		if from == "" {
			from, to = last.Payload.CommittedAtTo, last.Payload.CommittedAtTo
			if from == "" {
				from = dj.FormatTime(time.Now())
				to = from
			}
		}
		sub.CommittedAtFrom, sub.CommittedAtTo = from, to
		_, err = h.Keeper.SubmitHeads(ctx, sub)
		var rej *RejectedError
		if errors.As(err, &rej) {
			// Тревогу поднимает сам хранитель (AD-8); в журнал её пишет
			// IntegrityPoller — здесь только журнал процесса, без повторов.
			if rej.Detail != h.lastReject && h.Log != nil {
				h.Log.Error("хранитель отверг головы — цепочку переписали или откатили", "code", rej.Code, "detail", rej.Detail)
			}
			h.lastReject = rej.Detail
			return err
		}
		h.lastReject = ""
		if err != nil || !full {
			return err
		}
	}
	return nil
}

// Run — передача раз в interval до отмены ctx.
func (h *HeadsSender) Run(ctx context.Context, interval time.Duration) error {
	return every(ctx, interval, func(ctx context.Context) {
		if err := h.Once(ctx); err != nil && ctx.Err() == nil && h.Log != nil {
			h.Log.Warn("хранитель: головы не переданы", "err", err)
		}
	})
}

// IntegrityPoller — ant забирает последний подписанный отчёт верификатора у
// хранителя и журналирует security.integrity.checked со ссылкой на его
// отпечаток (AD-46); нарушения из отчёта — security.integrity.violated с
// местом и типом (UJ-5). Тревоги хранителя (молчание сервера, отказ голов)
// — security.keeper.alert. Индикатор на столах читает эти записи.
type IntegrityPoller struct {
	Journal appjournal.JournalStore
	Keeper  Keeper
	Emit    Emitter
	Log     *slog.Logger
	// MaxViolations — сколько нарушений отчёта журналировать отдельными записями.
	MaxViolations int

	lastAlarm int64
	loaded    bool
	// seen — нарушения, уже записанные в журнал: пока нарушение держится,
	// каждый новый отчёт его повторяет, а в шину оно уходит один раз.
	seen map[string]bool
}

func violationKey(kind, detail string) string { return kind + "\x1f" + detail }

// Once — забрать отчёт и тревоги.
func (p *IntegrityPoller) Once(ctx context.Context) error {
	if err := p.alarms(ctx); err != nil && p.Log != nil {
		p.Log.Warn("хранитель: тревоги не получены", "err", err)
	}
	r, err := p.Keeper.LatestReport(ctx)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if seen, err := p.journaled(ctx, r.Digest); err != nil || seen {
		return err
	}
	return p.record(ctx, r)
}

// journaled — отчёт с этим отпечатком уже в журнале.
func (p *IntegrityPoller) journaled(ctx context.Context, digest string) (bool, error) {
	es, err := p.Journal.Read(ctx, appjournal.ReadQuery{EventType: string(catalog.SecurityIntegrityChecked), Backward: true, Limit: 20})
	if err != nil {
		return false, err
	}
	for _, e := range es {
		ev, err := open(ctx, p.Journal, e)
		if err != nil {
			continue
		}
		var d struct {
			ReportDigest string `json:"report_digest"`
		}
		if json.Unmarshal(ev.Data, &d) == nil && d.ReportDigest == digest {
			return true, nil
		}
	}
	return false, nil
}

// violationKinds — вид нарушения security.integrity.violated по коду находки.
var violationKinds = []string{"chain_link_mismatch", "checkpoint_mismatch", "projection_mismatch", "source_seq_gap", "late_write", "signature_invalid"}

// ViolationKind — вид нарушения по коду находки отчёта (`‹вид›` или `‹вид›.…`)
// и проверке.
func ViolationKind(code, check string) string {
	for _, k := range violationKinds {
		if code == k || strings.HasPrefix(code, k+".") {
			return k
		}
	}
	switch check {
	case "checkpoints":
		return "checkpoint_mismatch"
	case "projections":
		return "projection_mismatch"
	case "source_seq":
		return "source_seq_gap"
	case "signatures":
		return "signature_invalid"
	case "late_write":
		return "late_write"
	}
	return "chain_link_mismatch"
}

func (p *IntegrityPoller) record(ctx context.Context, r Report) error {
	at, err := dj.ParseTime(r.Payload.GeneratedAt)
	if err != nil {
		at = time.Now()
	}
	data := map[string]any{"report_digest": r.Digest, "verdict": string(r.Payload.Verdict), "checked_up_to_seq": r.Payload.Range.MainToSeq}
	if r.Payload.VerifierBuild != "" {
		data["verifier_build"] = r.Payload.VerifierBuild
	}
	outs := []Out{{Type: catalog.SecurityIntegrityChecked, OccurredAt: at, Data: data}}
	max := p.MaxViolations
	if max <= 0 {
		max = 20
	}
	if p.seen == nil {
		p.seen = p.journaledViolations(ctx)
	}
	var keys []string
	for _, c := range r.Payload.Checks {
		if c.Status != "rejected" {
			continue
		}
		for _, f := range c.Findings {
			if len(outs) > max {
				break
			}
			kind, detail := ViolationKind(f.Code, string(c.Check)), clip(f.Detail, 4000)
			if k := violationKey(kind, detail); p.seen[k] {
				continue
			} else {
				keys = append(keys, k)
			}
			v := map[string]any{"violation": kind, "detail": detail}
			if f.Seq != nil {
				v["seq"] = *f.Seq
			}
			if f.CaRef != nil {
				v["ca_ref"] = *f.CaRef
			}
			outs = append(outs, Out{Type: catalog.SecurityIntegrityViolated, OccurredAt: at, Causation: "", Data: v})
		}
	}
	if _, err = p.Emit.Emit(ctx, outs...); err != nil {
		return err
	}
	for _, k := range keys {
		p.seen[k] = true
	}
	return nil
}

// journaledViolations — нарушения, уже записанные в журнал (последние записи).
func (p *IntegrityPoller) journaledViolations(ctx context.Context) map[string]bool {
	out := map[string]bool{}
	es, err := p.Journal.Read(ctx, appjournal.ReadQuery{EventType: string(catalog.SecurityIntegrityViolated), Backward: true, Limit: 500})
	if err != nil {
		return out
	}
	for _, e := range es {
		ev, err := open(ctx, p.Journal, e)
		if err != nil {
			continue
		}
		var d struct {
			Violation string `json:"violation"`
			Detail    string `json:"detail"`
		}
		if json.Unmarshal(ev.Data, &d) == nil {
			out[violationKey(d.Violation, d.Detail)] = true
		}
	}
	return out
}

// alarms — тревоги хранителя, ещё не записанные в журнал.
func (p *IntegrityPoller) alarms(ctx context.Context) error {
	st, err := p.Keeper.Status(ctx)
	if err != nil {
		return err
	}
	if !p.loaded {
		p.lastAlarm, p.loaded = p.lastJournaledAlarm(ctx), true
	}
	var outs []Out
	for _, a := range st.Alarms {
		if a.No <= p.lastAlarm || !slices.Contains([]string{"heads_silent", "fork_attempt", "rollback_attempt", "checkpoint_gap"}, a.Alert) {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, a.At)
		if err != nil {
			at = time.Now()
		}
		chain := a.Chain
		if chain != "ca" {
			chain = "main"
		}
		outs = append(outs, Out{Type: catalog.SecurityKeeperAlert, OccurredAt: at, Data: map[string]any{
			"alert": a.Alert, "chain": chain, "last_seq": max(a.Seq, 0), "detail": clip(fmt.Sprintf("хранитель, тревога №%d: %s", a.No, a.Detail), 4000),
		}})
		p.lastAlarm = max(p.lastAlarm, a.No)
	}
	if len(outs) == 0 {
		return nil
	}
	_, err = p.Emit.Emit(ctx, outs...)
	return err
}

// lastJournaledAlarm — номер последней тревоги хранителя, уже записанной в журнал.
func (p *IntegrityPoller) lastJournaledAlarm(ctx context.Context) int64 {
	es, err := p.Journal.Read(ctx, appjournal.ReadQuery{EventType: string(catalog.SecurityKeeperAlert), Backward: true, Limit: 50})
	if err != nil {
		return 0
	}
	var last int64
	for _, e := range es {
		ev, err := open(ctx, p.Journal, e)
		if err != nil {
			continue
		}
		var d struct {
			Detail string `json:"detail"`
		}
		_ = json.Unmarshal(ev.Data, &d)
		var n int64
		if _, err := fmt.Sscanf(d.Detail, "хранитель, тревога №%d:", &n); err == nil {
			last = max(last, n)
		}
	}
	return last
}

// Run — опрос раз в interval до отмены ctx.
func (p *IntegrityPoller) Run(ctx context.Context, interval time.Duration) error {
	return every(ctx, interval, func(ctx context.Context) {
		if err := p.Once(ctx); err != nil && ctx.Err() == nil && p.Log != nil {
			p.Log.Warn("целостность: отчёт верификатора не получен", "err", err)
		}
	})
}

func every(ctx context.Context, d time.Duration, fn func(context.Context)) error {
	if d <= 0 {
		d = 10 * time.Second
	}
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		fn(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
