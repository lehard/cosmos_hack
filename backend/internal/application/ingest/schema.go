package ingest

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	"ant/internal/contracts/schemas"
	dom "ant/internal/domain/ingest"
)

// schemaSet — схемы контракта для проверки сырых тел (AD-20): те же файлы
// contracts/**/*.json, что встроены в бинарник, тем же валидатором
// santhosh-tekuri/jsonschema; компилируются лениво и кешируются.
type schemaSet struct {
	mu    sync.Mutex
	c     *jsonschema.Compiler
	cache map[string]*jsonschema.Schema
}

// fsLoader отдаёт схемы из встроенной копии contracts/ по $id; по сети схемы
// не загружаются никогда (домен .invalid).
type fsLoader struct{}

func (fsLoader) Load(url string) (any, error) {
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

func newSchemaSet() *schemaSet {
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft7)
	c.UseLoader(fsLoader{})
	return &schemaSet{c: c, cache: map[string]*jsonschema.Schema{}}
}

// Пути схем от contracts/.
const (
	envelopeSchema = "events/common/envelope.v1.json"
	dsseSchema     = "crypto/dsse-envelope.schema.json"
)

// dataSchemaPath — схема data типа: events/‹семейство›/‹тип›.v‹N›.json (AD-20).
func dataSchemaPath(eventType string, version int) string {
	family, _, _ := strings.Cut(eventType, ".")
	return fmt.Sprintf("events/%s/%s.v%d.json", family, eventType, version)
}

func (s *schemaSet) get(path string) (*jsonschema.Schema, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sch, ok := s.cache[path]; ok {
		return sch, nil
	}
	sch, err := s.c.Compile(schemas.BaseURI + path)
	if err != nil {
		return nil, err
	}
	s.cache[path] = sch
	return sch, nil
}

// parseDoc разбирает JSON для валидатора (числа — json.Number).
func parseDoc(b []byte) (any, error) { return jsonschema.UnmarshalJSON(bytes.NewReader(b)) }

// validate проверяет doc схемой path и переводит ошибки в нейтральные нарушения
// домена; prefix — JSON Pointer места doc в сообщении («/data»).
func (s *schemaSet) validate(path string, doc any, prefix string) ([]dom.Violation, error) {
	sch, err := s.get(path)
	if err != nil {
		return nil, err
	}
	err = sch.Validate(doc)
	if err == nil {
		return nil, nil
	}
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return nil, err
	}
	var out []dom.Violation
	collect(ve, prefix, &out)
	return out, nil
}

// collect собирает листья дерева ошибок валидатора.
func collect(ve *jsonschema.ValidationError, prefix string, out *[]dom.Violation) {
	if len(ve.Causes) > 0 {
		for _, c := range ve.Causes {
			collect(c, prefix, out)
		}
		return
	}
	ptr := prefix + pointer(ve.InstanceLocation)
	switch k := ve.ErrorKind.(type) {
	case *kind.Required:
		for _, m := range k.Missing {
			*out = append(*out, dom.Violation{Kind: dom.ViolationRequired, Pointer: ptr, Missing: m})
		}
	case *kind.Enum:
		*out = append(*out, dom.Violation{Kind: dom.ViolationEnum, Pointer: ptr, Value: fmt.Sprint(k.Got)})
	default:
		*out = append(*out, dom.Violation{Kind: dom.ViolationOther, Pointer: ptr, Message: ve.Error()})
	}
}

// pointer — JSON Pointer (RFC 6901) из пути валидатора.
func pointer(loc []string) string {
	var b strings.Builder
	for _, p := range loc {
		b.WriteByte('/')
		b.WriteString(strings.NewReplacer("~", "~0", "/", "~1").Replace(p))
	}
	return b.String()
}

var defaultSchemas = newSchemaSet()

// ValidateEnvelope проверяет событие (сырые байты JSON) схемой конверта и
// схемой data его типа и версии, без требований приёма к виду записи. Нужна
// самопроверке служебных записей и источникам, проверяющим сообщение до отправки.
func ValidateEnvelope(raw []byte) (dom.Decision, error) {
	canon, err := dom.Canonicalize(raw)
	if err != nil {
		return dom.Decision{Outcome: dom.OutcomeQuarantined, Detail: err.Error()}, nil
	}
	h := parseHeader(canon)
	doc, err := parseDoc(canon)
	if err != nil {
		return dom.Decision{}, err
	}
	vs, err := defaultSchemas.validate(envelopeSchema, doc, "")
	if err != nil {
		return dom.Decision{}, err
	}
	if obj, ok := doc.(map[string]any); ok && h.EventType != "" {
		dv, err := defaultSchemas.validate(dataSchemaPath(h.EventType, h.SchemaVersion), obj["data"], "/data")
		if err != nil {
			return dom.Decision{}, err
		}
		vs = append(vs, dv...)
	}
	return dom.Classify(h.EventType, vs), nil
}
