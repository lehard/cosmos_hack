package stand

import (
	"cmp"
	"encoding/json"
	"html/template"
	"net/http"
	"slices"
	"strings"
	"time"

	app "ant/internal/application/ingest"
)

// Страница /stand/1c/ — полученное «глазами 1С» (AD-18, FR-91): журнал
// обмена HTTP-сервиса qc, документы 1С по сообщениям, где числится изделие и
// его качество, этапы производства, номенклатура и серии, включённые сбои.
// Оформление — в духе интерфейса 1С «Такси»: жёлтая панель разделов, серые
// таблицы, жёлтая кнопка по умолчанию. Обновляется сама раз в 5 секунд.

type pageData struct {
	Now       string
	Created   string
	Messages  []pageMessage
	Documents []Document
	Stock     []stock
	Stages    []Row
	Nom       []Row
	Series    []Row
	Faults    []app.Fault
	Supported []app.FaultKind
	Due       string
}

type pageMessage struct {
	Message
	Short string
}

// stock — где числится субъект по документам 1С и его качество.
type stock struct {
	Subject   string
	Warehouse string
	Quality   string
	Last      string
	Date      time.Time
}

var tpl = template.Must(template.New("page").Funcs(template.FuncMap{
	"t": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.In(time.FixedZone("MSK", 3*3600)).Format("02.01.2006 15:04:05")
	},
	"s": func(v any) string { return strings.TrimSpace(strings.Trim(jsonString(v), `"`)) },
	"one": func(v any) any {
		xs, _ := v.([]any)
		if len(xs) > 0 {
			return xs[0]
		}
		return map[string]any{}
	},
}).Parse(pageHTML))

func jsonString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func (s *Stand) page(w http.ResponseWriter, r *http.Request) {
	st, err := s.Snapshot(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d := pageData{Now: s.opt.Now().In(time.FixedZone("MSK", 3*3600)).Format("02.01.2006 15:04:05"), Created: r.URL.Query().Get("created"),
		Faults: s.faults.Active(), Supported: s.faults.Supported(), Due: s.opt.Now().AddDate(0, 0, 21).Format("2006-01-02")}
	for i := len(st.Messages) - 1; i >= 0; i-- {
		m := st.Messages[i]
		short := m.BusinessKey
		d.Messages = append(d.Messages, pageMessage{Message: m, Short: short})
	}
	docs := slices.Clone(st.Documents)
	slices.SortStableFunc(docs, func(a, b Document) int { return b.Date.Compare(a.Date) })
	d.Documents = docs
	// Регистр «где числится и какого качества» — по документам в порядке проведения.
	pos := map[string]*stock{}
	var order []string
	for _, doc := range st.Documents {
		if doc.Storno || doc.Subject == "" {
			continue
		}
		x, ok := pos[doc.Subject]
		if !ok {
			x = &stock{Subject: doc.Subject, Quality: "Новый"}
			pos[doc.Subject] = x
			order = append(order, doc.Subject)
		}
		if doc.WarehouseTo != "" {
			x.Warehouse = doc.WarehouseTo
		} else if x.Warehouse == "" && doc.WarehouseFrom != "" {
			x.Warehouse = doc.WarehouseFrom
		}
		if doc.Quality != "" {
			x.Quality = doc.Quality
		}
		if strings.HasPrefix(doc.Type, "Document_ВнутреннееПотребление") || strings.HasPrefix(doc.Type, "Document_ВозвратТоваров") || strings.HasPrefix(doc.Type, "Document_Порча") {
			x.Warehouse = "— (списано / возвращено)"
		}
		x.Last, x.Date = doc.Title, doc.Date
	}
	for _, k := range order {
		d.Stock = append(d.Stock, *pos[k])
	}
	slices.SortFunc(d.Stock, func(a, b stock) int { return cmp.Compare(a.Subject, b.Subject) })
	stages, _ := s.rows(r.Context(), "Document_ЭтапПроизводства2_2")
	d.Stages = stages
	d.Nom, _ = s.rows(r.Context(), "Catalog_Номенклатура")
	d.Series, _ = s.rows(r.Context(), "Catalog_СерииНоменклатуры")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = tpl.Execute(w, d)
}

const pageHTML = `<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><meta http-equiv="refresh" content="5">
<title>1С:Предприятие — ERP (stand ant)</title>
<style>
 body{margin:0;font:13px Arial,Helvetica,sans-serif;color:#333;background:#fff}
 .top{background:#fbed9e;border-bottom:1px solid #e3cf6d;padding:6px 12px;display:flex;gap:16px;align-items:center}
 .top b{font-size:15px}.top .mut{color:#6b5d1a}
 .wrap{display:flex}
 .side{width:210px;background:#f2f2f2;border-right:1px solid #d9d9d9;min-height:100vh;padding:8px 0}
 .side a{display:block;padding:6px 14px;color:#333;text-decoration:none}.side a:hover{background:#fbed9e}
 .main{flex:1;padding:10px 16px}
 h2{font-size:15px;font-weight:normal;color:#4d4d4d;border-bottom:1px solid #d9d9d9;padding-bottom:4px;margin:18px 0 8px}
 table{border-collapse:collapse;width:100%;margin-bottom:6px}
 th{background:#f0f0f0;border:1px solid #d0d0d0;font-weight:normal;text-align:left;padding:4px 6px;color:#555}
 td{border:1px solid #e0e0e0;padding:3px 6px;vertical-align:top}
 tr:nth-child(even) td{background:#fafafa}
 .ok{color:#2e7d32}.bad{color:#c62828}.warn{color:#b26a00}.gray{color:#888}
 .storno td{text-decoration:line-through;color:#999}
 .btn{background:#fbd84d;border:1px solid #d9b52c;border-radius:3px;padding:4px 14px;cursor:pointer}
 .note{background:#fff8d6;border:1px solid #efdf92;padding:6px 10px;margin:8px 0}
 code{font-size:12px}
</style></head><body>
<div class="top"><b>1С:Предприятие</b><span>ERP Управление предприятием 2.5 · расширение «КонтрольКачества» qc.v1</span>
<span class="mut">stand ant — эмулятор 1С (AD-18) · {{.Now}}</span></div>
<div class="wrap"><div class="side">
<a href="#exchange">Журнал обмена (qc)</a><a href="#docs">Документы</a><a href="#stock">Склады и качество</a>
<a href="#stages">Этапы производства</a><a href="#nom">Номенклатура и серии</a><a href="#faults">Сбои stand-а</a>
</div><div class="main">
{{if .Created}}<div class="note">Создан этап производства {{.Created}} — ant получит задание при следующем опросе OData.</div>{{end}}
{{if .Faults}}<div class="note warn">Включены сбои: {{range .Faults}}<b>{{.Kind}}</b>{{if .Param}} {{.Param}}{{end}}{{if .Match}} по «{{.Match}}»{{end}}{{if .Detail}} — {{.Detail}}{{end}}; {{end}}</div>{{end}}

<h2 id="exchange">Журнал обмена HTTP-сервиса qc (регистр входящих сообщений)</h2>
<table><tr><th>Получено</th><th>Номер сообщения (X-Message-Id)</th><th>Вид</th><th>Бизнес-ключ ant</th><th>Верс.</th><th>Итог</th><th>Квитанция / ошибка</th><th>Доставок</th></tr>
{{range .Messages}}<tr><td>{{t .ReceivedAt}}</td><td><code>{{.MessageID}}</code></td><td>{{.Kind}}</td><td><code>{{.Short}}</code></td><td>{{.Version}}</td>
<td>{{if .Receipt}}<span class="ok">принято ({{.HTTPStatus}})</span>{{else}}<span class="bad">отклонено ({{.HTTPStatus}})</span>{{end}}</td>
<td>{{if .Receipt}}{{.Receipt.Receipt}} · {{.Receipt.DocumentNumber}}{{else if .Error}}{{.Error.Code}}: {{.Error.Message}}{{end}}{{range .Faults}}<div class="warn">сбой: {{.}}</div>{{end}}</td>
<td>{{.Deliveries}}{{if gt .Deliveries 1}} <span class="gray">(повтор — та же квитанция)</span>{{end}}{{if .Lost}} <span class="warn">ответ потерян: {{.Lost}}</span>{{end}}</td></tr>
{{else}}<tr><td colspan="8" class="gray">Сообщений от ant ещё не было</td></tr>{{end}}</table>

<h2 id="docs">Документы, созданные по сообщениям</h2>
<table><tr><th>Дата</th><th>Документ</th><th>Номер</th><th>Изделие / партия</th><th>Со склада</th><th>На склад</th><th>Качество</th><th>Комментарий</th></tr>
{{range .Documents}}<tr{{if .Storno}} class="storno"{{end}}><td>{{t .Date}}</td><td>{{.Title}}{{if .Storno}} (сторнирован){{end}}</td><td>{{.Number}}</td>
<td>{{.Subject}}{{if .Quantity}} × {{.Quantity}}{{end}}</td><td>{{.WarehouseFrom}}</td><td>{{.WarehouseTo}}</td><td>{{.Quality}}</td><td>{{.Comment}}</td></tr>
{{else}}<tr><td colspan="8" class="gray">Документов нет</td></tr>{{end}}</table>

<h2 id="stock">Где числится изделие и его качество (по документам 1С)</h2>
<table><tr><th>Изделие / партия</th><th>Склад</th><th>Качество</th><th>Последний документ</th><th>Дата</th></tr>
{{range .Stock}}<tr><td>{{.Subject}}</td><td>{{.Warehouse}}</td><td>{{if eq .Quality "Не годен"}}<span class="bad">{{.Quality}}</span>{{else}}{{.Quality}}{{end}}</td><td>{{.Last}}</td><td>{{t .Date}}</td></tr>
{{else}}<tr><td colspan="5" class="gray">Движений нет</td></tr>{{end}}</table>

<h2 id="stages">Этапы производства (OData: Document_ЭтапПроизводства2_2)</h2>
<table><tr><th>Номер</th><th>Дата</th><th>Статус</th><th>Количество</th><th>Срок</th><th>Ref_Key</th></tr>
{{range .Stages}}<tr><td>{{s (index . "Number")}}</td><td>{{s (index . "Date")}}</td><td>{{s (index . "Статус")}}</td>
<td>{{with one (index . "ВыходныеИзделия")}}{{s (index . "Количество")}}{{end}}</td><td>{{s (index . "ДатаОкончания")}}</td><td><code>{{s (index . "Ref_Key")}}</code></td></tr>{{end}}</table>
<form method="post" action="stages">Новый этап к выполнению: фланец ФЛ-100.00.000 СБ, количество <input name="quantity" value="6" size="4">
срок <input name="due" value="{{.Due}}" size="10"> <button class="btn">Провести и закрыть</button></form>

<h2 id="nom">Номенклатура и серии</h2>
<table><tr><th>Артикул</th><th>Наименование</th><th>Ref_Key</th></tr>
{{range .Nom}}<tr><td>{{s (index . "Артикул")}}</td><td>{{s (index . "Description")}}</td><td><code>{{s (index . "Ref_Key")}}</code></td></tr>{{end}}</table>
<table><tr><th>Серия</th><th>Плавка</th><th>Сертификат</th><th>Годен до</th></tr>
{{range .Series}}<tr><td>{{s (index . "Номер")}}</td><td>{{s (index . "НомерПлавки")}}</td><td>{{s (index . "НомерСертификата")}}</td><td>{{s (index . "ГоденДо")}}</td></tr>{{end}}</table>

<h2 id="faults">Сбои stand-а</h2>
<p>Сбои включаются только со страницы тестовых сценариев через служебный порт <code>/stand/_control/1c/faults</code> (AD-18).
Поддерживаются: {{range .Supported}}<b>{{.}}</b> {{end}}— недоступность (разрыв), ошибка (param — код ответа: 503 по умолчанию, 422 — ошибка данных; match — образец «вид:изделие»),
задержка (мс), дубль (документ проведён, ответ потерян), несовместимые метаданные ($metadata с переименованным реквизитом).</p>
</div></div></body></html>`
