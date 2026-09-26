package httpapi

import (
	"encoding/json"
	"reflect"

	"github.com/danielgtaylor/huma/v2"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
)

// enumRef — схема перечисления T отдельным компонентом (components.schemas):
// так именованные перечисления приложения (ось момента, режим, класс операции,
// вид объекта…) получают имя в клиенте фронтенда, а приложение не зависит от Huma.
type enumRef[T platform.Enum] struct{}

// Schema реализует huma.SchemaProvider.
func (enumRef[T]) Schema(r huma.Registry) *huma.Schema {
	var zero T
	name := reflect.TypeFor[T]().Name()
	if n, ok := any(zero).(interface{ SchemaName() string }); ok {
		name = n.SchemaName()
	}
	if _, ok := r.Map()[name]; !ok {
		vals := zero.Values()
		enum := make([]any, len(vals))
		for i, v := range vals {
			enum[i] = v
		}
		s := &huma.Schema{Type: huma.TypeString, Enum: enum}
		if d, ok := any(zero).(interface{ Describe() string }); ok {
			s.Description = d.Describe()
		}
		r.Map()[name] = s
	}
	return &huma.Schema{Ref: "#/components/schemas/" + name}
}

// Enum публикует перечисление T компонентом схемы (вызывают transport/‹модуль›
// для своих перечислений до регистрации операций).
func Enum[T platform.Enum](api *API) {
	api.huma.OpenAPI().Components.Schemas.RegisterTypeAlias(reflect.TypeFor[T](), reflect.TypeFor[enumRef[T]]())
}

// registerComponents — общие компоненты: перечисления соглашений и сообщение SSE.
func registerComponents(h huma.API) {
	reg := h.OpenAPI().Components.Schemas
	reg.RegisterTypeAlias(reflect.TypeFor[platform.Mode](), reflect.TypeFor[enumRef[platform.Mode]]())
	reg.RegisterTypeAlias(reflect.TypeFor[platform.Axis](), reflect.TypeFor[enumRef[platform.Axis]]())
	reg.RegisterTypeAlias(reflect.TypeFor[platform.Class](), reflect.TypeFor[enumRef[platform.Class]]())
	reg.RegisterTypeAlias(reflect.TypeFor[platform.EntityKind](), reflect.TypeFor[enumRef[platform.EntityKind]]())
	// Компоненты, на которые ссылается клиент фронтенда, даже если их пока не
	// использует ни одна операция.
	reg.Schema(reflect.TypeFor[enumRef[platform.Mode]](), true, "")
	reg.Schema(reflect.TypeFor[enumRef[platform.Axis]](), true, "")
	reg.Schema(reflect.TypeFor[enumRef[platform.Class]](), true, "")
	reg.Schema(reflect.TypeFor[enumRef[platform.EntityKind]](), true, "")
	reg.Schema(reflect.TypeFor[EntityChanged](), true, "")
	reg.Schema(reflect.TypeFor[Problem](), true, "")
}

func moduleTitle(m string) string {
	for _, mi := range catalog.Modules {
		if mi.Name == m {
			return mi.Title
		}
	}
	return ""
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
