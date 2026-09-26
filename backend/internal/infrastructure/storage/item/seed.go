package item

import (
	"embed"
	"io/fs"

	appitem "ant/internal/application/item"
	dom "ant/internal/domain/item"
	dj "ant/internal/domain/journal"
)

// seed — встроенная копия файлов нормативного слоя модуля item (пути от корня
// репозитория): номенклатура с зонами и связями по КД и BPMN процесса (хеш
// версии процесса, закрепляемой при регистрации, AD-17). Совпадение с
// normative/ проверяет TestSeedMatchesRepo.
//
//go:embed all:seed
var seed embed.FS

// PathProcess — BPMN процесса фланца.
const PathProcess = "normative/process/flange-process.bpmn"

// Seed — файлы нормативного слоя модуля item: корень — как корень репозитория.
func Seed() fs.FS {
	sub, err := fs.Sub(seed, "seed")
	if err != nil {
		panic(err) // каталог seed встроен всегда
	}
	return sub
}

// SeedEnv — Env модуля item по встроенной номенклатуре.
func SeedEnv() (dom.Env, error) { return appitem.EnvFromFS(Seed()) }

// SeedProcessVersion — хеш встроенной версии процесса `streebog256:…` (AD-17:
// XML как загружен), пока источник версии (эпик 17) не закрепляет её сам.
func SeedProcessVersion() (string, error) {
	b, err := fs.ReadFile(Seed(), PathProcess)
	if err != nil {
		return "", err
	}
	return dj.H(b).String(), nil
}
