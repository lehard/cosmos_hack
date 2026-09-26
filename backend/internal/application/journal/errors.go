package journal

import (
	"errors"
	"fmt"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// RejectError — отказ Append с параметрами для problem+json: Kind — одна из
// ошибок ErrFenced, ErrStaleState, ErrStalePolicy, ErrConcessionExhausted,
// ErrTimeRegression (errors.Is работает по Kind), Params — подстановки шаблона
// detail из contracts/errors.yaml.
type RejectError struct {
	Kind   error
	Params map[string]string
	// BasisSeq — на каком seq проверено (journal.stale_*).
	BasisSeq int64
}

func (e *RejectError) Error() string {
	return fmt.Sprintf("%v %v", e.Kind, e.Params)
}

func (e *RejectError) Unwrap() error { return e.Kind }

// Reject — отказ kind с параметрами (пары ключ, значение).
func Reject(kind error, basisSeq int64, kv ...string) *RejectError {
	e := &RejectError{Kind: kind, BasisSeq: basisSeq, Params: map[string]string{}}
	for i := 0; i+1 < len(kv); i += 2 {
		e.Params[kv[i]] = kv[i+1]
	}
	return e
}

var codes = []struct {
	err  error
	code errcodes.Code
}{
	{ErrFenced, errcodes.JournalFenced},
	{ErrStaleState, errcodes.JournalStaleState},
	{ErrStalePolicy, errcodes.JournalStalePolicy},
	{ErrConcessionExhausted, errcodes.JournalConcessionExhausted},
	{ErrTimeRegression, errcodes.JournalTimeRegression},
}

// Problem приводит ошибку Append к ошибке ведущего порта с кодом journal.*
// (409 для конкурентности, AD-39): её возвращают сценарии команд модулей.
func Problem(err error) (*platform.Error, bool) {
	for _, c := range codes {
		if !errors.Is(err, c.err) {
			continue
		}
		var kv []string
		var basis int64
		var re *RejectError
		if errors.As(err, &re) {
			basis = re.BasisSeq
			for _, k := range []string{"stream", "basis_seq", "policy_seq", "concession_id", "remaining", "limit", "partition"} {
				if v, ok := re.Params[k]; ok {
					kv = append(kv, k, v)
				}
			}
		}
		pe := platform.Fail(c.code, kv...)
		pe.BasisSeq = basis
		return pe, true
	}
	return nil, false
}
