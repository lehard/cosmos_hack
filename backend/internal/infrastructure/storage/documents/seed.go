package documents

import (
	"embed"
	"io/fs"

	app "ant/internal/application/documents"
	dom "ant/internal/domain/documents"
)

// seed — встроенная копия файлов нормативного слоя модуля documents (пути от
// корня репозитория): шаблоны документов и стартовая политика (срез для
// проверки подписей до проекции политики эпика 26). Совпадение с normative/
// проверяет TestSeedMatchesRepo.
//
//go:embed all:seed
var seed embed.FS

// Seed — файлы нормативного слоя модуля documents: корень — как корень репозитория.
func Seed() fs.FS {
	sub, err := fs.Sub(seed, "seed")
	if err != nil {
		panic(err) // каталог seed встроен всегда
	}
	return sub
}

// SeedEnv — Env модуля documents по встроенной копии; verification — full |
// demo (демо-профиль: подписи без агента токена, Д-30).
func SeedEnv(verification string) (dom.Env, error) {
	return app.EnvFromFS(Seed(), verification)
}
