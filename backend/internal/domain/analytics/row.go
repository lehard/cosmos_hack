package analytics

import (
	"slices"
	"time"
)

// Показатели уровня строки вклада (AD-45). Это «сырьё» агрегатов: показатели
// экрана (application/analytics) — суммы, доли и средние этих строк.
const (
	// RowInspectedItems — изделие проверено (есть результат контроля или
	// решение точки предъявления, кроме «оценка невозможна»); 1 на изделие.
	RowInspectedItems = "inspected_items"
	// RowInspections — результат контроля с оценкой (не «оценка невозможна»); 1 на наблюдение.
	RowInspections = "inspections"
	// RowInspectionsWithDefect — результат контроля с признаками дефекта; 1 на наблюдение.
	RowInspectionsWithDefect = "inspections_with_defect"
	// RowUnableToAssess — «оценка невозможна» — отдельная корзина (FR-36, NFR-UI-4).
	RowUnableToAssess = "unable_to_assess"
	// RowDefectsDetected — физический дефект с признаками, не отклонённый
	// (ключ изделие + зона + место в пределах выполнения, без вида, FR-37).
	RowDefectsDetected = "defects_detected"
	// RowConfirmedDefects — физический дефект, подтверждённый несоответствием.
	RowConfirmedDefects = "confirmed_defects"
	// RowItemsWithConfirmedNC — изделие с подтверждённым несоответствием; 1 на изделие.
	RowItemsWithConfirmedNC = "items_with_confirmed_nc"
	// RowNCConfirmed — подтверждённое несоответствие; 1 на несоответствие.
	RowNCConfirmed = "nc_confirmed"
	// RowNCOpen — интервал «несоответствие открыто» (подтверждено или
	// зарегистрировано правилом → закрыто по изделию).
	RowNCOpen = "nc_open"
	// RowInProgress — интервал выполнения операции (начата → завершена или прервана).
	RowInProgress = "in_progress"
	// RowQueue — интервал ожидания изделия в узле (очередь).
	RowQueue = "queue"
	// RowPassed — изделие прошло узел (операция завершена, решение точки
	// предъявления, результат автоматического контроля).
	RowPassed = "passed"
	// RowInterrupted — операция прервана.
	RowInterrupted = "interrupted_operations"
	// RowOperationDuration — длительность выполнения операции, с.
	RowOperationDuration = "operation_duration"
	// RowReworkRuns — повторное выполнение операции (rework_of, FR-47).
	RowReworkRuns = "rework_runs"
	// RowReworkTime — длительность повторного выполнения, с (потери от брака).
	RowReworkTime = "rework_time"
	// RowComparableRuns — завершённое выполнение для сравнения сопоставимых
	// работ: тип операции × тип изделия × исполнитель (оборудование).
	RowComparableRuns = "comparable_runs"
	// RowPresentations — предъявление на точке предъявления.
	RowPresentations = "presentations"
	// RowRepresentations — повторное предъявление (номер > 1).
	RowRepresentations = "representations"
	// RowFPYTotal — изделие с первым вердиктом контроля (Д-11: «Прохождение
	// контроля с первого раза»); Until — когда стало известно «не с первого
	// раза» (нет — прошло с первого раза).
	RowFPYTotal = "fpy_total"
	// RowFPYStepTotal — то же по узлу (изделие × узел).
	RowFPYStepTotal = "fpy_step_total"
	// RowLeadTime — время детали в системе (запуск → сдача на склад), с.
	RowLeadTime = "lead_time"
	// RowDetectionDelay — задержка обнаружения: конец выполнения → первое
	// наблюдение дефекта, с.
	RowDetectionDelay = "detection_delay"
	// RowScrappedItems — изделие списано решением по несоответствию.
	RowScrappedItems = "scrapped_items"
	// RowEquipmentDowntime — интервал простоя оборудования или остановки
	// точки процесса (глобальная проекция, не вклад изделия).
	RowEquipmentDowntime = "equipment_downtime"
)

// Единицы строк: штуки и секунды (целые, AD-4).
const (
	UnitPcs = "pcs"
	UnitSec = "s"
)

// Происхождение и смысл интервала длительности (соглашение «Длительности»).
const (
	OriginSource = "source_reported"
	OriginSystem = "system_computed"

	MeaningActive  = "active_processing"
	MeaningStation = "time_at_station"
	MeaningOther   = "other"
)

// Происхождение дефекта: входной брак отличим от производственного (FR-87, FR-88).
const (
	DefectIncoming   = "incoming"
	DefectProduction = "production"
)

// Вид источника строки для раскрытия (FR-140): source_kind факта; для
// решения человека — ручной ввод; для вывода системы — SourceSystem.
const (
	SourceManual = "manual_entry"
	SourceSystem = "system"
)

// Dims — срез строки вклада: измерения, по которым показатель делится
// (участок, операция, оборудование, исполнитель, вид дефекта, происхождение).
// Пустое поле — измерение неизвестно (FR-123: не додумывать).
type Dims struct {
	// Run — прогон сценария (AD-38); пусто — вне прогона.
	Run string `json:"run,omitempty"`
	// Step — step_key узла процесса.
	Step string `json:"step,omitempty"`
	// Station, Line — участок (пост) и линия.
	Station string `json:"station,omitempty"`
	Line    string `json:"line,omitempty"`
	// Equipment — оборудование выполнения или средство контроля.
	Equipment string `json:"eq,omitempty"`
	// Performer — исполнитель (псевдоним).
	Performer string `json:"perf,omitempty"`
	// Operation — код операции по маршрутной карте.
	Operation string `json:"op,omitempty"`
	// ItemType — тип изделия (номенклатура).
	ItemType string `json:"type,omitempty"`
	// DefectType — вид дефекта по последнему наблюдению.
	DefectType string `json:"dt,omitempty"`
	// Origin — происхождение дефекта или несоответствия: incoming | production.
	Origin string `json:"org,omitempty"`
	// Meaning, DurationOrigin — смысл интервала и происхождение длительности.
	Meaning        string `json:"mean,omitempty"`
	DurationOrigin string `json:"dorg,omitempty"`
	// NC — несоответствие строки.
	NC string `json:"nc,omitempty"`
	// Ref — ключ объекта строки: физический дефект, выполнение, причина.
	Ref string `json:"ref,omitempty"`
	// Label — подпись изделия (номер детали).
	Label string `json:"label,omitempty"`
}

// Row — строка вклада изделия в показатель (AD-45): показатель, момент, срез,
// значение и исходные записи для раскрытия («Соглашения/Показатели»).
type Row struct {
	Metric string
	// Item — изделие строки; функция вклада его не ставит (строки и так
	// принадлежат изделию), заполняет чтение агрегатов.
	Item string
	// At — момент, к которому относится вклад (occurred_at записи-основания):
	// по нему строка попадает в период (FR-3) и в момент «как было» (AD-22).
	At time.Time
	// Interval — строка-состояние [At, Until): очередь, выполнение, открытое
	// несоответствие; Until пусто — состояние длится. У точечной строки
	// «с первого раза» Until — момент провала.
	Interval bool
	Until    *time.Time
	// Value — целое значение в единице Unit (pcs или s), Scale — 0.
	Value int64
	Unit  string
	Dims  Dims
	// Sources — event_id исходных записей; Kinds — виды их источников (FR-140).
	Sources []string
	Kinds   []string
}

// ActiveAt — строка-интервал активна в момент t.
func (r Row) ActiveAt(t time.Time) bool {
	return r.Interval && !r.At.After(t) && (r.Until == nil || r.Until.After(t))
}

// FailedBy — у строки «с первого раза» провал известен к моменту t.
func (r Row) FailedBy(t time.Time) bool { return r.Until != nil && !r.Until.After(t) }

// In — точечная строка попадает в период [from, to].
func (r Row) In(from, to time.Time) bool {
	return !r.At.Before(from) && !r.At.After(to)
}

// Overlap — длительность пересечения интервала строки с [from, to] в
// секундах; незакрытый интервал длится до to.
func (r Row) Overlap(from, to time.Time) int64 {
	end := to
	if r.Until != nil && r.Until.Before(end) {
		end = *r.Until
	}
	start := r.At
	if start.Before(from) {
		start = from
	}
	if !end.After(start) {
		return 0
	}
	return int64(end.Sub(start) / time.Second)
}

// SortRows — канонический порядок строк (AD-4): показатель, момент, ссылка,
// шаг, первая исходная запись.
func SortRows(rows []Row) {
	slices.SortStableFunc(rows, func(a, b Row) int {
		switch {
		case a.Metric != b.Metric:
			return cmpStr(a.Metric, b.Metric)
		case !a.At.Equal(b.At):
			return a.At.Compare(b.At)
		case a.Dims.Ref != b.Dims.Ref:
			return cmpStr(a.Dims.Ref, b.Dims.Ref)
		case a.Dims.Step != b.Dims.Step:
			return cmpStr(a.Dims.Step, b.Dims.Step)
		}
		return cmpStr(first(a.Sources), first(b.Sources))
	})
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func cmpStr(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// addUnique — добавить значения, сохранив порядок и без повторов.
func addUnique(dst []string, vs ...string) []string {
	for _, v := range vs {
		if v != "" && !slices.Contains(dst, v) {
			dst = append(dst, v)
		}
	}
	return dst
}
