package engine

import (
	"encoding/hex"
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"

	"go.stargrave.org/gogost/v7/gost34112012256"

	"ant/internal/domain/kernel"
)

// HashPrefix — префикс алгоритма у хешей движка: тот же H, что у формата
// цепочки v1 (Стрибог-256, AD-44).
const HashPrefix = "streebog256:"

// Canonical — канонический JSON значения (RFC 8785, AD-10): ключи объектов
// отсортированы, числа и строки в нормальной форме. Одно и то же значение
// даёт одни и те же байты в любом процессе — на этом стоят сравнение реакций
// (AD-3) и хеш состояния (NFR-DET-1).
func Canonical(v any) ([]byte, error) {
	var raw []byte
	switch x := v.(type) {
	case json.RawMessage:
		raw = x
	case []byte:
		raw = x
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	if len(raw) == 0 {
		raw = []byte("null")
	}
	val := jsontext.Value(append([]byte(nil), raw...))
	if err := val.Canonicalize(); err != nil {
		return nil, fmt.Errorf("канонический JSON: %w", err)
	}
	return []byte(val), nil
}

// Hash — H(байты) в виде `streebog256:‹hex›` (AD-44).
func Hash(b []byte) string {
	h := gost34112012256.New()
	h.Write(b)
	return HashPrefix + hex.EncodeToString(h.Sum(nil))
}

// fingerprintBody — то, что определяет «вывод» реакции при сравнении с
// записанной (AD-3): тип, данные, причины, ревизия правила и режим. Версия,
// supersedes и basis_seq в отпечаток не входят — это координаты записи, а не
// содержание вывода.
type fingerprintBody struct {
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Causes  []string        `json:"causes"`
	RuleRev string          `json:"rule_rev"`
	Mode    int             `json:"mode"`
}

func fingerprint(t string, data json.RawMessage, causes []string, ruleRev string, mode int) (string, error) {
	d, err := Canonical(data)
	if err != nil {
		return "", err
	}
	if causes == nil {
		causes = []string{}
	}
	b, err := Canonical(fingerprintBody{Type: t, Data: d, Causes: causes, RuleRev: ruleRev, Mode: mode})
	if err != nil {
		return "", err
	}
	return Hash(b), nil
}

// Fingerprint — отпечаток вычисленной реакции для сравнения со слотом
// записанной (AD-3, AD-5).
func Fingerprint(r kernel.Reaction) (string, error) {
	d, err := json.Marshal(r.Data)
	if err != nil {
		return "", fmt.Errorf("реакция %s: данные: %w", r.Type, err)
	}
	return fingerprint(string(r.Type), d, r.Causes, r.RuleRev, r.AutomationMode)
}

// Fingerprint — отпечаток записанной версии слота; совпадает с отпечатком
// вычисленной реакции тогда и только тогда, когда вывод не изменился.
func (r Recorded) Fingerprint() (string, error) {
	return fingerprint(string(r.Type), r.Data, r.Causes, r.RuleRev, r.AutomationMode)
}

// hashedState — то, что входит в хеш состояния: состояние модулей и
// вычисленные реакции (слот, тип, отпечаток). basis_seq, время записи и
// история версий не входят (AD-6: state_hash — только итоговое состояние).
type hashedState struct {
	Modules   Snapshot         `json:"modules"`
	Reactions []hashedReaction `json:"reactions"`
}

type hashedReaction struct {
	Slot        string `json:"slot"`
	Type        string `json:"type"`
	Fingerprint string `json:"fingerprint"`
}

// StateHash — хеш итога свёртки изделия: состояние всех модулей композиции и
// вычисленные реакции (NFR-DET-1, AD-4, AD-6). Две свёртки одного входа в
// разных процессах, на 1 и N воркерах, у верификатора дают один хеш.
func StateHash(s Snapshot, reactions []kernel.Reaction) (string, error) {
	h := hashedState{Modules: s, Reactions: make([]hashedReaction, 0, len(reactions))}
	h.Modules.BasisSeq = 0
	for _, r := range reactions {
		fp, err := Fingerprint(r)
		if err != nil {
			return "", err
		}
		h.Reactions = append(h.Reactions, hashedReaction{Slot: r.Slot.Key(), Type: string(r.Type), Fingerprint: fp})
	}
	b, err := Canonical(h)
	if err != nil {
		return "", err
	}
	return Hash(b), nil
}
