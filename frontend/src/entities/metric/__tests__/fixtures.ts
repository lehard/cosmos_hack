// Входные данные тестов аналитики — в форме ответов контракта (как отдаёт мир
// заготовок «плохой день ИС-2», backend/internal/infrastructure/fixtures/world/render_analytics.go).
import type { AnalyticsOverview, ControlChart, MetricDrilldown, MetricTileList, MetricValue } from '@/shared/api/generated/model'

const pcs = (value: number): MetricValue => ({ value, scale: 0, unit: 'pcs' })
const bp = (value: number): MetricValue => ({ value, scale: 0, unit: 'bp' })
const mins = (value: number, origin?: MetricValue['origin']): MetricValue => ({ value, scale: 0, unit: 'min', ...(origin ? { origin } : {}) })
const period = { kind: 'day' as const, from: '2026-09-23T05:00:00.000Z', to: '2026-09-23T09:30:00.000Z' }

export const tiles = (): MetricTileList => ({
  period,
  items: [
    { metric_id: 'inspected_items', title: 'Проверено изделий', value: pcs(41), previous: pcs(38), unknown: false },
    { metric_id: 'items_with_confirmed_nc', title: 'Изделия с подтверждёнными несоответствиями', value: pcs(3), unknown: false },
    { metric_id: 'first_pass_yield', title: 'Прохождение контроля с первого раза (ЗТ-3)', value: bp(9250), previous: bp(9500), unknown: false },
    { metric_id: 'cause_established', title: 'Причина установлена', value: bp(0), unknown: true },
    { metric_id: 'lead_time', title: 'Время детали в системе (выпущенные)', value: mins(75, 'computed_by_system'), unknown: false },
  ],
})

export const overview = (): AnalyticsOverview => ({
  period,
  basis_seq: 1402,
  items: [
    { metric_id: 'inspected_items', title: 'Проверено изделий', group: 'inspection', total: pcs(41), unknown: false, slices: [] },
    { metric_id: 'items_with_confirmed_nc', title: 'Изделия с подтверждёнными несоответствиями', group: 'defects', total: pcs(3), unknown: false, slices: [] },
    {
      metric_id: 'confirmed_defects',
      title: 'Подтверждённые дефекты (производственные)',
      group: 'defects',
      total: pcs(5),
      unknown: false,
      slices: [
        { dimension: 'defect_type', key: 'porosity', label: 'Пористость', value: pcs(4) },
        { dimension: 'defect_type', key: 'undercut', label: 'Подрез', value: pcs(1) },
        { dimension: 'location', key: 'LINE-FL-1', label: 'Линия ФЛ-100 № 1', value: pcs(5) },
      ],
    },
    {
      metric_id: 'incoming_defects',
      title: 'Входной брак (отдельно от производственных ошибок)',
      group: 'causes',
      total: pcs(2),
      unknown: false,
      slices: [{ dimension: 'origin', key: 'incoming', label: 'Поставщик-3, партия П-117', value: pcs(2) }],
    },
    { metric_id: 'unable_to_assess', title: 'Оценка невозможна (отдельная корзина)', group: 'inspection', total: pcs(2), unknown: false, slices: [] },
    { metric_id: 'first_pass_yield', title: 'Прохождение контроля с первого раза (ЗТ-3)', group: 'inspection', total: bp(9250), unknown: false, slices: [] },
    { metric_id: 'cause_established', title: 'Причина установлена', group: 'causes', total: bp(5000), unknown: false, slices: [] },
    { metric_id: 'rework_runs', title: 'Повторные выполнения операций', group: 'defects', total: pcs(1), unknown: false, slices: [] },
    {
      metric_id: 'comparable_welds',
      title: 'Сопоставимые сварки по исполнителям',
      group: 'comparison',
      total: pcs(40),
      unknown: false,
      slices: [
        { dimension: 'performer', key: 'W21', label: 'Сварщик С-21', value: pcs(22) },
        { dimension: 'performer', key: 'W22', label: 'Сварщик С-22', value: pcs(18) },
      ],
    },
    {
      metric_id: 'confirmed_performer_errors',
      title: 'Подтверждённые ошибки исполнителей',
      group: 'people',
      total: pcs(0),
      unknown: false,
      slices: [{ dimension: 'performer', key: 'W21', label: 'Сварщик С-21', value: pcs(0) }],
    },
    {
      metric_id: 'equipment_downtime',
      title: 'Простой оборудования: остановка по качеству',
      group: 'equipment',
      total: mins(37, 'computed_by_system'),
      unknown: false,
      slices: [{ dimension: 'equipment', key: 'IS-2', label: 'Сварочный источник ИС-2', value: mins(37, 'computed_by_system') }],
    },
    { metric_id: 'lead_time', title: 'Время детали в системе', group: 'time', total: mins(75, 'computed_by_system'), unknown: false, slices: [] },
    { metric_id: 'weld_cycle', title: 'Цикл сварки', group: 'time', total: mins(12, 'reported_by_source'), unknown: false, slices: [] },
    { metric_id: 'waiting_raw', title: 'Ожидание без пометки', group: 'time', total: mins(5), unknown: false, slices: [] },
  ],
})

export const drilldown = (): MetricDrilldown => ({
  metric_id: 'items_with_confirmed_nc',
  period,
  total: pcs(2),
  items: [
    { item_id: 'ANT:FL-0042', label: 'ФЛ-100 № 0042', slice_key: 'LINE-FL-1', value: pcs(1), source_event_ids: ['ev-1', 'ev-2'], source_kinds: ['decision'] },
    { item_id: 'ANT:FL-0043', label: 'ФЛ-100 № 0043', slice_key: 'LINE-FL-1', value: pcs(1), source_event_ids: [], source_kinds: ['camera', 'robot'] },
  ],
})

export const chart = (): ControlChart => ({
  step_key: 'welding.weld',
  metric_id: 'current_a',
  title: 'Ток сварки',
  chart_kind: 'xmr',
  center: { value: 160, scale: 0, unit: 'A' },
  upper: { value: 170, scale: 0, unit: 'A' },
  lower: { value: 150, scale: 0, unit: 'A' },
  points: [
    { at: '2026-09-23T06:00:00.000Z', value: { value: 161, scale: 0, unit: 'A' }, out_of_control: false, ref: { entity: 'item', id: 'ANT:FL-0040' } },
    { at: '2026-09-23T06:20:00.000Z', value: { value: 158, scale: 0, unit: 'A' }, out_of_control: false, ref: { entity: 'item', id: 'ANT:FL-0041' } },
    { at: '2026-09-23T06:40:00.000Z', value: { value: 178, scale: 0, unit: 'A' }, out_of_control: true, ref: { entity: 'item', id: 'ANT:FL-0042' } },
  ],
})
