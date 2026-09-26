package documents

import (
	"bytes"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Документы по событию-триггеру (эпик 44, FR-65, AD-12; каталог документов
// спайна): шаблон нормативного слоя с построителем event_record и полем
// trigger оформляется свёрткой изделия сам, как только в журнале изделия
// появляется запись одного из типов trigger.events. Содержимое — шапка
// изделия, запись-триггер (тип, время, операция, автор, источник) и её data
// целиком; поля и подписи для людей задаёт отрисовка шаблона (layout), а
// маршрут подписей — route шаблона (AD-43). Два вида:
//   - per: event — документ на каждую запись (акт, извещение, запись
//     вмешательства): id `‹ШАБЛОН›-‹изделие›-‹event_id›`;
//   - per: item — журнал изделия (изолятор, предъявление ОТК, маршрутные
//     листы): один документ на изделие, каждая запись — строка и новая
//     версия (AD-12: новая версия — новый отпечаток).
// Записи вне изделия (приёмка партии, сменные документы) свёртка изделия не
// видит — их оформляют запросом (documents.version.request) или модуль,
// владеющий потоком, намерением Draft.

// DocEventRecord — построитель «документ по событию-триггеру».
const DocEventRecord = "event_record"

// Виды триггера (Trigger.Per).
const (
	TriggerPerEvent = "event"
	TriggerPerItem  = "item"
)

// Trigger — когда шаблон оформляет документ сам (нормативный слой).
type Trigger struct {
	// Events — типы записей журнала изделия, по которым оформляется документ.
	Events []string `json:"events"`
	// Per — event (документ на запись) | item (журнал изделия); нет — event.
	Per string `json:"per,omitempty"`
	// When — отбор по полю data записи (решение «списать», предъявление ВП).
	When *TriggerWhen `json:"when,omitempty"`
}

// TriggerWhen — значение поля data записи-триггера — одно из Values.
type TriggerWhen struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}

// Fires — запись r — триггер шаблона.
func (t *Trigger) Fires(r kernel.Record) bool {
	if t == nil || !slices.Contains(t.Events, string(r.Type)) {
		return false
	}
	return t.When == nil || slices.Contains(t.When.Values, dataString(r.Data, t.When.Field))
}

// EventRow — запись-триггер в содержимом документа.
type EventRow struct {
	EventID string          `json:"event_id"`
	Type    string          `json:"type"`
	At      string          `json:"at"`
	Step    string          `json:"step,omitempty"`
	Actor   string          `json:"actor,omitempty"`
	Source  string          `json:"source,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// TriggeredID — id документа по триггеру.
func TriggeredID(t Template, itemID, eventID string) string {
	id := strings.ToUpper(t.ID) + "-" + itemID
	if t.Trigger != nil && t.Trigger.Per == TriggerPerItem {
		return id
	}
	return id + "-" + eventID
}

// reduceTriggers — оформить документы шаблонов, чей триггер — запись r.
func (s *State) reduceTriggers(r kernel.Record, env Env, up Upstream) {
	if r.ItemID == "" || r.Kind == catalog.KindReaction {
		return
	}
	for _, t := range env.Templates {
		if t.DocType != DocEventRecord || !t.Complete() || !t.Trigger.Fires(r) {
			continue
		}
		if latest, ok := env.Templates.ByRef(t.ID); !ok || latest.Ref() != t.Ref() {
			continue // по триггеру оформляет только последняя версия шаблона
		}
		id := TriggeredID(t, s.itemID(), r.EventID)
		d := s.doc(id)
		if d == nil {
			s.Docs = append(s.Docs, Doc{ID: id, TemplateRef: t.Ref(), DocType: t.DocType, Class: t.Class, Title: t.Title, Subject: "item:" + s.itemID()})
			d = &s.Docs[len(s.Docs)-1]
		}
		row := EventRow{EventID: r.EventID, Type: string(r.Type), At: FormatTime(r.OccurredAt), Actor: PersonOf(r.Actor), Source: r.SourceID, Data: r.Data}
		if k := dataString(r.Data, "step_key"); k != "" {
			row.Step = env.StepTitle(k)
		}
		d.Context.Rows = append(d.Context.Rows, row)
		d.Context.SourceEvent = r.EventID
		s.draft(env, d, r, up)
	}
}

// eventBody — содержимое документа по триггеру: изделие, последняя запись
// (event, data) и все записи-триггеры строками; by_source — автор записи
// закрывает первый этап, если шаблон так велит.
func (s *State) eventBody(d *Doc) (map[string]any, []Ref, []Signature) {
	rows := d.Context.Rows
	if len(rows) == 0 {
		return nil, nil, nil
	}
	last := rows[len(rows)-1]
	table := make([]any, 0, len(rows))
	var src []Ref
	for i, x := range rows {
		table = append(table, map[string]any{"no": strconv.Itoa(i + 1), "at": x.At, "event": x.Type, "step": x.Step, "actor": x.Actor, "source": x.Source,
			"summary": dataSummary(x.Data)})
		src = append(src, Ref{ID: x.EventID})
	}
	body := map[string]any{
		"item":  map[string]any{"item_id": s.itemID(), "item_type_id": s.Header.ItemTypeID, "order_id": s.Header.OrderID, "lots": strings.Join(s.Header.Lots, ", ")},
		"event": map[string]any{"event_id": last.EventID, "type": last.Type, "at": last.At, "step": last.Step, "actor": last.Actor, "source": last.Source},
		"data":  decodeData(last.Data),
		"rows":  table,
	}
	var signers []Signature
	if last.Actor != "" {
		signers = append(signers, Signature{EventID: last.EventID, Stage: 1, Person: last.Actor, Method: MethodSource, Level: 1})
	}
	return body, src, signers
}

// decodeData — data записи как дерево без float (AD-4); не разобралось — пусто.
func decodeData(raw json.RawMessage) any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil || v == nil {
		return map[string]any{}
	}
	return v
}

func dataString(raw json.RawMessage, key string) string {
	m, _ := decodeData(raw).(map[string]any)
	v, _ := m[key].(string)
	return v
}

// dataSummary — строка «ключ: значение» простых полей data для таблицы журнала.
func dataSummary(raw json.RawMessage) string {
	m, _ := decodeData(raw).(map[string]any)
	var parts []string
	for _, k := range sortedKeys(m) {
		switch v := m[k].(type) {
		case string:
			parts = append(parts, k+": "+v)
		case json.Number:
			parts = append(parts, k+": "+v.String())
		case bool:
			parts = append(parts, k+": "+strconv.FormatBool(v))
		}
	}
	return strings.Join(parts, "; ")
}
