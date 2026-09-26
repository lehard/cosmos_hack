package ingest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/ingest"
)

// Реализация ведущих портов Queries и Commands (формы операций — views.go):
// операции ingest.* эпика 02 вызывают эти методы; внутри — конвейер process.

// outcomeOf переводит итог конвейера в форму операции.
func outcomeOf(i int, r Result) IngestOutcome {
	o := IngestOutcome{Index: i, Flags: []string{}, SignatureChecked: r.SignatureVerified}
	if r.EventID != "" {
		o.EventID = &r.EventID
	}
	switch r.Outcome {
	case OutcomeAccepted, OutcomeAcceptedWithFlag:
		o.Status = "accepted"
	case OutcomeDuplicate:
		o.Status = "duplicate"
	default:
		o.Status = "quarantined"
	}
	if r.Seq > 0 {
		o.Seq = &r.Seq
	}
	if r.Code != "" {
		c := string(r.Code)
		o.Code = &c
	}
	if r.QuarantineID != "" {
		o.QuarantineID = &r.QuarantineID
	}
	for _, f := range r.Flags {
		o.Flags = append(o.Flags, f.Flag)
	}
	return o
}

// SubmitBatch — приём пачки конвертов (ingest.batch.submit, FR-26…FR-31, FR-39):
// каждый конверт проходит конвейер отдельно; итог — по каждому.
func (s *Service) SubmitBatch(ctx context.Context, in IngestBatch) (IngestResult, error) {
	if err := s.ready(); err != nil {
		return IngestResult{}, platform.NotImplemented("ingest.batch.submit")
	}
	if len(in.Envelopes) > s.cfg.MaxBatch {
		return IngestResult{}, platform.Fail(errcodes.IngestBatchTooLarge, "count", strconv.Itoa(len(in.Envelopes)), "limit", strconv.Itoa(s.cfg.MaxBatch))
	}
	mc := msgCtx{}
	if in.SentAt != nil {
		t := in.SentAt.UTC()
		mc.SentAt = &t
	}
	out := IngestResult{Items: make([]IngestOutcome, 0, len(in.Envelopes))}
	for i, env := range in.Envelopes {
		raw, err := json.Marshal(env)
		if err != nil {
			return out, err
		}
		r, err := s.process(ctx, raw, mc)
		if err != nil {
			return out, err
		}
		o := outcomeOf(i, r)
		switch o.Status {
		case "accepted":
			out.Accepted++
		case "duplicate":
			out.Duplicates++
		default:
			out.Quarantined++
		}
		out.Items = append(out.Items, o)
	}
	return out, nil
}

// SubmitEvent — одно событие ручного ввода или терминала участка
// (ingest.event.submit, FR-137, FR-141): подписанный конверт; проходит тот же
// конвейер, пометка источника — из конверта (source_kind = manual_entry, FR-140).
func (s *Service) SubmitEvent(ctx context.Context, in ManualEvent) (IngestOutcome, error) {
	if err := s.ready(); err != nil {
		return IngestOutcome{}, platform.NotImplemented("ingest.event.submit")
	}
	raw, err := json.Marshal(in.Envelope)
	if err != nil {
		return IngestOutcome{}, err
	}
	r, err := s.process(ctx, raw, msgCtx{})
	if err != nil {
		return IngestOutcome{}, err
	}
	if r.Outcome == OutcomeQuarantined || r.Outcome == OutcomeConflict {
		// FR-29, кейс §4.7: «нет обязательного поля — problem+json с кодом и
		// карантин»: отказ по одному сообщению — problem+json с кодом, ссылкой
		// на запись карантина и, при конфликте, на критическое действие.
		return IngestOutcome{}, rejection(r)
	}
	return outcomeOf(0, r), nil
}

// rejection — отказ по сообщению как ошибка порта: код из contracts/errors.yaml
// и параметры шаблона текста (field, value, event_type, version, reason…).
func rejection(r Result) error {
	e := platform.Fail(r.Code, "field", r.Field, "value", r.Value, "event_type", r.EventType,
		"version", strconv.Itoa(r.SchemaVersion), "reason", r.Detail, "source_id", r.SourceID, "event_id", r.EventID,
		"key_ref", r.KeyRef, "quarantine_id", r.QuarantineID)
	e.CARef = r.CARef
	return e
}

// Import — импорт CSV (ingest.import.submit, FR-141). Mapping — тип события
// строк (`operation.run.finished`) или имя журнала (`weld-journal`): источник
// импорта — `import:‹mapping›`. Excel (xlsx) — сохранить как CSV (разделитель
// «;» понимается). DryRun — только проверка, без записи.
func (s *Service) Import(ctx context.Context, in ImportFile) (ImportResult, error) {
	if err := s.ready(); err != nil {
		return ImportResult{}, platform.NotImplemented("ingest.import.submit")
	}
	if in.Format == "xlsx" {
		e := platform.Fail(errcodes.ApiValidationFailed, "field", "format", "reason", "xlsx пока не поддержан — сохраните лист как CSV")
		e.Detail = "Импорт Excel: сохраните лист как CSV (разделитель «;» или «,»)"
		return ImportResult{}, e
	}
	content, err := base64.StdEncoding.DecodeString(in.Content)
	if err != nil {
		e := platform.Fail(errcodes.ApiValidationFailed, "field", "content", "reason", err.Error())
		e.Detail = "Содержимое файла — не base64"
		return ImportResult{}, e
	}
	src, def := "import:csv", map[string]string{}
	if m := strings.TrimSpace(in.Mapping); m != "" {
		if _, ok := catalog.Lookup(catalog.Type(m)); ok {
			def["event_type"] = m
		}
		src = "import:" + m
	}
	if s.deps.Materials != nil && !in.DryRun {
		// FR-141: файл хранится в хранилище материалов по адресу H(байты).
		if _, err := s.deps.Materials.Put(ctx, strings.NewReader(string(content)), materialsMeta(len(content))); err != nil {
			return ImportResult{}, err
		}
	}
	rep, err := s.ImportCSV(ctx, Cmd[ImportInput]{Body: ImportInput{SourceID: src, FileName: in.FileName, Content: content,
		Defaults: def, DryRun: in.DryRun}})
	if err != nil {
		return ImportResult{}, err
	}
	out := ImportResult{SourceID: src, FileDigest: rep.FileDigest, RowsTotal: rep.Total, RowsAccepted: rep.Accepted,
		RowsDuplicate: rep.Duplicate, RowsRejected: rep.Rejected, Rows: make([]IngestOutcome, 0, len(rep.Rows))}
	for _, r := range rep.Rows {
		out.Rows = append(out.Rows, outcomeOf(r.Line, r.Result))
	}
	return out, nil
}

// Reprocess — переобработка из карантина (ingest.message.reprocess, FR-30).
func (s *Service) Reprocess(ctx context.Context, quarantineID string, in ReprocessMessage) (platform.Receipt, error) {
	meta := in.CommandMeta()
	meta.Reason = in.Reason.Text
	r, err := s.ReprocessCommand(ctx, Cmd[ReprocessInput]{Meta: meta, Body: ReprocessInput{QuarantineID: quarantineID, Discard: in.Discard}})
	if err != nil {
		return platform.Receipt{}, err
	}
	return r.Receipt, nil
}

// entryOf — запись карантина в форме операции.
func entryOf(v QuarantineView) QuarantineEntry {
	e := QuarantineEntry{QuarantineID: v.ID, SourceID: v.SourceID, Fingerprint: v.Fingerprint, MaterialAddress: v.MaterialAddress,
		ProblemCode: string(v.Code), QuarantinedAt: v.ReceivedAt, State: string(v.Status)}
	if v.SourceSeq > 0 {
		e.SourceSeq = &v.SourceSeq
	}
	if v.EventID != "" {
		e.EventID = &v.EventID
	}
	if v.EventType != "" {
		e.EventType = &v.EventType
	}
	if v.Detail != "" {
		e.Detail = &v.Detail
	}
	return e
}

// pageOffset — непрозрачный курсор страницы — смещение.
func pageOffset(p platform.Page) (int, int) {
	off, _ := strconv.Atoi(p.Cursor)
	lim := p.Limit
	if lim <= 0 || lim > 500 {
		lim = 50
	}
	return max(off, 0), lim
}

// Quarantine — карантин сообщений (ingest.quarantine.list, FR-30).
func (s *Service) Quarantine(ctx context.Context, f QuarantineFilter, p platform.Page) (QuarantineList, error) {
	if err := s.ready(); err != nil {
		return QuarantineList{}, platform.NotImplemented("ingest.quarantine.list")
	}
	basis, err := s.basisSeq(ctx)
	if err != nil {
		return QuarantineList{}, err
	}
	off, lim := pageOffset(p)
	vs, err := s.QuarantineRecords(ctx, QuarantineQuery{Status: QuarantineStatus(f.State), SourceID: f.SourceID,
		Code: errcodes.Code(f.Code), Offset: off, Limit: lim + 1})
	if err != nil {
		return QuarantineList{}, err
	}
	out := QuarantineList{Items: []QuarantineEntry{}}
	for i, v := range vs {
		if i == lim {
			out.NextCursor = strconv.Itoa(off + lim)
			break
		}
		e := entryOf(v)
		e.BasisSeq = basis
		out.Items = append(out.Items, e)
	}
	return out, nil
}

// basisSeq — голова основной цепочки до чтения: seq, на котором построен ответ
// (basis_seq команд над карантином и источниками, AD-39). Берётся раньше чтения
// проекций, поэтому не больше того, что они отражают.
func (s *Service) basisSeq(ctx context.Context) (int64, error) {
	h, err := s.deps.Journal.Head(ctx)
	if err != nil {
		return 0, err
	}
	return h.MainSeq, nil
}

// QuarantineEntry — сообщение в карантине с содержимым (ingest.quarantine.read).
func (s *Service) QuarantineEntry(ctx context.Context, id string) (QuarantineEntry, error) {
	if err := s.ready(); err != nil {
		return QuarantineEntry{}, platform.NotImplemented("ingest.quarantine.read")
	}
	basis, err := s.basisSeq(ctx)
	if err != nil {
		return QuarantineEntry{}, err
	}
	v, err := s.QuarantineItem(ctx, id)
	if err != nil {
		return QuarantineEntry{}, err
	}
	e := entryOf(v)
	e.BasisSeq = basis
	if rec, err := s.deps.Quarantine.Get(ctx, id); err == nil {
		if raw, err := s.content(ctx, rec); err == nil {
			c := string(raw)
			e.Content = &c
		}
	}
	return e, nil
}

// Sources — источники событий: непрерывность source_seq, расхождение часов,
// карантин (ingest.source.list, AD-7, AD-9, FR-33).
func (s *Service) Sources(ctx context.Context, p platform.Page) (SourceList, error) {
	if err := s.ready(); err != nil {
		return SourceList{}, platform.NotImplemented("ingest.source.list")
	}
	basis, err := s.basisSeq(ctx)
	if err != nil {
		return SourceList{}, err
	}
	srcs, err := s.deps.Registry.Sources(ctx)
	if err != nil {
		return SourceList{}, err
	}
	q, err := s.deps.Quarantine.CountBySource(ctx)
	if err != nil {
		return SourceList{}, err
	}
	off, lim := pageOffset(p)
	out := SourceList{Items: []SourceView{}}
	for i, st := range srcs {
		if i < off {
			continue
		}
		if len(out.Items) == lim {
			out.NextCursor = strconv.Itoa(off + lim)
			break
		}
		v := SourceView{SourceID: st.SourceID, SourceKind: st.SourceKind, State: "active", GapCount: len(st.Gaps),
			Quarantined: int(q[st.SourceID]), BasisSeq: basis}
		if v.SourceKind == "" {
			v.SourceKind = "unknown"
		}
		if st.HighWater > 0 {
			hw := st.HighWater
			v.LastSeq = &hw
		}
		if !st.LastReceivedAt.IsZero() {
			t := st.LastReceivedAt
			v.LastReceivedAt = &t
		}
		if st.HasSkew {
			sk := st.SkewMS
			v.ClockSkewMs = &sk
		}
		if slices.ContainsFunc(st.Gaps, func(g dom.Gap) bool { return g.Reported }) {
			v.State = "loss_suspected"
		}
		if st.KeyRef != "" {
			k := st.KeyRef
			v.KeyRef = &k
		}
		out.Items = append(out.Items, v)
	}
	return out, nil
}

// Metrics — метрики приёма (ingest.metrics.read, FR-41).
func (s *Service) Metrics(ctx context.Context) (IngestMetrics, error) {
	st, err := s.Stats(ctx)
	if err != nil {
		return IngestMetrics{}, err
	}
	now := s.deps.InfraClock.Now()
	m := IngestMetrics{WindowFrom: s.started, WindowTo: now, Accepted: st.Accepted + st.AcceptedWithFlag, Duplicates: st.Duplicates,
		Rejected: st.Conflicts, Quarantined: st.Quarantined, QuarantineOpen: st.QuarantineOpen,
		LatencyP50Ms: st.LatencyP50MS, LatencyP95Ms: st.LatencyP95MS, CompletenessBP: 10000}
	m.Received = m.Accepted + m.Duplicates + m.Rejected + m.Quarantined
	var have, want int64
	for _, src := range st.Sources {
		have += src.HighWater - src.Missing
		want += src.HighWater
	}
	if want > 0 {
		m.CompletenessBP = int(have * 10000 / want)
	}
	return m, nil
}
