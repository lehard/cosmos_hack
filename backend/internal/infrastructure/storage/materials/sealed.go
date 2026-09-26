package materials

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// Шифрование материалов при хранении (AD-23): та же конвертная схема, что у
// журнала — DEK на объект, обёртка DEK ключом KEK (из отдельного тома),
// AEAD с дополнительными данными «адрес материала»; адрес — H(открытых
// байтов), после расшифрования он проверяется всегда. Обёртка лежит в
// заголовке файла материала (том материалов вне БД), поэтому ротация KEK —
// перешифрование заголовков.

// Sealer — шифр материала (реализация — infrastructure/security/atrest).
type Sealer interface {
	AEAD() string
	KEKID() string
	NewDEK() (dekID string, dek, wrapped []byte, err error)
	Unwrap(dekID string, wrapped []byte) ([]byte, error)
	Seal(dek, plain, aad []byte) (nonce, ct []byte, err error)
	Open(dek, nonce, ct, aad []byte) ([]byte, error)
}

// Option — настройка Volume.
type Option func(*Volume)

// WithSealer — шифрование материалов при хранении (AD-23); без него байты
// лежат открыто (разработка).
func WithSealer(s Sealer) Option { return func(v *Volume) { v.sealer = s } }

// sealedMagic — начало зашифрованного файла материала.
const sealedMagic = "ANT-SEALED-1\n"

type sealedHeader struct {
	AEAD    string `json:"aead"`
	KEKID   string `json:"kek_id"`
	DEKID   string `json:"dek_id"`
	Wrapped string `json:"wrapped_b64"`
	Nonce   string `json:"nonce_b64"`
}

// ErrSealed — материал зашифрован, а KEK нет (или KEK другой).
var ErrSealed = errors.New("materials: материал зашифрован — нужен KEK")

func (v *Volume) seal(plain []byte, addr string) ([]byte, error) {
	id, dek, wrapped, err := v.sealer.NewDEK()
	if err != nil {
		return nil, err
	}
	nonce, ct, err := v.sealer.Seal(dek, plain, []byte(addr))
	if err != nil {
		return nil, err
	}
	h, err := json.Marshal(sealedHeader{AEAD: v.sealer.AEAD(), KEKID: v.sealer.KEKID(), DEKID: id,
		Wrapped: base64.StdEncoding.EncodeToString(wrapped), Nonce: base64.StdEncoding.EncodeToString(nonce)})
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(sealedMagic)+len(h)+1+len(ct))
	out = append(out, sealedMagic...)
	out = append(out, h...)
	out = append(out, '\n')
	return append(out, ct...), nil
}

// open — открытые байты файла материала (зашифрованного или старого открытого).
func (v *Volume) open(raw []byte, addr string) ([]byte, error) {
	if !bytes.HasPrefix(raw, []byte(sealedMagic)) {
		return raw, nil
	}
	rest := raw[len(sealedMagic):]
	i := bytes.IndexByte(rest, '\n')
	if i < 0 {
		return nil, fmt.Errorf("%w: заголовок", ErrTampered)
	}
	var h sealedHeader
	if err := json.Unmarshal(rest[:i], &h); err != nil {
		return nil, fmt.Errorf("%w: заголовок: %v", ErrTampered, err)
	}
	if v.sealer == nil || v.sealer.KEKID() != h.KEKID {
		return nil, fmt.Errorf("%w: KEK %s", ErrSealed, h.KEKID)
	}
	wrapped, err1 := base64.StdEncoding.DecodeString(h.Wrapped)
	nonce, err2 := base64.StdEncoding.DecodeString(h.Nonce)
	if err := errors.Join(err1, err2); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTampered, err)
	}
	dek, err := v.sealer.Unwrap(h.DEKID, wrapped)
	if err != nil {
		return nil, err
	}
	plain, err := v.sealer.Open(dek, nonce, rest[i+1:], []byte(addr))
	if err != nil {
		return nil, fmt.Errorf("%w: не расшифровывается: %v", ErrTampered, err)
	}
	return plain, nil
}
