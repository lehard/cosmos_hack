package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"ant/cmd/token-agent/internal/agent"
	"ant/internal/contracts/crypto"
	"ant/internal/contracts/procs"
	dom "ant/internal/domain/signing"
)

// Native Messaging (AD-14, contracts/internal/token-agent): сообщение —
// 4 байта длины в порядке байтов машины и JSON. Хост — посредник между
// расширением и ядром агента; PIN держит только этот процесс (в памяти, до
// извлечения токена или конца сеанса браузера).

// ProtocolVersion — версия протокола.
const ProtocolVersion = 1

// maxMessage — предел сообщения от браузера (Chrome — до 64 МиБ; пачке 500
// решений хватает 8 МиБ).
const maxMessage = 8 << 20

func readMsg(r io.Reader) ([]byte, error) {
	var n uint32
	if err := binary.Read(r, binary.NativeEndian, &n); err != nil {
		return nil, err
	}
	if n == 0 || n > maxMessage {
		return nil, fmt.Errorf("сообщение %d байт", n)
	}
	b := make([]byte, n)
	_, err := io.ReadFull(r, b)
	return b, err
}

func writeMsg(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := binary.Write(w, binary.NativeEndian, uint32(len(b))); err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// session — состояние хоста: разблокированный ключ и отпечаток токена, на
// котором он разблокирован (извлекли или сменили токен — PIN забыт).
type session struct {
	cfg    Config
	prompt Prompter
	now    func() time.Time
	dk     []byte
	token  [32]byte
}

func serveNative(in io.Reader, out io.Writer, origin string, cfg Config) int {
	s := &session{cfg: cfg, prompt: systemPrompter(), now: time.Now}
	for {
		raw, err := readMsg(in)
		if err != nil {
			return 0 // браузер закрыл порт
		}
		if err := writeMsg(out, s.handle(raw, origin)); err != nil {
			return 1
		}
	}
}

func reply(rq procs.TokenAgentRequestV1, t procs.TokenAgentResponseV1Type) procs.TokenAgentResponseV1 {
	return procs.TokenAgentResponseV1{ProtocolVersion: ProtocolVersion, RequestID: rq.RequestID, Type: t}
}

func refusal(rq procs.TokenAgentRequestV1, err error) procs.TokenAgentResponseV1 {
	r := reply(rq, procs.TokenAgentResponseV1TypeError)
	msg := err.Error()
	r.Error = &procs.TokenAgentResponseV1Error{Code: agent.CodeOf(err), Message: &msg}
	return r
}

// sealed — хранилище на токене; нет — токен не вставлен. Смена содержимого
// (другой токен) сбрасывает PIN.
func (s *session) sealed() (agent.Sealed, error) {
	var sl agent.Sealed
	b, err := os.ReadFile(s.cfg.sealedPath())
	if err != nil {
		s.dk = nil
		return sl, &agent.Error{Code: agent.CodeTokenMissing, Message: "токен не вставлен (" + s.cfg.sealedPath() + ")"}
	}
	if h := sha256.Sum256(b); h != s.token {
		s.dk, s.token = nil, h
	}
	if err := json.Unmarshal(b, &sl); err != nil {
		return sl, &agent.Error{Code: agent.CodeTokenMissing, Message: "хранилище на токене не читается"}
	}
	return sl, nil
}

func (s *session) handle(raw []byte, origin string) procs.TokenAgentResponseV1 {
	var rq procs.TokenAgentRequestV1
	if err := json.Unmarshal(raw, &rq); err != nil {
		return refusal(rq, &agent.Error{Code: agent.CodeInvalid, Message: err.Error()})
	}
	if rq.ProtocolVersion != ProtocolVersion {
		return refusal(rq, &agent.Error{Code: agent.CodeInvalid, Message: fmt.Sprintf("протокол %d не поддерживается", rq.ProtocolVersion)})
	}
	switch rq.Type {
	case procs.TokenAgentRequestV1TypeHello:
		return s.hello(rq)
	case procs.TokenAgentRequestV1TypeStatus:
		return s.status(rq)
	case procs.TokenAgentRequestV1TypeSign:
		if rq.Sign == nil {
			return refusal(rq, &agent.Error{Code: agent.CodeInvalid, Message: "нет блока sign"})
		}
		return s.sign(rq, []procs.SignBlock{*rq.Sign})
	case procs.TokenAgentRequestV1TypeSignBatch:
		return s.sign(rq, rq.SignBatch)
	case procs.TokenAgentRequestV1TypeLocalJournal:
		return s.journal(rq)
	case procs.TokenAgentRequestV1TypeShiftReport:
		return s.shift(rq)
	}
	return refusal(rq, &agent.Error{Code: agent.CodeInvalid, Message: "неизвестный вид запроса " + string(rq.Type)})
}

func buildDigest() string {
	exe, err := os.Executable()
	if err != nil {
		return dom.Digest([]byte(agent.Build))
	}
	b, err := os.ReadFile(exe)
	if err != nil {
		return dom.Digest([]byte(agent.Build))
	}
	return dom.Digest(b)
}

func (s *session) hello(rq procs.TokenAgentRequestV1) procs.TokenAgentResponseV1 {
	r := reply(rq, procs.TokenAgentResponseV1TypeHello)
	h := &procs.TokenAgentResponseV1Hello{AgentVersion: agent.Version, BuildDigest: buildDigest(), DocFormatVersions: agent.DocFormatVersions,
		Keys: []procs.TokenAgentResponseV1HelloKeysElem{}}
	if sl, err := s.sealed(); err == nil {
		ks := procs.TokenAgentResponseV1HelloKeysElemKeyStorage(sl.KeyStorage)
		for _, k := range sl.Keys {
			h.Keys = append(h.Keys, procs.TokenAgentResponseV1HelloKeysElem{KeyRef: k.KeyRef, PersonID: sl.PersonID,
				Profile: procs.TokenAgentResponseV1HelloKeysElemProfile(k.Profile), KeyStorage: &ks})
		}
	}
	r.Hello = h
	return r
}

func (s *session) status(rq procs.TokenAgentRequestV1) procs.TokenAgentResponseV1 {
	r := reply(rq, procs.TokenAgentResponseV1TypeStatus)
	st := &procs.TokenAgentResponseV1Status{}
	if s.cfg.WorkplaceID != "" {
		wp := s.cfg.WorkplaceID
		st.WorkplaceID = &wp
	}
	if sl, err := s.sealed(); err == nil {
		st.TokenPresent, st.PinUnlocked = true, s.dk != nil
		p := sl.PersonID
		st.PersonID = &p
		ks := procs.TokenAgentResponseV1StatusKeyStorage(sl.KeyStorage)
		st.KeyStorage = &ks
		st.KeyRefs = sl.KeyRefs()
	}
	r.Status = st
	return r
}

// sign — подпись одного или пачки: сводку и отпечаток считает ядро, окно —
// эта программа; пачка — одно окно на всё (AD-13).
func (s *session) sign(rq procs.TokenAgentRequestV1, blocks []procs.SignBlock) procs.TokenAgentResponseV1 {
	sl, err := s.sealed()
	if err != nil {
		return refusal(rq, err)
	}
	if len(blocks) == 0 {
		return refusal(rq, &agent.Error{Code: agent.CodeInvalid, Message: "пустая пачка"})
	}
	ctx := agent.Context{Now: s.now().UTC(), WorkplaceID: s.cfg.WorkplaceID}
	preps := make([]agent.Prepared, 0, len(blocks))
	level := blocks[0].Level
	for _, b := range blocks {
		if len(blocks) > 1 && b.Level != dom.Level2 {
			return refusal(rq, &agent.Error{Code: agent.CodeLevel, Message: "пачка — только уровень 2"})
		}
		p, err := agent.Prepare(b, sl.PersonID, sl.Keys, ctx)
		if err != nil {
			return refusal(rq, err)
		}
		preps = append(preps, p)
	}
	journal := s.loadJournal()
	if level == dom.Level1 {
		if wait, ok := agent.RateCheck(journal, ctx.Now, agent.DefaultRate); !ok {
			return refusal(rq, &agent.Error{Code: agent.CodeRate, Message: fmt.Sprintf("подождите %d с", int(wait.Seconds())+1)})
		}
	}
	fields := preps[0].Summary
	title := "Подписать решение?"
	if len(preps) > 1 {
		items := make([]dom.BatchItem, 0, len(preps))
		for _, p := range preps {
			items = append(items, dom.BatchItem{Payload: p.Payload()})
		}
		sum, err := dom.Summarize(items)
		if err != nil {
			return refusal(rq, &agent.Error{Code: agent.CodeLevel, Message: err.Error()})
		}
		title = fmt.Sprintf("Подписать пачку: %d изделий, одно решение?", sum.Count)
		fields = append([]agent.Field{{Label: "Изделия", Value: strings.Join(sum.Items, ", ")}}, fields[:1]...)
	}
	// Уровень 2 — всегда окно; уровень 1 — окно только чтобы ввести PIN.
	if level == dom.Level2 || s.dk == nil {
		pin, ok, err := s.prompt.Confirm(title, fields, s.dk == nil)
		if err != nil {
			return refusal(rq, &agent.Error{Code: agent.CodeTokenMissing, Message: err.Error()})
		}
		if !ok {
			return refusal(rq, &agent.Error{Code: agent.CodeCancelled, Message: "подписант отказался в окне агента"})
		}
		if s.dk == nil {
			dk := agent.DeriveKey(sl, pin)
			if _, err := agent.Open(sl, dk); err != nil {
				return refusal(rq, err)
			}
			s.dk = dk
		}
	}
	bundle, err := agent.Open(sl, s.dk)
	if err != nil {
		s.dk = nil
		return refusal(rq, err)
	}
	r := reply(rq, procs.TokenAgentResponseV1TypeSigned)
	ks := procs.TokenAgentResponseV1SignedElemKeyStorage(sl.KeyStorage)
	for _, p := range preps {
		env, err := agent.Sign(p, bundle)
		if err != nil {
			return refusal(rq, &agent.Error{Code: agent.CodeTampered, Message: err.Error()})
		}
		var e agent.Entry
		journal, e = agent.Append(journal, p, env)
		var de crypto.DsseEnvelope
		_ = json.Unmarshal(env.Marshal(), &de)
		d := p.DocDigest
		sum := make([]procs.TokenAgentResponseV1SignedElemSummaryElem, 0, len(p.Summary))
		for _, f := range p.Summary {
			sum = append(sum, procs.TokenAgentResponseV1SignedElemSummaryElem{Label: f.Label, Value: f.Value})
		}
		r.Signed = append(r.Signed, procs.TokenAgentResponseV1SignedElem{Envelope: de, DocDigest: &d, Summary: sum,
			LocalJournalSeq: int(e.Seq), ClientSignedAt: p.ClientSignedAt, KeyStorage: &ks})
	}
	if err := s.saveJournal(journal); err != nil {
		return refusal(rq, &agent.Error{Code: agent.CodeInvalid, Message: "локальный журнал не записан: " + err.Error()})
	}
	return r
}

func (s *session) loadJournal() []agent.Entry {
	var j []agent.Entry
	if b, err := os.ReadFile(s.cfg.journalPath()); err == nil {
		_ = json.Unmarshal(b, &j)
	}
	return j
}

func (s *session) saveJournal(j []agent.Entry) error {
	if err := os.MkdirAll(s.cfg.StateDir, 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(j)
	tmp := s.cfg.journalPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.cfg.journalPath())
}

func (s *session) journal(rq procs.TokenAgentRequestV1) procs.TokenAgentResponseV1 {
	r := reply(rq, procs.TokenAgentResponseV1TypeLocalJournal)
	since := 0
	if rq.LocalJournal != nil {
		since = rq.LocalJournal.SinceSeq
	}
	r.LocalJournal = []procs.TokenAgentResponseV1LocalJournalElem{}
	for _, e := range s.loadJournal() {
		if e.Seq <= int64(since) {
			continue
		}
		et := e.EventType
		el := procs.TokenAgentResponseV1LocalJournalElem{Seq: int(e.Seq), EventType: &et, Digest: e.Digest, SignedAt: e.SignedAt}
		if e.ServerSeq > 0 {
			n := int(e.ServerSeq)
			el.ServerSeq = &n
		}
		r.LocalJournal = append(r.LocalJournal, el)
	}
	return r
}

func (s *session) shift(rq procs.TokenAgentRequestV1) procs.TokenAgentResponseV1 {
	sl, err := s.sealed()
	if err != nil {
		return refusal(rq, err)
	}
	if rq.ShiftReport == nil {
		return refusal(rq, &agent.Error{Code: agent.CodeInvalid, Message: "нет блока shift_report"})
	}
	base := dom.ShiftReport{FormatVersion: 1, CryptoProfile: dom.ProfileGost, Signers: sl.KeyRefs()[:1], PersonID: sl.PersonID,
		ShiftID: rq.ShiftReport.ShiftID, WorkplaceID: s.cfg.WorkplaceID, WindowFrom: rq.ShiftReport.WindowFrom, WindowTo: rq.ShiftReport.WindowTo}
	rep, err := agent.ShiftReport(s.loadJournal(), base)
	if err != nil {
		return refusal(rq, &agent.Error{Code: agent.CodeInvalid, Message: err.Error()})
	}
	var out procs.ShiftReportV1
	b, _ := json.Marshal(rep)
	if err := json.Unmarshal(b, &out); err != nil {
		return refusal(rq, err)
	}
	r := reply(rq, procs.TokenAgentResponseV1TypeShiftReport)
	r.ShiftReport = &out
	return r
}
