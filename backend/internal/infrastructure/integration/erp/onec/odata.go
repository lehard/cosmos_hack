package onec

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	app "ant/internal/application/erp"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Чтение 1С через стандартный интерфейс OData v3 (порт учёта, вход):
// задания («Этап производства» к выполнению), номенклатура, поступившие
// партии («Приобретение товаров и услуг» + серии) и соответствия внешних ID
// (AD-18, FR-95). Правила сопоставления — проектное предположение
// (docs/integrations/1c.md, §5): наш ID не выводится из Ref_Key, а
// складывается из естественного ключа (обозначение, номер серии, номер
// документа) латиницей; соответствие Ref_Key ↔ наш ID уходит записью
// reference.external_id.mapped.

// Row — строка ответа OData (реквизиты 1С на русском).
type Row map[string]any

func (r Row) str(k string) string {
	switch v := r[k].(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	}
	return ""
}

func (r Row) int(k string) int {
	s := r.str(k)
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(f)
	}
	return 0
}

func (r Row) bool(k string) bool { return r.str(k) == "true" }

func (r Row) rows(k string) []Row {
	xs, _ := r[k].([]any)
	out := make([]Row, 0, len(xs))
	for _, x := range xs {
		if m, ok := x.(map[string]any); ok {
			out = append(out, Row(m))
		}
	}
	return out
}

// EmptyRef — пустая ссылка 1С.
const EmptyRef = "00000000-0000-0000-0000-000000000000"

// Entities — сущность OData целиком (`?$format=json[&$filter=…]`).
func (c *Client) Entities(ctx context.Context, entity, filter string) ([]Row, error) {
	q := url.Values{"$format": {"json"}}
	if filter != "" {
		q.Set("$filter", filter)
	}
	code, b, err := c.do(ctx, http.MethodGet, c.manifest.OData.Path+url.PathEscape(entity)+"?"+q.Encode(), nil, nil)
	if err != nil {
		return nil, err
	}
	if err := statusErr(code, b); err != nil {
		return nil, err
	}
	var resp struct {
		Value []map[string]any `json:"value"`
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.UseNumber()
	if err := dec.Decode(&resp); err != nil {
		return nil, &app.ContractError{Detail: "ответ OData " + entity + " не разобран: " + err.Error()}
	}
	seen := map[string]bool{}
	out := make([]Row, 0, len(resp.Value))
	for _, v := range resp.Value {
		r := Row(v)
		// Дубль строки в ответе (сбой «дубль») не даёт второго факта.
		if k := r.str("Ref_Key"); k != "" {
			if seen[k] {
				continue
			}
			seen[k] = true
		}
		out = append(out, r)
	}
	return out, nil
}

// msk — время 1С без зоны считается московским (проектное предположение).
var msk = time.FixedZone("MSK", 3*3600)

func parse1C(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, msk)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// Pull — входящие факты порта учёта (application/erp.Ledger).
func (c *Client) Pull(ctx context.Context) ([]app.Inbound, error) {
	nom, err := c.Entities(ctx, "Catalog_Номенклатура", "")
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	types := map[string]string{} // Ref_Key → наш item_type_id
	var entries []ev.NomenclatureEntry
	var marks []string
	var out []app.Inbound
	for _, r := range nom {
		if r.bool("DeletionMark") {
			continue
		}
		des := r.str("Артикул")
		if des == "" {
			des = r.str("Code")
		}
		id := ItemTypeID(des)
		types[r.str("Ref_Key")] = id
		e := ev.NomenclatureEntry{ExternalID: r.str("Ref_Key"), Designation: des, Name: r.str("Description")}
		if id != "" {
			x := ev.ObjectID(id)
			e.ItemTypeID = &x
			out = append(out, mapping("item_type", r.str("Ref_Key"), id))
		}
		entries = append(entries, e)
		marks = append(marks, r.str("Ref_Key")+":"+r.str("DataVersion"))
	}
	if len(entries) > 0 {
		slices.Sort(marks)
		out = append(out, app.Inbound{Type: catalog.ErpNomenclatureSynced, EventID: uid("onec.nomenclature", strings.Join(marks, ",")), OccurredAt: now,
			Data: ev.ErpNomenclatureSyncedV1{ExternalSystem: ev.ErpNomenclatureSyncedV1ExternalSystemOnec, Entries: entries}})
	}
	stages, err := c.Entities(ctx, "Document_ЭтапПроизводства2_2", "Posted eq true and Статус eq 'КВыполнению'")
	if err != nil {
		return nil, err
	}
	for _, r := range stages {
		if !r.bool("Posted") || r.bool("DeletionMark") || r.str("Статус") != "КВыполнению" {
			continue
		}
		lines := r.rows("ВыходныеИзделия")
		if len(lines) == 0 || types[lines[0].str("Номенклатура_Key")] == "" {
			continue
		}
		oid := OrderID(r.str("Number"))
		d := ev.ErpOrderReceivedV1{OrderID: ev.ObjectID(oid), ExternalSystem: ev.ErpOrderReceivedV1ExternalSystemOnec, ExternalNumber: r.str("Number"),
			ItemTypeID: ev.ObjectID(types[lines[0].str("Номенклатура_Key")]), Quantity: max(lines[0].int("Количество"), 1)}
		if due := r.str("ДатаОкончания"); len(due) >= 10 {
			x := ev.Date(due[:10])
			d.DueDate = &x
		}
		if rev := r.str("РевизияКД"); rev != "" {
			d.ItemRevision = &rev
		}
		out = append(out, app.Inbound{Type: catalog.ErpOrderReceived, EventID: uid("onec.order", r.str("Ref_Key")+":"+r.str("DataVersion")),
			OccurredAt: parse1C(r.str("Date")), Data: d}, mapping("order", r.str("Ref_Key"), oid))
	}
	lots, err := c.lots(ctx, types)
	if err != nil {
		return nil, err
	}
	out = append(out, lots...)
	for i := range out {
		if out[i].OccurredAt.IsZero() {
			out[i].OccurredAt = now
		}
	}
	return out, nil
}

func (c *Client) lots(ctx context.Context, types map[string]string) ([]app.Inbound, error) {
	series, err := c.Entities(ctx, "Catalog_СерииНоменклатуры", "")
	if err != nil {
		return nil, err
	}
	byRef := map[string]Row{}
	for _, s := range series {
		byRef[s.str("Ref_Key")] = s
	}
	partners, err := c.Entities(ctx, "Catalog_Контрагенты", "")
	if err != nil {
		return nil, err
	}
	supplier := map[string]string{}
	for _, p := range partners {
		supplier[p.str("Ref_Key")] = Latin(p.str("Code"))
	}
	docs, err := c.Entities(ctx, "Document_ПриобретениеТоваровУслуг", "Posted eq true")
	if err != nil {
		return nil, err
	}
	var out []app.Inbound
	for _, d := range docs {
		if !d.bool("Posted") || d.bool("DeletionMark") {
			continue
		}
		sup := supplier[d.str("Контрагент_Key")]
		for _, l := range d.rows("Товары") {
			s, ok := byRef[l.str("Серия_Key")]
			it := types[l.str("Номенклатура_Key")]
			if !ok || it == "" || sup == "" {
				continue
			}
			lot := LotID(s.str("Номер"))
			x := ev.ErpLotReceivedV1{LotID: ev.ObjectID(lot), ExternalSystem: ev.ErpLotReceivedV1ExternalSystemOnec, ExternalNumber: d.str("Number"),
				SupplierID: ev.ObjectID(sup), ItemTypeID: ev.ObjectID(it), Quantity: max(l.int("Количество"), 1)}
			if h := s.str("НомерПлавки"); h != "" {
				x.HeatNo = &h
			}
			if cert := s.str("НомерСертификата"); cert != "" {
				x.CertificateNo = &cert
			}
			if g := s.str("ГоденДо"); len(g) >= 10 && !strings.HasPrefix(g, "0001") {
				dd := ev.Date(g[:10])
				x.ExpiryDate = &dd
			}
			out = append(out, app.Inbound{Type: catalog.ErpLotReceived, EventID: uid("onec.lot", d.str("Ref_Key")+":"+d.str("DataVersion")+":"+l.str("LineNumber")),
				OccurredAt: parse1C(d.str("Date")), Data: x}, mapping("lot", s.str("Ref_Key"), lot))
		}
	}
	return out, nil
}

// mapping — соответствие внешнего ID (reference.external_id.mapped, FR-95).
func mapping(kind, ref, id string) app.Inbound {
	return app.Inbound{Type: catalog.ReferenceExternalIdMapped, EventID: uid("onec.map", kind+":"+ref+":"+id),
		Data: ev.ReferenceExternalIDMappedV1{System: ev.ReferenceExternalIDMappedV1SystemOnec, ObjectKind: ev.ReferenceExternalIDMappedV1ObjectKind(kind),
			ExternalID: ref, InternalID: ev.ObjectID(id)}}
}

func uid(kind, key string) string { return kernel.UUIDv5(constants.NsAnt, kind+"\x1f"+key) }

var latin = map[rune]string{'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "E", 'Ж': "ZH", 'З': "Z", 'И': "I", 'Й': "Y",
	'К': "K", 'Л': "L", 'М': "M", 'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U", 'Ф': "F", 'Х': "KH", 'Ц': "TS",
	'Ч': "CH", 'Ш': "SH", 'Щ': "SHCH", 'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "YU", 'Я': "YA"}

// Latin — естественный ключ 1С латиницей (транслитерация, пробел и прочее — «-»).
func Latin(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		u := unicode.ToUpper(r)
		switch {
		case r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("._:/@-", r)):
			b.WriteRune(r)
		case latin[u] != "" || u == 'Ъ' || u == 'Ь':
			b.WriteString(latin[u])
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// ItemTypeID — наш тип изделия по обозначению 1С: обозначение по ЕСКД
// латиницей без суффикса документа («ФЛ-100.00.000 СБ» → FL-100.00.000).
func ItemTypeID(designation string) string {
	f := strings.Fields(designation)
	if len(f) == 0 {
		return ""
	}
	return Latin(f[0])
}

// LotID — наш ID партии по номеру серии 1С («П-117» → LOT-P-117; уже с LOT- — как есть).
func LotID(number string) string {
	l := Latin(number)
	if strings.HasPrefix(l, "LOT-") {
		return l
	}
	return "LOT-" + l
}

// OrderID — наш ID задания по номеру документа 1С: последняя группа цифр
// без ведущих нулей, не короче четырёх знаков («ЭП00-000917» → ORD-0917).
func OrderID(number string) string {
	end := len(number)
	for end > 0 && (number[end-1] < '0' || number[end-1] > '9') {
		end--
	}
	start := end
	for start > 0 && number[start-1] >= '0' && number[start-1] <= '9' {
		start--
	}
	digits := strings.TrimLeft(number[start:end], "0")
	if digits == "" {
		return "ORD-" + Latin(number)
	}
	return fmt.Sprintf("ORD-%04s", digits)
}
