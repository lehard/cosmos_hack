package verify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	appjournal "ant/internal/application/journal"
	app "ant/internal/application/security"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/contracts/procs"
	dj "ant/internal/domain/journal"
	dom "ant/internal/domain/security"
)

type jcEntry = jc.JournalEntry

// rec — запись журнала в индексе верификатора (без содержимого).
type rec struct {
	chain       string
	seq         int64
	eventID     string
	eventType   string
	stream      string
	itemID      string
	sourceID    string
	sourceSeq   int64
	runID       string
	provenance  string
	basisSeq    int64
	commit      string
	link        string
	committedAt time.Time
	occurredAt  time.Time
	signed      bool
	sealed      bool // содержимое не расшифровать (нет KEK)
}

// caRec — запись журнала критических действий с содержимым.
type caRec struct {
	seq int64
	r   dom.Record
}

type run struct {
	in      Input
	checks  map[procs.VerifierReportV1ChecksElemCheck]*check
	order   []procs.VerifierReportV1ChecksElemCheck
	idx     map[string][]rec // цепочка → записи по порядку
	byID    map[string]rec   // event_id основной цепочки → запись
	ca      []caRec
	quar    map[string][]int64 // источник → source_seq из записей карантина
	losses  map[string][][2]int64
	classes procs.VerifierReportV1SignatureClasses
	virtual bool
}

func (v *run) lastSeq(chain string) int64 {
	es := v.idx[chain]
	if len(es) == 0 {
		return 0
	}
	return es[len(es)-1].seq
}

// readChains — цепочки и обязательства (AD-8, AD-44): звено каждой записи
// по формуле, номера подряд, commit сходится с содержимым.
func (v *run) readChains(ctx context.Context) error {
	c := v.checks["chains"]
	v.idx, v.byID = map[string][]rec{}, map[string]rec{}
	v.quar, v.losses = map[string][]int64{}, map[string][][2]int64{}
	sealed := 0
	for _, chain := range []string{"main", "ca"} {
		prev, prevSeq := dj.ZeroLink, int64(0)
		after := int64(0)
		for {
			page, err := v.in.Journal.Read(ctx, appjournal.ReadQuery{Chain: chain, AfterSeq: after, Limit: 1000})
			if err != nil {
				return err
			}
			for _, e := range page {
				c.checked++
				seq := int64(e.Seq)
				if seq != prevSeq+1 {
					c.reject("chain_link_mismatch.gap", fmt.Sprintf("цепочка %s: записей seq %d…%d нет — удалены в обход системы", chain, prevSeq+1, seq-1), chain, prevSeq+1, "")
				}
				if _, err := dj.VerifyLinks(prev, int(seq-1), []jc.JournalEntry{e}); err != nil && seq == prevSeq+1 {
					c.reject("chain_link_mismatch", fmt.Sprintf("цепочка %s, запись seq %d (%s%s): звено не сходится — запись изменена в обход системы",
						chain, seq, e.EventType, itemSuffix(e)), chain, seq, "")
				}
				if l, err := dj.ParseDigest(e.Link); err == nil {
					prev = l
				}
				prevSeq = seq
				r := v.index(e)
				salt, env, err := v.in.Journal.OpenUnverified(ctx, e)
				switch {
				case errors.Is(err, appjournal.ErrSealed) || errors.Is(err, dj.ErrSealed):
					r.sealed = true
					sealed++
				case err != nil:
					c.reject("chain_link_mismatch.sealed", fmt.Sprintf("цепочка %s, запись seq %d: содержимое не читается — %v", chain, seq, err), chain, seq, "")
				default:
					if err := dj.VerifyCommit(e, salt, env); err != nil {
						c.reject("chain_link_mismatch.commit", fmt.Sprintf("цепочка %s, запись seq %d (%s%s): обязательство commit не сходится с содержимым — содержимое изменено в обход системы",
							chain, seq, e.EventType, itemSuffix(e)), chain, seq, "")
					}
					r.signed = v.content(r, env)
				}
				v.idx[chain] = append(v.idx[chain], r)
				if chain == "main" {
					v.byID[r.eventID] = r
				}
			}
			if len(page) < 1000 {
				break
			}
			after = int64(page[len(page)-1].Seq)
		}
	}
	if sealed > 0 {
		c.unverifiable("chains.sealed", fmt.Sprintf("%d записей зашифрованы, KEK верификатору не смонтирован: commit не сверен", sealed))
	}
	return nil
}

func itemSuffix(e jc.JournalEntry) string {
	if e.ItemID != nil && *e.ItemID != "" {
		return ", изделие " + *e.ItemID
	}
	return ""
}

func (v *run) index(e jc.JournalEntry) rec {
	r := rec{chain: string(e.Chain), seq: int64(e.Seq), eventID: e.EventID, eventType: e.EventType, stream: e.Stream,
		sourceID: e.SourceID, provenance: string(e.ProvenanceClass), commit: e.Commit, link: e.Link}
	if e.ItemID != nil {
		r.itemID = *e.ItemID
	}
	if e.SourceSeq != nil {
		r.sourceSeq = int64(*e.SourceSeq)
	}
	if e.RunID != nil {
		r.runID = *e.RunID
		v.virtual = true
	}
	if e.BasisSeq != nil {
		r.basisSeq = int64(*e.BasisSeq)
	}
	r.committedAt, _ = dj.ParseTime(e.CommittedAt)
	r.occurredAt, _ = dj.ParseTime(e.OccurredAt)
	return r
}

// content — то, что верификатору нужно из содержимого: запись CA, номера
// источника в карантине, объявленные потери. Возвращает, есть ли подпись.
func (v *run) content(r rec, env []byte) (signed bool) {
	ev, d, err := app.ParseEnvelope(env)
	if err != nil {
		return false
	}
	for _, s := range d.Signatures {
		if s.Sig != "" {
			signed = true
		}
	}
	if r.chain == "ca" {
		var cr dom.Record
		if json.Unmarshal(ev.Data, &cr) == nil {
			v.ca = append(v.ca, caRec{seq: r.seq, r: cr})
		}
		return signed
	}
	switch catalog.Type(r.eventType) {
	case catalog.IngestMessageQuarantined:
		var q struct {
			SourceID  string `json:"source_id"`
			SourceSeq int64  `json:"source_seq"`
		}
		if json.Unmarshal(ev.Data, &q) == nil && q.SourceSeq > 0 {
			v.quar[q.SourceID] = append(v.quar[q.SourceID], q.SourceSeq)
		}
	case catalog.IngestSourceLossSuspected:
		var l struct {
			SourceID string `json:"source_id"`
			From     int64  `json:"missing_from_seq"`
			To       int64  `json:"missing_to_seq"`
		}
		if json.Unmarshal(ev.Data, &l) == nil {
			v.losses[l.SourceID] = append(v.losses[l.SourceID], [2]int64{l.From, l.To})
		}
	}
	return signed
}

// criticalActions — журнал критических действий один к одному с решениями
// (AD-28): у каждой записи критического типа ровно одна запись CA; CA
// ссылается на существующую запись и её commit. Расхождение — «критическое
// действие CA-… изменено вне разрешённого процесса».
func (v *run) criticalActions() {
	c := v.checks["chains"]
	byMain := map[string][]caRec{}
	for _, a := range v.ca {
		byMain[a.r.MainEventID] = append(byMain[a.r.MainEventID], a)
		ref := dom.Ref(a.seq)
		m, ok := v.byID[a.r.MainEventID]
		switch {
		case !ok:
			c.reject("chain_link_mismatch.ca_orphan", fmt.Sprintf("критическое действие %s ссылается на запись %s, которой нет в журнале — изменено вне разрешённого процесса", ref, a.r.MainEventID), "ca", a.seq, ref)
		case m.commit != a.r.MainCommit:
			c.reject("chain_link_mismatch.ca_commit", fmt.Sprintf("критическое действие %s изменено вне разрешённого процесса: основная запись seq %d (%s) не совпадает с обязательством, записанным в CA", ref, m.seq, m.eventType), "main", m.seq, ref)
		case a.r.CANo != a.seq:
			c.reject("chain_link_mismatch.ca_no", fmt.Sprintf("критическое действие %s: номер в записи %d не совпадает с позицией в цепочке", ref, a.r.CANo), "ca", a.seq, ref)
		}
	}
	for _, r := range v.idx["main"] {
		if !dom.Critical(catalog.Type(r.eventType)) {
			continue
		}
		switch n := len(byMain[r.eventID]); {
		case n == 0 && r.committedAt.After(caSince(v.ca, v.idx["ca"])):
			c.reject("chain_link_mismatch.ca_missing", fmt.Sprintf("запись seq %d (%s): критическое действие без записи в журнале критических действий", r.seq, r.eventType), "main", r.seq, "")
		case n > 1:
			c.reject("chain_link_mismatch.ca_duplicate", fmt.Sprintf("запись seq %d (%s): %d записей CA вместо одной", r.seq, r.eventType, n), "main", r.seq, "")
		}
	}
}

// caSince — начало ведения журнала CA (первая запись цепочки ca): решения
// до подключения сервиса доверенных решений (демо-трек до эпика 29) записей
// CA не имеют — это не нарушение.
func caSince(_ []caRec, ca []rec) time.Time {
	if len(ca) == 0 {
		return time.Unix(1<<40, 0)
	}
	return ca[0].committedAt.Add(-time.Millisecond)
}

// checkpoints — совпадение с контрольными точками хранителя (AD-8):
// подпись hybrid, цепочка отпечатков, звено головы в журнале, звенья,
// переданные хранителю, задержка передачи.
func (v *run) checkpoints() {
	c := v.checks["checkpoints"]
	if v.in.CheckpointsErr != nil {
		c.unverifiable("checkpoints.keeper_unavailable", "хранитель недоступен: "+v.in.CheckpointsErr.Error())
		return
	}
	if len(v.in.Checkpoints) == 0 {
		c.unverifiable("checkpoints.none", "контрольных точек у хранителя ещё нет")
		return
	}
	link := func(chain string, seq int64) (string, bool) {
		es := v.idx[chain]
		i, ok := slices.BinarySearchFunc(es, seq, func(r rec, s int64) int { return int(r.seq - s) })
		if !ok {
			return "", false
		}
		return es[i].link, true
	}
	prevDigest := ""
	for i, cp := range v.in.Checkpoints {
		c.checked++
		p := cp.Payload
		no := p.CheckpointNo
		if cp.SigErr != nil {
			c.reject("checkpoint_mismatch.signature", fmt.Sprintf("контрольная точка №%d: подпись хранителя не проверяется по trust-anchors — %v", no, cp.SigErr), "", 0, "")
			continue
		}
		if i > 0 && (p.PreviousCheckpointDigest == nil || *p.PreviousCheckpointDigest != prevDigest) {
			c.reject("checkpoint_mismatch.previous", fmt.Sprintf("контрольная точка №%d не продолжает точку №%d — последовательность точек нарушена", no, no-1), "", 0, "")
		}
		prevDigest = cp.Digest
		for _, h := range p.Heads {
			if h.Seq == 0 {
				continue
			}
			l, ok := link(string(h.Chain), int64(h.Seq))
			switch {
			case !ok:
				c.reject("checkpoint_mismatch.rollback", fmt.Sprintf("контрольная точка №%d (время хранителя %s): записи seq %d цепочки %s в журнале нет — записи удалены (откат)", no, p.KeeperTime, h.Seq, h.Chain), string(h.Chain), int64(h.Seq), "")
			case l != h.Link:
				c.reject("checkpoint_mismatch", fmt.Sprintf("голова цепочки %s seq %d не совпадает с контрольной точкой №%d хранителя (время хранителя %s) — цепочку переписали после точки",
					h.Chain, h.Seq, no, p.KeeperTime), string(h.Chain), int64(h.Seq), "")
			}
		}
	}
	// Звенья, принятые хранителем: место переписанной записи — точно.
	for _, chain := range []string{"main", "ca"} {
		kl := v.in.KeeperLinks[chain]
		first := true
		for _, r := range v.idx[chain] {
			k, ok := kl[r.seq]
			if !ok || k == r.link {
				continue
			}
			if first {
				m, _ := v.byID[r.eventID]
				c.reject("checkpoint_mismatch.link", fmt.Sprintf("запись seq %d цепочки %s (%s%s) не совпадает со звеном, переданным хранителю, — изменена и цепочка пересчитана в обход системы (дальше расходятся все звенья)",
					r.seq, chain, r.eventType, itemOf(m)), chain, r.seq, "")
				first = false
			}
		}
	}
	// Задержка передачи хранителю: запись покрыта первой точкой, у которой
	// голова её цепочки не меньше её seq; дольше MaxGap — сервер молчал.
	if v.in.MaxGap > 0 {
		for _, chain := range []string{"main", "ca"} {
			j := 0
			cps := v.in.Checkpoints
			late := 0
			for _, r := range v.idx[chain] {
				for j < len(cps) {
					if s, _ := cps[j].Head(chain); s >= r.seq {
						break
					}
					j++
				}
				if j == len(cps) {
					break
				}
				kt, err := time.Parse(time.RFC3339Nano, cps[j].Payload.KeeperTime)
				if err == nil && kt.Sub(r.committedAt) > v.in.MaxGap {
					if late == 0 {
						c.reject("checkpoint_mismatch.gap", fmt.Sprintf("запись seq %d цепочки %s передана хранителю через %s после фиксации (порог %s) — сервер молчал перед хранителем",
							r.seq, chain, kt.Sub(r.committedAt).Round(time.Second), v.in.MaxGap), chain, r.seq, "")
					}
					late++
				}
			}
		}
	}
	last := v.in.Checkpoints[len(v.in.Checkpoints)-1]
	for _, chain := range []string{"main", "ca"} {
		s, _ := last.Head(chain)
		if top := v.lastSeq(chain); top > s {
			c.add("intact", "checkpoints.tail", fmt.Sprintf("цепочка %s: записи seq %d…%d ещё не покрыты контрольной точкой (последняя №%d)", chain, s+1, top, last.Payload.CheckpointNo), chain, s+1, "")
		}
	}
}

func itemOf(r rec) string {
	if r.itemID != "" {
		return ", изделие " + r.itemID
	}
	return ""
}

// signatures — подписи и их классы (AD-9, AD-10).
func (v *run) signatures() {
	c := v.checks["signatures"]
	unsigned, sig := 0, 0
	for _, chain := range []string{"main", "ca"} {
		for _, r := range v.idx[chain] {
			c.checked++
			switch r.provenance {
			case "device":
				v.classes.Device++
			case "personal":
				v.classes.Personal++
			case "paper":
				v.classes.Paper++
			case "partner":
				v.classes.Partner++
			case "scenario":
				v.classes.Scenario++
			case "genesis":
				v.classes.Genesis++
			default:
				v.classes.ServerAttested++
			}
			if r.signed {
				sig++
			} else {
				unsigned++
			}
		}
	}
	if unsigned > 0 {
		c.unverifiable("signatures.unsigned", fmt.Sprintf("%d записей без подписи (демо без агента токена и ключей источников, Д-28, Д-30) — «подпись не проверялась»", unsigned))
	}
	if sig > 0 {
		c.unverifiable("signatures.registry", fmt.Sprintf("%d подписанных записей: реестра ключей и профилей на момент подписи в журнале ещё нет (эпики 05, 27) — «ключ недоступен», не «валидно»", sig))
	}
}

// sourceSeq — непрерывность source_seq (AD-9, AD-7): каждый номер источника
// есть в журнале или в записи о карантине.
func (v *run) sourceSeq() {
	c := v.checks["source_seq"]
	seqs := map[string][]int64{}
	for _, r := range v.idx["main"] {
		if r.sourceSeq > 0 {
			seqs[r.sourceID] = append(seqs[r.sourceID], r.sourceSeq)
		}
	}
	for src, q := range v.quar {
		seqs[src] = append(seqs[src], q...)
	}
	srcs := make([]string, 0, len(seqs))
	for s := range seqs {
		srcs = append(srcs, s)
	}
	for _, src := range uniq(srcs) {
		c.checked++
		ns := seqs[src]
		slices.Sort(ns)
		ns = slices.Compact(ns)
		for i := 1; i < len(ns); i++ {
			if ns[i] == ns[i-1]+1 {
				continue
			}
			from, to := ns[i-1]+1, ns[i]-1
			declared := slices.ContainsFunc(v.losses[src], func(l [2]int64) bool { return l[0] <= from && l[1] >= to })
			if declared {
				c.add("not_verifiable", "source_seq_gap.declared", fmt.Sprintf("источник %s: номера %d…%d объявлены потерянными (потеря данных источника)", src, from, to), "", 0, "")
				continue
			}
			// Объявление потери — реакция планировщика ingest.source.loss_suspected
			// (роль scheduler, эпик 24); пока её нет, разрыв без объявления —
			// оговорка «не проверяемо», а не нарушение: номер мог не дойти от
			// устройства. Удаление записи из журнала ловит проверка цепочек.
			// TODO(24): после планировщика — «отвергнуто».
			c.add("not_verifiable", "source_seq_gap", fmt.Sprintf("источник %s: номеров %d…%d нет ни в журнале, ни в записи о карантине, потеря не объявлена", src, from, to), "", 0, "")
		}
	}
}

// lateWrite — задержка записи (AD-9): факт устройства зафиксирован позже
// порога после события. В прогоне сценария время виртуальное — не проверяется.
func (v *run) lateWrite() {
	c := v.checks["late_write"]
	if v.in.LateWrite <= 0 {
		c.unverifiable("late_write.off", "порог задержки записи не задан")
		return
	}
	for _, r := range v.idx["main"] {
		if r.provenance != "device" || r.runID != "" {
			continue
		}
		c.checked++
		if d := r.committedAt.Sub(r.occurredAt); d > v.in.LateWrite {
			c.add("intact", "late_write.flagged", fmt.Sprintf("факт seq %d источника %s зафиксирован через %s после события (порог %s) — флаг «задержка записи»",
				r.seq, r.sourceID, d.Round(time.Second), v.in.LateWrite), "main", r.seq, "")
		}
	}
	if v.virtual {
		c.add("intact", "late_write.virtual_time", "факты прогонов сценария — виртуальное время: задержка записи для них не оценивается", "", 0, "")
	}
}

// pending — проверки, для которых в системе ещё нет данных: «не проверяемо»
// с причиной (общий вердикт — «цело с оговорками»).
func (v *run) pending() {
	for n, why := range map[procs.VerifierReportV1ChecksElemCheck]string{
		"genesis":        "блока генезиса в журнале нет — эпик 05 (ant init); доверие ключам хранителя и верификатора — из trust-anchors",
		"signing_moment": "момент подписи (seen_checkpoint) — вместе с реестром ключей, эпик 27",
		"authority":      "права, клеймо и сеанс подписанта на момент — политика из журнала, эпики 08, 26",
		"bpmn_quorum":    "кворум подписей версии процесса — записи подписей версии, эпики 17, 27",
		"rendering":      "повторная отрисовка документов (rendering_hash) и QR бумажных подписей — эпик 28",
		"build":          "перечень допустимых сборок домена в нормативном слое ещё не ведётся (domain_build = H(версия бинарника))",
	} {
		v.checks[n].unverifiable(string(n)+".pending", why)
	}
}
