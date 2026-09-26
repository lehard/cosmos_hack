package vision

import (
	"context"
	"time"

	dom "ant/internal/domain/vision"
)

// Ведомые порты модуля vision (AD-18, AD-35): сигналы внешних систем
// видеофиксации приходят через адаптер на краю и уходят в ant обычным приёмом
// (edge-агент → ingest), а не в обход журнала. Адаптер переводит протокол
// внешней системы в наблюдения на нашем языке; Relay делает из наблюдений
// события контракта. Замена stand-а реальной системой — смена адреса;
// замена системы другой — новый SignalAdapter в cmd/edge-agent; ядро не меняется
// (FR-97, FR-126, NFR-TEST-2).

// SignalAdapter — порт сигналов видеофиксации: адаптер внешней системы
// (VisionQC, OperatorVision) переводит её сообщение в наблюдения. Внешний
// формат дальше адаптера не выходит. Ошибка — сообщение не по протоколу:
// адаптер отвечает системе отказом, в ant ничего не уходит.
type SignalAdapter interface {
	// System — имя системы в пути локального входа edge-агента: /v1/vision/‹system›.
	System() string
	// Translate — наблюдения из одного сообщения системы.
	Translate(raw []byte) (Signals, error)
}

// Signals — наблюдения из сообщения внешней системы.
type Signals struct {
	// System — система-источник (для event_id и source_note).
	System      string
	Inspections []Inspection
	Actions     []OperatorAction
}

// ItemRef — ссылка на изделие, которую система получила от линии (PartId
// OPC UA MV): внутренний ID или значение носителя. Привязку делает приём (AD-41).
type ItemRef struct {
	CarrierType         string
	Value               string
	IdentificationLevel string
}

// Stage — ступень анализатора с версией и уверенностью (FR-38).
type Stage struct {
	Name         string
	Version      string
	ConfidenceBP *int
	Output       string
}

// Finding — признак дефекта.
type Finding struct {
	DefectCode   string
	Zone         string
	Location     string
	Severity     string
	ConfidenceBP *int
	Note         string
}

// Inspection — наблюдение VisionQC на нашем языке (FR-38, FR-97, AD-29):
// состояние обработки отдельно от вывода, уверенность отдельно от качества
// наблюдения, ступени, вектор версий.
type Inspection struct {
	// ObservationID — идентификатор результата у системы (ResultId): из него
	// детерминированный event_id — повтор системы приём узнает как дубль (AD-7).
	ObservationID string
	OccurredAt    time.Time
	RunID         string
	Item          *ItemRef
	EquipmentID   string
	StepKey       string
	Point         string
	Phase         string
	// ProcessingState — completed | aborted | failed; Outcome — исход источника.
	ProcessingState string
	Outcome         string
	UnableReason    string
	ConfidenceBP    *int
	QualityBP       *int
	Stages          []Stage
	Findings        []Finding
	Zones           []string
	Recommendation  string
	Limitations     []string
	Versions        dom.Versions
	// Simulated — результат получен системой в режиме симуляции: пометка в
	// ограничениях (показатели его не учитывают — задача потребителей).
	Simulated bool
}

// OperatorAction — гипотеза OperatorVision о действии у рабочего места
// (FR-126). Сведений о человеке нет и быть не может: исполнитель — по входу
// на рабочее место (domain/vision.ExecutorAt).
type OperatorAction struct {
	HypothesisID string
	OccurredAt   time.Time
	RunID        string
	Item         *ItemRef
	EquipmentID  string
	WorkplaceID  string
	TPStep       string
	Action       string
	ConfidenceBP *int
	QualityBP    *int
	Versions     dom.Versions
	Simulated    bool
}

// IllustrationFiles — файлы открытого набора иллюстраций на краю (FR-102):
// образец класса. Нет файла — ошибка; наблюдение идёт без иллюстрации.
type IllustrationFiles interface {
	Sample(datasetID, class, sample string) (content []byte, err error)
}

// MaterialSink — хранилище материалов по адресу содержимого (AD-23):
// принимает иллюстрацию и отдаёт её адрес. Иллюстрация прикладывается к
// наблюдению только после того, как хранилище её приняло: ссылка на
// несуществующий материал — фиктивное доказательство (FR-102).
type MaterialSink interface {
	PutIllustration(ctx context.Context, content []byte, mediaType, note string) (address string, err error)
}

// RouteGate — порт «маршрут подписей протокола допуска закрыт» (AD-43,
// FR-98): паспорт вводится в действие только по закрытому маршруту. Маршрут
// считает documents (эпик 28), полный допуск — эпик 40. Ответ:
// route_closed | pending | demo_stub.
type RouteGate interface {
	Approvals(ctx context.Context, action, documentID string) (string, error)
}

// Состояния подписей протокола допуска.
const (
	ApprovalsClosed   = "route_closed"
	ApprovalsPending  = "pending"
	ApprovalsDemoStub = "demo_stub"
)

// DemoRoutes — разрешающая заглушка порта для профилей demo и fixtures:
// «подписи маршрута не проверялись» (как у nonconformity, Д-47).
type DemoRoutes struct{}

// Approvals — всегда demo_stub.
func (DemoRoutes) Approvals(context.Context, string, string) (string, error) {
	return ApprovalsDemoStub, nil
}

// PendingRoutes — строгая заглушка: подписи не собраны, допуск ждёт маршрута.
type PendingRoutes struct{}

// Approvals — всегда pending.
func (PendingRoutes) Approvals(context.Context, string, string) (string, error) {
	return ApprovalsPending, nil
}
