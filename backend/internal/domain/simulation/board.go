package simulation

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Табло «ожидалось → получилось» (FR-108, кейс §5.1, AD-26): утверждения
// scenarios/expected/‹сценарий›.yaml хранятся отдельно от входных событий и
// проверяются теми же операциями API (operationId), что показывают столы.
// Здесь — чистая часть: путь в ответе и сравнение; чтение делает application.

// Expected — ожидания одной карточки.
type Expected struct {
	Scenario    string       `json:"scenario"`
	Version     string       `json:"version"`
	Title       string       `json:"title"`
	Run         string       `json:"run"`
	Source      string       `json:"source,omitempty"`
	Checkpoints []Checkpoint `json:"checkpoints"`
}

// Checkpoint — момент проверки (время определения) и утверждения на нём.
type Checkpoint struct {
	// At — момент определения (RFC 3339) или «end» — конец прогона.
	At         string      `json:"at"`
	Label      string      `json:"label,omitempty"`
	MustNot    bool        `json:"must_not,omitempty"`
	Note       string      `json:"note,omitempty"`
	Assertions []Assertion `json:"assertions"`
}

// Assertion — утверждение табло.
type Assertion struct {
	ID   string `json:"id"`
	What string `json:"what"`
	// Check — исполняемая форма: операция API, параметры, путь, сравнение.
	Check Check `json:"check"`
	// Mapping — насколько точно утверждение процессной сессии сведено с
	// ответом операции: exact | draft (поле по смыслу, путь уточняет
	// модуль-владелец) | manual (видно на экране, API не проверяется).
	Mapping string `json:"mapping"`
	// Baseline — момент «до» для delta и unchanged (время определения).
	Baseline string `json:"baseline,omitempty"`
	// Proposal — утверждение по предложению процессной сессии, ждёт решения.
	Proposal bool `json:"proposal,omitempty"`
	// Source — исходное утверждение процессной сессии без изменений.
	Source map[string]any `json:"source,omitempty"`
	Note   string         `json:"note,omitempty"`
}

// Check — проверка: operationId, параметры (с плейсхолдерами), путь, сравнение.
type Check struct {
	Operation string         `json:"operation"`
	Params    map[string]any `json:"params,omitempty"`
	Path      string         `json:"path"`
	Op        string         `json:"op"`
	Value     any            `json:"value,omitempty"`
}

// Ops — допустимые сравнения.
var Ops = []string{"eq", "ne", "gte", "lte", "in", "set_eq", "contains", "not_contains", "exists", "not_exists", "delta", "unchanged", "before"}

// Status — итог строки табло.
type Status string

// Итоги строки табло.
const (
	StatusPending    Status = "pending"     // ещё не проверено
	StatusPassed     Status = "passed"      // ожидалось = получилось
	StatusFailed     Status = "failed"      // расхождение
	StatusNotReached Status = "not_reached" // прогон не дошёл до момента проверки
)

// Result — сравнение одной строки.
type Result struct {
	Status Status
	// Actual — полученное значение (JSON); пусто — не получено.
	Actual string
	Detail string
}

// Extract — значения по пути в ответе.
//
// Путь: сегменты через «/»: имя поля; число — элемент списка; «*» — все
// элементы; «#» — число элементов; фильтр «[поле=значение]» (поле может быть
// вложенным через «.», несколько условий через «,») после имени или отдельно.
// Результат — найденные значения (пусто — пути нет).
func Extract(doc any, path string) ([]any, error) {
	cur := []any{doc}
	for _, seg := range splitPath(path) {
		if seg == "" {
			continue
		}
		name, filter, err := parseSegment(seg)
		if err != nil {
			return nil, err
		}
		var next []any
		switch {
		case name == "#":
			n := 0
			for _, v := range cur {
				if a, ok := v.([]any); ok {
					n += len(a)
				} else if v != nil {
					n++
				}
			}
			next = []any{json.Number(strconv.Itoa(n))}
		case name == "*":
			for _, v := range cur {
				if a, ok := v.([]any); ok {
					next = append(next, a...)
				}
			}
		case isIndex(name):
			i, _ := strconv.Atoi(name)
			for _, v := range cur {
				if a, ok := v.([]any); ok && i < len(a) {
					next = append(next, a[i])
				}
			}
		case name != "":
			for _, v := range cur {
				if m, ok := v.(map[string]any); ok {
					if e, ok := m[name]; ok {
						next = append(next, e)
					}
				}
			}
		default:
			next = cur
		}
		if filter != nil {
			var kept []any
			for _, v := range next {
				arr, ok := v.([]any)
				if !ok {
					continue
				}
				var sel []any
				for _, e := range arr {
					if matchFilter(e, filter) {
						sel = append(sel, e)
					}
				}
				kept = append(kept, any(sel))
			}
			next = kept
		}
		cur = next
	}
	return cur, nil
}

func splitPath(p string) []string {
	var out []string
	depth := 0
	start := 0
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case '[':
			depth++
		case ']':
			depth--
		case '/':
			if depth == 0 {
				out = append(out, p[start:i])
				start = i + 1
			}
		}
	}
	return append(out, p[start:])
}

func isIndex(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

type cond struct{ key, value string }

func parseSegment(seg string) (string, []cond, error) {
	i := strings.IndexByte(seg, '[')
	if i < 0 {
		return seg, nil, nil
	}
	if !strings.HasSuffix(seg, "]") {
		return "", nil, fmt.Errorf("путь: фильтр %q не закрыт", seg)
	}
	var cs []cond
	for _, part := range strings.Split(seg[i+1:len(seg)-1], ",") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return "", nil, fmt.Errorf("путь: условие %q без «=»", part)
		}
		cs = append(cs, cond{strings.TrimSpace(k), strings.TrimSpace(v)})
	}
	return seg[:i], cs, nil
}

func matchFilter(e any, cs []cond) bool {
	for _, c := range cs {
		v := any(e)
		for _, k := range strings.Split(c.key, ".") {
			m, ok := v.(map[string]any)
			if !ok {
				return false
			}
			v = m[k]
		}
		if scalarString(v) != c.value {
			return false
		}
	}
	return true
}

// scalarString — строковое представление скаляра для фильтров и сравнения.
func scalarString(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return x
	case json.Number:
		return x.String()
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// Single — одно значение из найденных: ровно одно — оно, иначе список.
func Single(vals []any) any {
	if len(vals) == 1 {
		return vals[0]
	}
	if vals == nil {
		return nil
	}
	return any(vals)
}

// Evaluate сравнивает полученное с ожидаемым. found — путь найден в ответе;
// baseline — значение в момент «до» (delta, unchanged).
func Evaluate(c Check, actual any, found bool, baseline any) Result {
	res := Result{Actual: marshal(actual)}
	if !found {
		res.Actual = ""
	}
	ok, detail := compare(c.Op, actual, found, c.Value, baseline)
	res.Detail = detail
	if ok {
		res.Status = StatusPassed
	} else {
		res.Status = StatusFailed
	}
	return res
}

func marshal(v any) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}

func compare(op string, actual any, found bool, want, baseline any) (bool, string) {
	switch op {
	case "exists":
		return found && actual != nil, ""
	case "not_exists":
		return !found || actual == nil || isEmpty(actual), ""
	}
	if !found {
		if op == "not_contains" || op == "ne" {
			return true, "путь не найден — значения нет"
		}
		return false, "путь не найден в ответе"
	}
	switch op {
	case "eq":
		return match(want, actual), ""
	case "ne":
		return !match(want, actual), ""
	case "gte", "lte":
		return cmpNumbers(op, actual, want)
	case "in":
		arr, ok := want.([]any)
		if !ok {
			return false, "in: ожидается список"
		}
		for _, w := range arr {
			if match(w, actual) {
				return true, ""
			}
		}
		return false, ""
	case "set_eq":
		return setEq(want, actual), ""
	case "contains":
		return contains(actual, want), ""
	case "not_contains":
		return notContains(actual, want), ""
	case "delta":
		return delta(actual, baseline, want)
	case "unchanged":
		if baseline == nil {
			return false, "нет значения «до»"
		}
		return match(baseline, actual), ""
	case "before":
		return before(actual, want)
	}
	return false, "неизвестное сравнение " + op
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	case string:
		return x == ""
	}
	return false
}

// match — ожидаемое совпадает с полученным: объекты — по ожидаемым полям
// (в ответе могут быть другие поля), списки — поэлементно, скаляры — по
// строковому виду; моменты времени — как моменты.
func match(want, got any) bool {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return false
		}
		for _, k := range slices.Sorted(maps.Keys(w)) {
			if !match(w[k], g[k]) {
				return false
			}
		}
		return true
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return false
		}
		for i := range w {
			if !match(w[i], g[i]) {
				return false
			}
		}
		return true
	}
	ws, gs := scalarString(want), scalarString(got)
	if ws == gs {
		return true
	}
	if wt, err := time.Parse(time.RFC3339Nano, ws); err == nil {
		if gt, err := time.Parse(time.RFC3339Nano, gs); err == nil {
			return wt.Equal(gt)
		}
	}
	return false
}

func setEq(want, got any) bool {
	w, ok1 := want.([]any)
	g, ok2 := got.([]any)
	if !ok1 || !ok2 || len(w) != len(g) {
		return false
	}
	used := make([]bool, len(g))
	for _, x := range w {
		found := false
		for i, y := range g {
			if !used[i] && match(x, y) {
				used[i], found = true, true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func contains(got, want any) bool {
	if s, ok := got.(string); ok {
		if ws, ok := want.(string); ok {
			return strings.Contains(s, ws)
		}
	}
	g, ok := got.([]any)
	if !ok {
		g = []any{got}
	}
	ws, ok := want.([]any)
	if !ok {
		ws = []any{want}
	}
	for _, w := range ws {
		found := false
		for _, x := range g {
			if match(w, x) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func notContains(got, want any) bool {
	if s, ok := got.(string); ok {
		if ws, ok := want.(string); ok {
			return !strings.Contains(s, ws)
		}
	}
	g, ok := got.([]any)
	if !ok {
		g = []any{got}
	}
	ws, ok := want.([]any)
	if !ok {
		ws = []any{want}
	}
	for _, w := range ws {
		for _, x := range g {
			if match(w, x) {
				return false
			}
		}
	}
	return true
}

func toInt(v any) (int64, bool) {
	switch x := v.(type) {
	case json.Number:
		n, err := strconv.ParseInt(x.String(), 10, 64)
		return n, err == nil
	case int:
		return int64(x), true
	case int64:
		return x, true
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		return n, err == nil
	}
	return 0, false
}

func cmpNumbers(op string, actual, want any) (bool, string) {
	if wm, ok := want.(map[string]any); ok {
		am, ok := actual.(map[string]any)
		if !ok {
			return false, "ожидается объект чисел"
		}
		for _, k := range slices.Sorted(maps.Keys(wm)) {
			if ok, d := cmpNumbers(op, am[k], wm[k]); !ok {
				return false, k + ": " + d
			}
		}
		return true, ""
	}
	a, ok1 := toInt(actual)
	w, ok2 := toInt(want)
	if !ok1 || !ok2 {
		return false, "сравнение только целых"
	}
	if op == "gte" {
		return a >= w, ""
	}
	return a <= w, ""
}

func delta(actual, baseline, want any) (bool, string) {
	if baseline == nil {
		return false, "нет значения «до»"
	}
	if wm, ok := want.(map[string]any); ok {
		am, _ := actual.(map[string]any)
		bm, _ := baseline.(map[string]any)
		for _, k := range slices.Sorted(maps.Keys(wm)) {
			if ok, d := delta(am[k], bm[k], wm[k]); !ok {
				return false, k + ": " + d
			}
		}
		return true, ""
	}
	a, ok1 := toInt(actual)
	b, ok2 := toInt(baseline)
	w, ok3 := toInt(want)
	if !ok1 || !ok2 || !ok3 {
		return false, "delta только для целых"
	}
	if a-b != w {
		return false, fmt.Sprintf("изменение %d, ожидалось %d", a-b, w)
	}
	return true, ""
}

func before(actual, want any) (bool, string) {
	g, ok := actual.([]any)
	w, ok2 := want.([]any)
	if !ok || !ok2 {
		return false, "before: ожидаются списки"
	}
	last := -1
	for _, x := range w {
		idx := -1
		for i, y := range g {
			if match(x, y) {
				idx = i
				break
			}
		}
		if idx < 0 {
			return false, "нет элемента " + scalarString(x)
		}
		if idx <= last {
			return false, "порядок нарушен у " + scalarString(x)
		}
		last = idx
	}
	return true, ""
}
