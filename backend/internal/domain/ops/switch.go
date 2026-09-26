package ops

import (
	"slices"
	"strings"
	"time"
)

// Управление интеграциями (FR-157, AD-47; эпик 48). Установка — конфигурация
// (integrations.enabled, адреса и секреты — в файлах), состояние включения —
// записи журнала ops.integration.state_set: администратор включает,
// выключает и переключает «стенд ↔ реальная система» единолично (Д-71).

// Состояния интеграции (ops.integration.state_set.state).
const (
	// SwitchEnabled — включена, обмен с реальной системой.
	SwitchEnabled = "enabled"
	// SwitchDisabled — выключена: исходящие копятся в очереди, входящие
	// отвергаются приёмом, входящий адаптер не опрашивает.
	SwitchDisabled = "disabled"
	// SwitchStand — включена, обмен со стендом (эмулятором).
	SwitchStand = "stand"
)

// ProfileProd — рабочий профиль: стенд запрещён доменным гардом (AD-47).
const ProfileProd = "prod"

// KnownSystems — внешние системы экрана «Интеграции» в порядке показа
// (FR-157): неустановленные видны как «не установлена».
var KnownSystems = []string{"onec", "galaktika", "mes", "kompas", "skud", "ca", "visionqc", "operatorvision", "partner"}

// systemSources — префиксы source_id, под которыми входящие системы приходят
// в приём (шлюзы erp.‹система›, mes.‹система›, cad.kompas; партнёры —
// partner:‹код›). Выключенная система — эти источники отвергаются приёмом.
var systemSources = map[string][]string{
	"onec":           {"erp.onec"},
	"galaktika":      {"erp.galaktika"},
	"mes":            {"mes."},
	"kompas":         {"cad.kompas"},
	"skud":           {"skud."},
	"visionqc":       {"visionqc"},
	"operatorvision": {"operatorvision"},
	"partner":        {"partner:"},
}

// SystemOfSource — внешняя система, от которой пришёл источник source_id;
// "" — источник не внешней системы (станок, камера, терминал).
func SystemOfSource(sourceID string) string {
	for _, sys := range KnownSystems {
		for _, p := range systemSources[sys] {
			if sourceID == p || strings.HasPrefix(sourceID, p) {
				return sys
			}
		}
	}
	return ""
}

// Installed — установленная конфигурацией интеграция и какие режимы для неё
// заданы адресами (стенд, реальная система).
type Installed struct {
	System string
	Stand  bool
	Real   bool
}

// StateRecord — решение ops.integration.state_set из журнала.
type StateRecord struct {
	Seq      int64
	EventID  string
	System   string
	State    string
	Previous string
	Reason   string
	Actor    string
	At       time.Time
}

// SwitchState — действующее состояние интеграции.
type SwitchState struct {
	System    string
	Installed bool
	// Stand, Real — какие режимы заданы конфигурацией.
	Stand, Real bool
	State       string
	// Default — решений не было: состояние по умолчанию профиля.
	Default bool
	// Last — последнее решение (nil — не было).
	Last *StateRecord
}

// DefaultState — состояние без решений (AD-47): неустановленная — выключена;
// в prod — реальная система; в demo и fixtures — стенд, если он задан.
func DefaultState(profile string, in Installed, installed bool) string {
	switch {
	case !installed:
		return SwitchDisabled
	case profile == ProfileProd:
		if in.Real {
			return SwitchEnabled
		}
		return SwitchDisabled
	case in.Stand:
		return SwitchStand
	case in.Real:
		return SwitchEnabled
	}
	return SwitchDisabled
}

// States сводит действующее состояние каждой известной системы и каждой
// установленной сверх списка: последнее решение побеждает; решение о
// неустановленной системе не действует (установку задаёт конфигурация);
// «стенд» в prod не действует, даже если записан до смены профиля.
func States(profile string, installed []Installed, records []StateRecord) []SwitchState {
	inst := map[string]Installed{}
	order := slices.Clone(KnownSystems)
	for _, in := range installed {
		inst[in.System] = in
		if !slices.Contains(order, in.System) {
			order = append(order, in.System)
		}
	}
	last := map[string]StateRecord{}
	for _, r := range records {
		if p, ok := last[r.System]; !ok || r.Seq > p.Seq {
			last[r.System] = r
		}
	}
	out := make([]SwitchState, 0, len(order))
	for _, sys := range order {
		in, ok := inst[sys]
		st := SwitchState{System: sys, Installed: ok, Stand: in.Stand, Real: in.Real}
		st.State, st.Default = DefaultState(profile, in, ok), true
		if r, has := last[sys]; has {
			rr := r
			st.Last = &rr
			if ok && Allowed(profile, in, r.State) {
				st.State, st.Default = r.State, false
			}
		}
		out = append(out, st)
	}
	return out
}

// Allowed — состояние допустимо для установленной интеграции в профиле.
func Allowed(profile string, in Installed, state string) bool {
	switch state {
	case SwitchDisabled:
		return true
	case SwitchStand:
		return profile != ProfileProd && in.Stand
	case SwitchEnabled:
		return in.Real
	}
	return false
}

// Rejection — отказ гарда смены состояния (коды семейства ops, errors.yaml).
type Rejection struct {
	Code   string
	Params map[string]string
}

// Коды отказа гарда (contracts/errors.yaml).
const (
	CodeNotInstalled    = "ops.integration_not_installed"
	CodeStandForbidden  = "ops.stand_forbidden"
	CodeModeUnavailable = "ops.integration_mode_unavailable"
	CodeUnchanged       = "ops.integration_state_unchanged"
)

// CheckSet — доменный гард ops.integration.set (AD-47): только
// установленная система; стенд в prod запрещён; режим должен быть задан
// адресом в конфигурации; то же состояние повторно не пишется.
func CheckSet(profile string, cur SwitchState, target string) *Rejection {
	p := map[string]string{"system": cur.System, "state": target, "profile": profile}
	in := Installed{System: cur.System, Stand: cur.Stand, Real: cur.Real}
	switch {
	case !cur.Installed:
		return &Rejection{Code: CodeNotInstalled, Params: p}
	case target == SwitchStand && profile == ProfileProd:
		return &Rejection{Code: CodeStandForbidden, Params: p}
	case !Allowed(profile, in, target):
		return &Rejection{Code: CodeModeUnavailable, Params: p}
	case cur.State == target:
		return &Rejection{Code: CodeUnchanged, Params: p}
	}
	return nil
}

// SourceBlock — почему приём отвергает источник: решение об источнике
// (ops.source.disabled) или выключенная система, к которой он относится.
type SourceBlock struct {
	Blocked bool
	Reason  string
}

// SourceBlocked — отвергать ли входящие от source_id (AD-28, AD-47):
// disabledSources — источники с последним решением ops.source.disabled
// (→ основание); states — действующие состояния систем.
func SourceBlocked(sourceID string, disabledSources map[string]string, states []SwitchState) SourceBlock {
	if r, ok := disabledSources[sourceID]; ok {
		return SourceBlock{Blocked: true, Reason: "источник отключён администратором: " + r}
	}
	sys := SystemOfSource(sourceID)
	if sys == "" {
		return SourceBlock{}
	}
	for _, st := range states {
		if st.System == sys && st.Installed && st.State == SwitchDisabled {
			return SourceBlock{Blocked: true, Reason: "интеграция " + sys + " выключена"}
		}
	}
	return SourceBlock{}
}
