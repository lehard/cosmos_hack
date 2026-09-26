package analytics_test

import (
	"testing"
	"time"

	domain "ant/internal/domain/analytics"
)

// Карта p: всплеск доли дефектных в последней подгруппе — выход за границу.
func TestPChart(t *testing.T) {
	var pts []domain.ChartPoint
	var d []int64
	for i := range 8 {
		pts = append(pts, domain.ChartPoint{At: t0.Add(time.Duration(i) * time.Hour), N: 20})
		d = append(d, int64(i%2)*2)
	}
	pts = append(pts, domain.ChartPoint{At: t0.Add(9 * time.Hour), N: 20})
	d = append(d, 10)
	ch := domain.PChart(pts, d)
	if ch.Center <= 0 || ch.Upper <= ch.Center || !ch.Points[len(ch.Points)-1].Out || ch.Points[0].Out {
		t.Fatalf("карта p: центр %d, граница %d, точки %+v", ch.Center, ch.Upper, ch.Points)
	}
}

func TestXmRChart(t *testing.T) {
	vals := []int64{100, 102, 98, 101, 99, 100, 180}
	var pts []domain.ChartPoint
	for i, v := range vals {
		pts = append(pts, domain.ChartPoint{At: t0.Add(time.Duration(i) * time.Minute), Value: v})
	}
	ch := domain.XmRChart(pts)
	if !ch.Points[6].Out || ch.Points[1].Out {
		t.Fatalf("карта XmR: %+v", ch)
	}
}
