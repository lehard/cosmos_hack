package materials

import "testing"

// Контрольный пример ГОСТ Р 34.11-2012 M1 (contracts/crypto/test-vectors,
// раздел streebog256): адрес — тот же Стрибог-256 с префиксом.
func TestAddressStreebogM1(t *testing.T) {
	m1 := []byte("012345678901234567890123456789012345678901234567890123456789012")
	want := "streebog256:9d151eefd8590b89daa6ba6cb74af9275dd051026bb149a452fd84e5e57b5500"
	if got := Address(m1); got != want {
		t.Fatalf("Address(M1) = %s, want %s", got, want)
	}
	if h, err := ParseAddress(want); err != nil || len(h) != 64 {
		t.Fatalf("ParseAddress: %q %v", h, err)
	}
	for _, bad := range []string{"", "sha256:00", Prefix + "zz", Prefix + "9D151EEFD8590B89DAA6BA6CB74AF9275DD051026BB149A452FD84E5E57B5500"} {
		if _, err := ParseAddress(bad); err == nil {
			t.Fatalf("ParseAddress(%q) без ошибки", bad)
		}
	}
}
