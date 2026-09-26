package stand

import (
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strconv"
)

// Страница stand-а «глазами MES»: экземпляры и партии с блоками ОТК,
// кнопки «выдать операцию» (заблокированному — отказ), «завершить
// операцию», «выдать задание»; журнал сообщений Главного.

var pageTpl = template.Must(template.New("mes").Parse(`<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><title>MES — stand «Главного»</title>
<style>body{font:14px system-ui,sans-serif;margin:24px;color:#1f2933}table{border-collapse:collapse;margin-bottom:16px}td,th{border-bottom:1px solid #d9e2ec;padding:5px 9px;text-align:left;vertical-align:top}
th{background:#f0f4f8}form{display:inline-block;margin:2px 12px 2px 0}input{width:12em}button{padding:3px 10px}.msg{color:#067647;margin:8px 0}.err{color:#b42318}.muted{color:#627d98}.hold{color:#b42318;font-weight:600}.free{color:#067647}</style></head>
<body><h1>MES цеха (stand)</h1>
<p>Эмулятор MES по B2MML V7 (контракт <b>{{range .About.Supported}}{{.}} {{end}}</b>, релиз {{.About.ReleaseID}}, логический ID <b>{{.About.LogicalID}}</b>): «Главный» передаёт сюда блоки ОТК и их снятие, MES подтверждает их ConfirmBOD и не выдаёт заблокированному экземпляру следующую операцию.
Привязка: <code>/stand/mes/b2mml/</code>.</p>
{{if .Msg}}<p class="msg {{if .Err}}err{{end}}">{{.Msg}}</p>{{end}}
<h2>Экземпляры и партии</h2>
<table><tr><th>ID</th><th>Состояние</th><th>Где</th><th>Основание</th><th>Изменено</th><th>Операция</th></tr>
{{range .Materials}}<tr><td>{{.ID}}{{if .Lot}} (партия){{end}}</td><td>{{if .Blocked}}<span class="hold">заблокирован ОТК ({{.Status}})</span>{{else}}<span class="free">свободен ({{.Status}})</span>{{end}}</td><td>{{.Location}}</td><td>{{.Description}}</td><td>{{.UpdatedAt.Format "15:04:05"}}</td>
<td>{{if not .Lot}}<form method="post" action="/stand/mes/ui/operation"><input type="hidden" name="sublot" value="{{.ID}}"><select name="code">{{range $.Operations}}<option>{{.}}</option>{{end}}</select> <button>Выдать операцию</button></form>{{end}}</td></tr>
{{else}}<tr><td colspan="6" class="muted">блоков от Главного ещё не было</td></tr>{{end}}</table>
<form method="post" action="/stand/mes/ui/operation">Экземпляр <input name="sublot" placeholder="ENT01:F-001"> операция <select name="code">{{range .Operations}}<option>{{.}}</option>{{end}}</select> <button>Выдать операцию</button></form>
<form method="post" action="/stand/mes/ui/schedule">Задание на фланцы: <input type="number" name="quantity" value="6" min="1" max="1000" style="width:5em"> шт. <button>Выдать задание</button></form>
<h2>Операции MES</h2>
<table><tr><th>Время</th><th>Экземпляр</th><th>Операция</th><th>Выполнение</th><th>Итог</th><th></th></tr>
{{range .OperationsLog}}<tr><td>{{.At.Format "15:04:05"}}</td><td>{{.SubLot}}</td><td>{{.Code}}</td><td>{{.Run}}</td>
<td>{{if .Refusal}}<span class="hold">отказ: {{.Reason}}</span>{{else if eq .State "started"}}выполняется (событие Start → Главный){{else}}завершена (событие End → Главный){{end}}</td>
<td>{{if eq .State "started"}}<form method="post" action="/stand/mes/ui/finish"><input type="hidden" name="run" value="{{.Run}}"><button>Завершить</button></form>{{end}}</td></tr>
{{else}}<tr><td colspan="6" class="muted">операций не выдавали</td></tr>{{end}}</table>
<p class="muted">Сообщений для Главного в outbox: {{.Outbox}}.</p>
<h2>Главный → MES: журнал обмена</h2>
<table><tr><th>Получено</th><th>BODID</th><th>Сообщение</th><th>Субъект</th><th>Disposition</th><th>ConfirmBOD</th><th>Доставок</th><th>Сбои</th></tr>
{{range .Messages}}<tr><td>{{.ReceivedAt.Format "15:04:05"}}</td><td><code>{{.BODID}}</code></td><td>{{.Message}}</td><td>{{range .Subjects}}{{.}} {{end}}</td><td>{{.Disposition}}</td>
<td>{{if eq .Outcome "accepted"}}<span class="free">успех</span>{{else}}<span class="hold">сбой</span>{{end}}</td><td>{{.Deliveries}}{{if .Lost}} (ответ потерян {{.Lost}}){{end}}</td><td>{{range .Faults}}{{.}}<br>{{end}}</td></tr>
{{else}}<tr><td colspan="8" class="muted">сообщений ещё не было</td></tr>{{end}}</table>
<p class="muted">Сбои (недоступность, ошибка по образцу, задержка, дубль, несовместимая шина) включает только пульт тестовых сценариев «Главного».</p>
</body></html>`))

func (s *Stand) page(w http.ResponseWriter, r *http.Request) {
	snap := s.Snapshot()
	slices.Reverse(snap.Messages)
	slices.Reverse(snap.Operations)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTpl.Execute(w, map[string]any{"About": s.About(), "Msg": r.URL.Query().Get("msg"), "Err": r.URL.Query().Get("err") != "",
		"Operations": Operations, "Materials": snap.Materials, "Messages": snap.Messages, "Outbox": snap.Outbox,
		"OperationsLog": snap.Operations})
}

func back(w http.ResponseWriter, r *http.Request, msg string, err error) {
	q := url.Values{}
	if err != nil {
		q.Set("msg", err.Error())
		q.Set("err", "1")
	} else {
		q.Set("msg", msg)
	}
	http.Redirect(w, r, "/stand/"+Name+"/?"+q.Encode(), http.StatusSeeOther)
}

func (s *Stand) uiOperation(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	op, err := s.NextOperation(r.PostForm.Get("sublot"), r.PostForm.Get("code"))
	if err != nil && op.Refusal {
		err = &refusal{op: op}
	}
	back(w, r, "Операция "+op.Code+" выдана экземпляру "+op.SubLot+" — событие Start передаётся Главному", err)
}

type refusal struct{ op Operation }

func (e *refusal) Error() string {
	return "MES отказала в операции " + e.op.Code + " экземпляру " + e.op.SubLot + ": " + e.op.Reason
}

func (s *Stand) uiFinish(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	op, err := s.FinishOperation(r.PostForm.Get("run"))
	back(w, r, "Операция "+op.Code+" экземпляра "+op.SubLot+" завершена — событие End передаётся Главному", err)
}

func (s *Stand) uiSchedule(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	qty, _ := strconv.Atoi(r.PostForm.Get("quantity"))
	req, err := s.Schedule(qty)
	back(w, r, "Задание "+req+" передаётся Главному", err)
}
