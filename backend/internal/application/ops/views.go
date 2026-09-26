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
	// Quarantined — у очереди исходящих: сообщений в карантине (ждут решения человека).
	Quarantined *int64 `json:"quarantined,omitempty" minimum:"0" doc:"Очередь исходящих: сообщений в карантине (ждут решения человека, FR-96)."`
}

// IntegrationState — состояние интеграции (ops.integration.degraded).
type IntegrationState struct {
	System string     `json:"system" enum:"onec,galaktika,mes,kompas,skud,ca,partner"`
	State  string     `json:"state" enum:"ok,degraded,disabled" doc:"disabled — система не включена (integrations.enabled) или канал выключен."`
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
	SelfCheck      *SelfCheckView     `json:"selfcheck,omitempty" doc:"Самопроверка после старта этой копии api (FR-109)."`
}

// SelfCheckFinding — находка самопроверки.
type SelfCheckFinding struct {
	Check    string `json:"check" enum:"journal,genesis,migrations,db_roles,roles"`
	Severity string `json:"severity" enum:"critical,warning"`
	Text     string `json:"text"`
}

// SelfCheckView — итог самопроверки после старта (FR-109, AD-25): журнал,
// генезис, миграции, роли БД и процесса.
type SelfCheckView struct {
	OK        bool               `json:"ok" doc:"Критических находок нет."`
	Summary   string             `json:"summary" doc:"«инициализация без критических ошибок» или перечень критических находок."`
	Findings  []SelfCheckFinding `json:"findings"`
	CheckedAt time.Time          `json:"checked_at"`
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
	Ports   []PortSetting  `json:"ports"`
	Modules []ModuleMode   `json:"modules"`
	Enabled []string       `json:"enabled_integrations" doc:"Включённые внешние системы (stand-ы или реальные адаптеры)."`
	Sources []SourceSwitch `json:"sources,omitempty" doc:"Источники, которые отключали или включали (ops.source.disabled / enabled)."`
}

// SourceSwitch — последнее решение об источнике событий (AD-28).
type SourceSwitch struct {
	SourceID string    `json:"source_id"`
	Enabled  bool      `json:"enabled"`
	Reason   string    `json:"reason"`
	Actor    string    `json:"actor,omitempty" doc:"Кто решил (псевдоним)."`
	At       time.Time `json:"at"`
	Seq      int64     `json:"seq" doc:"basis_seq для следующей команды над источником (AD-39)."`
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

// Формы экрана «Интеграции» стола администратора (FR-157, AD-47; эпик 48).

// IntegrationDecision — последнее решение администратора о состоянии интеграции.
type IntegrationDecision struct {
	State    string    `json:"state" enum:"enabled,disabled,stand"`
	Previous *string   `json:"previous,omitempty" enum:"enabled,disabled,stand"`
	Reason   string    `json:"reason"`
	Actor    string    `json:"actor,omitempty" doc:"Кто решил (псевдоним)."`
	At       time.Time `json:"at"`
	Seq      int64     `json:"seq"`
}

// IntegrationCheck — последняя проверка соединения (ops.integration.checked).
type IntegrationCheck struct {
	Result   string    `json:"result" enum:"ok,degraded,unreachable,not_supported"`
	Detail   *string   `json:"detail,omitempty"`
	Endpoint *string   `json:"endpoint,omitempty"`
	At       time.Time `json:"at"`
	Seq      int64     `json:"seq"`
}

// IntegrationError — последний сбой канала (ops.integration.degraded, AD-18).
type IntegrationError struct {
	At     time.Time `json:"at"`
	Detail string    `json:"detail"`
}

// IntegrationEntry — внешняя система на экране «Интеграции».
type IntegrationEntry struct {
	System    string `json:"system" enum:"onec,galaktika,mes,kompas,skud,ca,visionqc,operatorvision,partner" doc:"Внешняя система."`
	Installed bool   `json:"installed" doc:"Установлена конфигурацией (integrations.enabled); нет — «не установлена», кнопок нет."`
	State     string `json:"state" enum:"enabled,disabled,stand" doc:"enabled — включена (реальная система), disabled — выключена, stand — стенд (эмулятор)."`
	Default   bool   `json:"default" doc:"Решений не было — состояние по умолчанию профиля (prod — реальная, demo и fixtures — стенд)."`
	// StandAvailable, RealAvailable — какие режимы можно включить: заданы адресом в конфигурации и разрешены профилем.
	StandAvailable bool                 `json:"stand_available" doc:"Можно переключить на стенд: стенд задан конфигурацией и профиль не prod."`
	RealAvailable  bool                 `json:"real_available" doc:"Можно переключить на реальную систему: её адрес задан конфигурацией."`
	Channel        *string              `json:"channel,omitempty" enum:"ok,degraded,disabled" doc:"Живое состояние канала обмена (сверка ответной стороны, AD-18)."`
	Detail         *string              `json:"detail,omitempty"`
	Endpoint       *string              `json:"endpoint,omitempty" doc:"Адрес ответной стороны без секретов."`
	CheckedAt      *time.Time           `json:"checked_at,omitempty"`
	LastExchangeAt *time.Time           `json:"last_exchange_at,omitempty" doc:"Последний обмен с системой."`
	Queued         *int64               `json:"queued,omitempty" minimum:"0" doc:"Исходящих в очереди (у выключенной копятся до включения)."`
	Quarantined    *int64               `json:"quarantined,omitempty" minimum:"0" doc:"Исходящих в карантине — ждут ручной переотправки (FR-96)."`
	LastError      *IntegrationError    `json:"last_error,omitempty" doc:"Последний сбой канала."`
	LastCheck      *IntegrationCheck    `json:"last_check,omitempty" doc:"Последняя проверка соединения."`
	LastDecision   *IntegrationDecision `json:"last_decision,omitempty" doc:"Последнее решение администратора."`
	BasisSeq       int64                `json:"basis_seq" minimum:"0" doc:"basis_seq для следующей команды над интеграцией (AD-39)."`
}

// IntegrationList — экран «Интеграции» (ops.integration.list).
type IntegrationList struct {
	Profile string             `json:"profile" enum:"fixtures,demo,load,prod" doc:"В prod стенд запрещён."`
	Items   []IntegrationEntry `json:"items"`
}

// SetIntegrationState — задать состояние интеграции (ops.integration.state_set).
type SetIntegrationState struct {
	platform.CommandHeader
	State  string    `json:"state" enum:"enabled,disabled,stand" doc:"enabled — включить реальную систему; disabled — выключить; stand — переключить на стенд (в prod — отказ ops.stand_forbidden)."`
	Reason OpsReason `json:"reason"`
}

// CheckIntegration — проверить соединение (ops.integration.checked).
type CheckIntegration struct {
	platform.CommandHeader
}
