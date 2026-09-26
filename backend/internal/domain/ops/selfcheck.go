package ops

import (
	"fmt"
	"strings"
)

// Самопроверка после старта (FR-109, AD-25): итог — одна строка в логе и
// ответ /readyz. Критическая находка — система не готова; предупреждение —
// работает, но администратору есть что проверить.

// Severity — тяжесть находки самопроверки.
type Severity string

const (
	// Critical — инициализация не прошла: система не готова (/readyz 503).
	Critical Severity = "critical"
	// Warning — работает, но есть что проверить.
	Warning Severity = "warning"
)

// Проверки самопроверки.
const (
	CheckJournal    = "journal"
	CheckGenesis    = "genesis"
	CheckMigrations = "migrations"
	CheckDBRoles    = "db_roles"
	CheckRoles      = "roles"
)

// MsgOK — строка лога и итог /readyz, когда критических находок нет.
const MsgOK = "инициализация без критических ошибок"

// MsgFailed — строка лога, когда есть критические находки.
const MsgFailed = "инициализация с критическими ошибками"

// Finding — находка самопроверки.
type Finding struct {
	Check    string   `json:"check"`
	Severity Severity `json:"severity"`
	Text     string   `json:"text"`
}

// HasCritical — есть ли критические находки.
func HasCritical(fs []Finding) bool {
	for _, f := range fs {
		if f.Severity == Critical {
			return true
		}
	}
	return false
}

// Summary — строка итога: MsgOK или MsgFailed с перечнем критических находок.
func Summary(fs []Finding) string {
	if !HasCritical(fs) {
		return MsgOK
	}
	var parts []string
	for _, f := range fs {
		if f.Severity == Critical {
			parts = append(parts, f.Check+": "+f.Text)
		}
	}
	return MsgFailed + ": " + strings.Join(parts, "; ")
}

// GenesisFindings — генезис (AD-33): блок журнала journal.genesis.recorded.
// Больше одного — критично (второй генезис — подмена доверия). Нет генезиса:
// критично, если он обязателен (роль init в сборке); иначе — предупреждение
// для непустого журнала (записи идут без блока доверия).
func GenesisFindings(headSeq int64, genesisSeqs []int64, required bool) []Finding {
	switch n := len(genesisSeqs); {
	case n > 1:
		return []Finding{{CheckGenesis, Critical, fmt.Sprintf("генезис записан %d раз (seq %v) — доверие к ключам неоднозначно", n, genesisSeqs)}}
	case n == 1:
		return nil
	case required:
		return []Finding{{CheckGenesis, Critical, "генезиса нет — выполните ant -role=init (ключи, стартовые справочники, блок доверия)"}}
	case headSeq > 0:
		return []Finding{{CheckGenesis, Warning, fmt.Sprintf("генезиса нет, в журнале %d записей — ключи и стартовые данные без блока доверия (роль init)", headSeq)}}
	}
	return nil
}

// Migration — состояние миграций модуля: версия в базе и последняя в бинарнике.
type Migration struct {
	Module  string
	Current int64
	Target  int64
	// Pending — неприменённых миграций бинарника.
	Pending int
	// Err — состояние не удалось прочитать (нет прав, нет таблицы версий…).
	Err string
}

// MigrationFindings — миграции модулей (AD-1): неприменённые — критично;
// база новее бинарника или состояние не прочитано — предупреждение.
func MigrationFindings(ms []Migration) []Finding {
	var out []Finding
	for _, m := range ms {
		switch {
		case m.Err != "":
			out = append(out, Finding{CheckMigrations, Warning, fmt.Sprintf("модуль %s: состояние миграций не прочитано: %s", m.Module, m.Err)})
		case m.Pending > 0 || m.Current < m.Target:
			out = append(out, Finding{CheckMigrations, Critical, fmt.Sprintf("модуль %s: не применено миграций %d (в базе версия %d, в сборке %d) — выполните ant -role=migrate", m.Module, max(m.Pending, 1), m.Current, m.Target)})
		case m.Current > m.Target:
			out = append(out, Finding{CheckMigrations, Warning, fmt.Sprintf("модуль %s: версия схемы в базе %d новее сборки %d", m.Module, m.Current, m.Target)})
		}
	}
	return out
}

// Privilege — право роли БД на объект: есть ли и должно ли быть (AD-1).
type Privilege struct {
	Role      string
	Object    string
	Privilege string
	Granted   bool
	Want      bool
}

// DBRoleFindings — роли БД (AD-1, AD-2): роли ant_owner, ant_app,
// ant_verifier должны быть; ant_app не меняет журнал, ant_verifier только читает.
func DBRoleFindings(missing []string, privs []Privilege) []Finding {
	var out []Finding
	for _, r := range missing {
		out = append(out, Finding{CheckDBRoles, Critical, "нет роли БД " + r + " — выполните ant -role=migrate"})
	}
	for _, p := range privs {
		if p.Granted == p.Want {
			continue
		}
		if p.Granted {
			out = append(out, Finding{CheckDBRoles, Critical, fmt.Sprintf("у роли %s есть право %s на %s — журнал не защищён от изменения (AD-1, AD-2)", p.Role, p.Privilege, p.Object)})
		} else {
			out = append(out, Finding{CheckDBRoles, Critical, fmt.Sprintf("у роли %s нет права %s на %s", p.Role, p.Privilege, p.Object)})
		}
	}
	return out
}

// ProcessRole — роль процесса ant (AD-25) глазами самопроверки.
type ProcessRole struct {
	Name string
	// Pending — роль-заглушка до своего эпика.
	Pending bool
	// Leader — роль с копией-лидером по аренде; Held — аренда взята
	// какой-либо копией к моменту проверки.
	Leader bool
	Held   bool
}

// RoleFindings — роли процесса: заглушки и роли-лидеры без аренды —
// предупреждения (лидером может быть копия, которая ещё стартует).
func RoleFindings(roles []ProcessRole) []Finding {
	var out []Finding
	for _, r := range roles {
		switch {
		case r.Pending:
			out = append(out, Finding{CheckRoles, Warning, "роль " + r.Name + " — заглушка, реализация в работе"})
		case r.Leader && !r.Held:
			out = append(out, Finding{CheckRoles, Warning, "роль " + r.Name + ": аренду лидера пока не взяла ни одна копия"})
		}
	}
	return out
}
