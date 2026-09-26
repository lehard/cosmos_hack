package ops

import (
	"context"

	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

// respond — ответ операции op из мира заготовок на шаге курсора (или на
// момент as_of); params — HTTP-параметры операции без момента и страницы.
func respond[T any](ctx context.Context, op string, params map[string]string, m *platform.Moment) (T, error) {
	var out T
	rt, err := loader.Default()
	if err != nil {
		return out, err
	}
	err = rt.Respond(ctx, op, params, m, &out)
	return out, err
}

// decide — команда на заготовках (FR-129): мир не меняется, кроме шага
// ожидания именно этого решения над этим объектом — тогда курсор идёт дальше.
func decide(ctx context.Context, op, kind, id string, meta platform.CommandMeta) (platform.Receipt, error) {
	rt, err := loader.Default()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Decide(ctx, op, loader.ObjectRef{Kind: kind, ID: id}, meta)
}
