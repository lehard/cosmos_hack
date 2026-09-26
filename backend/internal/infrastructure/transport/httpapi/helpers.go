package httpapi

import (
	"context"
	"fmt"
	"net/http"

	"ant/internal/application/platform"
)

// Read — операция чтения (GET): вход I (параметры пути и запроса; для чтения
// состояния — встраивает MomentQuery), fn получает разобранный момент (AD-22) и
// возвращает тело ответа. Заголовок Ant-Backend ставит Register.
func Read[I, T any](a *API, route Route, act platform.Action, fn func(ctx context.Context, in *I, m platform.Moment) (T, error)) {
	if act.Class == "" {
		act.Class = platform.ClassRead
	}
	Register(a, route, act, func(ctx context.Context, in *I) (*Out[T], error) {
		var m platform.Moment
		if mo, ok := any(in).(momenter); ok {
			var err error
			if m, err = mo.Moment(); err != nil {
				return nil, platformValidation("query", err.Error())
			}
		}
		v, err := fn(ctx, in, m)
		if err != nil {
			return nil, err
		}
		return OK(v), nil
	})
}

// Do — команда (POST по умолчанию): вход I с полем Body, тип которого
// встраивает platform.CommandHeader (command_id, basis_seq, policy_seq); fn
// вызывает ведущий порт Commands и возвращает квитанцию (AD-7).
func Do[I any](a *API, route Route, act platform.Action, fn func(ctx context.Context, in *I) (platform.Receipt, error)) {
	if route.Method == "" {
		route.Method = http.MethodPost
	}
	if !act.IsCommand() {
		panic(fmt.Errorf("x-ant-action %s: Do — только для команд", act.ID))
	}
	Register(a, route, act, func(ctx context.Context, in *I) (*Out[Receipt], error) {
		r, err := fn(ctx, in)
		if err != nil {
			return nil, err
		}
		return ReceiptOf(r), nil
	})
}
