package signing

import (
	"context"
	"slices"
	"strings"

	dom "ant/internal/domain/signing"
)

// StaticAuthorities — полномочия второй подписи и заверения бумаги по
// стартовой политике normative/policy/policy.v1.yaml (AD-11, AD-43) до
// модуля access (эпик 26): начальник ОТК — вторая подпись ОТК и заверение;
// руководитель производства — вторая подпись производства; Аудитор ИБ —
// вторая подпись администраторов и аудита; мастера — заверение бумаги.
// Демо-персоны demo-signer — те же псевдонимы.
type StaticAuthorities map[string][]string

// DefaultAuthorities — полномочия демо-политики.
func DefaultAuthorities() StaticAuthorities {
	return StaticAuthorities{
		"HQC-01": {dom.AuthSecondQC, dom.AuthPaperAttestation},
		"PM-01":  {dom.AuthSecondProduction, dom.AuthPaperAttestation},
		"AUD-01": {dom.AuthSecondAudit},
		"FOR-SK": {dom.AuthPaperAttestation}, "FOR-MC": {dom.AuthPaperAttestation},
		"FOR-WC": {dom.AuthPaperAttestation}, "FOR-AC": {dom.AuthPaperAttestation},
		"HWS-WC": {dom.AuthPaperAttestation}, "HWS-AC": {dom.AuthPaperAttestation},
	}
}

// Has — есть ли полномочие.
func (a StaticAuthorities) Has(_ context.Context, person, authority string, _ int64) (bool, error) {
	return slices.Contains(a[person], authority), nil
}

// Domain — сфера сотрудника по псевдониму: контролёры и ОТК — qc;
// администраторы и аудит — admin; прочие — production.
func (StaticAuthorities) Domain(_ context.Context, person string) string {
	switch {
	case strings.HasPrefix(person, "INS-"), strings.HasPrefix(person, "HQC-"), strings.HasPrefix(person, "QC-"):
		return dom.DomainQC
	case strings.HasPrefix(person, "ADM-"), strings.HasPrefix(person, "AUD-"):
		return dom.DomainAdmin
	}
	return dom.DomainProduction
}
