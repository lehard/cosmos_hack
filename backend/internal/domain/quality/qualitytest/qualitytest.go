// Пакет qualitytest — тестовая опора модуля quality: реакция
// quality.signal.raised, построенная от имени её эмитента (AD-40), для
// пустышки движка (domain/engine/enginetest) и тестов сравнения реакций,
// пока настоящих правил качества нет (волна 4). В сборку ролей не входит.
//
// Слой: domain (AD-4) — чистые функции.
package qualitytest

import (
	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality"
)

// SignalRaised — реакция quality.signal.raised в слоте slot с data и причинами.
func SignalRaised(slot kernel.Slot, data any, causes ...kernel.Record) (kernel.Reaction, error) {
	return kernel.NewReaction(quality.Module, catalog.QualitySignalRaised, slot, data, causes...)
}
