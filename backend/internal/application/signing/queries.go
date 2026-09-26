package signing

import (
	"context"
	"slices"
	"strconv"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/signing"
)

// Keys — реестр ключей (signing.key.list, FR-79): статус на голове журнала.
func (s *Service) Keys(ctx context.Context, subjectKind, subjectID, status string, _ platform.Moment, p platform.Page) (KeyList, error) {
	if err := s.readyRead(); err != nil {
		return KeyList{}, err
	}
	keys, _, _, err := s.d.Registry.Snapshot(ctx)
	if err != nil {
		return KeyList{}, err
	}
	head, err := s.head(ctx)
	if err != nil {
		return KeyList{}, err
	}
	var all []KeyView
	for _, k := range keys.Keys() {
		v := s.view(k, head)
		if (subjectKind == "" || v.SubjectKind == subjectKind) && (subjectID == "" || v.SubjectID == subjectID) && (status == "" || v.Status == status) {
			all = append(all, v)
		}
	}
	off, _ := strconv.Atoi(p.Cursor)
	limit := p.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	out := KeyList{Items: []KeyView{}}
	if off < len(all) {
		end := min(off+limit, len(all))
		out.Items = all[off:end]
		if end < len(all) {
			out.NextCursor = strconv.Itoa(end)
		}
	}
	return out, nil
}

// Key — ключ, подписи его акта и история ротаций (signing.key.read).
func (s *Service) Key(ctx context.Context, ref string, _ platform.Moment) (KeyDetails, error) {
	if err := s.readyRead(); err != nil {
		return KeyDetails{}, err
	}
	keys, _, _, err := s.d.Registry.Snapshot(ctx)
	if err != nil {
		return KeyDetails{}, err
	}
	head, err := s.head(ctx)
	if err != nil {
		return KeyDetails{}, err
	}
	k, ok := keys.Key(ref)
	if !ok {
		return KeyDetails{}, fail(errcodes.ApiNotFound, "ключ "+ref+" не зарегистрирован", "object", "ключ", "id", ref)
	}
	d := KeyDetails{Key: s.view(k, head), Signatures: []KeyActSignature{}, History: []KeyView{}, BasisSeq: head}
	for prev := k.Rotates; prev != ""; {
		pk, ok := keys.Key(prev)
		if !ok {
			break
		}
		d.History = append(d.History, s.view(pk, head))
		prev = pk.Rotates
	}
	d.Signatures = s.actSignatures(ctx, keys, k)
	return d, nil
}

// actSignatures — подписи конверта акта регистрации: кто и каким профилем.
func (s *Service) actSignatures(ctx context.Context, keys *dom.Registry, k dom.Key) []KeyActSignature {
	out := []KeyActSignature{}
	for _, a := range s.d.Registry.Acts(k.KeyRef) {
		if a.EventID != k.EventID {
			continue
		}
		es, err := s.d.Journal.Read(ctx, journalReadSeq(a.Seq))
		if err != nil || len(es) == 0 {
			continue
		}
		env, err := s.d.Journal.Open(ctx, es[0])
		if err != nil {
			continue
		}
		e, _, err := dom.ParseEnvelope(env.Raw)
		if err != nil {
			continue
		}
		for _, sg := range e.Signatures {
			sk, _ := keys.Key(sg.KeyID)
			role := "вторая подпись"
			switch sk.SubjectID {
			case k.SubjectID:
				role = "субъект (действующий ключ)"
			}
			out = append(out, KeyActSignature{SignerPersonID: sk.SubjectID, Role: role, ProfileID: sk.ProfileID, SignedAt: k.CommittedAt})
		}
	}
	return out
}

// Profiles — криптопрофили и обязательные классы на голове журнала (AD-32).
func (s *Service) Profiles(ctx context.Context, _ platform.Moment) (CryptoProfileList, error) {
	if err := s.readyRead(); err != nil {
		return CryptoProfileList{}, err
	}
	_, book, _, err := s.d.Registry.Snapshot(ctx)
	if err != nil {
		return CryptoProfileList{}, err
	}
	head, err := s.head(ctx)
	if err != nil {
		return CryptoProfileList{}, err
	}
	titles := map[string]string{dom.ProfileGost: "ГОСТ Р 34.10-2012, 256 бит", dom.ProfilePQ: "ML-DSA-65 (FIPS 204) — демонстрационный",
		dom.ProfileHybrid: "Гибрид: ГОСТ + ML-DSA-65, обе подписи обязательны"}
	out := CryptoProfileList{Items: []CryptoProfile{}}
	for _, p := range []string{dom.ProfileGost, dom.ProfilePQ, dom.ProfileHybrid} {
		cp := CryptoProfile{ProfileID: p, Title: titles[p], Algorithm: dom.AlgorithmOf(p), SignaturesRequired: len(dom.ProfileComponents(p)),
			Production: p == dom.ProfileGost, ObjectClasses: []string{}}
		if p == dom.ProfileHybrid {
			cp.Components = dom.ProfileComponents(p)
		}
		for _, cl := range dom.Classes {
			if book.Required(cl, head+1) == p {
				cp.ObjectClasses = append(cp.ObjectClasses, cl)
			}
		}
		for _, ch := range book.Changes() {
			if ch.Profile == p {
				f := ch.EffectiveFromSeq
				cp.EffectiveFromSeq = &f
			}
		}
		out.Items = append(out.Items, cp)
	}
	return out, nil
}

// view — представление ключа на голове журнала.
func (s *Service) view(k dom.Key, head int64) KeyView {
	v := KeyView{KeyRef: k.KeyRef, SubjectKind: k.SubjectKind, SubjectID: k.SubjectID, ProfileID: k.ProfileID, Algorithm: k.Algorithm,
		Fingerprint: k.Fingerprint, PayloadClasses: slices.Clone(k.PayloadClasses), ProvenanceClass: k.Provenance,
		Status: "active", ValidFrom: k.ValidFrom, ValidUntil: k.ValidUntil, Rotates: k.Rotates,
		RegistrationEventID: k.EventID, RegistrationDocID: k.DocumentID}
	if v.PayloadClasses == nil {
		v.PayloadClasses = []string{}
	}
	if v.ProvenanceClass == "" {
		v.ProvenanceClass = dom.NaturalProvenance(k.SubjectKind) // Д-67
	}
	switch {
	case k.Revoked != nil:
		v.Status, v.RevocationEventID, v.CompromisedSince = "revoked", k.Revoked.EventID, k.Revoked.CompromisedSince
	case len(k.PublicKey()) == 0:
		v.Status = "unavailable"
	case !k.ActiveAt(head+1, s.now()):
		v.Status = "expired"
	}
	if v.ValidFrom.IsZero() {
		v.ValidFrom = time.Unix(0, 0).UTC()
	}
	return v
}

func (s *Service) readyRead() error {
	if s.d.Registry == nil {
		return platform.NotImplemented("signing")
	}
	return nil
}
