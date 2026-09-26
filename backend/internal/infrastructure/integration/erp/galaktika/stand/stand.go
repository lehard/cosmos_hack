package stand

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/infrastructure/integration/erp/galaktika"
	"ant/internal/infrastructure/integration/ingest/stands"
)

// Name — имя stand-а Галактики: страница /stand/galaktika/, служебный порт
// /stand/_control/galaktika/, имя в сценариях.
const Name = "galaktika"

// Facade — путь REST-фасада под /stand/galaktika (адрес адаптера —
// http://‹stands›/stand/galaktika/esb/v1).
const Facade = "/esb/v1"

// Узел и база Галактики stand-а (описание ответной стороны about).
const (
	Node     = "GAL-ERP-STAND"
	Database = "stand"
)

// Faults — сбои stand-а Галактики: недоступность, ошибка, дубль, задержка,
// несовместимый обработчик (corrupt).
var Faults = []app.FaultKind{app.FaultOffline, app.FaultError, app.FaultDuplicate, app.FaultDelay, app.FaultCorrupt}

// Options — параметры stand-а.
type Options struct {
	// Dir — корень каталога обмена (erp.galaktika.dir); пусто — только REST-фасад.
	Dir string
	// Interval — период обхода каталога обмена (0 — 1 с).
	Interval time.Duration
	// Now — часы stand-а (nil — системные).
	Now func() time.Time
	Log *slog.Logger
}

// Message — пакет Главного в журнале обмена stand-а.
type Message struct {
	MessageID   string    `json:"message_id"`
	Transport   string    `json:"transport"`
	Kind        string    `json:"kind"`
	BusinessKey string    `json:"business_key"`
	Subject     string    `json:"subject"`
	Status      string    `json:"status"` // ok | error | fault
	Code        string    `json:"code,omitempty"`
	Text        string    `json:"text,omitempty"`
	Document    string    `json:"document,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`
	LastAt      time.Time `json:"last_at"`
	Deliveries  int       `json:"deliveries"`
	Lost        int       `json:"lost,omitempty"`
	Faults      []string  `json:"faults,omitempty"`
}

// Document — документ Галактики по принятому пакету.
type Document struct {
	Number    string    `json:"number"`
	NRec      string    `json:"nrec"`
	Title     string    `json:"title"`
	Comment   string    `json:"comment,omitempty"`
	Subject   string    `json:"subject"`
	MessageID string    `json:"message_id"`
	Date      time.Time `json:"date"`
	Storno    bool      `json:"storno,omitempty"`
}

// Packet — пакет Галактики для Главного (задание, партия) и квитанция
// Главного на него (каталог обмена: in-ack/).
type Packet struct {
	MessageID string             `json:"message_id"`
	Title     string             `json:"title"`
	Exchange  galaktika.Exchange `json:"exchange"`
	CreatedAt time.Time          `json:"created_at"`
	// AckStatus — квитанция Главного: ok | error | "" (ещё нет).
	AckStatus string `json:"ack_status,omitempty"`
	AckText   string `json:"ack_text,omitempty"`
}

// Snapshot — состояние stand-а (страница, проверки).
type Snapshot struct {
	Messages  []Message  `json:"messages"`
	Documents []Document `json:"documents"`
	Packets   []Packet   `json:"packets"`
}

// Stand — stand Галактика ERP (stands.Stand).
type Stand struct {
	opt    Options
	faults *stands.FaultSwitch
	mux    *http.ServeMux

	mu       sync.Mutex
	messages []Message
	acks     map[string]galaktika.Ack // принятые пакеты: номер → квитанция
	docs     []Document
	packets  []Packet
	seen     map[string]bool // пакеты каталога, на которые сбой уже записан
}

var _ stands.Stand = (*Stand)(nil)

// New создаёт stand.
func New(opt Options) *Stand {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if opt.Log == nil {
		opt.Log = slog.New(slog.DiscardHandler)
	}
	if opt.Interval <= 0 {
		opt.Interval = time.Second
	}
	s := &Stand{opt: opt, faults: stands.NewFaultSwitch(Faults, opt.Now).Manual(), acks: map[string]galaktika.Ack{}, seen: map[string]bool{}}
	s.routes()
	return s
}

// Info — что эмулирует и граница эмуляции (NFR-TEST-2).
func (s *Stand) Info() app.StandInfo {
	return app.StandInfo{Name: Name,
		Emulates: "Галактика ERP 9.x с обработчиком обмена gal.qc.v1: каталог обмена (about.xml, out/ → ack/, in/ ← in-ack/) и REST-фасад «под Галактика ESB» (/stand/galaktika/esb/v1/), квитанции GalAck 202/200/422/400, документы Галактики по пакетам, задания и партии кнопками страницы /stand/galaktika/",
		Boundary: "нет проводок, складских остатков и себестоимости, VIP-интерфейса «Атлантиса» и прав пользователей; состояние в памяти роли stands (квитанции каталога — в томе); форматы — проектное предположение (docs/integrations/galaktika.md)"}
}

// Faults — сбои stand-а.
func (s *Stand) Faults() *stands.FaultSwitch { return s.faults }

// Handler — REST-фасад и страница stand-а (после /stand/galaktika).
func (s *Stand) Handler() http.Handler { return s.mux }

func (s *Stand) routes() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.page)
	mux.HandleFunc("GET /state", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, s.Snapshot()) })
	mux.HandleFunc("POST /ui/task", s.uiTask)
	mux.HandleFunc("POST /ui/lot", s.uiLot)
	mux.Handle("GET "+Facade+"/about", s.protocol(http.HandlerFunc(s.aboutJSON)))
	mux.Handle("GET "+Facade+"/exchange/outbox", s.protocol(http.HandlerFunc(s.outbox)))
	mux.Handle("POST "+Facade+"/quality/lot-results", s.protocol(http.HandlerFunc(s.post)))
	mux.Handle("POST "+Facade+"/production/postings", s.protocol(http.HandlerFunc(s.post)))
	s.mux = mux
}

// About — описание ответной стороны; с несовместимым обработчиком (corrupt)
// — только следующая версия контракта: адаптер переводит канал в degraded.
func (s *Stand) About() galaktika.About {
	a := galaktika.About{Node: Node, Database: Database, Contract: galaktika.ContractVersion,
		Platform: "Галактика ERP 9.1 (stand «Главного»), обработчик обмена ant-qc", Supported: []string{galaktika.ContractVersion}}
	if _, ok := s.faults.Get(app.FaultCorrupt); ok {
		a.Contract, a.Supported = "gal.qc.v2", []string{"gal.qc.v2"}
	}
	return a
}

// protocol — сбои REST-фасада (страница stand-а работает всегда): offline —
// разрыв соединения; delay — задержка; error без образца — ответ кодом param.
func (s *Stand) protocol(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.faults.Get(app.FaultOffline); ok {
			drop(w)
			return
		}
		if x, ok := s.faults.Get(app.FaultDelay); ok && x.Param > 0 {
			t := time.NewTimer(time.Duration(x.Param) * time.Millisecond)
			select {
			case <-r.Context().Done():
				t.Stop()
				return
			case <-t.C:
			}
		}
		if x, ok := s.faults.Get(app.FaultError); ok && x.Match == "" {
			code, ack := faultAck(x, "unknown")
			writeJSON(w, code, ack)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// drop — разрыв соединения без ответа (недоступность, потерянный ответ).
func drop(w http.ResponseWriter) {
	if hj, ok := w.(http.Hijacker); ok {
		if c, _, err := hj.Hijack(); err == nil {
			_ = c.Close()
			return
		}
	}
	http.Error(w, "stand Галактики недоступен", http.StatusServiceUnavailable)
}

// faultAck — ответ сбоя «ошибка»: 4xx — квитанция status=error (ошибка
// данных; текст — detail), иначе код param (503 по умолчанию) — транспорт.
func faultAck(x app.Fault, id string) (int, galaktika.Ack) {
	code := int(x.Param)
	if code < 400 || code > 599 {
		code = http.StatusServiceUnavailable
	}
	a := galaktika.Ack{MessageID: id, Status: "error", Code: "UNAVAILABLE", Text: "Сервер приложений Галактики временно недоступен"}
	if code < 500 {
		a.Code, a.Text = dataCode(x.Detail), "Ошибка данных Галактики"
	}
	if x.Detail != "" {
		a.Text = x.Detail
	}
	return code, a
}

// dataCode — код ошибки данных GalAck по тексту сбоя (карта кодов — README контракта).
func dataCode(detail string) string {
	d := strings.ToLower(detail)
	switch {
	case strings.Contains(d, "договор"):
		return "CONTRACT_NOT_FOUND"
	case strings.Contains(d, "парти"):
		return "LOT_NOT_FOUND"
	case strings.Contains(d, "склад"):
		return "WAREHOUSE_NOT_FOUND"
	}
	return "DATA_ERROR"
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Stand) aboutJSON(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.About())
}

// outbox — пакеты Галактики для Главного (REST-фасад).
func (s *Stand) outbox(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	in := galaktika.Inbox{Packets: []galaktika.Exchange{}}
	for _, p := range s.packets {
		in.Packets = append(in.Packets, p.Exchange)
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, in)
}

// post — пакет через REST-фасад: контракт, идемпотентность по номеру пакета,
// сбои, правила обработчика, документ и квитанция.
func (s *Stand) post(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		drop(w)
		return
	}
	var e galaktika.Exchange
	if err := json.Unmarshal(raw, &e); err != nil {
		writeJSON(w, http.StatusBadRequest, galaktika.Ack{MessageID: "unknown", Status: "error", Code: "SCHEMA_VIOLATION", Text: "пакет не разобран: " + err.Error()})
		return
	}
	id := or(e.MessageID, "unknown")
	if v := r.Header.Get("X-Contract-Version"); v != galaktika.ContractVersion {
		writeJSON(w, http.StatusBadRequest, galaktika.Ack{MessageID: id, Status: "error", Code: "CONTRACT_VERSION",
			Text: fmt.Sprintf("версия контракта %q не поддерживается обработчиком (поддерживается %s)", v, galaktika.ContractVersion)})
		return
	}
	if err := galaktika.Check(e); err != nil {
		writeJSON(w, http.StatusBadRequest, galaktika.Ack{MessageID: id, Status: "error", Code: "SCHEMA_VIOLATION", Text: trim("пакет не по gal.qc.v1: "+err.Error(), 1000)})
		return
	}
	lotResults := strings.HasSuffix(r.URL.Path, "/lot-results")
	if lotResults != (e.QualityLotResult != nil) {
		writeJSON(w, http.StatusBadRequest, galaktika.Ack{MessageID: id, Status: "error", Code: "SCHEMA_VIOLATION", Text: "сообщение " + e.Kind() + " не для ресурса " + r.URL.Path})
		return
	}
	if r.Header.Get("X-Message-Id") != e.MessageID {
		writeJSON(w, http.StatusBadRequest, galaktika.Ack{MessageID: id, Status: "error", Code: "SCHEMA_VIOLATION", Text: "X-Message-Id не совпал с номером пакета"})
		return
	}
	if x, ok := s.faults.Get(app.FaultError); ok && x.Match != "" && Match(x.Match, e) {
		code, ack := faultAck(x, e.MessageID)
		s.record(e, galaktika.TransportREST, fmt.Sprintf("%d %s", code, ack.Text))
		writeJSON(w, code, ack)
		return
	}
	ack, fresh := s.Apply(e, galaktika.TransportREST)
	switch {
	case ack.Status != "ok":
		writeJSON(w, http.StatusUnprocessableEntity, ack)
	case ack.Duplicate:
		writeJSON(w, http.StatusOK, ack)
	default:
		if _, dup := s.faults.Get(app.FaultDuplicate); dup && fresh {
			// Дубль: документ создан, ответ потерян — фасад повторит тем же
			// номером и получит ту же квитанцию (duplicate), второго документа нет.
			s.lost(e.MessageID)
			drop(w)
			return
		}
		writeJSON(w, http.StatusAccepted, ack)
	}
}

// Apply — пакет Главного в состоянии stand-а: повтор принятого номера — та
// же квитанция с duplicate; новый — правила обработчика, документ и
// квитанция. fresh — пакет принят впервые.
func (s *Stand) Apply(e galaktika.Exchange, transport string) (galaktika.Ack, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.opt.Now().UTC()
	idx := slices.IndexFunc(s.messages, func(m Message) bool { return m.MessageID == e.MessageID })
	if a, ok := s.acks[e.MessageID]; ok {
		if idx >= 0 {
			s.messages[idx].Deliveries++
			s.messages[idx].LastAt = now
		}
		a.Duplicate = true
		return a, false
	}
	m := Message{MessageID: e.MessageID, Transport: transport, Kind: kind(e), BusinessKey: source(e).BusinessKey, Subject: subject(e),
		ReceivedAt: now, LastAt: now, Deliveries: 1}
	if idx >= 0 {
		m.ReceivedAt, m.Deliveries, m.Faults, m.Lost = s.messages[idx].ReceivedAt, s.messages[idx].Deliveries+1, s.messages[idx].Faults, s.messages[idx].Lost
	}
	if code, text := rules(e); code != "" {
		m.Status, m.Code, m.Text = "error", code, text
		s.put(idx, m)
		return galaktika.Ack{MessageID: e.MessageID, Status: "error", Code: code, Text: text}, true
	}
	d := s.document(e, now)
	if c := source(e).CorrectsMessageID; c != "" {
		for i := range s.docs {
			if s.docs[i].MessageID == c && !s.docs[i].Storno {
				s.docs[i].Storno = true
			}
		}
	}
	s.docs = append(s.docs, d)
	a := galaktika.Ack{MessageID: e.MessageID, Status: "ok", NRecCreated: d.NRec, Document: d.Number, ReceivedAt: now.Format("2006-01-02T15:04:05.000Z")}
	s.acks[e.MessageID] = a
	m.Status, m.Document = "ok", d.Number
	s.put(idx, m)
	return a, true
}

func (s *Stand) put(idx int, m Message) {
	if idx >= 0 {
		s.messages[idx] = m
		return
	}
	s.messages = append(s.messages, m)
}

// record — пакет, на который stand ответил сбоем (виден в журнале обмена).
func (s *Stand) record(e galaktika.Exchange, transport, fault string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.opt.Now().UTC()
	idx := slices.IndexFunc(s.messages, func(m Message) bool { return m.MessageID == e.MessageID })
	m := Message{MessageID: e.MessageID, Transport: transport, Kind: kind(e), BusinessKey: source(e).BusinessKey, Subject: subject(e),
		Status: "fault", ReceivedAt: now}
	if idx >= 0 {
		m = s.messages[idx]
		if m.Status == "ok" {
			return
		}
	}
	m.Deliveries++
	m.LastAt = now
	m.Faults = append(m.Faults, now.Format("15:04:05")+" "+fault)
	s.put(idx, m)
}

// lost — отметка «ответ потерян» (сбой «дубль»).
func (s *Stand) lost(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.messages {
		if s.messages[i].MessageID == id {
			s.messages[i].Lost++
		}
	}
}

// rules — правила обработчика Галактики: возврат поставщику — с основанием
// претензии (ошибка данных, без автоповтора у отправителя).
func rules(e galaktika.Exchange) (code, text string) {
	if p := e.Posting; p != nil && p.Kind == "return_to_supplier" && p.ClaimBasis == "" && len(p.Nonconformities) == 0 {
		return "CLAIM_BASIS_REQUIRED", "Для возвратной накладной поставщику нужно основание претензии"
	}
	return "", ""
}

// document — документ Галактики по пакету (вид — по сообщению и виду действия).
func (s *Stand) document(e galaktika.Exchange, now time.Time) Document {
	d := Document{MessageID: e.MessageID, Subject: subject(e), Date: now}
	if q := e.QualityLotResult; q != nil {
		d.Title = "Акт контроля качества (Управление качеством продукции)"
		d.Comment = verdictTitle(q.Verdict)
		if q.Presentation > 0 {
			d.Comment += ", предъявление " + strconv.Itoa(q.Presentation)
		}
	} else if p := e.Posting; p != nil {
		d.Title = postingTitle(p.Kind)
		if p.Concession != "" {
			d.Comment = "по разрешению на отклонение " + p.Concession
		}
		if p.AfterRework != nil && *p.AfterRework {
			d.Comment = strings.TrimSpace(d.Comment + " после переделки")
		}
		if p.ClaimBasis != "" {
			d.Comment = p.ClaimBasis
		}
	}
	n := len(s.docs) + 1
	d.Number = fmt.Sprintf("ДК-%06d", n)
	d.NRec = strconv.FormatInt(4611686018427500000+int64(n), 10)
	return d
}

func postingTitle(kind string) string {
	switch kind {
	case "accept_to_work":
		return "Требование-накладная (передача в производство)"
	case "internal_move":
		return "Накладная на внутреннее перемещение"
	case "defect_rework":
		return "Акт о браке: исправимый, на доработку"
	case "defect_writeoff":
		return "Акт о браке: списание"
	case "defect_reprocess":
		return "Акт о браке: переработка"
	case "return_to_supplier":
		return "Возвратная накладная поставщику"
	case "return_from_defect":
		return "Акт о возврате из брака"
	case "release":
		return "Сдача готовой продукции на склад"
	}
	return "Документ обмена (" + kind + ")"
}

func verdictTitle(v string) string {
	switch v {
	case "accepted":
		return "годно"
	case "accepted_with_concession":
		return "годно по разрешению на отклонение"
	case "accepted_partially":
		return "годно частично"
	case "rejected":
		return "не годно"
	}
	return "мало данных"
}

func source(e galaktika.Exchange) galaktika.Source {
	switch {
	case e.Posting != nil:
		return e.Posting.Source
	case e.QualityLotResult != nil:
		return e.QualityLotResult.Source
	}
	return galaktika.Source{}
}

// kind — вид пакета для журнала и образцов сбоя: вид Posting или quality.
func kind(e galaktika.Exchange) string {
	if e.Posting != nil {
		return e.Posting.Kind
	}
	if e.QualityLotResult != nil {
		return "quality"
	}
	return e.Kind()
}

func subject(e galaktika.Exchange) string {
	var it *galaktika.Item
	var lot *galaktika.Lot
	switch {
	case e.Posting != nil:
		it, lot = e.Posting.Item, e.Posting.Lot
	case e.QualityLotResult != nil:
		it, lot = e.QualityLotResult.Item, e.QualityLotResult.Lot
	}
	if it != nil && it.ItemID != "" {
		return it.ItemID
	}
	if lot != nil {
		return lot.LotID
	}
	return ""
}

// Match — пакет подходит под образец сбоя: «вид:субъект» (вид — учётное
// действие нашего языка, вид Posting gal.qc.v1 или quality; субъект — часть
// ID изделия, партии или бизнес-ключа) или подстрока бизнес-ключа / номер пакета.
func Match(pattern string, e galaktika.Exchange) bool {
	key := source(e).BusinessKey
	head, subj, ok := strings.Cut(pattern, ":")
	if !ok {
		return strings.Contains(key, pattern) || e.MessageID == pattern
	}
	action := ""
	if parts := strings.Split(key, "/"); len(parts) >= 3 {
		action = parts[len(parts)-2]
	}
	names := []string{action, kind(e)}
	switch action {
	case "release":
		names = append(names, "release_good", "released", "output")
	case "inspection_result":
		names = append(names, "quality", "control_result", "inspection")
	case "accept_into_work":
		names = append(names, "accepted_to_work", "accept")
	case "warehouse_transfer":
		names = append(names, "transfer", "move")
	case "scrap_transfer_rework", "scrap_transfer_writeoff", "scrap_transfer_reprocess":
		names = append(names, "moved_to_defect", "defect", "scrap")
	case "return_to_supplier":
		names = append(names, "return")
	}
	if !slices.Contains(names, head) {
		return false
	}
	return subj == "" || strings.Contains(subject(e), subj) || strings.Contains(key, subj)
}

// Snapshot — копия состояния stand-а.
func (s *Stand) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Snapshot{Messages: slices.Clone(s.messages), Documents: slices.Clone(s.docs), Packets: slices.Clone(s.packets)}
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// Run — обход каталога обмена (exchange-dir): about.xml, квитанции на пакеты
// out/, квитанции Главного на пакеты in/. Без каталога — только REST-фасад.
func (s *Stand) Run(ctx context.Context) error {
	if s.opt.Dir == "" {
		<-ctx.Done()
		return nil
	}
	if err := s.prepare(); err != nil {
		// Каталог недоступен (том не смонтирован): stand работает фасадом,
		// роль stands не падает — адаптер увидит «нет about.xml».
		s.opt.Log.Error("stand Галактики: каталог обмена недоступен", "dir", s.opt.Dir, "err", err)
		<-ctx.Done()
		return nil
	}
	t := time.NewTicker(s.opt.Interval)
	defer t.Stop()
	for {
		s.Scan()
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}
