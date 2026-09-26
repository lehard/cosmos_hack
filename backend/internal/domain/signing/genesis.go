package signing

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// Блок генезиса доверия (AD-33, AD-10, FR-10, FR-109; критерий Т1): записи
// seq 1…k класса genesis, подписанные установочным ключом-якорем hybrid.
// Первая запись — заголовок journal.genesis.recorded (код предприятия, формат
// цепочки, открытые ключи и отпечаток якоря, размер блока, отпечаток
// остальных записей), последняя — journal.anchor.destroyed. Здесь — чистые
// правила: отпечатки якоря и блока и проверка блока «целиком по якорю»; сами
// подписи проверяет порт Verifier (результат — CryptoCheck по каждой подписи).
// Одни и те же правила исполняют ant init (после записи), ядро при старте и
// независимый верификатор (AD-9).

// Константы блока генезиса.
const (
	// GenesisSource — source_id записей блока (роль init).
	GenesisSource = "ant-init"
	// ChainFormatVersion — версия формата цепочки v1 (AD-44: H = Стрибог-256).
	ChainFormatVersion = 1
	// AnchorSubject — субъект ключа-якоря; ключи — anchor-gost@1 и anchor-pq@1.
	AnchorSubject = "anchor"
	// TypeGenesis, TypeAnchorDestroyed, TypeKeyRegistration — типы записей,
	// которые правило блока различает (contracts/events/catalog.yaml).
	TypeGenesis         = "journal.genesis.recorded"
	TypeAnchorDestroyed = "journal.anchor.destroyed"
	TypeKeyRegistration = "key.registration.recorded"
)

// AnchorRefs — key_ref ключей якоря: ГОСТ и ML-DSA-65 (профиль hybrid).
var AnchorRefs = []string{AnchorSubject + "-gost@1", AnchorSubject + "-pq@1"}

// AnchorKey — открытый ключ якоря (заголовок генезиса, anchor_keys).
type AnchorKey struct {
	KeyRef    string `json:"key_ref"`
	ProfileID string `json:"profile_id"`
	PublicB64 string `json:"public_key_b64"`
}

// GenesisHeader — data заголовка journal.genesis.recorded.
type GenesisHeader struct {
	EnterpriseCode       string      `json:"enterprise_code"`
	ChainFormatVersion   int         `json:"chain_format_version"`
	AnchorFingerprint    string      `json:"anchor_fingerprint"`
	BlockSize            int         `json:"block_size"`
	NormativeVersionHash string      `json:"normative_version_hash,omitempty"`
	AnchorKeys           []AnchorKey `json:"anchor_keys,omitempty"`
	BlockDigest          string      `json:"block_digest,omitempty"`
}

// ErrGenesis — блок генезиса не проходит проверку по якорю (AD-33): «отвергнуто».
var ErrGenesis = errors.New("signing: блок генезиса отвергнут")

// ErrSecondGenesis — второй блок генезиса в журнале — нарушение (AD-33).
var ErrSecondGenesis = errors.New("signing: второй блок генезиса в журнале — нарушение")

// AnchorFingerprint — общий отпечаток ключей якоря: H(ключ₁ ‖ ключ₂) в порядке
// anchor_keys (ГОСТ, затем ML-DSA-65). Его закрепляет файл trust-anchors вне
// системы: верификатор принимает блок, только если отпечаток совпал (AD-33).
func AnchorFingerprint(keys []AnchorKey) string {
	var parts [][]byte
	for _, k := range keys {
		parts = append(parts, AnchorKeyBytes(k))
	}
	return HashPrefix + hex.EncodeToString(Hash(parts...))
}

// AnchorKeyBytes — сырые байты открытого ключа якоря.
func AnchorKeyBytes(k AnchorKey) []byte {
	return Registration{PublicKeyB64: k.PublicB64}.PublicKey()
}

// BlockDigest — отпечаток записей блока после заголовка: H(H(p₂) ‖ … ‖ H(pₖ))
// по порядку seq, p — канонический payload записи (под подписью якоря).
// Подмена, удаление или перестановка любой записи меняют его.
func BlockDigest(payloads [][]byte) string {
	parts := make([][]byte, 0, len(payloads))
	for _, p := range payloads {
		parts = append(parts, Hash(p))
	}
	return HashPrefix + hex.EncodeToString(Hash(parts...))
}

// GenesisDigest — отпечаток генезиса: H(payload заголовка). Заголовок
// связывает отпечаток якоря и отпечаток остальных записей — значит, и весь
// блок. Его записывают trust-anchors и первая контрольная точка (AD-33).
func GenesisDigest(headerPayload []byte) string { return Digest(headerPayload) }

// GenesisEntry — запись блока для проверки: открытые поля, подписанное
// содержимое и итог криптопроверки каждой подписи конверта.
type GenesisEntry struct {
	Seq         int64
	EventType   string
	Provenance  string
	PayloadType string
	Payload     []byte
	// Present — keyid подписей конверта по порядку.
	Present []string
	// Crypto — проверка каждой подписи портом Verifier.
	Crypto []CryptoCheck
}

// GenesisCheck — вход проверки блока.
type GenesisCheck struct {
	// Block — записи seq 1…k по порядку.
	Block []GenesisEntry
	// Pinned — отпечаток якоря из trust-anchors; пусто — только
	// внутренняя согласованность (ядро без тома хранителя).
	Pinned string
	// GenesisCount — сколько записей journal.genesis.recorded в журнале.
	GenesisCount int
	// DigestOnly — быстрая проверка (ядро при старте): криптографически
	// проверены только заголовок, «якорь уничтожен» и записи с кворумом
	// (Crypto остальных пусто); целостность остальных — через block_digest
	// под подписью якоря в заголовке, наличие обеих подписей якоря — по
	// Present. Полную проверку каждой подписи делает верификатор.
	DigestOnly bool
}

// genesisEvent — поля события, которые сверяет правило блока.
type genesisEvent struct {
	EventType string          `json:"event_type"`
	Integrity Integrity       `json:"integrity"`
	Data      json.RawMessage `json:"data"`
}

// ParseGenesisHeader — заголовок из payload записи seq 1.
func ParseGenesisHeader(payload []byte) (GenesisHeader, error) {
	var ev genesisEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return GenesisHeader{}, fmt.Errorf("%w: заголовок: %v", ErrGenesis, err)
	}
	if ev.EventType != TypeGenesis {
		return GenesisHeader{}, fmt.Errorf("%w: seq 1 — %s, а не заголовок генезиса", ErrGenesis, ev.EventType)
	}
	var h GenesisHeader
	if err := json.Unmarshal(ev.Data, &h); err != nil {
		return GenesisHeader{}, fmt.Errorf("%w: заголовок: %v", ErrGenesis, err)
	}
	return h, nil
}

// CheckGenesisBlock — AD-33: верификатор принимает блок целиком по якорю.
// Правила: блок — seq 1…k без пропусков, k = block_size; первая запись —
// заголовок, последняя — «якорь уничтожен» с тем же отпечатком; все записи
// класса genesis и пакета класса genesis; ключи якоря в заголовке дают
// anchor_fingerprint (и совпадают с закреплённым в trust-anchors); отпечаток
// остальных записей совпадает с block_digest; каждую запись подписали оба
// ключа якоря (hybrid: без любой подписи — понижение профиля); каждый
// обязательный подписант (integrity.signers, в том числе кворум нормативного
// слоя ключами этого же блока) подписал верно; неверных подписей нет;
// генезис в журнале один.
func CheckGenesisBlock(c GenesisCheck) (GenesisHeader, error) {
	fail := func(f string, a ...any) (GenesisHeader, error) {
		return GenesisHeader{}, fmt.Errorf("%w: "+f, append([]any{ErrGenesis}, a...)...)
	}
	if c.GenesisCount > 1 {
		return GenesisHeader{}, fmt.Errorf("%w: записей journal.genesis.recorded — %d", ErrSecondGenesis, c.GenesisCount)
	}
	if len(c.Block) == 0 {
		return fail("блока нет")
	}
	h, err := ParseGenesisHeader(c.Block[0].Payload)
	if err != nil {
		return GenesisHeader{}, err
	}
	switch {
	case h.BlockSize != len(c.Block):
		return fail("block_size %d, записей %d", h.BlockSize, len(c.Block))
	case h.ChainFormatVersion != ChainFormatVersion:
		return fail("формат цепочки %d", h.ChainFormatVersion)
	case len(h.AnchorKeys) != len(AnchorRefs):
		return fail("ключей якоря %d, нужно %d (hybrid)", len(h.AnchorKeys), len(AnchorRefs))
	case AnchorFingerprint(h.AnchorKeys) != h.AnchorFingerprint:
		return fail("отпечаток якоря не совпадает с ключами заголовка")
	case c.Pinned != "" && c.Pinned != h.AnchorFingerprint:
		return fail("якорь не тот, что закреплён в trust-anchors")
	}
	for i, k := range h.AnchorKeys {
		want := ProfileComponents(ProfileHybrid)[i]
		if k.KeyRef != AnchorRefs[i] || k.ProfileID != want {
			return fail("ключ якоря %d: %s/%s", i+1, k.KeyRef, k.ProfileID)
		}
	}
	payloads := make([][]byte, 0, len(c.Block)-1)
	pt := PayloadType(ClassGenesis, 1)
	for i, e := range c.Block {
		switch {
		case e.Seq != int64(i+1):
			return fail("запись %d на позиции seq %d — блок не с seq 1 или с пропуском", i+1, e.Seq)
		case e.Provenance != ProvGenesis:
			return fail("seq %d: класс происхождения %s", e.Seq, e.Provenance)
		case e.PayloadType != pt:
			return fail("seq %d: пакет %q, а не класса genesis", e.Seq, e.PayloadType)
		}
		var ev genesisEvent
		if err := json.Unmarshal(e.Payload, &ev); err != nil {
			return fail("seq %d: содержимое: %v", e.Seq, err)
		}
		if ev.EventType != e.EventType {
			return fail("seq %d: подписан тип %s, в открытых полях %s", e.Seq, ev.EventType, e.EventType)
		}
		if i > 0 {
			if e.EventType == TypeGenesis {
				return fail("seq %d: второй заголовок внутри блока", e.Seq)
			}
			payloads = append(payloads, e.Payload)
		}
		if c.DigestOnly && len(e.Crypto) == 0 {
			for _, ref := range AnchorRefs {
				if !slices.Contains(e.Present, ref) {
					return fail("seq %d: нет подписи якоря %s (%s)", e.Seq, ref, ReasonDowngrade)
				}
			}
		} else if err := checkGenesisSignatures(e, ev.Integrity); err != nil {
			return GenesisHeader{}, err
		}
		if i == len(c.Block)-1 {
			var d struct {
				AnchorFingerprint string `json:"anchor_fingerprint"`
			}
			if e.EventType != TypeAnchorDestroyed || json.Unmarshal(ev.Data, &d) != nil || d.AnchorFingerprint != h.AnchorFingerprint {
				return fail("последняя запись блока — не «якорь уничтожен» этого якоря")
			}
		}
	}
	if h.BlockDigest != BlockDigest(payloads) {
		return fail("отпечаток записей блока не совпадает с заголовком — блок изменён")
	}
	return h, nil
}

// checkGenesisSignatures — подписи одной записи блока: оба ключа якоря и все
// обязательные подписанты — «ok»; неверная подпись — «изменён».
func checkGenesisSignatures(e GenesisEntry, ig Integrity) error {
	ok := func(ref string) bool {
		return slices.ContainsFunc(e.Crypto, func(c CryptoCheck) bool { return c.KeyRef == ref && c.Result == CryptoOK })
	}
	for _, c := range e.Crypto {
		if c.Result == CryptoBad {
			return fmt.Errorf("%w: seq %d: подпись %s не сходится — запись изменена", ErrGenesis, e.Seq, c.KeyRef)
		}
	}
	for _, ref := range AnchorRefs {
		if !ok(ref) {
			return fmt.Errorf("%w: seq %d: нет действительной подписи якоря %s (%s)", ErrGenesis, e.Seq, ref, ReasonDowngrade)
		}
	}
	if ig.CryptoProfile != ProfileHybrid {
		return fmt.Errorf("%w: seq %d: профиль %q, обязателен hybrid", ErrGenesis, e.Seq, ig.CryptoProfile)
	}
	for _, ref := range ig.Signers {
		if !ok(ref) {
			return fmt.Errorf("%w: seq %d: нет действительной подписи обязательного подписанта %s", ErrGenesis, e.Seq, ref)
		}
	}
	return nil
}
