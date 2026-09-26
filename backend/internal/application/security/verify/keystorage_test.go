package verify

import (
	"strings"
	"testing"
)

// AD-14, Д-72: отчёт верификатора называет подписи людей ключом в браузере
// отдельно от физического ключа — класс берётся из актов регистрации.
func TestKeyStorageLine(t *testing.T) {
	v := &run{}
	if v.keyStorageLine() != "" {
		t.Fatal("без актов с классом строки нет")
	}
	v.storage = map[string][2]string{"ins-01-ta@1": {"software_browser", "extension"}, "hqc-01-hw@1": {"hardware_token", ""}}
	v.sigByKey = map[string]int{"ins-01-ta@1": 2, "hqc-01-hw@1": 1, "engine@1": 40}
	got := v.keyStorageLine()
	if !strings.Contains(got, "ключ в браузере (расширение) — 2") || !strings.Contains(got, "физический ключ — 1") || strings.Contains(got, "40") {
		t.Fatalf("строка: %s", got)
	}
}
