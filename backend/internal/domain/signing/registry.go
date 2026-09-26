package signing

import (
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"
)

// Реестр ключей (AD-11, AD-32, FR-68–70, FR-79): доверие ключу — только от
// акта регистрации в журнале (или генезиса); отзыв и «скомпрометирован с X» —
// тоже акты. Реестр — свёртка записей key.registration.recorded и
// key.revocation.recorded; его строят одинаково сервер (приём, проверка
// команд) и независимый верификатор (AD-9) — поэтому он здесь, в домене.
// Криптографические моменты (действие и отзыв ключа, «скомпрометирован с») —
// по seq и committed_at (AD-37).

// Виды субъектов ключа (key.registration.recorded.subject_kind).
const (
	SubjectPerson            = "person"
	SubjectDevice            = "device"
	SubjectEngine            = "engine"
	SubjectGateway           = "gateway"
	SubjectEnterpriseGateway = "enterprise_gateway"
	SubjectKeeper            = "keeper"
	SubjectVerifier          = "verifier"
	SubjectDemoPersona       = "demo_persona"
	SubjectPartnerRoot       = "partner_root"
)

// Классы происхождения подписи (AD-2) — они же классы доверия ключа.
const (
	ProvPersonal       = "personal"
	ProvDevice         = "device"
	ProvServerAttested = "server_attested"
	ProvPartner        = "partner"
	ProvScenario       = "scenario"
	ProvGenesis        = "genesis"
	ProvPaper          = "paper"
)

// Подтверждение субъекта при регистрации (AD-11).
const (
	ConfirmRotation = "rotation_signature"
	ConfirmPaper    = "paper_receipt"
	ConfirmGenesis  = "genesis"
)

// SubjectKinds — все виды субъектов ключа.
var SubjectKinds = []string{SubjectPerson, SubjectDevice, SubjectEngine, SubjectGateway, SubjectEnterpriseGateway,
	SubjectKeeper, SubjectVerifier, SubjectDemoPersona, SubjectPartnerRoot}

// Registration — данные акта регистрации ключа (key.registration.recorded) и
// позиция записи в журнале.
type Registration struct {
	KeyRef              string
	SubjectKind         string
	SubjectID           string
	ProfileID           string
	Algorithm           string
	PublicKeyB64        string
	Fingerprint         string
	PayloadClasses      []string
	Rotates             string
	SubjectConfirmation string
	DocumentID          string
	ValidFrom           time.Time
	ValidUntil          *time.Time
	// Provenance — класс доверия ключа (наследуется от регистрирующих подписей).
	Provenance string
	// EventID, Seq, CommittedAt — запись акта (AD-37: криптографический момент).
	EventID     string
	Seq         int64
	CommittedAt time.Time
}

// Revocation — данные акта отзыва (key.revocation.recorded).
type Revocation struct {
	KeyRef string
	// CompromisedSince — «скомпрометирован с X»; nil — отзыв без компрометации.
	CompromisedSince *time.Time
	// CompromisedSinceSeq — X как позиция журнала (первая запись с committed_at ≥ X).
	CompromisedSinceSeq int64
	ReasonCode          string
	ReasonText          string
	DocumentID          string
	EventID             string
	Seq                 int64
	CommittedAt         time.Time
}

// Key — ключ в реестре.
type Key struct {
	Registration
	Revoked *Revocation
	// RotatedBy — ключ, заменивший этот при ротации, и позиция его регистрации.
	RotatedBy    string
	RotatedAtSeq int64
}

// PublicKey — сырые байты открытого ключа.
func (k Registration) PublicKey() []byte {
	b, _ := base64.StdEncoding.DecodeString(k.PublicKeyB64)
	return b
}

// Registry — реестр ключей: свёртка актов в порядке seq.
type Registry struct {
	keys map[string]*Key
}

// NewRegistry — пустой реестр.
func NewRegistry() *Registry { return &Registry{keys: map[string]*Key{}} }

// Ошибки реестра.
var (
	ErrUnknownKey = errors.New("signing: ключ не зарегистрирован")
	ErrKeyAct     = errors.New("signing: акт ключа отвергнут")
)

// Key — ключ по key_id@версия.
func (r *Registry) Key(ref string) (Key, bool) {
	k, ok := r.keys[ref]
	if !ok {
		return Key{}, false
	}
	return *k, true
}

// Keys — все ключи по порядку key_ref.
func (r *Registry) Keys() []Key {
	out := make([]Key, 0, len(r.keys))
	for _, ref := range slices.Sorted(maps.Keys(r.keys)) {
		out = append(out, *r.keys[ref])
	}
	return out
}

// ActiveAt — ключ действует на позиции seq (момент committed_at): записан до
// seq, не отозван к seq, не заменён ротацией к seq, в сроке ValidFrom…ValidUntil.
func (k Key) ActiveAt(seq int64, at time.Time) bool {
	if k.Seq > seq || (k.Revoked != nil && k.Revoked.Seq <= seq) || (k.RotatedBy != "" && k.RotatedAtSeq <= seq) {
		return false
	}
	if !k.ValidFrom.IsZero() && !at.IsZero() && at.Before(k.ValidFrom) {
		return false
	}
	return k.ValidUntil == nil || at.IsZero() || at.Before(*k.ValidUntil)
}

// CompromisedAt — подпись на позиции seq (момент at) сделана после
// «скомпрометирован с X» — «под сомнением» (AD-9, AD-11).
func (k Key) CompromisedAt(seq int64, at time.Time) bool {
	rv := k.Revoked
	if rv == nil || (rv.CompromisedSince == nil && rv.CompromisedSinceSeq == 0) {
		return false
	}
	if rv.CompromisedSinceSeq > 0 {
		return seq >= rv.CompromisedSinceSeq
	}
	return !at.IsZero() && !at.Before(*rv.CompromisedSince)
}

// Active — действующие ключи субъекта профиля на позиции seq.
func (r *Registry) Active(subjectKind, subjectID, profile string, seq int64, at time.Time) []Key {
	var out []Key
	for _, k := range r.Keys() {
		if k.SubjectKind == subjectKind && k.SubjectID == subjectID && (profile == "" || k.ProfileID == profile) && k.ActiveAt(seq, at) {
			out = append(out, k)
		}
	}
	return out
}

// ApplyRegistration — принять записанный акт регистрации (свёртка журнала;
// проверка правил — ValidateRegistration до записи). Ротация закрывает
// заменяемый ключ с позиции нового акта.
func (r *Registry) ApplyRegistration(g Registration) {
	g.PayloadClasses = slices.Clone(g.PayloadClasses)
	r.keys[g.KeyRef] = &Key{Registration: g}
	if old, ok := r.keys[g.Rotates]; ok && g.Rotates != "" && old.RotatedBy == "" {
		old.RotatedBy, old.RotatedAtSeq = g.KeyRef, g.Seq
	}
}

// ApplyRevocation — принять записанный акт отзыва.
func (r *Registry) ApplyRevocation(v Revocation) {
	if k, ok := r.keys[v.KeyRef]; ok && k.Revoked == nil {
		vv := v
		k.Revoked = &vv
	}
}

// NaturalProvenance — класс происхождения подписи ключом субъекта (AD-2).
func NaturalProvenance(subjectKind string) string {
	switch subjectKind {
	case SubjectPerson:
		return ProvPersonal
	case SubjectDevice:
		return ProvDevice
	case SubjectDemoPersona:
		return ProvScenario
	case SubjectPartnerRoot:
		return ProvPartner
	}
	return ProvServerAttested
}

// trustRank — ранг класса доверия: scenario < genesis < прочие (AD-11:
// ключ, зарегистрированный подписями класса scenario или genesis, получает
// класс не выше исходного).
func trustRank(p string) int {
	switch p {
	case ProvScenario:
		return 0
	case ProvGenesis:
		return 1
	}
	return 2
}

// InheritProvenance — класс доверия нового ключа: естественный класс
// субъекта, ограниченный сверху самым слабым классом регистрирующих подписей.
func InheritProvenance(subjectKind string, signers []string) string {
	p := NaturalProvenance(subjectKind)
	for _, s := range signers {
		if trustRank(s) < trustRank(p) {
			p = s
		}
	}
	return p
}

// Approval — подпись под актом ключа: кто, с каким полномочием, каким ключом
// и её класс доверия. Криптографию подписи проверяет порт Verifier до вызова
// домена (Valid).
type Approval struct {
	PersonID    string
	AuthorityID string
	KeyRef      string
	Provenance  string
	Valid       bool
}

// Полномочия второй подписи (normative/policy, AD-11, PRD §11.16).
const (
	AuthSecondQC         = "second_signature_qc"
	AuthSecondProduction = "second_signature_production"
	AuthSecondAudit      = "second_signature_audit"
	AuthPaperAttestation = "paper_attestation"
)

// Сферы субъекта ключа для выбора второй подписи (AD-11).
const (
	DomainQC         = "qc"
	DomainProduction = "production"
	DomainAdmin      = "admin"
)

// SecondSignatureAuthority — чьё полномочие нужно для второй подписи акта
// ключа (AD-11): ключи контролёров и ОТК, корни партнёра — начальник ОТК;
// ключи производства и устройств — руководитель производства; ключи
// администраторов, аудита и системные ключи — Аудитор ИБ. Демо-персоны
// (класс scenario) — без второй подписи: их регистрирует генезис.
func SecondSignatureAuthority(subjectKind, domain string) string {
	switch subjectKind {
	case SubjectDemoPersona:
		return ""
	case SubjectPartnerRoot:
		return AuthSecondQC
	case SubjectDevice:
		return AuthSecondProduction
	case SubjectPerson:
		switch domain {
		case DomainQC:
			return AuthSecondQC
		case DomainAdmin:
			return AuthSecondAudit
		}
		return AuthSecondProduction
	}
	return AuthSecondAudit
}

// RegistrationAct — акт регистрации до записи: данные, результат проверки
// доказательства владения (подпись нового ключа над актом), подтверждение
// субъекта и подписи маршрута.
type RegistrationAct struct {
	Registration
	// SubjectDomain — сфера субъекта-человека (qc / production / admin).
	SubjectDomain string
	// Registrar — кто подаёт акт (администратор безопасности).
	Registrar string
	// ProofValid — подпись нового ключа над актом сходится (доказательство владения).
	ProofValid bool
	// RotationValid — при ротации: действующий ключ подписал акт.
	RotationValid bool
	// Receipt — при первичной выдаче: бумажная расписка субъекта с отпечатком
	// (учётный номер оригинала и заверитель — путь бумаги, AD-43).
	ReceiptOriginalNo string
	ReceiptAttestedBy string
	// Approvals — подписи маршрута акта (вторая подпись независимой стороны).
	Approvals []Approval
}

// KeyAlert — тревога по ключу (security.key.alert, эмитент — security, AD-24).
type KeyAlert struct {
	Alert  string // duplicate_key | foreign_key | agent_journal_mismatch | level1_rate_exceeded
	KeyRef string
	Person string
	Detail string
}

// ValidateRegistration — AD-11, FR-70, FR-79: правила акта регистрации до
// записи. Возвращает класс доверия ключа и тревоги. Правила:
//   - key_ref свободен; профиль и алгоритм согласованы; открытый ключ нужной
//     длины и отпечаток = H(ключ); классы пакетов известны;
//   - доказательство владения: новый ключ подписал акт (кроме генезиса);
//   - подтверждение субъекта: при ротации — подпись действующим ключом того же
//     субъекта и профиля; при первичной выдаче — бумажная расписка с
//     отпечатком (учётный номер оригинала и заверитель ≠ субъект);
//   - вторая подпись — от независимой стороны по сфере субъекта; «выдача
//     себе» (подписант = субъект или подавший акт) не засчитывается;
//   - у человека не больше одного действующего ключа на профиль: второй —
//     тревога «повторный ключ» (регистрация не блокируется: тревога и
//     уведомление субъекту через его агент — так в AD-11).
func ValidateRegistration(r *Registry, a RegistrationAct, seq int64, at time.Time) (string, []KeyAlert, error) {
	fail := func(f string, args ...any) (string, []KeyAlert, error) {
		return "", nil, fmt.Errorf("%w: "+f, append([]any{ErrKeyAct}, args...)...)
	}
	if _, _, err := SplitKeyRef(a.KeyRef); err != nil {
		return fail("key_ref %q", a.KeyRef)
	}
	if _, ok := r.keys[a.KeyRef]; ok {
		return fail("ключ %s уже зарегистрирован — новый ключ регистрируется новой версией", a.KeyRef)
	}
	if a.SubjectID == "" || !slices.Contains(SubjectKinds, a.SubjectKind) {
		return fail("субъект %s/%q", a.SubjectKind, a.SubjectID)
	}
	if AlgorithmOf(a.ProfileID) == "" || AlgorithmOf(a.ProfileID) != a.Algorithm {
		return fail("профиль %s и алгоритм %s не согласованы", a.ProfileID, a.Algorithm)
	}
	pub := a.PublicKey()
	want := GostPublicKeySize
	if a.ProfileID == ProfilePQ {
		want = MLDSAPublicKeySize
	}
	if len(pub) != want {
		return fail("открытый ключ %d байт, ожидается %d", len(pub), want)
	}
	if a.Fingerprint != Digest(pub) {
		return fail("отпечаток не совпадает с ключом: сверьте отпечаток")
	}
	if len(a.PayloadClasses) == 0 {
		return fail("нет допустимых классов пакетов")
	}
	for _, c := range a.PayloadClasses {
		if !KnownClass(c) {
			return fail("неизвестный класс пакета %q", c)
		}
	}
	if a.SubjectConfirmation == ConfirmGenesis {
		return fail("подтверждение genesis — только в блоке генезиса (ant init)")
	}
	if !a.ProofValid {
		return fail("нет доказательства владения: подпись нового ключа над актом не сходится")
	}
	switch {
	case a.Rotates != "":
		old, ok := r.keys[a.Rotates]
		if !ok || !old.ActiveAt(seq, at) {
			return fail("заменяемый ключ %s не действует", a.Rotates)
		}
		if old.SubjectKind != a.SubjectKind || old.SubjectID != a.SubjectID || old.ProfileID != a.ProfileID {
			return fail("ротация меняет субъект или профиль ключа %s", a.Rotates)
		}
		if a.SubjectConfirmation != ConfirmRotation || !a.RotationValid {
			return fail("ротация без подписи действующим ключом %s", a.Rotates)
		}
	case a.SubjectConfirmation != ConfirmPaper:
		return fail("первичная выдача — только с бумажной распиской субъекта")
	case a.ReceiptOriginalNo == "" || a.ReceiptAttestedBy == "":
		return fail("расписка без учётного номера оригинала или заверителя")
	case a.ReceiptAttestedBy == a.SubjectID:
		return fail("расписку субъекта заверяет второй человек — заверитель ≠ субъект")
	}
	signers := []string{}
	if need := SecondSignatureAuthority(a.SubjectKind, a.SubjectDomain); need != "" {
		ok := false
		for _, ap := range a.Approvals {
			if ap.Valid && ap.AuthorityID == need && ap.PersonID != a.SubjectID && ap.PersonID != a.Registrar {
				ok = true
			}
		}
		if !ok {
			return fail("нет второй подписи независимой стороны с полномочием %s (выдача себе не засчитывается)", need)
		}
	}
	for _, ap := range a.Approvals {
		if ap.Valid {
			signers = append(signers, ap.Provenance)
		}
	}
	prov := InheritProvenance(a.SubjectKind, signers)
	var alerts []KeyAlert
	if a.SubjectKind == SubjectPerson && a.Rotates == "" {
		for _, k := range r.Active(SubjectPerson, a.SubjectID, a.ProfileID, seq, at) {
			alerts = append(alerts, KeyAlert{Alert: "duplicate_key", KeyRef: a.KeyRef, Person: a.SubjectID,
				Detail: "у сотрудника уже есть действующий ключ " + k.KeyRef + " профиля " + a.ProfileID})
		}
	}
	return prov, alerts, nil
}

// RevocationAct — акт отзыва до записи.
type RevocationAct struct {
	Revocation
	// Requester — кто отзывает; Approvals — подписи акта.
	Requester string
	Approvals []Approval
}

// ValidateRevocation — AD-11, FR-70, FR-79: отзыв — защитное действие
// (AD-27): достаточно подписи подавшего; ключ должен быть зарегистрирован и
// ещё не отозван; «скомпрометирован с X» может быть раньше даты отзыва, но не
// позже её.
func ValidateRevocation(r *Registry, v RevocationAct, at time.Time) error {
	k, ok := r.keys[v.KeyRef]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownKey, v.KeyRef)
	}
	if k.Revoked != nil {
		return fmt.Errorf("%w: ключ %s уже отозван", ErrKeyAct, v.KeyRef)
	}
	if v.CompromisedSince != nil && !at.IsZero() && v.CompromisedSince.After(at) {
		return fmt.Errorf("%w: «скомпрометирован с» позже отзыва", ErrKeyAct)
	}
	if v.ReasonText == "" {
		return fmt.Errorf("%w: нет причины отзыва", ErrKeyAct)
	}
	valid := false
	for _, ap := range v.Approvals {
		valid = valid || ap.Valid
	}
	if !valid {
		return fmt.Errorf("%w: акт отзыва не подписан", ErrKeyAct)
	}
	return nil
}
