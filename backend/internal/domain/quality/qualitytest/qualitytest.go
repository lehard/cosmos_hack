// Пакет qualitytest — тестовая опора модуля quality для тестов движка и
// заготовок: реакция quality.signal.raised, построенная функцией модуля-
// эмитента (quality.SignalRaised, AD-40). Настоящие правила качества — в
// domain/quality (Reduce, React, Assess); опора лишь даёт тестам движка
// (domain/engine/enginetest, diff_test) построить реакцию в произвольном слоте.
// В сборку ролей не входит.
//
// Слой: domain (AD-4) — чистые функции.
package qualitytest

import (
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality"
)

// SignalRaised — реакция quality.signal.raised в слоте slot с data и причинами.
func SignalRaised(slot kernel.Slot, data any, causes ...kernel.Record) (kernel.Reaction, error) {
	return quality.SignalRaised(slot, data, causes...)
}
