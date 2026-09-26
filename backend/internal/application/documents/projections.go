package documents

import (
	engineapp "ant/internal/application/engine"
	dom "ant/internal/domain/documents"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// ItemProjection — проекция изделия модуля documents (AD-45, один писатель):
// документы изделия со статусами и отпечатками и счётчик FR-65 «документов
// собрано из истории — вручную не понадобилось» (AD-12). Пишет воркер
// эффектами в транзакции Append; экраны модуля читают свёртку (AD-22).
const ItemProjection = "documents.item"

// ItemSummary — значение проекции documents.item.
type ItemSummary struct {
	ItemID string `json:"item_id"`
	// CollectedFromHistory — документов собрано из истории изделия.
	CollectedFromHistory int `json:"collected_from_history"`
	// ManualEntries — полей, заполненных вручную: у проекций журнала — 0.
	ManualEntries int          `json:"manual_entries"`
	Documents     []dom.DocRef `json:"documents"`
}

// ItemView — чистая функция итога свёртки изделия → значение проекции (nil —
// документов у изделия нет).
func ItemView(itemID string, s engine.Snapshot, _ []kernel.Reaction) (any, error) {
	refs := s.Documents.Refs()
	if len(refs) == 0 {
		return nil, nil
	}
	return ItemSummary{ItemID: itemID, CollectedFromHistory: len(refs), Documents: refs}, nil
}

// RegisterProjections регистрирует проекции модуля в реестре движка
// (cmd/ant engineRegistry, одна строка на модуль).
func RegisterProjections(r *engineapp.Registry) error {
	return r.AddItem(engineapp.ItemProjection{Name: ItemProjection, Writer: dom.Module, View: ItemView})
}
