package reference

import (
	"slices"
	"strings"
)

// Соответствия внешних ID (FR-95, AD-18): внешний ID не становится нашим
// ключом — хранится соответствие отдельной записью reference.external_id.mapped.
// Конфликт — сигнал, а не перезапись: первое соответствие ключа (система,
// вид объекта, внешний ID) остаётся действующим, иное позднее соответствие
// того же ключа видно рядом с пометкой «конфликт», обе записи помечаются.

// MappingView — соответствие в срезе с признаком конфликта.
type MappingView struct {
	Mapping
	// Conflict — у ключа есть соответствия разным внутренним ID.
	Conflict bool
	// Effective — это соответствие действует (первое по seq для ключа).
	Effective bool
}

func mappingKey(m Mapping) string {
	return string(m.Data.System) + "|" + string(m.Data.ObjectKind) + "|" + m.Data.ExternalID
}

// MappingsView — соответствия среза: по ключу — различные внутренние ID в
// порядке seq (повтор того же соответствия — не конфликт и не новая строка).
// system ≠ "" — только эта система.
func (b Book) MappingsView(system string) []MappingView {
	groups := map[string][]Mapping{}
	var keys []string
	ms := slices.Clone(b.Mappings)
	slices.SortStableFunc(ms, func(a, c Mapping) int { return cmpInt(a.Seq, c.Seq) })
	for _, m := range ms {
		if system != "" && string(m.Data.System) != system {
			continue
		}
		k := mappingKey(m)
		g := groups[k]
		if slices.ContainsFunc(g, func(x Mapping) bool { return x.Data.InternalID == m.Data.InternalID }) {
			continue
		}
		if len(g) == 0 {
			keys = append(keys, k)
		}
		groups[k] = append(g, m)
	}
	slices.SortFunc(keys, strings.Compare)
	var out []MappingView
	for _, k := range keys {
		g := groups[k]
		for i, m := range g {
			out = append(out, MappingView{Mapping: m, Conflict: len(g) > 1, Effective: i == 0})
		}
	}
	return out
}

// Resolve — внутренний ID по внешнему (FR-95): действующее соответствие и
// признак конфликта. ok=false — соответствия нет.
func (b Book) Resolve(system, objectKind, externalID string) (internalID string, conflict bool, ok bool) {
	for _, v := range b.MappingsView(system) {
		if string(v.Data.ObjectKind) == objectKind && v.Data.ExternalID == externalID && v.Effective {
			return string(v.Data.InternalID), v.Conflict, true
		}
	}
	return "", false, false
}

// Conflicts — ключи соответствий с конфликтом (сигнал администратору данных).
func (b Book) Conflicts() []MappingView {
	var out []MappingView
	for _, v := range b.MappingsView("") {
		if v.Conflict {
			out = append(out, v)
		}
	}
	return out
}
