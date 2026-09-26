package documents

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"
	"uuid"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/documents"
	"ant/internal/domain/kernel"
)

// Запрос версии и печать (FR-146, FR-136, FR-139; AD-12).

// TemplateForAction — шаблон документа по недоступному действию («Запросить
// решение», FR-146): версия процесса — лист утверждения (кворум, AD-43),
// прочее — запрос решения уполномоченного.
func TemplateForAction(action string) string {
	switch {
	case strings.HasPrefix(action, "process.version."):
		return dom.TemplateProcessApproval
	case action == "documents.traveler.version" || action == "traveler":
		return dom.TemplateTraveler
	}
	return dom.TemplateDecisionRequest
}

func itoa(n int) string { return strconv.Itoa(n) }

func shortID(id string) string {
	h := strings.ToUpper(strings.ReplaceAll(id, "-", ""))
	if len(h) > 12 {
		h = h[len(h)-12:]
	}
	return h
}

// RequestVersion — «Запросить решение» / новая версия документа
// (documents.version.request): запись document.version.requested; версию
// оформляет свёртка (у изделия — воркер, вне изделия — api в той же пачке).
func (s *Service) RequestVersion(ctx context.Context, in RequestVersion) (RequestAccepted, error) {
	if !s.live() {
		return s.Unimplemented.RequestVersion(ctx, in)
	}
	me, err := person(ctx)
	if err != nil {
		return RequestAccepted{}, err
	}
	kind, subjectID, ok := strings.Cut(in.SubjectRef, ":")
	if !ok || kind == "" || subjectID == "" {
		return RequestAccepted{}, platform.Fail(errcodes.ApiValidationFailed, "field", "subject_ref", "reason", "ожидается ‹вид›:‹id›")
	}
	tref := in.TemplateRef
	if tref == "" {
		tref = TemplateForAction(in.Action)
	}
	env := s.d.Env
	t, ok := env.Templates.ByRef(tref)
	if !ok || !t.Complete() || (t.DocType != dom.DocGeneric && t.DocType != dom.DocTraveler) {
		return RequestAccepted{}, platform.Fail(errcodes.ApiValidationFailed, "field", "template_ref", "reason", "шаблон "+tref+" не оформляется по запросу")
	}
	docID := in.DocumentID
	switch {
	case t.DocType == dom.DocTraveler:
		if kind != string(platform.EntityItem) {
			return RequestAccepted{}, platform.Fail(errcodes.ApiValidationFailed, "field", "subject_ref", "reason", "сопроводительная карта — только у изделия")
		}
		docID = dom.TravelerID(subjectID)
	case docID == "":
		base := in.CommandID
		if base == "" {
			base = strconv.FormatInt(s.d.Now().UnixNano(), 16)
		}
		docID = "DOC-" + shortID(base)
	}
	if r, ok, err := s.replayed(ctx, docID, in.CommandID); err != nil || ok {
		return RequestAccepted{CommandID: r.CommandID, Seq: r.Seq, EventIDs: r.EventIDs, Replayed: r.Replayed, DocumentID: docID}, err
	}
	var v *view
	if kind == string(platform.EntityItem) {
		v, err = s.loadItem(ctx, subjectID, platform.Moment{})
	} else {
		v, err = s.loadStream(ctx, docID, platform.Moment{})
	}
	if err != nil {
		return RequestAccepted{}, err
	}
	data := map[string]any{"document_id": docID, "template_ref": t.Ref(), "subject_ref": in.SubjectRef}
	if len(in.SourceEventIDs) > 0 {
		data["source_event_ids"] = in.SourceEventIDs
	}
	if in.Decision != "" {
		data["decision"] = in.Decision
	}
	if in.Comment != "" {
		data["comment"] = in.Comment
	}
	now, err := s.now(ctx, v.RunID)
	if err != nil {
		return RequestAccepted{}, err
	}
	o := out{Type: catalog.DocumentVersionRequested, DocumentID: docID, ItemID: v.ItemID, RunID: v.RunID, Data: data, Meta: in.CommandMeta(), Actor: me, Level: 2, OccurredAt: now}
	p, rec, err := s.pending(o)
	if err != nil {
		return RequestAccepted{}, err
	}
	batch, err := s.withReactions(ctx, v, docID, []kernel.Record{rec}, []appjournal.Pending{p})
	if err != nil {
		return RequestAccepted{}, err
	}
	r, err := s.commit(ctx, docID, o.Meta, batch, now)
	if err != nil {
		return RequestAccepted{}, err
	}
	acc := RequestAccepted{CommandID: r.CommandID, Seq: r.Seq, EventIDs: r.EventIDs, Replayed: r.Replayed, DocumentID: docID}
	// Какую версию оформит свёртка: та же доменная функция над состоянием с
	// новой записью (AD-5).
	st := dom.Reduce(v.State, rec, v.Env, dom.Upstream{})
	if d := st.Doc(docID); d != nil {
		if cur := d.Current(); cur != nil && cur.Trigger.ID == rec.EventID {
			acc.Version, acc.DocDigest = cur.No, cur.Digest
		}
	}
	return acc, nil
}

// RequestDecision — прежняя операция «Запросить решение» (documents.document.request, волна 1).
func (s *Service) RequestDecision(ctx context.Context, in RequestDecision) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RequestDecision(ctx, in)
	}
	acc, err := s.RequestVersion(ctx, RequestVersion{CommandHeader: in.CommandHeader, SubjectRef: string(in.Subject.Entity) + ":" + in.Subject.ID,
		TemplateRef: in.Template, Decision: in.Decision, Comment: in.Comment, SourceEventIDs: in.SourceEventIDs})
	if err != nil {
		return platform.Receipt{}, err
	}
	return platform.Receipt{CommandID: acc.CommandID, Seq: acc.Seq, EventIDs: acc.EventIDs, Replayed: acc.Replayed}, nil
}

// Print — напечатать бумажный экземпляр с QR (documents.paper.print, FR-139):
// запись document.paper.status_changed (printed). У документа изделия без
// зафиксированной версии или с изменившимся содержимым (сопроводительная
// карта) печать сначала фиксирует новую версию запросом в той же пачке —
// печатается ровно то, что будет подписано (AD-12).
func (s *Service) Print(ctx context.Context, documentID string, in PrintPaper) (PrintAccepted, error) {
	if !s.live() {
		return s.Unimplemented.Print(ctx, documentID, in)
	}
	me, err := person(ctx)
	if err != nil {
		return PrintAccepted{}, err
	}
	if r, ok, err := s.replayed(ctx, documentID, in.CommandID); err != nil || ok {
		return PrintAccepted{CommandID: r.CommandID, Seq: r.Seq, EventIDs: r.EventIDs, Replayed: r.Replayed}, err
	}
	v, d, err := s.load(ctx, documentID, platform.Moment{})
	if err != nil {
		return PrintAccepted{}, err
	}
	now, err := s.now(ctx, v.RunID)
	if err != nil {
		return PrintAccepted{}, err
	}
	var batch []appjournal.Pending
	no, digest := in.Version, ""
	if no == 0 && d.DocType == dom.DocTraveler && v.ItemID != "" {
		pv, _, changed, err := v.State.Preview(v.Env, d)
		if err != nil {
			return PrintAccepted{}, err
		}
		no, digest = pv.No, pv.Digest
		if changed {
			p, _, err := s.pending(out{Type: catalog.DocumentVersionRequested, DocumentID: documentID, ItemID: v.ItemID, RunID: v.RunID, Meta: in.CommandMeta(),
				Actor: me, Level: 1, OccurredAt: now, Data: map[string]any{"document_id": documentID, "template_ref": d.TemplateRef,
					"subject_ref": d.Subject, "decision": "print", "comment": "Печать бумажного экземпляра"}})
			if err != nil {
				return PrintAccepted{}, err
			}
			batch = append(batch, p)
		}
	} else {
		ver := d.Version(no)
		if ver == nil {
			return PrintAccepted{}, notFound("Версия документа", documentID)
		}
		no, digest = ver.No, ver.Digest
	}
	mark := out{Type: catalog.DocumentPaperStatusChanged, DocumentID: documentID, ItemID: v.ItemID, RunID: v.RunID, Meta: in.CommandMeta(), Actor: me,
		OccurredAt: now.Add(time.Millisecond), Data: map[string]any{"document_id": documentID, "version": no, "paper_status": "printed"}}
	if in.CopyNo != "" {
		mark.Data.(map[string]any)["copy_no"] = in.CopyNo
	}
	if len(batch) > 0 {
		mark.EventID = uuid.NewV7().String() // command_id — у запроса версии
	}
	p, rec, err := s.pending(mark)
	if err != nil {
		return PrintAccepted{}, err
	}
	batch = append(batch, p)
	batch, err = s.withReactions(ctx, v, documentID, []kernel.Record{rec}, batch)
	if err != nil {
		return PrintAccepted{}, err
	}
	r, err := s.commit(ctx, documentID, in.CommandMeta(), batch, now)
	if err != nil {
		return PrintAccepted{}, err
	}
	return PrintAccepted{CommandID: r.CommandID, Seq: r.Seq, EventIDs: r.EventIDs, Replayed: r.Replayed, Version: no, DocDigest: digest,
		QR: dom.QR(documentID, digest), PrintURL: s.cfg.APIPrefix + "/documents/" + url.PathEscape(documentID) + "/print?version=" + itoa(no)}, nil
}
