package documents

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/access"
	dom "ant/internal/domain/documents"
)

// Чтение документов (AD-12, AD-22): всё — из журнала той же свёрткой, что
// у воркера; проекций модуль не читает.

// Documents — документы объекта (documents.document.list, FR-65) и счётчик
// «документов собрано из истории — вручную не понадобилось».
func (s *Service) Documents(ctx context.Context, subject platform.DrillRef, m platform.Moment, p platform.Page) (DocumentList, error) {
	if !s.live() {
		return s.Unimplemented.Documents(ctx, subject, m, p)
	}
	out := DocumentList{Items: []DocumentSummary{}}
	if subject.Entity == platform.EntityItem {
		v, err := s.loadItem(ctx, subject.ID, m)
		if err != nil {
			return out, err
		}
		docs := slices.Clone(v.State.Docs)
		if v.State.Doc(dom.TravelerID(subject.ID)) == nil {
			if t, ok := v.State.Traveler(v.Env); ok && len(v.State.Rows) > 0 {
				docs = append([]dom.Doc{t}, docs...)
			}
		}
		for i := range docs {
			out.Items = append(out.Items, summary(&docs[i], v))
		}
	} else {
		ref := string(subject.Entity) + ":" + subject.ID
		found, err := s.bySubject(ctx, ref, m)
		if err != nil {
			return out, err
		}
		for _, f := range found {
			out.Items = append(out.Items, summary(f.doc, f.view))
		}
	}
	n, zero := len(out.Items), 0
	out.CollectedFromHistory, out.ManualEntries = &n, &zero
	return pageOf(out, p), nil
}

func pageOf(l DocumentList, p platform.Page) DocumentList {
	from, _ := strconv.Atoi(p.Cursor)
	limit := p.Limit
	if limit <= 0 {
		limit = 100
	}
	from = min(from, len(l.Items))
	to := min(from+limit, len(l.Items))
	res := l
	res.Items = l.Items[from:to]
	res.NextCursor = ""
	if to < len(l.Items) {
		res.NextCursor = strconv.Itoa(to)
	}
	return res
}

// found — документ и его состояние.
type found struct {
	doc  *dom.Doc
	view *view
}

// bySubject — документы объекта вне изделия-субъекта: по записям версий и
// запросов (subject_ref) — документы изделий и собственные потоки.
func (s *Service) bySubject(ctx context.Context, ref string, m platform.Moment) ([]found, error) {
	ids, err := s.documentIDs(ctx, m)
	if err != nil {
		return nil, err
	}
	var out []found
	cache := map[string]*view{}
	for _, id := range ids {
		v, d, err := s.cached(ctx, cache, id, m)
		if err != nil || d == nil || d.Subject != ref {
			continue
		}
		out = append(out, found{doc: d, view: v})
	}
	return out, nil
}

// docRef — документ из записей журнала: id и изделие.
type docRef struct {
	ID     string
	ItemID string
}

// documentIDs — документы, у которых есть версии или запросы, в порядке журнала.
func (s *Service) documentIDs(ctx context.Context, m platform.Moment) ([]docRef, error) {
	var out []docRef
	seen := map[string]bool{}
	for _, t := range []catalog.Type{catalog.DocumentVersionDrafted, catalog.DocumentVersionRequested} {
		es, err := s.readAll(ctx, appjournal.ReadQuery{EventType: string(t), RunID: m.RunID, Moment: m})
		if err != nil {
			return nil, err
		}
		for _, e := range es {
			id, ok := strings.CutPrefix(e.Stream, "document:")
			if !ok || seen[id] {
				continue
			}
			seen[id] = true
			r := docRef{ID: id}
			if e.ItemID != nil {
				r.ItemID = *e.ItemID
			}
			out = append(out, r)
		}
	}
	return out, nil
}

// cached — документ по ссылке; свёртка изделия — один раз на изделие.
func (s *Service) cached(ctx context.Context, cache map[string]*view, r docRef, m platform.Moment) (*view, *dom.Doc, error) {
	key := "stream:" + r.ID
	if r.ItemID != "" {
		key = "item:" + r.ItemID
	}
	v, ok := cache[key]
	if !ok {
		var err error
		if r.ItemID != "" {
			v, err = s.loadItem(ctx, r.ItemID, m)
		} else {
			v, err = s.loadStream(ctx, r.ID, m)
		}
		if err != nil {
			return nil, nil, err
		}
		cache[key] = v
	}
	return v, v.State.Doc(r.ID), nil
}

// summary — документ в списке объекта.
func summary(d *dom.Doc, v *view) DocumentSummary {
	out := DocumentSummary{DocumentID: d.ID, Template: d.TemplateRef, Title: d.Title, Subject: drill(d.Subject), Status: d.Status(),
		DocType: d.DocType, Class: d.Class, Versions: len(d.Versions)}
	if cur := d.Current(); cur != nil {
		out.Version, out.DocDigest = cur.No, cur.Digest
		t := cur.At
		out.DraftedAt = &t
		if cur.Closed {
			c := cur.ClosedAt
			out.ClosedAt = &c
		}
		out.PaperStatus = d.PaperStatus(cur.No)
	} else if pv, _, _, err := v.State.Preview(v.Env, d); err == nil {
		out.Version, out.DocDigest = pv.No, pv.Digest
	}
	return out
}

// drill — поток объекта как ссылка для перехода.
func drill(stream string) platform.DrillRef {
	k, id, _ := strings.Cut(stream, ":")
	return platform.DrillRef{Entity: platform.EntityKind(k), ID: id}
}

// Document — документ для подписи (documents.document.read, AD-12, AD-43).
func (s *Service) Document(ctx context.Context, documentID string, version int, m platform.Moment) (DocumentView, error) {
	if !s.live() {
		return s.Unimplemented.Document(ctx, documentID, version, m)
	}
	v, d, err := s.load(ctx, documentID, m)
	if err != nil {
		return DocumentView{}, err
	}
	ver, built, live, err := v.version(d, version)
	if err != nil {
		return DocumentView{}, err
	}
	var content map[string]any
	dec := json.NewDecoder(bytes.NewReader(built.Content))
	dec.UseNumber()
	if err := dec.Decode(&content); err != nil {
		return DocumentView{}, err
	}
	out := DocumentView{DocumentID: d.ID, Version: ver.No, Template: d.TemplateRef, DocFormatVersion: dom.DocFormatVersion, Title: d.Title,
		Subject: drill(d.Subject), Status: d.Status(), Content: content, RenderingHash: built.RenderingHash, DocDigest: built.Digest,
		SummaryFields: summaryFields(built.Content), SourceEventIDs: []string{}, Stages: []DocumentApprovalStage{}, Signatures: []DocumentSignatureView{},
		QR: dom.QR(d.ID, built.Digest), BasisSeq: v.BasisSeq, DocType: d.DocType, Class: d.Class, Live: live, Verification: ver.Verification,
		SigningPayloadB64: signingPayload(built.Content, built.RenderingHash, d.TemplateRef, built.Digest)}
	if !live && ver.No != d.Current().No {
		out.Status = versionStatus(ver)
	}
	for _, r := range ver.Sources {
		out.SourceEventIDs = append(out.SourceEventIDs, r.ID)
	}
	if ver.Supersedes > 0 {
		sup := ver.Supersedes
		out.SupersedesVersion = &sup
	}
	route := routeView(d, ver, v)
	out.Route = route
	for _, st := range route {
		das := DocumentApprovalStage{Stage: st.Stage, Authority: st.AuthorityID, StampKind: st.StampKind, Quorum: st.Quorum, Required: st.Required,
			Level: st.SignatureLevel, PaperAllowed: st.PaperAllowed, ExternalParty: st.ExternalParty, Candidates: candidates(st, v), SignedBy: []string{},
			Status: "pending"}
		for _, sg := range st.Signatures {
			if sg.Counted {
				das.SignedBy = append(das.SignedBy, sg.SignerID)
			}
			out.Signatures = append(out.Signatures, DocumentSignatureView{EventID: sg.EventID, Stage: st.Stage, Method: sg.Method, SignerPersonID: sg.SignerID,
				AttestedBy: sg.AttestedBy, Level: sg.Level, SignedAt: sg.SignedAt, DocDigest: sigDigest(d, sg.EventID, ver), CurrentVersion: !sg.PreviousVersion,
				Verification: verification(sg), ProvenanceClass: provenanceClass(sg.Class)})
		}
		switch {
		case st.Done:
			das.Status = "done"
		case len(das.SignedBy) > 0:
			das.Status = "in_progress"
		}
		if slices.Contains(st.Separation, access.SeparationNotItemParticipant) {
			das.SeparationNote = "Подписывает не участвовавший в изготовлении изделия (FR-56)"
		} else if slices.Contains(st.Separation, access.SeparationDistinctSigners) {
			das.SeparationNote = "Этапы подписывают разные люди (AD-43)"
		}
		out.Stages = append(out.Stages, das)
	}
	for _, x := range d.Versions {
		out.Versions = append(out.Versions, DocumentVersionRef{Version: x.No, DocDigest: x.Digest, Status: versionStatus(&x), DraftedAt: x.At, Signatures: len(x.Signatures)})
	}
	for _, p := range d.Paper {
		out.Paper = append(out.Paper, DocumentPaperMark{EventID: p.EventID, Version: p.Version, Status: p.Status, CopyNo: p.CopyNo, At: p.At})
	}
	out.SubjectLabel = out.Subject.ID
	for _, dc := range ver.Declines {
		out.Declines = append(out.Declines, DocumentDecline{EventID: dc.EventID, Version: ver.No, Stage: dc.Stage, SignerID: dc.Person, Comment: dc.Comment, At: dc.At})
	}
	if ver.Closed {
		for _, r := range v.Reactions {
			if r.Type == catalog.DocumentRouteClosed && r.Slot == dom.Slot(dom.RuleRoute, d.ID, ver.No) {
				id := r.EventID
				out.RouteClosed = &id
			}
		}
	}
	return out, nil
}

func versionStatus(v *dom.Version) string {
	switch {
	case v.Annulled:
		return dom.StatusAnnulled
	case v.Closed:
		return dom.StatusRouteClosed
	case len(v.Declines) > 0:
		return dom.StatusReturned
	case len(v.Signatures) > 0:
		return dom.StatusSigning
	}
	return dom.StatusDrafted
}

func verification(sg DocumentStageSignature) string {
	if sg.Check == "valid" {
		return "valid"
	}
	if sg.Check == "rejected" {
		return "invalid"
	}
	return "pending"
}

func provenanceClass(c string) string {
	if c == "" {
		return "personal"
	}
	return c
}

func sigDigest(d *dom.Doc, eventID string, cur *dom.Version) string {
	for _, v := range d.Versions {
		for _, sg := range v.Signatures {
			if sg.EventID == eventID {
				return sg.Digest
			}
		}
	}
	return cur.Digest
}

// summaryFields — поля сводки уровня 2 из content (AD-12).
func summaryFields(content json.RawMessage) []DocumentSummaryField {
	var c struct {
		Summary []struct {
			Label string `json:"label"`
			Value string `json:"value"`
		} `json:"summary"`
	}
	out := []DocumentSummaryField{}
	if json.Unmarshal(content, &c) != nil {
		return out
	}
	for i, f := range c.Summary {
		out = append(out, DocumentSummaryField{Key: "f" + strconv.Itoa(i+1), Label: f.Label, Value: f.Value})
	}
	return out
}

// candidates — кто по политике может подписать этап.
func candidates(st DocumentRouteStage, v *view) []string {
	out := []string{}
	if st.BySource {
		return out
	}
	for _, p := range v.Env.People {
		excluded := slices.Contains(st.Separation, access.SeparationNotItemParticipant) && slices.Contains(v.State.Participants, p.ID)
		if canSign(p, st.Role, st.AuthorityID) && !excluded {
			out = append(out, p.ID)
		}
	}
	return out
}

func canSign(p dom.Person, role, authority string) bool {
	return (role == "" || p.HasRole(role)) && (p.HasAuthority(authority) || p.HasRole(authority))
}

// routeView — маршрут версии с подписями: засчитанные, не засчитанные
// (почему) и подписи прежних версий (видны, не засчитываются, AD-43).
func routeView(d *dom.Doc, ver *dom.Version, v *view) []DocumentRouteStage {
	ev := dom.Evaluate(ver, v.Env.People, v.State.Participants)
	why := map[string]string{}
	for _, x := range ev.Verdicts {
		why[x.EventID] = x.Why
	}
	out := []DocumentRouteStage{}
	for i, st := range ver.Stages {
		rs := DocumentRouteStage{Stage: st.Stage, Title: st.Title, Role: st.Role, AuthorityID: st.AuthorityID, AuthorityLabel: st.Title, StampKind: st.StampKind,
			Quorum: st.Quorum, K: st.K, Required: st.Required, SignatureLevel: st.SignatureLevel, PaperAllowed: st.PaperAllowed,
			AttesterAuthorityID: st.AttesterAuthorityID, ExternalParty: st.ExternalParty, BySource: st.BySource, Separation: st.Separation,
			Signatures: []DocumentStageSignature{}}
		if rs.AuthorityLabel == "" {
			rs.AuthorityLabel = st.AuthorityID
		}
		if i < len(ev.Stages) {
			rs.Done = ev.Stages[i].Done
			for _, sg := range ev.Stages[i].Counted {
				if sg.Method == dom.MethodSource {
					rs.Signatures = append(rs.Signatures, stageSig(sg, true, false, "", v))
				}
			}
		}
		for _, x := range d.Versions {
			for _, sg := range x.Signatures {
				if sg.Stage != st.Stage {
					continue
				}
				prev := x.No != ver.No
				w := why[sg.EventID]
				counted := !prev && w == ""
				if prev {
					w = dom.WhyPriorVersion
				}
				rs.Signatures = append(rs.Signatures, stageSig(sg, counted, prev, w, v))
			}
		}
		out = append(out, rs)
	}
	return out
}

func stageSig(sg dom.Signature, counted, prev bool, why string, v *view) DocumentStageSignature {
	class := sg.Provenance
	switch {
	case sg.Method == dom.MethodPaper:
		class = "paper"
	case class == "server_attested" || class == "":
		class = "personal"
	}
	if sg.Method == dom.MethodSource && sg.Person == "" {
		class = "device"
	}
	who := sg.Person
	if who == "" {
		who = "устройство"
		if r, ok := v.record(sg.EventID); ok && r.SourceID != "" {
			who = r.SourceID
		}
	}
	method := sg.Method
	if method == "" {
		method = dom.MethodDemo
	}
	return DocumentStageSignature{EventID: sg.EventID, SignerID: who, Class: class, Level: sg.Level, Check: "unchecked", AttestedBy: sg.AttestedBy,
		PaperOriginalRef: sg.PaperNo, ScanAddress: sg.ScanAddress, Method: method, SignedAt: sg.At, Counted: counted, PreviousVersion: prev, Why: why,
		KeyRef: sg.KeyRef}
}

// Render — каноническая отрисовка HTML (documents.document.render, AD-12).
func (s *Service) Render(ctx context.Context, documentID string, version int) (DocumentRendering, error) {
	if !s.live() {
		return s.Unimplemented.Render(ctx, documentID, version)
	}
	v, d, err := s.load(ctx, documentID, platform.Moment{})
	if err != nil {
		return DocumentRendering{}, err
	}
	ver, b, _, err := v.version(d, version)
	if err != nil {
		return DocumentRendering{}, err
	}
	return DocumentRendering{DocumentID: d.ID, Version: ver.No, RenderingHash: b.RenderingHash, HTML: b.HTML}, nil
}

// PrintView — печатная форма (documents.paper.print_view, FR-139): отрисовка
// в рамке с QR, датой печати и колонтитулом «получено из системы»; рамка в
// отпечаток не входит (AD-12).
func (s *Service) PrintView(ctx context.Context, documentID string, version int) (PrintView, error) {
	if !s.live() {
		return s.Unimplemented.PrintView(ctx, documentID, version)
	}
	v, d, err := s.load(ctx, documentID, platform.Moment{})
	if err != nil {
		return PrintView{}, err
	}
	ver, b, _, err := v.version(d, version)
	if err != nil {
		// Печать зафиксировала новую версию, воркер её ещё не оформил: та же
		// версия по текущему содержимому (AD-5: свёртка детерминирована).
		pv, pb, changed, perr := v.State.Preview(v.Env, d)
		if perr != nil || !changed || pv.No != version {
			return PrintView{}, err
		}
		ver, b = &pv, pb
	}
	now, err := s.now(ctx, v.RunID)
	if err != nil {
		return PrintView{}, err
	}
	qr := dom.QR(d.ID, b.Digest)
	svg, err := QRSVG(qr)
	if err != nil {
		return PrintView{}, err
	}
	return PrintView{DocumentID: d.ID, Version: ver.No, DocDigest: b.Digest, RenderingHash: b.RenderingHash, QR: qr, QRSVG: svg, PrintedAt: now,
		HTML: PrintFrame(b.HTML, qr, svg, dom.FormatTime(now))}, nil
}

// DecisionCard — карточка «требуется ваше решение» (documents.decision_card.read, FR-136).
func (s *Service) DecisionCard(ctx context.Context, documentID string, m platform.Moment) (DecisionCard, error) {
	if !s.live() {
		return s.Unimplemented.DecisionCard(ctx, documentID, m)
	}
	v, d, err := s.load(ctx, documentID, m)
	if err != nil {
		return DecisionCard{}, err
	}
	rq, ok := s.request(ctx, v, d, "")
	if !ok {
		return DecisionCard{}, notFound("Версия документа", documentID)
	}
	card := DecisionCard{DocumentID: d.ID, Version: rq.Document.Version, Title: d.Title, Question: rq.Proposal.Summary, WhyYou: rq.Escalation.Reason,
		Subject: drill(d.Subject), Basis: []platform.DrillRef{}, SummaryFields: []DocumentSummaryField{}, OtherSigners: []string{},
		AfterSignature: afterSignature(d), DocDigest: rq.Document.DocDigest, BasisSeq: v.BasisSeq}
	if cur := d.Current(); cur != nil {
		card.SummaryFields = summaryFields(cur.Content)
	}
	card.Basis = append(card.Basis, drill(d.Subject))
	for _, st := range rq.Document.Route {
		if rq.MyStage != nil && st.Stage == *rq.MyStage {
			card.Stage = DocumentApprovalStage{Stage: st.Stage, Authority: st.AuthorityID, StampKind: st.StampKind, Quorum: st.Quorum, Required: st.Required,
				Level: st.SignatureLevel, PaperAllowed: st.PaperAllowed, ExternalParty: st.ExternalParty, Candidates: candidates(st, v), SignedBy: []string{}, Status: "pending"}
			continue
		}
		card.OtherSigners = append(card.OtherSigners, st.AuthorityLabel)
	}
	if card.Stage.Stage == 0 && len(rq.Document.Route) > 0 {
		st := rq.Document.Route[len(rq.Document.Route)-1]
		card.Stage = DocumentApprovalStage{Stage: st.Stage, Authority: st.AuthorityID, Quorum: st.Quorum, Required: st.Required, Level: st.SignatureLevel,
			PaperAllowed: st.PaperAllowed, Candidates: []string{}, SignedBy: []string{}, Status: "pending"}
	}
	return card, nil
}

func afterSignature(d *dom.Doc) string {
	switch d.DocType {
	case dom.DocNCDisposition:
		return "Когда подпишут все этапы, маршрут закроется (document.route.closed) и решение по несоответствию исполнится (AD-43)"
	case dom.DocTraveler:
		return "Итоговая годность изделия зафиксирована подписями контролёра ОТК и мастера"
	}
	return "Когда подпишут все этапы, маршрут закроется (document.route.closed)"
}

// DecisionRequests — запросы решения, ждущие подписи текущего пользователя
// (documents.request.list, FR-136): открытые версии документов, чей
// ближайший незакрытый этап вправе подписать пользователь сеанса.
func (s *Service) DecisionRequests(ctx context.Context, m platform.Moment) (DecisionRequestList, error) {
	if !s.live() {
		return s.Unimplemented.DecisionRequests(ctx, m)
	}
	out := DecisionRequestList{Items: []DecisionRequest{}}
	ids, err := s.documentIDs(ctx, m)
	if err != nil {
		return out, err
	}
	person := platform.PrincipalFrom(ctx).PersonID
	cache := map[string]*view{}
	for _, id := range ids {
		v, d, err := s.cached(ctx, cache, id, m)
		if err != nil || d == nil {
			continue
		}
		if rq, ok := s.request(ctx, v, d, person); ok && (rq.MyStage != nil || person == "") {
			out.Items = append(out.Items, rq)
		}
	}
	return out, nil
}

// request — карточка запроса решения по текущей версии документа d; ok=false
// — версия не ждёт подписей.
func (s *Service) request(_ context.Context, v *view, d *dom.Doc, person string) (DecisionRequest, bool) {
	cur := d.Current()
	if cur == nil || cur.Closed || cur.Annulled || len(cur.Declines) > 0 {
		return DecisionRequest{}, false
	}
	ev := dom.Evaluate(cur, v.Env.People, v.State.Participants)
	if ev.Next == 0 {
		return DecisionRequest{}, false
	}
	route := routeView(d, cur, v)
	status := "drafted"
	if len(cur.Signatures) > 0 {
		status = "in_route"
	}
	rq := DecisionRequest{
		Document: RoutedDocument{DocumentID: d.ID, Version: cur.No, TemplateRef: d.TemplateRef, DocType: d.DocType, Title: d.Title, DocDigest: cur.Digest,
			Status: status, DraftedAt: cur.At, Route: route, BasisSeq: v.BasisSeq,
			SigningPayloadB64: signingPayload(cur.Content, cur.RenderingHash, d.TemplateRef, cur.Digest)},
		Evidence: []DecisionEvidence{}, SimilarAccepted: []SimilarDecision{}, SimilarRejected: []SimilarDecision{},
	}
	for _, r := range cur.Sources {
		t := ""
		if rec, ok := v.record(r.ID); ok {
			t = string(rec.Type)
		}
		rq.Evidence = append(rq.Evidence, DecisionEvidence{EventID: r.ID, EventType: t, OccurredAt: r.At})
	}
	next := slices.IndexFunc(route, func(st DocumentRouteStage) bool { return st.Stage == ev.Next })
	st := route[next]
	for _, c := range candidates(st, v) {
		if c == person {
			n := st.Stage
			rq.MyStage = &n
		}
	}
	if len(v.Env.People) == 0 && person != "" {
		n := st.Stage
		rq.MyStage = &n
	}
	if cs := candidates(st, v); len(cs) > 0 {
		rq.ExpectedSigner = &cs[0]
	}
	rq.Proposal, rq.Escalation = proposal(d, v)
	return rq, true
}

// proposal — что предлагается и почему пришло к подписанту (FR-50, FR-136).
func proposal(d *dom.Doc, v *view) (DecisionProposal, DecisionEscalation) {
	item := v.ItemID
	p := DecisionProposal{Kind: "other", Code: d.Context.Decision, Summary: d.Title, ItemID: item, ItemLabel: item}
	e := DecisionEscalation{Reason: "Документ «" + d.Title + "» ждёт подписей по маршруту шаблона " + d.TemplateRef, RuleID: "documents.route", RuleRev: d.TemplateRef}
	switch d.DocType {
	case dom.DocNCDisposition:
		p.Kind, p.NCID, p.NCNumber = "disposition", d.Key, dom.NCNumber(d.Key)
		p.Summary = "Решение по несоответствию " + dom.NCNumber(d.Key) + ": " + d.Context.Decision
		e.AutomationMode = 3
		if d.Context.Decision == "repair" || d.Context.Decision == "use_as_is" {
			e.AutomationMode = 4
			e.Reason = "Режим 4: решение «" + d.Context.Decision + "» исполняется только после подписей технолога и начальника ОТК (AD-43)"
		}
		if d.Context.CustomerAcceptance {
			e.AutomationMode = 5
			e.Reason += "; режим 5 — с представителем заказчика"
		}
	case dom.DocTraveler:
		p.Code, p.Summary = "final_acceptance", "Итоговая годность изделия "+item+" по сопроводительной карте"
		e.Reason = "Итоговая годность — подписи контролёра ОТК и мастера участка (каталог документов, AD-43)"
	case dom.DocNCStatement:
		p.NCID, p.NCNumber = d.Key, dom.NCNumber(d.Key)
		p.Summary = "Заявление о несоответствии " + dom.NCNumber(d.Key)
	default:
		if d.Context.Comment != "" {
			p.Summary = d.Title + ": " + d.Context.Comment
		}
		if d.Context.Author != "" {
			e.Reason = "Запрошено " + d.Context.Author + ": " + d.Title
		}
	}
	if k, id, ok := strings.Cut(d.Subject, ":"); ok && item == "" {
		p.ItemID, p.ItemLabel = id, k+" "+id
	}
	return p, e
}

// streamEntries — записи потока документа (для проверки повтора команды, AD-7).
func (s *Service) streamEntries(ctx context.Context, documentID string) ([]jc.JournalEntry, error) {
	return s.readAll(ctx, appjournal.ReadQuery{Stream: streamOf(documentID)})
}

// signingPayload — содержимое для подписи агентом токена (AD-12, AD-14):
// канонические байты, от которых взят отпечаток; base64. Отпечаток по ним не
// сходится с записанным (шаблон сменился) — пусто: подписывать нечего,
// остаётся бумага.
func signingPayload(content json.RawMessage, renderingHash, templateRef, digest string) string {
	in, err := dom.DigestInput(content, renderingHash, templateRef, dom.DocFormatVersion)
	if err != nil || dom.Hash(in) != digest {
		return ""
	}
	return base64.StdEncoding.EncodeToString(in)
}
