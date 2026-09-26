// Проваливание в детали (FR-7): изделие → паспорт; прочее — маршрут эпика-владельца,
// если он зарегистрирован; нет экрана — нет ссылки. Длительность просрочки текстом.
import { describe, expect, it } from 'vitest'
import { i18n } from '@/shared/i18n'
import { formatMinutes } from '@/shared/lib/duration'
import { drillTarget } from '../index'

describe('проваливание в детали', () => {
  it('изделие → паспорт; несоответствие — только если маршрут есть', () => {
    expect(drillTarget({ entity: 'item', id: 'ENT:FL-1' }, () => false)).toEqual({ name: 'item', params: { id: 'ENT:FL-1' } })
    expect(drillTarget({ entity: 'nonconformity', id: 'NC-1' }, () => false)).toBeNull()
    expect(drillTarget({ entity: 'nonconformity', id: 'NC-1' }, (n) => n === 'nonconformity')).toEqual({ name: 'nonconformity', params: { id: 'NC-1' } })
  })

  it('минуты → «37 мин», «2 ч», «1 ч 15 мин»', () => {
    const t = i18n.global.t as (k: string, p: Record<string, unknown>) => string
    const norm = (s: string) => s.replace(/\u00a0/g, ' ')
    expect(norm(formatMinutes(t, 37))).toBe('37 мин')
    expect(norm(formatMinutes(t, 120))).toBe('2 ч')
    expect(norm(formatMinutes(t, 75))).toBe('1 ч 15 мин')
  })
})
