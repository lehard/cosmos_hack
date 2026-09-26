// Образцы данных паспорта — только в тестах (PRD §11.10: заготовок в коде
// интерфейса нет). Сценарий «плохой день сварочного участка»: фланец FL-0042
// после сварки с признаком дефекта на КТ-3 (FR-38), ручная отметка конца
// операции (FR-140), черновик карточки и блок по правилу (FR-49).
import type { ItemGenealogy, ItemHistory, ItemPassport, PassportEntry } from '@/entities/item'

/** Время 23.09.2026, UTC. */
export const at = (hhmm: string) => `2026-09-23T${hhmm}:00.000Z`

const deviceSig = { check: 'valid', class: 'device', level: 0, signer_id: 'cam-kt3' } as const
const serverSig = { check: 'valid', class: 'server_attested', level: 0, signer_id: 'ant' } as const

export const flangeEntries = (): PassportEntry[] => [
  {
    event_id: 'e-reg',
    event_type: 'item.item.registered',
    kind: 'decision',
    occurred_at: at('06:10'),
    recorded_at: at('06:10'),
    seq: 1001,
    author: 'master-07',
    summary: 'Изделие запущено в работу',
    signatures: [{ check: 'valid', class: 'personal', level: 1, signer_id: 'master-07' }],
  },
  {
    event_id: 'e-weld-start',
    event_type: 'operation.run.started',
    kind: 'fact',
    occurred_at: at('08:10'),
    recorded_at: at('08:10'),
    seq: 1210,
    source_kind: 'machine',
    reliability: 'high',
    summary: 'Сварка начата на ИС-3',
    signatures: [{ check: 'valid', class: 'device', level: 0, signer_id: 'is-3' }],
  },
  {
    // FR-140: оператор отметил конец операции вручную — это ручной ввод, не данные станка.
    event_id: 'e-weld-end',
    event_type: 'operation.run.finished',
    kind: 'fact',
    occurred_at: at('08:40'),
    recorded_at: at('08:41'),
    seq: 1231,
    source_kind: 'manual_entry',
    reliability: 'medium',
    author: 'op-12',
    summary: 'Исполнитель отметил конец сварки в 08:40',
    signatures: [{ check: 'unchecked', class: 'personal', level: 1, signer_id: 'op-12' }],
  },
  {
    event_id: 'e-kt3',
    event_type: 'inspection.result.recorded',
    kind: 'fact',
    occurred_at: at('08:52'),
    recorded_at: at('08:52'),
    seq: 1240,
    source_kind: 'camera',
    reliability: 'high',
    summary: 'КТ-3, камера: обнаружен признак дефекта',
    signatures: [deviceSig],
  },
  {
    event_id: 'e-signal',
    event_type: 'quality.signal.raised',
    kind: 'reaction',
    occurred_at: at('08:52'),
    recorded_at: at('08:53'),
    seq: 1241,
    summary: 'Сигнал о признаке дефекта: пора в шве',
    signatures: [serverSig],
  },
  {
    event_id: 'e-hold',
    event_type: 'decision.containment.applied',
    kind: 'reaction',
    occurred_at: at('08:52'),
    recorded_at: at('08:53'),
    seq: 1242,
    ca_ref: 'CA-311',
    summary: 'Изделие заблокировано по правилу RM-weld-7',
    signatures: [serverSig],
  },
  {
    event_id: 'e-fix',
    event_type: 'operation.run.finished',
    kind: 'fact',
    occurred_at: at('08:40'),
    recorded_at: at('09:05'),
    seq: 1250,
    source_kind: 'manual_entry',
    author: 'master-07',
    corrects: 'e-weld-end',
    summary: 'Исправлено время конца сварки: 08:38',
    signatures: [{ check: 'not_verifiable', class: 'paper', level: 2, signer_id: 'master-07', attested_by: 'qc-03' }],
  },
]

export const flangePassport = (over: Partial<ItemPassport> = {}): ItemPassport => ({
  basis_seq: 1250,
  item_id: 'ENT:FL-0042',
  label: 'FL-0042',
  item_type_id: 'ФЛ-100.02',
  item_revision: 'КД rev.C',
  process_version: 'streebog256:9f1c0000',
  order_id: 'ORD-77',
  identification: 'unique',
  status: {
    position: 'isolated',
    quality: 'signal',
    disposition: 'none',
    containment: 'item_hold',
    erp_accounting: 'accepted_into_work',
    summary: 'hold',
  },
  carriers: [
    { carrier_type: 'tag_qr', value: 'TAG-7788', temporary: true, state: 'removed', applied_at: at('06:12'), removed_at: at('07:30') },
    { carrier_type: 'dpm_datamatrix', value: 'DM-FL-0042', temporary: false, state: 'verified', applied_at: at('07:30') },
  ],
  zones: [
    { zone_id: 'Z-weld-1', title: 'Шов 1', closed: false },
    { zone_id: 'Z-bore', title: 'Отверстие Ø12', closed: true, closed_by: 'assembly' },
    { zone_id: 'Z-seal', title: 'Уплотнение', closed: false, open_intervention: 'INT-5' },
  ],
  documents: [{ document_id: 'DOC-TRAV-42', title: 'Сопроводительная карта изделия', template: 'traveler@3', digest: 'streebog256:aa11', status: 'in_route' }],
  entries: flangeEntries(),
  incidents: ['INC-7'],
  nonconformities: ['NC-0142'],
  lot_ids: ['LOT-ST-19'],
  ...over,
})

export const flangeHistory = (): ItemHistory => ({
  items: [
    { seq: 1241, event_id: 'e-signal', event_type: 'quality.signal.raised', field: 'quality', before: 'not_inspected', after: 'signal', recorded_at: at('08:53') },
    { seq: 1242, event_id: 'e-hold', event_type: 'decision.containment.applied', field: 'containment', before: 'none', after: 'item_hold', recorded_at: at('08:53'), reason: 'Правило RM-weld-7' },
    { seq: 1236, event_id: 'e-dm', event_type: 'item.carrier.applied', field: 'carrier', before: 'TAG-7788', after: 'DM-FL-0042', author: 'master-07', recorded_at: at('07:30') },
  ],
})

export const flangeGenealogy = (): ItemGenealogy => ({
  item_id: 'ENT:FL-0042',
  nodes: [
    { ref: 'ENT:FL-0042', kind: 'item', label: 'FL-0042', parent_ref: 'ENT:ASM-9' },
    { ref: 'ENT:ASM-9', kind: 'item', label: 'Сборка ASM-9', summary: 'in_process' },
    { ref: 'ENT:RING-3', kind: 'item', label: 'Кольцо RING-3', parent_ref: 'ENT:FL-0042', position: 'поз. 2', summary: 'released' },
    { ref: 'LOT-ST-19', kind: 'lot', label: 'Плавка ST-19' },
    { ref: 'PX-44', kind: 'partner_extract', label: 'Выписка поставщика ПЗ-44', provenance: 'Происхождение не подтверждено' },
  ],
})
