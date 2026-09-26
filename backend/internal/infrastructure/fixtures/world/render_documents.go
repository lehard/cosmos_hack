package world

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"time"

	docsapp "ant/internal/application/documents"
	itemapp "ant/internal/application/item"
	"ant/internal/application/platform"
	dom "ant/internal/domain/documents"
	"ant/internal/infrastructure/fixtures/loader"
)

// Документы мира заготовок (FR-65, FR-66, FR-139; AD-12, AD-13, AD-43):
// реестр документов раздела «Документы» столов, документ с маршрутом подписей
// и каноническая отрисовка — тем же построителем domain/documents, что у live
// (Build: канонический content, HTML, rendering_hash, отпечаток). Набор —
// правдоподобная история «Плохого дня»: листы утверждения версий процессов,
// акты входного контроля и ярлык партии, заявления и решения по
// несоответствиям, разрешение на отклонение (возвращено держателем КД), акты о
// браке (один — на бумаге, ждёт подписи ручкой), журнал изолятора, запись
// вмешательства, маршрутный лист, документы выпуска эталона Ф-001 (извещение
// ВП, свидетельство о приёмке с бумажной подписью ВП и заверением,
// сопроводительная карта, паспорт), живые сопроводительные карты изделий в
// работе, журнал предъявления ОТК смены и назначения контролёров на посты.
// Подписи — с ключом и классом его хранения (Д-72): физический ключ или ключ
// в браузере.

// Классы хранения ключа (Д-72).
const (
	keyHW      = "hardware_token"
	keyBrowser = "software_browser"
)

// wsig — подпись этапа.
type wsig struct {
	Person   string
	At       time.Time
	Method   string // token_agent | paper | source_decision
	Storage  string // hardware_token | software_browser (у бумаги и источника — пусто)
	Attester string // заверитель бумажной подписи
	PaperNo  string // учётный номер бумажного оригинала
}

// wstage — этап маршрута подписей.
type wstage struct {
	Title, Role, Authority string
	Level                  int
	Paper                  bool
	BySource               bool
	// Who — кто может подписать этап, если не по роли (автор решения, источник).
	Who []string
	Sig *wsig
}

// wdecline — отказ в согласовании с замечанием.
type wdecline struct {
	Stage   int
	Person  string
	Comment string
	At      time.Time
}

// wpaper — статус бумажного экземпляра.
type wpaper struct {
	Status string // printed | signed | destroyed
	At     time.Time
	CopyNo string
}

// wver — версия документа.
type wver struct {
	At          time.Time
	Stages      []wstage
	Declines    []wdecline
	Paper       []wpaper
	AnnulledAt  time.Time
	AnnulledBy  string
	AnnulReason string
}

// wrow — строка журнала документа с моментом появления.
type wrow struct {
	At  time.Time
	Row map[string]any
}

// wfield — поле содержимого: путь в content (как в шаблоне) и значение.
type wfield struct{ Key, Label, Value string }

// wdoc — документ мира.
type wdoc struct {
	ID, Template, Title string
	Subject             platform.DrillRef
	SubjectLabel        string
	Items               []string // полные ID изделий
	ProcessID, PVID     string
	Fields              []wfield
	// Rows — строки таблиц (сопроводительная карта: операции и несоответствия) на момент.
	Rows func(c *Ctx) map[string][]map[string]any
	// Table — строки журнала документа по триггеру (rows), видимые к моменту — по полю at_time.
	Table []wrow
	// Live — сопроводительная карта собирается из истории, версия не зафиксирована.
	Live     bool
	LiveFrom time.Time
	Versions []wver
}

func sig(person string, at time.Time, storage string) *wsig {
	return &wsig{Person: person, At: at, Method: dom.MethodTokenAgent, Storage: storage}
}

// ─────────────────────────────── набор документов ───────────────────────────────

// documents — документы мира (строятся один раз; видимость — по часам шага).
func (m *Model) documents() []*wdoc {
	if m.docs != nil {
		return m.docs
	}
	at := m.clk.at
	flange := func(d *wdoc) *wdoc { d.ProcessID, d.PVID = FlangeProcessID, ProcessVersionID; return d }
	var out []*wdoc
	add := func(d *wdoc) { out = append(out, d) }

	// Листы утверждения версий процессов (эпик 39): кворум технолог + ОТК + руководитель.
	for _, p := range []struct{ id, pid, pv, name string }{
		{"DOC-PVA-FLANGE-1", FlangeProcessID, ProcessVersionID, FlangeProcessName},
		{"DOC-PVA-BRACKET-1", BracketProcessID, BracketVersionID, BracketProcess},
	} {
		add(&wdoc{ID: p.id, Template: dom.TemplateProcessApproval, Title: "Лист утверждения версии процесса «" + p.name + "» v1",
			Subject: platform.DrillRef{Entity: platform.EntityProcessVersion, ID: p.pv}, SubjectLabel: p.name + ", версия v1", ProcessID: p.pid, PVID: p.pv,
			Fields: []wfield{{"subject", "Версия процесса", p.name + " v1 (" + p.pv + ")"}, {"decision", "Что утверждается", "Ввод версии v1 в действие"},
				{"comment", "Читаемая разница", "Первая версия процесса: все узлы и точки контроля новые"}, {"requested_by", "Запросил", "TEC-01 (Технолог)"},
				{"requested_at", "Дата запроса", "14.09.2026 08:30"}, {"sources", "Основания", "ТП ФЛ-100.00.000 ТП, КД изм. Б"}},
			Versions: []wver{{At: at(14, 8, 30), Stages: []wstage{
				{Title: "Технолог-автор", Role: "technologist", Authority: "quorum_signature", Level: 2, Sig: sig("TEC-01", at(14, 8, 40), keyHW)},
				{Title: "Контролёр качества (кворум)", Role: "quality_inspector", Authority: "quorum_signature", Level: 2, Sig: sig("INS-01", at(14, 8, 50), keyBrowser)},
				{Title: "Руководитель производства (кворум)", Role: "production_manager", Authority: "quorum_signature", Level: 2, Sig: sig("PM-01", at(14, 9, 0), keyHW)},
			}}}})
	}

	// Входной контроль партий (ЗТ-1): акты и ярлык несоответствия партии П-117.
	for _, l := range []struct {
		id, lot, label, cert string
		day                  int
		qty                  string
	}{{"DOC-IIA-LOT-ZF-201", "LOT-ZF-201", "ЗФ-201", "С-201", 17, "42"}, {"DOC-IIA-LOT-R-117", "LOT-R-117", "П-117", "С-117", 22, "10"}} {
		add(flange(&wdoc{ID: l.id, Template: "incoming-inspection-act", Title: "Акт входного контроля партии " + l.label,
			Subject: platform.DrillRef{Entity: platform.EntityLot, ID: l.lot}, SubjectLabel: "Партия " + l.label,
			Fields: []wfield{{"subject", "Партия", "Партия " + l.label + ", " + l.qty + " шт."}, {"decision", "Заключение", "Принята, соответствует КД и сертификату"},
				{"comment", "Результаты контроля", "Внешний осмотр, размеры по выборке, сертификат проверен"}, {"sources", "Сертификат, протоколы", "Сертификат " + l.cert},
				{"requested_by", "Контролёр ВК", "INS-02"}, {"requested_at", "Дата", fmt.Sprintf("%02d.09.2026 10:55", l.day)}},
			Versions: []wver{{At: at(l.day, 10, 50), Stages: []wstage{
				{Title: "Контролёр входного контроля", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Paper: true, Sig: sig("INS-02", at(l.day, 10, 55), keyBrowser)},
				{Title: "Мастер склада — приёмка на хранение", Role: "site_foreman", Authority: "site_foreman", Level: 2, Who: []string{"FOR-SK"}, Sig: sig("FOR-SK", at(l.day, 11, 0), keyHW)},
			}}}}))
	}
	add(flange(&wdoc{ID: "DOC-LBL-LOT-R-117", Template: "lot-conformity-label", Title: "Ярлык несоответствия партии П-117",
		Subject: platform.DrillRef{Entity: platform.EntityLot, ID: "LOT-R-117"}, SubjectLabel: "Партия П-117",
		Fields: []wfield{{"subject", "Партия", "Партия П-117 (кольца К-101…К-110)"}, {"decision", "Ярлык", "Несоответствие — в изолятор, к возврату поставщику"},
			{"comment", "Основание", "НС-05: пора в теле кольца К-105 (РК-0923-11)"}, {"requested_by", "Контролёр", "INS-02"}, {"requested_at", "Дата", "23.09.2026 14:56"}},
		Versions: []wver{{At: at(23, 14, 55), Stages: []wstage{
			{Title: "Контролёр ОТК", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Sig: sig("INS-02", at(23, 14, 56), keyBrowser)},
		}}}}))

	// Заявления о несоответствии — по каждому подтверждённому НС.
	for _, n := range m.NCs {
		d := flange(&wdoc{ID: "DOC-" + n.ID, Template: dom.TemplateNCStatement, Title: "Заявление о несоответствии " + n.Number,
			Subject: platform.DrillRef{Entity: platform.EntityNonconformity, ID: n.ID}, SubjectLabel: n.Number})
		item, stage := "—", stepTitle[n.StepKey]
		for _, it := range n.Items {
			d.Items = append(d.Items, FullID(it.ID))
		}
		if len(n.Items) > 0 {
			var ls []string
			for _, it := range n.Items {
				ls = append(ls, it.Label)
			}
			item = strings.Join(ls, ", ")
		}
		found := n.SignalAt
		if found.IsZero() {
			found = n.ConfirmedAt
		}
		src := n.Spec.Source
		if src == "" {
			src = n.Spec.By
		}
		d.Fields = []wfield{{"nc.number", "Номер", n.Number}, {"item.item_id", "Изделие", item}, {"item.item_type_id", "Обозначение ДСЕ", flangeType},
			{"nc.found_at", "Обнаружено", dom.FormatTime(found)}, {"nc.found_by", "Кто обнаружил", src}, {"nc.stage", "Этап (операция)", orDash(stage)},
			{"nc.requirement", "Требование", "ТП ФЛ-100.00.000: шов W-1 без прожогов и пор, режим сварки 160 ± 10 А"}, {"nc.fact", "Факт", ncFact(n)},
			{"nc.severity", "Значимость", n.Spec.Severity}, {"nc.defect_type", "Вид дефекта", ncDefect(n)}, {"nc.evidence", "Доказательства", orDash(n.Spec.Report)},
			{"nc.confirmed_by", "Подтвердил", n.Spec.By}, {"nc.confirmed_at", "Дата подтверждения", dom.FormatTime(n.ConfirmedAt)}, {"nc.reason", "Основание", "Контроль по ТП"}}
		d.Versions = []wver{{At: found, Stages: []wstage{
			{Title: "Обнаруживший", Authority: "nc_detection", Level: 1, BySource: true, Who: []string{src}, Sig: &wsig{Person: src, At: found, Method: dom.MethodSource}},
			{Title: "Контролёр качества подтверждает", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Paper: true, BySource: true,
				Sig: &wsig{Person: n.Spec.By, At: n.ConfirmedAt, Method: dom.MethodTokenAgent, Storage: map[bool]string{true: keyHW, false: keyBrowser}[n.Spec.By == "INS-01"]}},
		}}}
		add(d)
	}

	disp := func(id, nc, number, title, decision string, items []string, fields []wfield, v wver) {
		// Поля разметки решения, которых у решения нет, — «—» (AD-12: разметка шаблона целиком).
		for _, f := range []wfield{{"item.item_id", "Изделие", dom.Empty}, {"nc.requirement", "Требование", "ТП ФЛ-100.00.000 / КД ФЛ-100.00.000 СБ"},
			{"nc.severity", "Значимость", "major"}, {"decision.scrap_kind", "Вид списания", dom.Empty}, {"decision.concession", "Разрешение на отклонение", dom.Empty},
			{"decision.claim", "Основание претензии", dom.Empty}} {
			if !slices.ContainsFunc(fields, func(x wfield) bool { return x.Key == f.Key }) {
				fields = append(fields, f)
			}
		}
		add(flange(&wdoc{ID: id, Template: dom.TemplateNCDisposition, Title: title, Subject: platform.DrillRef{Entity: platform.EntityNonconformity, ID: nc},
			SubjectLabel: number, Items: items, Fields: append([]wfield{{"nc.number", "Несоответствие", number}, {"decision.label", "Решение", decision}}, fields...),
			Versions: []wver{v}}))
	}
	f := func(ids ...string) []string {
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			out = append(out, FullID(id))
		}
		return out
	}
	// Решение по НС-01 «переделка» — на подписи; аннулировано групповым решением НС-И1.
	disp("DOC-DISP-NC-01", "NC-01", "НС-01", "Решение по несоответствующей продукции НС-01", "Переделка (исправимый брак)", f("F-017"),
		[]wfield{{"item.item_id", "Изделие", "Ф-017"}, {"nc.severity", "Значимость", "major"}, {"decision.reason", "Основание", "Прожог У2 исправим переваркой по ТП"},
			{"decision.author", "Автор решения", "INS-01"}, {"decision.at", "Дата решения", "23.09.2026 11:32"}},
		wver{At: at(23, 11, 30), AnnulledAt: at(23, 15, 30), AnnulledBy: "HQC-01", AnnulReason: "Заменено групповым решением НС-И1 (переделка ×6)", Stages: []wstage{
			{Title: "Автор решения (контролёр качества)", Authority: "nc_disposition", Level: 2, Who: []string{"INS-01"}, Sig: sig("INS-01", at(23, 11, 32), keyHW)},
			{Title: "Технолог", Role: "technologist", Authority: "nc_disposition", Level: 2, Paper: true},
			{Title: "Начальник ОТК", Role: "head_of_qc", Authority: "nc_disposition", Level: 2, Paper: true},
		}})
	// Групповое решение комиссии НС-И1: переделка ×6, режим 5 — с представителем заказчика.
	disp("DOC-DISP-NC-G1", "NC-G1", "НС-И1", "Решение комиссии по НС-И1: переделка шести фланцев", "Переделка ×6", f("F-015", "F-017", "F-019", "F-021", "F-023", "F-025"),
		[]wfield{{"item.item_id", "Изделие", "Ф-015, Ф-017, Ф-019, Ф-021, Ф-023, Ф-025"}, {"nc.severity", "Значимость", "major"},
			{"decision.reason", "Основание", "Сварка вне режима ТП на ИС-2 (дрейф регулятора тока); область риска RS-01, версия 4"},
			{"decision.author", "Автор решения", "TEC-01"}, {"decision.at", "Дата решения", "23.09.2026 15:30"}},
		wver{At: at(23, 15, 5), Stages: []wstage{
			{Title: "Автор решения (технолог)", Authority: "nc_disposition", Level: 2, Who: []string{"TEC-01"}, Sig: sig("TEC-01", at(23, 15, 8), keyHW)},
			{Title: "Технолог (главный сварщик)", Role: "technologist", Authority: "nc_disposition", Level: 2, Paper: true, Sig: sig("CWL-01", at(23, 15, 15), keyBrowser)},
			{Title: "Начальник ОТК", Role: "head_of_qc", Authority: "nc_disposition", Level: 2, Paper: true, Sig: sig("HQC-01", at(23, 15, 22), keyHW)},
			{Title: "Представитель заказчика", Role: "customer_representative", Authority: "customer_acceptance", Level: 2, Paper: true, Sig: sig("CR-71", at(23, 15, 30), keyHW)},
		}})
	disp("DOC-DISP-NC-05", "NC-05", "НС-05", "Решение по несоответствующей продукции НС-05: возврат поставщику", "Возврат поставщику", nil,
		[]wfield{{"item.item_id", "Изделие", "Кольцо К-105, партия П-117"}, {"decision.claim", "Основание претензии", "Сертификат С-117, заключение РК-0923-11"},
			{"decision.reason", "Основание", "Входной брак партии П-117 (пора в теле кольца)"}, {"decision.author", "Автор решения", "TEC-01"}, {"decision.at", "Дата решения", "23.09.2026 15:45"}},
		wver{At: at(23, 15, 40), Stages: []wstage{
			{Title: "Автор решения (технолог)", Authority: "nc_disposition", Level: 2, Who: []string{"TEC-01"}, Sig: sig("TEC-01", at(23, 15, 42), keyHW)},
			{Title: "Начальник ОТК", Role: "head_of_qc", Authority: "nc_disposition", Level: 2, Paper: true, Sig: sig("HQC-01", at(23, 15, 45), keyHW)},
		}})
	disp("DOC-DISP-NC-04", "NC-04", "НС-04", "Решение по несоответствующей продукции НС-04: замена кольца", "Переделка: замена кольца К-101", f("F-019"),
		[]wfield{{"item.item_id", "Изделие", "Ф-019"}, {"decision.reason", "Основание", "Отклонение не согласовано держателем КД; кольцо К-101 снять, поставить новое"},
			{"decision.author", "Автор решения", "TEC-01"}, {"decision.at", "Дата решения", "23.09.2026 16:02"}},
		wver{At: at(23, 16, 0), Stages: []wstage{
			{Title: "Автор решения (технолог)", Authority: "nc_disposition", Level: 2, Who: []string{"TEC-01"}, Sig: sig("TEC-01", at(23, 16, 2), keyHW)},
			{Title: "Начальник ОТК", Role: "head_of_qc", Authority: "nc_disposition", Level: 2, Paper: true, Sig: sig("HQC-01", at(23, 16, 10), keyHW)},
		}})
	disp("DOC-DISP-NC-06", "NC-06", "НС-06", "Решение по несоответствующей продукции НС-06: окончательный брак", "Окончательный брак (списание)", f("F-021"),
		[]wfield{{"item.item_id", "Изделие", "Ф-021"}, {"decision.scrap_kind", "Вид списания", "Окончательный брак"},
			{"decision.reason", "Основание", "Новый дефект в новом шве после переварки; повторная переварка ТП не допускается"},
			{"decision.author", "Автор решения", "CWL-01"}, {"decision.at", "Дата решения", "24.09.2026 15:30"}},
		wver{At: at(24, 14, 50), Stages: []wstage{
			{Title: "Автор решения (главный сварщик)", Authority: "nc_disposition", Level: 2, Who: []string{"CWL-01"}, Sig: sig("CWL-01", at(24, 14, 55), keyBrowser)},
			{Title: "Начальник ОТК", Role: "head_of_qc", Authority: "nc_disposition", Level: 2, Paper: true, Sig: sig("HQC-01", at(24, 15, 30), keyHW)},
		}})

	// Разрешение на отклонение по Ф-019 — держатель КД вернул с замечанием.
	add(flange(&wdoc{ID: "DOC-CONC-NC-04", Template: "concession", Title: "Разрешение на отклонение: пора в теле кольца К-101 (Ф-019)",
		Subject: platform.DrillRef{Entity: platform.EntityNonconformity, ID: "NC-04"}, SubjectLabel: "НС-04", Items: f("F-019"),
		Fields: []wfield{{"item.item_id", "Изделие", "Ф-019 (кольцо К-101)"}, {"item.item_type_id", "Обозначение ДСЕ", flangeType}, {"data.nc_id", "Несоответствие", "НС-04"},
			{"data.disposition", "Решение", "«Как есть»: одиночная пора Ø0,4 мм в теле кольца при требовании КД ФЛ-100.01.002 «без пор»"},
			{"data.concession_id", "Номер разрешения", "РО-0923-01"}, {"data.reason", "Основание", "Пора вне зоны шва, прочность по расчёту обеспечена; на одно изделие Ф-019"},
			{"event.actor", "Инициатор", "TEC-01"}, {"event.at", "Дата", "23.09.2026 14:00"}},
		Versions: []wver{{At: at(23, 14, 0), Declines: []wdecline{{Stage: 2, Person: "DA-81", At: at(23, 14, 30),
			Comment: "Пора в теле кольца вне допуска КД ФЛ-100.01.002; отклонение не согласовано — кольцо заменить"}}, Stages: []wstage{
			{Title: "Технолог — инициатор", Role: "technologist", Authority: "nc_disposition", Level: 2, Sig: sig("TEC-01", at(23, 14, 2), keyHW)},
			{Title: "Держатель КД", Role: "design_authority", Authority: "design_authority", Level: 2, Paper: true},
			{Title: "Представитель заказчика", Role: "customer_representative", Authority: "customer_acceptance", Level: 2, Paper: true},
		}}}}))

	// Запрос решения по Ф-015 «годно при неполных данных» — начальник ОТК вернул.
	add(flange(&wdoc{ID: "DOC-REQ-F-015", Template: dom.TemplateDecisionRequest, Title: "Запрос решения: допуск Ф-015 к сборке при неполных данных ИС-2",
		Subject: platform.DrillRef{Entity: platform.EntityItem, ID: FullID("F-015")}, SubjectLabel: "Ф-015", Items: f("F-015"),
		Fields: []wfield{{"subject", "Объект", "Ф-015, ЗТ-3"}, {"decision", "Что решается", "Допустить к сборке при неполном журнале ИС-2"},
			{"comment", "Комментарий", "Опоздавший журнал ИС-2: 35 записей потеряно в окне 16:40–17:15"}, {"requested_by", "Запросил", "INS-01"},
			{"requested_at", "Дата запроса", "23.09.2026 12:10"}, {"sources", "Основания", "EV-WS2-0412, пересмотр ЗТ-3"}},
		Versions: []wver{{At: at(23, 12, 10), Declines: []wdecline{{Stage: 1, Person: "HQC-01", At: at(23, 12, 40),
			Comment: "Журнал ИС-2 неполон — допуск не согласован; пересмотреть приёмку ЗТ-3 и вскрыть изделие"}}, Stages: []wstage{
			{Title: "Начальник ОТК", Role: "head_of_qc", Authority: "nc_disposition", Level: 2, Paper: true},
		}}}}))

	// Акты о браке: групповой (переделка ×6) закрыт; Ф-021 — напечатан, ждёт подписи ручкой.
	add(flange(&wdoc{ID: "DOC-ACT-NC-G1", Template: "scrap-act", Title: "Акт о браке (исправимый) на 6 изделий, НС-И1",
		Subject: platform.DrillRef{Entity: platform.EntityNonconformity, ID: "NC-G1"}, SubjectLabel: "НС-И1", Items: f("F-015", "F-017", "F-019", "F-021", "F-023", "F-025"),
		Fields: []wfield{{"item.item_id", "Изделие", "Ф-015, Ф-017, Ф-019, Ф-021, Ф-023, Ф-025"}, {"item.item_type_id", "Обозначение ДСЕ", flangeType},
			{"item.order_id", "Задание", "ЗП-0917"}, {"item.lots", "Партии", "ЗФ-201, П-116, П-117"}, {"data.nc_id", "Несоответствие", "НС-И1"},
			{"data.disposition", "Решение", "Переделка ×6"}, {"data.scrap_kind", "Вид брака", "Исправимый — переделка"}, {"data.claim_basis", "Основание претензии", "Нет: причина — оборудование ИС-2 (ошибок сварщиков — 0)"},
			{"data.reason", "Основание", "Сварка вне режима ТП: дрейф регулятора тока ИС-2; 1С — перевод в брак (переделка) OUT-000127"},
			{"event.actor", "Решение принял", "TEC-01 (комиссия)"}, {"event.at", "Дата решения", "23.09.2026 15:30"}},
		Versions: []wver{{At: at(23, 15, 31), Stages: []wstage{
			{Title: "Контролёр ОТК", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Paper: true, Sig: sig("INS-01", at(23, 15, 33), keyHW)},
			{Title: "Начальник ОТК", Role: "head_of_qc", Authority: "nc_disposition", Level: 2, Paper: true, Sig: sig("HQC-01", at(23, 15, 40), keyHW)},
			{Title: "Руководитель производства", Role: "production_manager", Authority: "quorum_signature", Level: 2, Paper: true, Sig: sig("PM-01", at(23, 15, 50), keyHW)},
		}}}}))
	add(flange(&wdoc{ID: "DOC-ACT-NC-06", Template: "scrap-act", Title: "Акт о браке Ф-021 (окончательный), НС-06",
		Subject: platform.DrillRef{Entity: platform.EntityNonconformity, ID: "NC-06"}, SubjectLabel: "НС-06", Items: f("F-021"),
		Fields: []wfield{{"item.item_id", "Изделие", "Ф-021"}, {"item.item_type_id", "Обозначение ДСЕ", flangeType}, {"item.order_id", "Задание", "ЗП-0917"},
			{"item.lots", "Партии", "ЗФ-201, П-116"}, {"data.nc_id", "Несоответствие", "НС-06"}, {"data.disposition", "Решение", "Списать"},
			{"data.scrap_kind", "Вид брака", "Окончательный — списание"}, {"data.claim_basis", "Основание претензии", "Нет"},
			{"data.reason", "Основание", "Пористость нового шва после переварки; 1С — перевод в брак (списание) OUT-000135"},
			{"event.actor", "Решение принял", "CWL-01"}, {"event.at", "Дата решения", "24.09.2026 15:30"}},
		Versions: []wver{{At: at(24, 15, 35), Paper: []wpaper{{Status: "printed", At: at(24, 16, 0), CopyNo: "1"}}, Stages: []wstage{
			{Title: "Контролёр ОТК", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Paper: true, Sig: sig("INS-01", at(24, 15, 40), keyBrowser)},
			{Title: "Начальник ОТК", Role: "head_of_qc", Authority: "nc_disposition", Level: 2, Paper: true},
			{Title: "Руководитель производства", Role: "production_manager", Authority: "quorum_signature", Level: 2, Paper: true},
		}}}}))

	// Журнал изолятора, запись вмешательства, маршрутный лист.
	add(flange(&wdoc{ID: "DOC-ISO-NC-01", Template: "isolator-log", Title: "Журнал изолятора: Ф-017 помещён (НС-01)",
		Subject: platform.DrillRef{Entity: platform.EntityNonconformity, ID: "NC-01"}, SubjectLabel: "НС-01", Items: f("F-017"),
		Fields: []wfield{{"item.item_id", "Изделие", "Ф-017"}, {"item.item_type_id", "Обозначение ДСЕ", flangeType}, {"data.isolator_location_id", "Место", "Изолятор сварочного цеха, ячейка 3"},
			{"data.reason", "Основание", "НС-01: прожог У2"}, {"data.decision_due_at", "Решение до", "23.09.2026 17:00"}, {"event.at", "Дата", "23.09.2026 11:20"}, {"event.actor", "Кто", "INS-01"}},
		Table: []wrow{{at(23, 11, 20), trow(1, at(23, 11, 20), "decision.item.isolated", "КТ-3 камера", "INS-01", "Помещён в изолятор, ячейка 3")},
			{at(23, 15, 30), trow(2, at(23, 15, 30), "decision.containment.released", "ЗТ-3", "HQC-01", "Выдан на переделку по решению НС-И1")}},
		Versions: []wver{{At: at(23, 11, 20), Stages: []wstage{
			{Title: "Контролёр ОТК", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Sig: sig("INS-01", at(23, 11, 21), keyHW)},
		}}}}))
	add(flange(&wdoc{ID: "DOC-INT-F-015", Template: "intervention-record", Title: "Запись вмешательства: вскрытие Ф-015",
		Subject: platform.DrillRef{Entity: platform.EntityItem, ID: FullID("F-015")}, SubjectLabel: "Ф-015", Items: f("F-015"),
		Fields: []wfield{{"item.item_id", "Изделие", "Ф-015"}, {"item.item_type_id", "Обозначение ДСЕ", flangeType}, {"data.intervention_id", "Вмешательство", "INT-015-1"},
			{"data.zone_ids", "Зоны", "S-1 канавка уплотнения"}, {"data.purpose", "Цель", "Возврат в сварку по пересмотру ЗТ-3 (журнал ИС-2 неполон)"},
			{"data.removed_components", "Снятые компоненты", "Уплотнение УП-401; крышку и крепёж не ставили"}, {"event.at", "Дата", "23.09.2026 15:50"}, {"event.actor", "Кто", "FOR-AC"}},
		Table: []wrow{{at(23, 15, 50), trow(1, at(23, 15, 50), "item.intervention.opened", "Сборка", "FOR-AC", "Уплотнение снято")},
			{at(23, 16, 0), trow(2, at(23, 16, 0), "item.intervention.closed", "Сборка", "INS-02", "Зона S-1 осмотрена, повторная проверка — после переварки")}},
		Versions: []wver{{At: at(23, 15, 50), Stages: []wstage{
			{Title: "Мастер участка — вскрытие", Role: "site_foreman", Authority: "site_foreman", Level: 2, Who: []string{"FOR-AC"}, Sig: sig("FOR-AC", at(23, 15, 51), keyHW)},
			{Title: "Контролёр ОТК — осмотр зоны", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Sig: sig("INS-02", at(23, 16, 0), keyBrowser)},
		}}}}))
	add(flange(&wdoc{ID: "DOC-RS-F-015", Template: "route-sheet", Title: "Маршрутный лист: Ф-015 из сборочного цеха в сварочный",
		Subject: platform.DrillRef{Entity: platform.EntityItem, ID: FullID("F-015")}, SubjectLabel: "Ф-015", Items: f("F-015"),
		Fields: []wfield{{"item.item_id", "Изделие", "Ф-015"}, {"item.item_type_id", "Обозначение ДСЕ", flangeType}, {"data.from_location_id", "Откуда", "Сборочно-испытательный цех (WS-AC)"},
			{"data.to_location_id", "Куда", "Сварочный цех (WS-WC)"}, {"data.container_id", "Тара", "Тележка Т-4"}, {"event.at", "Дата", "23.09.2026 16:00"}, {"event.actor", "Кто", "FOR-AC"}},
		Table: []wrow{{at(23, 16, 0), trow(1, at(23, 16, 0), "operation.movement.sent", "Передача в сварку", "FOR-AC", "WS-AC → WS-WC; 1С OUT-000129")},
			{at(23, 16, 10), trow(2, at(23, 16, 10), "operation.movement.received", "Передача в сварку", "FOR-WC", "Принят сварочным цехом")}},
		Versions: []wver{{At: at(23, 16, 0), Stages: []wstage{
			{Title: "Мастер — передал", Role: "site_foreman", Authority: "site_foreman", Level: 2, Who: []string{"FOR-AC"}, Sig: sig("FOR-AC", at(23, 16, 0), keyHW)},
			{Title: "Мастер — принял", Role: "site_foreman", Authority: "site_foreman", Level: 2, Who: []string{"FOR-WC"}, Sig: sig("FOR-WC", at(23, 16, 10), keyHW)},
		}}}}))

	// Выпуск эталона Ф-001: извещение ВП, свидетельство о приёмке (ВП — на бумаге), карта, паспорт.
	f001 := func(id, tpl, title string, fields []wfield, v wver) {
		add(flange(&wdoc{ID: id, Template: tpl, Title: title, Subject: platform.DrillRef{Entity: platform.EntityItem, ID: FullID("F-001")}, SubjectLabel: "Ф-001",
			Items: f("F-001"), Fields: fields, Versions: []wver{v}}))
	}
	f001("DOC-PRES-F-001", "customer-presentation-notice", "Извещение о предъявлении ВП: Ф-001",
		[]wfield{{"item.item_id", "Изделие", "Ф-001"}, {"item.item_type_id", "Обозначение ДСЕ", flangeType}, {"item.order_id", "Задание", "ЗП-0917"}, {"item.lots", "Партии", "ЗФ-201, П-116"},
			{"data.step_key", "Точка приёмки", "ЗТ-6 — приёмка ВП, приёмо-сдаточные испытания пройдены"}, {"data.presentation_no", "Предъявление №", "1"},
			{"data.presented_by", "Предъявил", "FOR-AC"}, {"event.at", "Дата предъявления", "23.09.2026 09:42"}},
		wver{At: at(23, 9, 40), Stages: []wstage{
			{Title: "Мастер — предъявил", Role: "site_foreman", Authority: "site_foreman", Level: 2, Who: []string{"FOR-AC"}, Sig: sig("FOR-AC", at(23, 9, 42), keyHW)},
			{Title: "ОТК — принято", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Sig: sig("INS-02", at(23, 9, 50), keyBrowser)},
			{Title: "Представитель заказчика — заключение", Role: "customer_representative", Authority: "customer_acceptance", Level: 2, Paper: true, Sig: sig("CR-71", at(23, 10, 10), keyHW)},
		}})
	f001("DOC-CERT-F-001", "acceptance-certificate", "Свидетельство о приёмке Ф-001",
		[]wfield{{"item.item_id", "Изделие", "Ф-001 — изготовлен и принят в соответствии с КД и ТУ, годен к эксплуатации"}, {"item.item_type_id", "Обозначение ДСЕ", flangeType},
			{"item.order_id", "Задание", "ЗП-0917"}, {"item.lots", "Партии", "ЗФ-201, П-116, КР-301, УП-401, КП-501"}, {"data.warehouse_id", "Склад", "Склад готовой продукции СГП-1"},
			{"data.received_by", "Принял на склад", "STK-51"}, {"data.after_rework", "После переделки", "нет"}, {"data.concession_id", "Разрешение на отклонение", "нет"},
			{"event.at", "Дата выпуска", "23.09.2026 10:30"}},
		wver{At: at(23, 10, 15), Paper: []wpaper{{Status: "printed", At: at(23, 10, 18), CopyNo: "1"}, {Status: "signed", At: at(23, 10, 25), CopyNo: "1"}}, Stages: []wstage{
			{Title: "Контролёр ОТК", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Paper: true, Sig: sig("INS-02", at(23, 10, 17), keyBrowser)},
			{Title: "Представитель заказчика", Role: "customer_representative", Authority: "customer_acceptance", Level: 2, Paper: true,
				Sig: &wsig{Person: "CR-71", At: at(23, 10, 25), Method: dom.MethodPaper, Attester: "HQC-01", PaperNo: "ОТК-А-0923-001"}},
		}})
	f001("DOC-PASS-F-001", "passport", "Паспорт изделия Ф-001",
		[]wfield{{"item.item_id", "Изделие", "Ф-001"}, {"item.item_type_id", "Обозначение ДСЕ", "ФЛ-100.00.000 СБ, изм. Б"}, {"item.order_id", "Задание", "ЗП-0917"},
			{"item.lots", "Партии", "ЗФ-201, П-116, КР-301, УП-401, КП-501"}, {"data.warehouse_id", "Склад", "СГП-1; 1С «Выпуск годного» № 0000-000120"},
			{"data.after_rework", "После переделки", "нет"}, {"event.at", "Дата выпуска", "23.09.2026 10:30"}},
		wver{At: at(23, 10, 26), Stages: []wstage{
			{Title: "Начальник ОТК", Role: "head_of_qc", Authority: "qc_acceptance", Level: 2, Paper: true, Sig: sig("HQC-01", at(23, 10, 28), keyHW)},
			{Title: "Представитель заказчика", Role: "customer_representative", Authority: "customer_acceptance", Level: 2, Paper: true, Sig: sig("CR-71", at(23, 10, 30), keyHW)},
		}})

	// Ф-017 после переделки предъявлен ВП повторно — ждёт заключения представителя заказчика.
	add(flange(&wdoc{ID: "DOC-PRES-F-017", Template: "customer-presentation-notice", Title: "Извещение о предъявлении ВП: Ф-017 после переделки",
		Subject: platform.DrillRef{Entity: platform.EntityItem, ID: FullID("F-017")}, SubjectLabel: "Ф-017", Items: f("F-017"),
		Fields: []wfield{{"item.item_id", "Изделие", "Ф-017"}, {"item.item_type_id", "Обозначение ДСЕ", flangeType}, {"item.order_id", "Задание", "ЗП-0917"}, {"item.lots", "Партии", "ЗФ-201, П-116"},
			{"data.step_key", "Точка приёмки", "ЗТ-3 повторно — шов W-1 после переварки по решению НС-И1"}, {"data.presentation_no", "Предъявление №", "2"},
			{"data.presented_by", "Предъявил", "FOR-WC"}, {"event.at", "Дата предъявления", "24.09.2026 16:10"}},
		Versions: []wver{{At: at(24, 16, 5), Stages: []wstage{
			{Title: "Мастер — предъявил", Role: "site_foreman", Authority: "site_foreman", Level: 2, Who: []string{"FOR-WC"}, Sig: sig("FOR-WC", at(24, 16, 10), keyHW)},
			{Title: "ОТК — принято", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Sig: sig("INS-01", at(24, 16, 20), keyHW)},
			{Title: "Представитель заказчика — заключение", Role: "customer_representative", Authority: "customer_acceptance", Level: 2, Paper: true},
		}}}}))

	// Редкие подписанты (FR-136): у держателя КД, метролога и представителя
	// заказчика на начальном шаге есть запрос решения — стол «Требуется ваше
	// решение» не пустой; к концу истории запросы подписаны.
	req := func(id, title string, subject platform.DrillRef, label string, items []string, fields []wfield, v wver) {
		add(flange(&wdoc{ID: id, Template: dom.TemplateDecisionRequest, Title: title, Subject: subject, SubjectLabel: label, Items: items, Fields: fields, Versions: []wver{v}}))
	}
	req("DOC-REQ-DA-F-023", "Запрос решения: переварка корня шва W-1 на Ф-023 — согласование держателя КД",
		platform.DrillRef{Entity: platform.EntityItem, ID: FullID("F-023")}, "Ф-023", f("F-023"),
		[]wfield{{"subject", "Объект", "Ф-023, шов W-1, участок У3 (прожог в корне, РК-0923-10)"},
			{"decision", "Что решается", "Допустима ли переварка корня шва W-1 с зачисткой до основного металла вместо списания"},
			{"comment", "Комментарий", "Глубина зачистки 1,8 мм при толщине стенки 6 мм; требование КД к шву сохраняется, повторный рентген обязателен"},
			{"requested_by", "Запросил", "TEC-01"}, {"requested_at", "Дата запроса", "23.09.2026 13:35"}, {"sources", "Основания", "РК-0923-10, НС-03, ТП ФЛ-100.00.000"}},
		wver{At: at(23, 13, 35), Stages: []wstage{
			{Title: "Согласующий — держатель КД", Role: "design_authority", Authority: "concession_approval", Level: 2, Paper: true, Sig: sig("DA-81", at(24, 9, 40), keyBrowser)},
		}})
	req("DOC-REQ-MET-KT3", "Запрос решения: допуск камеры КТ-3 после перекалибровки",
		platform.DrillRef{Entity: platform.EntityEquipment, ID: "CAM-KT3"}, "Камера КТ-3", f("F-025"),
		[]wfield{{"subject", "Объект", "Камера КТ-3 сварочного участка (анализатор VisionQC)"},
			{"decision", "Что решается", "Допуск камеры к работе после перекалибровки освещения"},
			{"comment", "Комментарий", "Блик на Ф-025: «признаков нет, 0,91» при качестве кадра 0,34; после перекалибровки — контрольный набор 40 кадров, качество ≥ 0,8"},
			{"requested_by", "Запросил", "HQC-01"}, {"requested_at", "Дата запроса", "23.09.2026 13:10"}, {"sources", "Основания", "Кадры Ф-025, протокол перекалибровки ПК-0923-02"}},
		wver{At: at(23, 13, 10), Stages: []wstage{
			{Title: "Метролог — согласование", Role: "metrologist", Authority: "analyzer_admission", Level: 2, Sig: sig("MET-82", at(24, 11, 0), keyBrowser)},
			{Title: "Начальник ОТК — допуск", Role: "head_of_qc", Authority: "analyzer_admission", Level: 2, Sig: sig("HQC-01", at(24, 11, 20), keyHW)},
		}})
	req("DOC-REQ-CR-RS-01", "Запрос решения: продолжение приёмки ВП по заданию ЗП-0917 при открытом инциденте RS-01",
		platform.DrillRef{Entity: platform.EntityIncident, ID: "RS-01"}, "Инцидент ИС-2 (RS-01)", nil,
		[]wfield{{"subject", "Объект", "Задание ЗП-0917, инцидент RS-01"},
			{"decision", "Что решается", "Продолжать приёмку ВП изделий вне области риска RS-01"},
			{"comment", "Комментарий", "Область риска сужена до 6 изделий; остальные сварены на ИС-1 или до отказа регулятора ИС-2 и придержаны не будут"},
			{"requested_by", "Запросил", "HQC-01"}, {"requested_at", "Дата запроса", "23.09.2026 13:20"}, {"sources", "Основания", "Область риска RS-01, версия 3"}},
		wver{At: at(23, 13, 20), Stages: []wstage{
			{Title: "Начальник ОТК", Role: "head_of_qc", Authority: "nc_disposition", Level: 2, Paper: true, Sig: sig("HQC-01", at(23, 13, 25), keyHW)},
			{Title: "Представитель заказчика", Role: "customer_representative", Authority: "customer_acceptance", Level: 2, Paper: true, Sig: sig("CR-71", at(23, 15, 32), keyHW)},
		}})

	// Сопроводительные карты: Ф-001 зафиксирована и подписана, изделия истории — собираются.
	for _, id := range []string{"F-001", "F-015", "F-017", "F-019", "F-021", "F-023", "F-025"} {
		it := m.itemByID[id]
		if it == nil {
			continue
		}
		d := flange(&wdoc{ID: dom.TravelerID(FullID(id)), Template: dom.TemplateTraveler, Title: "Сопроводительная карта изделия " + it.Label,
			Subject: platform.DrillRef{Entity: platform.EntityItem, ID: FullID(id)}, SubjectLabel: it.Label, Items: []string{FullID(id)},
			Fields: []wfield{{"item.item_id", "Изделие", it.Label}, {"item.item_type_id", "Обозначение ДСЕ", it.ItemType}, {"item.item_revision", "Изменение КД", "Б"},
				{"item.order_id", "Задание", it.Order}, {"item.lots", "Партии", lotOfItem(it)}, {"item.process_version_hash", "Версия техпроцесса", ProcessVersionID},
				{"origin", "Происхождение", "Собрано из журнала системы «Главный»; вручную не заполнялось"}},
			Rows: func(c *Ctx) map[string][]map[string]any { return c.travelerRows(it) }})
		if id == "F-001" {
			d.Fields = append(d.Fields, wfield{"final.status", "Итоговая годность", "Годен"}, wfield{"final.at", "Дата", "23.09.2026 10:27"})
			d.Versions = []wver{{At: at(23, 10, 26), Stages: []wstage{
				{Title: "Контролёр ОТК — итоговая годность", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Paper: true, Sig: sig("INS-02", at(23, 10, 27), keyBrowser)},
				{Title: "Мастер участка — итоговая годность", Role: "site_foreman", Authority: "site_foreman", Level: 2, Paper: true, Who: []string{"FOR-AC"}, Sig: sig("FOR-AC", at(23, 10, 28), keyHW)},
			}}}
		} else {
			d.Fields = append(d.Fields, wfield{"final.status", "Итоговая годность", "в работе"}, wfield{"final.at", "Дата", dom.Empty})
			d.Live, d.LiveFrom = true, it.Launch
			d.Versions = []wver{{At: it.Launch, Stages: []wstage{
				{Title: "Контролёр ОТК — итоговая годность", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Paper: true},
				{Title: "Мастер участка — итоговая годность", Role: "site_foreman", Authority: "site_foreman", Level: 2, Paper: true},
			}}}
		}
		add(d)
	}

	// Смена: журнал предъявления ОТК поста сварочного цеха и назначения контролёров.
	add(flange(&wdoc{ID: "DOC-PLOG-WP-QC-WC-0923", Template: "presentation-log", Title: "Журнал предъявления ОТК: пост ОТК сварочного цеха, смена 23.09",
		Subject: platform.DrillRef{Entity: platform.EntityWorkplace, ID: "WP-QC-WC"}, SubjectLabel: "Пост ОТК сварочного цеха, смена 1 · 23.09",
		Fields: []wfield{{"log.post", "Пост", "WP-QC-WC"}, {"log.shift", "Смена", "1-я, 23.09.2026"}, {"log.inspector", "Контролёр", "INS-01"},
			{"log.presented", "Предъявлено", "14 изделий"}, {"log.accepted", "Принято", "8"}, {"log.rejected", "Не принято", "6 (НС-01…НС-03, НС-И1)"}},
		Versions: []wver{{At: at(23, 8, 0), Stages: []wstage{
			{Title: "Контролёр ОТК — закрытие смены", Role: "quality_inspector", Authority: "qc_acceptance", Level: 2, Who: []string{"INS-01"}, Sig: sig("INS-01", at(23, 17, 0), keyHW)},
		}}}}))
	assign := func(id, wp, label, who, shift, foreman string, drafted time.Time, signed *wsig) {
		st := wstage{Title: "Начальник ОТК — согласование", Role: "head_of_qc", Authority: "controller_assignment_approval", Level: 2, Sig: signed}
		add(&wdoc{ID: id, Template: "controller-assignment", Title: "Назначение контролёра " + who + " на пост: " + label,
			Subject: platform.DrillRef{Entity: platform.EntityWorkplace, ID: wp}, SubjectLabel: label,
			Fields: []wfield{{"subject", "Пост", label + " (" + wp + ")"}, {"decision", "Кто и в какую смену", "quality_inspector:" + who + "@" + shift},
				{"comment", "Комментарий", "Контроль сварных швов на смену"}, {"requested_by", "Запросил мастер", foreman}, {"requested_at", "Дата запроса", dom.FormatTime(drafted)}},
			Versions: []wver{{At: drafted, Stages: []wstage{st}}}})
	}
	assign("DOC-CA-WP-QC-WC-0921", "WP-QC-WC", "Пост ОТК сварочного цеха", "INS-01", "SHIFT-1", "FOR-WC", at(21, 7, 30), sig("HQC-01", at(21, 7, 40), keyHW))
	assign("DOC-CA-WP-QC-AC-0924", "WP-QC-AC", "Пост ОТК сборочно-испытательного цеха", "INS-01", "SHIFT-2", "FOR-AC", at(24, 12, 0), nil)

	m.docs = out
	return out
}

func orDash(s string) string {
	if s == "" {
		return dom.Empty
	}
	return s
}

func ncFact(n *NC) string {
	var parts []string
	for _, d := range n.Spec.Defects {
		parts = append(parts, d.Kind+" в зоне "+d.Zone)
	}
	if len(parts) == 0 {
		return orDash(n.Spec.Kind)
	}
	return strings.Join(parts, "; ")
}

func ncDefect(n *NC) string {
	if len(n.Spec.Defects) > 0 {
		return n.Spec.Defects[0].Kind
	}
	return orDash(n.Spec.Kind)
}

// travelerRows — строки сопроводительной карты на момент: операции изделия из
// журнала (последние 20) и его несоответствия.
func (c *Ctx) travelerRows(it *Item) map[string][]map[string]any {
	rows := []map[string]any{}
	for _, e := range c.itemEvents(it) {
		if e.StepKey == "" || (e.Kind != "fact" && e.Kind != "decision") {
			continue
		}
		op := stepTitle[e.StepKey]
		if op == "" {
			op = e.StepKey
		}
		who := e.Author
		if who == "" {
			who = e.Source
		}
		otk := ""
		if e.Kind == "decision" {
			otk = "принято"
		}
		rows = append(rows, map[string]any{"operation": op, "executor": who, "date": dom.FormatTime(e.Occurred), "signature": sigLevel(e), "params": e.Summary,
			"otk": otk, "remarks": ""})
	}
	if len(rows) > 20 {
		rows = rows[len(rows)-20:]
	}
	for i, r := range rows {
		r["no"] = strconv.Itoa(i + 1)
	}
	ncs := []map[string]any{}
	for _, n := range c.M.NCs {
		if slices.Contains(n.Items, it) && !n.ConfirmedAt.IsZero() && !n.ConfirmedAt.After(c.T) {
			disp := ""
			if d := n.DispositionAt(c.M); d != nil && !d.After(c.T) {
				disp = "решение принято"
			}
			ncs = append(ncs, map[string]any{"number": n.Number, "status": n.Status(c.M, c.T), "disposition": disp, "concession": ""})
		}
	}
	return map[string][]map[string]any{"rows": rows, "nonconformities": ncs}
}

func sigLevel(e *Event) string {
	if e.Kind == "decision" {
		return "уровень 2"
	}
	return "уровень 0"
}

// ─────────────────────────────── состояние на момент ───────────────────────────────

func (s wstage) done(t time.Time) bool { return s.Sig != nil && !s.Sig.At.After(t) }

func (v wver) declined(t time.Time) []wdecline {
	var out []wdecline
	for _, d := range v.Declines {
		if !d.At.After(t) {
			out = append(out, d)
		}
	}
	return out
}

func (v wver) closedAt(t time.Time) (time.Time, bool) {
	var last time.Time
	for _, s := range v.Stages {
		if !s.done(t) {
			return time.Time{}, false
		}
		if s.Sig.At.After(last) {
			last = s.Sig.At
		}
	}
	return last, len(v.Stages) > 0
}

func (v wver) status(t time.Time) string {
	if !v.AnnulledAt.IsZero() && !v.AnnulledAt.After(t) {
		return dom.StatusAnnulled
	}
	if _, ok := v.closedAt(t); ok {
		return dom.StatusRouteClosed
	}
	if len(v.declined(t)) > 0 {
		return dom.StatusReturned
	}
	for _, s := range v.Stages {
		if s.done(t) {
			return dom.StatusSigning
		}
	}
	return dom.StatusDrafted
}

func (v wver) paperStatus(t time.Time) string {
	st := ""
	for _, p := range v.Paper {
		if !p.At.After(t) {
			st = p.Status
		}
	}
	return st
}

// visible — версии документа, известные к моменту t.
func (d *wdoc) visible(t time.Time) []wver {
	var out []wver
	for _, v := range d.Versions {
		if !v.At.After(t) {
			out = append(out, v)
		}
	}
	return out
}

// ─────────────────────────────── рендер ───────────────────────────────

// docTemplate — шаблон нормативного слоя; объявленный без отрисовки — с
// отрисовкой по полям документа мира (поля и маршрут).
func (m *Model) docTemplate(d *wdoc) dom.Template {
	t, ok := m.templates.ByRef(d.Template)
	if !ok {
		t = dom.Template{ID: d.Template, Title: d.Title, Class: dom.ClassRecord, Version: 1}
	}
	if t.Complete() && d.covers(t) {
		return t
	}
	fb := t
	fb.DocType = dom.DocGeneric
	fields := make([]dom.Field, 0, len(d.Fields))
	for _, f := range d.Fields {
		fields = append(fields, dom.Field{Key: f.Key, Label: f.Label})
	}
	fb.Layout = []dom.Section{{Kind: "fields", Title: t.Title, Fields: fields}, {Kind: "route", Title: "Подписи"}}
	return fb
}

// covers — у документа мира есть все поля разделов «поля» шаблона: тогда он
// отрисовывается разметкой нормативного слоя, иначе — своими полями.
func (d *wdoc) covers(t dom.Template) bool {
	for _, sec := range t.Layout {
		if sec.Kind != "fields" {
			continue
		}
		for _, f := range sec.Fields {
			if !slices.ContainsFunc(d.Fields, func(x wfield) bool { return x.Key == f.Key }) {
				return false
			}
		}
	}
	return true
}

// trow — строка журнала документа по триггеру (rows шаблона event_record).
func trow(no int, at time.Time, event, step, actor, summary string) map[string]any {
	return map[string]any{"no": strconv.Itoa(no), "at": dom.FormatTime(at), "event": event, "step": step, "actor": actor, "source": "", "summary": summary}
}

// setPath — значение по пути через точку в content.
func setPath(m map[string]any, path string, v any) {
	parts := strings.Split(path, ".")
	for _, p := range parts[:len(parts)-1] {
		next, ok := m[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[p] = next
		}
		m = next
	}
	m[parts[len(parts)-1]] = v
}

func (d *wdoc) fieldValue(key string) string {
	for _, f := range d.Fields {
		if f.Key == key {
			return f.Value
		}
	}
	return dom.Empty
}

// build — канонический content, отрисовка и отпечаток версии no.
func (c *Ctx) build(d *wdoc, v wver, no int) (dom.Built, error) {
	t := c.M.docTemplate(d)
	content := map[string]any{}
	for _, f := range d.Fields {
		setPath(content, f.Key, f.Value)
	}
	if d.Rows != nil {
		for k, rows := range d.Rows(c) {
			content[k] = rows
		}
	}
	if len(d.Table) > 0 {
		rows := []map[string]any{}
		for _, r := range d.Table {
			if !r.At.After(c.T) {
				rows = append(rows, r.Row)
			}
		}
		content["rows"] = rows
	}
	approvals := []map[string]any{}
	for i, s := range v.Stages {
		a := map[string]any{"stage": i + 1, "title": s.Title, "authority_id": s.Authority, "signature_level": s.Level, "required_count": 1, "quorum": "one",
			"paper_allowed": s.Paper}
		if s.Role != "" {
			a["role"] = s.Role
		}
		if s.BySource {
			a["by_source"] = true
		}
		approvals = append(approvals, a)
	}
	content["doc_type"], content["title"], content["document_id"], content["version"] = t.DocType, d.Title, d.ID, no
	content["template_ref"], content["subject"], content["approvals"] = t.Ref(), string(d.Subject.Entity)+":"+d.Subject.ID, approvals
	sum := []map[string]string{}
	if len(t.Summary) > 0 {
		for _, f := range t.Summary {
			sum = append(sum, map[string]string{"label": f.Label, "value": d.fieldValue(f.Key)})
		}
	} else {
		for i, f := range d.Fields {
			if i == 5 {
				break
			}
			sum = append(sum, map[string]string{"label": f.Label, "value": f.Value})
		}
	}
	content["summary"] = sum
	return dom.Build(t, content)
}

// candidates — кто может подписать этап: явный список или сотрудники с ролью
// этапа (с наследованием ролей политики).
func (m *Model) candidates(s wstage) []string {
	if len(s.Who) > 0 {
		return slices.Clone(s.Who)
	}
	out := []string{}
	if m.policy == nil || s.Role == "" {
		return out
	}
	for _, p := range m.policy.Persons {
		for _, r := range p.Roles {
			if m.policy.hasRole(r.Role, s.Role) && !slices.Contains(out, p.ID) {
				out = append(out, p.ID)
			}
		}
	}
	return out
}

// hasRole — роль role совпадает с want или наследует её.
func (p *Policy) hasRole(role, want string) bool {
	seen := map[string]bool{}
	var walk func(r string) bool
	walk = func(r string) bool {
		if r == want {
			return true
		}
		if seen[r] {
			return false
		}
		seen[r] = true
		for _, x := range p.Roles {
			if x.ID == r {
				for _, in := range x.Inherits {
					if walk(in) {
						return true
					}
				}
			}
		}
		return false
	}
	return walk(role)
}

func sigEventID(d *wdoc, no, stage int) string {
	return fmt.Sprintf("EV-%s-V%d-S%d", strings.TrimPrefix(d.ID, "DOC-"), no, stage)
}

func sigClass(s *wsig) string {
	switch s.Method {
	case dom.MethodPaper:
		return "paper"
	case dom.MethodSource:
		if !strings.Contains(s.Person, "-") || strings.HasPrefix(s.Person, "CAM") || strings.HasPrefix(s.Person, "XRAY") {
			return "device"
		}
	}
	return "personal"
}

func keyRef(s *wsig) string {
	if s.Method != dom.MethodTokenAgent {
		return ""
	}
	return "key-" + strings.ToLower(s.Person) + "@1"
}

// docView — документ с маршрутом на часах шага: текущая (или запрошенная) версия.
func (c *Ctx) docView(d *wdoc, vers []wver, idx int, built []dom.Built) docsapp.DocumentView {
	t := c.T
	v, b, no := vers[idx], built[idx], idx+1
	live := d.Live
	status := v.status(t)
	if live {
		status = dom.StatusLive
	}
	var content map[string]any
	dec := json.NewDecoder(bytes.NewReader(b.Content))
	dec.UseNumber()
	_ = dec.Decode(&content)
	tpl := c.M.docTemplate(d)
	out := docsapp.DocumentView{DocumentID: d.ID, Version: no, Template: tpl.Ref(), DocFormatVersion: dom.DocFormatVersion, Title: d.Title, Subject: d.Subject,
		Status: status, Content: content, RenderingHash: b.RenderingHash, DocDigest: b.Digest, SummaryFields: []docsapp.DocumentSummaryField{},
		SourceEventIDs: []string{}, Stages: []docsapp.DocumentApprovalStage{}, Signatures: []docsapp.DocumentSignatureView{}, QR: dom.QR(d.ID, b.Digest),
		BasisSeq: c.Seq(), DocType: tpl.DocType, Class: tpl.Class, Live: live, Verification: "full", SubjectLabel: d.SubjectLabel}
	if sum, ok := content["summary"].([]any); ok {
		for i, x := range sum {
			f, _ := x.(map[string]any)
			l, _ := f["label"].(string)
			val, _ := f["value"].(string)
			out.SummaryFields = append(out.SummaryFields, docsapp.DocumentSummaryField{Key: "f" + strconv.Itoa(i+1), Label: l, Value: val})
		}
	}
	if no > 1 {
		p := no - 1
		out.SupersedesVersion = &p
	}
	for i, s := range v.Stages {
		n := i + 1
		cands := c.M.candidates(s)
		as := docsapp.DocumentApprovalStage{Stage: n, Authority: s.Authority, Quorum: "one", Required: 1, Level: s.Level, PaperAllowed: s.Paper, Candidates: cands,
			SignedBy: []string{}, Status: "pending"}
		rs := docsapp.DocumentRouteStage{Stage: n, Title: s.Title, Role: s.Role, AuthorityID: s.Authority, AuthorityLabel: s.Title, Quorum: "one", Required: 1,
			SignatureLevel: s.Level, PaperAllowed: s.Paper, BySource: s.BySource, Separation: []string{"distinct_signers"}, Signatures: []docsapp.DocumentStageSignature{}}
		if s.Paper {
			rs.AttesterAuthorityID = "paper_attestation"
		}
		if s.Role == "customer_representative" {
			rs.ExternalParty = "customer_representative"
			as.ExternalParty = "customer_representative"
		}
		if s.done(t) && !live {
			g := s.Sig
			ev := sigEventID(d, no, n)
			as.SignedBy, as.Status = []string{g.Person}, "done"
			rs.Done = true
			method := g.Method
			level := s.Level
			rs.Signatures = append(rs.Signatures, docsapp.DocumentStageSignature{EventID: ev, SignerID: g.Person, Class: sigClass(g), Level: level, Check: "valid",
				AttestedBy: g.Attester, PaperOriginalRef: g.PaperNo, Method: method, SignedAt: g.At, Counted: true, KeyRef: keyRef(g), KeyStorage: g.Storage})
			out.Signatures = append(out.Signatures, docsapp.DocumentSignatureView{EventID: ev, Stage: n, Method: method, SignerPersonID: g.Person, AttestedBy: g.Attester,
				Level: level, SignedAt: g.At, DocDigest: b.Digest, CurrentVersion: true, Verification: "valid", ProvenanceClass: sigClass(g)})
			out.SourceEventIDs = append(out.SourceEventIDs, ev)
		}
		out.Stages = append(out.Stages, as)
		out.Route = append(out.Route, rs)
	}
	for _, dc := range v.declined(t) {
		out.Declines = append(out.Declines, docsapp.DocumentDecline{EventID: fmt.Sprintf("EV-%s-V%d-D%d", strings.TrimPrefix(d.ID, "DOC-"), no, dc.Stage),
			Version: no, Stage: dc.Stage, SignerID: dc.Person, Comment: dc.Comment, At: dc.At})
	}
	if !live {
		for i, x := range vers {
			out.Versions = append(out.Versions, docsapp.DocumentVersionRef{Version: i + 1, DocDigest: built[i].Digest, Status: versionStatus(x.status(t)),
				DraftedAt: x.At, Signatures: countSigs(x, t)})
		}
	}
	for _, p := range v.Paper {
		if !p.At.After(t) {
			out.Paper = append(out.Paper, docsapp.DocumentPaperMark{EventID: fmt.Sprintf("EV-%s-V%d-P-%s", strings.TrimPrefix(d.ID, "DOC-"), no, strings.ToUpper(p.Status)),
				Version: no, Status: p.Status, CopyNo: p.CopyNo, At: p.At})
		}
	}
	if status == dom.StatusRouteClosed {
		id := fmt.Sprintf("EV-%s-V%d-CLOSED", strings.TrimPrefix(d.ID, "DOC-"), no)
		out.RouteClosed = &id
	}
	return out
}

// versionStatus — статус версии в списке версий (live — не бывает).
func versionStatus(st string) string {
	if st == dom.StatusLive {
		return dom.StatusDrafted
	}
	return st
}

func countSigs(v wver, t time.Time) int {
	n := 0
	for _, s := range v.Stages {
		if s.done(t) {
			n++
		}
	}
	return n
}

// docRow — строка реестра по виду документа.
func (c *Ctx) docRow(d *wdoc, view docsapp.DocumentView, v wver) docsapp.DocumentSummary {
	t := c.T
	row := docsapp.DocumentSummary{DocumentID: d.ID, Version: view.Version, Template: view.Template, Title: d.Title, Subject: d.Subject, Status: view.Status,
		DocDigest: view.DocDigest, DocType: view.DocType, Class: view.Class, Versions: len(view.Versions), SubjectLabel: d.SubjectLabel,
		ItemIDs: slices.Clone(d.Items), ProcessID: d.ProcessID, ProcessVersionID: d.PVID, StagesTotal: len(v.Stages)}
	if !d.Live {
		row.DraftedAt = tptr(v.At)
		row.PaperStatus = v.paperStatus(t)
	}
	row.State = docsapp.RegistryState(row.Status, row.PaperStatus)
	updated := v.At
	for i, s := range v.Stages {
		if s.done(t) && !d.Live {
			row.StagesDone++
			if s.Sig.At.After(updated) {
				updated = s.Sig.At
			}
			continue
		}
		if row.Awaiting == nil && !d.Live && (row.Status == dom.StatusDrafted || row.Status == dom.StatusSigning) {
			row.Awaiting = &docsapp.DocumentAwaiting{Stage: i + 1, Title: s.Title, Role: s.Role, Candidates: c.M.candidates(s)}
		}
	}
	for _, dc := range v.declined(t) {
		if dc.At.After(updated) {
			updated = dc.At
		}
	}
	for _, p := range v.Paper {
		if !p.At.After(t) && p.At.After(updated) {
			updated = p.At
		}
	}
	if !v.AnnulledAt.IsZero() && !v.AnnulledAt.After(t) && v.AnnulledAt.After(updated) {
		updated = v.AnnulledAt
	}
	if at, ok := v.closedAt(t); ok && !d.Live {
		row.ClosedAt = tptr(at)
	}
	if d.Live {
		updated = c.T
		if evs := c.itemEvents(c.M.itemByID[strings.TrimPrefix(d.Subject.ID, enterprise+":")]); len(evs) > 0 {
			updated = evs[len(evs)-1].Occurred
		}
	}
	row.UpdatedAt = tptr(updated)
	return row
}

// docsAt — документы, известные к шагу, с их текущим видом (кэш на шаг).
type docAt struct {
	doc   *wdoc
	vers  []wver
	built []dom.Built
	view  docsapp.DocumentView
	row   docsapp.DocumentSummary
}

func (c *Ctx) docsAt() []docAt {
	if c.docs != nil {
		return c.docs
	}
	out := []docAt{}
	for _, d := range c.M.documents() {
		if d.Live && (d.LiveFrom.After(c.T) || c.M.itemByID[strings.TrimPrefix(d.Subject.ID, enterprise+":")] == nil ||
			!c.S(c.M.itemByID[strings.TrimPrefix(d.Subject.ID, enterprise+":")]).Exists) {
			continue
		}
		vers := d.visible(c.T)
		if len(vers) == 0 {
			continue
		}
		built := make([]dom.Built, len(vers))
		for i, v := range vers {
			b, err := c.build(d, v, i+1)
			if err != nil {
				panic(fmt.Sprintf("world: документ %s: %v", d.ID, err)) // входы встроены — ошибка сборки
			}
			built[i] = b
		}
		view := c.docView(d, vers, len(vers)-1, built)
		out = append(out, docAt{doc: d, vers: vers, built: built, view: view, row: c.docRow(d, view, vers[len(vers)-1])})
	}
	c.docs = out
	return out
}

// renderDocuments — реестр документов, документ с маршрутом и отрисовка (FR-65).
func renderDocuments(c *Ctx) []loader.Response {
	list := docsapp.DocumentList{Items: []docsapp.DocumentSummary{}}
	var out []loader.Response
	for _, x := range c.docsAt() {
		list.Items = append(list.Items, x.row)
		out = append(out, resp("documents.document.read", x.view, "document_id", x.doc.ID),
			resp("documents.document.render", docsapp.DocumentRendering{DocumentID: x.doc.ID, Version: x.view.Version,
				RenderingHash: x.view.RenderingHash, HTML: x.built[len(x.built)-1].HTML}, "document_id", x.doc.ID))
		for i := 0; i+1 < len(x.vers); i++ {
			v := c.docView(x.doc, x.vers, i, x.built)
			out = append(out, resp("documents.document.read", v, "document_id", x.doc.ID, "version", strconv.Itoa(i+1)),
				resp("documents.document.render", docsapp.DocumentRendering{DocumentID: x.doc.ID, Version: i + 1, RenderingHash: x.built[i].RenderingHash,
					HTML: x.built[i].HTML}, "document_id", x.doc.ID, "version", strconv.Itoa(i+1)))
		}
	}
	n, zero := len(list.Items), 0
	list.CollectedFromHistory, list.ManualEntries = &n, &zero
	return append([]loader.Response{resp("documents.document.list", list)}, out...)
}

// itemDocuments — документы изделия для паспорта (FR-65): из того же реестра.
func (c *Ctx) itemDocuments(it *Item) []itemapp.ItemDocumentRef {
	out := []itemapp.ItemDocumentRef{}
	for _, x := range c.docsAt() {
		if !slices.Contains(x.doc.Items, FullID(it.ID)) {
			continue
		}
		out = append(out, itemapp.ItemDocumentRef{DocumentID: x.doc.ID, Template: x.view.Template, Title: x.doc.Title, Status: dom.PassportStatus(x.view.Status),
			Digest: x.view.DocDigest})
	}
	return out
}

// LoadTemplates — шаблоны документов нормативного слоя
// (normative/documents/templates.v1.yaml; нет файла — документы с отрисовкой
// по полям мира).
func LoadTemplates(fsys fs.FS) (dom.Templates, error) {
	b, err := fs.ReadFile(fsys, docsapp.PathTemplates)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return docsapp.TemplatesFromYAML(b)
}
