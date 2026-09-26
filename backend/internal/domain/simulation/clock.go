package simulation

import (
	"slices"
	"time"
)

// Clock — виртуальные часы прогона (AD-37, FR-106, FR-129): доменное
// «сейчас» сценария = виртуальный момент последней установки + скорость ×
// прошедшее реальное время (InfraClock). Пауза останавливает часы и поток.
// Реальное время приходит аргументом: домен часов не читает (AD-4).
type Clock struct {
	// VirtualAt — виртуальное время в момент RealAt.
	VirtualAt time.Time `json:"virtual_at"`
	// RealAt — реальное время (InfraClock) последней установки.
	RealAt time.Time `json:"real_at"`
	Speed  int       `json:"speed"`
	Paused bool      `json:"paused"`
}

// Now — виртуальное «сейчас» при реальном времени real.
func (c Clock) Now(real time.Time) time.Time {
	if c.Paused || real.Before(c.RealAt) {
		return c.VirtualAt
	}
	return c.VirtualAt.Add(real.Sub(c.RealAt) * time.Duration(max(c.Speed, 1)))
}

// Rebase — часы, переставленные на виртуальный момент v в реальный момент real.
func (c Clock) Rebase(real, v time.Time) Clock {
	c.VirtualAt, c.RealAt = v, real
	return c
}

// WithSpeed — новая скорость без скачка времени.
func (c Clock) WithSpeed(real time.Time, speed int) Clock {
	c = c.Rebase(real, c.Now(real))
	c.Speed = speed
	return c
}

// Pause — пауза: часы стоят на текущем виртуальном моменте.
func (c Clock) Pause(real time.Time) Clock {
	c = c.Rebase(real, c.Now(real))
	c.Paused = true
	return c
}

// Resume — продолжение с того же виртуального момента.
func (c Clock) Resume(real time.Time) Clock {
	c.RealAt = real
	c.Paused = false
	return c
}

// Point — момент проверки табло или снятия значения «до» (baseline).
type Point struct {
	At       time.Time
	Baseline bool
	// Index — номер точки в списке прогона (application).
	Index int
}

// Cursor — положение прогона: сколько событий доставлено, шагов исполнено,
// точек проверено. Продолжение после паузы — с того же курсора: без потерь
// и дублей (FR-129).
type Cursor struct {
	Emissions int `json:"emissions"`
	Actions   int `json:"actions"`
	Points    int `json:"points"`
}

// DueKind — что наступило.
type DueKind string

// Что наступает в прогоне.
const (
	DueEmissions DueKind = "emissions" // пачка событий источников
	DueAction    DueKind = "action"    // решение или служебный шаг
	DuePoint     DueKind = "point"     // проверка табло или значение «до»
	DueEnd       DueKind = "end"       // конец прогона
)

// Due — ближайшее, что наступает в прогоне.
type Due struct {
	Kind DueKind
	At   time.Time
	// From, To — диапазон событий [From, To) для DueEmissions.
	From, To int
	// Index — номер шага или точки.
	Index int
}

// NextDue — ближайшее после курсора: при равном времени сначала значения
// «до», затем события источников, затем шаги людей, затем проверки (проверка
// в момент T видит всё, что случилось в T).
func NextDue(p *Plan, points []Point, cur Cursor) Due {
	best := Due{Kind: DueEnd, At: p.End}
	rank := func(d Due) int {
		switch d.Kind {
		case DuePoint:
			if points[d.Index].Baseline {
				return 0
			}
			return 3
		case DueEmissions:
			return 1
		case DueAction:
			return 2
		}
		return 4
	}
	consider := func(d Due) {
		if d.At.Before(best.At) || (d.At.Equal(best.At) && rank(d) < rank(best)) {
			best = d
		}
	}
	if cur.Emissions < len(p.Emissions) {
		at := p.Emissions[cur.Emissions].DeliverAt
		to := cur.Emissions
		for to < len(p.Emissions) && p.Emissions[to].DeliverAt.Equal(at) {
			to++
		}
		consider(Due{Kind: DueEmissions, At: at, From: cur.Emissions, To: to})
	}
	if cur.Actions < len(p.Actions) {
		consider(Due{Kind: DueAction, At: p.Actions[cur.Actions].At, Index: cur.Actions})
	}
	if cur.Points < len(points) {
		consider(Due{Kind: DuePoint, At: points[cur.Points].At, Index: cur.Points})
	}
	return best
}

// SortPoints — точки по времени (значения «до» раньше проверок в тот же момент).
func SortPoints(ps []Point) {
	slices.SortStableFunc(ps, func(a, b Point) int {
		if c := a.At.Compare(b.At); c != 0 {
			return c
		}
		if a.Baseline != b.Baseline {
			if a.Baseline {
				return -1
			}
			return 1
		}
		return a.Index - b.Index
	})
}
