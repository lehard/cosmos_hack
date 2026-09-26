package simulation

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// IDMap — сведение идентификаторов прогона (AD-38, AD-16):
//   - ID процессной сессии → ID контракта (World.Aliases: люди, оборудование, зоны);
//   - локальные ID прогона (партии, задания, выполнения, носители) → ‹run_id›/‹id›;
//   - метки событий определения → event_id генератора;
//   - изделия и объекты, рождённые системой (F-017 → item_id, NC-01 → nc_id), —
//     заполняются во время прогона теми же Queries (Items, Refs).
//
// Плейсхолдеры в строках определений и ожиданий:
//
//	{run}          — run_id прогона
//	{local:X}      — ‹run_id›/X (партии, задания, выполнения, наблюдения)
//	{carrier:X}    — значение носителя ‹run_id›/X (TAG:F-001, DM:F-001)
//	{event:L}      — event_id события с меткой L
//	{person:X}     — персона контракта для сотрудника X (WLD-02 → W21)
//	{equipment:X}, {zone:X}, {role:X} — ID контракта
//	{item:F-017}   — внутренний ID изделия (во время прогона)
//	{ref:NC-01}    — объект, рождённый системой (во время прогона, RunDef.Refs)
type IDMap struct {
	RunID      string
	Seed       int64
	Enterprise string
	aliases    Aliases
	// Labels — метка события определения → event_id.
	Labels map[string]string
	// Items — изделие определения → внутренний item_id (после регистрации).
	Items map[string]string
	// Refs — объект, рождённый системой → его ID.
	Refs map[string]string
}

// NewIDMap — сведение идентификаторов прогона runID мира w.
func NewIDMap(runID string, seed int64, w World) *IDMap {
	return &IDMap{RunID: runID, Seed: seed, Enterprise: w.Enterprise, aliases: w.Aliases,
		Labels: map[string]string{}, Items: map[string]string{}, Refs: map[string]string{}}
}

// Local — локальный ID прогона: ‹run_id›/‹id› (AD-38).
func (m *IDMap) Local(id string) string { return m.RunID + "/" + id }

// SourceID — source_id источника прогона: ‹run_id›/‹источник› (AD-38).
func (m *IDMap) SourceID(local string) string { return m.RunID + "/" + local }

// EventID — event_id события генератора: UUIDv5(NS_ANT, run_id ‖ seed ‖ номер)
// (AD-38: ставит генератор, edge-агент не переназначает).
func (m *IDMap) EventID(n int) string {
	return kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("simulation|%s|%d|%d", m.RunID, m.Seed, n))
}

// CommandID — command_id решения шага прогона (повтор шага — тот же ответ, AD-7).
// ActionCommandID — command_id решения прогона: от метки шага (метки
// уникальны в плане и не меняются при вставке шагов), без метки — от номера.
// От номера — нельзя: вставка шагов в историю сдвигает номера, и изделие,
// рождённое регистрацией (item_id — из command_id), получает id другого
// изделия прежней версии плана (Ф-101 ↔ Ф-001 в SHOW-IS2).
func (m *IDMap) ActionCommandID(label string, step int) string {
	if label == "" {
		return m.CommandID(step)
	}
	return kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("simulation-command|%s|%d|label|%s", m.RunID, m.Seed, label))
}

func (m *IDMap) CommandID(step int) string {
	return kernel.UUIDv5(constants.NsAnt, fmt.Sprintf("simulation-command|%s|%d|%d", m.RunID, m.Seed, step))
}

// Person — персона контракта для сотрудника процессной сессии (или как есть).
func (m *IDMap) Person(id string) string { return alias(m.aliases.People, id) }

// Equipment — ID оборудования контракта.
func (m *IDMap) Equipment(id string) string { return alias(m.aliases.Equipment, id) }

// Zone — ID зоны контракта.
func (m *IDMap) Zone(id string) string { return alias(m.aliases.Zones, id) }

// Role — роль контракта.
func (m *IDMap) Role(id string) string { return alias(m.aliases.Roles, id) }

// Code — код ошибки контракта для рабочего кода процессной сессии (E_REWORK_LIMIT → process.rework_limit_exceeded).
func (m *IDMap) Code(id string) string { return alias(m.aliases.Codes, id) }

func alias(m map[string]string, id string) string {
	if v, ok := m[id]; ok && v != "" {
		return v
	}
	return id
}

var rePlaceholder = regexp.MustCompile(`\{(run|local|carrier|event|person|equipment|zone|role|code|item|ref)(?::([^{}]+))?\}`)

// ErrUnresolved — плейсхолдер ещё не известен (изделие не зарегистрировано,
// объект системы не найден).
type ErrUnresolved struct{ Placeholder string }

func (e *ErrUnresolved) Error() string { return "не разрешено: " + e.Placeholder }

// ExpandString подставляет плейсхолдеры. strict=false оставляет неизвестные
// (изделия и объекты системы) как есть — так план строится до прогона;
// strict=true требует разрешить всё.
func (m *IDMap) ExpandString(s string, strict bool) (string, error) {
	if !strings.Contains(s, "{") {
		return s, nil
	}
	var firstErr error
	out := rePlaceholder.ReplaceAllStringFunc(s, func(ph string) string {
		sub := rePlaceholder.FindStringSubmatch(ph)
		kind, arg := sub[1], sub[2]
		v, ok := m.resolve(kind, arg)
		if !ok {
			if strict && firstErr == nil {
				firstErr = &ErrUnresolved{Placeholder: ph}
			}
			return ph
		}
		return v
	})
	return out, firstErr
}

func (m *IDMap) resolve(kind, arg string) (string, bool) {
	switch kind {
	case "run":
		return m.RunID, true
	case "local":
		return m.Local(arg), true
	case "carrier":
		return m.Local(arg), true
	case "event":
		v, ok := m.Labels[arg]
		return v, ok
	case "person":
		return m.Person(arg), true
	case "equipment":
		return m.Equipment(arg), true
	case "zone":
		return m.Zone(arg), true
	case "role":
		return m.Role(arg), true
	case "code":
		return m.Code(arg), true
	case "item":
		v, ok := m.Items[arg]
		return v, ok
	case "ref":
		v, ok := m.Refs[arg]
		return v, ok
	}
	return "", false
}

// Expand подставляет плейсхолдеры во всём значении (строки, списки, объекты).
func (m *IDMap) Expand(v any, strict bool) (any, error) {
	switch x := v.(type) {
	case string:
		return m.ExpandString(x, strict)
	case []any:
		out := make([]any, len(x))
		for i := range x {
			e, err := m.Expand(x[i], strict)
			if err != nil {
				return nil, err
			}
			out[i] = e
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(x))
		for _, k := range slices.Sorted(maps.Keys(x)) {
			e, err := m.Expand(x[k], strict)
			if err != nil {
				return nil, err
			}
			out[k] = e
		}
		return out, nil
	}
	return v, nil
}

// ExpandMap — Expand для объекта.
func (m *IDMap) ExpandMap(v map[string]any, strict bool) (map[string]any, error) {
	if v == nil {
		return nil, nil
	}
	e, err := m.Expand(v, strict)
	if err != nil {
		return nil, err
	}
	return e.(map[string]any), nil
}

// Translate переводит ID процессной сессии в ожидаемом значении в ID
// прогона: изделия (F-017 → item_id, если известен), партии, задания и
// выполнения (LOT-R-117 → ‹run›/LOT-R-117), люди, оборудование, зоны,
// объекты системы (NC-01 → nc_id). Строки, которые ни с чем не совпали,
// остаются как есть.
func (m *IDMap) Translate(v any) any {
	switch x := v.(type) {
	case string:
		if s, err := m.ExpandString(x, false); err == nil && s != x {
			return s
		}
		if id, ok := m.Items[x]; ok {
			return id
		}
		if id, ok := m.Refs[x]; ok {
			return id
		}
		for _, a := range []map[string]string{m.aliases.People, m.aliases.Equipment, m.aliases.Zones, m.aliases.Codes} {
			if id, ok := a[x]; ok && id != "" {
				return id
			}
		}
		if isLocalID(x) {
			return m.Local(x)
		}
		return x
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = m.Translate(x[i])
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for _, k := range slices.Sorted(maps.Keys(x)) {
			out[k] = m.Translate(x[k])
		}
		return out
	}
	return v
}

// reLocal — локальные ID прогона в словаре процессной сессии: партии, задания,
// выполнения операций (SV-017-1, MO-004-1, TQ-001-1), кольца, крышки, клапаны.
var reLocal = regexp.MustCompile(`^(LOT-[A-Z0-9-]+|ORD-\d+|(SV|MO|EQ|AS|CV|FS|TQ|LT|CS)-\d{3}-\d+|R-\d{3}|C-\d{3}|V-\d{3})$`)

func isLocalID(s string) bool { return reLocal.MatchString(s) }
