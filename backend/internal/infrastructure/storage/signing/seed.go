package signing

import (
	"embed"
	"io/fs"
)

// seed — встроенная копия затравки normative/ (кроме README; пути от корня
// репозитория): из неё ant init собирает блок генезиса (AD-33), а ядро
// сверяет с генезисом байты стартового процесса. Совпадение с normative/
// репозитория проверяет TestSeedMatchesRepo — при правке normative/
// скопируйте файл в seed/.
//
//go:embed all:seed
var seed embed.FS

// Seed — затравка нормативного слоя: корень — как корень репозитория.
func Seed() fs.FS {
	sub, err := fs.Sub(seed, "seed")
	if err != nil {
		panic(err) // каталог seed встроен всегда
	}
	return sub
}
