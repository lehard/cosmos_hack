package vision

import (
	"slices"
	"strings"
	"unicode/utf8"

	"ant/internal/contracts/normative"
)

// IllustrationChoice — образец открытого набора к наблюдению (FR-102):
// иллюстрация, а не доказательство; не относится к изделию.
type IllustrationChoice struct {
	Dataset normative.IllustrationsCatalogDatasetsElem
	Class   normative.IllustrationsCatalogDatasetsElemClassesElem
}

// ChooseIllustration — класс набора для наблюдения: по первому виду дефекта,
// который класс иллюстрирует; «признаков нет» — класс без видов дефектов.
// «Оценка невозможна» не иллюстрируется: картинка годного шва к плохому кадру
// выдавала бы несуществующее доказательство (FR-102).
func ChooseIllustration(cat normative.IllustrationsCatalog, outcome string, defectCodes []string) (IllustrationChoice, bool) {
	for _, ds := range cat.Datasets {
		switch outcome {
		case "defect_indicated":
			for _, code := range defectCodes {
				for _, c := range ds.Classes {
					if slices.Contains(c.DefectCodes, code) {
						return IllustrationChoice{Dataset: ds, Class: c}, true
					}
				}
			}
		case "no_defect_indicated":
			for _, c := range ds.Classes {
				if len(c.DefectCodes) == 0 {
					return IllustrationChoice{Dataset: ds, Class: c}, true
				}
			}
		}
	}
	return IllustrationChoice{}, false
}

// SourceNote — пометка материала: «ИЛЛЮСТРАЦИЯ», оговорка, набор и класс,
// источник, лицензия, автор (FR-102, PRD §11.12); не длиннее 500 знаков.
func (c IllustrationChoice) SourceNote() string {
	d := c.Dataset
	parts := []string{d.Mark + " — " + strings.TrimSuffix(d.Note, "."),
		"набор «" + d.Title + "», класс " + c.Class.DatasetClass,
		"источник " + d.URL, "лицензия " + d.License, "автор " + d.Author}
	return clip(strings.Join(parts, "; ")+".", 500)
}

// Unavailable — ограничение наблюдения, когда иллюстрацию приложить нельзя:
// ссылка и метаданные вместо файла (отсутствие — не ошибка, кейс §5.4).
func (c IllustrationChoice) Unavailable(why string) string {
	d := c.Dataset
	return clip(d.Mark+" не приложена ("+why+"): набор «"+d.Title+"», класс "+c.Class.DatasetClass+", "+d.URL+", "+d.License+", "+d.Author, 256)
}

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}
