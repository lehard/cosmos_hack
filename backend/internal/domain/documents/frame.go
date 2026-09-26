package documents

import (
	"encoding/json"
	"slices"
	"strconv"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/access"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
	"ant/internal/domain/process"
	"ant/internal/domain/quality"
	"ant/internal/domain/vision"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "documents"

// Правила реакций модуля (слоты AD-3): одна версия документа — один слот,
// так все версии остаются в журнале, а пересвёртка пересматривает только
// изменившиеся.
const (
	// RuleVersion — document.version.drafted версии документа.
	RuleVersion = "documents.version"
	// RuleRoute — document.route.closed версии документа.
	RuleRoute = "documents.route"
)

// State — состояние модуля documents в свёртке одного изделия (AD-5):
// модель сопроводительной карты (шапка, строки операций, несоответствия,
// итог), участники изготовления (FR-56) и документы изделия с версиями,
// подписями и маршрутами. Та же модель — у документа вне изделия
// (FoldStream): тогда заполнены только Docs.
type State struct {
	ItemID       string   `json:"item_id,omitempty"`
	RunID        string   `json:"run_id,omitempty"`
	Header       Header   `json:"header"`
	Rows         []Row    `json:"rows,omitempty"`
	NCs          []NCInfo `json:"ncs,omitempty"`
	Participants []string `json:"participants,omitempty"`
	Final        *Final   `json:"final,omitempty"`
	// Completed — процесс изделия завершён: карта зафиксирована версией.
	Completed bool `json:"completed,omitempty"`
	// TravelerRef — шаблон сопроводительной карты версии нормативного слоя изделия.
	TravelerRef string `json:"traveler_ref,omitempty"`
	Docs        []Doc  `json:"docs,omitempty"`
	// Requested — черновики, запрошенные намерением documents.Draft (AD-40);
	// оформляются на следующей записи изделия.
	Requested []DraftContext `json:"requested,omitempty"`
	// Undecodable — записи, которые не удалось разобрать: факт не теряется молча.
	Undecodable []string `json:"undecodable,omitempty"`
}

// Upstream — состояния модулей раньше documents в композиции на этом шаге
// (только чтение, AD-40): поздний модуль видит вывод раннего, обратно — только
// через функцию-намерение раннего модуля.
type Upstream struct {
	Item        *item.State
	Process     *process.State
	Vision      *vision.State
	Quality     *quality.State
	Machinelogs *machinelogs.State
}

func (s *State) bad(r kernel.Record) {
	if !slices.Contains(s.Undecodable, r.EventID) {
		s.Undecodable = append(s.Undecodable, r.EventID)
	}
}

// Doc — документ по id (nil — нет).
func (s *State) Doc(id string) *Doc {
	for i := range s.Docs {
		if s.Docs[i].ID == id {
			return &s.Docs[i]
		}
	}
	return nil
}

func (s *State) doc(id string) *Doc { return s.Doc(id) }

// clone — глубокая копия: прежнее состояние свёртки не меняется (AD-4).
func (s State) clone() State {
	s.Header.Lots = slices.Clone(s.Header.Lots)
	s.Rows = slices.Clone(s.Rows)
	for i := range s.Rows {
		r := &s.Rows[i]
		r.Sigs, r.Params, r.Checks, r.Remarks, r.Sources = slices.Clone(r.Sigs), slices.Clone(r.Params), slices.Clone(r.Checks), slices.Clone(r.Remarks), slices.Clone(r.Sources)
		r.Regime = slices.Clone(r.Regime)
		if r.OTK != nil {
			m := *r.OTK
			r.OTK = &m
		}
	}
	s.NCs = slices.Clone(s.NCs)
	for i := range s.NCs {
		n := &s.NCs[i]
		n.Evidence, n.Sources = slices.Clone(n.Evidence), slices.Clone(n.Sources)
	}
	s.Participants = slices.Clone(s.Participants)
	s.Docs = slices.Clone(s.Docs)
	for i := range s.Docs {
		d := &s.Docs[i]
		d.Context.Sources = slices.Clone(d.Context.Sources)
		d.Paper = slices.Clone(d.Paper)
		d.Versions = slices.Clone(d.Versions)
		for j := range d.Versions {
			v := &d.Versions[j]
			v.Stages, v.Sources, v.SourceSigners = slices.Clone(v.Stages), slices.Clone(v.Sources), slices.Clone(v.SourceSigners)
			v.Signatures, v.Declines, v.ClosedBy, v.ClosedCauses = slices.Clone(v.Signatures), slices.Clone(v.Declines), slices.Clone(v.ClosedBy), slices.Clone(v.ClosedCauses)
		}
	}
	s.Requested = slices.Clone(s.Requested)
	s.Undecodable = slices.Clone(s.Undecodable)
	return s
}

// Reduce применяет запись входа изделия к состоянию модуля (AD-5): строит
// модель карты, оформляет документы по записям-триггерам (подтверждение и
// регистрация несоответствия, решение по нему, запрос версии, завершение
// процесса изделия) и учитывает подписи. Реакции в свёртку не входят (AD-3).
func Reduce(s State, r kernel.Record, env Env, up Upstream) State {
	if !env.Active() {
		return s
	}
	s = s.clone()
	if s.ItemID == "" {
		s.ItemID, s.RunID = r.ItemID, r.RunID
	}
	if s.TravelerRef == "" && s.ItemID != "" {
		if t, ok := env.Templates.ByRef(TemplateTraveler); ok && t.Complete() {
			s.TravelerRef = t.Ref()
		}
	}
	for _, c := range s.Requested {
		s.fromIntent(env, c, r, up)
	}
	s.Requested = nil
	s.reduceTraveler(r)
	s.fromMachinelogs(up)
	s.reduceNC(r, env, up)
	s.reduceDocs(r, env, up)
	// Карта фиксируется версией при сдаче изделия на склад и при завершении
	// процесса изделия (выпуск или списание, AD-12: новая версия — только
	// если содержимое изменилось).
	completed := up.Process != nil && up.Process.Completed
	if r.Type == catalog.ItemReleaseRecorded || (completed && !s.Completed) {
		s.Completed = s.Completed || completed
		s.draft(env, s.traveler(env), r, up)
	}
	return s
}

// traveler — документ «сопроводительная карта» изделия (создаётся при
// первой фиксации версии; до неё карта «собирается из истории»).
func (s *State) traveler(env Env) *Doc {
	id := TravelerID(s.itemID())
	if d := s.doc(id); d != nil {
		return d
	}
	t, ok := env.Templates.ByRef(TemplateTraveler)
	if !ok || !t.Complete() || s.itemID() == "" {
		return nil
	}
	s.Docs = append(s.Docs, Doc{ID: id, TemplateRef: t.Ref(), DocType: t.DocType, Class: t.Class, Title: t.Title, Subject: "item:" + s.itemID()})
	return &s.Docs[len(s.Docs)-1]
}

// Traveler — сопроводительная карта изделия: документ (если версия уже
// фиксировалась) или его заготовка без версий для показа «собирается из
// истории».
func (s *State) Traveler(env Env) (Doc, bool) {
	if d := s.doc(TravelerID(s.itemID())); d != nil {
		return *d, true
	}
	t, ok := env.Templates.ByRef(TemplateTraveler)
	if !ok || !t.Complete() || s.itemID() == "" {
		return Doc{}, false
	}
	return Doc{ID: TravelerID(s.itemID()), TemplateRef: t.Ref(), DocType: t.DocType, Class: t.Class, Title: t.Title, Subject: "item:" + s.itemID()}, true
}

// Preview — следующая версия документа d по текущему состоянию: номер,
// content, отпечаток и маршрут; changed=false — содержимое не изменилось,
// новой версии не будет (возвращается текущая). Им api показывает «живую»
// карту и заранее знает отпечаток печатаемой версии.
func (s *State) Preview(env Env, d *Doc) (v Version, built Built, changed bool, err error) {
	t, ok := env.Templates.ByRef(d.TemplateRef)
	if !ok {
		return Version{}, Built{}, false, nil
	}
	body, _, _ := s.body(d, env)
	if body == nil {
		return Version{}, Built{}, false, nil
	}
	raw, err := Canonical(body)
	if err != nil {
		return Version{}, Built{}, false, err
	}
	cur := d.Current()
	if cur != nil && cur.BodyHash == Hash(raw) && !cur.Annulled && len(cur.Declines) == 0 {
		b, err := Rebuild(t, cur)
		return *cur, b, false, err
	}
	no := 1
	if cur != nil {
		no = cur.No + 1
	}
	stages := approvalsFor(t, d, env)
	built, err = Compose(t, d, no, body, stages)
	v = Version{No: no, Content: built.Content, BodyHash: Hash(raw), RenderingHash: built.RenderingHash, Digest: built.Digest, Stages: stages}
	if cur != nil {
		v.Supersedes = cur.No
	}
	return v, built, true, err
}

// Rebuild — отрисовка и отпечаток записанной версии заново из её content
// (проверка «отпечаток не изменился»).
func Rebuild(t Template, v *Version) (Built, error) {
	html, err := Render(t, v.Content)
	if err != nil {
		return Built{}, err
	}
	rh := RenderingHash(html)
	d, err := DocDigest(v.Content, rh, t.Ref(), DocFormatVersion)
	return Built{Content: v.Content, HTML: html, RenderingHash: rh, Digest: d}, err
}

// reduceDocs — жизненный цикл документов записями семейства document.
func (s *State) reduceDocs(r kernel.Record, env Env, up Upstream) {
	switch r.Type {
	case catalog.DocumentVersionRequested:
		d, err := kernel.Decode[ev.DocumentVersionRequestedV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		c := DraftContext{Template: d.TemplateRef, Subject: string(d.SubjectRef), Key: string(d.DocumentID)}
		if d.Decision != nil {
			c.Decision = *d.Decision
		}
		if d.Comment != nil {
			c.Comment = *d.Comment
		}
		for _, id := range d.SourceEventIds {
			c.Sources = append(c.Sources, string(id))
		}
		if d.RequestedBy != nil {
			c.Author = string(*d.RequestedBy)
		}
		s.fromIntent(env, c, r, up)
	case catalog.DocumentSignatureRecorded:
		d, err := kernel.Decode[ev.DocumentSignatureRecordedV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		doc := s.doc(string(d.DocumentID))
		if doc == nil {
			return
		}
		i := slices.IndexFunc(doc.Versions, func(v Version) bool { return v.No == d.Version })
		if i < 0 {
			return
		}
		sg := Signature{EventID: r.EventID, Version: d.Version, Stage: d.Stage, Person: string(d.SignerPersonID), Method: string(d.Method),
			Digest: string(d.DocDigest), Level: d.SignatureLevel, Provenance: r.Provenance, At: r.OccurredAt}
		if d.AuthorityID != nil {
			sg.AuthorityID = string(*d.AuthorityID)
		}
		if d.StampID != nil {
			sg.StampID = string(*d.StampID)
		}
		if d.AttestedBy != nil {
			sg.AttestedBy = string(*d.AttestedBy)
		}
		if d.PaperOriginalNo != nil {
			sg.PaperNo = *d.PaperOriginalNo
		}
		if d.ScanAddress != nil {
			sg.ScanAddress = string(*d.ScanAddress)
		}
		if d.KeyRef != nil {
			sg.KeyRef = *d.KeyRef
		}
		doc.Versions[i].Signatures = append(doc.Versions[i].Signatures, sg)
		s.evaluate(env, doc, i, r)
	case catalog.DocumentSignatureDeclined:
		d, err := kernel.Decode[ev.DocumentSignatureDeclinedV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		if doc := s.doc(string(d.DocumentID)); doc != nil {
			if v := doc.Version(d.Version); v != nil && !v.Closed {
				v.Declines = append(v.Declines, Decline{EventID: r.EventID, Stage: d.Stage, Person: string(d.SignerPersonID), Comment: string(d.Comment), At: r.OccurredAt})
			}
		}
	case catalog.DocumentVersionAnnulled:
		d, err := kernel.Decode[ev.DocumentVersionAnnulledV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		if doc := s.doc(string(d.DocumentID)); doc != nil {
			if v := doc.Version(d.Version); v != nil {
				v.Annulled, v.AnnulledBy, v.AnnulReason = true, r.EventID, string(d.Reason.Text)
			}
		}
	case catalog.DocumentPaperStatusChanged:
		d, err := kernel.Decode[ev.DocumentPaperStatusChangedV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		if doc := s.doc(string(d.DocumentID)); doc != nil {
			m := PaperMark{EventID: r.EventID, Version: d.Version, Status: string(d.PaperStatus), At: r.OccurredAt}
			if d.CopyNo != nil {
				m.CopyNo = *d.CopyNo
			}
			doc.Paper = append(doc.Paper, m)
		}
	}
}

// fromIntent — оформить документ по запросу: сопроводительная карта (новая
// версия, если содержимое изменилось) или документ по шаблону запроса.
func (s *State) fromIntent(env Env, c DraftContext, r kernel.Record, up Upstream) {
	t, ok := env.Templates.ByRef(c.Template)
	if !ok && c.Key != "" && c.Key == TravelerID(s.itemID()) {
		t, ok = env.Templates.ByRef(TemplateTraveler)
	}
	if !ok || !t.Complete() {
		return
	}
	if t.DocType == DocTraveler {
		s.draft(env, s.traveler(env), r, up)
		return
	}
	if t.DocType != DocGeneric || c.Key == "" {
		return
	}
	d := s.doc(c.Key)
	if d == nil {
		subject := c.Subject
		if subject == "" && s.itemID() != "" {
			subject = "item:" + s.itemID()
		}
		s.Docs = append(s.Docs, Doc{ID: c.Key, TemplateRef: t.Ref(), DocType: t.DocType, Class: t.Class, Title: t.Title, Subject: subject})
		d = &s.Docs[len(s.Docs)-1]
	}
	author := c.Author
	if author == "" {
		author = PersonOf(r.Actor)
	}
	d.Context = DocContext{Decision: c.Decision, Comment: c.Comment, Author: author, At: FormatTime(r.OccurredAt),
		Sources: slices.Clone(c.Sources), SourceEvent: r.EventID}
	s.draft(env, d, r, up)
}

// draftedData — data реакции document.version.drafted (схема
// contracts/events/document/document.version.drafted.v1.json).
type draftedData struct {
	DocumentID       string   `json:"document_id"`
	Version          int      `json:"version"`
	TemplateRef      string   `json:"template_ref"`
	DocFormatVersion int      `json:"doc_format_version"`
	DocDigest        string   `json:"doc_digest"`
	RenderingHash    string   `json:"rendering_hash"`
	SubjectRef       string   `json:"subject_ref"`
	SourceEventIDs   []string `json:"source_event_ids"`
	// RequiredApprovals — замороженный набор обязательных подписей (AD-43).
	RequiredApprovals any    `json:"required_approvals"`
	Supersedes        *int   `json:"supersedes_document_version,omitempty"`
	Title             string `json:"title,omitempty"`
}

// routeClosedData — data реакции document.route.closed.
type routeClosedData struct {
	DocumentID        string   `json:"document_id"`
	Version           int      `json:"version"`
	DocDigest         string   `json:"doc_digest"`
	SignatureEventIDs []string `json:"signature_event_ids"`
	Verification      string   `json:"verification,omitempty"`
}

func records(rs []Ref) []kernel.Record {
	out := make([]kernel.Record, 0, len(rs))
	for _, r := range rs {
		out = append(out, kernel.Record{EventID: r.ID, OccurredAt: r.At})
	}
	return out
}

// Slot — слот реакции версии документа (AD-3).
func Slot(rule, documentID string, version int) kernel.Slot {
	return kernel.Slot{RuleID: rule, Subject: "document:" + documentID, TriggerKey: "v" + strconv.Itoa(version)}
}

// Reactions — реакции документа d: document.version.drafted каждой версии и
// document.route.closed каждой версии с закрытым маршрутом (AD-12, AD-43).
func Reactions(d *Doc) []kernel.Reaction {
	var out []kernel.Reaction
	for i := range d.Versions {
		v := &d.Versions[i]
		ids := make([]string, 0, len(v.Sources))
		for _, r := range v.Sources {
			ids = append(ids, r.ID)
		}
		data := draftedData{DocumentID: d.ID, Version: v.No, TemplateRef: d.TemplateRef, DocFormatVersion: DocFormatVersion, DocDigest: v.Digest,
			RenderingHash: v.RenderingHash, SubjectRef: d.Subject, SourceEventIDs: ids, RequiredApprovals: v.Stages, Title: d.Title}
		if v.Supersedes > 0 {
			sup := v.Supersedes
			data.Supersedes = &sup
		}
		if re, err := kernel.NewReaction(Module, catalog.DocumentVersionDrafted, Slot(RuleVersion, d.ID, v.No), data, records(v.Sources)...); err == nil {
			re.RuleRev = d.TemplateRef
			out = append(out, re)
		}
		if !v.Closed {
			continue
		}
		sigs := slices.Clone(v.ClosedBy)
		slices.Sort(sigs)
		sigs = slices.Compact(sigs)
		cd := routeClosedData{DocumentID: d.ID, Version: v.No, DocDigest: v.Digest, SignatureEventIDs: sigs, Verification: v.Verification}
		if re, err := kernel.NewReaction(Module, catalog.DocumentRouteClosed, Slot(RuleRoute, d.ID, v.No), cd, records(v.ClosedCauses)...); err == nil {
			re.RuleRev = d.TemplateRef
			out = append(out, re)
		}
	}
	return out
}

// React — реакции модуля по состоянию после записи (AD-3): версии
// документов и закрытые маршруты. Реакции строятся только через
// kernel.NewReaction с собственными типами модуля (AD-40).
func React(s State, env Env, up Upstream) kernel.Output {
	_ = up
	if !env.Active() {
		return kernel.Output{}
	}
	var out kernel.Output
	for i := range s.Docs {
		out.Reactions = append(out.Reactions, Reactions(&s.Docs[i])...)
	}
	return out
}

// Действия (x-ant-action) операций модуля с доменным гардом.
const (
	ActSign    = "documents.signature.record"
	ActAttest  = "documents.paper.attest"
	ActDecline = "documents.signature.decline"
	// ActSignLegacy — прежний id операции подписи (волна 1).
	ActSignLegacy = "documents.document.sign"
)

// Guard — доменный гард операций модуля documents (AD-39): подпись,
// заверение бумаги, отказ в согласовании. Его вызывают api до записи,
// свёртка при применении (Evaluate) и верификатор.
func Guard(s State, env Env, up Upstream, cmd kernel.Command) error {
	_ = up
	c, ok := cmd.Payload.(SignRequest)
	if !ok {
		return nil
	}
	switch cmd.Action {
	case ActSign, ActAttest, ActSignLegacy:
		return CheckSign(s.Doc(c.DocumentID), c, env.People, s.Participants)
	case ActDecline:
		return CheckDecline(s.Doc(c.DocumentID), c, env.People, s.Participants)
	}
	return nil
}

// Apply применяет намерение, адресованное модулю documents (AD-40):
// documents.Draft запоминает запрос черновика; документ оформляется на
// следующей записи изделия (у Apply нет нормативного слоя). Модули, которым
// документ нужен в том же шаге, передают его данные в своём решении —
// documents читает решение сам (decision.disposition.set, document_id).
func Apply(s State, in kernel.Intent) State {
	if in.Target != Module || in.Name != IntentDraft {
		return s
	}
	c, ok := in.Payload.(DraftContext)
	if !ok {
		return s
	}
	s = s.clone()
	s.Requested = append(s.Requested, c)
	return s
}

// RawData — канонический JSON data реакции (для сравнения в тестах и api).
func RawData(r kernel.Reaction) (json.RawMessage, error) { return Canonical(r.Data) }

// approvalsFor — обязательные подписи документа d по маршруту шаблона t
// (AD-43: одна функция access.RequiredApprovals) на политике env.Policy;
// инициатор — кто запросил документ (для документа выдачи прав он не ставит
// вторую подпись, AD-11).
func approvalsFor(t Template, d *Doc, env Env) []access.ApprovalStage {
	c := access.ApprovalContext{Decision: d.Context.Decision, CustomerAcceptance: d.Context.CustomerAcceptance, Initiator: d.Context.Author}
	return access.RequiredApprovals(t.Route, c, env.Policy)
}
