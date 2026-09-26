package stand

import (
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"
)

// Страница stand-а «глазами Галактики»: описание обработчика, кнопки
// «выдать сменное задание» и «поступила партия», журнал обмена, документы
// Галактики и пакеты для Главного с его квитанциями.

var pageTpl = template.Must(template.New("galaktika").Parse(`<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><title>Галактика ERP — stand «Главного»</title>
<style>body{font:14px system-ui,sans-serif;margin:24px;color:#1f2933}table{border-collapse:collapse;margin-bottom:16px}td,th{border-bottom:1px solid #d9e2ec;padding:5px 9px;text-align:left;vertical-align:top}
th{background:#f0f4f8}form{display:inline-block;margin:4px 16px 4px 0}input{width:7em}button{padding:3px 10px}.msg{color:#067647;margin:8px 0}.err{color:#b42318}.muted{color:#627d98}.ok{color:#067647}.bad{color:#b42318}</style></head>
<body><h1>Галактика ERP (stand)</h1>
<p>Эмулятор Галактики ERP 9.x с обработчиком обмена <b>gal.qc.v1</b>: «Главный» передаёт сюда результаты контроля и учётные действия, Галактика отвечает квитанциями GalAck и выдаёт задания и партии.
Узел <b>{{.About.Node}}</b>, база <b>{{.About.Database}}</b>, контракт <b>{{range .About.Supported}}{{.}} {{end}}</b>.
Каталог обмена: {{if .Dir}}<code>{{.Dir}}</code>{{else}}<span class="muted">не подключён</span>{{end}}; REST-фасад: <code>/stand/galaktika/esb/v1/</code>.</p>
{{if .Msg}}<p class="msg {{if .Err}}err{{end}}">{{.Msg}}</p>{{end}}
<h2>Галактика → Главный</h2>
<form method="post" action="/stand/galaktika/ui/task">Сменное задание на фланец ФЛ-100.00.000 СБ: <input type="number" name="quantity" value="6" min="1" max="1000"> шт., срок <input type="date" name="due" value="{{.Due}}" style="width:10em"> <button>Выдать задание</button></form>
<form method="post" action="/stand/galaktika/ui/lot">Поступила партия патрубков: № <input name="number" value="{{.LotNo}}" style="width:9em"> <input type="number" name="quantity" value="12" min="1"> шт. <button>Оприходовать</button></form>
<table><tr><th>Пакет</th><th>Что</th><th>Сформирован</th><th>Квитанция Главного</th></tr>
{{range .Packets}}<tr><td>{{.MessageID}}</td><td>{{.Title}}</td><td>{{.CreatedAt.Format "15:04:05"}}</td><td>{{if eq .AckStatus "ok"}}<span class="ok">принято</span>{{else if .AckStatus}}<span class="bad">{{.AckText}}</span>{{else}}<span class="muted">{{if $.Dir}}ещё нет{{else}}забирается фасадом{{end}}</span>{{end}}</td></tr>
{{else}}<tr><td colspan="4" class="muted">пакетов нет</td></tr>{{end}}</table>
<h2>Главный → Галактика: журнал обмена</h2>
<table><tr><th>Получен</th><th>Номер пакета</th><th>Транспорт</th><th>Вид</th><th>Субъект</th><th>Квитанция</th><th>Доставок</th><th>Сбои</th></tr>
{{range .Messages}}<tr><td>{{.ReceivedAt.Format "15:04:05"}}</td><td><code>{{.MessageID}}</code></td><td>{{.Transport}}</td><td>{{.Kind}}</td><td>{{.Subject}}</td>
<td>{{if eq .Status "ok"}}<span class="ok">ok — {{.Document}}</span>{{else if eq .Status "error"}}<span class="bad">{{.Code}}: {{.Text}}</span>{{else}}<span class="bad">сбой</span>{{end}}</td>
<td>{{.Deliveries}}{{if .Lost}} (ответ потерян {{.Lost}}){{end}}</td><td>{{range .Faults}}{{.}}<br>{{end}}</td></tr>
{{else}}<tr><td colspan="8" class="muted">пакетов от Главного ещё не было</td></tr>{{end}}</table>
<h2>Документы Галактики</h2>
<table><tr><th>Номер</th><th>NRec</th><th>Документ</th><th>Субъект</th><th>Примечание</th><th>Дата</th></tr>
{{range .Documents}}<tr{{if .Storno}} class="muted"{{end}}><td>{{.Number}}{{if .Storno}} (сторно){{end}}</td><td>{{.NRec}}</td><td>{{.Title}}</td><td>{{.Subject}}</td><td>{{.Comment}}</td><td>{{.Date.Format "02.01.2006 15:04:05"}}</td></tr>
{{else}}<tr><td colspan="6" class="muted">документов нет</td></tr>{{end}}</table>
<p class="muted">Сбои (недоступность, ошибка по образцу, задержка, дубль, несовместимый обработчик) включает только пульт тестовых сценариев «Главного».</p>
</body></html>`))

func (s *Stand) page(w http.ResponseWriter, r *http.Request) {
	if s.opt.Dir != "" {
		s.readInAcks()
	}
	snap := s.Snapshot()
	slices.Reverse(snap.Messages)
	slices.Reverse(snap.Documents)
	slices.Reverse(snap.Packets)
	msk := s.opt.Now().In(time.FixedZone("MSK", 3*3600))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pageTpl.Execute(w, map[string]any{"About": s.About(), "Dir": s.opt.Dir, "Msg": r.URL.Query().Get("msg"), "Err": r.URL.Query().Get("err") != "",
		"Due": msk.AddDate(0, 0, 7).Format("2006-01-02"), "LotNo": "П-" + msk.Format("2006-0102"),
		"Messages": snap.Messages, "Documents": snap.Documents, "Packets": snap.Packets})
}

// back — возврат на страницу с сообщением.
func back(w http.ResponseWriter, r *http.Request, p Packet, err error) {
	q := url.Values{}
	if err != nil {
		q.Set("msg", err.Error())
		q.Set("err", "1")
	} else {
		q.Set("msg", p.Title+" — пакет "+p.MessageID+" передан Главному")
	}
	http.Redirect(w, r, "/stand/"+Name+"/?"+q.Encode(), http.StatusSeeOther)
}

func (s *Stand) uiTask(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	qty, _ := strconv.Atoi(r.PostForm.Get("quantity"))
	p, err := s.Task(qty, r.PostForm.Get("due"))
	back(w, r, p, err)
}

func (s *Stand) uiLot(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	qty, _ := strconv.Atoi(r.PostForm.Get("quantity"))
	p, err := s.Lot(r.PostForm.Get("number"), qty)
	back(w, r, p, err)
}
