package analysis

import (
	"maps"
	"slices"
	"strings"
)

// Общие факторы группы несоответствий (FR-135) и похожие случаи (FR-60) —
// правила, без ИИ.

// FactorOrder — порядок факторов таблицы «сколько из N» (FR-135).
var FactorOrder = []string{FactorMachine, FactorTool, FactorFixture, FactorProgram, FactorPerformer, FactorMaterialBatch}

// FactorRow — строка таблицы общих факторов: самое частое значение (пусто —
// неизвестно), сколько несоответствий его разделяют и сколько разных
// значений встретилось.
type FactorRow struct {
	Factor   string `json:"factor"`
	Value    string `json:"value,omitempty"`
	Matches  int    `json:"matches"`
	Distinct int    `json:"distinct"`
}

// factorValue — значение фактора в профиле.
func factorValue(p Profile, f string) string {
	switch f {
	case FactorMachine:
		return p.Equipment
	case FactorTool:
		return p.Tool
	case FactorFixture:
		return p.Fixture
	case FactorProgram:
		return p.Program
	case FactorPerformer:
		return p.Performer
	case FactorMaterialBatch:
		return p.MaterialBatch
	}
	return ""
}

// CommonFactors — таблица «сколько из N» по группе несоответствий (FR-135).
// Фактор, общий для всех несоответствий группы, — первым (проверка FR-135);
// дальше — по числу совпадений, при равенстве — в порядке FactorOrder.
// Неизвестное значение совпадением не считается (FR-123: не додумывать);
// у входного дефекта факторов операции нет (FR-58).
func CommonFactors(ps []Profile) []FactorRow {
	n := len(ps)
	rows := make([]FactorRow, 0, len(FactorOrder))
	for _, f := range FactorOrder {
		counts := map[string]int{}
		for _, p := range ps {
			if v := factorValue(p, f); v != "" {
				counts[v]++
			}
		}
		row := FactorRow{Factor: f, Distinct: len(counts)}
		for _, v := range slices.Sorted(maps.Keys(counts)) {
			if counts[v] > row.Matches {
				row.Value, row.Matches = v, counts[v]
			}
		}
		rows = append(rows, row)
	}
	slices.SortStableFunc(rows, func(a, b FactorRow) int {
		aAll, bAll := n > 0 && a.Matches == n, n > 0 && b.Matches == n
		switch {
		case aAll && !bAll:
			return -1
		case bAll && !aAll:
			return 1
		}
		return b.Matches - a.Matches
	})
	return rows
}

// GroupKey — ключ группы несоответствий «вид дефекта × операция × оборудование».
func GroupKey(p Profile) string {
	part := func(s string) string {
		if s == "" {
			return "unknown"
		}
		return s
	}
	op := p.StepKey
	if p.Incoming {
		op = "incoming"
	}
	return part(p.DefectType) + "|" + part(op) + "|" + part(p.Equipment)
}

// SplitGroupKey — части ключа группы.
func SplitGroupKey(k string) (defect, operation, equipment string) {
	parts := strings.SplitN(k, "|", 3)
	for len(parts) < 3 {
		parts = append(parts, "")
	}
	return parts[0], parts[1], parts[2]
}

// SimilarCase — похожий прошлый случай и почему он похож.
type SimilarCase struct {
	Profile
	SameOperation bool `json:"same_operation"`
	SameEquipment bool `json:"same_equipment"`
}

// Similar — похожие прошлые случаи для несоответствия target (FR-60):
// тот же вид дефекта и та же операция или то же оборудование; не позже
// target. Порядок: и операция, и оборудование → только операция → только
// оборудование; внутри — сначала более поздние.
func Similar(target Profile, others []Profile) []SimilarCase {
	var out []SimilarCase
	if target.DefectType == "" {
		return out
	}
	for _, p := range others {
		if p.NCID == target.NCID || p.DefectType != target.DefectType || p.At.After(target.At) {
			continue
		}
		sc := SimilarCase{Profile: p,
			SameOperation: p.StepKey != "" && p.StepKey == target.StepKey && p.Incoming == target.Incoming,
			SameEquipment: p.Equipment != "" && p.Equipment == target.Equipment}
		if target.Incoming && p.Incoming {
			sc.SameOperation = true
		}
		if sc.SameOperation || sc.SameEquipment {
			out = append(out, sc)
		}
	}
	score := func(s SimilarCase) int {
		n := 0
		if s.SameOperation {
			n += 2
		}
		if s.SameEquipment {
			n++
		}
		return n
	}
	slices.SortStableFunc(out, func(a, b SimilarCase) int {
		if d := score(b) - score(a); d != 0 {
			return d
		}
		switch {
		case a.At.After(b.At):
			return -1
		case b.At.After(a.At):
			return 1
		}
		return strings.Compare(a.NCID, b.NCID)
	})
	return out
}
