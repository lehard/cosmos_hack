package documents

import (
	"context"
	"slices"
	"testing"

	app "ant/internal/application/documents"
	"ant/internal/application/platform"
)

func hdr(id string) platform.CommandHeader { return platform.CommandHeader{CommandID: id} }

func requestIDs(t *testing.T, ctx context.Context) []string {
	t.Helper()
	l, err := New().DecisionRequests(ctx, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, rq := range l.Items {
		out = append(out, rq.Document.DocumentID)
	}
	return out
}

// TestSessionSignatureAndRequest — подпись закрывает этап и убирает запрос у
// подписанта, документ в реестре меняет состояние; отказ возвращает версию;
// «Запросить решение» создаёт документ у уполномоченных (сессионное наложение).
func TestSessionSignatureAndRequest(t *testing.T) {
	a := New()
	cr := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "CR-71", Role: "customer_representative"})
	const doc = "DOC-REQ-CR-RS-01"
	if !slices.Contains(requestIDs(t, cr), doc) {
		t.Fatalf("нет запроса %s у представителя заказчика", doc)
	}
	before, _ := a.Document(cr, doc, 0, platform.Moment{})
	card, err := a.DecisionCard(cr, doc, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.RecordSignature(cr, doc, app.RecordSignature{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000d001"), Version: before.Version, Stage: card.Stage.Stage}); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(requestIDs(t, cr), doc) {
		t.Fatal("подписанный запрос остался у подписанта")
	}
	after, err := a.Document(cr, doc, 0, platform.Moment{})
	if err != nil || len(after.Signatures) != len(before.Signatures)+1 {
		t.Fatalf("подпись не видна в документе: %+v %v", after.Signatures, err)
	}
	reg, err := a.Registry(cr, app.DocumentFilter{}, platform.Moment{}, platform.Page{Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range reg.Items {
		if d.DocumentID == doc && d.Awaiting != nil && slices.Contains(d.Awaiting.Candidates, "CR-71") && d.Awaiting.Stage == card.Stage.Stage {
			t.Fatalf("реестр ждёт подписи прежнего этапа: %+v", d)
		}
	}
	// Отказ: у держателя КД запрос уходит, версия «возвращена».
	da := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "DA-81", Role: "design_authority"})
	const doc2 = "DOC-REQ-DA-F-023"
	c2, err := a.DecisionCard(da, doc2, platform.Moment{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Decline(da, doc2, app.DeclineSignature{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000d002"), Version: c2.Version, Stage: c2.Stage.Stage, Comment: "нет обоснования"}); err != nil {
		t.Fatal(err)
	}
	if v, _ := a.Document(da, doc2, 0, platform.Moment{}); v.Status != "returned" || slices.Contains(requestIDs(t, da), doc2) {
		t.Fatalf("отказ: статус %s", v.Status)
	}
	// «Запросить решение»: контролёр просит решение технолога по несоответствию.
	ins := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "INS-01", Role: "quality_inspector"})
	acc, err := a.RequestVersion(ins, app.RequestVersion{CommandHeader: hdr("0192e4a0-0000-7000-8000-00000000d003"), SubjectRef: "nonconformity:NC-01", Action: "nonconformity.disposition.set", Comment: "прошу решение"})
	if err != nil || acc.DocumentID == "" {
		t.Fatalf("запрос решения: %+v %v", acc, err)
	}
	tec := platform.WithPrincipal(context.Background(), platform.Principal{PersonID: "TEC-01", Role: "technologist"})
	if !slices.Contains(requestIDs(t, tec), acc.DocumentID) {
		t.Fatalf("запрос %s не дошёл до технолога", acc.DocumentID)
	}
	if slices.Contains(requestIDs(t, ins), acc.DocumentID) {
		t.Fatal("запрос виден самому запросившему")
	}
	if _, err := a.Render(tec, acc.DocumentID, 0); err != nil {
		t.Fatal(err)
	}
}
