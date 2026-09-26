package documents

import (
	"slices"
	"strings"

	"ant/internal/application/platform"
	dom "ant/internal/domain/documents"
)

// Запрос решения и карточка редкого подписанта по виду документа (FR-136,
// AD-43): тот же ответ, что live DecisionRequests / DecisionCard, но из уже
// собранного DocumentView — им пользуются заготовки (мир заготовок отдаёт
// документы с маршрутом тем же построителем domain/documents). Правило то же:
// версия ждёт подписей (не закрыта, не аннулирована, не возвращена),
// ближайший незакрытый этап вправе подписать пользователь сеанса.

// awaitingStage — первый незакрытый этап маршрута версии; nil — не ждёт подписей.
func awaitingStage(v DocumentView) *DocumentRouteStage {
	if v.Live || v.Status == dom.StatusRouteClosed || v.Status == dom.StatusAnnulled || v.Status == dom.StatusReturned || len(v.Declines) > 0 {
		return nil
	}
	for i := range v.Route {
		if !v.Route[i].Done {
			return &v.Route[i]
		}
	}
	return nil
}

// candidatesOf — кто может подписать этап n (замороженный набор, AD-43).
func candidatesOf(v DocumentView, n int) []string {
	for _, s := range v.Stages {
		if s.Stage == n {
			return s.Candidates
		}
	}
	return nil
}

// RequestFromView — карточка «требуется ваше решение» по документу v для
// person (пусто — без отбора по подписанту); itemIDs — изделия документа из
// реестра. ok=false — документ не ждёт подписей или не ждёт этого человека.
func RequestFromView(v DocumentView, itemIDs []string, person string) (DecisionRequest, bool) {
	st := awaitingStage(v)
	if st == nil {
		return DecisionRequest{}, false
	}
	cands := candidatesOf(v, st.Stage)
	mine := person == "" || slices.Contains(cands, person)
	if !mine {
		return DecisionRequest{}, false
	}
	status := "drafted"
	if len(v.Signatures) > 0 || slices.ContainsFunc(v.Route, func(s DocumentRouteStage) bool { return s.Done }) {
		status = "in_route"
	}
	drafted := v.Versions
	rq := DecisionRequest{
		Document: RoutedDocument{DocumentID: v.DocumentID, Version: v.Version, TemplateRef: v.Template, DocType: v.DocType, Title: v.Title, DocDigest: v.DocDigest,
			Status: status, Route: v.Route, BasisSeq: v.BasisSeq},
		Evidence: []DecisionEvidence{}, SimilarAccepted: []SimilarDecision{}, SimilarRejected: []SimilarDecision{},
	}
	for _, x := range drafted {
		if x.Version == v.Version {
			rq.Document.DraftedAt = x.DraftedAt
		}
	}
	if rq.Document.Route == nil {
		rq.Document.Route = []DocumentRouteStage{}
	}
	for _, sg := range v.Signatures {
		rq.Evidence = append(rq.Evidence, DecisionEvidence{EventID: sg.EventID, EventType: "document.signature.recorded", OccurredAt: sg.SignedAt})
	}
	n := st.Stage
	rq.MyStage = &n
	if len(cands) > 0 {
		c := cands[0]
		rq.ExpectedSigner = &c
	}
	rq.Proposal, rq.Escalation = proposalFromView(v, itemIDs, st)
	return rq, true
}

// proposalFromView — что предлагается и почему пришло к подписанту (FR-50, FR-136).
func proposalFromView(v DocumentView, itemIDs []string, st *DocumentRouteStage) (DecisionProposal, DecisionEscalation) {
	tpl, _, _ := strings.Cut(v.Template, "@")
	subject := v.SubjectLabel
	if subject == "" {
		subject = v.Subject.ID
	}
	p := DecisionProposal{Kind: "other", Code: tpl, Summary: v.Title}
	switch {
	case v.Subject.Entity == platform.EntityItem:
		p.ItemID, p.ItemLabel = v.Subject.ID, subject
	case len(itemIDs) > 0:
		p.ItemID, p.ItemLabel = itemIDs[0], strings.Join(itemIDs, ", ")
	default:
		p.ItemID, p.ItemLabel = v.Subject.ID, subject
	}
	if v.Subject.Entity == platform.EntityNonconformity {
		p.NCID, p.NCNumber = v.Subject.ID, subject
		if len(itemIDs) > 0 {
			p.ItemLabel = subject + " · " + strings.Join(itemIDs, ", ")
		}
	}
	e := DecisionEscalation{Reason: "Документ «" + v.Title + "» ждёт подписи этапа " + itoa(st.Stage) + " «" + st.AuthorityLabel + "» по маршруту шаблона " + v.Template,
		RuleID: "documents.route", RuleRev: v.Template}
	switch {
	case v.DocType == dom.DocNCDisposition:
		p.Kind = "disposition"
		e.AutomationMode = 4
	case tpl == "concession":
		p.Kind = "concession"
		e.AutomationMode = 4
	case tpl == "customer-presentation-notice" || tpl == "customer-conclusion":
		p.Kind = "presentation"
	}
	if st.ExternalParty == "customer_representative" || st.Role == "customer_representative" {
		e.AutomationMode = 5
		e.Reason += "; режим 5 — приёмка представителем заказчика"
	}
	return p, e
}

// CardFromView — карточка «требуется ваше решение» документа v для редкого
// подписанта (documents.decision_card.read): что решается, почему вы, кто ещё
// подписывает, что будет после подписи. ok=false — документ не ждёт подписей.
func CardFromView(v DocumentView, itemIDs []string, person string) (DecisionCard, bool) {
	rq, ok := RequestFromView(v, itemIDs, "")
	if !ok {
		return DecisionCard{}, false
	}
	card := DecisionCard{DocumentID: v.DocumentID, Version: v.Version, Title: v.Title, Question: rq.Proposal.Summary, WhyYou: rq.Escalation.Reason,
		Subject: v.Subject, Basis: []platform.DrillRef{v.Subject}, SummaryFields: v.SummaryFields, OtherSigners: []string{},
		AfterSignature: AfterSignatureText(v.DocType), DocDigest: v.DocDigest, BasisSeq: v.BasisSeq}
	if card.SummaryFields == nil {
		card.SummaryFields = []DocumentSummaryField{}
	}
	for _, id := range itemIDs {
		if id != v.Subject.ID {
			card.Basis = append(card.Basis, platform.DrillRef{Entity: platform.EntityItem, ID: id})
		}
	}
	my := *rq.MyStage
	if person != "" && !slices.Contains(candidatesOf(v, my), person) {
		card.WhyYou = "Этап " + itoa(my) + " ждёт другого подписанта; вы видите весь маршрут (AD-43)"
	}
	for _, s := range v.Stages {
		if s.Stage == my {
			card.Stage = s
			continue
		}
	}
	for _, st := range v.Route {
		if st.Stage != my {
			card.OtherSigners = append(card.OtherSigners, st.AuthorityLabel)
		}
	}
	return card, true
}

// AfterSignatureText — что произойдёт после подписи, по построителю документа.
func AfterSignatureText(docType string) string {
	switch docType {
	case dom.DocNCDisposition:
		return "Когда подпишут все этапы, маршрут закроется (document.route.closed) и решение по несоответствию исполнится (AD-43)"
	case dom.DocTraveler:
		return "Итоговая годность изделия зафиксирована подписями контролёра ОТК и мастера"
	}
	return "Когда подпишут все этапы, маршрут закроется (document.route.closed)"
}
