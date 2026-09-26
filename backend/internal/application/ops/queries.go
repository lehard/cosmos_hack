package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/ops"
)

// Health — состояние системы (ops.health.read, FR-127): сервисы и роли по
// арендам, очереди и отставание курсоров, интеграции, карантин,
// остановленные изделия, последний отчёт верификатора «по данным сервера».
// Недоступный источник не роняет ответ: его часть — «unknown» с пояснением.
func (s *Service) Health(ctx context.Context) (OpsHealth, error) {
	if !s.live {
		return Unimplemented{}.Health(ctx)
	}
	now := s.cfg.Now().UTC()
	h := OpsHealth{Components: []ComponentState{}, Queues: []QueueState{}, Integrations: []IntegrationState{},
		Profile: s.profile(), Mode: s.mode(), Version: s.cfg.Version, SelfCheck: s.selfCheck()}
	h.Components = append(h.Components, ComponentState{Component: "ant/api", State: dom.StateOK, Instances: 1,
		Detail: ptr("эта копия, версия " + s.cfg.Version), CheckedAt: &now})
	h.Components = append(h.Components, s.postgres(ctx, now))

	var backlogs []Backlog
	if rt := s.cfg.Runtime; rt != nil {
		if leases, err := rt.Leases(ctx); err == nil {
			for _, c := range dom.Components(leases, now, s.cfg.Partitions) {
				h.Components = append(h.Components, component(c, now))
			}
		} else {
			h.Components = append(h.Components, ComponentState{Component: "ant/roles", State: dom.StateUnknown, Detail: ptr("аренды не прочитаны: " + err.Error()), CheckedAt: &now})
		}
		if b, err := rt.Backlogs(ctx, s.cfg.Partitions); err == nil {
			backlogs = b
		} else {
			s.cfg.Log.Warn("ops: курсоры не прочитаны", "err", err)
		}
	}
	verifier, report := s.verifier(ctx, now)
	h.Components = append(h.Components, verifier)
	h.Verifier = report

	lag := map[string]int64{}
	for _, b := range backlogs {
		q := QueueState{Name: b.Consumer, Scope: "global", LagSeq: max(b.Head-b.Seq, 0), Pending: b.Pending}
		if b.Partition >= 0 {
			q.Name, q.Scope = fmt.Sprintf("%s/%d", b.Consumer, b.Partition), "partition"
			if b.Pending == 0 {
				q.LagSeq = 0
			}
		} else {
			lag[b.Consumer] = q.LagSeq
		}
		h.Queues = append(h.Queues, q)
	}
	var channels []dom.Channel
	for _, x := range s.cfg.Exchanges {
		qs, chs, err := x.Exchange(ctx)
		if err != nil {
			s.cfg.Log.Warn("ops: очереди исходящих не прочитаны", "err", err)
			continue
		}
		channels = append(channels, chs...)
		for _, q := range qs {
			quar := q.Quarantined
			h.Queues = append(h.Queues, QueueState{Name: "outbox:" + q.System, Scope: "outbox", LagSeq: lag[q.Consumer], Pending: q.Queued, Quarantined: &quar})
		}
	}
	recs, err := integrationRecords(ctx, s.cfg.Journal, s.cfg.Codec)
	if err != nil {
		return OpsHealth{}, err
	}
	for _, it := range dom.Integrations(s.cfg.Enabled, channels, recs) {
		v := IntegrationState{System: it.System, State: it.State, Since: it.Since}
		if it.Detail != "" {
			v.Detail = ptr(it.Detail)
		}
		h.Integrations = append(h.Integrations, v)
	}
	if q := s.cfg.Quarantine; q != nil {
		if n, err := q.QuarantineOpen(ctx); err == nil {
			h.QuarantineOpen = int(n)
		} else {
			s.cfg.Log.Warn("ops: карантин не прочитан", "err", err)
		}
	}
	failures, retries, err := failuresAndRetries(ctx, s.cfg.Journal, s.cfg.Codec)
	if err != nil {
		return OpsHealth{}, err
	}
	h.StoppedItems = len(dom.StoppedItems(failures, retries))
	return h, nil
}

func (s *Service) postgres(ctx context.Context, now time.Time) ComponentState {
	c := ComponentState{Component: "postgres", State: dom.StateUnknown, CheckedAt: &now}
	if s.cfg.Database == nil {
		return c
	}
	pctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := s.cfg.Database.Ping(pctx); err != nil {
		c.State, c.Detail = dom.StateDown, ptr(err.Error())
		return c
	}
	c.State, c.Instances = dom.StateOK, 1
	return c
}

// verifier — последний отчёт верификатора, который ant забрал у хранителя
// (security.integrity.checked, AD-46): свежесть — два интервала проверок.
func (s *Service) verifier(ctx context.Context, now time.Time) (ComponentState, *VerifierReportRef) {
	c := ComponentState{Component: "verifier", State: dom.StateUnknown, Detail: ptr("отчётов верификатора ещё не было"), CheckedAt: &now}
	d, err := last(ctx, s.cfg.Journal, s.cfg.Codec, catalog.SecurityIntegrityChecked, "")
	if err != nil {
		c.Detail = ptr("отчёт не прочитан: " + err.Error())
		return c, nil
	}
	if d == nil {
		return c, nil
	}
	var x ev.SecurityIntegrityCheckedV1
	if err := json.Unmarshal(d.Record.Data, &x); err != nil {
		c.Detail = ptr("отчёт не разобран: " + err.Error())
		return c, nil
	}
	at := d.Record.RecordedAt
	ref := &VerifierReportRef{Verdict: verdict(string(x.Verdict)), CheckedAt: at, ReportRef: string(x.ReportDigest)}
	c.Instances = 1
	if age := now.Sub(at); age > 2*s.cfg.VerifierInterval {
		c.State, c.Detail = dom.StateDegraded, ptr(fmt.Sprintf("свежего отчёта нет %s (дольше двух интервалов %s)", age.Round(time.Second), s.cfg.VerifierInterval))
	} else {
		c.State, c.Detail = dom.StateOK, ptr("проверено до seq "+strconv.Itoa(int(x.CheckedUpToSeq))+", вердикт «"+ref.Verdict+"»")
	}
	return c, ref
}

// verdict — вердикт отчёта в словаре экрана.
func verdict(v string) string {
	switch v {
	case "intact", "violated":
		return v
	case "intact_with_reservations":
		return "intact_with_caveats"
	}
	return "unknown"
}

func component(c dom.Component, now time.Time) ComponentState {
	v := ComponentState{Component: c.Name, State: c.State, Instances: c.Instances, CheckedAt: &now}
	if c.Leader != "" {
		v.Leader = ptr(c.Leader)
	}
	if c.Detail != "" {
		v.Detail = ptr(c.Detail)
	}
	return v
}

// StoppedItems — изделия «обработка остановлена» (ops.stopped_item.list, AD-45).
func (s *Service) StoppedItems(ctx context.Context, p platform.Page) (StoppedItemList, error) {
	if !s.live {
		return Unimplemented{}.StoppedItems(ctx, p)
	}
	failures, retries, err := failuresAndRetries(ctx, s.cfg.Journal, s.cfg.Codec)
	if err != nil {
		return StoppedItemList{}, err
	}
	all := dom.StoppedItems(failures, retries)
	off, _ := strconv.Atoi(p.Cursor)
	lim := p.Limit
	if lim <= 0 || lim > 500 {
		lim = 100
	}
	out := StoppedItemList{Items: []StoppedItem{}}
	for i := max(off, 0); i < len(all); i++ {
		if len(out.Items) == lim {
			out.NextCursor = strconv.Itoa(i)
			break
		}
		f := all[i].Failure
		out.Items = append(out.Items, StoppedItem{ItemID: f.ItemID, FailureEventID: f.EventID, Consumer: f.Consumer,
			FailedSeq: f.FailedSeq, Error: f.Error, FailedAt: f.At, Retries: all[i].Retries})
	}
	return out, nil
}

// portInfo — зона и описанная замена ведомого порта (AD-35).
var portInfo = map[platform.PortKey][2]string{
	platform.PortJournalStore:     {"storage", "BFT-реестр (A11)"},
	platform.PortLeaseStore:       {"storage", "k8s Lease"},
	platform.PortMaterialStore:    {"storage", "S3-совместимое хранилище в контуре"},
	platform.PortWorkFeed:         {"transport", "Kafka, key = item_id"},
	platform.PortPublisher:        {"transport", "брокер сообщений"},
	platform.PortTelemetry:        {"observability", "OTLP (OpenTelemetry)"},
	platform.PortSigner:           {"security", "PKCS#11; сертифицированное СКЗИ"},
	platform.PortVerifier:         {"security", "сертифицированное СКЗИ"},
	platform.PortCipher:           {"security", "сертифицированное СКЗИ"},
	platform.PortAccessControl:    {"security", ""},
	platform.PortIdentityProvider: {"security", "LDAP / ALD Pro / FreeIPA"},
	platform.PortDomainClock:      {"storage", "доменное «сейчас» из журнала (режим сценария)"},
	platform.PortInfraClock:       {"storage", ""},
	platform.PortFixtureCursor:    {"storage", ""},
	platform.PortConsumer:         {"storage", "Kafka consumer group"},
}

// Settings — адаптеры ведомых портов, режимы модулей, включённые интеграции
// и решения об источниках (ops.setting.list, FR-127, AD-35, AD-36).
func (s *Service) Settings(ctx context.Context) (SettingList, error) {
	if !s.live {
		return Unimplemented{}.Settings(ctx)
	}
	out := SettingList{Ports: []PortSetting{}, Modules: []ModuleMode{}, Enabled: slices.Clone(s.cfg.Enabled)}
	if out.Enabled == nil {
		out.Enabled = []string{}
	}
	for _, k := range platform.PortKeys {
		info := portInfo[k]
		a := s.cfg.Adapters[string(k)]
		if a == "" {
			a = "по умолчанию"
		}
		zone := info[0]
		if zone == "" {
			zone = "application"
		}
		out.Ports = append(out.Ports, PortSetting{Port: string(k), Adapter: a, Zone: zone, Replace: info[1]})
	}
	s.mu.RLock()
	modules := s.cfg.Modules
	s.mu.RUnlock()
	if modules != nil {
		out.Modules = append(out.Modules, modules()...)
	}
	src, err := s.sources(ctx)
	if err != nil {
		return SettingList{}, err
	}
	out.Sources = src
	return out, nil
}

// sources — последнее решение по каждому источнику (ops.source.disabled / enabled).
func (s *Service) sources(ctx context.Context) ([]SourceSwitch, error) {
	last := map[string]SourceSwitch{}
	for _, t := range []catalog.Type{catalog.OpsSourceDisabled, catalog.OpsSourceEnabled} {
		ds, err := records(ctx, s.cfg.Journal, s.cfg.Codec, t, "")
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

func (s *Service) profile() string {
	switch s.cfg.Profile {
	case "fixtures", "demo", "load", "prod":
		return s.cfg.Profile
	}
	return "demo"
}

func (s *Service) mode() platform.Mode {
	if s.cfg.Mode == "" {
		return platform.ModeLive
	}
	return s.cfg.Mode
}

func ptr[T any](v T) *T { return &v }
