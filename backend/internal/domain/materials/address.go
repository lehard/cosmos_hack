package materials

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"go.stargrave.org/gogost/v7/gost34112012256"
)

// Адрес материала — H(байты) с префиксом алгоритма `streebog256:‹64 hex›`
// (AD-23, AD-44): тот же H, что у звеньев журнала и отпечатков документов.
// В журнале — только адрес и метаданные; байты — в MaterialStore.

// Prefix — префикс алгоритма адреса.
const Prefix = "streebog256:"

// ErrAddress — строка не является адресом материала.
var ErrAddress = errors.New("materials: адрес — streebog256:‹64 hex›")

// Address — адрес содержимого data.
func Address(data []byte) string {
	h := gost34112012256.New()
	h.Write(data)
	return FormatAddress(h.Sum(nil))
}

// FormatAddress — адрес по готовому дайджесту Стрибога-256 (потоковое
// хеширование в адаптере хранилища).
func FormatAddress(sum []byte) string { return Prefix + hex.EncodeToString(sum) }

// ParseAddress проверяет адрес и возвращает 64 hex-знака дайджеста (без
// префикса) — по ним адаптер раскладывает файлы.
func ParseAddress(addr string) (string, error) {
	h, ok := strings.CutPrefix(addr, Prefix)
	if !ok || len(h) != 64 {
		return "", fmt.Errorf("%w: %q", ErrAddress, addr)
	}
	if _, err := hex.DecodeString(h); err != nil || strings.ToLower(h) != h {
		return "", fmt.Errorf("%w: %q", ErrAddress, addr)
	}
	return h, nil
}
