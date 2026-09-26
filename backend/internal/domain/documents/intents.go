package documents

import "ant/internal/domain/kernel"

// IntentDraft — имя намерения «черновик документа».
const IntentDraft = "draft"

// DraftContext — контекст черновика документа (AD-12, AD-43): шаблон шага,
// субъект, решение, для которого нужен документ; набор обязательных подписей
// вычисляется один раз при document.version.drafted (access.RequiredApprovals).
type DraftContext struct {
	// Template — шаблон@версия из нормативного слоя (normative/documents).
	Template string
	// Subject — поток субъекта (`item:‹id›`, `nonconformity:‹id›` …).
	Subject string
	// Decision — решение, которое оформляет документ (например, disposition=scrap).
	Decision string
	// Key — ключ документа внутри субъекта (идемпотентность черновика).
	Key string
}

// Draft — функция-намерение documents (AD-40): nonconformity создаёт черновик
// документа «Решение по несоответствию», «Разрешение на отклонение», «Акт о браке».
func Draft(from kernel.Module, c DraftContext, causes ...kernel.Record) kernel.Intent {
	return kernel.NewIntent(Module, from, IntentDraft, c, causes...)
}

// RouteClosed — закрыт ли маршрут подписей документа (AD-43): вычисляется в
// свёртке заново по подписям; модули-исполнители действуют только по нему и
// подписи сами не считают.
func RouteClosed(s State, documentID string) bool {
	_, _ = s, documentID
	return false
}
