package analytics

import (
	"maps"
	"slices"
	"time"
)

// Контрольные карты Шухарта (ГОСТ Р ИСО 7870-2, FR-5, FR-89) в целых числах
// (AD-4): доли — в базисных пунктах, корень — целочисленный.

// ChartPoint — точка карты.
type ChartPoint struct {
	At    time.Time
	Value int64
	// N — объём подгруппы (для карты p).
	N int64
	// Out — выход за контрольные границы или неслучайная структура
	// (8 точек подряд по одну сторону от центра).
	Out bool
	// Sources — записи подгруппы; Item — изделие точки (карта XmR).
	Sources []string
	Item    string
}

// Chart — контрольная карта: центр и границы в единицах точек.
type Chart struct {
	Center, Upper, Lower int64
	Points               []ChartPoint
}

// isqrt — целая часть квадратного корня.
func isqrt(n int64) int64 {
	if n <= 0 {
		return 0
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// PChart — карта p (доля дефектных) по подгруппам: n — объём, d — число с
// признаком дефекта. Центр p̄ = Σd/Σn; границы p̄ ± 3·√(p̄(1−p̄)/n̄) по
// среднему объёму подгруппы, в б. п.
func PChart(points []ChartPoint, defects []int64) Chart {
	var sn, sd int64
	for i, p := range points {
		sn += p.N
		sd += defects[i]
	}
	if sn == 0 || len(points) == 0 {
		return Chart{Points: points}
	}
	center := sd * 10000 / sn
	nbar := sn / int64(len(points))
	if nbar < 1 {
		nbar = 1
	}
	sigma := isqrt(center * (10000 - center) / nbar)
	ch := Chart{Center: center, Upper: min(10000, center+3*sigma), Lower: max(0, center-3*sigma)}
	for i := range points {
		p := &points[i]
		p.Value = defects[i] * 10000 / max(p.N, 1)
		p.Out = p.Value > ch.Upper || p.Value < ch.Lower
	}
	markRuns(points, center)
	ch.Points = points
	return ch
}

// XmRChart — карта индивидуальных значений: центр x̄, границы x̄ ± 2,66·MR̄.
func XmRChart(points []ChartPoint) Chart {
	if len(points) == 0 {
		return Chart{Points: points}
	}
	var sum, mr int64
	for i, p := range points {
		sum += p.Value
		if i > 0 {
			d := p.Value - points[i-1].Value
			if d < 0 {
				d = -d
			}
			mr += d
		}
	}
	center := sum / int64(len(points))
	var mrbar int64
	if len(points) > 1 {
		mrbar = mr / int64(len(points)-1)
	}
	spread := mrbar * 266 / 100
	ch := Chart{Center: center, Upper: center + spread, Lower: max(0, center-spread)}
	for i := range points {
		points[i].Out = len(points) > 1 && (points[i].Value > ch.Upper || points[i].Value < ch.Lower)
	}
	markRuns(points, center)
	ch.Points = points
	return ch
}

// markRuns — неслучайная структура: 8 точек подряд по одну сторону от центра.
func markRuns(points []ChartPoint, center int64) {
	run, side := 0, 0
	for i, p := range points {
		s := 0
		switch {
		case p.Value > center:
			s = 1
		case p.Value < center:
			s = -1
		}
		if s != 0 && s == side {
			run++
		} else {
			run, side = 1, s
		}
		if side != 0 && run >= 8 {
			for j := i - run + 1; j <= i; j++ {
				points[j].Out = true
			}
		}
	}
}

// StepPChart — карта p узла: доля результатов контроля с признаком дефекта
// по подгруппам-интервалам bucket в периоде [from, t].
func StepPChart(rows []Row, step string, from, t time.Time, bucket time.Duration) Chart {
	if bucket <= 0 {
		bucket = time.Hour
	}
	type sub struct {
		n, d int64
		src  []string
	}
	by := map[int64]*sub{}
	for _, r := range rows {
		if r.Dims.Step != step || !r.In(from, t) || (r.Metric != RowInspections && r.Metric != RowInspectionsWithDefect) {
			continue
		}
		k := int64(r.At.Sub(from) / bucket)
		s := by[k]
		if s == nil {
			s = &sub{}
			by[k] = s
		}
		if r.Metric == RowInspections {
			s.n += r.Value
			s.src = addUnique(s.src, r.Sources...)
		} else {
			s.d += r.Value
		}
	}
	var keys []int64
	for _, k := range slices.Sorted(maps.Keys(by)) {
		if by[k].n > 0 {
			keys = append(keys, k)
		}
	}
	points := make([]ChartPoint, 0, len(keys))
	defects := make([]int64, 0, len(keys))
	for _, k := range keys {
		s := by[k]
		points = append(points, ChartPoint{At: from.Add(time.Duration(k+1) * bucket), N: s.n, Sources: s.src})
		defects = append(defects, min(s.d, s.n))
	}
	return PChart(points, defects)
}

// StepDurationChart — карта XmR длительности операций узла по выполнениям.
func StepDurationChart(rows []Row, step string, from, t time.Time) Chart {
	var points []ChartPoint
	for _, r := range rows {
		if r.Metric == RowOperationDuration && r.Dims.Step == step && r.In(from, t) {
			points = append(points, ChartPoint{At: r.At, Value: r.Value, N: 1, Sources: r.Sources, Item: r.Item})
		}
	}
	slices.SortStableFunc(points, func(a, b ChartPoint) int { return a.At.Compare(b.At) })
	return XmRChart(points)
}
