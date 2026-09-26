package cad

import (
	"strings"
	"unicode"
)

// Правило внутренних ID по обозначению КД (FR-95, AD-18): наш ID типа
// изделия не выводится из ключей внешних систем (Ref_Key 1С, NRec Галактики,
// UniqueNum КОМПАС), а складывается из естественного ключа — обозначения по
// ЕСКД латиницей. Поэтому КОМПАС, 1С и Галактика, называющие позицию одним
// обозначением, приходят к одному нашему типу, а соответствия внешних ID
// хранятся записями reference.external_id.mapped.

var latin = map[rune]string{'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "E", 'Ж': "ZH", 'З': "Z", 'И': "I", 'Й': "Y",
	'К': "K", 'Л': "L", 'М': "M", 'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U", 'Ф': "F", 'Х': "KH", 'Ц': "TS",
	'Ч': "CH", 'Ш': "SH", 'Щ': "SHCH", 'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "YU", 'Я': "YA"}

// Latin — естественный ключ латиницей: транслитерация кириллицы в верхнем
// регистре, ASCII-буквы, цифры и `._:/@-` — как есть, прочее — «-»; повторные
// и крайние «-» убираются. Совпадает с правилом адаптера 1С
// (integration/erp/onec.Latin) на обозначениях без повторных разделителей.
func Latin(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		u := unicode.ToUpper(r)
		switch {
		case r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("._:/@-", r)):
			b.WriteRune(r)
		case latin[u] != "" || u == 'Ъ' || u == 'Ь':
			b.WriteString(latin[u])
		default:
			b.WriteByte('-')
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return strings.Trim(out, "-")
}

// Clean — обозначение без пометок в квадратных скобках («[П]», «[ПП]» —
// проектные допущения во входных файлах) и лишних пробелов.
func Clean(designation string) string {
	var b strings.Builder
	depth := 0
	for _, r := range designation {
		switch {
		case r == '[':
			depth++
		case r == ']' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// ItemTypeID — наш тип изделия по обозначению КД (FR-95):
//   - обозначение по ЕСКД (первое слово с точкой: «ФЛ-100.00.000 СБ») — первое
//     слово латиницей без кода документа: FL-100.00.000; так же, как адаптер 1С
//     сопоставляет реквизит «Артикул»;
//   - стандартные и покупные изделия без обозначения ЕСКД («Болт М6×20»,
//     «КВД-6») — всё обозначение латиницей: BOLT-M6-20, KVD-6.
//
// Пустое обозначение — пустой ID (ошибка импорта, а не новая деталь).
func ItemTypeID(designation string) string {
	d := Clean(designation)
	f := strings.Fields(d)
	if len(f) == 0 {
		return ""
	}
	if strings.Contains(f[0], ".") {
		return Latin(f[0])
	}
	return Latin(d)
}
