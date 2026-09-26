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
	engineapp "ant/internal/application/engine"
	machinelogsapp "ant/internal/application/machinelogs"
	nonconformityapp "ant/internal/application/nonconformity"
	processapp "ant/internal/application/process"
	qualityapp "ant/internal/application/quality"
	mldomain "ant/internal/domain/machinelogs"
	"ant/internal/domain/quality"
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
	must(nonconformityapp.RegisterProjections(r))
	must(qualityapp.Register(r))
	return r
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// qualityRev — ревизия встроенной стартовой версии нормативного слоя quality (как в cmd/ant).
const qualityRev = "normative-seed-v1"

var qualityEnv = sync.OnceValues(func() (quality.Env, error) { return qualitystore.SeedEnv(qualityRev) })

// Bundles — нормативный слой изделия: версия процесса, закреплённая при
// запуске изделия (схема process), и слой quality поверх неё (как cmd/ant bundleSource).
func Bundles(pool *pgxpool.Pool, codec *engineapp.Codec) (engineapp.BundleSource, error) {
	env, err := qualityEnv()
	if err != nil {
		return nil, err
	}
	next := &processapp.Bundles{Store: &processstore.Versions{Pool: pool}, Quorum: processapp.RecordedQuorum{}, Now: time.Now}
	return qualityapp.Bundles{Next: next, Env: env, Passports: qualitystore.JournalPassports{Journal: codec.Store, Codec: codec}}, nil
}
