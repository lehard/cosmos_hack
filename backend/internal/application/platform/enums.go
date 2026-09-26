package platform

// Enum — именованное строковое перечисление слоя application; transport
// публикует его отдельной схемой в components.schemas (имя — имя типа или
// SchemaName()), так клиент фронтенда получает именованный тип.
type Enum interface{ Values() []string }

// Values — значения оси момента.
func (Axis) Values() []string { return []string{string(AxisOccurred), string(AxisRecorded)} }

// Describe — описание для спецификации.
func (Axis) Describe() string {
	return "Ось момента (AD-37): occurred — «как было» (по умолчанию), recorded — «что мы знали»."
}

// Values — режимы ведущих портов.
func (Mode) Values() []string { return []string{string(ModeFixtures), string(ModeLive)} }

// SchemaName — имя схемы в спецификации.
func (Mode) SchemaName() string { return "BackendMode" }

// Describe — описание для спецификации.
func (Mode) Describe() string {
	return "Режим ведущих портов, отдавших ответ (AD-36, FR-150)."
}

// Values — классы операций.
func (Class) Values() []string {
	out := make([]string, len(Classes))
	for i, c := range Classes {
		out[i] = string(c)
	}
	return out
}

// SchemaName — имя схемы в спецификации.
func (Class) SchemaName() string { return "ActionClass" }

// Describe — описание для спецификации.
func (Class) Describe() string {
	return "Класс операции из x-ant-action (AD-27, AD-40): read — чтение, record — запись без последствий на осях, protective — защитное, permissive — разрешающее, irreversible — необратимое."
}

// EntityKind — вид сущности: первый элемент ключа Vue Query, объект прав и
// сообщения SSE (contracts/events/common/sse-entity-changed.v1.json).
type EntityKind string

// Виды сущностей.
const (
	EntityItem             EntityKind = "item"
	EntityLot              EntityKind = "lot"
	EntityNonconformity    EntityKind = "nonconformity"
	EntityIncident         EntityKind = "incident"
	EntityDocument         EntityKind = "document"
	EntityTask             EntityKind = "task"
	EntityNotification     EntityKind = "notification"
	EntityWorkplace        EntityKind = "workplace"
	EntityEquipment        EntityKind = "equipment"
	EntityProcessVersion   EntityKind = "process_version"
	EntityAnalyzerPassport EntityKind = "analyzer_passport"
	EntityErpMessage       EntityKind = "erp_message"
	EntityRun              EntityKind = "run"
	EntityIntegrity        EntityKind = "integrity"
	EntityLiveMap          EntityKind = "live_map"
	EntityPolicy           EntityKind = "policy"
	EntityQuarantine       EntityKind = "quarantine"
)

// EntityKinds — все виды в порядке контракта.
var EntityKinds = []EntityKind{
	EntityItem, EntityLot, EntityNonconformity, EntityIncident, EntityDocument, EntityTask,
	EntityNotification, EntityWorkplace, EntityEquipment, EntityProcessVersion, EntityAnalyzerPassport,
	EntityErpMessage, EntityRun, EntityIntegrity, EntityLiveMap, EntityPolicy, EntityQuarantine,
}

// Values — виды сущностей.
func (EntityKind) Values() []string {
	out := make([]string, len(EntityKinds))
	for i, k := range EntityKinds {
		out[i] = string(k)
	}
	return out
}

// Describe — описание для спецификации.
func (EntityKind) Describe() string {
	return "Вид сущности — первый элемент ключа Vue Query и объект прав (как в contracts/events/common/sse-entity-changed.v1.json)."
}
