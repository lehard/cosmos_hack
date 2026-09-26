package vision

import (
	"embed"
	"io/fs"

	appvision "ant/internal/application/vision"
	"ant/internal/contracts/normative"
)

// seed — встроенная копия затравки normative/vision (пути от корня
// репозитория): стартовые паспорта демо и каталог иллюстраций. Совпадение с
// normative/ проверяет TestSeedMatchesRepo.
//
//go:embed all:seed
var seed embed.FS

// Seed — файлы затравки модуля vision: корень — как корень репозитория.
func Seed() fs.FS {
	sub, err := fs.Sub(seed, "seed")
	if err != nil {
		panic(err) // каталог seed встроен всегда
	}
	return sub
}

// PassportsSeed — стартовые паспорта допуска демо (AD-29, AD-33).
func PassportsSeed() (normative.AnalyzerPassportsSeed, error) {
	return appvision.LoadPassportsSeed(Seed())
}

// Illustrations — каталог иллюстраций открытых наборов (FR-102).
func Illustrations() (normative.IllustrationsCatalog, error) {
	return appvision.LoadIllustrations(Seed())
}
