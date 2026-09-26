package documents

import (
	"strings"
	"testing"

	"ant/internal/domain/access"
)

func people() People {
	return People{
		{ID: "INS-02", Roles: []string{"quality_inspector"}, Authorities: []string{"qc_acceptance"}, Stamps: []Stamp{{ID: "S-FINAL", Kind: "final"}}},
		{ID: "INS-01", Roles: []string{"quality_inspector"}, Authorities: []string{"qc_acceptance"}},
		{ID: "FOR-WC", Roles: []string{"site_foreman"}, Authorities: []string{"paper_attestation"}},
		{ID: "W21", Roles: []string{"performer"}},
	}
}

func stages() []access.ApprovalStage {
	return access.RequiredApprovals([]access.RouteStage{
		{Stage: 1, Title: "ОТК", Role: "quality_inspector", AuthorityID: "qc_acceptance", StampKind: "final", Quorum: "one", SignatureLevel: 2,
			PaperAllowed: true, AttesterAuthorityID: "paper_attestation", Separation: []string{access.SeparationDistinctSigners, access.SeparationNotItemParticipant}},
		{Stage: 2, Title: "Мастер", Role: "site_foreman", AuthorityID: "site_foreman", Quorum: "one", SignatureLevel: 2, Separation: []string{access.SeparationDistinctSigners}},
		{Stage: 3, Title: "Только для ремонта", AuthorityID: "x", Quorum: "one", SignatureLevel: 2, When: &access.StageCondition{Decisions: []string{"repair"}}},
	}, access.ApprovalContext{Decision: "scrap"}, access.Policy{})
}

// AD-43: подпись засчитывается только над текущим отпечатком, с полномочием,
// клеймом, уровнем и разделением обязанностей; бумага — с заверителем ≠
// подписанту; этап по условию режима не входит в набор.
func TestEvaluate(t *testing.T) {
	st := stages()
	if len(st) != 2 || st[1].PaperAllowed {
		t.Fatalf("набор: %+v", st)
	}
	v := &Version{No: 2, Digest: "D2", Stages: st}
	sig := func(id, who, method string, stage int, digest string) Signature {
		return Signature{EventID: id, Version: 2, Stage: stage, Person: who, Method: method, Digest: digest, Level: 2}
	}
	v.Signatures = []Signature{
		sig("a", "INS-02", MethodDemo, 1, "D1"), // прежний отпечаток
		sig("b", "INS-01", MethodDemo, 1, "D2"), // нет клейма final
		sig("c", "W21", MethodDemo, 1, "D2"),    // участник изготовления и не контролёр
		sig("d", "INS-02", MethodDemo, 1, "D2"), // засчитана
		sig("e", "INS-02", MethodDemo, 2, "D2"), // не мастер
		{EventID: "f", Version: 2, Stage: 2, Person: "FOR-WC", Method: MethodPaper, Digest: "D2", Level: 2, AttestedBy: "FOR-WC"}, // бумага запрещена на этапе 2
	}
	ev := Evaluate(v, people(), []string{"W21"})
	want := map[string]string{"a": WhyPriorVersion, "b": WhyNoStamp, "c": WhyAuthority, "d": "", "e": WhyAuthority, "f": WhyPaperForbidden}
	for _, x := range ev.Verdicts {
		if x.Why != want[x.EventID] || x.Counted != (want[x.EventID] == "") {
			t.Errorf("%s: %+v, ожидалось %q", x.EventID, x, want[x.EventID])
		}
	}
	if ev.Closed || ev.Next != 2 {
		t.Fatalf("маршрут: закрыт=%v, следующий этап %d", ev.Closed, ev.Next)
	}
	v.Signatures = append(v.Signatures, sig("g", "FOR-WC", MethodDemo, 2, "D2"))
	if ev := Evaluate(v, people(), []string{"W21"}); !ev.Closed {
		t.Fatalf("маршрут не закрыт: %+v", ev.Verdicts)
	}
	// Один человек — два этапа: второй не засчитан (distinct_signers).
	both := append(people(), Person{ID: "BOTH", Roles: []string{"quality_inspector", "site_foreman"}, Authorities: []string{"qc_acceptance"},
		Stamps: []Stamp{{ID: "S2", Kind: "final"}}})
	dv := &Version{No: 1, Digest: "D", Stages: st, Signatures: []Signature{
		{EventID: "x1", Version: 1, Stage: 1, Person: "BOTH", Method: MethodDemo, Digest: "D", Level: 2},
		{EventID: "x2", Version: 1, Stage: 2, Person: "BOTH", Method: MethodDemo, Digest: "D", Level: 2},
	}}
	if ev := Evaluate(dv, both, nil); !ev.Verdicts[0].Counted || ev.Verdicts[1].Why != WhyDistinct || ev.Closed {
		t.Fatalf("разделение: %+v", ev.Verdicts)
	}
	// Бумага на этапе 1: заверитель = подписант — не засчитана; заверитель с полномочием — засчитана.
	p := &Version{No: 1, Digest: "D", Stages: st[:1], Signatures: []Signature{
		{EventID: "p1", Version: 1, Stage: 1, Person: "INS-02", Method: MethodPaper, Digest: "D", Level: 2, AttestedBy: "INS-02"},
		{EventID: "p2", Version: 1, Stage: 1, Person: "INS-02", Method: MethodPaper, Digest: "D", Level: 2, AttestedBy: "FOR-WC"},
	}}
	ev = Evaluate(p, people(), nil)
	if ev.Verdicts[0].Why != WhyAttesterSigner || !ev.Verdicts[1].Counted || !ev.Closed {
		t.Fatalf("бумага: %+v", ev.Verdicts)
	}
}

// Отрисовка детерминирована и экранирует данные; маршрут входит в документ.
func TestRenderEscapes(t *testing.T) {
	tp := Template{ID: "t", Version: 1, Title: "Т", DocType: DocGeneric, Layout: []Section{
		{Kind: "fields", Title: "Поля", Fields: []Field{{Key: "a.b", Label: "Поле"}, {Key: "nope", Label: "Нет"}}},
		{Kind: "table", Rows: "rows", Columns: []Field{{Key: "x", Label: "X"}}},
		{Kind: "route", Title: "Подписи"},
	}}
	d := &Doc{ID: "DOC-1", Subject: "other:1"}
	b1, err := Compose(tp, d, 1, map[string]any{"a": map[string]any{"b": "<script>&"}, "rows": []any{map[string]any{"x": []string{"1", "2"}}}}, stages())
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := Compose(tp, d, 1, map[string]any{"rows": []any{map[string]any{"x": []string{"1", "2"}}}, "a": map[string]any{"b": "<script>&"}}, stages())
	if b1.Digest != b2.Digest || b1.HTML != b2.HTML {
		t.Fatal("отрисовка зависит от порядка ключей")
	}
	for _, want := range []string{"&lt;script&gt;&amp;", "<dd>" + Empty + "</dd>", "<td>1; 2</td>", "Этап 1. ОТК", "клеймо «final»"} {
		if !strings.Contains(b1.HTML, want) {
			t.Errorf("нет %q в отрисовке:\n%s", want, b1.HTML)
		}
	}
	b3, _ := Compose(tp, d, 2, map[string]any{"a": map[string]any{"b": "<script>&"}, "rows": []any{}}, stages())
	if b3.Digest == b1.Digest {
		t.Fatal("новая версия — тот же отпечаток")
	}
}
