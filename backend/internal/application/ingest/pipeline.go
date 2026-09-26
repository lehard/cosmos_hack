package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"ant/internal/application/journal"
	"ant/internal/application/materials"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/crossitem"
	dom "ant/internal/domain/ingest"
	"ant/internal/domain/kernel"
)

// header — поля конверта, нужные конвейеру; читаются снисходительно (до
// проверки схемой тип поля может быть неверным — тогда поле пустое).
type header struct {
	EventID       string  `json:"event_id"`
	EventType     string  `json:"event_type"`
	SchemaVersion int     `json:"schema_version"`
	SourceID      string  `json:"source_id"`
	SourceSeq     int64   `json:"source_seq"`
	SourceKind    string  `json:"source_kind"`
	Reliability   string  `json:"reliability"`
	OccurredAt    string  `json:"occurred_at"`
	CorrelationID string  `json:"correlation_id"`
	CausationID   *string `json:"causation_id"`
	RunID         string  `json:"run_id"`
	ItemID        string  `json:"item_id"`
	ItemRef       *struct {
		CarrierType         string `json:"carrier_type"`
		Value               string `json:"value"`
		IdentificationLevel string `json:"identification_level"`
	} `json:"item_ref"`
	Integrity struct {
		Signers []string `json:"signers"`
	} `json:"integrity"`
	Data json.RawMessage `json:"data"`
}

func parseHeader(canon []byte) header {
	var h header
	_ = json.Unmarshal(canon, &h) // ошибки типов: поле остаётся пустым, схема скажет точнее
	return h
}

// msgCtx — контекст сообщения в пачке.
type msgCtx struct {
	// SentAt — время отправки пачки по часам источника (для сдвига часов, FR-33).
	SentAt *time.Time
	// Manual — ручной ввод под сеансом без подписи (FR-141): подпись не требуется.
	Manual *manualAuth
}

type manualAuth struct {
	Provenance string
	Note       string
}

// Ingest — приём одного сообщения (FR-26…FR-34).
func (s *Service) Ingest(ctx context.Context, raw []byte) (Result, error) {
	if err := s.ready(); err != nil {
		return Result{}, err
	}
	return s.process(ctx, raw, msgCtx{})
}

// batch — пачка edge-агента (FR-39).
type batch struct {
	SourceID string            `json:"source_id"`
	SentAt   string            `json:"sent_at"`
	Messages []json.RawMessage `json:"messages"`
}

// IngestBatch — приём пачки (FR-39): сообщения обрабатываются по одному в
// порядке пачки; ответ — по элементу на сообщение. Досылка после
// недоступности — такие же пачки с исходным occurred_at.
func (s *Service) IngestBatch(ctx context.Context, raw []byte) (BatchResult, error) {
	if err := s.ready(); err != nil {
		return BatchResult{}, err
	}
	var b batch
	if err := json.Unmarshal(raw, &b); err != nil {
		e := platform.Fail(errcodes.ApiValidationFailed, "field", "/", "reason", err.Error())
		e.Detail = "пачка не разобрана: " + err.Error()
		return BatchResult{}, e
	}
	if len(b.Messages) > s.cfg.MaxBatch {
		return BatchResult{}, platform.Fail(errcodes.IngestBatchTooLarge, "count", fmt.Sprint(len(b.Messages)), "limit", fmt.Sprint(s.cfg.MaxBatch))
	}
	mc := msgCtx{}
	if t, err := parseTS(b.SentAt); err == nil {
		mc.SentAt = &t
	}
	out := BatchResult{SourceID: b.SourceID, Results: make([]Result, 0, len(b.Messages))}
	for _, m := range b.Messages {
		r, err := s.process(ctx, m, mc)
		if err != nil {
			return out, err
		}
		out.Results = append(out.Results, r)
	}
	return out, nil
}

// process — конвейер приёма одного сообщения: подпись → схема → пять случаев →
// идемпотентность → source_seq → карантин → флаги → привязка → журнал.
// Ошибка возвращается только при сбое инфраструктуры (журнал, хранилище);
// отказы по содержимому — Result с кодом.
func (s *Service) process(ctx context.Context, raw []byte, mc msgCtx) (Result, error) {
	start := s.deps.InfraClock.Now()
	received, err := s.deps.DomainClock.Now(ctx)
	if err != nil {
		return Result{}, err
	}
	received = received.UTC().Truncate(time.Millisecond)
	q := quarantineIn{raw: raw, received: received}

	// 1. Конверт DSSE и канонический вид (AD-10).
	f, err := parseFrame(raw)
	if err != nil {
		q.dec = dom.Decision{Code: errcodes.IngestCanonicalFormViolation, Detail: "сообщение не разобрано: " + err.Error()}
		return s.finishQ(start)(s.quarantine(ctx, q))
	}
	if f.DSSE && f.PayloadType != PayloadTypeEvent {
		q.dec = dom.Decision{Code: errcodes.IngestSchemaViolation, Field: "/payloadType", Value: f.PayloadType,
			Detail: "класс пакета «" + f.PayloadType + "» не принимается приёмом событий"}
		return s.finishQ(start)(s.quarantine(ctx, q))
	}
	canon, err := dom.Canonicalize(f.Payload)
	if err != nil {
		q.dec = dom.Decision{Code: errcodes.IngestCanonicalFormViolation, Detail: err.Error()}
		return s.finishQ(start)(s.quarantine(ctx, q))
	}
	h := parseHeader(canon)
	q.h = h
	q.keyRef = first(f.KeyRefs)

	// 2. Подпись источника (FR-26).
	var auth authResult
	if mc.Manual != nil {
		auth = authResult{Provenance: mc.Manual.Provenance, Note: mc.Manual.Note}
	} else {
		auth = s.authenticate(ctx, f, h)
	}
	if auth.Fail != nil {
		q.dec = dom.Decision{Code: auth.Fail.Code, Detail: auth.Fail.Detail}
		q.sig = &SignatureFailure{SourceID: h.SourceID, EventID: h.EventID, KeyRef: auth.Fail.KeyRef, Failure: auth.Fail.Failure, OccurredAt: received}
		return s.finishQ(start)(s.quarantine(ctx, q))
	}
	q.authed = true

	// 3. Схема теми же JSON Schema и пять случаев FR-29 (кейс §4.7, AD-20).
	dec, err := s.checkContract(canon, h)
	if err != nil {
		return Result{}, err
	}
	if dec.Outcome == dom.OutcomeQuarantined {
		q.dec = dec
		return s.finishQ(start)(s.quarantine(ctx, q))
	}
	occurred, _ := parseTS(h.OccurredAt)

	// 4. Идемпотентность (FR-31, AD-7): ключ source_id + event_id, содержимое — отпечаток payload.
	unlock := s.lock(h.SourceID)
	defer unlock()
	fp, err := dom.ContentFingerprint(canon)
	if err != nil {
		q.dec = dom.Decision{Code: errcodes.IngestCanonicalFormViolation, Detail: err.Error()}
		return s.finishQ(start)(s.quarantineLocked(ctx, q))
	}
	prior, err := s.deps.Registry.Seen(ctx, h.SourceID, h.EventID)
	if err != nil {
		return Result{}, err
	}
	switch dom.CheckRepeat(prior, fp) {
	case dom.RepeatDuplicate:
		// FR-31, кейс §4.5: повтор подтверждается прежним ответом и не учитывается повторно.
		r := Result{Outcome: OutcomeDuplicate, Status: 200, SourceID: h.SourceID, EventID: h.EventID, EventType: h.EventType,
			SourceSeq: prior.SourceSeq, Seq: prior.Seq, SignatureVerified: prior.SignatureVerified, Replayed: true,
			Detail: "повтор: уже принято, повторно не учитывается"}
		if !prior.SignatureVerified {
			r.SignatureNote = SignatureNotVerified
		}
		return s.finish(start, 0, r, nil)
	case dom.RepeatConflict:
		// FR-31, AD-7: тот же ключ, другое содержимое — конфликт целостности:
		// карантин (исходное не перезаписывается), CA и событие security.
		q.dec = dom.Decision{Code: errcodes.IngestDuplicateConflict,
			Detail: h.SourceID + "/" + h.EventID + ": содержимое отличается от принятого (seq " + fmt.Sprint(prior.Seq) + ")"}
		q.conflict = &IdempotencyConflict{SourceID: h.SourceID, EventID: h.EventID, FirstDigest: prior.Fingerprint,
			ConflictDigest: fp, FirstSeq: prior.Seq, OccurredAt: received}
		q.seqCounted = true // номер уже учтён первым сообщением
		return s.finishQ(start)(s.quarantineLocked(ctx, q))
	}

	// 5. Учёт source_seq (AD-7, FR-39).
	state, err := s.deps.Registry.SourceState(ctx, h.SourceID)
	if err != nil {
		return Result{}, err
	}
	state.SourceID = h.SourceID
	state2, obs := dom.ObserveSeq(state, h.SourceSeq, received)
	state2.SourceKind, state2.LastReceivedAt = h.SourceKind, received
	if len(h.Integrity.Signers) > 0 {
		state2.KeyRef = h.Integrity.Signers[0]
	}

	// 6. Флаги: UNKNOWN(значение), часы, последовательность (FR-29, FR-33, AD-5).
	var flags []FlagView
	for _, ef := range dec.Flags {
		flags = append(flags, FlagView{Flag: string(dom.FlagUnknownEnumValue), Field: ef.Field, RawValue: ef.StoredAs()})
	}
	for _, cf := range dom.CheckClock(occurred, received, mc.SentAt, s.cfg.Clock) {
		flags = append(flags, FlagView{Flag: string(cf.Flag), SkewMS: cf.SkewMS})
	}
	if mc.SentAt != nil {
		// FR-33: сдвиг часов оценивается по каждому источнику.
		state2.SkewMS, state2.HasSkew = mc.SentAt.Sub(received).Milliseconds(), true
	}
	if obs.Kind == dom.SeqViolation {
		flags = append(flags, FlagView{Flag: string(dom.FlagSequenceViolation), Field: "/source_seq", RawValue: fmt.Sprint(h.SourceSeq)})
	}

	// 7. Привязка к изделию до выбора партиции (AD-41, FR-34).
	var ref *dom.ItemRef
	if h.ItemRef != nil {
		ref = &dom.ItemRef{CarrierType: h.ItemRef.CarrierType, Value: h.ItemRef.Value, IdentificationLevel: h.ItemRef.IdentificationLevel}
	}
	b := dom.Bind(h.ItemID, ref)
	if b.NeedsLookup {
		reg := s.deps.Carriers
		if reg == nil {
			reg = noCarriers{}
		}
		res := crossitem.ResolveCarrier(reg, crossitem.CarrierRef{Type: ref.CarrierType, Value: ref.Value}, occurred)
		if !res.Ambiguous {
			b.ItemID = res.ItemID
		}
	}
	info, _ := catalog.Lookup(catalog.Type(h.EventType))
	stream, partition := s.streamOf(info, h, b)

	// 8. Запись факта и флагов одной пачкой (AD-44).
	envelope := raw
	if !f.DSSE {
		envelope = dsse(canon)
	}
	fact := s.entry(entryMeta{
		Type: catalog.Type(h.EventType), Version: h.SchemaVersion, EventID: h.EventID, SourceID: h.SourceID,
		SourceSeq: h.SourceSeq, RunID: h.RunID, ItemID: b.ItemID, CarrierRef: b.CarrierRef, Stream: stream,
		Partition: partition, OccurredAt: occurred, ReceivedAt: received, CorrelationID: h.CorrelationID,
		CausationID: deref(h.CausationID), Provenance: auth.Provenance,
	}, envelope)
	pend := []journal.Pending{fact}
	for _, fl := range flags {
		data := map[string]any{"subject_event_id": h.EventID, "flag": fl.Flag}
		if fl.Field != "" {
			data["field"] = fl.Field
		}
		if fl.RawValue != "" {
			data["raw_value"] = fl.RawValue
		}
		if fl.SkewMS != 0 {
			data["skew_ms"] = fl.SkewMS
		}
		fstream, fpart := stream, partition
		if b.ItemID == "" {
			fstream, fpart = "global", s.cfg.StagePartition
		}
		p, _, err := s.buildService(ctx, serviceRecord{Type: catalog.IngestAnomalyFlagged, Data: data, Stream: fstream,
			Partition: fpart, ItemID: b.ItemID, RunID: h.RunID, OccurredAt: occurred, ReceivedAt: received,
			CorrelationID: h.CorrelationID, CausationID: h.EventID})
		if err != nil {
			return Result{}, err
		}
		pend = append(pend, p)
	}
	seen := dom.Seen{SourceID: h.SourceID, EventID: h.EventID, Fingerprint: fp, SourceSeq: h.SourceSeq, SignatureVerified: auth.Verified}
	// Реестр приёма пишется в транзакции журнала (AppendRequest.Project, AD-45):
	// сбой между записью факта и реестром невозможен — повтор после сбоя
	// опознаётся как дубль, а не учитывается дважды (FR-31, FR-39).
	ar, err := s.deps.Journal.Append(ctx, journal.AppendRequest{Batch: pend, Project: func(ctx context.Context, res journal.AppendResult) error {
		if len(res.Seqs) > 0 {
			seen.Seq = res.Seqs[0]
		}
		return s.deps.Registry.Commit(ctx, seen, state2)
	}})
	if errors.Is(err, journal.ErrDuplicate) {
		// event_id уже в журнале (гонка копий api или сбой между записью и
		// реестром): журнал не пишет второй раз — это повтор (AD-7).
		seen.Seq = 0
		if err := s.deps.Registry.Commit(ctx, seen, state2); err != nil {
			return Result{}, err
		}
		return s.finish(start, 0, Result{Outcome: OutcomeDuplicate, Status: 200, SourceID: h.SourceID, EventID: h.EventID,
			EventType: h.EventType, SourceSeq: h.SourceSeq, SignatureVerified: auth.Verified, SignatureNote: auth.Note, Replayed: true,
			Detail: "повтор: запись уже есть в журнале"}, nil)
	}
	if err != nil {
		return Result{}, fmt.Errorf("журнал: %w", err)
	}
	seq := int64(0)
	if len(ar.Seqs) > 0 {
		seq = ar.Seqs[0]
	}
	s.gauge(MetricCompleteness, state2.CompletenessBP(), "source_id", h.SourceID)

	r := Result{Outcome: OutcomeAccepted, Status: 202, SourceID: h.SourceID, EventID: h.EventID, EventType: h.EventType,
		SourceSeq: h.SourceSeq, Seq: seq, ItemID: b.ItemID, Stream: stream, Partition: partition, Flags: flags,
		SignatureVerified: auth.Verified, SignatureNote: auth.Note}
	if dec.Outcome == dom.OutcomeAcceptedWithFlag {
		r.Outcome, r.Code, r.Field, r.Value, r.Detail = OutcomeAcceptedWithFlag, dec.Code, dec.Field, dec.Value, dec.Detail
	}
	if obs.Kind == dom.SeqGapFilled {
		r.Detail = strings.TrimPrefix(r.Detail+"; досылка: закрыт разрыв номеров", "; ")
	}
	return s.finish(start, received.Sub(occurred), r, nil)
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// finish учитывает метрики и возвращает итог; delay — задержка доставки.
func (s *Service) finish(start time.Time, delay time.Duration, r Result, err error) (Result, error) {
	if err != nil {
		return Result{}, err
	}
	s.observe(r, s.deps.InfraClock.Now().Sub(start), max(delay, 0))
	return r, nil
}

// finishQ — итог карантина: func(r, err) для вызова с результатом quarantine.
func (s *Service) finishQ(start time.Time) func(Result, error) (Result, error) {
	return func(r Result, err error) (Result, error) { return s.finish(start, 0, r, err) }
}

// checkContract — случаи FR-29 по каталогу и схемам конверта и data.
func (s *Service) checkContract(canon []byte, h header) (dom.Decision, error) {
	info, known := catalog.Lookup(catalog.Type(h.EventType))
	ti := dom.TypeInfo{Known: known, Versions: info.Versions, Fact: info.Kind == catalog.KindFact}
	if h.EventType == "" {
		ti.Known = true // нет обязательного поля event_type — скажет схема конверта
		ti.Fact = true
		ti.Versions = []int{h.SchemaVersion}
	}
	if d, stop := dom.ClassifyVersion(h.EventType, h.SchemaVersion, ti); stop {
		return d, nil
	}
	doc, err := parseDoc(canon)
	if err != nil {
		return dom.Decision{Outcome: dom.OutcomeQuarantined, Code: errcodes.IngestCanonicalFormViolation, Detail: err.Error()}, nil
	}
	vs, err := s.schemas.validate(envelopeSchema, doc, "")
	if err != nil {
		return dom.Decision{}, fmt.Errorf("схема конверта: %w", err)
	}
	if obj, ok := doc.(map[string]any); ok && h.EventType != "" {
		if data, ok := obj["data"]; ok {
			dv, err := s.schemas.validate(dataSchemaPath(h.EventType, h.SchemaVersion), data, "/data")
			if err != nil {
				return dom.Decision{}, fmt.Errorf("схема %s v%d: %w", h.EventType, h.SchemaVersion, err)
			}
			vs = append(vs, dv...)
		}
	}
	return dom.Classify(h.EventType, vs), nil
}

// streamOf — поток и партиция факта (AD-39, AD-41): у типа с потоком «изделие» —
// item:‹id›, партиция по item_id; неразрешённое — поток межизделийной стадии
// (global, партиция стадии); у прочих — ‹вид›:‹id› из data.‹вид›_id, иначе
// source:‹source_id›.
func (s *Service) streamOf(info catalog.Info, h header, b dom.Binding) (string, int) {
	if b.ItemID != "" {
		return "item:" + b.ItemID, kernel.PartitionOf(b.ItemID, s.cfg.Partitions)
	}
	if info.Stream == "item" || info.Stream == "" || info.Stream == "global" {
		return "global", s.cfg.StagePartition
	}
	var data map[string]any
	_ = json.Unmarshal(h.Data, &data)
	if id, ok := data[info.Stream+"_id"].(string); ok && id != "" {
		return info.Stream + ":" + id, s.cfg.StagePartition
	}
	return "source:" + h.SourceID, s.cfg.StagePartition
}

// noCarriers — реестра носителей нет: носитель не разрешается (стадия разберёт).
type noCarriers struct{}

func (noCarriers) Lookup(crossitem.CarrierRef, time.Time) []string { return nil }

// quarantineIn — вход записи карантина.
type quarantineIn struct {
	raw      []byte
	received time.Time
	h        header
	dec      dom.Decision
	authed   bool
	keyRef   string
	// sig, conflict — событие безопасности вместе с карантином.
	sig      *SignatureFailure
	conflict *IdempotencyConflict
	// seqCounted — номер источника учитывать не нужно.
	seqCounted bool
}

func (s *Service) quarantine(ctx context.Context, q quarantineIn) (Result, error) {
	if q.h.SourceID != "" {
		defer s.lock(q.h.SourceID)()
	}
	return s.quarantineLocked(ctx, q)
}

// quarantineLocked — карантин (FR-30, AD-2): содержимое — в MaterialStore по
// адресу H(байты), в журнал — служебная запись ingest.message.quarantined с
// источником, номером, отпечатком и кодом; повтор тех же байтов с тем же кодом
// новой записи не создаёт. Номер источника учитывается (приём ведёт учёт
// последовательности вместе с карантином, AD-7), если источник подлинный.
func (s *Service) quarantineLocked(ctx context.Context, q quarantineIn) (Result, error) {
	code := q.dec.Code
	fp := dom.Digest(q.raw)
	src := q.h.SourceID
	if !sourceIDRe(src) {
		src = "unknown"
	}
	res := Result{Outcome: OutcomeQuarantined, Status: statusOf(code, 422), Code: code, Field: q.dec.Field, Value: q.dec.Value,
		Detail: q.dec.Detail, SourceID: src, EventID: q.h.EventID, EventType: q.h.EventType, SourceSeq: q.h.SourceSeq,
		SchemaVersion: q.h.SchemaVersion, KeyRef: q.keyRef}
	if q.conflict != nil {
		res.Outcome = OutcomeConflict
	}
	if prev, err := s.deps.Quarantine.Find(ctx, fp, code); err != nil {
		return Result{}, err
	} else if prev != nil {
		res.QuarantineID, res.Replayed = prev.ID, true
		return res, nil
	}
	addr := fp
	rec := QuarantineRecord{SourceID: src, EventID: q.h.EventID, SourceSeq: q.h.SourceSeq, EventType: q.h.EventType,
		Fingerprint: fp, Code: code, Field: q.dec.Field, Detail: q.dec.Detail, Status: QuarantineOpen, ReceivedAt: q.received}
	if s.deps.Materials != nil {
		a, err := s.deps.Materials.Put(ctx, strings.NewReader(string(q.raw)), materials.Meta{ContentType: "application/json", Size: int64(len(q.raw)), Kind: "quarantine"})
		if err != nil {
			return Result{}, fmt.Errorf("хранилище материалов: %w", err)
		}
		addr = a
	} else {
		rec.Raw = q.raw
	}
	rec.MaterialAddress = addr
	data := map[string]any{"source_id": src, "fingerprint": fp, "material_address": addr, "problem_code": string(code)}
	if q.h.SourceSeq > 0 {
		data["source_seq"] = q.h.SourceSeq
	}
	if isUUID(q.h.EventID) {
		data["event_id"] = q.h.EventID
	}
	if q.dec.Detail != "" {
		data["detail"] = truncate(q.dec.Detail, 4000)
	}
	p, qid, err := s.buildService(ctx, serviceRecord{Type: catalog.IngestMessageQuarantined, Data: data, Stream: "source:" + src,
		Partition: s.cfg.StagePartition, OccurredAt: q.received, ReceivedAt: q.received})
	if err != nil {
		return Result{}, err
	}
	rec.ID = qid
	res.QuarantineID = qid
	req := journal.AppendRequest{Batch: []journal.Pending{p}}
	now := s.deps.InfraClock.Now()
	switch {
	case q.sig != nil && s.limiter.allow("sig:"+src, now):
		q.sig.QuarantineID = qid
		m, c, err := s.deps.Security.SignatureInvalid(ctx, *q.sig)
		if err != nil {
			return Result{}, err
		}
		req.Batch, req.Critical = append(req.Batch, m...), append(req.Critical, c...)
	case q.conflict != nil && s.limiter.allow("conflict:"+src, now):
		q.conflict.QuarantineID = qid
		m, c, err := s.deps.Security.IdempotencyConflict(ctx, *q.conflict)
		if err != nil {
			return Result{}, err
		}
		req.Batch, req.Critical = append(req.Batch, m...), append(req.Critical, c...)
	case q.sig != nil || q.conflict != nil:
		s.stats.mu.Lock()
		s.stats.rejects["security_suppressed"]++
		s.stats.mu.Unlock()
		if s.deps.Telemetry != nil {
			s.deps.Telemetry.Counter(MetricSecuritySuppressed, 1, "source_id", src)
		}
	}
	// Номер подлинного источника учитывается и у сообщения в карантине (AD-7).
	var seqState *dom.SourceState
	if q.authed && !q.seqCounted && q.h.SourceSeq > 0 && src != "unknown" {
		st, err := s.deps.Registry.SourceState(ctx, src)
		if err != nil {
			return Result{}, err
		}
		st.SourceID = src
		if st2, obs := dom.ObserveSeq(st, q.h.SourceSeq, q.received); obs.Kind != dom.SeqViolation {
			seqState = &st2
		}
	}
	// Запись карантина и учёт номера — в транзакции журнала (AD-45).
	req.Project = func(ctx context.Context, r journal.AppendResult) error {
		if len(r.Seqs) > 0 {
			rec.JournalSeq = r.Seqs[0]
		}
		if err := s.deps.Quarantine.Put(ctx, rec); err != nil {
			return err
		}
		if seqState != nil {
			return s.deps.Registry.SaveSourceState(ctx, *seqState)
		}
		return nil
	}
	ar, err := s.deps.Journal.Append(ctx, req)
	if err != nil {
		return Result{}, fmt.Errorf("журнал: %w", err)
	}
	if len(ar.CARefs) > 0 {
		res.CARef = ar.CARefs[0]
		res.Detail += " (" + ar.CARefs[0] + ")"
	}
	if n, err := s.unresolved(ctx); err == nil {
		s.gauge(MetricQuarantineOpen, n)
	}
	return res, nil
}

func sourceIDRe(s string) bool {
	if s == "" || len(s) > 128 {
		return false
	}
	for i, r := range s {
		ok := r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if i > 0 {
			ok = ok || strings.ContainsRune("._:/@-", r)
		}
		if !ok {
			return false
		}
	}
	return true
}

func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil && len(s) == 36 && s == strings.ToLower(s)
}

// truncate обрезает строку до n символов (не байтов: UTF-8 не рвётся).
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
