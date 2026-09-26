package signing

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"time"

	"ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/signing"
)

// TimeLayout — время в конверте и открытых полях: RFC 3339 UTC, три знака
// после секунд («Соглашения/Время»).
const TimeLayout = "2006-01-02T15:04:05.000Z"

// Registry — проекция реестра ключей и профилей над журналом (AD-11, AD-32):
// свёртка domain/signing по записям key.* в порядке seq. Исходные ключи
// (генезис — эпик 05; до него — затравка Bootstrap) применяются первыми.
// Потокобезопасна; догоняет журнал при каждом чтении (Sync).
type Registry struct {
	mu      sync.Mutex
	journal journal.JournalStore
	keys    *dom.Registry
	book    *dom.ProfileBook
	after   int64
	// incomplete — есть записи key.*, конверт которых не открылся (нет KEK,
	// AD-23): неизвестный ключ тогда — «не проверяемо», а не «чужой».
	incomplete bool
	// events — записи актов по key_ref (для истории ключа).
	acts map[string][]actRef
}

type actRef struct {
	EventID string
	Seq     int64
	Type    string
}

// NewRegistry — реестр над журналом j (nil — только затравка) с исходными
// ключами boot и исходными профилями base (nil — profiles.yaml).
func NewRegistry(j journal.JournalStore, boot []dom.Registration, base map[string]string) *Registry {
	r := &Registry{journal: j, keys: dom.NewRegistry(), book: dom.NewProfileBook(base), acts: map[string][]actRef{}}
	for _, g := range boot {
		if g.Provenance == "" {
			// Д-67: исходный ключ — класс по назначению, не genesis.
			g.Provenance = dom.NaturalProvenance(g.SubjectKind)
		}
		r.keys.ApplyRegistration(g)
	}
	return r
}

// Sync догоняет журнал: новые записи key.* после последней применённой.
func (r *Registry) Sync(ctx context.Context) error {
	if r.journal == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var all []jc.JournalEntry
	for _, t := range []catalog.Type{catalog.KeyProfileRegistered, catalog.KeyRegistrationRecorded, catalog.KeyRevocationRecorded} {
		after := r.after
		for {
			es, err := r.journal.Read(ctx, journal.ReadQuery{EventType: string(t), AfterSeq: after, Limit: 1000})
			if err != nil {
				return err
			}
			all = append(all, es...)
			if len(es) < 1000 {
				break
			}
			after = int64(es[len(es)-1].Seq)
		}
	}
	slices.SortFunc(all, func(a, b jc.JournalEntry) int { return a.Seq - b.Seq })
	for _, e := range all {
		if err := r.apply(ctx, e); err != nil {
			r.incomplete = true
		}
		if int64(e.Seq) > r.after {
			r.after = int64(e.Seq)
		}
	}
	return nil
}

// apply — одна запись key.* в свёртку.
func (r *Registry) apply(ctx context.Context, e jc.JournalEntry) error {
	env, err := r.journal.Open(ctx, e)
	if err != nil {
		return err
	}
	data, err := EventData(env.Raw)
	if err != nil {
		return err
	}
	at, _ := time.Parse(TimeLayout, e.CommittedAt)
	seq := int64(e.Seq)
	switch catalog.Type(e.EventType) {
	case catalog.KeyRegistrationRecorded:
		g, err := RegistrationFromData(data)
		if err != nil {
			return err
		}
		g.EventID, g.Seq, g.CommittedAt = e.EventID, seq, at
		g.Provenance = provenanceOf(e, g)
		r.keys.ApplyRegistration(g)
		r.acts[g.KeyRef] = append(r.acts[g.KeyRef], actRef{e.EventID, seq, e.EventType})
	case catalog.KeyRevocationRecorded:
		v, err := RevocationFromData(data)
		if err != nil {
			return err
		}
		v.EventID, v.Seq, v.CommittedAt = e.EventID, seq, at
		r.keys.ApplyRevocation(v)
		r.acts[v.KeyRef] = append(r.acts[v.KeyRef], actRef{e.EventID, seq, e.EventType})
	case catalog.KeyProfileRegistered:
		var p struct {
			ProfileID        string   `json:"profile_id"`
			ObjectClasses    []string `json:"object_classes"`
			EffectiveFromSeq int64    `json:"effective_from_seq"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return err
		}
		r.book.Apply(dom.ProfileChange{Profile: p.ProfileID, Classes: p.ObjectClasses, EffectiveFromSeq: p.EffectiveFromSeq, Seq: seq})
	}
	return nil
}

// provenanceOf — класс доверия ключа: из класса происхождения записи акта,
// ограниченный естественным классом субъекта (AD-11: наследуется).
func provenanceOf(e jc.JournalEntry, g dom.Registration) string {
	return dom.InheritProvenance(g.SubjectKind, []string{string(e.ProvenanceClass)})
}

// Snapshot — копия реестра ключей и профилей после Sync (для чистых функций домена).
func (r *Registry) Snapshot(ctx context.Context) (*dom.Registry, *dom.ProfileBook, bool, error) {
	if err := r.Sync(ctx); err != nil {
		return nil, nil, false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	keys, book := r.keys.Clone(), r.book.Clone()
	return keys, book, r.incomplete, nil
}

// Acts — акты ключа по порядку.
func (r *Registry) Acts(ref string) []actRef {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.acts[ref])
}

// ErrKeyUnavailable — открытая часть ключа недоступна (AD-32).
var ErrKeyUnavailable = errors.New("signing.key_unavailable")

// PublicKey — открытый ключ по key_ref для порта Verifier: профиль и байты.
// Незарегистрированный ключ при неполном реестре — ErrKeyUnavailable.
func (r *Registry) PublicKey(ctx context.Context, ref string) (string, []byte, error) {
	if err := r.Sync(ctx); err != nil {
		return "", nil, errors.Join(ErrKeyUnavailable, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys.Key(ref)
	if !ok {
		if r.incomplete {
			return "", nil, ErrKeyUnavailable
		}
		return "", nil, dom.ErrUnknownKey
	}
	pub := k.PublicKey()
	if len(pub) == 0 {
		return k.ProfileID, nil, ErrKeyUnavailable
	}
	return k.ProfileID, pub, nil
}

// EventData — data события из конверта записи: DSSE (payload — событие) или
// событие без конверта.
func EventData(raw []byte) (json.RawMessage, error) {
	payload := raw
	if env, p, err := dom.ParseEnvelope(raw); err == nil && env.PayloadType != "" {
		payload = p
	}
	var ev struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	if len(ev.Data) == 0 {
		return nil, errors.New("signing: в событии нет data")
	}
	return ev.Data, nil
}

// RegistrationData — data записи key.registration.recorded (contracts/events/key).
type RegistrationData struct {
	KeyRef               string   `json:"key_ref"`
	SubjectKind          string   `json:"subject_kind"`
	SubjectID            string   `json:"subject_id"`
	ProfileID            string   `json:"profile_id"`
	Algorithm            string   `json:"algorithm"`
	PublicKeyB64         string   `json:"public_key_b64"`
	Fingerprint          string   `json:"fingerprint"`
	PayloadClasses       []string `json:"payload_classes"`
	Rotates              string   `json:"rotates,omitempty"`
	ProofOfPossessionB64 string   `json:"proof_of_possession_b64,omitempty"`
	SubjectConfirmation  string   `json:"subject_confirmation"`
	DocumentID           string   `json:"document_id,omitempty"`
	ValidFrom            string   `json:"valid_from"`
	ValidUntil           string   `json:"valid_until,omitempty"`
}

// RegistrationFromData — регистрация домена из data записи.
func RegistrationFromData(raw []byte) (dom.Registration, error) {
	var d RegistrationData
	if err := json.Unmarshal(raw, &d); err != nil {
		return dom.Registration{}, err
	}
	return d.Registration()
}

// Registration — регистрация домена.
func (d RegistrationData) Registration() (dom.Registration, error) {
	g := dom.Registration{KeyRef: d.KeyRef, SubjectKind: d.SubjectKind, SubjectID: d.SubjectID, ProfileID: d.ProfileID,
		Algorithm: d.Algorithm, PublicKeyB64: d.PublicKeyB64, Fingerprint: d.Fingerprint, PayloadClasses: d.PayloadClasses,
		Rotates: d.Rotates, SubjectConfirmation: d.SubjectConfirmation, DocumentID: d.DocumentID}
	var err error
	if d.ValidFrom != "" {
		if g.ValidFrom, err = time.Parse(time.RFC3339Nano, d.ValidFrom); err != nil {
			return g, err
		}
	}
	if d.ValidUntil != "" {
		t, err := time.Parse(time.RFC3339Nano, d.ValidUntil)
		if err != nil {
			return g, err
		}
		g.ValidUntil = &t
	}
	return g, nil
}

// RevocationData — data записи key.revocation.recorded.
type RevocationData struct {
	KeyRef              string `json:"key_ref"`
	CompromisedSince    string `json:"compromised_since,omitempty"`
	CompromisedSinceSeq int64  `json:"compromised_since_seq,omitempty"`
	Reason              struct {
		Code string `json:"code,omitempty"`
		Text string `json:"text"`
	} `json:"reason"`
	DocumentID string `json:"document_id,omitempty"`
}

// RevocationFromData — отзыв домена из data записи.
func RevocationFromData(raw []byte) (dom.Revocation, error) {
	var d RevocationData
	if err := json.Unmarshal(raw, &d); err != nil {
		return dom.Revocation{}, err
	}
	v := dom.Revocation{KeyRef: d.KeyRef, CompromisedSinceSeq: d.CompromisedSinceSeq, ReasonCode: d.Reason.Code,
		ReasonText: d.Reason.Text, DocumentID: d.DocumentID}
	if d.CompromisedSince != "" {
		t, err := time.Parse(time.RFC3339Nano, d.CompromisedSince)
		if err != nil {
			return v, err
		}
		v.CompromisedSince = &t
	}
	return v, nil
}

// journalReadSeq — чтение одной записи по seq.
func journalReadSeq(seq int64) journal.ReadQuery {
	return journal.ReadQuery{AfterSeq: seq - 1, Limit: 1}
}
