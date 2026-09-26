package stand

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"
	"uuid"

	app "ant/internal/application/ingest"
	"ant/internal/infrastructure/integration/access/skud"
	"ant/internal/infrastructure/integration/ingest/stands"
)

// Name — имя stand-а в роли stands (/stand/skud/) и в сценариях.
const Name = "skud"

// Zone — зона доступа (здание → цех; FR-82): id как в справочнике мест
// (access_zone_id) и название.
type Zone struct {
	ID   string
	Name string
}

// Holder — владелец пропуска: псевдоним сотрудника (кейс §4.6), имя и
// «домашняя» зона — куда он проходит в начале смены.
type Holder struct {
	ID   string
	Name string
	Home string
}

// Options — настройки stand-а.
type Options struct {
	Zones   []Zone
	Holders []Holder
	// Now — часы stand-а (время проходов); nil — системные.
	Now func() time.Time
	// Arrive — при старте все владельцы с домашней зоной проходят в неё
	// («смена пришла»): панель «Посты» сразу видит присутствие.
	Arrive bool
}

// Stand — эмулятор СКУД (AD-18, NFR-TEST-2): турникеты зон, журнал событий
// проходов и HTTP-протокол skud.v1, который опрашивает адаптер
// infrastructure/integration/access/skud. Состояние — в памяти процесса роли
// stands (граница эмуляции: нет считывателей, карт и биометрии — проход
// заказывается кнопкой страницы /stand/skud/ или POST …/passes).
type Stand struct {
	opt    Options
	faults *stands.FaultSwitch

	mu     sync.Mutex
	events []skud.Event
	inside map[string]map[string]bool // владелец → зона → внутри
}

// New — stand СКУД; с Arrive — проходы «смена пришла».
func New(opt Options) *Stand {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	s := &Stand{opt: opt, inside: map[string]map[string]bool{},
		faults: stands.NewFaultSwitch([]app.FaultKind{app.FaultOffline, app.FaultDelay, app.FaultError}, opt.Now)}
	if opt.Arrive {
		for _, h := range opt.Holders {
			if h.Home != "" {
				_, _ = s.Pass(h.ID, h.Home, true)
			}
		}
	}
	return s
}

var _ stands.Stand = (*Stand)(nil)

// Info — что эмулирует и где граница эмуляции.
func (s *Stand) Info() app.StandInfo {
	return app.StandInfo{Name: Name, Emulates: "СКУД предприятия: проходы через турникеты зон цехов (протокол skud.v1)",
		Boundary: "без считывателей, карт и биометрии; проход — кнопкой страницы /stand/skud/ или POST /stand/skud/api/v1/passes; состояние в памяти"}
}

// Faults — сбои stand-а: недоступен, задержка, ошибка ответа.
func (s *Stand) Faults() *stands.FaultSwitch { return s.faults }

// Run — фоновой работы нет: проходы — по запросу.
func (s *Stand) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// Pass — проход владельца holder через турникет зоны zone (enter — вход).
// Повторный вход в зону, где он уже есть (и выход из зоны, где его нет), —
// не проход: турникет не пропустит.
func (s *Stand) Pass(holder, zone string, enter bool) (skud.Event, error) {
	if !slices.ContainsFunc(s.opt.Zones, func(z Zone) bool { return z.ID == zone }) {
		return skud.Event{}, fmt.Errorf("зона %q не описана в СКУД", zone)
	}
	if !slices.ContainsFunc(s.opt.Holders, func(h Holder) bool { return h.ID == holder }) {
		return skud.Event{}, fmt.Errorf("пропуск %q не выдан", holder)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inside[holder] == nil {
		s.inside[holder] = map[string]bool{}
	}
	if s.inside[holder][zone] == enter {
		return skud.Event{}, fmt.Errorf("турникет не пропустил: %s уже %s зоны %s", holder, map[bool]string{true: "внутри", false: "вне"}[enter], zone)
	}
	s.inside[holder][zone] = enter
	dir := skud.DirOut
	if enter {
		dir = skud.DirIn
	}
	e := skud.Event{Seq: int64(len(s.events) + 1), ID: uuid.NewV7().String(), Time: s.opt.Now().UTC().Truncate(time.Millisecond),
		Holder: holder, Zone: zone, Reader: "R-" + zone + "-1", Direction: dir}
	s.events = append(s.events, e)
	return e, nil
}

// Handler — протокол skud.v1 и служебная страница «глазами СКУД».
func (s *Stand) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/about", func(w http.ResponseWriter, _ *http.Request) {
		zs := make([]skud.ZoneInfo, len(s.opt.Zones))
		for i, z := range s.opt.Zones {
			zs[i] = skud.ZoneInfo{ID: z.ID, Name: z.Name}
		}
		writeJSON(w, http.StatusOK, skud.About{System: "СКУД (stand)", Contract: skud.Contract, Zones: zs})
	})
	mux.HandleFunc("GET /api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 1000 {
			limit = 500
		}
		s.mu.Lock()
		out := skud.EventPage{Events: []skud.Event{}, LastSeq: int64(len(s.events))}
		for _, e := range s.events {
			if e.Seq > after && len(out.Events) < limit {
				out.Events = append(out.Events, e)
			}
		}
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, out)
	})
	mux.HandleFunc("POST /api/v1/passes", func(w http.ResponseWriter, r *http.Request) {
		var rq skud.PassRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&rq); err != nil {
			http.Error(w, "проход не разобран: "+err.Error(), http.StatusBadRequest)
			return
		}
		e, err := s.Pass(rq.Holder, rq.Zone, rq.Direction == skud.DirIn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, http.StatusCreated, e)
	})
	mux.HandleFunc("POST /ui/pass", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		back := "/stand/" + Name + "/"
		if _, err := s.Pass(r.Form.Get("holder"), r.Form.Get("zone"), r.Form.Get("direction") == skud.DirIn); err != nil {
			back += "?msg=" + url.QueryEscape(err.Error())
		}
		http.Redirect(w, r, back, http.StatusSeeOther)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { s.page(w, r.URL.Query().Get("msg")) })
	return mux
}

// pageRow — строка страницы: владелец пропуска и где он сейчас.
type pageRow struct {
	Holder
	Inside []string
}

var pageTpl = template.Must(template.New("skud").Parse(`<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><title>СКУД — stand «Главного»</title>
<style>body{font:14px system-ui,sans-serif;margin:24px;color:#1f2933}table{border-collapse:collapse}td,th{border-bottom:1px solid #d9e2ec;padding:6px 10px;text-align:left}
form{display:inline}button{margin:2px;padding:3px 8px}.msg{color:#b42318;margin:8px 0}.in{color:#067647;font-weight:600}</style></head>
<body><h1>СКУД (stand)</h1><p>Эмулятор турникетов зон цехов: «Главный» опрашивает журнал проходов по протоколу skud.v1 и видит присутствие на постах.</p>
{{if .Msg}}<p class="msg">{{.Msg}}</p>{{end}}
<table><tr><th>Пропуск</th><th>Сотрудник</th><th>Сейчас в зонах</th><th>Проход</th></tr>
{{range .Rows}}<tr><td>{{.ID}}</td><td>{{.Name}}</td><td class="in">{{range .Inside}}{{.}} {{end}}</td><td>
{{$h := .ID}}{{range $.Zones}}<form method="post" action="/stand/skud/ui/pass"><input type="hidden" name="holder" value="{{$h}}"><input type="hidden" name="zone" value="{{.ID}}"><button name="direction" value="in" title="{{.Name}}">→ {{.ID}}</button><button name="direction" value="out" title="{{.Name}}">{{.ID}} →</button></form>{{end}}
</td></tr>{{end}}</table>
<h2>Последние проходы</h2><table><tr><th>№</th><th>Время</th><th>Пропуск</th><th>Зона</th><th>Направление</th></tr>
{{range .Last}}<tr><td>{{.Seq}}</td><td>{{.Time.Format "15:04:05"}}</td><td>{{.Holder}}</td><td>{{.Zone}}</td><td>{{if eq .Direction "in"}}вход{{else}}выход{{end}}</td></tr>{{end}}</table>
</body></html>`))

func (s *Stand) page(w http.ResponseWriter, msg string) {
	s.mu.Lock()
	rows := make([]pageRow, 0, len(s.opt.Holders))
	for _, h := range s.opt.Holders {
		r := pageRow{Holder: h}
		for _, z := range s.opt.Zones {
			if s.inside[h.ID][z.ID] {
				r.Inside = append(r.Inside, z.ID)
			}
		}
		rows = append(rows, r)
	}
	var last []skud.Event
	for i := len(s.events) - 1; i >= 0 && len(last) < 20; i-- {
		last = append(last, s.events[i])
	}
	s.mu.Unlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTpl.Execute(w, map[string]any{"Msg": msg, "Rows": rows, "Zones": s.opt.Zones, "Last": last})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
