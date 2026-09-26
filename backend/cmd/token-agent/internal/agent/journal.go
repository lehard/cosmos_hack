package agent

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	dom "ant/internal/domain/signing"
)

// Локальный журнал подписанного и ограничение частоты (AD-14, PRD §11.16):
// код подписи в браузере приходит с сервера, поэтому ограничения уровня 1
// (перечень, частота, журнал) обеспечивает сам пакет подписи. Сменный рапорт
// (уровень 3) — корень Меркла RFC 6962 по отпечаткам подписей журнала.

// Entry — запись локального журнала подписанного.
type Entry struct {
	Seq       int64  `json:"seq"`
	EventType string `json:"event_type,omitempty"`
	Level     int    `json:"level"`
	// Digest — отпечаток подписи (лист сменного рапорта, AD-12); DocDigest —
	// отпечаток подписанного содержимого.
	Digest    string `json:"digest"`
	DocDigest string `json:"doc_digest,omitempty"`
	SignedAt  string `json:"signed_at"`
	ServerSeq int64  `json:"server_seq,omitempty"`
}

// Append — дописать запись о подписи env; номер — следующий за последним.
func Append(j []Entry, p Prepared, env dom.Envelope) ([]Entry, Entry) {
	var seq int64 = 1
	if n := len(j); n > 0 {
		seq = j[n-1].Seq + 1
	}
	sd := p.DocDigest
	if len(env.Signatures) > 0 {
		sd = dom.HashPrefix + hex.EncodeToString(dom.SignatureDigest(p.PayloadType, p.Payload(), env.Signatures[0]))
	}
	e := Entry{Seq: seq, EventType: p.EventType, Level: p.Level, Digest: sd, DocDigest: p.DocDigest, SignedAt: p.ClientSignedAt}
	return append(j, e), e
}

// JournalLimit — сколько последних записей держит журнал в браузере.
const JournalLimit = 5000

// Trim — оставить последние JournalLimit записей.
func Trim(j []Entry) []Entry {
	if len(j) > JournalLimit {
		return j[len(j)-JournalLimit:]
	}
	return j
}

// RateLimit — не больше Max подписей уровня 1 за Window.
type RateLimit struct {
	Max    int
	Window time.Duration
}

// DefaultRate — 20 подписей уровня 1 в минуту: сотни «тихих» подписей
// взломанной страницей упираются в предел и видны в журнале.
var DefaultRate = RateLimit{Max: 20, Window: time.Minute}

// RateCheck — можно ли подписать уровнем 1 сейчас; иначе — через сколько.
// Уровень 2 не ограничивается: у него окно подтверждения.
func RateCheck(j []Entry, now time.Time, lim RateLimit) (time.Duration, bool) {
	if lim.Max <= 0 {
		return 0, true
	}
	from := now.Add(-lim.Window)
	var inWindow []time.Time
	for i := len(j) - 1; i >= 0; i-- {
		if j[i].Level != dom.Level1 {
			continue
		}
		t, err := time.Parse(TimeLayout, j[i].SignedAt)
		if err != nil {
			continue
		}
		if !t.After(from) {
			break
		}
		inWindow = append(inWindow, t)
	}
	if len(inWindow) < lim.Max {
		return 0, true
	}
	oldest := inWindow[len(inWindow)-1]
	return oldest.Add(lim.Window).Sub(now), false
}

// ErrWindow — окно сменного рапорта задано неверно.
var ErrWindow = errors.New("agent: окно сменного рапорта")

// ShiftReport — сменный рапорт по журналу за окно [from, to] (AD-12).
func ShiftReport(j []Entry, base dom.ShiftReport) (dom.ShiftReport, error) {
	from, err1 := time.Parse(TimeLayout, base.WindowFrom)
	to, err2 := time.Parse(TimeLayout, base.WindowTo)
	if err1 != nil || err2 != nil || to.Before(from) {
		return base, ErrWindow
	}
	var leaves []dom.ShiftLeaf
	for _, e := range j {
		t, err := time.Parse(TimeLayout, e.SignedAt)
		if err != nil || t.Before(from) || t.After(to) {
			continue
		}
		d, err := hex.DecodeString(strings.TrimPrefix(e.Digest, dom.HashPrefix))
		if err != nil {
			continue
		}
		seq := e.Seq
		if e.ServerSeq > 0 {
			seq = e.ServerSeq
		}
		leaves = append(leaves, dom.ShiftLeaf{Digest: d, EventType: e.EventType, Seq: seq})
	}
	return dom.BuildShiftReport(base, leaves), nil
}
