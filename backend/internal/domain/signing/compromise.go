package signing

import (
	"slices"
	"time"
)

// Защитная реакция на компрометацию ключа (AD-11, AD-27, FR-79): отзыв несёт
// «скомпрометирован с X»; решения, подписанные этим ключом после X (и до
// отзыва — после отзыва подпись не принимается вовсе), получают сдерживание
// «подпись под сомнением» и задачу на переподписание. Реакцию исполняют
// модули-владельцы осей (nonconformity — сдерживание, notifications —
// задачи, AD-30); здесь — чистое вычисление затронутого набора.

// SignedDecision — запись журнала, подписанная ключом (решение человека).
type SignedDecision struct {
	EventID     string
	EventType   string
	ItemID      string
	Seq         int64
	CommittedAt time.Time
	KeyRefs     []string
	// Signer — псевдоним подписанта.
	Signer string
}

// Doubt — решение «под сомнением» и что с ним делать.
type Doubt struct {
	SignedDecision
	KeyRef string
	// Contain — изделие ставится на сдерживание «подпись под сомнением».
	Contain bool
	// Resign — задача подписанту (или его руководителю) переподписать решение.
	Resign bool
}

// ContainmentReason — код основания сдерживания для nonconformity.
const ContainmentReason = "signature_doubtful"

// ProtectiveReaction — решения, подписанные отозванным ключом после
// «скомпрометирован с X», по порядку seq. Отзыв без компрометации затрагивает
// только будущие подписи — прошлые решения остаются в силе.
func ProtectiveReaction(k Key, decisions []SignedDecision) []Doubt {
	rv := k.Revoked
	if rv == nil || (rv.CompromisedSince == nil && rv.CompromisedSinceSeq == 0) {
		return nil
	}
	var out []Doubt
	for _, d := range decisions {
		if !slices.Contains(d.KeyRefs, k.KeyRef) || d.Seq >= rv.Seq || !k.CompromisedAt(d.Seq, d.CommittedAt) {
			continue
		}
		out = append(out, Doubt{SignedDecision: d, KeyRef: k.KeyRef, Contain: d.ItemID != "", Resign: true})
	}
	slices.SortFunc(out, func(a, b Doubt) int { return int(a.Seq - b.Seq) })
	return out
}

// CompromisedSinceSeq — X как позиция журнала: первая запись с committed_at ≥ X
// (AD-37: криптографические моменты — по seq и committed_at). commits — пары
// (seq, committed_at) по возрастанию seq; нет такой записи — после головы.
func CompromisedSinceSeq(x time.Time, commits []SeqTime) int64 {
	last := int64(0)
	for _, c := range commits {
		if !c.At.Before(x) {
			return c.Seq
		}
		last = c.Seq
	}
	return last + 1
}

// SeqTime — позиция записи и её committed_at.
type SeqTime struct {
	Seq int64
	At  time.Time
}
