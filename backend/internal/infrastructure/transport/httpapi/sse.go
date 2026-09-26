package httpapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"ant/internal/application/platform"
)

// EntityChanged — сообщение SSE-канала живых обновлений (AD-21;
// contracts/events/common/sse-entity-changed.v1.json, канал sse в asyncapi.yaml):
// сущность, id и seq записи журнала, после которой она изменилась. Данных
// сущности нет — фронтенд инвалидирует ключ Vue Query [сущность, id].
type EntityChanged struct {
	Entity platform.EntityKind `json:"entity" doc:"Вид сущности — первый элемент ключа Vue Query."`
	ID     string              `json:"id" maxLength:"128" doc:"Идентификатор сущности; для live_map и integrity — global."`
	Seq    int64               `json:"seq" minimum:"1" doc:"Позиция журнала, после которой сущность изменилась; она же id события SSE."`
	RunID  string              `json:"run_id,omitempty" maxLength:"128" doc:"Прогон сценария, если изменение в его пространстве имён (AD-38)."`
	Mode   platform.Mode       `json:"mode,omitempty" doc:"Режим ведущих портов, отдавших изменение (AD-36)."`
}

// Subscribe открывает подписку на изменения после afterSeq в пределах прогона;
// ошибка открытия — ответ problem+json до начала потока (501, 401 …). next
// блокирует до следующего изменения или отмены ctx; closeFn освобождает подписку.
type Subscribe func(ctx context.Context, afterSeq int64, runID string) (next func(context.Context) (EntityChanged, error), closeFn func(), err error)

type streamIn struct {
	LastEventID string `header:"Last-Event-ID" doc:"seq последнего полученного события (переподключение)."`
	RunID       string `query:"run_id" maxLength:"128" doc:"Прогон сценария (AD-38)."`
}

type streamOut struct {
	Body func(huma.Context)
}

// RegisterStream объявляет SSE-операцию journal.stream.subscribe (GET /api/v1/stream).
// Схема ответа text/event-stream — строка (поток), тип сообщения data: —
// компонент EntityChanged: так клиент orval собирается без ручного разбора.
func RegisterStream(a *API, subscribe Subscribe) {
	act := platform.Action{ID: "journal.stream.subscribe", Class: platform.ClassRead, Owner: "journal", Subject: "live_map"}
	if err := act.Validate(); err != nil {
		panic(err)
	}
	a.actions = append(a.actions, act)
	op := huma.Operation{
		OperationID: act.ID,
		Method:      http.MethodGet,
		Path:        Prefix + "/stream",
		Summary:     "Живые обновления столов (SSE)",
		Description: "Канал sse из contracts/events/asyncapi.yaml: `id:` = seq, `event: entity_changed`, `data:` — EntityChanged (components). " +
			"Фронтенд инвалидирует ключ Vue Query [сущность, id]; переподключение — с Last-Event-ID (AD-21). " +
			"Бюджет «событие → экран» ≤ 2 с (FR-2).",
		Tags: []string{act.Owner},
		Extensions: map[string]any{
			"x-ant-action": actionExtension(act),
			// Тип сообщения data: — ссылкой на компонент (иначе Huma вычистит
			// неиспользуемую схему, а клиенту фронтенда нужен тип EntityChanged).
			"x-sse-message": map[string]any{"event": "entity_changed", "data": map[string]any{"$ref": "#/components/schemas/EntityChanged"}},
		},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Поток событий",
				Content: map[string]*huma.MediaType{
					"text/event-stream": {Schema: &huma.Schema{Type: huma.TypeString, Description: "Поток SSE; data: — EntityChanged"}},
				},
			},
		},
	}
	huma.Register(a.huma, op, func(ctx context.Context, in *streamIn) (*streamOut, error) {
		if err := a.before(ctx, act, in); err != nil {
			return nil, err
		}
		if subscribe == nil {
			return nil, problemFrom(platform.NotImplemented(act.ID), act.ID)
		}
		var after int64
		if _, err := sscanInt(in.LastEventID, &after); err != nil {
			return nil, problemFrom(platformValidation("Last-Event-ID", err.Error()), act.ID)
		}
		next, closeFn, err := subscribe(ctx, after, in.RunID)
		if err != nil {
			return nil, problemFrom(err, act.ID)
		}
		mode := a.ModeFor(act.Owner)
		return &streamOut{Body: func(hctx huma.Context) {
			defer closeFn()
			hctx.SetHeader("Content-Type", "text/event-stream")
			hctx.SetHeader("Cache-Control", "no-store")
			hctx.SetHeader("Ant-Backend", string(mode))
			w := hctx.BodyWriter()
			for {
				ev, err := next(hctx.Context())
				if err != nil {
					return
				}
				b, err := jsonMarshal(ev)
				if err != nil {
					return
				}
				if _, err := w.Write([]byte("id: " + itoa(ev.Seq) + "\nevent: entity_changed\ndata: " + string(b) + "\n\n")); err != nil {
					return
				}
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}}, nil
	})
}
