package platform

import (
	"errors"
	"fmt"

	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Error — ошибка ведущего порта с кодом из contracts/errors.yaml; transport
// превращает её в problem+json RFC 9457 (FR-28). Доменный отказ kernel.Refusal
// превращается так же.
type Error struct {
	Code   errcodes.Code
	Params map[string]string
	Detail string
	// AllowedActions — что пользователь может сделать вместо (FR-146).
	AllowedActions []string
	// BasisSeq — для journal.stale_*: на каком seq проверено.
	BasisSeq int64
	// CARef — критическое действие, записанное из-за отказа.
	CARef string
	cause error
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return string(e.Code) + ": " + e.Detail
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error { return e.cause }

// Fail — ошибка с кодом и параметрами шаблона (пары ключ, значение).
func Fail(code errcodes.Code, kv ...string) *Error {
	e := &Error{Code: code}
	if len(kv) > 1 {
		e.Params = make(map[string]string, len(kv)/2)
		for i := 0; i+1 < len(kv); i += 2 {
			e.Params[kv[i]] = kv[i+1]
		}
	}
	return e
}

// NotImplemented — операция объявлена в контракте, реализация в работе (501,
// api.not_implemented). Её возвращают заглушки ведущих портов волны 1.
func NotImplemented(operationID string) *Error {
	e := Fail(errcodes.ApiNotImplemented, "operation_id", operationID)
	e.Detail = fmt.Sprintf("Операция %s объявлена в контракте, реализация — в работе", operationID)
	return e
}

// AsError приводит ошибку порта к *Error: доменный отказ — его кодом, прочее — nil.
func AsError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	var r *kernel.Refusal
	if errors.As(err, &r) {
		return &Error{Code: r.Code, Params: r.Params, Detail: r.Detail, cause: err}, true
	}
	return nil, false
}
