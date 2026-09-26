package ingest

import (
	"context"
	"encoding/json"
	"strings"
	"uuid"

	"ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// ManualNote — пометка ручного ввода под сеансом без подписи (FR-141, AD-13
// уровень 0): источник — человек, а не станок (FR-140).
const ManualNote = "ручной ввод под сеансом, без подписи"

// SubmitManual — FR-141: ручной ввод — полноправный источник событий с
// пометкой источника (source_kind = manual_entry, класс personal), а не
// заглушка; проходит тот же конвейер: схема, дубли (event_id = command_id
// клиента — повтор формы не удваивает факт), привязка к изделию.
// «Оператор отметил конец операции в 12:03» хранится как ручной ввод, а не как
// данные станка (FR-140).
func (s *Service) SubmitManual(ctx context.Context, cmd Cmd[ManualInput]) (Result, error) {
	if err := s.ready(); err != nil {
		return Result{}, platform.NotImplemented("ingest.manual.submit")
	}
	in := cmd.Body
	p := platform.PrincipalFrom(ctx)
	src := in.SourceID
	if src == "" {
		who := p.PersonID
		if who == "" {
			who = "anonymous"
		}
		src = "manual:" + who
	}
	id := cmd.Meta.CommandID
	if !isUUID(id) {
		id = uuid.NewV7().String()
	}
	now, err := s.deps.DomainClock.Now(ctx)
	if err != nil {
		return Result{}, err
	}
	occ := now
	if in.OccurredAt != nil {
		occ = *in.OccurredAt
	}
	ver := in.SchemaVersion
	if ver == 0 {
		ver = 1
	}
	signer := "person-" + strings.ToLower(p.PersonID) + "@1"
	if p.PersonID == "" {
		signer = "person-anonymous@1"
	}
	ev := map[string]any{
		"event_id": id, "event_type": in.EventType, "schema_version": ver, "source_id": src,
		"source_kind": "manual_entry", "reliability": "medium", "occurred_at": ts(occ),
		"correlation_id": id, "causation_id": nil,
		"integrity": map[string]any{"format_version": 1, "crypto_profile": "gost", "signers": []string{signer}},
		"data":      in.Data,
	}
	if in.Data == nil {
		ev["data"] = map[string]any{}
	}
	if in.ItemID != "" {
		ev["item_id"] = in.ItemID
	}
	// Прогон сценария (AD-38): команда демо-подписанта идёт с прогоном в
	// контексте — факт принадлежит прогону (эпик 16).
	if run := journal.RunFrom(ctx); run != "" {
		ev["run_id"] = run
	} else if run := runOfItem(in.ItemID); run != "" {
		// Действие человека со стола по изделию прогона (команда без прогона
		// в контексте): факт — того же прогона, что и изделие (AD-38).
		ev["run_id"] = run
	}
	if in.CarrierType != "" {
		ev["item_ref"] = map[string]any{"carrier_type": in.CarrierType, "value": in.CarrierValue, "identification_level": "probable"}
	}
	if in.Restored {
		// Сбой №11 каталога: внесено с бумаги задним числом — отдельный признак.
		ev["entry_mode"] = "restored_from_paper"
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		e := platform.Fail(errcodes.ApiValidationFailed, "field", "data", "reason", err.Error())
		return Result{}, e
	}
	mc := msgCtx{Manual: &manualAuth{Provenance: "personal", Note: ManualNote}}
	if len(cmd.Meta.Signature) > 0 {
		// Подписанная форма (уровень ≥ 1): пакет DSSE проходит обычную проверку подписи.
		raw, mc = cmd.Meta.Signature, msgCtx{}
	}
	return s.process(ctx, raw, mc)
}

// runOfItem — прогон изделия по его id (‹ENT›:‹run_id›/‹локальный номер›,
// AD-38); изделие вне прогона — пусто.
func runOfItem(itemID string) string {
	_, local, ok := strings.Cut(itemID, ":")
	if !ok {
		return ""
	}
	i := strings.LastIndexByte(local, '/')
	if i <= 0 {
		return ""
	}
	return local[:i]
}
