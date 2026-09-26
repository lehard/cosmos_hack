package ingest

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/ingest"
	"ant/internal/domain/kernel"
)

// Формат CSV импорта журнала (FR-141): первая строка — заголовок. Столбцы:
//
//	event_type, schema_version     — тип события (или Defaults["event_type"]);
//	occurred_at                    — RFC 3339 или «ГГГГ-ММ-ДД ЧЧ:ММ[:СС]» со смещением Defaults["utc_offset"] (по умолчанию +00:00);
//	item_id | carrier_type + carrier_value [+ identification_level] — привязка;
//	row_id                         — номер строки журнала (устойчивый ключ дубля);
//	reliability                    — надёжность (по умолчанию medium);
//	data.‹путь›[:int|:bool|:string] — поле data (точки — вложенность).
//
// Разделитель — «,» или «;» (Excel в русской локали). event_id строки =
// UUIDv5(NS_ANT, «import» ‖ source_id ‖ row_id или содержимое строки): повторный
// импорт того же файла даёт дубли, а не второй учёт.

// ImportCSV — FR-141: журнал, загруженный из CSV, даёт те же события, что и
// данные источника, с пометкой «импорт» (source_kind = import), через тот же
// конвейер с проверкой входов, дублей и привязки; итог — ingest.import.completed.
func (s *Service) ImportCSV(ctx context.Context, cmd Cmd[ImportInput]) (ImportReport, error) {
	if err := s.ready(); err != nil {
		return ImportReport{}, platform.NotImplemented("ingest.import.submit")
	}
	if v, ok := s.replay(cmd.Meta.CommandID); ok && cmd.Meta.CommandID != "" {
		r := v.(ImportReport)
		r.Receipt.Replayed = true
		return r, nil
	}
	in := cmd.Body
	src := in.SourceID
	if src == "" {
		src = "import:csv"
	}
	rows, err := readCSV(in.Content)
	if err != nil {
		e := platform.Fail(errcodes.ApiValidationFailed, "field", "content", "reason", err.Error())
		e.Detail = "CSV не разобран: " + err.Error()
		return ImportReport{}, e
	}
	p := platform.PrincipalFrom(ctx)
	out := ImportReport{FileDigest: dom.Digest(in.Content)}
	for i, row := range rows {
		line := i + 2
		raw, perr := s.importRow(src, row, in.Defaults, p.PersonID)
		var r Result
		if perr != nil {
			r = Result{Outcome: OutcomeQuarantined, Status: 422, Code: errcodes.IngestSchemaViolation, SourceID: src,
				Detail: fmt.Sprintf("строка %d: %v", line, perr)}
		} else if in.DryRun {
			if r, err = s.dryCheck(ctx, raw); err != nil {
				return ImportReport{}, err
			}
		} else if r, err = s.process(ctx, raw, msgCtx{Manual: &manualAuth{Provenance: "personal", Note: "импорт журнала, " + ManualNote}}); err != nil {
			return ImportReport{}, err
		}
		out.Rows = append(out.Rows, ImportRow{Line: line, Result: r})
		out.Total++
		switch r.Outcome {
		case OutcomeAccepted, OutcomeAcceptedWithFlag:
			out.Accepted++
		case OutcomeDuplicate:
			out.Duplicate++
		default:
			out.Rejected++
		}
	}
	if in.DryRun {
		return out, nil
	}
	now, err := s.deps.DomainClock.Now(ctx)
	if err != nil {
		return ImportReport{}, err
	}
	data := map[string]any{"source_id": src, "file_digest": out.FileDigest, "rows_total": out.Total,
		"rows_accepted": out.Accepted, "rows_duplicate": out.Duplicate, "rows_rejected": out.Rejected}
	if in.FileName != "" {
		data["file_name"] = truncate(in.FileName, 256)
	}
	if p.PersonID != "" {
		data["imported_by"] = p.PersonID
	}
	pend, id, err := s.buildService(ctx, serviceRecord{Type: catalog.IngestImportCompleted, Data: data, Stream: "source:" + src,
		Partition: s.cfg.StagePartition, OccurredAt: now, ReceivedAt: now})
	if err != nil {
		return ImportReport{}, err
	}
	ar, err := s.deps.Journal.Append(ctx, journal.AppendRequest{Batch: []journal.Pending{pend}})
	if err != nil {
		return ImportReport{}, err
	}
	out.Receipt = platform.Receipt{CommandID: cmd.Meta.CommandID, EventIDs: []string{id}, RecordedAt: now}
	if len(ar.Seqs) > 0 {
		out.Receipt.Seq = ar.Seqs[0]
	}
	s.remember(cmd.Meta.CommandID, out)
	return out, nil
}

// csvRow — строка CSV: столбец → значение.
type csvRow map[string]string

func readCSV(content []byte) ([]csvRow, error) {
	content = bytes.TrimPrefix(content, []byte("\xef\xbb\xbf")) // BOM Excel
	sep := ','
	if first, _, _ := bytes.Cut(content, []byte("\n")); bytes.Count(first, []byte(";")) > bytes.Count(first, []byte(",")) {
		sep = ';'
	}
	r := csv.NewReader(bytes.NewReader(content))
	r.Comma = sep
	r.TrimLeadingSpace = true
	head, err := r.Read()
	if err != nil {
		return nil, err
	}
	for i := range head {
		head[i] = strings.TrimSpace(head[i])
	}
	var out []csvRow
	for {
		rec, err := r.Read()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		row := csvRow{}
		for i, v := range rec {
			if i < len(head) && head[i] != "" {
				row[head[i]] = strings.TrimSpace(v)
			}
		}
		out = append(out, row)
	}
}

// importRow строит событие из строки CSV.
func (s *Service) importRow(src string, row csvRow, def map[string]string, person string) ([]byte, error) {
	get := func(k string) string {
		if v := row[k]; v != "" {
			return v
		}
		return def[k]
	}
	et := get("event_type")
	if et == "" {
		return nil, fmt.Errorf("нет event_type")
	}
	ver := 1
	if v := get("schema_version"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("schema_version %q: %v", v, err)
		}
		ver = n
	}
	occ, err := parseLocal(get("occurred_at"), get("utc_offset"))
	if err != nil {
		return nil, err
	}
	data := map[string]any{}
	for _, k := range slices.Sorted(maps.Keys(row)) {
		path, ok := strings.CutPrefix(k, "data.")
		if !ok || row[k] == "" {
			continue
		}
		path, typ, _ := strings.Cut(path, ":")
		var v any = row[k]
		switch typ {
		case "int":
			n, err := strconv.ParseInt(row[k], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s: %q — не целое", k, row[k])
			}
			v = n
		case "bool":
			v = row[k] == "true" || row[k] == "1" || strings.EqualFold(row[k], "да")
		}
		setPath(data, strings.Split(path, "."), v)
	}
	key := get("row_id")
	if key == "" {
		b, _ := json.Marshal(row)
		key = dom.Digest(b)
	}
	id := kernel.UUIDv5(constants.NsAnt, "import\x1f"+src+"\x1f"+key)
	signer := "person-" + strings.ToLower(person) + "@1"
	if person == "" {
		signer = "person-anonymous@1"
	}
	rel := get("reliability")
	if rel == "" {
		rel = "medium"
	}
	ev := map[string]any{
		"event_id": id, "event_type": et, "schema_version": ver, "source_id": src, "source_kind": "import",
		"reliability": rel, "occurred_at": ts(occ), "correlation_id": id, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{signer}},
		"data":      data,
	}
	if v := get("item_id"); v != "" {
		ev["item_id"] = v
	}
	if ct := get("carrier_type"); ct != "" {
		lvl := get("identification_level")
		if lvl == "" {
			lvl = "probable"
		}
		ev["item_ref"] = map[string]any{"carrier_type": ct, "value": get("carrier_value"), "identification_level": lvl}
	}
	return json.Marshal(ev)
}

func setPath(m map[string]any, path []string, v any) {
	for _, p := range path[:len(path)-1] {
		next, ok := m[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[p] = next
		}
		m = next
	}
	m[path[len(path)-1]] = v
}

// parseLocal — время строки импорта: RFC 3339 или «ГГГГ-ММ-ДД ЧЧ:ММ[:СС]» со смещением.
func parseLocal(v, offset string) (time.Time, error) {
	if v == "" {
		return time.Time{}, fmt.Errorf("нет occurred_at")
	}
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return t.UTC().Truncate(time.Millisecond), nil
	}
	if offset == "" {
		offset = "+00:00"
	}
	for _, layout := range []string{"2006-01-02 15:04:05Z07:00", "2006-01-02 15:04Z07:00"} {
		if t, err := time.Parse(layout, v+offset); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("occurred_at %q: ждали RFC 3339 или ГГГГ-ММ-ДД ЧЧ:ММ[:СС]", v)
}

// dryCheck — проверка строки без записи: схема, пять случаев и дубль по реестру.
func (s *Service) dryCheck(ctx context.Context, raw []byte) (Result, error) {
	canon, err := dom.Canonicalize(raw)
	if err != nil {
		return Result{Outcome: OutcomeQuarantined, Status: 422, Code: errcodes.IngestCanonicalFormViolation, Detail: err.Error()}, nil
	}
	h := parseHeader(canon)
	r := Result{SourceID: h.SourceID, EventID: h.EventID, EventType: h.EventType, Status: 202, Outcome: OutcomeAccepted}
	d, err := s.checkContract(canon, h)
	if err != nil {
		return Result{}, err
	}
	if d.Outcome == dom.OutcomeQuarantined {
		r.Outcome, r.Status, r.Code, r.Field, r.Detail = OutcomeQuarantined, statusOf(d.Code, 422), d.Code, d.Field, d.Detail
		return r, nil
	}
	fp, err := dom.ContentFingerprint(canon)
	if err != nil {
		return Result{}, err
	}
	prior, err := s.deps.Registry.Seen(ctx, h.SourceID, h.EventID)
	if err != nil {
		return Result{}, err
	}
	switch dom.CheckRepeat(prior, fp) {
	case dom.RepeatDuplicate:
		r.Outcome, r.Status, r.Seq = OutcomeDuplicate, 200, prior.Seq
	case dom.RepeatConflict:
		r.Outcome, r.Status, r.Code = OutcomeConflict, 409, errcodes.IngestDuplicateConflict
	}
	return r, nil
}
