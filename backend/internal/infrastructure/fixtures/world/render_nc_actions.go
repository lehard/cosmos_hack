package world

import (
	"maps"
	"slices"
	"strconv"

	ncapp "ant/internal/application/nonconformity"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	ncdom "ant/internal/domain/nonconformity"
)

// Решения карточки с последствиями и исполнение решения по изделию в
// заготовках (интерфейс 7) — теми же текстами, что у live (DecisionTexts,
// HandoffTitle): заготовки показывают то же, что стенд.

// decisionFacts — факты для текстов решений карточки из мира.
func (c *Ctx) decisionFacts(n *NC, it *Item, run *OpRun, card *ncapp.NCCard) ncapp.DecisionFacts {
	f := ncapp.DecisionFacts{NCNumber: n.Number, ItemLabel: card.ItemLabel, DecisionDays: 3, Approvals: ncdom.ApprovalsDemoStub,
		Commission: card.Commission, Containment: "none"}
	if it != nil {
		st := c.S(it)
		if name := c.nodeName(st.Step); name != nil {
			f.Where = "«" + *name + "»"
		}
		f.Containment = first(st.Containment, "none")
		f.Isolated = st.Containment == "item_hold"
		f.Incidents = slices.Sorted(maps.Keys(st.Incidents))
	}
	if run != nil {
		f.Operation, f.RunLabel = derefOr(c.nodeName(run.StepKey)), run.Label
		if nd := c.M.Bpmn[run.StepKey]; nd != nil {
			if l, err := strconv.Atoi(nd.Props["reworkLimit"]); err == nil {
				f.ReworkLimit = l
			}
		}
	}
	return f
}

// decisionActions — to_decide.actions: решения списка decisions с
// последствиями; ремонт и «как есть» без действующего разрешения и возврат
// обработанного изделия — недоступны с отказом гарда словами.
func (c *Ctx) decisionActions(n *NC, it *Item, run *OpRun, card *ncapp.NCCard) {
	f := c.decisionFacts(n, it, run, card)
	for _, op := range card.ToDecide.Decisions {
		if op != ncdom.ActDisposition {
			a := ncapp.NCDecisionAction{Operation: op, Allowed: true}
			a.Label, a.WhyAvailable, a.Consequences, a.TechnicalConsequences = ncapp.DecisionTexts(op, n.Disposition(c.M), f)
			card.ToDecide.Actions = append(card.ToDecide.Actions, a)
			continue
		}
		policy := ptr("полномочие «" + ncapp.AuthorityNCDisposition + "» (решение по несоответствию, рамки по тяжести)")
		for _, d := range ncapp.Dispositions {
			a := ncapp.NCDecisionAction{Operation: op, Disposition: ptr(d), Allowed: true, PolicyRef: policy}
			a.Label, a.WhyAvailable, a.Consequences, a.TechnicalConsequences = ncapp.DecisionTexts(op, d, f)
			switch d {
			case "repair", "use_as_is":
				a.Allowed, a.WhyAvailable = false, ncapp.RefusalText(kernel.Refuse(errcodes.NonconformityConcessionRequired, "decision", ncdom.DispositionLabel(d)))
			case "return_to_supplier":
				a.Allowed, a.WhyAvailable = false, ncapp.RefusalText(kernel.Refuse(errcodes.NonconformityReturnOnlyUnprocessed, "item_id", card.ItemID))
			}
			card.ToDecide.Actions = append(card.ToDecide.Actions, a)
		}
	}
}

// handoff — кому передано исполнение решения по изделию: мастеру участка;
// выполнение-переделка после решения — «исполняется» / «исполнено»,
// подтверждение исполнения — «исполнено».
func (c *Ctx) handoff(n *NC, it *Item, run *OpRun, card *ncapp.NCCard) *ncapp.NCHandoff {
	at := n.DispositionAt(c.M)
	if at == nil || at.After(c.T) {
		return nil
	}
	h := &ncapp.NCHandoff{RoleID: "site_foreman", Status: "waiting", Since: *at}
	// Решение — своё или группового несоответствия окна (FR-151).
	owners := []string{n.ID}
	if g := c.M.groupFor(n); g != nil {
		owners = append(owners, g.ID)
	}
	for _, e := range c.M.Events {
		if e.Step <= c.N && e.Type == "decision.disposition.set" && slices.Contains(owners, e.Entity.ID) {
			h.DecisionEventID = e.ID
		}
	}
	kind := n.Disposition(c.M)
	h.TaskTitle = ncapp.HandoffTitle(kind, c.decisionFacts(n, it, run, card))
	if it != nil && (kind == "rework" || kind == "repair") {
		for _, r := range it.Runs {
			if r.ReworkOf == "" || r.From.Before(*at) || r.From.After(c.T) {
				continue
			}
			h.Status, h.Since = "in_progress", r.From
			if r.Performer != "" {
				h.Person = ptr(r.Performer)
			}
			if !r.To.IsZero() && !r.To.After(c.T) {
				h.Status, h.Since = "done", r.To
			}
		}
	}
	if !n.VerifiedAt.IsZero() && !n.VerifiedAt.After(c.T) && h.Status != "done" {
		h.Status, h.Since = "done", n.VerifiedAt
	}
	h.RoleLabel, h.StatusLabel = ncapp.RoleLabel(h.RoleID), ncapp.HandoffStatusLabel(h.Status)
	return h
}
