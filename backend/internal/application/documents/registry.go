package documents

import (
	"context"
	"slices"
	"strings"

	"ant/internal/application/platform"
	dom "ant/internal/domain/documents"
)

// Реестр документов (раздел «Документы» столов технолога, начальника ОТК,
// руководителя производства, администратора, аудитора ИБ; FR-65, FR-66,
// AD-12, AD-43): все документы, собранные из журнала, с отбором по объекту,
// изделию, процессу, виду и состоянию. Отбор и состояние — одна функция для
// live и заготовок, чтобы фильтры вели себя одинаково.

// Состояния реестра (DocumentSummary.State).
const (
	StateDraft    = "draft"
	StateSigning  = "signing"
	StateSigned   = "signed"
	StateAnnulled = "annulled"
	StateReturned = "returned"
	StatePaper    = "paper"
)

// RegistryState — состояние документа для людей по статусу маршрута и
// бумажного экземпляра: напечатанный и ещё не подписанный по маршруту
// документ — «на бумаге» (ждёт подписи ручкой и заверения скана, AD-43).
func RegistryState(status, paperStatus string) string {
	switch status {
	case dom.StatusAnnulled:
		return StateAnnulled
	case dom.StatusReturned:
		return StateReturned
	case dom.StatusRouteClosed:
		return StateSigned
	}
	if paperStatus == "printed" {
		return StatePaper
	}
	if status == dom.StatusSigning {
		return StateSigning
	}
	return StateDraft
}

// templateID — id шаблона из template_ref `‹id›@‹версия›`.
func templateID(ref string) string {
	id, _, _ := strings.Cut(ref, "@")
	return id
}

// Match — подходит ли документ под отбор реестра.
func (f DocumentFilter) Match(d DocumentSummary) bool {
	if f.Subject.ID != "" && (d.Subject.ID != f.Subject.ID || (f.Subject.Entity != "" && d.Subject.Entity != f.Subject.Entity)) {
		return false
	}
	if f.ItemID != "" && !slices.Contains(d.ItemIDs, f.ItemID) && (d.Subject.Entity != platform.EntityItem || d.Subject.ID != f.ItemID) {
		return false
	}
	if f.ProcessID != "" && d.ProcessID != f.ProcessID {
		return false
	}
	if f.ProcessVersionID != "" && d.ProcessVersionID != f.ProcessVersionID {
		return false
	}
	if f.Template != "" && d.Template != f.Template && templateID(d.Template) != f.Template {
		return false
	}
	if f.State != "" && d.State != f.State {
		return false
	}
	if q := strings.ToLower(strings.TrimSpace(f.Q)); q != "" {
		hay := strings.ToLower(strings.Join(append([]string{d.DocumentID, d.Title, d.SubjectLabel, d.Subject.ID, d.Template}, d.ItemIDs...), " "))
		if !strings.Contains(hay, q) {
			return false
		}
	}
	return true
}

// FilterDocuments — документы реестра, подходящие под отбор; счётчик FR-65
// «собрано из истории» — по отобранным.
func FilterDocuments(l DocumentList, f DocumentFilter) DocumentList {
	out := DocumentList{Items: []DocumentSummary{}}
	for _, d := range l.Items {
		if d.State == "" {
			d.State = RegistryState(d.Status, d.PaperStatus)
		}
		if f.Match(d) {
			out.Items = append(out.Items, d)
		}
	}
	n, zero := len(out.Items), 0
	out.CollectedFromHistory, out.ManualEntries = &n, &zero
	return out
}

// Registry — реестр документов (documents.document.list без объекта или с
// отбором): все документы журнала — по записям версий и запросов.
func (s *Service) Registry(ctx context.Context, f DocumentFilter, m platform.Moment, p platform.Page) (DocumentList, error) {
	if !s.live() {
		return s.Unimplemented.Registry(ctx, f, m, p)
	}
	ids, err := s.documentIDs(ctx, m)
	if err != nil {
		return DocumentList{}, err
	}
	all := DocumentList{Items: []DocumentSummary{}}
	cache := map[string]*view{}
	for _, id := range ids {
		v, d, err := s.cached(ctx, cache, id, m)
		if err != nil || d == nil {
			continue
		}
		all.Items = append(all.Items, registryRow(d, v))
	}
	return pageOf(FilterDocuments(all, f), p), nil
}

// registryRow — строка реестра: сводка документа, объект, изделие, версия
// процесса, кто должен подписать сейчас и прогресс маршрута.
func registryRow(d *dom.Doc, v *view) DocumentSummary {
	out := summary(d, v)
	out.State = RegistryState(out.Status, out.PaperStatus)
	out.SubjectLabel = out.Subject.ID
	if v.ItemID != "" {
		out.ItemIDs = []string{v.ItemID}
	}
	if out.Subject.Entity == "process_version" {
		out.ProcessVersionID = out.Subject.ID
	}
	cur := d.Current()
	if cur == nil {
		return out
	}
	updated := cur.At
	for _, sg := range cur.Signatures {
		if sg.At.After(updated) {
			updated = sg.At
		}
	}
	for _, dc := range cur.Declines {
		if dc.At.After(updated) {
			updated = dc.At
		}
	}
	if cur.Closed && cur.ClosedAt.After(updated) {
		updated = cur.ClosedAt
	}
	out.UpdatedAt = &updated
	route := routeView(d, cur, v)
	out.StagesTotal = len(route)
	for _, st := range route {
		if st.Done {
			out.StagesDone++
			continue
		}
		if out.Awaiting == nil && !cur.Closed && !cur.Annulled && len(cur.Declines) == 0 {
			out.Awaiting = &DocumentAwaiting{Stage: st.Stage, Title: st.AuthorityLabel, Role: st.Role, Candidates: candidates(st, v)}
		}
	}
	return out
}

// PageDocuments — страница списка документов по курсору (номер строки).
func PageDocuments(l DocumentList, p platform.Page) DocumentList { return pageOf(l, p) }
