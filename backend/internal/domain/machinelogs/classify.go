package machinelogs

import "ant/internal/contracts/catalog"

// Классификация наблюдений оборудования как в MTConnect (AD-29, FR-147):
// измерение (SAMPLE) — непрерывная величина; событие (EVENT) — смена
// дискретного состояния (режим работы, режим управления, программа и её
// ревизия, инструмент и его ресурс, коррекция режима); условие (CONDITION) —
// исправность: норма, предупреждение, неисправность. Сырые измерения в
// журнал не попадают: edge-агент сводит их в сводку на окно цикла
// (equipment.cycle.summarized), поэтому SAMPLE в журнале — это сводка.

// Class — вид наблюдения по MTConnect.
type Class string

// Виды наблюдений.
const (
	ClassSample    Class = "sample"
	ClassEvent     Class = "event"
	ClassCondition Class = "condition"
)

// Layer — слой журнала оборудования (FR-147): что делал станок, чем, как шёл
// процесс, отклонения.
type Layer string

// Слои.
const (
	LayerWhat      Layer = "what"
	LayerWithWhat  Layer = "with_what"
	LayerHow       Layer = "how"
	LayerDeviation Layer = "deviation"
)

// Категории телеметрии stand-а (contracts/internal/stands/telemetry.v1.json)
// — словарь MTConnect на краю.
const (
	CategoryExecution      = "execution"
	CategoryControllerMode = "controller_mode"
	CategoryCondition      = "condition"
	CategoryProgram        = "program"
	CategoryTool           = "tool"
	CategoryCycle          = "cycle"
	CategoryAlarm          = "alarm"
	CategoryOverride       = "override"
)

// CategoryClass — вид наблюдения MTConnect для категории телеметрии: условие
// исправности и авария — CONDITION, остальное — EVENT. Отсчёты (samples) —
// всегда SAMPLE. Неизвестная категория — ok=false: выделитель её пропускает.
func CategoryClass(category string) (Class, bool) {
	switch category {
	case CategoryCondition, CategoryAlarm:
		return ClassCondition, true
	case CategoryExecution, CategoryControllerMode, CategoryProgram, CategoryTool, CategoryCycle, CategoryOverride:
		return ClassEvent, true
	}
	return "", false
}

// Classify — вид наблюдения и слой для события оборудования (один тип на
// смысл от любого источника, AD-29). Смена состояния с исправностью «не
// норма» — CONDITION, иначе EVENT; сводка цикла — SAMPLE («как шёл
// процесс»); выделенное отклонение — слой «отклонения».
func Classify(ev Event) (Class, Layer) {
	switch ev.Type {
	case catalog.EquipmentStateChanged:
		if ev.Condition != "" && ev.Condition != ConditionNormal {
			return ClassCondition, LayerWhat
		}
		return ClassEvent, LayerWhat
	case catalog.EquipmentProgramChanged:
		if ev.Planned != nil && !*ev.Planned {
			return ClassEvent, LayerDeviation
		}
		return ClassEvent, LayerWhat
	case catalog.EquipmentToolChanged:
		return ClassEvent, LayerWithWhat
	case catalog.EquipmentCycleSummarized:
		return ClassSample, LayerHow
	case catalog.EquipmentDeviationDetected:
		if ev.DeviationKind == DeviationAlarm {
			return ClassCondition, LayerDeviation
		}
		return ClassEvent, LayerDeviation
	}
	return ClassEvent, LayerWhat
}

// Значения исправности (MTConnect CONDITION) и режимов из контракта
// equipment.state.changed v2.
const (
	ConditionNormal  = "normal"
	ConditionWarning = "warning"
	ConditionFault   = "fault"
	ConditionUnknown = "unknown"

	ExecutionRunning = "running"
	ModeAutomatic    = "automatic"
	ModeManual       = "manual"
	ModeUnknown      = "unknown"
)

// Виды отклонений (equipment.deviation.detected).
const (
	DeviationOutOfSetpoint  = "out_of_setpoint"
	DeviationOverload       = "overload"
	DeviationToolLife       = "tool_life_warning"
	DeviationManualOverride = "manual_override"
	DeviationProgramChange  = "unplanned_program_change"
	DeviationAlarm          = "alarm"
	DeviationOther          = "other"
)

// ViolatesRegime — отклонение нарушает режим специального процесса (FR-151):
// параметр вне уставки (FR-121; карта реакций, правило R-12). Ручная
// коррекция режима, перегрузка и предупреждения — отклонения профиля
// выполнения, но окна нарушения не образуют.
func ViolatesRegime(kind string) bool { return kind == DeviationOutOfSetpoint }

// SourceKind — пометка источника факта в словаре контракта (FR-140, AD-2):
// ручной ввод, станок, датчик, камера, внешняя система, импорт. Один тип
// события принимается от любого из них — различается только пометка;
// неизвестное значение — пусто (FR-123: не додумывать).
func SourceKind(raw string) string {
	switch raw {
	case "manual_entry", "machine", "sensor", "camera", "external_system", "import":
		return raw
	}
	return ""
}
