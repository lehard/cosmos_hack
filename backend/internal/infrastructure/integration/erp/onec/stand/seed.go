package stand

import (
	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// Затравка stand-а 1С — «база предприятия» демо-сценария фланца ФЛ-100
// (normative/reference/flange): справочники, поступления партий и задание на
// производство. Ref_Key — UUIDv5 от вида и кода: одинаковы при каждом старте.

// Ref — Ref_Key объекта затравки.
func Ref(kind, code string) string {
	return kernel.UUIDv5(constants.NsAnt, "stand-1c\x1f"+kind+"\x1f"+code)
}

const dataVersion = "AAAAAQAAAAA="

func catalogRow(kind, code, name string, extra Row) Row {
	r := Row{"Ref_Key": Ref(kind, code), "DataVersion": dataVersion, "DeletionMark": false, "Code": code, "Description": name}
	for k, v := range extra {
		r[k] = v
	}
	return r
}

// nomenclature — номенклатура: обозначение по ЕСКД (Артикул) и наименование.
var nomenclature = []struct{ Code, Article, Name string }{
	{"00-00000100", "ФЛ-100.00.000 СБ", "Фланец люка гермокорпуса в сборе"},
	{"00-00000101", "ФЛ-100.01.000", "Узел сварной: фланец + патрубок"},
	{"00-00000102", "ФЛ-100.01.001", "Фланец (заготовка — поковка АМг6)"},
	{"00-00000103", "ФЛ-100.01.002", "Патрубок"},
	{"00-00000104", "ФЛ-100.00.003", "Крышка"},
	{"00-00000105", "ФЛ-100.00.004", "Уплотнение"},
	{"00-00000106", "ФЛ-100.00.005", "Болт М8"},
	{"00-00000107", "ФЛ-100.00.006", "Шайба 8"},
	{"00-00000108", "ФЛ-100.01.003", "Кольцо усиливающее"},
}

// warehouses — склады цехов (коды — как у складов цехов нормативного слоя, FR-130).
var warehouses = []struct{ Code, Name string }{
	{"WH-SK", "Склад покупных и заготовок"},
	{"WH-MC", "Кладовая механического цеха"},
	{"WH-WC", "Кладовая сварочного цеха"},
	{"WH-AC", "Кладовая сборочно-испытательного цеха"},
	{"WH-FG", "Склад готовой продукции"},
}

var departments = []struct{ Code, Name string }{
	{"WS-SK", "Склад и входной контроль"}, {"WS-MC", "Механический цех"}, {"WS-WC", "Сварочный цех"},
	{"WS-AC", "Сборочно-испытательный цех"}, {"WS-QA", "ОТК и выпуск"},
}

var partners = []struct{ Code, Name, INN string }{
	{"SUP-METAL", "Металлургический завод", "7700000001"},
	{"SUP-PIPE", "Поставщик патрубков", "7700000002"},
	{"SUP-SEAL", "Поставщик уплотнений", "7700000003"},
	{"SUP-FAST", "Поставщик крепежа", "7700000004"},
	{"SUP-3", "Поставщик-3", "7700000005"},
}

// seedLine — строка поступления: серия, номенклатура, количество, плавка, сертификат, срок годности.
type seedLine struct {
	Series, Article string
	Qty             int
	Heat, Cert      string
	Expiry          string
}

// seedReceipt — поступление партий от поставщика.
type seedReceipt struct {
	Number, Date, Partner string
	Lines                 []seedLine
}

// receipts — поступления партий.
var receipts = []seedReceipt{
	{"ПТ00-000211", "2026-09-10T09:00:00", "SUP-METAL", []seedLine{
		{"BLANK-01", "ФЛ-100.01.001", 20, "П-2211", "С-8841", ""},
		{"COVER-01", "ФЛ-100.00.003", 20, "П-2212", "С-8842", ""},
	}},
	{"ПТ00-000212", "2026-09-10T10:00:00", "SUP-PIPE", []seedLine{{"PIPE-01", "ФЛ-100.01.002", 20, "П-7302", "С-1190", ""}}},
	{"ПТ00-000213", "2026-09-10T11:00:00", "SUP-SEAL", []seedLine{{"SEAL-01", "ФЛ-100.00.004", 40, "", "С-5520", "2027-06-30T00:00:00"}}},
	{"ПТ00-000214", "2026-09-10T12:00:00", "SUP-FAST", []seedLine{
		{"BOLT-01", "ФЛ-100.00.005", 500, "", "С-3310", ""},
		{"WASH-01", "ФЛ-100.00.006", 500, "", "С-3311", ""},
	}},
	{"ПТ00-000217", "2026-09-22T08:00:00", "SUP-3", []seedLine{{"R-117", "ФЛ-100.01.003", 10, "П-117", "С-117", ""}}},
}

// seed — сущности OData затравки по имени сущности.
func seed() map[string][]Row {
	out := map[string][]Row{}
	for _, n := range nomenclature {
		out["Catalog_Номенклатура"] = append(out["Catalog_Номенклатура"],
			catalogRow("nomenclature", n.Article, n.Name, Row{"Code": n.Code, "Артикул": n.Article, "НаименованиеПолное": n.Name}))
	}
	for _, w := range warehouses {
		out["Catalog_Склады"] = append(out["Catalog_Склады"], catalogRow("warehouse", w.Code, w.Name, nil))
	}
	for _, d := range departments {
		out["Catalog_СтруктураПредприятия"] = append(out["Catalog_СтруктураПредприятия"], catalogRow("department", d.Code, d.Name, nil))
	}
	for _, p := range partners {
		out["Catalog_Контрагенты"] = append(out["Catalog_Контрагенты"], catalogRow("partner", p.Code, p.Name, Row{"ИНН": p.INN}))
	}
	for _, rc := range receipts {
		var lines []any
		for i, l := range rc.Lines {
			expiry := "0001-01-01T00:00:00"
			if l.Expiry != "" {
				expiry = l.Expiry
			}
			out["Catalog_СерииНоменклатуры"] = append(out["Catalog_СерииНоменклатуры"], Row{
				"Ref_Key": Ref("series", l.Series), "DataVersion": dataVersion, "DeletionMark": false, "Description": l.Series, "Номер": l.Series,
				"НомерПлавки": l.Heat, "НомерСертификата": l.Cert, "ГоденДо": expiry})
			lines = append(lines, map[string]any{"LineNumber": itoa(i + 1), "Номенклатура_Key": Ref("nomenclature", article(l.Article)),
				"Серия_Key": Ref("series", l.Series), "Количество": l.Qty})
		}
		out["Document_ПриобретениеТоваровУслуг"] = append(out["Document_ПриобретениеТоваровУслуг"], Row{
			"Ref_Key": Ref("receipt", rc.Number), "DataVersion": dataVersion, "DeletionMark": false, "Number": rc.Number, "Date": rc.Date,
			"Posted": true, "Контрагент_Key": Ref("partner", rc.Partner), "Склад_Key": Ref("warehouse", "WH-SK"), "Товары": lines})
	}
	out["Document_ЗаказНаПроизводство2_2"] = []Row{{"Ref_Key": Ref("order", "ЗП00-000917"), "DataVersion": dataVersion, "DeletionMark": false,
		"Number": "ЗП00-000917", "Date": "2026-09-14T08:30:00", "Posted": true}}
	out["Document_ЭтапПроизводства2_2"] = []Row{Stage("ЭП00-000917", "2026-09-14T09:00:00", "2026-10-09T00:00:00", 40, "Ref:"+Ref("order", "ЗП00-000917"))}
	return out
}

// Stage — «Этап производства» к выполнению на фланец в сборе.
func Stage(number, date, due string, qty int, order string) Row {
	if len(order) > 4 && order[:4] == "Ref:" {
		order = order[4:]
	}
	return Row{"Ref_Key": Ref("stage", number), "DataVersion": dataVersion, "DeletionMark": false, "Number": number, "Date": date,
		"Posted": true, "Распоряжение_Key": order, "Подразделение_Key": Ref("department", "WS-MC"), "Статус": "КВыполнению",
		"НаправлениеДеятельности_Key": Ref("direction", "ГОЗ-2026"), "ДатаОкончания": due, "РевизияКД": "Б",
		"ВыходныеИзделия": []any{map[string]any{"LineNumber": "1", "Номенклатура_Key": Ref("nomenclature", "ФЛ-100.00.000 СБ"),
			"Серия_Key": "00000000-0000-0000-0000-000000000000", "Количество": qty}}}
}

// article — артикул затравки по обозначению (как в номенклатуре).
func article(a string) string { return a }

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoa(i/10) + string(rune('0'+i%10))
}
