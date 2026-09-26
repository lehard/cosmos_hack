package ops

import (
	"time"

	"ant/internal/application/platform"
)

// Формы экрана «Настройки и состояние компонентов» (FR-109, FR-127; AD-25,
// AD-35, AD-45). Экран — эпик 14.

// ComponentState — состояние сервиса или роли процесса.
type ComponentState struct {
	Component string     `json:"component" doc:"ant/api, ant/worker, crossitem, projector, scheduler, outbox, stands, keeper, verifier, edge-agent, demo-signer, postgres…"`
	State     string     `json:"state" enum:"ok,degraded,down,not_implemented,unknown"`
	Instances int        `json:"instances" minimum:"0"`
	Leader    *string    `json:"leader,omitempty" doc:"Копия-лидер по аренде (AD-6)."`
	Detail    *string    `json:"detail,omitempty"`
	CheckedAt *time.Time `nullable:"true" json:"checked_at"`
}

// QueueState — очередь или потребитель журнала: отставание курсора (AD-45).
type QueueState struct {
	Name    string `json:"name" doc:"Партиция воркера, глобальный потребитель, outbox."`
	Scope   string `json:"scope" enum:"partition,global,outbox"`
	LagSeq  int64  `json:"lag_seq" minimum:"0" doc:"Отставание курсора от головы журнала."`
	Pending int64  `json:"pending" minimum:"0"`
}

// IntegrationState — состояние интеграции (ops.integration.degraded).
type IntegrationState struct {
	System string     `json:"system" enum:"onec,galaktika,mes,kompas,skud,ca,partner"`
	State  string     `json:"state" enum:"ok,degraded"`
	Detail *string    `json:"detail,omitempty"`
	Since  *time.Time `nullable:"true" json:"since"`
}

// VerifierReportRef — последний отчёт верификатора «по данным сервера» (AD-46).
type VerifierReportRef struct {
	Verdict   string    `json:"verdict" enum:"intact,intact_with_caveats,violated,unknown"`
	CheckedAt time.Time `json:"checked_at"`
	ReportRef string    `json:"report_ref"`
}

// OpsHealth — сводка состояния системы (FR-127): сервисы, очереди, карантин,
// интеграции, остановленные изделия, последний отчёт верификатора.
type OpsHealth struct {
	Components     []ComponentState   `json:"components"`
	Queues         []QueueState       `json:"queues"`
	Integrations   []IntegrationState `json:"integrations"`
	QuarantineOpen int                `json:"quarantine_open" minimum:"0"`
	StoppedItems   int                `json:"stopped_items" minimum:"0" doc:"Изделия «обработка остановлена» (AD-45)."`
	Verifier       *VerifierReportRef `json:"verifier,omitempty" doc:"Нет — отчёта ещё не было."`
	Profile        string             `json:"profile" enum:"fixtures,demo,load,prod"`
	Mode           platform.Mode      `json:"mode"`
	Version        string             `json:"version"`
}

// StoppedItem — изделие, обработка которого остановлена ошибкой свёртки или
// проекции (ops.processing.failed, AD-45): партиция продолжает работу.
type StoppedItem struct {
	ItemID         string    `json:"item_id"`
	FailureEventID string    `json:"failure_event_id"`
	Consumer       string    `json:"consumer"`
	FailedSeq      int64     `json:"failed_seq"`
	Error          string    `json:"error"`
	FailedAt       time.Time `json:"failed_at"`
	Retries        int       `json:"retries" minimum:"0"`
}

// StoppedItemList — остановленные изделия.
type StoppedItemList struct {
	Items      []StoppedItem `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

// PortSetting — адаптер ведомого порта платформы по ключу конфигурации (AD-35).
type PortSetting struct {
	Port    string `json:"port" doc:"Ключ порта: journal_store, work_feed, publisher, telemetry, access_control…"`
	Adapter string `json:"adapter" doc:"Имя адаптера из deploy/config/ant.yaml."`
	Zone    string `json:"zone" enum:"storage,transport,observability,security,application"`
	Replace string `json:"replace,omitempty" doc:"Замена (описано): Kafka, OTLP, PKCS#11, СКЗИ…"`
}

// ModuleMode — режим ведущих портов модуля (AD-36).
type ModuleMode struct {
	Module string        `json:"module"`
	Mode   platform.Mode `json:"mode"`
}

// SettingList — настройки адаптеров и режимов (FR-127).
type SettingList struct {
	Ports   []PortSetting `json:"ports"`
	Modules []ModuleMode  `json:"modules"`
	Enabled []string      `json:"enabled_integrations" doc:"Включённые внешние системы (stand-ы или реальные адаптеры)."`
}

// OpsReason — основание решения: код и текст.
type OpsReason struct {
	Code *string `json:"code,omitempty"`
	Text string  `json:"text" minLength:"1" maxLength:"2000"`
}

// RetryProcessing — повторить обработку остановленного изделия (ops.processing.retried, AD-45).
type RetryProcessing struct {
	platform.CommandHeader
	FailureEventID string     `json:"failure_event_id" format:"uuid"`
	Reason         *OpsReason `json:"reason,omitempty"`
}

// SwitchSource — отключить или включить источник событий (ops.source.disabled / enabled).
type SwitchSource struct {
	platform.CommandHeader
	Reason OpsReason `json:"reason"`
}
