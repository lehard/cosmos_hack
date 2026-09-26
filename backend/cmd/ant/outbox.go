package main

import (
	"context"
	"encoding/json"
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
	"ant/internal/infrastructure/integration/erp/onec"
	erpstore "ant/internal/infrastructure/storage/erp"
	"ant/internal/infrastructure/storage/journal/clock"
	"ant/internal/infrastructure/storage/journal/feed"
)

// Эпик 30: модуль erp целиком — роль outbox (исходящие учётные сообщения и
// квитанции, шлюз входящих), реакция «учётное сообщение сформировано» в роли
// projector (engine.go), живые операции erp.* в роли api (api.go), stand 1С в
// роли stands (stands.go). Реестр ролей в main.go не меняется: тело роли
// outbox подменяется здесь.
func init() {
	roles["outbox"] = role{run: runOutbox}
}

// onecEnabled — канал 1С включён (integrations.enabled содержит onec).
func onecEnabled(env *environment) bool {
	return slices.Contains(env.cfg.Integrations.Enabled, "onec")
}

// onecClient — адаптер порта учёта для 1С по конфигурации erp.onec.
func onecClient(env *environment) (*onec.Client, error) {
	c := env.cfg.ERP.OneC
	cfg := onec.Config{BaseURL: c.BaseURL, User: c.User, Enterprise: c.Enterprise, Timeout: c.Timeout, Stand: c.Stand}
	if cfg.BaseURL == "" {
		cfg.BaseURL, cfg.Stand = "http://127.0.0.1:8491/stand/1c/erp", true
	}
	if c.PasswordFile != "" {
		pw, err := os.ReadFile(c.PasswordFile)
		if err != nil {
			return nil, fmt.Errorf("erp.onec.password_file: %w", err)
		}
		cfg.Password = strings.TrimSpace(string(pw))
	}
	return onec.New(cfg)
}

// erpEnv — окружение правил erp: цеха и склады процесса из нормативного слоя
// (дорожки BPMN, FR-130) — встроенная копия normative/process, пока версия
// процесса не приходит из журнала (эпик 17).
func erpEnv() (dom.Env, error) {
	b, err := fs.ReadFile(world.Inputs(), "normative/process/flange-process.bpmn")
	if err != nil {
		return dom.Env{}, err
	}
	topo, err := erpapp.TopologyFromBPMN(b)
	if err != nil {
		return dom.Env{}, err
	}
	return dom.Env{Topology: topo, System: "onec"}, nil
}

// erpReactor — реакция «учётное сообщение сформировано» (роль projector,
// потребитель erp.postings с собственной арендой).
func erpReactor(env *environment, c *core) (*erpapp.Reactor, error) {
	e, err := erpEnv()
	if err != nil {
		return nil, err
	}
	return &erpapp.Reactor{Consumer: feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "projector")),
		Codec: c.codec, Store: c.engine, Env: e, Log: env.log.With("module", "erp")}, nil
}

// runOutbox — роль outbox (AD-6, AD-7, AD-18): одна копия-лидер по аренде
// outbox отправляет исходящие учётные сообщения в 1С, пишет квитанции и
// карантин в журнал, сверяет ответную сторону и подаёт входящие факты в
// приём. При воспроизведении и пересборке роль не запускается.
func runOutbox(ctx context.Context, env *environment) error {
	if !onecEnabled(env) {
		env.log.Warn("outbox: обмен с 1С выключен (integrations.enabled без onec) — ожидаю остановки")
		<-ctx.Done()
		return nil
	}
	c, err := env.readyCore(ctx)
	if err != nil {
		return err
	}
	ledger, err := onecClient(env)
	if err != nil {
		return err
	}
	intake, err := ingestLive(ctx, env)
	if err != nil {
		return err
	}
	oc := env.cfg.ERP.OneC
	retry := erpapp.DefaultRetry()
	if oc.RetryMax > 0 {
		retry.Max = oc.RetryMax
	}
	ob := &erpapp.Outbox{
		Journal:  c.journal,
		Consumer: feed.NewConsumer(c.journal, c.leases, c.listener, c.feedOptions(env, "outbox")),
		Codec:    c.codec, Store: erpstore.NewStore(c.pool), Ledger: ledger, Intake: ingestIntake{intake},
		Clock: clock.NewJournal(c.journal), Now: c.codec.Now, Retry: retry,
		Poll: oc.Poll, Recheck: oc.Recheck, PullEvery: oc.PullEvery, Log: env.log.With("module", "erp"),
	}
	env.log.Info("outbox: старт", "system", "onec", "endpoint", ledger.Info().Endpoint)
	return c.leader(env, "outbox").Run(ctx, ob.Run)
}

// erpLive — живая реализация операций erp.* для роли api: проекции erp.*,
// состояние каналов и попытки роли outbox (схема erp), решения — в журнал.
func erpLive(ctx context.Context, env *environment) (*erpapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	var channels []erpapp.LedgerInfo
	if onecEnabled(env) {
		if l, err := onecClient(env); err == nil {
			channels = append(channels, l.Info())
		}
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
	b := ingestapp.IngestBatch{SourceID: source}
	for _, raw := range envs {
		var e crypto.DsseEnvelope
		if err := json.Unmarshal(raw, &e); err != nil {
			return erpapp.IntakeResult{}, err
		}
		b.Envelopes = append(b.Envelopes, e)
	}
	now := time.Now().UTC()
	b.SentAt = &now
	r, err := i.s.SubmitBatch(ctx, b)
	return erpapp.IntakeResult{Accepted: r.Accepted, Duplicates: r.Duplicates, Quarantined: r.Quarantined}, err
}
