package kernel

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
)

// Slot — слот реакции: правило, субъект, ключ срабатывания (AD-3).
type Slot struct {
	RuleID     string
	Subject    string
	TriggerKey string
}

// Key — канонический ключ слота для UUIDv5.
func (s Slot) Key() string { return s.RuleID + "\x1f" + s.Subject + "\x1f" + s.TriggerKey }

// Reaction — реакция, вычисленная свёрткой (AD-3). Не входит в свёртку: при
// пересвёртке её вычисляют заново, записанная служит для сравнения (AD-5).
// Версию, supersedes и basis_seq проставляет сценарий воркера (application/engine).
type Reaction struct {
	// Module — модуль-эмитент; равен эмитенту типа в каталоге (AD-40).
	Module Module
	// Type — тип записи-реакции.
	Type catalog.Type
	// Slot — слот реакции.
	Slot Slot
	// RuleRev, AutomationMode — ревизия нормативного слоя и режим правила 1–5 (FR-50).
	RuleRev        string
	AutomationMode int
	// Causes — отсортированные event_id записей, влияющих на вывод (AD-3).
	Causes []string
	// OccurredAt — наибольший occurred_at среди причин (AD-37).
	OccurredAt time.Time
	// Data — значение сгенерированного типа data.
	Data any
}

// ID — reaction_id = UUIDv5(NS_ANT, слот ‖ версия) (AD-3).
func (r Reaction) ID(version int) string {
	return UUIDv5(constants.NsAnt, r.Slot.Key()+"\x1f"+strconv.Itoa(version))
}

// NewReaction строит реакцию модуля m. Ошибка — если тип не из каталога, не
// реакция или его эмитент не m («модуль эмитит чужой тип», AD-40).
func NewReaction(m Module, t catalog.Type, slot Slot, data any, causes ...Record) (Reaction, error) {
	info, ok := catalog.Lookup(t)
	if !ok {
		return Reaction{}, fmt.Errorf("kernel: тип %q не из каталога", t)
	}
	if info.Emitter != string(m) {
		return Reaction{}, fmt.Errorf("kernel: модуль %s эмитит чужой тип %s (эмитент — %s, AD-40)", m, t, info.Emitter)
	}
	if info.Kind != catalog.KindReaction {
		return Reaction{}, fmt.Errorf("kernel: тип %s — не реакция (%s)", t, info.Kind)
	}
	ids := make([]string, 0, len(causes))
	for _, c := range causes {
		ids = append(ids, c.EventID)
	}
	slices.Sort(ids)
	return Reaction{Module: m, Type: t, Slot: slot, Causes: slices.Compact(ids), OccurredAt: MaxOccurred(causes), Data: data}, nil
}

// Intent — намерение позднего модуля композиции, адресованное раннему модулю-
// владельцу оси или объекта (AD-40, AD-30): «quality продвигает токен точки
// предъявления функцией process», «nonconformity создаёт черновик через
// documents.Draft». Строит намерение только функция модуля-владельца (Target);
// применяет — его Apply в конце шага свёртки.
type Intent struct {
	// Target — модуль-владелец, который применит намерение.
	Target Module
	// Name — имя функции-намерения владельца (например, "advance_presentation").
	Name string
	// From — модуль, выразивший намерение.
	From Module
	// Causes — записи-основания.
	Causes []string
	// Payload — данные намерения (тип задаёт владелец).
	Payload any
}

// Output — результат React модуля за шаг свёртки.
type Output struct {
	Reactions []Reaction
	Intents   []Intent
}

// Merge дописывает o2 к o.
func (o *Output) Merge(o2 Output) {
	o.Reactions = append(o.Reactions, o2.Reactions...)
	o.Intents = append(o.Intents, o2.Intents...)
}

// Addressed — адресованная запись межизделийной стадии (AD-42): выход стадии
// в поток изделия или объекта, occurred_at = наибольший среди причин, basis_seq.
type Addressed struct {
	// Module — модуль-эмитент типа (для стадии: crossitem, analysis, nonconformity, reference…).
	Module Module
	// Type — тип записи.
	Type catalog.Type
	// Stream — поток-адресат (`item:‹id›`, `incident:‹id›` …).
	Stream string
	// Causes — отсортированные event_id причин.
	Causes []string
	// OccurredAt — наибольший occurred_at среди причин.
	OccurredAt time.Time
	// Key — ключ идемпотентности адресованной записи внутри стадии.
	Key string
	// Data — значение сгенерированного типа data.
	Data any
}

// NewAddressed строит адресованную запись модуля m в поток stream; эмитент
// типа по каталогу обязан быть m (AD-40).
func NewAddressed(m Module, t catalog.Type, stream, key string, data any, causes ...Record) (Addressed, error) {
	info, ok := catalog.Lookup(t)
	if !ok {
		return Addressed{}, fmt.Errorf("kernel: тип %q не из каталога", t)
	}
	if info.Emitter != string(m) {
		return Addressed{}, fmt.Errorf("kernel: модуль %s эмитит чужой тип %s (эмитент — %s, AD-40)", m, t, info.Emitter)
	}
	ids := make([]string, 0, len(causes))
	for _, c := range causes {
		ids = append(ids, c.EventID)
	}
	slices.Sort(ids)
	return Addressed{Module: m, Type: t, Stream: stream, Key: key, Causes: slices.Compact(ids), OccurredAt: MaxOccurred(causes), Data: data}, nil
}

// NewIntent — намерение модуля from к модулю-владельцу target. Вызывают только
// функции-намерения модуля-владельца (например, process.AdvancePresentation),
// чтобы словарь намерений принадлежал владельцу оси (AD-30).
func NewIntent(target, from Module, name string, payload any, causes ...Record) Intent {
	ids := make([]string, 0, len(causes))
	for _, c := range causes {
		ids = append(ids, c.EventID)
	}
	slices.Sort(ids)
	return Intent{Target: target, Name: name, From: from, Causes: slices.Compact(ids), Payload: payload}
}
