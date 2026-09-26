package process

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Язык условий на стрелках urn:ant:expr:1 (решение Д-7; грамматика —
// contracts/bpmn-ext/README.md): сравнения поля с литералом, and, or, not,
// скобки. Без вызовов функций, арифметики, float и часов (AD-4): условие
// проверяется при загрузке (FR-13) и вычисляется детерминированно.

// ExprLanguage — идентификатор языка условий.
const ExprLanguage = "urn:ant:expr:1"

// Типы переменных языка условий.
const (
	VarEnum    = "enum"
	VarBoolean = "boolean"
	VarInteger = "integer"
)

// VarSpec — переменная языка условий: тип и допустимые значения перечисления.
type VarSpec struct {
	Type   string
	Values []string
}

// Переменные языка условий (contracts/bpmn-ext/rules.yaml →
// conditions.variables; соответствие проверяет тест application/process).
const (
	VarDecision          = "decision"
	VarDisposition       = "disposition"
	VarNCOutcome         = "nc.outcome"
	VarTestResult        = "test.result"
	VarToolsAccounted    = "tools.accounted"
	VarPlanRadiography   = "plan.radiography_required"
	VarPresentationNo    = "presentation.no"
	VarReworkCount       = "rework.count"
	maxSafeInteger int64 = 1<<53 - 1
)

// Variables — перечень переменных и их типы (Д-7: поля только из состояния
// изделия и решения).
var Variables = map[string]VarSpec{
	VarDecision:        {Type: VarEnum, Values: []string{ResolutionAccept, ResolutionAcceptWithConcession, ResolutionReject, ResolutionInsufficientData}},
	VarDisposition:     {Type: VarEnum, Values: []string{"rework", "repair", "use_as_is", "scrap", "return_to_supplier"}},
	VarNCOutcome:       {Type: VarEnum, Values: []string{"rework_or_repair", "use_as_is", "scrapped", "returned"}},
	VarTestResult:      {Type: VarEnum, Values: []string{"tight", "leak", "invalid"}},
	VarToolsAccounted:  {Type: VarBoolean},
	VarPlanRadiography: {Type: VarBoolean},
	VarPresentationNo:  {Type: VarInteger},
	VarReworkCount:     {Type: VarInteger},
}

// Value — значение переменной или литерала: строка (перечисление), целое или логическое.
type Value struct {
	Kind string `json:"k"`
	S    string `json:"s,omitempty"`
	I    int64  `json:"i,omitempty"`
	B    bool   `json:"b,omitempty"`
}

// Str, Int, Bool — конструкторы значений.
func Str(s string) Value  { return Value{Kind: VarEnum, S: s} }
func Int(i int64) Value   { return Value{Kind: VarInteger, I: i} }
func Bool(b bool) Value   { return Value{Kind: VarBoolean, B: b} }
func (v Value) String() string {
	switch v.Kind {
	case VarInteger:
		return strconv.FormatInt(v.I, 10)
	case VarBoolean:
		return strconv.FormatBool(v.B)
	}
	return v.S
}

// Expr — узел дерева условия.
type Expr struct {
	// Op — or | and | not | cmp.
	Op    string
	L, R  *Expr
	Field string
	Cmp   string
	Lit   Value
}

// ParseExpr разбирает условие и проверяет его по перечню переменных vars
// (nil — Variables). Ошибка — текст причины для process.condition_invalid.
func ParseExpr(src string, vars map[string]VarSpec) (*Expr, error) {
	if vars == nil {
		vars = Variables
	}
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &exprParser{toks: toks, vars: vars}
	e, err := p.or()
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.toks) {
		return nil, fmt.Errorf("лишнее после выражения: %q", p.toks[p.pos].text)
	}
	return e, nil
}

type token struct {
	kind string // ident | str | int | op | lp | rp | kw
	text string
}

func lex(s string) ([]token, error) {
	var out []token
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '(':
			out = append(out, token{"lp", "("})
			i++
		case c == ')':
			out = append(out, token{"rp", ")"})
			i++
		case c == '\'':
			j := strings.IndexByte(s[i+1:], '\'')
			if j < 0 {
				return nil, fmt.Errorf("незакрытая строка с позиции %d", i)
			}
			out = append(out, token{"str", s[i+1 : i+1+j]})
			i += j + 2
		case c == '=' || c == '!' || c == '<' || c == '>':
			if i+1 < len(s) && s[i+1] == '=' {
				out = append(out, token{"op", s[i : i+2]})
				i += 2
				continue
			}
			if c == '<' || c == '>' {
				out = append(out, token{"op", string(c)})
				i++
				continue
			}
			return nil, fmt.Errorf("недопустимый оператор %q (допустимы ==, !=, <, <=, >, >=)", string(c))
		case c == '-' || (c >= '0' && c <= '9'):
			j := i + 1
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			if s[i:j] == "-" {
				return nil, fmt.Errorf("минус без числа")
			}
			if j < len(s) && s[j] == '.' {
				return nil, fmt.Errorf("числа с точкой не поддерживаются (AD-4)")
			}
			out = append(out, token{"int", s[i:j]})
			i = j
		case c >= 'a' && c <= 'z':
			j := i + 1
			for j < len(s) && (s[j] >= 'a' && s[j] <= 'z' || s[j] >= '0' && s[j] <= '9' || s[j] == '_' || s[j] == '.') {
				j++
			}
			w := s[i:j]
			if w == "and" || w == "or" || w == "not" || w == "true" || w == "false" {
				out = append(out, token{"kw", w})
			} else {
				out = append(out, token{"ident", w})
			}
			i = j
		default:
			return nil, fmt.Errorf("недопустимый символ %q (вызовы функций, арифметика и || не поддерживаются — пишите or)", string(c))
		}
	}
	return out, nil
}

type exprParser struct {
	toks []token
	pos  int
	vars map[string]VarSpec
}

func (p *exprParser) peek() (token, bool) {
	if p.pos < len(p.toks) {
		return p.toks[p.pos], true
	}
	return token{}, false
}

func (p *exprParser) or() (*Expr, error) {
	l, err := p.and()
	if err != nil {
		return nil, err
	}
	for {
		t, ok := p.peek()
		if !ok || t.kind != "kw" || t.text != "or" {
			return l, nil
		}
		p.pos++
		r, err := p.and()
		if err != nil {
			return nil, err
		}
		l = &Expr{Op: "or", L: l, R: r}
	}
}

func (p *exprParser) and() (*Expr, error) {
	l, err := p.not()
	if err != nil {
		return nil, err
	}
	for {
		t, ok := p.peek()
		if !ok || t.kind != "kw" || t.text != "and" {
			return l, nil
		}
		p.pos++
		r, err := p.not()
		if err != nil {
			return nil, err
		}
		l = &Expr{Op: "and", L: l, R: r}
	}
}

func (p *exprParser) not() (*Expr, error) {
	if t, ok := p.peek(); ok && t.kind == "kw" && t.text == "not" {
		p.pos++
		e, err := p.not()
		if err != nil {
			return nil, err
		}
		return &Expr{Op: "not", L: e}, nil
	}
	return p.primary()
}

func (p *exprParser) primary() (*Expr, error) {
	t, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("выражение оборвано")
	}
	if t.kind == "lp" {
		p.pos++
		e, err := p.or()
		if err != nil {
			return nil, err
		}
		if t, ok := p.peek(); !ok || t.kind != "rp" {
			return nil, fmt.Errorf("нет закрывающей скобки")
		}
		p.pos++
		return e, nil
	}
	if t.kind != "ident" {
		return nil, fmt.Errorf("ожидалось поле, получено %q", t.text)
	}
	p.pos++
	spec, known := p.vars[t.text]
	if !known {
		return nil, fmt.Errorf("неизвестное поле %q (поля — только из состояния изделия и решения, rules.yaml)", t.text)
	}
	op, ok := p.peek()
	if !ok || op.kind != "op" {
		return nil, fmt.Errorf("после поля %s ожидалось сравнение", t.text)
	}
	p.pos++
	lt, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("после %s %s нет литерала", t.text, op.text)
	}
	p.pos++
	e := &Expr{Op: "cmp", Field: t.text, Cmp: op.text}
	switch spec.Type {
	case VarEnum:
		if lt.kind != "str" {
			return nil, fmt.Errorf("поле %s — перечисление: справа строковый литерал", t.text)
		}
		if !slices.Contains(spec.Values, lt.text) {
			return nil, fmt.Errorf("значения %q нет в перечне поля %s", lt.text, t.text)
		}
		if op.text != "==" && op.text != "!=" {
			return nil, fmt.Errorf("поле %s — перечисление: только == и !=", t.text)
		}
		e.Lit = Str(lt.text)
	case VarBoolean:
		if lt.kind != "kw" || (lt.text != "true" && lt.text != "false") {
			return nil, fmt.Errorf("поле %s — логическое: справа true или false", t.text)
		}
		if op.text != "==" && op.text != "!=" {
			return nil, fmt.Errorf("поле %s — логическое: только == и !=", t.text)
		}
		e.Lit = Bool(lt.text == "true")
	case VarInteger:
		if lt.kind != "int" {
			return nil, fmt.Errorf("поле %s — целое: справа целый литерал", t.text)
		}
		n, err := strconv.ParseInt(lt.text, 10, 64)
		if err != nil || n > maxSafeInteger || n < -maxSafeInteger {
			return nil, fmt.Errorf("целое %s вне ±(2^53−1)", lt.text)
		}
		e.Lit = Int(n)
	}
	return e, nil
}

// Eval вычисляет условие над значениями переменных. Неизвестная (не
// заданная) переменная делает сравнение ложным — ветка не выбирается по
// догадке (NFR-UI-4: «нет данных» ≠ значение).
func (e *Expr) Eval(vars map[string]Value) bool {
	if e == nil {
		return true
	}
	switch e.Op {
	case "or":
		return e.L.Eval(vars) || e.R.Eval(vars)
	case "and":
		return e.L.Eval(vars) && e.R.Eval(vars)
	case "not":
		return !e.L.Eval(vars)
	}
	v, ok := vars[e.Field]
	if !ok || v.Kind != e.Lit.Kind {
		return false
	}
	switch v.Kind {
	case VarInteger:
		switch e.Cmp {
		case "==":
			return v.I == e.Lit.I
		case "!=":
			return v.I != e.Lit.I
		case "<":
			return v.I < e.Lit.I
		case "<=":
			return v.I <= e.Lit.I
		case ">":
			return v.I > e.Lit.I
		case ">=":
			return v.I >= e.Lit.I
		}
	case VarBoolean:
		return (v.B == e.Lit.B) == (e.Cmp == "==")
	default:
		return (v.S == e.Lit.S) == (e.Cmp == "==")
	}
	return false
}
