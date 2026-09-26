package quality

import (
	"embed"
	"io/fs"

	appquality "ant/internal/application/quality"
	"ant/internal/domain/quality"
)

// seed — встроенная копия файлов нормативного слоя модуля quality (пути от
// корня репозитория); совпадение с normative/ проверяет TestSeedMatchesRepo.
//
//go:embed all:seed
var seed embed.FS

// Seed — файлы нормативного слоя модуля quality: корень — как корень репозитория.
func Seed() fs.FS {
	sub, err := fs.Sub(seed, "seed")
	if err != nil {
		panic(err) // каталог seed встроен всегда
	}
	return sub
}

// SeedEnv — Env модуля quality по встроенной стартовой версии нормативного
// слоя (rev — ревизия для normative_rev реакций).
func SeedEnv(rev string) (quality.Env, error) {
	return appquality.EnvFromFS(Seed(), rev)
}
