package access

import "slices"

// Обязательные подписи (AD-43): одна чистая функция RequiredApprovals над
// маршрутом шаблона документа, контекстом решения и политикой на basis.
//
// ЗАГОТОВКА эпика 28 для эпика 26 (владелец политики). Сигнатура — по AD-43:
// RequiredApprovals(маршрут шаблона, контекст решения, политика@basis). Пока
// политика не проекция журнала (эпик 26), Policy пуста, а функция только
// отбирает этапы маршрута по условиям `when` и считает число подписей. Эпик
// 26 наполняет Policy (полномочия, клейма, делегирование, эскалация на k-м
// предъявлении) и уточняет этапы, не меняя сигнатуры: её уже вызывают
// documents (при черновике) и верификатор.

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

// RouteStage — этап маршрута подписей в шаблоне документа (нормативный слой,
// normative/documents): кто, сколько, каким уровнем и при каком условии (AD-13).
type RouteStage struct {
	Stage               int             `json:"stage"`
	Title               string          `json:"title"`
	Role                string          `json:"role,omitempty"`
	AuthorityID         string          `json:"authority_id"`
	StampKind           string          `json:"stamp_kind,omitempty"`
	Quorum              string          `json:"quorum"`
	K                   int             `json:"k,omitempty"`
	SignatureLevel      int             `json:"signature_level"`
	PaperAllowed        bool            `json:"paper_allowed"`
	AttesterAuthorityID string          `json:"attester_authority_id,omitempty"`
	ExternalParty       string          `json:"external_party,omitempty"`
	BySource            bool            `json:"by_source,omitempty"`
	Separation          []string        `json:"separation,omitempty"`
	When                *StageCondition `json:"when,omitempty"`
}

// StageCondition — условие этапа: решения (режим 4 — ремонт, «как есть») и
// приёмка представителем заказчика (режим 5, FR-50).
type StageCondition struct {
	Decisions          []string `json:"decisions,omitempty"`
	CustomerAcceptance *bool    `json:"customer_acceptance,omitempty"`
}

// ApprovalContext — контекст решения, для которого строится маршрут.
type ApprovalContext struct {
	// Decision — оформляемое решение (repair, use_as_is, scrap…); пусто — любое.
	Decision string
	// CustomerAcceptance — продукция с приёмкой представителя заказчика (режим 5).
	CustomerAcceptance bool
}

// Policy — политика доступа на basis_seq. ЗАГОТОВКА: наполняет эпик 26
// (полномочия с рамками, клейма, делегирование, эскалация).
type Policy struct{}

// ApprovalStage — этап обязательных подписей, замороженный при
// document.version.drafted (AD-43): номер по порядку после отбора условий и
// сколько засчитанных подписей нужно.
type ApprovalStage struct {
	Stage               int      `json:"stage"`
	Title               string   `json:"title,omitempty"`
	Role                string   `json:"role,omitempty"`
	AuthorityID         string   `json:"authority_id"`
	StampKind           string   `json:"stamp_kind,omitempty"`
	Quorum              string   `json:"quorum"`
	K                   int      `json:"k,omitempty"`
	Required            int      `json:"required_count"`
	SignatureLevel      int      `json:"signature_level"`
	PaperAllowed        bool     `json:"paper_allowed"`
	AttesterAuthorityID string   `json:"attester_authority_id,omitempty"`
	ExternalParty       string   `json:"external_party,omitempty"`
	BySource            bool     `json:"by_source,omitempty"`
	Separation          []string `json:"separation,omitempty"`
}

// Has — у этапа есть правило разделения обязанностей rule.
func (s ApprovalStage) Has(rule string) bool { return slices.Contains(s.Separation, rule) }

// RequiredApprovals — обязательные подписи документа (AD-43): этапы маршрута
// шаблона, условия которых выполнены для контекста решения, в порядке
// маршрута с номерами 1…n. Бумага на этапе допустима, только если задан
// заверитель (AD-43: «если заверитель не задан, бумага на этапе запрещена»).
// Чистая функция: её вызывают documents при черновике, api для объяснения
// прав и «Запросить решение», верификатор (AD-9).
func RequiredApprovals(route []RouteStage, c ApprovalContext, _ Policy) []ApprovalStage {
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
		out = append(out, st)
	}
	return out
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
	return true
}
