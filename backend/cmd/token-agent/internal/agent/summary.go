package agent

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/procs"
	dom "ant/internal/domain/signing"
)

// Сводка уровня 2 (AD-13, AD-14): 3–7 полей, которые агент вычисляет сам из
// подписываемых байтов — не из того, что прислала страница. Взломанный
// сервер может подсунуть другое содержимое, но тогда и сводка покажет другое.

// summaryHidden — поля тела, которые в сводку не идут: заголовок команды,
// идентификаторы (изделие показано отдельно), подпись.
var summaryHidden = []string{"command_id", "basis_seq", "policy_seq", "workplace_id", "signature", "reason", "item_id", "nc_id",
	"signal_ids", "observation_id", "operation_run_id", "signal_id"}

// fieldNames — русские подписи частых полей решения.
var fieldNames = map[string]string{
	"disposition": "Решение", "verdict": "Результат", "result": "Результат", "severity": "Тяжесть", "method": "Способ",
	"defect_type_code": "Вид дефекта", "requirement_ref": "Требование", "concession_id": "Разрешение на отклонение",
	"zone_ids": "Зоны", "decision": "Решение", "mode": "Режим", "step_key": "Шаг", "outcome": "Исход",
}

func summarize(p Prepared, b procs.SignBlock, payload []byte) []Field {
	var ev struct {
		EventType string          `json:"event_type"`
		ItemID    string          `json:"item_id"`
		Data      json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(payload, &ev)
	out := []Field{{Label: "Действие", Value: actionTitle(p.EventType)}}
	item := ev.ItemID
	if item == "" && b.CommandRequest != nil {
		item = deref(b.CommandRequest.ItemID)
	}
	if item != "" {
		out = append(out, Field{Label: "Изделие", Value: item})
	}
	var data map[string]any
	_ = json.Unmarshal(ev.Data, &data)
	body := data
	if inner, ok := data["body"].(map[string]any); ok && data["operation"] != nil {
		body = inner // соглашение «подписан запрос»: data = {operation, params, body}
	}
	if c := content(body); c != "" {
		out = append(out, Field{Label: "Содержание", Value: c})
	}
	if r, ok := body["reason"].(map[string]any); ok {
		if t, _ := r["text"].(string); t != "" {
			out = append(out, Field{Label: "Основание", Value: clip(t, 300)})
		}
	}
	out = append(out, Field{Label: "Подписант", Value: p.PersonID + " · ключ " + strings.Join(p.Signers, " + ")})
	lvl := "2 — с окном подтверждения"
	if p.Level == dom.Level1 {
		lvl = "1 — без окна (перечень уровня 1)"
	}
	out = append(out, Field{Label: "Уровень подписи", Value: lvl})
	out = append(out, Field{Label: "Отпечаток", Value: short(p.DocDigest)})
	if len(out) > 7 {
		out = out[:7]
	}
	return out
}

// actionTitle — название действия по каталогу типов записей.
func actionTitle(t string) string {
	if t == "" {
		return "—"
	}
	if info, ok := catalog.Lookup(catalog.Type(t)); ok && info.Title != "" {
		s := info.Title
		if info.Critical {
			s += " (критическое действие)"
		}
		return s
	}
	return t
}

// content — значимые поля решения в одну строку.
func content(body map[string]any) string {
	keys := make([]string, 0, len(body))
	for k := range body {
		if !slices.Contains(summaryHidden, k) {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		name := fieldNames[k]
		if name == "" {
			name = k
		}
		parts = append(parts, name+": "+scalar(body[k]))
	}
	return clip(strings.Join(parts, "; "), 500)
}

func scalar(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []any:
		ss := make([]string, 0, len(x))
		for _, e := range x {
			ss = append(ss, scalar(e))
		}
		return strings.Join(ss, ", ")
	case nil:
		return "—"
	case map[string]any:
		b, _ := json.Marshal(x)
		return string(b)
	}
	return fmt.Sprint(v)
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
