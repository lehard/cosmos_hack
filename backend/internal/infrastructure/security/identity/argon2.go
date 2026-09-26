package identity

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"ant/internal/application/access"
)

// Argon2id — хеш паролей argon2id (FR-128, AD-15: готовый компонент
// golang.org/x/crypto, своей криптографии нет) в формате PHC
// `$argon2id$v=19$m=…,t=…,p=…$соль$хеш`. Параметры по умолчанию — минимум
// OWASP (19 МиБ, 2 прохода, 1 поток): машина демо — 2 ГБ памяти.
type Argon2id struct {
	Memory  uint32 // КиБ
	Time    uint32
	Threads uint8
	KeyLen  uint32
	SaltLen int
}

// DefaultArgon2id — параметры хеша по умолчанию.
var DefaultArgon2id = Argon2id{Memory: 19 * 1024, Time: 2, Threads: 1, KeyLen: 32, SaltLen: 16}

var _ access.PasswordHasher = Argon2id{}

// Hash — хеш пароля со случайной солью.
func (a Argon2id) Hash(password string) (string, error) {
	salt := make([]byte, a.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, a.Time, a.Memory, a.Threads, a.KeyLen)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, a.Memory, a.Time, a.Threads,
		enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

// Verify — пароль соответствует хешу (сравнение за постоянное время).
func (Argon2id) Verify(hash, password string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("identity: хеш не argon2id")
	}
	var v int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &v); err != nil || v != argon2.Version {
		return false, errors.New("identity: версия argon2id")
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, errors.New("identity: параметры argon2id")
	}
	enc := base64.RawStdEncoding
	salt, err := enc.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := enc.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
