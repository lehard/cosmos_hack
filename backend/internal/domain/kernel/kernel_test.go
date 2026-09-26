package kernel

import "testing"

// Эталон RFC 9562 приложение A.4: UUIDv5(NameSpace_DNS, "www.example.com").
func TestUUIDv5(t *testing.T) {
	got := UUIDv5("6ba7b810-9dad-11d1-80b4-00c04fd430c8", "www.example.com")
	if want := "2ed6657d-e927-568b-95e1-2665a8aea6a2"; got != want {
		t.Fatalf("UUIDv5 = %s, ждали %s", got, want)
	}
}

func TestNewReactionForeignType(t *testing.T) {
	if _, err := NewReaction("item", "quality.signal.raised", Slot{}, nil); err == nil {
		t.Fatal("item не эмитит quality.signal.raised — ждали ошибку AD-40")
	}
}

// Эталон партиции (AD-6, AD-41): FNV-1a 32 над байтами item_id mod P —
// одна функция у приёма, движка, стадии и верификатора; значения
// посчитаны независимо (offset 0x811c9dc5, prime 0x01000193).
func TestPartitionOf(t *testing.T) {
	for _, c := range []struct {
		item string
		p    int
		want int
	}{
		{"ENT01:I-1", 16, 14}, // 0xca2c5c0e
		{"ENT01:I-7", 16, 4},  // 0xcc2c5f34
		{"ENT01:I-7", 64, 52},
		{"ENT01:I-7", 1, 0},
		{"ENT01:I-7", 0, 0},
	} {
		if got := PartitionOf(c.item, c.p); got != c.want {
			t.Errorf("PartitionOf(%q, %d) = %d, ждали %d", c.item, c.p, got, c.want)
		}
	}
}
