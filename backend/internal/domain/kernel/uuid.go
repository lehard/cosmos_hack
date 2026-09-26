package kernel

import (
	"crypto/sha1" // RFC 9562 §5.5: UUIDv5 определён над SHA-1; применение не криптографическое.
	"encoding/hex"
	"strings"
)

// UUIDv5 — детерминированный идентификатор по RFC 9562 §5.5 от пространства
// имён ns (строка UUID, например constants.NsAnt) и имени name. Домен не
// использует случайность (AD-4): reaction_id, id производных записей и событий
// сценария — UUIDv5 («Соглашения/Идентификаторы», AD-3, AD-38).
func UUIDv5(ns, name string) string {
	nsb, ok := parseUUID(ns)
	if !ok {
		panic("kernel.UUIDv5: неверное пространство имён " + ns)
	}
	h := sha1.New()
	h.Write(nsb[:])
	h.Write([]byte(name))
	sum := h.Sum(nil)
	var u [16]byte
	copy(u[:], sum[:16])
	u[6] = (u[6] & 0x0f) | 0x50 // версия 5
	u[8] = (u[8] & 0x3f) | 0x80 // вариант RFC 9562
	return formatUUID(u)
}

func parseUUID(s string) ([16]byte, bool) {
	var u [16]byte
	h := strings.ReplaceAll(s, "-", "")
	if len(h) != 32 {
		return u, false
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return u, false
	}
	copy(u[:], b)
	return u, true
}

func formatUUID(u [16]byte) string {
	h := hex.EncodeToString(u[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
