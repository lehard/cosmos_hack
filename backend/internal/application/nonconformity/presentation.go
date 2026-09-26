package nonconformity

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/nonconformity"
	dp "ant/internal/domain/process"
)

// Presentation — точка предъявления изделия для решения контролёра
// (nonconformity.presentation.read; FR-19, FR-56): ждущее решения
// предъявление, а если его нет — пересмотр решения, принятого до новых
// данных (AD-3). Допустимые решения — те, что пройдут гарды команды
// nonconformity.presentation.resolve для вошедшего.
func (s *Service) Presentation(ctx context.Context, itemID string, m platform.Moment) (NCPresentationView, error) {
	if !s.live() {
		return s.Unimplemented.Presentation(ctx, itemID, m)
	}
	v, err := s.loadItem(ctx, itemID, m)
	if err != nil {
		return NCPresentationView{}, err
	}
	st := v.State()
	out := NCPresentationView{ItemID: v.ItemID, ItemLabel: v.ItemID, MethodResults: []NCRecordRef{}, BasisSeq: v.BasisSeq}
	var p NCPresentationPoint
	reviews := s.reviewsOf(v)
	switch pr := st.PendingPresentation(); {
	case pr != nil:
		p = NCPresentationPoint{EventID: pr.EventID, StepKey: pr.StepKey, ClosingPoint: pr.ClosingPoint, PresentationNo: pr.PresentationNo,
			MethodEventIDs: slices.Clone(st.Inspections)}
	case len(reviews) > 0:
		rv := reviews[len(reviews)-1]
		var d dom.PresentationResolvedData
		_ = json.Unmarshal(rv.decision.Data, &d)
		p = NCPresentationPoint{StepKey: d.StepKey, ClosingPoint: d.ClosingPoint, PresentationNo: max(d.PresentationNo, 1), MethodEventIDs: slices.Clone(d.MethodEventIDs)}
		for _, x := range st.Presentations {
			if x.StepKey == d.StepKey && x.PresentationNo == d.PresentationNo {
				p.EventID = x.EventID
			}
		}
		review := &NCPresentationReview{Decision: ref(rv.decision, summaryOf(rv.decision)), KnownAtDecision: []NCRecordRef{}, NewFacts: []NCRecordRef{}}
		for _, id := range d.MethodEventIDs {
			if r, ok := v.Record(id); ok {
				review.KnownAtDecision = append(review.KnownAtDecision, ref(r, summaryOf(r)))
			}
		}
		for _, r := range rv.facts {
			review.NewFacts = append(review.NewFacts, ref(r, summaryOf(r)))
		}
		out.Review = review
	default:
		e := platform.Fail(errcodes.ApiNotFound, "object", "Предъявление", "id", itemID)
		e.Detail = "Изделие " + itemID + " не ждёт решения на точке предъявления"
		return NCPresentationView{}, e
	}
	if p.MethodEventIDs == nil {
		p.MethodEventIDs = []string{}
	}
	for _, id := range p.MethodEventIDs {
		if r, ok := v.Record(id); ok {
			out.MethodResults = append(out.MethodResults, ref(r, summaryOf(r)))
		}
	}
	if name := stepName(v, p.StepKey); name != "" {
		p.StepLabel = &name
	}
	if def := v.Env.Process.Def; def != nil {
		p.ClosingPointLabel = closingPointLabel(def, p.ClosingPoint)
		p.NextStepLabel = nextStepLabel(def, p.StepKey)
	}
	p.AllowedResolutions = s.allowedResolutions(ctx, v, p)
	out.Presentation = p
	return out, nil
}

// allowedResolutions — решения, которые пройдут гарды команды для вошедшего:
// доменный гард (разделение обязанностей, блок, результаты методов,
// вмешательство), полномочие точки; «принять по разрешению» — только при
// действующем разрешении на отклонение в области изделия.
func (s *Service) allowedResolutions(ctx context.Context, v *itemView, p NCPresentationPoint) []string {
	actor := platform.PrincipalFrom(ctx).PersonID
	now := s.at(ctx, platform.Moment{}, v.RunID)
	concession := false
	if book, err := s.concessionBook(ctx, platform.Moment{}); err == nil {
		for _, c := range book.Sorted() {
			if c.InScope(v.ItemID) && dom.ConcessionGuard(&c, c.GrantedEventID, v.ItemID, "", now) == nil {
				concession = true
			}
		}
	}
	out := []string{}
	if s.gateAuthority(v, actor, p.StepKey) != nil {
		return out
	}
	stream := "item:" + v.ItemID
	for _, r := range []string{"accept", "accept_with_concession", "reject", "insufficient_data"} {
		if r == "accept_with_concession" && !concession {
			continue
		}
		data := dom.PresentationResolvedData{StepKey: p.StepKey, ClosingPoint: p.ClosingPoint, Resolution: r, PresentationNo: p.PresentationNo,
			MethodEventIDs: p.MethodEventIDs}
		cmd := kernel.Command{Action: dom.ActPresentation, Actor: actor, Object: stream, BasisSeq: v.BasisSeq, GuardStreams: []string{stream},
			OccurredAt: now, SignatureLevel: 2, Payload: data}
		if dom.Guard(v.State(), v.Env, v.Upstream(), cmd) == nil {
			out = append(out, r)
		}
	}
	return out
}

// passBranch — условие ветки «годно» на развилке после точки: решение
// «принять» (decision == 'accept') или испытание «герметично» (test.result == 'tight').
func passBranch(cond string) bool {
	return strings.Contains(cond, "'accept'") || strings.Contains(cond, "'tight'")
}

// closingPointLabel — имя узла процесса с закрывающей точкой cp; нет — nil.
func closingPointLabel(def *dp.Definition, cp string) *string {
	if cp == "" {
		return nil
	}
	ids := make([]string, 0, len(def.Nodes))
	for id := range def.Nodes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		if n := def.Nodes[id]; n != nil && n.Props.ClosingPoint == cp && n.Name != "" {
			name := n.Name
			return &name
		}
	}
	return nil
}

// nextStepLabel — имя следующего шага после stepKey при «Принять»: первый по
// стрелкам узел с именем, не шлюз и не событие; на развилке — по ветке с
// условием «годно» (passBranch), если она есть; нет — nil.
func nextStepLabel(def *dp.Definition, stepKey string) *string {
	n := def.ByStep(stepKey)
	if n == nil {
		return nil
	}
	seen := map[string]bool{n.ID: true}
	next := func(x *dp.Node) []string {
		for _, f := range x.Out {
			if fl := def.Flows[f]; passBranch(fl.CondText) {
				return []string{fl.Target}
			}
		}
		return def.Next(x)
	}
	queue := next(n)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		x := def.Node(id)
		if x == nil {
			continue
		}
		switch x.Type {
		case dp.NodeTask, dp.NodeUserTask, dp.NodeServiceTask, dp.NodeCallActivity, dp.NodeSubProcess:
			if x.Name != "" {
				name := x.Name
				return &name
			}
		}
		queue = append(queue, next(x)...)
	}
	return nil
}
