package world

import (
	"testing"

	docsapp "ant/internal/application/documents"
	"ant/internal/application/platform"
)

// TestDocumentsRegistry — реестр документов мира: на начальном шаге и в конце
// истории есть документы во всех состояниях, строки знают процесс, изделия и
// кто должен подписать; отбор реестра находит документы изделия через его НС.
func TestDocumentsRegistry(t *testing.T) {
	m := buildMain(t)
	tpl, err := LoadTemplates(repoFS())
	if err != nil {
		t.Fatal(err)
	}
	m.templates = tpl
	list := func(n int) docsapp.DocumentList {
		for _, r := range renderDocuments(m.ctx(n)) {
			if r.Op == "documents.document.list" {
				return r.Body.(docsapp.DocumentList)
			}
		}
		t.Fatal("нет documents.document.list")
		return docsapp.DocumentList{}
	}
	states := func(l docsapp.DocumentList) map[string]int {
		out := map[string]int{}
		for _, d := range l.Items {
			out[d.State]++
		}
		return out
	}
	start := states(list(m.Spec.InitialStep))
	for _, s := range []string{docsapp.StateDraft, docsapp.StateSigning, docsapp.StateSigned, docsapp.StateReturned} {
		if start[s] == 0 {
			t.Errorf("шаг %d: нет документов в состоянии %s (%v)", m.Spec.InitialStep, s, start)
		}
	}
	last := list(len(m.Steps) - 1)
	end := states(last)
	for _, s := range []string{docsapp.StateDraft, docsapp.StateSigning, docsapp.StateSigned, docsapp.StateAnnulled, docsapp.StateReturned, docsapp.StatePaper} {
		if end[s] == 0 {
			t.Errorf("конец истории: нет документов в состоянии %s (%v)", s, end)
		}
	}
	for _, d := range last.Items {
		if d.State == docsapp.StateSigning && d.Awaiting == nil {
			t.Errorf("%s на подписи, но не сказано, кто подписывает", d.DocumentID)
		}
		if d.Subject.Entity != platform.EntityProcessVersion && d.ProcessID == "" && d.Subject.Entity != platform.EntityWorkplace {
			t.Errorf("%s без процесса", d.DocumentID)
		}
	}
	f017 := docsapp.FilterDocuments(last, docsapp.DocumentFilter{ItemID: FullID("F-017")})
	found := map[string]bool{}
	for _, d := range f017.Items {
		found[d.DocumentID] = true
	}
	for _, id := range []string{"DOC-NC-01", "DOC-DISP-NC-G1", "DOC-ISO-NC-01", "TRV-" + FullID("F-017")} {
		if !found[id] && !found["DOC-"+id] {
			t.Errorf("документы Ф-017: нет %s (%v)", id, found)
		}
	}
}
