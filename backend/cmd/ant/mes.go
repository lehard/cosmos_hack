package main

import (
	"context"
	"io/fs"
	"slices"

	ingestapp "ant/internal/application/ingest"
	mesapp "ant/internal/application/mes"
	mesdom "ant/internal/domain/mes"
	"ant/internal/infrastructure/fixtures/world"
	"ant/internal/infrastructure/integration/mes/b2mml"
	"ant/internal/infrastructure/storage/journal/feed"
)

// Эпик 31: модуль mes — реакция «блок передаётся в MES» в роли projector
// (engine.go), отправка блоков и шлюз входящих заданий и событий операций в
// роли outbox (outbox.go, аренда outbox.mes), чтение блоков и заданий в роли
// api (api.go). Канал — порт mes.Channel, адаптер B2MML-JSON (mes.isa95.v1);
// работает, если "mes" есть в integrations.enabled (stand MES — эпик 43).

// mesEnabled — канал MES включён (integrations.enabled содержит mes).
func mesEnabled(env *environment) bool {
	return slices.Contains(env.cfg.Integrations.Enabled, "mes")
}

// mesChannel — адаптер канала MES по конфигурации mes.b2mml.
func mesChannel(env *environment) (*b2mml.Client, error) {
	c := env.cfg.MES.B2MML
	pw, err := secretFile("mes.b2mml.password_file", c.PasswordFile)
	if err != nil {
		return nil, err
	}
	return b2mml.New(b2mml.Config{BaseURL: c.BaseURL, LogicalID: c.LogicalID, User: c.User, Password: pw, Timeout: c.Timeout, Stand: c.Stand})
}

// processXML — стартовая версия процесса (из генезиса; без генезиса —
// встроенная копия normative/): нормативный слой для перевода связей КОМПАС
// и кодов операций MES (AD-17).
func processXML(ctx context.Context, c *core, env *environment) ([]byte, error) {
	return genesisProcessXML(ctx, c, env, func() ([]byte, error) { return fs.ReadFile(world.Inputs(), processSeedFile) })
}

// mesEnvOf — соответствия MES на момент опроса (FR-95, AD-18): экземпляры,
// сотрудники и оборудование MES — по записям reference.external_id.mapped
// системы mes (справочник из журнала, эпик 19); шаги — по кодам операций
// версии процесса.
func mesEnvOf(c *core, steps map[string]string) func(context.Context) (mesdom.Env, error) {
	return func(ctx context.Context) (mesdom.Env, error) {
		env := mesdom.Env{Steps: steps, Items: map[string]string{}, Persons: map[string]string{}, Equipment: map[string]string{}}
		b, err := c.refSource.Book(ctx, 0)
		if err != nil {
			return env, err
		}
		for _, v := range b.MappingsView("mes") {
			if !v.Effective {
				continue
			}
			ext, id := v.Data.ExternalID, string(v.Data.InternalID)
			switch string(v.Data.ObjectKind) {
			case "item":
				env.Items[ext] = id
			case "person":
				env.Persons[ext] = id
			case "equipment":
				env.Equipment[ext] = id
			}
		}
		return env, nil
	}
}

// mesOutbox — часть роли outbox для MES: сверка канала, отправка блоков
// (Sender) и опрос входящих (Gateway) под арендой outbox.mes.
func mesOutbox(ctx context.Context, env *environment, c *core, intake *ingestapp.Service) (*mesapp.Outbox, error) {
	ch, err := mesChannel(env)
	if err != nil {
		return nil, err
	}
	xml, err := processXML(ctx, c, env)
	if err != nil {
		return nil, err
	}
	steps, err := mesapp.StepsFromBPMN(xml)
	if err != nil {
		return nil, err
	}
	cfg := env.cfg.MES.B2MML
	log := env.log.With("module", "mes")
	info := ch.Info()
	log.Info("outbox: канал MES", "system", info.System, "endpoint", info.Endpoint, "stand", info.Stand)
	return &mesapp.Outbox{
		// Эпик 48: блоки выключенной MES ждут в проекции (switchedBlocks).
		Sender: &mesapp.Sender{Journal: c.journal, Codec: c.codec, Store: switchedBlocks{ProjectionStore: c.engine, sw: integrationSwitch(env, c)}, Channel: ch,
			Clock: c.domainClock(), Now: c.codec.Now, Max: cfg.RetryMax, Log: log},
		// Эпик 48: входящие выключенной MES не опрашиваются.
		Gateway: &mesapp.Gateway{Channel: switchedMES{Channel: ch, sw: integrationSwitch(env, c)}, Intake: mesIntake{intake}, Env: mesEnvOf(c, steps), Log: log},
		Poll:    cfg.Poll, PullEvery: cfg.PullEvery, Recheck: cfg.Recheck, Now: c.codec.Now, Log: log,
	}, nil
}

// mesHoldReactor — реакция «блок передаётся в MES» (роль projector,
// потребитель mes.holds со своим курсором, AD-45); nil — канал MES выключен:
// блок без канала никуда не уйдёт, записи mes.hold.requested не пишутся.
func mesHoldReactor(env *environment, c *core) *mesapp.HoldReactor {
	if !mesEnabled(env) {
		return nil
	}
	return &mesapp.HoldReactor{Consumer: feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "projector")),
		Codec: c.codec, Store: c.engine, Log: env.log.With("module", "mes")}
}

// mesLive — живая реализация операций mes.* для роли api: блоки и задания
// из проекций mes.block и mes.job.
func mesLive(ctx context.Context, env *environment) (*mesapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	return mesapp.NewLive(mesapp.Config{Projections: c.engine}), nil
}

// mesIntake — порт входа mes (application/mes.Intake) над живым приёмом.
type mesIntake struct{ s *ingestapp.Service }

func (i mesIntake) Submit(ctx context.Context, source string, envs [][]byte) error {
	_, err := submitEnvelopes(ctx, i.s, source, envs)
	return err
}
