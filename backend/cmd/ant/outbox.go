package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"time"

	erpapp "ant/internal/application/erp"
	ingestapp "ant/internal/application/ingest"
	"ant/internal/contracts/crypto"
	dom "ant/internal/domain/erp"
	"ant/internal/infrastructure/fixtures/world"
	"ant/internal/infrastructure/integration/erp/galaktika"
	galstand "ant/internal/infrastructure/integration/erp/galaktika/stand"
	"ant/internal/infrastructure/integration/erp/onec"
	erpstore "ant/internal/infrastructure/storage/erp"
	"ant/internal/infrastructure/storage/journal/clock"
	"ant/internal/infrastructure/storage/journal/feed"
)

// Эпик 30: модуль erp целиком — роль outbox (исходящие учётные сообщения и
// квитанции, шлюз входящих), реакция «учётное сообщение сформировано» в роли
// projector (engine.go), живые операции erp.* в роли api (api.go), stand 1С в
// роли stands (stands.go). Эпик 31: порт учёта — 1С или Галактика по
// integrations.enabled, канал MES в той же роли (mes.go). Реестр ролей в main.go не меняется: тело роли
// outbox подменяется здесь.
func init() {
	roles["outbox"] = role{run: runOutbox}
}

// onecEnabled — канал 1С включён (integrations.enabled содержит onec).
func onecEnabled(env *environment) bool {
	return slices.Contains(env.cfg.Integrations.Enabled, "onec")
}

// ledgerSystem — учётная система порта учёта (эпики 30, 31): erp.ledger, если
// она в integrations.enabled (эпик 43: в demo установлены обе); иначе
// galaktika, если она в integrations.enabled, иначе onec; "" — обмен с учётом
// выключен. Одна учётная система на экземпляр (docs/new-adapter.md, §5).
func ledgerSystem(env *environment) string {
	if l := env.cfg.ERP.Ledger; l != "" && slices.Contains(env.cfg.Integrations.Enabled, l) {
		return l
	}
	switch {
	case slices.Contains(env.cfg.Integrations.Enabled, "galaktika"):
		return "galaktika"
	case onecEnabled(env):
		return "onec"
	}
	return ""
}

// secretFile — секрет из файла (пусто — без секрета).
func secretFile(key, path string) (string, error) {
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}
	return strings.TrimSpace(string(b)), nil
}

// onecClient — адаптер порта учёта для 1С по конфигурации erp.onec.
func onecClient(env *environment) (*onec.Client, error) {
	c := env.cfg.ERP.OneC
	cfg := onec.Config{BaseURL: c.BaseURL, User: c.User, Enterprise: c.Enterprise, Timeout: c.Timeout, Stand: c.Stand}
	if cfg.BaseURL == "" {
		cfg.BaseURL, cfg.Stand = "http://127.0.0.1:8491/stand/1c/erp", true
	}
	pw, err := secretFile("erp.onec.password_file", c.PasswordFile)
	if err != nil {
		return nil, err
	}
	cfg.Password = pw
	return onec.New(cfg)
}

// galaktikaClient — адаптер порта учёта для Галактики по конфигурации
// erp.galaktika (эпик 31, gal.qc.v1: каталог обмена или REST-фасад).
func galaktikaClient(env *environment) (*galaktika.Client, error) {
	c := env.cfg.ERP.Galaktika
	pw, err := secretFile("erp.galaktika.password_file", c.PasswordFile)
	if err != nil {
		return nil, err
	}
	if c.BaseURL == "" && c.Stand {
		// Эпик 43: фасад stand-а Галактики роли stands этого хоста (для rest-facade).
		c.BaseURL = standURL(env, galstand.Name+galstand.Facade)
	}
	return galaktika.New(galaktika.Config{Transport: c.Transport, Dir: c.Dir, BaseURL: c.BaseURL, Node: c.Node,
		Peer: c.Peer, Database: c.Database, Enterprise: c.Enterprise, User: c.User, Password: pw,
		Timeout: c.Timeout, Stand: c.Stand})
}

// ledgerTiming — опросы и повторы роли outbox выбранной учётной системы.
type ledgerTiming struct {
	Poll, Recheck, PullEvery time.Duration
	RetryMax                 int
}

// ledger — адаптер порта учёта по конфигурации (1С — эпик 30, Галактика —
// эпик 31); nil — обмен с учётом выключен. Роль outbox, реакция, шлюз
// входящих и операции API берут порт, а не конкретную систему.
func ledger(env *environment) (erpapp.Ledger, ledgerTiming, error) {
	switch ledgerSystem(env) {
	case "galaktika":
		c := env.cfg.ERP.Galaktika
		l, err := galaktikaClient(env)
		return l, ledgerTiming{c.Poll, c.Recheck, c.PullEvery, c.RetryMax}, err
	case "onec":
		c := env.cfg.ERP.OneC
		l, err := onecClient(env)
		return l, ledgerTiming{c.Poll, c.Recheck, c.PullEvery, c.RetryMax}, err
	}
	return nil, ledgerTiming{}, nil
}

// erpEnv — окружение правил erp: цеха и склады процесса из нормативного слоя
// (дорожки BPMN, FR-130) — встроенная копия normative/process, пока версия
// процесса не приходит из журнала (эпик 17); учётная система — из порта
// учёта (Ledger.Info().System; обмен выключен — onec по умолчанию домена).
func erpEnv(env *environment) (dom.Env, error) {
	system := "onec"
	if l, _, err := ledger(env); err != nil {
		return dom.Env{}, err
	} else if l != nil {
		system = l.Info().System
	}
	return erpEnvFor(system)
}

// erpEnvFor — окружение правил erp для учётной системы system.
func erpEnvFor(system string) (dom.Env, error) {
	b, err := fs.ReadFile(world.Inputs(), "normative/process/flange-process.bpmn")
	if err != nil {
		return dom.Env{}, err
	}
	topo, err := erpapp.TopologyFromBPMN(b)
	if err != nil {
		return dom.Env{}, err
	}
	return dom.Env{Topology: topo, System: system}, nil
}

// erpReactor — реакция «учётное сообщение сформировано» (роль projector,
// потребитель erp.postings с собственной арендой).
func erpReactor(env *environment, c *core) (*erpapp.Reactor, error) {
	e, err := erpEnv(env)
	if err != nil {
		return nil, err
	}
	return &erpapp.Reactor{Consumer: feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "projector")),
		Codec: c.codec, Store: c.engine, Env: e, Log: env.moduleLog("erp")}, nil
}

// runOutbox — роль outbox (AD-6, AD-7, AD-18): одна копия-лидер по аренде
// outbox отправляет исходящие учётные сообщения в учётную систему (1С или
// Галактика), пишет квитанции и карантин в журнал, сверяет ответную сторону
// и подаёт входящие факты в приём; копия-лидер по аренде outbox.mes
// отправляет блоки в MES и опрашивает её входящие (эпик 31, mes.go). При
// воспроизведении и пересборке роль не запускается.
func runOutbox(ctx context.Context, env *environment) error {
	l, timing, err := ledger(env)
	if err != nil {
		return err
	}
	mesOn := mesEnabled(env)
	if l == nil && !mesOn && !skudEnabled(env) {
		env.log.Warn("outbox: обмен с учётной системой, MES и СКУД выключен (integrations.enabled) — ожидаю остановки")
		<-ctx.Done()
		return nil
	}
	c, err := env.readyCore(ctx)
	if err != nil {
		return err
	}
	intake, err := ingestLive(ctx, env)
	if err != nil {
		return err
	}
	var runs []func(context.Context) error
	if l != nil {
		retry := erpapp.DefaultRetry()
		if timing.RetryMax > 0 {
			retry.Max = timing.RetryMax
		}
		ob := &erpapp.Outbox{
			Journal:  c.journal,
			Consumer: feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "outbox")),
			// Смена состояния канала — ops.integration.degraded через порт ops (эпик 34).
			// Эпик 48: выключенная администратором система — исходящие копятся
			// в очереди, входящие не опрашиваются (switchedStore, switchedLedger).
			Codec: c.codec, Store: switchedStore{OutboxStore: channelWatch{OutboxStore: erpstore.NewStore(c.pool), rep: opsReporter(env, c), log: env.moduleLog("ops")}, sw: integrationSwitch(env, c)},
			Ledger: switchedLedger{Ledger: l, sw: integrationSwitch(env, c)}, Intake: ingestIntake{intake},
			Clock: clock.NewJournal(c.journal), Now: c.codec.Now, Retry: retry,
			Poll: timing.Poll, Recheck: timing.Recheck, PullEvery: timing.PullEvery, Log: env.moduleLog("erp"),
		}
		info := l.Info()
		env.log.Info("outbox: старт", "system", info.System, "endpoint", info.Endpoint)
		runs = append(runs, func(ctx context.Context) error { return c.leader(env, "outbox").Run(ctx, ob.Run) })
	}
	if mesOn {
		mo, err := mesOutbox(ctx, env, c, intake)
		if err != nil {
			return err
		}
		runs = append(runs, func(ctx context.Context) error { return c.leader(env, "outbox.mes").Run(ctx, mo.Run) })
	}
	// Эпик 37: опрос журнала проходов СКУД (access_skud.go).
	if skudEnabled(env) {
		run, err := skudRun(ctx, env, c)
		if err != nil {
			return err
		}
		runs = append(runs, run)
	}
	return runAll(ctx, runs)
}

// runAll — функции до отмены ctx или первой ошибки (остальные тогда останавливаются).
func runAll(ctx context.Context, runs []func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errc := make(chan error, len(runs))
	for _, run := range runs {
		go func() {
			err := run(ctx)
			cancel()
			errc <- err
		}()
	}
	var errs []error
	for range runs {
		if err := <-errc; err != nil && !errors.Is(err, context.Canceled) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// erpLive — живая реализация операций erp.* для роли api: проекции erp.*,
// состояние каналов и попытки роли outbox (схема erp), решения — в журнал.
func erpLive(ctx context.Context, env *environment) (*erpapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	var channels []erpapp.LedgerInfo
	if l, _, err := ledger(env); err == nil && l != nil {
		channels = append(channels, l.Info())
	}
	return erpapp.NewLive(erpapp.Config{
		Projections: c.engine, Outbox: erpstore.NewStore(c.pool),
		Decisions: erpapp.JournalDecisions{Journal: c.journal, DomainBuild: c.codec.DomainBuild, Now: c.codec.Now},
		Clock:     clock.NewJournal(c.journal), Channels: channels,
	}), nil
}

// ingestIntake — порт входа erp (application/erp.Intake) над живым приёмом.
type ingestIntake struct{ s *ingestapp.Service }

func (i ingestIntake) Submit(ctx context.Context, source string, envs [][]byte) (erpapp.IntakeResult, error) {
	r, err := submitEnvelopes(ctx, i.s, source, envs)
	return erpapp.IntakeResult{Accepted: r.Accepted, Duplicates: r.Duplicates, Quarantined: r.Quarantined}, err
}

// submitEnvelopes — пачка конвертов DSSE шлюза внешней системы в обычный
// приём (AD-18): ingest.SubmitBatch от источника source (erp, mes, cad).
func submitEnvelopes(ctx context.Context, s *ingestapp.Service, source string, envs [][]byte) (ingestapp.IngestResult, error) {
	b := ingestapp.IngestBatch{SourceID: source}
	for _, raw := range envs {
		var e crypto.DsseEnvelope
		if err := json.Unmarshal(raw, &e); err != nil {
			return ingestapp.IngestResult{}, err
		}
		b.Envelopes = append(b.Envelopes, e)
	}
	now := time.Now().UTC()
	b.SentAt = &now
	return s.SubmitBatch(ctx, b)
}
