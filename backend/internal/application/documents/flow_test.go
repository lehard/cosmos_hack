package documents_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	app "ant/internal/application/documents"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	"ant/internal/application/ingest"
	appjournal "ant/internal/application/journal"
	ncapp "ant/internal/application/nonconformity"
	"ant/internal/application/nonconformity/nctest"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/documents"
	domingest "ant/internal/domain/ingest"
)

// world — журнал в памяти, воркер движка (свёртка nctest.DraftingFold —
// модель quality) с нормативным слоем documents из репозитория и сервисы
// nonconformity и documents над ними (AD-36: те же порты, что в живом ядре).
type world struct {
	t     *testing.T
	j     *enginemem.Journal
	codec *engineapp.Codec
	w     *engineapp.WorkerService
	feed  *enginemem.Feed
	part  engineapp.Partition
	now   time.Time
	env   dom.Env
	docs  *app.Service
	nc    *ncapp.Service
}

type clock struct{ t *time.Time }

func (c clock) Now(context.Context) (time.Time, error) { return *c.t, nil }

func repoEnv(t *testing.T) dom.Env {
	t.Helper()
	env, err := app.EnvFromFS(os.DirFS("../../../.."), dom.VerificationDemo)
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func newWorld(t *testing.T) *world {
	j := enginemem.New(nil)
	part := engineapp.Partition{Number: 0, Epoch: 1}
	j.SetEpoch(appjournal.PartitionLease(0), 1)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}
	env := repoEnv(t)
	bundles := app.Bundles{Env: env}
	reg := engineapp.NewRegistry()
	if err := app.RegisterProjections(reg); err != nil {
		t.Fatal(err)
	}
	feed := &enginemem.Feed{J: j, Parts: []engineapp.Partition{part}}
	wk := engineapp.NewWorker(engineapp.WorkerConfig{Feed: feed, Codec: codec, Fold: nctest.DraftingFold, Projections: reg, Bundles: bundles})
	w := &world{t: t, j: j, codec: codec, w: wk, feed: feed, part: part, now: nctest.T0.Add(8 * time.Hour), env: env}
	w.docs = app.NewService(app.WithDeps(app.Deps{Journal: j, Codec: codec, Bundles: bundles, Fold: nctest.DraftingFold, Env: env,
		DomainClock: clock{&w.now}, Now: func() time.Time { return w.now }}), app.WithConfig(app.Config{DomainBuild: "streebog256:" + zeros, Partitions: 1}))
	w.nc = ncapp.NewService(ncapp.WithDeps(ncapp.Deps{Journal: j, Codec: codec, Bundles: bundles, Fold: nctest.DraftingFold, Routes: ncapp.PendingRoutes{},
		DomainClock: clock{&w.now}, Now: func() time.Time { return w.now }}), ncapp.WithConfig(ncapp.Config{DomainBuild: "streebog256:" + zeros, Partitions: 1}))
	return w
}

const zeros = "0000000000000000000000000000000000000000000000000000000000000000"

func (w *world) add(ps ...appjournal.Pending) {
	w.t.Helper()
	if _, err := w.j.Append(context.Background(), appjournal.AppendRequest{Batch: ps}); err != nil {
		w.t.Fatal(err)
	}
}

// settle — воркер обрабатывает весь необработанный вход (AD-5).
func (w *world) settle() {
	w.t.Helper()
	for range 20 {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		works, err := w.feed.Next(ctx, w.part)
		cancel()
		if errors.Is(err, context.DeadlineExceeded) {
			return
		}
		if err != nil {
			w.t.Fatal(err)
		}
		if err := w.w.Process(context.Background(), w.part, works); err != nil {
			w.t.Fatal(err)
		}
	}
	w.t.Fatal("воркер не успокоился")
}

func as(person, role string) context.Context {
	return platform.WithPrincipal(context.Background(), platform.Principal{PersonID: person, Role: role})
}

func hdr() platform.CommandHeader { return platform.CommandHeader{CommandID: nctest.ID()} }

func codeOf(err error) errcodes.Code {
	if e, ok := platform.AsError(err); ok {
		return e.Code
	}
	return ""
}

// entries — записи журнала типа t.
func (w *world) entries(t catalog.Type) []string {
	var out []string
	for _, e := range w.j.Entries() {
		if e.EventType == string(t) {
			out = append(out, e.Stream)
		}
	}
	return out
}

// doc — документ по id.
func (w *world) doc(id string, version int) app.DocumentView {
	w.t.Helper()
	v, err := w.docs.Document(context.Background(), id, version, platform.Moment{})
	if err != nil {
		w.t.Fatalf("документ %s@%d: %v", id, version, err)
	}
	return v
}

// run — выполнение операции исполнителем: начало, окончание.
func (w *world) run(item, runID, step, operator string, at time.Time) {
	w.add(nctest.Record(catalog.OperationRunStarted, item, at, map[string]any{"operation_run_id": runID, "operation_code": "020",
		"step_key": step, "equipment_id": "WELD-1", "operator_id": operator, "program_ref": "UP-17"}),
		nctest.Record(catalog.OperationRunFinished, item, at.Add(20*time.Minute), map[string]any{"operation_run_id": runID, "completion": "completed",
			"reported_duration": map[string]any{"value": 20, "unit": "min", "meaning": "active_processing", "origin": "reported_by_source"}}))
}

// Сквозной путь позвоночника (FR-65, AD-12, AD-43): сопроводительная карта
// собирается из истории; подтверждение сигнала даёт заявление о
// несоответствии; решение режима 4 ждёт маршрута подписей и исполняется
// только после document.route.closed; новая версия — новый отпечаток,
// подписи прежней версии остаются при ней.
func TestSpineFlow(t *testing.T) {
	w := newWorld(t)
	const item = "FL:0001"
	w.run(item, "RUN-1", "welding.weld", "W21", nctest.T0)
	w.add(nctest.Inspection(item, nctest.T0.Add(30*time.Minute), "Z-1"))
	w.settle()

	// Живая карта: строка операции с исполнителем, датой, подписью, фактическими параметрами и отметкой контроля.
	trv := w.doc(dom.TravelerID(item), 0)
	if !trv.Live || trv.Status != dom.StatusLive || trv.DocType != dom.DocTraveler {
		t.Fatalf("живая карта: %+v", trv)
	}
	rows, _ := trv.Content["rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("строки карты: %v", trv.Content["rows"])
	}
	row := rows[0].(map[string]any)
	for _, k := range []string{"executor", "date", "signature", "params", "otk"} {
		if v := row[k]; v == nil || v == "" || (isList(v) && len(v.([]any)) == 0) {
			t.Fatalf("в строке карты нет %s: %v", k, row)
		}
	}
	if row["executor"] != "W21" || !strings.Contains(strings.Join(strs(row["params"]), ";"), "WELD-1") || !strings.Contains(row["otk"].(string), "признак дефекта") {
		t.Fatalf("строка карты: %v", row)
	}

	// Подтверждение сигнала → заявление о несоответствии (запись: маршрут закрыт решениями-источниками).
	l, err := w.nc.List(context.Background(), ncapp.NCFilter{ItemID: item}, platform.Moment{}, platform.Page{})
	if err != nil || len(l.Items) != 1 {
		t.Fatalf("несоответствия: %+v %v", l, err)
	}
	ncID := l.Items[0].NCID
	card, _ := w.nc.Card(context.Background(), ncID, platform.Moment{})
	ins := as("INS-01", "quality_inspector")
	if _, err := w.nc.Confirm(ins, ncID, ncapp.ConfirmNonconformity{CommandHeader: hdr(), SignalIDs: []string{card.Evidence.Signals[0].SignalID},
		Severity: "major", Reason: ncapp.NCReason{Text: "Прожог подтверждён"}}); err != nil {
		t.Fatal(err)
	}
	w.settle()
	st := w.doc(dom.NCStatementID(ncID), 0)
	if st.Status != dom.StatusRouteClosed || st.Version != 1 || st.Content["nc"].(map[string]any)["confirmed_by"] != "INS-01" {
		t.Fatalf("заявление: %s v%d %v", st.Status, st.Version, st.Content["nc"])
	}

	// Решение режима 4 («как есть» по разрешению): записано, не исполняется до закрытия маршрута.
	hqc := as("HQC-01", "head_of_qc")
	if _, err := w.nc.GrantConcession(hqc, ncapp.GrantConcession{CommandHeader: hdr(), ConcessionID: "CON-1", Title: "Отклонение катета шва",
		Kind: "use_as_is", ScopeItemIDs: []string{item}, Limit: 1, ValidUntil: "2026-12-31T00:00:00Z", Reason: ncapp.NCReason{Text: "ТУ п. 4.2"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.nc.SetDisposition(ins, ncID, ncapp.SetDisposition{CommandHeader: hdr(), Disposition: "use_as_is", ConcessionID: "CON-1",
		Reason: ncapp.NCReason{Text: "в допуске по разрешению"}}); err != nil {
		t.Fatal(err)
	}
	w.settle()
	card, _ = w.nc.Card(context.Background(), ncID, platform.Moment{})
	if card.Axes.Disposition != "none" || card.ApprovalsStatus == nil || *card.ApprovalsStatus != "pending" {
		t.Fatalf("режим 4 до подписей: %+v %v", card.Axes, card.ApprovalsStatus)
	}
	docID := "ncd-" + ncID
	d := w.doc(docID, 0)
	if d.Status != dom.StatusDrafted || len(d.Route) != 3 || !d.Route[0].Done || d.Route[1].Role != "technologist" || d.Route[2].Role != "head_of_qc" {
		t.Fatalf("маршрут решения: %s %+v", d.Status, d.Route)
	}
	// Запрос решения у технолога (FR-136).
	reqs, err := w.docs.DecisionRequests(as("TEC-01", "technologist"), platform.Moment{})
	if err != nil || len(reqs.Items) != 1 || reqs.Items[0].MyStage == nil || *reqs.Items[0].MyStage != 2 || reqs.Items[0].Escalation.AutomationMode != 4 {
		t.Fatalf("запросы технолога: %+v %v", reqs, err)
	}
	// Порядок этапов, полномочие, подпись.
	sign := func(ctx context.Context, stage int) error {
		_, err := w.docs.RecordSignature(ctx, docID, app.RecordSignature{CommandHeader: hdr(), Version: d.Version, Stage: stage, DocDigest: d.DocDigest})
		return err
	}
	if err := sign(hqc, 3); codeOf(err) != errcodes.DocumentStageNotOpen {
		t.Fatalf("этап 3 раньше этапа 2: %v", err)
	}
	if err := sign(as("W21", "performer"), 2); codeOf(err) != errcodes.AccessSignatureRequired {
		t.Fatalf("не технолог на этапе 2: %v", err)
	}
	if _, err := w.docs.RecordSignature(as("TEC-01", "technologist"), docID, app.RecordSignature{CommandHeader: hdr(), Version: 1, Stage: 2,
		DocDigest: strings.Replace(d.DocDigest, "0", "1", 1)}); codeOf(err) != errcodes.SigningDocumentChanged {
		t.Fatalf("чужой отпечаток: %v", err)
	}
	if err := sign(as("TEC-01", "technologist"), 2); err != nil {
		t.Fatal(err)
	}
	w.settle()
	card, _ = w.nc.Card(context.Background(), ncID, platform.Moment{})
	if card.Axes.Disposition != "none" || len(w.entries(catalog.DocumentRouteClosed)) != 1 {
		t.Fatalf("после этапа 2 решение не исполняется: %+v, закрытых маршрутов %d", card.Axes, len(w.entries(catalog.DocumentRouteClosed)))
	}
	if err := sign(hqc, 3); err != nil {
		t.Fatal(err)
	}
	w.settle()
	closed := w.entries(catalog.DocumentRouteClosed)
	if !slices.Contains(closed, "document:"+docID) {
		t.Fatalf("нет document.route.closed решения: %v", closed)
	}
	card, _ = w.nc.Card(context.Background(), ncID, platform.Moment{})
	if card.Axes.Disposition != "use_as_is" || card.Axes.Quality != "accepted_with_concession" || card.ApprovalsStatus == nil || *card.ApprovalsStatus != "route_closed" {
		t.Fatalf("решение режима 4 после закрытия маршрута: %+v %v", card.Axes, card.ApprovalsStatus)
	}
	if d = w.doc(docID, 0); d.Status != dom.StatusRouteClosed || d.Verification != dom.VerificationDemo || d.RouteClosed == nil {
		t.Fatalf("документ решения: %s %s %v", d.Status, d.Verification, d.RouteClosed)
	}

	// Сопроводительная карта: версия 1 по запросу, подпись контролёра над v1;
	// новая операция → версия 2 с новым отпечатком; подпись v1 остаётся при v1.
	req := func() app.RequestAccepted {
		t.Helper()
		acc, err := w.docs.RequestVersion(as("FOR-WC", "site_foreman"), app.RequestVersion{CommandHeader: hdr(), SubjectRef: "item:" + item, TemplateRef: "traveler"})
		if err != nil {
			t.Fatal(err)
		}
		w.settle()
		return acc
	}
	a1 := req()
	v1 := w.doc(dom.TravelerID(item), 0)
	if v1.Live || v1.Version != 1 || a1.Version != 1 || a1.DocDigest != v1.DocDigest {
		t.Fatalf("карта v1: live=%v v%d, предсказано v%d %s ≠ %s", v1.Live, v1.Version, a1.Version, a1.DocDigest, v1.DocDigest)
	}
	trvID := dom.TravelerID(item)
	if _, err := w.docs.RecordSignature(ins, trvID, app.RecordSignature{CommandHeader: hdr(), Version: 1, Stage: 1}); codeOf(err) != errcodes.AccessNoStamp {
		t.Fatalf("контролёр без клейма «final»: %v", err)
	}
	if _, err := w.docs.RecordSignature(as("INS-02", "quality_inspector"), trvID, app.RecordSignature{CommandHeader: hdr(), Version: 1, Stage: 1}); err != nil {
		t.Fatal(err)
	}
	w.settle()
	w.now = w.now.Add(time.Hour)
	w.run(item, "RUN-2", "welding.weld", "W22", w.now)
	w.now = w.now.Add(time.Hour)
	w.settle()
	req()
	v2 := w.doc(trvID, 0)
	if v2.Version != 2 || v2.DocDigest == v1.DocDigest || v2.SupersedesVersion == nil || *v2.SupersedesVersion != 1 {
		t.Fatalf("карта v2: v%d %s (v1 %s)", v2.Version, v2.DocDigest, v1.DocDigest)
	}
	old := v2.Route[0].Signatures
	if len(old) != 1 || old[0].Counted || !old[0].PreviousVersion || old[0].SignerID != "INS-02" {
		t.Fatalf("подпись v1 в маршруте v2: %+v", old)
	}
	if got := w.doc(trvID, 1); !got.Route[0].Signatures[0].Counted || got.DocDigest != v1.DocDigest {
		t.Fatalf("подпись v1 при v1: %+v", got.Route[0])
	}
	if _, err := w.docs.RecordSignature(as("INS-02", "quality_inspector"), trvID, app.RecordSignature{CommandHeader: hdr(), Version: 1, Stage: 1,
		DocDigest: v1.DocDigest}); codeOf(err) != errcodes.SigningDocumentChanged {
		t.Fatalf("подпись прежней версии: %v", err)
	}
	if _, err := w.docs.RecordSignature(as("W22", "performer"), trvID, app.RecordSignature{CommandHeader: hdr(), Version: 2, Stage: 1}); codeOf(err) == "" {
		t.Fatal("исполнитель подписал итоговую годность")
	}
	if _, err := w.docs.RecordSignature(as("INS-02", "quality_inspector"), trvID, app.RecordSignature{CommandHeader: hdr(), Version: 2, Stage: 1}); err != nil {
		t.Fatal(err)
	}
	w.settle()
	// Этап 2 — на бумаге с заверением (FR-139, AD-43): мастер подписал
	// распечатку с QR, скан заверяет начальник цеха; заверитель ≠ подписант;
	// чужой QR не принимается.
	qr := dom.QR(trvID, v2.DocDigest)
	scan := "streebog256:" + strings.Repeat("ab", 32)
	attest := func(ctx context.Context, signer, qrText string) error {
		_, err := w.docs.AttestPaper(ctx, trvID, app.AttestPaper{CommandHeader: hdr(), Version: 2, Stage: 2, DocDigest: qrText,
			SignerPersonID: signer, ScanAddress: scan, PaperOriginalNo: "ОТК-АРХ-17"})
		return err
	}
	if err := attest(as("FOR-WC", "site_foreman"), "FOR-WC", qr); codeOf(err) != errcodes.SigningAttesterIsSigner {
		t.Fatalf("заверитель = подписант: %v", err)
	}
	if err := attest(as("HWS-WC", "head_of_workshop"), "FOR-WC", dom.QR(trvID, v1.DocDigest)); codeOf(err) != errcodes.SigningQrMismatch {
		t.Fatalf("QR прежней версии: %v", err)
	}
	if err := attest(as("HWS-WC", "head_of_workshop"), "FOR-WC", qr); err != nil {
		t.Fatal(err)
	}
	w.settle()
	if v := w.doc(trvID, 2); v.Status != dom.StatusRouteClosed || v.Route[1].Signatures[0].Class != "paper" || v.Route[1].Signatures[0].AttestedBy != "HWS-WC" {
		t.Fatalf("карта v2: %s %+v", v.Status, v.Route[1].Signatures)
	}

	// Счётчик FR-65 и паспортная проекция. Решение «ремонт» по каталогу (эпик 44)
	// оформляет ещё и разрешение на отклонение — документ по событию-триггеру.
	list, err := w.docs.Documents(context.Background(), platform.DrillRef{Entity: platform.EntityItem, ID: item}, platform.Moment{}, platform.Page{})
	if err != nil || list.CollectedFromHistory == nil || *list.CollectedFromHistory != 4 || *list.ManualEntries != 0 ||
		!slices.ContainsFunc(list.Items, func(d app.DocumentSummary) bool { return strings.HasPrefix(d.Template, "concession@") }) {
		t.Fatalf("документы изделия: %+v %v", list, err)
	}

	// Печать с QR: версия и отпечаток; печатная форма содержит QR и рамку.
	pa, err := w.docs.Print(as("FOR-WC", "site_foreman"), trvID, app.PrintPaper{CommandHeader: hdr()})
	if err != nil || pa.Version != 2 || pa.QR != dom.QR(trvID, v2.DocDigest) {
		t.Fatalf("печать: %+v %v", pa, err)
	}
	pv, err := w.docs.PrintView(context.Background(), trvID, pa.Version)
	if err != nil || !strings.Contains(pv.HTML, "<svg") || !strings.Contains(pv.HTML, "Получено из системы") || pv.DocDigest != v2.DocDigest {
		t.Fatalf("печатная форма: %v", err)
	}

	w.checkSchemas()

	// Отпечаток сервера = отпечаток агента: записанный document.version.drafted
	// совпадает с пересчётом из content и шаблона (как у агента токена, AD-12).
	for _, e := range w.j.Entries() {
		if e.EventType != string(catalog.DocumentVersionDrafted) {
			continue
		}
		dd, err := w.codec.Decode(context.Background(), e)
		if err != nil {
			t.Fatal(err)
		}
		var data struct {
			DocumentID string `json:"document_id"`
			Version    int    `json:"version"`
			DocDigest  string `json:"doc_digest"`
		}
		_ = json.Unmarshal(dd.Record.Data, &data)
		view := w.doc(data.DocumentID, data.Version)
		tp, _ := w.env.Templates.ByRef(view.Template)
		raw, _ := json.Marshal(view.Content)
		agent, err := dom.Rebuild(tp, &dom.Version{Content: raw})
		if err != nil || agent.Digest != data.DocDigest || agent.RenderingHash != view.RenderingHash {
			t.Fatalf("%s@%d: агент %s, сервер %s (%v)", data.DocumentID, data.Version, agent.Digest, data.DocDigest, err)
		}
	}
}

// Документ вне изделия (AD-12, AD-42): лист утверждения версии процесса —
// версия сразу в той же пачке, что и запрос; маршрут кворума закрывает
// реакция document.route.closed после третьей подписи.
func TestProcessApprovalSheet(t *testing.T) {
	w := newWorld(t)
	acc, err := w.docs.RequestVersion(as("TEC-01", "technologist"), app.RequestVersion{CommandHeader: hdr(), SubjectRef: "process_version:flange-2",
		Action: "process.version.submit", Decision: "Утвердить версию flange-2", Comment: "Окно кромок 8 ч → 6 ч"})
	if err != nil || acc.Version != 1 || !strings.HasPrefix(acc.DocumentID, "DOC-") {
		t.Fatalf("запрос: %+v %v", acc, err)
	}
	if len(w.entries(catalog.DocumentVersionDrafted)) != 1 {
		t.Fatal("версия вне изделия не записана в пачке запроса")
	}
	d := w.doc(acc.DocumentID, 0)
	if d.Template != "process-version-approval@1" || len(d.Route) != 3 || d.DocDigest != acc.DocDigest {
		t.Fatalf("лист: %+v", d)
	}
	for i, s := range []struct{ who, role string }{{"TEC-01", "technologist"}, {"INS-01", "quality_inspector"}, {"PM-01", "production_manager"}} {
		if _, err := w.docs.RecordSignature(as(s.who, s.role), acc.DocumentID, app.RecordSignature{CommandHeader: hdr(), Version: 1, Stage: i + 1}); err != nil {
			t.Fatalf("%s: %v", s.who, err)
		}
	}
	if !slices.Contains(w.entries(catalog.DocumentRouteClosed), "document:"+acc.DocumentID) {
		t.Fatal("нет document.route.closed листа утверждения")
	}
	if d = w.doc(acc.DocumentID, 0); d.Status != dom.StatusRouteClosed {
		t.Fatalf("лист: %s", d.Status)
	}
	sh, ok, err := w.docs.ApprovalSheet(context.Background(), "process_version:flange-2")
	if err != nil || !ok || !sh.Closed || sh.RouteClosedEventID == "" || len(sh.Signatures) != 3 || sh.Signatures[2].Role != "production_manager" {
		t.Fatalf("лист для process: %+v %v %v", sh, ok, err)
	}
	// Отказ в согласовании и повтор команды (AD-7).
	acc2, _ := w.docs.RequestVersion(as("TEC-01", "technologist"), app.RequestVersion{CommandHeader: hdr(), SubjectRef: "process_version:flange-3", Action: "process.version.submit"})
	h := hdr()
	if _, err := w.docs.Decline(as("TEC-01", "technologist"), acc2.DocumentID, app.DeclineSignature{CommandHeader: h, Version: 1, Stage: 1, Comment: "нет читаемой разницы"}); err != nil {
		t.Fatal(err)
	}
	r, err := w.docs.Decline(as("TEC-01", "technologist"), acc2.DocumentID, app.DeclineSignature{CommandHeader: h, Version: 1, Stage: 1, Comment: "нет читаемой разницы"})
	if err != nil || !r.Replayed {
		t.Fatalf("повтор: %+v %v", r, err)
	}
	if d := w.doc(acc2.DocumentID, 0); d.Status != dom.StatusReturned {
		t.Fatalf("возвращён: %s", d.Status)
	}
	w.checkSchemas()
}

// checkSchemas — каждая запись семейства document в журнале проходит схему
// конверта и схему data своего типа (AD-20): реакции и решения модуля
// соответствуют контракту.
func (w *world) checkSchemas() {
	w.t.Helper()
	n := 0
	for _, e := range w.j.Entries() {
		if !strings.HasPrefix(e.EventType, "document.") {
			continue
		}
		env, err := w.j.Open(context.Background(), e)
		if err != nil {
			w.t.Fatal(err)
		}
		var dsse struct {
			Payload string `json:"payload"`
		}
		if err := json.Unmarshal(env.Raw, &dsse); err != nil {
			w.t.Fatal(err)
		}
		raw, err := base64.StdEncoding.DecodeString(dsse.Payload)
		if err != nil {
			w.t.Fatal(err)
		}
		dec, err := ingest.ValidateEnvelope(raw)
		if err != nil || dec.Outcome != domingest.OutcomeAccepted {
			w.t.Fatalf("%s %s: %+v %v\n%s", e.EventType, e.EventID, dec, err, raw)
		}
		n++
	}
	if n == 0 {
		w.t.Fatal("нет записей document.*")
	}
}

func isList(v any) bool { _, ok := v.([]any); return ok }

func strs(v any) []string {
	var out []string
	l, _ := v.([]any)
	for _, x := range l {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
