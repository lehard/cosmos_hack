package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	dom "ant/internal/domain/security"
)

// Сервис доверенных решений (CriticalActions, AD-28, FR-77, FR-146; кейс
// §3.2): единственный путь критического действия —
//
//	Execute(команда, fn) → гард → fn формирует основную запись и вызывает
//	journal.Append → Append в той же транзакции вызывает CriticalHook →
//	domain/security.BuildCA → запись цепочки ca с номером CA-‹n› и commit
//	основной записи.
//
// Модули записей CA не создают; для реакций движка и фактов приёма с
// признаком критичности (действия исполнителя) Append вызывает тот же
// CriticalHook — критическое действие без записи CA в журнал не попадает.
// Отмена — только новой записью с Command.Cancels и причиной.

type commandKey struct{}

// Execute — критическое действие (AD-28): гард команды, затем fn пишет
// основную запись через journal.Append; запись CA по сведениям команды
// (кто, полномочие и клеймо с ревизией политики, основание, было → стало,
// отмена) формирует Append в той же транзакции (CriticalHook). Кто —
// субъект сеанса, если команда его не назвала.
func Execute(ctx context.Context, cmd dom.Command, fn func(ctx context.Context) error) error {
	if cmd.ActorID == "" {
		cmd.ActorID = platform.PrincipalFrom(ctx).PersonID
	}
	if err := guard(cmd); err != nil {
		return err
	}
	return fn(context.WithValue(ctx, commandKey{}, cmd))
}

// ErrCancel — отмена критического действия без причины или не по форме.
var ErrCancel = errors.New("security: отмена критического действия — только новой записью «отменяет CA-‹n›» с причиной (AD-28)")

// guard — гард команды критического действия: отмена — с причиной и по
// номеру; полномочия и класс действия проверяет общий декоратор прав
// (access.Gate) до команды, гарды предметной области — модуль-владелец.
func guard(cmd dom.Command) error {
	if cmd.Cancels == "" {
		return nil
	}
	if strings.TrimSpace(cmd.CancelReason) == "" || !strings.HasPrefix(cmd.Cancels, "CA-") {
		return ErrCancel
	}
	if n, err := strconv.ParseInt(strings.TrimPrefix(cmd.Cancels, "CA-"), 10, 64); err != nil || n < 1 {
		return ErrCancel
	}
	return nil
}

// CommandFrom — сведения команды критического действия из контекста Execute.
func CommandFrom(ctx context.Context) (dom.Command, bool) {
	c, ok := ctx.Value(commandKey{}).(dom.Command)
	return c, ok
}

// CriticalHook — реализация journal.CriticalBuilder (AD-28): по каждой
// записи основной пачки с признаком критичности строит запись цепочки ca
// с настоящим номером и commit основной записи. Подключается к хранилищу
// журнала (storage/journal.WithCritical) — поэтому ни решение модуля, ни
// реакция движка, ни факт приёма не проходят мимо журнала CA.
type CriticalHook struct {
	Enc Encoder
}

var _ appjournal.CriticalBuilder = CriticalHook{}

// Critical — записи CA для пачки.
func (h CriticalHook) Critical(ctx context.Context, batch []appjournal.Sealed, next func() (int64, error)) ([]appjournal.Pending, error) {
	cmd, _ := CommandFrom(ctx)
	var out []appjournal.Pending
	for _, s := range batch {
		t := catalog.Type(s.Entry.EventType)
		if !dom.Critical(t) {
			continue
		}
		ev, _, err := ParseEnvelope(s.Envelope)
		if err != nil {
			return nil, fmt.Errorf("запись %s: %w", s.Entry.EventID, err)
		}
		m := dom.Main{EventID: s.Entry.EventID, EventType: t, Stream: s.Entry.Stream, Commit: s.Entry.Commit,
			Data: ev.Data, Signers: ev.Integrity.Signers}
		if ev.Command != nil && ev.Command.OnBehalfOf != "" {
			m.Signers = append([]string{ev.Command.OnBehalfOf}, m.Signers...)
		}
		if ev.Corrects != nil {
			m.Corrects = ev.Corrects.EventID
		}
		if s.Entry.CausationID != nil {
			m.Causation = *s.Entry.CausationID
		}
		// Основание — ссылки на записи в data решения (basis_event_ids и т. п.).
		c := cmd
		c.Basis = append(slices.Clone(c.Basis), basisOf(ev.Data)...)
		no, err := next()
		if err != nil {
			return nil, err
		}
		if c.Cancels != "" {
			if n, _ := strconv.ParseInt(strings.TrimPrefix(c.Cancels, "CA-"), 10, 64); n >= no {
				return nil, fmt.Errorf("%w: %s ещё нет", ErrCancel, c.Cancels)
			}
		}
		rec, err := dom.BuildCA(m, no, c)
		if err != nil {
			return nil, err
		}
		occurred, _ := dj.ParseTime(s.Entry.OccurredAt)
		part := s.Entry.Partition
		o := Out{EventID: dom.EventID(s.Entry.EventID), Type: catalog.SecurityCriticalActionRecorded, Stream: s.Entry.Stream,
			Partition: &part, OccurredAt: occurred, Correlation: s.Entry.CorrelationID, Causation: s.Entry.EventID, Data: rec}
		if s.Entry.RunID != nil {
			o.RunID = *s.Entry.RunID
		}
		p, err := h.Enc.Encode(ctx, o)
		if err != nil {
			return nil, err
		}
		p.Entry.Chain = jc.JournalEntryChainCa
		// recorded_at записи CA — как у основной (режим scenario, AD-37).
		p.Entry.RecordedAt = s.Entry.RecordedAt
		out = append(out, p)
	}
	return out, nil
}

// basisOf — идентификаторы оснований из data основной записи.
func basisOf(data json.RawMessage) []string {
	var d map[string]json.RawMessage
	if json.Unmarshal(data, &d) != nil {
		return nil
	}
	var out []string
	for _, k := range []string{"basis_event_ids", "released_event_ids", "basis", "signal_event_ids", "evidence_event_ids"} {
		var ids []string
		if json.Unmarshal(d[k], &ids) == nil {
			out = append(out, ids...)
		}
	}
	return out
}
