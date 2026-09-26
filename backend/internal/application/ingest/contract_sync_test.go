package ingest_test

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	dom "ant/internal/domain/ingest"
)

// Сверка доменных правил приёма с файлами contracts/ (в домене нет ввода-вывода,
// поэтому сверка — здесь).

func vectors(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repo, "contracts/crypto/test-vectors/vectors.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]json.RawMessage
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// JCS и Стрибог — на тест-векторах контракта (AD-10, AD-44): отпечаток сервера
// совпадает с отпечатком источника.
func TestCanonicalizeAndDigestVectors(t *testing.T) {
	v := vectors(t)
	var jcs []struct {
		Input     string `json:"input"`
		Canonical string `json:"canonical_utf8"`
	}
	if err := json.Unmarshal(v["jcs"], &jcs); err != nil {
		t.Fatal(err)
	}
	for _, c := range jcs {
		got, err := dom.Canonicalize([]byte(c.Input))
		if err != nil || string(got) != c.Canonical {
			t.Fatalf("JCS(%s) = %s, %v; ждали %s", c.Input, got, err, c.Canonical)
		}
	}
	var st []struct {
		Msg    string `json:"message_hex"`
		Digest string `json:"digest_hex"`
	}
	if err := json.Unmarshal(v["streebog256"], &st); err != nil {
		t.Fatal(err)
	}
	for _, c := range st {
		m, _ := hex.DecodeString(c.Msg)
		if got := dom.Digest(m); got != "streebog256:"+c.Digest {
			t.Fatalf("dom.Digest = %s, ждали %s", got, c.Digest)
		}
	}
	var se struct {
		JSON      json.RawMessage `json:"json"`
		Canonical string          `json:"canonical_utf8"`
	}
	if err := json.Unmarshal(v["sample_event"], &se); err != nil {
		t.Fatal(err)
	}
	if got, err := dom.Canonicalize(se.JSON); err != nil || string(got) != se.Canonical {
		t.Fatalf("JCS(sample_event) = %s, %v", got, err)
	}
}

// Перечень полей безопасности совпадает с contracts/events/safety-critical-enums.yaml.
func TestSafetyCriticalEnumsMatchContract(t *testing.T) {
	f, err := os.Open(filepath.Join(repo, "contracts/events/safety-critical-enums.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	got := map[string][]string{}
	in := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "fields:") {
			in = true
			continue
		}
		if !in || !strings.HasPrefix(line, "  ") {
			continue
		}
		k, v, _ := strings.Cut(strings.TrimSpace(line), ":")
		v = strings.Trim(strings.TrimSpace(v), "[]")
		got[k] = []string{}
		for p := range strings.SplitSeq(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				got[k] = append(got[k], p)
			}
		}
	}
	if len(got) != len(dom.SafetyCriticalEnums) {
		t.Fatalf("в контракте %d типов, в коде %d", len(got), len(dom.SafetyCriticalEnums))
	}
	for _, k := range slices.Sorted(maps.Keys(got)) {
		if v := got[k]; !slices.Equal(v, dom.SafetyCriticalEnums[k]) {
			t.Fatalf("%s: контракт %v, код %v", k, v, dom.SafetyCriticalEnums[k])
		}
	}
}
