package documents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"time"

	app "ant/internal/application/documents"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/documents"
	"ant/internal/infrastructure/fixtures/loader"
)

// Сессионное наложение документов (FR-129, FR-136, AD-43): подпись, отказ,
// бумага и «Запросить решение» мир заготовок не меняют, но стенд показывает
// их до сброса прогона — этап маршрута закрывается, документ меняет состояние
// в реестре, запрос уходит из «Требуется ваше решение» у подписанта; запрос
// решения создаёт новый документ с маршрутом к тем, у кого есть полномочие.

// Операции-факты документа.
const (
	opSign    = "documents.document.sign"
	opRecord  = "documents.signature.record"
	opDecline = "documents.signature.decline"
	opAttest  = "documents.paper.attest"
	opPaper   = "documents.paper.status_set"
	opPrint   = "documents.paper.print"
	opRequest = "documents.version.request"
)

// kindDocument — вид объекта фактов документа.
const kindDocument = string(platform.EntityDocument)

// facts — факты документа id на момент m.
func facts(ctx context.Context, m *platform.Moment, id string) []loader.Fact {
	rt, err := loader.Default()
	if err != nil {
		return nil
	}
	return rt.FactsOf(ctx, m, kindDocument, id)
}

// requested — документы, созданные «Запросить решение» в этой сессии: id → вид.
func requested(ctx context.Context, m *platform.Moment) ([]app.DocumentView, map[string][]string) {
	rt, err := loader.Default()
	if err != nil {
		return nil, nil
	}
	var out []app.DocumentView
	items := map[string][]string{}
	for _, f := range rt.Facts(ctx, m, kindDocument) {
		rq, ok := f.Body.(requestedDoc)
		if !ok {
			continue
		}
		out = slices.DeleteFunc(out, func(v app.DocumentView) bool { return v.DocumentID == rq.View.DocumentID })
		out = append(out, rq.View)
		items[rq.View.DocumentID] = rq.Items
	}
	return out, items
}

// requestedDoc — тело факта «Запросить решение»: готовый вид новой версии.
type requestedDoc struct {
	View  app.DocumentView
	Items []string
}

// overlay накладывает факты сессии на документ: подписи закрывают этапы,
// отказ возвращает версию, бумага — отметки экземпляра.
func overlay(v app.DocumentView, fs []loader.Fact) app.DocumentView {
	if len(fs) == 0 {
		return v
	}
	v.Route = slices.Clone(v.Route)
	v.Stages = slices.Clone(v.Stages)
	for i := range v.Route {
		v.Route[i].Signatures = slices.Clone(v.Route[i].Signatures)
	}
	for i := range v.Stages {
		v.Stages[i].SignedBy = slices.Clone(v.Stages[i].SignedBy)
	}
	v.Signatures = slices.Clone(v.Signatures)
	for _, f := range fs {
		evt := eventID(f)
		switch b := f.Body.(type) {
		case app.SignDocument:
			if b.Version == 0 || b.Version == v.Version {
				v = sign(v, b.Stage, f.Actor, "", f, evt, "demo_signer", "personal", "", "")
			}
		case app.RecordSignature:
			if b.Version == 0 || b.Version == v.Version {
				method := "demo_signer"
				if b.SignatureB64 != "" {
					method = "token_agent"
				}
				v = sign(v, b.Stage, f.Actor, "", f, evt, method, "personal", "", "")
			}
		case app.AttestPaper:
			if b.Version == 0 || b.Version == v.Version {
				signer := b.SignerPersonID
				if signer == "" {
					signer = f.Actor
				}
				v = sign(v, b.Stage, signer, f.Actor, f, evt, "paper", "paper", b.PaperOriginalNo, b.ScanAddress)
			}
		case app.DeclineSignature:
			v.Declines = append(slices.Clone(v.Declines), app.DocumentDecline{EventID: evt, Version: v.Version, Stage: b.Stage, SignerID: f.Actor, Comment: b.Comment, At: f.At})
			v.Status = dom.StatusReturned
		case app.SetPaperStatus:
			v.Paper = append(slices.Clone(v.Paper), app.DocumentPaperMark{EventID: evt, Version: v.Version, Status: b.PaperStatus, CopyNo: b.CopyNo, At: f.At})
		case app.PrintPaper:
			v.Paper = append(slices.Clone(v.Paper), app.DocumentPaperMark{EventID: evt, Version: v.Version, Status: "printed", CopyNo: b.CopyNo, At: f.At})
		}
	}
	if v.Status != dom.StatusReturned && v.Status != dom.StatusAnnulled {
		done := len(v.Route) > 0
		for _, s := range v.Route {
			done = done && s.Done
		}
		switch {
		case done:
			v.Status = dom.StatusRouteClosed
		case len(v.Signatures) > 0:
			v.Status = dom.StatusSigning
		}
	}
	v.Versions = slices.Clone(v.Versions)
	for i := range v.Versions {
		if v.Versions[i].Version == v.Version {
			v.Versions[i].Status = v.Status
			if v.Status == dom.StatusSigning || v.Status == dom.StatusRouteClosed || v.Status == dom.StatusReturned {
				v.Versions[i].Signatures = len(v.Signatures)
			}
		}
	}
	return v
}

// sign — подпись этапа stage (0 — ближайший незакрытый этап, где signer
// среди кандидатов): подпись засчитана, этап закрыт при достаточном числе.
func sign(v app.DocumentView, stage int, signer, attester string, f loader.Fact, evt, method, class, paperNo, scan string) app.DocumentView {
	i := slices.IndexFunc(v.Route, func(s app.DocumentRouteStage) bool { return s.Stage == stage && !s.Done })
	if i < 0 {
		i = slices.IndexFunc(v.Route, func(s app.DocumentRouteStage) bool { return !s.Done })
	}
	if i < 0 {
		return v
	}
	st := &v.Route[i]
	if slices.ContainsFunc(st.Signatures, func(s app.DocumentStageSignature) bool { return s.SignerID == signer && s.Counted }) {
		return v
	}
	st.Signatures = append(st.Signatures, app.DocumentStageSignature{EventID: evt, SignerID: signer, Class: class, Level: st.SignatureLevel, Check: "unchecked",
		AttestedBy: attester, PaperOriginalRef: paperNo, ScanAddress: scan, Method: method, SignedAt: f.At, Counted: true})
	counted := 0
	for _, s := range st.Signatures {
		if s.Counted {
			counted++
		}
	}
	st.Done = counted >= max(st.Required, 1)
	v.Signatures = append(v.Signatures, app.DocumentSignatureView{EventID: evt, Stage: st.Stage, Method: method, SignerPersonID: signer, AttestedBy: attester,
		Level: st.SignatureLevel, SignedAt: f.At, DocDigest: v.DocDigest, CurrentVersion: true, Verification: "pending", ProvenanceClass: class})
	for j := range v.Stages {
		if v.Stages[j].Stage != st.Stage {
			continue
		}
		v.Stages[j].SignedBy = append(v.Stages[j].SignedBy, signer)
		v.Stages[j].Status = "in_progress"
		if st.Done {
			v.Stages[j].Status = "done"
		}
	}
	return v
}

// summaryOf — строка реестра по виду документа с наложением: состояние,
// кто подписывает сейчас, сколько этапов закрыто.
func summaryOf(d app.DocumentSummary, v app.DocumentView) app.DocumentSummary {
	d.Status = v.Status
	d.Versions = max(d.Versions, len(v.Versions))
	if d.Version != v.Version {
		d.Version, d.DocDigest = v.Version, v.DocDigest
	}
	d.PaperStatus = ""
	for _, p := range v.Paper {
		if p.Version == v.Version {
			d.PaperStatus = p.Status
		}
	}
	d.State = app.RegistryState(d.Status, d.PaperStatus)
	d.Awaiting, d.StagesDone, d.StagesTotal = nil, 0, len(v.Route)
	var last time.Time
	for _, s := range v.Route {
		if s.Done {
			d.StagesDone++
		} else if d.Awaiting == nil && v.Status != dom.StatusReturned && v.Status != dom.StatusAnnulled {
			var cands []string
			for _, x := range v.Stages {
				if x.Stage == s.Stage {
					cands = x.Candidates
				}
			}
			d.Awaiting = &app.DocumentAwaiting{Stage: s.Stage, Title: s.AuthorityLabel, Role: s.Role, Candidates: cands}
		}
		for _, sg := range s.Signatures {
			if sg.SignedAt.After(last) {
				last = sg.SignedAt
			}
		}
	}
	for _, x := range v.Declines {
		if x.At.After(last) {
			last = x.At
		}
	}
	if !last.IsZero() {
		d.UpdatedAt = &last
		if v.Status == dom.StatusRouteClosed {
			d.ClosedAt = &last
		}
	}
	return d
}

// eventID — id записи факта для подписи или отказа (детерминированный от квитанции).
func eventID(f loader.Fact) string {
	return "EV-SESSION-" + strings.ToUpper(shortHash(f.CommandID+f.Op))
}

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:6])
}

// newRequest — вид документа «Запросить решение» (FR-146): маршрут из одного
// этапа к тем, у кого есть полномочие на действие; новая версия существующего
// документа — его копия с маршрутом без подписей.
func newRequest(ctx context.Context, in app.RequestVersion, now time.Time) (app.DocumentView, []string, error) {
	kind, subjectID, ok := strings.Cut(in.SubjectRef, ":")
	if !ok || kind == "" || subjectID == "" {
		return app.DocumentView{}, nil, platform.Fail(errcodes.ApiValidationFailed, "field", "subject_ref", "reason", "ожидается ‹вид›:‹id›")
	}
	var items []string
	if kind == string(platform.EntityItem) {
		items = []string{subjectID}
	}
	if in.DocumentID != "" {
		v, err := Adapter{}.Document(ctx, in.DocumentID, 0, platform.Moment{})
		if err != nil {
			return v, nil, err
		}
		v.Versions = append(slices.Clone(v.Versions), app.DocumentVersionRef{Version: v.Version + 1, DocDigest: v.DocDigest, Status: dom.StatusDrafted, DraftedAt: now})
		prev := v.Version
		v.Version++
		v.SupersedesVersion = &prev
		v.Status, v.Signatures, v.Declines, v.Paper, v.RouteClosed, v.Live = dom.StatusDrafted, []app.DocumentSignatureView{}, nil, nil, nil, false
		v.DocDigest = "streebog256:" + shortHash(v.DocDigest+in.CommandID) + strings.Repeat("0", 52)
		v.Route = slices.Clone(v.Route)
		for i := range v.Route {
			v.Route[i].Done, v.Route[i].Signatures = false, []app.DocumentStageSignature{}
		}
		v.Stages = slices.Clone(v.Stages)
		for i := range v.Stages {
			v.Stages[i].SignedBy, v.Stages[i].Status = []string{}, "pending"
		}
		return v, items, nil
	}
	action := in.Action
	if action == "" {
		action = "documents.document.sign"
	}
	var cands []string
	if loader.Holders != nil {
		me := platform.PrincipalFrom(ctx).PersonID
		for _, p := range loader.Holders(action) {
			if p != me {
				cands = append(cands, p)
			}
		}
	}
	if len(cands) == 0 {
		return app.DocumentView{}, nil, platform.Fail(errcodes.ApiValidationFailed, "field", "action", "reason", "нет сотрудников с полномочием "+action)
	}
	tref := in.TemplateRef
	if tref == "" {
		tref = app.TemplateForAction(action) + "@1"
	}
	id := "DOC-" + strings.ToUpper(shortHash(in.CommandID))
	title := "Запрос решения: " + subjectID
	if in.Decision != "" {
		title += " — " + in.Decision
	}
	label := "Уполномоченный на " + action
	digest := "streebog256:" + shortHash(id) + strings.Repeat("0", 52)
	content := map[string]any{"subject_ref": in.SubjectRef, "action": action, "decision": in.Decision, "comment": in.Comment, "source_event_ids": in.SourceEventIDs}
	v := app.DocumentView{DocumentID: id, Version: 1, Template: tref, DocFormatVersion: 1, Title: title,
		Subject: platform.DrillRef{Entity: platform.EntityKind(kind), ID: subjectID}, Status: dom.StatusDrafted, Content: content,
		RenderingHash: digest, DocDigest: digest, SummaryFields: []app.DocumentSummaryField{
			{Key: "action", Label: "Действие", Value: action}, {Key: "decision", Label: "Решение", Value: in.Decision}, {Key: "comment", Label: "Комментарий", Value: in.Comment}},
		SourceEventIDs: append([]string{}, in.SourceEventIDs...),
		Stages: []app.DocumentApprovalStage{{Stage: 1, Authority: action, Quorum: "one", Required: 1, Level: 2, Candidates: cands, SignedBy: []string{}, Status: "pending"}},
		Signatures: []app.DocumentSignatureView{}, QR: dom.QR(id, digest), DocType: dom.DocGeneric, Class: "decision",
		Route: []app.DocumentRouteStage{{Stage: 1, AuthorityID: action, AuthorityLabel: label, Quorum: "one", Required: 1, SignatureLevel: 2, Signatures: []app.DocumentStageSignature{}}},
		Versions: []app.DocumentVersionRef{{Version: 1, DocDigest: digest, Status: dom.StatusDrafted, DraftedAt: now}}, SubjectLabel: subjectID}
	return v, items, nil
}

// summaryOfView — строка реестра нового документа сессии.
func summaryOfView(v app.DocumentView, items []string) app.DocumentSummary {
	at := v.Versions[len(v.Versions)-1].DraftedAt
	d := app.DocumentSummary{DocumentID: v.DocumentID, Version: v.Version, Template: v.Template, Title: v.Title, Subject: v.Subject, Status: v.Status,
		DocDigest: v.DocDigest, DraftedAt: &at, DocType: v.DocType, Class: v.Class, Versions: len(v.Versions), SubjectLabel: v.SubjectLabel, ItemIDs: items, UpdatedAt: &at}
	return summaryOf(d, v)
}
