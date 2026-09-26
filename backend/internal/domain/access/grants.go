package access

import (
	"slices"
	"strings"
	"time"

	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Изменение политики и вторая подпись независимой стороны (AD-11, AD-15,
// PRD §11.16, FR-85). Выдача — документ «Выдача ролей, полномочий, клейм» с
// маршрутом подписей (каталог документов спайна): администратор безопасности
// → обычная роль без второй подписи; полномочия ОТК и клейма — начальник ОТК;
// полномочия производства — руководитель производства; привилегии
// администраторов и аудита — Аудитор ИБ. «Выдача себе» проверяется по
// эффективным правам: расширение собственных прав и расширение прав
// администратора — привилегированная выдача, кто бы её ни делал.
//
// Одни и те же функции вызывают api (до записи), RequiredApprovals (этапы
// документа выдачи) и верификатор (право подписанта на момент подписи).

// Виды изменения политики (GrantPolicy.kind).
const (
	ChangeRole      = "role"
	ChangeAuthority = "authority"
	ChangeStamp     = "stamp"
)

// PolicyChange — выдача или отзыв роли, полномочия или клейма.
type PolicyChange struct {
	Kind   string
	Revoke bool
	// PersonID — кому выдаётся (или у кого отзывается).
	PersonID string
	// SubjectID — роль, полномочие или id клейма.
	SubjectID string
	// InspectionKind — вид контроля клейма.
	InspectionKind string
	Scope          string
	OrderRef       string
	ValidFrom      time.Time
	ValidUntil     time.Time
}

// With — политика после изменения c (для оценки «расширяет ли права»; в
// журнал пишет только api записью policy.*).
func (p Policy) With(c PolicyChange) Policy {
	n := p.Clone()
	if c.Revoke {
		at := c.ValidFrom
		switch c.Kind {
		case ChangeRole:
			for i, a := range n.Assignments {
				if a.PersonID == c.PersonID && a.RoleID == c.SubjectID && (c.Scope == "" || a.Scope == c.Scope) && closes(a.ValidUntil, at) {
					n.Assignments[i].ValidUntil = at
				}
			}
		case ChangeAuthority:
			for i, a := range n.Authorities {
				if a.PersonID == c.PersonID && a.AuthorityID == c.SubjectID && (c.Scope == "" || a.Scope == c.Scope) && closes(a.ValidUntil, at) {
					n.Authorities[i].ValidUntil = at
				}
			}
		case ChangeStamp:
			for i, s := range n.Stamps {
				if s.PersonID == c.PersonID && s.StampID == c.SubjectID && closes(s.ValidUntil, at) {
					n.Stamps[i].ValidUntil, n.Stamps[i].Revoked = at, true
				}
			}
		}
		return n
	}
	switch c.Kind {
	case ChangeRole:
		n.Assignments = append(n.Assignments, Assignment{PersonID: c.PersonID, RoleID: c.SubjectID, Scope: c.Scope, ValidFrom: c.ValidFrom, ValidUntil: c.ValidUntil})
	case ChangeAuthority:
		n.Authorities = append(n.Authorities, Authority{PersonID: c.PersonID, AuthorityID: c.SubjectID, Scope: c.Scope, ValidFrom: c.ValidFrom, ValidUntil: c.ValidUntil})
	case ChangeStamp:
		n.Stamps = append(n.Stamps, Stamp{StampID: c.SubjectID, PersonID: c.PersonID, Kind: c.InspectionKind, Scope: c.Scope,
			OrderRef: c.OrderRef, ValidFrom: c.ValidFrom, ValidUntil: c.ValidUntil})
	}
	return n
}

// Assessment — оценка изменения политики: сфера, нужна ли вторая подпись и чья.
type Assessment struct {
	// Domain — ordinary | qc | production | admin.
	Domain string
	// SecondAuthority — полномочие второй подписи; пусто — одной подписью.
	SecondAuthority string
	// SelfGrant — изменение расширяет права самого инициатора.
	SelfGrant bool
	// AdminExpansion — расширяются права администратора или аудита.
	AdminExpansion bool
	// Expands — какие права добавляет изменение получателю (пусто — не расширяет).
	Expands Rights
	// Excluded — кто не может ставить вторую подпись (инициатор и получатель).
	Excluded []string
	// Reason — объяснение по-русски.
	Reason string
}

// Assess — чья вторая подпись нужна для изменения c, которое делает
// initiator, в момент at (AD-11). Отзыв — защитное действие (AD-27): одной
// подписью. Выдача, которая не расширяет права получателя, — одной подписью.
func Assess(p Policy, c PolicyChange, initiator string, at time.Time) Assessment {
	out := Assessment{Domain: DomainOrdinary, Excluded: excluded(initiator, c.PersonID)}
	if c.Revoke {
		out.Reason = "отзыв — защитное действие, одной подписью"
		return out
	}
	t := at
	if c.ValidFrom.After(t) {
		t = c.ValidFrom
	}
	out.Expands = Expands(p.EffectiveRights(c.PersonID, t), p.With(c).EffectiveRights(c.PersonID, t))
	if len(out.Expands) == 0 {
		out.Reason = "права получателя не расширяются"
		return out
	}
	switch c.Kind {
	case ChangeRole:
		if p.RoleTraitsOf(c.SubjectID).Domain == DomainAdmin {
			out.Domain = DomainAdmin
		}
	case ChangeAuthority:
		if d, ok := p.Catalog.Authority(c.SubjectID); ok && d.Domain != "" {
			out.Domain = d.Domain
		} else {
			out.Domain = DomainProduction
		}
	case ChangeStamp:
		out.Domain = DomainQC
	}
	// Расширение прав администратора или аудита — привилегированная выдача, кто бы её ни делал.
	if p.PersonDomain(c.PersonID, t) == DomainAdmin {
		out.AdminExpansion, out.Domain = true, DomainAdmin
	}
	// Выдача себе — по эффективным правам: инициатор расширяет собственные права.
	if initiator != "" && initiator == c.PersonID {
		out.SelfGrant = true
		if out.Domain == DomainOrdinary {
			out.Domain = p.PersonDomain(initiator, t)
		}
	}
	out.SecondAuthority = SecondSignatureOf(out.Domain)
	switch {
	case out.SecondAuthority == "":
		out.Reason = "обычная роль — одной подписью администратора безопасности"
	case out.SelfGrant:
		out.Reason = "выдача себе: расширение собственных прав — только со второй подписью независимой стороны (" + DomainTitle(out.Domain) + ")"
	case out.AdminExpansion:
		out.Reason = "расширение прав администратора или аудита — вторая подпись Аудитора ИБ"
	default:
		out.Reason = "привилегированная выдача — вторая подпись: " + DomainTitle(out.Domain)
	}
	return out
}

// DomainTitle — кто ставит вторую подпись для сферы (по-русски).
func DomainTitle(domain string) string {
	switch domain {
	case DomainQC:
		return "начальник ОТК"
	case DomainProduction:
		return "руководитель производства"
	case DomainAdmin:
		return "Аудитор ИБ"
	}
	return "не нужна"
}

func excluded(ids ...string) []string {
	var out []string
	for _, id := range ids {
		if id != "" && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

// Approval — подпись под документом выдачи (этап маршрута): кто и когда
// (доменное время, AD-37) и на какой позиции журнала.
type Approval struct {
	PersonID string
	Stage    int
	At       time.Time
	Seq      int64
}

// CheckApprovals — гард второй подписи (AD-11, AD-39): для оценки a нужна
// хотя бы одна подпись независимой стороны — не инициатора и не получателя —
// с полномочием a.SecondAuthority в области выдачи на момент своей подписи
// (pol — политика на basis; верификатор передаёт политику на seq подписи).
// Отказ: access.self_grant (выдача себе) или access.signature_required.
func CheckApprovals(pol Policy, a Assessment, c PolicyChange, approvals []Approval) error {
	if a.SecondAuthority == "" {
		return nil
	}
	for _, ap := range approvals {
		if slices.Contains(a.Excluded, ap.PersonID) {
			continue
		}
		if pol.HasAuthority(ap.PersonID, a.SecondAuthority, c.Scope, ap.At) {
			return nil
		}
	}
	who := DomainTitle(a.Domain) + " (полномочие " + a.SecondAuthority + ")"
	if a.SelfGrant {
		return kernel.Refuse(errcodes.AccessSelfGrant, "who", who)
	}
	return kernel.Refuse(errcodes.AccessSignatureRequired, "who", who)
}

// PolicyStream — поток записи политики (catalog streams: `policy:‹область›`):
// выдачи и отзывы — в области выдачи, определения ролей и параметры аудита —
// в корне предприятия.
func PolicyStream(scope string) string {
	s := strings.Trim(scope, "/")
	if s == "" || s == "*" {
		s = "*"
	}
	return "policy:" + s
}

// SubjectStreams — потоки политики, от которых зависят права сотрудника
// (AD-39: «политика субъекта не менялась после policy_seq»): корень
// (определения ролей, параметры аудита, выдачи на всё предприятие) и
// области его действующих назначений, полномочий и клейм. Отзыв пишется в
// область отзываемого — поэтому он всегда попадает в эти потоки.
func (p Policy) SubjectStreams(personID string, at time.Time) []string {
	out := []string{PolicyStream(p.Root)}
	add := func(scope string) {
		if s := PolicyStream(scope); !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	for _, a := range p.AssignmentsOf(personID, at) {
		add(a.Scope)
	}
	for _, a := range p.AuthoritiesOf(personID, at) {
		add(a.Scope)
	}
	for _, s := range p.StampsOf(personID, at) {
		add(s.Scope)
	}
	return out
}

// GrantDecisionPrefix — префикс решения документа выдачи (RequestDecision.decision).
const GrantDecisionPrefix = "grant:"

// FormatGrantDecision — решение документа выдачи в одну строку:
// `grant:‹вид›:‹роль|полномочие|клеймо›:‹кому›:‹область›` (для клейма —
// `‹id клейма›/‹вид контроля›`). По этой строке documents строит контекст
// RequiredApprovals, а access сверяет, что закрытый документ выдаёт именно это.
func FormatGrantDecision(c PolicyChange) string {
	subject := c.SubjectID
	if c.Kind == ChangeStamp && c.InspectionKind != "" {
		subject += "/" + c.InspectionKind
	}
	return GrantDecisionPrefix + c.Kind + ":" + subject + ":" + c.PersonID + ":" + c.Scope
}

// ParseGrantDecision — обратное к FormatGrantDecision; не решение выдачи — false.
func ParseGrantDecision(s string) (PolicyChange, bool) {
	rest, ok := strings.CutPrefix(s, GrantDecisionPrefix)
	if !ok {
		return PolicyChange{}, false
	}
	parts := strings.SplitN(rest, ":", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return PolicyChange{}, false
	}
	c := PolicyChange{Kind: parts[0], SubjectID: parts[1], PersonID: parts[2], Scope: parts[3]}
	switch c.Kind {
	case ChangeRole, ChangeAuthority:
	case ChangeStamp:
		c.SubjectID, c.InspectionKind, _ = strings.Cut(parts[1], "/")
	default:
		return PolicyChange{}, false
	}
	return c, true
}
