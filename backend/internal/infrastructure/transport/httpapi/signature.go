package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"ant/internal/application/platform"
)

// Подпись команды в общем декораторе (FR-66, FR-69, AD-13, AD-14, Д-59):
// подписывается запрос JCS{operation, params, body}, поэтому тело нужно
// ровно в том виде, в каком его подписал клиент, — до разбора в тип
// команды (разбор теряет порядок полей, пустые поля и запись чисел).
// Middleware captureSigned сохраняет байты тела и параметры пути команд с
// уровнем подписи; before после прав (Gate) вызывает порт Signatures и
// кладёт принятую подпись в контекст — её пишет в запись модуль-исполнитель.

// maxSignedBody — предел тела подписываемой команды, которое сохраняется для проверки.
const maxSignedBody = 1 << 20

type signedRequestKey struct{}

// captureSigned — байты тела и параметры пути команды с уровнем подписи ≥ 1.
func (a *API) captureSigned(ctx huma.Context, next func(huma.Context)) {
	op := ctx.Operation()
	if a.cfg.Signatures == nil || op == nil || ctx.Method() == http.MethodGet {
		next(ctx)
		return
	}
	act, ok := a.action(op.OperationID)
	if !ok || act.SignatureLevel < 1 || !act.IsCommand() {
		next(ctx)
		return
	}
	r, _ := humago.Unwrap(ctx)
	if r == nil || r.Body == nil {
		next(ctx)
		return
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, maxSignedBody+1))
	if err != nil {
		next(ctx)
		return
	}
	if len(b) > maxSignedBody {
		// Слишком большое тело: не сохраняем — подпись такой команды не проверяется.
		r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(b), r.Body))
		next(ctx)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(b))
	params := map[string]string{}
	for _, p := range op.Parameters {
		if p != nil && p.In == "path" {
			params[p.Name] = ctx.Param(p.Name)
		}
	}
	rq := platform.SignedRequest{Params: params, Body: json.RawMessage(b)}
	next(huma.WithContext(ctx, context.WithValue(ctx.Context(), signedRequestKey{}, rq)))
}

// action — описание операции по id.
func (a *API) action(id string) (platform.Action, bool) {
	for _, x := range a.actions {
		if x.ID == id {
			return x, true
		}
	}
	return platform.Action{}, false
}

// checkSignature — подпись команды уровня ≥ 1 через порт Signatures:
// отказ — ошибка с кодом семейства signing; принятая — в контекст.
func (a *API) checkSignature(ctx context.Context, act platform.Action, meta *platform.CommandMeta) (context.Context, error) {
	if a.cfg.Signatures == nil || act.SignatureLevel < 1 || !act.IsCommand() {
		return ctx, nil
	}
	rq, ok := ctx.Value(signedRequestKey{}).(platform.SignedRequest)
	if !ok {
		return ctx, nil
	}
	if meta != nil {
		rq.Meta = *meta
	}
	sig, skip, err := a.cfg.Signatures.CheckRequest(ctx, act, rq)
	if err != nil {
		return ctx, problemFrom(err, act.ID)
	}
	if skip {
		return ctx, nil
	}
	return platform.WithSignature(ctx, sig), nil
}
