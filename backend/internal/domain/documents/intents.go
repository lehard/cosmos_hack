package documents

import "ant/internal/domain/kernel"

// IntentDraft — имя намерения «черновик документа».
const IntentDraft = "draft"

// DraftContext — контекст черновика документа (AD-12, AD-43): шаблон шага,
// субъект, решение, для которого нужен документ; набор обязательных подписей
// вычисляется один раз при document.version.drafted (access.RequiredApprovals).
type DraftContext struct {
	// Template — шаблон@версия из нормативного слоя (normative/documents).
	Template string `json:"template"`
	// Subject — поток субъекта (`item:‹id›`, `nonconformity:‹id›` …).
	Subject string `json:"subject,omitempty"`
	// Decision — решение, которое оформляет документ (например, disposition=scrap).
	Decision string `json:"decision,omitempty"`
	// Key — id документа (идемпотентность черновика).
	Key string `json:"key,omitempty"`
	// Comment — комментарий автора запроса.
	Comment string `json:"comment,omitempty"`
	// Sources — записи-основания.
	Sources []string `json:"sources,omitempty"`
	// Author — кто запросил (псевдоним сотрудника).
	Author string `json:"author,omitempty"`
}

// Draft — функция-намерение documents (AD-40): поздний модуль просит
// оформить документ по шаблону. Документ оформляется на следующей записи
// изделия; если он нужен в том же шаге, модуль кладёт document_id в своё
// решение, и documents оформляет его по решению (decision.disposition.set).
func Draft(from kernel.Module, c DraftContext, causes ...kernel.Record) kernel.Intent {
	return kernel.NewIntent(Module, from, IntentDraft, c, causes...)
}

// RouteClosed — закрыт ли маршрут подписей текущей версии документа (AD-43):
// вычисляется в свёртке заново по подписям (Evaluate), записанная реакция
// document.route.closed служит для сравнения и потоков вне изделия (AD-3).
// Модули-исполнители (nonconformity: решения режима 4, AD-40) действуют
// только по нему и подписи сами не считают.
func RouteClosed(s State, documentID string) bool {
	d := s.Doc(documentID)
	if d == nil {
		return false
	}
	v := d.Current()
	return v != nil && v.Closed && !v.Annulled
}

// FoldStream — документ вне изделия (AD-12: политика, ключи, версия
// процесса, смена, партнёр): записи потока `document:‹id›` по порядку той же
// функцией Reduce, что и в свёртке изделия. Им api проверяет подписи и
// закрывает маршрут реакцией, верификатор — перепроверяет.
func FoldStream(input []kernel.Record, env Env) State {
	var s State
	for _, r := range input {
		s = Reduce(s, r, env, Upstream{})
	}
	return s
}
