package stand_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	app "ant/internal/application/erp"
	"ant/internal/application/erp/ledgertest"
	appingest "ant/internal/application/ingest"
	domingest "ant/internal/domain/ingest"
	"ant/internal/infrastructure/integration/erp/galaktika"
	"ant/internal/infrastructure/integration/erp/galaktika/stand"
	"ant/internal/infrastructure/integration/ingest/stands"
)

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

// withPackets — Галактика выдала задание и оприходовала партию (кнопки страницы).
func withPackets(t *testing.T, st *stand.Stand) {
	t.Helper()
	if _, err := st.Task(6, "2026-10-02"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Lot("П-2026-0915", 12); err != nil {
		t.Fatal(err)
	}
}

func server(t *testing.T, st *stand.Stand) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(stands.NewRegistry(st).Handler())
	t.Cleanup(srv.Close)
	return srv
}

func restClient(t *testing.T, srv *httptest.Server) *galaktika.Client {
	t.Helper()
	c, err := galaktika.New(galaktika.Config{Transport: galaktika.TransportREST, BaseURL: srv.URL + "/stand/galaktika/esb/v1", Stand: true})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Эпик 43: общий контрактный тест порта учёта на stand-е Галактики — REST-фасад.
func TestLedgerContractREST(t *testing.T) {
	st := stand.New(stand.Options{})
	withPackets(t, st)
	ledgertest.Run(t, restClient(t, server(t, st)), ledgertest.Options{System: "galaktika", Envelope: envelope})
	snap := st.Snapshot()
	if len(snap.Documents) != len(ledgertest.Signals()) {
		t.Fatalf("документов %d, сигналов %d — повтор не должен давать второй документ", len(snap.Documents), len(ledgertest.Signals()))
	}
}

// Эпик 43: общий контрактный тест порта учёта на stand-е Галактики — каталог
// обмена: квитанцию stand кладёт при обходе каталога (Settle — один обход).
func TestLedgerContractDir(t *testing.T) {
	dir := t.TempDir()
	st := stand.New(stand.Options{Dir: dir})
	st.Scan() // about.xml и подкаталоги
	withPackets(t, st)
	c, err := galaktika.New(galaktika.Config{Transport: galaktika.TransportDir, Dir: dir, Stand: true})
	if err != nil {
		t.Fatal(err)
	}
	ledgertest.Run(t, c, ledgertest.Options{System: "galaktika", Envelope: envelope, Settle: func(*testing.T) { st.Scan() }})
	if acks, _ := filepath.Glob(filepath.Join(dir, "ack", "*.xml")); len(acks) != len(ledgertest.Signals()) {
		t.Fatalf("квитанций Галактики %d", len(acks))
	}
	// Обе стороны подтвердили приём: квитанции Главного на пакеты Галактики.
	st.Scan()
	for _, p := range st.Snapshot().Packets {
		if p.AckStatus != "ok" {
			t.Fatalf("квитанция Главного на %s: %q %s", p.MessageID, p.AckStatus, p.AckText)
		}
	}
}

// Сбои stand-а: ошибка по образцу — ошибка данных; недоступность — транспорт;
// дубль — ответ потерян, повтор той же квитанцией; несовместимый обработчик — контракт.
func TestFaultsREST(t *testing.T) {
	st := stand.New(stand.Options{})
	srv := server(t, st)
	c := restClient(t, srv)
	ctx := context.Background()
	sig := ledgertest.Signals()
	release := sig[len(sig)-1]

	if err := st.Faults().Set(appingestFault("error", 422, "release:F-002", "Договор с поставщиком не найден")); err != nil {
		t.Fatal(err)
	}
	r, err := c.Post(ctx, release)
	if err != nil || r.Outcome != "rejected" || r.Code != "CONTRACT_NOT_FOUND" {
		t.Fatalf("ошибка по образцу: %+v %v", r, err)
	}
	// Другое сообщение образцу не подходит.
	if r, err := c.Post(ctx, sig[1]); err != nil || r.Outcome != "accepted" {
		t.Fatalf("не по образцу: %+v %v", r, err)
	}
	st.Faults().Clear()

	_ = st.Faults().Set(appingestFault("offline", 0, "", ""))
	if _, err := c.Post(ctx, release); err == nil {
		t.Fatal("недоступность: ожидалась транспортная ошибка")
	} else if _, ok := app.AsTransport(err); !ok {
		t.Fatalf("недоступность: %v", err)
	}
	st.Faults().Clear()

	_ = st.Faults().Set(appingestFault("duplicate", 0, "", ""))
	if _, err := c.Post(ctx, release); err == nil {
		t.Fatal("дубль: ответ должен потеряться")
	}
	r2, err := c.Post(ctx, release)
	if err != nil || r2.Outcome != "duplicate" || r2.Receipt == "" {
		t.Fatalf("дубль: повтор — та же квитанция: %+v %v", r2, err)
	}
	st.Faults().Clear()

	_ = st.Faults().Set(appingestFault("corrupt", 0, "", ""))
	if _, err := c.Check(ctx); err == nil {
		t.Fatal("несовместимый обработчик: ожидалась ошибка контракта")
	} else if _, ok := app.AsContract(err); !ok {
		t.Fatalf("несовместимый обработчик: %v", err)
	}
	st.Faults().Clear()

	// Страница stand-а и кнопки.
	resp, err := http.Get(srv.URL + "/stand/galaktika/")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("страница: %v %v", resp, err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(b), "Главного") || !strings.Contains(string(b), "Сдача готовой продукции") {
		t.Fatalf("страница без журнала обмена: %.300s", b)
	}
	resp, err = http.PostForm(srv.URL+"/stand/galaktika/ui/task", map[string][]string{"quantity": {"3"}, "due": {"2026-10-05"}})
	if err != nil || resp.StatusCode != http.StatusOK || len(st.Snapshot().Packets) != 1 {
		t.Fatalf("кнопка «выдать задание»: %v %v", resp, err)
	}
	_ = resp.Body.Close()
}

// Каталог обмена: недоступность — квитанций нет; ошибка 5xx — квитанций нет;
// ошибка данных — квитанция status=error.
func TestFaultsDir(t *testing.T) {
	dir := t.TempDir()
	st := stand.New(stand.Options{Dir: dir})
	st.Scan()
	c, _ := galaktika.New(galaktika.Config{Transport: galaktika.TransportDir, Dir: dir})
	ctx := context.Background()
	sig := ledgertest.Signals()
	_ = st.Faults().Set(appingestFault("offline", 0, "", ""))
	if _, err := c.Post(ctx, sig[0]); err == nil {
		t.Fatal("квитанции ещё нет — транспорт")
	}
	st.Scan()
	if _, err := os.Stat(filepath.Join(dir, "ack")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Post(ctx, sig[0]); err == nil {
		t.Fatal("недоступность: квитанции быть не должно")
	}
	st.Faults().Clear()
	_ = st.Faults().Set(appingestFault("error", 422, "", "Партия не найдена в каталоге партий"))
	st.Scan()
	r, err := c.Post(ctx, sig[0])
	if err != nil || r.Outcome != "rejected" || r.Code != "LOT_NOT_FOUND" {
		t.Fatalf("ошибка данных каталогом: %+v %v", r, err)
	}
}

// appingestFault — сбой служебного порта stand-ов.
func appingestFault(kind string, param int64, match, detail string) appingest.Fault {
	return appingest.Fault{Kind: appingest.FaultKind(kind), Param: param, Match: match, Detail: detail}
}
