package signing

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/signing"
)

// Сменный рапорт — уровень подписи 3 (FR-66, FR-81, AD-12, AD-14): агент
// токена подписывает корень Меркла отпечатков подписей смены из своего
// локального журнала; сервер строит тот же корень по своему журналу (подписи
// ключами этого человека в окне смены, без бумажных) и сверяет. Расхождение —
// тревога agent_journal_mismatch в шину безопасности. Рапорт пишется в журнал
// записью key.shift_report.recorded: данные сверки и подписанный конверт
// агента как есть (его проверяет и верификатор).

// SubmitShiftReport — принять сменный рапорт (поле signature команды — конверт
// класса shift-report).
type SubmitShiftReport struct {
	platform.CommandHeader
}

// ShiftCheck — итог сверки рапорта.
type ShiftCheck struct {
	Match      bool
	Report     dom.ShiftReport
	Diff       dom.ShiftDiff
	ServerLeaf int
}

// SubmitShiftReport — порт команд: сменный рапорт (итог сверки — в записи журнала).
func (s *Service) SubmitShiftReport(ctx context.Context, in SubmitShiftReport) (platform.Receipt, error) {
	r, _, err := s.ShiftReport(ctx, in)
	return r, err
}

// ShiftReport — проверить и записать сменный рапорт.
func (s *Service) ShiftReport(ctx context.Context, in SubmitShiftReport) (platform.Receipt, ShiftCheck, error) {
	if err := s.ready(); err != nil {
		return platform.Receipt{}, ShiftCheck{}, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	meta := in.CommandMeta()
	env, payload, err := dom.ParseEnvelope(meta.Signature)
	if err != nil || len(env.Signatures) == 0 {
		return platform.Receipt{}, ShiftCheck{}, fail(errcodes.SigningNoSignaturePath, "сменный рапорт — подпись уровня 3 агентом токена")
	}
	if class, _, _ := dom.ParsePayloadType(env.PayloadType); class != dom.ClassShiftReport {
		return platform.Receipt{}, ShiftCheck{}, fail(errcodes.SigningPackageTampered, "ожидался пакет класса shift-report")
	}
	a, err := s.verify(ctx, meta.Signature, payload, dom.ClassShiftReport, actor, false)
	if err != nil {
		return platform.Receipt{}, ShiftCheck{}, err
	}
	var rep dom.ShiftReport
	if err := json.Unmarshal(payload, &rep); err != nil || rep.PersonID != actor {
		return platform.Receipt{}, ShiftCheck{}, fail(errcodes.SigningForeignKey, "рапорт не того сотрудника", "key_ref", firstRef(a.Verdict))
	}
	from, err1 := time.Parse(TimeLayout, rep.WindowFrom)
	to, err2 := time.Parse(TimeLayout, rep.WindowTo)
	if err1 != nil || err2 != nil || !from.Before(to) {
		return platform.Receipt{}, ShiftCheck{}, fail(errcodes.ApiValidationFailed, "окно смены", "field", "window", "reason", "window_from < window_to")
	}
	leaves, err := s.ShiftLeaves(ctx, actor, from, to)
	if err != nil {
		return platform.Receipt{}, ShiftCheck{}, err
	}
	diff, cmpErr := dom.CompareShift(rep, leaves)
	chk := ShiftCheck{Match: cmpErr == nil, Report: rep, Diff: diff, ServerLeaf: len(leaves)}
	id, _ := commandID(meta.CommandID)
	data := map[string]any{"person_id": rep.PersonID, "shift_id": rep.ShiftID, "window_from": rep.WindowFrom, "window_to": rep.WindowTo,
		"merkle_root": rep.MerkleRoot, "leaf_count": rep.LeafCount, "server_merkle_root": diff.RootServer, "server_leaf_count": diff.CountSrv,
		"matches": chk.Match, "signed_report": json.RawMessage(meta.Signature)}
	if rep.WorkplaceID != "" {
		data["workplace_id"] = rep.WorkplaceID
	}
	if len(diff.Types) > 0 {
		data["mismatch_types"] = diff.Types
	}
	raw, _ := json.Marshal(data)
	a.Payload, a.Envelope = s.serverEvent(catalog.KeyShiftReportRecorded, id, raw, a.KeyRefs, rep.CryptoProfile)
	var alerts []dom.KeyAlert
	if !chk.Match {
		alerts = append(alerts, dom.KeyAlert{Alert: "agent_journal_mismatch", KeyRef: firstRef(a.Verdict), Person: actor, Detail: cmpErr.Error()})
	}
	r, err := s.write(ctx, catalog.KeyShiftReportRecorded, "key:shift:"+actor, id, a, meta, alerts)
	return r, chk, err
}

// serverEvent — событие-запись сервера: конверт без подписей сервера, а
// подпись человека — внутри data (signed_report), её проверяет верификатор.
func (s *Service) serverEvent(t catalog.Type, id string, data json.RawMessage, signers []string, profile string) ([]byte, []byte) {
	info, _ := catalog.Lookup(t)
	if len(signers) == 0 {
		signers = []string{"anonymous@1"}
	}
	if profile == "" {
		profile = dom.ProfileGost
	}
	ev := map[string]any{"event_id": id, "event_type": string(t), "schema_version": info.CurrentVersion, "source_id": SourceAPI,
		"source_kind": "manual_entry", "occurred_at": s.now().Format(TimeLayout), "correlation_id": id, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": profile, "signers": signers}, "data": data}
	c, _ := dom.CanonicalOf(ev)
	return c, dom.Seal(dom.PayloadType(dom.ClassEvent, 1), c).Marshal()
}

// ShiftLeaves — подписи сотрудника в окне смены по журналу сервера: для
// каждой записи с подписью его ключом — отпечаток подписи (лист), тип и seq.
// Бумажные подписи (класс paper) в рапорт не входят.
func (s *Service) ShiftLeaves(ctx context.Context, person string, from, to time.Time) ([]dom.ShiftLeaf, error) {
	keys, _, _, err := s.d.Registry.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	var mine []string
	for _, k := range keys.Keys() {
		if k.SubjectID == person && (k.SubjectKind == dom.SubjectPerson || k.SubjectKind == dom.SubjectDemoPersona) {
			mine = append(mine, k.KeyRef)
		}
	}
	var out []dom.ShiftLeaf
	if len(mine) == 0 {
		return out, nil
	}
	after := int64(0)
	for {
		es, err := s.d.Journal.Read(ctx, journal.ReadQuery{AfterSeq: after, Limit: 1000})
		if err != nil {
			return nil, err
		}
		for _, e := range es {
			at, _ := time.Parse(TimeLayout, e.CommittedAt)
			if at.Before(from) || !at.Before(to) || e.ProvenanceClass == "paper" || e.EventType == string(catalog.KeyShiftReportRecorded) {
				continue
			}
			env, err := s.d.Journal.Open(ctx, e)
			if err != nil {
				continue
			}
			// Подписи записи или подписанного пакета команды рядом с ней (Д-59).
			pt, payload, sigs, _ := dom.RecordSignatures(env.Raw)
			for _, sg := range sigs {
				if slices.Contains(mine, sg.KeyID) {
					out = append(out, dom.ShiftLeaf{Digest: dom.SignatureDigest(pt, payload, sg), EventType: e.EventType, Seq: int64(e.Seq)})
				}
			}
		}
		if len(es) < 1000 {
			break
		}
		after = int64(es[len(es)-1].Seq)
	}
	return out, nil
}

// Doubts — защитная реакция на компрометацию ключа (AD-11): решения,
// подписанные ключом после «скомпрометирован с X» и до отзыва. Исполняют
// nonconformity (сдерживание «подпись под сомнением») и notifications
// (задачи на переподписание) — по этому списку.
func (s *Service) Doubts(ctx context.Context, keyRef string) ([]dom.Doubt, error) {
	if err := s.readyRead(); err != nil {
		return nil, err
	}
	keys, _, _, err := s.d.Registry.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	k, ok := keys.Key(keyRef)
	if !ok {
		return nil, fail(errcodes.ApiNotFound, "ключ "+keyRef+" не зарегистрирован", "object", "ключ", "id", keyRef)
	}
	var ds []dom.SignedDecision
	after := int64(0)
	for {
		es, err := s.d.Journal.Read(ctx, journal.ReadQuery{AfterSeq: after, Limit: 1000})
		if err != nil {
			return nil, err
		}
		for _, e := range es {
			if e.EntryKind != "decision" {
				continue
			}
			env, err := s.d.Journal.Open(ctx, e)
			if err != nil {
				continue
			}
			_, _, sigs, _ := dom.RecordSignatures(env.Raw)
			d := dom.SignedDecision{EventID: e.EventID, EventType: e.EventType, Seq: int64(e.Seq)}
			d.CommittedAt, _ = time.Parse(TimeLayout, e.CommittedAt)
			if e.ItemID != nil {
				d.ItemID = *e.ItemID
			}
			for _, sg := range sigs {
				d.KeyRefs = append(d.KeyRefs, sg.KeyID)
			}
			ds = append(ds, d)
		}
		if len(es) < 1000 {
			break
		}
		after = int64(es[len(es)-1].Seq)
	}
	return dom.ProtectiveReaction(k, ds), nil
}
