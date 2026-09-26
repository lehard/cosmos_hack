package galaktika_test

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	app "ant/internal/application/erp"
	"ant/internal/application/erp/ledgertest"
	appingest "ant/internal/application/ingest"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/erp"
	domingest "ant/internal/domain/ingest"
	"ant/internal/infrastructure/integration/erp/galaktika"
	"ant/internal/infrastructure/integration/schemacheck"
)

// Контрактные тесты адаптера Галактики (AD-20, AD-35): эталонные пакеты
// contracts/integrations/erp/galaktika/examples — по схемам gal.qc.v1 (JSON)
// и именам gal.qc.v1.xsd (XML); XML- и JSON-формы одного пакета совпадают;
// тот же внутренний сигнал порта учёта, что уходит в 1С, даёт эталонные
// пакеты; общий контрактный тест порта учёта (ledgertest) — на обоих
// транспортах; классы ответа, сбои, входящие факты. Stand Галактики — эпик 43;
// ответная сторона здесь — эталонный фасад и каталог обмена в тесте.
//
// GALAKTIKA_UPDATE=1 — перезаписать эталоны из кода (после осознанного
// изменения контракта).

const contractRel = "contracts/integrations/erp/galaktika/"

func root() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "../../../../../..")
}

func example(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root(), contractRel, "examples", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

var update = os.Getenv("GALAKTIKA_UPDATE") == "1"

// golden — сравнить с эталоном или перезаписать его.
func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	p := filepath.Join(root(), contractRel, "examples", name)
	if update {
		if err := os.WriteFile(p, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(want), bytes.TrimSpace(got)) {
		t.Errorf("%s не совпал с эталоном:\n--- эталон\n%s\n--- адаптер\n%s", name, want, got)
	}
}

func jsonOf(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(b, '\n')
}

func sameJSON(a, b []byte) bool {
	var x, y any
	return json.Unmarshal(a, &x) == nil && json.Unmarshal(b, &y) == nil && reflect.DeepEqual(x, y)
}

// Входящие пакеты, квитанции и описание: XML-эталон разбирается, проходит
// схему, а его JSON-форма совпадает с JSON-эталоном — одна модель, два транспорта.
func TestExamplesXMLAndJSON(t *testing.T) {
	var packets []galaktika.Exchange
	for _, n := range []string{"item-catalog", "production-task", "lot-received"} {
		e, err := galaktika.ParseExchangeXML(example(t, n+".xml"))
		if err != nil {
			t.Fatalf("%s.xml: %v", n, err)
		}
		if err := galaktika.Check(e); err != nil {
			t.Fatalf("%s.xml не по схеме: %v", n, err)
		}
		golden(t, n+".json", jsonOf(t, e))
		packets = append(packets, e)
	}
	inbox := jsonOf(t, galaktika.Inbox{Packets: packets})
	golden(t, "inbox.json", inbox)
	if err := schemacheck.Validate("integrations/erp/galaktika/gal.qc.v1/inbox.schema.json", inbox); err != nil {
		t.Fatalf("inbox.json: %v", err)
	}
	for n, sch := range map[string]string{"ack.ok": "ack", "ack.error": "ack", "about": "about"} {
		var v any = &galaktika.Ack{}
		if sch == "about" {
			v = &galaktika.About{}
		}
		if err := xml.Unmarshal(example(t, n+".xml"), v); err != nil {
			t.Fatalf("%s.xml: %v", n, err)
		}
		j := jsonOf(t, v)
		if err := schemacheck.Validate("integrations/erp/galaktika/gal.qc.v1/"+sch+".schema.json", j); err != nil {
			t.Fatalf("%s не по схеме %s: %v", n, sch, err)
		}
		golden(t, n+".json", j)
	}
}

// Тот же внутренний сигнал, что уходит в 1С (ledgertest.Signals), даёт
// пакеты gal.qc.v1: каждый проходит схему до отправки, эталонные — совпадают.
func TestSignalToPackets(t *testing.T) {
	goldens := map[dom.Action]string{dom.InspectionResult: "quality-lot-result", dom.Release: "posting.release",
		dom.ReturnToSupplier: "posting.return-to-supplier", dom.ScrapRework: "posting.defect-rework", dom.WarehouseTransfer: "posting.internal-move"}
	for _, m := range ledgertest.Signals() {
		res, e := galaktika.Encode("ant", "GAL-ERP-ZAVOD2", "ENT01", m)
		if err := galaktika.Check(e); err != nil {
			t.Fatalf("%s: пакет не по схеме: %v", m.Action(), err)
		}
		wantRes := "/production/postings"
		if m.Action() == dom.InspectionResult {
			wantRes = "/quality/lot-results"
		}
		if res != wantRes {
			t.Errorf("%s: ресурс %s", m.Action(), res)
		}
		x, err := galaktika.MarshalXML(e)
		if err != nil {
			t.Fatal(err)
		}
		// XML туда и обратно — без потерь (модель одна для обоих транспортов).
		back, err := galaktika.ParseExchangeXML(x)
		if err != nil || !sameJSON(jsonOf(t, back), jsonOf(t, e)) {
			t.Fatalf("%s: XML туда-обратно: %v\n%s", m.Action(), err, x)
		}
		if n, ok := goldens[m.Action()]; ok {
			golden(t, n+".xml", x)
			if m.Action() == dom.InspectionResult {
				golden(t, n+".json", jsonOf(t, e))
			}
		}
	}
}

// Имена элементов и атрибутов всех XML-эталонов объявлены в gal.qc.v1.xsd
// (XSD — для обработчика на стороне Галактики; исполняемая проверка — JSON Schema).
func TestXSDDeclaresExamples(t *testing.T) {
	xsd, err := os.ReadFile(filepath.Join(root(), contractRel, "gal.qc.v1.xsd"))
	if err != nil {
		t.Fatal(err)
	}
	elems, attrs := map[string]bool{}, map[string]bool{}
	dec := xml.NewDecoder(bytes.NewReader(xsd))
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if s, ok := tok.(xml.StartElement); ok {
			for _, a := range s.Attr {
				if a.Name.Local == "name" && s.Name.Local == "element" {
					elems[a.Value] = true
				}
				if a.Name.Local == "name" && s.Name.Local == "attribute" {
					attrs[a.Value] = true
				}
			}
		}
	}
	files, _ := filepath.Glob(filepath.Join(root(), contractRel, "examples", "*.xml"))
	if len(files) < 10 {
		t.Fatalf("XML-эталонов %d", len(files))
	}
	for _, f := range files {
		b, _ := os.ReadFile(f)
		dec := xml.NewDecoder(bytes.NewReader(b))
		for {
			tok, err := dec.Token()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("%s: %v", filepath.Base(f), err)
			}
			if s, ok := tok.(xml.StartElement); ok {
				if s.Name.Space != galaktika.Namespace || !elems[s.Name.Local] {
					t.Errorf("%s: элемент %s {%s} не объявлен в XSD", filepath.Base(f), s.Name.Local, s.Name.Space)
				}
				for _, a := range s.Attr {
					if a.Name.Space == "" && a.Name.Local != "xmlns" && !attrs[a.Name.Local] {
						t.Errorf("%s: атрибут %s элемента %s не объявлен в XSD", filepath.Base(f), a.Name.Local, s.Name.Local)
					}
				}
			}
		}
	}
}

// facade — эталонная ответная сторона REST-фасада по контракту gal.qc.v1:
// сверка контракта, приём пакета с идемпотентностью по номеру, сбои по образцу.
type facade struct {
	t     *testing.T
	about []byte
	inbox []byte
	mu    sync.Mutex
	acks  map[string]galaktika.Ack
	docs  int
	// fault — ответ по подстроке бизнес-ключа: HTTP-статус и код GalAck.
	fault map[string][2]string
}

func newFacade(t *testing.T) *facade {
	return &facade{t: t, about: example(t, "about.json"), inbox: example(t, "inbox.json"), acks: map[string]galaktika.Ack{}, fault: map[string][2]string{}}
}

func (f *facade) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	send := func(code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/about":
		_, _ = w.Write(f.about)
	case r.Method == http.MethodGet && r.URL.Path == "/exchange/outbox":
		_, _ = w.Write(f.inbox)
	case r.Method == http.MethodPost && (r.URL.Path == "/quality/lot-results" || r.URL.Path == "/production/postings"):
		b, _ := io.ReadAll(r.Body)
		var e galaktika.Exchange
		if err := json.Unmarshal(b, &e); err != nil || galaktika.Check(e) != nil {
			send(400, galaktika.Ack{MessageID: "unknown", Status: "error", Code: "SCHEMA_VIOLATION", Text: "пакет не по gal.qc.v1"})
			return
		}
		if r.Header.Get("X-Contract-Version") != galaktika.ContractVersion {
			send(400, galaktika.Ack{MessageID: e.MessageID, Status: "error", Code: "CONTRACT_VERSION", Text: "неизвестная версия контракта"})
			return
		}
		if r.Header.Get("X-Message-Id") != e.MessageID {
			send(400, galaktika.Ack{MessageID: e.MessageID, Status: "error", Code: "SCHEMA_VIOLATION", Text: "X-Message-Id не совпал с номером пакета"})
			return
		}
		src := galaktika.Source{}
		if e.Posting != nil {
			src = e.Posting.Source
		} else {
			src = e.QualityLotResult.Source
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		for k, v := range f.fault {
			if strings.Contains(src.BusinessKey, k) {
				code, _ := strconv.Atoi(v[0])
				send(code, galaktika.Ack{MessageID: e.MessageID, Status: "error", Code: v[1], Text: "сбой по образцу " + k})
				return
			}
		}
		if a, ok := f.acks[e.MessageID]; ok {
			a.Duplicate = true
			send(200, a)
			return
		}
		f.docs++
		a := galaktika.Ack{MessageID: e.MessageID, Status: "ok", NRecCreated: strconv.Itoa(4611686018427400000 + f.docs),
			Document: "ДК-" + strconv.Itoa(1000+f.docs), ReceivedAt: "2026-09-25T10:42:18.120Z"}
		f.acks[e.MessageID] = a
		send(202, a)
	default:
		http.NotFound(w, r)
	}
}

func restClient(t *testing.T, f *facade) *galaktika.Client {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	c, err := galaktika.New(galaktika.Config{Transport: galaktika.TransportREST, BaseURL: srv.URL, Stand: true})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// envelope — входящий факт в конверте шлюза проходит схемы приёма.
func envelope(f app.Inbound) error {
	raw, err := app.Envelope(app.GatewaySource("galaktika"), app.GatewayKey("galaktika"), 1, f)
	if err != nil {
		return err
	}
	var d struct {
		Payload []byte `json:"payload"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	dec, err := appingest.ValidateEnvelope(d.Payload)
	if err != nil {
		return err
	}
	if dec.Outcome != domingest.OutcomeAccepted {
		return errors.New(string(dec.Code) + " " + dec.Field + " " + dec.Detail)
	}
	return nil
}

// Общий контрактный тест порта учёта — REST-фасад.
func TestLedgerContractREST(t *testing.T) {
	ledgertest.Run(t, restClient(t, newFacade(t)), ledgertest.Options{System: "galaktika", Envelope: envelope, WantInbound: 12})
}

// Общий контрактный тест порта учёта — каталог обмена: пакет кладётся в
// out/, квитанцию Галактика кладёт в ack/ позже; до неё — транспортная
// ошибка и повтор тем же номером (как роль outbox).
func TestLedgerContractDir(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"in", "out", "ack"} {
		_ = os.MkdirAll(filepath.Join(dir, d), 0o750)
	}
	_ = os.WriteFile(filepath.Join(dir, "about.xml"), example(t, "about.xml"), 0o644)
	for _, n := range []string{"item-catalog", "production-task", "lot-received"} {
		_ = os.WriteFile(filepath.Join(dir, "in", n+".xml"), example(t, n+".xml"), 0o644)
	}
	c, err := galaktika.New(galaktika.Config{Transport: galaktika.TransportDir, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	// Обработчик Галактики: квитанция на каждый пакет out/.
	n := 0
	settle := func(t *testing.T) {
		files, _ := filepath.Glob(filepath.Join(dir, "out", "*.xml"))
		for _, f := range files {
			ack := filepath.Join(dir, "ack", filepath.Base(f))
			if _, err := os.Stat(ack); err == nil {
				continue
			}
			b, _ := os.ReadFile(f)
			e, err := galaktika.ParseExchangeXML(b)
			if err != nil || galaktika.Check(e) != nil {
				t.Fatalf("пакет %s в каталоге не по контракту: %v", filepath.Base(f), err)
			}
			n++
			x, _ := galaktika.MarshalXML(galaktika.Ack{MessageID: e.MessageID, Status: "ok", NRecCreated: strconv.Itoa(4611686018427400000 + n), Document: "ДК-" + strconv.Itoa(n)})
			_ = os.WriteFile(ack, x, 0o644)
		}
	}
	ledgertest.Run(t, c, ledgertest.Options{System: "galaktika", Envelope: envelope, WantInbound: 12, Settle: settle})
	if acks, _ := filepath.Glob(filepath.Join(dir, "in-ack", "*.xml")); len(acks) != 3 {
		t.Errorf("квитанций на входящие %d", len(acks))
	}
	if outs, _ := filepath.Glob(filepath.Join(dir, "out", "*.xml")); len(outs) != len(ledgertest.Signals()) {
		t.Errorf("пакетов в out/ %d — повтор не должен дублировать файл", len(outs))
	}
}

// Сверка контракта: обработчик без нашей версии — канал degraded (ContractError).
func TestCheckContractMismatch(t *testing.T) {
	f := newFacade(t)
	f.about = []byte(`{"node":"GAL-ERP-ZAVOD2","database":"zavod2","contract":"gal.qc.v2","supported":["gal.qc.v2"]}`)
	_, err := restClient(t, f).Check(context.Background())
	if ce, ok := app.AsContract(err); !ok || !strings.Contains(ce.Detail, "gal.qc.v2") {
		t.Fatalf("несовместимая версия — ContractError: %v", err)
	}
	dir := t.TempDir()
	c, _ := galaktika.New(galaktika.Config{Transport: galaktika.TransportDir, Dir: dir})
	if _, err := c.Check(context.Background()); err == nil {
		t.Fatal("без about.xml канал не может быть ok")
	} else if _, ok := app.AsContract(err); !ok {
		t.Fatalf("без about.xml — ContractError: %v", err)
	}
}

// Классы ответа: 503 — транспорт (повтор тем же номером); 422 LOT_NOT_FOUND —
// ошибка данных с нашим кодом (карантин, без автоповтора); 400 CONTRACT_VERSION —
// несовместимость (канал degraded); пакет не по схеме — в Галактику не уходит.
func TestResponseClasses(t *testing.T) {
	f := newFacade(t)
	c := restClient(t, f)
	ctx := context.Background()
	sig := ledgertest.Signals()
	f.fault["release"] = [2]string{"503", "UNAVAILABLE"}
	if _, err := c.Post(ctx, sig[8]); err == nil {
		t.Fatal("503")
	} else if te, ok := app.AsTransport(err); !ok || te.HTTPStatus != 503 {
		t.Fatalf("503 — транспорт: %v", err)
	}
	f.fault = map[string][2]string{"LOT-R-117": {"422", "LOT_NOT_FOUND"}}
	r, err := c.Post(ctx, sig[7])
	if err != nil || r.Outcome != ev.ErpPostingRespondedV1OutcomeRejected || r.ErrorCode != "erp.id_mapping_missing" || r.Code != "LOT_NOT_FOUND" {
		t.Fatalf("422 — ошибка данных: %+v %v", r, err)
	}
	f.fault = map[string][2]string{"LOT-B-0915": {"400", "CONTRACT_VERSION"}}
	if _, err := c.Post(ctx, sig[0]); err == nil {
		t.Fatal("400 CONTRACT_VERSION")
	} else if ce, ok := app.AsContract(err); !ok || ce.Local {
		t.Fatalf("400 CONTRACT_VERSION — несовместимость ответной стороны: %v", err)
	}
	f.fault = map[string][2]string{}
	bad := sig[1]
	bad.MessageID = "номер с пробелом"
	if _, err := c.Post(ctx, bad); err == nil {
		t.Fatal("пакет не по схеме ушёл")
	} else if ce, ok := app.AsContract(err); !ok || !ce.Local {
		t.Fatalf("пакет не по схеме — ContractError{Local}: %v", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.docs != 0 {
		t.Fatalf("при сбоях документов в Галактике %d", f.docs)
	}
}

// Входящие пакеты → факты порта учёта и соответствия внешних ID (FR-95).
func TestInboundFacts(t *testing.T) {
	var got []app.Inbound
	for _, n := range []string{"item-catalog", "production-task", "lot-received"} {
		e, err := galaktika.ParseExchangeXML(example(t, n+".xml"))
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, galaktika.Inbound("zavod2", e)...)
	}
	byType := map[catalog.Type][]app.Inbound{}
	for _, f := range got {
		byType[f.Type] = append(byType[f.Type], f)
	}
	nom := byType[catalog.ErpNomenclatureSynced][0].Data.(ev.ErpNomenclatureSyncedV1)
	if len(nom.Entries) != 4 || nom.Entries[0].ExternalID != "galaktika:zavod2:KatMC:4611686018427388123" || string(*nom.Entries[0].ItemTypeID) != "FL-100.00.000" {
		t.Fatalf("каталог МЦ (помеченная на удаление — без записи): %+v", nom.Entries)
	}
	o := byType[catalog.ErpOrderReceived][0].Data.(ev.ErpOrderReceivedV1)
	if o.OrderID != "ORD-SZ-000812" || o.ItemTypeID != "FL-100.00.000" || o.Quantity != 6 || *o.ItemRevision != "Б" || string(*o.DueDate) != "2026-10-02" {
		t.Fatalf("задание: %+v", o)
	}
	l := byType[catalog.ErpLotReceived][0].Data.(ev.ErpLotReceivedV1)
	if l.LotID != "LOT-P-2026-0915" || l.SupplierID != "POST-003" || l.ItemTypeID != "FL-100.01.002" || l.Quantity != 12 || *l.HeatNo != "Пл-4471" {
		t.Fatalf("партия: %+v", l)
	}
	var maps []string
	for _, m := range byType[catalog.ReferenceExternalIdMapped] {
		d := m.Data.(ev.ReferenceExternalIDMappedV1)
		if d.System != "galaktika" {
			t.Fatalf("система соответствия: %+v", d)
		}
		maps = append(maps, string(d.ObjectKind)+":"+d.ExternalID+"→"+string(d.InternalID))
	}
	for _, want := range []string{"order:galaktika:zavod2:MnPlan:4611686018427399001→ORD-SZ-000812",
		"supplier:galaktika:zavod2:KatOrg:4611686018427377001→POST-003",
		"item_type:galaktika:zavod2:KatMC:4611686018427388130→FL-100.01.002"} {
		if !slices.Contains(maps, want) {
			t.Errorf("нет соответствия %s: %v", want, maps)
		}
	}
}

// Пакет не по контракту в каталоге обмена: фактов нет, квитанция с ошибкой.
func TestDirRejectsBadPacket(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "in"), 0o750)
	bad := bytes.Replace(example(t, "production-task.xml"), []byte(`quantity="6"`), []byte(`quantity="шесть"`), 1)
	_ = os.WriteFile(filepath.Join(dir, "in", "GAL-BAD.xml"), bad, 0o644)
	c, _ := galaktika.New(galaktika.Config{Transport: galaktika.TransportDir, Dir: dir})
	facts, err := c.Pull(context.Background())
	if err != nil || len(facts) != 0 {
		t.Fatalf("плохой пакет дал факты: %v %v", facts, err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "in-ack", "GAL-BAD.xml"))
	if err != nil || !bytes.Contains(b, []byte(`status="error"`)) || !bytes.Contains(b, []byte("SCHEMA_VIOLATION")) {
		t.Fatalf("квитанция на плохой пакет: %s %v", b, err)
	}
}
