package analytics

import (
	"time"

	"ant/internal/application/platform"
)

// Формы ответов аналитики (FR-3, FR-5, FR-86…FR-89; AD-21, AD-45; кейс §2.4,
// §5.2). Показатели — только проекции из строк вклада изделия; каждое число
// раскрывается до исходных записей (соглашение «Показатели»). Экраны — эпики 10, 15.

// Period — период показателей и счётчиков (FR-3): смена, сутки, неделя, месяц,
// произвольный (from, to).
type Period struct {
	Kind string    `json:"kind" enum:"shift,day,week,month,custom"`
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// MetricValue — значение показателя с происхождением: целое + масштаб
// (без float, AD-4) и единица; доли — в базисных пунктах.
type MetricValue struct {
	Value int64  `json:"value" doc:"Значение = value × 10^(−scale)."`
	Scale int    `json:"scale" minimum:"0" maximum:"9"`
	Unit  string `json:"unit" doc:"Единица: pcs, bp (доли), s, min…"`
	// Origin — происхождение времени (соглашение «Длительности»).
	Origin *string `json:"origin,omitempty" enum:"reported_by_source,computed_by_system,mixed" doc:"Для длительностей: передано источником / вычислено системой."`
	// Meaning — смысл интервала длительности (соглашение «Длительности», FR-88).
	Meaning *string `json:"meaning,omitempty" enum:"active_processing,time_at_station,other" doc:"Для длительностей: смысл интервала — активная обработка / полное время на участке / иной интервал (пояснение — meaning_note)."`
	// MeaningNote — что за интервал, если смысл «иной» или смыслы строк разные.
	MeaningNote *string `json:"meaning_note,omitempty" maxLength:"500" doc:"Пояснение смысла интервала: от чего до чего считается время."`
}

// MetricTile — плитка показателя стола руководителя (PRD §3a).
type MetricTile struct {
	MetricID string      `json:"metric_id" doc:"Идентификатор показателя: inspected_items, items_with_confirmed_nc, first_pass_yield, defects_by_type, cause_established, lead_time, waiting_time…"`
	Title    string      `json:"title" doc:"Название по словарю продукта («Прохождение контроля с первого раза», Д-11)."`
	Value    MetricValue `json:"value"`
	// Unknown — оценка невозможна (нет данных) ≠ ноль (NFR-UI-4).
	Unknown  bool               `json:"unknown"`
	Previous *MetricValue       `json:"previous,omitempty" doc:"Значение за предыдущий такой же период."`
	Drill    *platform.DrillRef `json:"drill,omitempty" doc:"Куда провалиться (FR-7)."`
}

// MetricTileList — плитки показателей.
type MetricTileList struct {
	Period Period       `json:"period"`
	Items  []MetricTile `json:"items"`
}

// NodeCounters — счётчики узла по step_key (FR-2, FR-3): очередь, в работе,
// прошло, дефекты за период (физические дефекты, не наблюдения — соглашение «Дефект»).
type NodeCounters struct {
	StepKey         string  `json:"step_key"`
	StepName        *string `json:"step_name,omitempty" doc:"Имя узла BPMN версии процесса; нет — показывать step_key."`
	Queue           int     `json:"queue" minimum:"0"`
	InProgress      int     `json:"in_progress" minimum:"0"`
	Passed          int     `json:"passed" minimum:"0"`
	Defects         int     `json:"defects" minimum:"0"`
	Nonconformities *int    `json:"nonconformities,omitempty" minimum:"0" doc:"Открытые несоответствия узла (FR-154)."`
}

// NodeAnomaly — аномалия узла (FR-5).
type NodeAnomaly struct {
	StepKey   string  `json:"step_key"`
	StepName  *string `json:"step_name,omitempty" doc:"Имя узла BPMN версии процесса; нет — показывать step_key."`
	Kind      string  `json:"kind" enum:"queue_above_norm,wait_above_norm,downtime_over_threshold,output_spike,defect_rate_out_of_control"`
	Threshold *string `json:"threshold,omitempty" doc:"Порог текстом с единицей."`
}

// Bottleneck — узел-ограничение линии (FR-5): наибольшее ожидание при наибольшей загрузке.
type Bottleneck struct {
	StepKey  string  `json:"step_key"`
	StepName *string `json:"step_name,omitempty" doc:"Имя узла BPMN версии процесса; нет — показывать step_key."`
	Wait     *string `json:"wait,omitempty" doc:"Среднее ожидание текстом с единицей («37 мин»)."`
}

// NodeCounterSet — счётчики узлов, ограничение линии и аномалии: считает сервер (AD-21).
type NodeCounterSet struct {
	Period           Period         `json:"period"`
	ProcessVersionID string         `json:"process_version_id"`
	Counters         []NodeCounters `json:"counters"`
	Bottleneck       *Bottleneck    `json:"bottleneck,omitempty"`
	Anomalies        []NodeAnomaly  `json:"anomalies"`
	DataGaps         []string       `json:"data_gaps" doc:"step_key узлов, где оценка невозможна (нет данных источника) — не «норма»."`
	BasisSeq         int64          `json:"basis_seq"`
	// StepNames — имена узлов ответа (UI-21): data_gaps — строки step_key.
	StepNames map[string]string `json:"step_names,omitempty" doc:"step_key → имя узла BPMN для всех узлов ответа (в том числе data_gaps); нет имени — ключа нет."`
}

// MetricRow — показатель раздела «Аналитика» с разбивкой по срезу.
type MetricRow struct {
	MetricID string `json:"metric_id"`
	Title    string `json:"title"`
	Group    string `json:"group" enum:"inspection,defects,causes,time,equipment,people,comparison" doc:"Раздел: раздельный учёт кейса §2.4, §5.2."`
	// Counts — что считается (единица счёта), чтобы число не читалось шире
	// своего смысла (NFR-UI-4): дефекты и изделия с дефектами — раздельно.
	Counts *string `json:"counts,omitempty" enum:"items,defects,nonconformities,operations,observations,presentations,hypotheses,time" doc:"Что считается: изделия / физические дефекты / несоответствия / выполнения операций / наблюдения (результаты контроля) / предъявления / гипотезы / время. Для долей — чья это доля."`
	// Account — графа раздельного учёта (FR-87): входной брак, проблемы
	// оборудования, ошибки исполнителей, гипотезы — не складываются.
	Account *string       `json:"account,omitempty" enum:"incoming,equipment,performer,hypotheses" doc:"Графа раздельного учёта FR-87: входной брак / оборудование / исполнители / гипотезы; нет — показатель вне раздельного учёта."`
	Total   MetricValue   `json:"total"`
	Unknown bool          `json:"unknown"`
	Slices  []MetricSlice `json:"slices"`
}

// MetricSlice — значение показателя в срезе (участок, операция, оборудование,
// исполнитель, смена, вид дефекта, источник: входной брак / производственные ошибки).
type MetricSlice struct {
	Dimension string      `json:"dimension" enum:"location,step,equipment,performer,shift,defect_type,origin" doc:"Измерение среза; origin — откуда брак: входной / производственный или категория подтверждённой причины (входной брак, оборудование, исполнитель…)."`
	Key       string      `json:"key"`
	Label     string      `json:"label"`
	Value     MetricValue `json:"value"`
}

// AnalyticsOverview — полный набор показателей кейса (FR-86…FR-89): раздельный
// учёт, дефекты и изделия с дефектами раздельно, сравнение сопоставимых работ,
// происхождение времени.
type AnalyticsOverview struct {
	Period   Period      `json:"period"`
	Items    []MetricRow `json:"items"`
	BasisSeq int64       `json:"basis_seq"`
}

// ContributionRow — строка вклада изделия в показатель (AD-45) с исходными записями.
type ContributionRow struct {
	ItemID         string      `json:"item_id" doc:"Изделие; для строк вне изделия (простой оборудования) — id объекта из ref."`
	Label          string      `json:"label"`
	SliceKey       string      `json:"slice_key"`
	Value          MetricValue `json:"value"`
	SourceEventIDs []string    `json:"source_event_ids" doc:"id исходных записей журнала (раскрытие до записей)."`
	SourceKinds    []string    `json:"source_kinds" doc:"Виды источника исходных записей (FR-140), а не виды записей: manual_entry, machine, sensor, camera, external_system, import — для фактов; manual_entry — решение человека; system — вывод системы."`
	// At — момент вклада (occurred_at записи-основания), по нему строка попадает в период.
	At *time.Time `json:"at,omitempty" doc:"Момент, к которому относится вклад (FR-3)."`
	// Ref — объект строки, если это не изделие (оборудование, несоответствие).
	Ref *platform.DrillRef `json:"ref,omitempty" doc:"Объект строки вне изделия — куда провалиться (FR-7)."`
}

// MetricDrilldown — раскрытие числа до строк вклада и исходных записей.
type MetricDrilldown struct {
	MetricID   string            `json:"metric_id"`
	Period     Period            `json:"period"`
	Total      MetricValue       `json:"total"`
	Items      []ContributionRow `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// ControlChartPoint — точка контрольной карты.
type ControlChartPoint struct {
	At           time.Time          `json:"at"`
	Value        MetricValue        `json:"value"`
	OutOfControl bool               `json:"out_of_control" doc:"Выход за контрольные границы или неслучайная структура."`
	Ref          *platform.DrillRef `json:"ref,omitempty"`
}

// ControlChart — контрольная карта по узлу (FR-5): доля дефектов или
// характеристика во времени, центральная линия и границы.
type ControlChart struct {
	StepKey string `json:"step_key"`
	// StepName — имя узла BPMN действующей версии (UI-21); нет — показывать step_key.
	StepName *string `json:"step_name,omitempty" doc:"Имя узла BPMN действующей версии процесса; нет — показывать step_key."`
	MetricID string  `json:"metric_id"`
	// Title — название показателя карты по словарю продукта.
	Title string `json:"title" doc:"Название показателя карты («Доля результатов контроля с признаком дефекта», «Длительность операции»)."`
	// ChartKind — вид карты Шухарта (ГОСТ Р ИСО 7870-2).
	ChartKind *string             `json:"chart_kind,omitempty" enum:"p,xmr" doc:"Вид карты: p — доля дефектных по подгруппам; xmr — индивидуальные значения и скользящий размах."`
	Center    MetricValue         `json:"center"`
	Upper     MetricValue         `json:"upper"`
	Lower     MetricValue         `json:"lower"`
	Points    []ControlChartPoint `json:"points"`
}

// PeriodQuery — параметры периода для порта.
type PeriodQuery struct {
	Kind string
	From *time.Time
	To   *time.Time
}
