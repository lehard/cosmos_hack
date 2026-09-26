package security

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
	dj "ant/internal/domain/journal"
	dom "ant/internal/domain/security"
)

// Service — реализация live ведущих портов модуля security (AD-36): журнал
// критических действий (цепочка ca), шина безопасности, индикатор
// целостности и отчёты верификатора — из журнала и у хранителя. Только
// чтение: записей CA через API не создают (AD-28). Без журнала
// (NewService()) — заглушка 501, как в волне 1.
type Service struct {
	Unimplemented
	journal appjournal.JournalStore
	keeper  Keeper
	// interval — интервал проверок верификатора: индикатор желтеет сам, если
	// свежего отчёта нет дольше двух интервалов (AD-46).
	interval time.Duration
	now      func() time.Time
}

// Option — настройка Service.
type Option func(*Service)

// WithJournal — журнал (чтение цепочек main и ca).
func WithJournal(j appjournal.JournalStore) Option { return func(s *Service) { s.journal = j } }

// WithKeeper — хранитель (отчёты верификатора целиком).
func WithKeeper(k Keeper) Option { return func(s *Service) { s.keeper = k } }

// WithInterval — интервал проверок верификатора.
func WithInterval(d time.Duration) Option { return func(s *Service) { s.interval = d } }

// NewService создаёт реализацию live; без WithJournal — заглушка 501.
func NewService(opts ...Option) *Service {
	s := &Service{interval: time.Minute, now: time.Now}
	for _, o := range opts {
		o(s)
	}
	return s
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// readAll — все записи по запросу (страницами).
func (s *Service) readAll(ctx context.Context, q appjournal.ReadQuery) ([]jc.JournalEntry, error) {
	var out []jc.JournalEntry
	q.Limit = 1000
	for {
		page, err := s.journal.Read(ctx, q)
		if err != nil {
			return nil, err
		}
		out = append(out, page...)
		if len(page) < q.Limit {
			return out, nil
		}
		q.AfterSeq = int64(page[len(page)-1].Seq)
	}
}

// Integrity — индикатор целостности «по данным сервера» (AD-46): последний
// security.integrity.checked; нет отчёта дольше двух интервалов — stale.
func (s *Service) Integrity(ctx context.Context) (IntegrityStatus, error) {
	if s.journal == nil {
		return s.Unimplemented.Integrity(ctx)
	}
	st := IntegrityStatus{Status: "unknown", IntervalSeconds: max(int(s.interval/time.Second), 1), ServerSide: true}
	es, err := s.journal.Read(ctx, appjournal.ReadQuery{EventType: string(catalog.SecurityIntegrityChecked), Backward: true, Limit: 1})
	if err != nil || len(es) == 0 {
		return st, err
	}
	ev, err := open(ctx, s.journal, es[0])
	if err != nil {
		return st, err
	}
	var d struct {
		ReportDigest string `json:"report_digest"`
		Verdict      string `json:"verdict"`
	}
	_ = json.Unmarshal(ev.Data, &d)
	at, _ := dj.ParseTime(es[0].OccurredAt)
	st.CheckedAt, st.ReportRef = &at, d.ReportDigest
	switch {
	case d.Verdict == "violated":
		st.Status = "violated"
	case s.now().Sub(at) > 2*s.interval:
		st.Status = "stale"
	default:
		st.Status = "ok"
	}
	return st, nil
}

// caEntry — запись CA с разобранным содержимым.
type caEntry struct {
	entry jc.JournalEntry
	rec   dom.Record
}

func (s *Service) caLog(ctx context.Context, m platform.Moment) ([]caEntry, error) {
	es, err := s.readAll(ctx, appjournal.ReadQuery{Chain: string(jc.JournalEntryChainCa), Moment: m})
	if err != nil {
		return nil, err
	}
	out := make([]caEntry, 0, len(es))
	for _, e := range es {
		ev, err := open(ctx, s.journal, e)
		if err != nil {
			return nil, err
		}
		var r dom.Record
		if err := json.Unmarshal(ev.Data, &r); err != nil {
			return nil, err
		}
		out = append(out, caEntry{entry: e, rec: r})
	}
	return out, nil
}

func toView(c caEntry, cancelledBy map[string]string) CriticalAction {
	r := c.rec
	no := int64(c.entry.Seq)
	v := CriticalAction{CARef: dom.Ref(no), CANo: no, CAGroup: r.CAGroup, ActionType: r.ActionType, ObjectRef: r.ObjectRef,
		Before: r.Before, After: r.After, ActorID: r.ActorID, AttestedBy: r.AttestedBy, AuthorityID: r.AuthorityID,
		StampID: r.StampID, BasisEventIDs: r.BasisEventIDs, MainEventID: r.MainEventID, MainCommit: r.MainCommit,
		Cancels: r.Cancels, CancelledBy: cancelledBy[dom.Ref(no)]}
	if v.BasisEventIDs == nil {
		v.BasisEventIDs = []string{}
	}
	if r.PolicySeq > 0 {
		p := r.PolicySeq
		v.PolicySeq = &p
	}
	if r.CancelReason != nil {
		v.CancelReason = r.CancelReason.Text
	}
	v.RecordedAt, _ = dj.ParseTime(c.entry.RecordedAt)
	if a, _, ok := dom.ActionFor(catalog.Type(r.ActionType)); ok {
		v.ActionName = a.Name
	}
	v.Object = platform.DrillRef{Entity: platform.EntityIntegrity, ID: dom.Ref(no)}
	if kind, id, ok := strings.Cut(r.ObjectRef, ":"); ok && id != "" {
		for _, k := range platform.EntityKinds {
			if string(k) == kind {
				v.Object = platform.DrillRef{Entity: k, ID: id}
			}
		}
	}
	return v
}

// CriticalActions — журнал критических действий на момент (AD-28, FR-77),
// новые сверху; курсор — номер CA, после которого читать дальше.
func (s *Service) CriticalActions(ctx context.Context, f CriticalActionFilter, m platform.Moment, p platform.Page) (CriticalActionList, error) {
	if s.journal == nil {
		return s.Unimplemented.CriticalActions(ctx, f, m, p)
	}
	log, err := s.caLog(ctx, m)
	if err != nil {
		return CriticalActionList{}, err
	}
	cancelled := map[string]string{}
	for _, c := range log {
		if c.rec.Cancels != "" {
			cancelled[c.rec.Cancels] = dom.Ref(int64(c.entry.Seq))
		}
	}
	before := int64(0)
	if p.Cursor != "" {
		before, _ = strconv.ParseInt(p.Cursor, 10, 64)
	}
	limit := cmp.Or(p.Limit, 50)
	out := CriticalActionList{Items: []CriticalAction{}}
	for i := len(log) - 1; i >= 0; i-- {
		c := log[i]
		v := toView(c, cancelled)
		if before > 0 && v.CANo >= before {
			continue
		}
		if (f.Group != "" && v.CAGroup != f.Group) || (f.ActorID != "" && v.ActorID != f.ActorID) {
			continue
		}
		if f.Object != nil && (v.Object.Entity != f.Object.Entity || (f.Object.ID != "" && v.Object.ID != f.Object.ID)) {
			continue
		}
		if m.RunID != "" && (c.entry.RunID == nil || *c.entry.RunID != m.RunID) {
			continue
		}
		if len(out.Items) == limit {
			out.NextCursor = strconv.FormatInt(out.Items[len(out.Items)-1].CANo, 10)
			break
		}
		s.withSigners(ctx, &v)
		out.Items = append(out.Items, v)
	}
	return out, nil
}

// withSigners — подписанты основной записи критического действия с классом
// ключа (AD-10, Д-72): из конверта записи main_event_id в потоке объекта.
// Запись не нашлась или не читается — подписантов нет (строка CA остаётся).
func (s *Service) withSigners(ctx context.Context, v *CriticalAction) {
	es, err := s.journal.Read(ctx, appjournal.ReadQuery{Stream: v.ObjectRef, EventType: v.ActionType})
	if err != nil {
		return
	}
	for _, e := range es {
		if e.EventID != v.MainEventID {
			continue
		}
		ev, err := open(ctx, s.journal, e)
		if err != nil {
			return
		}
		class := string(e.ProvenanceClass)
		storage := ""
		if ev.Command != nil {
			storage = ev.Command.KeyStorage
		}
		for _, k := range ev.Integrity.Signers {
			id, _, _ := strings.Cut(k, "@")
			sg := CriticalActionSigner{SignerID: id, KeyClass: class, KeyID: k}
			if class == string(jc.JournalEntryProvenanceClassPersonal) {
				sg.KeyStorage = storage
			}
			v.Signers = append(v.Signers, sg)
		}
		if len(v.Signers) == 0 {
			v.Signers = []CriticalActionSigner{{SignerID: e.SourceID, KeyClass: class}}
		}
		return
	}
}

// CriticalAction — запись CA-‹n›.
func (s *Service) CriticalAction(ctx context.Context, caRef string) (CriticalAction, error) {
	if s.journal == nil {
		return s.Unimplemented.CriticalAction(ctx, caRef)
	}
	log, err := s.caLog(ctx, platform.Moment{})
	if err != nil {
		return CriticalAction{}, err
	}
	cancelled := map[string]string{}
	for _, c := range log {
		if c.rec.Cancels != "" {
			cancelled[c.rec.Cancels] = dom.Ref(int64(c.entry.Seq))
		}
	}
	for _, c := range log {
		if dom.Ref(int64(c.entry.Seq)) == caRef {
			v := toView(c, cancelled)
			s.withSigners(ctx, &v)
			return v, nil
		}
	}
	return CriticalAction{}, platform.Fail(errcodes.ApiNotFound)
}

// Events — шина безопасности на момент (AD-24, FR-118), новые сверху;
// курсор — seq, после которого (вниз) читать дальше.
func (s *Service) Events(ctx context.Context, eventType string, m platform.Moment, p platform.Page) (SecurityEventList, error) {
	if s.journal == nil {
		return s.Unimplemented.Events(ctx, eventType, m, p)
	}
	types := BusTypes
	if eventType != "" {
		types = []catalog.Type{catalog.Type(eventType)}
	}
	var all []jc.JournalEntry
	for _, t := range types {
		es, err := s.readAll(ctx, appjournal.ReadQuery{EventType: string(t), Moment: m})
		if err != nil {
			return SecurityEventList{}, err
		}
		// Стартовая политика генезиса — не «смена прав»: на шине только выдачи после него.
		es = slices.DeleteFunc(es, func(e jc.JournalEntry) bool {
			return e.ProvenanceClass == jc.JournalEntryProvenanceClassGenesis
		})
		all = append(all, es...)
	}
	slices.SortFunc(all, func(a, b jc.JournalEntry) int { return b.Seq - a.Seq })
	before := int64(0)
	if p.Cursor != "" {
		before, _ = strconv.ParseInt(p.Cursor, 10, 64)
	}
	limit := cmp.Or(p.Limit, 50)
	out := SecurityEventList{Items: []SecurityEvent{}}
	for _, e := range all {
		if before > 0 && int64(e.Seq) >= before {
			continue
		}
		if len(out.Items) == limit {
			out.NextCursor = strconv.FormatInt(out.Items[len(out.Items)-1].Seq, 10)
			break
		}
		ev, err := open(ctx, s.journal, e)
		if err != nil {
			return SecurityEventList{}, err
		}
		out.Items = append(out.Items, ToSecurityEvent(e, ev))
	}
	return out, nil
}

// VerifierReports — отчёты верификатора, полученные ant у хранителя
// (записи security.integrity.checked), новые сверху.
func (s *Service) VerifierReports(ctx context.Context, p platform.Page) (VerifierReportList, error) {
	if s.journal == nil {
		return s.Unimplemented.VerifierReports(ctx, p)
	}
	es, err := s.readAll(ctx, appjournal.ReadQuery{EventType: string(catalog.SecurityIntegrityChecked)})
	if err != nil {
		return VerifierReportList{}, err
	}
	limit := cmp.Or(p.Limit, 50)
	before := int64(0)
	if p.Cursor != "" {
		before, _ = strconv.ParseInt(p.Cursor, 10, 64)
	}
	out := VerifierReportList{Items: []VerifierReportSummary{}}
	for i := len(es) - 1; i >= 0; i-- {
		e := es[i]
		if before > 0 && int64(e.Seq) >= before {
			continue
		}
		if len(out.Items) == limit {
			out.NextCursor = strconv.Itoa(es[i+1].Seq)
			break
		}
		ev, err := open(ctx, s.journal, e)
		if err != nil {
			return VerifierReportList{}, err
		}
		var d struct {
			ReportDigest   string `json:"report_digest"`
			Verdict        string `json:"verdict"`
			CheckedUpToSeq int64  `json:"checked_up_to_seq"`
			VerifierBuild  string `json:"verifier_build"`
		}
		_ = json.Unmarshal(ev.Data, &d)
		at, _ := dj.ParseTime(e.OccurredAt)
		out.Items = append(out.Items, VerifierReportSummary{ReportDigest: d.ReportDigest, Verdict: d.Verdict,
			CheckedUpToSeq: d.CheckedUpToSeq, CheckedAt: at, VerifierBuild: d.VerifierBuild, ServerSide: true})
	}
	return out, nil
}

// VerifierReport — отчёт верификатора целиком — у хранителя (первичный
// канал вердикта, AD-46); ant показывает его «по данным сервера».
func (s *Service) VerifierReport(ctx context.Context, digest string) (VerifierReport, error) {
	if s.keeper == nil {
		return s.Unimplemented.VerifierReport(ctx, digest)
	}
	r, err := s.keeper.Report(ctx, digest)
	if errors.Is(err, ErrNotFound) {
		return VerifierReport{}, platform.Fail(errcodes.ApiNotFound)
	}
	if err != nil {
		return VerifierReport{}, err
	}
	return ReportView(r), nil
}

// ReportView — отчёт верификатора в представлении API.
func ReportView(r Report) VerifierReport {
	p := r.Payload
	at, _ := dj.ParseTime(p.GeneratedAt)
	v := VerifierReport{
		Summary: VerifierReportSummary{ReportDigest: r.Digest, Verdict: string(p.Verdict), CheckedUpToSeq: int64(p.Range.MainToSeq),
			CheckedAt: at, VerifierBuild: p.VerifierBuild, ServerSide: true},
		Checks: []VerifierCheckRow{}, PaperDecisions: []platform.DrillRef{}, VirtualTime: p.VirtualTime,
		SignedBy: strings.Join(p.Signers, ", "),
		SignatureClasses: map[string]int{"device": p.SignatureClasses.Device, "personal": p.SignatureClasses.Personal,
			"paper": p.SignatureClasses.Paper, "partner": p.SignatureClasses.Partner, "scenario": p.SignatureClasses.Scenario,
			"genesis": p.SignatureClasses.Genesis, "server_attested": p.SignatureClasses.ServerAttested},
	}
	status := map[string]string{"intact": "intact", "rejected": "rejected", "not_verifiable": "unverifiable"}
	headRejected := false
	for _, c := range p.Checks {
		row := VerifierCheckRow{Check: string(c.Check), Status: status[string(c.Status)], Count: c.Checked}
		var details []string
		for i, f := range c.Findings {
			if i < 20 {
				details = append(details, f.Detail)
				vf := VerifierFinding{Code: f.Code, Detail: f.Detail, Status: map[string]string{"intact": "note"}[string(c.Status)]}
				if vf.Status == "" {
					vf.Status = row.Status
				}
				if f.Chain != nil {
					vf.Chain = string(*f.Chain)
				}
				if f.Seq != nil {
					vf.Seq = int64(*f.Seq)
				}
				if f.CaRef != nil {
					vf.CARef = *f.CaRef
				}
				row.Findings = append(row.Findings, vf)
				// Главная находка — первое нарушение, без нарушений — первая «не проверяемо».
				if row.Status == "rejected" && !headRejected || v.Summary.Headline == "" && row.Status == "unverifiable" {
					v.Summary.Headline, headRejected = f.Detail, row.Status == "rejected"
				}
			}
			if row.CARef == "" && f.CaRef != nil {
				row.CARef = *f.CaRef
			}
		}
		if len(c.Findings) > 20 {
			details = append(details, "… ещё "+strconv.Itoa(len(c.Findings)-20))
		}
		row.Details = strings.Join(details, "\n")
		v.Checks = append(v.Checks, row)
	}
	for _, d := range p.PaperRegister {
		v.PaperDecisions = append(v.PaperDecisions, platform.DrillRef{Entity: platform.EntityDocument, ID: d.DocumentID})
	}
	return v
}
