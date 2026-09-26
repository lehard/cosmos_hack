package signing

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"uuid"

	"ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/signing"
)

// Акты ключей (AD-11, AD-13, FR-70, FR-79): регистрация (выпуск, ротация,
// ключ устройства по акту ввода), отзыв со «скомпрометирован с X» и смена
// криптопрофиля (AD-32). Акт — событие-команда класса key-act, подписанное
// подавшим и второй стороной (профиль hybrid по profiles.yaml); в журнал
// пишется подписанный конверт как есть — верификатор проверяет его сам.

func (s *Service) ready() error {
	if s.d.Registry == nil || s.d.Journal == nil {
		return platform.NotImplemented("signing")
	}
	return nil
}

// RegisterKey — акт регистрации ключа.
func (s *Service) RegisterKey(ctx context.Context, in RegisterKey) (platform.Receipt, error) {
	if err := s.ready(); err != nil {
		return platform.Receipt{}, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	id, err := commandID(in.CommandID)
	if err != nil {
		return platform.Receipt{}, err
	}
	pub, _ := base64.StdEncoding.DecodeString(in.PublicKeyB64)
	data := RegistrationData{KeyRef: in.KeyRef, SubjectKind: in.SubjectKind, SubjectID: in.SubjectID, ProfileID: in.ProfileID,
		Algorithm: in.Algorithm, PublicKeyB64: in.PublicKeyB64, Fingerprint: dom.Digest(pub), PayloadClasses: in.PayloadClasses,
		Rotates: in.Rotates, ProofOfPossessionB64: in.ProofOfPossessionB64, SubjectConfirmation: in.SubjectConfirmation,
		DocumentID: in.DocumentID, ValidFrom: in.ValidFrom, ValidUntil: in.ValidUntil,
		KeyStorage: in.KeyStorage, StorageVariant: in.StorageVariant}
	g, err := data.Registration()
	if err != nil {
		return platform.Receipt{}, fail(errcodes.ApiValidationFailed, err.Error(), "field", "valid_from", "reason", err.Error())
	}
	raw, _ := json.Marshal(data)
	meta := in.CommandMeta()
	a, err := s.CheckCommand(ctx, meta.Signature, Expect{Class: dom.ClassKeyAct, EventType: string(catalog.KeyRegistrationRecorded),
		CommandID: id, Data: raw, Actor: actor, Level: dom.Level2, Critical: true, CoSigners: true})
	if err != nil {
		return platform.Receipt{}, err
	}
	if a.Status != dom.StatusValid {
		// Выдача ключа — разрешающее критическое действие: без подписей не делается и в демо.
		return platform.Receipt{}, fail(errcodes.SigningNoSignaturePath, "акт регистрации ключа подписывают подавший и вторая сторона")
	}
	pop, _ := base64.StdEncoding.DecodeString(in.ProofOfPossessionB64)
	content, _ := PoPContent(data)
	act := dom.RegistrationAct{Registration: g, SubjectDomain: s.domain(ctx, in.SubjectID), Registrar: actor,
		ProofValid:        s.d.Crypto != nil && s.d.Crypto.VerifyRaw(in.ProfileID, pub, dom.PayloadType(dom.ClassKeyAct, 1), content, pop),
		RotationValid:     in.Rotates != "" && (contains(a.KeyRefs, in.Rotates) || contains(a.ExtraValid, in.Rotates)),
		ReceiptOriginalNo: in.ReceiptOriginalNo, ReceiptAttestedBy: in.ReceiptAttestedBy,
		Approvals: s.approvals(ctx, a, in.SubjectKind, in.SubjectID)}
	keys, _, _, err := s.d.Registry.Snapshot(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	head, err := s.head(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	prov, alerts, err := dom.ValidateRegistration(keys, act, head+1, s.now())
	if err != nil {
		code := errcodes.ApiValidationFailed
		if !act.ProofValid {
			code = errcodes.SigningPackageTampered
		}
		return platform.Receipt{}, fail(code, err.Error(), "field", "key_ref", "reason", err.Error())
	}
	_ = prov // класс доверия ключа выводит свёртка из класса записи акта (provenanceOf)
	keyID, _, _ := dom.SplitKeyRef(in.KeyRef)
	return s.write(ctx, catalog.KeyRegistrationRecorded, "key:"+keyID, id, a, meta, alerts)
}

// PoPContent — что подписывает новый ключ (доказательство владения, AD-11):
// канонический data акта без самого поля proof_of_possession_b64.
func PoPContent(d RegistrationData) ([]byte, error) {
	d.ProofOfPossessionB64 = ""
	return dom.CanonicalOf(d)
}

// approvals — подписи акта кроме подавшего, с полномочием второй подписи по
// сфере субъекта (AD-11).
func (s *Service) approvals(ctx context.Context, a Accepted, kind, subject string) []dom.Approval {
	need := dom.SecondSignatureAuthority(kind, s.domain(ctx, subject))
	head, _ := s.head(ctx)
	out := make([]dom.Approval, 0, len(a.CoSigners))
	for _, c := range a.CoSigners {
		if need != "" && s.d.Authorities != nil {
			if ok, _ := s.d.Authorities.Has(ctx, c.PersonID, need, head); ok {
				c.AuthorityID = need
			}
		}
		out = append(out, c)
	}
	return out
}

func (s *Service) domain(ctx context.Context, person string) string {
	if s.d.Authorities == nil {
		return dom.DomainProduction
	}
	return s.d.Authorities.Domain(ctx, person)
}

// RevokeKey — акт отзыва ключа; «скомпрометирован с X» — защитная реакция
// (Doubts). Отзыв — защитное действие (AD-27): в демо без агента токена
// допускается без подписи с пометкой.
func (s *Service) RevokeKey(ctx context.Context, keyRef string, in RevokeKey) (platform.Receipt, error) {
	if err := s.ready(); err != nil {
		return platform.Receipt{}, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	id, err := commandID(in.CommandID)
	if err != nil {
		return platform.Receipt{}, err
	}
	data := RevocationData{KeyRef: keyRef, CompromisedSince: in.CompromisedSince, DocumentID: in.DocumentID}
	data.Reason.Code, data.Reason.Text = in.Reason.Code, in.Reason.Text
	var x *time.Time
	if in.CompromisedSince != "" {
		t, err := time.Parse(time.RFC3339Nano, in.CompromisedSince)
		if err != nil {
			return platform.Receipt{}, fail(errcodes.ApiValidationFailed, err.Error(), "field", "compromised_since", "reason", err.Error())
		}
		t = t.UTC()
		x = &t
		data.CompromisedSince = t.Format(TimeLayout)
		if data.CompromisedSinceSeq, err = s.seqSince(ctx, t); err != nil {
			return platform.Receipt{}, err
		}
	}
	raw, _ := json.Marshal(data)
	meta := in.CommandMeta()
	a, err := s.CheckCommand(ctx, meta.Signature, Expect{Class: dom.ClassKeyAct, EventType: string(catalog.KeyRevocationRecorded),
		CommandID: id, Data: raw, Actor: actor, Level: dom.Level2, Critical: true, CoSigners: true})
	if err != nil {
		return platform.Receipt{}, err
	}
	keys, _, _, err := s.d.Registry.Snapshot(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	v := dom.RevocationAct{Revocation: dom.Revocation{KeyRef: keyRef, CompromisedSince: x, CompromisedSinceSeq: data.CompromisedSinceSeq,
		ReasonCode: in.Reason.Code, ReasonText: in.Reason.Text}, Requester: actor,
		Approvals: []dom.Approval{{PersonID: actor, Valid: a.Status == dom.StatusValid || a.Status == dom.StatusUnsigned}}}
	if err := dom.ValidateRevocation(keys, v, s.now()); err != nil {
		if errors.Is(err, dom.ErrUnknownKey) {
			return platform.Receipt{}, fail(errcodes.ApiNotFound, err.Error(), "object", "ключ", "id", keyRef)
		}
		return platform.Receipt{}, fail(errcodes.ApiValidationFailed, err.Error(), "field", "key_ref", "reason", err.Error())
	}
	keyID, _, _ := dom.SplitKeyRef(keyRef)
	if a.Envelope == nil {
		a.Payload, a.Envelope = s.unsignedEvent(catalog.KeyRevocationRecorded, id, raw, actor)
	}
	return s.write(ctx, catalog.KeyRevocationRecorded, "key:"+keyID, id, a, meta, nil)
}

// RegisterProfile — смена обязательного профиля (AD-32, FR-76): понижение
// отвергается; повышение — защитное действие (в демо допускается без подписи
// с пометкой). Старые подписи не трогаются.
func (s *Service) RegisterProfile(ctx context.Context, in RegisterProfile) (platform.Receipt, error) {
	if err := s.ready(); err != nil {
		return platform.Receipt{}, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	id, err := commandID(in.CommandID)
	if err != nil {
		return platform.Receipt{}, err
	}
	_, book, _, err := s.d.Registry.Snapshot(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	head, err := s.head(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	ch := dom.ProfileChange{Profile: in.ProfileID, Classes: in.ObjectClasses, EffectiveFromSeq: in.EffectiveFromSeq, Seq: head + 1}
	if err := book.CheckChange(ch); err != nil {
		if errors.Is(err, dom.ErrDowngrade) {
			return platform.Receipt{}, fail(errcodes.SigningProfileDowngrade, err.Error(), "object_class", strings.Join(in.ObjectClasses, ", "),
				"required", "не ниже действующего", "actual", in.ProfileID)
		}
		return platform.Receipt{}, fail(errcodes.ApiValidationFailed, err.Error(), "field", "profile_id", "reason", err.Error())
	}
	payload := map[string]any{"profile_id": in.ProfileID, "object_classes": in.ObjectClasses}
	if in.EffectiveFromSeq > 0 {
		payload["effective_from_seq"] = in.EffectiveFromSeq
	}
	raw, _ := json.Marshal(payload)
	meta := in.CommandMeta()
	a, err := s.CheckCommand(ctx, meta.Signature, Expect{Class: dom.ClassKeyAct, EventType: string(catalog.KeyProfileRegistered),
		CommandID: id, Data: raw, Actor: actor, Level: dom.Level2, Critical: true, CoSigners: true})
	if err != nil {
		return platform.Receipt{}, err
	}
	if a.Envelope == nil {
		a.Payload, a.Envelope = s.unsignedEvent(catalog.KeyProfileRegistered, id, raw, actor)
	}
	return s.write(ctx, catalog.KeyProfileRegistered, "global", id, a, meta, nil)
}

// unsignedEvent — событие-команда без подписей (демо, Д-30).
func (s *Service) unsignedEvent(t catalog.Type, id string, data json.RawMessage, actor string) ([]byte, []byte) {
	info, _ := catalog.Lookup(t)
	ev := map[string]any{"event_id": id, "event_type": string(t), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
		"source_kind": "manual_entry", "occurred_at": s.now().Format(TimeLayout), "correlation_id": id, "causation_id": nil,
		"command":   map[string]any{"command_id": id, "basis_seq": 0, "guard_streams": []string{}, "policy_seq": 0, "signature_level": dom.Level2},
		"integrity": map[string]any{"format_version": 1, "crypto_profile": dom.ProfileGost, "signers": []string{signerRef(actor)}},
		"data":      data}
	c, _ := dom.CanonicalOf(ev)
	return c, dom.Seal(dom.PayloadType(dom.ClassKeyAct, 1), c).Marshal()
}

func signerRef(person string) string {
	if person == "" {
		return "anonymous@1"
	}
	return strings.ToLower(person) + "@1"
}

// write — запись акта через journal.Append (AD-44): подписанный конверт как
// есть; тревоги по ключу — в той же пачке (AD-24).
func (s *Service) write(ctx context.Context, t catalog.Type, stream, id string, a Accepted, meta platform.CommandMeta, alerts []dom.KeyAlert) (platform.Receipt, error) {
	info, _ := catalog.Lookup(t)
	now := s.now()
	prov := jc.JournalEntryProvenanceClass(a.Provenance)
	if prov == "" {
		prov = jc.JournalEntryProvenanceClassPersonal
	}
	e := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKind(info.Kind), EventType: string(t),
		SchemaVersion: info.CurrentVersion, EventID: id, SourceID: SourceAPI, Stream: stream, Partition: s.cfg.Partitions,
		OccurredAt: now.Format(TimeLayout), ReceivedAt: now.Format(TimeLayout), CorrelationID: id,
		ProvenanceClass: prov, DomainBuild: s.cfg.DomainBuild,
	}
	if meta.BasisSeq > 0 {
		b := int(meta.BasisSeq)
		e.BasisSeq = &b
	}
	if meta.PolicySeq > 0 {
		p := int(meta.PolicySeq)
		e.PolicySeq = &p
	}
	rq := journal.AppendRequest{Batch: []journal.Pending{{Entry: e, Envelope: a.Envelope}},
		Checks: []journal.Check{{Stream: stream, BasisSeq: meta.BasisSeq}}}
	if meta.BasisSeq == 0 {
		rq.Checks = nil
	}
	for _, al := range alerts {
		if s.d.Alerts != nil {
			if ps, err := s.d.Alerts.KeyAlert(ctx, al); err == nil {
				rq.Batch = append(rq.Batch, ps...)
			}
		}
	}
	res, err := s.d.Journal.Append(ctx, rq)
	if errors.Is(err, journal.ErrDuplicate) {
		return platform.Receipt{CommandID: id, EventIDs: []string{id}, Replayed: true}, nil
	}
	if err != nil {
		if pe, ok := journal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	r := platform.Receipt{CommandID: id, EventIDs: []string{id}, RecordedAt: res.Committed.UTC()}
	if len(res.Seqs) > 0 {
		r.Seq = res.Seqs[0]
	}
	return r, s.d.Registry.Sync(ctx)
}

// seqSince — «скомпрометирован с X» как позиция журнала: первая запись с
// committed_at ≥ X (AD-37), поиском назад от головы.
func (s *Service) seqSince(ctx context.Context, x time.Time) (int64, error) {
	head, err := s.head(ctx)
	if err != nil {
		return 0, err
	}
	if !x.Before(s.now()) {
		return head + 1, nil
	}
	var commits []dom.SeqTime
	after := int64(0)
	for {
		es, err := s.d.Journal.Read(ctx, journal.ReadQuery{AfterSeq: after, Limit: 1000})
		if err != nil {
			return 0, err
		}
		for _, e := range es {
			t, _ := time.Parse(TimeLayout, e.CommittedAt)
			commits = append(commits, dom.SeqTime{Seq: int64(e.Seq), At: t})
		}
		if len(es) < 1000 {
			break
		}
		after = int64(es[len(es)-1].Seq)
	}
	return dom.CompromisedSinceSeq(x, commits), nil
}

func commandID(s string) (string, error) {
	id := strings.ToLower(s)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		return uuid.NewV7().String(), nil
	}
	return id, nil
}

func contains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}
