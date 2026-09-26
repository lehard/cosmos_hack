// Пакет enginewire — сборка движка для независимого верификатора (AD-9):
// тот же реестр проекций и тот же нормативный слой изделия, что у воркера в
// cmd/ant (engineRegistry, bundleSource). Совпадение реестров проверяет тест
// cmd/ant (verifier_registry_test.go): модуль, добавивший проекцию в cmd/ant,
// добавляет её и сюда — иначе верификатор её не сверит.
//
// Слой: сборка зависимостей (cmd/*). Владелец: эпик 29.
package enginewire

import (
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	analysisapp "ant/internal/application/analysis"
	analyticsapp "ant/internal/application/analytics"
	cadapp "ant/internal/application/cad"
	documentsapp "ant/internal/application/documents"
	engineapp "ant/internal/application/engine"
	erpapp "ant/internal/application/erp"
	itemapp "ant/internal/application/item"
	machinelogsapp "ant/internal/application/machinelogs"
	mesapp "ant/internal/application/mes"
	nonconformityapp "ant/internal/application/nonconformity"
	notificationsapp "ant/internal/application/notifications"
	processapp "ant/internal/application/process"
	qualityapp "ant/internal/application/quality"
	referenceapp "ant/internal/application/reference"
	itemdom "ant/internal/domain/item"
	mldomain "ant/internal/domain/machinelogs"
	"ant/internal/domain/quality"
	storagedocs "ant/internal/infrastructure/storage/documents"
	itemstore "ant/internal/infrastructure/storage/item"
	processstore "ant/internal/infrastructure/storage/process"
	qualitystore "ant/internal/infrastructure/storage/quality"
)

// Registry — реестр проекций движка (как cmd/ant engineRegistry).
func Registry() *engineapp.Registry {
	r := engineapp.NewRegistry()
	must(machinelogsapp.RegisterProjections(r, mldomain.Env{}))
	must(processapp.RegisterProjections(r))
	analysisapp.MustRegister(r)
	must(analyticsapp.Register(r))
	erpapp.MustRegister(r)
	must(nonconformityapp.RegisterProjections(r))
	must(qualityapp.Register(r))
	must(notificationsapp.Register(r))
	must(itemapp.Register(r))
	must(documentsapp.RegisterProjections(r))
	cadapp.MustRegister(r)
	mesapp.MustRegister(r)
	return r
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// qualityRev — ревизия встроенной стартовой версии нормативного слоя quality (как в cmd/ant).
const qualityRev = "normative-seed-v1"

var (
	qualityEnv = sync.OnceValues(func() (quality.Env, error) { return qualitystore.SeedEnv(qualityRev) })
	itemEnv    = sync.OnceValues(func() (itemdom.Env, error) { return itemstore.SeedEnv() })
)

// Bundles — нормативный слой изделия, как cmd/ant bundleSource: версия
// процесса, закреплённая при запуске изделия (схема process), слои quality,
// item, notifications и reference поверх неё. Совпадение цепочки проверяет тест cmd/ant.
func Bundles(pool *pgxpool.Pool, codec *engineapp.Codec, docsVerification string) (engineapp.BundleSource, error) {
	qenv, err := qualityEnv()
	if err != nil {
		return nil, err
	}
	ienv, err := itemEnv()
	if err != nil {
		return nil, err
	}
	var process engineapp.BundleSource = &processapp.Bundles{Store: &processstore.Versions{Pool: pool}, Quorum: processapp.RecordedQuorum{}, Now: time.Now}
	q := qualityapp.Bundles{Next: process, Env: qenv, Passports: qualitystore.JournalPassports{Journal: codec.Store, Codec: codec}}
	// Слой справочников (эпик 19) — внешний, как у воркера: срез поверки,
	// квалификаций и календаря на basis_seq входа изделия.
	ref := &referenceapp.JournalSource{Journal: codec.Store, Codec: codec}
	// Слой documents (эпик 28) — в том же режиме проверки подписей, что у
	// воркера (demo | full по профилю), иначе пересвёртка даст ложные тревоги.
	denv, err := storagedocs.SeedEnv(docsVerification)
	if err != nil {
		return nil, err
	}
	docs := documentsapp.Bundles{Next: notificationsapp.Bundles{Next: itemapp.Bundles{Next: q, Env: ienv}}, Env: denv}
	return &referenceapp.Bundles{Next: docs, Source: ref}, nil
}
