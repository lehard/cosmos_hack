package signing

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

// Проверка подписи пакета (FR-68, FR-76, AD-9, AD-10, AD-32): подпись,
// действительность ключа на момент подписи, отзыв, «скомпрометирован с»,
// допустимый класс пакета и обязательный профиль на момент подписи.
// Криптографию каждой подписи делает порт Verifier (результат — CryptoCheck);
// здесь — чистое решение, одинаковое у сервера и независимого верификатора.

// Status — итог проверки пакета (статусы верификатора AD-9).
type Status string

const (
	// StatusValid — «цело»: все обязательные подписи сходятся, ключи действуют,
	// профиль не ниже обязательного.
	StatusValid Status = "valid"
	// StatusRejected — «отвергнуто»: подпись не сходится, пакет изменён,
	// понижение профиля, ключ не зарегистрирован, отозван или не для этого класса.
	StatusRejected Status = "rejected"
	// StatusUnverifiable — «не проверяемо»: ключ недоступен — никогда не «валидно» (AD-32).
	StatusUnverifiable Status = "unverifiable"
	// StatusDoubtful — «под сомнением»: подписано после «скомпрометирован с X» (AD-11).
	StatusDoubtful Status = "doubtful"
	// StatusUnsigned — пакет без подписей (профиль demo, Д-28, Д-30): «подпись не проверялась».
	StatusUnsigned Status = "unsigned"
)

// Причины (коды contracts/errors.yaml, семейство signing).
const (
	ReasonTampered       = "signing.package_tampered"
	ReasonDowngrade      = "signing.profile_downgrade"
	ReasonRevoked        = "signing.key_revoked"
	ReasonUnavailable    = "signing.key_unavailable"
	ReasonUnknownKey     = "signing.unknown_key"
	ReasonClassForbidden = "signing.class_not_allowed"
	ReasonKeyInactive    = "signing.key_inactive"
	ReasonCompromised    = "signing.key_compromised"
	ReasonNotCanonical   = "signing.not_canonical"
	ReasonUnsigned       = "signing.unsigned"
)

// Crypto — итог криптографической проверки одной подписи портом Verifier.
type Crypto string

const (
	// CryptoOK — подпись сходится с PAE открытым ключом ключа.
	CryptoOK Crypto = "ok"
	// CryptoBad — подпись не сходится (пакет изменён или подпись чужая).
	CryptoBad Crypto = "bad"
	// CryptoUnavailable — открытого ключа нет (ключ недоступен, KEK нет, том не смонтирован).
	CryptoUnavailable Crypto = "unavailable"
)

// CryptoCheck — результат порта по одной подписи конверта.
type CryptoCheck struct {
	KeyRef string
	Result Crypto
}

// Integrity — блок целостности под подписью (events/common/envelope.v1.json,
// shift-report, paper-attestation): профиль и обязательные подписанты.
type Integrity struct {
	FormatVersion int      `json:"format_version"`
	CryptoProfile string   `json:"crypto_profile"`
	Signers       []string `json:"signers"`
}

// IntegrityOf — блок целостности из содержимого пакета: поле integrity (событие)
// или поля верхнего уровня crypto_profile/signers (рапорт, заверение).
func IntegrityOf(payload []byte) (Integrity, error) {
	var p struct {
		Integrity     *Integrity `json:"integrity"`
		FormatVersion int        `json:"format_version"`
		CryptoProfile string     `json:"crypto_profile"`
		Signers       []string   `json:"signers"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return Integrity{}, err
	}
	if p.Integrity != nil {
		return *p.Integrity, nil
	}
	return Integrity{FormatVersion: p.FormatVersion, CryptoProfile: p.CryptoProfile, Signers: p.Signers}, nil
}

// Judgement — вход решения по пакету.
type Judgement struct {
	// Class — класс пакета (из payloadType).
	Class string
	// Declared — блок целостности из содержимого (под подписью).
	Declared Integrity
	// Present — keyid подписей конверта по порядку.
	Present []string
	// Crypto — проверка каждой подписи портом.
	Crypto []CryptoCheck
	// Seq, At — момент подписи: позиция записи в журнале и её committed_at
	// (для проверки команды до записи — голова журнала и «сейчас» InfraClock).
	Seq int64
	At  time.Time
}

// SignerVerdict — итог по одному обязательному подписанту.
type SignerVerdict struct {
	KeyRef     string
	Status     Status
	Reason     string
	Profile    string
	SubjectID  string
	Subject    string
	Provenance string
}

// Verdict — итог проверки пакета.
type Verdict struct {
	Status   Status
	Reason   string
	Detail   string
	Required string // обязательный профиль на момент подписи
	Signers  []SignerVerdict
	// Extra — подписи сверх перечня обязательных: в итог не засчитываются
	// (AD-10), но проверяются по отдельности — ими, например, субъект
	// подтверждает ротацию действующим ключом (AD-11).
	Extra []SignerVerdict
}

// Judge — решение по пакету (AD-10, AD-32):
//  1. обязательный профиль класса на момент подписи; заявленный в пакете
//     профиль не может быть слабее, иначе «отвергнуто: понижение профиля»;
//  2. каждый обязательный подписант (integrity.signers) должен иметь подпись
//     в конверте; лишние подписи и повторы одного ключа не засчитываются;
//  3. ключ подписанта зарегистрирован, действует на момент подписи, допускает
//     класс пакета; открытый ключ доступен (иначе «не проверяемо»); подпись
//     сходится; после «скомпрометирован с X» — «под сомнением»;
//  4. профили действительных подписей покрывают обязательный профиль (hybrid —
//     ГОСТ и ML-DSA-65 одного субъекта): удалённая подпись — понижение профиля.
//
// Приоритет итога: отвергнуто > не проверяемо > под сомнением > цело.
func Judge(reg *Registry, book *ProfileBook, j Judgement) Verdict {
	v := Verdict{Status: StatusValid, Required: book.Required(j.Class, j.Seq)}
	if len(j.Present) == 0 {
		v.Status, v.Reason, v.Detail = StatusUnsigned, ReasonUnsigned, "пакет без подписей — подпись не проверялась"
		return v
	}
	worse := func(s Status, reason, detail string) {
		if rank(s) > rank(v.Status) {
			v.Status, v.Reason, v.Detail = s, reason, detail
		}
	}
	if !KnownProfile(j.Declared.CryptoProfile) {
		worse(StatusRejected, ReasonDowngrade, "в пакете не указан криптопрофиль")
	} else if !Covers(j.Declared.CryptoProfile, v.Required) {
		worse(StatusRejected, ReasonDowngrade, fmt.Sprintf("для %s на момент подписи обязателен профиль %s, а подписано %s",
			j.Class, v.Required, j.Declared.CryptoProfile))
	}
	if len(j.Declared.Signers) == 0 {
		worse(StatusRejected, ReasonTampered, "в пакете нет перечня обязательных подписантов")
	}
	have := map[string][]string{} // субъект → профили действительных подписей
	var subjects []string
	for _, ref := range dedupe(j.Present) {
		if !slices.Contains(j.Declared.Signers, ref) {
			v.Extra = append(v.Extra, judgeOne(reg, j, ref))
		}
	}
	for _, ref := range dedupe(j.Declared.Signers) {
		sv := SignerVerdict{KeyRef: ref, Status: StatusValid}
		k, ok := reg.Key(ref)
		switch {
		case !slices.Contains(j.Present, ref):
			sv.Status, sv.Reason = StatusRejected, ReasonDowngrade // обязательную подпись удалили
		default:
			sv = judgeOne(reg, j, ref)
		}
		if ok {
			sv.Profile, sv.SubjectID, sv.Subject, sv.Provenance = k.ProfileID, k.SubjectID, k.SubjectKind, k.Provenance
		}
		if sv.Status == StatusValid || sv.Status == StatusDoubtful {
			sid := k.SubjectKind + "/" + k.SubjectID
			if !slices.Contains(subjects, sid) {
				subjects = append(subjects, sid)
			}
			have[sid] = append(have[sid], k.ProfileID)
		}
		worse(sv.Status, sv.Reason, "подпись "+ref+": "+string(sv.Status))
		v.Signers = append(v.Signers, sv)
	}
	if v.Status == StatusValid || v.Status == StatusDoubtful {
		// AD-10: профиль покрыт подписями одного субъекта (hybrid — два ключа одного субъекта).
		for _, sid := range subjects {
			if !coveredBy(have[sid], j.Declared.CryptoProfile) {
				worse(StatusRejected, ReasonDowngrade, "подписи "+sid+" не покрывают профиль "+j.Declared.CryptoProfile)
			}
		}
	}
	return v
}

// judgeOne — одна подпись: ключ зарегистрирован, действует на момент
// подписи, допускает класс пакета, открытая часть доступна, подпись сходится,
// не после «скомпрометирован с».
func judgeOne(reg *Registry, j Judgement, ref string) SignerVerdict {
	sv := SignerVerdict{KeyRef: ref, Status: StatusValid}
	k, ok := reg.Key(ref)
	res := cryptoOf(j.Crypto, ref)
	switch {
	case !ok:
		sv.Status, sv.Reason = StatusRejected, ReasonUnknownKey
	case k.Revoked != nil && k.Revoked.Seq <= j.Seq:
		sv.Status, sv.Reason = StatusRejected, ReasonRevoked
	case !k.ActiveAt(j.Seq, j.At):
		sv.Status, sv.Reason = StatusRejected, ReasonKeyInactive
	case !slices.Contains(k.PayloadClasses, j.Class):
		sv.Status, sv.Reason = StatusRejected, ReasonClassForbidden
	case res == CryptoUnavailable || res == "":
		sv.Status, sv.Reason = StatusUnverifiable, ReasonUnavailable
	case res == CryptoBad:
		sv.Status, sv.Reason = StatusRejected, ReasonTampered
	case k.CompromisedAt(j.Seq, j.At):
		sv.Status, sv.Reason = StatusDoubtful, ReasonCompromised
	}
	if ok {
		sv.Profile, sv.SubjectID, sv.Subject, sv.Provenance = k.ProfileID, k.SubjectID, k.SubjectKind, k.Provenance
	}
	return sv
}

func coveredBy(profiles []string, need string) bool {
	for _, c := range ProfileComponents(need) {
		if !slices.Contains(profiles, c) {
			return false
		}
	}
	return true
}

func rank(s Status) int {
	switch s {
	case StatusRejected:
		return 4
	case StatusUnverifiable:
		return 3
	case StatusDoubtful:
		return 2
	case StatusUnsigned:
		return 1
	}
	return 0
}

func cryptoOf(cs []CryptoCheck, ref string) Crypto {
	var out Crypto
	for _, c := range cs {
		if c.KeyRef != ref {
			continue
		}
		if c.Result == CryptoOK {
			return CryptoOK // повтор одного ключа не засчитывается дважды, но хватает одной верной
		}
		if out == "" || c.Result == CryptoBad {
			out = c.Result
		}
	}
	return out
}

func dedupe(s []string) []string {
	var out []string
	for _, x := range s {
		if !slices.Contains(out, x) {
			out = append(out, x)
		}
	}
	return out
}
