// Сжатая шкала дорожек (UI-33): дни без событий — разрыв, операция в минуты — видна.
import { describe, expect, it } from 'vitest'
import { makeCompressedScale } from '../model/timescale'

describe('сжатая шкала', () => {
  const moments = ['2026-09-18T12:05:00Z', '2026-09-18T12:45:00Z', '2026-09-21T04:35:00Z', '2026-09-23T07:40:00Z', '2026-09-23T08:05:00Z']

  it('промежутки дольше трёх часов — разрывы; события по порядку, операция 20 минут шире процента', () => {
    const s = makeCompressedScale(moments)
    expect(s.breaks).toHaveLength(2)
    const p = moments.map(s.pos)
    expect([...p].sort((a, b) => a - b)).toEqual(p)
    expect(s.pos('2026-09-23T08:00:00Z') - s.pos('2026-09-23T07:40:00Z')).toBeGreaterThan(5)
    // Момент внутри разрыва — в пределах разрыва.
    const b = s.breaks![0]!
    const mid = s.pos('2026-09-19T12:00:00Z')
    expect(mid).toBeGreaterThanOrEqual(b.left)
    expect(mid).toBeLessThanOrEqual(b.left + b.width)
  })

  it('у первого деления каждого скопления — дата', () => {
    const s = makeCompressedScale(moments)
    expect(s.dayTicks!.size).toBe(3)
  })

  it('без длинных промежутков — обычная шкала без разрывов', () => {
    const s = makeCompressedScale(['2026-09-23T07:40:00Z', '2026-09-23T08:05:00Z'])
    expect(s.breaks ?? []).toHaveLength(0)
  })
})
