package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	accessdom "ant/internal/domain/access"
)

// authority — право подписанта на момент подписи (FR-85, AD-9, AD-11, AD-15;
// эпик 26): политика сворачивается из журнала той же функцией, что у api
// (domain/access.Policy.Apply), и для каждого решения человека об изменении
// политики проверяется по политике ДО этой записи: у автора была действующая
// роль с правом этой операции; выдача, расширяющая права (сфера ОТК,
// производства, администраторов и аудита, выдача себе, расширение прав
// администратора), несёт вторую подпись независимой стороны с полномочием
// второй подписи на момент выдачи (Assess, CheckApprovals).
//
// Начало свёртки — политика генезиса (записи с происхождением genesis) или
// Input.Policy (стартовая политика нормативного слоя). Ни того ни другого —
// «не проверяемо».
func (v *run) authority(ctx context.Context) error {
	c := v.checks["authority"]
	if v.in.Codec == nil || v.in.Codec.Store == nil {
		c.unverifiable("authority.no_codec", "нет чтения содержимого записей — права подписантов не проверить")
		return nil
	}
	var recs []accessdom.Record
	for _, t := range accessdom.Types {
		after := int64(0)
		for {
			es, err := v.in.Codec.Store.Read(ctx, appjournal.ReadQuery{EventType: string(t), AfterSeq: after, Limit: 1000})
			if err != nil {
				return err
			}
			for _, e := range es {
				after = int64(e.Seq)
				d, err := v.in.Codec.Decode(ctx, e)
				if err != nil {
					c.unverifiable("authority.sealed", fmt.Sprintf("запись seq %d %s не прочитать: %v", e.Seq, t, err))
					continue
				}
				recs = append(recs, accessdom.Record{Seq: d.Record.Seq, Type: string(t), Data: d.Record.Data, OccurredAt: d.Record.OccurredAt,
					Genesis: e.ProvenanceClass == jc.JournalEntryProvenanceClassGenesis, EventID: d.Record.EventID, Actor: d.Record.Actor})
			}
			if len(es) < 1000 {
				break
			}
		}
	}
	slices.SortFunc(recs, func(a, b accessdom.Record) int { return int(a.Seq - b.Seq) })
	genesis := slices.ContainsFunc(recs, func(r accessdom.Record) bool { return r.Genesis })
	var pol accessdom.Policy
	switch {
	case genesis && v.in.Policy != nil:
		pol = accessdom.Policy{Root: v.in.Policy.Root, Unauthenticated: v.in.Policy.Unauthenticated, Catalog: v.in.Policy.Catalog}
	case genesis:
	case v.in.Policy != nil:
		pol = v.in.Policy.Clone()
	default:
		c.unverifiable("authority.no_policy", "политики генезиса в журнале нет и стартовая политика не передана — права подписантов проверить не на чем")
		return nil
	}
	for _, r := range recs {
		if !r.Genesis && r.Actor != "" {
			v.authorityOf(pol, r)
		}
		if err := pol.Apply(r); err != nil {
			c.reject("authority.bad_record", err.Error(), "main", r.Seq, "")
		}
	}
	return nil
}

// operationOf — операция API (x-ant-action), которой пишется тип записи политики.
var operationOf = map[catalog.Type]string{
	catalog.PolicyRoleAssigned: "access.policy.grant", catalog.PolicyAuthorityGranted: "access.policy.grant", catalog.PolicyStampIssued: "access.policy.grant",
	catalog.PolicyRoleUnassigned: "access.policy.revoke", catalog.PolicyAuthorityRevoked: "access.policy.revoke", catalog.PolicyStampRevoked: "access.policy.revoke",
	catalog.PolicyAuditParametersSet: "access.audit.set_parameters", catalog.AccessPersonRegistered: "access.person.register",
	catalog.AccessAccountActivated: "access.account.activate",
	catalog.AccessAssignmentSet:    "access.assignment.set", catalog.AccessAssignmentCleared: "access.assignment.clear",
	catalog.AccessQualificationGranted: "access.qualification.grant", catalog.AccessQualificationRevoked: "access.qualification.revoke",
}

// authorityOf — проверка одной записи по политике pol до неё.
func (v *run) authorityOf(pol accessdom.Policy, r accessdom.Record) {
	c := v.checks["authority"]
	c.checked++
	actor := personOf(pol, r.Actor)
	at := r.OccurredAt
	op := operationOf[catalog.Type(r.Type)]
	if op == "" {
		return
	}
	// Выдача с начальной ролью при активации учётной записи проверяется как выдача роли.
	if catalog.Type(r.Type) == catalog.PolicyRoleAssigned && !pol.EffectiveRights(actor, at).Allows(op) && pol.EffectiveRights(actor, at).Allows("access.account.activate") {
		op = "access.account.activate"
	}
	if !pol.EffectiveRights(actor, at).Allows(op) {
		c.reject("authority.no_right", fmt.Sprintf("seq %d %s: у %s на %s нет права %s", r.Seq, r.Type, actor, at.Format(time.RFC3339), op), "main", r.Seq, "")
		return
	}
	ch, second, ok := grantOf(r)
	if !ok {
		return
	}
	a := accessdom.Assess(pol, ch, actor, at)
	if a.SecondAuthority == "" {
		return
	}
	if second == "" {
		c.reject("authority.second_signature_missing", fmt.Sprintf("seq %d %s: %s — без второй подписи (%s)", r.Seq, r.Type, a.Reason, accessdom.DomainTitle(a.Domain)), "main", r.Seq, "")
		return
	}
	if err := accessdom.CheckApprovals(pol, a, ch, []accessdom.Approval{{PersonID: second, At: at, Seq: r.Seq}}); err != nil {
		c.reject("authority.second_signature_invalid", fmt.Sprintf("seq %d %s: вторая подпись %s не засчитывается — %v", r.Seq, r.Type, second, err), "main", r.Seq, "")
	}
}

// grantOf — выдача из записи и кто поставил вторую подпись.
func grantOf(r accessdom.Record) (accessdom.PolicyChange, string, bool) {
	var d struct {
		PersonID          string    `json:"person_id"`
		RoleID            string    `json:"role_id"`
		AuthorityID       string    `json:"authority_id"`
		StampID           string    `json:"stamp_id"`
		InspectionKind    string    `json:"inspection_kind"`
		Scope             string    `json:"scope"`
		ValidFrom         time.Time `json:"valid_from"`
		SecondSignatureBy string    `json:"second_signature_by"`
	}
	if json.Unmarshal(r.Data, &d) != nil {
		return accessdom.PolicyChange{}, "", false
	}
	c := accessdom.PolicyChange{PersonID: d.PersonID, Scope: d.Scope, ValidFrom: d.ValidFrom}
	switch catalog.Type(r.Type) {
	case catalog.PolicyRoleAssigned:
		c.Kind, c.SubjectID = accessdom.ChangeRole, d.RoleID
	case catalog.PolicyAuthorityGranted:
		c.Kind, c.SubjectID = accessdom.ChangeAuthority, d.AuthorityID
	case catalog.PolicyStampIssued:
		c.Kind, c.SubjectID, c.InspectionKind = accessdom.ChangeStamp, d.StampID, d.InspectionKind
	default:
		return c, "", false
	}
	return c, d.SecondSignatureBy, true
}

// personOf — псевдоним сотрудника по подписанту записи (`‹псевдоним›@версия`).
func personOf(pol accessdom.Policy, actor string) string {
	id, _, _ := strings.Cut(actor, "@")
	for _, x := range pol.Persons {
		if strings.EqualFold(x.ID, id) {
			return x.ID
		}
	}
	return id
}
