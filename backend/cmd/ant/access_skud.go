package main

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	accessapp "ant/internal/application/access"
	documentsapp "ant/internal/application/documents"
	appjournal "ant/internal/application/journal"
	referenceapp "ant/internal/application/reference"
	accessdom "ant/internal/domain/access"
	"ant/internal/infrastructure/fixtures/world"
	"ant/internal/infrastructure/integration/access/skud"
	skudstand "ant/internal/infrastructure/integration/access/skud/stand"
	"ant/internal/infrastructure/security/identity"
	accessstore "ant/internal/infrastructure/storage/access"
)

// Эпик 37 — СКУД, допуск к рабочему месту, клейма (FR-6, FR-80…FR-84,
// FR-145; AD-15 барьер 2). Сборка: проекция присутствия и запись фактов СКУД
// и отклонений — в пакете доступа (accessBundle); опрос журнала проходов СКУД —
// роль outbox (лидер outbox.skud); stand СКУД — роль stands (/stand/skud/);
// реестр клейм — порт documents (подпись этапа с видом контроля).

// skudEnabled — канал СКУД включён (integrations.enabled содержит skud).
func skudEnabled(env *environment) bool {
	return slices.Contains(env.cfg.Integrations.Enabled, "skud")
}

// skudBaseURL — адрес протокола skud.v1: из конфигурации или stand этого хоста.
func skudBaseURL(env *environment) string {
	if u := strings.TrimSpace(env.cfg.Access.SKUD.BaseURL); u != "" {
		return u
	}
	addr := env.cfg.Stands.Addr
	if addr == "" {
		addr = ":8491"
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	return "http://" + addr + "/stand/" + skudstand.Name + "/api/v1"
}

// withPresence — проекция присутствия, запись фактов СКУД и отклонений, график
// смен справочника — в пакете доступа (эпик 37).
func (b *accessBundle) withPresence(c *core) {
	b.presence = accessapp.NewPresenceProjection(accessstore.PolicyLog{Journal: c.journal, Codec: c.codec, Signal: c.listener, Types: accessdom.PresenceTypes})
	b.pw = accessstore.PresenceWriter{Journal: c.journal, Codec: c.codec}
	b.shifts = referenceShiftWindows{src: c.refSource}
}

// stampRegistry — реестр действующих клейм для documents (FR-145); nil — пакета доступа нет.
func (b *accessBundle) stampRegistry() documentsapp.StampRegistry {
	if b == nil || b.policy == nil {
		return nil
	}
	return accessapp.StampRegistry{Policy: b.policy}
}

// referenceShiftWindows — график смен справочника (эпик 19) для проверки
// «по графику должен быть, ключа нет»: смены, идущие в момент at.
type referenceShiftWindows struct{ src *referenceapp.JournalSource }

// ShiftsAt — смены справочника, идущие в момент at.
func (r referenceShiftWindows) ShiftsAt(ctx context.Context, at time.Time) ([]accessapp.ShiftWindow, error) {
	if r.src == nil {
		return nil, nil
	}
	b, err := r.src.Book(ctx, 0)
	if err != nil {
		return nil, err
	}
	var out []accessapp.ShiftWindow
	for _, s := range b.ShiftsBetween(at, at.Add(time.Nanosecond), "") {
		out = append(out, accessapp.ShiftWindow{ID: s.ShiftID, Start: s.From, End: s.To})
	}
	return out, nil
}

// switchedSKUD — журнал СКУД, который не опрашивается у выключенной
// администратором интеграции (AD-47, эпик 48).
type switchedSKUD struct {
	feed accessapp.SKUDFeed
	on   func(ctx context.Context) bool
}

func (s switchedSKUD) Events(ctx context.Context, after int64) ([]accessapp.SKUDEvent, int64, error) {
	if s.on != nil && !s.on(ctx) {
		return nil, 0, errors.New("интеграция skud выключена администратором")
	}
	return s.feed.Events(ctx, after)
}

// skudRun — опрос журнала проходов СКУД под лидером outbox.skud (одна копия).
func skudRun(ctx context.Context, env *environment, c *core) (func(context.Context) error, error) {
	b, err := accessLive(ctx, env)
	if err != nil {
		return nil, err
	}
	sc := env.cfg.Access.SKUD
	svc := accessapp.NewService(accessapp.WithDirectory(b.directory), accessapp.WithPolicy(b.policy),
		accessapp.WithDecisions(b.decisions, b.now), accessapp.WithLiveRoster(b.facts), accessapp.WithPresence(b.presence, b.pw, b.shifts))
	sw := integrationSwitch(env, c)
	client := skud.NewClient(skudBaseURL(env), sc.Timeout)
	p := &accessapp.SKUDPoller{Feed: switchedSKUD{feed: client, on: func(ctx context.Context) bool { return sw.Active(ctx, "skud") }},
		Intake: svc, Every: sc.Poll, CheckEvery: sc.Check, Log: env.moduleLog("access")}
	env.log.Info("outbox: СКУД", "endpoint", client.Endpoint())
	return func(ctx context.Context) error {
		return c.leader(env, "outbox.skud").Run(ctx, func(ctx context.Context, _ appjournal.Fence) error { return p.Run(ctx) })
	}, nil
}

// skudStandOf — stand СКУД: зоны цехов справочника мест, пропуска сотрудников
// стартовой политики; «домашняя» зона — зона цеха области роли сотрудника.
func skudStandOf() (*skudstand.Stand, error) {
	fsys := world.Inputs()
	zones, err := identity.LoadZones(fsys)
	if err != nil {
		return nil, err
	}
	_, seed, _, err := seedFS()
	if err != nil {
		return nil, err
	}
	opt := skudstand.Options{Arrive: true}
	for _, z := range zones {
		opt.Zones = append(opt.Zones, skudstand.Zone{ID: z.ID, Name: z.Name})
	}
	for _, x := range seed.Persons {
		h := skudstand.Holder{ID: x.ID, Name: x.Name}
		for _, a := range seed.Assignments {
			if a.PersonID != x.ID || h.Home != "" {
				continue
			}
			for _, z := range zones {
				if accessdom.ScopeCovers(z.Scope, a.Scope) {
					h.Home = z.ID
				}
			}
		}
		opt.Holders = append(opt.Holders, h)
	}
	return skudstand.New(opt), nil
}
