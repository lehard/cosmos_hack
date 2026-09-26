package ops

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/ops"
)

// Управление интеграциями (FR-157, AD-47; эпик 48): экран «Интеграции»
// стола администратора, решения ops.integration.state_set, проверка
// соединения ops.integration.checked и порт IntegrationSwitch, через который
// роли outbox и приём читают состояние без перезапуска.

// Probe — «проверить соединение» у клиентского адаптера (AD-18, AD-47):
// та же сверка ответной стороны, что при старте. Собирает cmd/ant по
// установленным адаптерам.
type Probe interface {
	Probe(ctx context.Context, system string) (ProbeResult, error)
}

// ProbeResult — итог сверки ответной стороны.
type ProbeResult struct {
	// Result — ok | degraded | unreachable | not_supported.
	Result   string
	Endpoint string
	Detail   string
}

// Итоги проверки соединения (ops.integration.checked.result).
const (
	ProbeOK           = "ok"
	ProbeDegraded     = "degraded"
	ProbeUnreachable  = "unreachable"
	ProbeNotSupported = "not_supported"
)

// SwitchTTL — сколько держится прочитанное состояние интеграций у
// IntegrationSwitch: процессы подхватывают решение администратора за это
// время, без перезапуска (AD-47).
const SwitchTTL = time.Second

// IntegrationSwitch — порт состояния интеграций для ролей outbox и приёма
// (AD-47): действующее состояние каждой системы и решения об источниках
// читаются из журнала (индекс по event_type), кэш — SwitchTTL.
type IntegrationSwitch struct {
	Journal   appjournal.JournalStore
	Codec     *engineapp.Codec
	Profile   string
	Installed []dom.Installed
	// Now — InfraClock для кэша; nil — time.Now.
	Now func() time.Time

	mu      sync.Mutex
	at      time.Time
	states  []dom.SwitchState
	sources map[string]string
}

func (w *IntegrationSwitch) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

// snapshot — состояния систем и выключенные источники (не старше SwitchTTL).
func (w *IntegrationSwitch) snapshot(ctx context.Context) ([]dom.SwitchState, map[string]string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.states != nil && w.now().Sub(w.at) < SwitchTTL {
		return w.states, w.sources, nil
	}
	recs, err := stateRecords(ctx, w.Journal, w.Codec)
	if err != nil {
		return nil, nil, err
	}
	srcs, err := sourceDecisions(ctx, w.Journal, w.Codec)
	if err != nil {
		return nil, nil, err
	}
	off := map[string]string{}
	for _, s := range srcs {
		if !s.Enabled {
			off[s.SourceID] = s.Reason
		}
	}
	w.states, w.sources, w.at = dom.States(w.Profile, w.Installed, recs), off, w.now()
	return w.states, w.sources, nil
}

// Invalidate — сбросить кэш (после решения на этой копии).
func (w *IntegrationSwitch) Invalidate() {
	w.mu.Lock()
	w.states = nil
	w.mu.Unlock()
}

// State — действующее состояние системы: enabled | disabled | stand.
// Ошибка чтения журнала — прежнее состояние, если оно было; иначе ошибка.
func (w *IntegrationSwitch) State(ctx context.Context, system string) (string, error) {
	states, _, err := w.snapshot(ctx)
	if err != nil {
		return "", err
	}
	for _, s := range states {
		if s.System == system {
			return s.State, nil
		}
	}
	return dom.SwitchDisabled, nil
}

// Active — система включена (стенд или реальная). Ошибка чтения — считаем
// включённой: сбой журнала не должен останавливать обмен молча.
func (w *IntegrationSwitch) Active(ctx context.Context, system string) bool {
	st, err := w.State(ctx, system)
	return err != nil || st != dom.SwitchDisabled
}

// SourceBlocked — отвергать ли приёму входящие от source_id (AD-28, AD-47):
// источник отключён решением ops.source.disabled или его система выключена.
func (w *IntegrationSwitch) SourceBlocked(ctx context.Context, sourceID string) (bool, string, error) {
	states, off, err := w.snapshot(ctx)
	if err != nil {
		return false, "", err
	}
	b := dom.SourceBlocked(sourceID, off, states)
	return b.Blocked, b.Reason, nil
}

// stateRecords — решения ops.integration.state_set из журнала.
func stateRecords(ctx context.Context, j appjournal.JournalStore, c *engineapp.Codec) ([]dom.StateRecord, error) {
	ds, err := records(ctx, j, c, catalog.OpsIntegrationStateSet, "")
	if err != nil {
		return nil, err
	}
	out := make([]dom.StateRecord, 0, len(ds))
	for _, d := range ds {
		var x ev.OpsIntegrationStateSetV1
		if err := json.Unmarshal(d.Record.Data, &x); err != nil {
			return nil, err
		}
		r := dom.StateRecord{Seq: d.Record.Seq, EventID: d.Record.EventID, System: string(x.System), State: string(x.State),
			Reason: string(x.Reason.Text), Actor: strings.TrimSuffix(d.Record.Actor, "@1"), At: d.Record.RecordedAt}
		if x.Previous != nil {
			r.Previous = string(*x.Previous)
		}
		out = append(out, r)
	}
	return out, nil
}

// sourceDecisions — последнее решение по каждому источнику (ops.source.disabled / enabled).
func sourceDecisions(ctx context.Context, j appjournal.JournalStore, c *engineapp.Codec) ([]SourceSwitch, error) {
	last := map[string]SourceSwitch{}
	for _, t := range []catalog.Type{catalog.OpsSourceDisabled, catalog.OpsSourceEnabled} {
		ds, err := records(ctx, j, c, t, "")
		if err != nil {
			return nil, err
		}
		for _, d := range ds {
			var x ev.OpsSourceDisabledV1
			if err := json.Unmarshal(d.Record.Data, &x); err != nil {
				return nil, err
			}
			id := string(x.SourceID)
			if p, ok := last[id]; ok && p.Seq > d.Record.Seq {
				continue
			}
			last[id] = SourceSwitch{SourceID: id, Enabled: t == catalog.OpsSourceEnabled, Reason: string(x.Reason.Text),
				Actor: strings.TrimSuffix(d.Record.Actor, "@1"), At: d.Record.RecordedAt, Seq: d.Record.Seq}
		}
	}
	out := make([]SourceSwitch, 0, len(last))
	for _, v := range last {
		out = append(out, v)
	}
	slices.SortFunc(out, func(a, b SourceSwitch) int { return strings.Compare(a.SourceID, b.SourceID) })
	return out, nil
}

// integrationStream — поток решений об интеграции (вид source, AD-39).
func integrationStream(system string) string { return "source:integration:" + system }

// switchStates — действующие состояния (live: из журнала без кэша).
func (s *Service) switchStates(ctx context.Context) ([]dom.SwitchState, error) {
	recs, err := stateRecords(ctx, s.cfg.Journal, s.cfg.Codec)
	if err != nil {
		return nil, err
	}
	return dom.States(s.profile(), s.cfg.Installed, recs), nil
}

// Integrations — экран «Интеграции» (ops.integration.list, FR-157):
// установленные и неустановленные системы, состояние включения, живой канал,
// последний обмен, ошибки, очередь и карантин исходящих, последняя проверка.
func (s *Service) Integrations(ctx context.Context) (IntegrationList, error) {
	if !s.live {
		return Unimplemented{}.Integrations(ctx)
	}
	states, err := s.switchStates(ctx)
	if err != nil {
		return IntegrationList{}, err
	}
	var channels []dom.Channel
	queues := map[string]ExchangeQueue{}
	for _, x := range s.cfg.Exchanges {
		qs, chs, err := x.Exchange(ctx)
		if err != nil {
			s.cfg.Log.Warn("ops: очереди исходящих не прочитаны", "err", err)
			continue
		}
		channels = append(channels, chs...)
		for _, q := range qs {
			queues[q.System] = q
		}
	}
	degraded, err := integrationRecords(ctx, s.cfg.Journal, s.cfg.Codec)
	if err != nil {
		return IntegrationList{}, err
	}
	checks, err := checkRecords(ctx, s.cfg.Journal, s.cfg.Codec)
	if err != nil {
		return IntegrationList{}, err
	}
	out := IntegrationList{Profile: s.profile(), Items: make([]IntegrationEntry, 0, len(states))}
	for _, st := range states {
		e := IntegrationEntry{System: st.System, Installed: st.Installed, State: st.State, Default: st.Default,
			StandAvailable: st.Installed && st.Stand && s.profile() != dom.ProfileProd, RealAvailable: st.Installed && st.Real}
		if st.Last != nil {
			e.BasisSeq = st.Last.Seq
			d := IntegrationDecision{State: st.Last.State, Reason: st.Last.Reason, Actor: st.Last.Actor, At: st.Last.At, Seq: st.Last.Seq}
			if st.Last.Previous != "" {
				d.Previous = ptr(st.Last.Previous)
			}
			e.LastDecision = &d
		}
		for _, c := range channels {
			if c.System != st.System {
				continue
			}
			e.Channel = ptr(c.State)
			if c.Detail != "" {
				e.Detail = ptr(c.Detail)
			}
			if c.Endpoint != "" {
				e.Endpoint = ptr(c.Endpoint)
			}
			if !c.CheckedAt.IsZero() {
				e.CheckedAt = ptr(c.CheckedAt)
			}
			e.LastExchangeAt = c.LastExchangeAt
		}
		if q, ok := queues[st.System]; ok {
			e.Queued, e.Quarantined = ptr(q.Queued), ptr(q.Quarantined)
		}
		for _, r := range degraded {
			if r.System == st.System && r.State == dom.IntegrationDegraded {
				at := r.At
				e.LastError = &IntegrationError{At: at, Detail: r.Detail}
			}
		}
		if c, ok := checks[st.System]; ok {
			e.LastCheck = &c
		}
		out.Items = append(out.Items, e)
	}
	return out, nil
}

// checkRecords — последняя проверка соединения по системе.
func checkRecords(ctx context.Context, j appjournal.JournalStore, c *engineapp.Codec) (map[string]IntegrationCheck, error) {
	ds, err := records(ctx, j, c, catalog.OpsIntegrationChecked, "")
	if err != nil {
		return nil, err
	}
	out := map[string]IntegrationCheck{}
	for _, d := range ds {
		var x ev.OpsIntegrationCheckedV1
		if err := json.Unmarshal(d.Record.Data, &x); err != nil {
			return nil, err
		}
		v := IntegrationCheck{Result: string(x.Result), At: d.Record.RecordedAt, Seq: d.Record.Seq}
		if x.Detail != nil {
			v.Detail = ptr(*x.Detail)
		}
		if x.Endpoint != nil {
			v.Endpoint = ptr(*x.Endpoint)
		}
		out[string(x.System)] = v
	}
	return out, nil
}

// SetIntegration — включить, выключить или переключить «стенд ↔ реальная
// система» (ops.integration.set, FR-157, AD-47): критическое действие
// администратора, единолично (Д-71) — запись ops.integration.state_set;
// гард: только установленная система, стенд в prod — отказ с кодом
// ops.stand_forbidden.
func (s *Service) SetIntegration(ctx context.Context, system string, in SetIntegrationState) (platform.Receipt, error) {
	if !s.live {
		return Unimplemented{}.SetIntegration(ctx, system, in)
	}
	meta := in.CommandMeta()
	if rc, ok, err := s.replay(ctx, meta.CommandID); ok || err != nil {
		return rc, err
	}
	states, err := s.switchStates(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	var cur *dom.SwitchState
	for i := range states {
		if states[i].System == system {
			cur = &states[i]
		}
	}
	if cur == nil {
		e := platform.Fail(errcodes.ApiNotFound, "object", "Интеграция", "id", system)
		e.Detail = "Система «" + system + "» не известна экрану «Интеграции»"
		return platform.Receipt{}, e
	}
	if rej := dom.CheckSet(s.profile(), *cur, in.State); rej != nil {
		return platform.Receipt{}, rejection(rej)
	}
	data := ev.OpsIntegrationStateSetV1{System: ev.OpsIntegrationStateSetV1System(system), State: ev.OpsIntegrationStateSetV1State(in.State),
		Reason: ev.Reason{Text: ev.Text(strings.TrimSpace(in.Reason.Text))}}
	prev := ev.OpsIntegrationStateSetV1Previous(cur.State)
	data.Previous = &prev
	if in.Reason.Code != nil && *in.Reason.Code != "" {
		c := ev.Code(*in.Reason.Code)
		data.Reason.Code = &c
	}
	rc, err := s.decide(ctx, decision{Type: catalog.OpsIntegrationStateSet, Stream: integrationStream(system), Data: data, Meta: meta})
	if err == nil && s.cfg.Switch != nil {
		s.cfg.Switch.Invalidate()
	}
	return rc, err
}

// rejection — отказ доменного гарда в ошибку API (errors.yaml).
func rejection(r *dom.Rejection) error {
	kv := make([]string, 0, len(r.Params)*2)
	for _, k := range []string{"system", "state", "profile"} {
		if v, ok := r.Params[k]; ok {
			kv = append(kv, k, v)
		}
	}
	e := platform.Fail(errcodes.Code(r.Code), kv...)
	switch r.Code {
	case dom.CodeStandForbidden:
		e.Detail = "Профиль " + r.Params["profile"] + ": переключить " + r.Params["system"] + " на стенд нельзя — только реальная система или «выключена»"
	case dom.CodeNotInstalled:
		e.Detail = "Система " + r.Params["system"] + " не установлена конфигурацией (integrations.enabled) — включить её с экрана нельзя"
	case dom.CodeModeUnavailable:
		e.Detail = "Для " + r.Params["system"] + " не задан адрес режима «" + r.Params["state"] + "» в конфигурации — переключение невозможно"
	case dom.CodeUnchanged:
		e.Detail = r.Params["system"] + " уже в состоянии «" + r.Params["state"] + "» — решение не записано"
	}
	return e
}

// CheckIntegration — «проверить соединение» (ops.integration.check, AD-47):
// сверка ответной стороны порта Probe; итог — служебная запись
// ops.integration.checked (видна в списке как последняя проверка).
func (s *Service) CheckIntegration(ctx context.Context, system string, in CheckIntegration) (platform.Receipt, error) {
	if !s.live {
		return Unimplemented{}.CheckIntegration(ctx, system, in)
	}
	states, err := s.switchStates(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	var cur *dom.SwitchState
	for i := range states {
		if states[i].System == system {
			cur = &states[i]
		}
	}
	if cur == nil || !cur.Installed {
		return platform.Receipt{}, rejection(&dom.Rejection{Code: dom.CodeNotInstalled, Params: map[string]string{"system": system}})
	}
	res := ProbeResult{Result: ProbeNotSupported, Detail: "у адаптера нет проверки ответной стороны"}
	if s.cfg.Probe != nil {
		pctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		r, err := s.cfg.Probe.Probe(pctx, system)
		cancel()
		switch {
		case err != nil:
			res = ProbeResult{Result: ProbeUnreachable, Detail: err.Error()}
		default:
			res = r
		}
	}
	data := ev.OpsIntegrationCheckedV1{System: ev.OpsIntegrationCheckedV1System(system), Result: ev.OpsIntegrationCheckedV1Result(res.Result)}
	mode := ev.OpsIntegrationCheckedV1Mode(cur.State)
	data.Mode = &mode
	if res.Detail != "" {
		d := dom.FailureMessage(res.Detail)
		data.Detail = &d
	}
	if res.Endpoint != "" {
		ep := res.Endpoint
		if len(ep) > 512 {
			ep = ep[:512]
		}
		data.Endpoint = &ep
	}
	// Id служебной записи — UUIDv5 от команды (AD-7): повтор кнопки с тем же
	// command_id не пишет вторую проверку.
	key := strings.ToLower(in.CommandID)
	if key == "" {
		key = s.cfg.Now().UTC().Format(time.RFC3339Nano)
	}
	id := kernel.UUIDv5(constants.NsAnt, string(catalog.OpsIntegrationChecked)+"\x1f"+system+"\x1f"+key)
	pend, err := s.cfg.Codec.Encode(ctx, engineapp.Out{EventID: id, Type: catalog.OpsIntegrationChecked, Kind: catalog.KindService,
		Stream: integrationStream(system), OccurredAt: s.domainNow(ctx), RunID: appjournal.RunFrom(ctx), Data: data})
	if err != nil {
		return platform.Receipt{}, err
	}
	r, err := s.cfg.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{pend}})
	if errors.Is(err, appjournal.ErrDuplicate) {
		return platform.Receipt{CommandID: id, EventIDs: []string{id}, Replayed: true}, nil
	}
	if err != nil {
		return platform.Receipt{}, err
	}
	s.cfg.Log.Info("ops: соединение проверено", "system", system, "result", res.Result, "event_id", id)
	rc := platform.Receipt{CommandID: id, EventIDs: []string{id}, RecordedAt: r.Committed}
	if len(r.Seqs) > 0 {
		rc.Seq = r.Seqs[0]
	}
	return rc, nil
}
