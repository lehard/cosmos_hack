package quality

import (
	"encoding/json"
	"slices"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality"
)

// Проекции модуля quality (AD-45): один писатель — quality; запись — только
// эффектами в транзакции journal.Append вместе с курсором; пересборка —
// `ant rebuild` (FR-115, FR-124).
const (
	// ItemProjection — выводы quality по изделию: наблюдения, дефекты,
	// сигналы, полнота, ось «состояние качества» (итог свёртки изделия).
	ItemProjection = "quality.item"
	// IndexProjection — перечень изделий с данными контроля по прогону
	// (ключ `items:‹run_id›`) — для списков сигналов, дефектов и пропусков брака.
	IndexProjection = "quality.index"
)

// ItemRecord — значение проекции quality.item.
type ItemRecord struct {
	ItemID   string        `json:"item_id"`
	RunID    string        `json:"run_id,omitempty"`
	BasisSeq int64         `json:"basis_seq"`
	State    quality.State `json:"state"`
}

// IndexRecord — значение проекции quality.index.
type IndexRecord struct {
	Items []string `json:"items"`
}

// IndexKey — ключ перечня изделий прогона.
func IndexKey(runID string) string { return "items:" + runID }

// indexed — типы записей изделия, после которых у изделия есть данные контроля.
var indexed = []catalog.Type{
	catalog.InspectionResultRecorded, catalog.OperatorCheckSkipped, catalog.OperatorDeviationReported,
	catalog.ItemPresentationRecorded, catalog.DecisionPresentationResolved,
}

// Register регистрирует проекции и вклады показателей модуля quality в
// реестре движка (одна строка в cmd/ant engineRegistry).
func Register(reg *engineapp.Registry) error {
	if err := reg.AddItem(engineapp.ItemProjection{Name: ItemProjection, Writer: quality.Module, View: itemView}); err != nil {
		return err
	}
	if err := reg.AddGlobal(engineapp.GlobalProjection{Name: IndexProjection, Writer: quality.Module,
		Keys: indexKeys, Step: indexStep, Entity: func(string) (platform.EntityKind, string, bool) { return "", "", false }}); err != nil {
		return err
	}
	reg.AddContributor(Contributions)
	return nil
}

func itemView(itemID string, s engine.Snapshot, _ []kernel.Reaction) (any, error) {
	q := s.Quality
	if q.Pos == 0 || (len(q.Observations) == 0 && len(q.Skips) == 0 && len(q.Presentations) == 0 && len(q.Reports) == 0) {
		return nil, nil
	}
	return ItemRecord{ItemID: itemID, RunID: q.RunID, BasisSeq: s.BasisSeq, State: q}, nil
}

func indexKeys(r kernel.Record) []string {
	if r.ItemID == "" || !slices.Contains(indexed, r.Type) {
		return nil
	}
	return []string{IndexKey(r.RunID)}
}

func indexStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var ix IndexRecord
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &ix); err != nil {
			return nil, err
		}
	}
	if i, found := slices.BinarySearch(ix.Items, r.ItemID); !found {
		ix.Items = slices.Insert(ix.Items, i, r.ItemID)
	}
	return json.Marshal(ix)
}

// Показатели качества (AD-45: строки вклада изделия заменяются целиком при
// пересвёртке; агрегат — сумма вкладов). Физический дефект считается один раз
// при любом числе наблюдений (FR-37); дефекты и изделия с дефектами — раздельно.
const (
	MetricDefects         = "quality.defects"
	MetricItemsWithDefect = "quality.items_with_defect"
	MetricObservations    = "quality.observations"
	MetricSignals         = "quality.signals"
	MetricUnable          = "quality.unable_to_assess"
	MetricMissing         = "quality.inspection_missing"
	MetricEscapes         = "quality.escapes"
)

// Contributions — строки вклада изделия в показатели качества.
func Contributions(itemID string, s engine.Snapshot) ([]engineapp.Contribution, error) {
	q := s.Quality
	rows := []engineapp.Contribution{}
	add := func(metric, slice string, v int64, sources []string) {
		if v == 0 {
			return
		}
		src := slices.Clone(sources)
		slices.Sort(src)
		rows = append(rows, engineapp.Contribution{ItemID: itemID, Metric: metric, Slice: slice, Value: v, Sources: slices.Compact(src)})
	}
	var defectSrc, obsSrc, sigSrc, unableSrc, missSrc, escSrc []string
	for _, d := range q.Defects {
		defectSrc = append(defectSrc, d.FirstObservation)
	}
	for _, o := range q.Observations {
		if !o.Superseded {
			obsSrc = append(obsSrc, o.EventID)
		}
	}
	for _, sg := range q.Signals {
		if !sg.Raised {
			continue
		}
		if sg.Unable {
			unableSrc = append(unableSrc, sg.Causes...)
		} else {
			sigSrc = append(sigSrc, sg.Causes...)
		}
	}
	nUnable, nSig := 0, 0
	for _, sg := range q.Signals {
		if sg.Raised && sg.Unable {
			nUnable++
		} else if sg.Raised {
			nSig++
		}
	}
	nMiss := 0
	for _, p := range q.Points {
		if p.Status == "missing" && p.Required {
			nMiss++
			missSrc = append(missSrc, p.Causes...)
		}
	}
	for _, e := range q.Escapes {
		escSrc = append(escSrc, e.Causes...)
	}
	add(MetricDefects, "all", int64(len(q.Defects)), defectSrc)
	if len(q.Defects) > 0 {
		add(MetricItemsWithDefect, "all", 1, defectSrc)
	}
	add(MetricObservations, "all", int64(len(obsSrc)), obsSrc)
	add(MetricSignals, "all", int64(nSig), sigSrc)
	add(MetricUnable, "all", int64(nUnable), unableSrc)
	add(MetricMissing, "all", int64(nMiss), missSrc)
	add(MetricEscapes, "all", int64(len(q.Escapes)), escSrc)
	return rows, nil
}
