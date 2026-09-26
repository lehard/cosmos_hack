package b2mml_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appingest "ant/internal/application/ingest"
	appjournal "ant/internal/application/journal"
	app "ant/internal/application/mes"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	domingest "ant/internal/domain/ingest"
	dom "ant/internal/domain/mes"
	"ant/internal/infrastructure/integration/mes/b2mml"
	"ant/internal/infrastructure/integration/schemacheck"
)

// Контрактные тесты канала MES (AD-20, AD-35): эталоны
// contracts/integrations/mes/examples — по схемам подмножества B2MML-JSON
// mes.isa95.v1; блок и снятие блока изделия и партии дают эталонные
// SyncMaterialSubLot / SyncMaterialLot; ConfirmBOD — классы ответа; задания и
// события операций → факты через шлюз и схемы приёма; сквозной путь
// «сдерживание → mes.hold.requested → MES → mes.hold.responded → mes.block.list».
// Stand MES — эпик 43; ответная сторона здесь — эталонная MES в тесте.
//
// MES_UPDATE=1 — перезаписать эталоны из кода.

func root() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "../../../../../..")
}

func example(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root(), "contracts/integrations/mes/examples", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func golden(t *testing.T, name string, v any) {
	t.Helper()
	got, _ := json.MarshalIndent(v, "", "  ")
	got = append(got, '\n')
	p := filepath.Join(root(), "contracts/integrations/mes/examples", name)
	if os.Getenv("MES_UPDATE") == "1" {
		if err := os.WriteFile(p, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var x, y any
	_ = json.Unmarshal(want, &x)
	_ = json.Unmarshal(got, &y)
	if !reflect.DeepEqual(x, y) {
		t.Errorf("%s не совпал с эталоном:\n%s", name, got)
	}
}

var t0 = time.Date(2026, 9, 25, 10, 42, 17, 305_000_000, time.UTC)

func holdMsg(hold bool, item, lot string, cycle string) app.HoldMessage {
	subject := item + lot
	key := subject + "/hold/" + cycle
	reason := "Блок по сдерживанию: ожидается решение контролёра"
	if !hold {
		key, reason = subject+"/release/"+cycle, "Снятие блока по решению человека"
	}
	return app.HoldMessage{MessageID: dom.MessageID(key), Key: key, Hold: hold, ItemID: item, LotID: lot, Reason: reason, OccurredAt: t0, Attempt: 1}
}

func confirm(bodid string, fail *b2mml.ErrorMessage, dup bool) b2mml.ConfirmBOD {
	var c b2mml.ConfirmBOD
	c.ConfirmBOD.ReleaseID = b2mml.ReleaseID
	c.ConfirmBOD.ApplicationArea = b2mml.ApplicationArea{Sender: b2mml.Sender{LogicalID: "mes-ceh-2"}, CreationDateTime: "2026-09-25T10:42:17.911Z", BODID: "mes-ack-" + bodid[:8]}
	var b b2mml.BOD
	b.OriginalApplicationArea.BODID = bodid
	if fail != nil {
		b.BODFailureMessage = &struct {
			ErrorMessage []b2mml.ErrorMessage `json:"ErrorMessage"`
		}{ErrorMessage: []b2mml.ErrorMessage{*fail}}
	} else {
		b.BODSuccessMessage = &struct {
			Duplicate bool `json:"Duplicate,omitempty"`
		}{Duplicate: dup}
	}
	c.ConfirmBOD.DataArea.BOD = []b2mml.BOD{b}
	return c
}

// Блок и снятие: эталонные сообщения, каждое проходит схему до отправки.
func TestHoldMessages(t *testing.T) {
	cases := map[string]app.HoldMessage{
		"SyncMaterialSubLot.hold.json":    holdMsg(true, "ENT01:F-007", "", "1"),
		"SyncMaterialSubLot.release.json": holdMsg(false, "ENT01:F-007", "", "1"),
		"SyncMaterialLot.hold.json":       holdMsg(true, "", "LOT-P-2026-0915", "1"),
	}
	for name, m := range cases {
		msg, body := b2mml.Encode("ant", m)
		raw, _ := json.Marshal(body)
		if err := schemacheck.Validate(b2mml.Schema(msg), raw); err != nil {
			t.Fatalf("%s не по схеме %s: %v", name, msg, err)
		}
		if !strings.HasPrefix(name, msg+".") {
			t.Errorf("%s: сообщение %s", name, msg)
		}
		golden(t, name, body)
	}
	hold := holdMsg(true, "ENT01:F-007", "", "1")
	golden(t, "ConfirmBOD.success.json", confirm(hold.MessageID, nil, false))
	golden(t, "ConfirmBOD.failure.json", confirm(hold.MessageID, &b2mml.ErrorMessage{ErrorCode: "SUBLOT_UNKNOWN", ErrorType: "Data",
		ErrorDescription: "Экземпляр ENT01:F-007 не найден"}, false))
}

// Все эталоны — по схемам своего вида.
func TestExamplesMatchSchemas(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join(root(), "contracts/integrations/mes/examples", "*.json"))
	if len(files) < 9 {
		t.Fatalf("эталонов %d", len(files))
	}
	for _, f := range files {
		b, _ := os.ReadFile(f)
		name := filepath.Base(f)
		sch := "integrations/mes/binding/" + strings.TrimSuffix(name, ".json") + ".schema.json"
		if first, _, _ := strings.Cut(name, "."); first != "about" && first != "outbox" {
			sch = b2mml.Schema(first)
		}
		if err := schemacheck.Validate(sch, b); err != nil {
			t.Errorf("%s не по схеме %s: %v", name, sch, err)
		}
	}
}

// Входящие: задания и события на нашем языке; лишние элементы стандарта не
// мешают; сообщение вне подмножества — отказ с причиной.
func TestDecodeInbound(t *testing.T) {
	jobs, _, err := b2mml.Decode(example(t, "ProcessOperationsSchedule.json"))
	if err != nil || len(jobs) != 2 || jobs[1].OperationCode != "030" || jobs[1].Station != "IS-1" || jobs[1].Material != "ФЛ-100.00.000 СБ" || jobs[1].Quantity != 6 {
		t.Fatalf("задания: %+v %v", jobs, err)
	}
	var m map[string]any
	_ = json.Unmarshal(example(t, "NotifyOperationsEvent.json"), &m)
	oe := m["NotifyOperationsEvent"].(map[string]any)["DataArea"].(map[string]any)["OperationsEvent"].([]any)[0].(map[string]any)
	oe["HierarchyScope"] = map[string]any{"EquipmentID": "WC-2", "EquipmentElementLevel": "WorkCenter"}
	b, _ := json.Marshal(m)
	_, evs, err := b2mml.Decode(b)
	if err != nil || len(evs) != 2 || evs[0].Category != dom.EventStart || evs[1].Category != dom.EventEnd || evs[0].RunRef != "SRSP-4471" {
		t.Fatalf("события (терпимый разбор): %+v %v", evs, err)
	}
	if _, _, err := b2mml.Decode([]byte(`{"SyncPersonnel":{"releaseID":"7.01"}}`)); err == nil || !strings.Contains(err.Error(), "вне подмножества") {
		t.Fatalf("сообщение вне подмножества: %v", err)
	}
	bad := bytes.Replace(example(t, "NotifyOperationsEvent.json"), []byte(`"Category": "Start"`), []byte(`"Category": "Begin"`), 1)
	if _, _, err := b2mml.Decode(bad); err == nil {
		t.Fatal("категория вне перечисления должна отвергаться схемой")
	}
}

// mes — эталонная MES: about, канал входящих, приём блоков с ConfirmBOD.
type mes struct {
	mu     sync.Mutex
	got    map[string]int
	fault  map[string]b2mml.ErrorMessage
	status int
}

func (s *mes) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	send := func(code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/about":
		b, _ := os.ReadFile(filepath.Join(root(), "contracts/integrations/mes/examples/about.json"))
		_, _ = w.Write(b)
	case r.Method == http.MethodGet && r.URL.Path == "/outbox":
		b, _ := os.ReadFile(filepath.Join(root(), "contracts/integrations/mes/examples/outbox.json"))
		_, _ = w.Write(b)
	case r.Method == http.MethodPost && (r.URL.Path == "/SyncMaterialSubLot" || r.URL.Path == "/SyncMaterialLot"):
		b, _ := io.ReadAll(r.Body)
		if schemacheck.Validate(b2mml.Schema(strings.TrimPrefix(r.URL.Path, "/")), b) != nil {
			http.Error(w, "не по схеме", 400)
			return
		}
		var any1 map[string]struct {
			ApplicationArea b2mml.ApplicationArea `json:"ApplicationArea"`
		}
		_ = json.Unmarshal(b, &any1)
		bodid := ""
		for _, v := range any1 {
			bodid = v.ApplicationArea.BODID
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.status != 0 {
			http.Error(w, "MES недоступна", s.status)
			return
		}
		for k, e := range s.fault {
			if bytes.Contains(b, []byte(k)) {
				e := e
				send(422, confirm(bodid, &e, false))
				return
			}
		}
		s.got[bodid]++
		send(200, confirm(bodid, nil, s.got[bodid] > 1))
	default:
		http.NotFound(w, r)
	}
}

func channel(t *testing.T) (*b2mml.Client, *mes) {
	t.Helper()
	m := &mes{got: map[string]int{}, fault: map[string]b2mml.ErrorMessage{}}
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	c, err := b2mml.New(b2mml.Config{BaseURL: srv.URL, Stand: true})
	if err != nil {
		t.Fatal(err)
	}
	return c, m
}

// Канал: сверка, блок — квитанция, повтор того же BODID — duplicate, ошибка
// данных — rejected с кодом MES, 503 — транспорт, неподдержанная версия —
// несовместимость.
func TestChannelContract(t *testing.T) {
	golden(t, "outbox.json", map[string]any{"messages": []json.RawMessage{example(t, "ProcessOperationsSchedule.json"), example(t, "NotifyOperationsEvent.json")}})
	c, m := channel(t)
	ctx := context.Background()
	if _, err := c.Check(ctx); err != nil {
		t.Fatal(err)
	}
	h := holdMsg(true, "ENT01:F-007", "", "1")
	if r, err := c.Post(ctx, h); err != nil || r.Outcome != "accepted" {
		t.Fatalf("блок: %+v %v", r, err)
	}
	if r, err := c.Post(ctx, h); err != nil || r.Outcome != "duplicate" {
		t.Fatalf("повтор: %+v %v", r, err)
	}
	m.fault["F-008"] = b2mml.ErrorMessage{ErrorCode: "SUBLOT_UNKNOWN", ErrorType: "Data", ErrorDescription: "нет экземпляра"}
	if r, err := c.Post(ctx, holdMsg(true, "ENT01:F-008", "", "1")); err != nil || r.Outcome != "rejected" || r.Code != "SUBLOT_UNKNOWN" {
		t.Fatalf("ошибка данных: %+v %v", r, err)
	}
	m.status = 503
	if _, err := c.Post(ctx, holdMsg(true, "ENT01:F-009", "", "1")); err == nil {
		t.Fatal("503")
	} else if _, ok := app.AsTransport(err); !ok {
		t.Fatalf("503 — транспорт: %v", err)
	}
	bad := holdMsg(true, "ENT01:F-010", "", "1")
	bad.MessageID = "BODID с пробелом"
	if _, err := c.Post(ctx, bad); err == nil {
		t.Fatal("не по схеме — не отправляется")
	} else if ce, ok := app.AsContract(err); !ok || !ce.Local {
		t.Fatalf("не по схеме — ContractError{Local}: %v", err)
	}
	if _, err := b2mml.Confirm("x", []byte(`{"ConfirmBOD":{}}`)); err == nil {
		t.Fatal("ConfirmBOD не по схеме")
	}
}

// intake — приём в памяти: конверт проверяется схемами приёма.
type intake struct{ events []map[string]any }

func (i *intake) Submit(_ context.Context, source string, envs [][]byte) error {
	for _, raw := range envs {
		var d struct {
			Payload string `json:"payload"`
		}
		_ = json.Unmarshal(raw, &d)
		p, _ := base64.StdEncoding.DecodeString(d.Payload)
		dec, err := appingest.ValidateEnvelope(p)
		if err != nil {
			return err
		}
		if dec.Outcome != domingest.OutcomeAccepted {
			return errors.New(string(dec.Code) + " " + dec.Field + " " + dec.Detail)
		}
		var e map[string]any
		_ = json.Unmarshal(p, &e)
		if e["source_id"] != source {
			return errors.New("источник")
		}
		i.events = append(i.events, e)
	}
	return nil
}

// Шлюз: задания и события операций MES → факты через обычный приём; шаг —
// по коду операции BPMN фланца, изделие и исполнитель — по соответствиям.
func TestGatewayInbound(t *testing.T) {
	c, _ := channel(t)
	bpmn, err := os.ReadFile(filepath.Join(root(), "normative/process/flange-process.bpmn"))
	if err != nil {
		t.Fatal(err)
	}
	steps, err := app.StepsFromBPMN(bpmn)
	if err != nil || steps["030"] != "welding.weld" || steps["010"] != "machining.cnc" {
		t.Fatalf("шаги по коду операции: %v %v", steps, err)
	}
	in := &intake{}
	g := &app.Gateway{Channel: c, Intake: in, Env: func(context.Context) (dom.Env, error) {
		return dom.Env{Steps: steps, Items: map[string]string{"ФЛ-100.00.000#0007": "ENT01:F-007"}, Persons: map[string]string{"TAB-1142": "welder-07"},
			Equipment: map[string]string{"IS-1": "IS-1"}}, nil
	}}
	res, err := g.PullOnce(context.Background())
	if err != nil || res.Submitted != 4 || len(res.Deferred) != 0 {
		t.Fatalf("опрос: %+v %v", res, err)
	}
	var types []string
	for _, e := range in.events {
		types = append(types, e["event_type"].(string))
	}
	if !slices.Equal(types, []string{"mes.job.received", "mes.job.received", "operation.run.started", "operation.run.finished"}) {
		t.Fatalf("типы: %v", types)
	}
	st := in.events[2]
	d := st["data"].(map[string]any)
	if st["item_id"] != "ENT01:F-007" || d["step_key"] != "welding.weld" || d["operator_id"] != "welder-07" || d["operation_run_id"] != "MES-SRSP-4471" ||
		st["source_kind"] != "external_system" {
		t.Fatalf("начало операции из MES: %v", st)
	}
	// Без соответствия экземпляра — отложено, факт не выдумывается (FR-123).
	g.Env = func(context.Context) (dom.Env, error) { return dom.Env{Steps: steps}, nil }
	res, _ = g.PullOnce(context.Background())
	if res.Submitted != 2 || len(res.Deferred) != 2 {
		t.Fatalf("без соответствия: %+v", res)
	}
}

// store — проекции mes.* над записями журнала в памяти.
type store struct{ m map[string]json.RawMessage }

func (s *store) Get(_ context.Context, name, key string) (json.RawMessage, bool, error) {
	v, ok := s.m[name+"/"+key]
	return v, ok, nil
}

func (s *store) project(t *testing.T, codec *engineapp.Codec, j *enginemem.Journal) {
	t.Helper()
	s.m = map[string]json.RawMessage{}
	for _, e := range j.Entries() {
		d, err := codec.Decode(context.Background(), e)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range app.Projections() {
			for _, k := range p.Keys(d.Record) {
				v, err := p.Step(k, s.m[p.Name+"/"+k], d.Record)
				if err != nil {
					t.Fatal(err)
				}
				s.m[p.Name+"/"+k] = v
			}
		}
	}
}

// Сквозной путь блока: сдерживание изделия → реакция mes.hold.requested →
// отправка в MES → mes.hold.responded → операция mes.block.list; повторный
// проход реакции ничего не добавляет.
func TestHoldToMES(t *testing.T) {
	ctx := context.Background()
	j := enginemem.New(func() time.Time { return t0 })
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", Partitions: 4, Now: func() time.Time { return t0 }}
	pend, err := codec.Encode(ctx, engineapp.Out{EventID: "019a0000-0000-7000-8000-00000000c001", Type: catalog.DecisionContainmentApplied,
		Kind: catalog.KindReaction, Stream: "item:ENT01:F-007", ItemID: "ENT01:F-007", OccurredAt: t0,
		Reaction: &engineapp.ReactionMeta{RuleID: "nc.hold", AutomationMode: 2, Slot: engineapp.SlotMeta{RuleID: "nc.hold", Subject: "item:ENT01:F-007"}, Version: 1, Causes: []string{}},
		Data:     ev.DecisionContainmentAppliedV1{Level: ev.AxisContainmentItemHold, Basis: []ev.UUID{"019a0000-0000-7000-8000-00000000c000"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{pend}}); err != nil {
		t.Fatal(err)
	}
	react := &app.HoldReactor{Codec: codec, Store: j}
	rq, err := react.Apply(ctx, j.Entries())
	if err != nil || len(rq.Batch) != 1 {
		t.Fatalf("реакция: %d записей %v", len(rq.Batch), err)
	}
	if _, err := j.Append(ctx, rq); err != nil {
		t.Fatal(err)
	}
	if rq, _ := react.Apply(ctx, j.Entries()[:1]); len(rq.Batch) != 0 {
		t.Fatal("повторный проход реакции дал новую запись")
	}
	st := &store{}
	st.project(t, codec, j)
	c, m := channel(t)
	snd := &app.Sender{Journal: j, Codec: codec, Store: st, Channel: c, Now: func() time.Time { return t0 }}
	if n, err := snd.SendDue(ctx, appjournal.Fence{}); err != nil || n != 1 {
		t.Fatalf("отправка: %d %v", n, err)
	}
	st.project(t, codec, j)
	if n, _ := snd.SendDue(ctx, appjournal.Fence{}); n != 0 {
		t.Fatal("подтверждённый блок отправлен повторно")
	}
	svc := app.NewLive(app.Config{Projections: st})
	list, err := svc.Blocks(ctx, platform.Moment{})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("mes.block.list: %+v %v", list, err)
	}
	b := list.Items[0]
	if b.ItemID != "ENT01:F-007" || !b.Hold || b.Outcome != "accepted" || b.BusinessKey != "ENT01:F-007/hold/1" {
		t.Fatalf("блок: %+v", b)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.got[dom.MessageID("ENT01:F-007/hold/1")] != 1 {
		t.Fatalf("MES получила блок %v", m.got)
	}
}
