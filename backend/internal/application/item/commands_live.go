package item

import (
	"context"
	"slices"
	"strings"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/item"
	"ant/internal/domain/kernel"
)

// Команды модуля item в режиме live (AD-39): гард модуля-владельца над
// свёрткой изделия на текущий момент (проекции — только кэш), межизделийные
// предусловия — по генеалогии стадии; затем запись одной пачкой с проверкой
// потока изделия. Классы (AD-27): закрытие вмешательства и подтверждение
// идентификации — разрешающие, критические (AD-28); вмешательство — защитное.

// write — записать факты или решения модуля item.
func (s *Service) write(ctx context.Context, meta platform.CommandMeta, level int, recs ...Record) (platform.Receipt, error) {
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	for i := range recs {
		recs[i].Meta, recs[i].Actor, recs[i].OccurredAt, recs[i].SignatureLevel = meta, actor, now, level
		if recs[i].GuardStreams == nil && recs[i].Stream != "" {
			recs[i].GuardStreams = []string{recs[i].Stream}
		}
	}
	return s.cfg.Writer.Write(ctx, dom.Module, recs)
}

func itemRecord(t catalog.Type, itemID string, data any) Record {
	return Record{Type: t, Stream: "item:" + itemID, ItemID: itemID, Data: data}
}

func reasonData(text string) map[string]any { return map[string]any{"text": text} }

// newItemID — ID нового изделия (AD-16): код предприятия и локальный номер;
// без номера — из command_id (детерминированно: повтор команды — тот же ID).
func (s *Service) newItemID(local, commandID string) string {
	if local != "" {
		return s.cfg.Enterprise + ":" + local
	}
	u := kernel.UUIDv5(constants.NsAnt, "item\x1f"+strings.ToLower(commandID))
	return s.cfg.Enterprise + ":I-" + strings.ToUpper(u[:8])
}

// registered — data item.item.registered: версия нормативного слоя закрепляется
// при запуске (AD-17).
func (s *Service) registered(id string, in RegisterItem, splitFrom string) map[string]any {
	d := map[string]any{"item_id": id, "item_type_id": in.ItemTypeID, "item_revision": in.ItemRevision,
		"process_version_hash": s.cfg.processVersion(), "normative_rev": s.cfg.normativeRev()}
	if in.OrderID != "" {
		d["order_id"] = in.OrderID
	}
	if len(in.LotIDs) > 0 {
		d["lot_ids"] = in.LotIDs
	}
	if in.EntryStepKey != "" {
		d["entry_step_key"] = in.EntryStepKey
	}
	if in.IsAssembly {
		d["is_assembly"] = true
	}
	if splitFrom != "" {
		d["split_from"] = splitFrom
	}
	return d
}

// Register — зарегистрировать изделие и запустить в работу (item.item.registered, FR-42, AD-16).
func (s *Service) Register(ctx context.Context, in RegisterItem) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.Register(ctx, in)
	}
	id := s.newItemID(in.LocalID, in.CommandID)
	if run := appjournal.RunFrom(ctx); run != "" && in.LocalID == "" {
		// AD-38: локальный ID изделия прогона несёт префикс прогона (эпик 16).
		_, local, _ := strings.Cut(id, ":")
		id = s.cfg.Enterprise + ":" + run + "/" + local
	}
	if _, err := s.guard(ctx, id, "item.item.register", in.CommandMeta(), nil); err != nil {
		return platform.Receipt{}, err
	}
	if in.ItemRevision == "" {
		in.ItemRevision = "-"
	}
	return s.write(ctx, in.CommandMeta(), 1, itemRecord(catalog.ItemItemRegistered, id, s.registered(id, in, "")))
}

// Split — разделение 1→N (FR-15): части регистрируются с split_from одной
// пачкой; перенос происхождения делает межизделийная стадия.
func (s *Service) Split(ctx context.Context, itemID string, in SplitItem) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.Split(ctx, itemID, in)
	}
	a, err := s.guard(ctx, itemID, "item.item.split", in.CommandMeta(), nil)
	if err != nil {
		return platform.Receipt{}, err
	}
	parent := a.Snap.Item
	recs := make([]Record, 0, len(in.Parts))
	seen := map[string]bool{}
	for i, part := range in.Parts {
		local := part.ItemID
		if local == "" {
			_, l, _ := strings.Cut(itemID, ":")
			local = l + "-" + itoa(i+1)
		} else if _, l, ok := strings.Cut(local, ":"); ok {
			local = l
		}
		id := s.cfg.Enterprise + ":" + local
		if seen[id] || id == itemID {
			return platform.Receipt{}, platform.Fail(errcodes.ItemAlreadyRegistered, "item_id", id)
		}
		seen[id] = true
		if _, err := s.guard(ctx, id, "item.item.register", in.CommandMeta(), nil); err != nil {
			return platform.Receipt{}, err
		}
		reg := RegisterItem{ItemTypeID: part.ItemTypeID, ItemRevision: part.ItemRevision, OrderID: parent.OrderID, LotIDs: part.LotIDs}
		if reg.ItemTypeID == "" {
			reg.ItemTypeID = parent.ItemTypeID
		}
		if reg.ItemRevision == "" {
			reg.ItemRevision = parent.ItemRevision
		}
		r := itemRecord(catalog.ItemItemRegistered, id, s.registered(id, reg, itemID))
		r.GuardStreams = []string{"item:" + itemID}
		recs = append(recs, r)
	}
	return s.write(ctx, in.CommandMeta(), 1, recs...)
}

func itoa(i int) string {
	const d = "0123456789"
	if i < 10 {
		return d[i : i+1]
	}
	return itoa(i/10) + d[i%10:i%10+1]
}

// ApplyCarrier — нанести носитель (item.carrier.applied, AD-16): носитель,
// действующий у другого изделия, — отказ item.carrier_in_use.
func (s *Service) ApplyCarrier(ctx context.Context, itemID string, in ApplyCarrier) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ApplyCarrier(ctx, itemID, in)
	}
	if _, err := s.guard(ctx, itemID, "item.carrier.apply", in.CommandMeta(), dom.CarrierCmd{Type: in.CarrierType, Value: in.Value, Replaces: in.ReplacesValue}); err != nil {
		return platform.Receipt{}, err
	}
	st, err := s.stage(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	// Ячейка тары может держать несколько изделий (партионный крепёж) — это
	// неоднозначность привязки, а не ошибка (FR-34); остальные носители уникальны.
	if other := st.Own.Genealogy.ActiveCarrier(in.CarrierType + ":" + in.Value); other != "" && other != itemID && in.CarrierType != "container_cell" {
		return platform.Receipt{}, platform.Fail(errcodes.ItemCarrierInUse, "carrier_ref", in.CarrierType+":"+in.Value, "other_item_id", other)
	}
	d := map[string]any{"carrier_type": in.CarrierType, "value": in.Value, "is_temporary": in.IsTemporary}
	if in.ZoneID != "" {
		d["zone_id"] = in.ZoneID
	}
	if in.ReplacesValue != "" {
		d["replaces_value"] = in.ReplacesValue
	}
	return s.write(ctx, in.CommandMeta(), 1, itemRecord(catalog.ItemCarrierApplied, itemID, d))
}

// RemoveCarrier — снять носитель (item.carrier.removed).
func (s *Service) RemoveCarrier(ctx context.Context, itemID string, in RemoveCarrier) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RemoveCarrier(ctx, itemID, in)
	}
	if _, err := s.guard(ctx, itemID, "item.carrier.remove", in.CommandMeta(), dom.CarrierCmd{Type: in.CarrierType, Value: in.Value}); err != nil {
		return platform.Receipt{}, err
	}
	d := map[string]any{"carrier_type": in.CarrierType, "value": in.Value}
	if in.ReasonText != "" {
		d["reason"] = reasonData(in.ReasonText)
	}
	return s.write(ctx, in.CommandMeta(), 1, itemRecord(catalog.ItemCarrierRemoved, itemID, d))
}

// RecordPresentation — предъявить изделие (item.presentation.recorded, FR-19).
func (s *Service) RecordPresentation(ctx context.Context, itemID string, in RecordPresentation) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RecordPresentation(ctx, itemID, in)
	}
	if _, err := s.guard(ctx, itemID, "item.presentation.record", in.CommandMeta(), nil); err != nil {
		return platform.Receipt{}, err
	}
	by := personOf(ctx)
	return s.write(ctx, in.CommandMeta(), 1, itemRecord(catalog.ItemPresentationRecorded, itemID, map[string]any{"step_key": in.StepKey,
		"presentation_no": in.PresentationNo, "presented_to": in.PresentedTo, "presented_by": by}))
}

// personOf — псевдоним автора команды (person_ref).
func personOf(ctx context.Context) string {
	if p := platform.PrincipalFrom(ctx).PersonID; p != "" {
		return p
	}
	return "anonymous"
}

// OpenIntervention — открыть вмешательство (item.intervention.opened, FR-21):
// зоны теряют статус проверенных до повторного контроля.
func (s *Service) OpenIntervention(ctx context.Context, itemID string, in OpenIntervention) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.OpenIntervention(ctx, itemID, in)
	}
	a, err := s.guard(ctx, itemID, "item.intervention.open", in.CommandMeta(), dom.InterventionCmd{Zones: in.ZoneIDs})
	if err != nil {
		return platform.Receipt{}, err
	}
	id := "IV-" + itoa(len(a.Snap.Item.Interventions)+1)
	d := map[string]any{"intervention_id": id, "zone_ids": in.ZoneIDs, "purpose": reasonData(in.Purpose)}
	if len(in.RemovedComponents) > 0 {
		d["removed_components"] = in.RemovedComponents
	}
	return s.write(ctx, in.CommandMeta(), 1, itemRecord(catalog.ItemInterventionOpened, itemID, d))
}

// CloseIntervention — закрыть вмешательство после повторной проверки зоны
// (item.intervention.closed; разрешающее, критическое).
func (s *Service) CloseIntervention(ctx context.Context, itemID, interventionID string, in CloseIntervention) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.CloseIntervention(ctx, itemID, interventionID, in)
	}
	if _, err := s.guard(ctx, itemID, "item.intervention.close", in.CommandMeta(), dom.InterventionCmd{ID: interventionID}); err != nil {
		return platform.Receipt{}, err
	}
	d := map[string]any{"intervention_id": interventionID, "recheck_event_ids": in.RecheckEventIDs}
	if in.RetestRequired {
		d["retest_required"] = true
	}
	return s.write(ctx, in.CommandMeta(), 2, itemRecord(catalog.ItemInterventionClosed, itemID, d))
}

// ConfirmIdentification — повторная идентификация человеком с подписью
// (item.identification.confirmed, AD-16): снимает изоляцию по этой причине.
func (s *Service) ConfirmIdentification(ctx context.Context, itemID string, in ConfirmIdentification) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ConfirmIdentification(ctx, itemID, in)
	}
	if _, err := s.guard(ctx, itemID, "item.identification.confirm", in.CommandMeta(), dom.ConfirmCmd{QuestionedEventID: in.QuestionedEventID}); err != nil {
		return platform.Receipt{}, err
	}
	d := map[string]any{"method": in.Method, "questioned_event_id": in.QuestionedEventID}
	if in.CarrierValue != "" {
		d["carrier_value"] = in.CarrierValue
	}
	return s.write(ctx, in.CommandMeta(), 2, itemRecord(catalog.ItemIdentificationConfirmed, itemID, d))
}

// RecordAssembly — установить компонент в сборку (item.assembly.recorded,
// FR-45): связь генеалогии пишет стадия. Компонент в другой сборке и цикл —
// отказ по генеалогии стадии.
func (s *Service) RecordAssembly(ctx context.Context, itemID string, in RecordAssembly) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RecordAssembly(ctx, itemID, in)
	}
	if _, err := s.guard(ctx, itemID, "item.assembly.record", in.CommandMeta(), dom.AssemblyCmd{ComponentItemID: in.ComponentItemID, ComponentLotID: in.ComponentLotID}); err != nil {
		return platform.Receipt{}, err
	}
	if in.ComponentItemID == "" && in.ComponentLotID == "" {
		return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "component_item_id", "reason", "нужен компонент-экземпляр или партия")
	}
	st, err := s.stage(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	g := st.Own.Genealogy
	if c := in.ComponentItemID; c != "" {
		if p := g.Items[c].Parent; p != "" && p != itemID {
			return platform.Receipt{}, platform.Fail(errcodes.ItemComponentAlreadyAssembled, "component_item_id", c, "assembly_item_id", p)
		}
		if slices.Contains(g.Descendants(c), itemID) || slices.Contains(g.Ancestors(itemID), c) {
			return platform.Receipt{}, platform.Fail(errcodes.ItemAssemblyCycle, "component_item_id", c, "item_id", itemID)
		}
	}
	d := map[string]any{"assembly_item_id": itemID, "component_type_id": in.ComponentTypeID, "binding_method": in.BindingMethod}
	for k, v := range map[string]string{"component_item_id": in.ComponentItemID, "component_lot_id": in.ComponentLotID, "position": in.Position, "operation_run_id": in.OperationRunID} {
		if v != "" {
			d[k] = v
		}
	}
	if in.Quantity > 0 {
		d["quantity"] = in.Quantity
	}
	r := itemRecord(catalog.ItemAssemblyRecorded, itemID, d)
	if in.ComponentItemID != "" {
		r.GuardStreams = []string{"item:" + itemID, "item:" + in.ComponentItemID}
	}
	return s.write(ctx, in.CommandMeta(), 1, r)
}

// RecordRelease — принять изделие на склад выпуска (item.release.recorded).
func (s *Service) RecordRelease(ctx context.Context, itemID string, in RecordRelease) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RecordRelease(ctx, itemID, in)
	}
	if _, err := s.guard(ctx, itemID, "item.release.record", in.CommandMeta(), nil); err != nil {
		return platform.Receipt{}, err
	}
	d := map[string]any{"warehouse_id": in.WarehouseID, "after_rework": in.AfterRework, "received_by": personOf(ctx)}
	if in.ConcessionID != "" {
		d["concession_id"] = in.ConcessionID
	}
	return s.write(ctx, in.CommandMeta(), 1, itemRecord(catalog.ItemReleaseRecorded, itemID, d))
}

// processVersion, normativeRev — закреплённая версия для новых изделий (AD-17).
func (c Config) processVersion() string {
	if c.ProcessVersion != "" {
		return c.ProcessVersion
	}
	return "streebog256:" + strings.Repeat("0", 64)
}

func (c Config) normativeRev() string {
	if c.NormativeRev != "" {
		return c.NormativeRev
	}
	return "normative-seed-v1"
}
