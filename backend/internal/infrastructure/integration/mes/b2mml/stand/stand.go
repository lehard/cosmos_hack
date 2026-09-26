package stand

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	app "ant/internal/application/ingest"
	"ant/internal/infrastructure/integration/ingest/stands"
	"ant/internal/infrastructure/integration/mes/b2mml"
	"ant/internal/infrastructure/integration/schemacheck"
)

// Name — имя stand-а MES: страница /stand/mes/, служебный порт
// /stand/_control/mes/, имя в сценариях.
const Name = "mes"

// Binding — путь HTTP-привязки под /stand/mes (адрес адаптера —
// http://‹stands›/stand/mes/b2mml).
const Binding = "/b2mml"

// LogicalID — логический ID MES stand-а (Sender/LogicalID).
const LogicalID = "mes-stand"

// Faults — сбои stand-а MES.
var Faults = []app.FaultKind{app.FaultOffline, app.FaultError, app.FaultDuplicate, app.FaultDelay, app.FaultCorrupt}

// Operations — коды операций, которые MES выдаёт (нормативный слой фланца,
// operationCode шагов BPMN): список кнопки «выдать операцию».
var Operations = []string{"010", "020", "030", "040", "050", "060", "070", "080"}

// Options — параметры stand-а.
type Options struct {
	// Now — часы stand-а (nil — системные).
	Now func() time.Time
}

// Material — экземпляр (sublot) или партия (lot) в MES и его состояние.
type Material struct {
	ID          string    `json:"id"`
	Lot         bool      `json:"lot,omitempty"`
	Disposition string    `json:"disposition"` // Restricted | UnRestricted
	Status      string    `json:"status,omitempty"`
	Location    string    `json:"location,omitempty"`
	Description string    `json:"description,omitempty"`
	BODID       string    `json:"bodid"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Blocked — экземпляр заблокирован: следующая операция не выдаётся.
func (m Material) Blocked() bool { return m.Disposition == "Restricted" }

// Message — сообщение Главного в журнале обмена stand-а.
type Message struct {
	BODID       string    `json:"bodid"`
	Message     string    `json:"message"`
	Subjects    []string  `json:"subjects"`
	Disposition string    `json:"disposition"`
	Outcome     string    `json:"outcome"` // accepted | rejected | fault
	Code        string    `json:"code,omitempty"`
	Text        string    `json:"text,omitempty"`
	ReceivedAt  time.Time `json:"received_at"`
	LastAt      time.Time `json:"last_at"`
	Deliveries  int       `json:"deliveries"`
	Lost        int       `json:"lost,omitempty"`
	Faults      []string  `json:"faults,omitempty"`
}

// Operation — операция, выданная MES (или отказ в ней).
type Operation struct {
	SubLot  string    `json:"sublot"`
	Code    string    `json:"code"`
	Run     string    `json:"run,omitempty"` // SegmentResponseID
	State   string    `json:"state"`         // started | finished | refused
	Reason  string    `json:"reason,omitempty"`
	At      time.Time `json:"at"`
	BODID   string    `json:"bodid,omitempty"`
	Refusal bool      `json:"refusal,omitempty"`
}

// Snapshot — состояние stand-а.
type Snapshot struct {
	Materials  []Material  `json:"materials"`
	Messages   []Message   `json:"messages"`
	Operations []Operation `json:"operations"`
	Outbox     int         `json:"outbox"`
}

// ErrBlocked — экземпляр заблокирован ОТК: MES не выдаёт следующую операцию.
var ErrBlocked = errors.New("экземпляр заблокирован ОТК (QC-HOLD): MES не выдаёт следующую операцию")

// Stand — stand MES (stands.Stand).
type Stand struct {
	opt    Options
	faults *stands.FaultSwitch
	mux    *http.ServeMux

	mu        sync.Mutex
	materials map[string]Material
	order     []string
	messages  []Message
	ops       []Operation
	outbox    []json.RawMessage
	seq       int
}

var _ stands.Stand = (*Stand)(nil)

// New создаёт stand.
func New(opt Options) *Stand {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	s := &Stand{opt: opt, faults: stands.NewFaultSwitch(Faults, opt.Now).Manual(), materials: map[string]Material{}}
	s.routes()
	return s
}

// Info — что эмулирует и граница эмуляции (NFR-TEST-2).
func (s *Stand) Info() app.StandInfo {
	return app.StandInfo{Name: Name,
		Emulates: "MES цеха по B2MML V7 (B2MML-JSON, mes.isa95.v1): блок и снятие экземпляра и партии (SyncMaterialSubLot / SyncMaterialLot → ConfirmBOD), задания и события операций для Главного (outbox), заблокированный экземпляр не получает следующей операции; /stand/mes/b2mml/, страница /stand/mes/",
		Boundary: "нет диспетчирования и расписания MES, учёта оборудования и персонала, брокера сообщений (только HTTP-привязка); состояние в памяти роли stands; формат — проектное предположение (docs/integrations/mes.md)"}
}

// Faults — сбои stand-а.
func (s *Stand) Faults() *stands.FaultSwitch { return s.faults }

// Handler — HTTP-привязка B2MML и страница stand-а (после /stand/mes).
func (s *Stand) Handler() http.Handler { return s.mux }

// Run — фоновой работы нет: stand отвечает и формирует сообщения по кнопкам.
func (s *Stand) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (s *Stand) routes() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.page)
	mux.HandleFunc("GET /state", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, s.Snapshot()) })
	mux.HandleFunc("POST /ui/operation", s.uiOperation)
	mux.HandleFunc("POST /ui/finish", s.uiFinish)
	mux.HandleFunc("POST /ui/schedule", s.uiSchedule)
	mux.Handle("GET "+Binding+"/about", s.protocol(http.HandlerFunc(s.about)))
	mux.Handle("GET "+Binding+"/outbox", s.protocol(http.HandlerFunc(s.outboxJSON)))
	mux.Handle("POST "+Binding+"/SyncMaterialSubLot", s.protocol(http.HandlerFunc(s.sync)))
	mux.Handle("POST "+Binding+"/SyncMaterialLot", s.protocol(http.HandlerFunc(s.sync)))
	s.mux = mux
}

// About — описание привязки; несовместимая шина (corrupt) — только mes.isa95.v2.
func (s *Stand) About() b2mml.About {
	a := b2mml.About{LogicalID: LogicalID, ReleaseID: b2mml.ReleaseID, Supported: []string{b2mml.ContractVersion}, Product: "MES цеха (stand «Главного»)"}
	if _, ok := s.faults.Get(app.FaultCorrupt); ok {
		a.Supported = []string{"mes.isa95.v2"}
	}
	return a
}

func (s *Stand) about(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, s.About()) }

func (s *Stand) outboxJSON(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	out := map[string]any{"messages": append([]json.RawMessage{}, s.outbox...)}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, out)
}

// protocol — сбои привязки (страница stand-а работает всегда).
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
		if x, ok := s.faults.Get(app.FaultError); ok && x.Match == "" && (r.Method == http.MethodGet || x.Param >= 500 || x.Param == 0) {
			http.Error(w, "MES временно недоступна", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func drop(w http.ResponseWriter) {
	if hj, ok := w.(http.Hijacker); ok {
		if c, _, err := hj.Hijack(); err == nil {
			_ = c.Close()
			return
		}
	}
	http.Error(w, "stand MES недоступен", http.StatusServiceUnavailable)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// confirm — ConfirmBOD на сообщение bodid: успех (dup — Duplicate) или ошибка.
func (s *Stand) confirm(bodid string, dup bool, errType, code, text string) map[string]any {
	bod := map[string]any{"OriginalApplicationArea": map[string]any{"BODID": bodid}}
	if errType == "" {
		ok := map[string]any{}
		if dup {
			ok["Duplicate"] = true
		}
		bod["BODSuccessMessage"] = ok
	} else {
		bod["BODFailureMessage"] = map[string]any{"ErrorMessage": []any{map[string]any{"ErrorCode": code, "ErrorType": errType, "ErrorDescription": trim(text, 1000)}}}
	}
	return map[string]any{"ConfirmBOD": map[string]any{
		"releaseID":       b2mml.ReleaseID,
		"ApplicationArea": map[string]any{"Sender": map[string]any{"LogicalID": LogicalID}, "CreationDateTime": s.now(), "BODID": "mes-ack-" + safeID(bodid)},
		"DataArea":        map[string]any{"Confirm": map[string]any{}, "BOD": []any{bod}},
	}}
}

func (s *Stand) now() string { return s.opt.Now().UTC().Format("2006-01-02T15:04:05.000Z") }

// sync — блок или снятие: схема, идемпотентность по BODID, сбои, состояние
// экземпляра, ConfirmBOD.
func (s *Stand) sync(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		drop(w)
		return
	}
	var env map[string]b2mml.SyncBody
	_ = json.Unmarshal(raw, &env)
	body := env[msg]
	bodid := or(body.ApplicationArea.BODID, "unknown")
	if err := schemacheck.Validate(b2mml.Schema(msg), raw); err != nil {
		f, d := schemacheck.Violation(err)
		writeJSON(w, http.StatusBadRequest, s.confirm(bodid, false, "Contract", "SCHEMA_VIOLATION", msg+" не по mes.isa95.v1: "+f+": "+d))
		return
	}
	if v := body.VersionID; v != "" && v != b2mml.ContractVersion {
		writeJSON(w, http.StatusBadRequest, s.confirm(bodid, false, "Contract", "CONTRACT_VERSION", "версия "+v+" не поддерживается (поддерживается "+b2mml.ContractVersion+")"))
		return
	}
	lots := body.DataArea.MaterialSubLot
	if msg == "SyncMaterialLot" {
		lots = body.DataArea.MaterialLot
	}
	if x, ok := s.faults.Get(app.FaultError); ok && (x.Match == "" || Match(x.Match, lots)) {
		code := int(x.Param)
		if code < 400 || code > 599 {
			code = http.StatusServiceUnavailable
		}
		text := or(x.Detail, "Экземпляр "+ids(lots)+" не найден в MES")
		s.record(bodid, msg, lots, fmt.Sprintf("%d %s", code, text))
		if code >= 500 {
			http.Error(w, "MES временно недоступна", code)
			return
		}
		errCode := "SUBLOT_UNKNOWN"
		if msg == "SyncMaterialLot" {
			errCode = "LOT_UNKNOWN"
		}
		writeJSON(w, code, s.confirm(bodid, false, "Data", errCode, text))
		return
	}
	dup, fresh := s.Apply(bodid, msg, lots)
	if _, lose := s.faults.Get(app.FaultDuplicate); lose && fresh {
		// Дубль: блок принят, ответ потерян — повтор тем же BODID получит Duplicate.
		s.lost(bodid)
		drop(w)
		return
	}
	writeJSON(w, http.StatusOK, s.confirm(bodid, dup, "", "", ""))
}

// Apply — блок или снятие в состоянии MES; повтор принятого BODID — dup
// (состояние не меняется). fresh — сообщение принято впервые.
func (s *Stand) Apply(bodid, msg string, lots []b2mml.Lot) (dup, fresh bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.opt.Now().UTC()
	idx := slices.IndexFunc(s.messages, func(m Message) bool { return m.BODID == bodid })
	if idx >= 0 && s.messages[idx].Outcome == "accepted" {
		s.messages[idx].Deliveries++
		s.messages[idx].LastAt = now
		return true, false
	}
	m := Message{BODID: bodid, Message: msg, ReceivedAt: now, LastAt: now, Deliveries: 1, Outcome: "accepted"}
	if idx >= 0 {
		m.ReceivedAt, m.Deliveries, m.Faults, m.Lost = s.messages[idx].ReceivedAt, s.messages[idx].Deliveries+1, s.messages[idx].Faults, s.messages[idx].Lost
	}
	for _, l := range lots {
		mt := Material{ID: l.ID, Lot: msg == "SyncMaterialLot", Disposition: l.Disposition, Status: l.Status, Location: l.StorageLocation,
			Description: l.Description, BODID: bodid, UpdatedAt: now}
		if _, ok := s.materials[l.ID]; !ok {
			s.order = append(s.order, l.ID)
		}
		s.materials[l.ID] = mt
		m.Subjects, m.Disposition = append(m.Subjects, l.ID), l.Disposition
	}
	s.putMessage(idx, m)
	return false, true
}

func (s *Stand) putMessage(idx int, m Message) {
	if idx >= 0 {
		s.messages[idx] = m
		return
	}
	s.messages = append(s.messages, m)
}

// record — сообщение, на которое stand ответил сбоем.
func (s *Stand) record(bodid, msg string, lots []b2mml.Lot, fault string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.opt.Now().UTC()
	idx := slices.IndexFunc(s.messages, func(m Message) bool { return m.BODID == bodid })
	m := Message{BODID: bodid, Message: msg, Outcome: "fault", ReceivedAt: now}
	for _, l := range lots {
		m.Subjects, m.Disposition = append(m.Subjects, l.ID), l.Disposition
	}
	if idx >= 0 {
		m = s.messages[idx]
		if m.Outcome == "accepted" {
			return
		}
	}
	m.Deliveries++
	m.LastAt = now
	m.Faults = append(m.Faults, now.Format("15:04:05")+" "+fault)
	s.putMessage(idx, m)
}

func (s *Stand) lost(bodid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.messages {
		if s.messages[i].BODID == bodid {
			s.messages[i].Lost++
		}
	}
}

// Match — сообщение подходит под образец сбоя: «hold:субъект» (блок),
// «release:субъект» (снятие) или подстрока ID экземпляра / партии.
func Match(pattern string, lots []b2mml.Lot) bool {
	head, subj, ok := strings.Cut(pattern, ":")
	for _, l := range lots {
		if !ok {
			if strings.Contains(l.ID, pattern) {
				return true
			}
			continue
		}
		kind := "release"
		if l.Disposition == "Restricted" {
			kind = "hold"
		}
		names := []string{kind}
		if kind == "hold" {
			names = append(names, "block", "mes.hold")
		} else {
			names = append(names, "unblock", "mes.release")
		}
		if slices.Contains(names, head) && (subj == "" || strings.Contains(l.ID, subj)) {
			return true
		}
	}
	return false
}

// NextOperation — MES выдаёт экземпляру sublot операцию code: заблокированный
// экземпляр (Restricted) — отказ ErrBlocked, событие не формируется; иначе
// NotifyOperationsEvent Start в outbox для Главного.
func (s *Stand) NextOperation(sublot, code string) (Operation, error) {
	sublot = strings.TrimSpace(sublot)
	if sublot == "" {
		return Operation{}, errors.New("не указан экземпляр")
	}
	if !slices.Contains(Operations, code) {
		return Operation{}, fmt.Errorf("операция %q не из маршрута (%s)", code, strings.Join(Operations, ", "))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.opt.Now().UTC()
	if m, ok := s.materials[sublot]; ok && m.Blocked() {
		op := Operation{SubLot: sublot, Code: code, State: "refused", Reason: ErrBlocked.Error(), At: now, Refusal: true}
		s.ops = append(s.ops, op)
		return op, ErrBlocked
	}
	s.seq++
	op := Operation{SubLot: sublot, Code: code, Run: fmt.Sprintf("SRSP-STAND-%s-%04d", now.Format("150405"), s.seq), State: "started", At: now}
	bodid, err := s.notify(op, "Start", now)
	if err != nil {
		return Operation{}, err
	}
	op.BODID = bodid
	s.ops = append(s.ops, op)
	return op, nil
}

// FinishOperation — MES завершила выполнение run (NotifyOperationsEvent End).
func (s *Stand) FinishOperation(run string) (Operation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.ops, func(o Operation) bool { return o.Run == run && o.State == "started" })
	if i < 0 {
		return Operation{}, fmt.Errorf("операция %q не выполняется", run)
	}
	now := s.opt.Now().UTC()
	op := s.ops[i]
	if _, err := s.notify(op, "End", now); err != nil {
		return Operation{}, err
	}
	s.ops[i].State = "finished"
	return s.ops[i], nil
}

// notify — NotifyOperationsEvent одного события в outbox (под s.mu);
// сообщение проверяется схемой подмножества до выдачи (FR-111).
func (s *Stand) notify(op Operation, category string, now time.Time) (string, error) {
	var n b2mml.NotifyOperationsEvent
	b := &n.NotifyOperationsEvent
	b.ReleaseID, b.VersionID = b2mml.ReleaseID, b2mml.ContractVersion
	b.ApplicationArea = b2mml.ApplicationArea{Sender: b2mml.Sender{LogicalID: LogicalID, ConfirmationCode: "OnError"}, CreationDateTime: now.Format("2006-01-02T15:04:05.000Z"),
		BODID: fmt.Sprintf("mes-stand-%s-%04d", now.Format("20060102T150405"), len(s.outbox)+1)}
	b.DataArea.OperationsEvent = []b2mml.OperationsEvent{{ID: fmt.Sprintf("OE-STAND-%04d", len(s.outbox)+1), Category: category,
		EffectiveTimestamp: now.Format("2006-01-02T15:04:05.000Z"), SegmentResponseID: op.Run, ProcessSegmentID: op.Code, MaterialSubLotID: op.SubLot}}
	raw, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	if err := schemacheck.Validate(b2mml.Schema("NotifyOperationsEvent"), raw); err != nil {
		f, d := schemacheck.Violation(err)
		return "", fmt.Errorf("событие MES не по схеме: %s: %s", f, d)
	}
	s.outbox = append(s.outbox, raw)
	return b.ApplicationArea.BODID, nil
}

// Schedule — MES выдала задание (ProcessOperationsSchedule) на quantity
// фланцев по маршруту Operations.
func (s *Stand) Schedule(quantity int) (string, error) {
	if quantity < 1 || quantity > 1000 {
		return "", errors.New("количество — от 1 до 1000")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.opt.Now().UTC()
	ts := now.Format("2006-01-02T15:04:05.000Z")
	n := len(s.outbox) + 1
	req := fmt.Sprintf("OR-STAND-%04d", n)
	var segs []any
	for i, code := range Operations {
		seg := map[string]any{"ID": fmt.Sprintf("SR-%d", i+1), "ProcessSegmentID": code, "EarliestStartTime": ts}
		if i == 0 {
			seg["MaterialRequirement"] = []any{map[string]any{"MaterialDefinitionID": "ФЛ-100.00.000", "MaterialUse": "Produced",
				"Quantity": map[string]any{"QuantityString": fmt.Sprint(quantity), "UnitOfMeasure": "шт"}}}
		}
		segs = append(segs, seg)
	}
	msg := map[string]any{"ProcessOperationsSchedule": map[string]any{
		"releaseID": b2mml.ReleaseID, "versionID": b2mml.ContractVersion,
		"ApplicationArea": map[string]any{"Sender": map[string]any{"LogicalID": LogicalID, "ConfirmationCode": "OnError"}, "CreationDateTime": ts,
			"BODID": fmt.Sprintf("mes-stand-%s-%04d", now.Format("20060102T150405"), n)},
		"DataArea": map[string]any{"Process": map[string]any{}, "OperationsSchedule": []any{map[string]any{
			"ID": fmt.Sprintf("OS-STAND-%04d", n), "OperationsRequest": []any{map[string]any{
				"ID": req, "Description": fmt.Sprintf("Фланец люка ФЛ-100.00.000 СБ, партия запуска %d шт.", quantity), "SegmentRequirement": segs}}}}},
	}}
	raw, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	if err := schemacheck.Validate(b2mml.Schema("ProcessOperationsSchedule"), raw); err != nil {
		f, d := schemacheck.Violation(err)
		return "", fmt.Errorf("задание MES не по схеме: %s: %s", f, d)
	}
	s.outbox = append(s.outbox, raw)
	return req, nil
}

// Snapshot — копия состояния stand-а.
func (s *Stand) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Snapshot{Messages: slices.Clone(s.messages), Operations: slices.Clone(s.ops), Outbox: len(s.outbox)}
	for _, id := range s.order {
		out.Materials = append(out.Materials, s.materials[id])
	}
	return out
}

func ids(lots []b2mml.Lot) string {
	var out []string
	for _, l := range lots {
		out = append(out, l.ID)
	}
	return strings.Join(out, ", ")
}

// safeID — BODID подтверждения по BODID сообщения (шаблон BODID схемы).
func safeID(id string) string {
	out := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '.' || r == '_' || r == ':' {
			return r
		}
		return '-'
	}, id)
	if len(out) > 50 {
		out = out[:50]
	}
	return out
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
