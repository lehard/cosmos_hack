package documents

import (
	"encoding/json"
	"maps"
	"slices"
	"strconv"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/access"
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality"
)

// Документы позвоночника кроме карты: заявление о несоответствии (запись,
// по подтверждению сигнала), решение по несоответствующей продукции
// (решение: черновик + маршрут, режимы 4–5, AD-43) и документ по запросу
// человека («Запросить решение», лист утверждения версии процесса).

// ID документов, которые строит сама свёртка (детерминированно, AD-4).
const (
	// PrefixTraveler — сопроводительная карта изделия: TRV-‹item_id›.
	PrefixTraveler = "TRV-"
	// PrefixNCStatement — заявление о несоответствии: NCS-‹nc_id›.
	PrefixNCStatement = "NCS-"
)

// TravelerID — id сопроводительной карты изделия.
func TravelerID(itemID string) string { return PrefixTraveler + itemID }

// NCStatementID — id заявления о несоответствии.
func NCStatementID(ncID string) string { return PrefixNCStatement + ncID }

func (s *State) nc(id string) *NCInfo {
	for i := range s.NCs {
		if s.NCs[i].NCID == id {
			return &s.NCs[i]
		}
	}
	return nil
}

// reduceNC — несоответствия изделия: подтверждение, регистрация правилом,
// решение, закрытие.
func (s *State) reduceNC(r kernel.Record, env Env, up Upstream) {
	switch r.Type {
	case catalog.DecisionNonconformityConfirmed:
		d, err := kernel.Decode[ev.DecisionNonconformityConfirmedV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		n := s.nc(string(d.NcID))
		if n == nil {
			s.NCs = append(s.NCs, NCInfo{NCID: string(d.NcID), Number: NCNumber(string(d.NcID)), Sources: []Ref{}})
			n = &s.NCs[len(s.NCs)-1]
		}
		n.Status, n.Severity, n.Reason = "confirmed", string(d.Severity), string(d.Reason.Text)
		if d.RequirementRef != nil {
			n.Requirement = *d.RequirementRef
		}
		if d.DefectTypeCode != nil {
			n.DefectType = *d.DefectTypeCode
		}
		sig := sigOf(r)
		n.Confirmed, n.ConfirmedAt = &sig, FormatTime(r.OccurredAt)
		n.Sources = append(n.Sources, refOf(r))
		for _, sid := range d.SignalIds {
			s.fromSignal(n, string(sid), up)
		}
		if n.FoundBy == "" {
			n.FoundBy = "контролёр " + sig.Person
		}
		s.draft(env, s.ncStatement(env, n), r, up)
	case catalog.DecisionNonconformityRegistered:
		var d struct {
			NCID    string `json:"nc_id"`
			StepKey string `json:"step_key"`
			RunID   string `json:"operation_run_id"`
		}
		if json.Unmarshal(r.Data, &d) != nil || d.NCID == "" {
			s.bad(r)
			return
		}
		if s.nc(d.NCID) != nil {
			return
		}
		s.NCs = append(s.NCs, NCInfo{NCID: d.NCID, Number: NCNumber(d.NCID), Status: "confirmed", StepKey: d.StepKey, RunID: d.RunID,
			Severity: "unknown", FoundBy: "правило специального процесса: нарушение режима оборудования (FR-151)", FoundAt: FormatTime(r.OccurredAt),
			FoundSource: r.EventID, Fact: "операция выполнена в окне нарушения режима специального процесса", Sources: []Ref{refOf(r)}})
		s.draft(env, s.ncStatement(env, &s.NCs[len(s.NCs)-1]), r, up)
	case catalog.DecisionDispositionSet:
		d, err := kernel.Decode[ev.DecisionDispositionSetV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		n := s.nc(string(d.NcID))
		if n == nil {
			s.NCs = append(s.NCs, NCInfo{NCID: string(d.NcID), Number: NCNumber(string(d.NcID)), Status: "confirmed", Sources: []Ref{}})
			n = &s.NCs[len(s.NCs)-1]
		}
		n.Disposition = string(d.Disposition)
		if d.ConcessionID != nil {
			n.Concession = string(*d.ConcessionID)
		}
		n.Sources = append(n.Sources, refOf(r))
		if d.DocumentID == nil || *d.DocumentID == "" {
			return
		}
		n.DocumentID = string(*d.DocumentID)
		t, ok := env.Templates.ByRef(TemplateNCDisposition)
		if !ok || !t.Complete() || s.doc(n.DocumentID) != nil {
			return
		}
		c := DocContext{Decision: n.Disposition, Author: PersonOf(r.Actor), At: FormatTime(r.OccurredAt), Reason: string(d.Reason.Text),
			Concession: n.Concession, SourceEvent: r.EventID}
		if d.ScrapKind != nil {
			c.ScrapKind = string(*d.ScrapKind)
		}
		if d.ClaimBasis != nil {
			c.Claim = *d.ClaimBasis
		}
		s.Docs = append(s.Docs, Doc{ID: n.DocumentID, TemplateRef: t.Ref(), DocType: t.DocType, Class: t.Class, Title: t.Title,
			Subject: "nonconformity:" + n.NCID, Key: n.NCID, Context: c})
		s.draft(env, &s.Docs[len(s.Docs)-1], r, up)
	case catalog.DecisionNonconformityClosed:
		var d struct {
			NCID string `json:"nc_id"`
		}
		if json.Unmarshal(r.Data, &d) == nil {
			if n := s.nc(d.NCID); n != nil {
				n.Status = "closed"
				n.Sources = append(n.Sources, refOf(r))
			}
		}
	}
}

// fromSignal — что обнаружено: сигнал quality (раньше documents в
// композиции, AD-40) и наблюдение, по которому он поднят.
func (s *State) fromSignal(n *NCInfo, signalID string, up Upstream) {
	if up.Quality == nil {
		return
	}
	i := slices.IndexFunc(up.Quality.Signals, func(x quality.Signal) bool { return x.SignalID == signalID })
	if i < 0 {
		return
	}
	sg := up.Quality.Signals[i]
	if n.StepKey == "" {
		n.StepKey = sg.StepKey
	}
	if n.Requirement == "" {
		n.Requirement = sg.RequirementRef
	}
	if n.DefectType == "" && sg.TypeKnown {
		n.DefectType = sg.TypeCode
	}
	fact := "признак дефекта"
	if sg.TypeCode != "" {
		fact += " «" + sg.TypeCode + "»"
	}
	if sg.Zone != "" {
		fact += " в зоне " + sg.Zone
	}
	if sg.Unable {
		fact = "оценка невозможна: " + sg.UnableReason
	}
	n.Fact = fact
	for _, o := range up.Quality.Observations {
		if o.EventID != sg.ObservationID {
			continue
		}
		n.FoundBy = "контроль «" + o.Method + "»"
		if o.SourceKind != "" {
			n.FoundBy += " (" + o.SourceKind + ")"
		}
		n.FoundAt, n.FoundSource = FormatTime(o.OccurredAt), o.EventID
		n.RunID = o.OperationRunID
		n.Sources = append(n.Sources, Ref{ID: o.EventID, At: o.OccurredAt})
	}
	for _, o := range sg.Observations {
		n.Evidence = append(n.Evidence, "наблюдение "+o)
	}
}

// ncStatement — документ «Заявление о несоответствии» несоответствия n.
func (s *State) ncStatement(env Env, n *NCInfo) *Doc {
	id := NCStatementID(n.NCID)
	if d := s.doc(id); d != nil {
		return d
	}
	t, ok := env.Templates.ByRef(TemplateNCStatement)
	if !ok || !t.Complete() {
		return nil
	}
	s.Docs = append(s.Docs, Doc{ID: id, TemplateRef: t.Ref(), DocType: t.DocType, Class: t.Class, Title: t.Title,
		Subject: "nonconformity:" + n.NCID, Key: n.NCID})
	return &s.Docs[len(s.Docs)-1]
}

// ncStatementBody — содержимое заявления и подписанты этапов by_source:
// обнаруживший (сигнал, наблюдение) и подтвердивший контролёр.
func (s *State) ncStatementBody(d *Doc) (map[string]any, []Ref, []Signature) {
	n := s.nc(d.Key)
	if n == nil {
		return nil, nil, nil
	}
	confirmedBy := ""
	if n.Confirmed != nil {
		confirmedBy = n.Confirmed.Person
	}
	body := map[string]any{
		"item": map[string]any{"item_id": s.itemID(), "item_type_id": s.Header.ItemTypeID},
		"nc": map[string]any{
			"nc_id": n.NCID, "number": n.Number, "found_at": n.FoundAt, "found_by": n.FoundBy, "stage": s.stageLabel(n),
			"requirement": n.Requirement, "fact": n.Fact, "severity": n.Severity, "defect_type": n.DefectType,
			"evidence": nonNil(n.Evidence), "confirmed_by": confirmedBy, "confirmed_at": n.ConfirmedAt, "reason": n.Reason,
		},
	}
	var signers []Signature
	switch {
	case n.FoundSource != "":
		signers = append(signers, Signature{EventID: n.FoundSource, Stage: 1, Method: MethodSource, Provenance: "device"})
	case n.Confirmed != nil:
		// Источник сигнала не известен свёртке (нет наблюдения): обнаружившим
		// считается зарегистрировавший несоответствие контролёр.
		signers = append(signers, Signature{EventID: n.Confirmed.EventID, Stage: 1, Person: n.Confirmed.Person, Method: MethodSource,
			Level: n.Confirmed.Level, Provenance: n.Confirmed.Provenance})
	}
	if n.Confirmed != nil {
		signers = append(signers, Signature{EventID: n.Confirmed.EventID, Stage: 2, Person: n.Confirmed.Person, Method: MethodSource,
			Level: n.Confirmed.Level, Provenance: n.Confirmed.Provenance})
	}
	return body, slices.Clone(n.Sources), signers
}

// stageLabel — этап обнаружения: операция и выполнение.
func (s *State) stageLabel(n *NCInfo) string {
	if n.StepKey == "" {
		return ""
	}
	l := n.StepKey
	if r := s.row(n.RunID, n.StepKey); r != nil && r.RunID != "" {
		l += " (выполнение " + r.RunID + ")"
	}
	return l
}

// dispositionBody — содержимое решения по несоответствующей продукции и
// подписант этапа by_source — автор решения.
func (s *State) dispositionBody(d *Doc) (map[string]any, []Ref, []Signature) {
	n := s.nc(d.Key)
	if n == nil {
		return nil, nil, nil
	}
	c := d.Context
	body := map[string]any{
		"item": map[string]any{"item_id": s.itemID()},
		"nc":   map[string]any{"nc_id": n.NCID, "number": n.Number, "requirement": n.Requirement, "severity": n.Severity},
		"decision": map[string]any{
			"disposition": c.Decision, "label": label(dispositionLabel, c.Decision), "scrap_kind": c.ScrapKind, "concession": c.Concession,
			"claim": c.Claim, "reason": c.Reason, "author": c.Author, "at": c.At,
		},
	}
	src := slices.Clone(n.Sources)
	signers := []Signature{{EventID: c.SourceEvent, Stage: 1, Person: c.Author, Method: MethodSource, Level: 2, Provenance: "personal"}}
	return body, src, signers
}

// genericBody — документ по запросу человека: объект, решение, комментарий,
// основания (FR-146, FR-136).
func genericBody(d *Doc) (map[string]any, []Ref) {
	c := d.Context
	return map[string]any{
		"subject": d.Subject, "decision": c.Decision, "comment": c.Comment, "requested_by": c.Author,
		"requested_at": c.At, "sources": nonNil(c.Sources),
	}, nil
}

func (s *State) itemID() string {
	if s.Header.ItemID != "" {
		return s.Header.ItemID
	}
	return s.ItemID
}

// body — содержательная часть документа, её источники и подписанты by_source.
func (s *State) body(d *Doc, env Env) (map[string]any, []Ref, []Signature) {
	switch d.DocType {
	case DocTraveler:
		b, src := s.travelerBody(env)
		return b, src, nil
	case DocNCStatement:
		return s.ncStatementBody(d)
	case DocNCDisposition:
		return s.dispositionBody(d)
	default:
		b, src := genericBody(d)
		return b, src, nil
	}
}

// Compose — канонический content версии: содержательная часть + шапка
// (вид, название, id, версия, шаблон, объект) + замороженный маршрут + поля
// сводки уровня 2 (AD-12, AD-43). Одна функция для свёртки и api.
func Compose(t Template, d *Doc, no int, body map[string]any, stages []access.ApprovalStage) (Built, error) {
	content := map[string]any{}
	for _, k := range sortedKeys(body) {
		content[k] = body[k]
	}
	if stages == nil {
		stages = []access.ApprovalStage{}
	}
	content["doc_type"], content["title"], content["document_id"] = t.DocType, t.Title, d.ID
	content["version"], content["template_ref"], content["subject"], content["approvals"] = no, t.Ref(), d.Subject, stages
	pre, err := Canonical(content)
	if err != nil {
		return Built{}, err
	}
	sum := make([]map[string]string, 0, len(t.Summary))
	for _, f := range t.Summary {
		sum = append(sum, map[string]string{"label": f.Label, "value": SummaryValue(pre, f.Key)})
	}
	content["summary"] = sum
	return Build(t, content)
}

func sortedKeys(m map[string]any) []string { return slices.Sorted(maps.Keys(m)) }

// draft — новая версия документа d по записи-триггеру r, если содержимое
// изменилось (AD-12: новая версия — новый отпечаток). Набор обязательных
// подписей вычисляется здесь один раз (AD-43).
func (s *State) draft(env Env, d *Doc, r kernel.Record, _ Upstream) {
	if d == nil {
		return
	}
	t, ok := env.Templates.ByRef(d.TemplateRef)
	if !ok {
		return
	}
	body, sources, signers := s.body(d, env)
	if body == nil {
		return
	}
	raw, err := Canonical(body)
	if err != nil {
		s.bad(r)
		return
	}
	bh := Hash(raw)
	cur := d.Current()
	if cur != nil && cur.BodyHash == bh && !cur.Annulled && len(cur.Declines) == 0 {
		return
	}
	no := 1
	if cur != nil {
		no = cur.No + 1
	}
	stages := approvalsFor(t, d, env)
	b, err := Compose(t, d, no, body, stages)
	if err != nil {
		s.bad(r)
		return
	}
	sources = append(sources, refOf(r))
	v := Version{No: no, Content: b.Content, BodyHash: bh, RenderingHash: b.RenderingHash, Digest: b.Digest, Stages: stages,
		Sources: uniqueRefs(sources), Trigger: refOf(r), At: r.OccurredAt}
	if cur != nil {
		v.Supersedes = cur.No
	}
	for _, sg := range signers {
		i := stageIndex(stages, sg.Stage)
		if i < 0 || !stages[i].BySource || sg.EventID == "" {
			continue
		}
		sg.Version, sg.Digest, sg.At = no, b.Digest, r.OccurredAt
		v.SourceSigners = append(v.SourceSigners, sg)
	}
	d.Versions = append(d.Versions, v)
	s.evaluate(env, d, len(d.Versions)-1, r)
}

// uniqueRefs — ссылки без повторов, по id (порядок причин реакции, AD-3).
func uniqueRefs(rs []Ref) []Ref {
	out := slices.Clone(rs)
	slices.SortFunc(out, func(a, b Ref) int {
		switch {
		case a.ID < b.ID:
			return -1
		case a.ID > b.ID:
			return 1
		}
		return 0
	})
	return slices.CompactFunc(out, func(a, b Ref) bool { return a.ID == b.ID })
}

// evaluate — пересчёт маршрута версии i документа d после подписи или
// черновика: маршрут закрыт — отметка и причины для document.route.closed.
func (s *State) evaluate(env Env, d *Doc, i int, r kernel.Record) {
	v := &d.Versions[i]
	if v.Closed {
		return
	}
	e := Evaluate(v, env.People, s.Participants)
	if !e.Closed {
		return
	}
	v.Closed, v.Verification = true, env.verification()
	v.ClosedBy = nil
	causes := []Ref{v.Trigger}
	for _, st := range e.Stages {
		for _, sg := range st.Counted {
			v.ClosedBy = append(v.ClosedBy, sg.EventID)
			causes = append(causes, Ref{ID: sg.EventID, At: sg.At})
		}
	}
	causes = append(causes, refOf(r))
	v.ClosedCauses = uniqueRefs(causes)
	v.ClosedAt = r.OccurredAt
}

// VersionLabel — «‹id› версия N» для людей.
func VersionLabel(d *Doc, no int) string { return d.ID + " версия " + strconv.Itoa(no) }
