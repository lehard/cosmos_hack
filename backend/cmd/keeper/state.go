package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"time"

	app "ant/internal/application/security"
	cc "ant/internal/contracts/crypto"
	dj "ant/internal/domain/journal"
	"ant/internal/infrastructure/security/hybrid"
)

// Store — состояние хранителя в его томе (AD-8): контрольные точки,
// отчёты верификатора, тревоги — файлы только на дописывание; в памяти —
// последняя точка и время последней передачи голов.
type Store struct {
	dir    string
	signer *hybrid.Signer
	// anchors — trust-anchors: ключ верификатора для проверки отчётов (AD-46).
	anchors hybrid.Anchors
	// interval — N: головы приходят раз в N секунд; тревога — после 2N молчания.
	interval time.Duration
	now      func() time.Time

	mu          sync.Mutex
	checkpoints []app.Checkpoint
	reports     []app.Report
	alarms      []app.KeeperAlarm
	// links — принятые звенья по цепочкам (seq по возрастанию): верификатор
	// по ним находит точное место переписанной записи.
	links       map[string][]app.Link
	lastSeen    time.Time
	silentFired bool
}

// Отказы приёма голов (409, AD-8).
var (
	ErrFork     = errors.New("fork_attempt")
	ErrRollback = errors.New("rollback_attempt")
	ErrInvalid  = errors.New("invalid")
)

// Reject — отказ приёма голов с видом и подробностями.
type Reject struct {
	Kind   error
	Chain  string
	Seq    int64
	Detail string
}

func (r *Reject) Error() string { return r.Kind.Error() + ": " + r.Detail }
func (r *Reject) Unwrap() error { return r.Kind }

var reDigest = regexp.MustCompile(`^streebog256:[0-9a-f]{64}$`)

// OpenStore читает состояние хранителя из dir.
func OpenStore(dir string, signer *hybrid.Signer, anchors hybrid.Anchors, interval time.Duration) (*Store, error) {
	s := &Store{dir: dir, signer: signer, anchors: anchors, interval: interval, now: time.Now, links: map[string][]app.Link{}}
	if err := readLines(filepath.Join(dir, "links.jsonl"), func(b []byte) error {
		var l app.Link
		err := json.Unmarshal(b, &l)
		if err == nil {
			s.links[l.Chain] = append(s.links[l.Chain], l)
		}
		return err
	}); err != nil {
		return nil, err
	}
	if err := readLines(filepath.Join(dir, "checkpoints.jsonl"), func(b []byte) error {
		c, err := app.ParseCheckpoint(b)
		if err == nil {
			s.checkpoints = append(s.checkpoints, c)
		}
		return err
	}); err != nil {
		return nil, err
	}
	if err := readLines(filepath.Join(dir, "reports.jsonl"), func(b []byte) error {
		r, err := app.ParseReport(b)
		if err == nil {
			s.reports = append(s.reports, r)
		}
		return err
	}); err != nil {
		return nil, err
	}
	if err := readLines(filepath.Join(dir, "alarms.jsonl"), func(b []byte) error {
		var a app.KeeperAlarm
		err := json.Unmarshal(b, &a)
		if err == nil {
			s.alarms = append(s.alarms, a)
		}
		return err
	}); err != nil {
		return nil, err
	}
	s.lastSeen = s.now()
	return s, nil
}

func readLines(path string, fn func([]byte) error) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		if err := fn(slices.Clone(sc.Bytes())); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	return sc.Err()
}

func (s *Store) appendLine(name string, b []byte) error {
	f, err := os.OpenFile(filepath.Join(s.dir, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(slices.Clone(b), '\n')); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func (s *Store) last() (app.Checkpoint, bool) {
	if len(s.checkpoints) == 0 {
		return app.Checkpoint{}, false
	}
	return s.checkpoints[len(s.checkpoints)-1], true
}

// alarm — тревога хранителя (в его том и в состояние для GET /v1/status).
// Повтор той же тревоги подряд (сервер снова шлёт переписанную цепочку) —
// одна запись.
func (s *Store) alarm(kind, chain string, seq int64, detail string) {
	if n := len(s.alarms); n > 0 {
		if l := s.alarms[n-1]; l.Alert == kind && l.Chain == chain && l.Seq == seq && kind != "heads_silent" {
			return
		}
	}
	a := app.KeeperAlarm{No: int64(len(s.alarms)) + 1, Alert: kind, Chain: chain, Seq: seq, Detail: detail,
		At: s.now().UTC().Format(time.RFC3339Nano)}
	s.alarms = append(s.alarms, a)
	b, _ := json.Marshal(a)
	_ = s.appendLine("alarms.jsonl", b)
}

// Submit — приём голов со звеньями (AD-8): голова принимается, только если
// звенья от прежней принятой головы сходятся (цепочка продолжена, а не
// переписана); seq меньше принятого — откат; тот же seq с другим звеном —
// вилка. Отказ — тревога. Продвижение любой цепочки или истёкший интервал —
// новая контрольная точка hybrid с головами обеих цепочек.
func (s *Store) Submit(sub app.HeadsSubmission) (app.Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	s.lastSeen, s.silentFired = now, false
	last, hasLast := s.last()
	heads := map[string]app.Head{}
	accepted := map[string][]app.Link{}
	advanced := false
	for _, ch := range app.Chains {
		prevSeq, prevLink := last.Head(ch)
		head := app.Head{Chain: ch, Seq: prevSeq, Link: prevLink}
		if i := slices.IndexFunc(sub.Heads, func(h app.Head) bool { return h.Chain == ch }); i >= 0 {
			head = sub.Heads[i]
		}
		var links []app.Link
		for _, l := range sub.Links {
			if l.Chain == ch {
				links = append(links, l)
			}
		}
		slices.SortFunc(links, func(a, b app.Link) int { return int(a.Seq - b.Seq) })
		if err := s.check(ch, prevSeq, prevLink, head, links); err != nil {
			var r *Reject
			if errors.As(err, &r) {
				s.alarm(r.Kind.Error(), ch, prevSeq, r.Detail)
			}
			return app.Checkpoint{}, err
		}
		advanced = advanced || head.Seq > prevSeq
		heads[ch] = head
		if head.Seq > prevSeq {
			accepted[ch] = links
		}
	}
	for _, ch := range app.Chains {
		for _, l := range accepted[ch] {
			b, _ := json.Marshal(l)
			if err := s.appendLine("links.jsonl", b); err != nil {
				return app.Checkpoint{}, err
			}
			s.links[ch] = append(s.links[ch], l)
		}
	}
	if hasLast && !advanced {
		if kt, err := time.Parse(time.RFC3339Nano, last.Payload.KeeperTime); err == nil && now.Sub(kt) < s.interval {
			return last, nil
		}
	}
	return s.checkpoint(sub, heads, last, hasLast, now)
}

func (s *Store) check(ch string, prevSeq int64, prevLink string, head app.Head, links []app.Link) error {
	if !reDigest.MatchString(head.Link) {
		return &Reject{Kind: ErrInvalid, Chain: ch, Detail: "звено головы не streebog256"}
	}
	switch {
	case head.Seq < prevSeq:
		return &Reject{Kind: ErrRollback, Chain: ch, Seq: head.Seq,
			Detail: fmt.Sprintf("цепочка %s: голова seq %d меньше принятой %d — откат", ch, head.Seq, prevSeq)}
	case head.Seq == prevSeq:
		if head.Link != prevLink {
			return &Reject{Kind: ErrFork, Chain: ch, Seq: head.Seq,
				Detail: fmt.Sprintf("цепочка %s: тот же seq %d с другим звеном — цепочку переписали", ch, head.Seq)}
		}
		return nil
	}
	prev, err := dj.ParseDigest(prevLink)
	if err != nil {
		return err
	}
	seq := prevSeq
	for _, l := range links {
		if l.Seq != seq+1 {
			return &Reject{Kind: ErrFork, Chain: ch, Seq: l.Seq, Detail: fmt.Sprintf("цепочка %s: звенья идут не подряд — после %d пришёл %d", ch, seq, l.Seq)}
		}
		commit, err1 := dj.ParseDigest(l.Commit)
		oh, err2 := dj.ParseDigest(l.OpenFieldsHash)
		link, err3 := dj.ParseDigest(l.Link)
		if err := errors.Join(err1, err2, err3); err != nil {
			return &Reject{Kind: ErrInvalid, Chain: ch, Seq: l.Seq, Detail: err.Error()}
		}
		if dj.Link(prev, commit, oh) != link {
			return &Reject{Kind: ErrFork, Chain: ch, Seq: l.Seq,
				Detail: fmt.Sprintf("цепочка %s: звено seq %d не продолжает принятую голову seq %d — цепочку переписали", ch, l.Seq, prevSeq)}
		}
		prev, seq = link, l.Seq
	}
	if seq != head.Seq || prev.String() != head.Link {
		return &Reject{Kind: ErrFork, Chain: ch, Seq: head.Seq, Detail: fmt.Sprintf("цепочка %s: звенья не доходят до головы seq %d", ch, head.Seq)}
	}
	return nil
}

func (s *Store) checkpoint(sub app.HeadsSubmission, heads map[string]app.Head, last app.Checkpoint, hasLast bool, now time.Time) (app.Checkpoint, error) {
	p := cc.KeeperCheckpoint{FormatVersion: 1, CryptoProfile: cc.KeeperCheckpointCryptoProfileHybrid, Signers: s.signer.KeyIDs(),
		CheckpointNo: 1, CommittedAtFrom: sub.CommittedAtFrom, CommittedAtTo: sub.CommittedAtTo, KeeperTime: dj.FormatTime(now)}
	for _, ch := range app.Chains {
		h := heads[ch]
		p.Heads = append(p.Heads, cc.KeeperCheckpointHeadsElem{Chain: cc.KeeperCheckpointHeadsElemChain(ch), Seq: int(h.Seq), Link: h.Link})
	}
	if hasLast {
		p.CheckpointNo = last.Payload.CheckpointNo + 1
		d := last.Digest
		p.PreviousCheckpointDigest = &d
	} else {
		// Первая точка: отпечаток генезиса (AD-33). До эпика 05 генезиса нет —
		// берётся звено первой записи основной цепочки, если она передана.
		for _, l := range sub.Links {
			if l.Chain == "main" && l.Seq == 1 {
				g := l.Link
				p.GenesisDigest = &g
			}
		}
	}
	payload, err := json.Marshal(p)
	if err != nil {
		return app.Checkpoint{}, err
	}
	payload, err = dj.Canonical(payload)
	if err != nil {
		return app.Checkpoint{}, err
	}
	env, err := s.signer.Sign("checkpoint", payload)
	if err != nil {
		return app.Checkpoint{}, err
	}
	cp, err := app.ParseCheckpoint(env)
	if err != nil {
		return app.Checkpoint{}, err
	}
	if err := s.appendLine("checkpoints.jsonl", env); err != nil {
		return app.Checkpoint{}, err
	}
	s.checkpoints = append(s.checkpoints, cp)
	return cp, nil
}

// Watch — тревога «головы не приходили дольше 2N секунд» (AD-8): хранитель
// поднимает её сам, не полагаясь на сервер.
func (s *Store) Watch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.silentFired || s.now().Sub(s.lastSeen) <= 2*s.interval {
		return
	}
	s.silentFired = true
	seq, _ := app.Checkpoint{}.Head("main")
	if l, ok := s.last(); ok {
		seq, _ = l.Head("main")
	}
	s.alarm("heads_silent", "main", seq, fmt.Sprintf("головы цепочек не приходили %s (порог 2N = %s); последняя передача %s",
		s.now().Sub(s.lastSeen).Round(time.Second), 2*s.interval, s.lastSeen.UTC().Format(time.RFC3339)))
}

// SubmitReport — отчёт верификатора (AD-46): принимается, только если
// подпись hybrid проверяется ключом верификатора из trust-anchors.
func (s *Store) SubmitReport(env []byte) (app.Report, error) {
	if _, err := s.anchors.Verify(env, "verifier-report", "verifier"); err != nil {
		return app.Report{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	r, err := app.ParseReport(env)
	if err != nil {
		return app.Report{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.ContainsFunc(s.reports, func(x app.Report) bool { return x.Digest == r.Digest }) {
		return r, nil
	}
	if err := s.appendLine("reports.jsonl", env); err != nil {
		return app.Report{}, err
	}
	s.reports = append(s.reports, r)
	return r, nil
}

// Links — принятые звенья цепочки после afterSeq.
func (s *Store) Links(chain string, afterSeq int64, limit int) []app.Link {
	s.mu.Lock()
	defer s.mu.Unlock()
	ls := s.links[chain]
	i, _ := slices.BinarySearchFunc(ls, afterSeq+1, func(l app.Link, seq int64) int { return int(l.Seq - seq) })
	out := slices.Clone(ls[i:min(len(ls), i+limit)])
	if out == nil {
		out = []app.Link{}
	}
	return out
}

// Status — состояние для GET /v1/status.
func (s *Store) Status() app.KeeperStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return app.KeeperStatus{LastSubmissionAt: s.lastSeen.UTC().Format(time.RFC3339Nano), IntervalSeconds: int(s.interval / time.Second),
		Checkpoints: int64(len(s.checkpoints)), Alarms: slices.Clone(s.alarms)}
}
