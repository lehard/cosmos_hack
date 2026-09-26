package stand

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/infrastructure/integration/erp/onec"
	"ant/internal/infrastructure/integration/ingest/stands"
)

// Name — имя stand-а 1С: страница /stand/1c/, служебный порт /stand/_control/1c/.
const Name = "1c"

// Base — имя информационной базы в адресе публикации (`/stand/1c/erp/…`).
const Base = "erp"

// Faults — сбои stand-а 1С (FR-91: недоступность, ошибка, дубль; задержка;
// несовместимые метаданные — corrupt).
var Faults = []app.FaultKind{app.FaultOffline, app.FaultError, app.FaultDuplicate, app.FaultDelay, app.FaultCorrupt}

// Options — параметры stand-а.
type Options struct {
	// Store — состояние (nil — в памяти).
	Store Store
	// Now — часы stand-а (nil — системные).
	Now func() time.Time
	Log *slog.Logger
}

// Stand — stand 1С:ERP (stands.Stand).
type Stand struct {
	mu       sync.Mutex
	opt      Options
	faults   *stands.FaultSwitch
	manifest onec.Manifest
	seed     map[string][]Row
	state    Snapshot
	loaded   bool
	mux      *http.ServeMux
}

var _ stands.Stand = (*Stand)(nil)

// New создаёт stand; состояние загружается при первом обращении.
func New(opt Options) (*Stand, error) {
	if opt.Store == nil {
		opt.Store = &Memory{}
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if opt.Log == nil {
		opt.Log = slog.New(slog.DiscardHandler)
	}
	m, err := onec.LoadManifest()
	if err != nil {
		return nil, err
	}
	s := &Stand{opt: opt, manifest: m, seed: seed(), faults: stands.NewFaultSwitch(Faults, opt.Now).Manual()}
	s.routes()
	return s, nil
}

// Info — что эмулирует и граница эмуляции (NFR-TEST-2).
func (s *Stand) Info() app.StandInfo {
	return app.StandInfo{Name: Name,
		Emulates: "1С:ERP Управление предприятием 2.5 + расширение «КонтрольКачества»: OData v3 на чтение (/stand/1c/erp/odata/standard.odata/), HTTP-сервис qc.v1 на запись (/stand/1c/erp/hs/qc/v1/), квитанции 202/200, ошибки 422/400/503, документы 1С по сообщениям, страница /stand/1c/",
		Boundary: "нет проведения по регистрам ERP, себестоимости, количественных остатков, прав пользователей 1С, плана обмена и EnterpriseData; имена реквизитов — проектное предположение (docs/integrations/1c.md, §9)"}
}

// Faults — сбои stand-а.
func (s *Stand) Faults() *stands.FaultSwitch { return s.faults }

// Handler — протокол 1С и страница stand-а (после /stand/1c).
func (s *Stand) Handler() http.Handler { return s.mux }

// Run — фоновой работы у stand-а 1С нет: он только отвечает.
func (s *Stand) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (s *Stand) routes() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.page)
	mux.HandleFunc("GET /state", s.stateJSON)
	mux.HandleFunc("POST /stages", s.createStage)
	odata := "/" + Base + strings.TrimSuffix(s.manifest.OData.Path, "/")
	qc := "/" + Base + strings.TrimSuffix(s.manifest.QC.Path, "/")
	mux.Handle(odata+"/", s.protocol(http.HandlerFunc(s.odata)))
	mux.Handle("POST "+qc+"/postings", s.protocol(http.HandlerFunc(s.post)))
	mux.Handle("POST "+qc+"/inspection-results", s.protocol(http.HandlerFunc(s.post)))
	mux.Handle("GET "+qc+"/messages/{id}", s.protocol(http.HandlerFunc(s.message)))
	mux.Handle("GET "+qc+"/about", s.protocol(http.HandlerFunc(s.about)))
	s.mux = mux
}

// load — состояние из хранилища (один раз).
func (s *Stand) load(ctx context.Context) error {
	if s.loaded {
		return nil
	}
	st, err := s.opt.Store.Load(ctx)
	if err != nil {
		return err
	}
	s.state, s.loaded = st, true
	return nil
}

// protocol — сбои протокола 1С (служебная страница stand-а работает всегда):
// offline — разрыв соединения; delay — задержка; error без образца — ответ
// кодом param (503 по умолчанию) на любой запрос. Сбой с образцом (match),
// дубль и несовместимые метаданные применяют обработчики.
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
			s.faultError(w, x, "")
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
	http.Error(w, "stand 1С недоступен", http.StatusServiceUnavailable)
}

// faultError — ответ сбоя «ошибка»: 422 — ошибка данных (текст — detail), иначе
// код param (503 по умолчанию) с телом ошибки сервиса qc.
func (s *Stand) faultError(w http.ResponseWriter, x app.Fault, messageID string) {
	code := int(x.Param)
	if code < 400 || code > 599 {
		code = http.StatusServiceUnavailable
	}
	e := onec.Error{MessageID: messageID, Status: "unavailable", Code: "UNAVAILABLE", Message: "Сервис 1С временно недоступен"}
	if x.Detail != "" {
		e.Message = x.Detail
	}
	if code == http.StatusUnprocessableEntity || (code >= 400 && code < 500) {
		e.Status, e.Code = "rejected", "DATA_ERROR"
		if strings.Contains(strings.ToLower(x.Detail), "договор") {
			e.Code = "CONTRACT_NOT_FOUND"
		}
		if x.Detail == "" {
			e.Message = "Ошибка данных 1С"
		}
	}
	writeJSON(w, code, e)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// odataError — ошибка OData в «родном» формате 1С.
func odataError(w http.ResponseWriter, code int, oc, msg string) {
	writeJSON(w, code, map[string]any{"odata.error": map[string]any{"code": oc, "message": map[string]any{"lang": "ru", "value": msg}}})
}

// odata — стандартный интерфейс OData v3: $metadata и чтение сущностей.
func (s *Stand) odata(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/"+Base+s.manifest.OData.Path)
	if rest == "$metadata" {
		rename := map[string]string{}
		if _, ok := s.faults.Get(app.FaultCorrupt); ok {
			// Несовместимое изменение конфигурации 1С (О8): реквизит переименован.
			rename["Статус"], rename["Количество"] = "СтатусЭтапа", "КоличествоПлан"
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write(onec.BuildMetadata(s.manifest, rename))
		return
	}
	entity, key := rest, ""
	if i := strings.Index(rest, "(guid'"); i > 0 && strings.HasSuffix(rest, "')") {
		entity, key = rest[:i], rest[i+6:len(rest)-2]
	}
	if _, ok := s.manifest.Entity(entity); !ok {
		odataError(w, http.StatusNotFound, "8", fmt.Sprintf("Тип сущности %q не найден", entity))
		return
	}
	rows, err := s.rows(r.Context(), entity)
	if err != nil {
		odataError(w, http.StatusInternalServerError, "1", err.Error())
		return
	}
	if key != "" {
		for _, x := range rows {
			if x["Ref_Key"] == key {
				writeJSON(w, http.StatusOK, s.envelope(entity, []Row{x}))
				return
			}
		}
		odataError(w, http.StatusNotFound, "9", fmt.Sprintf("Экземпляр сущности %q не найден по переданному ключу.", strings.TrimPrefix(strings.TrimPrefix(entity, "Document_"), "Catalog_")))
		return
	}
	rows = filter(rows, r.URL.Query().Get("$filter"))
	if _, ok := s.faults.Get(app.FaultDuplicate); ok {
		// Дубль: каждая строка приходит дважды (повторная доставка).
		var twice []Row
		for _, x := range rows {
			twice = append(twice, x, x)
		}
		rows = twice
	}
	writeJSON(w, http.StatusOK, s.envelope(entity, rows))
}

func (s *Stand) envelope(entity string, rows []Row) map[string]any {
	if rows == nil {
		rows = []Row{}
	}
	return map[string]any{"odata.metadata": "$metadata#" + entity, "value": rows}
}

// rows — строки сущности: затравка + созданные этапы.
func (s *Stand) rows(ctx context.Context, entity string) ([]Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	out := slices.Clone(s.seed[entity])
	if entity == "Document_ЭтапПроизводства2_2" {
		out = append(out, s.state.Stages...)
	}
	return out, nil
}

// filter — подмножество $filter: условия `Реквизит eq значение`, соединённые `and`.
func filter(rows []Row, f string) []Row {
	f = strings.TrimSpace(f)
	if f == "" {
		return rows
	}
	type cond struct{ k, v string }
	var cs []cond
	for _, part := range strings.Split(f, " and ") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), " eq ")
		if !ok {
			continue
		}
		cs = append(cs, cond{strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), "'")})
	}
	var out []Row
	for _, r := range rows {
		ok := true
		for _, c := range cs {
			ok = ok && fmt.Sprint(r[c.k]) == c.v
		}
		if ok {
			out = append(out, r)
		}
	}
	return out
}

// about — версия контракта расширения.
func (s *Stand) about(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, onec.About{Service: "qc", Contract: onec.ContractVersion, Supported: []string{onec.ContractVersion},
		Configuration: "1С:ERP Управление предприятием 2.5 + расширение КонтрольКачества 1.0 (stand ant)"})
}

// message — статус сообщения по номеру (повторно выдаёт квитанцию).
func (s *Stand) message(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, onec.Error{Status: "unavailable", Code: "INTERNAL", Message: err.Error()})
		return
	}
	id := r.PathValue("id")
	for _, m := range s.state.Messages {
		if m.MessageID == id {
			if m.Receipt != nil {
				writeJSON(w, http.StatusOK, m.Receipt)
			} else {
				writeJSON(w, m.HTTPStatus, m.Error)
			}
			return
		}
	}
	writeJSON(w, http.StatusNotFound, onec.Error{MessageID: id, Status: "rejected", Code: "MESSAGE_NOT_FOUND", Message: "Сообщение с таким номером не поступало"})
}

// incoming — общая часть учётного действия и результата контроля.
type incoming struct {
	MessageID         string          `json:"message_id"`
	Contract          string          `json:"contract"`
	Kind              string          `json:"kind"`
	DefectKind        string          `json:"defect_kind"`
	OccurredAt        string          `json:"occurred_at"`
	Source            onec.Source     `json:"source"`
	Item              *onec.Item      `json:"item"`
	Lot               *onec.Lot       `json:"lot"`
	WarehouseFrom     *onec.Warehouse `json:"warehouse_from"`
	WarehouseTo       *onec.Warehouse `json:"warehouse_to"`
	ConcessionNumber  string          `json:"concession_number"`
	AfterRework       *bool           `json:"after_rework"`
	ClaimBasis        string          `json:"claim_basis"`
	ClosingPoint      string          `json:"closing_point"`
	PresentationNo    int             `json:"presentation_no"`
	Result            string          `json:"result"`
	Nonconformities   []string        `json:"nonconformities"`
	CorrectsMessageID string          `json:"corrects_message_id"`
}

func (m incoming) subject() string {
	if m.Item != nil {
		return m.Item.ItemID
	}
	if m.Lot != nil {
		return m.Lot.LotID
	}
	return ""
}

// action — учётное действие нашего языка (второй сегмент бизнес-ключа).
func (m incoming) action() string {
	parts := strings.Split(m.Source.BusinessKey, "/")
	if len(parts) >= 3 {
		return parts[len(parts)-2]
	}
	return m.Kind
}

// post — запись через HTTP-сервис qc: контракт, идемпотентность, сбои,
// правила расширения, документ 1С и квитанция.
func (s *Stand) post(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		drop(w)
		return
	}
	if v := r.Header.Get("X-Contract-Version"); v != onec.ContractVersion {
		writeJSON(w, http.StatusBadRequest, onec.Error{Status: "rejected", Code: "CONTRACT_VERSION",
			Message: fmt.Sprintf("Версия контракта %q не поддерживается расширением (поддерживается %s)", v, onec.ContractVersion)})
		return
	}
	schema := "integrations/erp/1c/qc.v1/posting.schema.json"
	if resource == "inspection-results" {
		schema = "integrations/erp/1c/qc.v1/inspection-result.schema.json"
	}
	if err := onec.Validate(schema, raw); err != nil {
		writeJSON(w, http.StatusBadRequest, onec.Error{Status: "rejected", Code: "SCHEMA_VIOLATION", Message: "Сообщение не по контракту qc.v1: " + err.Error()})
		return
	}
	var in incoming
	_ = json.Unmarshal(raw, &in)
	if id := r.Header.Get("X-Message-Id"); id == "" || id != in.MessageID {
		writeJSON(w, http.StatusBadRequest, onec.Error{MessageID: in.MessageID, Status: "rejected", Code: "SCHEMA_VIOLATION",
			Message: "X-Message-Id не совпадает с message_id сообщения"})
		return
	}
	if resource == "inspection-results" {
		in.Kind = "inspection_result"
	}
	if x, ok := s.faults.Get(app.FaultError); ok && x.Match != "" && Match(x.Match, in) {
		s.record(r.Context(), in, resource, raw, nil, &x)
		s.faultError(w, x, in.MessageID)
		return
	}
	code, body, fresh := s.apply(r.Context(), in, resource, raw)
	if fresh && code == http.StatusAccepted {
		if _, ok := s.faults.Get(app.FaultDuplicate); ok {
			// Дубль: документ проведён, ответ потерян — отправитель повторит с тем
			// же номером и получит ту же квитанцию (200), второго документа нет.
			s.lost(r.Context(), in.MessageID)
			drop(w)
			return
		}
	}
	writeJSON(w, code, body)
}

// apply — сообщение в состоянии stand-а: повтор — та же квитанция; новое —
// правила расширения, документ и квитанция.
func (s *Stand) apply(ctx context.Context, in incoming, resource string, raw []byte) (int, any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(ctx); err != nil {
		return http.StatusServiceUnavailable, onec.Error{MessageID: in.MessageID, Status: "unavailable", Code: "UNAVAILABLE", Message: err.Error()}, false
	}
	now := s.opt.Now().UTC()
	idx := slices.IndexFunc(s.state.Messages, func(m Message) bool { return m.MessageID == in.MessageID })
	if idx >= 0 && s.state.Messages[idx].Receipt != nil {
		m := s.state.Messages[idx]
		m.Deliveries++
		m.LastAt = now
		s.state.Messages[idx] = m
		s.persistMessage(ctx, m)
		return http.StatusOK, m.Receipt, false
	}
	m := Message{MessageID: in.MessageID, Resource: resource, Kind: in.Kind, BusinessKey: in.Source.BusinessKey, Version: in.Source.MessageVersion,
		Subject: in.subject(), Body: raw, ReceivedAt: now, LastAt: now, Deliveries: 1}
	if idx >= 0 {
		m.ReceivedAt, m.Deliveries = s.state.Messages[idx].ReceivedAt, s.state.Messages[idx].Deliveries+1
	}
	if e := s.rules(in); e != nil {
		e.MessageID = in.MessageID
		m.Status, m.Error, m.HTTPStatus = "rejected", e, http.StatusUnprocessableEntity
		s.putMessage(ctx, idx, m)
		return http.StatusUnprocessableEntity, e, true
	}
	doc := s.document(in, now)
	if in.CorrectsMessageID != "" {
		for i := range s.state.Documents {
			if d := s.state.Documents[i]; d.MessageID == in.CorrectsMessageID && !d.Storno {
				d.Storno, d.StornoBy = true, doc.RefKey
				s.state.Documents[i] = d
				s.persistDocument(ctx, d)
			}
		}
	}
	s.state.Documents = append(s.state.Documents, doc)
	s.persistDocument(ctx, doc)
	rc := &onec.Receipt{MessageID: in.MessageID, Status: "accepted", Receipt: fmt.Sprintf("R-%d-%06d", now.Year(), s.receipts()+1),
		DocumentRefKey: doc.RefKey, DocumentType: doc.Type, DocumentNumber: doc.Number, ReceivedAt: now.Format("2006-01-02T15:04:05.000Z")}
	m.Status, m.Receipt, m.HTTPStatus, m.Error = "accepted", rc, http.StatusAccepted, nil
	s.putMessage(ctx, idx, m)
	return http.StatusAccepted, rc, true
}

func (s *Stand) receipts() int {
	n := 0
	for _, m := range s.state.Messages {
		if m.Receipt != nil {
			n++
		}
	}
	return n
}

func (s *Stand) putMessage(ctx context.Context, idx int, m Message) {
	if idx >= 0 {
		s.state.Messages[idx] = m
	} else {
		s.state.Messages = append(s.state.Messages, m)
	}
	s.persistMessage(ctx, m)
}

func (s *Stand) persistMessage(ctx context.Context, m Message) {
	if err := s.opt.Store.SaveMessage(ctx, m); err != nil {
		s.opt.Log.Error("stand 1С: сообщение не сохранено", "message_id", m.MessageID, "err", err)
	}
}

func (s *Stand) persistDocument(ctx context.Context, d Document) {
	if err := s.opt.Store.SaveDocument(ctx, d); err != nil {
		s.opt.Log.Error("stand 1С: документ не сохранён", "ref_key", d.RefKey, "err", err)
	}
}

// record — сообщение, на которое stand ответил сбоем (видно в журнале обмена).
func (s *Stand) record(ctx context.Context, in incoming, resource string, raw []byte, _ *onec.Receipt, f *app.Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.load(ctx) != nil {
		return
	}
	now := s.opt.Now().UTC()
	idx := slices.IndexFunc(s.state.Messages, func(m Message) bool { return m.MessageID == in.MessageID })
	m := Message{MessageID: in.MessageID, Resource: resource, Kind: in.Kind, BusinessKey: in.Source.BusinessKey, Version: in.Source.MessageVersion,
		Subject: in.subject(), Body: raw, Status: "rejected", ReceivedAt: now, LastAt: now, Deliveries: 1}
	if idx >= 0 {
		m = s.state.Messages[idx]
		m.Deliveries++
		m.LastAt = now
		if m.Receipt != nil {
			return
		}
	}
	code := int(f.Param)
	if code == 0 {
		code = http.StatusServiceUnavailable
	}
	m.HTTPStatus = code
	m.Faults = append(m.Faults, fmt.Sprintf("%s %d %s", now.Format("15:04:05"), code, f.Detail))
	s.putMessage(ctx, idx, m)
}

// lost — отметка «ответ потерян» (сбой «дубль»).
func (s *Stand) lost(ctx context.Context, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.state.Messages {
		if s.state.Messages[i].MessageID == id {
			s.state.Messages[i].Lost++
			s.persistMessage(ctx, s.state.Messages[i])
		}
	}
}

// rules — правила расширения «КонтрольКачества»: склады и номенклатура 1С
// должны существовать (422 — ошибка данных, без автоповтора у отправителя).
func (s *Stand) rules(in incoming) *onec.Error {
	known := map[string]bool{}
	for _, r := range s.seed["Catalog_Склады"] {
		known[fmt.Sprint(r["Code"])] = true
	}
	for f, wh := range map[string]*onec.Warehouse{"warehouse_from.code": in.WarehouseFrom, "warehouse_to.code": in.WarehouseTo} {
		if wh != nil && !known[wh.Code] {
			return &onec.Error{Status: "rejected", Code: "WAREHOUSE_NOT_FOUND", Field: f, Message: "Склад «" + wh.Code + "» не найден в справочнике «Склады»"}
		}
	}
	if in.Kind == "returned_to_supplier" && in.ClaimBasis == "" && len(in.Nonconformities) == 0 {
		return &onec.Error{Status: "rejected", Code: "CLAIM_BASIS_REQUIRED", Field: "claim_basis", Message: "Для возврата поставщику нужно основание претензии"}
	}
	return nil
}

// document — документ 1С по сообщению (вид — по учётному действию).
func (s *Stand) document(in incoming, now time.Time) Document {
	d := Document{Date: now, MessageID: in.MessageID, Posted: true, Subject: in.subject(), Lot: in.Lot != nil}
	if in.Lot != nil {
		d.Quantity = in.Lot.Quantity
	}
	if in.WarehouseFrom != nil {
		d.WarehouseFrom = in.WarehouseFrom.Code
	}
	if in.WarehouseTo != nil {
		d.WarehouseTo = in.WarehouseTo.Code
	}
	switch in.Kind {
	case "accepted_to_work":
		d.Type, d.Title = "Document_ПередачаМатериаловВПроизводство", "Передача материалов в производство"
	case "warehouse_transfer":
		d.Type, d.Title = "Document_ПеремещениеТоваров", "Перемещение товаров"
	case "moved_to_defect":
		switch in.DefectKind {
		case "scrap":
			d.Type, d.Title, d.Quality = "Document_ВнутреннееПотреблениеТоваров", "Внутреннее потребление товаров (списание брака)", "Не годен"
		case "reprocessing":
			d.Type, d.Title, d.Quality = "Document_ПорчаТоваров", "Порча товаров (переработка)", "Не годен"
		default:
			d.Type, d.Title, d.Quality = "Document_ПересортицаТоваров", "Пересортица товаров: качество «Не годен» (переделка)", "Не годен"
		}
	case "returned_to_supplier":
		d.Type, d.Title, d.Comment = "Document_ВозвратТоваровПоставщику", "Возврат товаров поставщику", in.ClaimBasis
	case "returned_from_defect":
		d.Type, d.Title, d.Quality = "Document_ПересортицаТоваров", "Пересортица товаров: качество «Новый» (возврат из брака)", "Новый"
	case "released":
		d.Type, d.Title, d.Quality = "Document_ПередачаПродукцииИзПроизводства", "Передача продукции из производства (выпуск)", "Новый"
		if in.ConcessionNumber != "" {
			d.Quality, d.Comment = "Ограниченно годен", "по разрешению на отклонение "+in.ConcessionNumber
		}
		if in.AfterRework != nil && *in.AfterRework {
			d.Comment = strings.TrimSpace(d.Comment + " после переделки")
		}
	default:
		d.Type, d.Title = "Document_РезультатКонтроляКачества", "Результат контроля качества ("+in.ClosingPoint+")"
		d.Comment = resultTitle(in.Result)
		if in.PresentationNo > 0 {
			d.Comment += ", предъявление " + strconv.Itoa(in.PresentationNo)
		}
	}
	n := 1
	for _, x := range s.state.Documents {
		if x.Type == d.Type {
			n++
		}
	}
	d.Number = fmt.Sprintf("0000-%06d", n)
	d.RefKey = Ref("document", d.Type+"/"+d.Number+"/"+in.MessageID)
	return d
}

func resultTitle(r string) string {
	switch r {
	case "conforming":
		return "годно"
	case "conforming_with_concession":
		return "годно по разрешению на отклонение"
	case "partially_conforming":
		return "годно частично"
	case "nonconforming":
		return "не годно"
	}
	return "мало данных"
}

// Match — сообщение подходит под образец сбоя: «вид:субъект» (вид — учётное
// действие нашего языка, вид qc.v1 или имя сценария: release_good,
// return_to_supplier…; субъект — часть ID изделия, партии или бизнес-ключа)
// или подстрока бизнес-ключа.
func Match(pattern string, in incoming) bool {
	head, subject, ok := strings.Cut(pattern, ":")
	if !ok {
		return strings.Contains(in.Source.BusinessKey, pattern) || in.MessageID == pattern
	}
	names := aliases(in.action(), in.Kind)
	if !slices.Contains(names, head) {
		return false
	}
	if subject == "" {
		return true
	}
	return strings.Contains(in.subject(), subject) || strings.Contains(in.Source.BusinessKey, subject)
}

func aliases(action, kind string) []string {
	out := []string{action, kind}
	switch action {
	case "release":
		out = append(out, "release_good", "released", "output")
	case "return_to_supplier":
		out = append(out, "returned_to_supplier", "return")
	case "accept_into_work":
		out = append(out, "accepted_to_work", "accept", "issue_to_production")
	case "warehouse_transfer":
		out = append(out, "transfer", "movement", "move")
	case "scrap_transfer_rework", "scrap_transfer_writeoff", "scrap_transfer_reprocess":
		out = append(out, "moved_to_defect", "scrap_transfer", "defect", "scrap")
	case "return_from_defect":
		out = append(out, "returned_from_defect")
	case "inspection_result":
		out = append(out, "control_result", "inspection")
	}
	return out
}

// stateJSON — состояние stand-а (для проверок и страницы).
func (s *Stand) stateJSON(w http.ResponseWriter, r *http.Request) {
	st, err := s.Snapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// Snapshot — копия состояния stand-а.
func (s *Stand) Snapshot(ctx context.Context) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(ctx); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Messages: slices.Clone(s.state.Messages), Documents: slices.Clone(s.state.Documents), Stages: slices.Clone(s.state.Stages)}, nil
}

// createStage — страница stand-а: «1С выдала задание» — новый этап производства к выполнению.
func (s *Stand) createStage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	qty, err := strconv.Atoi(r.PostForm.Get("quantity"))
	if err != nil || qty < 1 || qty > 1000 {
		http.Error(w, "количество — от 1 до 1000", http.StatusBadRequest)
		return
	}
	due := r.PostForm.Get("due")
	if _, err := time.Parse("2006-01-02", due); err != nil {
		http.Error(w, "срок — дата ГГГГ-ММ-ДД", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	if err := s.load(r.Context()); err != nil {
		s.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := s.opt.Now().In(time.FixedZone("MSK", 3*3600))
	number := fmt.Sprintf("ЭП00-%06d", 918+len(s.state.Stages))
	row := Stage(number, now.Format("2006-01-02T15:04:05"), due+"T00:00:00", qty, Ref("order", "ЗП00-000917"))
	s.state.Stages = append(s.state.Stages, row)
	err = s.opt.Store.SaveStage(r.Context(), fmt.Sprint(row["Ref_Key"]), row)
	s.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "./?created="+url.QueryEscape(number)+"#stages", http.StatusSeeOther)
}
