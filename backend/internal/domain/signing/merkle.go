package signing

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"slices"
)

// Сменный рапорт — уровень подписи 3 (FR-66, AD-12, AD-13): одна подпись над
// корнем дерева Меркла RFC 6962 по отпечаткам подписей смены из локального
// журнала агента токена; счётчики по типам. Сервер сверяет набор листьев со
// своим журналом по окну смены (FR-81); расхождение — тревога
// agent_journal_mismatch (AD-14). Бумажные подписи в рапорт не входят: их
// покрывают заверение и реестр бумажных решений (AD-9).

// MerkleLeaf — хеш листа RFC 6962: H(0x00 ‖ d).
func MerkleLeaf(d []byte) []byte { return Hash([]byte{0}, d) }

// MerkleNode — хеш узла RFC 6962: H(0x01 ‖ l ‖ r).
func MerkleNode(l, r []byte) []byte { return Hash([]byte{1}, l, r) }

// MerkleRoot — корень RFC 6962 (MTH) над данными листьев d по порядку:
// разбиение по наибольшей степени двойки меньше n; пустое дерево — H("").
func MerkleRoot(leaves [][]byte) []byte {
	switch len(leaves) {
	case 0:
		return Hash()
	case 1:
		return MerkleLeaf(leaves[0])
	}
	k := 1
	for k<<1 < len(leaves) {
		k <<= 1
	}
	return MerkleNode(MerkleRoot(leaves[:k]), MerkleRoot(leaves[k:]))
}

// MerkleRootString — корень в записи streebog256:‹hex›.
func MerkleRootString(leaves [][]byte) string {
	return HashPrefix + hex.EncodeToString(MerkleRoot(leaves))
}

// ShiftReport — содержимое сменного рапорта (contracts/internal/token-agent/shift-report.v1.json).
type ShiftReport struct {
	FormatVersion  int              `json:"format_version"`
	CryptoProfile  string           `json:"crypto_profile"`
	Signers        []string         `json:"signers"`
	PersonID       string           `json:"person_id"`
	ShiftID        string           `json:"shift_id"`
	WorkplaceID    string           `json:"workplace_id,omitempty"`
	WindowFrom     string           `json:"window_from"`
	WindowTo       string           `json:"window_to"`
	MerkleRoot     string           `json:"merkle_root"`
	LeafCount      int64            `json:"leaf_count"`
	CountsByType   map[string]int64 `json:"counts_by_type"`
	FirstLocalSeq  int64            `json:"first_local_seq,omitempty"`
	LastLocalSeq   int64            `json:"last_local_seq,omitempty"`
	SeenCheckpoint int64            `json:"seen_checkpoint,omitempty"`
}

// ShiftLeaf — подпись смены: отпечаток подписи (SignatureDigest), тип
// записи и позиция в журнале (агент получает seq квитанцией приёма).
type ShiftLeaf struct {
	Digest    []byte
	EventType string
	Seq       int64
}

// BuildShiftReport — корень и счётчики рапорта над подписями смены: листья
// упорядочены по seq журнала (агент хранит seq рядом с записью локального
// журнала — порядок у агента и сервера один и тот же).
func BuildShiftReport(base ShiftReport, leaves []ShiftLeaf) ShiftReport {
	ls := slices.Clone(leaves)
	slices.SortStableFunc(ls, func(a, b ShiftLeaf) int { return int(a.Seq - b.Seq) })
	d := make([][]byte, len(ls))
	base.CountsByType = map[string]int64{}
	for i, l := range ls {
		d[i] = l.Digest
		base.CountsByType[l.EventType]++
	}
	base.MerkleRoot, base.LeafCount = MerkleRootString(d), int64(len(ls))
	return base
}

// ErrShiftMismatch — рапорт агента расходится с журналом сервера (тревога
// agent_journal_mismatch, AD-14).
var ErrShiftMismatch = errors.New("signing: сменный рапорт расходится с журналом сервера")

// ShiftDiff — чем рапорт расходится с журналом сервера.
type ShiftDiff struct {
	RootAgent, RootServer string
	CountAgent, CountSrv  int64
	// Types — типы, по которым расходятся счётчики, по порядку.
	Types []string
}

// CompareShift — FR-81, AD-12: сервер строит тот же рапорт по своему журналу
// (подписи этого человека в окне смены, без бумажных) и сверяет корень,
// число листьев и счётчики по типам. Совпало — nil.
func CompareShift(agent ShiftReport, server []ShiftLeaf) (ShiftDiff, error) {
	srv := BuildShiftReport(ShiftReport{}, server)
	d := ShiftDiff{RootAgent: agent.MerkleRoot, RootServer: srv.MerkleRoot, CountAgent: agent.LeafCount, CountSrv: srv.LeafCount}
	types := slices.Sorted(maps.Keys(srv.CountsByType))
	for _, t := range slices.Sorted(maps.Keys(agent.CountsByType)) {
		if !slices.Contains(types, t) {
			types = append(types, t)
		}
	}
	slices.Sort(types)
	for _, t := range types {
		if agent.CountsByType[t] != srv.CountsByType[t] {
			d.Types = append(d.Types, t)
		}
	}
	if agent.MerkleRoot != srv.MerkleRoot || agent.LeafCount != srv.LeafCount || len(d.Types) > 0 {
		return d, fmt.Errorf("%w: корень агента %s, сервера %s; листьев %d и %d", ErrShiftMismatch,
			agent.MerkleRoot, srv.MerkleRoot, agent.LeafCount, srv.LeafCount)
	}
	return d, nil
}

// EqualDigest — сравнение отпечатков.
func EqualDigest(a, b []byte) bool { return bytes.Equal(a, b) }
