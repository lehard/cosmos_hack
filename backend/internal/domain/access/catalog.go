package access

import (
	"slices"
	"time"

	signing "ant/internal/domain/signing"
)

// Справочная часть политики (normative/policy/policy.v1.yaml): то, что
// журнал не несёт записями policy.*, но что нужно правилам второй подписи и
// карточки решения — сферы ролей и полномочий, «только карточка» у редких
// подписантов, маршрут документа «Выдача ролей, полномочий, клейм» и
// параметры аудита по умолчанию. Это данные нормативного слоя, а не код
// (AD-15): новая роль, наследующая базовую, получает её сферу без правки кода.

// Сферы (административные домены) для выбора второй подписи (AD-11, PRD §11.16).
const (
	// DomainOrdinary — обычная выдача: одной подписью администратора безопасности.
	DomainOrdinary = "ordinary"
	// DomainQC — полномочия ОТК, клейма, ключи контролёров: вторая подпись начальника ОТК.
	DomainQC = signing.DomainQC
	// DomainProduction — полномочия и ключи производства: вторая подпись руководителя производства.
	DomainProduction = signing.DomainProduction
	// DomainAdmin — привилегии администраторов и аудита: вторая подпись Аудитора ИБ.
	DomainAdmin = signing.DomainAdmin
)

// SecondSignatureOf — полномочие второй подписи для сферы выдачи (AD-11):
// обычной выдаче вторая подпись не нужна.
func SecondSignatureOf(domain string) string {
	switch domain {
	case DomainQC:
		return signing.AuthSecondQC
	case DomainProduction:
		return signing.AuthSecondProduction
	case DomainAdmin:
		return signing.AuthSecondAudit
	}
	return ""
}

// AuthorityDef — полномочие каталога: название и сфера (qc | production | admin).
type AuthorityDef struct {
	ID     string
	Title  string
	Domain string
}

// RoleTraits — свойства роли из нормативного слоя.
type RoleTraits struct {
	// Domain — сфера роли для второй подписи: admin — привилегии
	// администраторов и аудита; qc — ОТК; пусто — обычная роль. Наследники
	// получают сферу базовой роли.
	Domain string
	// CardOnly — редкий подписант (FR-136): видит только адресованные ему
	// карточки решения, без остальных экранов.
	CardOnly bool
}

// AuditParameters — параметры аудита policy.audit.* (AD-8, AD-15):
// принадлежат Аудитору ИБ, меняются записью policy.audit.parameters_set.
type AuditParameters struct {
	CriticalTypes          []string
	CheckpointIntervalS    int
	CheckpointMaxGapS      int
	KeeperKeyFingerprint   string
	SecurityBusSubscribers []string
	// Seq, EventID, SetBy, SetAt — запись, установившая параметры (0 — затравка).
	Seq     int64
	EventID string
	SetBy   string
	SetAt   time.Time
}

// Catalog — справочная часть политики из нормативного слоя.
type Catalog struct {
	Authorities []AuthorityDef
	Roles       map[string]RoleTraits
	StampKinds  []string
	// GrantTemplate, GrantRoute — шаблон документа «Выдача ролей, полномочий,
	// клейм» и его маршрут подписей (AD-13); этапы второй подписи выбираются
	// условием grant_domains (RequiredApprovals).
	GrantTemplate string
	GrantRoute    []RouteStage
	// Audit — параметры аудита по умолчанию (до записи Аудитора ИБ).
	Audit AuditParameters
}

// Authority — полномочие каталога по id.
func (c Catalog) Authority(id string) (AuthorityDef, bool) {
	i := slices.IndexFunc(c.Authorities, func(a AuthorityDef) bool { return a.ID == id })
	if i < 0 {
		return AuthorityDef{}, false
	}
	return c.Authorities[i], true
}
