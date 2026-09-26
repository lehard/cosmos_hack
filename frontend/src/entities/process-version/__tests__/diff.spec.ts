// Читаемая разница версии процесса с действующей (FR-24).
import { describe, expect, it } from 'vitest'
import { activeVersion, diffVersions } from '@/entities/process-version'
import { processVersions } from './fixtures'

describe('разница версий процесса', () => {
  it('точка предъявления после сварки, порог для прожога, норма времени', () => {
    const [draft, active] = processVersions()
    expect(activeVersion(processVersions())?.version_id).toBe('pv-0.1')
    expect(diffVersions(active!, draft!)).toEqual([
      { kind: 'propertyChanged', element: 'Сварка', property: 'timeNorm', from: '40 мин', to: '35 мин' },
      { kind: 'presentationPointAdded', step: 'Сварка' },
      { kind: 'thresholdChanged', element: 'ВИК после сварки', defectType: 'burn_through', from: 8500, to: 8000 },
    ])
  })

  it('одинаковые версии — отличий нет; удалённый элемент виден', () => {
    const [draft, active] = processVersions()
    expect(diffVersions(active!, active!)).toEqual([])
    expect(diffVersions(draft!, active!)).toContainEqual({ kind: 'elementRemoved', element: 'Предъявление ОТК' })
  })

  it('точка предъявления, включённая у существующего шага, — «после» этого шага', () => {
    const [, active] = processVersions()
    const next = structuredClone(active!)
    next.elements.find((e) => e.id === 'edge')!.properties.presentationPoint = true
    expect(diffVersions(active!, next)).toEqual([{ kind: 'presentationPointAdded', step: 'Подготовка кромок' }])
  })
})
