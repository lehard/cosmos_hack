package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ant/cmd/internal/db"
	engineapp "ant/internal/application/engine"
	processapp "ant/internal/application/process"
	dj "ant/internal/domain/journal"
	"ant/internal/infrastructure/security/permissive"
	enginestore "ant/internal/infrastructure/storage/engine"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/clock"
	"ant/internal/infrastructure/storage/journal/feed"
	processstore "ant/internal/infrastructure/storage/process"
)

// engineKeyRef — ключ движка key_id@версия для записей воркера, стадии и
// проектора (AD-10). TODO(05): ключ из тома ключей и реестра доверия.
const engineKeyRef = "engine@1"

// defaultLeaseTTL — срок аренд, если engine.lease_ttl не задан.
const defaultLeaseTTL = 10 * time.Second

// core — ядро процесса, общее для его ролей (api, worker, crossitem,
// projector, rebuild): пул роли ant_app, журнал (эпик 04), аренды, сигнал
// «есть новое», хранение движка и кодек записей (эпик 07). Один на процесс:
// роли, совмещённые в одном процессе, делят пул и соединение LISTEN.
type core struct {
	pool     *pgxpool.Pool
	journal  *journalstore.Store
	leases   *journalstore.Leases
	listener *journalstore.Listener
	engine   *enginestore.Store
	codec    *engineapp.Codec
	registry *engineapp.Registry
	// holder — идентификатор копии для аренд (хост:pid).
	holder string
	ttl    time.Duration
	// versions, bundles — версии процесса (схема process) и нормативный слой
	// изделия по закреплённой версии (эпик 17, AD-17); seedOnce — загрузка
	// стартовой версии (FR-10).
	versions *processstore.Versions
	bundles  *processapp.Bundles
	seedOnce sync.Once
}

// coreHolder — ленивое создание ядра и его остановка после ролей.
type coreHolder struct {
	once sync.Once
	c    *core
	err  error
	bg   sync.WaitGroup
}

// core возвращает ядро процесса, создавая его при первом вызове. Фоновые
// циклы ядра (LISTEN) живут, пока не отменён общий ctx ролей процесса
// (runRoles), а не ctx роли, первой попросившей ядро.
func (e *environment) core(ctx context.Context) (*core, error) {
	e.coreH.once.Do(func() {
		bg := e.ctx
		if bg == nil {
			bg = ctx
		}
		e.coreH.c, e.coreH.err = openCore(bg, e)
	})
	return e.coreH.c, e.coreH.err
}

// closeCore ждёт фоновые циклы ядра и закрывает пул (после остановки ролей).
func (e *environment) closeCore() {
	e.coreH.bg.Wait()
	if c := e.coreH.c; c != nil {
		c.engine.Close()
		c.pool.Close()
	}
}

func openCore(ctx context.Context, env *environment) (*core, error) {
	cfg := env.cfg
	pc, err := db.Config(cfg.DB, "ant-core")
	if err != nil {
		return nil, err
	}
	// Вход — пользователем конфигурации (в демо ant_admin), работа — ролью
	// ant_app (SET ROLE): UPDATE, DELETE и TRUNCATE журнала ей недоступны (AD-1, AD-2).
	pool, err := journalstore.NewAppPool(ctx, pc)
	if err != nil {
		return nil, err
	}
	infra := clock.System{}
	host, _ := os.Hostname()
	ttl := cfg.Engine.LeaseTTL
	if ttl <= 0 {
		ttl = defaultLeaseTTL
	}
	batch := cfg.Journal.BatchMax
	if batch <= 0 {
		batch = journalstore.DefaultBatchMax
	}
	c := &core{
		pool:     pool,
		journal:  journalstore.NewStore(pool, infra, append([]journalstore.Option{journalstore.WithBatchMax(batch)}, trustOptions(cfg, env)...)...),
		leases:   journalstore.NewLeases(pool, infra),
		listener: journalstore.NewListener(pool, env.log),
		engine:   &enginestore.Store{Pool: pool},
		registry: engineRegistry(),
		versions: &processstore.Versions{Pool: pool},
		holder:   fmt.Sprintf("%s:%d", host, os.Getpid()),
		ttl:      ttl,
	}
	c.bundles = &processapp.Bundles{Store: c.versions, Quorum: processapp.RecordedQuorum{}, Now: infra.Now}
	c.codec = &engineapp.Codec{
		Store:  c.journal,
		Sealer: engineapp.SignerSealer{Signer: permissive.Signer{}, KeyRef: engineKeyRef},
		KeyRef: engineKeyRef, Profile: "gost",
		DomainBuild: domainBuild(), Partitions: cfg.Engine.Partitions, Now: infra.Now,
	}
	env.coreH.bg.Go(func() {
		if err := c.listener.Run(ctx); err != nil {
			env.log.Error("журнал: LISTEN остановлен", "err", err)
		}
	})
	return c, nil
}

// feedOptions — параметры потребителей и подачи работы роли role.
func (c *core) feedOptions(env *environment, role string) feed.Options {
	return feed.Options{Holder: c.holder + "/" + role, TTL: c.ttl, Log: env.log}
}

// leader — копия-лидер роли по аренде `role` (AD-6).
func (c *core) leader(env *environment, role string) engineapp.Leader {
	return engineapp.Leader{Leases: c.leases, Name: role, Holder: c.holder, TTL: c.ttl, Log: env.log}
}

// domainBuild — domain_build записей движка (AD-9): хеш сборки доменного
// пакета. До воспроизводимой сборки домена (эпик 29) — H(«ant-domain» ‖
// версия бинарника): записи разных версий различимы.
func domainBuild() string {
	return dj.H([]byte("ant-domain"), []byte(version)).String()
}
