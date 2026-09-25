package gost

import (
	"encoding/hex"
	"testing"

	"go.stargrave.org/gogost/v7/gost34112012256"
)

// TestStreebog256Vector — контрольный пример 1 RFC 6986 (ГОСТ Р 34.11-2012):
// поставленный в third_party код GoGOST даёт эталонный хеш.
func TestStreebog256Vector(t *testing.T) {
	m := []byte("012345678901234567890123456789012345678901234567890123456789012")
	const want = "9d151eefd8590b89daa6ba6cb74af9275dd051026bb149a452fd84e5e57b5500"
	h := gost34112012256.New()
	h.Write(m)
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		t.Fatalf("Стрибог-256: %s, ждали %s", got, want)
	}
}
