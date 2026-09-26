package main

import (
	"context"
	"strings"

	ingestapp "ant/internal/application/ingest"
	itemapp "ant/internal/application/item"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	processapp "ant/internal/application/process"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Команды модуля item — ОБХОД до эпика 18 (живой срез эпика 16).
//
// TODO(18): при слиянии эпика 18 этот файл удаляется, а в buildAPI live-реализация
// item — сервис эпика 18. До него живой путь UJ-6 (задание → изделие →
// операции → сигнал камеры → несоответствие) не начинался: изделие рождает
// только item.item.register. Здесь — регистрация, носители, предъявление,
// сборка, выпуск и вмешательство фактами через ручной ввод приёма (схема,
// дубли, журнал — как команды исполнителя process), без гардов item и без
// чтений (они — 501 до эпика 18).

// enterpriseCode — код предприятия в item_id (AD-16; мир сценариев — ENT01).
const enterpriseCode = "ENT01"

// itemBypass — live-команды item поверх ручного ввода приёма.
type itemBypass struct {
	itemapp.Unimplemented
	facts ingestFacts
	// active — действующая версия процесса: хеш и ревизия для закрепления при
	// регистрации (AD-17).
	active func(ctx context.Context) (processapp.VersionRecord, error)
}

// newItemID — item_id нового изделия: `ENT01:[‹прогон›/]I-‹8 hex›` от
// command_id (повтор команды — тот же id, AD-7).
func newItemID(ctx context.Context, commandID string) string {
	u := kernel.UUIDv5(constants.NsAnt, "item|"+commandID)
	local := "I-" + strings.ToUpper(u[:8])
	if run := appjournal.RunFrom(ctx); run != "" {
		local = run + "/" + local
	}
	return enterpriseCode + ":" + local
}

func (b itemBypass) fact(ctx context.Context, meta platform.CommandMeta, t catalog.Type, itemID string, data map[string]any) (platform.Receipt, error) {
	return b.facts.Submit(ctx, processapp.Fact{Meta: meta, EventType: t, ItemID: itemID, Data: data})
}

func actorOf(ctx context.Context) string { return platform.PrincipalFrom(ctx).PersonID }

// Register — зарегистрировать изделие (item.item.registered): закреплённая
// версия процесса — действующая (AD-17).
func (b itemBypass) Register(ctx context.Context, in itemapp.RegisterItem) (platform.Receipt, error) {
	v, err := b.active(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if v.Hash == "" {
		return platform.Receipt{}, platform.Fail(errcodes.ProcessVersionUnknown, "hash", "действующей версии нет")
	}
	id := newItemID(ctx, in.CommandID)
	rev := in.ItemRevision
	if rev == "" {
		rev = "-"
	}
	data := map[string]any{"item_id": id, "item_type_id": in.ItemTypeID, "item_revision": rev,
		"process_version_hash": v.Hash, "normative_rev": v.ID}
	if in.OrderID != "" {
		data["order_id"] = in.OrderID
	}
	if len(in.LotIDs) > 0 {
		data["lot_ids"] = in.LotIDs
	}
	if in.EntryStepKey != "" {
		data["entry_step_key"] = in.EntryStepKey
	}
	if in.IsAssembly {
		data["is_assembly"] = true
	}
	return b.fact(ctx, in.CommandMeta(), catalog.ItemItemRegistered, id, data)
}

// ApplyCarrier — нанести носитель (item.carrier.applied).
func (b itemBypass) ApplyCarrier(ctx context.Context, itemID string, in itemapp.ApplyCarrier) (platform.Receipt, error) {
	data := map[string]any{"carrier_type": in.CarrierType, "value": in.Value, "is_temporary": in.IsTemporary}
	if in.ZoneID != "" {
		data["zone_id"] = in.ZoneID
	}
	if in.ReplacesValue != "" {
		data["replaces_value"] = in.ReplacesValue
	}
	return b.fact(ctx, in.CommandMeta(), catalog.ItemCarrierApplied, itemID, data)
}

// RecordPresentation — предъявить на точке (item.presentation.recorded).
func (b itemBypass) RecordPresentation(ctx context.Context, itemID string, in itemapp.RecordPresentation) (platform.Receipt, error) {
	return b.fact(ctx, in.CommandMeta(), catalog.ItemPresentationRecorded, itemID, map[string]any{"step_key": in.StepKey,
		"presentation_no": in.PresentationNo, "presented_to": in.PresentedTo, "presented_by": actorOf(ctx)})
}

// RecordAssembly — сборка (item.assembly.recorded).
func (b itemBypass) RecordAssembly(ctx context.Context, itemID string, in itemapp.RecordAssembly) (platform.Receipt, error) {
	data := map[string]any{"assembly_item_id": itemID, "component_type_id": in.ComponentTypeID, "binding_method": in.BindingMethod}
	for k, v := range map[string]string{"component_item_id": in.ComponentItemID, "component_lot_id": in.ComponentLotID,
		"position": in.Position, "operation_run_id": in.OperationRunID} {
		if v != "" {
			data[k] = v
		}
	}
	if in.Quantity > 0 {
		data["quantity"] = in.Quantity
	}
	return b.fact(ctx, in.CommandMeta(), catalog.ItemAssemblyRecorded, itemID, data)
}

// RecordRelease — принять на склад выпуска (item.release.recorded).
func (b itemBypass) RecordRelease(ctx context.Context, itemID string, in itemapp.RecordRelease) (platform.Receipt, error) {
	data := map[string]any{"warehouse_id": in.WarehouseID, "after_rework": in.AfterRework, "received_by": actorOf(ctx)}
	if in.ConcessionID != "" {
		data["concession_id"] = in.ConcessionID
	}
	return b.fact(ctx, in.CommandMeta(), catalog.ItemReleaseRecorded, itemID, data)
}

// OpenIntervention — открыть вмешательство (item.intervention.opened).
func (b itemBypass) OpenIntervention(ctx context.Context, itemID string, in itemapp.OpenIntervention) (platform.Receipt, error) {
	id := "IV-" + strings.ToUpper(kernel.UUIDv5(constants.NsAnt, "intervention|"+in.CommandID)[:8])
	data := map[string]any{"intervention_id": id, "zone_ids": in.ZoneIDs, "purpose": map[string]any{"text": in.Purpose}}
	if len(in.RemovedComponents) > 0 {
		data["removed_components"] = in.RemovedComponents
	}
	return b.fact(ctx, in.CommandMeta(), catalog.ItemInterventionOpened, itemID, data)
}

var _ itemapp.Commands = itemBypass{}

// itemLive — команды item для роли api: обход до эпика 18 над ручным вводом
// приёма; без живого приёма — nil (501).
func itemLive(ctx context.Context, env *environment, ingest *ingestapp.Service) (itemapp.Commands, error) {
	if ingest == nil {
		return nil, nil
	}
	c, err := env.readyCore(ctx)
	if err != nil {
		return nil, err
	}
	return itemBypass{facts: ingestFacts{ingest}, active: func(ctx context.Context) (processapp.VersionRecord, error) {
		vs, err := c.versions.List(ctx)
		if err != nil {
			return processapp.VersionRecord{}, err
		}
		v, _ := processapp.Active(vs)
		return v, nil
	}}, nil
}
