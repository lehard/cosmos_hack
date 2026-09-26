package main

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	accessapp "ant/internal/application/access"
	documentsapp "ant/internal/application/documents"
	appjournal "ant/internal/application/journal"
	opsapp "ant/internal/application/ops"
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

// ShiftsAt — смены справочника, идущие в момент at; смен в справочнике нет
// (мастер их ещё не завёл) — шаблоны смен нормативного слоя
// (normative/reference/flange/shifts.yaml).
func (r referenceShiftWindows) ShiftsAt(ctx context.Context, at time.Time) ([]accessapp.ShiftWindow, error) {
	var out []accessapp.ShiftWindow
	if r.src != nil {
		b, err := r.src.Book(ctx, 0)
		if err != nil {
			return nil, err
		}
		for _, s := range b.ShiftsBetween(at, at.Add(time.Nanosecond), "") {
			out = append(out, accessapp.ShiftWindow{ID: s.ShiftID, Start: s.From, End: s.To})
		}
	}
	if len(out) > 0 {
		return out, nil
	}
	return patternShiftsAt(at), nil
}

// patternShiftsAt — смены шаблонов нормативного слоя, идущие в момент at
// (время шаблона — в часовом поясе файла; смена через полночь — до утра).
func patternShiftsAt(at time.Time) []accessapp.ShiftWindow {
	b, err := fs.ReadFile(world.Inputs(), "normative/reference/flange/shifts.yaml")
	if err != nil {
		return nil
	}
	var f struct {
		TimeZone string `yaml:"time_zone"`
		Patterns []struct {
			ID     string `yaml:"id"`
			Starts string `yaml:"starts"`
			Ends   string `yaml:"ends"`
		} `yaml:"patterns"`
	}
	if yaml.Unmarshal(b, &f) != nil {
		return nil
	}
	loc, err := time.LoadLocation(f.TimeZone)
	if err != nil {
		loc = time.UTC
	}
	clock := func(day time.Time, hm string) (time.Time, bool) {
		t, err := time.Parse("15:04", hm)
		if err != nil {
			return time.Time{}, false
		}
		return time.Date(day.Year(), day.Month(), day.Day(), t.Hour(), t.Minute(), 0, 0, loc), true
	}
	var out []accessapp.ShiftWindow
	local := at.In(loc)
	for _, p := range f.Patterns {
		for _, day := range []time.Time{local.AddDate(0, 0, -1), local} {
			from, ok1 := clock(day, p.Starts)
			to, ok2 := clock(day, p.Ends)
			if !ok1 || !ok2 {
				continue
			}
			if !to.After(from) {
				to = to.AddDate(0, 0, 1)
			}
			if !at.Before(from) && at.Before(to) {
				out = append(out, accessapp.ShiftWindow{ID: p.ID, Start: from, End: to})
			}
		}
	}
	return out
}

// skudProbe — «проверить соединение» со СКУД (эпик 48, AD-47): сверка
// ответной стороны по about — версия протокола skud.v1.
func skudProbe(ctx context.Context, env *environment) opsapp.ProbeResult {
	c := skud.NewClient(skudBaseURL(env), env.cfg.Access.SKUD.Timeout)
	a, err := c.About(ctx)
	switch {
	case err == nil:
		return opsapp.ProbeResult{Result: opsapp.ProbeOK, Endpoint: c.Endpoint(), Detail: a.System + " отвечает, протокол " + a.Contract}
	case a.Contract != "":
		return opsapp.ProbeResult{Result: opsapp.ProbeDegraded, Endpoint: c.Endpoint(), Detail: err.Error()}
	}
	return opsapp.ProbeResult{Result: opsapp.ProbeUnreachable, Endpoint: c.Endpoint(), Detail: err.Error()}
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
