package analysis

import (
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Карта дефицита данных (FR-143, эпик 42): «нехватка сведений», которую уже
// собирает разбор обстоятельств (FR-58), агрегируется по всем
// расследованиям — сколько раз не хватало каждого вида сведений и где; для
// каждого вида — какая цифровизация закрыла бы пробел и насколько она сузила
// бы область риска. Оценка сужения — по уже сделанным людьми сужениям с
// основаниями: «со сведениями область сразу была бы такой, какой её сузили
// вручную по доказательствам». Нет таких сужений — оценки нет (не ноль).

// MissingKinds — виды недостающих сведений в порядке показа (перечисление контракта).
var MissingKinds = []string{MissingTool, MissingCycleEnd, MissingAfter, MissingBefore, MissingOperator, MissingEquipmentLog, MissingOther}

// PlaceCount — где не хватало сведений: оборудование или узел процесса.
type PlaceCount struct {
	Place string `json:"place"`
	Count int    `json:"count"`
}

// Narrowing — оценка сужения области риска: средний размер области до и
// после сужения (в десятых долях детали, фиксированная точка — AD-4).
type Narrowing struct {
	FromTenths int `json:"from_tenths"`
	ToTenths   int `json:"to_tenths"`
	// Incidents — сколько инцидентов легло в оценку.
	Incidents int `json:"incidents"`
}

// DeficitRow — строка карты дефицита: вид сведений.
type DeficitRow struct {
	Kind string `json:"kind"`
	// Investigations — в скольких расследованиях не хватало этих сведений.
	Investigations int `json:"investigations"`
	// Places — где (по убыванию, не больше пяти).
	Places []PlaceCount `json:"places"`
	// NCIDs — несоответствия (для раскрытия).
	NCIDs []string `json:"nc_ids"`
	// IncidentIDs — инциденты, связанные с этими расследованиями.
	IncidentIDs []string `json:"incident_ids"`
	// Digitization — какая цифровизация закрыла бы пробел.
	Digitization string `json:"digitization"`
	// Estimate — оценка сужения; nil — оценка невозможна.
	Estimate *Narrowing `json:"estimate,omitempty"`
	// Basis — записи, на которых построены выводы разборов (не больше 50).
	Basis []string `json:"basis"`
}

// DeficitMap — итог карты: всего расследований и строки по видам сведений
// (только виды, которых хоть раз не хватило, в порядке MissingKinds).
type DeficitMap struct {
	Investigations int          `json:"investigations"`
	Rows           []DeficitRow `json:"rows"`
}

// Digitization — какая цифровизация закрыла бы пробел вида сведений.
func Digitization(kind string) string {
	switch kind {
	case MissingTool:
		return "Учёт инструмента на станках (ввод при смене инструмента, метка на оправке)"
	case MissingCycleEnd:
		return "Сигнал конца цикла со станка (шлюз: шпиндель, дверь, программа)"
	case MissingAfter:
		return "Контроль сразу после операции (камера или замер на посту)"
	case MissingBefore:
		return "Контроль зоны до операции (входной или межоперационный)"
	case MissingOperator:
		return "Вход исполнителя на рабочее место (терминал, СКУД)"
	case MissingEquipmentLog:
		return "Журнал оборудования: датчики тока, вибрации, состояния"
	}
	return "Дополнительные сведения по разбору"
}

// BuildDeficitMap — карта дефицита данных по разборам несоответствий и
// инцидентам. items — изделие каждого несоответствия (nc_id → item_id): по
// нему несоответствие связывается с инцидентами, в чью область входило изделие.
func BuildDeficitMap(analyses []Analysis, incidents []IncidentRecord, items map[string]string) DeficitMap {
	out := DeficitMap{Investigations: len(analyses), Rows: []DeficitRow{}}
	type acc struct {
		places    map[string]int
		ncs       []string
		incidents []string
		basis     []string
	}
	by := map[string]*acc{}
	for _, a := range analyses {
		place := firstNonEmpty(a.Profile.Equipment, a.Profile.StepKey, "не установлено")
		for _, k := range sortedUnique(a.Missing) {
			x := by[k]
			if x == nil {
				x = &acc{places: map[string]int{}}
				by[k] = x
			}
			x.places[place]++
			x.ncs = appendUnique(x.ncs, a.NCID)
			x.basis = append(x.basis, a.Causes...)
			for _, v := range incidents {
				if v.InScope(items[a.NCID]) || memberEver(v, items[a.NCID]) {
					x.incidents = appendUnique(x.incidents, v.IncidentID)
				}
			}
		}
	}
	for _, k := range MissingKinds {
		x := by[k]
		if x == nil {
			continue
		}
		row := DeficitRow{Kind: k, Investigations: len(x.ncs), NCIDs: x.ncs, IncidentIDs: x.incidents, Digitization: Digitization(k), Places: []PlaceCount{}}
		for _, p := range slices.Sorted(maps.Keys(x.places)) {
			row.Places = append(row.Places, PlaceCount{Place: p, Count: x.places[p]})
		}
		slices.SortStableFunc(row.Places, func(a, b PlaceCount) int { return b.Count - a.Count })
		if len(row.Places) > 5 {
			row.Places = row.Places[:5]
		}
		row.Estimate = narrowing(incidents, x.incidents)
		row.Basis = sortedUnique(x.basis)
		if len(row.Basis) > 50 {
			row.Basis = row.Basis[:50]
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

// memberEver — изделие входило в какую-либо версию области (в том числе исключено позже).
func memberEver(v IncidentRecord, itemID string) bool {
	if itemID == "" {
		return false
	}
	if _, ok := v.Members[itemID]; ok {
		return true
	}
	for _, x := range v.Versions {
		if slices.Contains(x.Added, itemID) || slices.Contains(x.Removed, itemID) {
			return true
		}
	}
	return false
}

// narrowing — средний размер области до и после сужения по инцидентам ids,
// где сужение с основаниями было (текущий размер меньше исходного).
func narrowing(incidents []IncidentRecord, ids []string) *Narrowing {
	from, to, n := 0, 0, 0
	for _, v := range incidents {
		if !slices.Contains(ids, v.IncidentID) {
			continue
		}
		initial := v.InitialSize
		if initial == 0 && len(v.Versions) > 0 {
			initial = v.Versions[0].Size
		}
		narrowed := false
		for _, x := range v.Versions {
			narrowed = narrowed || x.Change == ChangeNarrowed
		}
		cur := v.Size()
		if !narrowed && excludedCount(v) == 0 {
			continue
		}
		if cur >= initial || initial == 0 {
			continue
		}
		from, to, n = from+initial, to+cur, n+1
	}
	if n == 0 {
		return nil
	}
	return &Narrowing{FromTenths: tenths(from, n), ToTenths: tenths(to, n), Incidents: n}
}

func excludedCount(v IncidentRecord) int {
	c := 0
	for _, k := range slices.Sorted(maps.Keys(v.Members)) {
		if v.Members[k].Status == StatusExcluded {
			c++
		}
	}
	return c
}

// tenths — среднее sum/n в десятых долях с округлением.
func tenths(sum, n int) int { return (sum*10*2 + n) / (2 * n) }

// TenthsText — «12,5» / «4» — среднее для людей.
func TenthsText(t int) string {
	if t%10 == 0 {
		return strconv.Itoa(t / 10)
	}
	return strconv.Itoa(t/10) + "," + strconv.Itoa(t%10)
}

// MissingText — вид недостающих сведений словами (как в разборе, эпик 12).
func MissingText(kind string) string {
	switch kind {
	case MissingTool:
		return "неизвестен инструмент"
	case MissingCycleEnd:
		return "неизвестно точное время конца цикла"
	case MissingAfter:
		return "нет наблюдения после операции"
	case MissingBefore:
		return "нет наблюдения до операции"
	case MissingOperator:
		return "неизвестен исполнитель"
	case MissingEquipmentLog:
		return "нет журнала оборудования"
	}
	return "не хватает других сведений"
}

// DeficitProposals — генератор «карта дефицита данных» (FR-143 + FR-63): для
// каждого вида сведений, которого не хватило хотя бы в двух расследованиях, —
// предложение цифровизации с оценкой сужения области риска. Ответственный —
// руководитель производства (планирование цифровизации).
func DeficitProposals(m DeficitMap) []Proposal {
	var out []Proposal
	for _, r := range m.Rows {
		if r.Investigations < 2 || r.Kind == MissingOther {
			continue
		}
		var places []string
		for _, p := range r.Places {
			places = append(places, p.Place)
		}
		where := strings.Join(places, ", ")
		st := "В " + strconv.Itoa(r.Investigations) + " расследованиях из " + strconv.Itoa(m.Investigations) + " " + MissingText(r.Kind) +
			" (" + where + "). Предлагается: " + lowerFirst(r.Digitization) + " — " + where + "."
		est := "Оценка сужения области риска невозможна — по этим расследованиям область ещё не сужали по основаниям"
		if r.Estimate != nil {
			est = "Сузило бы область риска в среднем с " + TenthsText(r.Estimate.FromTenths) + " до " + TenthsText(r.Estimate.ToTenths) +
				" дет. (по " + strconv.Itoa(r.Estimate.Incidents) + " инцидент(ам) с сужением по основаниям)"
		}
		out = append(out, Proposal{Generator: "rules.data_deficit", Kind: SuggestDataDeficit, MissingKind: r.Kind, ResponsibleRole: RoleProductionManager,
			Title:     "Цифровизация: " + MissingText(r.Kind) + " — " + strconv.Itoa(r.Investigations) + " из " + strconv.Itoa(m.Investigations),
			Statement: st, Estimate: est, DedupKey: "deficit|" + r.Kind + "|" + strconv.Itoa(r.Investigations), Basis: slices.Clone(r.Basis)})
	}
	return out
}

// lowerFirst — первая буква строчной (по рунам: текст русский).
func lowerFirst(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return strings.ToLower(string(r[0])) + string(r[1:])
}
