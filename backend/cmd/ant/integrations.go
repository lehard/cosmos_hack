package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ant/cmd/internal/config"
	engineapp "ant/internal/application/engine"
	erpapp "ant/internal/application/erp"
	mesapp "ant/internal/application/mes"
	opsapp "ant/internal/application/ops"
	dom "ant/internal/domain/ops"
)

// Эпик 48 — управление интеграциями (FR-157, AD-47). Установку задаёт
// конфигурация: integrations.enabled и адреса систем (erp.onec,
// erp.galaktika, mes.b2mml); состояние включения — решения администратора
// ops.integration.state_set в журнале. Порт IntegrationSwitch — один на
// процесс: роль outbox не отправляет исходящие выключенной системе (они
// копятся в очереди и уходят после включения) и не опрашивает её входящие;
// приём отвергает входящие от выключенного источника или системы.

// switches — порт состояния интеграций на процесс (одно ядро — один кэш).
var switches = struct {
	sync.Mutex
	m map[*environment]*opsapp.IntegrationSwitch
}{m: map[*environment]*opsapp.IntegrationSwitch{}}

// integrationSwitch — порт состояния интеграций процесса на ядре c.
func integrationSwitch(env *environment, c *core) *opsapp.IntegrationSwitch {
	switches.Lock()
	defer switches.Unlock()
	if w, ok := switches.m[env]; ok {
		return w
	}
	w := &opsapp.IntegrationSwitch{Journal: c.journal, Codec: c.codec, Profile: env.cfg.Profile, Installed: installedIntegrations(env.cfg), Now: c.codec.Now}
	switches.m[env] = w
	return w
}

// installedIntegrations — установленные конфигурацией интеграции и режимы,
// заданные адресами (AD-47): у 1С, Галактики и MES адрес один — стенд или
// реальная система по флагу stand (пустой адрес 1С — стенд роли stands); у
// прочих систем раздельных адресов нет — доступны оба режима (в prod стенд
// всё равно запрещён гардом).
func installedIntegrations(cfg *config.Config) []dom.Installed {
	var out []dom.Installed
	seen := map[string]bool{}
	for _, sys := range cfg.Integrations.Enabled {
		if seen[sys] {
			continue
		}
		seen[sys] = true
		in := dom.Installed{System: sys, Stand: true, Real: true}
		switch sys {
		case "onec":
			stand := cfg.ERP.OneC.Stand || cfg.ERP.OneC.BaseURL == ""
			in.Stand, in.Real = stand, !stand
		case "galaktika":
			in.Stand, in.Real = cfg.ERP.Galaktika.Stand, !cfg.ERP.Galaktika.Stand
		case "mes":
			in.Stand, in.Real = cfg.MES.B2MML.Stand, !cfg.MES.B2MML.Stand
		}
		out = append(out, in)
	}
	return out
}

// switchedStore — хранилище очереди исходящих, которое не отдаёт к отправке
// сообщения выключенной системы (AD-47): строки остаются в очереди (Apply
// потребителя их наполняет) и уходят после включения — отправитель
// опрашивает очередь каждые Poll, а решение видно через SwitchTTL.
type switchedStore struct {
	erpapp.OutboxStore
	sw *opsapp.IntegrationSwitch
}

func (s switchedStore) Due(ctx context.Context, system string, now time.Time, limit int) ([]erpapp.OutboxRow, error) {
	if !s.sw.Active(ctx, system) {
		return nil, nil
	}
	return s.OutboxStore.Due(ctx, system, now, limit)
}

// switchedLedger — порт учёта, который не опрашивает входящие выключенной
// системы (AD-47: входящий адаптер перестаёт опрашивать).
type switchedLedger struct {
	erpapp.Ledger
	sw *opsapp.IntegrationSwitch
}

func (l switchedLedger) Pull(ctx context.Context) ([]erpapp.Inbound, error) {
	if !l.sw.Active(ctx, l.Info().System) {
		return nil, nil
	}
	return l.Ledger.Pull(ctx)
}

// switchedMES — канал MES, который не опрашивает входящие выключенной MES.
type switchedMES struct {
	mesapp.Channel
	sw *opsapp.IntegrationSwitch
}

func (m switchedMES) Pull(ctx context.Context) (mesapp.Inbound, error) {
	if !m.sw.Active(ctx, "mes") {
		return mesapp.Inbound{}, nil
	}
	return m.Channel.Pull(ctx)
}

// integrationProbe — «проверить соединение» (AD-47): у учётной системы и MES —
// та же сверка ответной стороны, что при старте роли outbox; у прочих
// адаптеров своей проверки нет.
type integrationProbe struct{ env *environment }

func (p integrationProbe) Probe(ctx context.Context, system string) (opsapp.ProbeResult, error) {
	switch system {
	case "onec", "galaktika":
		if ledgerSystem(p.env) != system {
			return opsapp.ProbeResult{Result: opsapp.ProbeNotSupported, Detail: "порт учёта обслуживает другую систему: " + ledgerSystem(p.env)}, nil
		}
		l, _, err := ledger(p.env)
		if err != nil || l == nil {
			return opsapp.ProbeResult{}, fmt.Errorf("адаптер учёта не собран: %v", err)
		}
		info := l.Info()
		c, err := l.Check(ctx)
		return probeOf(info.Endpoint, c.Detail, err, erpContract), nil
	case "mes":
		ch, err := mesChannel(p.env)
		if err != nil {
			return opsapp.ProbeResult{}, err
		}
		detail, err := ch.Check(ctx)
		return probeOf(ch.Info().Endpoint, detail, err, mesContract), nil
	}
	return opsapp.ProbeResult{Result: opsapp.ProbeNotSupported, Detail: "у адаптера " + system + " нет проверки ответной стороны"}, nil
}

func erpContract(err error) (string, bool) {
	if ce, ok := erpapp.AsContract(err); ok {
		return ce.Detail, true
	}
	return "", false
}

func mesContract(err error) (string, bool) {
	if ce, ok := mesapp.AsContract(err); ok {
		return ce.Detail, true
	}
	return "", false
}

// probeOf — итог сверки: контракт не совпал — degraded, прочая ошибка — unreachable.
func probeOf(endpoint, detail string, err error, contract func(error) (string, bool)) opsapp.ProbeResult {
	r := opsapp.ProbeResult{Result: opsapp.ProbeOK, Endpoint: endpoint, Detail: detail}
	if err != nil {
		if d, ok := contract(err); ok {
			r.Result, r.Detail = opsapp.ProbeDegraded, d
		} else {
			r.Result, r.Detail = opsapp.ProbeUnreachable, err.Error()
		}
	}
	if r.Detail == "" && r.Result == opsapp.ProbeOK {
		r.Detail = "ответная сторона отвечает, версия контракта совпала"
	}
	return r
}

// switchedBlocks — проекции для отправителя блоков MES: у выключенной MES
// индекс неподтверждённых блоков пуст — блоки остаются в проекции
// «ждут отправки» и уходят после включения (AD-47), попытки не тратятся.
type switchedBlocks struct {
	engineapp.ProjectionStore
	sw *opsapp.IntegrationSwitch
}

func (p switchedBlocks) Get(ctx context.Context, name, key string) (json.RawMessage, bool, error) {
	if name == mesapp.ProjectionBlockIndex && !p.sw.Active(ctx, "mes") {
		return nil, false, nil
	}
	return p.ProjectionStore.Get(ctx, name, key)
}
