package analytics

import (
	"maps"
	"slices"
	"time"
)

// Счётчики узлов, ограничение линии и аномалии (FR-2, FR-3, FR-5; AD-21:
// считает сервер). Вход — строки вклада изделий (и простои оборудования),
// момент T и период; всё — чистые функции над строками.

// NodeCount — счётчики узла по step_key.
type NodeCount struct {
	Step string
	// Queue, InProgress, OpenNC — состояние в момент T.
	Queue, InProgress, OpenNC int
	// Passed, Defects — за период (физические дефекты, не наблюдения).
	Passed, Defects int
	// WaitSum, WaitMax — текущее ожидание изделий в очереди, с.
	WaitSum, WaitMax int64
	// Load — загрузка: в очереди + в работе.
	Load int
	// Sources — записи для раскрытия счётчиков узла (очередь и в работе).
	Sources []string
}

// Nodes — счётчики всех узлов, где есть строки, в порядке step_key.
func Nodes(rows []Row, from, t time.Time) []NodeCount {
	by := map[string]*NodeCount{}
	get := func(step string) *NodeCount {
		n := by[step]
		if n == nil {
			n = &NodeCount{Step: step}
			by[step] = n
		}
		return n
	}
	for _, r := range rows {
		if r.Dims.Step == "" {
			continue
		}
		switch r.Metric {
		case RowQueue:
			if r.ActiveAt(t) {
				n := get(r.Dims.Step)
				n.Queue++
				w := int64(t.Sub(r.At) / time.Second)
				n.WaitSum += w
				n.WaitMax = max(n.WaitMax, w)
				n.Sources = addUnique(n.Sources, r.Sources...)
			}
		case RowInProgress:
			if r.ActiveAt(t) {
				n := get(r.Dims.Step)
				n.InProgress++
				n.Sources = addUnique(n.Sources, r.Sources...)
			}
		case RowNCOpen:
			if r.ActiveAt(t) {
				get(r.Dims.Step).OpenNC++
			}
		case RowPassed:
			if r.In(from, t) {
				get(r.Dims.Step).Passed++
			}
		case RowDefectsDetected:
			if r.In(from, t) {
				get(r.Dims.Step).Defects++
			}
		}
	}
	out := make([]NodeCount, 0, len(by))
	for _, k := range slices.Sorted(maps.Keys(by)) {
		n := by[k]
		n.Load = n.Queue + n.InProgress
		out = append(out, *n)
	}
	return out
}

// Bottleneck — узел-ограничение линии (FR-5): наибольшее ожидание при
// наибольшей загрузке — наибольшее суммарное текущее ожидание в очереди
// (очередь × среднее ожидание); при равенстве — большая загрузка, затем
// step_key. ok=false — никто не ждёт: ограничение не выявлено.
func Bottleneck(nodes []NodeCount) (NodeCount, bool) {
	var best NodeCount
	found := false
	for _, n := range nodes {
		if n.Queue == 0 || n.WaitSum <= 0 {
			continue
		}
		if !found || n.WaitSum > best.WaitSum || (n.WaitSum == best.WaitSum && n.Load > best.Load) {
			best, found = n, true
		}
	}
	return best, found
}

// Norm — норма узла (FR-5, FR-12: из свойств элемента процесса; до модуля
// process — значения по умолчанию).
type Norm struct {
	// Queue — допустимая очередь, изделий.
	Queue int
	// Wait — допустимое ожидание изделия в очереди, с.
	Wait int64
	// Downtime — допустимый простой оборудования в рабочую смену, с.
	Downtime int64
	// Spike — выработка за час после простоя, от которой это «всплеск», изделий.
	Spike int
}

// DefaultNorm — норма узла по умолчанию: очередь 3, ожидание 30 мин, простой
// 15 мин, всплеск — 3 изделия за час после часа без выработки.
var DefaultNorm = Norm{Queue: 3, Wait: 30 * 60, Downtime: 15 * 60, Spike: 3}

// Anomaly — аномалия узла.
type Anomaly struct {
	Step string
	// Kind — queue_above_norm | wait_above_norm | downtime_over_threshold |
	// output_spike | defect_rate_out_of_control.
	Kind string
	// Limit — порог в единицах вида (изделия или секунды).
	Limit int64
	Unit  string
}

// Виды аномалий узла (контракт NodeAnomaly.kind).
const (
	AnomalyQueue    = "queue_above_norm"
	AnomalyWait     = "wait_above_norm"
	AnomalyDowntime = "downtime_over_threshold"
	AnomalySpike    = "output_spike"
	AnomalyDefects  = "defect_rate_out_of_control"
)

// Anomalies — аномалии узлов в момент t (FR-5): очередь или ожидание выше
// нормы узла; простой оборудования узла дольше порога; всплеск выработки
// после простоя; доля дефектов вне контрольных границ (последняя точка
// карты p узла). norm(step) — норма узла.
func Anomalies(nodes []NodeCount, rows, downtime []Row, from, t time.Time, bucket time.Duration, norm func(step string) Norm) []Anomaly {
	var out []Anomaly
	for _, n := range nodes {
		nm := norm(n.Step)
		if nm.Queue > 0 && n.Queue > nm.Queue {
			out = append(out, Anomaly{Step: n.Step, Kind: AnomalyQueue, Limit: int64(nm.Queue), Unit: UnitPcs})
		}
		if nm.Wait > 0 && n.WaitMax > nm.Wait {
			out = append(out, Anomaly{Step: n.Step, Kind: AnomalyWait, Limit: nm.Wait, Unit: UnitSec})
		}
		if nm.Spike > 0 && spike(rows, n.Step, t, nm.Spike) {
			out = append(out, Anomaly{Step: n.Step, Kind: AnomalySpike, Limit: int64(nm.Spike), Unit: UnitPcs})
		}
		if ch := StepPChart(rows, n.Step, from, t, bucket); len(ch.Points) > 0 && ch.Points[len(ch.Points)-1].Out {
			out = append(out, Anomaly{Step: n.Step, Kind: AnomalyDefects, Limit: ch.Upper, Unit: "bp"})
		}
	}
	// Простой: оборудование → узел его последнего выполнения (или узел остановки).
	eqStep := map[string]string{}
	eqAt := map[string]time.Time{}
	for _, r := range rows {
		if r.Metric == RowInProgress && r.Dims.Equipment != "" && !r.At.After(t) && !r.At.Before(eqAt[r.Dims.Equipment]) {
			eqStep[r.Dims.Equipment], eqAt[r.Dims.Equipment] = r.Dims.Step, r.At
		}
	}
	seen := map[string]bool{}
	for _, d := range downtime {
		if !d.ActiveAt(t) {
			continue
		}
		step := d.Dims.Step
		if step == "" {
			step = eqStep[d.Dims.Equipment]
		}
		nm := norm(step)
		if step == "" || seen[step] || nm.Downtime <= 0 || int64(t.Sub(d.At)/time.Second) <= nm.Downtime {
			continue
		}
		seen[step] = true
		out = append(out, Anomaly{Step: step, Kind: AnomalyDowntime, Limit: nm.Downtime, Unit: UnitSec})
	}
	slices.SortStableFunc(out, func(a, b Anomaly) int {
		if a.Step != b.Step {
			return cmpStr(a.Step, b.Step)
		}
		return cmpStr(a.Kind, b.Kind)
	})
	return out
}

// spike — всплеск выработки после простоя: за последний час до t узел
// пропустил не меньше limit изделий, а за час до того — ни одного.
func spike(rows []Row, step string, t time.Time, limit int) bool {
	last, prev := 0, 0
	for _, r := range rows {
		if r.Metric != RowPassed || r.Dims.Step != step || r.At.After(t) {
			continue
		}
		switch age := t.Sub(r.At); {
		case age < time.Hour:
			last++
		case age < 2*time.Hour:
			prev++
		}
	}
	return last >= limit && prev == 0
}
