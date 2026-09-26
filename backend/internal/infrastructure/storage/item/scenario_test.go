package item_test

import (
	"testing"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	"ant/internal/application/item/itemtest"
	dj "ant/internal/domain/journal"
	enginestore "ant/internal/infrastructure/storage/engine"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// Сквозной путь эпика 18 на своей БД (make dev-db): журнал Postgres (эпик 04,
// проверки AD-39 в journal.Append), стадия crossitem.Fold со всеми модулями,
// состояние стадии и реестр носителей — в хранении движка, живые операции
// item и crossitem: результат свидетеля садки в паспортах всех изделий
// садки, блок партии до собранных изделий, неоднозначное событие с
// кандидатами. Без ANT_DB_HOST — пропуск.
func TestScenarioOnPostgres(t *testing.T) {
	p := journaltest.NewDB(t).AppPool(t)
	journal := journalstore.NewStore(p, journaltest.SysClock{})
	eng := &enginestore.Store{Pool: p}
	t.Cleanup(eng.Close)
	codec := &engineapp.Codec{Store: journal, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1",
		DomainBuild: dj.ZeroLink.String(), Partitions: 4}
	itemtest.Scenario(t, itemtest.New(journal, codec, eng, "pg"))
}
