package analysis_test

import (
	"testing"

	"ant/internal/application/analysis/analysistest"
	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	dj "ant/internal/domain/journal"
	enginestore "ant/internal/infrastructure/storage/engine"
	journalstore "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/journaltest"
)

// «Станок сломался» на своей БД (make dev-db): журнал Postgres (эпик 04,
// проверки AD-39 в journal.Append), стадия через точку подключения
// crossitem.Fold → analysis.Stage, проекции analysis.* в хранении движка,
// операции analysis в режиме live. Область 34 → 13 → 6 с основаниями,
// устаревший basis_seq — 409 journal.stale_state. Без ANT_DB_HOST — пропуск.
func TestScenarioOnPostgres(t *testing.T) {
	p := journaltest.NewDB(t).AppPool(t)
	journal := journalstore.NewStore(p, journaltest.SysClock{})
	eng := &enginestore.Store{Pool: p}
	t.Cleanup(eng.Close)
	codec := &engineapp.Codec{Store: journal, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1",
		DomainBuild: dj.ZeroLink.String(), Partitions: 4}
	analysistest.Scenario(t, analysistest.New(journal, codec, "pg"), eng)
}
