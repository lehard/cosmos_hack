package simulation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// EventSchemas — проверка исходных событий генератора схемами контракта
// (contracts/events: конверт v1 и схема data по типу и версии). Подмножество
// draft-07, которое разрешает диалект схем событий (AD-20): type, $ref,
// properties, required, enum, pattern, minimum/maximum, minLength/maxLength,
// items, minItems/maxItems, uniqueItems, additionalProperties, propertyNames.
// Приём проверяет теми же схемами полной библиотекой (эпик 06); здесь —
// чтобы генератор не слал в приём событий не по контракту без умысла.
type EventSchemas struct {
	root  string
	cache map[string]map[string]any
	re    map[string]*regexp.Regexp
}

// NewEventSchemas — схемы из каталога contracts/events.
func NewEventSchemas(eventsDir string) *EventSchemas {
	return &EventSchemas{root: eventsDir, cache: map[string]map[string]any{}, re: map[string]*regexp.Regexp{}}
}

// Validate проверяет конверт и data события.
func (s *EventSchemas) Validate(event []byte) error {
	dec := json.NewDecoder(bytes.NewReader(event))
	dec.UseNumber()
	var ev map[string]any
	if err := dec.Decode(&ev); err != nil {
		return err
	}
	if err := s.check(filepath.Join(s.root, "common", "envelope.v1.json"), ev); err != nil {
		return fmt.Errorf("конверт: %w", err)
	}
	typ, _ := ev["event_type"].(string)
	ver, _ := ev["schema_version"].(json.Number)
	fam, _, _ := strings.Cut(typ, ".")
	file := filepath.Join(s.root, fam, typ+".v"+ver.String()+".json")
	if err := s.check(file, ev["data"]); err != nil {
		return fmt.Errorf("%s v%s data: %w", typ, ver, err)
	}
	return nil
}

func (s *EventSchemas) load(file string) (map[string]any, error) {
	if sc, ok := s.cache[file]; ok {
		return sc, nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var sc map[string]any
	if err := json.Unmarshal(b, &sc); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	s.cache[file] = sc
	return sc, nil
}

func (s *EventSchemas) check(file string, v any) error {
	sc, err := s.load(file)
	if err != nil {
		return err
	}
	return s.validate(file, sc, v, "")
}

// resolve — схема по $ref относительно файла.
func (s *EventSchemas) resolve(file, ref string) (string, map[string]any, error) {
	target, frag, _ := strings.Cut(ref, "#")
	if target != "" {
		file = filepath.Join(filepath.Dir(file), target)
	}
	sc, err := s.load(file)
	if err != nil {
		return "", nil, err
	}
	for _, p := range strings.Split(strings.TrimPrefix(frag, "/"), "/") {
		if p == "" {
			continue
		}
		next, ok := sc[p].(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("$ref %s: нет %s", ref, p)
		}
		sc = next
	}
	return file, sc, nil
}

func (s *EventSchemas) validate(file string, sc map[string]any, v any, at string) error {
	if ref, ok := sc["$ref"].(string); ok {
		f2, target, err := s.resolve(file, ref)
		if err != nil {
			return err
		}
		if err := s.validate(f2, target, v, at); err != nil {
			return err
		}
	}
	if t, ok := sc["type"]; ok && !typeOK(t, v) {
		return fmt.Errorf("%s: тип %s, ожидался %v", pathOr(at), kind(v), t)
	}
	if e, ok := sc["enum"].([]any); ok {
		found := false
		for _, x := range e {
			if fmt.Sprint(x) == fmt.Sprint(v) {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("%s: значение %v не из перечисления", pathOr(at), v)
		}
	}
	switch x := v.(type) {
	case string:
		if p, ok := sc["pattern"].(string); ok {
			re, err := s.regexp(p)
			if err != nil {
				return err
			}
			if !re.MatchString(x) {
				return fmt.Errorf("%s: %q не по шаблону %s", pathOr(at), x, p)
			}
		}
		if n, ok := num(sc["maxLength"]); ok && int64(len([]rune(x))) > n {
			return fmt.Errorf("%s: длиннее %d", pathOr(at), n)
		}
		if n, ok := num(sc["minLength"]); ok && int64(len([]rune(x))) < n {
			return fmt.Errorf("%s: короче %d", pathOr(at), n)
		}
	case json.Number:
		iv, err := strconv.ParseInt(x.String(), 10, 64)
		if err == nil {
			if n, ok := num(sc["minimum"]); ok && iv < n {
				return fmt.Errorf("%s: %d меньше %d", pathOr(at), iv, n)
			}
			if n, ok := num(sc["maximum"]); ok && iv > n {
				return fmt.Errorf("%s: %d больше %d", pathOr(at), iv, n)
			}
		}
	case []any:
		if n, ok := num(sc["minItems"]); ok && int64(len(x)) < n {
			return fmt.Errorf("%s: элементов меньше %d", pathOr(at), n)
		}
		if n, ok := num(sc["maxItems"]); ok && int64(len(x)) > n {
			return fmt.Errorf("%s: элементов больше %d", pathOr(at), n)
		}
		if u, _ := sc["uniqueItems"].(bool); u {
			seen := map[string]bool{}
			for _, e := range x {
				b, _ := json.Marshal(e)
				if seen[string(b)] {
					return fmt.Errorf("%s: повтор элемента %s", pathOr(at), b)
				}
				seen[string(b)] = true
			}
		}
		if it, ok := sc["items"].(map[string]any); ok {
			for i, e := range x {
				if err := s.validate(file, it, e, fmt.Sprintf("%s/%d", at, i)); err != nil {
					return err
				}
			}
		}
	case map[string]any:
		if req, ok := sc["required"].([]any); ok {
			for _, r := range req {
				if _, ok := x[r.(string)]; !ok {
					return fmt.Errorf("%s: нет обязательного поля %s", pathOr(at), r)
				}
			}
		}
		props, _ := sc["properties"].(map[string]any)
		for _, k := range slices.Sorted(maps.Keys(x)) {
			if ps, ok := props[k].(map[string]any); ok {
				if err := s.validate(file, ps, x[k], at+"/"+k); err != nil {
					return err
				}
				continue
			}
			switch ap := sc["additionalProperties"].(type) {
			case bool:
				if !ap {
					return fmt.Errorf("%s: лишнее поле %s", pathOr(at), k)
				}
			case map[string]any:
				if err := s.validate(file, ap, x[k], at+"/"+k); err != nil {
					return err
				}
			}
		}
		if pn, ok := sc["propertyNames"].(map[string]any); ok {
			for _, k := range slices.Sorted(maps.Keys(x)) {
				if err := s.validate(file, pn, k, at+"/"+k); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *EventSchemas) regexp(p string) (*regexp.Regexp, error) {
	if re, ok := s.re[p]; ok {
		return re, nil
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return nil, err
	}
	s.re[p] = re
	return re, nil
}

func pathOr(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

func num(v any) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case json.Number:
		n, err := x.Int64()
		return n, err == nil
	}
	return 0, false
}

func kind(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case json.Number:
		if _, err := x.Int64(); err == nil {
			return "integer"
		}
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return fmt.Sprintf("%T", v)
}

func typeOK(t, v any) bool {
	k := kind(v)
	ok := func(name string) bool {
		return name == k || (name == "number" && k == "integer")
	}
	switch x := t.(type) {
	case string:
		return ok(x)
	case []any:
		for _, e := range x {
			if s, _ := e.(string); ok(s) {
				return true
			}
		}
	}
	return false
}
