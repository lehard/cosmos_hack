package journal

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strings"

	"go.stargrave.org/gogost/v7/gost34112012256"

	jc "ant/internal/contracts/journal"
)

// Формат цепочки v1 (contracts/journal/chain-format.v1.md, AD-44, AD-23):
//
//	H(x)      = Стрибог-256(x)                  — 32 байта в порядке hash.Sum()
//	commit    = H(salt ‖ JCS(конверт DSSE))
//	open_hash = H(JCS(открытые поля без link))
//	link      = H(prev_link ‖ commit ‖ open_hash)
//
// Все функции здесь чистые: их исполняют journal.Append, хранитель и
// независимый верификатор (AD-9) — одна реализация на всех.

// HashPrefix — префикс алгоритма в строковом виде хеша (AD-44: отпечаток
// документа, адрес материала, commit и link — `streebog256:‹hex›`).
const HashPrefix = "streebog256:"

// FormatVersion — версия формата цепочки. Смена H — новый сегмент (AD-32).
const FormatVersion = 1

// Digest — значение H: 32 байта в порядке hash.Sum() GoGOST.
type Digest [32]byte

// ZeroLink — prev_link первой записи каждой цепочки: 32 нулевых байта.
var ZeroLink Digest

// ErrDigest — строка не является хешем `streebog256:‹64 hex›`.
var ErrDigest = errors.New("journal: ожидается streebog256:‹64 hex›")

// H — Стрибог-256 (ГОСТ Р 34.11-2012) от склейки сырых байтов частей.
func H(parts ...[]byte) Digest {
	h := gost34112012256.New()
	for _, p := range parts {
		h.Write(p)
	}
	var d Digest
	copy(d[:], h.Sum(nil))
	return d
}

// String — `streebog256:` + 64 hex.
func (d Digest) String() string { return HashPrefix + hex.EncodeToString(d[:]) }

// Bytes — сырые 32 байта.
func (d Digest) Bytes() []byte { return d[:] }

// ParseDigest разбирает строку `streebog256:‹64 hex›`.
func ParseDigest(s string) (Digest, error) {
	var d Digest
	h, ok := strings.CutPrefix(s, HashPrefix)
	if !ok || len(h) != 64 {
		return d, fmt.Errorf("%w: %q", ErrDigest, s)
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return d, fmt.Errorf("%w: %q", ErrDigest, s)
	}
	copy(d[:], b)
	return d, nil
}

// DigestFromBytes — хеш из сырых 32 байтов (например, из столбца bytea).
func DigestFromBytes(b []byte) (Digest, error) {
	var d Digest
	if len(b) != len(d) {
		return d, fmt.Errorf("%w: длина %d", ErrDigest, len(b))
	}
	copy(d[:], b)
	return d, nil
}

// Canonical — JCS (RFC 8785) над JSON-значением raw. Реализация — stdlib
// encoding/json/jsontext (спайн, «Stack»).
func Canonical(raw []byte) ([]byte, error) {
	v := jsontext.Value(bytes.Clone(raw))
	if err := v.Canonicalize(); err != nil {
		return nil, fmt.Errorf("JCS: %w", err)
	}
	return []byte(v), nil
}

// Commit — обязательство записи: H(salt ‖ JCS(конверт)) (AD-23, AD-44).
// envelope — байты конверта DSSE (канонизируются здесь же); salt — 128
// случайных бит на запись (SaltSize), хранится в зашифрованном блоке: по
// открытому commit нельзя перебором узнать содержимое.
func Commit(salt, envelope []byte) (Digest, error) {
	c, err := Canonical(envelope)
	if err != nil {
		return Digest{}, fmt.Errorf("конверт: %w", err)
	}
	return H(salt, c), nil
}

// OpenFields — JCS открытых полей записи: все поля entry.schema.json, кроме
// link и sealed; отсутствующие необязательные поля опущены, null — только у
// causation_id и supersedes (chain-format.v1.md). commit входит в открытые поля.
func OpenFields(e jc.JournalEntry) ([]byte, error) {
	raw, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	delete(m, "link")
	delete(m, "sealed")
	raw, err = json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return Canonical(raw)
}

// OpenHash — H(JCS(открытые поля без link)).
func OpenHash(e jc.JournalEntry) (Digest, error) {
	b, err := OpenFields(e)
	if err != nil {
		return Digest{}, err
	}
	return H(b), nil
}

// Link — звено: H(prev_link ‖ commit ‖ open_hash) над сырыми 32-байтными значениями.
func Link(prev, commit, openHash Digest) Digest {
	return H(prev[:], commit[:], openHash[:])
}

// Seal ставит записи commit и link по формуле AD-44: e должна быть полностью
// заполнена (включая seq, chain, committed_at, recorded_at), кроме commit и link.
// Возвращает звено и JCS открытых полей (заголовок, который хранится как есть).
func Seal(e *jc.JournalEntry, prev Digest, salt, envelope []byte) (Digest, []byte, error) {
	commit, err := Commit(salt, envelope)
	if err != nil {
		return Digest{}, nil, err
	}
	e.Commit = commit.String()
	e.Link = ""
	open, err := OpenFields(*e)
	if err != nil {
		return Digest{}, nil, err
	}
	link := Link(prev, commit, H(open))
	e.Link = link.String()
	return link, open, nil
}

// Break — место и вид нарушения цепочки.
type Break struct {
	Chain  string
	Seq    int
	Reason string
}

func (b *Break) Error() string {
	return fmt.Sprintf("цепочка %s, seq %d: %s", b.Chain, b.Seq, b.Reason)
}

// VerifyLinks проверяет последовательный отрезок одной цепочки от звена prev:
// seq идут подряд, звено каждой записи сходится с формулой, commit — по
// открытому полю (сам commit сверяется с конвертом в VerifyCommit). Так
// проверяет хранитель (AD-8) и верификатор (AD-9). Возвращает звено последней записи.
func VerifyLinks(prev Digest, prevSeq int, entries []jc.JournalEntry) (Digest, error) {
	for _, e := range entries {
		if e.Seq != prevSeq+1 {
			return prev, &Break{Chain: string(e.Chain), Seq: e.Seq, Reason: fmt.Sprintf("разрыв seq: ожидался %d", prevSeq+1)}
		}
		commit, err := ParseDigest(e.Commit)
		if err != nil {
			return prev, &Break{Chain: string(e.Chain), Seq: e.Seq, Reason: "commit: " + err.Error()}
		}
		link, err := ParseDigest(e.Link)
		if err != nil {
			return prev, &Break{Chain: string(e.Chain), Seq: e.Seq, Reason: "link: " + err.Error()}
		}
		oh, err := OpenHash(e)
		if err != nil {
			return prev, &Break{Chain: string(e.Chain), Seq: e.Seq, Reason: "открытые поля: " + err.Error()}
		}
		if Link(prev, commit, oh) != link {
			return prev, &Break{Chain: string(e.Chain), Seq: e.Seq, Reason: "звено не сходится"}
		}
		prev, prevSeq = link, e.Seq
	}
	return prev, nil
}

// VerifyCommit сверяет commit записи с конвертом и солью (после расшифрования
// commit проверяется всегда, AD-23).
func VerifyCommit(e jc.JournalEntry, salt, envelope []byte) error {
	c, err := Commit(salt, envelope)
	if err != nil {
		return err
	}
	if c.String() != e.Commit {
		return &Break{Chain: string(e.Chain), Seq: e.Seq, Reason: "commit не сходится с конвертом"}
	}
	return nil
}
