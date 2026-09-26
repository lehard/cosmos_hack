package simulation

import (
	"slices"
	"time"
)

// noise — случайный шум фона по seed (FR-104, AD-26): дубли, опоздания,
// пропуски, расхождение часов, нарушения порядка. Главная история задаёт шум
// карточками (все доли 0) — числа кейса должны сходиться; прогоны сбоев
// каталога (F-…) включают доли и проверяют поведение приёма.
func (g *gen) noise() {
	n := g.b.Run.Noise
	if n.DuplicateBP == 0 && n.LateBP == 0 && n.LossBP == 0 && n.ReorderBP == 0 && n.ClockSkewSec == 0 {
		return
	}
	slices.SortStableFunc(g.drafts, func(a, b *draft) int {
		if c := a.at.Compare(b.at); c != 0 {
			return c
		}
		return a.order - b.order
	})
	rnd := NewRand(g.seed, "noise")
	skew := time.Duration(n.ClockSkewSec) * time.Second
	late := time.Duration(max(n.LateMin, 10)) * time.Minute
	var prevBySource = map[string]*draft{}
	for _, d := range g.drafts {
		if !d.background || (len(n.Sources) > 0 && !slices.Contains(n.Sources, d.source)) {
			continue
		}
		if skew != 0 {
			d.skew = skew
		}
		switch {
		case rnd.ChanceBP(n.LossBP):
			d.lost = true
		case rnd.ChanceBP(n.LateBP):
			d.deliver = d.at.Add(late + time.Duration(rnd.Between(0, 600))*time.Second)
		}
		if !d.lost && rnd.ChanceBP(n.DuplicateBP) {
			d.dups = append(d.dups, d.at.Add(time.Duration(rnd.Between(2, 120))*time.Second))
		}
		if prev, ok := prevBySource[d.source]; ok && !prev.lost && !d.lost && prev.deliver.IsZero() && d.deliver.IsZero() && rnd.ChanceBP(n.ReorderBP) {
			// нарушение порядка: предыдущая запись источника придержана и
			// доставлена после более поздней («конец» раньше «начала»)
			d.deliver = d.at.Add(500 * time.Millisecond)
			prev.deliver = d.at.Add(2 * time.Second)
			prev.reordered = true
		}
		prevBySource[d.source] = d
	}
}
