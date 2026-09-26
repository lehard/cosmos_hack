package journal

import "context"

// Проверка политики команды (AD-39, AD-15): «политика субъекта не менялась
// после policy_seq». Решение о доступе принимает общий декоратор (access.Gate)
// до вызова модуля по своей версии политики; модули пишут в журнал своими
// писателями и о политике не знают. Поэтому декоратор кладёт проверки
// политики в контекст команды, а Append каждого адаптера журнала добавляет их
// к Checks пачки — в той же транзакции, после блокировки головы: между
// решением о доступе и записью отзыв роли не вклинится ни на одной копии api
// (ответ — 409 journal.stale_policy, запись CA не делается).

type policyChecksKey struct{}

// WithPolicyChecks — контекст команды с проверками политики субъекта
// (Check.PolicyStream, Check.PolicySeq). Пустой список — контекст без изменений.
func WithPolicyChecks(ctx context.Context, checks ...Check) context.Context {
	if len(checks) == 0 {
		return ctx
	}
	prev := PolicyChecksFrom(ctx)
	all := make([]Check, 0, len(prev)+len(checks))
	all = append(append(all, prev...), checks...)
	return context.WithValue(ctx, policyChecksKey{}, all)
}

// PolicyChecksFrom — проверки политики команды из контекста; нет — nil.
func PolicyChecksFrom(ctx context.Context) []Check {
	c, _ := ctx.Value(policyChecksKey{}).([]Check)
	return c
}

// WithContextChecks — запрос записи с проверками политики из контекста
// команды (AD-39): вызывают адаптеры JournalStore в начале Append. Пачка без
// записей (только курсор потребителя) не проверяется.
func WithContextChecks(ctx context.Context, rq AppendRequest) AppendRequest {
	pc := PolicyChecksFrom(ctx)
	if len(pc) == 0 || (len(rq.Batch) == 0 && len(rq.Critical) == 0) {
		return rq
	}
	out := rq
	out.Checks = append(append(make([]Check, 0, len(rq.Checks)+len(pc)), rq.Checks...), pc...)
	return out
}
