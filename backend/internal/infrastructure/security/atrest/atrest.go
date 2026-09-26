// Пакет atrest — шифрование при хранении конвертной схемой (AD-23, FR-75):
// KEK — ключ шифрования ключей из отдельного тома (в БД его нет), DEK — ключ
// данных на объект (запись журнала, материал), обёртка DEK — AEAD ключом KEK.
// AEAD — AES-256-GCM: «Кузнечик»-MGM в поставке GoGOST 7 нет (third_party),
// в промышленной эксплуатации — сертифицированное СКЗИ за тем же интерфейсом
// (storage/journal.Cipher).
//
// Слой: infrastructure/security — технический механизм защиты (AD-1), без
// модульных подпапок; используется storage/journal и демо-инструментами.
//
// Цель шифрования (сказать на защите, AD-23): защищает резервные копии,
// носители и чтение таблиц без KEK; не защищает от взломанного ant и
// администратора ОС (KEK доступен процессу); на проекции не распространяется.
//
// Требования: FR-75, AD-23, AD-34. Владелец: эпик 29 (доверие).
package atrest

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.stargrave.org/gogost/v7/gost34112012256"
)

// AlgAES256GCM — имя AEAD в sealed_block.aead (contracts/journal/entry.schema.json).
const AlgAES256GCM = "aes_256_gcm"

// KeySize — длина KEK и DEK, байт.
const KeySize = 32

// KEK — ключ шифрования ключей (AD-23): монтируется только на чтение в ant и
// verifier из отдельного тома.
type KEK struct {
	id   string
	aead cipher.AEAD
}

var _ interface {
	AEAD() string
	KEKID() string
} = (*KEK)(nil)

// ErrNoKEK — файла KEK нет.
var ErrNoKEK = errors.New("atrest: файла KEK нет")

// Load читает KEK из файла: 64 hex-символа (32 байта).
func Load(path string) (*KEK, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrNoKEK, path)
	}
	if err != nil {
		return nil, err
	}
	raw, err := hex.DecodeString(strings.TrimSpace(string(b)))
	if err != nil || len(raw) != KeySize {
		return nil, fmt.Errorf("atrest: KEK %s — ожидается 64 hex-символа", path)
	}
	return New(raw)
}

// New — KEK из 32 байт. Идентификатор — `kek-` + 16 hex Стрибога ключа:
// по нему ищутся обёртки в journal.dek_wraps.
func New(raw []byte) (*KEK, error) {
	a, err := gcm(raw)
	if err != nil {
		return nil, err
	}
	h := gost34112012256.New()
	h.Write([]byte("ant-kek-id\x00"))
	h.Write(raw)
	return &KEK{id: "kek-" + hex.EncodeToString(h.Sum(nil))[:16], aead: a}, nil
}

// Generate создаёт файл KEK (0400), если его ещё нет. Возвращает true, если создан.
func Generate(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	raw := make([]byte, KeySize)
	if _, err := rand.Read(raw); err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(hex.EncodeToString(raw)+"\n"), 0o400)
}

func gcm(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("atrest: ключ %d байт, нужно %d", len(key), KeySize)
	}
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(b)
}

// AEAD — алгоритм блока.
func (k *KEK) AEAD() string { return AlgAES256GCM }

// KEKID — идентификатор ключа.
func (k *KEK) KEKID() string { return k.id }

// NewDEK — новый DEK, его идентификатор и обёртка (нонс ‖ шифротекст;
// дополнительные данные — идентификатор DEK: обёртку нельзя подставить другому).
func (k *KEK) NewDEK() (string, []byte, []byte, error) {
	var idb [12]byte
	if _, err := rand.Read(idb[:]); err != nil {
		return "", nil, nil, err
	}
	id := "dek-" + hex.EncodeToString(idb[:])
	dek := make([]byte, KeySize)
	if _, err := rand.Read(dek); err != nil {
		return "", nil, nil, err
	}
	nonce, ct, err := seal(k.aead, dek, []byte(id))
	if err != nil {
		return "", nil, nil, err
	}
	return id, dek, append(nonce, ct...), nil
}

// Unwrap — DEK из обёртки.
func (k *KEK) Unwrap(dekID string, wrapped []byte) ([]byte, error) {
	n := k.aead.NonceSize()
	if len(wrapped) < n {
		return nil, errors.New("atrest: обёртка DEK короче нонса")
	}
	dek, err := k.aead.Open(nil, wrapped[:n], wrapped[n:], []byte(dekID))
	if err != nil {
		return nil, fmt.Errorf("atrest: обёртка DEK %s не открывается этим KEK: %w", dekID, err)
	}
	return dek, nil
}

// Seal — AEAD(dek, plain, aad).
func (k *KEK) Seal(dek, plain, aad []byte) ([]byte, []byte, error) {
	a, err := gcm(dek)
	if err != nil {
		return nil, nil, err
	}
	return seal(a, plain, aad)
}

// Open — расшифрование блока и проверка тега.
func (k *KEK) Open(dek, nonce, ct, aad []byte) ([]byte, error) {
	a, err := gcm(dek)
	if err != nil {
		return nil, err
	}
	return a.Open(nil, nonce, ct, aad)
}

func seal(a cipher.AEAD, plain, aad []byte) ([]byte, []byte, error) {
	nonce := make([]byte, a.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return nonce, a.Seal(nil, nonce, plain, aad), nil
}
