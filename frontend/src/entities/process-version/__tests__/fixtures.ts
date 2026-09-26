// Образцы версий процесса для тестов раздела «Процесс» — только в тестах
// (PRD §11.10). Версия 0.2 добавляет точку предъявления после сварки и меняет
// порог уверенности для прожога (FR-24).
import type { ProcessElement, ProcessVersion } from '@/entities/process-version'

const base = (): ProcessElement[] => [
  { id: 'start', kind: 'startEvent', name: 'Задание из 1С', properties: {}, next: ['edge'] },
  { id: 'edge', step_key: 'edge_prep', kind: 'operation', name: 'Подготовка кромок', lane: 'Сварочный цех', properties: { timeNorm: '20 мин' }, next: ['weld'] },
  {
    id: 'weld',
    step_key: 'welding',
    kind: 'operation',
    name: 'Сварка',
    lane: 'Сварочный цех',
    properties: { specialProcess: true, timeNorm: '40 мин', presentationPoint: false },
    next: ['vt'],
  },
  {
    id: 'vt',
    step_key: 'vt_after_weld',
    kind: 'automatedInspection',
    name: 'ВИК после сварки',
    properties: { inspectionMethod: 'ВИК', zone: 'Шов фланца' },
    thresholds: { burn_through: 8500, porosity: 8000 },
    next: ['end'],
  },
  { id: 'end', kind: 'endEvent', name: 'Выпуск', properties: {} },
]

export const processVersions = (): ProcessVersion[] => {
  const v1 = base()
  const v2 = base().map((e) => {
    if (e.id === 'weld') return { ...e, properties: { ...e.properties, timeNorm: '35 мин' }, next: ['otk'] }
    if (e.id === 'vt') return { ...e, thresholds: { burn_through: 8000, porosity: 8000 } }
    return e
  })
  v2.splice(3, 0, {
    id: 'otk',
    step_key: 'otk_hold',
    kind: 'humanInspection',
    name: 'Предъявление ОТК',
    properties: { presentationPoint: true, roleOrAuthority: 'Контролёр ОТК' },
    next: ['vt'],
  })
  return [
    { version_id: 'pv-0.2', label: '0.2', status: 'on_approval', author: 'Технолог Т-03', created_at: '2026-09-24T09:00:00.000Z', quorum: { have: 1, need: 3 }, elements: v2 },
    { version_id: 'pv-0.1', label: '0.1', status: 'active', author: 'Технолог Т-01', created_at: '2026-09-20T09:00:00.000Z', effective_from: '2026-09-21T06:00:00.000Z', elements: v1 },
  ]
}
