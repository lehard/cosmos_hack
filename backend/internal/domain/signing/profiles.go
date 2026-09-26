package signing

import (
	"errors"
	"fmt"
	"slices"
	"sort"
)

// Криптопрофили и реестр профилей (AD-10, AD-32, FR-76, кейс §6.3; критерий О9).
// Профиль подписи задаёт, какие подписи обязательны; хеш-функцию задаёт версия
// формата цепочки, а не профиль (AD-32). Реестр профилей — записи журнала
// key.profile.registered: какой профиль обязателен для класса пакета с какой
// позиции журнала. Старые подписи проверяются по профилю на момент подписи.

// Профили (contracts/crypto/profiles.yaml).
const (
	ProfileGost   = "gost"
	ProfilePQ     = "pq"
	ProfileHybrid = "hybrid"
)

// Алгоритмы ключей (key.registration.recorded.algorithm).
const (
	AlgGost  = "gost3410_2012_256_paramset_a"
	AlgMLDSA = "ml_dsa_65"
)

// Размеры открытых ключей и подписей по профилю (contracts/crypto/README.md).
const (
	GostPublicKeySize  = 64
	GostSignatureSize  = 64
	MLDSAPublicKeySize = 1952
	MLDSASignatureSize = 3309
)

// Классы пакетов (contracts/crypto/payload-classes.yaml).
const (
	ClassEvent             = "event"
	ClassDocumentSignature = "document-signature"
	ClassPaperAttestation  = "paper-attestation"
	ClassShiftReport       = "shift-report"
	ClassCheckpoint        = "checkpoint"
	ClassGenesis           = "genesis"
	ClassKeyAct            = "key-act"
	ClassVerifierReport    = "verifier-report"
	ClassPassportExtract   = "passport-extract"
)

// Classes — все классы пакетов v1 по порядку.
var Classes = []string{ClassCheckpoint, ClassDocumentSignature, ClassEvent, ClassGenesis, ClassKeyAct,
	ClassPaperAttestation, ClassPassportExtract, ClassShiftReport, ClassVerifierReport}

// KnownClass — класс есть в payload-classes.yaml.
func KnownClass(c string) bool { return slices.Contains(Classes, c) }

// ProfileComponents — из каких подписей состоит профиль: gost → [gost],
// pq → [pq], hybrid → [gost, pq] (обе обязательны).
func ProfileComponents(p string) []string {
	switch p {
	case ProfileGost:
		return []string{ProfileGost}
	case ProfilePQ:
		return []string{ProfilePQ}
	case ProfileHybrid:
		return []string{ProfileGost, ProfilePQ}
	}
	return nil
}

// KnownProfile — профиль есть в profiles.yaml.
func KnownProfile(p string) bool { return ProfileComponents(p) != nil }

// AlgorithmOf — алгоритм ключа профиля gost или pq.
func AlgorithmOf(profile string) string {
	switch profile {
	case ProfileGost:
		return AlgGost
	case ProfilePQ:
		return AlgMLDSA
	}
	return ""
}

// Covers — профиль have покрывает профиль need: каждая обязательная
// подпись need есть в have (hybrid покрывает gost; gost не покрывает hybrid).
func Covers(have, need string) bool {
	hc := ProfileComponents(have)
	if hc == nil {
		return false
	}
	for _, c := range ProfileComponents(need) {
		if !slices.Contains(hc, c) {
			return false
		}
	}
	return true
}

// DefaultObjectProfiles — «объект → обязательный профиль» MVP
// (profiles.yaml, object_profiles; AD-32): контрольные точки, генезис, акты
// ключей и отчёты верификатора — hybrid; прочее — gost.
func DefaultObjectProfiles() map[string]string {
	return map[string]string{
		ClassCheckpoint: ProfileHybrid, ClassGenesis: ProfileHybrid, ClassKeyAct: ProfileHybrid, ClassVerifierReport: ProfileHybrid,
		ClassEvent: ProfileGost, ClassDocumentSignature: ProfileGost, ClassShiftReport: ProfileGost,
		ClassPaperAttestation: ProfileGost, ClassPassportExtract: ProfileGost,
	}
}

// ProfileChange — запись реестра профилей (key.profile.registered): профиль
// обязателен для классов с позиции EffectiveFromSeq (не раньше Seq самой записи).
type ProfileChange struct {
	Profile          string
	Classes          []string
	EffectiveFromSeq int64
	// Seq — позиция записи в журнале (порядок знания).
	Seq int64
}

// ProfileBook — реестр профилей: исходные обязательные профили (генезис) и
// смены в порядке записи.
type ProfileBook struct {
	base    map[string]string
	changes []ProfileChange
}

// NewProfileBook — реестр с исходными профилями base (nil — DefaultObjectProfiles).
func NewProfileBook(base map[string]string) *ProfileBook {
	if base == nil {
		base = DefaultObjectProfiles()
	}
	return &ProfileBook{base: base}
}

// ErrProfile — смена профиля не по правилам.
var ErrProfile = errors.New("signing: смена криптопрофиля отвергнута")

// ErrDowngrade — понижение профиля (AD-10, AD-32): удаление обязательной
// подписи из пакета или смена обязательного профиля на более слабый.
var ErrDowngrade = errors.New("signing.profile_downgrade")

// Required — обязательный профиль для класса пакета на позиции seq.
func (b *ProfileBook) Required(class string, seq int64) string {
	p := b.base[class]
	for _, c := range b.changes {
		if c.EffectiveFromSeq <= seq && slices.Contains(c.Classes, class) {
			p = c.Profile
		}
	}
	if p == "" {
		p = ProfileGost
	}
	return p
}

// CheckChange — можно ли записать смену профиля: профиль известен, классы
// известны, и новый профиль покрывает действующий для каждого класса — иначе
// «отвергнуто: понижение профиля» (AD-32: миграция не ослабляет защиту молча).
func (b *ProfileBook) CheckChange(c ProfileChange) error {
	if !KnownProfile(c.Profile) {
		return fmt.Errorf("%w: неизвестный профиль %q", ErrProfile, c.Profile)
	}
	if len(c.Classes) == 0 {
		return fmt.Errorf("%w: нет классов пакетов", ErrProfile)
	}
	if c.EffectiveFromSeq != 0 && c.EffectiveFromSeq < c.Seq {
		return fmt.Errorf("%w: смена профиля задним числом (с %d при записи %d)", ErrProfile, c.EffectiveFromSeq, c.Seq)
	}
	for _, cl := range c.Classes {
		if !KnownClass(cl) {
			return fmt.Errorf("%w: неизвестный класс пакета %q", ErrProfile, cl)
		}
		if cur := b.Required(cl, maxSeq); !Covers(c.Profile, cur) {
			return fmt.Errorf("%w: для %s обязателен %s, предложен %s", ErrDowngrade, cl, cur, c.Profile)
		}
	}
	return nil
}

const maxSeq = int64(1<<53 - 1)

// Apply — принять записанную смену профиля (свёртка журнала). EffectiveFromSeq
// 0 — с позиции самой записи.
func (b *ProfileBook) Apply(c ProfileChange) {
	if c.EffectiveFromSeq == 0 {
		c.EffectiveFromSeq = c.Seq
	}
	c.Classes = slices.Clone(c.Classes)
	b.changes = append(b.changes, c)
	sort.SliceStable(b.changes, func(i, j int) bool { return b.changes[i].Seq < b.changes[j].Seq })
}

// Changes — смены профиля по порядку записи.
func (b *ProfileBook) Changes() []ProfileChange { return slices.Clone(b.changes) }

// Base — исходный обязательный профиль класса.
func (b *ProfileBook) Base(class string) string { return b.base[class] }

// Clone — копия реестра профилей.
func (b *ProfileBook) Clone() *ProfileBook {
	c := &ProfileBook{base: b.base, changes: make([]ProfileChange, len(b.changes))}
	for i, ch := range b.changes {
		ch.Classes = slices.Clone(ch.Classes)
		c.changes[i] = ch
	}
	return c
}
