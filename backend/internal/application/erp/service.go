package erp

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/erp"
)

// Config — зависимости живой реализации ведущих портов (AD-36, режим live).
type Config struct {
	// Projections — проекции erp.* движка (роль projector).
	Projections engineapp.ProjectionStore
	// Outbox — состояние каналов и транспортные попытки роли outbox (nil — только журнал).
	Outbox OutboxStore
	// Decisions — запись решений людей (journal.Append, AD-44).
	Decisions DecisionWriter
	// Clock — доменное «сейчас» при приёме команды (AD-37); nil — системное.
	Clock appjournal.DomainClock
	// Channels — настроенные каналы (включённые системы — конфигурация, AD-18).
	Channels []LedgerInfo
}

// Service — реализация live ведущих портов модуля erp (AD-36): чтение — над
// проекциями erp.* и состоянием каналов; команды — гард над сообщением и
// решение человека в журнал. Без зависимостей (NewService) — заглушка 501.
type Service struct {
	Unimplemented
	cfg Config
}

// NewService создаёт заглушку live (все операции — 501); живую реализацию
// собирает NewLive.
func NewService() *Service { return &Service{} }

// NewLive создаёт живую реализацию.
func NewLive(cfg Config) *Service { return &Service{cfg: cfg} }

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

func (s *Service) live() bool { return s.cfg.Projections != nil }

func (s *Service) get(ctx context.Context, name, key string, v any) (bool, error) {
	raw, ok, err := s.cfg.Projections.Get(ctx, name, key)
	if err != nil || !ok {
		return false, err
	}
	return true, json.Unmarshal(raw, v)
}

func notFound(object, id string) error {
	e := platform.Fail(errcodes.ApiNotFound, "object", object, "id", id)
	e.Detail = object + " «" + id + "» не найден"
	return e
}

// Orders — задания учётных систем (erp.order.list).
func (s *Service) Orders(ctx context.Context, _ platform.Moment, p platform.Page) (ErpOrderList, error) {
	if !s.live() {
		return s.Unimplemented.Orders(ctx, platform.Moment{}, p)
	}
	var ids []string
	if _, err := s.get(ctx, ProjectionOrderIndex, "all", &ids); err != nil {
		return ErpOrderList{}, err
	}
	out := ErpOrderList{Items: []ErpOrder{}}
	for i := len(ids) - 1; i >= 0; i-- {
		var o Order
		ok, err := s.get(ctx, ProjectionOrder, ids[i], &o)
		if err != nil {
			return out, err
		}
		if !ok {
			continue
		}
		x := ErpOrder{OrderID: o.OrderID, ExternalSystem: o.ExternalSystem, ExternalNumber: o.ExternalNumber, ItemTypeID: o.ItemTypeID,
			Quantity: o.Quantity, Launched: len(o.Items), ReceivedAt: o.ReceivedAt}
		if o.ItemRevision != "" {
			x.ItemRevision = &o.ItemRevision
		}
		if o.DueDate != "" {
			x.DueDate = &o.DueDate
		}
		out.Items = append(out.Items, x)
	}
	out.Items, out.NextCursor = page(out.Items, p)
	return out, nil
}

// Messages — исходящие сообщения (erp.message.list): новые сверху, фильтр
// по изделию, состоянию, системе.
func (s *Service) Messages(ctx context.Context, f MessageFilter, m platform.Moment, p platform.Page) (ErpMessageList, error) {
	if !s.live() {
		return s.Unimplemented.Messages(ctx, f, m, p)
	}
	var x dom.Index
	if _, err := s.get(ctx, ProjectionMessageIndex, "all", &x); err != nil {
		return ErpMessageList{}, err
	}
	var keys []string
	for i := len(x.Entries) - 1; i >= 0; i-- {
		e := x.Entries[i]
		if f.ItemID != "" && e.ItemID != f.ItemID {
			continue
		}
		if f.Status != "" && string(e.Status) != f.Status {
			continue
		}
		keys = append(keys, e.Key)
	}
	keys, next := page(keys, p)
	out := ErpMessageList{Items: []ErpMessage{}, NextCursor: next}
	for _, k := range keys {
		msg, err := s.Message(ctx, k, m)
		if err != nil {
			if pe, ok := platform.AsError(err); ok && pe.Code == errcodes.ApiNotFound {
				continue
			}
			return out, err
		}
		if f.System != "" && msg.ExternalSystem != f.System {
			continue
		}
		out.Items = append(out.Items, msg)
	}
	return out, nil
}

// Message — сообщение по бизнес-ключу (erp.message.read).
func (s *Service) Message(ctx context.Context, key string, _ platform.Moment) (ErpMessage, error) {
	if !s.live() {
		return s.Unimplemented.Message(ctx, key, platform.Moment{})
	}
	v, err := s.view(ctx, key)
	if err != nil {
		return ErpMessage{}, err
	}
	msg := ErpMessage{BusinessKey: v.Key, ExternalSystem: v.System, Action: string(v.Action), MessageVersion: v.Version,
		Status: string(v.Status), AfterRework: v.AfterRework, RequestEventID: v.RequestEventID, RequestedAt: v.RequestedAt,
		Attempts: []ErpMessageAttempt{}, BasisSeq: v.BasisSeq}
	if v.ItemID != "" {
		msg.ItemID = &v.ItemID
	}
	if v.LotID != "" {
		msg.LotID = &v.LotID
	}
	if v.Accounting != "" {
		msg.AccountingState = &v.Accounting
	}
	for _, a := range v.Attempts {
		x := ErpMessageAttempt{At: a.At, Outcome: a.Outcome, ExternalDocumentRef: opt(a.Receipt), ErrorCode: opt(a.ErrorCode), ErrorMessage: opt(a.ErrorMessage)}
		if a.Receipt != "" && a.DocumentRef != "" {
			ref := a.Receipt + " / " + a.DocumentRef
			x.ExternalDocumentRef = &ref
		}
		if a.Outcome == "quarantined" {
			if a.ErrorCode != string(errcodes.ErpUnavailable) {
				continue // карантин виден состоянием; в попытки — только исчерпанные повторы
			}
			x.Outcome = "transport_error"
		}
		msg.Attempts = append(msg.Attempts, x)
	}
	if s.cfg.Outbox != nil && v.Status == dom.StatusQueued {
		// Транспортные попытки текущего поколения — операционное состояние роли outbox.
		if row, ok, err := s.cfg.Outbox.Row(ctx, key); err == nil && ok && row.RetryToken == TokenOf(v) && row.Tries > 0 && row.Handled != row.RetryToken {
			at := row.NextAt
			msg.Attempts = append(msg.Attempts, ErpMessageAttempt{At: at, Outcome: "transport_error",
				ErrorCode: opt(string(errcodes.ErpUnavailable)), ErrorMessage: opt(row.LastError + " — попыток " + strconv.Itoa(row.Tries) + ", следующая в " + at.UTC().Format(time.RFC3339))})
			msg.Status = "sent"
		}
	}
	return msg, nil
}

func (s *Service) view(ctx context.Context, key string) (dom.View, error) {
	var v dom.View
	ok, err := s.get(ctx, ProjectionMessage, key, &v)
	if err != nil {
		return v, err
	}
	if !ok || v.Key == "" {
		return v, notFound("Учётное сообщение", key)
	}
	return v, nil
}

// Channels — состояние каналов обмена (erp.channel.list, AD-18).
func (s *Service) Channels(ctx context.Context) (ErpChannelList, error) {
	if !s.live() {
		return s.Unimplemented.Channels(ctx)
	}
	out := ErpChannelList{Items: []ErpChannel{}}
	var x dom.Index
	if _, err := s.get(ctx, ProjectionMessageIndex, "all", &x); err != nil {
		return out, err
	}
	queued, quarantined := 0, 0
	for _, e := range x.Entries {
		switch e.Status {
		case dom.StatusQueued:
			queued++
		case dom.StatusQuarantined, dom.StatusRejected:
			quarantined++
		}
	}
	systems := map[string]ErpChannel{}
	var order []string
	for _, c := range s.cfg.Channels {
		systems[c.System] = ErpChannel{System: c.System, State: ChannelDisabled, Endpoint: c.Endpoint, Stand: c.Stand, ContractVersion: c.ContractVersion,
			Detail: opt("роль outbox не сверяла канал — обмен выключен или роль не запущена")}
		order = append(order, c.System)
	}
	if s.cfg.Outbox != nil {
		chs, err := s.cfg.Outbox.Channels(ctx)
		if err != nil {
			return out, err
		}
		for _, c := range chs {
			if _, ok := systems[c.System]; !ok {
				order = append(order, c.System)
			}
			systems[c.System] = ErpChannel{System: c.System, State: c.State, Endpoint: c.Endpoint, Stand: c.Stand,
				ContractVersion: c.ContractVersion, Detail: opt(c.Detail), LastExchangeAt: c.LastExchangeAt}
		}
	}
	for _, k := range order {
		c := systems[k]
		if k == "onec" || len(order) == 1 {
			c.Queued, c.Quarantined = queued, quarantined
		}
		out.Items = append(out.Items, c)
	}
	return out, nil
}

// now — доменное «сейчас» команды (AD-37).
func (s *Service) now(ctx context.Context) (time.Time, error) {
	if s.cfg.Clock == nil {
		return time.Now().UTC(), nil
	}
	return s.cfg.Clock.Now(ctx)
}

// ResendPosting — ручная переотправка из карантина (erp.posting.resend,
// FR-96): тот же бизнес-ключ и номер сообщения; гард — сообщение в карантине
// или отклонено, запрос — текущая версия.
func (s *Service) ResendPosting(ctx context.Context, key string, in ResendPosting) (platform.Receipt, error) {
	if !s.live() || s.cfg.Decisions == nil {
		return s.Unimplemented.ResendPosting(ctx, key, in)
	}
	v, err := s.view(ctx, key)
	if err != nil {
		return platform.Receipt{}, err
	}
	if v.Status != dom.StatusQuarantined && v.Status != dom.StatusRejected {
		return platform.Receipt{}, errNotQuarantined(key, v.Status)
	}
	if v.LastErrorCode == string(errcodes.ErpCorrectionPending) {
		e := platform.Fail(errcodes.ErpCorrectionPending, "business_key", key)
		e.Detail = "Содержимое сообщения " + key + " изменилось после подтверждения — нужно решение об исправлении (erp.posting.compensate), а не переотправка"
		return platform.Receipt{}, e
	}
	if in.RequestEventID != "" && in.RequestEventID != v.RequestEventID {
		e := platform.Fail(errcodes.JournalStaleState, "business_key", key)
		e.Detail = "Сообщение " + key + " уже в новой версии — обновите карточку"
		return platform.Receipt{}, e
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	data := ev.ErpPostingResendRequestedV1{BusinessKey: key, RequestEventID: ev.UUID(v.RequestEventID), Reason: reasonOf(in.Reason)}
	return s.cfg.Decisions.Write(ctx, Decision{Type: catalog.ErpPostingResendRequested, Key: key, Data: data, Meta: in.CommandMeta(),
		Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now, RunID: v.RunID})
}

// CompensatePosting — решение человека по новой версии подтверждённого
// сообщения (erp.posting.compensate, AD-7): отправить исправление (сторно +
// новое) или оставить как отправлено.
func (s *Service) CompensatePosting(ctx context.Context, key string, in CompensatePosting) (platform.Receipt, error) {
	if !s.live() || s.cfg.Decisions == nil {
		return s.Unimplemented.CompensatePosting(ctx, key, in)
	}
	v, err := s.view(ctx, key)
	if err != nil {
		return platform.Receipt{}, err
	}
	if v.AckedVersion == 0 || v.Version <= v.AckedVersion || v.Status == dom.StatusAcknowledged {
		return platform.Receipt{}, errNotQuarantined(key, v.Status)
	}
	if in.SupersededRequestEventID != "" && in.SupersededRequestEventID != v.AckedRequest {
		e := platform.Fail(errcodes.JournalStaleState, "business_key", key)
		e.Detail = "Подтверждённая версия сообщения " + key + " другая — обновите карточку"
		return platform.Receipt{}, e
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	data := ev.ErpPostingCompensationDecidedV1{BusinessKey: key, SupersededRequestEventID: ev.UUID(v.AckedRequest),
		Decision: ev.ErpPostingCompensationDecidedV1Decision(in.Decision), Reason: reasonOf(in.Reason)}
	nr := ev.UUID(v.RequestEventID)
	data.NewRequestEventID = &nr
	return s.cfg.Decisions.Write(ctx, Decision{Type: catalog.ErpPostingCompensationDecided, Key: key, Data: data, Meta: in.CommandMeta(),
		Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now, RunID: v.RunID})
}

func reasonOf(r ErpReason) ev.Reason {
	x := ev.Reason{Text: ev.Text(r.Text)}
	if r.Code != nil && *r.Code != "" {
		c := ev.Code(*r.Code)
		x.Code = &c
	}
	return x
}

func opt(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// page — страница по непрозрачному курсору (смещение) и пределу (1…500, по умолчанию 100).
func page[T any](xs []T, p platform.Page) ([]T, string) {
	off, _ := strconv.Atoi(p.Cursor)
	lim := p.Limit
	if lim <= 0 || lim > 500 {
		lim = 100
	}
	if off < 0 || off > len(xs) {
		off = len(xs)
	}
	end := min(off+lim, len(xs))
	next := ""
	if end < len(xs) {
		next = strconv.Itoa(end)
	}
	return slices.Clone(xs[off:end]), next
}

// AccountingOf — ось «учёт в 1С» изделия (AD-30, владелец erp) для паспорта
// изделия и столов (эпики 10, 18): проекция erp.item_accounting — значение
// оси, где изделие числится по подтверждённым сообщениям и бизнес-ключи его
// сообщений; сообщений ещё не было — not_sent.
func AccountingOf(ctx context.Context, p engineapp.ProjectionStore, itemID string) (dom.ItemAccounting, error) {
	a := dom.ItemAccounting{ItemID: itemID, State: string(ev.AxisErpAccountingNotSent), Keys: []string{}}
	raw, ok, err := p.Get(ctx, ProjectionItemAccounting, itemID)
	if err != nil || !ok {
		return a, err
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return a, err
	}
	return a, nil
}
