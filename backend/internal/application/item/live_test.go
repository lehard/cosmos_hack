package item_test

import (
	"testing"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	"ant/internal/application/item/itemtest"
	dj "ant/internal/domain/journal"
)

// Сквозной путь эпика 18 на фейках (журнал в памяти): садка и свидетель, блок
// партии до собранных изделий, неоднозначное событие с кандидатами и ручная
// привязка — живыми операциями item и crossitem (itemtest.Scenario).
func TestScenarioOnFakes(t *testing.T) {
	j := enginemem.New(nil)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: dj.ZeroLink.String(), Partitions: 4}
	itemtest.Scenario(t, itemtest.New(j, codec, j, "mem"))
}
