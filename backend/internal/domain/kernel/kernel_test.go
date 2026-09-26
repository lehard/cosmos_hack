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
