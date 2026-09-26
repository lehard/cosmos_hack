package simulation

import (
	"encoding/json"
	"time"
)

// Plan — результат генератора: «истина» прогона, события с шумом, которые
// источники отправят в обычный приём, и шаги людей и служебные шаги.
// Один и тот же вход (определения, seed, run_id, «сейчас») даёт один и тот же
// план (AD-4, AD-38). Его же берут заготовки (эпик 09): ID изделий совпадают.
type Plan struct {
	RunID  string
	RunDef string
	Seed   int64
	// Shift — сдвиг дат определения к доменному «сейчас» (целые недели, AD-38).
	Shift time.Duration
	// Start, End — виртуальное время начала и конца прогона (уже со сдвигом).
	Start, End time.Time
	// LiveFrom — начало живой части (RunDef.Live, уже со сдвигом); нуль — нет:
	// всё раньше — история, её интерактивный прогон проигрывает сразу.
	LiveFrom time.Time
	// Emissions — события источников в порядке доставки (DeliverAt, Order).
	Emissions []Emission
	// Actions — шаги людей и служебные шаги в порядке времени.
	Actions []Action
	// Truth — «истина» генератора: изделия, этапы, сварки, счётчики шума.
	Truth Truth
	// IDs — сведение идентификаторов прогона.
	IDs *IDMap
}

// Delivery — вид доставки события.
type Delivery string

// Виды доставки (шум кейса §4.6, таблица сбоев PRD §4.3).
const (
	DeliveryNormal    Delivery = "normal"    // сразу после возникновения
	DeliveryLate      Delivery = "late"      // досылка после восстановления связи
	DeliveryDuplicate Delivery = "duplicate" // повтор того же сообщения (тот же номер)
	DeliveryConflict  Delivery = "conflict"  // тот же номер, другое содержимое
	DeliveryReordered Delivery = "reordered" // доставлено не в порядке номеров
)

// Emission — одно исходное событие источника (строка JSONL, AD-26): его
// подписывает stand или edge-агент и отправляет в обычный приём.
type Emission struct {
	// N — номер события генератора: event_id = UUIDv5(run_id, seed, N) (AD-38).
	N int `json:"n"`
	// Order — порядок доставки при равном DeliverAt.
	Order int `json:"order"`
	// DeliverAt — виртуальное время, когда источник отправляет сообщение.
	DeliverAt  time.Time `json:"deliver_at"`
	SourceKey  string    `json:"source_key"`
	SourceID   string    `json:"source_id"`
	SourceSeq  int64     `json:"source_seq"`
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	OccurredAt time.Time `json:"occurred_at"`
	// TrueAt — когда событие случилось на самом деле (для сдвига часов ≠ OccurredAt).
	TrueAt   time.Time `json:"true_at"`
	Item     string    `json:"item,omitempty"`
	Scenario string    `json:"scenario"`
	Label    string    `json:"label,omitempty"`
	Delivery Delivery  `json:"delivery"`
	// Contract — false у сообщений не по контракту (S12): схемы их не пропускают.
	Contract bool `json:"contract"`
	// Quarantine — приём должен положить сообщение в карантин (FR-30).
	Quarantine bool `json:"quarantine,omitempty"`
	// Event — исходное событие (конверт v1, канонический JSON), без подписи.
	Event json.RawMessage `json:"event"`
}

// ActionKind — вид шага, который исполняет прогон, а не источник.
type ActionKind string

// Виды шагов.
const (
	// ActionDecision — решение человека операцией API (AD-26): интерактивно —
	// стол роли, в автосверке — demo-signer.
	ActionDecision ActionKind = "decision"
	// ActionStand — служебный порт stand-а: включить или снять сбой (AD-18).
	ActionStand ActionKind = "stand"
	// ActionTamper — подделка в обход системы демо-инструментом cmd/tamper (AD-26, AD-28).
	ActionTamper ActionKind = "tamper"
)

// Action — шаг прогона, исполняемый не источником: решение, сбой stand-а, подделка.
type Action struct {
	// Seq — номер шага прогона (по времени): command_id = UUIDv5(run, seed, Seq).
	Seq      int        `json:"seq"`
	Kind     ActionKind `json:"kind"`
	At       time.Time  `json:"at"`
	Scenario string     `json:"scenario"`
	Label    string     `json:"label,omitempty"`
	Note     string     `json:"note,omitempty"`
	// Operation — operationId API (решение).
	Operation string `json:"operation,omitempty"`
	Role      string `json:"role,omitempty"`
	// Actor — персона контракта (после сведения имён).
	Actor string `json:"actor,omitempty"`
	// Stop — интерактивный прогон ждёт решения на столе роли (FR-129).
	Stop bool `json:"stop,omitempty"`
	// Params, Body — параметры пути и запроса и тело команды; плейсхолдеры
	// {item:…} и {ref:…} разрешаются во время прогона.
	Params map[string]any `json:"params,omitempty"`
	Body   map[string]any `json:"body,omitempty"`
	// Refusal — ожидаемый код отказа (шаг проверяет запрет).
	Refusal string `json:"refusal,omitempty"`
	// Auto — решение машины или лаборатории (Step.Auto): в живой части не
	// останавливает прогон, подписывает demo-signer.
	Auto bool `json:"auto,omitempty"`
	// Binds — какое изделие определения рождает решение (регистрация):
	// после исполнения прогон узнаёт его item_id из записи журнала.
	Binds string `json:"binds,omitempty"`
	// Item — изделие определения, к которому относится шаг.
	Item   string       `json:"item,omitempty"`
	Stand  *StandAction `json:"stand,omitempty"`
	Tamper *Tamper      `json:"tamper,omitempty"`
}

// Truth — «истина» генератора (FR-104): что было на самом деле, до шума.
type Truth struct {
	Items []ItemTruth `json:"items"`
	Welds []WeldTruth `json:"welds"`
	// Sources — счётчики по источникам: номера, потери, дубли, опоздания.
	Sources []SourceTruth `json:"sources"`
	// Totals — счётчики прогона.
	Totals Counters `json:"totals"`
}

// ItemTruth — изделие и моменты его этапов.
type ItemTruth struct {
	ID         string               `json:"id"`
	Order      string               `json:"order"`
	Line       string               `json:"line,omitempty"`
	Reference  bool                 `json:"reference,omitempty"`
	Stages     map[string]time.Time `json:"stages"`
	RemarkedAt time.Time            `json:"remarked_at,omitzero"`
	Until      string               `json:"until"`
}

// WeldTruth — сварка (выполнение операции) по «истине».
type WeldTruth struct {
	Item    string    `json:"item"`
	Run     string    `json:"run"` // SV-017-1
	Station string    `json:"station"`
	Welder  string    `json:"welder"`
	Shift   string    `json:"shift"`
	Start   time.Time `json:"start"`
	End     time.Time `json:"end"`
	// CurrentMin, CurrentMax — ток по журналу, А.
	CurrentMin int       `json:"current_min"`
	CurrentMax int       `json:"current_max"`
	OutFrom    time.Time `json:"out_from,omitzero"`
	// ArcFrom, ArcTo — окно горения дуги (сводки источника только в нём).
	ArcFrom time.Time `json:"arc_from"`
	ArcTo   time.Time `json:"arc_to"`
	Rework  string    `json:"rework_of,omitempty"`
}

// SourceTruth — источник: первый и последний номер, потери, повторы.
type SourceTruth struct {
	Key      string   `json:"key"`
	SourceID string   `json:"source_id"`
	FirstSeq int64    `json:"first_seq"`
	LastSeq  int64    `json:"last_seq"`
	Counters Counters `json:"counters"`
	// LostSeqs — потерянные номера (разрывы source_seq).
	LostSeqs []int64 `json:"lost_seqs,omitempty"`
	// SkewSec — сдвиг часов источника, с (0 — нет).
	SkewSec int `json:"skew_s,omitempty"`
}

// Counters — счётчики доставки.
type Counters struct {
	Records    int `json:"records"`    // событий у источника (с потерянными)
	Deliveries int `json:"deliveries"` // отправок (с повторами)
	Unique     int `json:"unique"`     // уникальных доставленных
	Duplicates int `json:"duplicates"` // повторов того же сообщения
	Conflicts  int `json:"conflicts"`  // тот же номер, другое содержимое
	Lost       int `json:"lost"`       // потеряно у источника
	Late       int `json:"late"`       // доставлено позже 5 мин после возникновения
	Skewed     int `json:"skewed"`     // время источника спешит
	Reordered  int `json:"reordered"`  // доставлено не по порядку номеров
	Invalid    int `json:"invalid"`    // не проходит схемы контракта
	Quarantine int `json:"quarantine"` // ожидается карантин
	Decisions  int `json:"decisions"`  // шагов людей (только в Totals)
}

// Add прибавляет счётчики.
func (c *Counters) Add(o Counters) {
	c.Records += o.Records
	c.Deliveries += o.Deliveries
	c.Unique += o.Unique
	c.Duplicates += o.Duplicates
	c.Conflicts += o.Conflicts
	c.Lost += o.Lost
	c.Late += o.Late
	c.Skewed += o.Skewed
	c.Reordered += o.Reordered
	c.Invalid += o.Invalid
	c.Quarantine += o.Quarantine
	c.Decisions += o.Decisions
}
