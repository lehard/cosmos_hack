package access

import (
	"slices"
	"time"
)

// Обязательные подписи (AD-43, AD-13): одна чистая функция
// RequiredApprovals(маршрут шаблона, контекст решения, политика@basis) → этапы
// (полномочие, клеймо, сколько, уровень, разделение обязанностей, внешняя
// сторона, допуск бумаги) и кто может подписать каждый этап по политике на
// basis. Её вызывают documents при черновике (набор замораживается в
// каноническом JSON документа — подписант видит, кто ещё подписывает), api
// для объяснения прав и «Запросить решение» (FR-136, FR-146) и верификатор
// (AD-9). Сигнатура — та, на которой работает заготовка эпика 28: вызовы с
// пустой политикой (Policy{}) дают те же этапы без кандидатов.

// Кворумы этапа (AD-13).
const (
	QuorumOne  = "one"
	QuorumAll  = "all"
	QuorumKOfN = "k_of_n"
)

// Правила разделения обязанностей этапа (FR-56, AD-43).
const (
	// SeparationDistinctSigners — один человек подписывает не больше одного этапа версии.
	SeparationDistinctSigners = "distinct_signers"
	// SeparationNotItemParticipant — подписант не участвовал в изготовлении изделия (FR-56).
	SeparationNotItemParticipant = "not_item_participant"
)

// RouteStage — этап маршрута подписей в шаблоне документа (нормативный слой):
// кто, сколько, каким уровнем и при каком условии (AD-13).
type RouteStage struct {
	Stage               int             `json:"stage" yaml:"stage"`
	Title               string          `json:"title" yaml:"title"`
	Role                string          `json:"role,omitempty" yaml:"role"`
	AuthorityID         string          `json:"authority_id" yaml:"authority_id"`
	StampKind           string          `json:"stamp_kind,omitempty" yaml:"stamp_kind"`
	Quorum              string          `json:"quorum" yaml:"quorum"`
	K                   int             `json:"k,omitempty" yaml:"k"`
	SignatureLevel      int             `json:"signature_level" yaml:"signature_level"`
	PaperAllowed        bool            `json:"paper_allowed" yaml:"paper_allowed"`
	AttesterAuthorityID string          `json:"attester_authority_id,omitempty" yaml:"attester_authority_id"`
	ExternalParty       string          `json:"external_party,omitempty" yaml:"external_party"`
	BySource            bool            `json:"by_source,omitempty" yaml:"by_source"`
	Separation          []string        `json:"separation,omitempty" yaml:"separation"`
	When                *StageCondition `json:"when,omitempty" yaml:"when"`
}

// StageCondition — условие этапа: решения (режим 4 — ремонт, «как есть»),
// приёмка представителем заказчика (режим 5, FR-50) и сфера выдачи прав
// (документ «Выдача ролей, полномочий, клейм», AD-11).
type StageCondition struct {
	Decisions          []string `json:"decisions,omitempty" yaml:"decisions"`
	CustomerAcceptance *bool    `json:"customer_acceptance,omitempty" yaml:"customer_acceptance"`
	// GrantDomains — этап нужен, если сфера выдачи (Assess) — одна из этих
	// (qc | production | admin); обычная выдача (ordinary) — без второй подписи.
	GrantDomains []string `json:"grant_domains,omitempty" yaml:"grant_domains"`
}

// ApprovalContext — контекст решения, для которого строится маршрут.
type ApprovalContext struct {
	// Decision — оформляемое решение (repair, use_as_is, scrap…; для выдачи
	// прав — FormatGrantDecision); пусто — любое.
	Decision string
	// CustomerAcceptance — продукция с приёмкой представителя заказчика (режим 5).
	CustomerAcceptance bool
	// Initiator — кто оформляет решение (инициатор не подписывает этапы
	// независимой стороны документа выдачи).
	Initiator string
	// Scope — область объекта решения: кандидаты — с полномочием в ней.
	Scope string
	// At — доменный момент basis (сроки полномочий, AD-37); нулевой — без сроков.
	At time.Time
	// Grant — выдача прав (документ выдачи); nil — из Decision, если это решение выдачи.
	Grant *PolicyChange
	// GrantDomain — сфера выдачи; пусто — вычисляется Assess по политике.
	GrantDomain string
	// Excluded — кто не подписывает этапы (кроме этапов «по источнику»).
	Excluded []string
}

// ApprovalStage — этап обязательных подписей, замороженный при
// document.version.drafted (AD-43): номер по порядку после отбора условий,
// сколько засчитанных подписей нужно и кто может подписать.
type ApprovalStage struct {
	Stage               int      `json:"stage"`
	Title               string   `json:"title,omitempty"`
	Role                string   `json:"role,omitempty"`
	AuthorityID         string   `json:"authority_id"`
	StampKind           string   `json:"stamp_kind,omitempty"`
	Quorum              string   `json:"quorum"`
	K                   int      `json:"k,omitempty"`
	Required            int      `json:"required"`
	SignatureLevel      int      `json:"signature_level"`
	PaperAllowed        bool     `json:"paper_allowed"`
	AttesterAuthorityID string   `json:"attester_authority_id,omitempty"`
	ExternalParty       string   `json:"external_party,omitempty"`
	BySource            bool     `json:"by_source,omitempty"`
	Separation          []string `json:"separation,omitempty"`
	// Candidates — кто может подписать этап по политике на basis (псевдонимы в
	// порядке политики); пусто — политика не передана или подписантов нет.
	Candidates []string `json:"candidates,omitempty"`
}

// Has — у этапа есть правило разделения обязанностей rule.
func (s ApprovalStage) Has(rule string) bool { return slices.Contains(s.Separation, rule) }

// RequiredApprovals — обязательные подписи документа (AD-43): этапы маршрута
// шаблона, условия которых выполнены для контекста решения, в порядке
// маршрута с номерами 1…n. Бумага на этапе допустима, только если задан
// заверитель (AD-43: «если заверитель не задан, бумага на этапе запрещена»).
// Кандидаты этапа — сотрудники политики, которые вправе подписать его в
// момент c.At (CanSign), кроме исключённых (для документа выдачи —
// инициатор и получатель: вторая подпись — от независимой стороны, AD-11).
func RequiredApprovals(route []RouteStage, c ApprovalContext, pol Policy) []ApprovalStage {
	if c.Grant == nil {
		if g, ok := ParseGrantDecision(c.Decision); ok {
			c.Grant = &g
		}
	}
	excludedPersons := slices.Clone(c.Excluded)
	if c.Grant != nil {
		if c.GrantDomain == "" {
			c.GrantDomain = Assess(pol, *c.Grant, c.Initiator, c.At).Domain
		}
		if c.Scope == "" {
			c.Scope = c.Grant.Scope
		}
		excludedPersons = append(excludedPersons, excluded(c.Initiator, c.Grant.PersonID)...)
	}
	out := make([]ApprovalStage, 0, len(route))
	for _, r := range route {
		if !applies(r.When, c) {
			continue
		}
		st := ApprovalStage{
			Stage: len(out) + 1, Title: r.Title, Role: r.Role, AuthorityID: r.AuthorityID, StampKind: r.StampKind,
			Quorum: r.Quorum, K: r.K, SignatureLevel: r.SignatureLevel, PaperAllowed: r.PaperAllowed && r.AttesterAuthorityID != "",
			AttesterAuthorityID: r.AttesterAuthorityID, ExternalParty: r.ExternalParty, BySource: r.BySource,
			Separation: slices.Clone(r.Separation),
		}
		if st.Quorum == "" {
			st.Quorum = QuorumOne
		}
		st.Required = 1
		if (st.Quorum == QuorumKOfN || st.Quorum == QuorumAll) && st.K > 1 {
			st.Required = st.K
		}
		if !st.BySource {
			for _, x := range pol.Persons {
				if !slices.Contains(excludedPersons, x.ID) && pol.CanSign(x.ID, st, c.Scope, c.At) {
					st.Candidates = append(st.Candidates, x.ID)
				}
			}
		}
		out = append(out, st)
	}
	return out
}

// CanSign — сотрудник вправе подписать этап в момент at (AD-43): роль этапа
// (с наследованием), полномочие этапа (или роль с тем же id — этапы вида
// «мастер участка») в области объекта и действующее клеймо, если этап его
// требует (FR-145). Верификатор вызывает её на политике на seq подписи.
func (p Policy) CanSign(personID string, st ApprovalStage, scope string, at time.Time) bool {
	if st.Role != "" && !p.HasRole(personID, st.Role, "", at) {
		return false
	}
	if st.AuthorityID != "" && !p.HasAuthority(personID, st.AuthorityID, scope, at) && !p.HasRole(personID, st.AuthorityID, scope, at) {
		return false
	}
	if st.StampKind != "" {
		if _, ok := p.StampFor(personID, st.StampKind, scope, at); !ok {
			return false
		}
	}
	return true
}

func applies(w *StageCondition, c ApprovalContext) bool {
	if w == nil {
		return true
	}
	if len(w.Decisions) > 0 && !slices.Contains(w.Decisions, c.Decision) {
		return false
	}
	if w.CustomerAcceptance != nil && *w.CustomerAcceptance != c.CustomerAcceptance {
		return false
	}
	if len(w.GrantDomains) > 0 && !slices.Contains(w.GrantDomains, c.GrantDomain) {
		return false
	}
	return true
}
