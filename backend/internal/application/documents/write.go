package documents

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"uuid"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/documents"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// Запись команд модуля documents (AD-39, AD-44): одна запись-решение (или
// факт бумажного экземпляра) в поток документа `document:‹id›`; у документа
// изделия — с item_id изделия, так что запись входит в его свёртку и версию
// и закрытие маршрута вычисляет воркер. У документа вне изделия api в той же
// пачке пишет реакции модуля (версию и закрытие маршрута), вычисленные той
// же доменной функцией (FoldStream + Reactions).
//
// Конверт — DSSE без подписей в демо (Д-30): подпись агента токена над
// отпечатком лежит в data (signature), её проверку делает signing (эпик 27);
// запись критического действия (AD-28) — через CriticalActions эпика 29.

// SourceAPI — source_id записей, принятых операциями API модуля.
const SourceAPI = "ant-api"

// objectID — шаблон object_id контракта (contracts/events/common/defs.v1.json).
var objectID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@-]{0,127}$`)

// digestRe — шаблон digest контракта: адрес материала (скан бумажной подписи).
var digestRe = regexp.MustCompile(`^streebog256:[0-9a-f]{64}$`)

// out — запись к Append.
type out struct {
	Type       catalog.Type
	DocumentID string
	ItemID     string
	RunID      string
	Data       any
	Meta       platform.CommandMeta
	Actor      string
	OccurredAt time.Time
	Level      int
	// EventID — id записи; пусто — command_id (AD-7) или новый UUIDv7.
	EventID string
	// Basis — seq, на котором гард проверил поток (basis_seq конверта, AD-39);
	// 0 — basis_seq клиента.
	Basis int64
}

// pending — запись журнала и её представление в домене (для свёртки до записи).
func (s *Service) pending(o out) (appjournal.Pending, kernel.Record, error) {
	info, ok := catalog.Lookup(o.Type)
	if !ok || info.Emitter != string(dom.Module) {
		return appjournal.Pending{}, kernel.Record{}, errors.New("documents: запись чужого типа " + string(o.Type))
	}
	id := strings.ToLower(o.EventID)
	if id == "" {
		id = strings.ToLower(o.Meta.CommandID)
	}
	if _, err := uuid.Parse(id); err != nil || id == "" {
		id = uuid.NewV7().String()
	}
	data, err := json.Marshal(o.Data)
	if err != nil {
		return appjournal.Pending{}, kernel.Record{}, err
	}
	canonData, err := dom.Canonical(data)
	if err != nil {
		return appjournal.Pending{}, kernel.Record{}, err
	}
	occurred := engineapp.FormatTime(o.OccurredAt)
	// Подписант конверта `key_id@версия`: до реестра ключей (эпики 05, 27) —
	// псевдоним в нижнем регистре (шаблон key_ref контракта).
	actor := strings.ToLower(o.Actor) + "@1"
	basis := max(o.Basis, o.Meta.BasisSeq, 1)
	policy := max(o.Meta.PolicySeq, 1)
	env := map[string]any{
		"event_id": id, "event_type": string(o.Type), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
		"source_kind": "manual_entry", "occurred_at": occurred, "correlation_id": id, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{actor}},
		"data":      json.RawMessage(canonData),
	}
	if info.Kind == catalog.KindDecision {
		cmd := map[string]any{"command_id": id, "basis_seq": basis, "guard_streams": []string{streamOf(o.DocumentID)},
			"policy_seq": policy, "signature_level": o.Level}
		if o.Meta.WorkplaceID != "" {
			cmd["workplace_id"] = o.Meta.WorkplaceID
		}
		env["command"] = cmd
	}
	if o.ItemID != "" {
		env["item_id"] = o.ItemID
	}
	if o.RunID != "" {
		env["run_id"] = o.RunID
	}
	canon, err := engine.Canonical(env)
	if err != nil {
		return appjournal.Pending{}, kernel.Record{}, err
	}
	sealed, _ := json.Marshal(struct {
		PayloadType string   `json:"payloadType"`
		Payload     string   `json:"payload"`
		Signatures  []string `json:"signatures"`
	}{engineapp.PayloadTypeEvent, base64.StdEncoding.EncodeToString(canon), []string{}})
	now := s.d.Now()
	e := jc.JournalEntry{
		Chain: jc.JournalEntryChainMain, EntryKind: jc.JournalEntryEntryKind(info.Kind), EventType: string(o.Type),
		SchemaVersion: info.CurrentVersion, EventID: id, SourceID: SourceAPI, Stream: streamOf(o.DocumentID),
		OccurredAt: occurred, ReceivedAt: engineapp.FormatTime(now), CorrelationID: id,
		ProvenanceClass: jc.JournalEntryProvenanceClassPersonal, DomainBuild: s.cfg.DomainBuild,
	}
	if s.cfg.ScenarioClock {
		e.RecordedAt = occurred
	}
	if o.ItemID != "" {
		item := o.ItemID
		e.ItemID = &item
		e.Partition = kernel.PartitionOf(o.ItemID, s.cfg.Partitions)
	} else {
		// Записи вне изделия — партиция стадии за пределами 0…P-1 (как у приёма).
		e.Partition = s.cfg.Partitions
	}
	if o.RunID != "" {
		run := o.RunID
		e.RunID = &run
	}
	if info.Kind == catalog.KindDecision {
		b, p := int(basis), int(policy)
		e.BasisSeq, e.PolicySeq = &b, &p
	}
	rec := kernel.Record{EventID: id, Type: o.Type, SchemaVersion: info.CurrentVersion, Kind: info.Kind, SourceID: SourceAPI, SourceKind: "manual_entry",
		Provenance: string(jc.JournalEntryProvenanceClassPersonal), ItemID: o.ItemID, Stream: streamOf(o.DocumentID), RunID: o.RunID,
		OccurredAt: o.OccurredAt.UTC(), ReceivedAt: now.UTC(), RecordedAt: o.OccurredAt.UTC(), CorrelationID: id, Actor: actor, Data: canonData}
	return appjournal.Pending{Entry: e, Envelope: sealed}, rec, nil
}

// replayed — команда с этим command_id уже записана в потоке документа (AD-7).
func (s *Service) replayed(ctx context.Context, documentID, commandID string) (platform.Receipt, bool, error) {
	id := strings.ToLower(commandID)
	if _, err := uuid.Parse(id); err != nil || id == "" {
		return platform.Receipt{}, false, nil
	}
	es, err := s.streamEntries(ctx, documentID)
	if err != nil {
		return platform.Receipt{}, false, err
	}
	for _, e := range es {
		if e.EventID == id {
			t, _ := time.Parse(time.RFC3339Nano, e.RecordedAt)
			return platform.Receipt{CommandID: id, Seq: int64(e.Seq), EventIDs: []string{id}, RecordedAt: t.UTC(), Replayed: true}, true, nil
		}
	}
	return platform.Receipt{}, false, nil
}

// commit — пачка записей одной транзакцией с проверкой AD-39 по потоку документа.
func (s *Service) commit(ctx context.Context, documentID string, meta platform.CommandMeta, batch []appjournal.Pending, at time.Time, basis int64) (platform.Receipt, error) {
	var checks []appjournal.Check
	if b := max(meta.BasisSeq, basis); b > 0 {
		checks = append(checks, appjournal.Check{Stream: streamOf(documentID), BasisSeq: b})
	}
	res, err := s.d.Journal.Append(ctx, appjournal.AppendRequest{Batch: batch, Checks: checks})
	if errors.Is(err, appjournal.ErrDuplicate) {
		if r, ok, rerr := s.replayed(ctx, documentID, batch[0].Entry.EventID); rerr == nil && ok {
			return r, nil
		}
	}
	if err != nil {
		if pe, ok := appjournal.Problem(err); ok {
			return platform.Receipt{}, pe
		}
		return platform.Receipt{}, err
	}
	r := platform.Receipt{CommandID: batch[0].Entry.EventID, RecordedAt: at}
	for _, p := range batch {
		r.EventIDs = append(r.EventIDs, p.Entry.EventID)
	}
	if len(res.Seqs) > 0 {
		r.Seq = res.Seqs[0]
	}
	if !s.cfg.ScenarioClock && !res.Committed.IsZero() {
		r.RecordedAt = res.Committed.UTC()
	}
	return r, nil
}

// withReactions — для документа вне изделия: реакции модуля (версии и
// закрытые маршруты), которых ещё нет в журнале, по свёртке потока с новыми
// записями (AD-12, AD-43: «маршрут закрыт» — только реакция documents).
func (s *Service) withReactions(ctx context.Context, v *view, documentID string, recs []kernel.Record, batch []appjournal.Pending) ([]appjournal.Pending, error) {
	if v.ItemID != "" {
		return batch, nil
	}
	st := dom.FoldStream(append(append([]kernel.Record(nil), v.Input...), recs...), v.Env)
	d := st.Doc(documentID)
	if d == nil {
		return batch, nil
	}
	have := map[kernel.Slot]bool{}
	for _, r := range v.Reactions {
		have[r.Slot] = true
	}
	trigger := recs[len(recs)-1]
	basis := max(v.BasisSeq, 1)
	for _, re := range dom.Reactions(d) {
		if have[re.Slot] {
			continue
		}
		p, err := s.d.Codec.Encode(ctx, engineapp.Out{
			EventID: re.ID(1), Type: re.Type, Kind: catalog.KindReaction, Stream: streamOf(documentID), RunID: trigger.RunID,
			OccurredAt: re.OccurredAt, Correlation: trigger.CorrelationID, Causation: trigger.EventID, BasisSeq: basis, NormRev: re.RuleRev,
			Reaction: &engineapp.ReactionMeta{RuleID: re.Slot.RuleID, RuleRev: re.RuleRev, AutomationMode: 1, Version: 1, Causes: nonNilStrings(re.Causes),
				BasisSeq: basis, Slot: engineapp.SlotMeta{RuleID: re.Slot.RuleID, Subject: re.Slot.Subject, TriggerKey: re.Slot.TriggerKey}},
			Data: re.Data,
		})
		if err != nil {
			return nil, err
		}
		// Реакция ядра без изделия — партиция стадии (как записи вне изделия).
		p.Entry.Partition = s.cfg.Partitions
		batch = append(batch, p)
	}
	return batch, nil
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

// person — псевдоним пользователя сеанса; без сеанса подписать нельзя.
func person(ctx context.Context) (string, error) {
	p := platform.PrincipalFrom(ctx).PersonID
	if p == "" {
		return "", platform.Fail(errcodes.AccessUnauthenticated)
	}
	return p, nil
}

// ── команды ──

// RecordSignature — подпись этапа (documents.signature.record, FR-66, AD-43):
// доменный гард над текущим состоянием документа, запись
// document.signature.recorded; маршрут закрывает реакция модуля.
func (s *Service) RecordSignature(ctx context.Context, documentID string, in RecordSignature) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RecordSignature(ctx, documentID, in)
	}
	method := dom.MethodDemo
	if in.SignatureB64 != "" {
		method = dom.MethodTokenAgent
	}
	return s.sign(ctx, documentID, in.CommandMeta(), signing{Version: in.Version, Stage: in.Stage, Digest: in.DocDigest, Method: method,
		KeyRef: in.KeyRef, Signature: in.SignatureB64})
}

// Sign — прежняя операция подписи (documents.document.sign, волна 1).
func (s *Service) Sign(ctx context.Context, documentID string, in SignDocument) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.Sign(ctx, documentID, in)
	}
	method := dom.MethodDemo
	if in.Signature != nil {
		method = dom.MethodTokenAgent
	}
	return s.sign(ctx, documentID, in.CommandMeta(), signing{Version: in.Version, Stage: in.Stage, Digest: in.DocDigest, Method: method})
}

// AttestPaper — заверение бумажной подписи (documents.paper.attest, FR-139,
// AD-43): подписант поставил подпись ручкой на распечатке с QR; скан
// загружен; заверитель (пользователь сеанса) ≠ подписант; отпечаток из QR
// скана должен совпасть с версией (signing.qr_mismatch).
func (s *Service) AttestPaper(ctx context.Context, documentID string, in AttestPaper) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.AttestPaper(ctx, documentID, in)
	}
	if in.ScanAddress != "" && !digestRe.MatchString(in.ScanAddress) {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "scan_address", "reason", "адрес скана — streebog256:‹64 hex› (materials.material.upload)")
	}
	digest := in.DocDigest
	if id, dg, ok := dom.ParseQR(in.DocDigest); ok {
		if id != documentID {
			return platform.Receipt{}, kernel.Refuse(errcodes.SigningQrMismatch, "qr_digest", dg, "doc_digest", documentID)
		}
		digest = dg
	}
	return s.sign(ctx, documentID, in.CommandMeta(), signing{Version: in.Version, Stage: in.Stage, Digest: digest, Method: dom.MethodPaper,
		Signer: in.SignerPersonID, PaperNo: in.PaperOriginalNo, Scan: in.ScanAddress})
}

// signing — параметры записи подписи.
type signing struct {
	Version   int
	Stage     int
	Digest    string
	Method    string
	KeyRef    string
	Signature string
	// Signer — подписант бумаги (у заверения); пусто — пользователь сеанса.
	Signer  string
	PaperNo string
	Scan    string
}

func (s *Service) sign(ctx context.Context, documentID string, meta platform.CommandMeta, g signing) (platform.Receipt, error) {
	me, err := person(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if r, ok, err := s.replayed(ctx, documentID, meta.CommandID); err != nil || ok {
		return r, err
	}
	v, d, err := s.load(ctx, documentID, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	if g.Digest == "" {
		if cur := d.Current(); cur != nil && cur.No == g.Version {
			g.Digest = cur.Digest
		}
	}
	signer, attester := me, ""
	if g.Method == dom.MethodPaper {
		signer, attester = g.Signer, me
	}
	req := dom.SignRequest{DocumentID: documentID, Version: g.Version, Stage: g.Stage, Digest: g.Digest, Person: signer, Method: g.Method, Level: 2, AttestedBy: attester}
	if err := dom.CheckSign(d, req, v.Env.People, v.State.Participants); err != nil {
		return platform.Receipt{}, err
	}
	cur := d.Current()
	st := cur.Stages[stageIdx(cur, g.Stage)]
	data := map[string]any{"document_id": documentID, "version": g.Version, "doc_digest": g.Digest, "stage": g.Stage, "method": g.Method,
		"signer_person_id": signer, "signature_level": 2, "authority_id": st.AuthorityID}
	if p, ok := v.Env.People.Find(signer); ok && st.StampKind != "" {
		// Номер клейма — object_id контракта (ASCII); номера стартовой политики
		// кириллические — тогда клеймо видно по полномочию, номер не пишется.
		if stamp := p.StampOf(st.StampKind); stamp != "" && objectID.MatchString(stamp) {
			data["stamp_id"] = stamp
		}
	}
	if sum := summaryFields(cur.Content); len(sum) > 0 {
		var fs []map[string]string
		for i, f := range sum {
			if i >= 7 {
				break
			}
			fs = append(fs, map[string]string{"label": clip(f.Label, 128), "value": clip(f.Value, 512)})
		}
		data["summary"] = fs
	}
	if g.KeyRef != "" {
		data["key_ref"] = g.KeyRef
	}
	if g.Signature != "" {
		data["signature"] = g.Signature
	}
	if g.Method == dom.MethodPaper {
		data["attested_by"] = attester
		if g.PaperNo != "" {
			data["paper_original_no"] = g.PaperNo
		}
		if g.Scan != "" {
			data["scan_address"] = g.Scan
		}
	}
	return s.record(ctx, v, documentID, out{Type: catalog.DocumentSignatureRecorded, Data: data, Meta: meta, Actor: me, Level: 2})
}

func stageIdx(v *dom.Version, stage int) int {
	for i, st := range v.Stages {
		if st.Stage == stage {
			return i
		}
	}
	return 0
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// record — одна запись команды в поток документа (+ реакции документа вне изделия).
func (s *Service) record(ctx context.Context, v *view, documentID string, o out) (platform.Receipt, error) {
	now, err := s.now(ctx, v.RunID)
	if err != nil {
		return platform.Receipt{}, err
	}
	o.DocumentID, o.ItemID, o.RunID, o.OccurredAt, o.Basis = documentID, v.ItemID, v.RunID, now, v.BasisSeq
	p, rec, err := s.pending(o)
	if err != nil {
		return platform.Receipt{}, err
	}
	batch, err := s.withReactions(ctx, v, documentID, []kernel.Record{rec}, []appjournal.Pending{p})
	if err != nil {
		return platform.Receipt{}, err
	}
	return s.commit(ctx, documentID, o.Meta, batch, now, v.BasisSeq)
}

// Decline — не согласовать версию, вернуть с замечанием (documents.signature.decline, FR-136).
func (s *Service) Decline(ctx context.Context, documentID string, in DeclineSignature) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.Decline(ctx, documentID, in)
	}
	me, err := person(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if strings.TrimSpace(in.Comment) == "" {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "comment", "reason", "замечание обязательно")
	}
	if r, ok, err := s.replayed(ctx, documentID, in.CommandID); err != nil || ok {
		return r, err
	}
	v, d, err := s.load(ctx, documentID, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	digest := in.DocDigest
	if digest == "" {
		if cur := d.Current(); cur != nil {
			digest = cur.Digest
		}
	}
	if err := dom.CheckDecline(d, dom.SignRequest{DocumentID: documentID, Version: in.Version, Stage: in.Stage, Digest: digest, Person: me}, v.Env.People, v.State.Participants); err != nil {
		return platform.Receipt{}, err
	}
	return s.record(ctx, v, documentID, out{Type: catalog.DocumentSignatureDeclined, Meta: in.CommandMeta(), Actor: me, Level: 2,
		Data: map[string]any{"document_id": documentID, "version": in.Version, "doc_digest": digest, "stage": in.Stage, "signer_person_id": me, "comment": in.Comment}})
}

// SetPaperStatus — статус бумажного экземпляра (documents.paper.status_set, AD-12).
func (s *Service) SetPaperStatus(ctx context.Context, documentID string, in SetPaperStatus) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.SetPaperStatus(ctx, documentID, in)
	}
	me, err := person(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if r, ok, err := s.replayed(ctx, documentID, in.CommandID); err != nil || ok {
		return r, err
	}
	v, d, err := s.load(ctx, documentID, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	if d.Version(in.Version) == nil {
		return platform.Receipt{}, notFound("Версия документа", documentID)
	}
	data := map[string]any{"document_id": documentID, "version": in.Version, "paper_status": in.PaperStatus}
	if in.CopyNo != "" {
		data["copy_no"] = in.CopyNo
	}
	return s.record(ctx, v, documentID, out{Type: catalog.DocumentPaperStatusChanged, Meta: in.CommandMeta(), Actor: me, Data: data})
}

// AnnulVersion — аннулирование версии новой записью (documents.version.annul, AD-12).
func (s *Service) AnnulVersion(ctx context.Context, documentID string, in AnnulVersion) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.AnnulVersion(ctx, documentID, in)
	}
	me, err := person(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if r, ok, err := s.replayed(ctx, documentID, in.CommandID); err != nil || ok {
		return r, err
	}
	v, d, err := s.load(ctx, documentID, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	ver := d.Version(in.Version)
	if ver == nil {
		return platform.Receipt{}, notFound("Версия документа", documentID)
	}
	if ver.Annulled {
		return platform.Receipt{}, kernel.Refuse(errcodes.DocumentStageNotOpen, "doc_id", documentID, "version", itoa(in.Version), "stage", "0", "why", "версия уже аннулирована")
	}
	reason := map[string]any{"text": in.Reason.Text}
	if in.Reason.Code != "" {
		reason["code"] = in.Reason.Code
	}
	return s.record(ctx, v, documentID, out{Type: catalog.DocumentVersionAnnulled, Meta: in.CommandMeta(), Actor: me, Level: 2,
		Data: map[string]any{"document_id": documentID, "version": in.Version, "reason": reason}})
}
