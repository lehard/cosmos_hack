package ingest

import (
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// SafetyCriticalEnums — поля-перечисления, важные для безопасности (FR-29,
// AD-20): неизвестное значение в них — карантин, а не UNKNOWN(значение) с
// флагом. Путь — JSON Pointer внутри `data`, `*` — любой элемент массива.
//
// Копия contracts/events/safety-critical-enums.yaml: генератор эпика 02 этот
// файл пока не переносит в Go, совпадение проверяет тест TestSafetyCriticalEnumsMatchContract.
var SafetyCriticalEnums = map[string][]string{
	"inspection.result.recorded":   {"/outcome", "/processing_state", "/defects/*/severity", "/measurements/*/verdict"},
	"equipment.state.changed":      {"/condition", "/execution"},
	"equipment.deviation.detected": {"/deviation_kind"},
	"operator.check.skipped":       {},
	"operation.run.finished":       {"/completion"},
	"erp.posting.responded":        {"/outcome"},
	"federation.extract.received":  {"/origin_status"},
}

var (
	critOnce sync.Once
	critRe   map[string][]*regexp.Regexp
)

// IsSafetyCritical — важно ли для безопасности поле pointer (JSON Pointer от
// корня сообщения) у типа eventType. Перечисления конверта вне `data`
// (вид источника, надёжность, криптопрофиль) важны всегда: без них нельзя
// честно пометить источник (FR-140) и проверить подпись (FR-26) — ограничивать
// безопаснее, чем разрешать (AD-27).
func IsSafetyCritical(eventType, pointer string) bool {
	rest, ok := strings.CutPrefix(pointer, "/data")
	if !ok || (rest != "" && rest[0] != '/') {
		return true
	}
	critOnce.Do(func() {
		critRe = make(map[string][]*regexp.Regexp, len(SafetyCriticalEnums))
		for _, t := range slices.Sorted(maps.Keys(SafetyCriticalEnums)) {
			for _, p := range SafetyCriticalEnums[t] {
				q := regexp.QuoteMeta(p)
				q = strings.ReplaceAll(q, `\*`, `[0-9]+`)
				critRe[t] = append(critRe[t], regexp.MustCompile("^"+q+"$"))
			}
		}
	})
	for _, re := range critRe[eventType] {
		if re.MatchString(rest) {
			return true
		}
	}
	return false
}
