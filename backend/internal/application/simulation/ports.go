package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	sim "ant/internal/domain/simulation"
)

// Ведомые порты модуля simulation (AD-26, AD-35): определения сценариев,
// отправка событий в обычный приём, чтение и команды теми же операциями API,
// служебный порт stand-ов, демо-инструмент подделки, служебные записи прогона
// в журнал, ожидание обработки, хранение состояния прогонов.

// Definitions — определения сценариев (scenarios/definitions, scenarios/expected).
type Definitions interface {
	// Catalog — каталог сценариев пульта (index.yaml).
	Catalog(ctx context.Context) (sim.Catalog, error)
	// Bundle — мир, прогон и его карточки.
	Bundle(ctx context.Context, run string) (sim.Bundle, error)
	// Expected — ожидания карточки (отдельно от входа, кейс §5.1); ok = false — нет файла.
	Expected(ctx context.Context, scenario string) (sim.Expected, bool, error)
}

// DeliveryStatus — итог доставки события в приём.
type DeliveryStatus string

// Итоги доставки (ответ приёма, application/ingest.IngestOutcome.Status).
const (
	DeliveryAccepted    DeliveryStatus = "accepted"
	DeliveryDuplicate   DeliveryStatus = "duplicate"
	DeliveryQuarantined DeliveryStatus = "quarantined"
	DeliveryRejected    DeliveryStatus = "rejected"
)

// Delivered — итог доставки одного события.
type Delivered struct {
	EventID string
	Status  DeliveryStatus
	Code    string
	// Seq — позиция записи в журнале (принято — новая, повтор — прежняя; 0 — не известна).
	Seq int64
}

// Gateway — отправка исходных событий источников в обычный приём (AD-26:
// stand-ы источников → edge-агент → приём). Адаптеры: прямой вызов порта
// приёма ingest.Commands (IngestGateway) и, когда edge-агент примет несколько
// источников, — локальный вход edge-агента.
type Gateway interface {
	Deliver(ctx context.Context, runID string, batch []sim.Emission) ([]Delivered, error)
}

// ErrUnavailable — операция пока не отвечает (501, модуль в работе): строка
// табло остаётся «ожидает» с пояснением, а не «не совпало».
var ErrUnavailable = errors.New("операция недоступна")

// ErrNotFound — объект не найден (404).
var ErrNotFound = errors.New("не найдено")

// Probe — чтение теми же операциями API, что показывают столы (AD-26:
// автосверка «через те же Queries»): operationId, параметры пути и запроса,
// прогон (run_id — фильтр AD-38). Ответ — JSON как у клиента (числа — json.Number).
type Probe interface {
	Read(ctx context.Context, operation string, params map[string]string, runID string) (any, error)
}

// ActResult — итог команды от имени персоны.
type ActResult struct {
	Seq      int64
	EventIDs []string
	// Code — код отказа problem+json (пусто — команда принята).
	Code   string
	Status int
	// Detail — пояснение отказа problem+json.
	Detail string
}

// Actor — решения людей теми же операциями API от имени демо-персоны
// (demo-signer, AD-26): обычный клиент, без служебных обходов; в интерактивном
// режиме — проверка, что человек сам принял решение на столе роли.
type Actor interface {
	// Act — выполнить команду operation от имени persona.
	Act(ctx context.Context, persona, operation string, params map[string]string, body map[string]any) (ActResult, error)
	// Decided — seq записей журнала после since, которые эмитит operation над
	// object (решение принято на столе роли), по возрастанию; object пусто —
	// любой объект. Ищет в журнале прогона, а при заданном object — и среди
	// записей без run_id (допуск к посту, остановка поста: у них нет изделия).
	Decided(ctx context.Context, runID, operation, object string, since int64) ([]int64, error)
}

// Stands — служебный порт stand-ов (AD-18): сбои включаются только со
// страницы тестовых сценариев (application/ingest.StandControl).
type Stands interface {
	SetFault(ctx context.Context, stand string, a sim.StandAction, until time.Time) error
	ClearFaults(ctx context.Context, stand string) error
}

// Tamperer — подделка в обход системы демо-инструментом cmd/tamper с отдельным
// подключением суперпользователя БД (AD-26, AD-28): есть только в профилях
// fixtures и demo.
type Tamperer interface {
	Apply(ctx context.Context, runID string, t sim.Tamper, eventID string) error
}

// Record — служебная запись прогона в журнал: simulation.run.* и
// time.clock.ticked (эмитент — simulation, AD-40).
type Record struct {
	Type       string
	RunID      string
	OccurredAt time.Time
	Data       map[string]any
}

// Recorder — служебные записи прогона через единственную функцию записи
// journal.Append (AD-44). Возвращает seq записи.
type Recorder interface {
	Record(ctx context.Context, r Record) (int64, error)
}

// Settler — дождаться, пока воркер и межизделийная стадия обработают всё
// доставленное прогоном (перед проверкой табло и решением человека).
type Settler interface {
	Settle(ctx context.Context, runID string) error
}

// InfraClock — реальные монотонные часы (AD-37): темп виртуальных часов.
type InfraClock interface{ Now() time.Time }

// DomainNow — доменное «сейчас» журнала (AD-37): последняя запись
// time.clock.ticked или системные часы; от него сдвигается прогон (AD-38).
type DomainNow interface {
	Now(ctx context.Context) (time.Time, error)
}

// RunStore — состояние прогонов (курсор, часы, табло). Истина — записи
// журнала прогона и план генератора (он детерминирован по seed); хранилище —
// их быстрый кэш.
type RunStore interface {
	Save(ctx context.Context, r *RunState) error
	Load(ctx context.Context, runID string) (*RunState, bool, error)
	List(ctx context.Context) ([]*RunState, error)
}

// RunState — состояние прогона.
type RunState struct {
	RunID   string `json:"run_id"`
	Entry   string `json:"entry"` // сценарий пульта (строка каталога)
	RunDef  string `json:"run_def"`
	Version string `json:"version"`
	Seed    int64  `json:"seed"`
	Mode    string `json:"mode"`
	State   string `json:"state"`
	// CommandID — команда запуска (повтор с тем же id — тот же прогон, AD-7).
	CommandID string `json:"command_id,omitempty"`
	// GenNow — доменное «сейчас» на старте: от него сдвинуты даты определения (AD-38).
	GenNow     time.Time  `json:"gen_now"`
	Clock      sim.Clock  `json:"clock"`
	Cursor     sim.Cursor `json:"cursor"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	// LastTick — виртуальный момент последней записи time.clock.ticked.
	LastTick time.Time `json:"last_tick"`
	// Waiting — решение человека, которого ждёт интерактивный прогон.
	Waiting *Waiting `json:"waiting,omitempty"`
	// Items, Refs — изделия и объекты системы, узнанные во время прогона.
	Items map[string]string `json:"items"`
	Refs  map[string]string `json:"refs"`
	// Steps — итоги шагов (решения, сбои stand-ов, подделки) по метке.
	Steps map[string]StepResult `json:"steps"`
	// Rows — строки табло по id утверждения; Baselines — значения «до».
	Rows      map[string]sim.Result `json:"rows"`
	Baselines map[string]any        `json:"baselines"`
	// Delivered — итоги доставки: принято, дубли, карантин, отказы.
	Delivered map[DeliveryStatus]int `json:"delivered"`
	// Error — почему прогон остановлен с ошибкой.
	Error string `json:"error,omitempty"`
	// BasisSeq — seq последней служебной записи прогона.
	BasisSeq int64 `json:"basis_seq"`
	// Injections — нажатия кнопок цифрового стенда (FR-152) и их строки табло.
	Injections []InjectionState `json:"injections,omitempty"`
	// StandSeq — последние source_seq источников stand-а цифрового стенда.
	StandSeq map[string]int64 `json:"stand_seq,omitempty"`
	// Live — прогон дошёл до живой части (Plan.LiveFrom): история проиграна,
	// часы идут от начала живой части (Д-85).
	Live bool `json:"live,omitempty"`
	// LiveSeq — seq журнала на входе в живую часть: решения людей живой части
	// ищутся после него — человек может нажать раньше, чем прогон дошёл до шага.
	LiveSeq int64 `json:"live_seq,omitempty"`
	// Consumed — seq решений людей, уже закрывших остановки: одна запись
	// закрывает одну остановку (два приёма Ф-003 — в цех и в изолятор).
	Consumed []int64 `json:"consumed,omitempty"`
}

// Waiting — ожидание решения человека (FR-129).
type Waiting struct {
	Action int    `json:"action"` // номер шага плана
	Role   string `json:"role"`
	Op     string `json:"op"`
	Object string `json:"object"`
	Title  string `json:"title"`
	// Since — seq журнала, после которого ищем решение.
	Since int64 `json:"since"`
}

// StepResult — итог шага прогона.
type StepResult struct {
	Operation string    `json:"operation,omitempty"`
	At        time.Time `json:"at"`
	Status    string    `json:"status"` // done | refused | skipped | failed | waiting
	Refusal   string    `json:"refusal,omitempty"`
	Seq       int64     `json:"seq,omitempty"`
	Detail    string    `json:"detail,omitempty"`
	// Delivered — ответы приёма на события шага (кнопки стенда): accepted, duplicate…
	Delivered []string `json:"delivered,omitempty"`
}

// Snapshot — глубокая копия состояния (хранилища отдают копии: без гонок).
func Snapshot(r *RunState) *RunState {
	b, _ := json.Marshal(r)
	var c RunState
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	_ = dec.Decode(&c)
	return &c
}
