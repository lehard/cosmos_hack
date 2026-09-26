/**
 * Тексты участка (NFR-UI-3): нормы, аномалии узлов, факты растущей очереди —
 * только ключами текстов; неизвестный код — UNKNOWN(код), а не похожий текст.
 */
import type { NodeAnomaly } from '@/entities/live-map'
import { codeToKey } from '@/shared/i18n'
import { PRESENCE_TEXT } from '@/entities/workplace'
import type { QueueFact } from './station'

type T = (key: string, params?: Record<string, unknown>) => string

/** Текст аномалии узла с порогом от сервера. */
export function anomalyText(t: T, te: (key: string) => boolean, a: NodeAnomaly): string {
  const key = `liveMap.anomalies.${codeToKey(a.kind)}`
  if (!te(key)) return `UNKNOWN(${a.kind})`
  const text = t(key, { threshold: a.threshold ?? '—' })
  return a.threshold && a.kind !== 'downtime_over_threshold' ? `${text} (${a.threshold})` : text
}

/** Текст факта участка рядом с растущей очередью. */
export function factText(t: T, f: QueueFact, value: (v: Extract<QueueFact, { kind: 'unfinished' }>['value']) => string): string {
  switch (f.kind) {
    case 'presence':
      return t('widgets.shopFloor.why.presence', { post: f.post, person: f.person, presence: t(PRESENCE_TEXT[f.presence]) })
    case 'not_assigned':
      return t('widgets.shopFloor.why.notAssigned', { post: f.post })
    case 'equipment_state':
      return t('widgets.shopFloor.why.equipmentState', {
        equipment: f.equipment,
        state: f.condition === 'fault' ? t('statuses.equipmentCondition.fault') : t(`widgets.shopFloor.execution.${f.execution}`),
      })
    case 'equipment_no_data':
      return t('widgets.shopFloor.why.equipmentNoData', { equipment: f.equipment })
    case 'equipment_unusable':
      return t('widgets.shopFloor.why.equipmentUnusable', { equipment: f.equipment, reason: t(`widgets.shopFloor.unusable.${codeToKey(f.reason || 'unknown')}`) })
    case 'unfinished':
      return t('widgets.shopFloor.why.unfinished', { n: value(f.value) })
    case 'data_gap':
      return t('widgets.shopFloor.station.dataGap')
  }
}
