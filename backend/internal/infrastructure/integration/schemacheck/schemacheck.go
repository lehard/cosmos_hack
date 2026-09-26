// Пакет schemacheck — проверка сообщений внешних систем схемами контракта
// contracts/integrations тем же валидатором, что у приёма (santhosh-tekuri/
// jsonschema, AD-20): адаптер проверяет исходящее сообщение до отправки, а
// входящее — до перевода на наш язык, и ошибка интеграции видна до передачи
// во внешнюю систему (FR-111, критерий О8).
//
// Слой: infrastructure, зона integration (AD-1) — общая опора адаптеров
// (Галактика, MES, КОМПАС и следующих, docs/new-adapter.md). Схемы — из
// встроенной копии contracts (backend/internal/contracts/schemas, make generate).
//
// Требования: FR-111, AD-18, AD-20. Владелец: эпик 31.
package schemacheck

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	"ant/internal/contracts/schemas"
)

var (
	mu    sync.Mutex
	comp  *jsonschema.Compiler
	cache = map[string]*jsonschema.Schema{}
)

type loader struct{}

func (loader) Load(url string) (any, error) {
	p, ok := strings.CutPrefix(url, schemas.BaseURI)
	if !ok {
		return nil, fmt.Errorf("схема %s вне contracts/", url)
	}
	b, err := schemas.FS.ReadFile("contracts/" + p)
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(b))
}

// Schema — скомпилированная схема контракта по пути от contracts/
// (например, "integrations/cad/assembly.schema.json").
func Schema(path string) (*jsonschema.Schema, error) {
	mu.Lock()
	defer mu.Unlock()
	if comp == nil {
		comp = jsonschema.NewCompiler()
		comp.DefaultDraft(jsonschema.Draft7)
		comp.UseLoader(loader{})
	}
	if s, ok := cache[path]; ok {
		return s, nil
	}
	s, err := comp.Compile(schemas.BaseURI + path)
	if err != nil {
		return nil, err
	}
	cache[path] = s
	return s, nil
}

// Validate — документ doc (JSON) по схеме контракта path; ошибка — что не так.
func Validate(path string, doc []byte) error {
	s, err := Schema(path)
	if err != nil {
		return err
	}
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(doc))
	if err != nil {
		return err
	}
	return s.Validate(v)
}

// Violation — первое нарушение схемы: JSON Pointer поля (для отсутствующего
// обязательного — путь к нему) и пояснение (для problem+json и карантина с
// полем, FR-30).
func Violation(err error) (field, detail string) {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return "/", err.Error()
	}
	leaf := ve
	for len(leaf.Causes) > 0 {
		leaf = leaf.Causes[0]
	}
	var b strings.Builder
	for _, p := range leaf.InstanceLocation {
		b.WriteByte('/')
		b.WriteString(strings.NewReplacer("~", "~0", "/", "~1").Replace(p))
	}
	field = b.String()
	switch k := leaf.ErrorKind.(type) {
	case *kind.Required:
		if len(k.Missing) > 0 {
			return field + "/" + k.Missing[0], "обязательное поле отсутствует"
		}
	case *kind.Enum:
		return nonEmpty(field), fmt.Sprintf("значение %v не из перечисления", k.Got)
	}
	detail = leaf.Error()
	if i := strings.Index(detail, "': "); i >= 0 {
		detail = detail[i+3:]
	}
	return nonEmpty(field), detail
}

func nonEmpty(f string) string {
	if f == "" {
		return "/"
	}
	return f
}
