package documents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"
)

// Каноническая отрисовка (AD-12): HTML — чистая функция шаблона@версия и
// content; серверного PDF нет. Печатная рамка (QR `ant:doc:‹id›:‹отпечаток›`,
// дата печати, колонтитул «получено из системы») в отрисовку не входит —
// её строит PrintFrame поверх готового HTML. Разметка намеренно простая и
// стабильная: любой байт отрисовки входит в rendering_hash, а значит и в
// отпечаток; меняется разметка — меняется doc_format_version.

// Empty — значение отсутствующего поля в отрисовке.
const Empty = "—"

// Render — каноническая отрисовка HTML документа по шаблону. content —
// канонический JSON (Canonical).
func Render(t Template, content json.RawMessage) (string, error) {
	doc, err := decode(content)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(`<article class="ant-doc" data-template="` + esc(t.Ref()) + `">` + "\n")
	title := text(lookup(doc, "title"))
	if title == Empty {
		title = t.Title
	}
	b.WriteString("<header><h1>" + esc(title) + "</h1>")
	b.WriteString(`<p class="ant-doc-id">` + esc(text(lookup(doc, "document_id"))) + " · версия " + esc(text(lookup(doc, "version"))) + "</p></header>\n")
	for _, s := range t.Layout {
		switch s.Kind {
		case "fields":
			b.WriteString("<section>" + heading(s.Title) + "<dl>")
			for _, f := range s.Fields {
				b.WriteString("<dt>" + esc(f.Label) + "</dt><dd>" + esc(text(lookup(doc, f.Key))) + "</dd>")
			}
			b.WriteString("</dl></section>\n")
		case "table":
			b.WriteString("<section>" + heading(s.Title) + "<table><thead><tr>")
			for _, c := range s.Columns {
				b.WriteString("<th>" + esc(c.Label) + "</th>")
			}
			b.WriteString("</tr></thead><tbody>")
			rows, _ := lookup(doc, s.Rows).([]any)
			if len(rows) == 0 {
				b.WriteString(`<tr><td colspan="` + strconv.Itoa(max(len(s.Columns), 1)) + `">` + Empty + "</td></tr>")
			}
			for _, row := range rows {
				b.WriteString("<tr>")
				for _, c := range s.Columns {
					b.WriteString("<td>" + esc(text(lookup(row, c.Key))) + "</td>")
				}
				b.WriteString("</tr>")
			}
			b.WriteString("</tbody></table></section>\n")
		case "route":
			b.WriteString("<section>" + heading(s.Title) + "<ol>")
			stages, _ := lookup(doc, "approvals").([]any)
			for _, st := range stages {
				b.WriteString("<li>" + esc(stageLine(st)) + "</li>")
			}
			b.WriteString("</ol></section>\n")
		case "text":
			b.WriteString("<section>" + heading(s.Title) + "<p>" + esc(text(lookup(doc, s.Key))) + "</p></section>\n")
		}
	}
	if sum, _ := lookup(doc, "summary").([]any); len(sum) > 0 {
		b.WriteString(`<section class="ant-doc-summary"><h2>Сводка для подписи</h2><dl>`)
		for _, f := range sum {
			b.WriteString("<dt>" + esc(text(lookup(f, "label"))) + "</dt><dd>" + esc(text(lookup(f, "value"))) + "</dd>")
		}
		b.WriteString("</dl></section>\n")
	}
	b.WriteString("</article>\n")
	return b.String(), nil
}

func heading(t string) string {
	if t == "" {
		return ""
	}
	return "<h2>" + esc(t) + "</h2>"
}

func esc(s string) string { return html.EscapeString(s) }

// stageLine — этап маршрута подписей для людей: кто, полномочие, уровень,
// сколько подписей, бумага (AD-43: подписант видит, кто ещё подписывает).
func stageLine(st any) string {
	parts := []string{"Этап " + text(lookup(st, "stage")) + ". " + text(lookup(st, "title"))}
	parts = append(parts, "полномочие "+text(lookup(st, "authority_id")))
	if r := text(lookup(st, "role")); r != Empty {
		parts = append(parts, "роль "+r)
	}
	if k := text(lookup(st, "stamp_kind")); k != Empty {
		parts = append(parts, "клеймо «"+k+"»")
	}
	parts = append(parts, "уровень "+text(lookup(st, "signature_level")), "подписей "+text(lookup(st, "required_count")))
	if lookup(st, "by_source") == true {
		parts = append(parts, "закрывается решением-источником")
	}
	if lookup(st, "paper_allowed") == true {
		parts = append(parts, "бумага с заверением допустима")
	} else {
		parts = append(parts, "бумага запрещена")
	}
	return strings.Join(parts, "; ")
}

// decode — разбор канонического content без float: числа — json.Number (AD-4).
func decode(content json.RawMessage) (any, error) {
	d := json.NewDecoder(bytes.NewReader(content))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, fmt.Errorf("documents: content: %w", err)
	}
	return v, nil
}

// lookup — значение по пути через точку; нет — nil.
func lookup(v any, path string) any {
	if path == "" {
		return nil
	}
	cur := v
	for _, p := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[p]
	}
	return cur
}

// text — значение поля для людей: строка как есть, число, «да»/«нет»,
// список строк через «; »; отсутствующее — «—».
func text(v any) string {
	switch x := v.(type) {
	case nil:
		return Empty
	case string:
		if x == "" {
			return Empty
		}
		return x
	case json.Number:
		return x.String()
	case bool:
		if x {
			return "да"
		}
		return "нет"
	case []any:
		if len(x) == 0 {
			return Empty
		}
		parts := make([]string, 0, len(x))
		for _, e := range x {
			parts = append(parts, text(e))
		}
		return strings.Join(parts, "; ")
	default:
		b, err := Canonical(x)
		if err != nil {
			return Empty
		}
		return string(b)
	}
}

// SummaryValue — значение поля сводки уровня 2 по пути в content (AD-12).
func SummaryValue(content json.RawMessage, path string) string {
	doc, err := decode(content)
	if err != nil {
		return Empty
	}
	return text(lookup(doc, path))
}
