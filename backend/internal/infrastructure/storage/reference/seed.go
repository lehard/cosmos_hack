package reference

import (
	"embed"
	"io/fs"

	app "ant/internal/application/reference"
)

// seed — встроенная копия справочников демо-изделия (пути от корня
// репозитория, normative/reference/flange/*.yaml): затравка генезисом при
// migrate и книга заготовок. Совпадение с normative/ проверяет TestSeedMatchesRepo.
//
//go:embed all:seed
var seed embed.FS

// Seed — файлы справочников: корень — как корень репозитория.
func Seed() fs.FS {
	sub, err := fs.Sub(seed, "seed")
	if err != nil {
		panic(err) // каталог seed встроен всегда
	}
	return sub
}

// SeedRecords — записи затравки справочников из встроенной копии.
func SeedRecords() ([]app.Record, error) { return app.SeedRecords(Seed()) }
